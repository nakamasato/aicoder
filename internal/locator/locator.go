package locator

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"html/template"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/loader"
)

//go:embed templates/locator_file.tmpl
var promptLocateFileTemplate string

//go:embed templates/locator_file_irrelevant.tmpl
var promptLocateFileIrrelevantTemplate string

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

func (l Locator) Locate(ctx context.Context, irrelevant bool, query string, repoStructure loader.RepoStructure) (string, error) {

	if query == "" {
		return "", fmt.Errorf("query is empty")
	}

	var templatefile string
	if irrelevant {
		templatefile = promptLocateFileIrrelevantTemplate
	} else {
		templatefile = promptLocateFileTemplate
	}

	prompt, err := makePrompt(templatefile, query, repoStructure)
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

func makePrompt(templatefile, query string, repoStructure loader.RepoStructure) (string, error) {

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
