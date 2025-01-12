package debug

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/applier"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/planner"
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
		Use:   "refactor [message]",
		Short: "Refactor the code using OpenAI suggestions",
		Args:  cobra.MinimumNArgs(1),
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
	message := strings.Join(args, " ")

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

	query := message
	prompt := fmt.Sprintf("The code content:\n--- %s start ---\n%s\n---- %s end ----", filename, string(data), filename)

	plnr := planner.NewPlanner(llm.NewClient(config.OpenAIAPIKey), entClient)

	changeFilePlan, err := plnr.GenerateChangeFilePlanWithRetry(ctx, prompt, query, 10)
	if err != nil {
		log.Fatalf("failed to generate plan: %v", err)
	}

	// Print plan
	planJSON, err := json.MarshalIndent(changeFilePlan, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal plan: %v", err)
	}
	fmt.Println(string(planJSON))

	// Save plan to file
	// if err := planner.SavePlan[planner.ChangeFilePlan](*changeFilePlan, outputFile); err != nil {
	// 	log.Fatalf("failed to save plan: %v", err)
	// }
	// fmt.Printf("Successfully saved to %s", outputFile)
	if err = applier.ApplyChangeFilePlan(changeFilePlan, changeFilePlan.Path); err != nil {
		log.Fatalf("failed to apply plan: %v", err)
	}
}
