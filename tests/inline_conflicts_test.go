package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type inlineItem struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

func newInlineConflictApp(t *testing.T, canEdit bool) (*golaravel.Engine, *inlineItem, *inlineItem) {
	t.Helper()
	item := &inlineItem{ID: 1, Name: "Lamp", Version: 1}
	child := &inlineItem{ID: 5, Name: "Bulb", Version: 1}

	rm := filament.NewRelationManager[*inlineItem]("parts", "Parts")
	rm.Table(func(tb *tables.Table[*inlineItem]) *tables.Table[*inlineItem] {
		return tb.Columns(tables.TextColumn[*inlineItem]("name"))
	})
	rm.Form(func(f *forms.Form) *forms.Form { return f.Schema(forms.TextInput("name")) })
	rm.RecordID(func(c *inlineItem) string { return fmt.Sprint(c.ID) })
	rm.OnList(func(ctx *filament.Context, parentID, search, sort, dir string, page, perPage int) ([]*inlineItem, int, error) {
		return []*inlineItem{child}, 1, nil
	})
	rm.OnFind(func(ctx *filament.Context, parentID, childID string) (*inlineItem, error) { return child, nil })
	rm.OnUpdate(func(ctx *filament.Context, parentID, childID string, data map[string]any) error {
		child.Name = fmt.Sprint(data["name"])
		child.Version++
		return nil
	})
	rm.Versioned(func(c *inlineItem) string { return fmt.Sprint(c.Version) })

	res := filament.NewResource[*inlineItem]("items")
	res.Table(func(tb *tables.Table[*inlineItem]) *tables.Table[*inlineItem] {
		return tb.Columns(tables.TextInputColumn[*inlineItem]("name"))
	})
	res.RecordID(func(i *inlineItem) string { return fmt.Sprint(i.ID) })
	res.OnList(func(ctx *filament.Context, search, sort, dir string, page, perPage int, filters map[string]string) ([]*inlineItem, int, error) {
		return []*inlineItem{item}, 1, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*inlineItem, error) { return item, nil })
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error { return nil })
	res.OnCellUpdate(func(ctx *filament.Context, id, column string, value any) error {
		item.Name = fmt.Sprint(value)
		item.Version++
		return nil
	})
	res.CanEdit(func(ctx *filament.Context, rec *inlineItem) bool { return canEdit })
	res.Versioned(func(i *inlineItem) string { return fmt.Sprint(i.Version) })
	res.Relations(rm)

	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: "1", Login: "admin"}, nil
		}).Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, item, child
}

func apiSend(app http.Handler, method, path string, body map[string]any) (int, map[string]any) {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestInlineCellEditConflict(t *testing.T) {
	app, item, _ := newInlineConflictApp(t, true)

	// The list's row meta carries each row's version.
	_, list := clientGet(t, app, "example.com", "/admin/api/resources/items", "t")
	meta, _ := list["row_meta"].(map[string]any)
	if row, _ := meta["1"].(map[string]any); row["version"] != "1" {
		t.Fatalf("row meta: %v", list["row_meta"])
	}

	cell := "/admin/api/resources/items/1/cell"
	code, out := apiSend(app, "POST", cell, map[string]any{"column": "name", "value": "A", "_version": "1"})
	if code != 200 || out["version"] != "2" || item.Name != "A" {
		t.Fatalf("first cell save: %d %v", code, out)
	}
	code, out = apiSend(app, "POST", cell, map[string]any{"column": "name", "value": "B", "_version": "1"})
	if code != http.StatusConflict || out["version"] != "2" || item.Name != "A" {
		t.Fatalf("stale cell save: %d %v", code, out)
	}
	// The returned version lets the same editor keep editing.
	if code, _ := apiSend(app, "POST", cell, map[string]any{"column": "name", "value": "B", "_version": "2"}); code != 200 {
		t.Fatalf("next cell save: %d", code)
	}
	if code, _ := apiSend(app, "POST", cell, map[string]any{"column": "name", "value": "C"}); code != 200 {
		t.Fatalf("unversioned cell save: %d", code)
	}
}

func TestInlineCellConflictNeedsEditRights(t *testing.T) {
	app, _, _ := newInlineConflictApp(t, false)
	code, out := apiSend(app, "POST", "/admin/api/resources/items/1/cell", map[string]any{"column": "name", "value": "X", "_version": "0"})
	if code != http.StatusForbidden || out["version"] != nil {
		t.Fatalf("forbidden editor learned the version: %d %v", code, out)
	}
}

func TestRelationEditConflict(t *testing.T) {
	app, _, child := newInlineConflictApp(t, true)
	_, form := clientGet(t, app, "example.com", "/admin/api/resources/items/1/relations/parts/form?child_id=5", "t")
	if form["version"] != "1" {
		t.Fatalf("relation form version: %v", form["version"])
	}
	path := "/admin/api/resources/items/1/relations/parts/5"
	if code, _ := apiSend(app, "PUT", path, map[string]any{"name": "A", "_version": "1"}); code != 200 || child.Name != "A" {
		t.Fatalf("first save: %d", code)
	}
	code, out := apiSend(app, "PUT", path, map[string]any{"name": "B", "_version": "1"})
	if code != http.StatusConflict || out["version"] != "2" || child.Name != "A" {
		t.Fatalf("stale save: %d %v", code, out)
	}
}

func TestWebRowsCarryVersion(t *testing.T) {
	app, _, _ := newInlineConflictApp(t, true)
	req := httptest.NewRequest("GET", "http://example.com/admin/items", nil)
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("list page: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	if !strings.Contains(rec.Body.String(), `data-version="1"`) {
		t.Fatal("rows have no data-version")
	}
}

func TestWebRelationEditConflict(t *testing.T) {
	app, _, child := newInlineConflictApp(t, true)
	get := httptest.NewRequest("GET", "http://example.com/admin/items/1/edit", nil)
	get.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, get)
	if !strings.Contains(rec.Body.String(), `name="_version" value="1"`) {
		t.Fatal("relation edit modal has no version")
	}
	post := func(v string) int {
		req := httptest.NewRequest("POST", "http://example.com/admin/items/1/relations/parts/5/edit", strings.NewReader("name=Z&_version="+v))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer t")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec.Code
	}
	if c := post("1"); c != http.StatusSeeOther || child.Name != "Z" {
		t.Fatalf("save: %d", c)
	}
	if c := post("1"); c != http.StatusConflict {
		t.Fatalf("stale save: %d", c)
	}
}
