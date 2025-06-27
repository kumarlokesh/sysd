package main

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	"github.com/kumarlokesh/hnsw-poc/internal/hnsw"
)

func main() {
	dim := flag.Int("dim", 128, "Dimensionality of vectors")
	size := flag.Int("size", 1000, "Number of vectors to generate")
	k := flag.Int("k", 5, "Number of nearest neighbors to find")
	flag.Parse()

	h := hnsw.New(*dim)

	fmt.Printf("Inserting %d random %d-dimensional vectors...\n", *size, *dim)
	for i := 0; i < *size; i++ {
		v := make([]float32, *dim)
		for j := range v {
			v[j] = rand.Float32()
		}
		h.Insert(i, v)
	}

	query := make([]float32, *dim)
	for i := range query {
		query[i] = rand.Float32()
	}
	start := time.Now()
	neighbors := h.Search(query, *k)
	duration := time.Since(start)

	fmt.Printf("Found %d nearest neighbors in %v:\n", len(neighbors), duration)
	for i, id := range neighbors {
		fmt.Printf("%d. ID: %d\n", i+1, id)
	}
}
