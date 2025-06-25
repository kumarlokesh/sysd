package trie

import (
	"reflect"
	"testing"
)

func TestTrie_InsertAndSearch(t *testing.T) {
	trie := New()

	key := "test"
	value := []byte("value")
	trie.Insert(key, value)

	if got := trie.Search(key); !reflect.DeepEqual(got, value) {
		t.Errorf("Search() = %v, want %v", got, value)
	}

	if got := trie.Search("nonexistent"); got != nil {
		t.Errorf("Search(nonexistent) = %v, want nil", got)
	}
}

func TestTrie_KeysWithPrefix(t *testing.T) {
	trie := New()

	testData := map[string][]byte{
		"apple":  []byte("fruit"),
		"app":    []byte("short"),
		"banana": []byte("yellow"),
		"orange": []byte("orange"),
	}

	for k, v := range testData {
		trie.Insert(k, v)
	}

	tests := []struct {
		name   string
		prefix string
		want   []string
	}{
		{
			name:   "prefix 'app'",
			prefix: "app",
			want:   []string{"app", "apple"},
		},
		{
			name:   "prefix 'ban'",
			prefix: "ban",
			want:   []string{"banana"},
		},
		{
			name:   "non-existent prefix",
			prefix: "xyz",
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trie.KeysWithPrefix(tt.prefix)
			if !stringSlicesEqual(got, tt.want) {
				t.Errorf("KeysWithPrefix() = %v, want %v", got, tt.want)
			}
		})
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
