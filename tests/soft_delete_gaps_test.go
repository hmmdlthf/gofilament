package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func newTrashApp(t *testing.T, soft, canDelete bool, forceErr error) (*golaravel.Engine, map[string]*SoftDeleteDocument) {
	t.Helper()
	now := time.Now()
	db := map[string]*SoftDeleteDocument{"1": {ID: "1", Title: "Live"}, "2": {ID: "2", Title: "Gone", DeletedAt: &now}}
	res := filament.NewResource[*SoftDeleteDocument]("docs").ModelName("Doc").
		SoftDeletes(soft).
		IsTrashed(func(d *SoftDeleteDocument) bool { return d.DeletedAt != nil }).
		Form(func(f *forms.Form) *forms.Form { return f.Schema(forms.TextInput("title")) }).
		Table(func(tb *tables.Table[*SoftDeleteDocument]) *tables.Table[*SoftDeleteDocument] {
			return tb.Columns(tables.TextColumn[*SoftDeleteDocument]("title"))
		}).
		RecordID(func(d *SoftDeleteDocument) string { return d.ID }).
		OnList(func(ctx *filament.Context, s, so, d string, p, pp int, f map[string]string) ([]*SoftDeleteDocument, int, error) {
			return nil, 0, nil
		}).
		OnFind(func(ctx *filament.Context, id string) (*SoftDeleteDocument, error) {
			if d, ok := db[id]; ok {
				return d, nil
			}
			return nil, errors.New("not found")
		}).
		OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
			db[id].Title = fmt.Sprint(data["title"])
			return nil
		}).
		OnRestore(func(ctx *filament.Context, id string) error { db[id].DeletedAt = nil; return nil }).
		OnForceDelete(func(ctx *filament.Context, id string) error {
			if forceErr != nil {
				return forceErr
			}
			delete(db, id)
			return nil
		}).
		CanDeleteAny(func(ctx *filament.Context) bool { return canDelete })
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

func trashCall(app http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func TestTrashedRecordsCannotBeChanged(t *testing.T) {
	app, db := newTrashApp(t, true, true, nil)
	if rec := trashCall(app, "PUT", "/admin/api/resources/docs/2", map[string]any{"title": "X"}); rec.Code != http.StatusConflict || db["2"].Title != "Gone" {
		t.Fatalf("update trashed: %d", rec.Code)
	}
	if rec := trashCall(app, "GET", "/admin/docs/2/edit", nil); rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "Restore") {
		t.Fatalf("edit page: %d", rec.Code)
	}
	if rec := trashCall(app, "PUT", "/admin/api/resources/docs/1", map[string]any{"title": "Y"}); rec.Code != 200 {
		t.Fatalf("update live: %d", rec.Code)
	}
	if rec := trashCall(app, "POST", "/admin/api/resources/docs/restore", map[string]any{"id": "2"}); rec.Code != 200 || db["2"].DeletedAt != nil {
		t.Fatalf("restore: %d", rec.Code)
	}
}

func TestTrashedFilterInAPISchema(t *testing.T) {
	app, _ := newTrashApp(t, true, true, nil)
	body := trashCall(app, "GET", "/admin/api/schema", nil).Body.String()
	if !strings.Contains(body, `"only_trashed"`) {
		t.Fatal("API schema has no trashed filter")
	}
}

func TestRestoreAndForceDeletePermissions(t *testing.T) {
	app, db := newTrashApp(t, true, false, nil)
	if rec := trashCall(app, "POST", "/admin/api/resources/docs/restore", map[string]any{"id": "2"}); rec.Code != http.StatusForbidden || db["2"].DeletedAt == nil {
		t.Fatalf("restore without delete rights: %d", rec.Code)
	}
	if rec := trashCall(app, "POST", "/admin/api/resources/docs/force-delete", map[string]any{"id": "2"}); rec.Code != http.StatusForbidden || db["2"] == nil {
		t.Fatalf("force delete without rights: %d", rec.Code)
	}
	off, _ := newTrashApp(t, false, true, nil)
	if rec := trashCall(off, "POST", "/admin/api/resources/docs/force-delete", map[string]any{"id": "2"}); rec.Code != http.StatusNotFound {
		t.Fatalf("force delete without soft deletes: %d", rec.Code)
	}
	failing, _ := newTrashApp(t, true, true, errors.New("db down"))
	if rec := trashCall(failing, "POST", "/admin/api/resources/docs/force-delete", map[string]any{"id": "2"}); rec.Code < 500 {
		t.Fatalf("force delete error hidden: %d", rec.Code)
	}
}
