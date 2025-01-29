package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type openaiClient struct {
	openai *openai.Client
}

func NewOpenAIClient(apiKey string) Client {
	return openaiClient{
		openai: openai.NewClient(option.WithAPIKey(apiKey)),
	}
}

// GenerateCompletion handles the common OpenAI chat completion logic
func (c openaiClient) GenerateCompletion(ctx context.Context, messages []Message, schema openai.ResponseFormatJSONSchemaJSONSchemaParam) (string, error) {
	msgs := c.convertMessages(messages)
	chat, err := c.openai.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Model:    openai.F(chatModel),
			Messages: openai.F(msgs),
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

func (c openaiClient) convertMessages(messages []Message) []openai.ChatCompletionMessageParamUnion {
	msgs := make([]openai.ChatCompletionMessageParamUnion, len(messages))
	for i, m := range messages {
		if m.Role == RoleSystem {
			msgs[i] = openai.SystemMessage(m.Content)
		} else if m.Role == RoleUser {
			msgs[i] = openai.UserMessage(m.Content)
		} else {
			msgs[i] = openai.SystemMessage(m.Content)
		}
	}
	return msgs
}

func (c openaiClient) GenerateCompletionSimple(ctx context.Context, messages []Message) (string, error) {
	msgs := c.convertMessages(messages)
	chat, err := c.openai.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Model:    openai.F(chatModel),
			Messages: openai.F(msgs),
		})
	if err != nil {
		return "", err
	}

	return chat.Choices[0].Message.Content, nil
}

// getEmbedding fetches the embedding for a given content using OpenAI.
func (c openaiClient) GetEmbedding(ctx context.Context, content string) ([]float32, error) {
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
