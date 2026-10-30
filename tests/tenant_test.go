package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/privacy"
)

func newTestContext(w http.ResponseWriter, r *http.Request) *gcontext.Context {
	c := gcontext.NewContext()
	c.Reset(w, r)
	return c
}

func TestPanelTenancyConfiguration(t *testing.T) {
	p := filament.NewPanel("admin").
		Tenancy(&filament.TenantConfig{
			Enabled:         true,
			PathScoping:     true,
			RegistrationURL: "/admin/tenants/new",
			ProfileURL:      "/admin/{tenant}/settings",
		})

	cfg := p.GetTenantConfig()
	if cfg == nil || !cfg.Enabled {
		t.Fatalf("expected tenantConfig to be enabled")
	}
	if !cfg.PathScoping {
		t.Fatalf("expected PathScoping to be true")
	}
	if cfg.RegistrationURL != "/admin/tenants/new" {
		t.Fatalf("expected registration URL mismatch: %s", cfg.RegistrationURL)
	}
}

func TestTenantResolutionAndPrivacyInjection(t *testing.T) {
	mockTenants := []filament.Tenant{
		filament.TenantItem{Key: "org_alpha", Name: "Alpha Corp"},
		filament.TenantItem{Key: "org_beta", Name: "Beta Ltd"},
	}

	p := filament.NewPanel("admin").
		Tenancy().
		TenantLoader(func(ctx *filament.Context) ([]filament.Tenant, error) {
			return mockTenants, nil
		})

	// 1. Fallback to first tenant when request has no specific identifier
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	gc := newTestContext(rec, req)
	fCtx := filament.NewContext(gc, p)

	resolved := p.ResolveTenant(fCtx)
	if resolved == nil || resolved.GetTenantKey() != "org_alpha" {
		t.Fatalf("expected org_alpha, got %+v", resolved)
	}
	if fCtx.Tenant() != "org_alpha" {
		t.Fatalf("expected fCtx.Tenant() to be org_alpha, got %s", fCtx.Tenant())
	}

	// Verify privacy context has been populated for Ent ORM
	entTenant, ok := privacy.TenantFromContext(fCtx.Request.Context())
	if !ok || entTenant != "org_alpha" {
		t.Fatalf("expected ent privacy tenant org_alpha, got %s (ok=%t)", entTenant, ok)
	}

	// 2. Resolve via Cookie
	reqCookie := httptest.NewRequest(http.MethodGet, "/admin", nil)
	reqCookie.AddCookie(&http.Cookie{Name: "filament_tenant", Value: "org_beta"})
	recCookie := httptest.NewRecorder()
	gcCookie := newTestContext(recCookie, reqCookie)
	fCtxCookie := filament.NewContext(gcCookie, p)

	if fCtxCookie.Tenant() != "org_beta" {
		t.Fatalf("expected org_beta from cookie, got %s", fCtxCookie.Tenant())
	}
	if fCtxCookie.TenantRecord().GetTenantName() != "Beta Ltd" {
		t.Fatalf("expected Beta Ltd, got %s", fCtxCookie.TenantRecord().GetTenantName())
	}

	// 3. Resolve via Header
	reqHeader := httptest.NewRequest(http.MethodGet, "/admin", nil)
	reqHeader.Header.Set("X-Tenant-ID", "org_beta")
	recHeader := httptest.NewRecorder()
	gcHeader := newTestContext(recHeader, reqHeader)
	fCtxHeader := filament.NewContext(gcHeader, p)

	if fCtxHeader.Tenant() != "org_beta" {
		t.Fatalf("expected org_beta from header, got %s", fCtxHeader.Tenant())
	}

	// 4. Manual SetTenant updates GoLaravel privacy context
	fCtx.SetTenant("org_gamma")
	if fCtx.Tenant() != "org_gamma" {
		t.Fatalf("expected org_gamma, got %s", fCtx.Tenant())
	}
	privacyTenant, ok := privacy.TenantFromContext(fCtx.Request.Context())
	if !ok || privacyTenant != "org_gamma" {
		t.Fatalf("expected org_gamma in privacy context, got %s", privacyTenant)
	}
}

func TestTenantDomainScoping(t *testing.T) {
	p := filament.NewPanel("admin").
		Tenancy().
		TenantDomainScoping("acme.io")

	req := httptest.NewRequest(http.MethodGet, "http://tenant123.acme.io/admin", nil)
	rec := httptest.NewRecorder()
	gc := newTestContext(rec, req)
	fCtx := filament.NewContext(gc, p)

	if fCtx.Tenant() != "tenant123" {
		t.Fatalf("expected subdomain tenant123, got %s", fCtx.Tenant())
	}
}

func TestTenantSwitcherTopbarRendering(t *testing.T) {
	mockTenants := []filament.Tenant{
		filament.TenantItem{Key: "t1", Name: "Acme Corp"},
		filament.TenantItem{Key: "t2", Name: "Wayne Enterprises"},
	}

	p := filament.NewPanel("admin").
		Tenancy().
		TenantLoader(func(ctx *filament.Context) ([]filament.Tenant, error) {
			return mockTenants, nil
		}).
		TenantRegistration("/admin/tenants/create").
		TenantProfile("/admin/{tenant}/settings")

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "filament_tenant", Value: "t2"})
	rec := httptest.NewRecorder()
	gc := newTestContext(rec, req)
	fCtx := filament.NewContext(gc, p)

	topbarHTML := p.RenderTopbar(fCtx)

	// Check current active tenant
	if !strings.Contains(topbarHTML, "Wayne Enterprises") {
		t.Fatalf("expected Wayne Enterprises in topbar, got: %s", topbarHTML)
	}
	// Check switcher dropdown items
	if !strings.Contains(topbarHTML, "Acme Corp") {
		t.Fatalf("expected Acme Corp in switcher dropdown")
	}
	if !strings.Contains(topbarHTML, "/admin/tenant/switch?tenant=t1") {
		t.Fatalf("expected switch URL for t1")
	}
	// Check action links
	if !strings.Contains(topbarHTML, "/admin/tenants/create") {
		t.Fatalf("expected registration URL in switcher")
	}
	if !strings.Contains(topbarHTML, "/admin/t2/settings") {
		t.Fatalf("expected profile URL with tenant replacement in switcher")
	}
}

func TestTenantPathScopingAndNavigation(t *testing.T) {
	p := filament.NewPanel("admin").
		Path("/admin").
		Tenancy().
		TenantPathScoping(true)

	req := httptest.NewRequest(http.MethodGet, "/admin/acme/dashboard", nil)
	rec := httptest.NewRecorder()
	gc := newTestContext(rec, req)
	fCtx := filament.NewContext(gc, p)
	fCtx.SetTenant("acme")

	// Check TenantURL helper
	usersURL := p.TenantURL(fCtx, "/users")
	if usersURL != "/admin/acme/users" {
		t.Fatalf("expected /admin/acme/users, got %s", usersURL)
	}

	// Check BuildNavigation uses tenant prefix
	navGroups := p.BuildNavigation(fCtx)
	if len(navGroups) == 0 || len(navGroups[0].Items) == 0 {
		t.Fatalf("expected navigation items")
	}
	dashItem := navGroups[0].Items[0]
	if dashItem.URL != "/admin/acme" {
		t.Fatalf("expected /admin/acme for dashboard, got %s", dashItem.URL)
	}
}

func TestTenantSwitchRoute(t *testing.T) {
	var switchedTenant string
	app := golaravel.New()

	p := filament.NewPanel("admin").
		Tenancy(&filament.TenantConfig{
			Enabled:    true,
			CookieName: "filament_tenant",
			OnTenantSwitch: func(ctx *filament.Context, newTenant filament.Tenant) error {
				switchedTenant = newTenant.GetTenantKey()
				return nil
			},
		})

	err := p.Register(app)
	if err != nil {
		t.Fatalf("failed to register panel: %v", err)
	}

	client := app.Test(t)
	resp := client.Get("/admin/tenant/switch?tenant=workspace_99")
	resp.AssertStatus(http.StatusSeeOther)

	// Check callback executed
	if switchedTenant != "workspace_99" {
		t.Fatalf("expected switchedTenant workspace_99, got %s", switchedTenant)
	}
}
