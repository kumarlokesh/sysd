package hnsw

import (
	"log"
	"time"
)

// Insert adds a new vector to the HNSW index
func (h *HNSW) Insert(id int, vector []float32) {
	h.mu.Lock()
	defer h.mu.Unlock()

	_ = time.Now() // Keep for potential future metrics

	// Check if node already exists
	h.nodesMutex.RLock()
	if existingNode, exists := h.nodes[id]; exists {
		h.nodesMutex.RUnlock()
		log.Printf("Node %d already exists at level %d", id, existingNode.Level)
		return
	}
	h.nodesMutex.RUnlock()

	level := h.randomLevel()
	node := NewNode(id, vector, level)

	for l := 0; l <= level; l++ {
		h.addNodeToLayer(node, l)
	}

	h.addNode(node)

	if level > h.maxLayer {
		h.maxLayer = level
	}

	if h.entryPointID == -1 {
		h.entryPointID = id
		return
	}

	// For each layer from top to bottom, find nearest neighbors and connect
	for l := min(level, h.maxLayer); l >= 0; l-- {
		// Find nearest neighbors in this layer
		efConstruction := max(h.efConstruction, 1)
		neighbors := h.searchLayer(vector, []*priorityQueueItem{{
			nodeID:   h.entryPointID,
			distance: h.distanceFunc(vector, h.nodes[h.entryPointID].Vector),
			node:     h.nodes[h.entryPointID],
		}}, efConstruction, l)

		// Connect the node to its nearest neighbors
		if len(neighbors) > 0 {
			h.connectNode(node, neighbors, l)

			// Update connections for existing nodes
			h.nodesMutex.RLock()
			for _, neighbor := range neighbors {
				if neighbor == nil || neighbor.nodeID == id {
					continue
				}

				neighborNode := h.nodes[neighbor.nodeID]
				if neighborNode == nil {
					continue
				}

				// Create a priority queue item for the reverse connection
				reverseItem := &priorityQueueItem{
					nodeID:   id,
					distance: neighbor.distance,
					node:     node,
				}
				h.connectNode(neighborNode, []*priorityQueueItem{reverseItem}, l)
			}
			h.nodesMutex.RUnlock()
		} else {
			log.Printf("No neighbors found to connect at layer %d for node %d", l, id)
		}

		// Update entry point if this is the top layer
		if l == h.maxLayer && len(neighbors) > 0 {
			h.entryPointID = id
		}
	}
}

// randomLevel generates a random level for a new node using geometric distribution
func (h *HNSW) randomLevel() int {
	level := 0
	// If maxLayer is -1 (initial state), allow any level
	for h.rand.Float64() < 1.0/float64(h.M) && (h.maxLayer < 0 || level < h.maxLayer) {
		level++
	}
	return level
}
