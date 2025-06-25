package sstable

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"sort"

	"github.com/kumarlokesh/sysd/exercises/cassandra-sstable/internal/trie"
)

const (
	// Magic number to identify SSTable files
	magicNumber = 0x53535442 // 'SSTB' in ASCII

	// Current version of the SSTable format
	version = 1

	// Block size for data storage (4KB)
	blockSize = 4 * 1024
)

// Entry and BlockInfo types are now defined in types.go

// Writer implements writing data to an SSTable file
type Writer struct {
	file       *os.File
	offset     int64
	index      *trie.Trie
	entries    []Entry
	blockInfos []BlockInfo
}

// NewWriter creates a new SSTable writer for the given file
func NewWriter(filename string) (*Writer, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSTable file: %w", err)
	}

	// Write the header
	header := make([]byte, 16) // magic (8) + version (8)
	binary.BigEndian.PutUint64(header[0:8], magicNumber)
	binary.BigEndian.PutUint64(header[8:16], version)

	if _, err := file.Write(header); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			err = fmt.Errorf("failed to write SSTable header: %v; failed to close file: %w", err, closeErr)
		}
		return nil, fmt.Errorf("failed to write SSTable header: %w", err)
	}

	w := &Writer{
		file:       file,
		offset:     int64(len(header)),
		index:      trie.New(),
		entries:    make([]Entry, 0, 1024),
		blockInfos: make([]BlockInfo, 0, 128),
	}

	return w, nil
}

// Add adds a key-value pair to the SSTable
func (w *Writer) Add(key, value []byte) error {
	// Create a copy of the key and value to avoid potential issues with the original slices
	keyCopy := make([]byte, len(key))
	valueCopy := make([]byte, len(value))
	copy(keyCopy, key)
	copy(valueCopy, value)

	w.entries = append(w.entries, Entry{
		Key:   keyCopy,
		Value: valueCopy,
	})

	return nil
}

// writeBlock writes a block of entries to the file
func (w *Writer) writeBlock(entries []Entry) (BlockInfo, error) {
	var buf bytes.Buffer

	// Write the number of entries in this block
	if err := binary.Write(&buf, binary.BigEndian, uint32(len(entries))); err != nil {
		return BlockInfo{}, fmt.Errorf("failed to write entry count: %w", err)
	}

	// Write each entry (key length, key, value length, value)
	for _, entry := range entries {
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(entry.Key))); err != nil {
			return BlockInfo{}, fmt.Errorf("failed to write key length: %w", err)
		}
		if _, err := buf.Write(entry.Key); err != nil {
			return BlockInfo{}, fmt.Errorf("failed to write key: %w", err)
		}
		if err := binary.Write(&buf, binary.BigEndian, uint32(len(entry.Value))); err != nil {
			return BlockInfo{}, fmt.Errorf("failed to write value length: %w", err)
		}
		if _, err := buf.Write(entry.Value); err != nil {
			return BlockInfo{}, fmt.Errorf("failed to write value: %w", err)
		}
	}

	// Write the block to the file
	blockOffset := w.offset
	blockData := buf.Bytes()
	n, err := w.file.Write(blockData)
	if err != nil {
		return BlockInfo{}, fmt.Errorf("failed to write block data: %w", err)
	}

	// Update the offset
	w.offset += int64(n)

	return BlockInfo{
		offset: blockOffset,
		size:   int64(n),
	}, nil
}

// writeIndex writes the index to the file
func (w *Writer) writeIndex() (int64, int64, error) {
	// Serialize the trie index
	indexData, err := w.index.Serialize()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to serialize index: %w", err)
	}

	// Write the index
	indexOffset := w.offset
	n, err := w.file.Write(indexData)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to write index: %w", err)
	}

	// Update the offset
	w.offset += int64(n)

	return indexOffset, int64(n), nil
}

// Flush writes all buffered data to disk
func (w *Writer) Flush() error {
	if len(w.entries) == 0 {
		return nil // Nothing to flush
	}

	// Sort entries by key
	sort.Slice(w.entries, func(i, j int) bool {
		return bytes.Compare(w.entries[i].Key, w.entries[j].Key) < 0
	})

	// Process entries in blocks
	for i := 0; i < len(w.entries); {
		// Find the end of the current block
		currentBlockSize := 0
		j := i
		for ; j < len(w.entries); j++ {
			// Estimate entry size: 4 (key len) + key + 4 (value len) + value
			extra := 8 + len(w.entries[j].Key) + len(w.entries[j].Value)
			if currentBlockSize+extra > blockSize && j > i {
				break // This entry would exceed the block size
			}
			currentBlockSize += extra
		}

		// Write the block
		blockInfo, err := w.writeBlock(w.entries[i:j])
		if err != nil {
			return fmt.Errorf("failed to write block %d: %w", len(w.blockInfos), err)
		}

		// Add the first key of the block to the index
		if len(w.entries[i:j]) > 0 {
			firstKey := string(w.entries[i].Key)
			value := fmt.Sprintf("%d:%d", blockInfo.offset, blockInfo.size)
			w.index.Insert(firstKey, []byte(value))
		}

		w.blockInfos = append(w.blockInfos, blockInfo)
		i = j
	}

	// Clear the entries since they've been written
	w.entries = w.entries[:0]

	return nil
}

// Close closes the writer and writes any remaining data
func (w *Writer) Close() error {
	if w.file == nil {
		return nil // Already closed
	}

	// Flush any remaining entries
	if err := w.Flush(); err != nil {
		if closeErr := w.file.Close(); closeErr != nil {
			err = fmt.Errorf("failed to flush remaining data: %v; failed to close file: %w", err, closeErr)
		}
		return fmt.Errorf("failed to flush remaining data: %w", err)
	}

	// Write the index
	indexOffset, indexSize, err := w.writeIndex()
	if err != nil {
		if closeErr := w.file.Close(); closeErr != nil {
			err = fmt.Errorf("%v; failed to close file: %w", err, closeErr)
		}
		return fmt.Errorf("failed to write index: %w", err)
	}

	// Write the footer
	footer := make([]byte, 24) // index offset (8) + index size (8) + magic (8)
	binary.BigEndian.PutUint64(footer[0:8], uint64(indexOffset))
	binary.BigEndian.PutUint64(footer[8:16], uint64(indexSize))
	binary.BigEndian.PutUint64(footer[16:24], magicNumber) // Magic number at the end for validation

	if _, err := w.file.Write(footer); err != nil {
		if closeErr := w.file.Close(); closeErr != nil {
			err = fmt.Errorf("%v; failed to close file: %w", err, closeErr)
		}
		return fmt.Errorf("failed to write footer: %w", err)
	}

	if err := w.file.Close(); err != nil {
		return fmt.Errorf("failed to close file: %w", err)
	}

	w.file = nil
	return nil
}
