package review

import (
	"fmt"
	"log"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/planner"
	"github.com/nakamasato/aicoder/internal/reviewer"
	"github.com/spf13/cobra"
)

var (
	planFile   string
	reviewfile string
)

// Command returns the review command
func Command() *cobra.Command {
	reviewCmd := &cobra.Command{
		Use:   "review",
		Short: "Review changes based on the ReviewPlan",
		Run:   runReview,
	}

	// Add a flag for specifying the plan file
	reviewCmd.Flags().StringVarP(&planFile, "planfile", "p", "plan.json", "Path to the plan file to review")
	reviewCmd.Flags().StringVarP(&reviewfile, "reviewfile", "r", "review.json", "Path to the review file to save the review results")

	return reviewCmd
}

func runReview(cmd *cobra.Command, args []string) {
	if planFile == "" {
		log.Fatalln("plan file is required")
	}

	ctx := cmd.Context()
	config := config.GetConfig()
	llmClient := llm.NewClient(config.OpenAIAPIKey)

	// Load the plan file
	changesPlan, err := planner.LoadPlanFile[planner.ChangesPlan](planFile)
	if err != nil {
		log.Fatalf("failed to read plan file: %v", err)
	}

	// Review the changes
	review, err := reviewer.ReviewChanges(ctx, llmClient, changesPlan)
	if err != nil {
		fmt.Println("Error reviewing changes:", err)
	}

	// Save review to file
	if err := file.SaveObject(review, reviewfile); err != nil {
		log.Fatalf("failed to save review: %v", err)
	}
}
