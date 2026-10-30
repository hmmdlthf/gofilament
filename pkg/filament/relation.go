package filament

import (
	"github.com/golaravel/gofilament/pkg/tables"
)

// RelationManager provides nested CRUD tables for related models.
type RelationManager interface {
	Relationship() string
	Title() string
	Table(ctx *Context) string
}

// BaseRelationManager provides a simple implementation of RelationManager.
type BaseRelationManager[T any] struct {
	relationship string
	title        string
	tableDef     *tables.Table[T]
}

func NewRelationManager[T any](relationship, title string) *BaseRelationManager[T] {
	return &BaseRelationManager[T]{
		relationship: relationship,
		title:        title,
		tableDef:     tables.New[T](),
	}
}

func (rm *BaseRelationManager[T]) Relationship() string {
	return rm.relationship
}

func (rm *BaseRelationManager[T]) Title() string {
	return rm.title
}

func (rm *BaseRelationManager[T]) Table() *tables.Table[T] {
	return rm.tableDef
}
