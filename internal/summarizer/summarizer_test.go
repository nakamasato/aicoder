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
	llmClient := llm.DummyClient{
		ReturnValue: `{"overview":"A simple Go project.","features":["Feature 1","Feature 2"],"configuration":"config.yaml","environment_variables":[{"name":"API_KEY","desc":"API key for authentication","required":true}],"directory_structure":"src/\n  main.go - Entry point\n  utils/ - Utility functions","entrypoints":["main.go"],"important_files":["README.md","LICENSE"],"important_functions":["InitConfig","StartServer"],"dependencies":"graph TD;\n  A-->B;\n  B-->C;","technologies":["Go","Cobra"]}`,
	}
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
	assert.EqualValues(t, `{"overview":"A simple Go project.","features":["Feature 1","Feature 2"],"configuration":"config.yaml","environment_variables":[{"name":"API_KEY","desc":"API key for authentication","required":true}],"directory_structure":"src/\n  main.go - Entry point\n  utils/ - Utility functions","entrypoints":["main.go"],"important_files":["README.md","LICENSE"],"important_functions":["InitConfig","StartServer"],"dependencies":"graph TD;\n  A-->B;\n  B-->C;","technologies":["Go","Cobra"]}`, summary)
}
