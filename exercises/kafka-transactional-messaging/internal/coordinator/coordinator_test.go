package coordinator_test

import (
	"testing"
	"time"

	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/common"
	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/coordinator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	// AddPartitionsToTransaction now returns the list of added partitions
	addedParts, err := c.AddPartitionsToTransaction(tx.ID, partitions)
	assert.NoError(t, err)
	assert.Len(t, addedParts, 2)

	// Verify the transaction was updated with the new partitions
	tx, err = c.GetTransaction(tx.ID)
	assert.NoError(t, err)
	assert.Len(t, tx.Partitions, 2)
}

func TestCoordinator_TransactionLifecycle(t *testing.T) {
	c := coordinator.NewCoordinator()

	// Test BeginTransaction with invalid timeout
	_, err := c.BeginTransaction("prod1", 0)
	assert.ErrorIs(t, err, coordinator.ErrInvalidTimeout)

	// Begin a valid transaction
	tx, err := c.BeginTransaction("prod1", 30*time.Second)
	require.NoError(t, err)
	assert.Equal(t, common.TransactionStateBegin, tx.State)

	// Test AddPartitionsToTransaction with empty partitions
	_, err = c.AddPartitionsToTransaction(tx.ID, []common.TopicPartition{})
	assert.ErrorIs(t, err, coordinator.ErrNoPartitions)

	// Add partitions
	partitions := []common.TopicPartition{{Topic: "test-topic", Partition: 0}}
	addedParts, err := c.AddPartitionsToTransaction(tx.ID, partitions)
	assert.NoError(t, err)
	assert.Len(t, addedParts, 1)

	// Try to add the same partition again (should be idempotent)
	addedParts, err = c.AddPartitionsToTransaction(tx.ID, partitions)
	assert.NoError(t, err)
	assert.Len(t, addedParts, 0) // No new partitions added (idempotent)

	// Test PrepareTransaction with invalid state (should work from Begin state)
	tx, err = c.PrepareTransaction(tx.ID)
	assert.NoError(t, err)
	assert.Equal(t, common.TransactionStatePrepared, tx.State)

	// Test CommitTransaction with invalid state (should work from Prepared state)
	tx, err = c.CommitTransaction(tx.ID)
	assert.NoError(t, err)
	assert.Equal(t, common.TransactionStateCommitted, tx.State)
	tx, err = c.GetTransaction(tx.ID)
	assert.NoError(t, err)
	assert.Equal(t, common.TransactionStateCommitted, tx.State)
}

func TestCoordinator_AbortTransaction(t *testing.T) {
	c := coordinator.NewCoordinator()

	// Test aborting non-existent transaction
	_, err := c.AbortTransaction("nonexistent-tx")
	assert.ErrorIs(t, err, coordinator.ErrTransactionNotFound)

	// Begin a transaction
	tx, err := c.BeginTransaction("prod1", 30*time.Second)
	require.NoError(t, err)

	// Add some partitions
	partitions := []common.TopicPartition{{Topic: "test-topic", Partition: 0}}
	_, err = c.AddPartitionsToTransaction(tx.ID, partitions)
	require.NoError(t, err)

	// Test aborting from Begin state
	tx, err = c.AbortTransaction(tx.ID)
	assert.NoError(t, err)
	assert.Equal(t, common.TransactionStateAborted, tx.State)

	// Test aborting already aborted transaction
	_, err = c.AbortTransaction(tx.ID)
	assert.ErrorIs(t, err, coordinator.ErrInvalidTransactionState)

	// Test aborting committed transaction
	tx, err = c.BeginTransaction("prod1", 30*time.Second)
	require.NoError(t, err)
	_, err = c.PrepareTransaction(tx.ID)
	require.NoError(t, err)
	_, err = c.CommitTransaction(tx.ID)
	require.NoError(t, err)
	_, err = c.AbortTransaction(tx.ID)
	assert.ErrorIs(t, err, coordinator.ErrInvalidTransactionState)
}

func TestCoordinator_TransactionExpiration(t *testing.T) {
	c := coordinator.NewCoordinator()

	// Test with zero timeout (should fail)
	_, err := c.BeginTransaction("prod1", 0)
	assert.ErrorIs(t, err, coordinator.ErrInvalidTimeout)

	// Begin a transaction with a short timeout
	tx, err := c.BeginTransaction("prod1", 50*time.Millisecond)
	require.NoError(t, err)

	// Verify transaction exists before expiration
	tx, err = c.GetTransaction(tx.ID)
	require.NoError(t, err)
	require.NotNil(t, tx, "transaction should exist before expiration")

	// Wait for the transaction to expire
	time.Sleep(100 * time.Millisecond)

	// Try to get the expired transaction - should return not found
	_, err = c.GetTransaction(tx.ID)
	require.Error(t, err, "expected error for expired transaction")

	// Try to prepare the expired transaction - should return not found
	_, err = c.PrepareTransaction(tx.ID)
	require.Error(t, err, "expected error for expired transaction")

	// Try to add partitions to expired transaction - should return not found
	_, err = c.AddPartitionsToTransaction(tx.ID, []common.TopicPartition{{Topic: "test", Partition: 0}})
	require.Error(t, err, "expected error for expired transaction")

	// Test cleanup of expired transactions with staggered timeouts
	// Add multiple transactions with different timeouts
	tx1, err := c.BeginTransaction("prod1", 50*time.Millisecond)
	require.NoError(t, err)
	tx2, err := c.BeginTransaction("prod2", 100*time.Millisecond)
	require.NoError(t, err)
	tx3, err := c.BeginTransaction("prod3", 200*time.Millisecond)
	require.NoError(t, err)

	// Verify all transactions exist initially
	tx1Check, err := c.GetTransaction(tx1.ID)
	require.NoError(t, err, "tx1 should exist initially")
	require.NotNil(t, tx1Check, "tx1 should not be nil")

	tx2Check, err := c.GetTransaction(tx2.ID)
	require.NoError(t, err, "tx2 should exist initially")
	require.NotNil(t, tx2Check, "tx2 should not be nil")

	tx3Check, err := c.GetTransaction(tx3.ID)
	require.NoError(t, err, "tx3 should exist initially")
	require.NotNil(t, tx3Check, "tx3 should not be nil")

	// Wait for transactions to expire in stages and verify cleanup
	time.Sleep(75 * time.Millisecond) // tx1 should be expired
	_, err = c.GetTransaction(tx1.ID)
	require.Error(t, err, "tx1 should be expired and cleaned up")

	// tx2 and tx3 should still exist
	tx, err = c.GetTransaction(tx2.ID)
	require.NoError(t, err, "tx2 should still exist")
	require.NotNil(t, tx, "tx2 should not be nil")

	tx, err = c.GetTransaction(tx3.ID)
	require.NoError(t, err, "tx3 should still exist")
	require.NotNil(t, tx, "tx3 should not be nil")

	time.Sleep(50 * time.Millisecond) // tx2 should be expired now (125ms total)
	_, err = c.GetTransaction(tx2.ID)
	require.Error(t, err, "tx2 should be expired and cleaned up")

	// tx3 should still exist
	tx, err = c.GetTransaction(tx3.ID)
	require.NoError(t, err, "tx3 should still exist")
	require.NotNil(t, tx, "tx3 should not be nil")

	time.Sleep(100 * time.Millisecond) // tx3 should be expired now (225ms total)
	_, err = c.GetTransaction(tx3.ID)
	require.Error(t, err, "tx3 should be expired and cleaned up")
}
