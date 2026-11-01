package widgets

import (
	"fmt"
	"html"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
)

// ProgressItem represents a target or quota progress meter.
type ProgressItem struct {
	Label      string  `json:"label"`
	Current    float64 `json:"current"`
	Max        float64 `json:"max"`
	Unit       string  `json:"unit,omitempty"`
	Percentage float64 `json:"percentage"`
	Color      string  `json:"color,omitempty"` // emerald, amber, red, sky, purple
	Subtitle   string  `json:"subtitle,omitempty"`
}

// ProgressWidget displays quota meters, target achievements, or capacity monitors.
type ProgressWidget struct {
	BaseWidget
	heading     string
	description string
	provider    func(ctx *filament.Context) []ProgressItem
}

func NewProgressWidget(heading string) *ProgressWidget {
	w := &ProgressWidget{
		heading: heading,
	}
	w.columnSpan = 2
	return w
}

func (w *ProgressWidget) Description(desc string) *ProgressWidget {
	w.description = desc
	return w
}

func (w *ProgressWidget) Items(provider func(ctx *filament.Context) []ProgressItem) *ProgressWidget {
	w.provider = provider
	return w
}

func (w *ProgressWidget) GetHeading() string {
	return w.heading
}

func (w *ProgressWidget) GetDescription() string {
	return w.description
}

func (w *ProgressWidget) EvaluateItems(ctx *filament.Context) []ProgressItem {
	if w.provider != nil {
		items := w.provider(ctx)
		for i := range items {
			if items[i].Max > 0 && items[i].Percentage == 0 {
				items[i].Percentage = (items[i].Current / items[i].Max) * 100.0
			}
		}
		return items
	}
	return nil
}

func (w *ProgressWidget) WidgetType() string {
	return "progress"
}

func (w *ProgressWidget) ToWidgetDTO(ctx *filament.Context) map[string]any {
	return map[string]any{
		"type":        "progress",
		"heading":     w.heading,
		"description": w.description,
		"items":       w.EvaluateItems(ctx),
	}
}

func (w *ProgressWidget) Render(ctx *filament.Context) string {
	items := w.EvaluateItems(ctx)
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf(`
<div class="filament-card">
  <div style="margin-bottom: 1rem;">
    <h3 style="font-size: 1.125rem; font-weight: 600;">%s</h3>`, html.EscapeString(w.heading)))

	if w.description != "" {
		sb.WriteString(fmt.Sprintf(`<p style="font-size: 0.875rem; color: var(--gray-500); margin-top: 0.25rem;">%s</p>`, html.EscapeString(w.description)))
	}
	sb.WriteString(`</div><div style="display: flex; flex-direction: column; gap: 1rem;">`)

	for _, item := range items {
		barColor := "#10b981"
		if item.Color == "amber" || item.Color == "warning" {
			barColor = "#f59e0b"
		} else if item.Color == "red" || item.Color == "danger" {
			barColor = "#ef4444"
		} else if item.Color == "sky" || item.Color == "info" {
			barColor = "#0284c7"
		}

		pct := item.Percentage
		if pct > 100 {
			pct = 100
		}

		sb.WriteString(fmt.Sprintf(`
  <div>
    <div style="display: flex; justify-content: space-between; font-size: 0.875rem; font-weight: 500; margin-bottom: 0.375rem;">
      <span>%s</span>
      <span style="font-weight: 600;">%.1f%%</span>
    </div>
    <div style="height: 8px; width: 100%%; background-color: var(--gray-200); border-radius: 9999px; overflow: hidden;">
      <div style="height: 100%%; width: %.1f%%; background-color: %s; border-radius: 9999px; transition: width 0.3s ease;"></div>
    </div>`,
			html.EscapeString(item.Label),
			item.Percentage,
			pct,
			barColor,
		))

		if item.Subtitle != "" {
			sb.WriteString(fmt.Sprintf(`<div style="font-size: 0.75rem; color: var(--gray-500); margin-top: 0.25rem;">%s</div>`, html.EscapeString(item.Subtitle)))
		}

		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div></div>`)
	return sb.String()
}
