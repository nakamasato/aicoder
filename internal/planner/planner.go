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

	"github.com/google/uuid"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/summarizer"
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
	Id      string        `json:"id" jsonschema_description:"ID of the plan"`
	Query   string        `json:"query" jsonschema_description:"The goal of the changes"`
	Changes []BlockChange `json:"changes" jsonschema_description:"List of changes to be made to meet the requirements"`
}

type ActionType string

const (
	ActionTypeAdd         ActionType = "add"
	ActionTypeUpdate      ActionType = "update"
	ActionTypeDelete      ActionType = "delete"
	ActionTypeInvestigate ActionType = "investigate"
)

type Action struct {
	Step string     `json:"step" jsonschema_description:"Actual step to be made to meet the requirements"`
	Type ActionType `json:"type" jsonschema_description:"Type of the action"`
}

// ActionPlan is a list of changes to be made to a file.
// This is a more comprehensive plan before making ChangesPlan.
type ActionPlan struct {
	InvestigateSteps []string `json:"investigate_steps" jsonschema_description:"List of steps to investigate and collect relevant information in advance before planning actual file changes"`
	ChangeSteps      []string `json:"change_steps" jsonschema_description:"List of steps to make changes to block/file to meet the requirements."`
}

func (c ChangesPlan) String() string {
	jsonData, err := json.Marshal(c)
	if err != nil {
		fmt.Printf("Error marshalling ChangesPlan to JSON: %v", err)
		return ""
	}
	return string(jsonData)
}

// BlockChange is a change to a block of code.
type BlockChange struct {
	Block      Block  `json:"block" jsonschema_description:"The target block to be changed"`
	NewContent string `json:"new_content" jsonschema_description:"The new content of the block. Leave it empty to keep the current content and just update comment."`
	NewComment string `json:"new_comment" jsonschema_description:"The new comment of the block that is written above the block. Leave it empty to keep the current comment and just update content. HCL file does not support updating comment yet."`
}

// TargetBlocks is a list of candidate blocks to be modified to achieve the goal.
type TargetBlocks struct {
	Changes []Block `json:"changes" jsonschema_description:"List of candidate blocks to be modified to achieve the goal"`
}

// Block represents a block of code to be changed.
type Block struct {
	Path       string `json:"path" jsonschema_description:"Path to the file to be changed"`
	TargetType string `json:"target_type" jsonschema_description:"Type of the target block. e.g. file, class, function, struct, variable, module, etc"`
	TargetName string `json:"target_name" jsonschema_description:"Name of the target block. Please set file path when TargetType is 'file'. e.g. Command, runPlan, Client"`
	Content    string `json:"content" jsonschema_description:"The content of the block"`
}

// ChangeDiff is diff of the block of code.
type ChangeDiff struct {
	NewContent string `json:"new_content" jsonschema_description:"The new content of the target block. Leave it empty to keep the current content and just update comment."`
	NewComment string `json:"new_comment" jsonschema_description:"The new comment of the target block that is written above the block. Leave it empty to keep the current comment and just update content. HCL file does not support updating comment yet."`
}

var (
	ChangeDiffSchemaParam          = llm.GenerateJsonSchemaParam[ChangeDiff]("changes", "List of changes to be made to achieve the goal")
	TargetBlocksSchemaParam        = llm.GenerateJsonSchemaParam[TargetBlocks]("block_changes", "List of changes to be made to achieve the goal")
	ActionPlanSchemaParam          = llm.GenerateJsonSchemaParam[ActionPlan]("action_plans", "There are two parts, investigation steps and change steps. First, in the investigation steps, please check the relevant file contents to collect information in advance. The collected information is passed to the next steps (change step) to plan file changes in . Usually steps are less than or equal to 5.")
	InvestigationResultSchemaParam = llm.GenerateJsonSchemaParam[InvestigationResult]("investigation_result", "The result of the investigation. Please provide the necessary information or pieces of contents from the relevant files.")
)

func makeFileBlocksString(fileBlocks map[string][]Block) string {
	var builder strings.Builder
	for _, blocks := range fileBlocks {
		for _, b := range blocks {
			builder.WriteString(fmt.Sprintf("\n- %s: %s", b.TargetType, b.TargetName))
		}
	}
	return builder.String()
}

// generateBlockPromptWithFiles creates a prompt to extract blocks of the given files to modify
func (p *Planner) generateBlockPromptWithFiles(prompt, goal string, files []file.File, fileBlocks map[string][]Block) (string, error) {
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
				llm.YesOrNoSchemaParam)
			if err != nil {
				fmt.Printf("failed to determine file relevance: %v", err)
				errChan <- err
				return
			}

			var yesOrNo llm.YesOrNo
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
func (p *Planner) GenerateBlockChangePlan(ctx context.Context, promptTemplate string, block Block, blockContent string, investigationResult string, currentPlan *ChangesPlan, review string) (*BlockChange, error) {
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You're an experienced software engineer who is tasked to refactor/update the existing code."),
		openai.SystemMessage(fmt.Sprintf("You can also utilize the investigation results: %s", investigationResult)),
		openai.UserMessage(fmt.Sprintf(promptTemplate, block.TargetName, block.Path, blockContent)),
	}
	if currentPlan != nil && review != "" {
		messages = append(messages, openai.SystemMessage(fmt.Sprintf("The followings are current plan and review. Please improve the existing plan based on the review:\nCurrent Plan: %s\nReview:%s", currentPlan.String(), review)))
	}
	content, err := p.llmClient.GenerateCompletion(ctx,
		messages,
		ChangeDiffSchemaParam)
	if err != nil {
		return nil, fmt.Errorf("failed to create GenerateCompletion: %w", err)
	}

	var change ChangeDiff
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

// GeneratePlan generates ChangesPlan
// If currentPlan is provided, it will be used as a base plan.
// If review is provided, it will be used as a review comment to improve the plan.
func (p *Planner) GeneratePlan(ctx context.Context, query string, summary *summarizer.RepoSummary, files []file.File, currentPlan *ChangesPlan, review string) (*ChangesPlan, error) {

	// identify files to change
	filteredFiles, err := p.removeUnrelevantFiles(ctx, query, files)
	if err != nil {
		return nil, fmt.Errorf("failed to remove irrelevant files: %w", err)
	}

	// 1. Identify candidate blocks to change
	fmt.Printf("---------- 1. Identify candidate blocks to change -----------\n")
	candidateBlocks := map[string][]Block{}
	for i, f := range filteredFiles {
		fmt.Printf("File %d: %s\n", i+1, f.Path)
		if filepath.Ext(f.Path) == ".go" {
			functions, _, err := file.ParseGo(f.Path)
			if err != nil {
				fmt.Printf("failed to parse go file: %v", err)
				continue
			}
			for _, fn := range functions {
				fmt.Printf("Function: %s\n", fn.Name)
				candidateBlocks[f.Path] = append(candidateBlocks[f.Path], Block{Path: f.Path, TargetType: "function", TargetName: fn.Name, Content: fn.Content})
			}
		} else if filepath.Ext(f.Path) == ".hcl" || filepath.Ext(f.Path) == ".tf" {
			blocks, _, err := file.ParseHCL(f.Path)
			if err != nil {
				fmt.Printf("failed to parse hcl file: %v", err)
				continue
			}
			for _, b := range blocks {
				fmt.Printf("Block: Type:%s, Labels:%s\n", b.Type, strings.Join(b.Labels, ","))
				candidateBlocks[f.Path] = append(candidateBlocks[f.Path], Block{Path: f.Path, TargetType: b.Type, TargetName: strings.Join(b.Labels, ","), Content: b.Content})
			}
		} else { // file: block is the entire file
			candidateBlocks[f.Path] = append(candidateBlocks[f.Path], Block{Path: f.Path, TargetType: "file", TargetName: f.Path, Content: f.Content})
		}
	}

	// 2. Make action plan (steps)
	fmt.Printf("---------- 2. Make action plan (steps) -----------\n")
	plan, err := p.makeActionPlan(ctx, candidateBlocks, currentPlan, query, review)
	if err != nil {
		return nil, fmt.Errorf("failed to make action plan: %w", err)
	}

	fmt.Println("ActionPlan:")
	for i, step := range plan.InvestigateSteps {
		fmt.Printf("Investigation Step %d: %s\n", i+1, step)
	}
	for i, step := range plan.ChangeSteps {
		fmt.Printf("Change Step %d: %s\n", i+1, step)
	}

	// 3. Investigation Step
	fmt.Printf("---------- 3. Investigation step -----------\n")
	investigationResultStr, err := p.executeInvestigation(ctx, query, plan.InvestigateSteps, files)
	if err != nil {
		return nil, fmt.Errorf("failed to investigate: %w", err)
	}

	// 4. Change file step
	fmt.Printf("---------- 4. Change file step -----------\n")
	changesPlan := &ChangesPlan{
		Id:      uuid.New().String(),
		Query:   query,
		Changes: []BlockChange{},
	}
	if currentPlan != nil {
		changesPlan.Id = currentPlan.Id
	}
	for i, step := range plan.ChangeSteps {
		fmt.Printf("Change Step %d: %s\n", i+1, step)

		// identify blocks to change for this step
		blocks, err := p.identifyBlocksToChangeForStep(ctx, step, filteredFiles, candidateBlocks)
		if err != nil {
			return nil, fmt.Errorf("failed to identify blocks to change: %w", err)
		}
		fmt.Printf("Step %d: Got %d blocks\n", i+1, len(blocks.Changes))

		for _, blkToChange := range blocks.Changes {
			// TODO: create Planner interface and implement planner for each Language
			if filepath.Ext(blkToChange.Path) == ".go" {
				// Use function as a unit of block for go
				blocks := candidateBlocks[blkToChange.Path]
				for _, blk := range blocks {
					if blk.TargetName == blkToChange.TargetName && blk.TargetType == blkToChange.TargetType {
						fmt.Printf("Step %d: Matched Block Go path:%s, type:%s, name:%s\n", i+1, blk.Path, blk.TargetType, blk.TargetName)
						blkChange, err := p.GenerateBlockChangePlan(ctx, GENERATE_FUNCTION_CHANGES_PLAN_PROMPT_GO, blkToChange, blk.Content, investigationResultStr, changesPlan, review)
						if err != nil {
							log.Fatalf("failed to generate plan: %v", err)
							return nil, fmt.Errorf("failed to generate plan: %w", err)
						}
						changesPlan.Changes = append(changesPlan.Changes, *blkChange)
						break
					}
				}
			} else if filepath.Ext(blkToChange.Path) == ".hcl" || filepath.Ext(blkToChange.Path) == ".tf" {
				// Use block as a unit of block for hcl
				blocks := candidateBlocks[blkToChange.Path]
				for _, blk := range blocks {
					if blk.TargetName == blkToChange.TargetName && blk.TargetType == blkToChange.TargetType { // variable, resource, module, etc.
						fmt.Printf("Step %d: Matched Block HCL path:%s, type:%s, name:%s\n", i+1, blk.Path, blk.TargetType, blk.TargetName)
						blkChange, err := p.GenerateBlockChangePlan(ctx, GENERATE_BLOCK_CHANGES_PLAN_PROMPT_HCL, blkToChange, blk.Content, investigationResultStr, changesPlan, review)
						if err != nil {
							log.Fatalf("failed to generate plan: %v", err)
							return nil, fmt.Errorf("failed to generate plan: %w", err)
						}
						changesPlan.Changes = append(changesPlan.Changes, *blkChange)
						break
					}
				}
				// TODO: enable to change attr in hcl
			} else if blkToChange.TargetType == "file" {
				for _, blk := range candidateBlocks[blkToChange.Path] {
					blkChange, err := p.GenerateBlockChangePlan(ctx, GENERATE_BLOCK_CHANGES_PROMPT_ENTIRE_FILE, blkToChange, blk.Content, investigationResultStr, changesPlan, review)
					if err != nil {
						log.Fatalf("failed to generate plan: %v", err)
					}
					changesPlan.Changes = append(changesPlan.Changes, *blkChange)
				}
			} else {
				fmt.Printf("Step %d: Unsupported file type: %s\n", i+1, blkToChange.Path)
			}
			fmt.Printf("Step %d: Block path:%s type:%s name:%s\n", i+1, blkToChange.Path, blkToChange.TargetType, blkToChange.TargetName)
		}
	}

	return changesPlan, nil
}

func (p *Planner) identifyBlocksToChangeForStep(ctx context.Context, step string, files []file.File, candidateBlocks map[string][]Block) (*TargetBlocks, error) {
	prompt_block, err := p.generateBlockPromptWithFiles(PLANNER_EXTRACT_BLOCK_FOR_STEP_PROMPT, step, files, candidateBlocks)
	if err != nil {
		return nil, fmt.Errorf("failed to generate goal prompt: %w", err)
	}

	// get blocks
	content, err := p.llmClient.GenerateCompletion(ctx,
		[]openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt_block),
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

// 2. Make action plan (steps)
func (p *Planner) makeActionPlan(ctx context.Context, candidateBlocks map[string][]Block, currentPlan *ChangesPlan, query, review string) (*ActionPlan, error) {
	// Use LLM to generate action plan
	candidateBlocksStr := makeFileBlocksString(candidateBlocks)
	examples_str := convertActionPlanExaplesToStr(DefaultActionPlanExamples)
	prompt := fmt.Sprintf(GENERATE_ACTION_PLAN_PROMPT, candidateBlocksStr, query, examples_str)
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You're an experienced software engineer who is tasked to refactor/update the existing code."),
		openai.UserMessage(prompt),
	}
	if currentPlan != nil && review != "" {
		messages = append(messages, openai.SystemMessage(fmt.Sprintf("The followings are current plan and review. Please improve the existing plan based on the review:\nCurrent Plan: %s\nReview:%s", currentPlan.String(), review)))
	}
	content, err := p.llmClient.GenerateCompletion(ctx,
		messages,
		ActionPlanSchemaParam,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GenerateCompletion: %w", err)
	}
	var plan ActionPlan
	if err = json.Unmarshal([]byte(content), &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal necessary changes plan: %w", err)
	}
	return &plan, nil
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
