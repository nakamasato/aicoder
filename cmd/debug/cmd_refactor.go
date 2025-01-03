package debug

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

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

	query := message
	prompt := fmt.Sprintf("The code content:\n--- %s start ---\n%s\n---- %s end ----", filename, string(data), filename)

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("changes"),
		Description: openai.F("List of changes to be made to achieve the goal"),
		Schema:      openai.F(planner.ChangeFileSchema),
		Strict:      openai.Bool(true),
	}

	chat, err := client.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage("You're an experienced software engineer who is tasked to refactor/update the existing code."),
				openai.UserMessage(query),
				openai.SystemMessage(prompt),
			}),
			Model: openai.F(openai.ChatModelGPT4oMini),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONSchemaParam{
					Type: openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
					JSONSchema: openai.F(schemaParam),
				},
			),
		},
	)
	if err != nil {
		log.Fatalf("failed to execute Chat.Completion: %v", err)
	}

	fmt.Printf("Answer: %s\n---\n", chat.Choices[0].Message.Content)
	var changeFile planner.ChangeFile
	if err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &changeFile); err != nil {
		log.Fatalf("failed to unmarshal content to ChanegFile: %v", err)
	}
	fmt.Printf("Parsed Answer: %s", changeFile)

	plan, err := planner.Plan(ctx, client, entClient, query, prompt, maxAttempts)
	if err != nil {
		log.Fatalf("failed to generate plan: %v", err)
	}

	// for _, change := range plan.Changes {
	// 	fmt.Println("-----------------------------")
	// 	fmt.Printf("Change %s:\n", change.Path)
	// 	fmt.Printf("  Add: %s\n", change.Add)
	// 	fmt.Printf("  Delete: %s\n", change.Delete)
	// 	fmt.Printf("  Line: %d\n", change.LineNum)
	// }

	// Save plan to file
	if err := planner.SavePlan(plan, outputFile); err != nil {
		log.Fatalf("failed to save plan: %v", err)
	}
}
