package filament

import (
	"github.com/golaravel/golaravel/pkg/router"
)

// ResourceInterface defines the contract for all CRUD resources.
type ResourceInterface interface {
	ID() string
	GetModelName() string
	GetPluralModelName() string
	Slug() string
	NavigationItem(ctx *Context) *NavigationItem
	RegisterRoutes(p *Panel, r *router.Router)
}

// PageInterface defines the contract for custom panel pages.
type PageInterface interface {
	ID() string
	Title() string
	Slug() string
	NavigationItem(ctx *Context) *NavigationItem
	RegisterRoutes(p *Panel, r *router.Router)
}

// WidgetInterface defines the contract for dashboard widgets.
type WidgetInterface interface {
	ID() string
	Sort() int
	ColumnSpan() int
}
