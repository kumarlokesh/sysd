package sstable

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/kumarlokesh/sysd/exercises/cassandra-sstable/internal/trie"
)

// Reader implements reading from an SSTable file
type Reader struct {
	file        *os.File
	index       *trie.Trie
	indexOffset int64
	indexSize   int64
}

// Open opens an existing SSTable file for reading
func Open(filename string) (*Reader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open SSTable file: %w", err)
	}

	fileInfo, err := file.Stat()
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("%v; failed to close file: %w", err, closeErr)
		}
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}
	fileSize := fileInfo.Size()

	// Read the footer (last 24 bytes)
	if fileSize < 24 {
		if closeErr := file.Close(); closeErr != nil {
			return nil, fmt.Errorf("file too small to be a valid SSTable; failed to close file: %w", closeErr)
		}
		return nil, fmt.Errorf("file too small to be a valid SSTable")
	}

	footer := make([]byte, 24)
	if _, err := file.ReadAt(footer, fileSize-24); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("%v; failed to close file: %w", err, closeErr)
		}
		return nil, fmt.Errorf("failed to read footer: %w", err)
	}

	// Verify magic number
	magic := binary.BigEndian.Uint64(footer[16:24])
	if magic != magicNumber {
		if closeErr := file.Close(); closeErr != nil {
			return nil, fmt.Errorf("invalid magic number: %x; failed to close file: %w", magic, closeErr)
		}
		return nil, fmt.Errorf("invalid magic number: %x", magic)
	}

	// Read index offset and size
	indexOffset := int64(binary.BigEndian.Uint64(footer[0:8]))
	indexSize := int64(binary.BigEndian.Uint64(footer[8:16]))

	// Read the index
	if indexOffset < 0 || indexOffset+indexSize > fileSize {
		if closeErr := file.Close(); closeErr != nil {
			return nil, fmt.Errorf("invalid index offset or size; failed to close file: %w", closeErr)
		}
		return nil, fmt.Errorf("invalid index offset or size")
	}

	indexData := make([]byte, indexSize)
	if _, err := file.ReadAt(indexData, indexOffset); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("%v; failed to close file: %w", err, closeErr)
		}
		return nil, fmt.Errorf("failed to read index: %w", err)
	}

	// Deserialize the index
	trieIndex := trie.New()
	if err := trieIndex.Deserialize(indexData); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("%v; failed to close file: %w", err, closeErr)
		}
		return nil, fmt.Errorf("failed to deserialize index: %w", err)
	}

	return &Reader{
		file:        file,
		index:       trieIndex,
		indexOffset: indexOffset,
		indexSize:   indexSize,
	}, nil
}

// Close closes the reader and its underlying file
func (r *Reader) Close() error {
	if r.file == nil {
		return nil // Already closed
	}
	err := r.file.Close()
	r.file = nil
	return err
}

// Get retrieves the value for the given key
func (r *Reader) Get(key []byte) ([]byte, error) {
	// Find the block that might contain the key
	blockInfo, err := r.findBlockFor(key)
	if err != nil {
		return nil, err
	}

	// Read the block
	blockData := make([]byte, blockInfo.size)
	if _, err := r.file.ReadAt(blockData, blockInfo.offset); err != nil {
		return nil, fmt.Errorf("failed to read block: %w", err)
	}

	return r.searchInBlock(blockData, key)
}

// EntryIterator is an iterator over key-value pairs in the SSTable
type EntryIterator interface {
	// Next advances the iterator to the next key-value pair.
	// Returns false if there are no more pairs.
	Next() bool
	// Key returns the current key.
	Key() []byte
	// Value returns the current value.
	Value() []byte
	// Error returns any error encountered during iteration.
	Error() error
}

type entryIterator struct {
	reader     *Reader
	startKey   []byte
	endKey     []byte
	blockData  []byte
	blockIdx   int
	numInBlock int
	key        []byte
	value      []byte
	blockInfo  *BlockInfo // Track current block info
	blockNum   int        // Track which block we're in
	err        error
}

func (it *entryIterator) Next() bool {
	iteration := 0
	// TODO: Consider making maxIterations configurable or removing it in production
	// with proper logging/metrics for error cases. Current value is a safety measure
	// during development to prevent infinite loops from bugs in iteration logic.
	maxIterations := 1000

	for iteration < maxIterations {
		iteration++

		// If we have entries in the current block, process them
		if it.blockData != nil && it.blockIdx < len(it.blockData) {
			// Read key length
			if it.blockIdx+4 > len(it.blockData) {
				it.err = fmt.Errorf("invalid block data: cannot read key length")
				return false
			}
			keyLen := int(binary.BigEndian.Uint32(it.blockData[it.blockIdx:]))
			it.blockIdx += 4

			// Read key
			if it.blockIdx+keyLen > len(it.blockData) {
				it.err = fmt.Errorf("invalid block data: cannot read key")
				return false
			}
			key := make([]byte, keyLen)
			copy(key, it.blockData[it.blockIdx:it.blockIdx+keyLen])
			it.blockIdx += keyLen

			// Read value length
			if it.blockIdx+4 > len(it.blockData) {
				it.err = fmt.Errorf("invalid block data: cannot read value length")
				return false
			}
			valueLen := int(binary.BigEndian.Uint32(it.blockData[it.blockIdx:]))
			it.blockIdx += 4

			// Read value
			if it.blockIdx+valueLen > len(it.blockData) {
				it.err = fmt.Errorf("invalid block data: cannot read value")
				return false
			}

			// Skip if before start key
			if it.startKey != nil && bytes.Compare(key, it.startKey) < 0 {
				it.blockIdx += valueLen // Skip the value
				continue
			}

			// Stop if after end key
			if it.endKey != nil && bytes.Compare(key, it.endKey) > 0 {
				return false
			}

			// If we get here, we have a valid key-value pair within our range
			value := make([]byte, valueLen)
			copy(value, it.blockData[it.blockIdx:it.blockIdx+valueLen])
			it.blockIdx += valueLen

			it.key = key
			it.value = value
			return true
		}

		// Get the next block
		if it.blockInfo == nil {
			// First block - find the block containing the start key
			blockInfo, err := it.reader.findBlockFor(it.startKey)
			if err != nil {
				it.err = fmt.Errorf("failed to find first block: %w", err)
				return false
			}
			it.blockInfo = blockInfo
		} else {
			// Next block - find any block with a key > our current block's last key
			// We can use the block's last key + 1 to find the next block
			lastKey := string(append(it.key, 0)) // Next possible key after current key
			blockInfo, err := it.reader.findBlockFor([]byte(lastKey))
			if err != nil || blockInfo.offset == it.blockInfo.offset {
				// No more blocks or we're stuck in the same block
				return false
			}
			it.blockInfo = blockInfo
		}

		it.loadBlock(it.blockInfo)
		it.blockNum++

		if it.blockData == nil {
			return false
		}
	}

	it.err = fmt.Errorf("reached maximum number of iterations (%d)", maxIterations)
	return false
}

func (it *entryIterator) loadBlock(blockInfo *BlockInfo) {
	blockData := make([]byte, blockInfo.size)
	n, err := it.reader.file.ReadAt(blockData, blockInfo.offset)
	if err != nil {
		if closeErr := it.reader.file.Close(); closeErr != nil {
			err = fmt.Errorf("%v; failed to close file: %w", err, closeErr)
		}
		it.err = fmt.Errorf("failed to read block at offset %d (size: %d, read: %d): %w",
			blockInfo.offset, blockInfo.size, n, err)
		return
	}

	if len(blockData) < 4 {
		it.err = fmt.Errorf("block too small to contain entry count (size: %d)", len(blockData))
		return
	}

	// Read the number of entries in the block
	it.numInBlock = int(binary.BigEndian.Uint32(blockData[:4]))
	it.blockData = blockData[4:] // Skip the count
	it.blockIdx = 0

	it.key = nil
	it.value = nil
}

func (it *entryIterator) Key() []byte   { return it.key }
func (it *entryIterator) Value() []byte { return it.value }
func (it *entryIterator) Error() error  { return it.err }

// RangeScan returns an iterator over all key-value pairs where startKey <= key <= endKey.
// If startKey is nil, the range starts from the first key.
// If endKey is nil, the range continues to the last key.
func (r *Reader) RangeScan(startKey, endKey []byte) EntryIterator {
	// Make copies of the keys to prevent modification of the original slices
	var startCopy, endCopy []byte
	if startKey != nil {
		startCopy = make([]byte, len(startKey))
		copy(startCopy, startKey)
	}
	if endKey != nil {
		endCopy = make([]byte, len(endKey))
		copy(endCopy, endKey)
	}

	return &entryIterator{
		reader:   r,
		startKey: startCopy,
		endKey:   endCopy,
	}
}

// findBlockFor finds the block that might contain the given key
// If key is nil or empty, returns the first block in the SSTable
func (r *Reader) findBlockFor(key []byte) (*BlockInfo, error) {
	keyStr := string(key)
	var bestKey string
	var bestValue []byte

	// Traverse the trie to find the best matching key
	r.index.Traverse("", func(k string, v []byte) bool {
		// If we're looking for the first key (empty key), take the first one we find
		if keyStr == "" {
			bestKey = k
			bestValue = v
			return false // Stop after first key
		}

		// Otherwise, find the largest key that is <= our target key
		if k <= keyStr && (bestKey == "" || k > bestKey) {
			bestKey = k
			bestValue = v
		}
		return true
	})

	if bestKey == "" {
		return nil, fmt.Errorf("no blocks found in SSTable")
	}

	blockInfo, err := r.parseBlockInfo(bestValue)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block info: %w", err)
	}

	return blockInfo, nil
}

// parseBlockInfo parses the block info from the format "offset:size"
func (r *Reader) parseBlockInfo(blockData []byte) (*BlockInfo, error) {
	var offset, size int64
	_, err := fmt.Sscanf(string(blockData), "%d:%d", &offset, &size)
	if err != nil {
		return nil, fmt.Errorf("invalid block info: %w", err)
	}

	return &BlockInfo{
		offset: offset,
		size:   size,
	}, nil
}

// searchInBlock searches for a key in a block of data
func (r *Reader) searchInBlock(blockData []byte, key []byte) ([]byte, error) {
	reader := bytes.NewReader(blockData)

	// Read the number of entries in the block
	var numEntries uint32
	if err := binary.Read(reader, binary.BigEndian, &numEntries); err != nil {
		return nil, fmt.Errorf("failed to read number of entries: %w", err)
	}

	// Search for the key in the block
	for i := uint32(0); i < numEntries; i++ {
		// Read key length
		var keyLen uint32
		if err := binary.Read(reader, binary.BigEndian, &keyLen); err != nil {
			return nil, fmt.Errorf("failed to read key length: %w", err)
		}

		// Read key
		currentKey := make([]byte, keyLen)
		if _, err := io.ReadFull(reader, currentKey); err != nil {
			return nil, fmt.Errorf("failed to read key: %w", err)
		}

		// Read value length
		var valueLen uint32
		if err := binary.Read(reader, binary.BigEndian, &valueLen); err != nil {
			return nil, fmt.Errorf("failed to read value length: %w", err)
		}

		// If this is the key we're looking for, read and return the value
		if bytes.Equal(currentKey, key) {
			value := make([]byte, valueLen)
			if _, err := io.ReadFull(reader, value); err != nil {
				return nil, fmt.Errorf("failed to read value: %w", err)
			}
			return value, nil
		}

		// Otherwise, skip the value
		if _, err := reader.Seek(int64(valueLen), io.SeekCurrent); err != nil {
			return nil, fmt.Errorf("failed to skip value: %w", err)
		}
	}

	return nil, fmt.Errorf("key not found")
}
