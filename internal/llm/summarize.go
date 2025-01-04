package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
)

// SummarizeFileContent uses OpenAI to summarize the given text content.
func SummarizeFileContent(ctx context.Context, client *openai.Client, content string) (string, error) {
	if len(content) == 0 {
		return "", nil
	}

	// Prepare the prompt for summarization
	prompt := fmt.Sprintf(SUMMARIZE_FILE_CONTENT_PROMPT, content)

	// Create a chat completion request
	resp, err := client.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(prompt),
			}),
			Model: openai.F(openai.ChatModelGPT4oMini),
		})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}
