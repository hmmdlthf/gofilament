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

func newRolesApp(t *testing.T) (*golaravel.Engine, *filament.MemoryRoleStore) {
	t.Helper()
	ctx := context.Background()
	store := filament.NewMemoryRoleStore()
	_ = store.SaveRole(ctx, filament.RoleDef{Name: "admin", Label: "Admin", Permissions: []string{"*"}})
	_ = store.SaveRole(ctx, filament.RoleDef{Name: "editor", Label: "Editor", Permissions: []string{"posts.viewAny", "posts.view"}})
	_ = store.SetUserRoles(ctx, "ann", []string{"admin"})
	_ = store.SetUserRoles(ctx, "bob", []string{"editor"})

	posts := filament.NewResource[*jobRow]("posts").PluralModelName("Posts")
	posts.Form(func(f *forms.Form) *forms.Form { return f.Schema(forms.TextInput("name")) })
	posts.Table(func(tb *tables.Table[*jobRow]) *tables.Table[*jobRow] {
		return tb.Columns(tables.TextColumn[*jobRow]("name"))
	})
	posts.RecordID(func(x *jobRow) string { return fmt.Sprint(x.ID) })
	posts.OnList(func(ctx *filament.Context, s, so, d string, p, pp int, f map[string]string) ([]*jobRow, int, error) {
		return []*jobRow{{ID: 1, Name: "Hello"}}, 1, nil
	})
	posts.OnCreate(func(ctx *filament.Context, data map[string]any) error { return nil })

	// A resource with its own rule keeps it.
	notes := filament.NewResource[*jobRow]("notes").PluralModelName("Notes")
	notes.OnList(func(ctx *filament.Context, s, so, d string, p, pp int, f map[string]string) ([]*jobRow, int, error) {
		return nil, 0, nil
	})
	notes.CanViewAny(func(ctx *filament.Context) bool { return true })

	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: token, Login: token}, nil
		}).
		Roles(store, filament.RolesConfig{
			Permissions: map[string]string{"reports.export": "Export reports"},
			FindUsers: func(ctx *filament.Context, q string) ([]filament.UserRef, error) {
				return []filament.UserRef{{ID: "ann", Name: "Ann"}, {ID: "bob", Name: "Bob"}, {ID: "cy", Name: "Cy"}}, nil
			},
		}).
		Resources(posts, notes)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, store
}

func roleReq(app http.Handler, method, path, user, body string) (int, map[string]any) {
	req := httptest.NewRequest(method, "http://x"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestRolePermissionsAreEnforced(t *testing.T) {
	app, _ := newRolesApp(t)
	if c, _ := roleReq(app, "GET", "/admin/api/resources/posts", "cy", ""); c != http.StatusForbidden {
		t.Fatalf("no role listed posts: %d", c)
	}
	if c, _ := roleReq(app, "GET", "/admin/api/resources/posts", "bob", ""); c != 200 {
		t.Fatalf("editor list: %d", c)
	}
	if c, _ := roleReq(app, "POST", "/admin/api/resources/posts", "bob", `{"name":"x"}`); c != http.StatusForbidden {
		t.Fatalf("editor created: %d", c)
	}
	if c, _ := roleReq(app, "POST", "/admin/api/resources/posts", "ann", `{"name":"x"}`); c >= 300 {
		t.Fatalf("admin create: %d", c)
	}
	if c, _ := roleReq(app, "GET", "/admin/api/resources/notes", "cy", ""); c != 200 {
		t.Fatalf("own CanViewAny was replaced: %d", c)
	}
}

func TestRolesAPIAndLockout(t *testing.T) {
	app, store := newRolesApp(t)
	if c, _ := roleReq(app, "GET", "/admin/api/roles", "bob", ""); c != http.StatusForbidden {
		t.Fatalf("editor managed roles: %d", c)
	}
	c, out := roleReq(app, "GET", "/admin/api/roles", "ann", "")
	perms := out["permissions"].(map[string]any)
	if c != 200 || perms["posts.create"] == nil || perms["reports.export"] == nil || perms["roles.manage"] == nil {
		t.Fatalf("roles: %d %v", c, out)
	}
	if c, out := roleReq(app, "POST", "/admin/api/roles", "ann", `{"name":"Bad Name","permissions":["nope"]}`); c != http.StatusUnprocessableEntity || out["errors"] == nil {
		t.Fatalf("invalid role: %d %v", c, out)
	}
	if c, _ := roleReq(app, "POST", "/admin/api/roles", "ann", `{"name":"writer","label":"Writer","permissions":["posts.viewAny","posts.create"]}`); c != 200 {
		t.Fatalf("create role: %d", c)
	}
	if c, _ := roleReq(app, "PUT", "/admin/api/roles/users/cy", "ann", `{"roles":["writer"]}`); c != 200 {
		t.Fatalf("assign: %d", c)
	}
	if c, _ := roleReq(app, "POST", "/admin/api/resources/posts", "cy", `{"name":"x"}`); c >= 300 {
		t.Fatalf("writer could not create: %d", c)
	}
	_, users := roleReq(app, "GET", "/admin/api/roles/users", "ann", "")
	if len(users["users"].([]any)) != 3 {
		t.Fatalf("users: %v", users)
	}
	// Lock-out protection.
	if c, _ := roleReq(app, "PUT", "/admin/api/roles/users/ann", "ann", `{"roles":["editor"]}`); c != http.StatusConflict {
		t.Fatalf("removed own admin role: %d", c)
	}
	if c, _ := roleReq(app, "DELETE", "/admin/api/roles/admin", "ann", ""); c != http.StatusConflict {
		t.Fatalf("deleted own admin role: %d", c)
	}
	if c, _ := roleReq(app, "PUT", "/admin/api/roles/admin", "ann", `{"name":"admin","permissions":["posts.view"]}`); c != http.StatusConflict {
		t.Fatalf("stripped own role: %d", c)
	}
	// Renaming keeps the holders.
	if c, _ := roleReq(app, "PUT", "/admin/api/roles/writer", "ann", `{"name":"author","label":"Author","permissions":["posts.viewAny"]}`); c != 200 {
		t.Fatalf("rename: %d", c)
	}
	if rs, _ := store.UserRoles(context.Background(), "cy"); strings.Join(rs, ",") != "author" {
		t.Fatalf("cy after rename: %v", rs)
	}
}

func TestRolesWebScreens(t *testing.T) {
	app, _ := newRolesApp(t)
	page := bulkReq(app, "GET", "/admin/roles", "", "")
	if page.Code != http.StatusForbidden {
		// bulkReq signs in as "t", who has no roles
		t.Fatalf("roles page without permission: %d", page.Code)
	}
	get := func(path, user string) (int, string) {
		req := httptest.NewRequest("GET", "http://x"+path, nil)
		req.Header.Set("Authorization", "Bearer "+user)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}
	if c, body := get("/admin/roles", "ann"); c != 200 || !strings.Contains(body, `data-role="editor"`) {
		t.Fatalf("roles page: %d", c)
	}
	if c, body := get("/admin/roles/editor", "ann"); c != 200 || !strings.Contains(body, `value="posts.view" checked`) {
		t.Fatalf("role form: %d", c)
	}
	if c, body := get("/admin/roles/users", "ann"); c != 200 || !strings.Contains(body, `data-user="cy"`) {
		t.Fatalf("users page: %d", c)
	}
	if _, body := get("/admin", "ann"); !strings.Contains(body, "/admin/roles") {
		t.Fatal("no Roles navigation for managers")
	}
	if _, body := get("/admin", "bob"); strings.Contains(body, `href="/admin/roles"`) {
		t.Fatal("Roles navigation for non-managers")
	}
}
