package widgets

import (
	"fmt"
	"html"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
)

// Stat represents an individual metric card.
type Stat struct {
	label           string
	value           string
	description     string
	descriptionIcon string
	color           string // success, danger, warning, primary, gray
	chart           []int
	url             string // click-through drilldown URL
}

// NewStat creates a new KPI stat metric.
func NewStat(label, value string) *Stat {
	return &Stat{
		label: label,
		value: value,
		color: "primary",
	}
}

func (s *Stat) Description(desc string) *Stat {
	s.description = desc
	return s
}

func (s *Stat) DescriptionIcon(icon string) *Stat {
	s.descriptionIcon = icon
	return s
}

func (s *Stat) Color(color string) *Stat {
	s.color = color
	return s
}

func (s *Stat) Chart(data []int) *Stat {
	s.chart = data
	return s
}

func (s *Stat) URL(u string) *Stat {
	s.url = u
	return s
}

func (s *Stat) GetLabel() string {
	return s.label
}

func (s *Stat) GetValue() string {
	return s.value
}

func (s *Stat) GetURL() string {
	return s.url
}

func (s *Stat) renderSparkline() string {
	if len(s.chart) < 2 {
		return ""
	}

	minVal := s.chart[0]
	maxVal := s.chart[0]
	for _, v := range s.chart {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	diff := maxVal - minVal
	if diff == 0 {
		diff = 1
	}

	width := 120
	height := 36
	step := float64(width) / float64(len(s.chart)-1)

	var points []string
	for i, v := range s.chart {
		x := float64(i) * step
		// Invert y because SVG y goes downwards
		normalized := float64(v-minVal) / float64(diff)
		y := float64(height) - (normalized * float64(height-6)) - 3
		points = append(points, fmt.Sprintf("%.1f,%.1f", x, y))
	}

	strokeColor := "#10b981"
	if s.color == "danger" {
		strokeColor = "#ef4444"
	} else if s.color == "warning" {
		strokeColor = "#f59e0b"
	} else if s.color == "primary" {
		strokeColor = "#3b82f6"
	}

	return fmt.Sprintf(`
<svg width="%d" height="%d" viewBox="0 0 %d %d" style="overflow: visible;">
  <polyline fill="none" stroke="%s" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" points="%s" />
</svg>`, width, height, width, height, strokeColor, strings.Join(points, " "))
}

// StatsOverviewWidget renders a grid of Stat cards with sparklines.
type StatsOverviewWidget struct {
	BaseWidget
	statsProvider func(ctx *filament.Context) []Stat
}

func NewStatsOverviewWidget(provider func(ctx *filament.Context) []Stat) *StatsOverviewWidget {
	w := &StatsOverviewWidget{
		statsProvider: provider,
	}
	w.columnSpan = 3
	return w
}

func (w *StatsOverviewWidget) EvaluateStats(ctx *filament.Context) []Stat {
	if w.statsProvider == nil {
		return nil
	}
	return w.statsProvider(ctx)
}

func (w *StatsOverviewWidget) EvaluateStatsMap(ctx *filament.Context) []map[string]any {
	stats := w.EvaluateStats(ctx)
	var result []map[string]any
	for _, s := range stats {
		result = append(result, map[string]any{
			"label":            s.label,
			"value":            s.value,
			"description":      s.description,
			"description_icon": s.descriptionIcon,
			"color":            s.color,
			"chart":            s.chart,
			"url":              s.url,
		})
	}
	return result
}

func (w *StatsOverviewWidget) WidgetType() string {
	return "stats_overview"
}

func (w *StatsOverviewWidget) ToWidgetDTO(ctx *filament.Context) map[string]any {
	return map[string]any{
		"type":  "stats_overview",
		"stats": w.EvaluateStatsMap(ctx),
	}
}

func (w *StatsOverviewWidget) Render(ctx *filament.Context) string {
	if w.statsProvider == nil {
		return ""
	}

	stats := w.statsProvider(ctx)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<div class="grid grid-cols-1 md:grid-cols-%d gap-6 mb-6">`, len(stats)))

	for _, s := range stats {
		colorClass := "text-emerald-600"
		if s.color == "danger" {
			colorClass = "text-red-500"
		} else if s.color == "warning" {
			colorClass = "text-amber-500"
		}

		descHTML := ""
		if s.description != "" {
			descHTML = fmt.Sprintf(`<div style="margin-top: 0.5rem; font-size: 0.8125rem;" class="%s">%s</div>`,
				colorClass, html.EscapeString(s.description))
		}

		cardWrapperOpen := `<div class="stat-card filament-card">`
		cardWrapperClose := `</div>`
		if s.url != "" {
			cardWrapperOpen = fmt.Sprintf(`<a href="%s" class="stat-card filament-card" style="text-decoration: none; color: inherit; display: block; cursor: pointer; transition: transform 0.15s ease, box-shadow 0.15s ease;">`, html.EscapeString(s.url))
			cardWrapperClose = `</a>`
		}

		sb.WriteString(fmt.Sprintf(`
%s
  <div style="display: flex; justify-content: space-between; align-items: flex-start;">
    <div>
      <span style="font-size: 0.875rem; font-weight: 500; color: var(--gray-500);">%s</span>
      <div class="stat-value" style="margin-top: 0.25rem;">%s</div>
      %s
    </div>
    <div style="margin-top: 0.5rem;">
      %s
    </div>
  </div>
%s`,
			cardWrapperOpen,
			html.EscapeString(s.label),
			html.EscapeString(s.value),
			descHTML,
			s.renderSparkline(),
			cardWrapperClose,
		))
	}

	sb.WriteString(`</div>`)
	return sb.String()
}
