package summarizer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/openai/openai-go"
)

type Language string

const (
	LanguageEnglish  Language = "en"
	LanguageJapanese Language = "ja"
)

type service struct {
	config      *config.AICoderConfig
	llmClient   llm.Client
	entClient   *ent.Client
	vectorstore vectorstore.VectorStore
}

func NewService(cfg *config.AICoderConfig, entClient *ent.Client, llmClient llm.Client, store vectorstore.VectorStore) *service {
	return &service{
		config:      cfg,
		llmClient:   llmClient,
		entClient:   entClient,
		vectorstore: store,
	}
}

func (s *service) UpdateRepoSummary(ctx context.Context, language Language, outputfile string) (string, error) {
	docs, err := s.entClient.Document.Query().Where(document.RepositoryEQ(s.config.Repository), document.ContextEQ(s.config.CurrentContext)).All(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to query documents: %v", err)
	}

	var documents []*vectorstore.Document
	for _, doc := range docs {
		documents = append(documents, &vectorstore.Document{
			Repository:  doc.Repository,
			Context:     doc.Context,
			Filepath:    doc.Filepath,
			Description: doc.Description,
		})
	}

	var builder strings.Builder
	for _, doc := range documents {
		builder.WriteString(fmt.Sprintf("File: %s\n", doc.Filepath))
		builder.WriteString(fmt.Sprintf("Summary: %s\n", doc.Description))
	}

	prompt := fmt.Sprintf(llm.SUMMARIZE_REPO_CONTENT_PROMPT, s.config.Repository, s.config.CurrentContext, builder.String(), language)
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(prompt),
	}
	summary, err := s.llmClient.GenerateCompletionSimple(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate completion: %v", err)
	}

	// Marshal the summary data to JSON
	summaryJSON, err := json.MarshalIndent(
		map[string]string{
			"summary": summary,
		}, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal summary to JSON: %v", err)
	}

	// Write the JSON to the output file
	if err := os.WriteFile(outputfile, summaryJSON, 0644); err != nil {
		log.Fatalf("failed to write summary to file: %v", err)
	}

	return summary, nil
}

// ReadSummary reads the summary from the given file
func ReadSummary(ctx context.Context, filename string) (string, error) {
	// Read the file content
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Unmarshal the JSON content
	var data map[string]string
	if err := json.Unmarshal(content, &data); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	// Return the summary
	summary, ok := data["summary"]
	if !ok {
		return "", fmt.Errorf("summary not found in file")
	}

	return summary, nil
}
