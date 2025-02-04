package repairer

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/internal/file"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/locator"
)

//go:embed templates/repair.tmpl
var promptRepairTemplate string

//go:embed templates/extract_block_content.tmpl
var promptExtractBlockContentTemplate string

const RepairInstruction = "Below are some code segments, each from a relevant file. One or more of these files may contain bugs."

type Repairer struct {
	config    *config.AICoderConfig
	llmClient llm.Client
}

type RepairSample struct {
	Sample int
	Repair llm.Repair
}

type BlockRepair struct {
	Block        llm.Block
	BlockContent string
	Repairs      []RepairSample
}

type FileRepair struct {
	File         string
	BlockRepairs []BlockRepair
}

type RepairOutput struct {
	Query       string
	FileRepairs []FileRepair
}

func NewRepairer(llmClient llm.Client, config *config.AICoderConfig) *Repairer {
	return &Repairer{
		config:    config,
		llmClient: llmClient,
	}
}

func (r Repairer) Repair(ctx context.Context, location *locator.LocationOutput, numOfSample int64) (RepairOutput, error) {
	// Repair the files
	result := RepairOutput{
		Query: location.Query,
	}
	for _, blk := range location.BlockList.Files {
		fileRepair, err := r.repairOneFile(ctx, location.Query, blk, numOfSample)
		if err != nil {
			return result, fmt.Errorf("failed to repair: %v", err)
		}
		result.FileRepairs = append(result.FileRepairs, *fileRepair)
	}
	return result, nil
}

func (r Repairer) repairOneFile(ctx context.Context, query string, blkList llm.BlockList, numOfSample int64) (*FileRepair, error) {

	// TODO: get the content of blocks
	content, err := file.ReadContent(blkList.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s, %v", blkList.Path, err)
	}

	fr := FileRepair{
		File: blkList.Path,
	}

	// get block content
	for i, blk := range blkList.Blocks {
		fmt.Printf("[repairOneBlockList][%d/%d] block Path:%s, Type:%s, Name:%s\n", i+1, len(blkList.Blocks), blkList.Path, blk.BlockType, blk.Name)
		getBlockContentPrompt, err := makeGetBlockContentPrompt(promptExtractBlockContentTemplate, blk, content)
		if err != nil {
			return nil, fmt.Errorf("failed to make prompt %v", err)
		}
		blockContent, err := r.llmClient.GenerateCompletionSimple(ctx, []llm.Message{{Content: getBlockContentPrompt, Role: llm.RoleSystem}})
		if err != nil {
			return nil, fmt.Errorf("failed to generate completion: %v", err)
		}
		fmt.Printf("[repairOneBlockList][%d/%d] block content: %s\n", i+1, len(blkList.Blocks), blockContent)

		blockRepair, err := r.repairOneBlock(ctx, r.llmClient, query, blk, blockContent, numOfSample)
		if err != nil {
			return nil, fmt.Errorf("failed to repair block: %v", err)
		}
		fr.BlockRepairs = append(fr.BlockRepairs, *blockRepair)
	}

	return &fr, nil
}

func (r Repairer) repairOneBlock(ctx context.Context, llmClient llm.Client, query string, block llm.Block, content string, numOfSample int64) (*BlockRepair, error) {
	br := BlockRepair{
		Block:        block,
		BlockContent: content,
	}
	prompt, err := makePrompt(promptRepairTemplate, query, content)
	if err != nil {
		return nil, fmt.Errorf("failed to make prompt %v", err)
	}

	res, err := r.llmClient.GenerateCompletions(ctx, []llm.Message{{Content: prompt, Role: llm.RoleSystem}}, llm.RepairSchemaParam, numOfSample)
	if err != nil {
		return nil, fmt.Errorf("failed to generate completions: %v", err)
	}

	for i, d := range res {
		fmt.Printf("[repairOneBlock] repair: %s\n", d)
		var r llm.Repair
		if err := json.Unmarshal([]byte(d), &r); err != nil {
			return nil, fmt.Errorf("failed to unmarshal repair: %v", err)
		}
		br.Repairs = append(br.Repairs, RepairSample{Sample: i, Repair: r})
	}
	return &br, nil
}

func makePrompt(templatefile, query, content string) (string, error) {

	var prompt string
	tmplData := struct {
		Query                         string
		RepairRelevantFileInstruction string
		Content                       string
	}{
		Query:                         query,
		RepairRelevantFileInstruction: RepairInstruction,
		Content:                       content,
	}

	tmpl, err := template.New("template").Parse(templatefile)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tmplData); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}

	prompt = buf.String()
	return prompt, nil
}

// makeGetBlockContentPrompt generates a prompt for extracting block content
func makeGetBlockContentPrompt(templatefile string, block llm.Block, content string) (string, error) {
	tmplData := struct {
		Block   llm.Block
		Content string
	}{
		Block:   block,
		Content: content,
	}
	tmpl, err := template.New("template").Parse(templatefile)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tmplData); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	return buf.String(), nil
}
