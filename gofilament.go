// Package gofilament provides a high-performance, type-safe full-stack admin panel
// and application toolkit for GoLaravel and Ent ORM.
package gofilament

import (
	"github.com/golaravel/gofilament/pkg/activitylog"
	"github.com/golaravel/gofilament/pkg/entdriver"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/support"
)

// Version is the current GoFilament release version.
const Version = "v1.26.0"

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

// LogActivity initiates a new fluent audit trail activity log.
func LogActivity(description ...string) *activitylog.Builder {
	return activitylog.Log(description...)
}

// EnableAuditLog automatically captures create, update, and delete audit trails on a resource.
func EnableAuditLog[T any](res *filament.BaseResource[T], store ...activitylog.Store) *filament.BaseResource[T] {
	return activitylog.EnableAuditLog[T](res, store...)
}

// NewActivityResource returns a pre-configured audit trail administrative resource.
func NewActivityResource(store ...activitylog.Store) *filament.BaseResource[*activitylog.Activity] {
	return activitylog.NewActivityResource(store...)
}

// NewSupportResource returns a pre-configured live chat and AI support concierge administrative resource.
func NewSupportResource(opts ...support.SupportResourceOption) *filament.BaseResource[*support.Ticket] {
	return support.NewSupportResource(opts...)
}
