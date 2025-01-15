package vectorstore

import (
	"context"
	"math"
	"testing"

	_ "github.com/lib/pq"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/nakamasato/aicoder/internal/llm"
)

func TestVectorStore_AddDocument(t *testing.T) {
	ctx := context.Background()

	// Initialize entgo client
	entClient, err := ent.Open("postgres", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable")
	if err != nil {
		t.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer entClient.Close()
	// Initialize vectorstore client
	vectorstoreClient := New(entClient, llm.DummyClient{})
	// Create a document
	doc := &Document{
		Repository:  "test-repository",
		Context:     "default",
		Filepath:    "test/filepath.go",
		Description: "This is a test description",
	}

	err = vectorstoreClient.AddDocument(ctx, doc)
	if err != nil {
		t.Fatalf("failed to add document: %v", err)
	}

	// Check if the document was added
	docs, err := vectorstoreClient.Search(ctx, "test-repository", "default", "test", 10)
	if err != nil {
		t.Fatalf("failed to search documents: %v", err)
	}
	if len(*docs.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(*docs.Documents))
	}
	if (*docs.Documents)[0].Document.Filepath != "test/filepath.go" {
		t.Fatalf("expected filepath 'test/filepath.go', got '%s'", (*docs.Documents)[0].Document.Filepath)
	}

	// Clean up
	_, err = entClient.Document.Delete().Where(document.FilepathEQ("test/filepath.go"), document.RepositoryEQ("test-repository")).Exec(ctx)
	if err != nil {
		t.Fatalf("failed to clean up: %v", err)
	}
}

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
			result := EuclideanDistance(tt.vector1, tt.vector2)
			if result != tt.expected {
				t.Errorf("got %f, want %f", result, tt.expected)
			}
		})
	}
}
