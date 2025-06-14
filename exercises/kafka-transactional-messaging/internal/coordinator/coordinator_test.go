package coordinator_test

import (
	"testing"
	"time"

	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/coordinator"
	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestCoordinator_BeginTransaction(t *testing.T) {
	c := coordinator.NewCoordinator()
	tx, err := c.BeginTransaction("prod1", 30*time.Second)

	assert.NoError(t, err)
	assert.NotEmpty(t, tx.ID)
	assert.Equal(t, common.TransactionStateBegin, tx.State)
	assert.Equal(t, "prod1", tx.ProducerID)
}

func TestCoordinator_AddPartitionsToTransaction(t *testing.T) {
	c := coordinator.NewCoordinator()
	tx, _ := c.BeginTransaction("prod1", 30*time.Second)

	partitions := []common.TopicPartition{
		{Topic: "test-topic", Partition: 0},
		{Topic: "test-topic", Partition: 1},
	}

	tx, err := c.AddPartitionsToTransaction(tx.ID, partitions)
	assert.NoError(t, err)
	assert.Len(t, tx.Partitions, 2)
}

func TestCoordinator_TransactionLifecycle(t *testing.T) {
	c := coordinator.NewCoordinator()
	tx, _ := c.BeginTransaction("prod1", 30*time.Second)

	// Add partitions
	partitions := []common.TopicPartition{{Topic: "test-topic", Partition: 0}}
	_, _ = c.AddPartitionsToTransaction(tx.ID, partitions)

	// Prepare
	tx, err := c.PrepareTransaction(tx.ID)
	assert.NoError(t, err)
	assert.Equal(t, common.TransactionStatePrepared, tx.State)

	// Commit
	tx, err = c.CommitTransaction(tx.ID)
	assert.NoError(t, err)
	assert.Equal(t, common.TransactionStateCommitted, tx.State)

	// Verify transaction is committed
	tx, err = c.GetTransaction(tx.ID)
	assert.NoError(t, err)
	assert.Equal(t, common.TransactionStateCommitted, tx.State)
}

func TestCoordinator_AbortTransaction(t *testing.T) {
	c := coordinator.NewCoordinator()
	tx, _ := c.BeginTransaction("prod1", 30*time.Second)

	// Add partitions
	partitions := []common.TopicPartition{{Topic: "test-topic", Partition: 0}}
	_, _ = c.AddPartitionsToTransaction(tx.ID, partitions)

	// Abort
	tx, err := c.AbortTransaction(tx.ID)
	assert.NoError(t, err)
	assert.Equal(t, common.TransactionStateAborted, tx.State)
}

func TestCoordinator_TransactionExpiration(t *testing.T) {
	c := coordinator.NewCoordinator()
	tx, _ := c.BeginTransaction("prod1", 100*time.Millisecond)

	// Wait for transaction to expire
	time.Sleep(150 * time.Millisecond)

	expired := c.CleanupExpiredTransactions()
	assert.Len(t, expired, 1)
	assert.Equal(t, tx.ID, expired[0])

	// Verify transaction was aborted
	tx, err := c.GetTransaction(tx.ID)
	assert.NoError(t, err)
	assert.Equal(t, common.TransactionStateAborted, tx.State)
}
