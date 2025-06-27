package hnsw

import (
	"container/heap"
	"sort"
)

// Search finds the k nearest neighbors to the query vector
func (h *HNSW) Search(query []float32, k int) []int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.layers) == 0 || h.entryPointID == -1 {
		return nil
	}

	// Ensure we have enough exploration factor
	ef := max(h.efSearch, k*4) // Explore at least 4x the requested k
	ef = max(ef, 20)           // But at least 20

	// Start from the top layer
	currentNode := h.getNode(h.entryPointID)
	if currentNode == nil {
		return nil
	}

	// Find the entry point in the top layer
	for l := h.maxLayer; l >= 1; l-- {
		changed := true
		for changed {
			changed = false
			neighbors := currentNode.OutEdges[l]
			minDist := h.distanceFunc(query, currentNode.Vector)

			for _, neighborID := range neighbors {
				neighbor := h.getNode(neighborID)
				if neighbor == nil {
					continue
				}

				dist := h.distanceFunc(query, neighbor.Vector)
				if dist < minDist {
					currentNode = neighbor
					minDist = dist
					changed = true
				}
			}
		}
	}

	// Search in bottom layer with full ef
	candidates := h.searchLayer(query, []*priorityQueueItem{{
		nodeID:   currentNode.ID,
		distance: h.distanceFunc(query, currentNode.Vector),
		node:     currentNode,
	}}, ef, 0)

	// Collect results
	results := h.selectNeighborsSimple(candidates, k, 0)
	neighbors := make([]int, 0, len(results))
	for _, item := range results {
		neighbors = append(neighbors, item.nodeID)
	}

	return neighbors
}

// searchLayer performs a search in a specific layer
func (h *HNSW) searchLayer(query []float32, eps []*priorityQueueItem, ef, layer int) []*priorityQueueItem {
	const maxIterations = 2000 // Increased for better exploration

	if len(query) == 0 || len(eps) == 0 {
		return nil
	}

	validEps := make([]*priorityQueueItem, 0, len(eps))
	for _, ep := range eps {
		if ep != nil && ep.node != nil {
			validEps = append(validEps, ep)
		}
	}

	if len(validEps) == 0 {
		return nil
	}

	// Initialize search state with a larger ef for better exploration
	state := &searchState{
		query:      query,
		layer:      layer,
		ef:         max(ef, 20), // Ensure minimum exploration
		visited:    make(map[int]bool),
		candidates: &priorityQueue{},
		results:    &priorityQueue{},
	}

	// Initialize with entry points
	for _, ep := range validEps {
		heap.Push(state.candidates, ep)
		heap.Push(state.results, &priorityQueueItem{
			nodeID:   ep.nodeID,
			distance: ep.distance,
			node:     ep.node,
		})
	}

	for state.candidates.Len() > 0 && state.iterations < maxIterations {
		candidate := heap.Pop(state.candidates).(*priorityQueueItem)
		if !h.processCandidate(state, candidate) {
			heap.Push(state.candidates, candidate)
			break
		}
		state.iterations++

		// Early termination if we've explored enough
		if state.results.Len() >= state.ef && state.candidates.Len() > 0 {
			nextBest := (*state.candidates)[0].distance
			worstInResults := (*state.results)[0].distance
			if nextBest > worstInResults*1.5 {
				break
			}
		}
	}

	results := make([]*priorityQueueItem, 0, state.results.Len())
	for state.results.Len() > 0 {
		item := heap.Pop(state.results).(*priorityQueueItem)
		results = append(results, item)
	}

	// Reverse to get results in order of increasing distance
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
	}

	return results
}

// processCandidate processes a single candidate in the search
func (h *HNSW) processCandidate(state *searchState, candidate *priorityQueueItem) bool {
	if state.visited[candidate.nodeID] {
		return true
	}
	state.visited[candidate.nodeID] = true
	node := h.getNode(candidate.nodeID)
	if node == nil {
		return true
	}

	heap.Push(state.results, &priorityQueueItem{
		nodeID:   candidate.nodeID,
		distance: candidate.distance,
		node:     node,
	})

	// If we have too many results, remove the worst one
	if state.results.Len() > state.ef {
		heap.Pop(state.results)
	}

	// Explore neighbors
	for _, neighborID := range node.OutEdges[state.layer] {
		if state.visited[neighborID] {
			continue
		}

		neighborNode := h.getNode(neighborID)
		if neighborNode == nil {
			continue
		}

		// Calculate distance to neighbor
		distance := h.distanceFunc(state.query, neighborNode.Vector)

		// Add to candidates if it's promising
		if state.results.Len() < state.ef || distance < (*state.results)[0].distance*1.5 {
			heap.Push(state.candidates, &priorityQueueItem{
				nodeID:   neighborID,
				distance: distance,
				node:     neighborNode,
			})
		}

		// Early exit if we have enough good candidates
		if state.results.Len() >= state.ef && state.candidates.Len() > 0 {
			nextBest := (*state.candidates)[0].distance
			if nextBest > (*state.results)[0].distance*1.5 {
				return false
			}
		}
	}

	return true
}

// selectNeighborsSimple selects the M nearest neighbors from candidates
func (h *HNSW) selectNeighborsSimple(candidates []*priorityQueueItem, M int, layer int) []*priorityQueueItem {
	if len(candidates) <= M {
		return candidates
	}

	// Sort candidates by distance (ascending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].distance < candidates[j].distance
	})

	// Return the M nearest
	if M < len(candidates) {
		return candidates[:M]
	}
	return candidates
}
