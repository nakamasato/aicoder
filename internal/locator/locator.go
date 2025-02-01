package locator

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
	"github.com/nakamasato/aicoder/internal/loader"
)

//go:embed templates/locator_file.tmpl
var promptLocateFileTemplate string

//go:embed templates/locator_file_irrelevant.tmpl
var promptLocateFileIrrelevantTemplate string

//go:embed templates/locator_block.tmpl
var promptLocateBlockTemplate string

//go:embed templates/locator_line.tmpl
var promptLocateLineTemplate string

//go:embed templates/file_content.tmpl
var fileContentTemplate string

type Locator struct {
	config    *config.AICoderConfig
	llmClient llm.Client
}

func NewLocator(llmClient llm.Client, config *config.AICoderConfig) *Locator {
	return &Locator{
		config:    config,
		llmClient: llmClient,
	}
}

type LocatorType string // implement pflag.Value

const (
	LocatorTypeFile           LocatorType = "file"
	LocatorTypeFileIrrelevant LocatorType = "file_irrelevant"
	LocatorTypeBlock          LocatorType = "block"
	LocatorTypeLine           LocatorType = "line"
)

// Set sets the value of the LocatorType.
func (lt *LocatorType) Set(value string) error {
	switch value {
	case string(LocatorTypeFile), string(LocatorTypeFileIrrelevant), string(LocatorTypeBlock), string(LocatorTypeLine):
		*lt = LocatorType(value)
		return nil
	default:
		return fmt.Errorf("invalid locator type: %s", value)
	}
}

// Type returns the type of the flag as a string.
func (lt *LocatorType) Type() string {
	return "locatorType"
}

// String returns the string representation of the LocatorType.
func (lt *LocatorType) String() string {
	return string(*lt)
}

var locatorTypeMap = map[LocatorType]string{
	LocatorTypeFile:           promptLocateFileTemplate,
	LocatorTypeFileIrrelevant: promptLocateFileIrrelevantTemplate,
	LocatorTypeBlock:          promptLocateBlockTemplate,
	LocatorTypeLine:           promptLocateLineTemplate,
}

func (l Locator) Locate(ctx context.Context, locatorType LocatorType, query string, repoStructure loader.RepoStructure) (string, error) {

	if query == "" {
		return "", fmt.Errorf("query is empty")
	}

	templatefile := locatorTypeMap[LocatorTypeFile]

	filelist, err := l.locateFile(ctx, templatefile, query, repoStructure)
	if err != nil {
		return "", fmt.Errorf("failed to locate file: %v", err)
	}
	fmt.Println(filelist)

	templatefile = locatorTypeMap[LocatorTypeBlock]
	blocks, err := l.locateBlock(ctx, templatefile, query, filelist)
	if err != nil {
		return "", fmt.Errorf("failed to locate block: %v", err)
	}

	return blocks, nil
}

func (l Locator) locateFile(ctx context.Context, templatefile, query string, repoStructure loader.RepoStructure) (*llm.FileList, error) {
	prompt, err := makeLocateFilePrompt(templatefile, query, repoStructure)
	if err != nil {
		return nil, fmt.Errorf("failed to make prompt: %v", err)
	}

	res, err := l.llmClient.GenerateCompletion(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: prompt},
	}, llm.FileListSchemaParam)
	if err != nil {
		return nil, fmt.Errorf("failed to generate completion: %v", err)
	}

	fmt.Println(res)

	var filelist llm.FileList
	if err = json.Unmarshal([]byte(res), &filelist); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relevant files: %v", err)
	}

	return &filelist, nil
}

func (l Locator) locateBlock(ctx context.Context, templatefile, query string, filelist *llm.FileList) (string, error) {

	if len(filelist.Paths) == 0 {
		return "", fmt.Errorf("no files found")
	}

	fileContents := make(map[string]string, len(filelist.Paths))
	for _, path := range filelist.Paths {
		content, err := file.ReadContent(path)
		if err != nil {
			return "", fmt.Errorf("failed to read content: %v", err)
		}
		fileContents[path] = content
	}

	fileContentsStr, err := formatFileContents(fileContents)

	prompt, err := makeLocateBlockOrLinePrompt(templatefile, query, fileContentsStr)
	if err != nil {
		return "", fmt.Errorf("failed to make prompt: %v", err)
	}

	res, err := l.llmClient.GenerateCompletionSimple(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: prompt},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate completion: %v", err)
	}

	return res, nil
}

func formatFileContents(fileContents map[string]string) (string, error) {
	tmpl, err := template.New("template").Parse(fileContentTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse file template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, fileContents); err != nil {
		return "", fmt.Errorf("failed to execute file template: %v", err)
	}

	return buf.String(), nil
}

func makeLocateFilePrompt(templatefile, query string, repoStructure loader.RepoStructure) (string, error) {

	var prompt string
	tmplData := struct {
		Query         string
		RepoStructure string
	}{
		Query:         query,
		RepoStructure: repoStructure.ToTreeString(),
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

func makeLocateBlockOrLinePrompt(templatefile, query, fileContents string) (string, error) {

	var prompt string
	tmplData := struct {
		Query        string
		FileContents string
	}{
		Query:        query,
		FileContents: fileContents,
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
