package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"
)

// getEmbedding fetches the embedding for a given content using OpenAI.
func GetEmbedding(ctx context.Context, client *openai.Client, content string) ([]float32, error) {
	if len(content) == 0 {
		return nil, fmt.Errorf("content is empty")
	}

	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.F(openai.EmbeddingModelTextEmbedding3Small),
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
