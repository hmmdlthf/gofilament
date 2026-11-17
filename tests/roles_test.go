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

// ann: super admin everywhere. tina: tenant admin of acme (admin panel).
// bob: editor in acme. cy: member of acme and globex, no binding here.
func newRolesApp(t *testing.T) (*golaravel.Engine, *filament.MemoryRoleStore) {
	t.Helper()
	ctx := context.Background()
	store := filament.NewMemoryRoleStore()
	for _, r := range []filament.RoleDef{
		{Name: "admin", Label: "Admin", Permissions: []string{"*"}},
		{Name: "tenant_admin", Label: "Tenant admin", Permissions: []string{"roles.assign", "posts.viewAny", "posts.view", "posts.create"}},
		{Name: "editor", Label: "Editor", Permissions: []string{"posts.viewAny", "posts.view"}},
		{Name: "writer", Label: "Writer", Permissions: []string{"posts.viewAny", "posts.create"}},
		{Name: "deleter", Label: "Deleter", Permissions: []string{"posts.delete"}},
	} {
		_ = store.SaveRole(ctx, r)
	}
	for _, b := range []filament.RoleBinding{
		{UserID: "ann", Role: "admin", Panel: "*", Tenant: "*"},
		{UserID: "tina", Role: "tenant_admin", Panel: "admin", Tenant: "acme"},
		{UserID: "bob", Role: "editor", Panel: "admin", Tenant: "acme"},
		{UserID: "cy", Role: "editor", Panel: "app", Tenant: "acme"},
	} {
		_ = store.AddBinding(ctx, b)
	}

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
	notes := filament.NewResource[*jobRow]("notes").PluralModelName("Notes")
	notes.OnList(func(ctx *filament.Context, s, so, d string, p, pp int, f map[string]string) ([]*jobRow, int, error) {
		return nil, 0, nil
	})
	notes.CanViewAny(func(ctx *filament.Context) bool { return true })

	all := []filament.Tenant{&filament.TenantItem{Key: "acme", Name: "Acme"}, &filament.TenantItem{Key: "globex", Name: "Globex"}}
	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: token, Login: token}, nil
		}).
		Tenancy(&filament.TenantConfig{TenantLoader: func(ctx *filament.Context) ([]filament.Tenant, error) {
			if u := ctx.User(); u != nil && u.GetAuthIdentifier() == "tina" {
				return all[:1], nil
			}
			return all, nil
		}}).
		Roles(store, filament.RolesConfig{
			Permissions: map[string]string{"reports.export": "Export reports"},
			Panels:      map[string]string{"admin": "Admin", "app": "App"},
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

func roleReq(app http.Handler, method, path, user, body string, tenant ...string) (int, map[string]any) {
	req := httptest.NewRequest(method, "http://x"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user)
	if len(tenant) > 0 {
		req.Header.Set("X-Tenant-ID", tenant[0])
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestPanelAccessAndScopedPermissions(t *testing.T) {
	app, _ := newRolesApp(t)
	if c, _ := roleReq(app, "GET", "/admin/api/resources/notes", "cy", ""); c != http.StatusForbidden {
		t.Fatalf("user without a binding in this panel got in: %d", c)
	}
	if c, _ := roleReq(app, "GET", "/admin/api/resources/posts", "bob", "", "acme"); c != 200 {
		t.Fatalf("editor in acme: %d", c)
	}
	if c, _ := roleReq(app, "GET", "/admin/api/resources/posts", "bob", "", "globex"); c != http.StatusForbidden {
		t.Fatalf("editor's acme role worked in globex: %d", c)
	}
	if c, _ := roleReq(app, "POST", "/admin/api/resources/posts", "bob", `{"name":"x"}`, "acme"); c != http.StatusForbidden {
		t.Fatalf("editor created: %d", c)
	}
	if c, _ := roleReq(app, "GET", "/admin/api/resources/notes", "bob", "", "acme"); c != 200 {
		t.Fatalf("own CanViewAny replaced: %d", c)
	}
}

func TestTenantAdminGrantsOnlyInTheirTenant(t *testing.T) {
	app, store := newRolesApp(t)
	grant := func(user, role, panels, tenants string) int {
		c, _ := roleReq(app, "POST", "/admin/api/roles/bindings", "tina",
			fmt.Sprintf(`{"user":%q,"role":%q,"panels":[%s],"tenants":[%s]}`, user, role, panels, tenants), "acme")
		return c
	}
	if c := grant("cy", "editor", `"admin"`, `"acme"`); c != 200 {
		t.Fatalf("tenant admin grant in own tenant: %d", c)
	}
	if c, _ := roleReq(app, "GET", "/admin/api/resources/posts", "cy", "", "acme"); c != 200 {
		t.Fatalf("granted editor cannot list: %d", c)
	}
	for name, c := range map[string]int{
		"other tenant":         grant("cy", "editor", `"admin"`, `"globex"`),
		"all tenants":          grant("cy", "editor", `"admin"`, `"*"`),
		"other panel":          grant("cy", "editor", `"app"`, `"acme"`),
		"role with *":          grant("cy", "admin", `"admin"`, `"acme"`),
		"permission she lacks": grant("cy", "deleter", `"admin"`, `"acme"`),
	} {
		if c != http.StatusForbidden {
			t.Errorf("%s: %d", name, c)
		}
	}
	if c := grant("cy", "writer", `"admin"`, `"acme"`); c != 200 {
		t.Fatalf("role within her permissions: %d", c)
	}
	if c, _ := roleReq(app, "POST", "/admin/api/roles", "tina", `{"name":"x","permissions":[]}`, "acme"); c != http.StatusForbidden {
		t.Fatalf("tenant admin defined a role: %d", c)
	}
	// She sees and can remove acme bindings, not Ann's global one.
	_, users := roleReq(app, "GET", "/admin/api/roles/users", "tina", "", "acme")
	for _, u := range users["users"].([]any) {
		for _, b := range u.(map[string]any)["bindings"].([]any) {
			bm := b.(map[string]any)
			if bm["tenant"] == "globex" {
				t.Fatalf("tenant admin sees another tenant: %v", bm)
			}
			if bm["role"] == "admin" && bm["can_remove"] == true {
				t.Fatal("tenant admin may remove the super admin's role")
			}
		}
	}
	if c, _ := roleReq(app, "DELETE", "/admin/api/roles/bindings", "tina", `{"user":"bob","role":"editor","panel":"admin","tenant":"acme"}`, "acme"); c != 200 {
		t.Fatalf("revoke in own tenant: %d", c)
	}
	if c, _ := roleReq(app, "DELETE", "/admin/api/roles/bindings", "tina", `{"user":"ann","role":"admin","panel":"*","tenant":"*"}`, "acme"); c != http.StatusForbidden {
		t.Fatalf("revoked the super admin: %d", c)
	}
	if bs, _ := store.ListBindings(context.Background(), filament.BindingFilter{UserID: "bob"}); len(bs) != 0 {
		t.Fatalf("bob: %+v", bs)
	}
}

func TestSuperAdminGrantsAcrossPanelsAndTenants(t *testing.T) {
	app, store := newRolesApp(t)
	c, out := roleReq(app, "GET", "/admin/api/roles", "ann", "", "acme")
	if c != 200 || out["can_define"] != true || len(out["panels"].([]any)) != 3 || len(out["tenants"].([]any)) != 3 {
		t.Fatalf("roles meta: %d %v %v", c, out["panels"], out["tenants"])
	}
	if c, _ := roleReq(app, "POST", "/admin/api/roles/bindings", "ann", `{"user":"cy","role":"editor","panels":["admin","app"],"tenants":["acme","globex"]}`, "acme"); c != 200 {
		t.Fatalf("multi grant: %d", c)
	}
	if bs, _ := store.ListBindings(context.Background(), filament.BindingFilter{UserID: "cy"}); len(bs) != 4 { // app/acme existed already
		t.Fatalf("cy: %+v", bs)
	}
	if c, _ := roleReq(app, "POST", "/admin/api/roles", "ann", `{"name":"auditor","label":"Auditor","permissions":["posts.viewAny"]}`, "acme"); c != 200 {
		t.Fatalf("define: %d", c)
	}
	// Lock-out protection.
	if c, _ := roleReq(app, "DELETE", "/admin/api/roles/bindings", "ann", `{"user":"ann","role":"admin","panel":"*","tenant":"*"}`, "acme"); c != http.StatusConflict {
		t.Fatalf("removed own super admin role: %d", c)
	}
	if c, _ := roleReq(app, "PUT", "/admin/api/roles/admin", "ann", `{"name":"admin","permissions":["posts.view"]}`, "acme"); c != http.StatusConflict {
		t.Fatalf("stripped own role: %d", c)
	}
	if c, _ := roleReq(app, "DELETE", "/admin/api/roles/admin", "ann", "", "acme"); c != http.StatusConflict {
		t.Fatalf("deleted own role: %d", c)
	}
	// Renaming keeps the holders.
	if c, _ := roleReq(app, "PUT", "/admin/api/roles/editor", "ann", `{"name":"reader","label":"Reader","permissions":["posts.viewAny","posts.view"]}`, "acme"); c != 200 {
		t.Fatalf("rename: %d", c)
	}
	if bs, _ := store.ListBindings(context.Background(), filament.BindingFilter{Role: "reader"}); len(bs) != 5 {
		t.Fatalf("renamed bindings: %d", len(bs))
	}
}

func TestRolesWebScreens(t *testing.T) {
	app, _ := newRolesApp(t)
	get := func(path, user string) (int, string) {
		req := httptest.NewRequest("GET", "http://x"+path, nil)
		req.Header.Set("Authorization", "Bearer "+user)
		req.Header.Set("X-Tenant-ID", "acme")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}
	if c, _ := get("/admin/roles", "bob"); c != http.StatusForbidden {
		t.Fatalf("editor opened roles: %d", c)
	}
	if c, body := get("/admin/roles", "ann"); c != 200 || !strings.Contains(body, `data-role="editor"`) || !strings.Contains(body, "/admin/roles/new") {
		t.Fatalf("roles page: %d", c)
	}
	if c, body := get("/admin/roles", "tina"); c != 200 || strings.Contains(body, "/admin/roles/new") {
		t.Fatalf("tenant admin roles page: %d", c)
	}
	c, body := get("/admin/roles/users", "tina")
	if c != 200 || !strings.Contains(body, `data-binding="editor|admin|acme"`) || strings.Contains(body, `value="globex"`) || strings.Contains(body, `name="tenants" value="*"`) {
		t.Fatalf("tenant admin users page: %d\n%s", c, body)
	}
	if _, body := get("/admin", "ann"); !strings.Contains(body, "/admin/roles") {
		t.Fatal("no Roles navigation for managers")
	}
}
