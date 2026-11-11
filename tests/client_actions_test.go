package tests

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type actDoc struct {
	ID     int     `json:"id"`
	Number string  `json:"number"`
	Posted bool    `json:"posted"`
	Total  float64 `json:"total"`
}

func newActionApp(t *testing.T) (*golaravel.Engine, map[string]*actDoc, *string) {
	t.Helper()
	docs := map[string]*actDoc{
		"1": {ID: 1, Number: "PUR-1", Total: 100},
		"2": {ID: 2, Number: "PUR-2", Posted: true, Total: 50},
	}
	var lastUser string

	res := filament.NewResource[*actDoc]("docs")
	res.ModelName("Doc").PluralModelName("Docs")
	res.RecordID(func(d *actDoc) string { return fmt.Sprint(d.ID) })
	res.Table(func(tb *tables.Table[*actDoc]) *tables.Table[*actDoc] {
		return tb.
			Columns(
				tables.TextColumn[*actDoc]("number"),
				tables.TextColumn[*actDoc]("total").Money(""),
			).
			Actions(
				tables.EditAction[*actDoc](),
				tables.Action[*actDoc]("post").Label("Post").RequiresConfirmation().
					Visible(func(d *actDoc) bool { return !d.Posted }).
					SuccessMessage("Posted").
					Handle(func(ctx context.Context, d *actDoc, data map[string]any) error {
						if d.Total <= 0 {
							return errors.New("nothing to post")
						}
						if fc := filament.FromContext(ctx); fc != nil && fc.User() != nil {
							lastUser = fc.User().GetAuthIdentifier()
						}
						d.Posted = true
						return nil
					}),
				tables.Action[*actDoc]("discount").Label("Discount").
					Form(func(f *forms.Form) *forms.Form {
						return f.Schema(forms.TextInput("percent").Numeric().Required())
					}).
					Authorize(func(ctx context.Context, d *actDoc) bool {
						fc := filament.FromContext(ctx)
						return fc != nil && fc.Query("deny") == ""
					}).
					Handle(func(ctx context.Context, d *actDoc, data map[string]any) error {
						var pct float64
						fmt.Sscan(fmt.Sprint(data["percent"]), &pct)
						d.Total = d.Total * (100 - pct) / 100
						return nil
					}),
				tables.Action[*actDoc]("pay").Label("Pay").
					Visible(func(d *actDoc) bool { return !d.Posted }).
					URL(func(d *actDoc) string { return "https://pay.example.test/" + d.Number }),
			)
	})
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*actDoc, int, error) {
		return []*actDoc{docs["1"], docs["2"]}, 2, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*actDoc, error) {
		if d, ok := docs[id]; ok {
			return d, nil
		}
		return nil, fmt.Errorf("record not found")
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error { return nil })
	res.CanEdit(func(ctx *filament.Context, d *actDoc) bool { return !d.Posted })

	p := filament.NewPanel("admin").Path("/admin").Engine(filament.EngineFlutter).Flutter(t.TempDir()).
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			if token == "good" {
				return &filament.PanelUser{ID: "u7", Login: "admin"}, nil
			}
			return nil, fmt.Errorf("bad token")
		}).
		CurrencyUsing(func(ctx *filament.Context) string { return "LKR" }).
		Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, docs, &lastUser
}

func TestRowMetaAndActionMetadata(t *testing.T) {
	app, _, _ := newActionApp(t)

	code, list := relDo(t, app, "GET", "/admin/api/resources/docs", "good", nil)
	if code != 200 {
		t.Fatalf("list: %d", code)
	}
	meta := list["row_meta"].(map[string]any)
	open, posted := meta["1"].(map[string]any), meta["2"].(map[string]any)
	if open["update"] != true || posted["update"] != false {
		t.Fatalf("per-row update ability: %v / %v", open, posted)
	}
	if fmt.Sprint(open["actions"]) != "[post discount pay]" || fmt.Sprint(posted["actions"]) != "[discount]" {
		t.Fatalf("per-row actions: %v / %v", open["actions"], posted["actions"])
	}
	if links, _ := open["links"].(map[string]any); links["pay"] != "https://pay.example.test/PUR-1" {
		t.Fatalf("link action URL in row meta: %v", open["links"])
	}
	if _, ok := posted["links"]; ok {
		t.Fatalf("hidden link action must not leak its URL: %v", posted)
	}

	_, schema := relDo(t, app, "GET", "/admin/api/schema", "good", nil)
	if schema["currency"] != "LKR" {
		t.Fatalf("panel currency: %v", schema["currency"])
	}
	actions := schema["resource_meta"].(map[string]any)["docs"].(map[string]any)["table"].(map[string]any)["actions"].([]any)
	post := actions[1].(map[string]any)
	if post["name"] != "post" || post["custom"] != true || post["requires_confirmation"] != true {
		t.Fatalf("custom action metadata: %v", post)
	}
	if actions[2].(map[string]any)["has_form"] != true {
		t.Fatalf("form action metadata: %v", actions[2])
	}
	if pay := actions[3].(map[string]any); pay["link"] != true || pay["custom"] != true {
		t.Fatalf("link action metadata: %v", pay)
	}
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/docs/1/actions/pay", "good", nil); code == 200 {
		t.Fatal("link actions are not run on the server")
	}
	col := schema["resources"].([]any)[0].(map[string]any)["columns"].([]any)[1].(map[string]any)
	if col["type"] != "money" || col["align"] != "right" {
		t.Fatalf("Money(\"\") should be a money column: %v", col)
	}
}

func TestRunningActions(t *testing.T) {
	app, docs, lastUser := newActionApp(t)

	if code, _ := relDo(t, app, "POST", "/admin/api/resources/docs/1/actions/post", "", nil); code != 401 {
		t.Fatalf("actions need auth, got %d", code)
	}
	code, out := relDo(t, app, "POST", "/admin/api/resources/docs/1/actions/post", "good", nil)
	if code != 200 || out["message"] != "Posted" || !docs["1"].Posted {
		t.Fatalf("post: %d %v", code, out)
	}
	if *lastUser != "admin" {
		t.Fatalf("handler should see the signed-in user through the context, got %q", *lastUser)
	}
	// hidden for an already posted record
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/docs/1/actions/post", "good", nil); code != 422 {
		t.Fatalf("posting twice should be refused with 422, got %d", code)
	}
	// unknown action, unknown record
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/docs/1/actions/nope", "good", nil); code != 422 {
		t.Fatalf("unknown action: %d", code)
	}
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/docs/9/actions/discount", "good", nil); code != 404 {
		t.Fatalf("unknown record: %d", code)
	}

	// action with a form
	code, form := relDo(t, app, "GET", "/admin/api/resources/docs/2/actions/discount/form", "good", nil)
	if code != 200 || form["mode"] != "action" || len(form["schema"].([]any)) != 1 {
		t.Fatalf("action form: %d %v", code, form)
	}
	code, _ = relDo(t, app, "POST", "/admin/api/resources/docs/2/actions/discount", "good", map[string]any{"percent": 10})
	if code != 200 || docs["2"].Total != 45 {
		t.Fatalf("discount: %d total=%v", code, docs["2"].Total)
	}
	// authorization callback
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/docs/2/actions/discount?deny=1", "good", map[string]any{"percent": 10}); code != 422 {
		t.Fatalf("unauthorized action should be refused, got %d", code)
	}
}

// Regression: a refused request must not run the handler behind it.
func TestRefusedRequestsDoNotRunHandlers(t *testing.T) {
	app, docs, _ := newActionApp(t)
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/docs/1/actions/post", "", nil); code != 401 {
		t.Fatalf("expected 401, got %d", code)
	}
	if docs["1"].Posted {
		t.Fatal("an unauthenticated request ran the action")
	}
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/docs/1/actions/post", "wrong-token", nil); code != 401 {
		t.Fatalf("expected 401 for a bad token, got %d", code)
	}
	if docs["1"].Posted {
		t.Fatal("a request with a bad token ran the action")
	}
}

func TestRefusedCreateDoesNotCreate(t *testing.T) {
	calls := 0
	res := filament.NewResource[*actDoc]("docs")
	res.ModelName("Doc").PluralModelName("Docs")
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error { calls++; return nil })
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error { calls++; return nil })
	res.OnDelete(func(ctx *filament.Context, id string) error { calls++; return nil })
	res.OnFind(func(ctx *filament.Context, id string) (*actDoc, error) { return &actDoc{ID: 1}, nil })
	p := filament.NewPanel("admin").Path("/admin").Engine(filament.EngineFlutter).Flutter(t.TempDir()).
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return nil, fmt.Errorf("never valid")
		}).
		Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ method, path string }{
		{"POST", "/admin/api/resources/docs"},
		{"PUT", "/admin/api/resources/docs/1"},
		{"DELETE", "/admin/api/resources/docs/1"},
		{"POST", "/admin/api/resources/docs/1/cell"},
		{"POST", "/admin/api/resources/docs/bulk"},
	} {
		if code, _ := relDo(t, app, c.method, c.path, "", map[string]any{"name": "x", "column": "name", "value": "y", "bulk_action": "delete_bulk", "ids": []string{"1"}}); code != 401 {
			t.Errorf("%s %s: expected 401, got %d", c.method, c.path, code)
		}
	}
	if calls != 0 {
		t.Fatalf("handlers ran %d times for unauthenticated requests", calls)
	}
}
