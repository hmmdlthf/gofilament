package tests

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type viewItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func newViewsApp(t *testing.T, canShare bool) *golaravel.Engine {
	t.Helper()
	res := filament.NewResource[*viewItem]("members").PluralModelName("Members")
	res.Table(func(tb *tables.Table[*viewItem]) *tables.Table[*viewItem] {
		return tb.Columns(tables.TextColumn[*viewItem]("name"))
	})
	res.OnList(func(ctx *filament.Context, search, sort, dir string, page, perPage int, filters map[string]string) ([]*viewItem, int, error) {
		return nil, 0, nil
	})
	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			if token == "" {
				return nil, errors.New("no token")
			}
			return &filament.PanelUser{ID: token, Login: token}, nil
		}).
		SavedViews(nil).
		SavedViewsShareable(func(ctx *filament.Context) bool { return canShare },
			func(ctx *filament.Context) bool { return ctx.User().GetAuthIdentifier() == "boss" }).
		Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app
}

func viewsCall(app http.Handler, method, path, user string, body any) (int, map[string]any) {
	var rdr *strings.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rdr = strings.NewReader(string(raw))
	} else {
		rdr = strings.NewReader("")
	}
	req := httptest.NewRequest(method, "http://example.com/admin/api/resources/members"+path, rdr)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestSavedViewsAPI(t *testing.T) {
	app := newViewsApp(t, true)
	state := map[string]any{"search": "ann", "sort": "name", "direction": "desc", "filters": map[string]string{"status": "active"}, "columns": []string{"name"}, "per_page": 50}
	code, v := viewsCall(app, "POST", "/views", "ann", map[string]any{"name": "My actives", "state": state})
	if code != http.StatusCreated || v["mine"] != true || v["id"] == "" {
		t.Fatalf("create: %d %v", code, v)
	}
	if code, _ := viewsCall(app, "POST", "/views", "ann", map[string]any{"name": "my ACTIVES", "state": state}); code != http.StatusUnprocessableEntity {
		t.Fatalf("duplicate name: %d", code)
	}
	if code, _ := viewsCall(app, "POST", "/views", "ann", map[string]any{"name": " "}); code != http.StatusUnprocessableEntity {
		t.Fatalf("empty name: %d", code)
	}
	code, shared := viewsCall(app, "POST", "/views", "ann", map[string]any{"name": "Team", "shared": true, "state": map[string]any{"group": "status"}})
	if code != http.StatusCreated {
		t.Fatalf("shared: %d", code)
	}

	// Bob sees only the shared view and cannot delete it; the boss can.
	_, list := viewsCall(app, "GET", "/views", "bob", nil)
	views := list["views"].([]any)
	if len(views) != 1 || views[0].(map[string]any)["name"] != "Team" || views[0].(map[string]any)["mine"] != false {
		t.Fatalf("bob's views: %v", list)
	}
	if code, _ := viewsCall(app, "DELETE", "/views/"+shared["id"].(string), "bob", nil); code != http.StatusForbidden {
		t.Fatalf("bob deleted ann's view: %d", code)
	}
	if code, _ := viewsCall(app, "DELETE", "/views/"+v["id"].(string), "boss", nil); code != http.StatusForbidden {
		t.Fatalf("boss deleted a private view: %d", code)
	}
	if code, _ := viewsCall(app, "DELETE", "/views/"+shared["id"].(string), "boss", nil); code != http.StatusOK {
		t.Fatalf("boss delete shared: %d", code)
	}
	_, list = viewsCall(app, "GET", "/views", "ann", nil)
	got := list["views"].([]any)[0].(map[string]any)
	st := got["state"].(map[string]any)
	if len(list["views"].([]any)) != 1 || st["search"] != "ann" || st["per_page"] != float64(50) || st["filters"].(map[string]any)["status"] != "active" {
		t.Fatalf("ann's views: %v", list)
	}
	if code, _ := viewsCall(app, "DELETE", "/views/"+got["id"].(string), "ann", nil); code != http.StatusOK {
		t.Fatalf("owner delete: %d", code)
	}
}

func TestSavedViewsSharingCanBeRestricted(t *testing.T) {
	app := newViewsApp(t, false)
	if code, _ := viewsCall(app, "POST", "/views", "ann", map[string]any{"name": "Team", "shared": true}); code != http.StatusForbidden {
		t.Fatalf("shared without permission: %d", code)
	}
	_, list := viewsCall(app, "GET", "/views", "ann", nil)
	if list["can_share"] != false {
		t.Fatalf("can_share: %v", list)
	}
}

var viewLinkRe = regexp.MustCompile(`href="/admin/members\?([^"]*view=view_[0-9a-f]+[^"]*)"`)

func TestSavedViewsWeb(t *testing.T) {
	app := newViewsApp(t, true)
	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", "http://example.com"+path, nil)
		req.Header.Set("Authorization", "Bearer ann")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec
	}
	first := get("/admin/members?search=bo&filter_status=active&sort=name&direction=desc")
	page := first.Body.String()
	if !strings.Contains(page, "Save current view") {
		t.Fatalf("no views menu:\n%s", page)
	}
	m := regexp.MustCompile(`name="_token" value="([^"]+)"`).FindStringSubmatch(page)
	if m == nil {
		t.Fatal("no CSRF token in the views form")
	}
	form := url.Values{"_token": {m[1]}, "name": {"Bo actives"}, "state": {"search=bo&filter_status=active&sort=name&direction=desc"}}
	req := httptest.NewRequest("POST", "http://example.com/admin/members/views", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer ann")
	for _, c := range first.Result().Cookies() {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "filter_status=active") {
		t.Fatalf("save: %d %s %s", rec.Code, rec.Header().Get("Location"), rec.Body.String())
	}
	page = get(rec.Header().Get("Location")).Body.String()
	if !strings.Contains(page, "View: Bo actives") || viewLinkRe.FindString(page) == "" {
		t.Fatalf("saved view not listed:\n%s", page)
	}
}
