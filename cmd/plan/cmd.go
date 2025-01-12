// cmd/plan/cmd.go
package plan

import (
	"fmt"
	"log"
	"strings"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/loader"
	"github.com/nakamasato/aicoder/internal/planner"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/spf13/cobra"
)

var (
	outputFile   string
	dbConnString string
	openaiAPIKey string
	maxAttempts  int
)

// Command creates the plan command.
func Command() *cobra.Command {
	planCmd := &cobra.Command{
		Use:   "plan [goal]",
		Short: "Generate a plan based on the repository structure and the given goal.",
		Args:  cobra.MinimumNArgs(1),
		Run:   runPlan,
	}

	// Define flags and configuration settings for planCmd
	planCmd.Flags().StringVarP(&outputFile, "output", "o", "plan.json", "Output JSON file for the generated plan")
	planCmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string")
	planCmd.Flags().StringVarP(&openaiAPIKey, "api-key", "k", "", "OpenAI API key (can also set via OPENAI_API_KEY environment variable)")
	planCmd.Flags().IntVarP(&maxAttempts, "max-attempts", "m", 10, "Maximum number of attempts to generate a plan")

	return planCmd
}

func runPlan(cmd *cobra.Command, args []string) {
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
	llmClient := llm.NewClient(config.OpenAIAPIKey)

	// Initialize entgo client
	entClient, err := ent.Open("postgres", dbConnString)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer entClient.Close()

	store := vectorstore.New(entClient, llmClient)

	res, err := store.Search(ctx, config.Repository, config.CurrentContext, query, 10)
	if err != nil {
		log.Fatalf("failed to search: %v", err)
	}

	// Load file content
	var files []file.File
	fmt.Printf("Found %d files\n", len(*res.Documents))
	for i, doc := range *res.Documents {
		fmt.Printf("%d: %s (score: %.2f)\n", i, doc.Document.Filepath, doc.Score)
		content, err := loader.LoadFileContent(doc.Document.Filepath)
		if err != nil {
			log.Fatalf("failed to load file content. you might need to refresh loader by `aicoder load -r`: %v", err)
		}
		files = append(files, file.File{Path: doc.Document.Filepath, Content: content})
	}

	// Generate plan based on the query and the files
	plnr := planner.NewPlanner(llmClient, entClient)
	p, err := plnr.GenerateChangesPlan2(ctx, query, maxAttempts, files)
	if err != nil {
		log.Fatalf("failed to generate plan: %v", err)
	}

	for _, change := range p.Changes {
		fmt.Printf("Change %s (type:%s, name:%s)\n", change.Block.Path, change.Block.TargetType, change.Block.TargetName)
	}

	// Save plan to file
	if err := planner.SavePlan[planner.ChangesPlan](p, outputFile); err != nil {
		log.Fatalf("failed to save plan: %v", err)
	}
}
