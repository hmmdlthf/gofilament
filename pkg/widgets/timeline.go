package widgets

import (
	"fmt"
	"html"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
)

// TimelineItem represents an individual event or audit log on the timeline.
type TimelineItem struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"` // emerald, amber, red, sky, indigo, purple
	User        string `json:"user,omitempty"`
	URL         string `json:"url,omitempty"`
	Badge       string `json:"badge,omitempty"`
}

// TimelineWidget renders an audit log, activity feed, or event history.
type TimelineWidget struct {
	BaseWidget
	heading     string
	description string
	provider    func(ctx *filament.Context) []TimelineItem
}

// NewTimelineWidget creates a new activity timeline widget.
func NewTimelineWidget(heading string) *TimelineWidget {
	w := &TimelineWidget{
		heading: heading,
	}
	w.columnSpan = 2
	return w
}

func (w *TimelineWidget) Description(desc string) *TimelineWidget {
	w.description = desc
	return w
}

func (w *TimelineWidget) Items(provider func(ctx *filament.Context) []TimelineItem) *TimelineWidget {
	w.provider = provider
	return w
}

func (w *TimelineWidget) GetHeading() string {
	return w.heading
}

func (w *TimelineWidget) GetDescription() string {
	return w.description
}

func (w *TimelineWidget) EvaluateItems(ctx *filament.Context) []TimelineItem {
	if w.provider != nil {
		return w.provider(ctx)
	}
	return nil
}

func (w *TimelineWidget) WidgetType() string {
	return "timeline"
}

func (w *TimelineWidget) ToWidgetDTO(ctx *filament.Context) map[string]any {
	return map[string]any{
		"type":        "timeline",
		"heading":     w.heading,
		"description": w.description,
		"items":       w.EvaluateItems(ctx),
	}
}

func (w *TimelineWidget) Render(ctx *filament.Context) string {
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

	if len(items) == 0 {
		sb.WriteString(`<div style="text-align: center; color: var(--gray-400); padding: 1.5rem 0;">No recent activities found.</div></div>`)
		return sb.String()
	}

	sb.WriteString(`<div class="timeline-container" style="display: flex; flex-direction: column; gap: 1rem;">`)
	for _, item := range items {
		dotColor := "#3b82f6"
		if item.Color == "emerald" || item.Color == "success" {
			dotColor = "#10b981"
		} else if item.Color == "amber" || item.Color == "warning" {
			dotColor = "#f59e0b"
		} else if item.Color == "red" || item.Color == "danger" {
			dotColor = "#ef4444"
		} else if item.Color == "purple" || item.Color == "indigo" {
			dotColor = "#8b5cf6"
		}

		sb.WriteString(fmt.Sprintf(`
  <div style="display: flex; gap: 0.875rem; align-items: flex-start;">
    <div style="width: 10px; height: 10px; border-radius: 50%%; background-color: %s; margin-top: 0.375rem; flex-shrink: 0;"></div>
    <div style="flex: 1;">
      <div style="display: flex; justify-content: space-between; align-items: baseline;">
        <span style="font-size: 0.875rem; font-weight: 600;">%s</span>
        <span style="font-size: 0.75rem; color: var(--gray-400);">%s</span>
      </div>`,
			dotColor,
			html.EscapeString(item.Title),
			html.EscapeString(item.Timestamp),
		))

		if item.Description != "" {
			sb.WriteString(fmt.Sprintf(`<div style="font-size: 0.8125rem; color: var(--gray-600); margin-top: 0.125rem;">%s</div>`, html.EscapeString(item.Description)))
		}

		sb.WriteString(`</div></div>`)
	}

	sb.WriteString(`</div></div>`)
	return sb.String()
}
