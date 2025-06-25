package trie

import (
	"bytes"
	"testing"
)

func TestTrie_Serialize_Deserialize(t *testing.T) {
	tests := []struct {
		name    string
		inserts map[string][]byte
	}{
		{
			name:    "empty trie",
			inserts: map[string][]byte{},
		},
		{
			name: "single key-value",
			inserts: map[string][]byte{
				"test": []byte("value"),
			},
		},
		{
			name: "multiple keys",
			inserts: map[string][]byte{
				"test":  []byte("value1"),
				"trie":  []byte("value2"),
				"tree":  []byte("value3"),
				"trial": []byte("value4"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := New()
			for k, v := range tt.inserts {
				trie.Insert(k, v)
			}

			data, err := trie.Serialize()
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			newTrie := New()
			if err := newTrie.Deserialize(data); err != nil {
				t.Fatalf("Deserialize() error = %v", err)
			}

			for k, want := range tt.inserts {
				got := newTrie.Search(k)
				if !bytes.Equal(got, want) {
					t.Errorf("key %q: got %v, want %v", k, got, want)
				}
			}

			verifyTrieStructure(t, trie.root, newTrie.root)
		})
	}
}

func verifyTrieStructure(t *testing.T, expected, actual *Node) {
	t.Helper()

	if expected.isEnd != actual.isEnd {
		t.Errorf("isEnd mismatch: expected %v, got %v", expected.isEnd, actual.isEnd)
	}
	if expected.isEnd && !bytes.Equal(expected.value, actual.value) {
		t.Errorf("value mismatch: expected %v, got %v", expected.value, actual.value)
	}

	if len(expected.children) != len(actual.children) {
		t.Errorf("number of children mismatch: expected %d, got %d",
			len(expected.children), len(actual.children))
	}
	for ch, expectedChild := range expected.children {
		actualChild, exists := actual.children[ch]
		if !exists {
			t.Errorf("missing child with character '%c'", ch)
			continue
		}
		verifyTrieStructure(t, expectedChild, actualChild)
	}
}

func TestTrie_Serialization_RoundTrip(t *testing.T) {
	trie := New()
	trie.Insert("apple", []byte("fruit"))
	trie.Insert("app", []byte("short"))
	trie.Insert("banana", []byte("yellow"))
	trie.Insert("orange", []byte("orange"))

	data, err := trie.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}
	newTrie := New()
	if err := newTrie.Deserialize(data); err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	verifyTrieStructure(t, trie.root, newTrie.root)

	tests := []struct {
		key  string
		want []byte
	}{
		{"apple", []byte("fruit")},
		{"app", []byte("short")},
		{"banana", []byte("yellow")},
		{"orange", []byte("orange")},
	}

	for _, tt := range tests {
		got := newTrie.Search(tt.key)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("Search(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}

	prefixTests := []struct {
		prefix string
		want   int
	}{
		{"app", 2},
		{"ban", 1},
		{"ora", 1},
	}

	for _, tt := range prefixTests {
		got := len(newTrie.KeysWithPrefix(tt.prefix))
		if got != tt.want {
			t.Errorf("KeysWithPrefix(\"%s\") length = %d, want %d", tt.prefix, got, tt.want)
		}
	}
}
