package wal_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kumarlokesh/sysd/exercises/wal/internal/wal"
)

func TestExamples(t *testing.T) {
	t.Run("BasicUsage", testBasicUsage)
	t.Run("TransactionExample", testTransactionExample)
}

func testBasicUsage(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "wal-example-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a new WAL instance
	config := &wal.Config{
		Dir:           tempDir,
		SegmentSize:   64 * 1024 * 1024, // 64MB
		BufferSize:    64 * 1024,        // 64KB
		FlushInterval: time.Second,
	}

	w, err := wal.Open(config)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer w.Close()

	// Example 1: Simple write without explicit transaction (txID=0)
	key1 := []byte("key1")
	value1 := []byte("value1")
	t.Logf("Writing record: key=%s, value=%s\n", key1, value1)
	lsn1, err := w.Write(0, key1, value1)
	if err != nil {
		t.Fatalf("Failed to write to WAL: %v", err)
	}
	t.Logf("Wrote record: LSN=%d, key=%s, value=%s\n", lsn1, key1, value1)

	// Read all records
	records, err := w.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read from WAL: %v", err)
	}

	t.Logf("\nRead %d records from WAL:", len(records))
	for _, r := range records {
		t.Logf("LSN=%d, TxID=%d, Type=%d, Key=%s, Value=%s",
			r.LSN, r.TxID, r.Type, r.Key, r.Value)
	}
}

func testTransactionExample(t *testing.T) {
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "wal-tx-example-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a new WAL instance
	config := &wal.Config{
		Dir:           tempDir,
		SegmentSize:   64 * 1024 * 1024, // 64MB
		BufferSize:    64 * 1024,        // 64KB
		FlushInterval: time.Second,
	}

	w, err := wal.Open(config)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer w.Close()

	// Example 2: Transaction with multiple writes
	txID := w.Begin()
	t.Logf("Started transaction: %d", txID)

	// Write multiple records in the transaction
	for i := 0; i < 3; i++ {
		key := []byte(fmt.Sprintf("tx-key-%d", i))
		value := []byte(fmt.Sprintf("tx-value-%d", i))
		lsn, err := w.Write(txID, key, value)
		if err != nil {
			t.Fatalf("Failed to write to WAL: %v", err)
		}
		t.Logf("  Wrote record: LSN=%d, TxID=%d, key=%s, value=%s", lsn, txID, key, value)
	}

	// Commit the transaction
	if err := w.Commit(txID); err != nil {
		t.Fatalf("Failed to commit transaction %d: %v", txID, err)
	}
	t.Logf("Committed transaction %d", txID)

	// Example 3: Aborted transaction
	txID = w.Begin()
	t.Logf("Started transaction: %d (will be aborted)", txID)

	// Write a record that will be aborted
	abortKey := []byte("aborted-key")
	abortValue := []byte("aborted-value")
	abortLSN, err := w.Write(txID, abortKey, abortValue)
	if err != nil {
		t.Fatalf("Failed to write to WAL: %v", err)
	}
	t.Logf("  Wrote record (will be aborted): LSN=%d, TxID=%d, key=%s, value=%s",
		abortLSN, txID, abortKey, abortValue)

	// Abort the transaction
	if err := w.Abort(txID); err != nil {
		t.Fatalf("Failed to abort transaction %d: %v", txID, err)
	}
	t.Logf("Aborted transaction %d", txID)

	// Read all records (should not include aborted transaction)
	records, err := w.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read from WAL: %v", err)
	}

	t.Logf("\nFinal records in WAL (should not include aborted transaction):")
	for _, r := range records {
		t.Logf("LSN=%d, TxID=%d, Type=%d, Key=%s, Value=%s",
			r.LSN, r.TxID, r.Type, r.Key, r.Value)
	}
}
