// Code generated by ent, DO NOT EDIT.

package document

import (
	"entgo.io/ent/dialect/sql"
)

const (
	// Label holds the string label denoting the document type in the database.
	Label = "document"
	// FieldID holds the string denoting the id field in the database.
	FieldID = "id"
	// FieldRepository holds the string denoting the repository field in the database.
	FieldRepository = "repository"
	// FieldContent holds the string denoting the content field in the database.
	FieldContent = "content"
	// FieldDescription holds the string denoting the description field in the database.
	FieldDescription = "description"
	// FieldEmbedding holds the string denoting the embedding field in the database.
	FieldEmbedding = "embedding"
	// Table holds the table name of the document in the database.
	Table = "documents"
)

// Columns holds all SQL columns for document fields.
var Columns = []string{
	FieldID,
	FieldRepository,
	FieldContent,
	FieldDescription,
	FieldEmbedding,
}

// ValidColumn reports if the column name is valid (part of the table columns).
func ValidColumn(column string) bool {
	for i := range Columns {
		if column == Columns[i] {
			return true
		}
	}
	return false
}

// OrderOption defines the ordering options for the Document queries.
type OrderOption func(*sql.Selector)

// ByID orders the results by the id field.
func ByID(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldID, opts...).ToFunc()
}

// ByRepository orders the results by the repository field.
func ByRepository(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldRepository, opts...).ToFunc()
}

// ByContent orders the results by the content field.
func ByContent(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldContent, opts...).ToFunc()
}

// ByDescription orders the results by the description field.
func ByDescription(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldDescription, opts...).ToFunc()
}

// ByEmbedding orders the results by the embedding field.
func ByEmbedding(opts ...sql.OrderTermOption) OrderOption {
	return sql.OrderByField(FieldEmbedding, opts...).ToFunc()
}
