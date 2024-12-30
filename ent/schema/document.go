package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/pgvector/pgvector-go"
)

// Document holds the schema definition for the Document entity.
type Document struct {
	ent.Schema
}

// Fields of the Document.
func (Document) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Text("repository"),
		field.Text("content"),
		field.Text("description"),
		field.Other("embedding", pgvector.Vector{}).
			SchemaType(map[string]string{
				dialect.Postgres: "vector(1536)",
			}),
	}
}

// Edges of the Document.
func (Document) Edges() []ent.Edge {
	return nil
}

func (Document) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("embedding").
			Annotations(
				entsql.IndexType("hnsw"),
				entsql.OpClass("vector_l2_ops"),
			),
		index.Fields("repository"),
	}
}
