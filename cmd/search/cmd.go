package search

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/pkg/vectorutils"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/pgvector/pgvector-go"
	"github.com/spf13/cobra"
)

// SearchResult represents a search result with similarity score.
type SearchResult struct {
	Path        string  `json:"path"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
}

var (
	dbConnString string
	openaiAPIKey string
)

// Command creates the searcher command.
func Command() *cobra.Command {
	searcherCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for files related to the given query.",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			query := strings.Join(args, " ")

			// Initialize OpenAI client
			if openaiAPIKey == "" {
				openaiAPIKey = os.Getenv("OPENAI_API_KEY")
			}
			if openaiAPIKey == "" {
				log.Fatal("OPENAI_API_KEY environment variable is not set")
			}
			client := openai.NewClient(option.WithAPIKey(openaiAPIKey))

			// Initialize entgo client
			entClient, err := ent.Open("postgres", dbConnString)
			if err != nil {
				log.Fatalf("failed opening connection to postgres: %v", err)
			}
			defer entClient.Close()

			// Generate embedding for the query
			queryEmbedding, err := getEmbeddingFromDescription(ctx, client, query)
			if err != nil {
				log.Fatalf("failed to get embedding for query: %v", err)
			}

			// Fetch top 5 similar documents
			results, err := fetchSimilarDocuments(ctx, entClient, queryEmbedding, 5)
			if err != nil {
				log.Fatalf("failed to fetch similar documents: %v", err)
			}

			// Sort results by similarity (ascending)
			sort.Slice(results, func(i, j int) bool {
				return results[i].Score < results[j].Score
			})

			// Display results
			fmt.Printf("Top %d related files:\n", len(results))
			for i, res := range results {
				fmt.Printf("%d. %s (Score: %.4f)\n", i+1, res.Path, res.Score)
			}
		},
	}

	// Define flags and configuration settings for searcherCmd
	searcherCmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string")
	searcherCmd.Flags().StringVarP(&openaiAPIKey, "api-key", "k", "", "OpenAI API key (can also set via OPENAI_API_KEY environment variable)")

	return searcherCmd
}

// getEmbeddingFromDescription fetches the embedding for a given description using OpenAI.
func getEmbeddingFromDescription(ctx context.Context, client *openai.Client, description string) ([]float32, error) {
	if len(description) == 0 {
		return nil, fmt.Errorf("description is empty")
	}

	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
		Model: openai.F(openai.EmbeddingModelTextEmbedding3Small),
		Input: openai.F(openai.EmbeddingNewParamsInputUnion(openai.EmbeddingNewParamsInputArrayOfStrings{description})),
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

// fetchSimilarDocuments retrieves top N similar documents based on cosine similarity.
func fetchSimilarDocuments(ctx context.Context, entClient *ent.Client, queryEmbedding []float32, topN int) ([]SearchResult, error) {

	// Convert to pgvector.Vector
	vector := pgvector.NewVector(queryEmbedding)

	docs, err := entClient.Document.
		Query().
		Order(func(s *sql.Selector) {
			s.OrderExpr(sql.ExprP("embedding <-> $1", vector))
		}).
		Select("content", "embedding").
		Limit(topN).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch similar documents: %w", err)
	}

	var results []SearchResult
	for _, doc := range docs {
		distance := vectorutils.EuclideanDistance(doc.Embedding.Slice(), queryEmbedding)
		results = append(results, SearchResult{
			Path:        doc.Content,
			Description: "",
			Score:       distance,
		})
	}

	return results, nil
}
