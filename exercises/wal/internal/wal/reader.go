package wal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

var (
	// ErrCorruptLog is returned when the log is corrupted.
	ErrCorruptLog = errors.New("log is corrupted")
	// ErrUnexpectedEOF is returned when we reach an unexpected end of file.
	ErrUnexpectedEOF = errors.New("unexpected end of file")
)

// LogReader reads records from the WAL.
type LogReader struct {
	dir      string   // Directory containing WAL segments
	segments []string // Sorted list of segment files
	current  int      // Current segment index
	file     *os.File // Current segment file
	offset   int64    // Current offset in the segment
}

// NewLogReader creates a new LogReader for the given directory.
func NewLogReader(dir string) (*LogReader, error) {
	// List all segment files
	files, err := filepath.Glob(filepath.Join(dir, "*.wal"))
	if err != nil {
		return nil, fmt.Errorf("failed to list segment files: %w", err)
	}

	// Sort segments by ID (filename without extension)
	sort.Slice(files, func(i, j int) bool {
		iID, _ := strconv.ParseUint(strings.TrimSuffix(filepath.Base(files[i]), ".wal"), 10, 64)
		jID, _ := strconv.ParseUint(strings.TrimSuffix(filepath.Base(files[j]), ".wal"), 10, 64)
		return iID < jID
	})

	if len(files) == 0 {
		return &LogReader{dir: dir}, nil
	}

	// Open the first segment
	file, err := os.Open(files[0])
	if err != nil {
		return nil, fmt.Errorf("failed to open segment %s: %w", files[0], err)
	}

	return &LogReader{
		dir:      dir,
		segments: files,
		file:     file,
	}, nil
}

// Read reads the next record from the WAL.
// This is an alias for Next() to satisfy the io.Reader interface.
func (r *LogReader) Read() (*Record, error) {
	return r.Next()
}

// Reset resets the reader to the beginning of the log.
// This is similar to SeekToStart but returns an error to match the expected interface.
func (r *LogReader) Reset() error {
	return r.SeekToStart()
}

// Next reads the next record from the WAL.
func (r *LogReader) Next() (*Record, error) {
	// If we have no file open, try to open the next segment
	if r.file == nil {
		if r.current >= len(r.segments) {
			return nil, io.EOF
		}

		file, err := os.Open(r.segments[r.current])
		if err != nil {
			return nil, fmt.Errorf("failed to open segment %s: %w", r.segments[r.current], err)
		}

		r.file = file
		r.offset = 0
	}

	// Read the header
	header := make([]byte, HeaderSize)
	n, err := io.ReadFull(r.file, header)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		// End of current segment, try next one
		_ = r.file.Close()
		r.file = nil
		r.current++
		return r.Next()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read record header: %w", err)
	}
	if n != HeaderSize {
		return nil, ErrCorruptLog
	}

	// Parse the header to get key and value lengths
	keyLen := binary.BigEndian.Uint16(header[18:20])
	valueLen := binary.BigEndian.Uint16(header[20:22])
	recordSize := int64(HeaderSize + int(keyLen) + int(valueLen))

	// Read the entire record
	buf := make([]byte, recordSize)
	copy(buf, header)

	if _, err := io.ReadFull(r.file, buf[HeaderSize:]); err != nil {
		return nil, fmt.Errorf("failed to read record data: %w", err)
	}

	// Decode the record
	record := &Record{}
	if err := record.Decode(buf); err != nil {
		return nil, fmt.Errorf("failed to decode record: %w", err)
	}

	r.offset += recordSize
	return record, nil
}

// Close closes the LogReader and any open segment files.
func (r *LogReader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// SeekToStart resets the reader to the beginning of the first segment.
func (r *LogReader) SeekToStart() error {
	if err := r.Close(); err != nil {
		return err
	}

	r.current = 0
	r.file = nil
	r.offset = 0

	if len(r.segments) > 0 {
		file, err := os.Open(r.segments[0])
		if err != nil {
			return fmt.Errorf("failed to open segment %s: %w", r.segments[0], err)
		}
		r.file = file
	}

	return nil
}
