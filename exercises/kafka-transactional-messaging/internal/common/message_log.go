package common

import (
	"errors"
	"sync"
	"time"
)

// MessageLogEntry represents an entry in the message log
type MessageLogEntry struct {
	Message   *Message
	TxID      TransactionID
	TxState   TransactionState
	Timestamp time.Time
	Offset    Offset
	IsMarker  bool
}

// MessageLog represents an in-memory message log
type MessageLog struct {
	partitions map[TopicPartition][]*MessageLogEntry
	offsets    map[TopicPartition]Offset
	mu         sync.RWMutex
}

// NewMessageLog creates a new message log
func NewMessageLog() *MessageLog {
	return &MessageLog{
		partitions: make(map[TopicPartition][]*MessageLogEntry),
		offsets:    make(map[TopicPartition]Offset),
	}
}

// Append adds a message to the log
func (l *MessageLog) Append(topic Topic, partition Partition, msg *Message, txID TransactionID) (Offset, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	tp := TopicPartition{Topic: topic, Partition: partition}
	if _, exists := l.partitions[tp]; !exists {
		l.partitions[tp] = make([]*MessageLogEntry, 0)
	}

	offset := l.offsets[tp]
	entry := &MessageLogEntry{
		Message:   msg,
		TxID:      txID,
		TxState:   TransactionStateBegin,
		Timestamp: time.Now(),
		Offset:    offset,
		IsMarker:  false,
	}

	l.partitions[tp] = append(l.partitions[tp], entry)
	l.offsets[tp] = offset + 1

	return offset, nil
}

// AddTransactionMarker adds a transaction marker to the log
func (l *MessageLog) AddTransactionMarker(
	topic Topic,
	partition Partition,
	txID TransactionID,
	state TransactionState,
) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	tp := TopicPartition{Topic: topic, Partition: partition}
	if _, exists := l.partitions[tp]; !exists {
		return errors.New("partition not found")
	}

	offset := l.offsets[tp]
	entry := &MessageLogEntry{
		TxID:      txID,
		TxState:   state,
		Timestamp: time.Now(),
		Offset:    offset,
		IsMarker:  true,
	}

	l.partitions[tp] = append(l.partitions[tp], entry)
	l.offsets[tp] = offset + 1

	return nil
}

// GetMessages returns messages from the specified partition and offset
func (l *MessageLog) GetMessages(
	topic Topic,
	partition Partition,
	offset Offset,
	maxMessages int,
) ([]*MessageLogEntry, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	tp := TopicPartition{Topic: topic, Partition: partition}
	entries, exists := l.partitions[tp]
	if !exists {
		return nil, errors.New("partition not found")
	}

	// If offset is beyond the last entry, return empty slice
	if offset >= Offset(len(entries)) {
		return []*MessageLogEntry{}, nil
	}

	// Calculate end index, ensuring we don't go beyond the slice bounds
	end := int(offset) + maxMessages
	if end > len(entries) {
		end = len(entries)
	}

	// Return a copy of the slice to prevent concurrent modification issues
	result := make([]*MessageLogEntry, end-int(offset))
	copy(result, entries[offset:end])

	return result, nil
}

// GetCommittedMessages returns only committed messages from the log
func (l *MessageLog) GetCommittedMessages(
	topic Topic,
	partition Partition,
	offset Offset,
	maxMessages int,
) ([]*Message, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	tp := TopicPartition{Topic: topic, Partition: partition}
	entries, exists := l.partitions[tp]
	if !exists {
		return nil, errors.New("partition not found")
	}

	var result []*Message
	txStates := make(map[TransactionID]TransactionState)

	for i := 0; i < len(entries) && len(result) < maxMessages; i++ {
		entry := entries[i]

		if entry.IsMarker {
			txStates[entry.TxID] = entry.TxState
		} else {
			if state, exists := txStates[entry.TxID]; exists && state == TransactionStateCommitted {
				result = append(result, entry.Message)
			}
		}
	}

	return result, nil
}

// GetLatestOffset returns the latest offset for a partition
func (l *MessageLog) GetLatestOffset(topic Topic, partition Partition) (Offset, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	tp := TopicPartition{Topic: topic, Partition: partition}
	offset, exists := l.offsets[tp]
	if !exists {
		return 0, errors.New("partition not found")
	}

	return offset, nil
}
