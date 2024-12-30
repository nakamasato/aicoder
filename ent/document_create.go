// Code generated by ent, DO NOT EDIT.

package ent

import (
	"context"
	"errors"
	"fmt"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"entgo.io/ent/schema/field"
	"github.com/nakamasato/aicoder/ent/document"
	pgvector "github.com/pgvector/pgvector-go"
)

// DocumentCreate is the builder for creating a Document entity.
type DocumentCreate struct {
	config
	mutation *DocumentMutation
	hooks    []Hook
}

// SetRepository sets the "repository" field.
func (dc *DocumentCreate) SetRepository(s string) *DocumentCreate {
	dc.mutation.SetRepository(s)
	return dc
}

// SetFilepath sets the "filepath" field.
func (dc *DocumentCreate) SetFilepath(s string) *DocumentCreate {
	dc.mutation.SetFilepath(s)
	return dc
}

// SetDescription sets the "description" field.
func (dc *DocumentCreate) SetDescription(s string) *DocumentCreate {
	dc.mutation.SetDescription(s)
	return dc
}

// SetEmbedding sets the "embedding" field.
func (dc *DocumentCreate) SetEmbedding(pg pgvector.Vector) *DocumentCreate {
	dc.mutation.SetEmbedding(pg)
	return dc
}

// SetID sets the "id" field.
func (dc *DocumentCreate) SetID(i int64) *DocumentCreate {
	dc.mutation.SetID(i)
	return dc
}

// Mutation returns the DocumentMutation object of the builder.
func (dc *DocumentCreate) Mutation() *DocumentMutation {
	return dc.mutation
}

// Save creates the Document in the database.
func (dc *DocumentCreate) Save(ctx context.Context) (*Document, error) {
	return withHooks(ctx, dc.sqlSave, dc.mutation, dc.hooks)
}

// SaveX calls Save and panics if Save returns an error.
func (dc *DocumentCreate) SaveX(ctx context.Context) *Document {
	v, err := dc.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (dc *DocumentCreate) Exec(ctx context.Context) error {
	_, err := dc.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (dc *DocumentCreate) ExecX(ctx context.Context) {
	if err := dc.Exec(ctx); err != nil {
		panic(err)
	}
}

// check runs all checks and user-defined validators on the builder.
func (dc *DocumentCreate) check() error {
	if _, ok := dc.mutation.Repository(); !ok {
		return &ValidationError{Name: "repository", err: errors.New(`ent: missing required field "Document.repository"`)}
	}
	if _, ok := dc.mutation.Filepath(); !ok {
		return &ValidationError{Name: "filepath", err: errors.New(`ent: missing required field "Document.filepath"`)}
	}
	if _, ok := dc.mutation.Description(); !ok {
		return &ValidationError{Name: "description", err: errors.New(`ent: missing required field "Document.description"`)}
	}
	if _, ok := dc.mutation.Embedding(); !ok {
		return &ValidationError{Name: "embedding", err: errors.New(`ent: missing required field "Document.embedding"`)}
	}
	return nil
}

func (dc *DocumentCreate) sqlSave(ctx context.Context) (*Document, error) {
	if err := dc.check(); err != nil {
		return nil, err
	}
	_node, _spec := dc.createSpec()
	if err := sqlgraph.CreateNode(ctx, dc.driver, _spec); err != nil {
		if sqlgraph.IsConstraintError(err) {
			err = &ConstraintError{msg: err.Error(), wrap: err}
		}
		return nil, err
	}
	if _spec.ID.Value != _node.ID {
		id := _spec.ID.Value.(int64)
		_node.ID = int64(id)
	}
	dc.mutation.id = &_node.ID
	dc.mutation.done = true
	return _node, nil
}

func (dc *DocumentCreate) createSpec() (*Document, *sqlgraph.CreateSpec) {
	var (
		_node = &Document{config: dc.config}
		_spec = sqlgraph.NewCreateSpec(document.Table, sqlgraph.NewFieldSpec(document.FieldID, field.TypeInt64))
	)
	if id, ok := dc.mutation.ID(); ok {
		_node.ID = id
		_spec.ID.Value = id
	}
	if value, ok := dc.mutation.Repository(); ok {
		_spec.SetField(document.FieldRepository, field.TypeString, value)
		_node.Repository = value
	}
	if value, ok := dc.mutation.Filepath(); ok {
		_spec.SetField(document.FieldFilepath, field.TypeString, value)
		_node.Filepath = value
	}
	if value, ok := dc.mutation.Description(); ok {
		_spec.SetField(document.FieldDescription, field.TypeString, value)
		_node.Description = value
	}
	if value, ok := dc.mutation.Embedding(); ok {
		_spec.SetField(document.FieldEmbedding, field.TypeOther, value)
		_node.Embedding = value
	}
	return _node, _spec
}

// DocumentCreateBulk is the builder for creating many Document entities in bulk.
type DocumentCreateBulk struct {
	config
	err      error
	builders []*DocumentCreate
}

// Save creates the Document entities in the database.
func (dcb *DocumentCreateBulk) Save(ctx context.Context) ([]*Document, error) {
	if dcb.err != nil {
		return nil, dcb.err
	}
	specs := make([]*sqlgraph.CreateSpec, len(dcb.builders))
	nodes := make([]*Document, len(dcb.builders))
	mutators := make([]Mutator, len(dcb.builders))
	for i := range dcb.builders {
		func(i int, root context.Context) {
			builder := dcb.builders[i]
			var mut Mutator = MutateFunc(func(ctx context.Context, m Mutation) (Value, error) {
				mutation, ok := m.(*DocumentMutation)
				if !ok {
					return nil, fmt.Errorf("unexpected mutation type %T", m)
				}
				if err := builder.check(); err != nil {
					return nil, err
				}
				builder.mutation = mutation
				var err error
				nodes[i], specs[i] = builder.createSpec()
				if i < len(mutators)-1 {
					_, err = mutators[i+1].Mutate(root, dcb.builders[i+1].mutation)
				} else {
					spec := &sqlgraph.BatchCreateSpec{Nodes: specs}
					// Invoke the actual operation on the latest mutation in the chain.
					if err = sqlgraph.BatchCreate(ctx, dcb.driver, spec); err != nil {
						if sqlgraph.IsConstraintError(err) {
							err = &ConstraintError{msg: err.Error(), wrap: err}
						}
					}
				}
				if err != nil {
					return nil, err
				}
				mutation.id = &nodes[i].ID
				if specs[i].ID.Value != nil && nodes[i].ID == 0 {
					id := specs[i].ID.Value.(int64)
					nodes[i].ID = int64(id)
				}
				mutation.done = true
				return nodes[i], nil
			})
			for i := len(builder.hooks) - 1; i >= 0; i-- {
				mut = builder.hooks[i](mut)
			}
			mutators[i] = mut
		}(i, ctx)
	}
	if len(mutators) > 0 {
		if _, err := mutators[0].Mutate(ctx, dcb.builders[0].mutation); err != nil {
			return nil, err
		}
	}
	return nodes, nil
}

// SaveX is like Save, but panics if an error occurs.
func (dcb *DocumentCreateBulk) SaveX(ctx context.Context) []*Document {
	v, err := dcb.Save(ctx)
	if err != nil {
		panic(err)
	}
	return v
}

// Exec executes the query.
func (dcb *DocumentCreateBulk) Exec(ctx context.Context) error {
	_, err := dcb.Save(ctx)
	return err
}

// ExecX is like Exec, but panics if an error occurs.
func (dcb *DocumentCreateBulk) ExecX(ctx context.Context) {
	if err := dcb.Exec(ctx); err != nil {
		panic(err)
	}
}
