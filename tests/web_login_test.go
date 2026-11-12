package tests

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

// webClient is a tiny cookie-keeping browser for golaravel apps.
type webClient struct {
	t       *testing.T
	app     http.Handler
	cookies map[string]*http.Cookie
}

func newWebClient(t *testing.T, app http.Handler) *webClient {
	return &webClient{t: t, app: app, cookies: map[string]*http.Cookie{}}
}

func (w *webClient) do(method, path string, form url.Values) *httptest.ResponseRecorder {
	w.t.Helper()
	var req *http.Request
	if form != nil {
		req = httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequest(method, "http://example.com"+path, nil)
	}
	for _, c := range w.cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	w.app.ServeHTTP(rec, req)
	for _, c := range rec.Result().Cookies() {
		if c.MaxAge < 0 {
			delete(w.cookies, c.Name)
		} else {
			w.cookies[c.Name] = c
		}
	}
	return rec
}

var csrfInput = regexp.MustCompile(`name="_token" value="([0-9a-f]+)"`)

func (w *webClient) csrfFrom(body string) string {
	w.t.Helper()
	m := csrfInput.FindStringSubmatch(body)
	if m == nil {
		w.t.Fatalf("no CSRF token in page:\n%s", body)
	}
	return m[1]
}

func loginValidator(ctx *filament.Context, username, password string) (auth.Authenticatable, string, error) {
	if username == "admin@example.com" && password == "secret" {
		return &filament.PanelUser{ID: "1", Login: "admin@example.com"}, "tok-admin", nil
	}
	return nil, "", errors.New("bad credentials")
}

func TestWebLoginStartsSession(t *testing.T) {
	app := golaravel.New()
	p := filament.NewPanel("admin").Path("/admin").Login().AuthValidator(loginValidator)
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	w := newWebClient(t, app)

	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/login" {
		t.Fatalf("anonymous dashboard: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	page := w.do("GET", "/admin/login", nil)
	token := w.csrfFrom(page.Body.String())

	// Missing CSRF token is refused.
	if rec := w.do("POST", "/admin/login", url.Values{"email": {"admin@example.com"}, "password": {"secret"}}); rec.Code != http.StatusForbidden {
		t.Fatalf("login without CSRF token: %d", rec.Code)
	}
	if _, ok := w.cookies[p.SessionCookieName()]; ok {
		t.Fatal("session started without CSRF token")
	}
	// Wrong password re-renders the form.
	if rec := w.do("POST", "/admin/login", url.Values{"_token": {token}, "email": {"admin@example.com"}, "password": {"nope"}}); rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "do not match") {
		t.Fatalf("bad password: %d", rec.Code)
	}

	rec := w.do("POST", "/admin/login", url.Values{"_token": {token}, "email": {"admin@example.com"}, "password": {"secret"}})
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin" {
		t.Fatalf("login: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	sess := w.cookies[p.SessionCookieName()]
	if sess == nil || !sess.HttpOnly || sess.SameSite != http.SameSiteLaxMode || sess.Path != "/admin" {
		t.Fatalf("session cookie: %+v", sess)
	}
	if sess.Value == "tok-admin" {
		t.Fatal("without a TokenValidator the cookie must be an opaque session id")
	}

	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusOK {
		t.Fatalf("dashboard after login: %d", rec.Code)
	}
	// The API keeps using bearer tokens only.
	if rec := w.do("GET", "/admin/api/me", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("API must not accept the web cookie: %d", rec.Code)
	}

	old := *sess
	w.do("GET", "/admin/logout", nil)
	if _, ok := w.cookies[p.SessionCookieName()]; ok {
		t.Fatal("logout did not clear the session cookie")
	}
	w.cookies[old.Name] = &old
	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("old session still valid after logout: %d", rec.Code)
	}
}

func TestWebLoginUsesTokenValidatorWhenSet(t *testing.T) {
	app := golaravel.New()
	revoked := false
	p := filament.NewPanel("admin").Path("/admin").Login().AuthValidator(loginValidator).
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			if token == "tok-admin" && !revoked {
				return &filament.PanelUser{ID: "1", Login: "admin@example.com"}, nil
			}
			return nil, errors.New("invalid")
		})
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	w := newWebClient(t, app)
	token := w.csrfFrom(w.do("GET", "/admin/login", nil).Body.String())
	w.do("POST", "/admin/login", url.Values{"_token": {token}, "email": {"admin@example.com"}, "password": {"secret"}})
	if c := w.cookies[p.SessionCookieName()]; c == nil || c.Value != "tok-admin" {
		t.Fatalf("cookie should carry the validator token: %+v", c)
	}
	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusOK {
		t.Fatalf("dashboard: %d", rec.Code)
	}
	revoked = true
	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("revoked token still accepted: %d", rec.Code)
	}
}
