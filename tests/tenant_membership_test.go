package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

// The tenant loader only knows the signed-in user's tenants.
func newMembershipPanel(t *testing.T) (*golaravel.Engine, *filament.Panel) {
	t.Helper()
	app := golaravel.New()
	p := filament.NewPanel("admin").Path("/admin").Login().AuthValidator(loginValidator).
		Tenancy(&filament.TenantConfig{Enabled: true, CookieName: "filament_tenant"}).
		TenantLoader(func(ctx *filament.Context) ([]filament.Tenant, error) {
			if ctx.User() == nil {
				return nil, nil
			}
			return []filament.Tenant{
				&filament.TenantItem{Key: "acme", Name: "Acme"},
				&filament.TenantItem{Key: "globex", Name: "Globex"},
			}, nil
		}).
		CustomDashboard(func(ctx *filament.Context) error {
			return ctx.String(http.StatusOK, "tenant=%s", ctx.Tenant())
		})
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, p
}

func signIn(t *testing.T, w *webClient) {
	t.Helper()
	token := w.csrfFrom(w.do("GET", "/admin/login", nil).Body.String())
	if rec := w.do("POST", "/admin/login", url.Values{"_token": {token}, "email": {"admin@example.com"}, "password": {"secret"}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("login: %d", rec.Code)
	}
}

func TestTenantResolvedAfterUser(t *testing.T) {
	app, _ := newMembershipPanel(t)
	w := newWebClient(t, app)
	signIn(t, w)
	w.cookies["filament_tenant"] = &http.Cookie{Name: "filament_tenant", Value: "globex"}
	if body := w.do("GET", "/admin", nil).Body.String(); body != "tenant=globex" {
		t.Fatalf("tenant should come from the user's memberships: %q", body)
	}
	// A cookie naming a tenant the user is not a member of is ignored.
	w.cookies["filament_tenant"] = &http.Cookie{Name: "filament_tenant", Value: "initech"}
	if body := w.do("GET", "/admin", nil).Body.String(); body != "tenant=acme" {
		t.Fatalf("foreign tenant cookie must fall back to a member tenant: %q", body)
	}
}

func TestTenantSwitchRefusesUnknownTenant(t *testing.T) {
	app, _ := newMembershipPanel(t)
	w := newWebClient(t, app)
	signIn(t, w)
	if rec := w.do("GET", "/admin/tenant/switch?tenant=initech", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("switch to a foreign tenant: %d", rec.Code)
	}
	if _, ok := w.cookies["filament_tenant"]; ok {
		t.Fatal("tenant cookie set for a foreign tenant")
	}
	if rec := w.do("GET", "/admin/tenant/switch?tenant=globex", nil); rec.Code != http.StatusSeeOther || w.cookies["filament_tenant"].Value != "globex" {
		t.Fatalf("switch to own tenant: %d", rec.Code)
	}
}

func TestTenantSwitchFromAnotherSiteNeedsConfirmation(t *testing.T) {
	app, _ := newMembershipPanel(t)
	w := newWebClient(t, app)
	signIn(t, w)
	w.fetchSite = "cross-site"
	rec := w.do("GET", "/admin/tenant/switch?tenant=globex", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `method="POST"`) {
		t.Fatalf("cross-site switch must show a confirmation page: %d", rec.Code)
	}
	if _, set := w.cookies["filament_tenant"]; set {
		t.Fatal("a cross-site GET switched the tenant")
	}
}
