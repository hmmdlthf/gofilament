package tests

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func corsRequest(t *testing.T, p *filament.Panel, origin string) (allow, creds string) {
	t.Helper()
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("OPTIONS", "http://panel.example.com/admin/api/schema", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec.Header().Get("Access-Control-Allow-Origin"), rec.Header().Get("Access-Control-Allow-Credentials")
}

func newCORSPanel(t *testing.T) *filament.Panel {
	return filament.NewPanel("admin").Path("/admin").Engine(filament.EngineFlutter).Flutter(t.TempDir())
}

func TestCORSDefaultIsSameOriginOnly(t *testing.T) {
	if allow, creds := corsRequest(t, newCORSPanel(t), "https://evil.example"); allow != "" || creds != "" {
		t.Fatalf("foreign origin must not be reflected: allow=%q creds=%q", allow, creds)
	}
	if allow, creds := corsRequest(t, newCORSPanel(t), "https://panel.example.com"); allow != "https://panel.example.com" || creds != "true" {
		t.Fatalf("same origin: allow=%q creds=%q", allow, creds)
	}
}

func TestCORSAllowList(t *testing.T) {
	p := newCORSPanel(t).CORSAllowedOrigins("https://app.example.com")
	if allow, creds := corsRequest(t, p, "https://app.example.com"); allow != "https://app.example.com" || creds != "true" {
		t.Fatalf("listed origin: allow=%q creds=%q", allow, creds)
	}
	p = newCORSPanel(t).CORSAllowedOrigins("https://app.example.com")
	if allow, _ := corsRequest(t, p, "https://other.example.com"); allow != "" {
		t.Fatalf("unlisted origin reflected: %q", allow)
	}

	p = newCORSPanel(t).CORSAllowOriginFunc(func(o string) bool { return strings.HasPrefix(o, "http://localhost:") })
	if allow, creds := corsRequest(t, p, "http://localhost:5173"); allow != "http://localhost:5173" || creds != "true" {
		t.Fatalf("origin func: allow=%q creds=%q", allow, creds)
	}
}

func TestCORSWildcardHasNoCredentials(t *testing.T) {
	p := newCORSPanel(t).CORSAllowedOrigins("*")
	if allow, creds := corsRequest(t, p, "https://any.example"); allow != "*" || creds != "" {
		t.Fatalf("wildcard: allow=%q creds=%q", allow, creds)
	}
}
