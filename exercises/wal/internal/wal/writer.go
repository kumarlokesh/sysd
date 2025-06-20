package wal

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	// ErrWALClosed is returned when attempting to write to a closed WAL.
	ErrWALClosed = errors.New("WAL is closed")
	// ErrSegmentFull is returned when the current segment is full.
	ErrSegmentFull = errors.New("segment is full")
)

const (
	// DefaultBufferSize is the default size of the write buffer.
	DefaultBufferSize = 4 * 1024 // 4KB
	// DefaultSegmentSize is the default size of each segment file.
	DefaultSegmentSize = 1 << 30 // 1GB
)

// LogWriter writes records to the WAL.
type LogWriter struct {
	mu          sync.Mutex
	dir         string         // Directory where WAL segments are stored
	file        *os.File       // Current segment file
	segmentID   uint64         // Current segment ID
	offset      int64          // Current offset in the segment
	segmentSize int64          // Maximum size of each segment file
	buf         *bytes.Buffer  // In-memory buffer for batching writes
	bufMu       sync.Mutex     // Protects the buffer
	sync        bool           // Whether to sync after each write
	closed      bool           // Whether the writer is closed
	flushTicker *time.Ticker   // Ticker for periodic flushes
	stopCh      chan struct{}  // Channel to stop background flusher
	wg          sync.WaitGroup // Wait group for background flusher
}

// NewLogWriter creates a new LogWriter.
func NewLogWriter(dir string, config *Config) (*LogWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Set default values if not specified
	bufferSize := config.BufferSize
	if bufferSize <= 0 {
		bufferSize = DefaultBufferSize
	}

	segmentSize := config.SegmentSize
	if segmentSize <= 0 {
		segmentSize = DefaultSegmentSize
	}

	flushInterval := config.FlushInterval
	if flushInterval <= 0 {
		flushInterval = time.Second // Default to 1 second
	}

	if segmentSize <= 0 {
		segmentSize = DefaultSegmentSize
	}

	w := &LogWriter{
		dir:         dir,
		sync:        config.Sync,
		segmentSize: segmentSize,
		buf:         bytes.NewBuffer(make([]byte, 0, bufferSize)),
		stopCh:      make(chan struct{}),
		flushTicker: time.NewTicker(flushInterval),
	}

	w.wg.Add(1)
	go w.backgroundFlusher()

	if err := w.openOrCreateSegment(); err != nil {
		w.Stop()
		return nil, err
	}

	return w, nil
}

// Write writes a record to the WAL.
func (w *LogWriter) Write(record *Record) (uint64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return 0, ErrWALClosed
	}

	data, err := record.Encode()
	if err != nil {
		return 0, err
	}

	// Check if we need to rotate the segment
	if w.offset+int64(len(data)) > w.segmentSize {
		if err := w.rotateSegment(); err != nil {
			return 0, fmt.Errorf("failed to rotate segment: %w", err)
		}
	}

	w.bufMu.Lock()
	_, err = w.buf.Write(data)
	w.bufMu.Unlock()
	if err != nil {
		return 0, fmt.Errorf("failed to write to buffer: %w", err)
	}

	if w.sync {
		if err := w.flushBuffer(); err != nil {
			return 0, fmt.Errorf("failed to flush buffer: %w", err)
		}
	}

	return record.LSN, nil
}

// Flush writes any buffered data to the underlying writer.
func (w *LogWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.flushBuffer(); err != nil {
		return fmt.Errorf("flush failed: %w", err)
	}
	return nil
}

// flushBuffer writes the buffered data to disk.
// Caller must hold w.mu
func (w *LogWriter) flushBuffer() error {
	if w.buf.Len() == 0 {
		return nil
	}

	n, err := w.file.Write(w.buf.Bytes())
	if err != nil {
		return err
	}

	w.offset += int64(n)

	w.buf.Reset()

	if w.sync {
		return w.file.Sync()
	}

	return nil
}

// backgroundFlusher periodically flushes the buffer to disk.
func (w *LogWriter) backgroundFlusher() {
	defer w.wg.Done()

	for {
		select {
		case <-w.stopCh:
			// Perform final flush before exiting
			_ = w.Flush()
			return

		case <-w.flushTicker.C:
			if w.mu.TryLock() {
				_ = w.flushBuffer()
				w.mu.Unlock()
			}
		}
	}
}

// Stop stops the background flusher and flushes any remaining data.
func (w *LogWriter) Stop() {
	if w.flushTicker != nil {
		w.flushTicker.Stop()
	}

	if w.stopCh != nil {
		close(w.stopCh)
		w.wg.Wait()
	}

	_ = w.Flush()
}

// Close closes the LogWriter.
func (w *LogWriter) Close() error {
	w.mu.Lock()

	if w.closed {
		w.mu.Unlock()
		return nil
	}

	w.closed = true

	// Stop the background flusher
	if w.flushTicker != nil {
		w.flushTicker.Stop()

		// Signal the background flusher to stop
		if w.stopCh != nil {
			close(w.stopCh)
		}

		// Release the lock while we wait for the background flusher to finish
		w.mu.Unlock()

		// Wait for background flusher to finish
		w.wg.Wait()

		// Re-acquire the lock for the rest of the cleanup
		w.mu.Lock()
	}

	// Flush any remaining data in the buffer
	if err := w.flushBuffer(); err != nil {
		w.mu.Unlock()
		return fmt.Errorf("failed to flush buffer during close: %w", err)
	}

	// Close the current segment file
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			w.mu.Unlock()
			return fmt.Errorf("failed to close segment file: %w", err)
		}
	}

	w.mu.Unlock()
	return nil
}

// openOrCreateSegment opens or creates a new segment file.
func (w *LogWriter) openOrCreateSegment() error {
	// Find the next available segment ID
	var segmentID uint64 = 1
	if files, err := filepath.Glob(filepath.Join(w.dir, "*.wal")); err == nil {
		// Find the highest segment ID
		for _, f := range files {
			var id uint64
			_, err := fmt.Sscanf(filepath.Base(f), "%d.wal", &id)
			if err == nil && id >= segmentID {
				segmentID = id + 1
			}
		}
	}

	// Create the segment file
	filename := filepath.Join(w.dir, fmt.Sprintf("%020d.wal", segmentID))
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// Get the current file offset
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		_ = file.Close()
		return err
	}

	// Update the writer state
	w.file = file
	w.segmentID = segmentID
	w.offset = offset

	return nil
}

// rotateSegment closes the current segment and opens a new one.
func (w *LogWriter) rotateSegment() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
	}

	// Create a new segment
	w.segmentID++
	filename := filepath.Join(w.dir, fmt.Sprintf("%020d.wal", w.segmentID))
	file, err := os.Create(filename)
	if err != nil {
		return err
	}

	w.file = file
	w.offset = 0

	return nil
}
