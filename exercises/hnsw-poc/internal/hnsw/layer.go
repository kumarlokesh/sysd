package hnsw

func (h *HNSW) addNodeToLayer(node *Node, layer int) {
	for len(h.layers) <= layer {
		h.layers = append(h.layers, &Layer{nodes: make([]*Node, 0)})
	}

	found := false
	nodeList := h.layers[layer].nodes
	for _, n := range nodeList {
		if n.ID == node.ID {
			found = true
			break
		}
	}

	if !found {
		h.layers[layer].nodes = append(h.layers[layer].nodes, node)
	}
}

func (h *HNSW) getNode(id int) *Node {
	h.nodesMutex.RLock()
	defer h.nodesMutex.RUnlock()
	return h.nodes[id]
}

func (h *HNSW) addNode(node *Node) {
	h.nodesMutex.Lock()
	defer h.nodesMutex.Unlock()
	h.nodes[node.ID] = node
}
