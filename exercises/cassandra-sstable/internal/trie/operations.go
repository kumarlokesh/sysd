package trie

import (
	"sort"
)

// Insert adds a key-value pair to the trie
func (t *Trie) Insert(key string, value []byte) {
	node := t.root
	for _, ch := range key {
		if _, exists := node.children[ch]; !exists {
			node.children[ch] = newNode()
		}
		node = node.children[ch]
	}
	node.isEnd = true
	node.value = make([]byte, len(value))
	copy(node.value, value)
}

// Search returns the value associated with the key, or nil if not found
func (t *Trie) Search(key string) []byte {
	node := t.findNode(key)
	if node != nil && node.isEnd {
		return node.value
	}
	return nil
}

// findNode returns the node corresponding to the key, or nil if not found
func (t *Trie) findNode(key string) *Node {
	node := t.root
	for _, ch := range key {
		if _, exists := node.children[ch]; !exists {
			return nil
		}
		node = node.children[ch]
	}
	return node
}

// KeysWithPrefix returns all keys in the trie that have the given prefix
func (t *Trie) KeysWithPrefix(prefix string) []string {
	var results []string
	node := t.findNode(prefix)
	if node == nil {
		return results
	}

	// Start collecting keys if the prefix itself is a complete key
	if node.isEnd {
		results = append(results, prefix)
	}

	// Collect all keys with the given prefix
	collectKeys(node, prefix, &results)
	return results
}

// collectKeys is a helper function to recursively collect all keys in the trie
func collectKeys(node *Node, prefix string, results *[]string) {
	for ch, child := range node.children {
		newPrefix := prefix + string(ch)
		if child.isEnd {
			*results = append(*results, newPrefix)
		}
		collectKeys(child, newPrefix, results)
	}
}

// TraverseFunc is the type of the function called for each key-value pair in the trie.
// If the function returns false, the traversal stops.
type TraverseFunc func(key string, value []byte) bool

// Traverse traverses all key-value pairs in the trie in lexicographical order.
// For each key-value pair, it calls the given function. If the function returns false,
// the traversal stops.
func (t *Trie) Traverse(prefix string, f TraverseFunc) {
	node := t.findNode(prefix)
	if node == nil {
		return
	}

	// If the prefix itself is a key, call the function with it
	if node.isEnd {
		if !f(prefix, node.value) {
			return
		}
	}

	// Traverse all keys with the given prefix
	t.traverseNode(node, prefix, f)
}

// traverseNode is a helper function that recursively traverses the trie
func (t *Trie) traverseNode(node *Node, prefix string, f TraverseFunc) bool {
	// Get all children in sorted order to ensure consistent traversal
	var children []rune
	for ch := range node.children {
		children = append(children, ch)
	}
	sort.Slice(children, func(i, j int) bool { return children[i] < children[j] })

	for _, ch := range children {
		child := node.children[ch]
		newPrefix := prefix + string(ch)
		if child.isEnd {
			if !f(newPrefix, child.value) {
				return false
			}
		}

		if !t.traverseNode(child, newPrefix, f) {
			return false
		}
	}

	return true
}
