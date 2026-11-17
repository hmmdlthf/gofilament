package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type versionedItem struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}

func newVersionedApp(t *testing.T) (*golaravel.Engine, *versionedItem, *[]string) {
	item := &versionedItem{ID: 1, Name: "Lamp", Version: 1}
	var expected []string
	res := filament.NewResource[*versionedItem]("items")
	res.Form(func(f *forms.Form) *forms.Form { return f.Schema(forms.TextInput("name")) })
	res.RecordID(func(i *versionedItem) string { return fmt.Sprint(i.ID) })
	res.OnFind(func(ctx *filament.Context, id string) (*versionedItem, error) { return item, nil })
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		expected = append(expected, filament.ExpectedVersion(ctx))
		item.Name = fmt.Sprint(data["name"])
		item.Version++
		return nil
	})
	res.Versioned(func(i *versionedItem) string { return fmt.Sprint(i.Version) })
	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: "1", Login: "admin"}, nil
		}).Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, item, &expected
}

func apiPut(app http.Handler, body map[string]any) (int, map[string]any) {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest("PUT", "http://example.com/admin/api/resources/items/1", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestAPIEditConflict(t *testing.T) {
	app, item, expected := newVersionedApp(t)
	code, form := clientGet(t, app, "example.com", "/admin/api/resources/items/form?id=1", "t")
	if code != 200 || form["version"] != "1" {
		t.Fatalf("form version: %d %v", code, form["version"])
	}
	if code, _ := apiPut(app, map[string]any{"name": "A", "_version": "1"}); code != 200 || item.Version != 2 {
		t.Fatalf("first save: %d", code)
	}
	// A second editor still holds version 1.
	code, out := apiPut(app, map[string]any{"name": "B", "_version": "1"})
	if code != http.StatusConflict || out["version"] != "2" || item.Name != "A" {
		t.Fatalf("stale save: %d %v name=%s", code, out, item.Name)
	}
	// Overwrite on purpose with the current version.
	if code, _ := apiPut(app, map[string]any{"name": "B", "_version": "2"}); code != 200 || item.Name != "B" {
		t.Fatalf("overwrite: %d", code)
	}
	// Clients that send no version are not checked.
	if code, _ := apiPut(app, map[string]any{"name": "C"}); code != 200 {
		t.Fatalf("no version: %d", code)
	}
	if got := strings.Join(*expected, ","); got != "1,2," {
		t.Fatalf("ExpectedVersion seen by OnUpdate: %q", got)
	}
}

var versionInput = regexp.MustCompile(`name="_version" value="([^"]*)"`)

func TestWebEditConflict(t *testing.T) {
	app, item, _ := newVersionedApp(t)
	get := func() string {
		req := httptest.NewRequest("GET", "http://example.com/admin/items/1/edit", nil)
		req.Header.Set("Authorization", "Bearer t")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec.Body.String()
	}
	post := func(v url.Values) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "http://example.com/admin/items/1/edit", strings.NewReader(v.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer t")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec
	}
	m := versionInput.FindStringSubmatch(get())
	if m == nil || m[1] != "1" {
		t.Fatalf("edit form has no version field")
	}
	item.Version = 5 // someone else saved meanwhile
	rec := post(url.Values{"name": {"Mine"}, "_version": {"1"}})
	body := rec.Body.String()
	if rec.Code != http.StatusConflict || !strings.Contains(body, "Someone else changed this record") || item.Name != "Lamp" {
		t.Fatalf("conflict: %d name=%s", rec.Code, item.Name)
	}
	if m := versionInput.FindStringSubmatch(body); m == nil || m[1] != "5" || !strings.Contains(body, `value="Mine"`) {
		t.Fatal("conflict page must keep the input and carry the current version")
	}
	if rec := post(url.Values{"name": {"Mine"}, "_version": {"5"}}); rec.Code != http.StatusSeeOther || item.Name != "Mine" {
		t.Fatalf("overwrite: %d", rec.Code)
	}
}
