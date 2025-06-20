package wal

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
)

// RecordType represents the type of a log record.
type RecordType byte

const (
	// RecordTypeWrite represents a write operation.
	RecordTypeWrite RecordType = iota + 1
	// RecordTypeCommit represents a commit operation.
	RecordTypeCommit
	// RecordTypeAbort represents an abort operation.
	RecordTypeAbort
	// RecordTypeCheckpoint represents a checkpoint record.
	RecordTypeCheckpoint
	// RecordTypeTxnBegin marks the beginning of a transaction
	RecordTypeTxnBegin
	// RecordTypeTxnCommit marks the successful end of a transaction
	RecordTypeTxnCommit
	// RecordTypeTxnRollback marks the unsuccessful end of a transaction
	RecordTypeTxnRollback
)

const (
	// HeaderSize is the size of the record header in bytes.
	// LSN (8) + TxID (8) + Type (1) + Flags (1) + KeyLen (2) + ValueLen (2) + Checksum (4) = 26 bytes
	HeaderSize = 26
	// LSNSize is the size of the Log Sequence Number in bytes.
	LSNSize = 8
	// TxIDSize is the size of the Transaction ID in bytes.
	TxIDSize = 8
)

// Header represents the header of a log record.
type Header struct {
	LSN      uint64     // Log Sequence Number (8 bytes)
	TxID     uint64     // Transaction ID (8 bytes)
	Type     RecordType // Record type (1 byte)
	Flags    byte       // Flags (1 byte)
	KeyLen   uint16     // Length of the key (2 bytes)
	ValueLen uint16     // Length of the value (2 bytes)
	Checksum uint32     // CRC32 checksum (4 bytes)
}

// Record represents a single log record.
type Record struct {
	Header
	Key   []byte
	Value []byte
}

// Encode encodes the record into a byte slice.
func (r *Record) Encode() ([]byte, error) {
	// Calculate total size
	totalSize := HeaderSize + len(r.Key) + len(r.Value)
	buf := make([]byte, totalSize)

	// Encode header (except checksum)
	offset := 0
	binary.BigEndian.PutUint64(buf[offset:], r.LSN)
	offset += 8
	binary.BigEndian.PutUint64(buf[offset:], r.TxID)
	offset += 8
	buf[offset] = byte(r.Type)
	offset++
	buf[offset] = r.Flags
	offset++
	binary.BigEndian.PutUint16(buf[offset:], uint16(len(r.Key)))
	offset += 2
	binary.BigEndian.PutUint16(buf[offset:], uint16(len(r.Value)))
	offset += 2
	// Leave space for checksum (4 bytes)
	checksumPos := offset
	offset += 4

	// Copy key and value
	copy(buf[offset:], r.Key)
	offset += len(r.Key)
	copy(buf[offset:], r.Value)

	// Calculate and write checksum (over everything after the header)
	r.Checksum = crc32.ChecksumIEEE(buf[HeaderSize:])
	binary.BigEndian.PutUint32(buf[checksumPos:], r.Checksum)

	return buf, nil
}

// Decode decodes a byte slice into a Record.
func (r *Record) Decode(data []byte) error {
	if len(data) < HeaderSize {
		return io.ErrShortBuffer
	}

	// Decode header (except checksum)
	offset := 0
	r.LSN = binary.BigEndian.Uint64(data[offset:])
	offset += 8
	r.TxID = binary.BigEndian.Uint64(data[offset:])
	offset += 8
	r.Type = RecordType(data[offset])
	offset++
	r.Flags = data[offset]
	offset++
	keyLen := binary.BigEndian.Uint16(data[offset:])
	offset += 2
	valueLen := binary.BigEndian.Uint16(data[offset:])
	offset += 2
	checksum := binary.BigEndian.Uint32(data[offset:])

	// Verify data length
	expectedLen := HeaderSize + int(keyLen) + int(valueLen)
	if len(data) < expectedLen {
		return io.ErrUnexpectedEOF
	}

	// Verify checksum
	actualChecksum := crc32.ChecksumIEEE(data[HeaderSize:expectedLen])
	if actualChecksum != checksum {
		return errors.New("checksum mismatch")
	}

	// Copy key and value
	r.Key = make([]byte, keyLen)
	copy(r.Key, data[HeaderSize:HeaderSize+int(keyLen)])

	r.Value = make([]byte, valueLen)
	copy(r.Value, data[HeaderSize+int(keyLen):expectedLen])

	// Set the checksum in the header
	r.Checksum = checksum

	return nil
}

// NewWriteRecord creates a new write record.
func NewWriteRecord(lsn, txID uint64, key, value []byte) *Record {
	return &Record{
		Header: Header{
			LSN:      lsn,
			TxID:     txID,
			Type:     RecordTypeWrite,
			KeyLen:   uint16(len(key)),
			ValueLen: uint16(len(value)),
		},
		Key:   key,
		Value: value,
	}
}

// NewCommitRecord creates a new commit record.
func NewCommitRecord(lsn, txID uint64) *Record {
	return &Record{
		Header: Header{
			LSN:  lsn,
			TxID: txID,
			Type: RecordTypeCommit,
		},
	}
}

// CommitTxnRecord creates a new transaction commit record
func CommitTxnRecord(txID, lsn uint64) *Record {
	return &Record{
		Header: Header{
			LSN:  lsn,
			TxID: txID,
			Type: RecordTypeTxnCommit,
		},
	}
}

// RollbackTxnRecord creates a new transaction rollback record
func RollbackTxnRecord(txID, lsn uint64) *Record {
	return &Record{
		Header: Header{
			LSN:  lsn,
			TxID: txID,
			Type: RecordTypeTxnRollback,
		},
	}
}

// BeginTxnRecord creates a new transaction begin record
func BeginTxnRecord(txID uint64) *Record {
	return &Record{
		Header: Header{
			TxID: txID,
			Type: RecordTypeTxnBegin,
		},
	}
}

// NewAbortRecord creates a new abort record.
func NewAbortRecord(lsn, txID uint64) *Record {
	return &Record{
		Header: Header{
			LSN:  lsn,
			TxID: txID,
			Type: RecordTypeAbort,
		},
	}
}

// NewCheckpointRecord creates a new checkpoint record.
func NewCheckpointRecord(lsn uint64) *Record {
	return &Record{
		Header: Header{
			LSN:  lsn,
			Type: RecordTypeCheckpoint,
		},
	}
}
