package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/widgets"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func TestHiddenWidgetsServeNoData(t *testing.T) {
	chart := widgets.NewChartWidget("Revenue secrets", widgets.ChartTypeLine).WidgetID("revenue").
		Data(func(ctx *filament.Context) widgets.ChartData {
			return widgets.ChartData{Labels: []string{"secret-label"}}
		})
	chart.VisibleWhen(func(ctx *filament.Context) bool { return ctx.User().GetAuthIdentifier() == "boss" })
	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: token, Login: token}, nil
		}).Widget(chart)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	get := func(path, user string) (int, string) {
		rec := jobReq(app, "GET", path, user, "", nil)
		return rec.Code, rec.Body.String()
	}
	for _, path := range []string{"/admin/widgets/revenue/data", "/admin/api/widgets/revenue/data"} {
		if code, body := get(path, "clerk"); code != http.StatusNotFound || strings.Contains(body, "secret-label") {
			t.Fatalf("%s for clerk: %d %s", path, code, body)
		}
		if code, body := get(path, "boss"); code != 200 || !strings.Contains(body, "secret-label") {
			t.Fatalf("%s for boss: %d", path, code)
		}
	}
	if _, body := get("/admin/api/widgets", "clerk"); strings.Contains(body, "revenue") {
		t.Fatal("hidden widget listed")
	}
	if _, body := get("/admin", "clerk"); strings.Contains(body, "Revenue secrets") {
		t.Fatal("hidden widget on the dashboard")
	}
	if _, body := get("/admin", "boss"); !strings.Contains(body, "Revenue secrets") {
		t.Fatal("visible widget missing")
	}
}
