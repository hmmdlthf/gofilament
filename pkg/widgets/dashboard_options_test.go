package widgets_test

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/widgets"
)

func TestStatChangeTrendAndLink(t *testing.T) {
	w := widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
		return []widgets.Stat{
			*widgets.NewStat("Sales", "1,200.00").ChangeBetween(1200, 1000, "vs last week").ChartValues([]float64{1.5, 2.25, 3}).URL("/app/resources/orders?filter_status=paid"),
			*widgets.NewStat("Returns", "40").Change(-5, "vs yesterday").LowerIsBetter(),
			*widgets.NewStat("New", "3").ChangeBetween(3, 0), // nothing to compare with
			*widgets.NewStat("Counts", "9").Chart([]int{1, 2, 3}),
		}
	}).Span(3).Order(-1)
	if w.ColumnSpan() != 3 || w.Sort() != -1 {
		t.Fatalf("span %d sort %d", w.ColumnSpan(), w.Sort())
	}
	stats := w.ToWidgetDTO(nil)["stats"].([]map[string]any)
	if c := stats[0]["change"].(*float64); math.Abs(*c-20) > 1e-9 || stats[0]["change_caption"] != "vs last week" {
		t.Fatalf("change: %v %v", *c, stats[0]["change_caption"])
	}
	if got := stats[0]["chart"].([]float64); len(got) != 3 || got[1] != 2.25 {
		t.Fatalf("trend values: %v", got)
	}
	if stats[1]["lower_is_better"] != true || *stats[1]["change"].(*float64) != -5 {
		t.Fatalf("lower is better: %v", stats[1])
	}
	if stats[2]["change"].(*float64) != nil {
		t.Fatalf("change against zero should be left out: %v", *stats[2]["change"].(*float64))
	}
	if got := stats[3]["chart"].([]float64); len(got) != 3 || got[2] != 3 {
		t.Fatalf("int chart as numbers: %v", got)
	}
	// JSON: an unset change is null
	b, _ := json.Marshal(stats[2])
	if !strings.Contains(string(b), `"change":null`) {
		t.Fatalf("json: %s", b)
	}
	// the web panel shows the change too
	html := w.Render(nil)
	if !strings.Contains(html, "+20.0% vs last week") || !strings.Contains(html, "-5.0% vs yesterday") {
		t.Fatalf("rendered stats lack the change")
	}
}

func TestChartFormatsOrderAndMixedSeries(t *testing.T) {
	w := widgets.NewChartWidget("P&L", widgets.ChartTypeBar).
		Description("Last 12 months").
		Money("LKR").
		Filters(map[string]string{"7d": "7 days", "30d": "30 days", "12m": "12 months", "90d": "90 days"}).
		FilterOrder("7d", "30d", "90d", "nope").
		DefaultFilter("30d").
		Span(2).
		Data(func(ctx *filament.Context) widgets.ChartData {
			return widgets.ChartData{Labels: []string{"Oct", "Nov"}, Datasets: []widgets.ChartDataset{
				{Label: "Income", Data: []float64{10, 20}},
				{Label: "Net", Type: "line", Data: []float64{4, 8}},
			}}
		})
	dto := w.ToWidgetDTO(nil)
	if dto["value_format"] != widgets.FormatMoney || dto["currency"] != "LKR" || dto["description"] != "Last 12 months" {
		t.Fatalf("formats: %v", dto)
	}
	if got := strings.Join(dto["filter_order"].([]string), ","); got != "7d,30d,90d,12m" {
		t.Fatalf("filter order %s (listed keys first, unknown ones dropped, the rest sorted)", got)
	}
	if w.ColumnSpan() != 2 {
		t.Fatalf("span %d", w.ColumnSpan())
	}
	b, _ := json.Marshal(dto["data"])
	if !strings.Contains(string(b), `"type":"line"`) || strings.Count(string(b), `"type"`) != 1 {
		t.Fatalf("dataset types: %s", b)
	}
	if w.ValueFormat(widgets.FormatPercent).ToWidgetDTO(nil)["value_format"] != "percent" {
		t.Fatal("percent format")
	}
}

func TestTableColumnsFormatsAndEmptyRows(t *testing.T) {
	w := widgets.NewTableWidget("Top products").
		Description("By revenue this month").
		EmptyText("No sales yet").
		Columns(
			widgets.TableWidgetColumn{Name: "name", Label: "Product"},
			widgets.TableWidgetColumn{Name: "revenue", Label: "Revenue", Format: "money"},
		).
		Rows(func(ctx *filament.Context) []map[string]any { return nil }).
		Span(1).Key("top")
	dto := w.ToWidgetDTO(nil)
	cols := dto["columns"].([]map[string]string)
	if cols[1]["format"] != "money" || dto["empty_text"] != "No sales yet" || dto["description"] != "By revenue this month" {
		t.Fatalf("table dto: %v", dto)
	}
	if rows := dto["rows"].([]map[string]any); rows == nil || len(rows) != 0 {
		t.Fatalf("no rows should be an empty list, got %v", dto["rows"])
	}
	if w.ID() != "top" || w.ColumnSpan() != 1 || widgets.RowURL != "_url" {
		t.Fatalf("id %q span %d", w.ID(), w.ColumnSpan())
	}
}

func TestWidgetVisibility(t *testing.T) {
	hidden := widgets.NewBreakdownWidget("Mix").Visible(func(*filament.Context) bool { return false })
	if hidden.CanView(nil) {
		t.Fatal("hidden widget visible")
	}
	if !widgets.NewProgressWidget("Targets").Span(1).CanView(nil) {
		t.Fatal("widget without a rule hidden")
	}
}
