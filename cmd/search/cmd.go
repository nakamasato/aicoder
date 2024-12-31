package search

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/llm"
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
			config := config.GetConfig()
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
			queryEmbedding, err := llm.GetEmbedding(ctx, client, query)
			if err != nil {
				log.Fatalf("failed to get embedding for query: %v", err)
			}

			// Fetch top N similar documents
			results, err := fetchSimilarDocuments(ctx, entClient, queryEmbedding, config.Search.TopN)
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


// fetchSimilarDocuments retrieves top N similar documents based on cosine similarity.
func fetchSimilarDocuments(ctx context.Context, entClient *ent.Client, queryEmbedding []float32, topN int) ([]SearchResult, error) {

	// Convert to pgvector.Vector
	vector := pgvector.NewVector(queryEmbedding)

	docs, err := entClient.Document.
		Query().
		Order(func(s *sql.Selector) {
			s.OrderExpr(sql.ExprP("embedding <-> $1", vector))
		}).
		Limit(topN).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch similar documents: %w", err)
	}

	var results []SearchResult
	for _, doc := range docs {
		distance := vectorutils.EuclideanDistance(doc.Embedding.Slice(), queryEmbedding)
		results = append(results, SearchResult{
			Path:        doc.Filepath,
			Description: doc.Description,
			Score:       distance,
		})
	}

	return results, nil
}
