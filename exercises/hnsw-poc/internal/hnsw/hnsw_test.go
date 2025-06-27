package hnsw

import (
	"math/rand"
	"testing"
	"time"
)

func runTestWithTimeout(t *testing.T, timeout time.Duration, testFunc func(*testing.T)) {
	done := make(chan bool, 1)
	go func() {
		testFunc(t)
		done <- true
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		t.Fatal("Test timed out")
	}
}

func TestHNSWInsertAndSearch(t *testing.T) {
	runTestWithTimeout(t, 30*time.Second, func(t *testing.T) {
		t.Log("Starting TestHNSWInsertAndSearch")
		dim := 4
		h := New(dim, Config{
			M:              4,
			EfConstruction: 10,
			EfSearch:       10,
		})
		t.Logf("Created new HNSW instance: %+v", h)

		// Insert test vectors (unit vectors)
		t.Log("Creating test vectors")
		vectors := [][]float32{
			{1.0, 0.0, 0.0, 0.0}, // 0
			{0.0, 1.0, 0.0, 0.0}, // 1
			{0.0, 0.0, 1.0, 0.0}, // 2
			{0.0, 0.0, 0.0, 1.0}, // 3
		}
		t.Logf("Created %d test vectors", len(vectors))

		h.Insert(0, vectors[0])

		// Verify the graph state after first insert
		if h.entryPointID != 0 {
			t.Fatalf("Expected entry point to be 0 after first insert, got %d", h.entryPointID)
		}

		// Insert remaining vectors
		for i := 1; i < len(vectors); i++ {
			t.Logf("Inserting vector %d: %v", i, vectors[i])
			h.Insert(i, vectors[i])
			t.Logf("Successfully inserted vector %d", i)
		}

		// Test search with a query vector similar to the first vector
		t.Log("Starting search test 1")
		query := []float32{0.9, 0.1, 0.1, 0.1}
		t.Logf("Searching with query: %v", query)

		// First, verify the graph structure
		t.Logf("Graph structure: %+v", h)
		for i, layer := range h.layers {
			t.Logf("Layer %d: %d nodes", i, len(layer.nodes))
		}

		results := h.Search(query, 2)
		t.Logf("Search results: %v", results)

		if len(results) == 0 {
			t.Fatal("Expected at least one result, got none")
		}

		// The first result should be the most similar vector (index 0)
		if results[0] != 0 {
			t.Errorf("Expected most similar vector to be at index 0, got %d", results[0])
		}

		// Test search with a different query
		query = []float32{0.1, 0.9, 0.1, 0.1}
		results = h.Search(query, 1)
		if len(results) == 0 || results[0] != 1 {
			t.Errorf("Expected most similar vector to be at index 1, got %v", results)
		}

		t.Log("Testing exact match search")
		exactMatch := []float32{0.0, 1.0, 0.0, 0.0}
		exactResults := h.Search(exactMatch, 1)
		t.Logf("Exact match search results: %v", exactResults)

		if len(exactResults) == 0 || exactResults[0] != 1 {
			t.Errorf("Expected exact match at index 1, got %v", exactResults)
		}
	})
}

func TestHNSWMultipleLayers(t *testing.T) {
	t.Log("Starting TestHNSWMultipleLayers")
	h := New(2, Config{
		M:              2,
		EfConstruction: 4,
		EfSearch:       4,
	})

	// Insert points in 2D space
	points := [][]float32{
		{0, 0}, {1, 0}, {0, 1}, {1, 1}, // Corners
		{0.5, 0.5}, {0.2, 0.8}, {0.8, 0.2}, // Points inside
	}

	for i, p := range points {
		h.Insert(i, p)
	}

	// Test search near the center
	query := []float32{0.6, 0.6}
	results := h.Search(query, 1)

	if len(results) == 0 || results[0] != 4 { // Point 4 is {0.5, 0.5}
		t.Errorf("Expected point 4 as nearest to %v, got %v", query, results)
	}
}

func TestHNSWParallelInserts(t *testing.T) {
	runTestWithTimeout(t, 30*time.Second, func(t *testing.T) {
		dim := 4
		h := New(dim, Config{
			M:              4,
			EfConstruction: 10,
			EfSearch:       10,
		})
		n := 100 // Number of parallel inserts
		errs := make(chan error, n)

		for i := 0; i < n; i++ {
			go func(id int) {
				vector := make([]float32, dim)
				for j := range vector {
					vector[j] = rand.Float32()
				}
				h.Insert(id, vector)
				errs <- nil
			}(i)
		}

		// Wait for all goroutines to complete
		for range n {
			if err := <-errs; err != nil {
				t.Fatal(err)
			}
		}

		query := make([]float32, 10)
		for i := range query {
			query[i] = rand.Float32()
		}

		results := h.Search(query, 5)
		if len(results) == 0 {
			t.Error("Search returned no results")
		}
	})
}

func TestRandomLevel(t *testing.T) {
	t.Log("Starting TestRandomLevel")
	h := New(4, Config{})
	levels := make(map[int]int)

	// Generate many levels and count their distribution
	const numTrials = 10000
	for i := 0; i < numTrials; i++ {
		level := h.randomLevel()
		levels[level]++
	}

	// Ensure we're generating multiple levels
	if len(levels) < 2 {
		t.Error("Expected multiple levels, got", len(levels))
	}
}
