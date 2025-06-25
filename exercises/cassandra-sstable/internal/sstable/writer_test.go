package sstable

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSTableWriter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sstable-test-")
	require.NoError(t, err)
	defer func() {
		err := os.RemoveAll(tempDir)
		assert.NoError(t, err, "failed to clean up temp directory")
	}()

	t.Run("write single entry", func(t *testing.T) {
		path := filepath.Join(tempDir, "test1.sst")
		writer, err := NewWriter(path)
		require.NoError(t, err)

		err = writer.Add([]byte("test-key"), []byte("test-value"))
		require.NoError(t, err)

		err = writer.Flush()
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(t, info.Size() > 0, "file should not be empty")
	})

	t.Run("write multiple entries", func(t *testing.T) {
		path := filepath.Join(tempDir, "test2.sst")
		writer, err := NewWriter(path)
		require.NoError(t, err)

		testData := []struct {
			key   string
			value string
		}{
			{"key1", "value1"},
			{"key2", "value2"},
			{"key3", "value3"},
		}

		for _, d := range testData {
			err = writer.Add([]byte(d.key), []byte(d.value))
			require.NoError(t, err)
		}

		err = writer.Flush()
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(t, info.Size() > 0, "file should not be empty")
	})

	t.Run("write entries larger than block size", func(t *testing.T) {
		path := filepath.Join(tempDir, "test3.sst")
		writer, err := NewWriter(path)
		require.NoError(t, err)

		// Create a value larger than the block size
		largeValue := bytes.Repeat([]byte("x"), 8192)

		err = writer.Add([]byte("large-value"), largeValue)
		require.NoError(t, err)

		err = writer.Flush()
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(t, info.Size() > 0, "file should not be empty")
	})

	t.Run("close without flush", func(t *testing.T) {
		path := filepath.Join(tempDir, "test4.sst")
		writer, err := NewWriter(path)
		require.NoError(t, err)

		err = writer.Add([]byte("key"), []byte("value"))
		require.NoError(t, err)

		// Close without explicit flush
		err = writer.Close()
		require.NoError(t, err)

		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(t, info.Size() > 0, "file should not be empty")
	})

	t.Run("multiple flushes", func(t *testing.T) {
		path := filepath.Join(tempDir, "test5.sst")
		writer, err := NewWriter(path)
		require.NoError(t, err)

		// First batch
		err = writer.Add([]byte("key1"), []byte("value1"))
		require.NoError(t, err)
		err = writer.Flush()
		require.NoError(t, err)

		// Second batch
		err = writer.Add([]byte("key2"), []byte("value2"))
		require.NoError(t, err)
		err = writer.Flush()
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		// Verify the file was created and has some content
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(t, info.Size() > 0, "file should not be empty")
	})
}
