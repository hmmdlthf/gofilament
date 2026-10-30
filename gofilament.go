// Package gofilament provides a high-performance, type-safe full-stack admin panel
// and application toolkit for GoLaravel and Ent ORM.
package gofilament

import (
	"github.com/golaravel/gofilament/pkg/entdriver"
	"github.com/golaravel/gofilament/pkg/filament"
)

// Version is the current GoFilament release version.
const Version = "v1.7.0"

// NewPanel creates a new admin panel instance.
func NewPanel(id string) *filament.Panel {
	return filament.NewPanel(id)
}

// NewResource creates a new typed CRUD resource specification.
func NewResource[T any](slug string) *filament.BaseResource[T] {
	return filament.NewResource[T](slug)
}

// BindEntClient binds an Ent entity client (e.g. client.User, client.Invoice) to a resource with zero boilerplate.
func BindEntClient[T any](res *filament.BaseResource[T], client any) *filament.BaseResource[T] {
	return entdriver.BindClient(res, client)
}
