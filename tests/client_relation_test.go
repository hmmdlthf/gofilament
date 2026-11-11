package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type relParent struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type relChild struct {
	ID       int     `json:"id"`
	ParentID int     `json:"parent_id"`
	SKU      string  `json:"sku"`
	Price    float64 `json:"price"`
}

func newRelationApp(t *testing.T, canEditParent bool) (*golaravel.Engine, *[]*relChild) {
	t.Helper()
	children := []*relChild{{ID: 1, ParentID: 7, SKU: "A-1", Price: 10}, {ID: 2, ParentID: 8, SKU: "B-1", Price: 20}}

	rm := filament.NewRelationManager[*relChild]("variants", "Variants")
	rm.Table(func(tb *tables.Table[*relChild]) *tables.Table[*relChild] {
		return tb.Columns(
			tables.TextColumn[*relChild]("sku").Label("SKU").Sortable(),
			tables.TextColumn[*relChild]("price").Money("LKR"),
		).Actions(tables.EditAction[*relChild](), tables.DeleteAction[*relChild]())
	})
	rm.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(forms.Grid(2).Schema(
			forms.TextInput("sku").Label("SKU").Required(),
			forms.TextInput("price").Numeric(),
		))
	})
	rm.RecordID(func(c *relChild) string { return fmt.Sprint(c.ID) })
	rm.OnList(func(ctx *filament.Context, parentID string, search, sort, dir string, page, perPage int) ([]*relChild, int, error) {
		var out []*relChild
		for _, c := range children {
			if fmt.Sprint(c.ParentID) == parentID {
				out = append(out, c)
			}
		}
		return out, len(out), nil
	})
	rm.OnFind(func(ctx *filament.Context, parentID, childID string) (*relChild, error) {
		for _, c := range children {
			if fmt.Sprint(c.ID) == childID && fmt.Sprint(c.ParentID) == parentID {
				return c, nil
			}
		}
		return nil, fmt.Errorf("record not found")
	})
	rm.OnCreate(func(ctx *filament.Context, parentID string, data map[string]any) error {
		children = append(children, &relChild{ID: len(children) + 1, ParentID: 7, SKU: fmt.Sprint(data["sku"])})
		return nil
	})
	rm.OnUpdate(func(ctx *filament.Context, parentID, childID string, data map[string]any) error { return nil })
	rm.OnDelete(func(ctx *filament.Context, parentID, childID string) error { return nil })

	res := filament.NewResource[*relParent]("parents")
	res.ModelName("Parent").PluralModelName("Parents")
	res.Form(func(f *forms.Form) *forms.Form { return f.Schema(forms.TextInput("name")) })
	res.Table(func(tb *tables.Table[*relParent]) *tables.Table[*relParent] {
		return tb.Columns(tables.TextColumn[*relParent]("name"))
	})
	res.OnFind(func(ctx *filament.Context, id string) (*relParent, error) {
		if id == "7" || id == "8" {
			return &relParent{ID: 7, Name: "Parent"}, nil
		}
		return nil, fmt.Errorf("record not found")
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error { return nil })
	res.CanEdit(func(ctx *filament.Context, p *relParent) bool { return canEditParent })
	res.CanView(func(ctx *filament.Context, p *relParent) bool { return ctx.Query("deny_view") == "" })
	res.Relations(rm)

	p := filament.NewPanel("admin").Path("/admin").Engine(filament.EngineFlutter).Flutter(t.TempDir()).
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			if token == "good" {
				return &filament.PanelUser{ID: "1", Login: "admin"}, nil
			}
			return nil, fmt.Errorf("bad token")
		}).
		Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, &children
}

func relDo(t *testing.T, app http.Handler, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, "http://example.com"+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestRelationSchemaAndForm(t *testing.T) {
	app, _ := newRelationApp(t, true)
	const base = "/admin/api/resources/parents/7/relations"

	code, meta := relDo(t, app, "GET", base, "good", nil)
	if code != 200 {
		t.Fatalf("relations meta: %d %v", code, meta)
	}
	rel := meta["relations"].([]any)[0].(map[string]any)
	if rel["relationship"] != "variants" || rel["has_form"] != true {
		t.Fatalf("relation meta: %v", rel)
	}
	cols := rel["columns"].([]any)
	if len(cols) != 2 || cols[1].(map[string]any)["type"] != "money" {
		t.Fatalf("relation columns: %v", cols)
	}
	if rel["abilities"].(map[string]any)["update"] != true {
		t.Fatalf("relation abilities: %v", rel["abilities"])
	}

	code, list := relDo(t, app, "GET", base+"/variants", "good", nil)
	if code != 200 || list["total"] != float64(1) {
		t.Fatalf("relation list should hold only this parent's rows: %d %v", code, list)
	}

	code, form := relDo(t, app, "GET", base+"/variants/form", "good", nil)
	if code != 200 || form["mode"] != "create" {
		t.Fatalf("create form: %d %v", code, form)
	}
	grid := form["schema"].([]any)[0].(map[string]any)
	if grid["kind"] != "grid" || grid["children"].([]any)[0].(map[string]any)["name"] != "sku" {
		t.Fatalf("relation form schema: %v", grid)
	}

	code, form = relDo(t, app, "GET", base+"/variants/form?child_id=1", "good", nil)
	if code != 200 || form["state"].(map[string]any)["sku"] != "A-1" {
		t.Fatalf("edit form: %d %v", code, form)
	}
	// a child of another parent is not reachable through this parent
	if code, _ := relDo(t, app, "GET", base+"/variants/form?child_id=2", "good", nil); code != 404 {
		t.Fatalf("foreign child should be 404, got %d", code)
	}
}

func TestRelationEndpointsRespectParentPermissions(t *testing.T) {
	app, children := newRelationApp(t, false)
	const base = "/admin/api/resources/parents/7/relations"

	if code, _ := relDo(t, app, "GET", base+"/variants", "", nil); code != 401 {
		t.Fatalf("relation list must require auth, got %d", code)
	}
	// reading is allowed, writing needs edit rights on the parent
	if code, _ := relDo(t, app, "GET", base+"/variants", "good", nil); code != 200 {
		t.Fatalf("read with view rights: %d", code)
	}
	before := len(*children)
	for _, c := range []struct{ method, path string }{
		{"POST", base + "/variants"},
		{"PUT", base + "/variants/1"},
		{"DELETE", base + "/variants/1"},
		{"GET", base + "/variants/form"},
	} {
		if code, _ := relDo(t, app, c.method, c.path, "good", map[string]any{"sku": "X"}); code != 403 {
			t.Errorf("%s %s without edit rights on the parent should be 403, got %d", c.method, c.path, code)
		}
	}
	if len(*children) != before {
		t.Fatal("a row was created despite missing permission")
	}
	// no view right on the parent: nothing of the relation is readable
	if code, _ := relDo(t, app, "GET", base+"/variants?deny_view=1", "good", nil); code != 403 {
		t.Errorf("list without view rights should be 403, got %d", code)
	}
	// unknown parent
	if code, _ := relDo(t, app, "GET", "/admin/api/resources/parents/999/relations/variants", "good", nil); code != 404 {
		t.Errorf("unknown parent should be 404, got %d", code)
	}
}
