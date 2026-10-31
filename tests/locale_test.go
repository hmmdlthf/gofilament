package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func TestPanelLocaleConfiguration(t *testing.T) {
	p := filament.NewPanel("admin").
		Locales("en", "ar", "fr", "es").
		DefaultLocale("ar").
		LocaleLabels(map[string]string{
			"en": "English (US)",
			"ar": "العربية الفصحى",
		}).
		LocaleFlags(map[string]string{
			"en": "🗽",
			"ar": "🇸🇦",
		})

	locales := p.GetLocales()
	if len(locales) != 4 {
		t.Fatalf("expected 4 locales, got %d", len(locales))
	}

	if p.GetDefaultLocale() != "ar" {
		t.Fatalf("expected default locale 'ar', got '%s'", p.GetDefaultLocale())
	}

	if p.GetLocaleLabel("en") != "English (US)" {
		t.Fatalf("expected custom label 'English (US)', got '%s'", p.GetLocaleLabel("en"))
	}

	// Uncustomized fallback from default dictionary
	if p.GetLocaleLabel("fr") != "Français" {
		t.Fatalf("expected fallback label 'Français', got '%s'", p.GetLocaleLabel("fr"))
	}

	if p.GetLocaleFlag("en") != "🗽" {
		t.Fatalf("expected custom flag '🗽', got '%s'", p.GetLocaleFlag("en"))
	}

	if p.GetLocaleFlag("fr") != "🇫🇷" {
		t.Fatalf("expected default flag '🇫🇷', got '%s'", p.GetLocaleFlag("fr"))
	}
}

func TestContextLocaleAndRTLHelpers(t *testing.T) {
	p := filament.NewPanel("admin").
		Locales("en", "ar")

	// 1. English LTR context
	reqEn := httptest.NewRequest(http.MethodGet, "/admin?lang=en", nil)
	recEn := httptest.NewRecorder()
	gcEn := newTestContext(recEn, reqEn)
	fCtxEn := filament.NewContext(gcEn, p)

	if fCtxEn.Locale() != "en" {
		t.Fatalf("expected locale 'en', got '%s'", fCtxEn.Locale())
	}
	if fCtxEn.Direction() != "ltr" {
		t.Fatalf("expected direction 'ltr', got '%s'", fCtxEn.Direction())
	}
	if fCtxEn.IsRTL() {
		t.Fatalf("expected IsRTL to be false for 'en'")
	}

	// 2. Arabic RTL context
	reqAr := httptest.NewRequest(http.MethodGet, "/admin?lang=ar", nil)
	recAr := httptest.NewRecorder()
	gcAr := newTestContext(recAr, reqAr)
	fCtxAr := filament.NewContext(gcAr, p)

	if fCtxAr.Locale() != "ar" {
		t.Fatalf("expected locale 'ar', got '%s'", fCtxAr.Locale())
	}
	if fCtxAr.Direction() != "rtl" {
		t.Fatalf("expected direction 'rtl', got '%s'", fCtxAr.Direction())
	}
	if !fCtxAr.IsRTL() {
		t.Fatalf("expected IsRTL to be true for 'ar'")
	}
}

func TestLocaleResolutionFromCookieAndFallback(t *testing.T) {
	p := filament.NewPanel("admin").
		Locales("en", "ar", "es").
		DefaultLocale("en")

	// 1. Resolves from cookie when no query param
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "locale", Value: "ar"})
	rec := httptest.NewRecorder()
	gc := newTestContext(rec, req)
	fCtx := filament.NewContext(gc, p)

	if fCtx.Locale() != "ar" {
		t.Fatalf("expected locale 'ar' from cookie, got '%s'", fCtx.Locale())
	}

	// 2. Query param takes precedence over cookie
	reqPrecedence := httptest.NewRequest(http.MethodGet, "/admin?lang=es", nil)
	reqPrecedence.AddCookie(&http.Cookie{Name: "locale", Value: "ar"})
	recPrecedence := httptest.NewRecorder()
	gcPrecedence := newTestContext(recPrecedence, reqPrecedence)
	fCtxPrecedence := filament.NewContext(gcPrecedence, p)

	if fCtxPrecedence.Locale() != "es" {
		t.Fatalf("expected locale 'es' from query to override cookie, got '%s'", fCtxPrecedence.Locale())
	}

	// 3. Unsupported locale falls back to default locale
	reqInvalid := httptest.NewRequest(http.MethodGet, "/admin?lang=xx_unsupported", nil)
	recInvalid := httptest.NewRecorder()
	gcInvalid := newTestContext(recInvalid, reqInvalid)
	fCtxInvalid := filament.NewContext(gcInvalid, p)

	if fCtxInvalid.Locale() != "en" {
		t.Fatalf("expected invalid locale to fall back to default 'en', got '%s'", fCtxInvalid.Locale())
	}
}

func TestWebLocaleSwitchEndpoint(t *testing.T) {
	app := golaravel.New()
	p := filament.NewPanel("admin").
		Path("/admin").
		Locales("en", "ar")

	p.Register(app)

	// GET /admin/locale/switch?lang=ar with Referer
	req := httptest.NewRequest(http.MethodGet, "/admin/locale/switch?lang=ar", nil)
	req.Header.Set("Referer", "/admin/users")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 See Other, got %d", rec.Code)
	}

	// Verify redirect location
	loc := rec.Header().Get("Location")
	if loc != "/admin/users" {
		t.Fatalf("expected redirect to '/admin/users', got '%s'", loc)
	}

	// Verify Set-Cookie header
	cookies := rec.Result().Cookies()
	var localeCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "locale" {
			localeCookie = c
			break
		}
	}

	if localeCookie == nil {
		t.Fatalf("expected 'locale' cookie in response headers")
	}
	if localeCookie.Value != "ar" {
		t.Fatalf("expected cookie value 'ar', got '%s'", localeCookie.Value)
	}
}

func TestRESTAPISchemaLocaleMetadata(t *testing.T) {
	app := golaravel.New()
	p := filament.NewPanel("admin").
		Path("/admin").
		Locales("en", "ar", "es").
		DefaultLocale("en")

	p.Register(app)

	// Request schema with Arabic locale query
	req := httptest.NewRequest(http.MethodGet, "/admin/api/schema?lang=ar", nil)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /admin/api/schema, got %d", rec.Code)
	}

	var schema map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&schema); err != nil {
		t.Fatalf("failed to decode schema JSON: %v", err)
	}

	if schema["locale"] != "ar" {
		t.Fatalf("expected schema locale 'ar', got '%v'", schema["locale"])
	}
	if schema["direction"] != "rtl" {
		t.Fatalf("expected schema direction 'rtl', got '%v'", schema["direction"])
	}
	if schema["is_rtl"] != true {
		t.Fatalf("expected schema is_rtl true, got '%v'", schema["is_rtl"])
	}

	supported, ok := schema["supported_locales"].([]any)
	if !ok || len(supported) != 3 {
		t.Fatalf("expected 3 supported locales in schema, got %v", schema["supported_locales"])
	}
}

func TestRESTAPILocaleSwitch(t *testing.T) {
	app := golaravel.New()
	p := filament.NewPanel("admin").
		Path("/admin").
		Locales("en", "ar", "fr").
		DefaultLocale("en")

	p.Register(app)

	// 1. Successful locale switch via JSON POST
	jsonBody := `{"locale": "ar"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/api/locale/switch", strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	if resp["success"] != true {
		t.Fatalf("expected success true, got %v", resp["success"])
	}
	if resp["locale"] != "ar" {
		t.Fatalf("expected locale 'ar', got %v", resp["locale"])
	}
	if resp["direction"] != "rtl" {
		t.Fatalf("expected direction 'rtl', got %v", resp["direction"])
	}
	if resp["is_rtl"] != true {
		t.Fatalf("expected is_rtl true, got %v", resp["is_rtl"])
	}

	// 2. Reject unsupported locale
	unsupportedBody := `{"locale": "zz_unknown"}`
	reqUnsup := httptest.NewRequest(http.MethodPost, "/admin/api/locale/switch", strings.NewReader(unsupportedBody))
	reqUnsup.Header.Set("Content-Type", "application/json")
	recUnsup := httptest.NewRecorder()

	app.ServeHTTP(recUnsup, reqUnsup)

	if recUnsup.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for unsupported locale, got %d", recUnsup.Code)
	}
}

func TestRTLHTMLLayoutRendering(t *testing.T) {
	app := golaravel.New()
	p := filament.NewPanel("admin").
		Path("/admin").
		Locales("en", "ar")

	p.Register(app)

	// 1. Request dashboard with Arabic locale
	reqAr := httptest.NewRequest(http.MethodGet, "/admin?lang=ar", nil)
	recAr := httptest.NewRecorder()
	app.ServeHTTP(recAr, reqAr)

	bodyAr := recAr.Body.String()
	if !strings.Contains(bodyAr, `<html lang="ar" dir="rtl">`) {
		t.Fatalf("expected `<html lang=\"ar\" dir=\"rtl\">` in dashboard HTML, got: %s", bodyAr[:300])
	}
	// Check language switcher pill is present
	if !strings.Contains(bodyAr, "filament-locale-switcher") {
		t.Fatalf("expected language switcher dropdown in topbar")
	}

	// 2. Request dashboard with English locale
	reqEn := httptest.NewRequest(http.MethodGet, "/admin?lang=en", nil)
	recEn := httptest.NewRecorder()
	app.ServeHTTP(recEn, reqEn)

	bodyEn := recEn.Body.String()
	if !strings.Contains(bodyEn, `<html lang="en" dir="ltr">`) {
		t.Fatalf("expected `<html lang=\"en\" dir=\"ltr\">` in dashboard HTML, got: %s", bodyEn[:300])
	}
}
