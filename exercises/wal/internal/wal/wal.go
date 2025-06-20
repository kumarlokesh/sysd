package wal

import (
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Config holds configuration options for the WAL.
type Config struct {
	Dir           string        // Directory to store WAL segments
	SegmentSize   int64         // Maximum size of each segment file in bytes
	Sync          bool          // Whether to sync writes to disk
	BufferSize    int           // Size of the write buffer in bytes
	FlushInterval time.Duration // Interval for background flushes
}

// WAL represents a write-ahead log.
type WAL struct {
	dir      string
	writer   *LogWriter
	reader   *LogReader
	config   *Config
	mu       sync.Mutex
	lastLSN  uint64 // Last used Log Sequence Number
	lastTxID uint64 // Last used Transaction ID

	txns     map[uint64]*Transaction
	txnsMu   sync.RWMutex
	nextTxID uint64 // Next transaction ID
}

// TransactionState represents the state of a transaction
type TransactionState string

const (
	// TransactionActive indicates a transaction is active
	TransactionActive TransactionState = "active"
	// TransactionCommitting indicates a transaction is being committed
	TransactionCommitting TransactionState = "committing"
	// TransactionCommitted indicates a transaction has been committed
	TransactionCommitted TransactionState = "committed"
	// TransactionAborted indicates a transaction has been aborted
	TransactionAborted TransactionState = "aborted"
)

// Transaction represents an active transaction
type Transaction struct {
	ID        uint64
	LSN       uint64
	State     TransactionState
	Records   []*Record
	StartedAt time.Time
}

// Open opens or creates a WAL in the given directory.
func Open(config *Config) (*WAL, error) {
	if err := os.MkdirAll(config.Dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create WAL directory: %w", err)
	}

	writer, err := NewLogWriter(config.Dir, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create log writer: %w", err)
	}

	reader, err := NewLogReader(config.Dir)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to create log reader: %w", err)
	}

	wal := &WAL{
		dir:      config.Dir,
		writer:   writer,
		reader:   reader,
		config:   config,
		txns:     make(map[uint64]*Transaction),
		nextTxID: 1,
	}

	// Recover any existing transactions
	if err := wal.recover(); err != nil {
		return nil, fmt.Errorf("recovery failed: %w", err)
	}

	return wal, nil
}

// Begin starts a new transaction and returns its ID.
func (w *WAL) Begin() uint64 {
	w.txnsMu.Lock()
	defer w.txnsMu.Unlock()

	txID := atomic.AddUint64(&w.lastTxID, 1)
	w.txns[txID] = &Transaction{
		ID:        txID,
		State:     TransactionActive,
		StartedAt: time.Now(),
	}
	return txID
}

// recover recovers the WAL state by reading all records and rebuilding in-memory state.
func (w *WAL) recover() error {
	w.txnsMu.Lock()
	defer w.txnsMu.Unlock()

	w.txns = make(map[uint64]*Transaction)
	w.nextTxID = 1

	if err := w.reader.SeekToStart(); err != nil {
		return fmt.Errorf("failed to reset reader during recovery: %w", err)
	}

	var maxTxID uint64

	// Track active transactions and their records
	transactions := make(map[uint64]*Transaction)

	// First pass: process all records to rebuild transaction state
	for {
		record, err := w.reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read record during recovery: %w", err)
		}

		// Update the last LSN
		if record.LSN > w.lastLSN {
			atomic.StoreUint64(&w.lastLSN, record.LSN)
		}

		switch record.Type {
		case RecordTypeTxnBegin:
			// Start a new transaction
			tx := &Transaction{
				ID:        record.TxID,
				State:     TransactionActive,
				StartedAt: time.Now(),
			}
			transactions[record.TxID] = tx
			if record.TxID > maxTxID {
				maxTxID = record.TxID
			}

		case RecordTypeTxnCommit:
			// Mark transaction as committed
			if tx, exists := transactions[record.TxID]; exists {
				tx.State = TransactionCommitted
				delete(transactions, record.TxID)
			}

		case RecordTypeTxnRollback:
			// Mark transaction as aborted
			if tx, exists := transactions[record.TxID]; exists {
				tx.State = TransactionAborted
				delete(transactions, record.TxID)
			}

		case RecordTypeWrite:
			// For write records, ensure the transaction exists if txID > 0
			if record.TxID > 0 {
				if _, exists := transactions[record.TxID]; !exists {
					tx := &Transaction{
						ID:        record.TxID,
						State:     TransactionActive,
						StartedAt: time.Now(),
					}
					transactions[record.TxID] = tx
					if record.TxID > maxTxID {
						maxTxID = record.TxID
					}
				}
			}
		}
	}

	// Set the next transaction ID to one more than the highest we've seen
	if maxTxID > 0 {
		w.nextTxID = maxTxID + 1
	}

	// Copy active transactions to the WAL's transaction map
	for txID, tx := range transactions {
		if tx.State == TransactionActive {
			w.txns[txID] = tx
		}
	}

	// Reset the reader again for normal operation
	if err := w.reader.SeekToStart(); err != nil {
		return fmt.Errorf("failed to reset reader after recovery: %w", err)
	}

	return nil
}

// generateLSN generates a new Log Sequence Number.
func (w *WAL) generateLSN() uint64 {
	return atomic.AddUint64(&w.lastLSN, 1)
}

// Write writes a new record to the WAL within the specified transaction.
// If txID is 0, the write is non-transactional and will be immediately committed.
// If txID > 0, the write is part of an existing transaction that must be committed separately.
func (w *WAL) Write(txID uint64, key, value []byte) (uint64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	lsn := w.generateLSN()
	record := NewWriteRecord(lsn, txID, key, value)

	// If this is a non-transactional write (txID=0), we need to ensure it's durable
	if txID == 0 {
		// For non-transactional writes, we write and flush immediately
		if _, err := w.writer.Write(record); err != nil {
			return 0, err
		}
		if err := w.writer.Flush(); err != nil {
			return 0, err
		}
		return lsn, nil
	}

	// For transactional writes, just write to the log
	// The transaction must be committed separately
	return w.writer.Write(record)
}

// Commit commits a transaction.
func (w *WAL) Commit(txID uint64) error {
	w.txnsMu.Lock()
	tx, exists := w.txns[txID]
	if !exists || tx.State != TransactionActive {
		w.txnsMu.Unlock()
		return fmt.Errorf("invalid or inactive transaction")
	}

	// Mark transaction as committing
	tx.State = TransactionCommitting
	w.txnsMu.Unlock()

	// Write commit record
	commitRecord := CommitTxnRecord(txID, w.generateLSN())
	if _, err := w.writer.Write(commitRecord); err != nil {
		return fmt.Errorf("failed to write commit record: %w", err)
	}

	// Ensure the commit is durable
	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush commit: %w", err)
	}

	// Mark transaction as committed
	w.txnsMu.Lock()
	defer w.txnsMu.Unlock()
	tx.State = TransactionCommitted
	delete(w.txns, txID)

	return nil
}

// Abort aborts a transaction.
func (w *WAL) Abort(txID uint64) error {
	w.txnsMu.Lock()
	defer w.txnsMu.Unlock()

	tx, exists := w.txns[txID]
	if !exists || tx.State != TransactionActive {
		return fmt.Errorf("invalid or inactive transaction")
	}

	// Write abort record
	abortRecord := RollbackTxnRecord(txID, w.generateLSN())
	if _, err := w.writer.Write(abortRecord); err != nil {
		return fmt.Errorf("failed to write abort record: %w", err)
	}

	// Mark transaction as aborted and clean up
	tx.State = TransactionAborted
	delete(w.txns, txID)

	return nil
}

// ReadAll reads all committed records from the WAL.
func (w *WAL) ReadAll() ([]*Record, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.reader.SeekToStart(); err != nil {
		return nil, fmt.Errorf("failed to reset reader: %w", err)
	}

	var (
		records      []*Record
		transactions = make(map[uint64]bool) // Tracks transaction commit/abort status
	)

	// First pass: collect all records and track transaction status
	var allRecords []*Record
	for {
		record, err := w.reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read record: %w", err)
		}

		allRecords = append(allRecords, record)

		switch record.Type {
		case RecordTypeTxnCommit:
			transactions[record.TxID] = true // Mark as committed
		case RecordTypeTxnRollback:
			transactions[record.TxID] = false // Mark as aborted
		}
	}

	// Second pass: include only records from committed transactions or non-transactional records (txID=0)
	for _, record := range allRecords {
		switch record.Type {
		case RecordTypeWrite:
			// Include non-transactional records (txID=0) or records from committed transactions
			if record.TxID == 0 || transactions[record.TxID] {
				records = append(records, record)
			}
		case RecordTypeTxnBegin, RecordTypeTxnCommit, RecordTypeTxnRollback:
			// Skip transaction control records in the final output
		default:
			// Include any other record types with txID=0 (non-transactional)
			if record.TxID == 0 {
				records = append(records, record)
			}
		}
	}

	return records, nil
}

// Close closes the WAL and releases any resources.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var err error

	if w.writer != nil {
		err = w.writer.Close()
	}
	if w.reader != nil {
		err2 := w.reader.Close()
		if err == nil {
			err = err2
		}
	}

	return err
}

// Sync flushes any buffered data to stable storage.
func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.writer == nil {
		return nil
	}

	return w.writer.Flush()
}
