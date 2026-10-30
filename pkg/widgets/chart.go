package widgets

import (
	"fmt"
	"html"

	"github.com/golaravel/gofilament/pkg/filament"
)

type ChartType string

const (
	ChartTypeLine     ChartType = "line"
	ChartTypeBar      ChartType = "bar"
	ChartTypePie      ChartType = "pie"
	ChartTypeDoughnut ChartType = "doughnut"
)

// ChartData represents data passed into Chart.js
type ChartData struct {
	Labels   []string
	Datasets []ChartDataset
}

type ChartDataset struct {
	Label           string
	Data            []float64
	BackgroundColor string
	BorderColor     string
	Fill            bool
}

// ChartWidget renders interactive Chart.js graphs.
type ChartWidget struct {
	BaseWidget
	heading      string
	chartType    ChartType
	maxHeight    int
	dataFunc     func(ctx *filament.Context) ChartData
	filterPeriod map[string]string
}

func NewChartWidget(heading string, chartType ChartType) *ChartWidget {
	w := &ChartWidget{
		heading:      heading,
		chartType:    chartType,
		maxHeight:    300,
		filterPeriod: make(map[string]string),
	}
	w.columnSpan = 2
	return w
}

func (w *ChartWidget) Heading(h string) *ChartWidget {
	w.heading = h
	return w
}

func (w *ChartWidget) MaxHeight(h int) *ChartWidget {
	w.maxHeight = h
	return w
}

func (w *ChartWidget) Data(fn func(ctx *filament.Context) ChartData) *ChartWidget {
	w.dataFunc = fn
	return w
}

func (w *ChartWidget) Filters(periods map[string]string) *ChartWidget {
	w.filterPeriod = periods
	return w
}

func (w *ChartWidget) Render(ctx *filament.Context) string {
	chartID := fmt.Sprintf("chart_%s", w.id)
	if w.id == "" {
		chartID = "chart_dashboard_main"
	}

	return fmt.Sprintf(`
<div class="filament-card">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem;">
    <h3 style="font-size: 1.125rem; font-weight: 600;">%s</h3>
  </div>
  <div style="position: relative; height: %dpx; display: flex; align-items: center; justify-content: center; background-color: var(--gray-50); border-radius: 0.5rem; border: 1px dashed var(--gray-300);">
    <canvas id="%s" style="max-height: 100%%; width: 100%%;"></canvas>
    <div style="position: absolute; color: var(--gray-400); font-size: 0.875rem;">Chart: %s (%s)</div>
  </div>
</div>`,
		html.EscapeString(w.heading),
		w.maxHeight,
		chartID,
		html.EscapeString(w.heading),
		w.chartType,
	)
}
