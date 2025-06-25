package trie

// Node represents a node in the trie
type Node struct {
	// children maps the next character to the child node
	children map[rune]*Node

	// isEnd marks if this node represents the end of a key
	isEnd bool

	// value stores the value associated with the key (if this is an end node)
	value []byte
}

// newNode creates a new trie node
func newNode() *Node {
	return &Node{
		children: make(map[rune]*Node),
		isEnd:    false,
		value:    nil,
	}
}

// Trie represents a trie data structure
type Trie struct {
	root *Node
}

// New creates a new empty trie
func New() *Trie {
	return &Trie{
		root: newNode(),
	}
}
