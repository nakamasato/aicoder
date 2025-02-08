package search

import (
	"fmt"
	"log"
	"strings"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/spf13/cobra"
)

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
		Run:   runSearch,
	}

	// Define flags and configuration settings for searcherCmd
	searcherCmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string")
	searcherCmd.Flags().StringVarP(&openaiAPIKey, "api-key", "k", "", "OpenAI API key (can also set via OPENAI_API_KEY environment variable)")

	return searcherCmd
}

func runSearch(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	config := config.GetConfig()
	query := strings.Join(args, " ")

	// Initialize OpenAI client
	if openaiAPIKey != "" {
		config.OpenAIAPIKey = openaiAPIKey
	}
	if config.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}

	// Initialize entgo client
	entClient, err := ent.Open("postgres", dbConnString)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer entClient.Close()

	store := vectorstore.New(entClient, llm.NewOpenAIClient(config.OpenAIAPIKey))

	res, err := store.Search(ctx, config.Repository, config.CurrentContext, query, config.Search.TopN)
	if err != nil {
		log.Fatalf("failed to search: %v", err)
	}

	// Display results
	fmt.Printf("Top %d related files:\n", len(*res.Documents))
	contextBuilder := strings.Builder{}
	for i, doc := range *res.Documents {
		fmt.Printf("%d. %s (Score: %.2f)\n", i+1, doc.Document.Filepath, doc.Score)
		content, err := file.ReadContent(doc.Document.Filepath)
		if err != nil {
			log.Fatalf("failed to read content of file %s: %v", doc.Document.Filepath, err)
		}
		contextBuilder.WriteString(fmt.Sprintf("----- File: %s\n -----\n```\n%s\n```\n", doc.Document.Filepath, content))
	}

	llmClient := llm.NewOpenAIClient(config.OpenAIAPIKey)
	answer, err := llmClient.GenerateCompletionSimple(ctx, []llm.Message{
		{
			Role: llm.RoleSystem,
			Content: fmt.Sprintf("Please answer the question about a repository '%s' from a user\n\n##query\n%s\n## Relevant Files\n%s\n",
				config.Repository, query, contextBuilder.String()),
		}})
	if err != nil {
		log.Fatalf("failed to generate completion: %v", err)
	}
	fmt.Println(answer)
}
