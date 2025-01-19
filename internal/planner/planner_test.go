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

func TestMakeActionPlan(t *testing.T) {
	// Mock LLM client
	mockLLMClient := &llm.DummyClient{
		ReturnValue: `{
  "investigate_steps": [
    "Analyze code structure",
    "Check file dependencies"
  ],
  "change_steps": [
    "Update module B"
  ]
}`,
	}
	mockEntClient := &ent.Client{} // Assuming you have a mock or a real client

	// Create a new planner instance
	planner := NewPlanner(mockLLMClient, mockEntClient)

	// Define test cases
	tests := []struct {
		name            string
		candidateBlocks map[string][]Block
		currentPlan     *ChangesPlan
		query           string
		review          string
		expectedError   bool
	}{
		{
			name: "Basic test case",
			candidateBlocks: map[string][]Block{
				"file1.go": {
					{Path: "file1.go", TargetType: "function", TargetName: "Func1", Content: "func Func1() {}"},
				},
			},
			currentPlan:   nil,
			query:         "Refactor Func1",
			review:        "",
			expectedError: false,
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the function
			actionPlan, err := planner.makeActionPlan(context.Background(), tt.candidateBlocks, tt.currentPlan, tt.query, tt.review)

			// Check for errors
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, actionPlan)
				assert.Equal(t, 2, len(actionPlan.InvestigateSteps))
				assert.Equal(t, 1, len(actionPlan.ChangeSteps))
			}
		})
	}
}
