package filament

import (
	"github.com/golaravel/golaravel/pkg/router"
)

// GlobalSearchAction represents a quick action on a search result.
type GlobalSearchAction struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Icon  string `json:"icon,omitempty"`
}

// GlobalSearchResult represents a single search match across any resource.
type GlobalSearchResult struct {
	Title   string               `json:"title"`
	URL     string               `json:"url"`
	Details map[string]string    `json:"details,omitempty"`
	Group   string               `json:"group"`
	Icon    string               `json:"icon,omitempty"`
	Badge   string               `json:"badge,omitempty"`
	Actions []GlobalSearchAction `json:"actions,omitempty"`
	Score   float64              `json:"score,omitempty"`
}

// GlobalSearchGroup represents a categorized group of search results.
type GlobalSearchGroup struct {
	Name  string               `json:"name"`
	Count int                  `json:"count"`
	Items []GlobalSearchResult `json:"items"`
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
	GetRelations() []RelationManager
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
