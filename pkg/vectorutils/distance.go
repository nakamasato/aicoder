package vectorutils

import (
	"log"
	"math"
)

func EuclideanDistance(a, b []float32) float64 {
	if len(a) != len(b) {
		log.Fatalf("Vectors must be of same length")
	}

	var sum float64

	for i := 0; i < len(a); i++ {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}

	return math.Sqrt(sum)
}
