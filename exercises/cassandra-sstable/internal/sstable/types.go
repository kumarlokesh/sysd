package sstable

// BlockInfo contains metadata about a data block
type BlockInfo struct {
	offset int64
	size   int64
}

// Entry represents a key-value pair in the SSTable
type Entry struct {
	Key   []byte
	Value []byte
}
