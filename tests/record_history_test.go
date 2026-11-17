package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/activitylog"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

// Fields without json tags: the record's keys ("FullName") differ from the
// form's ("full_name").
type historyCustomer struct {
	ID       int
	FullName string
	Email    string
	Password string
	VIP      bool
}

func newHistoryApp(t *testing.T, canView bool) (*golaravel.Engine, *activitylog.MemoryStore, map[int]*historyCustomer) {
	t.Helper()
	store := activitylog.NewMemoryStore()
	db := map[int]*historyCustomer{1: {ID: 1, FullName: "Ann Lee", Email: "ann@example.com", Password: "x"}}
	res := filament.NewResource[*historyCustomer]("customers").ModelName("Customer")
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.TextInput("full_name").Label("Full name"),
			forms.TextInput("email").Label("Email"),
			forms.TextInput("password").Password(),
			forms.Toggle("vip").Label("VIP"),
		)
	})
	res.Table(func(tb *tables.Table[*historyCustomer]) *tables.Table[*historyCustomer] {
		return tb.Columns(tables.TextColumn[*historyCustomer]("full_name"), tables.ToggleColumn[*historyCustomer]("vip"))
	})
	res.RecordID(func(c *historyCustomer) string { return fmt.Sprint(c.ID) })
	res.OnFind(func(ctx *filament.Context, id string) (*historyCustomer, error) {
		for _, c := range db {
			if fmt.Sprint(c.ID) == id {
				cp := *c
				return &cp, nil
			}
		}
		return nil, fmt.Errorf("not found")
	})
	res.OnList(func(ctx *filament.Context, search, sort, dir string, page, perPage int, filters map[string]string) ([]*historyCustomer, int, error) {
		return nil, 0, nil
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		c := &historyCustomer{ID: len(db) + 1, FullName: fmt.Sprint(data["full_name"]), Email: fmt.Sprint(data["email"])}
		db[c.ID] = c
		filament.SetCreatedID(ctx, fmt.Sprint(c.ID))
		return nil
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		for _, c := range db {
			if fmt.Sprint(c.ID) == id {
				if v, ok := data["full_name"]; ok {
					c.FullName = fmt.Sprint(v)
				}
				if v, ok := data["email"]; ok {
					c.Email = fmt.Sprint(v)
				}
				if v, ok := data["password"]; ok && v != "" {
					c.Password = fmt.Sprint(v)
				}
			}
		}
		return nil
	})
	res.OnCellUpdate(func(ctx *filament.Context, id, column string, value any) error {
		for _, c := range db {
			if fmt.Sprint(c.ID) == id && column == "vip" {
				c.VIP = value == true || value == "true"
			}
		}
		return nil
	})
	res.CanView(func(ctx *filament.Context, rec *historyCustomer) bool { return canView })
	activitylog.EnableAuditLog(res, store)

	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: "7", Login: "olivia", Name: "Olivia"}, nil
		}).Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, store, db
}

func historyOf(t *testing.T, app *golaravel.Engine, id string) (int, []filament.HistoryEntry) {
	t.Helper()
	rec := histSend(app, "GET", "/admin/api/resources/customers/"+id+"/history", "", nil)
	var out struct {
		Entries []filament.HistoryEntry `json:"entries"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out.Entries
}

func putJSON(app *golaravel.Engine, path string, body map[string]any) int {
	raw, _ := json.Marshal(body)
	return histSend(app, "PUT", path, "application/json", raw).Code
}

func TestRecordHistory(t *testing.T) {
	app, store, _ := newHistoryApp(t, true)

	raw, _ := json.Marshal(map[string]any{"full_name": "Bo Diaz", "email": "bo@example.com", "password": "hunter22"})
	if rec := histSend(app, "POST", "/admin/api/resources/customers", "application/json", raw); rec.Code >= 300 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	_, entries := historyOf(t, app, "2")
	if len(entries) != 1 || entries[0].Event != "created" || entries[0].CauserName != "Olivia" {
		t.Fatalf("created entry: %+v", entries)
	}
	acts, _ := store.GetBySubject(t.Context(), "Customer", "2")
	if acts[0].NewValues["password"] != "(hidden)" {
		t.Fatalf("password stored: %v", acts[0].NewValues)
	}

	// Only the changed field, matched to the struct's FullName, with its label.
	if code := putJSON(app, "/admin/api/resources/customers/1", map[string]any{"full_name": "Ann Park", "email": "ann@example.com", "password": ""}); code != 200 {
		t.Fatalf("update: %d", code)
	}
	// Nothing changed: no entry.
	putJSON(app, "/admin/api/resources/customers/1", map[string]any{"full_name": "Ann Park", "email": "ann@example.com"})
	// A password change is recorded without its value.
	putJSON(app, "/admin/api/resources/customers/1", map[string]any{"password": "new-secret"})
	// Inline edit.
	cell, _ := json.Marshal(map[string]any{"column": "vip", "value": true})
	if rec := histSend(app, "POST", "/admin/api/resources/customers/1/cell", "application/json", cell); rec.Code != 200 {
		t.Fatalf("cell: %d %s", rec.Code, rec.Body.String())
	}

	code, entries := historyOf(t, app, "1")
	if code != 200 || len(entries) != 3 {
		t.Fatalf("history: %d %+v", code, entries)
	}
	vip, pw, name := entries[0], entries[1], entries[2]
	if len(vip.Changes) != 1 || vip.Changes[0].Label != "VIP" || vip.Changes[0].Old != "false" || vip.Changes[0].New != "true" {
		t.Fatalf("inline change: %+v", vip)
	}
	if len(pw.Changes) != 1 || pw.Changes[0].Old != "(hidden)" || pw.Changes[0].New != "(changed)" {
		t.Fatalf("password change: %+v", pw)
	}
	if len(name.Changes) != 1 || name.Changes[0].Field != "full_name" || name.Changes[0].Label != "Full name" ||
		name.Changes[0].Old != "Ann Lee" || name.Changes[0].New != "Ann Park" {
		t.Fatalf("name change: %+v", name)
	}
	for _, a := range func() []*activitylog.Activity { a, _ := store.GetBySubject(t.Context(), "Customer", "1"); return a }() {
		if strings.Contains(fmt.Sprint(a.NewValues, a.OldValues), "new-secret") {
			t.Fatal("password value stored")
		}
	}

	// The web edit page is filled from the struct record and shows the history.
	page := histSend(app, "GET", "/admin/customers/1/edit", "", nil).Body.String()
	if strings.Contains(page, `name="password" value="x"`) {
		t.Fatal("stored password echoed into the edit form")
	}
	if !strings.Contains(page, `value="Ann Park"`) || !strings.Contains(page, "filament-record-history") || !strings.Contains(page, "Ann Lee") {
		t.Fatalf("edit page:\n%s", page)
	}
	_, schema := clientGet(t, app, "example.com", "/admin/api/schema", "t")
	meta := schema["resource_meta"].(map[string]any)["customers"].(map[string]any)
	if meta["abilities"].(map[string]any)["history"] != true {
		t.Fatalf("history ability: %v", meta["abilities"])
	}
}

func TestRecordHistoryNeedsViewRights(t *testing.T) {
	app, _, _ := newHistoryApp(t, false)
	if code, entries := historyOf(t, app, "1"); code != http.StatusForbidden || entries != nil {
		t.Fatalf("history without view rights: %d %+v", code, entries)
	}
}

func histSend(app http.Handler, method, path, contentType string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://example.com"+path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}
