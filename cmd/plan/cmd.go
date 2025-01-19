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
	"github.com/nakamasato/aicoder/internal/planner"
	"github.com/nakamasato/aicoder/internal/retriever"
	"github.com/nakamasato/aicoder/internal/reviewer"
	"github.com/nakamasato/aicoder/internal/summarizer"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/spf13/cobra"
)

var (
	outputFile   string
	dbConnString string
	openaiAPIKey string
	maxAttempts  int
	reviewFile   string
)

// Command creates the plan command.
func Command() *cobra.Command {
	planCmd := &cobra.Command{
		Use:   "plan [goal]",
		Short: "Generate a plan based on the repository structure and the given goal.",
		Run:   runPlan,
	}

	// Define flags and configuration settings for planCmd
	planCmd.Flags().StringVarP(&outputFile, "output", "o", "plan.json", "Output JSON file for the generated plan")
	planCmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string")
	planCmd.Flags().StringVarP(&openaiAPIKey, "api-key", "k", "", "OpenAI API key (can also set via OPENAI_API_KEY environment variable)")
	planCmd.Flags().IntVarP(&maxAttempts, "max-attempts", "m", 10, "Maximum number of attempts to generate a plan")
	planCmd.Flags().StringVar(&reviewFile, "review", "", "Optional review file to improve the plan")

	return planCmd
}

func runPlan(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	config := config.GetConfig()
	if len(args) == 0 && reviewFile == "" {
		log.Fatal("either [goal] or review must be specified.")
	}

	var query string

	// set query from plan query and load review data if review file is provided
	var review reviewer.ReviewResult
	var plan planner.ChangesPlan
	if reviewFile != "" {
		if err := file.ReadObject(reviewFile, &review); err != nil {
			log.Fatalf("failed to read review file: %v", err)
		}

		// Load the plan file
		if err := file.ReadObject(outputFile, &plan); err != nil {
			log.Fatalf("failed to read plan file: %v", err)
		}
		if plan.Id != review.PlanId {
			log.Fatalf("plan ID mismatch: %s != %s", plan.Id, review.PlanId)
		}
		query = plan.Query
	} else {
		query = strings.Join(args, " ")
	}

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
	vr := retriever.NewVectorstoreRetriever(store, file.DefaultFileReader{}, &config)
	summary, err := summarizer.ReadSummary(ctx, "repo_summary.json")
	if err != nil {
		log.Fatalf("failed to read summary: %v", err)
	}
	lr := retriever.NewLLMRetriever(llmClient, file.DefaultFileReader{}, &config, summary)
	r := retriever.NewEnsembleRetriever(vr, lr)
	files, err := r.Retrieve(ctx, query)
	if err != nil {
		log.Fatalf("failed to retrieve files: %v", err)
	}

	// Generate plan based on the query and the files
	plnr := planner.NewPlanner(llmClient, entClient)
	p, err := plnr.GeneratePlan(ctx, query, maxAttempts, files, &plan, review.Comment)
	if err != nil {
		log.Fatalf("failed to generate plan: %v", err)
	}

	for _, change := range p.Changes {
		fmt.Printf("Change %s (type:%s, name:%s)\n", change.Block.Path, change.Block.TargetType, change.Block.TargetName)
	}

	// Save plan to file
	if err := file.SaveObject(p, outputFile); err != nil {
		log.Fatalf("failed to save plan: %v", err)
	}
}
