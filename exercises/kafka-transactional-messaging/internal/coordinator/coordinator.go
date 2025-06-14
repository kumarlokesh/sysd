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
)

// Coordinator manages the lifecycle of transactions
type Coordinator struct {
	transactions map[common.TransactionID]*common.Transaction
	mu          sync.RWMutex
}

// NewCoordinator creates a new transaction coordinator
func NewCoordinator() *Coordinator {
	return &Coordinator{
		transactions: make(map[common.TransactionID]*common.Transaction),
	}
}

// BeginTransaction starts a new transaction
func (c *Coordinator) BeginTransaction(producerID string, timeout time.Duration) (*common.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	txID := common.TransactionID(fmt.Sprintf("tx-%d", time.Now().UnixNano()))
	tx := common.NewTransaction(txID, producerID, timeout)
	c.transactions[tx.ID] = tx

	return tx, nil
}

// AddPartitionsToTransaction adds partitions to an existing transaction
func (c *Coordinator) AddPartitionsToTransaction(
	txID common.TransactionID,
	partitions []common.TopicPartition,
) (*common.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tx, exists := c.transactions[txID]
	if !exists {
		return nil, ErrTransactionNotFound
	}

	tx.UpdateState(common.TransactionStateBegin)
	for _, p := range partitions {
		tx.AddPartition(p.Topic, p.Partition)
	}

	return tx, nil
}

// PrepareTransaction prepares a transaction for commit
func (c *Coordinator) PrepareTransaction(txID common.TransactionID) (*common.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tx, exists := c.transactions[txID]
	if !exists {
		return nil, ErrTransactionNotFound
	}

	if tx.State != common.TransactionStateBegin {
		return nil, fmt.Errorf("%w: cannot prepare transaction in state %s", 
			ErrInvalidTransactionState, tx.State)
	}

	tx.UpdateState(common.TransactionStatePrepared)
	return tx, nil
}

// CommitTransaction commits a prepared transaction
func (c *Coordinator) CommitTransaction(txID common.TransactionID) (*common.Transaction, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	tx, exists := c.transactions[txID]
	if !exists {
		return nil, ErrTransactionNotFound
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
		return nil, ErrTransactionNotFound
	}

	tx.UpdateState(common.TransactionStateAborted)
	return tx, nil
}

// GetTransaction returns a transaction by ID
func (c *Coordinator) GetTransaction(txID common.TransactionID) (*common.Transaction, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tx, exists := c.transactions[txID]
	if !exists {
		return nil, ErrTransactionNotFound
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
