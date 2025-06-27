package hnsw

import (
	"container/heap"
	"slices"
	"sort"
)

// connectNode connects a node to its nearest neighbors in a specific layer
func (h *HNSW) connectNode(node *Node, neighbors []*priorityQueueItem, layer int) {
	if len(neighbors) == 0 {
		return
	}

	// Get the maximum number of connections for this layer
	M := h.getM(layer)
	if M < 1 {
		M = 1
	}

	// Ensure we have at least 2 connections to prevent linked list formation
	minConnections := 2
	if M < minConnections {
		minConnections = M
	}

	connected := make(map[int]bool)
	connectionsMade := 0

	h.nodesMutex.RLock()
	defer h.nodesMutex.RUnlock()

	// Sort neighbors by distance (ascending)
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].distance < neighbors[j].distance
	})

	// Connect to up to M nearest neighbors
	for i := 0; i < len(neighbors) && connectionsMade < M; i++ {
		neighbor := neighbors[i]
		if neighbor == nil || neighbor.nodeID == node.ID {
			continue
		}

		neighborNode := h.nodes[neighbor.nodeID]
		if neighborNode == nil {
			continue
		}

		if connected[neighbor.nodeID] {
			continue
		}

		node.OutEdges[layer] = append(node.OutEdges[layer], neighbor.nodeID)
		connected[neighbor.nodeID] = true
		connectionsMade++

		reverseConnected := false
		neighborEdges := neighborNode.OutEdges[layer]
		if slices.Contains(neighborEdges, node.ID) {
			reverseConnected = true
		}

		if !reverseConnected {
			neighborNode.OutEdges[layer] = append(neighborNode.OutEdges[layer], node.ID)
		}
	}

	// If we didn't make enough connections, find the closest nodes in the layer
	if connectionsMade < minConnections && len(h.layers) > layer && h.layers[layer] != nil {
		pq := make(priorityQueue, 0)
		heap.Init(&pq)

		for _, n := range h.layers[layer].nodes {
			if n == nil || n.ID == node.ID || connected[n.ID] {
				continue
			}

			distance := h.distanceFunc(node.Vector, n.Vector)
			heap.Push(&pq, &priorityQueueItem{
				nodeID:   n.ID,
				distance: distance,
				node:     n,
			})
		}

		// Connect to the closest nodes until we have enough connections
		for pq.Len() > 0 && connectionsMade < minConnections {
			item := heap.Pop(&pq).(*priorityQueueItem)
			if item == nil || item.nodeID == node.ID || connected[item.nodeID] {
				continue
			}

			node.OutEdges[layer] = append(node.OutEdges[layer], item.nodeID)
			connected[item.nodeID] = true
			connectionsMade++
			neighborNode := h.nodes[item.nodeID]
			if neighborNode != nil {
				reverseConnected := false
				neighborEdges := neighborNode.OutEdges[layer]
				if slices.Contains(neighborEdges, node.ID) {
					reverseConnected = true
				}

				if !reverseConnected {
					neighborNode.OutEdges[layer] = append(neighborNode.OutEdges[layer], node.ID)
				}
			}
		}
	}
}
