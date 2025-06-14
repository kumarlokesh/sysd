package producer

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/common"
	"github.com/kumarlokesh/sysd/exercises/kafka-transactional-messaging/internal/coordinator"
	"slices"
)

// Package-level error variables
var (
	ErrTransactionInProgress = errors.New("transaction already in progress")
	ErrNoActiveTransaction   = errors.New("no active transaction")
	ErrMessageLogFailure     = errors.New("failed to append to message log")
	ErrPartitionAddFailed    = errors.New("failed to add partition to transaction")
)

// Producer represents a transactional message producer
type Producer struct {
	producerID   string
	coordinator  *coordinator.Coordinator
	messageLog   *common.MessageLog
	currentTx    *common.Transaction
	currentTxMux sync.Mutex
}

// NewProducer creates a new transactional producer
func NewProducer(producerID string, coordinator *coordinator.Coordinator, messageLog *common.MessageLog) *Producer {
	return &Producer{
		producerID:  producerID,
		coordinator: coordinator,
		messageLog:  messageLog,
	}
}

// BeginTransaction starts a new transaction
func (p *Producer) BeginTransaction(timeout time.Duration) error {
	p.currentTxMux.Lock()
	defer p.currentTxMux.Unlock()

	if p.currentTx != nil {
		return ErrTransactionInProgress
	}

	tx, err := p.coordinator.BeginTransaction(p.producerID, timeout)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	p.currentTx = tx
	return nil
}

// Send sends a message within the current transaction
func (p *Producer) Send(topic common.Topic, partition common.Partition, key, value []byte) (common.Offset, error) {
	p.currentTxMux.Lock()
	defer p.currentTxMux.Unlock()

	if p.currentTx == nil {
		return 0, ErrNoActiveTransaction
	}

	// Add partition to transaction if not already added
	tp := common.TopicPartition{Topic: topic, Partition: partition}
	found := slices.Contains(p.currentTx.Partitions, tp)

	if !found {
		_, err := p.coordinator.AddPartitionsToTransaction(p.currentTx.ID, []common.TopicPartition{tp})
		if err != nil {
			return 0, fmt.Errorf("%w: %w", ErrPartitionAddFailed, err)
		}
	}

	// Create and append the message
	msg := &common.Message{
		Key:       key,
		Value:     value,
		Headers:   make(map[string]string),
		Topic:     topic,
		Partition: partition,
	}

	offset, err := p.messageLog.Append(topic, partition, msg, p.currentTx.ID)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrMessageLogFailure, err)
	}

	return offset, nil
}

// CommitTransaction commits the current transaction
func (p *Producer) CommitTransaction() error {
	p.currentTxMux.Lock()
	defer p.currentTxMux.Unlock()

	if p.currentTx == nil {
		return errors.New("no transaction in progress")
	}

	// Prepare the transaction
	tx, err := p.coordinator.PrepareTransaction(p.currentTx.ID)
	if err != nil {
		return fmt.Errorf("failed to prepare transaction: %w", err)
	}

	// Add commit markers to all participating partitions
	for _, tp := range tx.Partitions {
		err := p.messageLog.AddTransactionMarker(tp.Topic, tp.Partition, tx.ID, common.TransactionStateCommitted)
		if err != nil {
			return fmt.Errorf("failed to add commit marker: %w", err)
		}
	}

	// Commit the transaction
	_, err = p.coordinator.CommitTransaction(tx.ID)
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	p.currentTx = nil
	return nil
}

// AbortTransaction aborts the current transaction
func (p *Producer) AbortTransaction() error {
	p.currentTxMux.Lock()
	defer p.currentTxMux.Unlock()

	if p.currentTx == nil {
		return errors.New("no transaction in progress")
	}

	// Add abort markers to all participating partitions
	for _, tp := range p.currentTx.Partitions {
		err := p.messageLog.AddTransactionMarker(tp.Topic, tp.Partition, p.currentTx.ID, common.TransactionStateAborted)
		if err != nil {
			return fmt.Errorf("failed to add abort marker: %w", err)
		}
	}

	// Abort the transaction
	_, err := p.coordinator.AbortTransaction(p.currentTx.ID)
	if err != nil {
		return fmt.Errorf("failed to abort transaction: %w", err)
	}

	p.currentTx = nil
	return nil
}

// CurrentTransaction returns the current transaction or nil if none is in progress
func (p *Producer) CurrentTransaction() *common.Transaction {
	p.currentTxMux.Lock()
	defer p.currentTxMux.Unlock()
	return p.currentTx
}
