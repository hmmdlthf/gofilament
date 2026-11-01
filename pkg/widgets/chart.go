package widgets

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
)

type ChartType string

const (
	ChartTypeLine      ChartType = "line"
	ChartTypeBar       ChartType = "bar"
	ChartTypePie       ChartType = "pie"
	ChartTypeDoughnut  ChartType = "doughnut"
	ChartTypePolarArea ChartType = "polarArea"
	ChartTypeRadar     ChartType = "radar"
)

// ChartDataset represents an individual dataset line or bar series.
type ChartDataset struct {
	Label           string    `json:"label"`
	Data            []float64 `json:"data"`
	BackgroundColor string    `json:"backgroundColor,omitempty"`
	BorderColor     string    `json:"borderColor,omitempty"`
	Fill            bool      `json:"fill,omitempty"`
	Tension         float64   `json:"tension,omitempty"`
}

// ChartData represents data passed into Chart.js or REST API.
type ChartData struct {
	Labels   []string       `json:"labels"`
	Datasets []ChartDataset `json:"datasets"`
}

// ChartWidget renders interactive Chart.js graphs with dynamic filter scopes and drilldowns.
type ChartWidget struct {
	BaseWidget
	heading          string
	description      string
	chartType        ChartType
	maxHeight        int
	dataFunc         func(ctx *filament.Context, filter string) ChartData
	filterPeriod     map[string]string
	defaultFilter    string
	drilldownPattern string
	enableDateRange  bool
}

// NewChartWidget initializes a new ChartWidget with a heading and chart type.
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

func (w *ChartWidget) WidgetID(id string) *ChartWidget {
	w.id = id
	return w
}

func (w *ChartWidget) Heading(h string) *ChartWidget {
	w.heading = h
	return w
}

func (w *ChartWidget) Description(d string) *ChartWidget {
	w.description = d
	return w
}

func (w *ChartWidget) MaxHeight(h int) *ChartWidget {
	w.maxHeight = h
	return w
}

func (w *ChartWidget) Type(t ChartType) *ChartWidget {
	w.chartType = t
	return w
}

func (w *ChartWidget) Data(fn func(ctx *filament.Context) ChartData) *ChartWidget {
	w.dataFunc = func(ctx *filament.Context, filter string) ChartData {
		return fn(ctx)
	}
	return w
}

func (w *ChartWidget) FilterData(fn func(ctx *filament.Context, filter string) ChartData) *ChartWidget {
	w.dataFunc = fn
	return w
}

func (w *ChartWidget) Filters(periods map[string]string) *ChartWidget {
	w.filterPeriod = periods
	return w
}

func (w *ChartWidget) DefaultFilter(df string) *ChartWidget {
	w.defaultFilter = df
	return w
}

func (w *ChartWidget) Drilldown(pattern string) *ChartWidget {
	w.drilldownPattern = pattern
	return w
}

func (w *ChartWidget) DateRangePicker(enabled ...bool) *ChartWidget {
	w.enableDateRange = len(enabled) == 0 || enabled[0]
	return w
}

func (w *ChartWidget) Polling(interval string) *ChartWidget {
	w.PollingInterval(interval)
	return w
}

func (w *ChartWidget) GetHeading() string {
	return w.heading
}

func (w *ChartWidget) GetDescription() string {
	return w.description
}

func (w *ChartWidget) GetChartType() ChartType {
	return w.chartType
}

func (w *ChartWidget) GetChartTypeString() string {
	return string(w.chartType)
}

func (w *ChartWidget) GetFilters() map[string]string {
	return w.filterPeriod
}

func (w *ChartWidget) GetDefaultFilter() string {
	return w.defaultFilter
}

func (w *ChartWidget) GetDrilldownPattern() string {
	return w.drilldownPattern
}

func (w *ChartWidget) EvaluateData(ctx *filament.Context, filter string) ChartData {
	if filter == "" {
		if ctx != nil {
			filter = ctx.Query("filter")
			if filter == "" {
				filter = ctx.DefaultQuery("period", "")
			}
		}
		if filter == "" {
			filter = w.defaultFilter
		}
	}
	if w.dataFunc != nil {
		return w.dataFunc(ctx, filter)
	}
	return ChartData{
		Labels:   []string{},
		Datasets: []ChartDataset{},
	}
}

func (w *ChartWidget) EvaluateChartData(ctx *filament.Context, filter string) any {
	return w.EvaluateData(ctx, filter)
}

func (w *ChartWidget) WidgetType() string {
	return "chart"
}

func (w *ChartWidget) ToWidgetDTO(ctx *filament.Context) map[string]any {
	return map[string]any{
		"type":           "chart",
		"heading":        w.GetHeading(),
		"chart_type":     w.GetChartTypeString(),
		"filters":        w.GetFilters(),
		"default_filter": w.GetDefaultFilter(),
		"drilldown_url":  w.GetDrilldownPattern(),
		"polling":        w.GetPolling(),
		"data":           w.EvaluateChartData(ctx, ""),
	}
}

func (w *ChartWidget) Render(ctx *filament.Context) string {
	chartID := "chart_dashboard_main"
	if w.id != "" {
		chartID = fmt.Sprintf("chart_%s", w.id)
	}

	activeFilter := w.defaultFilter
	if ctx != nil && ctx.Query("filter") != "" {
		activeFilter = ctx.Query("filter")
	}

	initialData := w.EvaluateData(ctx, activeFilter)
	dataJSON, _ := json.Marshal(initialData)

	basePath := "/admin"
	if ctx != nil && ctx.Panel() != nil {
		basePath = ctx.Panel().GetPath()
	}
	dataEndpoint := fmt.Sprintf("%s/widgets/%s/data", basePath, w.ID())

	// Filter Controls HTML
	var filterControls strings.Builder
	if w.enableDateRange {
		filterControls.WriteString(fmt.Sprintf(`
<div style="display: inline-flex; align-items: center; gap: 0.375rem; font-size: 0.8125rem;">
  <input type="date" id="%s_from" class="form-control" style="padding: 0.25rem 0.5rem; font-size: 0.75rem;" onchange="window.FilamentCharts && window.FilamentCharts.dateRangeChange('%s')" />
  <span style="color: var(--gray-400);">&ndash;</span>
  <input type="date" id="%s_until" class="form-control" style="padding: 0.25rem 0.5rem; font-size: 0.75rem;" onchange="window.FilamentCharts && window.FilamentCharts.dateRangeChange('%s')" />
</div>`, chartID, chartID, chartID, chartID))
	}

	if len(w.filterPeriod) > 0 {
		filterControls.WriteString(fmt.Sprintf(`<select id="%s_filter" class="form-control" style="padding: 0.25rem 0.75rem; font-size: 0.8125rem; border-radius: 0.375rem;" onchange="window.FilamentCharts && window.FilamentCharts.update('%s', this.value)">`, chartID, chartID))
		for k, v := range w.filterPeriod {
			selected := ""
			if k == activeFilter {
				selected = "selected"
			}
			filterControls.WriteString(fmt.Sprintf(`<option value="%s" %s>%s</option>`, html.EscapeString(k), selected, html.EscapeString(v)))
		}
		filterControls.WriteString(`</select>`)
	}

	descHTML := ""
	if w.description != "" {
		descHTML = fmt.Sprintf(`<p style="font-size: 0.8125rem; color: var(--gray-500); margin: 0.25rem 0 0 0;">%s</p>`, html.EscapeString(w.description))
	}

	return fmt.Sprintf(`
<div class="filament-card chart-widget" id="widget_%s" style="padding: 1.25rem 1.5rem; position: relative;">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; flex-wrap: wrap; gap: 0.75rem;">
    <div>
      <h3 style="font-size: 1.125rem; font-weight: 600; color: var(--gray-900); margin: 0;">%s</h3>
      %s
    </div>
    <div style="display: flex; align-items: center; gap: 0.5rem;">
      %s
    </div>
  </div>
  <div style="position: relative; height: %dpx; width: 100%%; display: flex; align-items: center; justify-content: center; background-color: var(--gray-50); border-radius: 0.5rem; border: 1px dashed var(--gray-300);">
    <canvas id="%s" style="max-height: 100%%; width: 100%%;"></canvas>
    <div style="position: absolute; color: var(--gray-400); font-size: 0.875rem; pointer-events: none; opacity: 0.7;">Chart: %s (%s)</div>
  </div>
  <script type="application/json" id="%s_data">%s</script>
  <script>
  (function() {
    var chartId = '%s';
    var canvasId = '%s';
    var chartType = '%s';
    var dataEndpoint = '%s';
    var drilldownPattern = '%s';
    var polling = '%s';

    window.FilamentCharts = window.FilamentCharts || {
      instances: {},
      update: function(id, filter) {
        var inst = this.instances[id];
        if (!inst || !inst.endpoint) return;
        fetch(inst.endpoint + '?filter=' + encodeURIComponent(filter))
          .then(function(r) { return r.json(); })
          .then(function(d) {
            inst.chart.data.labels = d.labels || [];
            inst.chart.data.datasets = d.datasets || [];
            inst.chart.update();
          })
          .catch(function(err) { console.error('Failed to update chart:', err); });
      },
      dateRangeChange: function(id) {
        var fromEl = document.getElementById(id + '_from');
        var untilEl = document.getElementById(id + '_until');
        var from = fromEl ? fromEl.value : '';
        var until = untilEl ? untilEl.value : '';
        var inst = this.instances[id];
        if (!inst || !inst.endpoint) return;
        fetch(inst.endpoint + '?from=' + encodeURIComponent(from) + '&until=' + encodeURIComponent(until))
          .then(function(r) { return r.json(); })
          .then(function(d) {
            inst.chart.data.labels = d.labels || [];
            inst.chart.data.datasets = d.datasets || [];
            inst.chart.update();
          });
      }
    };

    function setupChart() {
      var canvas = document.getElementById(canvasId);
      if (!canvas) return;
      var dataScript = document.getElementById(canvasId + '_data');
      var chartData = { labels: [], datasets: [] };
      if (dataScript) {
        try { chartData = JSON.parse(dataScript.textContent); } catch(e) {}
      }

      var chartOptions = {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: true, position: 'top' }
        }
      };

      if (drilldownPattern) {
        chartOptions.onClick = function(evt, elements) {
          if (elements && elements.length > 0) {
            var elem = elements[0];
            var inst = window.FilamentCharts.instances[chartId];
            var labels = (inst && inst.chart) ? inst.chart.data.labels : chartData.labels;
            var datasets = (inst && inst.chart) ? inst.chart.data.datasets : chartData.datasets;
            var label = labels[elem.index] || '';
            var ds = datasets[elem.datasetIndex] || {};
            var val = (ds.data && ds.data[elem.index] !== undefined) ? ds.data[elem.index] : '';
            var target = drilldownPattern
              .replace(/{label}/g, encodeURIComponent(label))
              .replace(/{dataset}/g, encodeURIComponent(ds.label || ''))
              .replace(/{value}/g, encodeURIComponent(val))
              .replace(/{index}/g, elem.index);
            window.location.href = target;
          }
        };
        chartOptions.onHover = function(evt, elements) {
          canvas.style.cursor = elements && elements.length > 0 ? 'pointer' : 'default';
        };
      }

      function create() {
        if (typeof Chart !== 'undefined') {
          var chartInstance = new Chart(canvas, {
            type: chartType,
            data: chartData,
            options: chartOptions
          });
          window.FilamentCharts.instances[chartId] = { chart: chartInstance, endpoint: dataEndpoint };
        }
      }

      if (typeof Chart === 'undefined') {
        var s = document.createElement('script');
        s.src = 'https://cdn.jsdelivr.net/npm/chart.js';
        s.onload = create;
        document.head.appendChild(s);
      } else {
        create();
      }

      if (polling) {
        var ms = 10000;
        var p = polling.toLowerCase();
        if (p.endsWith('s')) ms = parseInt(p) * 1000;
        else if (p.endsWith('m')) ms = parseInt(p) * 60000;
        if (ms > 0) {
          setInterval(function() {
            var sel = document.getElementById(chartId + '_filter');
            var filter = sel ? sel.value : '';
            window.FilamentCharts.update(chartId, filter);
          }, ms);
        }
      }
    }

    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', setupChart);
    } else {
      setupChart();
    }
  })();
  </script>
</div>`,
		chartID,
		html.EscapeString(w.heading),
		descHTML,
		filterControls.String(),
		w.maxHeight,
		chartID,
		html.EscapeString(w.heading),
		w.chartType,
		chartID,
		string(dataJSON),
		chartID,
		chartID,
		w.chartType,
		dataEndpoint,
		w.drilldownPattern,
		w.polling,
	)
}
