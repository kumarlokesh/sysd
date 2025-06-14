package coordinator

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/common"
)

var (
	// ErrTransactionNotFound is returned when a transaction is not found
	ErrTransactionNotFound = errors.New("transaction not found")
	// ErrInvalidTransactionState is returned for invalid state transitions
	ErrInvalidTransactionState = errors.New("invalid transaction state")
	// ErrTransactionAlreadyExists is returned when creating a duplicate transaction
	ErrTransactionAlreadyExists = errors.New("transaction already exists")
	// ErrInvalidTimeout is returned for invalid timeout values
	ErrInvalidTimeout = errors.New("invalid timeout value")
	// ErrNoPartitions is returned when no partitions are provided
	ErrNoPartitions = errors.New("no partitions provided")
)

// Coordinator manages the lifecycle of transactions
type Coordinator struct {
	transactions map[common.TransactionID]*common.Transaction
	mu           sync.RWMutex
}

// NewCoordinator creates a new transaction coordinator
func NewCoordinator() *Coordinator {
	return &Coordinator{
		transactions: make(map[common.TransactionID]*common.Transaction),
	}
}

// BeginTransaction starts a new transaction
func (c *Coordinator) BeginTransaction(producerID string, timeout time.Duration) (*common.Transaction, error) {
	if timeout <= 0 {
		return nil, ErrInvalidTimeout
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	txID := common.TransactionID(fmt.Sprintf("tx-%d", time.Now().UnixNano()))
	tx := common.NewTransaction(txID, producerID, timeout)

	// Check for duplicate transaction ID (should be extremely rare with UUIDs)
	if _, exists := c.transactions[tx.ID]; exists {
		return nil, fmt.Errorf("%w: %s", ErrTransactionAlreadyExists, tx.ID)
	}

	c.transactions[tx.ID] = tx
	return tx, nil
}

// AddPartitionsToTransaction adds partitions to a transaction
func (c *Coordinator) AddPartitionsToTransaction(txID common.TransactionID, partitions []common.TopicPartition) ([]common.TopicPartition, error) {
	if len(partitions) == 0 {
		return nil, ErrNoPartitions
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	tx, exists := c.transactions[txID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTransactionNotFound, txID)
	}

	if tx.State != common.TransactionStateBegin {
		return nil, fmt.Errorf("%w: cannot add partitions to transaction in state %s",
			ErrInvalidTransactionState, tx.State)
	}

	// Track newly added partitions
	var added []common.TopicPartition

	// Add each partition to the transaction if not already present
	for _, p := range partitions {
		// Check if partition already exists
		found := false
		for _, existing := range tx.Partitions {
			if existing == p {
				found = true
				break
			}
		}
		if !found {
			tx.AddPartition(p.Topic, p.Partition)
			added = append(added, p)
		}
	}

	// Return only the newly added partitions
	return added, nil
}

// PrepareTransaction prepares a transaction for commit
func (c *Coordinator) PrepareTransaction(txID common.TransactionID) (*common.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tx, exists := c.transactions[txID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTransactionNotFound, txID)
	}

	if tx.State != common.TransactionStateBegin {
		return nil, fmt.Errorf("%w: cannot prepare transaction in state %s",
			ErrInvalidTransactionState, tx.State)
	}

	tx.UpdateState(common.TransactionStatePrepared)
	return tx, nil
}

// CommitTransaction commits a transaction
func (c *Coordinator) CommitTransaction(txID common.TransactionID) (*common.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tx, exists := c.transactions[txID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTransactionNotFound, txID)
	}

	if tx.State != common.TransactionStatePrepared {
		return nil, fmt.Errorf("%w: cannot commit transaction in state %s",
			ErrInvalidTransactionState, tx.State)
	}

	tx.UpdateState(common.TransactionStateCommitted)
	return tx, nil
}

// AbortTransaction aborts a transaction
func (c *Coordinator) AbortTransaction(txID common.TransactionID) (*common.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tx, exists := c.transactions[txID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTransactionNotFound, txID)
	}

	// Allow aborting in any state except already completed states
	if tx.State == common.TransactionStateCommitted || tx.State == common.TransactionStateAborted {
		return nil, fmt.Errorf("%w: cannot abort transaction in state %s",
			ErrInvalidTransactionState, tx.State)
	}

	tx.UpdateState(common.TransactionStateAborted)
	return tx, nil
}

// GetTransaction returns a transaction by ID
func (c *Coordinator) GetTransaction(txID common.TransactionID) (*common.Transaction, error) {
	if txID == "" {
		return nil, fmt.Errorf("%w: empty transaction ID", ErrTransactionNotFound)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	tx, exists := c.transactions[txID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTransactionNotFound, txID)
	}

	// Check if transaction has expired
	if tx.IsExpired() {
		// Clean up the expired transaction
		delete(c.transactions, txID)
		return nil, fmt.Errorf("%w: transaction %s has expired", ErrTransactionNotFound, txID)
	}

	return tx, nil
}

// CleanupExpiredTransactions removes transactions that have timed out
func (c *Coordinator) CleanupExpiredTransactions() []common.TransactionID {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expired []common.TransactionID
	now := time.Now()

	for id, tx := range c.transactions {
		if tx.State != common.TransactionStateCommitted &&
			tx.State != common.TransactionStateAborted &&
			now.Sub(tx.StartTimestamp) > tx.Timeout {
			tx.UpdateState(common.TransactionStateAborted)
			expired = append(expired, id)
		}
	}

	return expired
}
