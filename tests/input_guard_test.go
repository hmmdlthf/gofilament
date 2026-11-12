package tests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

type guardItem struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
	Status string `json:"status"`
	Locked bool   `json:"locked"`
}

type guardCalls struct {
	created, updated []map[string]any
	cells            []string
	deleted          [][]string
	relCreated       []map[string]any
}

func newGuardApp(t *testing.T, mutate ...func(*filament.BaseResource[*guardItem])) (*golaravel.Engine, *guardCalls) {
	t.Helper()
	calls := &guardCalls{}
	items := map[string]*guardItem{"1": {ID: 1, Name: "One"}, "2": {ID: 2, Name: "Two", Locked: true}}

	rm := filament.NewRelationManager[*guardItem]("notes", "Notes")
	rm.Form(func(f *forms.Form) *forms.Form { return f.Schema(forms.TextInput("body").Required()) })
	rm.OnCreate(func(ctx *filament.Context, parentID string, data map[string]any) error {
		calls.relCreated = append(calls.relCreated, data)
		return nil
	})

	nickname := forms.TextInput("nickname")
	nickname.Visible(func(get forms.Get) bool { return get("kind") == "person" })

	res := filament.NewResource[*guardItem]("items")
	res.RecordID(func(i *guardItem) string { return fmt.Sprint(i.ID) })
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Section("Main").Schema(
				forms.TextInput("name").Required().MaxLength(10),
				forms.TextInput("email").Email(),
				forms.TextInput("price").Numeric(),
				forms.TextInput("code").Disabled(),
				forms.TextInput("password").RequiredOnCreate(),
				forms.Select("kind").OptionList(forms.Option{Value: "person", Label: "Person"}, forms.Option{Value: "company", Label: "Company"}),
				forms.Select("owner_id").OptionsUsing(func(ctx context.Context) []forms.Option {
					return []forms.Option{{Value: "u1", Label: "User 1"}}
				}),
				nickname,
				forms.DatePicker("due_on"),
				forms.Toggle("active"),
			),
			forms.Repeater("lines").Schema(forms.TextInput("sku").Required()).MinItems(1),
		)
	})
	res.Table(func(tb *tables.Table[*guardItem]) *tables.Table[*guardItem] {
		return tb.Columns(
			tables.TextColumn[*guardItem]("name"),
			tables.ToggleColumn[*guardItem]("active"),
			tables.SelectColumn[*guardItem]("status").Options(map[string]string{"open": "Open", "done": "Done"}),
		)
	})
	res.OnFind(func(ctx *filament.Context, id string) (*guardItem, error) {
		if it, ok := items[id]; ok {
			return it, nil
		}
		return nil, errors.New("database is down")
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		calls.created = append(calls.created, data)
		return nil
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		calls.updated = append(calls.updated, data)
		return nil
	})
	res.OnCellUpdate(func(ctx *filament.Context, id, column string, value any) error {
		calls.cells = append(calls.cells, column)
		return nil
	})
	res.OnDelete(func(ctx *filament.Context, id string) error { return nil })
	res.OnDeleteBulk(func(ctx *filament.Context, ids []string) error {
		calls.deleted = append(calls.deleted, ids)
		return nil
	})
	res.CanEdit(func(ctx *filament.Context, it *guardItem) bool { return !it.Locked })
	res.CanDelete(func(ctx *filament.Context, it *guardItem) bool { return !it.Locked })
	res.Relations(rm)
	for _, m := range mutate {
		m(res)
	}

	p := filament.NewPanel("guard").Path("/admin").Engine(filament.EngineFlutter).Flutter(t.TempDir()).
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			if token == "good" {
				return &filament.PanelUser{ID: "u1", Login: "admin"}, nil
			}
			return nil, fmt.Errorf("bad token")
		}).
		Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, calls
}

func validCreate() map[string]any {
	return map[string]any{
		"name": "Widget", "password": "secret", "kind": "company",
		"lines": []any{map[string]any{"sku": "A", "extra": "x"}},
	}
}

func TestCreateKeepsOnlyFormFields(t *testing.T) {
	app, calls := newGuardApp(t)
	body := validCreate()
	body["role"] = "admin"     // not a form field
	body["code"] = "forged"    // disabled field
	body["nickname"] = "Bobby" // hidden: kind is not person
	code, out := relDo(t, app, "POST", "/admin/api/resources/items", "good", body)
	if code != http.StatusCreated {
		t.Fatalf("create: %d %v", code, out)
	}
	got := calls.created[0]
	for _, k := range []string{"role", "code", "nickname"} {
		if _, ok := got[k]; ok {
			t.Errorf("%s must not reach the create handler: %v", k, got)
		}
	}
	if got["name"] != "Widget" {
		t.Fatalf("form fields are kept: %v", got)
	}
	if line := got["lines"].([]any)[0].(map[string]any); line["extra"] != nil || line["sku"] != "A" {
		t.Fatalf("repeater items keep only their fields: %v", line)
	}
}

func TestCreateValidatesRules(t *testing.T) {
	app, calls := newGuardApp(t)
	cases := []struct {
		field string
		value any
	}{
		{"name", nil},
		{"name", "far too long a name"},
		{"email", "not-an-email"},
		{"price", "abc"},
		{"password", ""},
		{"kind", "alien"},
		{"owner_id", "u9"},
		{"due_on", "31/01/2026"},
		{"active", "maybe"},
		{"lines", []any{}},
	}
	for _, c := range cases {
		body := validCreate()
		if c.value == nil {
			delete(body, c.field)
		} else {
			body[c.field] = c.value
		}
		code, out := relDo(t, app, "POST", "/admin/api/resources/items", "good", body)
		errs, _ := out["errors"].(map[string]any)
		if code != http.StatusUnprocessableEntity || errs[c.field] == nil {
			t.Errorf("%s=%v: expected 422 with an error for the field, got %d %v", c.field, c.value, code, out)
		}
	}
	body := validCreate()
	body["lines"] = []any{map[string]any{"sku": ""}}
	if code, out := relDo(t, app, "POST", "/admin/api/resources/items", "good", body); code != 422 || out["errors"].(map[string]any)["lines.0.sku"] == nil {
		t.Errorf("repeater item rules: %d %v", code, out)
	}
	if len(calls.created) != 0 {
		t.Fatalf("invalid submissions must not reach the handler: %v", calls.created)
	}
}

func TestUpdateIsPartialButKeepsRequiredFields(t *testing.T) {
	app, calls := newGuardApp(t)
	// absent required fields (name, password) are left alone
	if code, out := relDo(t, app, "PUT", "/admin/api/resources/items/1", "good", map[string]any{"email": "a@b.co", "role": "admin"}); code != 200 {
		t.Fatalf("partial update: %d %v", code, out)
	}
	if _, ok := calls.updated[0]["role"]; ok || calls.updated[0]["email"] != "a@b.co" {
		t.Fatalf("update data: %v", calls.updated[0])
	}
	if code, _ := relDo(t, app, "PUT", "/admin/api/resources/items/1", "good", map[string]any{"name": ""}); code != 422 {
		t.Fatalf("emptying a required field: %d", code)
	}
}

func TestWritesNeedAFoundAndAllowedRecord(t *testing.T) {
	app, calls := newGuardApp(t, func(r *filament.BaseResource[*guardItem]) {
		r.CanView(func(ctx *filament.Context, it *guardItem) bool { return it.ID != 3 })
	})
	// lookup errors block the write
	if code, _ := relDo(t, app, "PUT", "/admin/api/resources/items/9", "good", map[string]any{"email": "a@b.co"}); code != 404 {
		t.Fatalf("update of a record that cannot be loaded: %d", code)
	}
	if code, _ := relDo(t, app, "DELETE", "/admin/api/resources/items/9", "good", nil); code != 404 {
		t.Fatalf("delete of a record that cannot be loaded: %d", code)
	}
	if code, _ := relDo(t, app, "PUT", "/admin/api/resources/items/2", "good", map[string]any{"email": "a@b.co"}); code != 403 {
		t.Fatalf("update of a locked record: %d", code)
	}
	if len(calls.updated) != 0 {
		t.Fatalf("refused updates reached the handler: %v", calls.updated)
	}
	// bulk delete checks every record
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/items/bulk", "good", map[string]any{"bulk_action": "delete_bulk", "ids": []string{"1", "2"}}); code != 403 {
		t.Fatalf("bulk delete including a locked record: %d", code)
	}
	if len(calls.deleted) != 0 {
		t.Fatalf("nothing may be deleted when one record is refused: %v", calls.deleted)
	}
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/items/bulk", "good", map[string]any{"bulk_action": "delete_bulk", "ids": []string{"1"}}); code != 200 || len(calls.deleted) != 1 {
		t.Fatalf("allowed bulk delete: %d %v", code, calls.deleted)
	}
}

func TestCellEditsOnlyEditableColumns(t *testing.T) {
	app, calls := newGuardApp(t)
	cell := func(col string, val any) int {
		code, _ := relDo(t, app, "POST", "/admin/api/resources/items/1/cell", "good", map[string]any{"column": col, "value": val})
		return code
	}
	if code := cell("role", "admin"); code != 422 {
		t.Fatalf("a column that is not in the table: %d", code)
	}
	if code := cell("name", "x"); code != 422 {
		t.Fatalf("a read-only text column: %d", code)
	}
	if code := cell("status", "bogus"); code != 422 {
		t.Fatalf("a select column with an unknown option: %d", code)
	}
	if code := cell("active", "maybe"); code != 422 {
		t.Fatalf("a toggle column follows the form's toggle rule: %d", code)
	}
	if code := cell("status", "done"); code != 200 {
		t.Fatalf("a valid select value: %d", code)
	}
	if code := cell("active", true); code != 200 {
		t.Fatalf("a valid toggle: %d", code)
	}
	if fmt.Sprint(calls.cells) != "[status active]" {
		t.Fatalf("only valid edits reach the handler: %v", calls.cells)
	}
}

func TestRelationAndActionFormsAreValidated(t *testing.T) {
	app, calls := newGuardApp(t)
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/items/1/relations/notes", "good", map[string]any{"title": "x"}); code != 422 {
		t.Fatalf("relation create without its required field: %d", code)
	}
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/items/1/relations/notes", "good", map[string]any{"body": "hi", "parent_id": "2"}); code != 201 {
		t.Fatalf("valid relation create: %d", code)
	}
	if _, ok := calls.relCreated[0]["parent_id"]; ok {
		t.Fatalf("relation data keeps only form fields: %v", calls.relCreated[0])
	}

	actions, docs, _ := newActionApp(t)
	if code, out := relDo(t, actions, "POST", "/admin/api/resources/docs/2/actions/discount", "good", map[string]any{}); code != 422 {
		t.Fatalf("action form rules: %d %v", code, out)
	}
	if code, _ := relDo(t, actions, "POST", "/admin/api/resources/docs/2/actions/discount", "good", map[string]any{"percent": "lots"}); code != 422 {
		t.Fatalf("numeric action input: %d", code)
	}
	if docs["2"].Total != 50 {
		t.Fatalf("a refused action must not run: total=%v", docs["2"].Total)
	}
}

func TestAcceptUnvalidatedInput(t *testing.T) {
	app, calls := newGuardApp(t, func(r *filament.BaseResource[*guardItem]) { r.AcceptUnvalidatedInput() })
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/items", "good", map[string]any{"role": "admin"}); code != 201 {
		t.Fatalf("opted-out resource: %d", code)
	}
	if calls.created[0]["role"] != "admin" {
		t.Fatalf("raw data reaches the handler when validation is off: %v", calls.created[0])
	}
}

func newLoginApp(t *testing.T, id string, configure func(*filament.Panel)) *golaravel.Engine {
	t.Helper()
	p := filament.NewPanel(id).Path("/" + id).Login().
		AuthValidator(func(ctx *filament.Context, username, password string) (auth.Authenticatable, string, error) {
			if password == "right" {
				return &filament.PanelUser{ID: "u1", Login: username}, "tok", nil
			}
			return nil, "", errors.New("bad credentials")
		})
	if configure != nil {
		configure(p)
	}
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app
}

func TestLoginThrottle(t *testing.T) {
	id := fmt.Sprintf("thr%d", time.Now().UnixNano())
	app := newLoginApp(t, id, func(p *filament.Panel) { p.LoginThrottle(3, time.Minute) })
	apiLogin := func(user, pass string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/"+id+"/api/login", strings.NewReader(fmt.Sprintf(`{"username":%q,"password":%q}`, user, pass)))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "203.0.113.7:5000"
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec
	}
	for i := 0; i < 3; i++ {
		if rec := apiLogin("ann", "wrong"); rec.Code != 401 {
			t.Fatalf("attempt %d: %d", i+1, rec.Code)
		}
	}
	rec := apiLogin("ann", "right")
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("after 3 failures even the right password waits: %d %q", rec.Code, rec.Header().Get("Retry-After"))
	}
	// another account from the same client is not locked by ann's failures
	if rec := apiLogin("bob", "right"); rec.Code != 200 {
		t.Fatalf("other names are limited separately: %d %s", rec.Code, rec.Body)
	}
	// ...but spraying many names hits the per-client limit (4x)
	for i := 0; i < 12; i++ {
		apiLogin(fmt.Sprintf("user%d", i), "wrong")
	}
	if rec := apiLogin("carol", "right"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("per-client limit: %d", rec.Code)
	}

	// the web login form is limited the same way
	webID := fmt.Sprintf("web%d", time.Now().UnixNano())
	web := newLoginApp(t, webID, func(p *filament.Panel) { p.LoginThrottle(2, time.Minute) })
	webLogin := func(pass string) int {
		get := httptest.NewRequest("GET", "/"+webID+"/login", nil)
		getRec := httptest.NewRecorder()
		web.ServeHTTP(getRec, get)
		form := url.Values{"email": {"ann"}, "password": {pass}}
		var cookies []*http.Cookie
		for _, c := range getRec.Result().Cookies() {
			cookies = append(cookies, c)
			if strings.Contains(strings.ToLower(c.Name), "csrf") || strings.Contains(strings.ToLower(c.Name), "xsrf") {
				form.Set("_token", c.Value)
			}
		}
		req := httptest.NewRequest("POST", "/"+webID+"/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "203.0.113.8:5000"
		for _, c := range cookies {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		web.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := webLogin("wrong"); code != 401 {
		t.Fatalf("web login failure: %d (CSRF token not found?)", code)
	}
	webLogin("wrong")
	if code := webLogin("right"); code != http.StatusTooManyRequests {
		t.Fatalf("web login throttled: %d", code)
	}
}

func TestLoginThrottleCanBeDisabled(t *testing.T) {
	id := fmt.Sprintf("off%d", time.Now().UnixNano())
	app := newLoginApp(t, id, func(p *filament.Panel) { p.LoginThrottle(0, 0) })
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("POST", "/"+id+"/api/login", strings.NewReader(`{"username":"a","password":"wrong"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		if rec.Code != 401 {
			t.Fatalf("attempt %d: %d", i+1, rec.Code)
		}
	}
}

func TestWebFormsAreValidated(t *testing.T) {
	calls := 0
	res := filament.NewResource[*guardItem]("items")
	res.Form(func(f *forms.Form) *forms.Form { return f.Schema(forms.TextInput("name").Required()) })
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		calls++
		if _, ok := data["role"]; ok {
			t.Error("web create received a non-form field")
		}
		return nil
	})
	p := filament.NewPanel("webguard").Path("/admin").Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	post := func(form url.Values) int {
		req := httptest.NewRequest("POST", "/admin/items/create", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := post(url.Values{"role": {"admin"}}); code != 422 || calls != 0 {
		t.Fatalf("missing required field: %d calls=%d", code, calls)
	}
	if code := post(url.Values{"name": {"Ann"}, "role": {"admin"}}); code != http.StatusSeeOther || calls != 1 {
		t.Fatalf("valid create: %d calls=%d", code, calls)
	}
}
