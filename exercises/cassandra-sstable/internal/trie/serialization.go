package trie

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
)

// Serialization format:
// [node type: 1 byte][value length: 4 bytes][value][num children: 4 bytes][(char: 4 bytes, child offset: 8 bytes)...]
// - node type: 0 = internal node, 1 = leaf node, 2 = internal node with value
// - value length: length of the value in bytes (0 if no value)
// - value: the actual value bytes (if any)
// - num children: number of child nodes
// - for each child: character (rune, 4 bytes) and offset (8 bytes) to the child node

const (
	nodeTypeInternal = iota
	nodeTypeLeaf
	nodeTypeInternalWithValue
)

// Serialize converts the trie to a byte slice
func (t *Trie) Serialize() ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := t.serializeNode(t.root, buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// serializeNode recursively serializes a node and its children
func (t *Trie) serializeNode(node *Node, w io.Writer) error {
	var nodeTypeStr string
	switch {
	case node.isEnd && len(node.children) > 0:
		nodeTypeStr = "internal with value"
	case node.isEnd:
		nodeTypeStr = "leaf"
	default:
		nodeTypeStr = "internal"
	}
	fmt.Printf("Serializing %s node with value: %v, children: %d\n",
		nodeTypeStr, string(node.value), len(node.children))

	// First, collect all children in a consistent order
	children := make([]rune, 0, len(node.children))
	for ch := range node.children {
		children = append(children, ch)
	}
	sort.Slice(children, func(i, j int) bool { return children[i] < children[j] })

	// Determine node type
	nodeType := nodeTypeInternal
	if node.isEnd && len(children) > 0 {
		nodeType = nodeTypeInternalWithValue
	} else if node.isEnd {
		nodeType = nodeTypeLeaf
	}

	// We need to know the size of this node before writing it
	// Node size = header (9) + valueLen + (numChildren * (4 + 8)) + sum of child sizes
	// But since we don't know child sizes yet, we'll use a two-pass approach

	// First, serialize all children to temporary buffers to get their sizes
	childBuffers := make([]*bytes.Buffer, len(children))
	for i, ch := range children {
		childBuf := &bytes.Buffer{}
		child := node.children[ch]
		if err := t.serializeNode(child, childBuf); err != nil {
			return fmt.Errorf("failed to serialize child node: %w", err)
		}
		childBuffers[i] = childBuf
	}

	// Calculate the size of this node's data (excluding children)
	headerSize := 9                                                 // type (1) + valueLen (4) + numChildren (4)
	nodeSize := headerSize + len(node.value) + (len(children) * 12) // 12 = 4 (rune) + 8 (offset)

	// Write node header
	header := make([]byte, 9)
	header[0] = byte(nodeType)
	binary.BigEndian.PutUint32(header[1:5], uint32(len(node.value)))
	binary.BigEndian.PutUint32(header[5:9], uint32(len(children)))

	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write value if present
	if len(node.value) > 0 {
		if _, err := w.Write(node.value); err != nil {
			return fmt.Errorf("failed to write value: %w", err)
		}
	}

	// If no children, we're done
	if len(children) == 0 {
		return nil
	}

	// Write children metadata (character + offset)
	currentOffset := nodeSize // Start of first child
	for i, ch := range children {
		// Write character
		if err := binary.Write(w, binary.BigEndian, ch); err != nil {
			return fmt.Errorf("failed to write child character: %w", err)
		}

		// Write offset to child
		offsetBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(offsetBytes, uint64(currentOffset))
		if _, err := w.Write(offsetBytes); err != nil {
			return fmt.Errorf("failed to write child offset: %w", err)
		}

		// Update offset for next child
		currentOffset += childBuffers[i].Len()
	}

	// Now write all child nodes
	for _, buf := range childBuffers {
		if _, err := w.Write(buf.Bytes()); err != nil {
			return fmt.Errorf("failed to write child node: %w", err)
		}
	}

	return nil
}

// Deserialize loads a trie from a byte slice
func (t *Trie) Deserialize(data []byte) error {
	if len(data) == 0 {
		t.root = newNode()
		return nil
	}

	r := bytes.NewReader(data)
	node, _, err := deserializeNode(r, 0)
	if err != nil {
		return fmt.Errorf("failed to deserialize trie: %w", err)
	}
	t.root = node
	return nil
}

// deserializeNode reads a node and its children from the reader
func deserializeNode(r io.ReadSeeker, offset int64) (*Node, int64, error) {
	fmt.Printf("Deserializing node at offset: %d\n", offset)
	node := newNode()

	// Seek to the node's position
	if _, err := r.Seek(offset, io.SeekStart); err != nil {
		return nil, 0, fmt.Errorf("failed to seek to offset %d: %w", offset, err)
	}

	// Read node header (type + valueLen + numChildren)
	header := make([]byte, 9)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, 0, fmt.Errorf("failed to read node header at offset %d: %w", offset, err)
	}

	nodeType := header[0]
	valueLen := binary.BigEndian.Uint32(header[1:5])
	numChildren := binary.BigEndian.Uint32(header[5:9])

	// Read value if present
	if valueLen > 0 {
		value := make([]byte, valueLen)
		if _, err := io.ReadFull(r, value); err != nil {
			return nil, 0, fmt.Errorf("failed to read node value: %w", err)
		}
		node.value = value
		node.isEnd = true
	} else {
		node.isEnd = (nodeType == nodeTypeLeaf || nodeType == nodeTypeInternalWithValue)
	}

	// If no children, we're done
	if numChildren == 0 {
		nextPos, err := r.Seek(0, io.SeekCurrent)
		return node, nextPos, err
	}

	// Read children characters and offsets
	childChars := make([]rune, 0, numChildren)
	childOffsets := make([]int64, 0, numChildren)

	// Read all children metadata first
	for i := uint32(0); i < numChildren; i++ {
		// Read character (4 bytes for rune)
		var ch rune
		if err := binary.Read(r, binary.BigEndian, &ch); err != nil {
			return nil, 0, fmt.Errorf("failed to read child character: %w", err)
		}

		// Read offset (8 bytes)
		var offsetBytes [8]byte
		if _, err := io.ReadFull(r, offsetBytes[:]); err != nil {
			return nil, 0, fmt.Errorf("failed to read child offset: %w", err)
		}
		offset := int64(binary.BigEndian.Uint64(offsetBytes[:]))

		childChars = append(childChars, ch)
		childOffsets = append(childOffsets, offset)
	}

	// Now deserialize each child node
	for i, ch := range childChars {
		childOffset := childOffsets[i]
		if childOffset < 0 {
			return nil, 0, fmt.Errorf("invalid child offset %d", childOffset)
		}

		// The offset is relative to the start of this node
		childNode, _, err := deserializeNode(r, offset+childOffset)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to deserialize child node: %w", err)
		}
		node.children[ch] = childNode
	}

	// Return the current position after reading this node
	nextPos, err := r.Seek(0, io.SeekCurrent)
	return node, nextPos, err
}
