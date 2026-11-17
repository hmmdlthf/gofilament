package tests

import (
	"errors"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/hash"
	"github.com/golaravel/golaravel/pkg/session"
)

type account struct {
	name, email, pwHash, secret string
}

type accountApp struct {
	t       *testing.T
	users   map[string]*account // by email (= login id)
	mails   []string
	handler http.Handler
	panel   *filament.Panel
}

func newAccountApp(t *testing.T) *accountApp {
	h, _ := hash.Make("secret-pass")
	a := &accountApp{t: t, users: map[string]*account{
		"ann@example.com": {name: "Ann", email: "ann@example.com", pwHash: h},
	}}
	user := func(id string) (auth.Authenticatable, error) {
		if _, ok := a.users[id]; ok {
			return &filament.PanelUser{ID: id, Login: id}, nil
		}
		return nil, errors.New("not found")
	}
	acct := func(u auth.Authenticatable) *account { return a.users[u.GetAuthIdentifier()] }
	mgr := session.NewManager(session.Config{SecretKey: "accounts-test-key"})
	app := golaravel.New()
	a.panel = filament.NewPanel("admin").Path("/admin").Login().Profile().PasswordReset().
		AuthValidator(func(ctx *filament.Context, login, pw string) (auth.Authenticatable, string, error) {
			if u, ok := a.users[login]; ok && hash.Check(pw, u.pwHash) {
				return &filament.PanelUser{ID: login, Login: login}, "", nil
			}
			return nil, "", errors.New("bad credentials")
		}).
		Sessions(mgr, func(ctx *filament.Context, id string) (auth.Authenticatable, error) { return user(id) }).
		Accounts(filament.Accounts{
			Profile: func(ctx *filament.Context, u auth.Authenticatable) (string, string, error) {
				return acct(u).name, acct(u).email, nil
			},
			UpdateProfile: func(ctx *filament.Context, u auth.Authenticatable, name, email string) error {
				acct(u).name = name
				return nil
			},
			PasswordHash:   func(ctx *filament.Context, u auth.Authenticatable) (string, error) { return acct(u).pwHash, nil },
			UpdatePassword: func(ctx *filament.Context, u auth.Authenticatable, h string) error { acct(u).pwHash = h; return nil },
			FindByEmail:    func(ctx *filament.Context, email string) (auth.Authenticatable, error) { return user(email) },
			SendResetLink: func(ctx *filament.Context, u auth.Authenticatable, email, link string) error {
				a.mails = append(a.mails, link)
				return nil
			},
			TwoFactorSecret:     func(ctx *filament.Context, u auth.Authenticatable) (string, error) { return acct(u).secret, nil },
			SaveTwoFactorSecret: func(ctx *filament.Context, u auth.Authenticatable, s string) error { acct(u).secret = s; return nil },
		})
	if err := a.panel.Register(app); err != nil {
		t.Fatal(err)
	}
	a.handler = app
	return a
}

func (a *accountApp) signIn(w *webClient, pw string) *http.Response {
	token := w.csrfFrom(w.do("GET", "/admin/login", nil).Body.String())
	return w.do("POST", "/admin/login", url.Values{"_token": {token}, "email": {"ann@example.com"}, "password": {pw}}).Result()
}

func formPost(w *webClient, page, path string, v url.Values) *http.Response {
	v.Set("_token", w.csrfFrom(w.do("GET", page, nil).Body.String()))
	return w.do("POST", path, v).Result()
}

func TestProfileAndPasswordChange(t *testing.T) {
	a := newAccountApp(t)
	w := newWebClient(t, a.handler)
	if rec := w.do("GET", "/admin/profile", nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("guest profile: %d", rec.Code)
	}
	if r := a.signIn(w, "secret-pass"); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("sign in: %d", r.StatusCode)
	}
	dash := w.do("GET", "/admin", nil).Body.String()
	if !strings.Contains(dash, `href="/admin/profile"`) || !strings.Contains(dash, `href="/admin/logout"`) || !strings.Contains(dash, "Ann") {
		t.Fatal("topbar user menu missing profile / sign out / name")
	}

	// A second device stays signed in until the password changes.
	other := newWebClient(t, a.handler)
	a.signIn(other, "secret-pass")

	if r := formPost(w, "/admin/profile", "/admin/profile", url.Values{"name": {"Ann Lee"}, "email": {"ann@example.com"}}); r.StatusCode != http.StatusSeeOther || a.users["ann@example.com"].name != "Ann Lee" {
		t.Fatalf("profile save: %d", r.StatusCode)
	}
	if r := formPost(w, "/admin/profile", "/admin/profile/password", url.Values{"current_password": {"wrong"}, "password": {"new-pass-123"}, "password_confirmation": {"new-pass-123"}}); r.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("wrong current password: %d", r.StatusCode)
	}
	if r := formPost(w, "/admin/profile", "/admin/profile/password", url.Values{"current_password": {"secret-pass"}, "password": {"new-pass-123"}, "password_confirmation": {"nope"}}); r.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("mismatched confirmation: %d", r.StatusCode)
	}
	r := formPost(w, "/admin/profile", "/admin/profile/password", url.Values{"current_password": {"secret-pass"}, "password": {"new-pass-123"}, "password_confirmation": {"new-pass-123"}})
	if r.StatusCode != http.StatusSeeOther || !hash.Check("new-pass-123", a.users["ann@example.com"].pwHash) {
		t.Fatalf("change password: %d", r.StatusCode)
	}
	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusOK {
		t.Fatal("current device signed out by its own password change")
	}
	if rec := other.do("GET", "/admin", nil); rec.Code != http.StatusSeeOther {
		t.Fatal("other device still signed in after password change")
	}
}

func TestForgotAndResetPassword(t *testing.T) {
	a := newAccountApp(t)
	w := newWebClient(t, a.handler)
	if !strings.Contains(w.do("GET", "/admin/login", nil).Body.String(), "/admin/forgot-password") {
		t.Fatal("no forgot-password link on the sign-in page")
	}
	unknown := formPost(w, "/admin/forgot-password", "/admin/forgot-password", url.Values{"email": {"nobody@example.com"}})
	known := formPost(w, "/admin/forgot-password", "/admin/forgot-password", url.Values{"email": {"ann@example.com"}})
	if unknown.StatusCode != 200 || known.StatusCode != 200 || len(a.mails) != 1 {
		t.Fatalf("forgot: %d %d mails=%d", unknown.StatusCode, known.StatusCode, len(a.mails))
	}
	link, _ := url.Parse(a.mails[0])
	if link.Host != "example.com" || link.Path != "/admin/reset-password" {
		t.Fatalf("link: %s", a.mails[0])
	}
	page := w.do("GET", link.RequestURI(), nil).Body.String()
	v := url.Values{"_token": {w.csrfFrom(page)}, "email": {"ann@example.com"}, "token": {link.Query().Get("token") + "x"}, "password": {"brand-new-1"}, "password_confirmation": {"brand-new-1"}}
	if rec := w.do("POST", "/admin/reset-password", v); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad token accepted: %d", rec.Code)
	}
	v.Set("token", link.Query().Get("token"))
	if rec := w.do("POST", "/admin/reset-password", v); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/login?status=reset" {
		t.Fatalf("reset: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	if rec := w.do("POST", "/admin/reset-password", v); rec.Code != http.StatusUnprocessableEntity {
		t.Fatal("reset link usable twice")
	}
	if r := a.signIn(w, "brand-new-1"); r.Header.Get("Location") != "/admin" {
		t.Fatal("cannot sign in with the reset password")
	}
}

var secretRe = regexp.MustCompile(`data-otpauth="[^"]*">([A-Z2-7]+)</p>`)

func TestTwoFactorSetupAndSignIn(t *testing.T) {
	a := newAccountApp(t)
	totp := auth.NewTOTPManager()
	w := newWebClient(t, a.handler)
	a.signIn(w, "secret-pass")

	page := formPost(w, "/admin/profile", "/admin/profile/two-factor/enable", url.Values{"current_password": {"secret-pass"}})
	body := readBody(page)
	m := secretRe.FindStringSubmatch(html.UnescapeString(body))
	if m == nil {
		t.Fatalf("no secret on the setup page:\n%s", body)
	}
	secret := m[1]
	if r := formPost(w, "/admin/profile", "/admin/profile/two-factor/confirm", url.Values{"code": {"000000"}}); r.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("wrong code accepted: %d", r.StatusCode)
	}
	code, _ := totp.GenerateCode(secret)
	if r := formPost(w, "/admin/profile", "/admin/profile/two-factor/confirm", url.Values{"code": {code}}); r.StatusCode != http.StatusSeeOther || a.users["ann@example.com"].secret != secret {
		t.Fatalf("confirm: %d", r.StatusCode)
	}

	// Signing in now needs a code.
	w2 := newWebClient(t, a.handler)
	if r := a.signIn(w2, "secret-pass"); r.Header.Get("Location") != "/admin/two-factor-challenge" {
		t.Fatalf("no challenge: %s", r.Header.Get("Location"))
	}
	if rec := w2.do("GET", "/admin", nil); rec.Code != http.StatusSeeOther {
		t.Fatal("password alone signed in")
	}
	if r := formPost(w2, "/admin/two-factor-challenge", "/admin/two-factor-challenge", url.Values{"code": {code}}); r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("used code accepted again: %d", r.StatusCode)
	}
	// The authenticator is re-enrolled (a new secret), so a fresh code works.
	secret, _ = totp.GenerateSecret()
	a.users["ann@example.com"].secret = secret
	next, _ := totp.GenerateCode(secret)
	if r := formPost(w2, "/admin/two-factor-challenge", "/admin/two-factor-challenge", url.Values{"code": {next}}); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("challenge: %d", r.StatusCode)
	}
	if rec := w2.do("GET", "/admin", nil); rec.Code != http.StatusOK {
		t.Fatalf("after challenge: %d", rec.Code)
	}

	// Turning it off needs the password.
	if r := formPost(w2, "/admin/profile", "/admin/profile/two-factor/disable", url.Values{"current_password": {"wrong"}}); r.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("disabled without password: %d", r.StatusCode)
	}
	if r := formPost(w2, "/admin/profile", "/admin/profile/two-factor/disable", url.Values{"current_password": {"secret-pass"}}); r.StatusCode != http.StatusSeeOther || a.users["ann@example.com"].secret != "" {
		t.Fatalf("disable: %d", r.StatusCode)
	}
}

func readBody(r *http.Response) string {
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := r.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			return sb.String()
		}
	}
}
