package widgets

import (
	"fmt"
	"html"
	"math"
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
	values          []float64 // trend line (ChartValues); takes precedence over chart
	url             string    // click-through drilldown URL
	change          *float64  // % change against the comparison period
	changeCaption   string    // e.g. "vs last week"
	lowerIsBetter   bool      // a fall is good news (costs, returns, overdue)
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

// ChartValues sets the stat's trend line from any numbers (amounts, not
// only counts). It replaces Chart.
func (s *Stat) ChartValues(values []float64) *Stat {
	s.values = values
	return s
}

func (s *Stat) URL(u string) *Stat {
	s.url = u
	return s
}

// Change shows the percentage change against a comparison period, e.g.
// Change(12.5, "vs last week") draws "+12.5% vs last week" with an up arrow.
func (s *Stat) Change(percent float64, caption ...string) *Stat {
	s.change = &percent
	if len(caption) > 0 {
		s.changeCaption = caption[0]
	}
	return s
}

// ChangeBetween computes the change from previous to current. Nothing is
// shown when previous is zero (there is nothing to compare with).
func (s *Stat) ChangeBetween(current, previous float64, caption ...string) *Stat {
	if previous == 0 {
		return s
	}
	return s.Change((current-previous)/math.Abs(previous)*100, caption...)
}

// LowerIsBetter colours a fall green and a rise red (costs, returns, debts).
func (s *Stat) LowerIsBetter() *Stat {
	s.lowerIsBetter = true
	return s
}

// GetChange returns the percentage change, if one was set.
func (s *Stat) GetChange() (float64, bool) {
	if s.change == nil {
		return 0, false
	}
	return *s.change, true
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

// trend is the stat's trend line as numbers.
func (s *Stat) trend() []float64 {
	if len(s.values) > 0 {
		return s.values
	}
	out := make([]float64, 0, len(s.chart))
	for _, v := range s.chart {
		out = append(out, float64(v))
	}
	return out
}

// changeText is the change as shown, e.g. "+12.5% vs last week".
func (s *Stat) changeText() string {
	if s.change == nil {
		return ""
	}
	t := fmt.Sprintf("%+.1f%%", *s.change)
	if s.changeCaption != "" {
		t += " " + s.changeCaption
	}
	return t
}

func (s *Stat) renderSparkline() string {
	values := s.trend()
	if len(values) < 2 {
		return ""
	}

	minVal := values[0]
	maxVal := values[0]
	for _, v := range values {
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
	step := float64(width) / float64(len(values)-1)

	var points []string
	for i, v := range values {
		x := float64(i) * step
		// Invert y because SVG y goes downwards
		normalized := (v - minVal) / diff
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
			"chart":            s.trend(),
			"url":              s.url,
			"change":           s.change,
			"change_caption":   s.changeCaption,
			"lower_is_better":  s.lowerIsBetter,
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
		if c := s.changeText(); c != "" {
			good := (*s.change >= 0) != s.lowerIsBetter
			changeColor := "text-emerald-600"
			if !good {
				changeColor = "text-red-500"
			}
			descHTML += fmt.Sprintf(`<div style="margin-top: 0.25rem; font-size: 0.8125rem;" class="%s">%s</div>`,
				changeColor, html.EscapeString(c))
		}
		if s.description != "" {
			descHTML += fmt.Sprintf(`<div style="margin-top: 0.5rem; font-size: 0.8125rem;" class="%s">%s</div>`,
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
