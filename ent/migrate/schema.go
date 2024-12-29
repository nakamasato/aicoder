// Code generated by ent, DO NOT EDIT.

package migrate

import (
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
)

var (
	// DocumentsColumns holds the columns for the "documents" table.
	DocumentsColumns = []*schema.Column{
		{Name: "id", Type: field.TypeInt64, Increment: true},
		{Name: "content", Type: field.TypeString, Size: 2147483647},
		{Name: "embedding", Type: field.TypeOther, SchemaType: map[string]string{"postgres": "vector(1536)"}},
	}
	// DocumentsTable holds the schema information for the "documents" table.
	DocumentsTable = &schema.Table{
		Name:       "documents",
		Columns:    DocumentsColumns,
		PrimaryKey: []*schema.Column{DocumentsColumns[0]},
		Indexes: []*schema.Index{
			{
				Name:    "document_embedding",
				Unique:  false,
				Columns: []*schema.Column{DocumentsColumns[2]},
				Annotation: &entsql.IndexAnnotation{
					OpClass: "vector_l2_ops",
					Type:    "hnsw",
				},
			},
		},
	}
	// Tables holds all the tables in the schema.
	Tables = []*schema.Table{
		DocumentsTable,
	}
)

func init() {
}
