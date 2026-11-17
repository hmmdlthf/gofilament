package tests

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/auth"
)

// newRegistrationApp turns on sign-up and email verification; links mailed
// to new users are kept in links.
func newRegistrationApp(t *testing.T) (*accountApp, *[]string) {
	var links []string
	a := newAccountAppWith(t, func(a *accountApp, acc *filament.Accounts) {
		acc.Register = func(ctx *filament.Context, name, email, h string) (auth.Authenticatable, error) {
			if _, ok := a.users[email]; ok {
				return nil, filament.ErrEmailTaken
			}
			a.users[email] = &account{name: name, email: email, pwHash: h}
			return &filament.PanelUser{ID: email, Login: email}, nil
		}
		acc.EmailVerified = func(ctx *filament.Context, u auth.Authenticatable) (bool, error) {
			return a.users[u.GetAuthIdentifier()].verified, nil
		}
		acc.MarkEmailVerified = func(ctx *filament.Context, u auth.Authenticatable) error {
			a.users[u.GetAuthIdentifier()].verified = true
			return nil
		}
		acc.SendVerificationLink = func(ctx *filament.Context, u auth.Authenticatable, email, link string) error {
			links = append(links, link)
			return nil
		}
		acc.VerificationSecret = "verify-secret"
	})
	return a, &links
}

func TestWebRegistrationAndVerification(t *testing.T) {
	a, links := newRegistrationApp(t)
	w := newWebClient(t, a.handler)
	if !strings.Contains(w.do("GET", "/admin/login", nil).Body.String(), "/admin/register") {
		t.Fatal("login page has no sign-up link")
	}
	reg := func(email, pw string) *http.Response {
		return formPost(w, "/admin/register", "/admin/register", url.Values{
			"name": {"Bob"}, "email": {email}, "password": {pw}, "password_confirmation": {pw}})
	}
	if r := reg("ann@example.com", "long-password"); r.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(readBody(r), "already exists") {
		t.Fatalf("taken email: %d", r.StatusCode)
	}
	if r := reg("bob@example.com", "short"); r.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("short password: %d", r.StatusCode)
	}
	if r := reg("Bob@Example.com", "long-password"); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("register: %d", r.StatusCode)
	}
	if a.users["bob@example.com"] == nil || len(*links) != 1 {
		t.Fatalf("user or link missing: %v", *links)
	}

	// Signed in but unverified: only the verify page.
	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/email/verify" {
		t.Fatalf("dashboard before verifying: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	if rec := w.do("GET", "/admin/email/verify", nil); rec.Code != 200 {
		t.Fatalf("verify page: %d", rec.Code)
	}
	if r := formPost(w, "/admin/email/verify", "/admin/email/verification-notification", url.Values{}); r.StatusCode != http.StatusSeeOther || len(*links) != 2 {
		t.Fatalf("resend: %d %d", r.StatusCode, len(*links))
	}

	// A tampered link fails; the real one verifies.
	link, _ := url.Parse((*links)[1])
	q := link.Query()
	bad := url.Values{}
	for k, v := range q {
		bad[k] = v
	}
	bad.Set("id", "ann@example.com")
	if rec := w.do("GET", "/admin/email/verify/confirm?"+bad.Encode(), nil); rec.Code != http.StatusForbidden {
		t.Fatalf("tampered link: %d", rec.Code)
	}
	if rec := w.do("GET", link.RequestURI(), nil); rec.Code != 200 || !a.users["bob@example.com"].verified {
		t.Fatalf("verify: %d", rec.Code)
	}
	if rec := w.do("GET", "/admin", nil); rec.Code != 200 {
		t.Fatalf("dashboard after verifying: %d", rec.Code)
	}
}

func TestAPIRegistrationAndVerification(t *testing.T) {
	a, links := newRegistrationApp(t)
	code, out := a.api("POST", "/register", "", map[string]any{"name": "Cy", "email": "cy@example.com", "password": "long-password", "password_confirmation": "long-password"})
	if code != http.StatusCreated || out["token"] != "tok-cy@example.com" || out["email_verified"] != false {
		t.Fatalf("register: %d %v", code, out)
	}
	tok := out["token"].(string)
	if code, out := a.api("POST", "/register", "", map[string]any{"name": "Cy", "email": "cy@example.com", "password": "long-password", "password_confirmation": "long-password"}); code != http.StatusUnprocessableEntity || out["errors"] == nil {
		t.Fatalf("duplicate: %d %v", code, out)
	}
	if code, out := a.api("GET", "/schema", tok, nil); code != http.StatusForbidden || out["email_unverified"] != true {
		t.Fatalf("schema before verifying: %d %v", code, out)
	}
	if code, me := a.api("GET", "/me", tok, nil); code != 200 || me["email_verified"] != false {
		t.Fatalf("me: %d %v", code, me)
	}
	if code, _ := a.api("POST", "/email/verification-notification", tok, nil); code != 200 || len(*links) != 2 {
		t.Fatalf("resend: %d", code)
	}
	w := newWebClient(t, a.handler)
	link, _ := url.Parse((*links)[1])
	if rec := w.do("GET", link.RequestURI(), nil); rec.Code != 200 {
		t.Fatalf("verify: %d", rec.Code)
	}
	if code, _ := a.api("GET", "/schema", tok, nil); code != 200 {
		t.Fatalf("schema after verifying: %d", code)
	}
	_, boot := a.api("GET", "/boot", "", nil)
	feats := boot["auth"].(map[string]any)["accounts"].(map[string]any)
	if feats["registration"] != true || feats["email_verification"] != true {
		t.Fatalf("features: %v", feats)
	}
}

func TestNoRegistrationWithoutCallback(t *testing.T) {
	a := newAccountApp(t) // Registration() on, but no Register callback
	w := newWebClient(t, a.handler)
	if rec := w.do("GET", "/admin/register", nil); rec.Code == 200 {
		t.Fatal("register page without a Register callback")
	}
	if strings.Contains(w.do("GET", "/admin/login", nil).Body.String(), "/admin/register") {
		t.Fatal("sign-up link without a Register callback")
	}
}
