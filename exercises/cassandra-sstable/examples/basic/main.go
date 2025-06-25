// Package main demonstrates basic usage of the SSTable package.
// This example shows how to create a new SSTable, write key-value pairs to it,
// and then read them back using both point lookups and range scans.
package main

import (
	"fmt"
	"log"

	"github.com/kumarlokesh/sysd/exercises/cassandra-sstable/internal/sstable"
)

func main() {
	writer, err := sstable.NewWriter("data.sst")
	if err != nil {
		log.Fatalf("Failed to create SSTable writer: %v", err)
	}

	err = writer.Add([]byte("key1"), []byte("value1"))
	if err != nil {
		log.Fatalf("Failed to add key1: %v", err)
	}
	err = writer.Add([]byte("key2"), []byte("value2"))
	if err != nil {
		log.Fatalf("Failed to add key2: %v", err)
	}

	err = writer.Flush()
	if err != nil {
		log.Fatalf("Failed to flush writer: %v", err)
	}
	err = writer.Close()
	if err != nil {
		log.Fatalf("Failed to close writer: %v", err)
	}

	reader, err := sstable.Open("data.sst")
	if err != nil {
		log.Fatalf("Failed to open SSTable: %v", err)
	}
	defer reader.Close()

	value, err := reader.Get([]byte("key1"))
	if err != nil {
		log.Fatalf("Failed to get key1: %v", err)
	}
	fmt.Printf("Value for key1: %s\n", value)

	fmt.Println("Range scan (key1 to key3):")
	iter := reader.RangeScan([]byte("key1"), []byte("key3"))
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		fmt.Printf("  Key: %s, Value: %s\n", key, value)
	}

	if err := iter.Error(); err != nil {
		log.Printf("Error during iteration: %v", err)
	}
}
