// Code generated by ent, DO NOT EDIT.

package ent

import (
	"fmt"
	"strings"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/nakamasato/aicoder/ent/document"
	pgvector "github.com/pgvector/pgvector-go"
)

// Document is the model entity for the Document schema.
type Document struct {
	config `json:"-"`
	// ID of the ent.
	ID int64 `json:"id,omitempty"`
	// Repository holds the value of the "repository" field.
	Repository string `json:"repository,omitempty"`
	// Filepath holds the value of the "filepath" field.
	Filepath string `json:"filepath,omitempty"`
	// Description holds the value of the "description" field.
	Description string `json:"description,omitempty"`
	// Embedding holds the value of the "embedding" field.
	Embedding    pgvector.Vector `json:"embedding,omitempty"`
	selectValues sql.SelectValues
}

// scanValues returns the types for scanning values from sql.Rows.
func (*Document) scanValues(columns []string) ([]any, error) {
	values := make([]any, len(columns))
	for i := range columns {
		switch columns[i] {
		case document.FieldEmbedding:
			values[i] = new(pgvector.Vector)
		case document.FieldID:
			values[i] = new(sql.NullInt64)
		case document.FieldRepository, document.FieldFilepath, document.FieldDescription:
			values[i] = new(sql.NullString)
		default:
			values[i] = new(sql.UnknownType)
		}
	}
	return values, nil
}

// assignValues assigns the values that were returned from sql.Rows (after scanning)
// to the Document fields.
func (d *Document) assignValues(columns []string, values []any) error {
	if m, n := len(values), len(columns); m < n {
		return fmt.Errorf("mismatch number of scan values: %d != %d", m, n)
	}
	for i := range columns {
		switch columns[i] {
		case document.FieldID:
			value, ok := values[i].(*sql.NullInt64)
			if !ok {
				return fmt.Errorf("unexpected type %T for field id", value)
			}
			d.ID = int64(value.Int64)
		case document.FieldRepository:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field repository", values[i])
			} else if value.Valid {
				d.Repository = value.String
			}
		case document.FieldFilepath:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field filepath", values[i])
			} else if value.Valid {
				d.Filepath = value.String
			}
		case document.FieldDescription:
			if value, ok := values[i].(*sql.NullString); !ok {
				return fmt.Errorf("unexpected type %T for field description", values[i])
			} else if value.Valid {
				d.Description = value.String
			}
		case document.FieldEmbedding:
			if value, ok := values[i].(*pgvector.Vector); !ok {
				return fmt.Errorf("unexpected type %T for field embedding", values[i])
			} else if value != nil {
				d.Embedding = *value
			}
		default:
			d.selectValues.Set(columns[i], values[i])
		}
	}
	return nil
}

// Value returns the ent.Value that was dynamically selected and assigned to the Document.
// This includes values selected through modifiers, order, etc.
func (d *Document) Value(name string) (ent.Value, error) {
	return d.selectValues.Get(name)
}

// Update returns a builder for updating this Document.
// Note that you need to call Document.Unwrap() before calling this method if this Document
// was returned from a transaction, and the transaction was committed or rolled back.
func (d *Document) Update() *DocumentUpdateOne {
	return NewDocumentClient(d.config).UpdateOne(d)
}

// Unwrap unwraps the Document entity that was returned from a transaction after it was closed,
// so that all future queries will be executed through the driver which created the transaction.
func (d *Document) Unwrap() *Document {
	_tx, ok := d.config.driver.(*txDriver)
	if !ok {
		panic("ent: Document is not a transactional entity")
	}
	d.config.driver = _tx.drv
	return d
}

// String implements the fmt.Stringer.
func (d *Document) String() string {
	var builder strings.Builder
	builder.WriteString("Document(")
	builder.WriteString(fmt.Sprintf("id=%v, ", d.ID))
	builder.WriteString("repository=")
	builder.WriteString(d.Repository)
	builder.WriteString(", ")
	builder.WriteString("filepath=")
	builder.WriteString(d.Filepath)
	builder.WriteString(", ")
	builder.WriteString("description=")
	builder.WriteString(d.Description)
	builder.WriteString(", ")
	builder.WriteString("embedding=")
	builder.WriteString(fmt.Sprintf("%v", d.Embedding))
	builder.WriteByte(')')
	return builder.String()
}

// Documents is a parsable slice of Document.
type Documents []*Document
