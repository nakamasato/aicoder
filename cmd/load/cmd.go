package load

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/loader"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/cobra"
    "entgo.io/ent/dialect"
    entsql "entgo.io/ent/dialect/sql"
    _ "github.com/jackc/pgx/v5/stdlib"
)

var (
	outputFile   string
	openaiAPIKey string
	openaiModel  string
	dbConnString string
	refresh      bool
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
	client := openai.NewClient(option.WithAPIKey(config.OpenAIAPIKey))

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
		log.Printf("Refreshing all documents for repository: %s", config.Repository)
		if _, err := entClient.Document.Delete().Where(document.RepositoryEQ(config.Repository)).Exec(ctx); err != nil {
			log.Fatalf("failed to delete existing documents: %v", err)
		}
	}

	store := vectorstore.New(entClient, client)

	// Load existing RepoStructure if exists
	var previousRepo loader.RepoStructure
	if _, err := os.Stat(outputFile); err == nil {
		data, err := os.ReadFile(outputFile)
		if err != nil {
			log.Fatalf("Failed to read existing repo structure: %v", err)
		}
		if err := json.Unmarshal(data, &previousRepo); err != nil {
			log.Fatalf("Failed to parse existing repo structure: %v", err)
		}
	}

	// Load current RepoStructure
	loadCfg := config.GetCurrentLoadConfig()
	currentRepo, err := loader.LoadRepoStructureFromHead(ctx, gitRootPath, loadCfg.TargetPath, loadCfg.Include, loadCfg.Exclude)
	if err != nil {
		fmt.Printf("Error loading repo structure: %v\n", err)
		os.Exit(1)
	}

	// Marshal to JSON
	output, err := json.MarshalIndent(currentRepo, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	err = os.WriteFile(outputFile, output, 0644)
	if err != nil {
		fmt.Printf("Error writing JSON to file: %v\n", err)
		os.Exit(1)
	}

	// var mu sync.Mutex
	var wg sync.WaitGroup
	var errChan = make(chan error, currentRepo.Root.Size)
	log.Printf("found %d files", currentRepo.Root.Size)
	for fileinfo := range currentRepo.Root.FileInfoGenerator() {
		wg.Add(1)
		go func(fileInfo loader.FileInfo) {
			defer wg.Done()
			if fileinfo.IsDir {
				return
			}

			loadCfg := config.GetCurrentLoadConfig()
			if loadCfg.IsExcluded(fileinfo.Path) && !loadCfg.IsIncluded(fileinfo.Path) {
				return
			}

			buf, err := os.ReadFile(fileinfo.Path)
			if err != nil {
				errChan <- fmt.Errorf("failed to open file %s: %v", fileinfo.Path, err)
				return
			}

			doc, err := entClient.Document.Query().Where(document.RepositoryEQ(config.Repository), document.FilepathEQ(fileinfo.Path), document.ContextEQ(config.CurrentContext)).First(ctx)
			if err == nil && doc.UpdatedAt.After(fileinfo.ModifiedAt) {
				fmt.Printf("Document %s is up-to-date\n", fileinfo.Path)
				return
			}

			if err != nil && !ent.IsNotFound(err) {
				errChan <- fmt.Errorf("failed to query document %s: %v", fileinfo.Path, err)
				return
			}

			if string(buf) == "" {
				fmt.Printf("File is empty: %s\n", fileinfo.Path)
				return
			}

			summary, err := llm.SummarizeFileContent(ctx, client, string(buf))
			if err != nil {
				errChan <- fmt.Errorf("failed to summarize content: %v", err)
				return
			}
			if len(summary) == 0 {
				fmt.Printf("Summary is empty: %s\ncontent:%s", fileinfo.Path, string(buf))
				return
			}
			vsDoc := &vectorstore.Document{
				Repository:  config.Repository,
				Context:     config.CurrentContext,
				Filepath:    fileinfo.Path,
				Description: summary,
			}

			err = store.AddDocument(ctx, vsDoc)
			if err != nil {
				errChan <- fmt.Errorf("Failed to add vectorstore document %s: %v", fileinfo.Path, err)
				return
			}
			fmt.Printf("upserted document %s\n", fileinfo.Path)
		}(fileinfo)
	}
	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	fmt.Printf("Repository structure has been written to %s (%s)\n", outputFile, time.Since(startTs))
}
