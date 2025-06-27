package hnsw

// priorityQueue implements a min-heap based priority queue
type priorityQueue []*priorityQueueItem

// Len returns the number of elements in the queue
func (pq priorityQueue) Len() int { return len(pq) }

// Less compares two elements in the queue
func (pq priorityQueue) Less(i, j int) bool {
	// We want a min-heap, so we use less than here
	return pq[i].distance < pq[j].distance
}

// Swap swaps two elements in the queue
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push adds an element to the queue
func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*priorityQueueItem)
	item.index = n
	*pq = append(*pq, item)
}

// Pop removes and returns the minimum element from the queue
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// Peek returns the minimum element without removing it
func (pq priorityQueue) Peek() *priorityQueueItem {
	if len(pq) == 0 {
		return nil
	}
	return pq[0]
}
