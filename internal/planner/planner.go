package planner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
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

func (c ChangesPlan) Validate() error {
	errorsMap := make(map[string][]error)
	changesPerFile := make(map[string]int)
	var err error
	for _, change := range c.Changes {
		changesPerFile[change.Path]++
		if change.Path == "" {
			errorsMap[change.Path] = append(errorsMap[change.Path], fmt.Errorf("path is required for all changes"))
		}
		if change.Add == "" && change.Delete == "" {
			errorsMap[change.Path] = append(errorsMap[change.Path], fmt.Errorf("either add or delete content is required for all changes"))
		}
		if change.LineNum < 0 {
			errorsMap[change.Path] = append(errorsMap[change.Path], fmt.Errorf("line number must be greater than or equal to 0"))
		}
		_, err := os.Stat(change.Path)
		if os.IsNotExist(err) && change.LineNum != 0 {
			errorsMap[change.Path] = append(errorsMap[change.Path], fmt.Errorf("file does not exist at path: %s", change.Path))
		}
		if err == nil && change.LineNum == 0 {
			errorsMap[change.Path] = append(errorsMap[change.Path], fmt.Errorf("file already exists at path: %s, need to specify the line num.", change.Path))
		}
		if change.Delete != "" {
			file, err := os.Open(change.Path)
			if err != nil {
				errorsMap[change.Path] = append(errorsMap[change.Path], fmt.Errorf("failed to open file: %v", err))
				continue
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			currentLine := 1
			found := false
			for scanner.Scan() {
				if currentLine == change.LineNum {
					if scanner.Text() != change.Delete {
						errorsMap[change.Path] = append(errorsMap[change.Path], fmt.Errorf("content to delete does not match at line %d", change.LineNum))
					} else {
						found = true
					}
					break
				}
				currentLine++
			}

			if !found {
				errorsMap[change.Path] = append(errorsMap[change.Path], fmt.Errorf("line number %d not found in file", change.LineNum))
			}

			if err := scanner.Err(); err != nil {
				errorsMap[change.Path] = append(errorsMap[change.Path], fmt.Errorf("error reading file: %v", err))
			}
		}
	}
	for path, count := range changesPerFile {
		if count > 1 {
			errorsMap[path] = append(errorsMap[path], fmt.Errorf("multiple changes for the same file"))
		}
	}
	if len(errorsMap) > 0 {
		var errorMessages []string
		for path, errors := range errorsMap {
			errorMessages = append(errorMessages, fmt.Sprintf("path: %s, errors: %v", path, errors))
		}
		err = fmt.Errorf("validation failed: %v", errorMessages)
	}
	return err
}

func (c ChangesPlan) String() string {
	jsonData, err := json.Marshal(c)
	if err != nil {
		log.Printf("Error marshalling ChangesPlan to JSON: %v", err)
		return ""
	}
	return string(jsonData)
}

type Change struct {
	Path        string `json:"path" jsonschema_description:"Path to the file to be changed"`
	Add         string `json:"add" jsonschema_description:"Content to be added to the file"`
	Delete      string `json:"delete" jsonschema_description:"Content to be deleted from the file"`
	Explanation string `json:"explanation" jsonschema_description:"Explanation for the change including why this change is needed and what is achieved by this change, etc."`
	LineNum     int    `json:"line" jsonschema_description:"Line number to insert the content. Line number starts from 1. To create a new file, set the line number to 0"`
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

func getRelevantDocsString(docsWithScore *[]vectorstore.DocumentWithScore) string {
	var relevantDocs strings.Builder
	for _, r := range *docsWithScore {
		content, err := loader.LoadFileContent(r.Document.Filepath)
		if err != nil {
			return ""
		}
		relevantDocs.WriteString(fmt.Sprintf("\n--------------------\nfilepath:%s\n--%s\n--- content end---", r.Document.Filepath, content))
	}
	return relevantDocs.String()
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

	// Aggregate the descriptions to provide context
	var contextInfo strings.Builder
	for _, doc := range docs {
		contextInfo.WriteString(fmt.Sprintf("- %s: %s\n", doc.Filepath, doc.Description))
	}

	// Create a comprehensive prompt
	prompt := fmt.Sprintf(PLANNER_PROMPT, contextInfo.String(), getRelevantDocsString(docsWithScore), goal)

	return prompt, nil
}

func Plan(ctx context.Context, client *openai.Client, entClient *ent.Client, goal, repo string, docsWithScore *[]vectorstore.DocumentWithScore, maxAttempts int) (ChangesPlan, error) {
	prompt, err := generatePrompt(ctx, entClient, goal, repo, docsWithScore)
	if err != nil {
		return ChangesPlan{}, fmt.Errorf("failed to generate prompt: %w", err)
	}

	changesPlan, err := generatePlan(ctx, prompt, client)
	if err != nil {
		return ChangesPlan{}, fmt.Errorf("failed to generate plan: %w", err)
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err = changesPlan.Validate(); err == nil {
			log.Println("Plan is valid")
			return changesPlan, nil
		}

		log.Printf("Invalid plan (attempt: %d): %v", attempt+1, err)
		prompt = fmt.Sprintf(REPLAN_PROMPT, goal, changesPlan, err)
		changesPlan, err = generatePlan(ctx, prompt, client)
		if err != nil {
			log.Printf("Failed to generate plan: %v", err)
			continue
		}
	}

	return ChangesPlan{}, fmt.Errorf("failed to generate a valid plan after %d attempts", maxAttempts)
}

func generatePlan(ctx context.Context, prompt string, client *openai.Client) (ChangesPlan, error) {

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

	responseJSON, err := json.MarshalIndent(chat.Choices[0].Message.Content, "", "  ")
	if err != nil {
		log.Printf("Error marshalling chat response: %v", err)
	} else {
		log.Printf("Chat completion response: %s", responseJSON)
	}

	changesPlan := ChangesPlan{}
	err = json.Unmarshal([]byte(chat.Choices[0].Message.Content), &changesPlan)
	if err != nil {
		return ChangesPlan{}, fmt.Errorf("failed to unmarshal changes plan: %w", err)
	}

	fmt.Printf("Plan: %s\n", changesPlan.String())

	return changesPlan, nil
}

// SavePlan saves the plan to a file.
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
