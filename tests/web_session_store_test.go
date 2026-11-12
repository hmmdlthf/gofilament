package tests

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
	_ "github.com/mattn/go-sqlite3"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/sqlstore"
)

func TestWebSessionsInTheDatabase(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := sqlstore.New(db, sqlstore.SQLite)
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	disabled := map[string]bool{}

	// boot builds a fresh panel and engine, like a restarted process.
	boot := func() *golaravel.Engine {
		p := filament.NewPanel("dbsess").Path("/admin").Login().LoginThrottle(0, 0).
			AuthValidator(func(ctx *filament.Context, username, password string) (auth.Authenticatable, string, error) {
				if password == "right" {
					return &filament.PanelUser{ID: username, Login: username}, "", nil
				}
				return nil, "", errors.New("bad")
			}).
			WebSessionsUsing(store.WebSessions(), func(ctx *filament.Context, id string) (auth.Authenticatable, error) {
				if disabled[id] {
					return nil, errors.New("disabled")
				}
				return &filament.PanelUser{ID: id, Login: id}, nil
			})
		app := golaravel.New()
		if err := p.Register(app); err != nil {
			t.Fatal(err)
		}
		return app
	}

	app := boot()
	getRec := httptest.NewRecorder()
	app.ServeHTTP(getRec, httptest.NewRequest("GET", "/admin/login", nil))
	form := url.Values{"email": {"ann"}, "password": {"right"}}
	var cookies []*http.Cookie
	for _, c := range getRec.Result().Cookies() {
		cookies = append(cookies, c)
		if strings.Contains(strings.ToLower(c.Name), "csrf") {
			form.Set("_token", c.Value)
		}
	}
	req := httptest.NewRequest("POST", "/admin/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	var session *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if strings.Contains(c.Name, "session") && c.Value != "" {
			session = c
		}
	}
	if session == nil {
		t.Fatal("no session cookie")
	}

	// the database holds a hash, never the cookie value
	var stored string
	_ = db.QueryRow(`SELECT id FROM gofilament_sessions`).Scan(&stored)
	if stored == "" || stored == session.Value {
		t.Fatalf("stored session id must be a hash of the cookie: %q", stored)
	}

	dashboard := func(app *golaravel.Engine) int {
		r := httptest.NewRequest("GET", "/admin", nil)
		r.AddCookie(session)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, r)
		return rec.Code
	}
	if code := dashboard(boot()); code != 200 {
		t.Fatalf("the session survives a restart: %d", code)
	}
	disabled["ann"] = true
	if code := dashboard(boot()); code == 200 {
		t.Fatal("a user the loader rejects is signed out")
	}
}
