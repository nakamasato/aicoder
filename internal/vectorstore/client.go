package vectorstore

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/nakamasato/aicoder/pkg/vectorutils"
	"github.com/openai/openai-go"
	"github.com/pgvector/pgvector-go"
)

type VectorStore interface {
	AddDocument(ctx context.Context, doc *Document) error
	Search(ctx context.Context, repository, query string, k int) (*SearchResult, error)
}

type vectorstore struct {
	openaiCli *openai.Client
	entClient *ent.Client
}

type Document struct {
	Repository  string
	Filepath    string
	Description string
}

type DocumentWithScore struct {
	Document *Document
	Score    float64
}

type SearchResult struct {
	Documents *[]DocumentWithScore
}

func New(entClient *ent.Client, openaiCli *openai.Client) VectorStore {
	return &vectorstore{entClient: entClient, openaiCli: openaiCli}
}

func (c *vectorstore) AddDocument(ctx context.Context, doc *Document) error {
	embedding, err := llm.GetEmbedding(ctx, c.openaiCli, doc.Description)
	if err != nil {
		return err
	}
	vector := pgvector.NewVector(embedding)
	err = c.entClient.Document.Create().
		SetFilepath(doc.Filepath).
		SetRepository(doc.Repository).
		SetDescription(doc.Description).
		SetEmbedding(vector).
		SetUpdatedAt(time.Now()).
		OnConflictColumns(document.FieldRepository, document.FieldFilepath).
		UpdateNewValues().
		Exec(ctx)
	return err
}

func (c *vectorstore) Search(ctx context.Context, repository, query string, k int) (*SearchResult, error) {
	queryEmbedding, err := llm.GetEmbedding(ctx, c.openaiCli, query)
	if err != nil {
		return nil, err
	}
	vector := pgvector.NewVector(queryEmbedding)

	docs, err := c.entClient.Document.
		Query().
		Where(document.RepositoryEQ(repository)).
		Order(func(s *sql.Selector) {
			s.OrderExpr(sql.ExprP("embedding <-> $2", vector))
		}).Limit(k).All(ctx)
	if err != nil {
		return nil, err
	}
	results := SearchResult{Documents: &[]DocumentWithScore{}}
	for _, doc := range docs {
		distance := vectorutils.EuclideanDistance(doc.Embedding.Slice(), queryEmbedding)
		*results.Documents = append(*results.Documents, DocumentWithScore{
			Document: &Document{
				Repository:  doc.Repository,
				Filepath:    doc.Filepath,
				Description: doc.Description,
			},
			Score: distance,
		})
	}
	return &results, nil
}
