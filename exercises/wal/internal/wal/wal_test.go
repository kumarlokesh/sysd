package wal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWAL_Minimal(t *testing.T) {
	t.Log("Running minimal WAL test...")

	tempDir, err := os.MkdirTemp("", "wal-minimal-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &Config{
		Dir:         tempDir,
		Sync:        true,
		SegmentSize: 1024 * 1024, // 1MB segments for testing
	}

	wal, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}

	key := []byte("test-key")
	value := []byte("test-value")

	_, err = wal.Write(0, key, value)
	if err != nil {
		t.Fatalf("Failed to write to WAL: %v", err)
	}

	if err := wal.Close(); err != nil {
		t.Fatalf("Failed to close WAL: %v", err)
	}
}

func TestWAL_BasicWriteRead(t *testing.T) {
	t.Log("Running basic write/read test...")

	tempDir, err := os.MkdirTemp("", "wal-basic-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &Config{
		Dir:         tempDir,
		Sync:        true,
		SegmentSize: 16 * 1024 * 1024, // 16MB segments
	}

	wal, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}

	testData := []struct {
		key   []byte
		value []byte
	}{
		{[]byte("key1"), []byte("value1")},
		{[]byte("key2"), []byte("value2")},
		{[]byte("key3"), []byte("value3")},
	}

	for _, td := range testData {
		_, err := wal.Write(0, td.key, td.value)
		if err != nil {
			t.Fatalf("Failed to write to WAL: %v", err)
		}
	}

	if err := wal.Close(); err != nil {
		t.Fatalf("Failed to close WAL: %v", err)
	}

	// Reopen WAL to test recovery
	wal, err = Open(config)
	if err != nil {
		t.Fatalf("Failed to reopen WAL: %v", err)
	}

	// Read all records
	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read from WAL: %v", err)
	}

	// Verify records
	if len(records) != len(testData) {
		t.Fatalf("Expected %d records, got %d", len(testData), len(records))
	}

	for i, rec := range records {
		expected := testData[i]
		if !bytes.Equal(rec.Key, expected.key) {
			t.Errorf("Record %d: expected key %s, got %s", i, expected.key, rec.Key)
		}
		if !bytes.Equal(rec.Value, expected.value) {
			t.Errorf("Record %d: expected value %s, got %s", i, expected.value, rec.Value)
		}
	}

	// Close WAL
	if err := wal.Close(); err != nil {
		t.Fatalf("Failed to close WAL: %v", err)
	}
}

func TestWAL_SegmentRotation(t *testing.T) {
	t.Log("Testing segment rotation...")

	tempDir, err := os.MkdirTemp("", "wal-rotation-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create WAL with small segment size to force rotation
	segmentSize := 1024 // 1KB segments
	config := &Config{
		Dir:         tempDir,
		Sync:        true,
		SegmentSize: int64(segmentSize),
	}

	wal, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}

	// Write enough data to trigger segment rotation
	// Each record is ~100 bytes, so 20 records should be enough to exceed 1KB
	for i := 0; i < 20; i++ {
		key := []byte("key" + string(rune('A'+i)))
		value := make([]byte, 90) // ~90 bytes per value
		for j := range value {
			value[j] = byte('A' + (i+j)%26)
		}
		_, err := wal.Write(0, key, value)
		if err != nil {
			t.Fatalf("Failed to write to WAL: %v", err)
		}
	}

	// Close WAL
	if err := wal.Close(); err != nil {
		t.Fatalf("Failed to close WAL: %v", err)
	}

	// Check that we have multiple segment files
	files, err := filepath.Glob(filepath.Join(tempDir, "*.wal"))
	if err != nil {
		t.Fatalf("Failed to list segment files: %v", err)
	}

	if len(files) <= 1 {
		t.Errorf("Expected multiple segment files, got %d", len(files))
	}

	if len(files) <= 1 {
		t.Errorf("Expected multiple segment files, got %d", len(files))
	}
}

func TestWAL_Transactions(t *testing.T) {
	t.Log("Running transaction tests...")

	tempDir, err := os.MkdirTemp("", "wal-transaction-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &Config{
		Dir:         tempDir,
		Sync:        true,
		SegmentSize: 16 * 1024 * 1024, // 16MB segments
	}

	wal, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}

	// Test 1: Simple transaction with single write
	t.Run("SingleWriteTransaction", func(t *testing.T) {
		txID := wal.Begin()
		t.Logf("Started transaction: %d", txID)

		// Write a record in the transaction
		key := []byte("tx-key-1")
		value := []byte("tx-value-1")
		lsn, err := wal.Write(txID, key, value)
		if err != nil {
			t.Fatalf("Failed to write to WAL: %v", err)
		}
		t.Logf("Wrote record: LSN=%d, TxID=%d, key=%s, value=%s", lsn, txID, key, value)

		// Commit the transaction
		if err := wal.Commit(txID); err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}
		t.Logf("Committed transaction %d", txID)

		// Verify the record was written
		records, err := wal.ReadAll()
		if err != nil {
			t.Fatalf("Failed to read records: %v", err)
		}

		if len(records) != 1 {
			t.Fatalf("Expected 1 record, got %d", len(records))
		}
		if !bytes.Equal(records[0].Key, key) || !bytes.Equal(records[0].Value, value) {
			t.Errorf("Record data mismatch. Got key=%s, value=%s, want key=%s, value=%s",
				records[0].Key, records[0].Value, key, value)
		}
	})

	// Test 2: Transaction with multiple writes
	t.Run("MultiWriteTransaction", func(t *testing.T) {
		txID := wal.Begin()
		t.Logf("Started transaction: %d", txID)

		// Write multiple records in the transaction
		testData := []struct {
			key   []byte
			value []byte
		}{
			{[]byte("tx-key-2"), []byte("tx-value-2")},
			{[]byte("tx-key-3"), []byte("tx-value-3")},
		}

		for _, td := range testData {
			_, err := wal.Write(txID, td.key, td.value)
			if err != nil {
				t.Fatalf("Failed to write to WAL: %v", err)
			}
		}

		// Commit the transaction
		if err := wal.Commit(txID); err != nil {
			t.Fatalf("Failed to commit transaction: %v", err)
		}

		// Verify the records were written
		records, err := wal.ReadAll()
		if err != nil {
			t.Fatalf("Failed to read records: %v", err)
		}

		// Should have 3 records now (1 from previous test + 2 from this test)
		if len(records) != 3 {
			t.Fatalf("Expected 3 records, got %d", len(records))
		}
	})

	// Test 3: Aborted transaction
	t.Run("AbortedTransaction", func(t *testing.T) {
		txID := wal.Begin()
		t.Logf("Started transaction: %d (will abort)", txID)

		// Write a record that will be aborted
		key := []byte("aborted-key")
		value := []byte("aborted-value")
		_, err := wal.Write(txID, key, value)
		if err != nil {
			t.Fatalf("Failed to write to WAL: %v", err)
		}

		// Abort the transaction
		if err := wal.Abort(txID); err != nil {
			t.Fatalf("Failed to abort transaction: %v", err)
		}

		// Verify the aborted record was not written
		records, err := wal.ReadAll()
		if err != nil {
			t.Fatalf("Failed to read records: %v", err)
		}

		// Should still only have 3 records (from previous tests)
		if len(records) != 3 {
			t.Errorf("Expected 3 records after abort, got %d", len(records))
		}

		// Verify the aborted record is not present
		for _, r := range records {
			if bytes.Equal(r.Key, key) && bytes.Equal(r.Value, value) {
				t.Error("Found record from aborted transaction")
			}
		}
	})

	// Close the WAL
	if err := wal.Close(); err != nil {
		t.Fatalf("Failed to close WAL: %v", err)
	}
}

func TestWAL_Recovery(t *testing.T) {
	t.Log("Testing recovery...")

	tempDir, err := os.MkdirTemp("", "wal-recovery-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config := &Config{
		Dir:         tempDir,
		Sync:        true,
		SegmentSize: 1024 * 1024, // 1MB segments
	}

	// First run: write some records
	func() {
		wal, err := Open(config)
		if err != nil {
			t.Fatalf("Failed to open WAL: %v", err)
		}
		defer wal.Close()

		// Write some records
		for i := 0; i < 10; i++ {
			key := []byte("key" + string(rune('A'+i)))
			value := []byte("value" + string(rune('0'+i)))
			_, err := wal.Write(0, key, value)
			if err != nil {
				t.Fatalf("Failed to write to WAL: %v", err)
			}
		}
	}()

	// Second run: append more records
	func() {
		wal, err := Open(config)
		if err != nil {
			t.Fatalf("Failed to open WAL: %v", err)
		}

		// Write more records
		for i := 10; i < 20; i++ {
			key := []byte("key" + string(rune('A'+i)))
			value := []byte("value" + string(rune('0'+(i%10))))
			_, err := wal.Write(0, key, value)
			if err != nil {
				t.Fatalf("Failed to write to WAL: %v", err)
			}
		}

		if err := wal.Close(); err != nil {
			t.Fatalf("Failed to close WAL: %v", err)
		}
	}()

	// Final run: verify all records are present
	wal, err := Open(config)
	if err != nil {
		t.Fatalf("Failed to open WAL: %v", err)
	}
	defer wal.Close()

	records, err := wal.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read from WAL: %v", err)
	}

	// Should have 20 records in total (0-19)
	if len(records) != 20 {
		t.Fatalf("Expected 20 records, got %d", len(records))
	}

	// Verify record contents
	for i, rec := range records {
		expectedKey := []byte("key" + string(rune('A'+i%20)))
		expectedValue := []byte("value" + string(rune('0'+(i%10))))

		if !bytes.Equal(rec.Key, expectedKey) {
			t.Errorf("Record %d: expected key %s, got %s", i, expectedKey, rec.Key)
		}
		if !bytes.Equal(rec.Value, expectedValue) {
			t.Errorf("Record %d: expected value %s, got %s", i, expectedValue, rec.Value)
		}
	}

}
