package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
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

	// 4. Verify responsive table cards on mobile & tablet (Zero Horizontal Scrolling)
	if !strings.Contains(css, "attr(data-label)") {
		t.Errorf("expected attr(data-label) for responsive card column labels in CSS")
	}
	if !strings.Contains(css, "Mobile Responsive Table to Stacked Cards") {
		t.Errorf("expected Mobile Responsive Table card rules in CSS")
	}
	if !strings.Contains(css, "Tablet Responsive Table to 2-Column Card Grid") {
		t.Errorf("expected Tablet Responsive Table card grid rules in CSS")
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

func TestMobileFirst_TableCardDataLabels(t *testing.T) {
	app := golaravel.Default()
	panel := filament.NewPanel("admin").Path("/admin")

	type Product struct {
		ID    string
		Name  string
		Price string
	}

	res := filament.NewResource[*Product]("products").
		RecordID(func(p *Product) string { return p.ID })

	res.Table(func(tbl *tables.Table[*Product]) *tables.Table[*Product] {
		return tbl.Columns(
			tables.TextColumn[*Product]("Name").Label("Product Name"),
			tables.TextColumn[*Product]("Price").Label("Unit Price"),
		)
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Product, int, error) {
		return []*Product{
			{ID: "1", Name: "Wireless Keyboard", Price: "$49.99"},
		}, 1, nil
	})

	panel.Resource(res)
	if err := panel.Register(app); err != nil {
		t.Fatalf("failed to register panel: %v", err)
	}

	req := httptest.NewRequest("GET", "/admin/products", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Verify data-label attributes are injected on td elements for mobile card display
	if !strings.Contains(body, `data-label="Product Name"`) {
		t.Errorf("expected data-label=\"Product Name\" on td element, got body: %s", body)
	}
	if !strings.Contains(body, `data-label="Unit Price"`) {
		t.Errorf("expected data-label=\"Unit Price\" on td element")
	}
}
