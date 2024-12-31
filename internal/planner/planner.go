package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/nakamasato/aicoder/internal/loader"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/openai/openai-go"
)

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
		content, err := loader.LoadFileContent(r.Document.Filepath)
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
	prompt := fmt.Sprintf(`You are a helpful assistant that generates detailed action plans based on provided project information.
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

func Plan(ctx context.Context, client *openai.Client, entClient *ent.Client, goal, repo string, docsWithScore *[]vectorstore.DocumentWithScore) (ChangesPlan, error) {

	prompt, err := generatePrompt(ctx, entClient, goal, repo, docsWithScore)
	if err != nil {
		return ChangesPlan{}, fmt.Errorf("failed to generate prompt: %w", err)
	}

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

func SavePlan(plan ChangesPlan, outputFile string) error {
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write plan to file: %w", err)
	}
	return nil
}
