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
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type gadget struct {
	ID         string `json:"id"`
	CategoryID string `json:"category_id"`
}

// createOptionApp: gadgets have a category select that can create
// categories (through the categories resource) and a tag select with a
// custom create form.
func createOptionApp(t *testing.T, canCreateCategories *bool) (http.Handler, *[]category, *[]string) {
	t.Helper()
	cats := &[]category{{"1", "Audio"}}
	tags := &[]string{"sale"}
	catOptions := func(context.Context) []forms.Option {
		var out []forms.Option
		for _, c := range *cats {
			out = append(out, forms.Option{Value: c.ID, Label: c.Name})
		}
		return out
	}

	categories := filament.NewResource[*category]("categories").RecordID(func(c *category) string { return c.ID })
	categories.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(forms.TextInput("name").Label("Name").Required())
	})
	categories.CanCreate(func(*filament.Context) bool { return *canCreateCategories })
	categories.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		id := fmt.Sprint(len(*cats) + 1)
		*cats = append(*cats, category{id, fmt.Sprint(data["name"])})
		filament.SetCreatedID(ctx, id)
		return nil
	})

	gadgets := filament.NewResource[*gadget]("gadgets").RecordID(func(g *gadget) string { return g.ID })
	gadgets.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Select("category_id").Label("Category").OptionsUsing(catOptions).CreateOptionFrom("categories"),
			forms.Select("tag").Label("Tag").OptionsUsing(func(context.Context) []forms.Option {
				var out []forms.Option
				for _, s := range *tags {
					out = append(out, forms.Option{Value: s, Label: s})
				}
				return out
			}).
				CreateOptionForm(forms.TextInput("name").Required()).
				CreateOptionUsing(func(ctx context.Context, data map[string]any) (forms.Option, error) {
					name := strings.ToLower(fmt.Sprint(data["name"]))
					*tags = append(*tags, name)
					return forms.Option{Value: name, Label: name}, nil
				}),
			forms.Select("color").Label("Color").Options(map[string]string{"red": "Red"}),
		)
	})
	gadgets.OnCreate(func(*filament.Context, map[string]any) error { return nil })

	app := golaravel.New()
	p := filament.NewPanel("admin").Path("/admin").Engine(filament.EngineFlutter)
	p.Resource(categories)
	p.Resource(gadgets)
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, cats, tags
}

func doJSON(t *testing.T, h http.Handler, method, path, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	out := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func formField(schema any, name string) map[string]any {
	nodes, _ := schema.([]any)
	for _, n := range nodes {
		m, _ := n.(map[string]any)
		if m["name"] == name {
			return m
		}
		if f := formField(m["children"], name); f != nil {
			return f
		}
	}
	return nil
}

func TestSelectCreatesOptionThroughAResource(t *testing.T) {
	allow := true
	app, cats, _ := createOptionApp(t, &allow)

	_, form := doJSON(t, app, "GET", "/admin/api/resources/gadgets/form", "")
	co, _ := formField(form["schema"], "category_id")["create_option"].(map[string]any)
	if co["resource"] != "categories" || co["label"] != "Create category" {
		t.Fatalf("create_option: %v", co)
	}
	if formField(form["schema"], "color")["create_option"] != nil {
		t.Fatal("a plain select offered create")
	}

	code, sub := doJSON(t, app, "GET", "/admin/api/resources/gadgets/form/options/create?field=category_id", "")
	if code != http.StatusOK || formField(sub["schema"], "name") == nil {
		t.Fatalf("create form: %d %v", code, sub)
	}

	code, out := doJSON(t, app, "POST", "/admin/api/resources/gadgets/form/options/create?field=category_id", `{"name":"Wearables"}`)
	opt, _ := out["option"].(map[string]any)
	if code != http.StatusCreated || opt["value"] != "2" || opt["label"] != "Wearables" || len(*cats) != 2 {
		t.Fatalf("create: %d %v cats=%v", code, out, *cats)
	}
	// the new value now passes the select's validation
	if code, out := doJSON(t, app, "POST", "/admin/api/resources/gadgets", `{"category_id":"2"}`); code != http.StatusCreated {
		t.Fatalf("saving with the new option: %d %v", code, out)
	}

	// the target resource's validation applies
	if code, _ := doJSON(t, app, "POST", "/admin/api/resources/gadgets/form/options/create?field=category_id", `{"name":""}`); code != http.StatusUnprocessableEntity && code != http.StatusBadRequest {
		t.Fatalf("empty name: %d", code)
	}
	if len(*cats) != 2 {
		t.Fatal("invalid input created a category")
	}
}

func TestSelectCreateOptionNeedsCreatePermission(t *testing.T) {
	allow := false
	app, cats, _ := createOptionApp(t, &allow)
	_, form := doJSON(t, app, "GET", "/admin/api/resources/gadgets/form", "")
	if formField(form["schema"], "category_id")["create_option"] != nil {
		t.Fatal("create offered without the permission")
	}
	if code, _ := doJSON(t, app, "POST", "/admin/api/resources/gadgets/form/options/create?field=category_id", `{"name":"X"}`); code != http.StatusForbidden {
		t.Fatalf("create without permission: %d", code)
	}
	if code, _ := doJSON(t, app, "GET", "/admin/api/resources/gadgets/form/options/create?field=color", ""); code != http.StatusNotFound {
		t.Fatalf("plain select: %d", code)
	}
	if len(*cats) != 1 {
		t.Fatal("category created without permission")
	}
}

func TestSelectCreatesOptionWithACustomForm(t *testing.T) {
	allow := true
	app, _, tags := createOptionApp(t, &allow)
	code, sub := doJSON(t, app, "GET", "/admin/api/resources/gadgets/form/options/create?field=tag", "")
	if code != http.StatusOK || formField(sub["schema"], "name") == nil {
		t.Fatalf("custom form: %d %v", code, sub)
	}
	if code, _ := doJSON(t, app, "POST", "/admin/api/resources/gadgets/form/options/create?field=tag", `{}`); code != http.StatusUnprocessableEntity {
		t.Fatalf("missing name: %d", code)
	}
	code, out := doJSON(t, app, "POST", "/admin/api/resources/gadgets/form/options/create?field=tag", `{"name":"New"}`)
	if opt, _ := out["option"].(map[string]any); code != http.StatusCreated || opt["value"] != "new" || len(*tags) != 2 {
		t.Fatalf("custom create: %d %v", code, out)
	}
}

func TestCreateAPIReturnsTheNewID(t *testing.T) {
	allow := true
	app, _, _ := createOptionApp(t, &allow)
	code, out := doJSON(t, app, "POST", "/admin/api/resources/categories", `{"name":"Books"}`)
	if code != http.StatusCreated || out["id"] != "2" {
		t.Fatalf("create: %d %v", code, out)
	}
}
