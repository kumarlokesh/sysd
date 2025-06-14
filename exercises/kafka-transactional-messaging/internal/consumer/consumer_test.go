package consumer_test

import (
	"testing"

	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/common"
	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/consumer"
	"github.com/stretchr/testify/assert"
)

func TestConsumer_SubscribeAndPoll(t *testing.T) {
	messageLog := common.NewMessageLog()
	cons := consumer.NewConsumer("test-group", messageLog)

	// Add some messages to the log directly first
	topic := common.Topic("test-topic")
	partition := common.Partition(0)

	// Add messages from a committed transaction
	txID := common.TransactionID("tx1")
	msg1 := &common.Message{Key: []byte("key1"), Value: []byte("value1"), Topic: topic, Partition: partition}
	msg2 := &common.Message{Key: []byte("key2"), Value: []byte("value2"), Topic: topic, Partition: partition}

	// Add the messages and commit marker
	_, _ = messageLog.Append(topic, partition, msg1, txID)
	_, _ = messageLog.Append(topic, partition, msg2, txID)
	_ = messageLog.AddTransactionMarker(topic, partition, txID, common.TransactionStateCommitted)

	// Now subscribe after adding messages
	err := cons.Subscribe(topic, partition)
	assert.NoError(t, err)

	// Poll for messages - should get both messages
	messages, err := cons.Poll(10)
	assert.NoError(t, err)
	assert.Len(t, messages, 2, "should get 2 messages from committed transaction")

	// Verify message contents
	if len(messages) >= 2 {
		assert.Equal(t, "key1", string(messages[0].Key))
		assert.Equal(t, "value1", string(messages[0].Value))
		assert.Equal(t, "key2", string(messages[1].Key))
		assert.Equal(t, "value2", string(messages[1].Value))
	}
}

func TestConsumer_FiltersAbortedTransactions(t *testing.T) {
	messageLog := common.NewMessageLog()
	consumer := consumer.NewConsumer("test-group", messageLog)

	// Subscribe to a topic
	_ = consumer.Subscribe("test-topic", 0)

	// Add messages from an aborted transaction
	topic := common.Topic("test-topic")
	partition := common.Partition(0)
	txID := common.TransactionID("tx-aborted")

	_, _ = messageLog.Append(topic, partition,
		&common.Message{Key: []byte("key1"), Value: []byte("aborted1"), Topic: topic, Partition: partition},
		txID)
	_ = messageLog.AddTransactionMarker(topic, partition, txID, common.TransactionStateAborted)

	// Add messages from a committed transaction
	txID2 := common.TransactionID("tx-committed")
	_, _ = messageLog.Append(topic, partition,
		&common.Message{Key: []byte("key2"), Value: []byte("committed1"), Topic: topic, Partition: partition},
		txID2)
	_ = messageLog.AddTransactionMarker(topic, partition, txID2, common.TransactionStateCommitted)

	// Poll for messages - should only get the committed message
	messages, err := consumer.Poll(10)
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "committed1", string(messages[0].Value))
}

func TestConsumer_SeeksToOffset(t *testing.T) {
	messageLog := common.NewMessageLog()
	consumer := consumer.NewConsumer("test-group", messageLog)

	// Add some messages to the log
	topic := common.Topic("test-topic")
	partition := common.Partition(0)
	txID := common.TransactionID("tx1")

	// Add 3 messages
	for i := 0; i < 3; i++ {
		_, _ = messageLog.Append(topic, partition,
			&common.Message{
				Key:       []byte("key"),
				Value:     []byte{byte('0' + i)},
				Topic:     topic,
				Partition: partition,
			},
			txID)
	}
	_ = messageLog.AddTransactionMarker(topic, partition, txID, common.TransactionStateCommitted)

	// Subscribe and seek to offset 1
	_ = consumer.Subscribe(topic, partition)
	err := consumer.Seek(topic, partition, 1)
	assert.NoError(t, err)

	// Should only get messages from offset 1 onwards
	messages, err := consumer.Poll(10)
	assert.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "1", string(messages[0].Value))
	assert.Equal(t, "2", string(messages[1].Value))
}
