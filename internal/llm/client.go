package llm

import (
	"context"
)

type Client interface {
	GenerateCompletion(ctx context.Context, messages []Message, schema Schema) (string, error)
	GenerateCompletions(ctx context.Context, messages []Message, schema Schema, n int64) ([]string, error)
	GenerateCompletionSimple(ctx context.Context, messages []Message) (string, error)
	GenerateFunctionCalling(ctx context.Context, messages []Message, tools []Tool) ([]ToolCall, error)
	GetEmbedding(ctx context.Context, content string) ([]float32, error)
}

type Tool struct {
	Name               string
	Description        string
	Properties         map[string]interface{}
	RequiredProperties []string
}

type ToolCall struct {
	ID           string
	FunctionName string
	Arguments    map[string]interface{}
}

type Role string

const (
	RoleSystem Role = "system"
	RoleUser   Role = "user"
)

type Message struct {
	Role    Role   `json:"type"`
	Content string `json:"content"`
}

