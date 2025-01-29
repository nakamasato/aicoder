package summarize

import (
	"database/sql"
	"log"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/summarizer"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/spf13/cobra"
)

var (
	outputFile   string
	openaiAPIKey string
	openaiModel  string
	dbConnString string
	languageStr  string
)

func Command() *cobra.Command {
	summarizeCmd := &cobra.Command{
		Use:   "summarize",
		Short: "Summarize the repository",
		Run:   runSummarize,
	}

	// Define flags and configuration settings for summarizeCmd
	summarizeCmd.Flags().StringVarP(&outputFile, "output", "o", "repo_structure.json", "Output JSON file")
	summarizeCmd.Flags().StringVarP(&openaiAPIKey, "api-key", "k", "", "OpenAI API key (can also set via OPENAI_API_KEY environment variable)")
	summarizeCmd.Flags().StringVarP(&openaiModel, "model", "m", "gpt-4o-mini", "OpenAI model to use for summarization")
	summarizeCmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string (e.g., postgres://aicoder:aicoder@localhost:5432/aicoder)")
	summarizeCmd.Flags().StringVarP(&languageStr, "language", "l", "en", "Language of the repository")

	return summarizeCmd
}

func runSummarize(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	config := config.GetConfig()

	// Initialize OpenAI client
	if openaiAPIKey != "" {
		config.OpenAIAPIKey = openaiAPIKey
	}
	if config.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}
	llmClient := llm.NewOpenAIClient(config.OpenAIAPIKey)

	// Initialize PostgreSQL connection
	if dbConnString == "" {
		log.Fatal("Database connection string must be provided via --db-conn")
	}

	db, err := sql.Open("pgx", dbConnString)
	if err != nil {
		log.Fatal(err)
	}

	// Create an ent.Driver from `db`.
	drv := entsql.OpenDB(dialect.Postgres, db)
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(30)
	db.SetConnMaxLifetime(time.Minute * 5)
	entClient := ent.NewClient(ent.Driver(drv))
	defer entClient.Close()

	store := vectorstore.New(entClient, llmClient)

	svc := summarizer.NewService(&config, entClient, llmClient, store)

	if _, err := svc.UpdateRepoSummary(ctx, summarizer.Language(languageStr), "repo_summary.json"); err != nil {
		log.Fatalf("failed to summarize repository: %v", err)
	}
}
