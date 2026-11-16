package tests

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/database"
	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/session"
)

func TestWebLoginOnGolaravelSessions(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	drv, err := session.NewSQLDriver(db, database.SQLite)
	if err != nil {
		t.Fatal(err)
	}
	if err := drv.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := session.NewManager(session.Config{Driver: drv, SecretKey: "test-app-key", CookieName: "app_session", Lifetime: time.Hour})

	disabled := false
	users := func(ctx *filament.Context, id string) (auth.Authenticatable, error) {
		if id == "admin@example.com" && !disabled {
			return &filament.PanelUser{ID: "1", Login: id}, nil
		}
		return nil, errors.New("gone")
	}
	newApp := func() http.Handler {
		app := golaravel.New()
		p := filament.NewPanel("admin").Path("/admin").Login().AuthValidator(loginValidator).Sessions(mgr, users)
		if err := p.Register(app); err != nil {
			t.Fatal(err)
		}
		return app
	}
	app := newApp()
	w := newWebClient(t, app)

	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("guest dashboard: %d", rec.Code)
	}
	token := w.csrfFrom(w.do("GET", "/admin/login", nil).Body.String())
	guestCookie := w.cookies["app_session"]
	if guestCookie == nil {
		t.Fatal("no golaravel session cookie on the login page")
	}
	if rec := w.do("POST", "/admin/login", url.Values{"email": {"admin@example.com"}, "password": {"secret"}}); rec.Code != http.StatusForbidden {
		t.Fatalf("login without CSRF: %d", rec.Code)
	}
	rec := w.do("POST", "/admin/login", url.Values{"_token": {token}, "email": {"admin@example.com"}, "password": {"secret"}})
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin" {
		t.Fatalf("login: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	if w.cookies["app_session"].Value == guestCookie.Value {
		t.Fatal("session id not regenerated on login")
	}
	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusOK {
		t.Fatalf("dashboard after login: %d", rec.Code)
	}
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id = 'admin@example.com'").Scan(&n)
	if n != 1 {
		t.Fatalf("sessions linked to the user: %d", n)
	}

	// The session survives a restart (new app, same database and key).
	w.app = newApp()
	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusOK {
		t.Fatalf("after restart: %d", rec.Code)
	}

	// The old CSRF token is dead after login; take the new one and log out.
	page := w.do("GET", "/admin/logout", nil) // same-origin GET logs out
	if page.Code != http.StatusSeeOther {
		t.Fatalf("logout: %d", page.Code)
	}
	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("still signed in after logout: %d", rec.Code)
	}
	_ = db.QueryRow("SELECT COUNT(*) FROM sessions WHERE user_id = 'admin@example.com'").Scan(&n)
	if n != 0 {
		t.Fatalf("logged-out session kept: %d", n)
	}

	// A disabled user loses access on the next request.
	token = w.csrfFrom(w.do("GET", "/admin/login", nil).Body.String())
	w.do("POST", "/admin/login", url.Values{"_token": {token}, "email": {"admin@example.com"}, "password": {"secret"}})
	disabled = true
	if rec := w.do("GET", "/admin", nil); rec.Code != http.StatusSeeOther {
		t.Fatalf("disabled user still in: %d", rec.Code)
	}
}

func TestPanelBearerGuard(t *testing.T) {
	jwt := auth.NewJWTManager("bearer-guard-secret-32-bytes-long!", time.Hour)
	tok, _ := jwt.GenerateToken("admin@example.com")
	app := golaravel.New()
	p := filament.NewPanel("admin").Path("/admin").BearerGuard(auth.BearerGuardFunc(jwt))
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	req := func(h string) int {
		r, _ := http.NewRequest("GET", "http://example.com/admin/api/schema", nil)
		if h != "" {
			r.Header.Set("Authorization", h)
		}
		rec := &recorder{header: http.Header{}}
		app.ServeHTTP(rec, r)
		return rec.code
	}
	if c := req(""); c != http.StatusUnauthorized {
		t.Fatalf("no token: %d", c)
	}
	if c := req("Bearer " + tok); c != http.StatusOK {
		t.Fatalf("valid token: %d", c)
	}
}

type recorder struct {
	header http.Header
	code   int
}

func (r *recorder) Header() http.Header { return r.header }
func (r *recorder) Write(b []byte) (int, error) {
	if r.code == 0 {
		r.code = 200
	}
	return len(b), nil
}
func (r *recorder) WriteHeader(c int) { r.code = c }
