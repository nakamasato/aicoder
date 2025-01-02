// cmd/plan/cmd.go
package plan

import (
	"fmt"
	"log"
	"strings"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/loader"
	"github.com/nakamasato/aicoder/internal/planner"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
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
	goal := strings.Join(args, " ")

	// Initialize OpenAI client
	if openaiAPIKey != "" {
		config.OpenAIAPIKey = openaiAPIKey
	}
	if config.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}
	client := openai.NewClient(option.WithAPIKey(config.OpenAIAPIKey))

	// Initialize entgo client
	entClient, err := ent.Open("postgres", dbConnString)
	if err != nil {
		log.Fatalf("failed opening connection to postgres: %v", err)
	}
	defer entClient.Close()

	store := vectorstore.New(entClient, client)

	res, err := store.Search(ctx, config.Repository, goal, 10)
	if err != nil {
		log.Fatalf("failed to search: %v", err)
	}

	// Load file content
	filepaths := res.FilePaths()
	var files file.Files
	for _, path := range filepaths {
		content, err := loader.LoadFileContent(path)
		if err != nil {
			log.Fatalf("failed to load file content: %v", err)
		}
		files = append(files, &file.File{Path: path, Content: content})
	}
	prompt, err := planner.GenerateGoalPrompt(ctx, client, entClient, goal, config.Repository, files)
	if err != nil {
		log.Fatalf("failed to generate goal prompt: %v", err)
	}
	plan, err := planner.Plan(ctx, client, entClient, goal, prompt, maxAttempts)
	if err != nil {
		log.Fatalf("failed to generate plan: %v", err)
	}

	for _, change := range plan.Changes {
		fmt.Println("-----------------------------")
		fmt.Printf("Change %s:\n", change.Path)
		fmt.Printf("  Add: %s\n", change.Add)
		fmt.Printf("  Delete: %s\n", change.Delete)
		fmt.Printf("  Line: %d\n", change.LineNum)
	}

	// Save plan to file
	if err := planner.SavePlan(plan, outputFile); err != nil {
		log.Fatalf("failed to save plan: %v", err)
	}
}
