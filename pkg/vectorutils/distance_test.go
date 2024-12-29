package vectorutils_test

import (
	"math"
	"testing"

	"github.com/nakamasato/aicoder/pkg/vectorutils"
)

func TestEuclideanDistance(t *testing.T) {
	// aicoder=# select embedding <-> '[1,2,3]' from pg_test ;
	//  ?column?
	// ----------
	//         0
	// (1 row)

	// aicoder=# select embedding <-> '[1,2,2]' from pg_test ;
	//  ?column?
	// ----------
	//         1
	// (1 row)

	// aicoder=# select embedding <-> '[1,1,2]' from pg_test ;
	//       ?column?
	// --------------------
	//  1.4142135623730951
	// (1 row)

	// aicoder=# select embedding <-> '[0,1,2]' from pg_test ;
	//       ?column?
	// --------------------
	//  1.7320508075688772
	// (1 row)
	tests := []struct {
		name     string
		vector1  []float32
		vector2  []float32
		expected float64
	}{
		{"Test case 1", []float32{1, 2, 3}, []float32{1, 2, 3}, 0},
		{"Test case 2", []float32{1, 2, 3}, []float32{1, 2, 2}, 1},
		{"Test case 3", []float32{1, 2, 3}, []float32{1, 1, 2}, math.Sqrt(2)},
		{"Test case 3", []float32{1, 2, 3}, []float32{0, 1, 2}, math.Sqrt(3)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vectorutils.EuclideanDistance(tt.vector1, tt.vector2)
			if result != tt.expected {
				t.Errorf("got %f, want %f", result, tt.expected)
			}
		})
	}
}
