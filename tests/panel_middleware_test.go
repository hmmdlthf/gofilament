package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func TestPanelMiddlewareAppliesToPanelRoutes(t *testing.T) {
	app := golaravel.New()
	var hits []string
	mw := func(c *gcontext.Context) error {
		hits = append(hits, c.Request.URL.Path)
		if c.Request.Header.Get("X-Block") != "" {
			c.Abort()
			return c.String(http.StatusTeapot, "blocked")
		}
		return nil
	}
	if err := filament.NewPanel("admin").Path("/admin").Middleware(mw).Register(app); err != nil {
		t.Fatal(err)
	}
	app.Router.GET("/elsewhere", func(c *gcontext.Context) error { return c.String(200, "ok") })
	app.Router.GET("/admin-other", func(c *gcontext.Context) error { return c.String(200, "ok") })

	do := func(method, path string, block bool) int {
		req := httptest.NewRequest(method, "http://example.com"+path, nil)
		if block {
			req.Header.Set("X-Block", "1")
		}
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
		return rec.Code
	}

	for _, path := range []string{"/admin", "/admin/api/schema", "/admin/api/version"} {
		hits = nil
		if code := do("GET", path, true); code != http.StatusTeapot {
			t.Fatalf("%s: middleware did not block, got %d", path, code)
		}
		if len(hits) != 1 {
			t.Fatalf("%s: middleware ran %d times", path, len(hits))
		}
	}

	hits = nil
	if code := do("GET", "/admin", false); code != http.StatusOK {
		t.Fatalf("dashboard with passing middleware: %d", code)
	}

	// Assets, preflight and routes outside the panel are not covered.
	for _, tc := range []struct{ method, path string }{
		{"GET", "/admin/assets/filament.css"},
		{"OPTIONS", "/admin/api/schema"},
		{"GET", "/elsewhere"},
		{"GET", "/admin-other"},
	} {
		hits = nil
		if code := do(tc.method, tc.path, true); code == http.StatusTeapot || len(hits) != 0 {
			t.Fatalf("%s %s: middleware should not run (code %d, hits %v)", tc.method, tc.path, code, hits)
		}
	}
}

func TestPanelMiddlewareFlutterCoversAPIOnly(t *testing.T) {
	app := golaravel.New()
	ran := 0
	mw := func(c *gcontext.Context) error { ran++; return nil }
	p := filament.NewPanel("admin").Path("/admin").Engine(filament.EngineFlutter).Flutter(t.TempDir()).Middleware(mw)
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/admin/api/version", "/admin"} {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, httptest.NewRequest("GET", "http://example.com"+path, nil))
	}
	if ran != 1 {
		t.Fatalf("flutter panel: middleware should run for /api only, ran %d", ran)
	}
}
