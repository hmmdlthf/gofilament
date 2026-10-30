package filament

import (
	"github.com/golaravel/golaravel/pkg/router"
)

// GlobalSearchResult represents a single search match across any resource.
type GlobalSearchResult struct {
	Title   string            `json:"title"`
	URL     string            `json:"url"`
	Details map[string]string `json:"details,omitempty"`
	Group   string            `json:"group"`
}

// ResourceInterface defines the contract for all CRUD resources.
type ResourceInterface interface {
	ID() string
	GetModelName() string
	GetPluralModelName() string
	Slug() string
	NavigationItem(ctx *Context) *NavigationItem
	RegisterRoutes(p *Panel, r *router.Router)
	IsGloballySearchable() bool
	SearchGlobal(ctx *Context, query string) []GlobalSearchResult
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
