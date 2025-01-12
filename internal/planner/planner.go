package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

// ChangesPlan is a list of changes each of which consists of BlockChange.
// This strategy changes a file partially by specifying the new content in each block.
// Block is different in each language. For example, in Go, a block is a function. In HCL, a block is a resource.
// This will replace ChangeFilePlan.
type ChangesPlan struct {
	Changes []BlockChange `json:"changes" jsonschema_description:"List of changes to be made to meet the requirements"`
}

// NecessaryChangesPlan is a list of changes to be made to a file.
// This is a more comprehensive plan before making ChangesPlan.
type NecessaryChangesPlan struct {
	Steps []string `json:"steps" jsonschema_description:"List of steps to be made to meet the requirements. What kind of changes are necessary to achieve the goal."`
}

func (c ChangesPlan) String() string {
	jsonData, err := json.Marshal(c)
	if err != nil {
		fmt.Printf("Error marshalling ChangesPlan to JSON: %v", err)
		return ""
	}
	return string(jsonData)
}

type BlockChange struct {
	Block      Block  `json:"block" jsonschema_description:"The target block to be changed"`
	NewContent string `json:"new_content" jsonschema_description:"The new content of the block. Leave it empty to keep the current content and just update comment."`
	NewComment string `json:"new_comment" jsonschema_description:"The new comment of the block. Leave it empty to keep the current comment and just update content. HCL file does not support updating comment yet."`
}

type TargetBlocks struct {
	Changes []Block `json:"changes" jsonschema_description:"List of candidate blocks to be modified to achieve the goal"`
}

type Block struct {
	Path       string `json:"path" jsonschema_description:"Path to the file to be changed"`
	TargetType string `json:"target_type" jsonschema_description:"Type of the target block. e.g. class, function, struct, variable, module, etc"`
	TargetName string `json:"target_name" jsonschema_description:"Name of the target block. e.g. Command, runPlan, Client"`
	Content    string `json:"content" jsonschema_description:"The content of the block"`
}

type LineNum struct {
	StartLine int `json:"start_line" jsonschema_description:"Start line number of the target block"`
	EndLine   int `json:"end_line" jsonschema_description:"End line number of the target block"`
}

// ChangeFilePlan is used to replace the entire content of the specified file with the modified content.
// This might work bettter than ChangesPlan.
type ChangeFilePlan struct {
	Path       string `json:"path" jsonschema_description:"Path to the file to be changed"`
	NewContent string `json:"new_content" jsonschema_description:"The new content of the target block. Leave it empty to keep the current content and just update comment."`
	NewComment string `json:"new_comment" jsonschema_description:"The new comment of the target block. Leave it empty to keep the current comment and just update content. HCL file does not support updating comment yet."`
}

type YesOrNo struct {
	Answer bool `json:"answer" jsonschema_description:"Answer to the yes or no question"`
}

type CodeValidation struct {
	IsValid         bool     `json:"is_valid" jsonschema_description:"true if the code syntax is valid."`
	InvalidSyntaxes []string `json:"reason" jsonschema_description:"The explanation of invalid syntaxes. Please write where is wrong and how to fix."`
}

var (
	ChangesPlanSchema          = GenerateSchema[ChangesPlan]()
	TargetBlocksSchema         = GenerateSchema[TargetBlocks]()
	LinenumSchema              = GenerateSchema[LineNum]()
	YesOrNoSchema              = GenerateSchema[YesOrNo]()
	ChangeFileSchema           = GenerateSchema[ChangeFilePlan]()
	CodeValidationSchema       = GenerateSchema[CodeValidation]()
	NecessaryChangesPlanSchema = GenerateSchema[NecessaryChangesPlan]()

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

	TargetBlocksSchemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("block_changes"),
		Description: openai.F("List of changes to be made to achieve the goal"),
		Schema:      openai.F(TargetBlocksSchema),
		Strict:      openai.Bool(true),
	}

	LinenumSchemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("line_num"),
		Description: openai.F("The start and end line number of the target location"),
		Schema:      openai.F(LinenumSchema),
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

	NecessaryChangesPlanSchemaParam = openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        openai.F("necessary_changes"),
		Description: openai.F("List of steps to be made to meet the requirements. What kind of changes are necessary to achieve the goal."),
		Schema:      openai.F(NecessaryChangesPlanSchema),
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

// GeneratePromptWithFiles creates a prompt to extract blocks of the given files to modify
func (p *Planner) GeneratePromptWithFiles(ctx context.Context, prompt, goal string, files []file.File, fileBlocks map[string][]Block) (string, error) {
	// Create a comprehensive prompt
	var builder strings.Builder
	for _, f := range files {
		var blockStr string
		blocks, ok := fileBlocks[f.Path]
		if !ok {
			blockStr = "No blocks found"
		} else {
			for _, b := range blocks {
				blockStr += fmt.Sprintf("\n- %s: %s", b.TargetType, b.TargetName)
			}
		}

		builder.WriteString(fmt.Sprintf("\n--------------------\nfilepath:%s\n--%s\n--- content end---\n--- blocks ---\n%s", f.Path, f.Content, blockStr))
	}
	return fmt.Sprintf(prompt, builder.String(), goal), nil
}

// removeUnrelevantFiles removes irrelevant files from the list of files using LLM.
func (p *Planner) removeUnrelevantFiles(ctx context.Context, query string, files []file.File) ([]file.File, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	filteredFiles := make([]file.File, 0, len(files))
	errChan := make(chan error, len(files))

	for _, f := range files {
		wg.Add(1)
		go func(f file.File) {
			defer wg.Done()
			// Use LLM to determine if the file is relevant to the query
			content, err := p.llmClient.GenerateCompletion(ctx,
				[]openai.ChatCompletionMessageParamUnion{
					openai.SystemMessage("You are a helpful assistant that determines if a file is relevant to a given query."),
					openai.UserMessage(fmt.Sprintf("Query: %s\nFile Content: %s", query, f.Content)),
				},
				YesOrNoSchemaParam)
			if err != nil {
				fmt.Printf("failed to determine file relevance: %v", err)
				errChan <- err
				return
			}

			var yesOrNo YesOrNo
			if err = json.Unmarshal([]byte(content), &yesOrNo); err != nil {
				fmt.Printf("failed to unmarshal content to YesOrNo: %v", err)
				errChan <- err
				return
			}

			if yesOrNo.Answer {
				mu.Lock()
				fmt.Printf("%d. relevant file: %s\n", len(filteredFiles), f.Path)
				filteredFiles = append(filteredFiles, f)
				mu.Unlock()
			}
		}(f)
	}

	wg.Wait()
	close(errChan)

	// Collect errors if needed
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return filteredFiles, nil
}

// GenerateBlockChangePlan generates a plan to change a block of code.
// Use an appropriate prompt template for each language.
func (p *Planner) GenerateBlockChangePlan(ctx context.Context, promptTemplate string, block Block, blockContent string) (*BlockChange, error) {
	content, err := p.llmClient.GenerateCompletion(ctx,
		[]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You're an experienced software engineer who is tasked to refactor/update the existing code."),
			openai.UserMessage(fmt.Sprintf(promptTemplate, block.TargetName, block.Path, blockContent)),
		},
		ChangeFileSchemaParam)
	if err != nil {
		return nil, fmt.Errorf("failed to create GenerateCompletion: %w", err)
	}

	var change ChangeFilePlan
	err = json.Unmarshal([]byte(content), &change)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal changes plan: %w", err)
	}

	plan := &BlockChange{
		Block:      block,
		NewContent: change.NewContent,
		NewComment: change.NewComment,
	}

	return plan, nil
}

// GenerateChangesPlan2 generates ChangesPlan for Go language.
func (p *Planner) GenerateChangesPlan2(ctx context.Context, query string, maxAttempts int, files []file.File) (*ChangesPlan, error) {

	// identify files to change
	files, err := p.removeUnrelevantFiles(ctx, query, files)
	if err != nil {
		return nil, fmt.Errorf("failed to remove irrelevant files: %w", err)
	}

	// identify blocks to change
	fileBlocks := map[string][]Block{}
	for i, f := range files {
		fmt.Printf("File %d: %s\n", i+1, f.Path)
		if filepath.Ext(f.Path) == ".go" {
			functions, _, err := file.ParseGo(f.Path)
			if err != nil {
				fmt.Printf("failed to parse go file: %v", err)
				continue
			}
			for _, fn := range functions {
				fmt.Printf("Function: %s\n", fn.Name)
				fileBlocks[f.Path] = append(fileBlocks[f.Path], Block{Path: f.Path, TargetType: "function", TargetName: fn.Name, Content: fn.Content})
			}
		} else if filepath.Ext(f.Path) == ".hcl" || filepath.Ext(f.Path) == ".tf" {
			blocks, _, err := file.ParseHCL(f.Path)
			if err != nil {
				fmt.Printf("failed to parse hcl file: %v", err)
				continue
			}
			for _, b := range blocks {
				fmt.Printf("Block: Type:%s, Labels:%s\n", b.Type, strings.Join(b.Labels, ","))
				fileBlocks[f.Path] = append(fileBlocks[f.Path], Block{Path: f.Path, TargetType: b.Type, TargetName: strings.Join(b.Labels, ","), Content: b.Content})
			}
		}
	}

	// Make a Plan: which file to change and what to change
	prompt, err := p.GeneratePromptWithFiles(ctx, NECESSARY_CHANGES_PLAN_PROMPT, query, files, fileBlocks)
	if err != nil {
		return nil, fmt.Errorf("failed to generate goal prompt: %w", err)
	}
	content, err := p.llmClient.GenerateCompletion(ctx,
		[]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You're an experienced software engineer who is tasked to refactor/update the existing code."),
			openai.UserMessage(prompt),
		},
		NecessaryChangesPlanSchemaParam,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GenerateCompletion: %w", err)
	}
	var plan NecessaryChangesPlan
	if err = json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal necessary changes plan: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create GenerateCompletion: %w", err)
	}

	// Generate ChangesPlan
	changesPlan := &ChangesPlan{}
	for i, step := range plan.Steps {
		fmt.Printf("Step %d: %s\n", i+1, step)

		// identify blocks to change
		prompt_block, err := p.GeneratePromptWithFiles(ctx, PLANNER_EXTRACT_BLOCK_FOR_STEP_PROMPT, query, files, fileBlocks)
		if err != nil {
			return nil, fmt.Errorf("failed to generate goal prompt: %w", err)
		}

		// get blocks
		blocks, err := p.getBlocks(ctx, prompt_block)
		if err != nil {
			return nil, fmt.Errorf("failed to get blocks: %w", err)
		}
		fmt.Printf("Step %d: Got %d blocks\n", i+1, len(blocks.Changes))

		for _, blkToChange := range blocks.Changes {
			// TODO: create Planner interface and implement planner for each Language
			if filepath.Ext(blkToChange.Path) == ".go" {
				// Use function as a unit of block for go
				blocks := fileBlocks[blkToChange.Path]
				for _, blk := range blocks {
					if blk.TargetName == blkToChange.TargetName && blk.TargetType == blkToChange.TargetType {
						fmt.Printf("Step %d: Matched Block Go path:%s, type:%s, name:%s\n", i+1, blk.Path, blk.TargetType, blk.TargetName)
						blkChange, err := p.GenerateBlockChangePlan(ctx, GENERATE_FUNCTION_CHANGES_PLAN_PROMPT_GO, blkToChange, blk.Content)
						if err != nil {
							log.Panicln("failed to generate plan: %w", err)
							return nil, fmt.Errorf("failed to generate plan: %w", err)
						}
						changesPlan.Changes = append(changesPlan.Changes, *blkChange)
						break
					}
				}
			} else if filepath.Ext(blkToChange.Path) == ".hcl" || filepath.Ext(blkToChange.Path) == ".tf" {
				// Use block as a unit of block for hcl
				blocks := fileBlocks[blkToChange.Path]
				for _, blk := range blocks {
					if blk.TargetName == blkToChange.TargetName && blk.TargetType == blkToChange.TargetType { // variable, resource, module, etc.
						fmt.Printf("Step %d: Matched Block HCL path:%s, type:%s, name:%s\n", i+1, blk.Path, blk.TargetType, blk.TargetName)
						blkChange, err := p.GenerateBlockChangePlan(ctx, GENERATE_BLOCK_CHANGES_PLAN_PROMPT_HCL, blkToChange, blk.Content)
						if err != nil {
							log.Panicln("failed to generate plan: %w", err)
							return nil, fmt.Errorf("failed to generate plan: %w", err)
						}
						changesPlan.Changes = append(changesPlan.Changes, *blkChange)
						break
					}
				}
				// TODO: enable to change attr in hcl
			} else {
				log.Printf("Step %d: unsupported file type %s\n", i+1, blkToChange.Path)
				continue
			}
			fmt.Printf("Step %d: Block path:%s type:%s name:%s\n", i+1, blkToChange.Path, blkToChange.TargetType, blkToChange.TargetName)
		}
	}

	return changesPlan, nil
}

func (p *Planner) getBlocks(ctx context.Context, prompt string) (*TargetBlocks, error) {

	content, err := p.llmClient.GenerateCompletion(ctx,
		[]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
		TargetBlocksSchemaParam)
	if err != nil {
		return nil, fmt.Errorf("failed to create GenerateCompletion: %w", err)
	}

	var blks TargetBlocks
	err = json.Unmarshal([]byte(content), &blks)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal changes plan: %w", err)
	}

	return &blks, nil
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

	var changesPlan ChangesPlan
	err = json.Unmarshal([]byte(content), &changesPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal changes plan: %w", err)
	}

	fmt.Printf("Plan: %d\n", len(changesPlan.Changes))

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
			fmt.Printf("failed to execute Chat.Completion: %v", err)
			continue
		}

		var validation CodeValidation
		if err = json.Unmarshal([]byte(content), &validation); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content to CodeValidation: %v", err)
		}
		if validation.IsValid {
			return changesPlan, nil
		}
		fmt.Printf("syntax validation failed. %d invalid syntaxes", len(validation.InvalidSyntaxes))
		for i, s := range validation.InvalidSyntaxes {
			fmt.Printf("%d. %s", i+1, s)
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
