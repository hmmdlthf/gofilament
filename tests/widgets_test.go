package tests

import (
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/widgets"
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
