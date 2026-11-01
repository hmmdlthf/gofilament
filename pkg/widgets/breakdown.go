package widgets

import (
	"fmt"
	"html"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
)

// BreakdownItem represents an individual category slice in a breakdown distribution.
type BreakdownItem struct {
	Label      string  `json:"label"`
	Value      float64 `json:"value"`
	Formatted  string  `json:"formatted,omitempty"`
	Percentage float64 `json:"percentage"`
	Color      string  `json:"color"` // hex or color token
	Icon       string  `json:"icon,omitempty"`
}

// BreakdownWidget displays a segmented distribution bar and category list.
type BreakdownWidget struct {
	BaseWidget
	heading     string
	description string
	provider    func(ctx *filament.Context) []BreakdownItem
}

func NewBreakdownWidget(heading string) *BreakdownWidget {
	w := &BreakdownWidget{
		heading: heading,
	}
	w.columnSpan = 2
	return w
}

func (w *BreakdownWidget) Description(desc string) *BreakdownWidget {
	w.description = desc
	return w
}

func (w *BreakdownWidget) Items(provider func(ctx *filament.Context) []BreakdownItem) *BreakdownWidget {
	w.provider = provider
	return w
}

func (w *BreakdownWidget) GetHeading() string {
	return w.heading
}

func (w *BreakdownWidget) GetDescription() string {
	return w.description
}

func (w *BreakdownWidget) EvaluateItems(ctx *filament.Context) []BreakdownItem {
	if w.provider != nil {
		items := w.provider(ctx)
		var total float64
		for _, it := range items {
			total += it.Value
		}
		if total > 0 {
			for i := range items {
				if items[i].Percentage == 0 {
					items[i].Percentage = (items[i].Value / total) * 100.0
				}
			}
		}
		return items
	}
	return nil
}

func (w *BreakdownWidget) WidgetType() string {
	return "breakdown"
}

func (w *BreakdownWidget) ToWidgetDTO(ctx *filament.Context) map[string]any {
	return map[string]any{
		"type":        "breakdown",
		"heading":     w.heading,
		"description": w.description,
		"items":       w.EvaluateItems(ctx),
	}
}

func (w *BreakdownWidget) Render(ctx *filament.Context) string {
	items := w.EvaluateItems(ctx)
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`
<div class="filament-card">
  <div style="margin-bottom: 1rem;">
    <h3 style="font-size: 1.125rem; font-weight: 600;">%s</h3>`, html.EscapeString(w.heading)))

	if w.description != "" {
		sb.WriteString(fmt.Sprintf(`<p style="font-size: 0.875rem; color: var(--gray-500); margin-top: 0.25rem;">%s</p>`, html.EscapeString(w.description)))
	}
	sb.WriteString(`</div>`)

	// Multi-color segmented progress bar
	sb.WriteString(`<div style="display: flex; height: 12px; width: 100%; border-radius: 9999px; overflow: hidden; margin-bottom: 1.25rem; background-color: var(--gray-100);">`)
	for _, it := range items {
		sb.WriteString(fmt.Sprintf(`<div style="width: %.1f%%; background-color: %s;" title="%s: %.1f%%"></div>`, it.Percentage, it.Color, html.EscapeString(it.Label), it.Percentage))
	}
	sb.WriteString(`</div>`)

	// Category items list
	sb.WriteString(`<div style="display: flex; flex-direction: column; gap: 0.75rem;">`)
	for _, it := range items {
		sb.WriteString(fmt.Sprintf(`
  <div style="display: flex; justify-content: space-between; align-items: center; font-size: 0.875rem;">
    <div style="display: flex; align-items: center; gap: 0.5rem;">
      <span style="display: inline-block; width: 10px; height: 10px; border-radius: 50%%; background-color: %s;"></span>
      <span style="font-weight: 500;">%s</span>
    </div>
    <div style="display: flex; gap: 0.75rem; align-items: center;">
      <span style="color: var(--gray-500); font-size: 0.8125rem;">%s</span>
      <span style="font-weight: 600;">%.1f%%</span>
    </div>
  </div>`,
			it.Color,
			html.EscapeString(it.Label),
			html.EscapeString(it.Formatted),
			it.Percentage,
		))
	}
	sb.WriteString(`</div></div>`)
	return sb.String()
}
