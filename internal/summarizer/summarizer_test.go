package summarizer

import (
	"context"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/stretchr/testify/assert"
)

func TestSummarizeRepo(t *testing.T) {
	// Mock dependencies
	cfg := &config.AICoderConfig{
		Repository:     "test-repo",
		CurrentContext: "test-context",
	}

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Fatal("TEST_DATABASE_URL is not set")
	}

	// Initialize entgo client
	entClient, err := ent.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed opening connection to postgres: %v", err)
	}
	llmClient := &llm.DummyClient{} // Use the DummyClient for testing
	store := vectorstore.New(entClient, llm.DummyClient{})

	// Create service
	svc := NewService(cfg, entClient, llmClient, store)

	// Define test context
	ctx := context.Background()

	// Call the method
	summary, err := svc.UpdateRepoSummary(ctx, LanguageEnglish, "test-output.json")

	defer func() {
		// delete the file
		err := os.Remove("test-output.json")
		if err != nil {
			t.Fatalf("failed to delete file: %v", err)
		}
	}()

	// Assert no error occurred
	assert.NoError(t, err)

	// Additional assertions can be added here to verify the behavior
	assert.EqualValues(t, "dummy simple result", summary)
}
