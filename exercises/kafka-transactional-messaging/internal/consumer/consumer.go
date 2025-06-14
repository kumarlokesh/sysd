package consumer

import (
	"errors"
	"fmt"
	"sync"

	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/common"
)

// Consumer represents a transactional message consumer
type Consumer struct {
	groupID    string
	messageLog *common.MessageLog
	offsets    map[common.TopicPartition]common.Offset
	offsetsMux sync.RWMutex
}

// NewConsumer creates a new transactional consumer
func NewConsumer(groupID string, messageLog *common.MessageLog) *Consumer {
	return &Consumer{
		groupID:    groupID,
		messageLog: messageLog,
		offsets:    make(map[common.TopicPartition]common.Offset),
	}
}

// Subscribe sets the consumer to read from the specified topic and partition
func (c *Consumer) Subscribe(topic common.Topic, partition common.Partition) error {
	tp := common.TopicPartition{Topic: topic, Partition: partition}
	c.offsetsMux.Lock()
	defer c.offsetsMux.Unlock()

	// Initialize offset to 0 if not set
	if _, exists := c.offsets[tp]; !exists {
		c.offsets[tp] = 0
	}

	return nil
}

// Poll fetches messages from the subscribed partitions
func (c *Consumer) Poll(maxMessages int) ([]*common.Message, error) {
	c.offsetsMux.RLock()
	defer c.offsetsMux.RUnlock()

	var messages []*common.Message

	// Process each subscribed partition
	for tp, offset := range c.offsets {
		// Get all messages (including transaction markers) from the current offset
		// Use a larger batch size to ensure we get enough messages after filtering
		batchSize := maxMessages * 10
		entries, err := c.messageLog.GetMessages(
			tp.Topic,
			tp.Partition,
			offset,
			batchSize,
		)

		if err != nil {
			return nil, fmt.Errorf("error fetching messages from %s: %w", tp, err)
		}

		// If no entries, continue to next partition
		if len(entries) == 0 {
			continue
		}

		// Track the highest offset we've seen
		lastOffset := entries[len(entries)-1].Offset + 1

		// Process entries to filter out messages from aborted transactions
		txStates := make(map[common.TransactionID]common.TransactionState)
		var pendingMessages []*common.MessageLogEntry

		for _, entry := range entries {
			// Update transaction state if this is a marker
			if entry.IsMarker {
				txStates[entry.TxID] = entry.TxState
				// Process any pending messages for this transaction
				for _, pending := range pendingMessages {
					if pending.TxID == entry.TxID && entry.TxState == common.TransactionStateCommitted {
						messages = append(messages, pending.Message)
					}
				}
				// Clear processed pending messages
				var newPending []*common.MessageLogEntry
				for _, pending := range pendingMessages {
					if pending.TxID != entry.TxID {
						newPending = append(newPending, pending)
					}
				}
				pendingMessages = newPending
			} else {
				// For regular messages, check if we know the transaction state
				if state, exists := txStates[entry.TxID]; exists {
					if state == common.TransactionStateCommitted {
						messages = append(messages, entry.Message)
					}
				} else {
					// If we don't know the state yet, queue it
					pendingMessages = append(pendingMessages, entry)
				}
			}

			// Stop if we've collected enough messages
			if len(messages) >= maxMessages {
				break
			}
		}

		// Update the offset to the last processed entry
		c.offsets[tp] = lastOffset

		// If we've collected enough messages, stop processing partitions
		if len(messages) >= maxMessages {
			break
		}
	}

	// Return only the requested number of messages
	if len(messages) > maxMessages {
		messages = messages[:maxMessages]
	}

	return messages, nil
}

// CommitOffsets commits the current offsets for all subscribed partitions
func (c *Consumer) CommitOffsets() (map[common.TopicPartition]common.Offset, error) {
	c.offsetsMux.RLock()
	defer c.offsetsMux.RUnlock()

	// In a real implementation, this would persist the offsets
	// For this PoC, we just return the current offsets
	offsets := make(map[common.TopicPartition]common.Offset, len(c.offsets))
	for tp, offset := range c.offsets {
		offsets[tp] = offset
	}

	return offsets, nil
}

// Seek sets the offset for a specific partition
func (c *Consumer) Seek(topic common.Topic, partition common.Partition, offset common.Offset) error {
	tp := common.TopicPartition{Topic: topic, Partition: partition}

	// Verify the offset is valid
	latestOffset, err := c.messageLog.GetLatestOffset(topic, partition)
	if err != nil {
		return fmt.Errorf("failed to get latest offset: %w", err)
	}

	if offset < 0 || offset > latestOffset {
		return fmt.Errorf("offset %d is out of range [0, %d]", offset, latestOffset)
	}

	c.offsetsMux.Lock()
	defer c.offsetsMux.Unlock()

	c.offsets[tp] = offset
	return nil
}

// GetCommittedOffset returns the current committed offset for a partition
func (c *Consumer) GetCommittedOffset(topic common.Topic, partition common.Partition) (common.Offset, error) {
	tp := common.TopicPartition{Topic: topic, Partition: partition}

	c.offsetsMux.RLock()
	defer c.offsetsMux.RUnlock()

	offset, exists := c.offsets[tp]
	if !exists {
		return 0, errors.New("partition not subscribed")
	}

	return offset, nil
}

// Close releases any resources used by the consumer
func (c *Consumer) Close() error {
	// In a real implementation, this would clean up any resources
	return nil
}
