package load

import (
	"context"
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
	"github.com/nakamasato/aicoder/internal/load"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/pgvector/pgvector-go"
	"github.com/spf13/cobra"
)

var (
	outputFile   string
	branch       string
	commitHash   string
	openaiAPIKey string
	openaiModel  string
	maxTokens    int
	dbConnString string
)

func Command() *cobra.Command {
	loadCmd := &cobra.Command{
		Use:   "load",
		Short: "Load the repository structure from a Git repository and export it to a JSON file with summaries.",
		Run: func(cmd *cobra.Command, args []string) {
			startTs := time.Now()
			ctx := context.Background()
			config := config.GetConfig()

			gitRootPath := "."

			// Initialize OpenAI client
			if openaiAPIKey == "" {
				openaiAPIKey = os.Getenv("OPENAI_API_KEY")
			}
			if openaiAPIKey == "" {
				log.Fatal("OPENAI_API_KEY environment variable is not set")
			}
			client := openai.NewClient(option.WithAPIKey(openaiAPIKey))

			// Initialize PostgreSQL connection
			if dbConnString == "" {
				log.Fatal("Database connection string must be provided via --db-conn")
			}

			entClient, err := ent.Open("postgres", dbConnString)
			if err != nil {
				log.Fatalf("failed opening connection to postgres: %v", err)
			}
			defer entClient.Close()

			store := vectorstore.New(entClient, client)

			// Load existing RepoStructure if exists
			var previousRepo load.RepoStructure
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
			currentRepo, err := load.LoadRepoStructure(ctx, gitRootPath, branch, commitHash, config.Load.TargetPath, config.Load.Include, config.Load.Exclude)
			// currentRepo, err := loadRepoStructure(ctx, gitRootPath, branch, commitHash, client, entClient, previousRepo, config)
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
			for fileinfo := range currentRepo.Root.FileInfoGenerator() {
				wg.Add(1)
				go func(fileInfo load.FileInfo) {
					defer wg.Done()
					if fileinfo.IsDir {
						return
					}
					buf, err := os.ReadFile(fileinfo.Path)
					if err != nil {
						errChan <- fmt.Errorf("failed to open file %s: %v", fileinfo.Path, err)
						return
					}

					doc, err := entClient.Document.Query().Where(document.FilepathEQ(fileinfo.Path)).First(ctx)
					if err == nil && doc.UpdatedAt.After(fileinfo.ModifiedAt) {
						fmt.Printf("Document %s is up-to-date\n", fileinfo.Path)
						return
					}

					summary, err := llm.SummarizeFileContent(client, string(buf))
					if err != nil {
						errChan <- fmt.Errorf("failed to summarize content: %v", err)
						return
					}
					if len(summary) == 0 {
						fmt.Printf("Summary is empty: %s\ncontent:%s", fileinfo.Path, string(buf))
						return
					}
					vsDoc := &vectorstore.Document{
						Repository: config.Repository,
						Filepath:  fileinfo.Path,
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
		},
	}

	// Define flags and configuration settings for loaderCmd
	loadCmd.Flags().StringVarP(&outputFile, "output", "o", "repo_structure.json", "Output JSON file")
	loadCmd.Flags().StringVarP(&branch, "branch", "b", "main", "Branch to load the structure from")
	loadCmd.Flags().StringVarP(&commitHash, "commit", "c", "", "Specific commit hash to load the structure from")
	loadCmd.Flags().StringVarP(&openaiAPIKey, "api-key", "k", "", "OpenAI API key (can also set via OPENAI_API_KEY environment variable)")
	loadCmd.Flags().StringVar(&openaiModel, "model", "gpt-4o-mini", "OpenAI model to use for summarization")
	loadCmd.Flags().IntVar(&maxTokens, "max-tokens", 150, "Maximum number of tokens for the summary")
	loadCmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string (e.g., postgres://aicoder:aicoder@localhost:5432/aicoder)")

	return loadCmd
}



// upsertDocument inserts or updates a document in the PostgreSQL database.
func upsertDocument(ctx context.Context, entClient *ent.Client, path, description string, embedding []float32, repository string) error {
	vector := pgvector.NewVector(embedding)

	err := entClient.Document.Create().
		SetFilepath(path).
		SetRepository(repository).
		SetDescription(description).
		SetEmbedding(vector).
		SetUpdatedAt(time.Now()).
		OnConflictColumns(document.FieldRepository, document.FieldFilepath).
		UpdateNewValues().
		Exec(ctx)
	return err
}
