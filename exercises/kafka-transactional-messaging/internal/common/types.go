package common

import (
	"fmt"
	"time"
)

// TransactionID is a unique identifier for a transaction
type TransactionID string

// Partition is a topic partition identifier
type Partition int32

// Offset represents a message offset within a partition
type Offset int64

// Topic represents a message topic
type Topic string

// TopicPartition uniquely identifies a partition within a topic
type TopicPartition struct {
	Topic     Topic
	Partition Partition
}

// String returns a string representation of TopicPartition
func (tp TopicPartition) String() string {
	return fmt.Sprintf("%s-%d", tp.Topic, tp.Partition)
}

// Message represents a message in the system
type Message struct {
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Topic     Topic
	Partition Partition
	Offset    Offset
}

// TransactionState represents the state of a transaction
type TransactionState string

const (
	// TransactionStateUnknown is the initial state
	TransactionStateUnknown TransactionState = "UNKNOWN"
	// TransactionStateBegin indicates a transaction has begun
	TransactionStateBegin TransactionState = "BEGIN"
	// TransactionStatePrepared indicates a transaction is prepared to commit
	TransactionStatePrepared TransactionState = "PREPARED"
	// TransactionStateCommitted indicates a transaction is committed
	TransactionStateCommitted TransactionState = "COMMITTED"
	// TransactionStateAborted indicates a transaction is aborted
	TransactionStateAborted TransactionState = "ABORTED"
)

// Transaction represents a transaction in the system
type Transaction struct {
	ID             TransactionID
	State          TransactionState
	ProducerID     string
	Timeout        time.Duration
	Partitions     []TopicPartition
	StartTimestamp time.Time
	LastUpdated    time.Time
}

// NewTransaction creates a new transaction with the given ID and producer ID
func NewTransaction(id TransactionID, producerID string, timeout time.Duration) *Transaction {
	now := time.Now()
	return &Transaction{
		ID:             id,
		State:          TransactionStateBegin,
		ProducerID:     producerID,
		Timeout:        timeout,
		Partitions:     make([]TopicPartition, 0),
		StartTimestamp: now,
		LastUpdated:    now,
	}
}

// AddPartition adds a partition to the transaction
func (t *Transaction) AddPartition(topic Topic, partition Partition) {
	t.Partitions = append(t.Partitions, TopicPartition{Topic: topic, Partition: partition})
	t.LastUpdated = time.Now()
}

// UpdateState updates the transaction state
func (t *Transaction) UpdateState(state TransactionState) {
	t.State = state
	t.LastUpdated = time.Now()
}

// IsExpired checks if the transaction has expired
func (t *Transaction) IsExpired() bool {
	return time.Since(t.StartTimestamp) > t.Timeout
}
