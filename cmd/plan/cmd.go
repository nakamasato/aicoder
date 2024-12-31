// cmd/plan/cmd.go
package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/nakamasato/aicoder/internal/load"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/cobra"
)

// PlanResult represents the generated plan.
type PlanResult struct {
	Goal  string `json:"goal"`
	Steps []Step `json:"steps"`
}

// Step represents a single step in the plan.
type Step struct {
	Name string `json:"name"`
	Tool string `json:"tool"`
}

type ChangesPlan struct {
	Changes []Change `json:"changes" jsonschema_description:"List of changes to be made to achieve the goal"`
}

type Change struct {
	Path   string `json:"path" jsonschema_description:"Path to the file to be changed"`
	Add    string `json:"content" jsonschema_description:"Content to be added to the file"`
	Delete string `json:"delete" jsonschema_description:"Content to be deleted from the file"`
	Line   int    `json:"line" jsonschema_description:"Line number to insert the content"`
}

var (
	outputFile        string
	dbConnString      string
	openaiAPIKey      string
	ChangesPlanSchema = GenerateSchema[ChangesPlan]()
)

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// Command creates the plan command.
func Command() *cobra.Command {
	planCmd := &cobra.Command{
		Use:   "plan [goal]",
		Short: "Generate a plan based on the repository structure and the given goal.",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
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

			// Generate a detailed prompt based on the goal and repository structure
			prompt, err := generatePrompt(ctx, entClient, goal, config.Repository, res.Documents)
			if err != nil {
				log.Fatalf("failed to generate prompt: %v", err)
			}

			// Generate plan using OpenAI
			plan, err := generatePlan(ctx, client, prompt)
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
		},
	}

	// Define flags and configuration settings for planCmd
	planCmd.Flags().StringVarP(&outputFile, "output", "o", "plan.json", "Output JSON file for the generated plan")
	planCmd.Flags().StringVar(&dbConnString, "db-conn", "postgres://aicoder:aicoder@localhost:5432/aicoder?sslmode=disable", "PostgreSQL connection string")
	planCmd.Flags().StringVarP(&openaiAPIKey, "api-key", "k", "", "OpenAI API key (can also set via OPENAI_API_KEY environment variable)")

	return planCmd
}

// generatePrompt creates a prompt for OpenAI based on the goal and repository data.
func generatePrompt(ctx context.Context, entClient *ent.Client, goal, repo string, docsWithScore *[]vectorstore.DocumentWithScore) (string, error) {
	// Fetch relevant documents or summaries from the database
	docs, err := entClient.Document.
		Query().
		Where(document.RepositoryEQ(repo)).
		All(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch documents: %w", err)
	}

	var relevantDocs strings.Builder
	for _, r := range *docsWithScore {
		content, err := load.LoadFileContent(r.Document.Filepath)
		if err != nil {
			return "", fmt.Errorf("failed to load file content: %w", err)
		}
		relevantDocs.WriteString(fmt.Sprintf("\n--------------------\nfilepath:%s\n--%s\n--- content end---", r.Document.Filepath, content))
	}

	// Aggregate the descriptions to provide context
	var contextInfo strings.Builder
	for _, doc := range docs {
		contextInfo.WriteString(fmt.Sprintf("- %s: %s\n", doc.Filepath, doc.Description))
	}

	// Create a comprehensive prompt
	prompt := fmt.Sprintf(`
I have a project with the following components:
-----------------------
Files in the repository:
%s
-----------------------
Possibly relevant documents:
%s

------------------------
My goal is: %s

Based on the above information, please provide a detailed plan with actionable steps to achieve this goal. Please specify the existing file to change or create to achieve the goal.

Ensure that each step is clear and actionable for human review and execution.
`, contextInfo.String(), relevantDocs.String(), goal)

	return prompt, nil
}

// generatePlan uses OpenAI to generate a plan based on the provided prompt.
func generatePlan(ctx context.Context, client *openai.Client, prompt string) (ChangesPlan, error) {

	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("changes"),
		Description: openai.F("List of changes to be made to achieve the goal"),
		Schema:      openai.F(ChangesPlanSchema),
		Strict:      openai.Bool(true),
	}

	chat, err := client.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Model: openai.F(openai.ChatModelGPT4oMini),
			Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage("You are a helpful assistant that generates detailed action plans based on provided project information."),
				openai.UserMessage(prompt),
			}),
			ResponseFormat: openai.F[openai.ChatCompletionNewParamsResponseFormatUnion](
				openai.ResponseFormatJSONSchemaParam{
					Type:       openai.F(openai.ResponseFormatJSONSchemaTypeJSONSchema),
					JSONSchema: openai.F(schemaParam),
				},
			),
		})
	if err != nil {
		return ChangesPlan{}, fmt.Errorf("failed to create chat completion: %w", err)
	}

	if len(chat.Choices) == 0 {
		return ChangesPlan{}, fmt.Errorf("no response from OpenAI")
	}

	changesPlan := ChangesPlan{}
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &changesPlan)
	if err != nil {
		panic(err.Error())
	}

	return changesPlan, nil
}
