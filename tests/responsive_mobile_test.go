package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func TestMobileFirst_ResponsiveSidebarAndTopbar(t *testing.T) {
	app := golaravel.Default()
	panel := filament.NewPanel("admin").
		Path("/admin").
		BrandName("Mobile Test Store")

	if err := panel.Register(app); err != nil {
		t.Fatalf("failed to register panel: %v", err)
	}

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// 1. Verify mobile hamburger menu button in topbar
	if !strings.Contains(body, "filament-mobile-menu-btn") {
		t.Errorf("expected filament-mobile-menu-btn in topbar for mobile screens")
	}

	// 2. Verify mobile sidebar close button in sidebar
	if !strings.Contains(body, "filament-sidebar-close-btn") {
		t.Errorf("expected filament-sidebar-close-btn in sidebar for mobile screens")
	}

	// 3. Verify sidebar has id="filament-sidebar"
	if !strings.Contains(body, `id="filament-sidebar"`) {
		t.Errorf("expected id=\"filament-sidebar\" on aside element")
	}

	// 4. Verify mobile drawer backdrop overlay
	if !strings.Contains(body, "filament-sidebar-backdrop") {
		t.Errorf("expected filament-sidebar-backdrop element in body")
	}
}

func TestMobileFirst_CSSResponsiveRules(t *testing.T) {
	app := golaravel.Default()
	panel := filament.NewPanel("admin").Path("/admin")
	if err := panel.Register(app); err != nil {
		t.Fatalf("failed to register panel: %v", err)
	}

	// Fetch embedded filament.css
	req := httptest.NewRequest("GET", "/admin/assets/filament.css", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for filament.css, got %d", w.Code)
	}

	css := w.Body.String()

	// 1. Verify mobile drawer media queries
	if !strings.Contains(css, "@media (max-width: 1024px)") {
		t.Errorf("expected @media (max-width: 1024px) for mobile sidebar drawer in CSS")
	}
	if !strings.Contains(css, "transform: translateX(-100%)") {
		t.Errorf("expected off-canvas drawer transform in CSS")
	}

	// 2. Verify mobile phone grid-collapse media queries
	if !strings.Contains(css, "@media (max-width: 640px)") {
		t.Errorf("expected @media (max-width: 640px) for phone screen single-column fallback in CSS")
	}

	// 3. Verify tablet grid media queries
	if !strings.Contains(css, "@media (min-width: 641px) and (max-width: 1024px)") {
		t.Errorf("expected tablet media query in CSS")
	}

	// 4. Verify responsive table horizontal pan
	if !strings.Contains(css, "min-width: 44rem") {
		t.Errorf("expected min-width for table in CSS to prevent crushed columns on mobile")
	}
}

func TestMobileFirst_JSSidebarController(t *testing.T) {
	app := golaravel.Default()
	panel := filament.NewPanel("admin").Path("/admin")
	if err := panel.Register(app); err != nil {
		t.Fatalf("failed to register panel: %v", err)
	}

	// Fetch embedded filament.js
	req := httptest.NewRequest("GET", "/admin/assets/filament.js", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for filament.js, got %d", w.Code)
	}

	js := w.Body.String()

	// Verify Filament.sidebar controller
	if !strings.Contains(js, "sidebar:") {
		t.Errorf("expected Filament.sidebar in filament.js")
	}
	if !strings.Contains(js, "toggle: function()") || !strings.Contains(js, "open: function()") || !strings.Contains(js, "close: function()") {
		t.Errorf("expected toggle, open, and close methods on Filament.sidebar")
	}
	if !strings.Contains(js, "sidebar-mobile-open") {
		t.Errorf("expected body lock class sidebar-mobile-open in filament.js")
	}
}
