package tests

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/gofilament/pkg/widgets"
	"github.com/golaravel/golaravel/pkg/features"
	"github.com/golaravel/golaravel/pkg/golaravel"
	_ "github.com/mattn/go-sqlite3"
)

// appFeatures are the features of one test app. Features bind to a single
// manager, so every test declares its own.
type appFeatures struct {
	mgr         *features.Manager
	recurring   *features.Feature[bool]   // SuperAdmin
	taxMode     *features.Feature[string] // TenantAdmin
	maxBranches *features.Feature[int]
	footer      *features.Feature[string]
	theme       *features.Feature[map[string]string]
}

func newAppFeatures(t *testing.T, mgr *features.Manager) *appFeatures {
	t.Helper()
	if mgr == nil {
		mgr = features.NewInMemory(features.Options{})
	}
	f := &appFeatures{
		mgr: mgr,
		recurring: features.Bool("invoices.recurring", false).
			Label("Recurring invoices").Group("Billing").ManagedBy(features.SuperAdmin).Public(),
		taxMode: features.Enum("tax.mode", "inclusive", "inclusive", "exclusive", "none").
			Label("Tax mode").Group("Billing").Description("How prices include tax."),
		maxBranches: features.Int("branches.max", 1).Rules(features.Range(1, 50)).
			Label("Branches").Group("Limits"),
		footer: features.String("invoice.footer", "").Rules(features.MaxLen(20)).
			Label("Invoice footer").Group("Billing"),
		theme: features.JSON("ui.theme", map[string]string{"color": "blue"}).Label("Theme").Group("Appearance"),
	}
	if err := mgr.Register(f.recurring, f.taxMode, f.maxBranches, f.footer, f.theme); err != nil {
		t.Fatal(err)
	}
	return f
}

type invoice struct {
	ID       string
	Number   string
	Interval string
}

// newFeaturePanel is a tenant panel: the tenant comes from the
// filament_tenant cookie or the X-Tenant-ID header (acme or globex).
func newFeaturePanel(t *testing.T, f *appFeatures) (*golaravel.Engine, *filament.Panel) {
	t.Helper()
	app := golaravel.New()
	on := filament.FeatureEnabled(f.recurring)
	recurring := filament.NewResource[*invoice]("recurring-invoices").
		ModelName("Recurring invoice").PluralModelName("Recurring invoices").
		RequiresFeature(on).
		RecordID(func(r *invoice) string { return r.ID }).
		OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*invoice, int, error) {
			return []*invoice{{ID: "1", Number: "R-1", Interval: "monthly"}}, 1, nil
		}).
		OnFind(func(ctx *filament.Context, id string) (*invoice, error) {
			return &invoice{ID: id, Number: "R-" + id}, nil
		})
	invoices := filament.NewResource[*invoice]("invoices").
		ModelName("Invoice").PluralModelName("Invoices").
		RecordID(func(r *invoice) string { return r.ID }).
		Form(func(form *forms.Form) *forms.Form {
			return form.Schema(
				forms.TextInput("number").Required(),
				forms.Select("interval").Options(map[string]string{"monthly": "Monthly", "yearly": "Yearly"}).RequiresFeature(on),
			)
		}).
		Table(func(tbl *tables.Table[*invoice]) *tables.Table[*invoice] {
			return tbl.Columns(
				tables.TextColumn[*invoice]("number"),
				tables.TextColumn[*invoice]("interval").Label("Repeats").RequiresFeature(on),
			).Actions(
				tables.Action[*invoice]("repeat").Label("Repeat invoice").RequiresFeature(on).
					Handle(func(ctx context.Context, r *invoice, data map[string]any) error { return nil }),
			)
		}).
		OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*invoice, int, error) {
			return []*invoice{{ID: "1", Number: "INV-1", Interval: "monthly"}}, 1, nil
		}).
		OnFind(func(ctx *filament.Context, id string) (*invoice, error) {
			return &invoice{ID: id, Number: "INV-" + id, Interval: "monthly"}, nil
		}).
		OnCreate(func(ctx *filament.Context, data map[string]any) error { return nil })
	p := filament.NewPanel("admin").Path("/admin").
		Tenancy(&filament.TenantConfig{
			CanAccessTenant: func(ctx *filament.Context, key string) bool { return key == "acme" || key == "globex" },
		}).
		Features(f.mgr).
		Resources(recurring, invoices).
		Page(filament.FlutterPage("pos", "Point of sale").RequiresFeature(on)).
		Widget(widgets.NewStatsOverviewWidget(func(*filament.Context) []widgets.Stat { return nil }).Key("mrr").RequiresFeature(on))
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, p
}

func tenantClient(t *testing.T, app http.Handler, tenant string) *webClient {
	w := newWebClient(t, app)
	if tenant != "" {
		w.cookies["filament_tenant"] = &http.Cookie{Name: "filament_tenant", Value: tenant}
	}
	return w
}

func apiCall(t *testing.T, app http.Handler, method, path, tenant string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		raw, _ := json.Marshal(body)
		req = httptest.NewRequest(method, "http://example.com"+path, strings.NewReader(string(raw)))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, "http://example.com"+path, nil)
	}
	if tenant != "" {
		req.Header.Set("X-Tenant-ID", tenant)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func mustValue(t *testing.T, mgr *features.Manager, tenant, key string) any {
	t.Helper()
	v, err := mgr.Get(features.ForTenant(context.Background(), tenant), key)
	if err != nil {
		t.Fatalf("get %s for %s: %v", key, tenant, err)
	}
	return v
}

func fieldHTML(t *testing.T, body, key string) string {
	t.Helper()
	i := strings.Index(body, `data-feature="`+key+`"`)
	if i < 0 {
		t.Fatalf("no field for %s in:\n%s", key, body)
	}
	rest := body[i:]
	if j := strings.Index(rest[1:], `data-feature="`); j >= 0 {
		rest = rest[:j+1]
	}
	return rest
}

// ---------------------------------------------------------------- settings

func TestTenantSettingsPageRenders(t *testing.T) {
	f := newAppFeatures(t, nil)
	app, _ := newFeaturePanel(t, f)
	w := tenantClient(t, app, "acme")

	rec := w.do("GET", "/admin/tenant-settings", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Tenant settings", "Billing", "Limits", "Appearance", "Tax mode", "How prices include tax.", `data-feature-group="Billing"`} {
		if !strings.Contains(body, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	// SuperAdmin features never appear on the tenant's own page.
	if strings.Contains(body, "invoices.recurring") || strings.Contains(body, "Recurring invoices</label>") {
		t.Error("tenant page shows a SuperAdmin feature")
	}
	// One control per type.
	if f := fieldHTML(t, body, "tax.mode"); !strings.Contains(f, "<select") || !strings.Contains(f, `<option value="exclusive">`) || !strings.Contains(f, "Default") {
		t.Errorf("enum control: %s", f)
	}
	if f := fieldHTML(t, body, "branches.max"); !strings.Contains(f, `type="number"`) || !strings.Contains(f, `min="1"`) || !strings.Contains(f, `max="50"`) {
		t.Errorf("int control: %s", f)
	}
	if f := fieldHTML(t, body, "invoice.footer"); !strings.Contains(f, `type="text"`) {
		t.Errorf("string control: %s", f)
	}
	if f := fieldHTML(t, body, "ui.theme"); !strings.Contains(f, "<textarea") || !strings.Contains(f, "&#34;color&#34;: &#34;blue&#34;") {
		t.Errorf("json control: %s", f)
	}
	if strings.Contains(body, "Reset to default") {
		t.Error("nothing is overridden yet")
	}
	// The settings page is in the navigation.
	if !strings.Contains(body, `href="/admin/tenant-settings"`) {
		t.Error("no navigation link to the settings page")
	}
}

func TestTenantSettingsBoolToggle(t *testing.T) {
	mgr := features.NewInMemory(features.Options{})
	beta := features.Bool("ui.beta", false).Label("Beta features")
	if err := mgr.Register(beta); err != nil {
		t.Fatal(err)
	}
	app := golaravel.New()
	p := filament.NewPanel("admin").Path("/admin").
		Tenancy(&filament.TenantConfig{CanAccessTenant: func(*filament.Context, string) bool { return true }}).
		Features(mgr)
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	w := tenantClient(t, app, "acme")
	body := w.do("GET", "/admin/tenant-settings", nil).Body.String()
	if f := fieldHTML(t, body, "ui.beta"); !strings.Contains(f, `type="checkbox"`) || strings.Contains(f, "checked") {
		t.Fatalf("toggle: %s", f)
	}
	token := w.csrfFrom(body)
	if rec := w.do("POST", "/admin/tenant-settings", url.Values{"_token": {token}, "ui.beta": {"false", "true"}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
	}
	if !features.Enabled(features.ForTenant(context.Background(), "acme"), beta) {
		t.Fatal("toggle on not saved")
	}
	// Unchecked: only the hidden "false" is sent.
	if rec := w.do("POST", "/admin/tenant-settings", url.Values{"_token": {token}, "ui.beta": {"false"}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("save: %d", rec.Code)
	}
	if features.Enabled(features.ForTenant(context.Background(), "acme"), beta) {
		t.Fatal("toggle off not saved")
	}
}

func TestTenantSettingsSave(t *testing.T) {
	f := newAppFeatures(t, nil)
	app, _ := newFeaturePanel(t, f)
	w := tenantClient(t, app, "acme")
	token := w.csrfFrom(w.do("GET", "/admin/tenant-settings", nil).Body.String())

	rec := w.do("POST", "/admin/tenant-settings", url.Values{
		"_token":         {token},
		"tax.mode":       {"exclusive"},
		"branches.max":   {"7"},
		"invoice.footer": {""}, // unchanged: no override is created
		"ui.theme":       {`{"color": "red"}`},
	})
	if rec.Code != http.StatusSeeOther || !strings.HasSuffix(rec.Header().Get("Location"), "/admin/tenant-settings?status=saved") {
		t.Fatalf("save: %d %q %s", rec.Code, rec.Header().Get("Location"), rec.Body.String())
	}
	if v := mustValue(t, f.mgr, "acme", "tax.mode"); v != "exclusive" {
		t.Fatalf("tax.mode = %v", v)
	}
	if v := mustValue(t, f.mgr, "acme", "branches.max"); v != 7 {
		t.Fatalf("branches.max = %v", v)
	}
	if v := mustValue(t, f.mgr, "acme", "ui.theme").(map[string]string); v["color"] != "red" {
		t.Fatalf("ui.theme = %v", v)
	}
	if over, _ := f.footer.Overridden(features.ForTenant(context.Background(), "acme")); over {
		t.Fatal("an unchanged field became an override")
	}
	// Only the request's tenant changed.
	if v := mustValue(t, f.mgr, "globex", "tax.mode"); v != "inclusive" {
		t.Fatalf("globex tax.mode = %v", v)
	}

	body := w.do("GET", "/admin/tenant-settings?status=saved", nil).Body.String()
	if !strings.Contains(body, "Settings saved.") {
		t.Error("no saved notice")
	}
	tax := fieldHTML(t, body, "tax.mode")
	if !strings.Contains(tax, "Overridden") || !strings.Contains(tax, "Reset to default") || !strings.Contains(tax, `<option value="exclusive" selected>`) {
		t.Errorf("overridden field: %s", tax)
	}
	if footer := fieldHTML(t, body, "invoice.footer"); strings.Contains(footer, "Overridden") {
		t.Errorf("footer is not overridden: %s", footer)
	}
}

func TestTenantSettingsValidationError(t *testing.T) {
	f := newAppFeatures(t, nil)
	app, _ := newFeaturePanel(t, f)
	w := tenantClient(t, app, "acme")
	token := w.csrfFrom(w.do("GET", "/admin/tenant-settings", nil).Body.String())

	rec := w.do("POST", "/admin/tenant-settings", url.Values{
		"_token":         {token},
		"branches.max":   {"99"},
		"invoice.footer": {strings.Repeat("x", 30)},
		"ui.theme":       {"{not json"},
		"tax.mode":       {"none"}, // valid: still saved
	})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	if f := fieldHTML(t, body, "branches.max"); !strings.Contains(f, `class="form-error">Must be between 1 and 50.`) || !strings.Contains(f, `value="99"`) {
		t.Errorf("int error: %s", f)
	}
	if f := fieldHTML(t, body, "invoice.footer"); !strings.Contains(f, "Must be at most 20 characters.") {
		t.Errorf("string error: %s", f)
	}
	if f := fieldHTML(t, body, "ui.theme"); !strings.Contains(f, "form-error") {
		t.Errorf("json error: %s", f)
	}
	if !strings.Contains(body, "Some settings were not saved") {
		t.Error("no summary error")
	}
	if v := mustValue(t, f.mgr, "acme", "branches.max"); v != 1 {
		t.Fatalf("invalid value stored: %v", v)
	}
	if v := mustValue(t, f.mgr, "acme", "tax.mode"); v != "none" {
		t.Fatalf("valid value not stored: %v", v)
	}

	// The same through the API.
	rec = apiCall(t, app, "PUT", "/admin/api/tenant-settings", "acme", map[string]any{"values": map[string]any{"tax.mode": "bogus"}})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("api status %d: %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Errors map[string]string `json:"errors"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if !strings.Contains(out.Errors["tax.mode"], "Must be one of") {
		t.Fatalf("api errors: %s", rec.Body.String())
	}
}

func TestTenantSettingsReset(t *testing.T) {
	f := newAppFeatures(t, nil)
	app, _ := newFeaturePanel(t, f)
	acme := features.ForTenant(context.Background(), "acme")
	if err := f.taxMode.Set(acme, "none"); err != nil {
		t.Fatal(err)
	}
	w := tenantClient(t, app, "acme")
	token := w.csrfFrom(w.do("GET", "/admin/tenant-settings", nil).Body.String())

	rec := w.do("POST", "/admin/tenant-settings/reset", url.Values{"_token": {token}, "_reset": {"tax.mode"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("reset: %d %s", rec.Code, rec.Body.String())
	}
	if over, _ := f.taxMode.Overridden(acme); over {
		t.Fatal("still overridden")
	}
	if !strings.Contains(w.do("GET", rec.Header().Get("Location"), nil).Body.String(), "back to its default") {
		t.Error("no reset notice")
	}

	// API reset.
	_ = f.maxBranches.Set(acme, 3)
	rec = apiCall(t, app, "DELETE", "/admin/api/tenant-settings/branches.max", "acme", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("api reset: %d %s", rec.Code, rec.Body.String())
	}
	if f.maxBranches.Value(acme) != 1 {
		t.Fatal("api reset failed")
	}
	// Reset without CSRF token is refused.
	if rec := w.do("POST", "/admin/tenant-settings/reset", url.Values{"_reset": {"tax.mode"}}); rec.Code != http.StatusForbidden {
		t.Fatalf("reset without token: %d", rec.Code)
	}
}

func TestTenantSettingsAPI(t *testing.T) {
	f := newAppFeatures(t, nil)
	app, _ := newFeaturePanel(t, f)
	rec := apiCall(t, app, "GET", "/admin/api/tenant-settings", "acme", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Tenant string `json:"tenant"`
		Groups []struct {
			Name     string              `json:"name"`
			Features []features.Resolved `json:"features"`
		} `json:"groups"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	keys := []string{}
	for _, g := range out.Groups {
		for _, r := range g.Features {
			keys = append(keys, r.Key)
		}
	}
	if out.Tenant != "acme" || strings.Join(keys, ",") != "ui.theme,invoice.footer,tax.mode,branches.max" {
		t.Fatalf("listing: %s %v", out.Tenant, keys)
	}
	rec = apiCall(t, app, "PUT", "/admin/api/tenant-settings", "acme", map[string]any{"values": map[string]any{"branches.max": 4, "ui.theme": map[string]string{"color": "green"}}})
	if rec.Code != http.StatusOK {
		t.Fatalf("update: %d %s", rec.Code, rec.Body.String())
	}
	if v := mustValue(t, f.mgr, "acme", "branches.max"); v != 4 {
		t.Fatalf("branches.max = %v", v)
	}
	// Without a tenant the API refuses.
	if rec := apiCall(t, app, "GET", "/admin/api/tenant-settings", "", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("no tenant: %d", rec.Code)
	}
	// The schema advertises the page and the Public features.
	rec = apiCall(t, app, "GET", "/admin/api/schema", "acme", nil)
	var schema map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &schema)
	ts, _ := schema["tenant_settings"].(map[string]any)
	if ts["enabled"] != true || ts["endpoint"] != "/admin/api/tenant-settings" {
		t.Fatalf("tenant_settings: %v", schema["tenant_settings"])
	}
	if pub, _ := schema["features"].(map[string]any); pub["invoices.recurring"] != false || len(pub) != 1 {
		t.Fatalf("public features: %v", schema["features"])
	}
}

// --------------------------------------------------------- SuperAdmin guard

func TestTenantCannotChangeSuperAdminFeatures(t *testing.T) {
	f := newAppFeatures(t, nil)
	app, _ := newFeaturePanel(t, f)
	acme := features.ForTenant(context.Background(), "acme")
	w := tenantClient(t, app, "acme")
	token := w.csrfFrom(w.do("GET", "/admin/tenant-settings", nil).Body.String())

	// A posted SuperAdmin key is not one of the page's fields: ignored.
	w.do("POST", "/admin/tenant-settings", url.Values{"_token": {token}, "invoices.recurring": {"true"}})
	if features.Enabled(acme, f.recurring) {
		t.Fatal("web form enabled a SuperAdmin feature")
	}
	// The API refuses it outright, and writes nothing else of the request.
	rec := apiCall(t, app, "PUT", "/admin/api/tenant-settings", "acme", map[string]any{"values": map[string]any{"invoices.recurring": true, "tax.mode": "none"}})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("api set: %d %s", rec.Code, rec.Body.String())
	}
	if features.Enabled(acme, f.recurring) || f.taxMode.Value(acme) != "inclusive" {
		t.Fatal("api wrote despite the forbidden key")
	}
	// Reset is refused too (web and API), so a platform override sticks.
	if err := f.recurring.Set(acme, true); err != nil {
		t.Fatal(err)
	}
	if rec := w.do("POST", "/admin/tenant-settings/reset", url.Values{"_token": {token}, "_reset": {"invoices.recurring"}}); rec.Code != http.StatusForbidden {
		t.Fatalf("web reset: %d", rec.Code)
	}
	if rec := apiCall(t, app, "DELETE", "/admin/api/tenant-settings/invoices.recurring", "acme", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("api reset: %d", rec.Code)
	}
	if !features.Enabled(acme, f.recurring) {
		t.Fatal("tenant reset a SuperAdmin feature")
	}
	// Unknown keys are refused.
	if rec := apiCall(t, app, "PUT", "/admin/api/tenant-settings", "acme", map[string]any{"values": map[string]any{"nope": 1}}); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown key: %d", rec.Code)
	}
}

func TestTenantSettingsCanManage(t *testing.T) {
	f := newAppFeatures(t, nil)
	app := golaravel.New()
	p := filament.NewPanel("admin").Path("/admin").
		Tenancy(&filament.TenantConfig{CanAccessTenant: func(*filament.Context, string) bool { return true }}).
		Features(f.mgr).
		TenantSettings(filament.TenantSettingsConfig{CanManage: func(*filament.Context) bool { return false }})
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	if rec := tenantClient(t, app, "acme").do("GET", "/admin/tenant-settings", nil); rec.Code != http.StatusForbidden {
		t.Fatalf("status %d", rec.Code)
	}
	if rec := apiCall(t, app, "PUT", "/admin/api/tenant-settings", "acme", map[string]any{"values": map[string]any{"tax.mode": "none"}}); rec.Code != http.StatusForbidden {
		t.Fatalf("api status %d", rec.Code)
	}
}

// ---------------------------------------------------------- central screen

func newCentralPanel(t *testing.T, f *appFeatures) *golaravel.Engine {
	t.Helper()
	app := golaravel.New()
	p := filament.NewPanel("central").Path("/central").
		TenantFeaturesAdmin(f.mgr, filament.TenantFeaturesAdminConfig{
			Tenants: func(*filament.Context) ([]filament.Tenant, error) {
				return []filament.Tenant{
					&filament.TenantItem{Key: "acme", Name: "Acme Inc"},
					&filament.TenantItem{Key: "globex", Name: "Globex"},
				}, nil
			},
		})
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app
}

func TestSuperAdminTenantFeatures(t *testing.T) {
	f := newAppFeatures(t, nil)
	app := newCentralPanel(t, f)
	w := newWebClient(t, app)

	list := w.do("GET", "/central/tenant-features", nil).Body.String()
	if !strings.Contains(list, "Acme Inc") || !strings.Contains(list, `href="/central/tenant-features/globex"`) {
		t.Fatalf("tenant list: %s", list)
	}
	rec := w.do("GET", "/central/tenant-features/acme", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	body := rec.Body.String()
	rf := fieldHTML(t, body, "invoices.recurring")
	if !strings.Contains(rf, `type="checkbox"`) || !strings.Contains(rf, "Super admin only") {
		t.Fatalf("SuperAdmin feature on central screen: %s", rf)
	}
	fieldHTML(t, body, "tax.mode") // tenant-managed ones too

	token := w.csrfFrom(body)
	rec = w.do("POST", "/central/tenant-features/acme", url.Values{"_token": {token}, "invoices.recurring": {"false", "true"}, "branches.max": {"20"}})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
	}
	if !features.Enabled(features.ForTenant(context.Background(), "acme"), f.recurring) {
		t.Fatal("not enabled for acme")
	}
	if features.Enabled(features.ForTenant(context.Background(), "globex"), f.recurring) {
		t.Fatal("enabled for globex too")
	}
	if v := mustValue(t, f.mgr, "acme", "branches.max"); v != 20 {
		t.Fatalf("branches.max = %v", v)
	}
	// Validation errors show on the field.
	rec = w.do("POST", "/central/tenant-features/acme", url.Values{"_token": {token}, "branches.max": {"0"}})
	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(fieldHTML(t, rec.Body.String(), "branches.max"), "Must be between 1 and 50.") {
		t.Fatalf("validation: %d", rec.Code)
	}
	// Reset.
	if rec := w.do("POST", "/central/tenant-features/acme/reset", url.Values{"_token": {token}, "_reset": {"invoices.recurring"}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("reset: %d", rec.Code)
	}
	if features.Enabled(features.ForTenant(context.Background(), "acme"), f.recurring) {
		t.Fatal("reset failed")
	}
	// Unknown tenants are refused.
	if rec := w.do("GET", "/central/tenant-features/initech", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown tenant: %d", rec.Code)
	}
	// API.
	rec = apiCall(t, app, "GET", "/central/api/tenant-features", "", nil)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `{"key":"acme","name":"Acme Inc"}`) || !strings.Contains(rec.Body.String(), `"free_entry":false`) {
		t.Fatalf("api tenant list: %d %s", rec.Code, rec.Body.String())
	}
	if pick, _ := schemaOfCentral(t, app)["tenant_features_admin"].(map[string]any); pick["path"] != "/tenant-features" {
		t.Fatalf("schema tenant_features_admin: %v", pick)
	}
	rec = apiCall(t, app, "PUT", "/central/api/tenant-features/globex", "", map[string]any{"values": map[string]any{"invoices.recurring": true}})
	if rec.Code != http.StatusOK {
		t.Fatalf("api: %d %s", rec.Code, rec.Body.String())
	}
	if !features.Enabled(features.ForTenant(context.Background(), "globex"), f.recurring) {
		t.Fatal("api did not enable for globex")
	}
	if rec := apiCall(t, app, "DELETE", "/central/api/tenant-features/globex/invoices.recurring", "", nil); rec.Code != http.StatusOK {
		t.Fatalf("api reset: %d", rec.Code)
	}
	if features.Enabled(features.ForTenant(context.Background(), "globex"), f.recurring) {
		t.Fatal("api reset failed")
	}
}

// ---------------------------------------------------------------- gating

func schemaOf(t *testing.T, app http.Handler, tenant string) map[string]any {
	t.Helper()
	rec := apiCall(t, app, "GET", "/admin/api/schema", tenant, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("schema: %d %s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func schemaResources(s map[string]any) []string {
	var out []string
	list, _ := s["resources"].([]any)
	for _, r := range list {
		out = append(out, r.(map[string]any)["slug"].(string))
	}
	return out
}

func TestFeatureHiddenResourceRefusesDirectAccess(t *testing.T) {
	f := newAppFeatures(t, nil)
	app, _ := newFeaturePanel(t, f)

	paths := []string{
		"/admin/recurring-invoices",
		"/admin/recurring-invoices/create",
		"/admin/recurring-invoices/1",
		"/admin/recurring-invoices/1/edit",
		"/admin/recurring-invoices/export",
		"/admin/api/resources/recurring-invoices",
		"/admin/api/resources/recurring-invoices/1",
		"/admin/api/resources/recurring-invoices/form",
	}
	for _, tenant := range []string{"acme", ""} { // off, and no tenant at all (fail closed)
		for _, path := range paths {
			if rec := apiCall(t, app, "GET", path, tenant, nil); rec.Code != http.StatusNotFound {
				t.Errorf("tenant %q GET %s: %d, want 404", tenant, path, rec.Code)
			}
		}
		if rec := apiCall(t, app, "POST", "/admin/api/resources/recurring-invoices", tenant, map[string]any{"number": "x"}); rec.Code != http.StatusNotFound {
			t.Errorf("tenant %q POST: %d", tenant, rec.Code)
		}
		if rec := apiCall(t, app, "DELETE", "/admin/api/resources/recurring-invoices/1", tenant, nil); rec.Code != http.StatusNotFound {
			t.Errorf("tenant %q DELETE: %d", tenant, rec.Code)
		}
	}
	// Left out of the navigation and the API schema.
	body := tenantClient(t, app, "acme").do("GET", "/admin", nil).Body.String()
	if strings.Contains(body, "/admin/recurring-invoices") {
		t.Error("hidden resource in navigation")
	}
	s := schemaOf(t, app, "acme")
	if strings.Contains(strings.Join(schemaResources(s), ","), "recurring-invoices") {
		t.Errorf("hidden resource in schema: %v", schemaResources(s))
	}
	if meta, _ := s["resource_meta"].(map[string]any); meta["recurring-invoices"] != nil {
		t.Error("hidden resource in resource_meta")
	}
	raw, _ := json.Marshal(s["navigation"])
	if strings.Contains(string(raw), "recurring-invoices") {
		t.Errorf("hidden resource in schema navigation: %s", raw)
	}
	if pages, _ := s["client_pages"].([]any); len(pages) != 0 {
		t.Errorf("hidden Flutter page listed: %v", pages)
	}
	if ws, _ := s["widgets"].([]any); len(ws) != 0 {
		t.Errorf("hidden widget listed: %v", ws)
	}
	if rec := apiCall(t, app, "GET", "/admin/widgets/mrr/data", "acme", nil); rec.Code != http.StatusNotFound {
		t.Errorf("hidden widget data: %d", rec.Code)
	}

	// Turned on for acme only.
	if err := f.recurring.Set(features.ForTenant(context.Background(), "acme"), true); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/admin/recurring-invoices", "/admin/api/resources/recurring-invoices"} {
		if rec := apiCall(t, app, "GET", path, "acme", nil); rec.Code != http.StatusOK {
			t.Errorf("acme %s after enabling: %d", path, rec.Code)
		}
		if rec := apiCall(t, app, "GET", path, "globex", nil); rec.Code != http.StatusNotFound {
			t.Errorf("globex %s: %d", path, rec.Code)
		}
	}
	if !strings.Contains(strings.Join(schemaResources(schemaOf(t, app, "acme")), ","), "recurring-invoices") {
		t.Error("enabled resource missing from schema")
	}
	if pages, _ := schemaOf(t, app, "acme")["client_pages"].([]any); len(pages) != 1 {
		t.Errorf("enabled Flutter page missing: %v", pages)
	}
	if !strings.Contains(tenantClient(t, app, "acme").do("GET", "/admin", nil).Body.String(), `href="/admin/recurring-invoices"`) {
		t.Error("enabled resource missing from navigation")
	}
}

func TestFeatureGatedColumnsActionsAndFields(t *testing.T) {
	f := newAppFeatures(t, nil)
	app, _ := newFeaturePanel(t, f)
	acme := features.ForTenant(context.Background(), "acme")

	columns := func(tenant string) string {
		for _, r := range schemaOf(t, app, tenant)["resources"].([]any) {
			m := r.(map[string]any)
			if m["slug"] == "invoices" {
				raw, _ := json.Marshal(m["columns"])
				return string(raw)
			}
		}
		t.Fatal("no invoices resource")
		return ""
	}

	// Off: column, action and field are gone everywhere.
	list := apiCall(t, app, "GET", "/admin/invoices", "acme", nil).Body.String()
	if strings.Contains(list, "Repeats") || strings.Contains(list, "Repeat invoice") {
		t.Error("gated column or action rendered")
	}
	if strings.Contains(columns("acme"), `"interval"`) {
		t.Errorf("gated column in schema: %s", columns("acme"))
	}
	form := apiCall(t, app, "GET", "/admin/invoices/create", "acme", nil).Body.String()
	if strings.Contains(form, `name="interval"`) {
		t.Error("gated form field rendered")
	}
	formAPI := apiCall(t, app, "GET", "/admin/api/resources/invoices/form", "acme", nil).Body.String()
	if strings.Contains(formAPI, `"interval"`) {
		t.Errorf("gated form field in API form: %s", formAPI)
	}
	if rec := apiCall(t, app, "POST", "/admin/api/resources/invoices/1/actions/repeat", "acme", map[string]any{}); rec.Code < 400 {
		t.Errorf("gated action ran: %d", rec.Code)
	}

	// On: all of them are back.
	if err := f.recurring.Set(acme, true); err != nil {
		t.Fatal(err)
	}
	list = apiCall(t, app, "GET", "/admin/invoices", "acme", nil).Body.String()
	if !strings.Contains(list, "Repeats") || !strings.Contains(list, "Repeat invoice") {
		t.Error("enabled column or action missing")
	}
	if !strings.Contains(columns("acme"), `"interval"`) {
		t.Errorf("enabled column missing from schema: %s", columns("acme"))
	}
	if !strings.Contains(apiCall(t, app, "GET", "/admin/invoices/create", "acme", nil).Body.String(), `name="interval"`) {
		t.Error("enabled form field missing")
	}
	if strings.Contains(columns("globex"), `"interval"`) {
		t.Error("globex sees acme's column")
	}
}

func TestFeatureGatedFormFieldValuesAreDropped(t *testing.T) {
	mgr := features.NewInMemory(features.Options{})
	on := features.Bool("x.on", false)
	if err := mgr.Register(on); err != nil {
		t.Fatal(err)
	}
	comps := []forms.Component{
		forms.TextInput("name"),
		forms.TextInput("secret").RequiresFeature(filament.FeatureEnabled(on)),
		forms.Section("Extra").RequiresFeature(filament.FeatureEnabled(on)).Schema(forms.TextInput("inner")),
	}
	ctx := features.ForTenant(context.Background(), "acme")
	data := map[string]any{"name": "a", "secret": "s", "inner": "i"}
	out, err := forms.Validate(ctx, comps, data, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out["secret"]; ok || out["inner"] != nil || out["name"] != "a" {
		t.Fatalf("off: %v", out)
	}
	if err := on.Set(ctx, true); err != nil {
		t.Fatal(err)
	}
	out, _ = forms.Validate(ctx, comps, data, true)
	if out["secret"] != "s" || out["inner"] != "i" {
		t.Fatalf("on: %v", out)
	}
	// No tenant: fail closed.
	out, _ = forms.Validate(context.Background(), comps, data, true)
	if _, ok := out["secret"]; ok {
		t.Fatalf("no tenant: %v", out)
	}
}

func TestFeatureConditions(t *testing.T) {
	mgr := features.NewInMemory(features.Options{})
	mode := features.Enum("tax.mode", "inclusive", "inclusive", "exclusive", "none")
	on := features.Bool("x.on", false)
	if err := mgr.Register(mode, on); err != nil {
		t.Fatal(err)
	}
	acme := features.ForTenant(context.Background(), "acme")
	none := context.Background()
	cases := []struct {
		name string
		c    filament.Condition
		ctx  context.Context
		want bool
	}{
		{"enabled off", filament.FeatureEnabled(on), acme, false},
		{"disabled off", filament.FeatureDisabled(on), acme, true},
		{"disabled no tenant", filament.FeatureDisabled(on), none, false},
		{"is default", filament.FeatureIs(mode, "inclusive"), acme, true},
		{"is other", filament.FeatureIs(mode, "exclusive"), acme, false},
		{"is no tenant", filament.FeatureIs(mode, "inclusive"), none, false},
		{"in", filament.FeatureIn(mode, "none", "inclusive"), acme, true},
		{"all", filament.AllOf(filament.FeatureIs(mode, "inclusive"), filament.FeatureEnabled(on)), acme, false},
		{"any", filament.AnyOf(filament.FeatureIs(mode, "inclusive"), filament.FeatureEnabled(on)), acme, true},
		{"nil ctx", filament.FeatureIs(mode, "inclusive"), nil, false},
	}
	for _, c := range cases {
		if got := c.c(c.ctx); got != c.want {
			t.Errorf("%s: %v", c.name, got)
		}
	}
}

// countingDB counts the queries that load a tenant's values.
type countingDB struct {
	*sql.DB
	loads atomic.Int64
}

func (d *countingDB) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	if strings.Contains(q, "FROM tenant_settings") {
		d.loads.Add(1)
	}
	return d.DB.QueryContext(ctx, q, args...)
}

func TestFeatureChecksLoadOncePerRequest(t *testing.T) {
	raw, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	raw.SetMaxOpenConns(1)
	defer raw.Close()
	db := &countingDB{DB: raw}
	mgr, err := features.New(features.Options{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	if err := mgr.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	f := newAppFeatures(t, mgr)
	app, _ := newFeaturePanel(t, f)

	db.loads.Store(0)
	// The schema checks the resource, page, widget, columns, actions and
	// form fields, several times each.
	schemaOf(t, app, "acme")
	if n := db.loads.Load(); n != 1 {
		t.Fatalf("schema request loaded the tenant's features %d times, want 1", n)
	}
	db.loads.Store(0)
	tenantClient(t, app, "acme").do("GET", "/admin/invoices", nil)
	if n := db.loads.Load(); n != 1 {
		t.Fatalf("list page loaded the tenant's features %d times, want 1", n)
	}
}

func TestTenantSettingsWithPathScoping(t *testing.T) {
	f := newAppFeatures(t, nil)
	app := golaravel.New()
	p := filament.NewPanel("admin").Path("/admin").
		Tenancy(&filament.TenantConfig{
			PathScoping:     true,
			CanAccessTenant: func(ctx *filament.Context, key string) bool { return key == "acme" || key == "globex" },
		}).
		Features(f.mgr)
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	w := newWebClient(t, app)
	rec := w.do("GET", "/admin/acme/tenant-settings", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `href="/admin/acme/tenant-settings"`) || !strings.Contains(body, `action="/admin/acme/tenant-settings"`) {
		t.Fatal("navigation link or form action is not tenant-scoped")
	}
	token := w.csrfFrom(body)
	rec = w.do("POST", "/admin/globex/tenant-settings", url.Values{"_token": {token}, "tax.mode": {"none"}})
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/globex/tenant-settings?status=saved" {
		t.Fatalf("save: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if mustValue(t, f.mgr, "globex", "tax.mode") != "none" || mustValue(t, f.mgr, "acme", "tax.mode") != "inclusive" {
		t.Fatal("saved for the wrong tenant")
	}
	// A tenant in the path the user may not use is refused.
	if rec := w.do("GET", "/admin/initech/tenant-settings", nil); rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
		t.Fatalf("foreign tenant: %d", rec.Code)
	}
}

func TestFeatureGatedSelectRefusesOptionEndpoints(t *testing.T) {
	f := newAppFeatures(t, nil)
	app := golaravel.New()
	on := filament.FeatureEnabled(f.recurring)
	plans := filament.NewResource[*invoice]("invoices").ModelName("Invoice").
		Form(func(form *forms.Form) *forms.Form {
			return form.Schema(
				forms.TextInput("number"),
				forms.Select("plan").RequiresFeature(on).
					OptionsUsing(func(context.Context) []forms.Option { return []forms.Option{{Value: "m", Label: "Monthly"}} }).
					CreateOptionForm(forms.TextInput("name").Required()).
					CreateOptionUsing(func(ctx context.Context, data map[string]any) (forms.Option, error) {
						return forms.Option{Value: "x", Label: "x"}, nil
					}),
			)
		}).
		OnList(func(*filament.Context, string, string, string, int, int, map[string]string) ([]*invoice, int, error) {
			return nil, 0, nil
		}).
		OnCreate(func(*filament.Context, map[string]any) error { return nil })
	p := filament.NewPanel("admin").Path("/admin").
		Tenancy(&filament.TenantConfig{CanAccessTenant: func(*filament.Context, string) bool { return true }}).
		Resources(plans)
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	create := "/admin/api/resources/invoices/form/options/create?field=plan"
	search := "/admin/api/resources/invoices/form/options?field=plan&q=m"
	for _, path := range []string{create, search} {
		if rec := apiCall(t, app, "GET", path, "acme", nil); rec.Code != http.StatusNotFound {
			t.Errorf("off: GET %s: %d", path, rec.Code)
		}
	}
	if rec := apiCall(t, app, "POST", create, "acme", map[string]any{"name": "Yearly"}); rec.Code != http.StatusNotFound {
		t.Errorf("off: create: %d", rec.Code)
	}
	if err := f.recurring.Set(features.ForTenant(context.Background(), "acme"), true); err != nil {
		t.Fatal(err)
	}
	if rec := apiCall(t, app, "GET", create, "acme", nil); rec.Code != http.StatusOK {
		t.Errorf("on: create form: %d %s", rec.Code, rec.Body.String())
	}
	if rec := apiCall(t, app, "GET", search, "acme", nil); rec.Code != http.StatusOK {
		t.Errorf("on: search: %d %s", rec.Code, rec.Body.String())
	}
}

func schemaOfCentral(t *testing.T, app http.Handler) map[string]any {
	t.Helper()
	rec := apiCall(t, app, "GET", "/central/api/schema", "", nil)
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("schema: %d %v", rec.Code, err)
	}
	return out
}
