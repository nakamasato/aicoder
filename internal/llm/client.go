package llm

import (
	"context"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Client struct {
	openai *openai.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		openai: openai.NewClient(option.WithAPIKey(apiKey)),
	}
}

// GenerateCompletion handles the common OpenAI chat completion logic
func (c *Client) GenerateCompletion(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion, schema openai.ResponseFormatJSONSchemaJSONSchemaParam) (string, error) {
	chat, err := c.openai.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Model:    openai.F(openai.ChatModelGPT4oMini),
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
