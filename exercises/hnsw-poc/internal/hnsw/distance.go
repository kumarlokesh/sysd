package hnsw

import "math"

func euclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1))
	}

	var sum float32
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return float32(math.Sqrt(float64(sum)))
}
