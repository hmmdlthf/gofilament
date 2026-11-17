package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/hash"
)

func TestImpersonation(t *testing.T) {
	var events []string
	a := newAccountAppWith(t, func(a *accountApp, acc *filament.Accounts) {
		h, _ := hash.Make("bob-pass-123")
		a.users["bob@example.com"] = &account{name: "Bob", email: "bob@example.com", pwHash: h, verified: true}
		a.panelHook = func(p *filament.Panel) {
			p.Impersonation(filament.Impersonation{
				CanImpersonate: func(ctx *filament.Context, target auth.Authenticatable) bool {
					return ctx.User().GetAuthIdentifier() == "ann@example.com"
				},
				OnImpersonate: func(ctx *filament.Context, from, to auth.Authenticatable, leaving bool) {
					events = append(events, from.GetAuthIdentifier()+">"+to.GetAuthIdentifier()+map[bool]string{true: " left", false: ""}[leaving])
				},
			})
		}
	})

	// Bob may not impersonate.
	wb := newWebClient(t, a.handler)
	token := wb.csrfFrom(wb.do("GET", "/admin/login", nil).Body.String())
	wb.do("POST", "/admin/login", url.Values{"_token": {token}, "email": {"bob@example.com"}, "password": {"bob-pass-123"}})
	if r := formPost(wb, "/admin/profile", "/admin/impersonate/ann@example.com", url.Values{}); r.StatusCode != http.StatusForbidden {
		t.Fatalf("bob impersonated: %d", r.StatusCode)
	}

	w := newWebClient(t, a.handler)
	a.signIn(w, "secret-pass")
	if r := formPost(w, "/admin/profile", "/admin/impersonate/ann@example.com", url.Values{}); r.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("impersonate self: %d", r.StatusCode)
	}
	if r := formPost(w, "/admin/profile", "/admin/impersonate/bob@example.com", url.Values{}); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("impersonate: %d", r.StatusCode)
	}
	page := w.do("GET", "/admin", nil).Body.String()
	if !strings.Contains(page, "filament-impersonation-banner") || !strings.Contains(page, "bob@example.com") {
		t.Fatalf("no banner:\n%s", page)
	}
	// Bob's password cannot be changed by the impersonator.
	if r := formPost(w, "/admin/profile", "/admin/profile/password", url.Values{"current_password": {"bob-pass-123"}, "password": {"hijacked-1"}, "password_confirmation": {"hijacked-1"}}); r.StatusCode != http.StatusForbidden {
		t.Fatalf("password change while impersonating: %d", r.StatusCode)
	}
	// No nested impersonation.
	if r := formPost(w, "/admin/profile", "/admin/impersonate/ann@example.com", url.Values{}); r.StatusCode != http.StatusConflict {
		t.Fatalf("nested: %d", r.StatusCode)
	}
	if r := formPost(w, "/admin/profile", "/admin/impersonate-leave", url.Values{}); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("leave: %d", r.StatusCode)
	}
	page = w.do("GET", "/admin", nil).Body.String()
	if strings.Contains(page, "filament-impersonation-banner") {
		t.Fatal("banner after leaving")
	}
	if strings.Join(events, ",") != "ann@example.com>bob@example.com,ann@example.com>bob@example.com left" {
		t.Fatalf("events: %v", events)
	}
}
