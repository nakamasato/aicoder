package load

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/loader"
	"github.com/nakamasato/aicoder/internal/summarizer"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/spf13/cobra"
)

var (
	outputFile   string
	openaiAPIKey string
	openaiModel  string
	dbConnString string
	refresh      bool
	summary      bool
)

func Command() *cobra.Command {
	loadCmd := &cobra.Command{
		Use:   "load",
		Short: "Load the repository structure from a Git repository and export it to a JSON file with summaries.",
		Run:   runLoad,
	}

	// Define flags and configuration settings for loaderCmd
	loadCmd.Flags().StringVarP(&outputFile, "output", "o", "repo_structure.json", "Output JSON file")
	loadCmd.Flags().StringVarP(&openaiAPIKey, "api-key", "k", "", "OpenAI API key (can also set via OPENAI_API_KEY environment variable)")
	loadCmd.Flags().StringVarP(&openaiModel, "model", "m", "gpt-4o-mini", "OpenAI model to use for summarization")
	loadCmd.Flags().BoolVarP(&refresh, "refresh", "r", false, "Refresh all the document summaries")
	loadCmd.Flags().BoolVarP(&summary, "summary", "s", false, "Update the repository summary")
	loadCmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string (e.g., postgres://aicoder:aicoder@localhost:5432/aicoder)")

	return loadCmd
}

func runLoad(cmd *cobra.Command, args []string) {
	startTs := time.Now()
	ctx := cmd.Context()
	config := config.GetConfig()

	gitRootPath := "."

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

	if refresh {
		fmt.Printf("Refreshing all documents for repository: %s", config.Repository)
		if _, err := entClient.Document.Delete().Where(document.RepositoryEQ(config.Repository)).Exec(ctx); err != nil {
			log.Fatalf("failed to delete existing documents: %v", err)
		}
	}

	store := vectorstore.New(entClient, llmClient)

	// loader
	loaderSvc := loader.NewService(&config, entClient, llmClient, store)
	if _, err := loaderSvc.UpdateRepoStructure(ctx, gitRootPath, outputFile); err != nil {
		log.Fatalf("failed to load repository structure: %v", err)
	}
	if err := loaderSvc.UpdateDocuments(ctx); err != nil {
		log.Fatalf("failed to load repository structure: %v", err)
	}

	// summarizer
	if summary {
		summarizerSvc := summarizer.NewService(&config, entClient, llmClient, store)
		summary, err := summarizerSvc.UpdateRepoSummary(ctx, summarizer.LanguageEnglish, "repo_summary.json")
		if err != nil {
			log.Fatalf("failed to summarize repository: %v", err)
		}

		fmt.Printf("Summary\n%s\n", summary)
	}

	fmt.Printf("Repository structure has been written to %s (%s)\n", outputFile, time.Since(startTs))
}
