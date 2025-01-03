package debug

import (
	"fmt"
	"log"
	"os"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/planner"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/cobra"
)

var (
	outputFile   string
	dbConnString string
	openaiAPIKey string
	filename     string
	maxAttempts  int
)

func refactorCommand() *cobra.Command {
	refactorCmd := &cobra.Command{
		Use:   "refactor",
		Short: "Refactor the code using OpenAI suggestions",
		Run:   runRefactor,
	}
	refactorCmd.Flags().StringVarP(&outputFile, "output", "o", "plan.json", "Output JSON file for the generated plan")
	refactorCmd.Flags().StringVarP(&openaiAPIKey, "api-key", "k", "", "OpenAI API key (can also set via OPENAI_API_KEY environment variable)")
	refactorCmd.Flags().StringVarP(&filename, "filename", "f", "", "File to refactor")
	refactorCmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string")
	refactorCmd.Flags().IntVarP(&maxAttempts, "max-attempts", "m", 10, "Maximum number of attempts to generate a plan")

	return refactorCmd
}

func runRefactor(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	config := config.GetConfig()

	if filename == "" {
		log.Fatal("Please provide a filename to refactor")
	}

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

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading file: %v\n", err)
	}

	query := "Refactor the code. If there's no need of refactoring, please no changes needed."
	prompt := fmt.Sprintf("Refactor the code\n--- %s start ---\n%s\n---- %s end ----", filename, string(data), filename)

	plan, err := planner.Plan(ctx, client, entClient, query, prompt, maxAttempts)
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
