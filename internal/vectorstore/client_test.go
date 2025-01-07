package vectorstore

import (
	"context"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func TestVectorStore_AddDocument(t *testing.T) {
	ctx := context.Background()
	// Initialize OpenAI client
	apiKey := os.Getenv("OPENAI_API_KEY")
	openaiClient := openai.NewClient(option.WithAPIKey(apiKey))
	// Initialize entgo client
	entClient, err := ent.Open("postgres", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable")
	if err != nil {
		t.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer entClient.Close()
	// Initialize vectorstore client
	vectorstoreClient := New(entClient, openaiClient)
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
