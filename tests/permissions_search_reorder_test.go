package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type secretDoc struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Owner string `json:"owner"`
}

func newPermApp(t *testing.T, viewAny bool) (http.Handler, *[]string) {
	t.Helper()
	docs := []*secretDoc{{1, "Payroll 2026", "hr"}, {2, "Team lunch", "all"}}
	var reordered []string
	res := filament.NewResource[*secretDoc]("docs")
	res.ModelName("Doc").PluralModelName("Docs")
	res.RecordID(func(d *secretDoc) string { return fmt.Sprint(d.ID) })
	res.GloballySearchable().GlobalSearchResultTitle(func(d *secretDoc) string { return d.Title })
	res.Table(func(tb *tables.Table[*secretDoc]) *tables.Table[*secretDoc] {
		return tb.Columns(tables.TextColumn[*secretDoc]("title"))
	})
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*secretDoc, int, error) {
		return docs, len(docs), nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*secretDoc, error) {
		for _, d := range docs {
			if fmt.Sprint(d.ID) == id {
				return d, nil
			}
		}
		return nil, fmt.Errorf("not found")
	})
	res.CanViewAny(func(ctx *filament.Context) bool { return viewAny })
	res.CanView(func(ctx *filament.Context, d *secretDoc) bool { return d.Owner == "all" })
	res.CanEdit(func(ctx *filament.Context, d *secretDoc) bool { return d.Owner == "all" })
	res.OnReorder(func(ctx *filament.Context, order []string) error { reordered = order; return nil })

	app := golaravel.New()
	if err := filament.NewPanel("admin").Path("/admin").GlobalSearch().Resource(res).Register(app); err != nil {
		t.Fatal(err)
	}
	return app, &reordered
}

func searchTitles(t *testing.T, app http.Handler, q string) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/admin/global-search?query="+url.QueryEscape(q), nil)
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	var results []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &results)
	var titles []string
	for _, r := range results {
		titles = append(titles, fmt.Sprint(r["title"]))
	}
	return strings.Join(titles, "|")
}

func TestGlobalSearchRespectsViewPermissions(t *testing.T) {
	app, _ := newPermApp(t, true)
	if got := searchTitles(t, app, "Payroll"); strings.Contains(got, "Payroll") {
		t.Fatalf("search leaked a record the user may not view: %q", got)
	}
	if got := searchTitles(t, app, "lunch"); !strings.Contains(got, "Team lunch") {
		t.Fatalf("visible record missing from search: %q", got)
	}
	app, _ = newPermApp(t, false)
	if got := searchTitles(t, app, "lunch"); strings.Contains(got, "Team lunch") {
		t.Fatalf("search leaked records of a resource the user may not list: %q", got)
	}
}

func TestReorderNeedsEditPermission(t *testing.T) {
	app, reordered := newPermApp(t, true)
	post := func(order string) int {
		ids, _ := json.Marshal(map[string]any{"order": strings.Split(order, ",")})
		req := httptest.NewRequest("POST", "/admin/docs/reorder", strings.NewReader(string(ids)))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := post("2,1"); code != http.StatusForbidden || *reordered != nil {
		t.Fatalf("reorder including a record the user may not edit: %d %v", code, *reordered)
	}
	if code := post("2"); code != http.StatusOK || len(*reordered) != 1 {
		t.Fatalf("reorder of editable records: %d %v", code, *reordered)
	}
}
