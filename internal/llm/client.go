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

type DummyClient struct {
	ReturnValue string
}

func (d DummyClient) GenerateCompletion(ctx context.Context, messages []Message, schema Schema) (string, error) {
	res, err := d.GenerateCompletions(ctx, messages, schema, 1)
	if err != nil {
		return "", err
	}
	return res[0], nil
}

func (d DummyClient) GenerateCompletions(ctx context.Context, messages []Message, schema Schema, n int64) ([]string, error) {
	if d.ReturnValue != "" {
		return []string{d.ReturnValue}, nil
	}
	return []string{"dummy result"}, nil
}

func (d DummyClient) GenerateCompletionSimple(ctx context.Context, messages []Message) (string, error) {
	if d.ReturnValue != "" {
		return d.ReturnValue, nil
	}
	return "dummy simple result", nil
}

func (d DummyClient) GenerateFunctionCalling(ctx context.Context, messages []Message, tools []Tool) ([]ToolCall, error) {
	return nil, nil
}

func (d DummyClient) GetEmbedding(ctx context.Context, content string) ([]float32, error) {
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = 0.1 // 任意の値で初期化
	}
	return embedding, nil
}
