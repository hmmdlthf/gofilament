package widgets

import (
	"fmt"
	"html"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
)

// QuickActionItem represents a fast shortcut button with an action route.
type QuickActionItem struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon"`
	Color       string `json:"color,omitempty"` // emerald, amber, sky, purple, red, gray
	URL         string `json:"url"`
	Badge       string `json:"badge,omitempty"`
}

// QuickActionsWidget displays a grid of rapid action buttons.
type QuickActionsWidget struct {
	BaseWidget
	heading     string
	description string
	provider    func(ctx *filament.Context) []QuickActionItem
}

func NewQuickActionsWidget(heading string) *QuickActionsWidget {
	w := &QuickActionsWidget{
		heading: heading,
	}
	w.columnSpan = 1
	return w
}

func (w *QuickActionsWidget) Description(desc string) *QuickActionsWidget {
	w.description = desc
	return w
}

func (w *QuickActionsWidget) Actions(provider func(ctx *filament.Context) []QuickActionItem) *QuickActionsWidget {
	w.provider = provider
	return w
}

func (w *QuickActionsWidget) GetHeading() string {
	return w.heading
}

func (w *QuickActionsWidget) GetDescription() string {
	return w.description
}

func (w *QuickActionsWidget) EvaluateActions(ctx *filament.Context) []QuickActionItem {
	if w.provider != nil {
		return w.provider(ctx)
	}
	return nil
}

func (w *QuickActionsWidget) WidgetType() string {
	return "quick_actions"
}

func (w *QuickActionsWidget) ToWidgetDTO(ctx *filament.Context) map[string]any {
	return map[string]any{
		"type":        "quick_actions",
		"heading":     w.heading,
		"description": w.description,
		"actions":     w.EvaluateActions(ctx),
	}
}

func (w *QuickActionsWidget) Render(ctx *filament.Context) string {
	actions := w.EvaluateActions(ctx)
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`
<div class="filament-card">
  <div style="margin-bottom: 0.875rem;">
    <h3 style="font-size: 1.125rem; font-weight: 600;">%s</h3>`, html.EscapeString(w.heading)))

	if w.description != "" {
		sb.WriteString(fmt.Sprintf(`<p style="font-size: 0.875rem; color: var(--gray-500); margin-top: 0.25rem;">%s</p>`, html.EscapeString(w.description)))
	}
	sb.WriteString(`</div><div style="display: flex; flex-direction: column; gap: 0.5rem;">`)

	for _, a := range actions {
		sb.WriteString(fmt.Sprintf(`
<a href="%s" style="display: flex; align-items: center; justify-content: space-between; padding: 0.625rem 0.875rem; border-radius: 0.5rem; border: 1px solid var(--gray-200); text-decoration: none; color: inherit; transition: background-color 0.15s ease;" onmouseover="this.style.backgroundColor='var(--gray-50)'" onmouseout="this.style.backgroundColor='transparent'">
  <div>
    <div style="font-size: 0.875rem; font-weight: 600;">%s</div>`,
			html.EscapeString(a.URL),
			html.EscapeString(a.Label),
		))

		if a.Description != "" {
			sb.WriteString(fmt.Sprintf(`<div style="font-size: 0.75rem; color: var(--gray-500); margin-top: 0.125rem;">%s</div>`, html.EscapeString(a.Description)))
		}

		sb.WriteString(`</div><span style="font-size: 1rem; color: var(--gray-400);">→</span></a>`)
	}

	sb.WriteString(`</div></div>`)
	return sb.String()
}
