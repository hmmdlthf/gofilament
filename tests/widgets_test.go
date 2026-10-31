package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/widgets"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func TestStatsOverviewWidget(t *testing.T) {
	widget := widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
		return []widgets.Stat{
			*widgets.NewStat("Total Revenue", "$128,430").
				Description("32% increase this month").
				Color("success").
				Chart([]int{15, 22, 35, 45, 60, 78, 128}),

			*widgets.NewStat("Active Subscriptions", "1,420").
				Description("3% decrease").
				Color("danger").
				Chart([]int{1500, 1480, 1460, 1450, 1430, 1420}),
		}
	})

	html := widget.Render(nil)

	// Assert labels and values
	if !strings.Contains(html, "Total Revenue") {
		t.Errorf("Expected stat label 'Total Revenue'")
	}
	if !strings.Contains(html, "$128,430") {
		t.Errorf("Expected stat value '$128,430'")
	}
	if !strings.Contains(html, "32% increase this month") {
		t.Errorf("Expected description '32%% increase this month'")
	}

	// Assert SVG sparklines
	if !strings.Contains(html, "<svg") || !strings.Contains(html, "<polyline") {
		t.Errorf("Expected SVG sparkline graph in rendered stat card, got:\n%s", html)
	}
	if !strings.Contains(html, `stroke="#10b981"`) {
		t.Errorf("Expected green sparkline for success stat")
	}
	if !strings.Contains(html, `stroke="#ef4444"`) {
		t.Errorf("Expected red sparkline for danger stat")
	}
}

func TestChartWidget(t *testing.T) {
	chart := widgets.NewChartWidget("Orders Per Month", widgets.ChartTypeLine).
		MaxHeight(350)

	html := chart.Render(nil)

	if !strings.Contains(html, "Orders Per Month") {
		t.Errorf("Expected chart heading 'Orders Per Month'")
	}
	if !strings.Contains(html, `<canvas id="chart_dashboard_main"`) {
		t.Errorf("Expected canvas element with id 'chart_dashboard_main'")
	}
}

func TestChartWidgetDynamicFiltersAndDrilldown(t *testing.T) {
	chart := widgets.NewChartWidget("Revenue Analytics", widgets.ChartTypeBar).
		WidgetID("revenue_analytics").
		MaxHeight(400).
		Filters(map[string]string{
			"today": "Today",
			"7d":    "Last 7 Days",
			"30d":   "Last 30 Days",
		}).
		DefaultFilter("7d").
		Drilldown("/admin/orders?status={label}&source=chart").
		DateRangePicker(true).
		Polling("15s").
		FilterData(func(ctx *filament.Context, filter string) widgets.ChartData {
			if filter == "today" {
				return widgets.ChartData{
					Labels: []string{"Morning", "Afternoon", "Evening"},
					Datasets: []widgets.ChartDataset{
						{Label: "Today Revenue", Data: []float64{1200, 3400, 2100}},
					},
				}
			}
			return widgets.ChartData{
				Labels: []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
				Datasets: []widgets.ChartDataset{
					{Label: "Weekly Revenue", Data: []float64{5000, 7200, 6100, 8900, 9400, 11200, 10500}},
				},
			}
		})

	html := chart.Render(nil)

	// Check filters dropdown rendered
	if !strings.Contains(html, `<select id="chart_revenue_analytics_filter"`) {
		t.Errorf("Expected filter select element in chart HTML")
	}
	if !strings.Contains(html, `<option value="7d" selected>Last 7 Days</option>`) {
		t.Errorf("Expected default selected option 7d")
	}

	// Check date range picker rendered
	if !strings.Contains(html, `id="chart_revenue_analytics_from"`) || !strings.Contains(html, `id="chart_revenue_analytics_until"`) {
		t.Errorf("Expected date range inputs in chart HTML")
	}

	// Check drilldown pattern and polling interval embedded in script
	if !strings.Contains(html, `/admin/orders?status={label}&source=chart`) {
		t.Errorf("Expected drilldownPattern embedded in script")
	}
	if !strings.Contains(html, `var polling = '15s';`) {
		t.Errorf("Expected polling 15s in script")
	}

	// Check initial data JSON embedded
	if !strings.Contains(html, `Weekly Revenue`) || !strings.Contains(html, `11200`) {
		t.Errorf("Expected pre-evaluated 7d dataset in embedded JSON data")
	}
}

func TestStatCardDrilldownLink(t *testing.T) {
	widget := widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
		return []widgets.Stat{
			*widgets.NewStat("Pending Invoices", "24").
				URL("/admin/invoices?filter_status=pending").
				Color("warning"),
			*widgets.NewStat("Active Projects", "12"),
		}
	})

	html := widget.Render(nil)

	// Pending Invoices should be wrapped in an anchor link
	if !strings.Contains(html, `<a href="/admin/invoices?filter_status=pending" class="stat-card filament-card"`) {
		t.Errorf("Expected stat card wrapped in drilldown anchor link, got:\n%s", html)
	}

	// Active Projects should be a regular div
	if !strings.Contains(html, `<div class="stat-card filament-card">`) {
		t.Errorf("Expected standard div for non-drilldown stat card")
	}
}

func TestDashboardWidgetGridRendering(t *testing.T) {
	app := golaravel.New()
	panel := filament.NewPanel("admin").Path("/admin")

	statsWidget := widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
		return []widgets.Stat{
			*widgets.NewStat("Total Users", "5,280").Color("primary"),
		}
	})
	statsWidget.SetSort(1)
	statsWidget.SetColumnSpan(3)

	chartWidget := widgets.NewChartWidget("Daily Signups", widgets.ChartTypeLine).
		WidgetID("daily_signups").
		Data(func(ctx *filament.Context) widgets.ChartData {
			return widgets.ChartData{
				Labels: []string{"Mon", "Tue", "Wed"},
				Datasets: []widgets.ChartDataset{
					{Label: "Signups", Data: []float64{10, 20, 30}},
				},
			}
		})
	chartWidget.SetSort(2)
	chartWidget.SetColumnSpan(2)

	panel.Widgets(statsWidget, chartWidget)
	_ = panel.Register(app)

	req := httptest.NewRequest("GET", "/admin", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from dashboard, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Total Users") || !strings.Contains(body, "5,280") {
		t.Errorf("Expected stats widget rendered on dashboard")
	}
	if !strings.Contains(body, "Daily Signups") || !strings.Contains(body, "chart_daily_signups") {
		t.Errorf("Expected chart widget rendered on dashboard")
	}
	if !strings.Contains(body, "col-span-1 md:col-span-3") {
		t.Errorf("Expected 3-column span grid container for stats widget")
	}
	if !strings.Contains(body, "col-span-1 md:col-span-2") {
		t.Errorf("Expected 2-column span grid container for chart widget")
	}
}

func TestWidgetDynamicDataWebAndRESTEndpoints(t *testing.T) {
	app := golaravel.New()
	panel := filament.NewPanel("admin").Path("/admin")

	chartWidget := widgets.NewChartWidget("Sales Performance", widgets.ChartTypeBar).
		WidgetID("sales_perf").
		FilterData(func(ctx *filament.Context, filter string) widgets.ChartData {
			if filter == "quarter" {
				return widgets.ChartData{
					Labels: []string{"Q1", "Q2", "Q3", "Q4"},
					Datasets: []widgets.ChartDataset{
						{Label: "Quarterly Sales", Data: []float64{100000, 150000, 200000, 250000}},
					},
				}
			}
			return widgets.ChartData{
				Labels: []string{"Jan", "Feb", "Mar"},
				Datasets: []widgets.ChartDataset{
					{Label: "Monthly Sales", Data: []float64{25000, 35000, 40000}},
				},
			}
		})

	panel.Widget(chartWidget)
	_ = panel.Register(app)

	// 1. Web AJAX data endpoint: GET /admin/widgets/sales_perf/data?filter=quarter
	webReq := httptest.NewRequest("GET", "/admin/widgets/sales_perf/data?filter=quarter", nil)
	webRec := httptest.NewRecorder()
	app.ServeHTTP(webRec, webReq)

	if webRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from web widget data endpoint, got %d", webRec.Code)
	}

	var webData widgets.ChartData
	if err := json.Unmarshal(webRec.Body.Bytes(), &webData); err != nil {
		t.Fatalf("Failed to parse widget data JSON: %v", err)
	}
	if len(webData.Labels) != 4 || webData.Labels[0] != "Q1" {
		t.Errorf("Expected quarterly data labels [Q1, Q2, Q3, Q4], got %+v", webData.Labels)
	}
	if len(webData.Datasets) == 0 || webData.Datasets[0].Data[0] != 100000 {
		t.Errorf("Expected dataset value 100000 for Q1")
	}

	// 2. REST API endpoint: GET /admin/api/widgets
	apiWidgetsReq := httptest.NewRequest("GET", "/admin/api/widgets", nil)
	apiWidgetsRec := httptest.NewRecorder()
	app.ServeHTTP(apiWidgetsRec, apiWidgetsReq)

	if apiWidgetsRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /admin/api/widgets, got %d", apiWidgetsRec.Code)
	}

	var apiWidgetsResp struct {
		Widgets []map[string]any `json:"widgets"`
	}
	if err := json.Unmarshal(apiWidgetsRec.Body.Bytes(), &apiWidgetsResp); err != nil {
		t.Fatalf("Failed to parse /admin/api/widgets response: %v", err)
	}
	if len(apiWidgetsResp.Widgets) == 0 {
		t.Fatalf("Expected at least 1 widget in API response")
	}
	w0 := apiWidgetsResp.Widgets[0]
	if w0["type"] != "chart" || w0["heading"] != "Sales Performance" {
		t.Errorf("Expected chart type and heading in API widget metadata, got %+v", w0)
	}

	// 3. REST API single data endpoint: GET /admin/api/widgets/sales_perf/data?filter=monthly
	apiDataReq := httptest.NewRequest("GET", "/admin/api/widgets/sales_perf/data?filter=monthly", nil)
	apiDataRec := httptest.NewRecorder()
	app.ServeHTTP(apiDataRec, apiDataReq)

	if apiDataRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from /admin/api/widgets/sales_perf/data, got %d", apiDataRec.Code)
	}

	var apiChartData widgets.ChartData
	_ = json.Unmarshal(apiDataRec.Body.Bytes(), &apiChartData)
	if len(apiChartData.Labels) != 3 || apiChartData.Labels[0] != "Jan" {
		t.Errorf("Expected monthly data [Jan, Feb, Mar], got %+v", apiChartData.Labels)
	}
}
