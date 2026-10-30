// Package gofilament provides a high-performance, type-safe full-stack admin panel
// and application toolkit for GoLaravel and Ent ORM.
package gofilament

import (
	"github.com/golaravel/gofilament/pkg/filament"
)

// Version is the current GoFilament release version.
const Version = "v1.0.0"

// NewPanel creates a new admin panel instance.
func NewPanel(id string) *filament.Panel {
	return filament.NewPanel(id)
}

// NewResource creates a new typed CRUD resource specification.
func NewResource[T any](slug string) *filament.BaseResource[T] {
	return filament.NewResource[T](slug)
}
