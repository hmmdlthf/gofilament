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
	fsupport "github.com/golaravel/gofilament/pkg/support"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/privacy"
	gsupport "github.com/golaravel/golaravel/pkg/support"
)

// Tenant isolation: bob belongs to acme only, but still has a leftover
// admin binding in initech (he left it, the role store was not cleaned up).
// Naming initech in the path, cookie or header must not let him read or
// change initech's data.

type isoApp struct {
	app   *golaravel.Engine
	store *filament.MemoryRoleStore
	docs  map[string][]*jobRow // tenant -> rows
}

// newIsoApp builds the panel. With loader, membership comes from a
// TenantLoader (bob: acme; eve: none); otherwise from CanAccessTenant.
func newIsoApp(t *testing.T, loader bool) *isoApp {
	t.Helper()
	ctx := context.Background()
	store := filament.NewMemoryRoleStore()
	_ = store.SaveRole(ctx, filament.RoleDef{Name: "admin", Label: "Admin", Permissions: []string{"*"}})
	_ = store.SaveRole(ctx, filament.RoleDef{Name: "viewer", Label: "Viewer", Permissions: []string{"docs.viewAny"}})
	for _, b := range []filament.RoleBinding{
		{UserID: "bob", Role: "admin", Panel: "admin", Tenant: "acme"},
		{UserID: "bob", Role: "admin", Panel: "admin", Tenant: "initech"}, // stale
		{UserID: "eve", Role: "admin", Panel: "admin", Tenant: "initech"}, // stale, no tenants left
		{UserID: "ivy", Role: "viewer", Panel: "admin", Tenant: "initech"},
	} {
		_ = store.AddBinding(ctx, b)
	}
	members := map[string][]string{"bob": {"acme"}, "ivy": {"initech"}}

	ia := &isoApp{store: store, docs: map[string][]*jobRow{
		"acme":    {{ID: 1, Name: "acme-doc"}},
		"initech": {{ID: 2, Name: "initech-secret"}},
	}}

	// An app resource that scopes its data by ctx.Tenant(), as apps do.
	docs := filament.NewResource[*jobRow]("docs").PluralModelName("Docs")
	docs.Form(func(f *forms.Form) *forms.Form { return f.Schema(forms.TextInput("name")) })
	docs.Table(func(tb *tables.Table[*jobRow]) *tables.Table[*jobRow] {
		return tb.Columns(tables.TextColumn[*jobRow]("name"))
	})
	docs.RecordID(func(x *jobRow) string { return fmt.Sprint(x.ID) })
	find := func(ctx *filament.Context, id string) *jobRow {
		for _, r := range ia.docs[ctx.Tenant()] {
			if fmt.Sprint(r.ID) == id {
				return r
			}
		}
		return nil
	}
	docs.OnList(func(ctx *filament.Context, s, so, d string, p, pp int, f map[string]string) ([]*jobRow, int, error) {
		rows := ia.docs[ctx.Tenant()]
		return rows, len(rows), nil
	})
	docs.OnFind(func(ctx *filament.Context, id string) (*jobRow, error) {
		if r := find(ctx, id); r != nil {
			return r, nil
		}
		return nil, filament.ErrRecordNotFound
	})
	docs.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		r := find(ctx, id)
		if r == nil {
			return filament.ErrRecordNotFound
		}
		r.Name = fmt.Sprint(data["name"])
		return nil
	})

	tickets := gsupport.NewMemorySupportStore()
	_ = tickets.CreateTicket(ctx, &gsupport.Ticket{ID: "t-acme", TenantID: "acme", UserKey: "c1", Subject: "acme-ticket", Status: "open"})
	_ = tickets.CreateTicket(ctx, &gsupport.Ticket{ID: "t-initech", TenantID: "initech", UserKey: "c2", Subject: "initech-ticket", Status: "open"})
	prev := gsupport.GetDefaultStore()
	gsupport.SetDefaultStore(tickets)
	t.Cleanup(func() { gsupport.SetDefaultStore(prev) })
	supportRes := fsupport.NewSupportResource(
		fsupport.WithStore(tickets),
		fsupport.WithScope(fsupport.ScopeTenantAdmin),
		fsupport.WithTenantScoping(true),
	)

	cfg := &filament.TenantConfig{PathScoping: true}
	if loader {
		cfg.TenantLoader = func(ctx *filament.Context) ([]filament.Tenant, error) {
			var out []filament.Tenant
			if u := ctx.User(); u != nil {
				for _, k := range members[u.GetAuthIdentifier()] {
					out = append(out, &filament.TenantItem{Key: k, Name: k})
				}
			}
			return out, nil
		}
	} else {
		cfg.CanAccessTenant = func(ctx *filament.Context, key string) bool {
			if u := ctx.User(); u != nil {
				for _, k := range members[u.GetAuthIdentifier()] {
					if k == key {
						return true
					}
				}
			}
			return false
		}
	}

	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: token, Login: token}, nil
		}).
		Tenancy(cfg).
		TenantAdminSupport().
		Roles(store, filament.RolesConfig{
			FindUsers: func(ctx *filament.Context, q string) ([]filament.UserRef, error) {
				return []filament.UserRef{{ID: "bob"}, {ID: "eve"}, {ID: "ivy"}, {ID: "mallory"}}, nil
			},
		}).
		Resources(docs, supportRes)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	// An app route under the tenant path, trusting ctx.Tenant() and UserCan.
	app.Router.GET("/admin/:tenant/probe", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		priv, _ := privacy.TenantFromContext(ctx.GoContext())
		return c.JSON(http.StatusOK, map[string]any{
			"tenant":    ctx.Tenant(),
			"requested": ctx.RequestedTenant(),
			"privacy":   priv,
			"admin":     p.UserCan(ctx, "docs.update"),
			"docs":      ia.docs[ctx.Tenant()],
		})
	})
	ia.app = app
	return ia
}

type tenantVia int

const (
	viaHeader tenantVia = iota
	viaCookie
)

func (ia *isoApp) req(method, path, user, body string, via tenantVia, tenant string) (int, string) {
	req := httptest.NewRequest(method, "http://x"+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+user)
	if tenant != "" {
		switch via {
		case viaHeader:
			req.Header.Set("X-Tenant-ID", tenant)
		case viaCookie:
			req.AddCookie(&http.Cookie{Name: "filament_tenant", Value: tenant})
		}
	}
	rec := httptest.NewRecorder()
	ia.app.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func eachIsoMode(t *testing.T, fn func(t *testing.T, ia *isoApp, via tenantVia)) {
	for _, loader := range []bool{false, true} {
		for _, via := range []tenantVia{viaHeader, viaCookie} {
			name := fmt.Sprintf("loader=%t/via=%s", loader, map[tenantVia]string{viaHeader: "header", viaCookie: "cookie"}[via])
			t.Run(name, func(t *testing.T) { fn(t, newIsoApp(t, loader), via) })
		}
	}
}

func TestRefusedTenantInPathIsNotActive(t *testing.T) {
	for _, loader := range []bool{false, true} {
		t.Run(fmt.Sprintf("loader=%t", loader), func(t *testing.T) {
			ia := newIsoApp(t, loader)
			for _, user := range []string{"bob", "eve"} {
				code, body := ia.req("GET", "/admin/initech/probe", user, "", viaHeader, "")
				if code != http.StatusOK {
					t.Fatalf("%s probe: %d %s", user, code, body)
				}
				var out map[string]any
				_ = json.Unmarshal([]byte(body), &out)
				if out["tenant"] == "initech" || out["privacy"] == "initech" {
					t.Fatalf("%s: refused path tenant became active: %s", user, body)
				}
				if out["requested"] != "initech" {
					t.Fatalf("%s: RequestedTenant should report the raw segment: %s", user, body)
				}
				if strings.Contains(body, "initech-secret") {
					t.Fatalf("%s read initech's data through the path: %s", user, body)
				}
			}
			// eve has no tenant at all: nothing is active, and her stale
			// initech binding gives no permissions.
			_, body := ia.req("GET", "/admin/initech/probe", "eve", "", viaHeader, "")
			if !strings.Contains(body, `"tenant":""`) || !strings.Contains(body, `"admin":false`) {
				t.Fatalf("eve: %s", body)
			}
			// Control: a member's own tenant works.
			_, body = ia.req("GET", "/admin/acme/probe", "bob", "", viaHeader, "")
			if !strings.Contains(body, `"tenant":"acme"`) || !strings.Contains(body, `"admin":true`) || !strings.Contains(body, "acme-doc") {
				t.Fatalf("bob in acme: %s", body)
			}
		})
	}
}

func TestRefusedTenantCannotReachResourceData(t *testing.T) {
	eachIsoMode(t, func(t *testing.T, ia *isoApp, via tenantVia) {
		for _, user := range []string{"bob", "eve"} {
			if _, body := ia.req("GET", "/admin/api/resources/docs", user, "", via, "initech"); strings.Contains(body, "initech-secret") {
				t.Fatalf("%s listed initech's docs: %s", user, body)
			}
			if code, body := ia.req("GET", "/admin/api/resources/docs/2", user, "", via, "initech"); code == http.StatusOK {
				t.Fatalf("%s read initech's doc: %d %s", user, code, body)
			}
			ia.req("PUT", "/admin/api/resources/docs/2", user, `{"name":"pwned"}`, via, "initech")
			if got := ia.docs["initech"][0].Name; got != "initech-secret" {
				t.Fatalf("%s changed initech's doc: %q", user, got)
			}
		}
		// Control: bob's own tenant.
		if code, body := ia.req("GET", "/admin/api/resources/docs", "bob", "", via, "acme"); code != http.StatusOK || !strings.Contains(body, "acme-doc") {
			t.Fatalf("bob in acme: %d %s", code, body)
		}
	})
}

func TestRefusedTenantCannotUseRoles(t *testing.T) {
	eachIsoMode(t, func(t *testing.T, ia *isoApp, via tenantVia) {
		grant := `{"user":"mallory","role":"admin","panels":["admin"],"tenants":["initech"]}`
		for _, user := range []string{"bob", "eve"} {
			for _, tenant := range []string{"initech", "acme"} {
				ia.req("POST", "/admin/api/roles/bindings", user, grant, via, tenant)
			}
			ia.req("DELETE", "/admin/api/roles/bindings", user, `{"user":"ivy","role":"viewer","panel":"admin","tenant":"initech"}`, via, "initech")
		}
		bs, _ := ia.store.ListBindings(context.Background(), filament.BindingFilter{})
		var ivy bool
		for _, b := range bs {
			if b.UserID == "mallory" {
				t.Fatalf("a stale initech binding granted a role in initech: %+v", b)
			}
			ivy = ivy || (b.UserID == "ivy" && b.Tenant == "initech")
		}
		if !ivy {
			t.Fatal("a stale initech binding removed ivy's initech role")
		}
		// eve has no tenant: the roles screens are closed to her.
		if code, _ := ia.req("GET", "/admin/api/roles", "eve", "", via, "initech"); code != http.StatusForbidden {
			t.Fatalf("eve opened the roles API: %d", code)
		}
		// Control: bob may still give roles in acme.
		if code, body := ia.req("POST", "/admin/api/roles/bindings", "bob",
			`{"user":"mallory","role":"viewer","panels":["admin"],"tenants":["acme"]}`, via, "acme"); code >= 300 {
			t.Fatalf("bob granting in acme: %d %s", code, body)
		}
	})
}

func TestRefusedTenantCannotReachSupportTickets(t *testing.T) {
	eachIsoMode(t, func(t *testing.T, ia *isoApp, via tenantVia) {
		for _, user := range []string{"bob", "eve"} {
			for _, path := range []string{"/admin/api/support/tickets", "/admin/api/support/tickets?tenant_id=initech"} {
				if _, body := ia.req("GET", path, user, "", via, "initech"); strings.Contains(body, "initech-ticket") {
					t.Fatalf("%s listed initech's tickets via %s: %s", user, path, body)
				}
			}
			if _, body := ia.req("GET", "/admin/api/resources/support-tickets", user, "", via, "initech"); strings.Contains(body, "initech-ticket") {
				t.Fatalf("%s listed initech's tickets as a resource: %s", user, body)
			}
		}
		// Even in his own tenant, bob cannot open initech's ticket by ID.
		if code, body := ia.req("GET", "/admin/api/resources/support-tickets/t-initech", "bob", "", via, "acme"); code == http.StatusOK || strings.Contains(body, "initech-ticket") {
			t.Fatalf("bob read initech's ticket by ID: %d %s", code, body)
		}
		if _, body := ia.req("GET", "/admin/api/support/tickets", "bob", "", via, "acme"); !strings.Contains(body, "acme-ticket") || strings.Contains(body, "initech-ticket") {
			t.Fatalf("bob's acme tickets: %s", body)
		}
	})
}

func TestRefusedTenantNotInClientSchema(t *testing.T) {
	eachIsoMode(t, func(t *testing.T, ia *isoApp, via tenantVia) {
		for _, path := range []string{"/admin/api/schema", "/admin/api/me", "/admin/api/boot"} {
			_, body := ia.req("GET", path, "bob", "", via, "initech")
			if strings.Contains(body, `"tenant":"initech"`) || strings.Contains(body, `"active_tenant":"initech"`) {
				t.Fatalf("%s reports the refused tenant as active: %s", path, body)
			}
		}
	})
}

// Middleware may put a tenant in the request context; on a panel with
// tenancy it is only a candidate, like a cookie.
func TestUpstreamTenantIsCheckedByPanel(t *testing.T) {
	p := filament.NewPanel("admin").Tenancy(&filament.TenantConfig{
		CanAccessTenant: func(ctx *filament.Context, key string) bool { return key == "acme" },
	})
	for key, want := range map[string]string{"acme": "acme", "initech": ""} {
		req := httptest.NewRequest(http.MethodGet, "http://x/admin", nil)
		req = req.WithContext(privacy.WithTenantID(req.Context(), key))
		ctx := filament.NewContext(newTestContext(httptest.NewRecorder(), req), p)
		if got := ctx.Tenant(); got != want {
			t.Fatalf("upstream %s: Tenant() = %q, want %q", key, got, want)
		}
		if got, _ := privacy.TenantFromContext(ctx.GoContext()); got != want {
			t.Fatalf("upstream %s: request context still scoped to %q", key, got)
		}
		if got := ctx.RequestedTenant(); got != key {
			t.Fatalf("RequestedTenant = %q, want %q", got, key)
		}
	}
	// Without panel tenancy the middleware's tenant is trusted as before.
	plain := filament.NewPanel("admin")
	req := httptest.NewRequest(http.MethodGet, "http://x/admin", nil)
	req = req.WithContext(privacy.WithTenantID(req.Context(), "acme"))
	if got := filament.NewContext(newTestContext(httptest.NewRecorder(), req), plain).Tenant(); got != "acme" {
		t.Fatalf("no tenancy: %q", got)
	}
}
