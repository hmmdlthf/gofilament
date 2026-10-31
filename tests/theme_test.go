package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/colors"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func TestColorPaletteGenerator(t *testing.T) {
	// 1. Generate palette from standard 6-char hex
	pal := colors.FromHex("#6366f1") // Indigo
	if len(pal) != 11 {
		t.Fatalf("expected 11 shades in generated palette, got %d", len(pal))
	}
	for _, shade := range colors.Shades {
		hex := pal.Shade(shade)
		if hex == "" || !strings.HasPrefix(hex, "#") {
			t.Fatalf("expected valid hex for shade %d, got '%s'", shade, hex)
		}
	}
	if pal.Shade(500) != "#6366f1" {
		t.Fatalf("expected shade 500 to match base hex '#6366f1', got '%s'", pal.Shade(500))
	}

	// 2. Shorthand hex and hex without #
	palShort := colors.FromHex("#38f")
	if len(palShort) != 11 {
		t.Fatalf("expected 11 shades from shorthand hex, got %d", len(palShort))
	}
	palNoHash := colors.FromHex("10b981")
	if palNoHash.Shade(500) != "#10b981" {
		t.Fatalf("expected shade 500 to match '#10b981', got '%s'", palNoHash.Shade(500))
	}

	// 3. RGB generator
	palRGB := colors.FromRGB(239, 68, 68) // Red
	if len(palRGB) != 11 {
		t.Fatalf("expected 11 shades from RGB, got %d", len(palRGB))
	}
	if palRGB.Shade(500) != "#ef4444" {
		t.Fatalf("expected shade 500 to match '#ef4444', got '%s'", palRGB.Shade(500))
	}

	// 4. CSSVariables formatter
	css := pal.CSSVariables("primary")
	if !strings.Contains(css, "--primary-50: #") || !strings.Contains(css, "--primary-500: #6366f1;") || !strings.Contains(css, "--primary-950: #") {
		t.Fatalf("CSSVariables formatting incorrect:\n%s", css)
	}

	// 5. Built-in presets
	if colors.Amber.Shade(500) != "#f59e0b" {
		t.Fatalf("Amber preset mismatch: %s", colors.Amber.Shade(500))
	}
	if colors.Slate.Shade(900) != "#0f172a" {
		t.Fatalf("Slate preset mismatch: %s", colors.Slate.Shade(900))
	}
}

func TestPanelThemeColorConfiguration(t *testing.T) {
	p := filament.NewPanel("admin").
		Colors(map[string]any{
			"primary": colors.Indigo,
			"gray":    "zinc",
			"danger":  "#e11d48", // Rose-600
		}).
		Color("warning", colors.Amber)

	primaryPal := p.GetColor("primary")
	if primaryPal == nil || primaryPal.Shade(500) != colors.Indigo.Shade(500) {
		t.Fatalf("expected primary to be Indigo, got %v", primaryPal)
	}

	grayPal := p.GetColor("gray")
	if grayPal == nil || grayPal.Shade(500) != colors.Zinc.Shade(500) {
		t.Fatalf("expected gray to be Zinc, got %v", grayPal)
	}

	dangerPal := p.GetColor("danger")
	if dangerPal == nil || dangerPal.Shade(500) != "#e11d48" {
		t.Fatalf("expected danger to be generated from '#e11d48', got %v", dangerPal)
	}

	warningPal := p.GetColor("warning")
	if warningPal == nil || warningPal.Shade(500) != colors.Amber.Shade(500) {
		t.Fatalf("expected warning to be Amber, got %v", warningPal)
	}

	// Default fallback
	infoPal := p.GetColor("info")
	if infoPal == nil || infoPal.Shade(500) != colors.Sky.Shade(500) {
		t.Fatalf("expected fallback info to be Sky, got %v", infoPal)
	}
}

func TestPanelFontConfiguration(t *testing.T) {
	// 1. Auto Google Fonts URL generation
	pAuto := filament.NewPanel("admin").
		Font("Poppins")

	if pAuto.GetFontFamily() != "Poppins" {
		t.Fatalf("expected font family 'Poppins', got '%s'", pAuto.GetFontFamily())
	}
	if !strings.Contains(pAuto.GetFontURL(), "family=Poppins") {
		t.Fatalf("expected Google Fonts URL for Poppins, got '%s'", pAuto.GetFontURL())
	}

	// 2. Custom Font URL
	customURL := "https://fonts.cdnfonts.com/css/satoshi"
	pCustom := filament.NewPanel("admin").
		Font("Satoshi", customURL)

	if pCustom.GetFontFamily() != "Satoshi" {
		t.Fatalf("expected font family 'Satoshi', got '%s'", pCustom.GetFontFamily())
	}
	if pCustom.GetFontURL() != customURL {
		t.Fatalf("expected custom font URL '%s', got '%s'", customURL, pCustom.GetFontURL())
	}
}

func TestThemeRenderHooks(t *testing.T) {
	app := golaravel.New()
	p := filament.NewPanel("admin").
		Path("/admin").
		RenderHook(filament.PanelsRenderHookHeadEnd, func(ctx *filament.Context) string {
			return `<meta name="custom-meta" content="filament-test">`
		}).
		RenderHook(filament.PanelsRenderHookBodyEnd, func(ctx *filament.Context) string {
			return `<div id="custom-footer-analytics">Analytics Loaded</div>`
		}).
		RenderHook(filament.PanelsRenderHookTopbarStart, func(ctx *filament.Context) string {
			return `<span class="brand-badge">PRO</span>`
		}).
		RenderHook(filament.PanelsRenderHookSidebarNavStart, func(ctx *filament.Context) string {
			return `<div class="sidebar-greeting">Hello Admin</div>`
		})

	p.Register(app)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, `<meta name="custom-meta" content="filament-test">`) {
		t.Fatalf("expected head.end hook output in HTML")
	}
	if !strings.Contains(body, `<div id="custom-footer-analytics">Analytics Loaded</div>`) {
		t.Fatalf("expected body.end hook output in HTML")
	}
	if !strings.Contains(body, `<span class="brand-badge">PRO</span>`) {
		t.Fatalf("expected topbar.start hook output in HTML")
	}
	if !strings.Contains(body, `<div class="sidebar-greeting">Hello Admin</div>`) {
		t.Fatalf("expected sidebar.nav.start hook output in HTML")
	}
}

func TestDynamicThemeCSSVariablesAndCustomAssets(t *testing.T) {
	app := golaravel.New()
	p := filament.NewPanel("admin").
		Path("/admin").
		Font("DM Sans").
		Color("primary", colors.Indigo).
		Color("danger", "#dc2626").
		Theme("/css/brand-override.css").
		Script("/js/analytics.js")

	p.Register(app)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	body := rec.Body.String()

	// 1. Google Font preconnect and link
	if !strings.Contains(body, "family=DM+Sans") {
		t.Fatalf("expected Google Fonts link in head for DM Sans")
	}

	// 2. Embedded theme variables style block
	if !strings.Contains(body, `<style id="filament-theme-vars">`) {
		t.Fatalf("expected style block #filament-theme-vars in HTML head")
	}
	if !strings.Contains(body, "--font-sans: 'DM Sans'") {
		t.Fatalf("expected --font-sans variable in HTML")
	}
	if !strings.Contains(body, fmt.Sprintf("--primary-500: %s;", colors.Indigo.Shade(500))) {
		t.Fatalf("expected custom primary-500 color in HTML")
	}
	if !strings.Contains(body, "--danger-500: #dc2626;") {
		t.Fatalf("expected custom danger-500 color in HTML")
	}

	// 3. Custom stylesheets and scripts
	if !strings.Contains(body, `<link rel="stylesheet" href="/css/brand-override.css">`) {
		t.Fatalf("expected custom stylesheet link in HTML")
	}
	if !strings.Contains(body, `<script src="/js/analytics.js"></script>`) {
		t.Fatalf("expected custom script in HTML")
	}
}

func TestRESTAPISchemaThemeMetadata(t *testing.T) {
	app := golaravel.New()
	p := filament.NewPanel("admin").
		Path("/admin").
		Font("Inter").
		Color("primary", colors.Indigo).
		Theme("/css/custom.css")

	p.Register(app)

	req := httptest.NewRequest(http.MethodGet, "/admin/api/schema", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var schema map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&schema); err != nil {
		t.Fatalf("failed to decode schema JSON: %v", err)
	}

	theme, ok := schema["theme"].(map[string]any)
	if !ok || theme == nil {
		t.Fatalf("expected theme object in schema, got %v", schema["theme"])
	}

	if theme["font_family"] != "Inter" {
		t.Fatalf("expected theme font_family 'Inter', got '%v'", theme["font_family"])
	}

	themeColors, ok := theme["colors"].(map[string]any)
	if !ok {
		t.Fatalf("expected theme.colors object, got %v", theme["colors"])
	}

	primaryPal, ok := themeColors["primary"].(map[string]any)
	if !ok {
		t.Fatalf("expected theme.colors.primary palette, got %v", themeColors["primary"])
	}

	if primaryPal["500"] != colors.Indigo.Shade(500) {
		t.Fatalf("expected primary 500 to be %s, got %v", colors.Indigo.Shade(500), primaryPal["500"])
	}
}
