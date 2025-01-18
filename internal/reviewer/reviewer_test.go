package reviewer

import (
	"context"
	"testing"

	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/planner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLMClient is a mock implementation of the llm.Client interface
type MockLLMClient struct {
	mock.Mock
}

func TestReviewChanges(t *testing.T) {
	mockLLMClient := llm.DummyClient{
		ReturnValue: `{"plan_id": "123", "result": true, "comment": "Looks good to me"}`,
	}

	changesPlan := &planner.ChangesPlan{
		Id:    "123",
		Query: "Improve performance",
		Changes: []planner.BlockChange{
			{
				Block: planner.Block{
					Path:       "path/to/file",
					TargetType: "function",
					TargetName: "Optimize",
				},
				NewContent: "optimized code",
			},
		},
	}

	_, err := ReviewChanges(context.Background(), mockLLMClient, changesPlan)
	assert.NoError(t, err)
}
