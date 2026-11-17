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
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type bulkTask struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

func newBulkApp(t *testing.T) (*golaravel.Engine, map[string]*bulkTask) {
	t.Helper()
	db := map[string]*bulkTask{"1": {1, "A", "open"}, "2": {2, "B", "open"}, "3": {3, "Locked", "open"}}
	res := filament.NewResource[*bulkTask]("tasks").PluralModelName("Tasks")
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.TextInput("title").Required(),
			forms.Select("status").Options(map[string]string{"open": "Open", "done": "Done"}).Required(),
		)
	})
	res.Table(func(tb *tables.Table[*bulkTask]) *tables.Table[*bulkTask] {
		return tb.Columns(tables.TextColumn[*bulkTask]("title")).BulkActions(tables.ExportBulkAction())
	})
	res.RecordID(func(x *bulkTask) string { return fmt.Sprint(x.ID) })
	res.OnList(func(ctx *filament.Context, s, so, d string, p, pp int, f map[string]string) ([]*bulkTask, int, error) {
		return []*bulkTask{db["1"], db["2"], db["3"]}, 3, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*bulkTask, error) {
		if x, ok := db[id]; ok {
			return x, nil
		}
		return nil, fmt.Errorf("not found")
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		if len(data) != 1 {
			return fmt.Errorf("expected one field, got %v", data)
		}
		db[id].Status = fmt.Sprint(data["status"])
		return nil
	})
	res.CanEdit(func(ctx *filament.Context, x *bulkTask) bool { return x.Title != "Locked" })
	res.CanDeleteAny(func(ctx *filament.Context) bool { return false })
	res.BulkEditable("status", "nope")
	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: "1", Login: "a"}, nil
		}).Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, db
}

func bulkReq(app http.Handler, method, path, ct, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(body))
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func TestBulkEditAPI(t *testing.T) {
	app, db := newBulkApp(t)
	call := func(body map[string]any) (int, map[string]any) {
		raw, _ := json.Marshal(body)
		rec := bulkReq(app, "POST", "/admin/api/resources/tasks/bulk-update", "application/json", string(raw))
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}
	if code, _ := call(map[string]any{"ids": []string{"1"}, "field": "status", "value": "bogus"}); code != http.StatusUnprocessableEntity || db["1"].Status != "open" {
		t.Fatalf("invalid value: %d", code)
	}
	if code, _ := call(map[string]any{"ids": []string{"1"}, "field": "title", "value": "x"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("field not bulk-editable: %d", code)
	}
	code, out := call(map[string]any{"ids": []string{"1", "2", "3"}, "field": "status", "value": "done"})
	if code != 200 || out["updated"] != float64(2) || out["failed"] != float64(1) || db["3"].Status != "open" || db["2"].Status != "done" {
		t.Fatalf("bulk: %d %v", code, out)
	}
	_, schema := clientGet(t, app, "example.com", "/admin/api/schema", "t")
	meta := schema["resource_meta"].(map[string]any)["tasks"].(map[string]any)
	if fmt.Sprint(meta["bulk_edit"]) != "[status]" {
		t.Fatalf("bulk_edit meta: %v", meta["bulk_edit"])
	}
}

func TestBulkEditWeb(t *testing.T) {
	app, db := newBulkApp(t)
	if page := bulkReq(app, "GET", "/admin/tasks", "", "").Body.String(); !strings.Contains(page, `value="edit_bulk"`) {
		t.Fatal("no Change Selected bulk action")
	}
	form := "bulk_action=edit_bulk&selected_ids[]=1&selected_ids[]=2"
	rec := bulkReq(app, "POST", "/admin/tasks/bulk", "application/x-www-form-urlencoded", form)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `data-bulk-field="status"`) || strings.Contains(rec.Body.String(), `data-bulk-field="nope"`) {
		t.Fatalf("field choice (no delete rights must not block it): %d", rec.Code)
	}
	v := url.Values{"selected_ids[]": {"1", "2"}, "field": {"status"}}
	if rec := bulkReq(app, "POST", "/admin/tasks/bulk-edit", "application/x-www-form-urlencoded", v.Encode()); rec.Code != 200 || !strings.Contains(rec.Body.String(), "Apply to 2 records") {
		t.Fatalf("input page: %d", rec.Code)
	}
	v.Set("apply", "1")
	v.Set("status", "done")
	rec = bulkReq(app, "POST", "/admin/tasks/bulk-edit", "application/x-www-form-urlencoded", v.Encode())
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "2 records changed.") || db["1"].Status != "done" {
		t.Fatalf("apply: %d %s", rec.Code, rec.Body.String())
	}
	// Bulk delete still needs the delete permission.
	if rec := bulkReq(app, "POST", "/admin/tasks/bulk", "application/x-www-form-urlencoded", "bulk_action=delete_bulk&selected_ids[]=1"); rec.Code != http.StatusForbidden {
		t.Fatalf("bulk delete without rights: %d", rec.Code)
	}
}
