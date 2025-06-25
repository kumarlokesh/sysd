package sstable

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSTableReader(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sstable-test-")
	require.NoError(t, err)
	defer func() {
		err := os.RemoveAll(tempDir)
		require.NoError(t, err)
	}()

	t.Run("read single entry", func(t *testing.T) {
		path := filepath.Join(tempDir, "test-read-1.sst")

		writer, err := NewWriter(path)
		require.NoError(t, err)

		err = writer.Add([]byte("test-key"), []byte("test-value"))
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		reader, err := Open(path)
		require.NoError(t, err)
		defer func() {
			err := reader.Close()
			assert.NoError(t, err, "failed to close reader")
		}()

		value, err := reader.Get([]byte("test-key"))
		require.NoError(t, err)
		assert.Equal(t, []byte("test-value"), value)

		_, err = reader.Get([]byte("non-existent"))
		assert.Error(t, err)
	})

	t.Run("read multiple entries", func(t *testing.T) {
		path := filepath.Join(tempDir, "test-read-2.sst")

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

		err = writer.Close()
		require.NoError(t, err)

		reader, err := Open(path)
		require.NoError(t, err)
		defer func() {
			err := reader.Close()
			assert.NoError(t, err, "failed to close reader")
		}()

		for _, d := range testData {
			value, err := reader.Get([]byte(d.key))
			require.NoError(t, err)
			assert.Equal(t, []byte(d.value), value)
		}

		_, err = reader.Get([]byte("non-existent"))
		assert.Error(t, err)
	})

	t.Run("read large value", func(t *testing.T) {
		path := filepath.Join(tempDir, "test-read-large.sst")

		// Create a large value (larger than block size)
		largeValue := make([]byte, 2*blockSize)
		for i := range largeValue {
			largeValue[i] = byte(i % 256)
		}

		writer, err := NewWriter(path)
		require.NoError(t, err)

		err = writer.Add([]byte("large-key"), largeValue)
		require.NoError(t, err)

		err = writer.Close()
		require.NoError(t, err)

		reader, err := Open(path)
		require.NoError(t, err)
		defer func() {
			err := reader.Close()
			assert.NoError(t, err, "failed to close reader")
		}()

		value, err := reader.Get([]byte("large-key"))
		require.NoError(t, err)
		assert.Equal(t, largeValue, value)
	})

	t.Run("invalid file", func(t *testing.T) {
		_, err := Open("non-existent-file.sst")
		assert.Error(t, err)
	})

	t.Run("corrupted_file", func(t *testing.T) {
		path := filepath.Join(tempDir, "corrupted.sst")
		err := os.WriteFile(path, []byte("not a valid sstable"), 0644)
		require.NoError(t, err)

		_, err = Open(path)
		assert.Error(t, err)
	})

	t.Run("range_scan", func(t *testing.T) {
		path := filepath.Join(tempDir, "test-range-scan.sst")

		// Create test data with keys: a1, a2, b1, b2, c1, c2
		writer, err := NewWriter(path)
		require.NoError(t, err)

		testData := []struct {
			key   string
			value string
		}{
			{"a1", "value-a1"},
			{"a2", "value-a2"},
			{"b1", "value-b1"},
			{"b2", "value-b2"},
			{"c1", "value-c1"},
			{"c2", "value-c2"},
		}

		for _, d := range testData {
			err = writer.Add([]byte(d.key), []byte(d.value))
			require.NoError(t, err)
		}

		err = writer.Close()
		require.NoError(t, err)

		reader, err := Open(path)
		require.NoError(t, err)
		defer func() {
			err := reader.Close()
			assert.NoError(t, err, "failed to close reader")
		}()

		t.Run("full_range", func(t *testing.T) {
			var results []string
			it := reader.RangeScan(nil, nil)
			for it.Next() {
				results = append(results, string(it.Key()))
			}
			require.NoError(t, it.Error())
			assert.Equal(t, []string{"a1", "a2", "b1", "b2", "c1", "c2"}, results)
		})

		t.Run("middle_range", func(t *testing.T) {
			var results []string
			it := reader.RangeScan([]byte("a2"), []byte("b2"))
			for it.Next() {
				results = append(results, string(it.Key()))
			}
			require.NoError(t, it.Error())
			assert.Equal(t, []string{"a2", "b1", "b2"}, results)
		})

		t.Run("start_only", func(t *testing.T) {
			var results []string
			it := reader.RangeScan([]byte("b1"), nil)
			for it.Next() {
				results = append(results, string(it.Key()))
			}
			require.NoError(t, it.Error())
			assert.Equal(t, []string{"b1", "b2", "c1", "c2"}, results)
		})

		t.Run("end_only", func(t *testing.T) {
			var results []string
			it := reader.RangeScan(nil, []byte("b1"))
			for it.Next() {
				results = append(results, string(it.Key()))
			}
			require.NoError(t, it.Error())
			assert.Equal(t, []string{"a1", "a2", "b1"}, results)
		})

		t.Run("no_results", func(t *testing.T) {
			it := reader.RangeScan([]byte("x"), []byte("z"))
			assert.False(t, it.Next())
			assert.NoError(t, it.Error())
		})

		t.Run("single_key", func(t *testing.T) {
			var results []string
			it := reader.RangeScan([]byte("b1"), []byte("b1"))
			for it.Next() {
				results = append(results, string(it.Key()))
			}
			require.NoError(t, it.Error())
			assert.Equal(t, []string{"b1"}, results)
		})
	})
}
