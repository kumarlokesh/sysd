package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// txManager implements the TxManager interface
type txManager struct {
	mu          sync.Mutex
	txStateFile string
}

// NewTxManager creates a new transaction manager
func NewTxManager(dir string) TxManager {
	return &txManager{
		txStateFile: filepath.Join(dir, ".txstate"),
	}
}

// TxManager defines the interface for transaction management
type TxManager interface {
	Begin() (uint64, error)
	End(txID uint64, commit bool) error
	GetActiveTx() (uint64, bool, error)
}

// txState represents the state of a transaction
type txState struct {
	Active bool   `json:"active"`
	TxID   uint64 `json:"tx_id"`
}

// Begin starts a new transaction
func (m *txManager) Begin() (uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Read current state
	state, err := m.readState()
	if err != nil {
		return 0, fmt.Errorf("failed to read transaction state: %w", err)
	}

	if state.Active {
		return 0, fmt.Errorf("transaction %d is already active", state.TxID)
	}

	// Generate new transaction ID
	state.TxID++
	state.Active = true

	// Save new state
	if err := m.writeState(state); err != nil {
		return 0, fmt.Errorf("failed to write transaction state: %w", err)
	}

	return state.TxID, nil
}

// End ends the current transaction
func (m *txManager) End(txID uint64, commit bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Read current state
	state, err := m.readState()
	if err != nil {
		return fmt.Errorf("failed to read transaction state: %w", err)
	}

	if !state.Active || state.TxID != txID {
		return fmt.Errorf("no active transaction with ID %d", txID)
	}

	// Update state
	state.Active = false
	if !commit {
		// On abort, don't increment the transaction ID
		state.TxID--
	}

	// Save state
	return m.writeState(state)
}

// GetActiveTx returns the currently active transaction ID, if any
func (m *txManager) GetActiveTx() (uint64, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.readState()
	if err != nil {
		return 0, false, fmt.Errorf("failed to read transaction state: %w", err)
	}

	if !state.Active {
		return 0, false, nil
	}

	return state.TxID, true, nil
}

// readState reads the transaction state from disk
func (m *txManager) readState() (*txState, error) {
	state := &txState{Active: false, TxID: 0}

	data, err := os.ReadFile(m.txStateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction state: %w", err)
	}

	return state, nil
}

// writeState writes the transaction state to disk
func (m *txManager) writeState(state *txState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction state: %w", err)
	}

	tmpFile := m.txStateFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write transaction state: %w", err)
	}

	return os.Rename(tmpFile, m.txStateFile)
}
