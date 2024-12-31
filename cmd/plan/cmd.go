// cmd/plan/cmd.go
package plan

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
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

	return planCmd
}

func runPlan(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	config := config.GetConfig()
	goal := strings.Join(args, " ")

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

	store := vectorstore.New(entClient, client)

	res, err := store.Search(ctx, config.Repository, goal, 10)
	if err != nil {
		log.Fatalf("failed to search: %v", err)
	}

	// Generate plan using OpenAI
	plan, err := planner.Plan(ctx, client, entClient, goal, config.Repository, res.Documents)
	if err != nil {
		log.Fatalf("failed to generate plan: %v", err)
	}

	for _, change := range plan.Changes {
		fmt.Println("-----------------------------")
		fmt.Printf("Change %s:\n", change.Path)
		fmt.Printf("  Add: %s\n", change.Add)
		fmt.Printf("  Delete: %s\n", change.Delete)
		fmt.Printf("  Line: %d\n", change.Line)
	}

	// Save plan to file
	if err := planner.SavePlan(plan, outputFile); err != nil {
		log.Fatalf("failed to save plan: %v", err)
	}
}
