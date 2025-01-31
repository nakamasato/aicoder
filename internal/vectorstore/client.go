package vectorstore

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/nakamasato/aicoder/ent"
	"github.com/nakamasato/aicoder/ent/document"
	"github.com/nakamasato/aicoder/internal/llm"
	"github.com/pgvector/pgvector-go"
)

type VectorStore interface {
	AddDocument(ctx context.Context, doc *Document) error
	Search(ctx context.Context, repository, context, query string, k int) (*SearchResult, error)
}

type DistanceFunc func(a, b []float32) float64

func EuclideanDistance(a, b []float32) float64 {
	if len(a) != len(b) {
		log.Fatalf("Vectors must be of same length")
	}

	var sum float64

	for i := 0; i < len(a); i++ {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}

	return math.Sqrt(sum)
}

type vectorstore struct {
	llmClient    llm.Client
	entClient    *ent.Client
	distanceFunc DistanceFunc
}

type Document struct {
	Repository  string
	Context     string
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

func (r *SearchResult) String() string {
	var b strings.Builder
	for i, doc := range *r.Documents {
		b.WriteString(fmt.Sprintf("%d. %s (Score: %.2f)\n", i+1, doc.Document.Filepath, doc.Score))
	}
	return b.String()
}

func New(entClient *ent.Client, llmClient llm.Client) VectorStore {
	return &vectorstore{entClient: entClient, llmClient: llmClient, distanceFunc: EuclideanDistance}
}

func (c *vectorstore) AddDocument(ctx context.Context, doc *Document) error {
	embedding, err := c.llmClient.GetEmbedding(ctx, doc.Description)
	if err != nil {
		return err
	}
	vector := pgvector.NewVector(embedding)
	err = c.entClient.Document.Create().
		SetFilepath(doc.Filepath).
		SetRepository(doc.Repository).
		SetContext(doc.Context).
		SetDescription(doc.Description).
		SetEmbedding(vector).
		SetUpdatedAt(time.Now()).
		OnConflictColumns(document.FieldRepository, document.FieldContext, document.FieldFilepath).
		UpdateNewValues().
		Exec(ctx)
	return err
}

func (c *vectorstore) Search(ctx context.Context, repository, context, query string, k int) (*SearchResult, error) {
	queryEmbedding, err := c.llmClient.GetEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}
	vector := pgvector.NewVector(queryEmbedding)

	docs, err := c.entClient.Document.
		Query().
		Where(document.RepositoryEQ(repository)).
		Where(document.ContextEQ(context)).
		Order(func(s *sql.Selector) {
			s.OrderExpr(sql.ExprP("embedding <-> $3", vector))
		}).Limit(k).All(ctx)
	if err != nil {
		return nil, err
	}
	results := SearchResult{Documents: &[]DocumentWithScore{}}
	for _, doc := range docs {
		distance := c.distanceFunc(doc.Embedding.Slice(), queryEmbedding)
		*results.Documents = append(*results.Documents, DocumentWithScore{
			Document: &Document{
				Repository:  doc.Repository,
				Context:     doc.Context,
				Filepath:    doc.Filepath,
				Description: doc.Description,
			},
			Score: distance,
		})
	}
	return &results, nil
}
