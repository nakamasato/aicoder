package summarizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/nakamasato/aicoder/config"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/internal/vectorstore"
	"github.com/openai/openai-go"
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

func (s *service) SummarizeRepo(ctx context.Context) error {
	docs, err := s.entClient.Document.Query().Where(document.RepositoryEQ(s.config.Repository), document.ContextEQ(s.config.CurrentContext)).All(ctx)
	if err != nil {
		return fmt.Errorf("failed to query documents: %v", err)
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

	prompt := fmt.Sprintf(llm.SUMMARIZE_REPO_CONTENT_PROMPT, s.config.Repository, s.config.CurrentContext, builder.String())
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(prompt),
	}
	summaries, err := s.llmClient.GenerateCompletionSimple(ctx, messages)
	if err != nil {
		return fmt.Errorf("failed to generate completion: %v", err)
	}
	fmt.Printf("Summary: %s\n", summaries)

	return nil
}
