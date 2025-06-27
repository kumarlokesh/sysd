// Package hnsw implements the Hierarchical Navigable Small World (HNSW) algorithm
// for approximate nearest neighbor search in high-dimensional spaces.
package hnsw

import (
	"math"
	"math/rand"
	"sync"
	"time"
)

// Node represents a vector in the HNSW graph.
// Each node maintains connections to other nodes at different layers of the graph.
// The bottom layer (index 0) contains all nodes, while higher layers contain
// progressively fewer nodes to enable efficient search.
type Node struct {
	// ID is a unique identifier for the node within the graph
	ID int

	// Vector contains the high-dimensional data point that this node represents
	Vector []float32

	// Level is the maximum layer this node appears in
	Level int

	// OutEdges is a 2D slice where OutEdges[layer] contains the IDs of
	// neighboring nodes at that layer. Layer 0 is the bottom layer.
	OutEdges [][]int
}

// Layer represents a single level in the HNSW hierarchy.
// Each layer is a graph where nodes are connected to their nearest neighbors.
// Higher layers have fewer nodes, enabling efficient search through the hierarchy.
type Layer struct {
	// nodes is a set of node pointers present in this layer
	nodes []*Node
}

// HNSW implements the Hierarchical Navigable Small World graph for approximate nearest neighbor search.
// It maintains multiple layers of graphs with decreasing densities, allowing for efficient search
// through the hierarchy.
type HNSW struct {
	// layers contains the hierarchical graph structure
	// layers[0] is the bottom layer containing all nodes
	// layers[maxLayer] is the top layer with the fewest nodes
	layers []*Layer

	// M is the maximum number of connections per layer (except layer 0)
	M int

	// M0 is the maximum number of connections for layer 0
	M0 int

	// efConstruction is the size of the dynamic candidate list during construction
	efConstruction int

	// efSearch is the size of the dynamic candidate list during search
	efSearch int

	// mL is the normalization factor for level generation
	// Controls the distribution of nodes across layers
	mL float64

	// distanceFunc calculates the distance between two vectors
	distanceFunc func([]float32, []float32) float32

	// entryPointID is the ID of the entry point at the top layer
	entryPointID int

	// maxLayer is the current maximum layer index
	maxLayer int

	// nodes maps node IDs to their corresponding Node structs
	nodes map[int]*Node

	// nodesMux provides concurrent read/write access to the nodes map
	nodesMutex sync.RWMutex

	// mu protects the entire graph structure during modifications
	// Synchronization
	mu sync.RWMutex

	// Random number generator
	rand *rand.Rand
}

// Config holds configuration parameters for initializing an HNSW index.
// These parameters control the trade-off between search speed, accuracy, and memory usage.
type Config struct {
	// M is the maximum number of connections per layer (except layer 0).
	// Higher values improve search quality but increase memory usage and search time.
	// Typical values are between 4-48.
	M int

	// M0 is the maximum number of connections for layer 0.
	// This is typically set to 2*M for better performance.
	M0 int

	// EfConstruction is the size of the dynamic candidate list during construction.
	// Higher values improve index quality but slow down index construction.
	// Typical values are between 100-2000.
	EfConstruction int

	// EfSearch is the size of the dynamic candidate list during search.
	// Higher values improve search quality but increase search time.
	// Typical values are between 10-400.
	EfSearch int

	// ML is the normalization factor for level generation.
	// Controls the distribution of nodes across layers.
	// The default value of 1/ln(M) usually works well.
	ML float64

	// DistanceFunction calculates the distance between two vectors.
	// If nil, Euclidean distance is used by default.
	// The function should return smaller values for more similar vectors.
	DistanceFunction func(a, b []float32) float32
}

// priorityQueueItem represents an item in the priority queue used during search.
// It implements the heap.Interface for efficient priority queue operations.
type priorityQueueItem struct {
	nodeID   int     // ID of the node
	distance float32 // Distance to the query vector
	node     *Node   // Reference to the node (optional, used in some operations)
	index    int     // Internal index used by the heap
}

// searchState holds the state during the search process in the HNSW graph.
// It maintains the candidate set and the result set for the search.
type searchState struct {
	// Query vector for the current search
	query []float32

	// Current layer being searched
	layer int

	// Exploration factor (ef) - number of candidates to consider
	ef int

	// Priority queue of candidate nodes to explore
	candidates *priorityQueue

	// Current nearest neighbors found
	results *priorityQueue

	// Tracks visited nodes to avoid processing them multiple times
	visited map[int]bool

	// Number of iterations performed in the search
	iterations int
}

// NewNode creates a new node with the given ID, vector, and level
func NewNode(id int, vector []float32, level int) *Node {
	node := &Node{
		ID:       id,
		Vector:   make([]float32, len(vector)),
		Level:    level,
		OutEdges: make([][]int, level+1),
	}
	copy(node.Vector, vector)
	for i := 0; i <= level; i++ {
		node.OutEdges[i] = make([]int, 0)
	}
	return node
}

// New creates a new HNSW index with default parameters
func New(dim int, config ...Config) *HNSW {
	// Default configuration
	cfg := Config{
		M:              16,
		EfConstruction: 200,
		EfSearch:       10,
	}
	if len(config) > 0 {
		cfg = config[0]
	}

	// Ensure M is at least 2
	if cfg.M < 2 {
		cfg.M = 2
	}

	// Set M0 if not specified
	if cfg.M0 == 0 {
		cfg.M0 = cfg.M * 2
	}

	// Calculate mL (level normalization factor)
	mL := 1.0
	if cfg.M > 1 {
		mL = 1.0 / math.Log(float64(cfg.M))
	}

	// Create a new random number generator
	randSrc := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(randSrc)

	h := &HNSW{
		layers:         []*Layer{{nodes: make([]*Node, 0)}},
		nodes:          make(map[int]*Node),
		M:              cfg.M,
		M0:             cfg.M0,
		efConstruction: cfg.EfConstruction,
		efSearch:       cfg.EfSearch,
		mL:             mL,
		distanceFunc:   euclideanDistance,
		entryPointID:   -1,
		maxLayer:       -1,
		rand:           randGen,
	}

	return h
}

// getM returns the maximum number of connections for a given layer
func (h *HNSW) getM(layer int) int {
	if layer == 0 {
		return h.M0
	}
	return h.M
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
