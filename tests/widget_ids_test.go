package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/widgets"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

// Charts registered without WidgetID must still each serve their own data.
func TestChartsWithoutIDsServeTheirOwnData(t *testing.T) {
	chart := func(label string) *widgets.ChartWidget {
		return widgets.NewChartWidget(label, widgets.ChartTypeBar).Data(func(*filament.Context) widgets.ChartData {
			return widgets.ChartData{Labels: []string{label}}
		})
	}
	explicit := chart("Third").WidgetID("third")
	app := golaravel.New()
	p := filament.NewPanel("admin").Path("/admin").Widgets(chart("First"), chart("Second")).Widget(explicit)
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, w := range p.GetWidgets() {
		if w.ID() == "" {
			t.Fatal("widget left without an id")
		}
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest("GET", "/admin/widgets/"+w.ID()+"/data", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", w.ID(), rec.Code, rec.Body.String())
		}
		var data widgets.ChartData
		_ = json.Unmarshal(rec.Body.Bytes(), &data)
		seen[w.ID()] = data.Labels[0]
	}
	if seen["widget-1"] != "First" || seen["widget-2"] != "Second" || seen["third"] != "Third" {
		t.Fatalf("widget data by id: %v", seen)
	}
}
