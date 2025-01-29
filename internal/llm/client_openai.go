package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

var (
	chatModel      = openai.ChatModelGPT4oMini
	embeddingModel = openai.EmbeddingModelTextEmbedding3Small
)

type openaiClient struct {
	openai         *openai.Client
	chatModel      openai.ChatModel
	embeddingModel openai.EmbeddingModel
}

type ClientOption func(*openaiClient)

func WithChatModel(model openai.ChatModel) ClientOption {
	return func(c *openaiClient) {
		c.chatModel = model
	}
}

func WithEmbeddingModel(model openai.EmbeddingModel) ClientOption {
	return func(c *openaiClient) {
		c.embeddingModel = model
	}
}

func NewOpenAIClient(apiKey string, opts ...ClientOption) Client {
	client := openaiClient{
		openai:         openai.NewClient(option.WithAPIKey(apiKey)),
		chatModel:      chatModel,      // default chat model
		embeddingModel: embeddingModel, // default embedding model
	}

	for _, opt := range opts {
		opt(&client)
	}

	return client
}

// GenerateCompletion handles the common OpenAI chat completion logic
// https://github.com/openai/openai-go/blob/8a8855d08ef84f47163deb4ce9febc4a7e02dd3d/examples/structured-outputs/main.go#L49
func (c openaiClient) GenerateCompletion(ctx context.Context, messages []Message, schema Schema) (string, error) {
	msgs := c.convertMessages(messages)
	chat, err := c.openai.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Model:    openai.F(chatModel),
			Messages: openai.F(msgs),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONSchemaParam{
					Type: openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
					JSONSchema: openai.F(openai.ResponseFormatJSONSchemaJSONSchemaParam{
						Name:        openai.F(schema.Name),
						Description: openai.F(schema.Description),
						Schema:      openai.F(schema.Schema),
						Strict:      openai.Bool(true),
					}),
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

// https://github.com/openai/openai-go/blob/8a8855d08ef84f47163deb4ce9febc4a7e02dd3d/examples/chat-completion-tool-calling/main.go
func (c openaiClient) GenerateFunctionCalling(ctx context.Context, messages []Message, tools []Tool) ([]ToolCall, error) {
	msgs := c.convertMessages(messages)
	ts := []openai.ChatCompletionToolParam{}
	for _, t := range tools {
		ts = append(ts, openai.ChatCompletionToolParam{
			Type: openai.F(openai.ChatCompletionToolTypeFunction),
			Function: openai.F(openai.FunctionDefinitionParam{
				Name:        openai.String(t.Name),
				Description: openai.String(t.Description),
				Parameters: openai.F(openai.FunctionParameters{
					"type":       "object",
					"properties": t.Properties,
					"required":   t.RequiredProperties,
				}),
			}),
		},
		)
	}
	chat, err := c.openai.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Model:    openai.F(chatModel),
			Messages: openai.F(msgs),
			Tools:    openai.F(ts),
			Seed:     openai.Int(0),
		})
	if err != nil {
		return nil, err
	}

	toolCalls := chat.Choices[0].Message.ToolCalls

	var calls []ToolCall
	for _, tc := range toolCalls {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			panic(err)
		}
		calls = append(calls, ToolCall{
			ID:           tc.ID,
			FunctionName: tc.Function.Name,
			Arguments:    args,
		})
	}
	return calls, nil
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
