// planner_test.go
package planner

import (
	"context"
	"testing"

	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/stretchr/testify/assert"
)

func TestNewPlanner(t *testing.T) {
	llmClient := llm.DummyClient{} // Assuming you have a dummy client
	entClient := &ent.Client{}     // Assuming you have a way to initialize this

	planner := NewPlanner(llmClient, entClient)

	assert.NotNil(t, planner)
	assert.Equal(t, llmClient, planner.llmClient)
	assert.Equal(t, entClient, planner.entClient)
}

func TestGeneratePromptWithFiles(t *testing.T) {
	llmClient := llm.DummyClient{} // Assuming you have a dummy client
	entClient := &ent.Client{}     // Assuming you have a way to initialize this
	planner := NewPlanner(llmClient, entClient)

	ctx := context.Background()
	prompt := "Modify the following files: %s to achieve the goal: %s"
	goal := "Refactor code"
	files := []file.File{
		{Path: "example.go", Content: "package main\nfunc main() {}"},
	}
	fileBlocks := map[string][]Block{
		"example.go": {
			{Path: "example.go", TargetType: "function", TargetName: "main", Content: "func main() {}"},
		},
	}

	result, err := planner.GeneratePromptWithFiles(ctx, prompt, goal, files, fileBlocks)
	assert.NoError(t, err)
	assert.Contains(t, result, "example.go")
	assert.Contains(t, result, "main")
}
