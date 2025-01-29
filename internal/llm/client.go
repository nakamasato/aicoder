package llm

import (
	"context"

	"github.com/openai/openai-go"
)

var (
	chatModel      = openai.ChatModelGPT4oMini
	embeddingModel = openai.EmbeddingModelTextEmbedding3Small
)

type Client interface {
	GenerateCompletion(ctx context.Context, messages []Message, schema openai.ResponseFormatJSONSchemaJSONSchemaParam) (string, error)
	GenerateCompletionSimple(ctx context.Context, messages []Message) (string, error)
	GetEmbedding(ctx context.Context, content string) ([]float32, error)
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

func (d DummyClient) GenerateCompletion(ctx context.Context, messages []Message, schema openai.ResponseFormatJSONSchemaJSONSchemaParam) (string, error) {
	if d.ReturnValue != "" {
		return d.ReturnValue, nil
	}
	return "dummy result", nil
}

func (d DummyClient) GenerateCompletionSimple(ctx context.Context, messages []Message) (string, error) {
	if d.ReturnValue != "" {
		return d.ReturnValue, nil
	}
	return "dummy simple result", nil
}

func (d DummyClient) GetEmbedding(ctx context.Context, content string) ([]float32, error) {
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = 0.1 // 任意の値で初期化
	}
	return embedding, nil
}
