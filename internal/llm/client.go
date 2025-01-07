package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

var (
	chatModel      = openai.ChatModelGPT4oMini
	embeddingModel = openai.EmbeddingModelTextEmbedding3Small
)

type Client interface {
	GenerateCompletion(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion, schema openai.ResponseFormatJSONSchemaJSONSchemaParam) (string, error)
	GenerateCompletionSimple(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (string, error)
	GetEmbedding(ctx context.Context, content string) ([]float32, error)
}

type client struct {
	openai *openai.Client
}

type DummyClient struct{}

func (d DummyClient) GenerateCompletion(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion, schema openai.ResponseFormatJSONSchemaJSONSchemaParam) (string, error) {
	return "", nil
}

func (d DummyClient) GenerateCompletionSimple(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (string, error) {
	return "", nil
}

func (d DummyClient) GetEmbedding(ctx context.Context, content string) ([]float32, error) {
	embedding := make([]float32, 1536)
	for i := range embedding {
		embedding[i] = 0.1 // 任意の値で初期化
	}
	return embedding, nil
}
func NewClient(apiKey string) Client {
	return client{
		openai: openai.NewClient(option.WithAPIKey(apiKey)),
	}
}

// GenerateCompletion handles the common OpenAI chat completion logic
func (c client) GenerateCompletion(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion, schema openai.ResponseFormatJSONSchemaJSONSchemaParam) (string, error) {
	chat, err := c.openai.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Model:    openai.F(chatModel),
			Messages: openai.F(messages),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONSchemaParam{
					Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
					JSONSchema: openai.F(schema),
				},
			),
		})
	if err != nil {
		return "", err
	}

	return chat.Choices[0].Message.Content, nil
}

func (c client) GenerateCompletionSimple(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (string, error) {
	chat, err := c.openai.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Model:    openai.F(chatModel),
			Messages: openai.F(messages),
		})
	if err != nil {
		return "", err
	}

	return chat.Choices[0].Message.Content, nil
}

// getEmbedding fetches the embedding for a given content using OpenAI.
func (c client) GetEmbedding(ctx context.Context, content string) ([]float32, error) {
	if len(content) == 0 {
		return nil, fmt.Errorf("content is empty")
	}

	resp, err := c.openai.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.F(embeddingModel),
		Input: openai.F(openai.EmbeddingNewParamsInputUnion(openai.EmbeddingNewParamsInputArrayOfStrings{content})),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	var embedding []float32
	for _, v := range resp.Data[0].Embedding {
		embedding = append(embedding, float32(v))
	}

	return embedding, nil
}
