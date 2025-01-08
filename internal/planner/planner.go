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
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/openai/openai-go"
)

type Planner struct {
	llmClient llm.Client
	entClient *ent.Client
}

func NewPlanner(llmClient llm.Client, entClient *ent.Client) *Planner {
	return &Planner{
		llmClient: llmClient,
		entClient: entClient,
	}
}

// ChangesPlan is a list of changes each of which consists of PATH, ADD, DELETE, LINE and EXPLANATION.
// This strategy changes a file partially by specifying the content to be added or deleted at a specific line.
// This migtht not work well empirically, seemingly because the line number is not properly predicted.
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

// Change is a single change to be made to a file.
type Change struct {
	Path        string `json:"path" jsonschema_description:"Path to the file to be changed"`
	Add         string `json:"add" jsonschema_description:"Content to be added to the file"`
	Delete      string `json:"delete" jsonschema_description:"Content to be deleted from the file. When this is specified, the line number must be provided. The content to be deleted must match the content in the file at the specified line number."`
	Explanation string `json:"explanation" jsonschema_description:"Explanation for the change including why this change is needed and what is achieved by this change, etc."`
	LineNum     int    `json:"line" jsonschema_description:"Line number to insert the content. Line number starts from 1. To create a new file, set the line number to 0"`
}

// ChangeFilePlan is used to replace the entire content of the specified file with the modified content.
// This might work bettter than ChangesPlan.
type ChangeFilePlan struct {
	Path string `json:"path" jsonschema_description:"Path to the file to be changed"`
	// OldContent string `json:"old_content" jsonschema_description:"Original content of the file"`
	NewContent string `json:"new_content" jsonschema_description:"The new content of the file."`
}

type YesOrNo struct {
	Answer bool `json:"answer" jsonschema_description:"Answer to the yes or no question"`
}

type CodeValidation struct {
	IsValid         bool     `json:"is_valid" jsonschema_description:"true if the code syntax is valid."`
	InvalidSyntaxes []string `json:"reason" jsonschema_description:"The explanation of invalid syntaxes. Please write where is wrong and how to fix."`
}

var (
	ChangesPlanSchema    = GenerateSchema[ChangesPlan]()
	YesOrNoSchema        = GenerateSchema[YesOrNo]()
	ChangeFileSchema     = GenerateSchema[ChangeFilePlan]()
	CodeValidationSchema = GenerateSchema[CodeValidation]()

	ChangeFileSchemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("changes"),
		Description: openai.F("List of changes to be made to achieve the goal"),
		Schema:      openai.F(ChangeFileSchema),
		Strict:      openai.Bool(true),
	}

	ChangesPlanSchemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("changes"),
		Description: openai.F("List of changes to be made to meet the requirements"),
		Schema:      openai.F(ChangesPlanSchema),
		Strict:      openai.Bool(true),
	}

	CodeValidationSchemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("syntax_validation"),
		Description: openai.F("The result of judging Whether the syntax of the generated code is correct"),
		Schema:      openai.F(CodeValidationSchema),
		Strict:      openai.Bool(true),
	}

	YesOrNoSchemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("yes_or_no"),
		Description: openai.F("Answer to the yes or no question"),
		Schema:      openai.F(YesOrNoSchema),
		Strict:      openai.Bool(true),
	}
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

// generateGoalPrompt creates a prompt for OpenAI based on the goal and repository data.
func (p *Planner) GenerateGoalPrompt(ctx context.Context, goal string, files []file.File) (string, error) {
	// Create a comprehensive prompt
	var builder strings.Builder
	for _, f := range files {
		builder.WriteString(fmt.Sprintf("\n--------------------\nfilepath:%s\n--%s\n--- content end---", f.Path, f.Content))
	}
	prompt := fmt.Sprintf(PLANNER_GOAL_PROMPT, builder.String(), goal)

	return prompt, nil
}

// removeUnrelevantFiles removes irrelevant files from the list of files using LLM.
func (p *Planner) removeUnrelevantFiles(ctx context.Context, query string, files []file.File) ([]file.File, error) {

	var filteredFiles []file.File
	for _, f := range files {
		// Use LLM to determine if the file is relevant to the query
		content, err := p.llmClient.GenerateCompletion(ctx,
			[]openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage("You are a helpful assistant that determines if a file is relevant to a given query."),
				openai.UserMessage(fmt.Sprintf("Query: %s\nFile Content: %s", query, f.Content)),
			},
			YesOrNoSchemaParam)
		if err != nil {
			log.Printf("failed to determine file relevance: %v", err)
			continue
		}

		var yesOrNo YesOrNo
		if err = json.Unmarshal([]byte(content), &yesOrNo); err != nil {
			log.Printf("failed to unmarshal content to YesOrNo: %v", err)
			continue
		}

		if yesOrNo.Answer {
			log.Printf("%d. relevant file: %s\n", len(filteredFiles), f.Path)
			filteredFiles = append(filteredFiles, f)
		}
	}
	return filteredFiles, nil
}

// GenerateChangesPlanWithRetry generates ChangesPlan with validation and retry attempts
func (p *Planner) GenerateChangesPlanWithRetry(ctx context.Context, query string, maxAttempts int, files []file.File) (*ChangesPlan, error) {

	// identify files to change
	files, err := p.removeUnrelevantFiles(ctx, query, files)
	if err != nil {
		return nil, fmt.Errorf("failed to remove irrelevant files: %w", err)
	}

	prompt, err := p.GenerateGoalPrompt(ctx, query, files)
	if err != nil {
		return nil, fmt.Errorf("failed to generate goal prompt: %w", err)
	}

	changesPlan, err := p.GenerateChangesPlan(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if err = changesPlan.Validate(); err == nil {
			log.Println("Plan is valid")
			return changesPlan, nil
		}

		log.Printf("Invalid plan (attempt: %d): %v", attempt+1, err)
		prompt = fmt.Sprintf(REPLAN_PROMPT, query, changesPlan, err)
		changesPlan, err = p.GenerateChangesPlan(ctx, prompt)
		if err != nil {
			log.Printf("Failed to generate plan: %v", err)
			continue
		}
	}

	return nil, fmt.Errorf("failed to generate a valid plan after %d attempts", maxAttempts)
}

// GenerateChangesPlan creates a ChangesPlan based on the prompt.
// If you need validate and replan, use GenerateChangesPlanWithRetry function instead.
func (p *Planner) GenerateChangesPlan(ctx context.Context, prompt string) (*ChangesPlan, error) {

	content, err := p.llmClient.GenerateCompletion(ctx,
		[]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a helpful assistant that generates detailed action plans based on provided project information."),
			openai.UserMessage(prompt),
		},
		ChangesPlanSchemaParam)
	if err != nil {
		return nil, fmt.Errorf("failed to create GenerateCompletion: %w", err)
	}

	responseJSON, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		log.Printf("Error marshalling chat response: %v", err)
	} else {
		log.Printf("Chat completion response: %s", responseJSON)
	}

	var changesPlan ChangesPlan
	err = json.Unmarshal([]byte(content), &changesPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal changes plan: %w", err)
	}

	fmt.Printf("Plan: %s\n", changesPlan.String())

	return &changesPlan, nil
}

func (p *Planner) GenerateChangeFilePlanWithRetry(ctx context.Context, prompt, query string, maxAttempts int) (*ChangeFilePlan, error) {
	changesPlan, err := p.GenerateChangeFilePlan(ctx, openai.UserMessage(query), openai.SystemMessage(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// validation
		messages := []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You're a code validator to check whether the generated code is correct syntax"),
			openai.UserMessage(VALIDATE_FILE_PROMPT),
			openai.SystemMessage(changesPlan.NewContent),
		}

		content, err := p.llmClient.GenerateCompletion(
			ctx, messages, CodeValidationSchemaParam)

		if err != nil {
			log.Printf("failed to execute Chat.Completion: %v", err)
			continue
		}

		var validation CodeValidation
		if err = json.Unmarshal([]byte(content), &validation); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content to CodeValidation: %v", err)
		}
		if validation.IsValid {
			return changesPlan, nil
		}
		log.Printf("syntax validation failed. %d invalid syntaxes", len(validation.InvalidSyntaxes))
		for i, s := range validation.InvalidSyntaxes {
			log.Printf("%d. %s", i+1, s)
		}

		// regenerate
		changesPlan, err = p.GenerateChangeFilePlan(ctx,
			openai.SystemMessage("You're an experienced software engineer who is tasked to refactor/update the existing code."),
			openai.UserMessage(query),
			openai.SystemMessage(prompt),
			openai.SystemMessage(changesPlan.NewContent),
			openai.SystemMessage(fmt.Sprintf("The syntax validation failed the reasons are the followings:\n%s", strings.Join(validation.InvalidSyntaxes, "\n"))),
			openai.UserMessage("Please fix the syntax errors."),
		)
		if err != nil {
			log.Fatalf("failed to generate ChangesFilePlan: %v", err)
		}
	}

	return nil, fmt.Errorf("failed to generate a valid plan after %d attempts", maxAttempts)
}

// GenerateChangeFilePlan creates a ChangeFilePlan based on the prompt.
func (p *Planner) GenerateChangeFilePlan(ctx context.Context, prompts ...openai.ChatCompletionMessageParamUnion) (*ChangeFilePlan, error) {

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You're an experienced software engineer who is tasked to refactor/update the existing code."),
	}
	messages = append(messages, prompts...)

	content, err := p.llmClient.GenerateCompletion(ctx, messages, ChangeFileSchemaParam)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Chat.Completion: %v", err)
	}

	fmt.Printf("Answer: %s\n---\n", content)
	var changeFile ChangeFilePlan
	if err = json.Unmarshal([]byte(content), &changeFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content to ChanegFile: %v", err)
	}
	fmt.Printf("Parsed Answer: %s", changeFile)
	return &changeFile, nil
}

func LoadPlanFile[T any](planFile string) (*T, error) {
	data, err := os.ReadFile(planFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan T
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	return &plan, nil
}

// SavePlan saves the plan to a file.
func SavePlan[T any](plan *T, outputFile string) error {
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
