package widgets_test

import (
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/widgets"
)

func TestStatsOverviewWidget(t *testing.T) {
	w := widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
		return []widgets.Stat{
			*widgets.NewStat("Total Revenue", "$45,200").Description("+12% from last month").Color("success").Chart([]int{10, 20, 15, 30, 45}),
			*widgets.NewStat("Pending Orders", "14").Description("3 urgent").Color("warning"),
		}
	})

	if w.WidgetType() != "stats_overview" {
		t.Fatalf("expected widget type stats_overview, got %s", w.WidgetType())
	}

	dto := w.ToWidgetDTO(nil)
	if dto["type"] != "stats_overview" {
		t.Fatalf("expected dto type stats_overview, got %v", dto["type"])
	}

	stats, ok := dto["stats"].([]map[string]any)
	if !ok || len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(stats))
	}
	if stats[0]["label"] != "Total Revenue" {
		t.Errorf("expected label Total Revenue, got %v", stats[0]["label"])
	}
}

func TestChartWidget(t *testing.T) {
	w := widgets.NewChartWidget("Monthly Fuel Sales", widgets.ChartTypeLine).
		Data(func(ctx *filament.Context) widgets.ChartData {
			return widgets.ChartData{
				Labels: []string{"Jan", "Feb", "Mar"},
				Datasets: []widgets.ChartDataset{
					{Label: "Diesel", Data: []float64{120, 150, 180}, BorderColor: "#10b981"},
				},
			}
		})

	if w.WidgetType() != "chart" {
		t.Fatalf("expected widget type chart, got %s", w.WidgetType())
	}

	dto := w.ToWidgetDTO(nil)
	if dto["heading"] != "Monthly Fuel Sales" {
		t.Errorf("expected heading Monthly Fuel Sales, got %v", dto["heading"])
	}
}

func TestTableWidget(t *testing.T) {
	w := widgets.NewTableWidget("Recent Orders").
		Columns(
			widgets.TableWidgetColumn{Name: "id", Label: "ID"},
			widgets.TableWidgetColumn{Name: "customer", Label: "Customer"},
			widgets.TableWidgetColumn{Name: "total", Label: "Total"},
		).
		Rows(func(ctx *filament.Context) []map[string]any {
			return []map[string]any{
				{"id": "ORD-001", "customer": "Ahmad", "total": "SAR 500"},
				{"id": "ORD-002", "customer": "Ali", "total": "SAR 350"},
			}
		})

	if w.WidgetType() != "table" {
		t.Fatalf("expected widget type table, got %s", w.WidgetType())
	}

	dto := w.ToWidgetDTO(nil)
	rows, ok := dto["rows"].([]map[string]any)
	if !ok || len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
}

func TestTimelineWidget(t *testing.T) {
	w := widgets.NewTimelineWidget("Audit History").
		Items(func(ctx *filament.Context) []widgets.TimelineItem {
			return []widgets.TimelineItem{
				{Title: "Station Shift Opened", Timestamp: "10 mins ago", Color: "emerald", User: "operator1"},
				{Title: "NFC Batch Issued", Timestamp: "1 hour ago", Color: "sky", User: "admin"},
			}
		})

	if w.WidgetType() != "timeline" {
		t.Fatalf("expected widget type timeline, got %s", w.WidgetType())
	}

	dto := w.ToWidgetDTO(nil)
	items, ok := dto["items"].([]widgets.TimelineItem)
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 timeline items, got %d", len(items))
	}
}

func TestQuickActionsWidget(t *testing.T) {
	w := widgets.NewQuickActionsWidget("Quick Actions").
		Actions(func(ctx *filament.Context) []widgets.QuickActionItem {
			return []widgets.QuickActionItem{
				{Label: "Add Partner", URL: "/admin/partners/create", Icon: "heroicon-plus"},
				{Label: "Issue NFC", URL: "/admin/nfc-chips/create", Icon: "heroicon-cpu-chip"},
			}
		})

	if w.WidgetType() != "quick_actions" {
		t.Fatalf("expected widget type quick_actions, got %s", w.WidgetType())
	}

	dto := w.ToWidgetDTO(nil)
	actions, ok := dto["actions"].([]widgets.QuickActionItem)
	if !ok || len(actions) != 2 {
		t.Fatalf("expected 2 quick actions, got %d", len(actions))
	}
}

func TestProgressWidget(t *testing.T) {
	w := widgets.NewProgressWidget("Quota Utilization").
		Items(func(ctx *filament.Context) []widgets.ProgressItem {
			return []widgets.ProgressItem{
				{Label: "Diesel Quota", Current: 7500, Max: 10000, Unit: "L", Color: "emerald"},
				{Label: "Super 95 Quota", Current: 3000, Max: 10000, Unit: "L", Color: "amber"},
			}
		})

	if w.WidgetType() != "progress" {
		t.Fatalf("expected widget type progress, got %s", w.WidgetType())
	}

	dto := w.ToWidgetDTO(nil)
	items, ok := dto["items"].([]widgets.ProgressItem)
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 progress items, got %d", len(items))
	}
	if items[0].Percentage != 75.0 {
		t.Errorf("expected 75%% percentage, got %.1f", items[0].Percentage)
	}
}

func TestBannerWidget(t *testing.T) {
	w := widgets.NewBannerWidget("Welcome to Benzat Portal", "All fuel dispensing services are operational.").
		Color("amber").
		Action("View Documentation", "https://benzat.com/docs")

	if w.WidgetType() != "banner" {
		t.Fatalf("expected widget type banner, got %s", w.WidgetType())
	}

	dto := w.ToWidgetDTO(nil)
	if dto["title"] != "Welcome to Benzat Portal" {
		t.Errorf("expected banner title, got %v", dto["title"])
	}
}

func TestBreakdownWidget(t *testing.T) {
	w := widgets.NewBreakdownWidget("Fuel Types Dispensed").
		Items(func(ctx *filament.Context) []widgets.BreakdownItem {
			return []widgets.BreakdownItem{
				{Label: "Diesel", Value: 500, Color: "#10b981"},
				{Label: "Octane 91", Value: 300, Color: "#f59e0b"},
				{Label: "Octane 95", Value: 200, Color: "#0284c7"},
			}
		})

	if w.WidgetType() != "breakdown" {
		t.Fatalf("expected widget type breakdown, got %s", w.WidgetType())
	}

	dto := w.ToWidgetDTO(nil)
	items, ok := dto["items"].([]widgets.BreakdownItem)
	if !ok || len(items) != 3 {
		t.Fatalf("expected 3 breakdown items, got %d", len(items))
	}
	if items[0].Percentage != 50.0 {
		t.Errorf("expected 50%% for Diesel, got %.1f", items[0].Percentage)
	}
}
