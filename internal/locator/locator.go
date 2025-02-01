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

// Locate locates the relevant block or line in the repository.
func (l Locator) Locate(ctx context.Context, locatorType LocatorType, query string, repoStructure loader.RepoStructure, numOfSample int64) (*llm.FileBlockList, error) {

	if query == "" {
		return nil, fmt.Errorf("query is empty")
	}

	// Locate relevant files
	templatefile := locatorTypeMap[LocatorTypeFile]
	filelist, err := l.locateFile(ctx, templatefile, query, repoStructure)
	if err != nil {
		return nil, fmt.Errorf("failed to locate file: %v", err)
	}
	fmt.Println(filelist)

	// Locate block or line
	templatefile = locatorTypeMap[LocatorTypeBlock]
	blocksSamples, err := l.locateBlock(ctx, templatefile, query, filelist)
	if err != nil {
		return nil, fmt.Errorf("failed to locate block: %v", err)
	}

	// Locate line
	templatefile = locatorTypeMap[LocatorTypeLine]
	_, err = l.locateLine(ctx, templatefile, query, filelist)
	if err != nil {
		return nil, fmt.Errorf("failed to locate line: %v", err)
	}

	return blocksSamples, nil
}

// locateFile locates the relevant files in the repository.
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

	var filelist llm.FileList
	if err = json.Unmarshal([]byte(res), &filelist); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relevant files: %v", err)
	}

	return &filelist, nil
}

// locateBlock locates the relevant block in the files.
func (l Locator) locateBlock(ctx context.Context, templatefile, query string, filelist *llm.FileList) (*llm.FileBlockList, error) {

	if len(filelist.Paths) == 0 {
		return nil, fmt.Errorf("no files found")
	}

	fileContents := make(map[string]string, len(filelist.Paths))
	for _, path := range filelist.Paths {
		content, err := file.ReadContent(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read content: %v", err)
		}
		fileContents[path] = content
	}

	fileContentsStr, err := formatFileContents(fileContents)
	if err != nil {
		return nil, fmt.Errorf("failed to format file contents: %v", err)
	}

	prompt, err := makeLocateBlockOrLinePrompt(templatefile, query, fileContentsStr)
	if err != nil {
		return nil, fmt.Errorf("failed to make prompt: %v", err)
	}

	res, err := l.llmClient.GenerateCompletion(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: prompt},
	}, llm.FileBlockListSchemaParam)
	if err != nil {
		return nil, fmt.Errorf("failed to generate completion: %v", err)
	}

	var fileBlockList llm.FileBlockList
	if err = json.Unmarshal([]byte(res), &fileBlockList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relevant blocks: %v", err)
	}

	return &fileBlockList, nil
}

// locateLine locates the relevant line in the files.
// TODO: check if blocks that are extracted in the previous step are passed as a parameter
func (l Locator) locateLine(ctx context.Context, templatefile, query string, filelist *llm.FileList) (*llm.FileBlockLineList, error) {

	if len(filelist.Paths) == 0 {
		return nil, fmt.Errorf("no files found")
	}

	fileContents := make(map[string]string, len(filelist.Paths))
	for _, path := range filelist.Paths {
		content, err := file.ReadContent(path)
		// TODO: add line number to the content
		if err != nil {
			return nil, fmt.Errorf("failed to read content: %v", err)
		}
		fileContents[path] = content
	}

	fileContentsStr, err := formatFileContents(fileContents)
	if err != nil {
		return nil, fmt.Errorf("failed to format file contents: %v", err)
	}

	prompt, err := makeLocateBlockOrLinePrompt(templatefile, query, fileContentsStr)
	if err != nil {
		return nil, fmt.Errorf("failed to make prompt: %v", err)
	}

	res, err := l.llmClient.GenerateCompletion(ctx, []llm.Message{
		{Role: llm.RoleUser, Content: prompt},
	}, llm.FileBlockLineListSchemaParam)
	if err != nil {
		return nil, fmt.Errorf("failed to generate completion: %v", err)
	}

	var fileBlockLineList llm.FileBlockLineList
	if err = json.Unmarshal([]byte(res), &fileBlockLineList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relevant lines: %v", err)
	}

	return &fileBlockLineList, nil
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
