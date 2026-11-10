package tests

import (
	"context"
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

type clientItem struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	CategoryID int    `json:"category_id"`
	SupplierID int    `json:"supplier_id"`
	Status     string `json:"status"`
	Active     bool   `json:"is_active"`
}

func newClientAPIApp(t *testing.T, canCreate bool) *golaravel.Engine {
	t.Helper()
	items := map[string]*clientItem{"7": {ID: 7, Name: "Gold Chain", CategoryID: 2, SupplierID: 40, Status: "live", Active: true}}

	res := filament.NewResource[*clientItem]("items")
	res.ModelName("Item").PluralModelName("Items").NavigationGroup("Catalogue")
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Section("Item").Description("Basics").Schema(
				forms.Grid(2).Schema(
					forms.TextInput("name").Label("Name").Required().MaxLength(80).Placeholder("Item name"),
					forms.TextInput("price").Numeric().Prefix("Rs.").Default(0),
					forms.Select("category_id").Label("Category").OptionsUsing(func(ctx context.Context) []forms.Option {
						return []forms.Option{{Value: "2", Label: "Chains"}, {Value: "1", Label: "Rings"}}
					}),
					forms.Select("supplier_id").Label("Supplier").SearchUsing(
						func(ctx context.Context, q string, limit int) []forms.Option {
							all := []forms.Option{{Value: "40", Label: "Agasthi Traders"}, {Value: "41", Label: "Bright Metals"}}
							var out []forms.Option
							for _, o := range all {
								if q == "" || strings.Contains(strings.ToLower(o.Label), strings.ToLower(q)) {
									out = append(out, o)
								}
							}
							return out
						},
						func(ctx context.Context, v string) string {
							if v == "40" {
								return "Agasthi Traders"
							}
							return ""
						}),
					forms.Select("status").OptionList(
						forms.Option{Value: "live", Label: "Live"},
						forms.Option{Value: "draft", Label: "Draft"},
						forms.Option{Value: "archived", Label: "Archived"},
					),
				),
				forms.Toggle("is_active").Label("Active"),
			),
		)
	})
	res.Table(func(tb *tables.Table[*clientItem]) *tables.Table[*clientItem] {
		return tb.
			Columns(
				tables.TextColumn[*clientItem]("name").Sortable().Searchable().Weight("bold"),
				tables.TextColumn[*clientItem]("price").Money("LKR"),
				tables.BadgeColumn[*clientItem]("status").Colors(map[string]string{"live": "success", "draft": "warning"}),
				tables.ToggleColumn[*clientItem]("is_active"),
			).
			Actions(tables.EditAction[*clientItem](), tables.DeleteAction[*clientItem]()).
			BulkActions(tables.DeleteBulkAction())
	})
	res.RecordID(func(i *clientItem) string { return fmt.Sprint(i.ID) })
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*clientItem, int, error) {
		return []*clientItem{items["7"]}, 1, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*clientItem, error) {
		if it, ok := items[id]; ok {
			return it, nil
		}
		return nil, fmt.Errorf("not found")
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error { return nil })
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error { return nil })
	res.OnDelete(func(ctx *filament.Context, id string) error { return nil })
	res.CanCreate(func(ctx *filament.Context) bool { return canCreate })
	res.CanDelete(func(ctx *filament.Context, it *clientItem) bool { return false })

	p := filament.NewPanel("admin").Path("/admin").BrandName("Static Brand").
		Engine(filament.EngineFlutter).Flutter(t.TempDir()).
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			if token == "good" {
				return &filament.PanelUser{ID: "1", Login: "admin", Name: "Admin"}, nil
			}
			return nil, fmt.Errorf("bad token")
		}).
		BrandUsing(func(ctx *filament.Context) *filament.Brand {
			if strings.HasPrefix(ctx.Request.Host, "acme.") {
				return &filament.Brand{Name: "Acme Stores", PrimaryColor: "#7C3AED", LogoURL: "/acme.png", HidePoweredBy: true}
			}
			return nil
		}).
		Resource(res)

	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app
}

func clientGet(t *testing.T, app http.Handler, host, path, token string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "http://"+host+path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestClientBootIsPublicAndBrandedPerRequest(t *testing.T) {
	app := newClientAPIApp(t, true)

	code, boot := clientGet(t, app, "example.com", "/admin/api/boot", "")
	if code != 200 || boot["brand_name"] != "Static Brand" {
		t.Fatalf("boot: %d %v", code, boot)
	}
	if _, leaked := boot["resources"]; leaked {
		t.Fatal("boot must not expose resources")
	}

	code, boot = clientGet(t, app, "acme.example.com", "/admin/api/boot", "")
	brand := boot["brand"].(map[string]any)
	if code != 200 || brand["name"] != "Acme Stores" || brand["hide_powered_by"] != true {
		t.Fatalf("per-host brand not applied: %v", boot)
	}
	primary := boot["theme"].(map[string]any)["colors"].(map[string]any)["primary"].(map[string]any)
	if len(primary) != 11 || primary["500"] == "" {
		t.Fatalf("brand primary colour should become an 11 shade palette, got %v", primary)
	}
}

func TestClientSchemaCarriesResourceMeta(t *testing.T) {
	app := newClientAPIApp(t, false)

	if code, _ := clientGet(t, app, "acme.example.com", "/admin/api/schema", ""); code != 401 {
		t.Fatalf("schema must require auth, got %d", code)
	}
	code, schema := clientGet(t, app, "acme.example.com", "/admin/api/schema", "good")
	if code != 200 || schema["brand_name"] != "Acme Stores" {
		t.Fatalf("schema: %d %v", code, schema["brand_name"])
	}
	meta := schema["resource_meta"].(map[string]any)["items"].(map[string]any)
	abilities := meta["abilities"].(map[string]any)
	if abilities["create"] != false || abilities["delete_any"] != true {
		t.Fatalf("unexpected abilities: %v", abilities)
	}
	table := meta["table"].(map[string]any)
	if len(table["actions"].([]any)) != 2 || len(table["bulk_actions"].([]any)) != 1 {
		t.Fatalf("table actions missing: %v", table)
	}

	first := schema["resources"].([]any)[0].(map[string]any)
	if first["model_name"] != "Item" || first["plural_model_name"] != "Items" {
		t.Errorf("model names: %v / %v", first["model_name"], first["plural_model_name"])
	}
	export := schema["export"].(map[string]any)
	if export["max_rows"] != float64(5000) || export["enabled"] != true {
		t.Errorf("default export limits: %v", export)
	}
	cols := schema["resources"].([]any)[0].(map[string]any)["columns"].([]any)
	byName := map[string]map[string]any{}
	for _, c := range cols {
		m := c.(map[string]any)
		byName[m["name"].(string)] = m
	}
	if byName["price"]["type"] != "money" || byName["price"]["currency"] != "LKR" {
		t.Errorf("money column metadata missing: %v", byName["price"])
	}
	if byName["status"]["type"] != "badge" || byName["status"]["colors"].(map[string]any)["live"] != "success" {
		t.Errorf("badge column metadata missing: %v", byName["status"])
	}
	if byName["is_active"]["type"] != "toggle" || byName["is_active"]["editable"] != true {
		t.Errorf("toggle column metadata missing: %v", byName["is_active"])
	}
	if byName["name"]["weight"] != "bold" {
		t.Errorf("weight missing: %v", byName["name"])
	}
}

func findNode(nodes []any, name string) map[string]any {
	for _, n := range nodes {
		m := n.(map[string]any)
		if m["name"] == name {
			return m
		}
		if kids, ok := m["children"].([]any); ok {
			if found := findNode(kids, name); found != nil {
				return found
			}
		}
	}
	return nil
}

func TestClientFormSchema(t *testing.T) {
	app := newClientAPIApp(t, true)

	code, form := clientGet(t, app, "example.com", "/admin/api/resources/items/form", "good")
	if code != 200 || form["mode"] != "create" {
		t.Fatalf("create form: %d %v", code, form)
	}
	nodes := form["schema"].([]any)
	section := nodes[0].(map[string]any)
	if section["kind"] != "section" || section["heading"] != "Item" || section["description"] != "Basics" {
		t.Fatalf("section not described: %v", section)
	}
	grid := section["children"].([]any)[0].(map[string]any)
	if grid["kind"] != "grid" || grid["columns"] != float64(2) {
		t.Fatalf("grid not described: %v", grid)
	}

	name := findNode(nodes, "name")
	if name["type"] != "text" || name["required"] != true || name["max_length"] != float64(80) || name["placeholder"] != "Item name" {
		t.Errorf("text field: %v", name)
	}
	price := findNode(nodes, "price")
	if price["type"] != "number" || price["prefix"] != "Rs." {
		t.Errorf("numeric field: %v", price)
	}
	// request-time options, in the order the loader returned them
	cat := findNode(nodes, "category_id")
	opts := cat["options"].([]any)
	if len(opts) != 2 || opts[0].(map[string]any)["label"] != "Chains" {
		t.Errorf("dynamic options: %v", cat)
	}
	// explicit order is kept
	status := findNode(nodes, "status")["options"].([]any)
	if status[0].(map[string]any)["value"] != "live" || status[2].(map[string]any)["value"] != "archived" {
		t.Errorf("ordered options: %v", status)
	}
	// remote selects send no options
	sup := findNode(nodes, "supplier_id")
	if sup["remote"] != true || sup["options"] != nil {
		t.Errorf("remote select: %v", sup)
	}
	if findNode(nodes, "is_active")["type"] != "toggle" {
		t.Errorf("toggle field missing")
	}

	// edit: record, abilities and the label of the remote value
	code, form = clientGet(t, app, "example.com", "/admin/api/resources/items/form?id=7", "good")
	if code != 200 || form["mode"] != "edit" {
		t.Fatalf("edit form: %d %v", code, form)
	}
	if form["state"].(map[string]any)["name"] != "Gold Chain" {
		t.Errorf("edit state: %v", form["state"])
	}
	ab := form["abilities"].(map[string]any)
	if ab["update"] != true || ab["delete"] != false {
		t.Errorf("record abilities: %v", ab)
	}
	if form["option_labels"].(map[string]any)["supplier_id"] != "Agasthi Traders" {
		t.Errorf("remote label: %v", form["option_labels"])
	}

	if code, _ := clientGet(t, app, "example.com", "/admin/api/resources/items/form?id=999", "good"); code != 404 {
		t.Errorf("missing record should be 404, got %d", code)
	}
	if code, _ := clientGet(t, app, "example.com", "/admin/api/resources/items/form", ""); code != 401 {
		t.Errorf("form must require auth, got %d", code)
	}
}

func TestClientFormDeniedWithoutCreate(t *testing.T) {
	app := newClientAPIApp(t, false)
	if code, _ := clientGet(t, app, "example.com", "/admin/api/resources/items/form", "good"); code != 403 {
		t.Fatalf("create form without permission should be 403, got %d", code)
	}
}

func TestClientFormOptionSearch(t *testing.T) {
	app := newClientAPIApp(t, true)

	code, out := clientGet(t, app, "example.com", "/admin/api/resources/items/form/options?field=supplier_id&q=bright", "good")
	opts := out["options"].([]any)
	if code != 200 || len(opts) != 1 || opts[0].(map[string]any)["value"] != "41" {
		t.Fatalf("remote search: %d %v", code, out)
	}
	// local selects are searchable through the same endpoint
	_, out = clientGet(t, app, "example.com", "/admin/api/resources/items/form/options?field=category_id&q=ring", "good")
	if opts := out["options"].([]any); len(opts) != 1 || opts[0].(map[string]any)["label"] != "Rings" {
		t.Fatalf("local search: %v", out)
	}
	if code, _ := clientGet(t, app, "example.com", "/admin/api/resources/items/form/options?field=name", "good"); code != 404 {
		t.Errorf("non-select field should be 404, got %d", code)
	}
	if code, _ := clientGet(t, app, "example.com", "/admin/api/resources/items/form/options?field=supplier_id", ""); code != 401 {
		t.Errorf("options must require auth, got %d", code)
	}
}
