package producer_test

import (
	"testing"
	"time"

	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/common"
	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/coordinator"
	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/producer"
	"github.com/stretchr/testify/assert"
)

func TestProducer_SendAndCommit(t *testing.T) {
	coord := coordinator.NewCoordinator()
	messageLog := common.NewMessageLog()
	prod := producer.NewProducer("test-producer", coord, messageLog)

	// Begin transaction
	err := prod.BeginTransaction(30 * time.Second)
	assert.NoError(t, err)

	// Send messages
	offset1, err := prod.Send("test-topic", 0, []byte("key1"), []byte("value1"))
	assert.NoError(t, err)
	assert.Equal(t, common.Offset(0), offset1)

	offset2, err := prod.Send("test-topic", 0, []byte("key2"), []byte("value2"))
	assert.NoError(t, err)
	assert.Equal(t, common.Offset(1), offset2)

	// Commit transaction
	err = prod.CommitTransaction()
	assert.NoError(t, err)

	// Verify messages are in the log
	entries, err := messageLog.GetMessages("test-topic", 0, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, entries, 3) // 2 messages + 1 commit marker

	// Verify the commit marker
	assert.True(t, entries[2].IsMarker)
	assert.Equal(t, common.TransactionStateCommitted, entries[2].TxState)
}

func TestProducer_AbortTransaction(t *testing.T) {
	coord := coordinator.NewCoordinator()
	messageLog := common.NewMessageLog()
	prod := producer.NewProducer("test-producer", coord, messageLog)

	// Begin transaction
	err := prod.BeginTransaction(30 * time.Second)
	assert.NoError(t, err)

	// Send a message
	_, err = prod.Send("test-topic", 0, []byte("key1"), []byte("value1"))
	assert.NoError(t, err)

	// Abort transaction
	err = prod.AbortTransaction()
	assert.NoError(t, err)

	// Verify abort marker is in the log
	entries, err := messageLog.GetMessages("test-topic", 0, 0, 10)
	assert.NoError(t, err)
	assert.Len(t, entries, 2) // 1 message + 1 abort marker

	// Verify the abort marker
	assert.True(t, entries[1].IsMarker)
	assert.Equal(t, common.TransactionStateAborted, entries[1].TxState)
}

func TestProducer_NoTransaction(t *testing.T) {
	coord := coordinator.NewCoordinator()
	messageLog := common.NewMessageLog()
	prod := producer.NewProducer("test-producer", coord, messageLog)

	// Try to send without starting a transaction
	_, err := prod.Send("test-topic", 0, []byte("key1"), []byte("value1"))
	assert.Error(t, err)
}

func TestProducer_DoubleBegin(t *testing.T) {
	coord := coordinator.NewCoordinator()
	messageLog := common.NewMessageLog()
	prod := producer.NewProducer("test-producer", coord, messageLog)

	// Begin first transaction
	err := prod.BeginTransaction(30 * time.Second)
	assert.NoError(t, err)

	// Try to begin another transaction
	err = prod.BeginTransaction(30 * time.Second)
	assert.Error(t, err)
}
