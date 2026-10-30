package forms

import (
	"fmt"
	"html"
	"strings"

	"context"
)

// SectionComponent groups form fields inside a bordered card container.
type SectionComponent struct {
	heading     string
	description string
	columns     int
	collapsible bool
	schema      []Component
}

func Section(heading string) *SectionComponent {
	return &SectionComponent{
		heading: heading,
		columns: 1,
		schema:  make([]Component, 0),
	}
}

func (s *SectionComponent) GetName() string {
	return ""
}

func (s *SectionComponent) Description(d string) *SectionComponent {
	s.description = d
	return s
}

func (s *SectionComponent) Columns(c int) *SectionComponent {
	s.columns = c
	return s
}

func (s *SectionComponent) Collapsible(c ...bool) *SectionComponent {
	s.collapsible = len(c) == 0 || c[0]
	return s
}

func (s *SectionComponent) Schema(components ...Component) *SectionComponent {
	s.schema = components
	return s
}

func (s *SectionComponent) GetSchema() []Component {
	return s.schema
}

func (s *SectionComponent) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	var body strings.Builder
	colClass := fmt.Sprintf("grid-cols-%d", s.columns)
	body.WriteString(fmt.Sprintf(`<div class="grid %s">`, colClass))
	for _, comp := range s.schema {
		body.WriteString(comp.Render(ctx, state, errors))
	}
	body.WriteString(`</div>`)

	descHTML := ""
	if s.description != "" {
		descHTML = fmt.Sprintf(`<p class="filament-description">%s</p>`, html.EscapeString(s.description))
	}

	return fmt.Sprintf(`
<div class="filament-card">
  <div style="margin-bottom: 1.25rem; border-bottom: 1px solid var(--gray-200); padding-bottom: 0.75rem;">
    <h3 style="font-size: 1.125rem; font-weight: 600;">%s</h3>
    %s
  </div>
  %s
</div>`,
		html.EscapeString(s.heading),
		descHTML,
		body.String(),
	)
}

// ----------------------------------------------------
// Grid
// ----------------------------------------------------

type GridComponent struct {
	columns int
	schema  []Component
}

func Grid(columns int) *GridComponent {
	return &GridComponent{
		columns: columns,
		schema:  make([]Component, 0),
	}
}

func (g *GridComponent) GetName() string {
	return ""
}

func (g *GridComponent) Schema(components ...Component) *GridComponent {
	g.schema = components
	return g
}

func (g *GridComponent) GetSchema() []Component {
	return g.schema
}

func (g *GridComponent) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	var body strings.Builder
	colClass := fmt.Sprintf("grid-cols-%d", g.columns)
	body.WriteString(fmt.Sprintf(`<div class="grid %s">`, colClass))
	for _, comp := range g.schema {
		body.WriteString(comp.Render(ctx, state, errors))
	}
	body.WriteString(`</div>`)
	return body.String()
}

// ----------------------------------------------------
// Tabs & Tab
// ----------------------------------------------------

type TabComponent struct {
	name   string
	label  string
	badge  string
	schema []Component
}

func Tab(name string) *TabComponent {
	return &TabComponent{
		name:   name,
		label:  name,
		schema: make([]Component, 0),
	}
}

func (t *TabComponent) Label(l string) *TabComponent {
	t.label = l
	return t
}

func (t *TabComponent) Badge(b string) *TabComponent {
	t.badge = b
	return t
}

func (t *TabComponent) Schema(components ...Component) *TabComponent {
	t.schema = components
	return t
}

type TabsComponent struct {
	tabs []*TabComponent
}

func Tabs(tabs ...*TabComponent) *TabsComponent {
	return &TabsComponent{tabs: tabs}
}

func (t *TabsComponent) GetName() string {
	return ""
}

func (t *TabsComponent) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	if len(t.tabs) == 0 {
		return ""
	}

	var nav strings.Builder
	var content strings.Builder

	nav.WriteString(`<div style="display: flex; gap: 0.5rem; border-bottom: 1px solid var(--gray-200); margin-bottom: 1.25rem;">`)
	for i, tab := range t.tabs {
		activeStyle := "border-bottom: 2px solid var(--primary-600); color: var(--primary-600); font-weight: 600;"
		if i != 0 {
			activeStyle = "color: var(--gray-500);"
		}
		badgeHTML := ""
		if tab.badge != "" {
			badgeHTML = fmt.Sprintf(` <span class="badge badge-gray">%s</span>`, tab.badge)
		}
		nav.WriteString(fmt.Sprintf(`<button type="button" style="padding: 0.5rem 1rem; background: none; border: none; cursor: pointer; %s">%s%s</button>`,
			activeStyle, html.EscapeString(tab.label), badgeHTML))

		// Render tab content
		var tabBody strings.Builder
		for _, comp := range tab.schema {
			tabBody.WriteString(comp.Render(ctx, state, errors))
		}
		display := "block"
		if i != 0 {
			display = "none"
		}
		content.WriteString(fmt.Sprintf(`<div style="display: %s;">%s</div>`, display, tabBody.String()))
	}
	nav.WriteString(`</div>`)

	return fmt.Sprintf(`<div class="filament-card">%s%s</div>`, nav.String(), content.String())
}
