package filament

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/features"
	"github.com/golaravel/golaravel/pkg/privacy"
	"github.com/golaravel/golaravel/pkg/router"
)

// PermManageTenantSettings lets a user change their tenant's settings on
// the tenant settings page (when the panel has Roles).
const PermManageTenantSettings = "tenant-settings.manage"

// PermManageTenantFeatures lets a user change any tenant's features on the
// central tenant features screen (when the panel has Roles).
const PermManageTenantFeatures = "tenant-features.manage"

// TenantSettingsConfig configures the tenant settings page that
// Panel.Features adds: <panel>/<Slug>, and <panel>/api/<Slug> for API
// clients. It lists the features a tenant may change itself
// (features.TenantAdmin), grouped by Meta.Group.
//
// Features managed by features.SuperAdmin never appear on it, and its
// endpoints refuse to change them (403), so a tenant cannot enable a paid
// feature by posting its key.
type TenantSettingsConfig struct {
	// Disabled removes the page and its API.
	Disabled bool
	// Slug defaults to "tenant-settings".
	Slug string
	// Label defaults to "Tenant settings".
	Label string
	// Icon defaults to "heroicon-adjustments-horizontal".
	Icon string
	// Group is the navigation group (default "Settings").
	Group string
	// Sort orders the navigation item (default 900).
	Sort int
	// CanManage decides who may see and change the settings. Defaults to
	// users with PermManageTenantSettings when the panel has Roles, and to
	// every user who may use the panel otherwise.
	CanManage func(ctx *Context) bool
}

// TenantSettings configures the tenant settings page (see
// TenantSettingsConfig). Call it with Features.
func (p *Panel) TenantSettings(cfg TenantSettingsConfig) *Panel {
	p.tenantSettings = &cfg
	return p
}

// TenantFeaturesAdminConfig configures the central tenant features screen
// (Panel.TenantFeaturesAdmin): <panel>/<Slug> lists tenants and
// <panel>/<Slug>/<tenant> edits one tenant's features, SuperAdmin ones
// included. API clients use <panel>/api/<Slug>/<tenant>.
type TenantFeaturesAdminConfig struct {
	// Tenants lists the tenants to pick from. When set, only these tenants
	// can be opened; without it any tenant key is accepted.
	Tenants func(ctx *Context) ([]Tenant, error)
	// Slug defaults to "tenant-features".
	Slug string
	// Label defaults to "Tenant features".
	Label string
	// Icon defaults to "heroicon-adjustments-vertical".
	Icon string
	// Group is the navigation group (default "Settings").
	Group string
	// Sort orders the navigation item (default 910).
	Sort int
	// CanManage decides who may use the screen. Defaults to users with
	// PermManageTenantFeatures when the panel has Roles, and to every user
	// who may use the panel otherwise (protect a central panel with
	// CanAccess).
	CanManage func(ctx *Context) bool

	manager *features.Manager
}

// TenantFeaturesAdmin adds the central (super admin) screen for editing
// any tenant's features with mgr, SuperAdmin-managed ones included. Values
// are written with features.ForTenant, so the panel itself needs no tenant.
func (p *Panel) TenantFeaturesAdmin(mgr *features.Manager, cfg TenantFeaturesAdminConfig) *Panel {
	if p.featureMgr == nil {
		p.featureMgr = mgr
	}
	cfg.manager = mgr
	p.featuresAdmin = &cfg
	return p
}

func (c *TenantSettingsConfig) slug() string {
	if c.Slug != "" {
		return strings.Trim(c.Slug, "/")
	}
	return "tenant-settings"
}

func (c *TenantFeaturesAdminConfig) slug() string {
	if c.Slug != "" {
		return strings.Trim(c.Slug, "/")
	}
	return "tenant-features"
}

func orDefault(s, def string) string {
	if s != "" {
		return s
	}
	return def
}

func (p *Panel) tenantSettingsOn() bool {
	return p.featureMgr != nil && p.tenantSettings != nil && !p.tenantSettings.Disabled
}

func (p *Panel) canManageTenantSettings(ctx *Context) bool {
	if !p.tenantSettingsOn() {
		return false
	}
	if p.tenantSettings.CanManage != nil {
		return p.tenantSettings.CanManage(ctx)
	}
	if p.roles != nil {
		return p.UserCan(ctx, PermManageTenantSettings)
	}
	return true
}

func (p *Panel) canManageTenantFeatures(ctx *Context) bool {
	if p.featuresAdmin == nil {
		return false
	}
	if p.featuresAdmin.CanManage != nil {
		return p.featuresAdmin.CanManage(ctx)
	}
	if p.roles != nil {
		return p.UserCan(ctx, PermManageTenantFeatures)
	}
	return true
}

// featureNavigation returns the navigation items of the settings screens
// the user may open.
func (p *Panel) featureNavigation(ctx *Context, basePrefix, currentPath string) []*NavigationItem {
	var out []*NavigationItem
	if p.canManageTenantSettings(ctx) && ctx.Tenant() != "" {
		c := p.tenantSettings
		u := basePrefix + "/" + c.slug()
		out = append(out, &NavigationItem{
			Label: orDefault(c.Label, "Tenant settings"), URL: u,
			Icon:  orDefault(c.Icon, "heroicon-adjustments-horizontal"),
			Group: orDefault(c.Group, "Settings"), Sort: sortOr(c.Sort, 900),
			Active: strings.HasPrefix(currentPath, u),
		})
	}
	if p.canManageTenantFeatures(ctx) {
		c := p.featuresAdmin
		u := p.path + "/" + c.slug()
		out = append(out, &NavigationItem{
			Label: orDefault(c.Label, "Tenant features"), URL: u,
			Icon:  orDefault(c.Icon, "heroicon-adjustments-vertical"),
			Group: orDefault(c.Group, "Settings"), Sort: sortOr(c.Sort, 910),
			Active: strings.HasPrefix(currentPath, u),
		})
	}
	return out
}

func sortOr(n, def int) int {
	if n != 0 {
		return n
	}
	return def
}

// featurePermissions adds the settings permissions to AllPermissions.
func (p *Panel) featurePermissions(out map[string]string) {
	if p.tenantSettingsOn() {
		out[PermManageTenantSettings] = "Change tenant settings"
	}
	if p.featuresAdmin != nil {
		out[PermManageTenantFeatures] = "Change any tenant's features"
	}
}

// ---------------------------------------------------------------------------
// Shared: listing, editing rules, saving
// ---------------------------------------------------------------------------

// featureScope is who is editing: a tenant admin on the tenant's own page
// or a super admin on the central screen.
type featureScope struct {
	mgr        *features.Manager
	superAdmin bool
}

// editable reports whether the scope may change the feature.
func (s featureScope) editable(m features.Meta) bool {
	return s.superAdmin || m.ManagedBy == features.TenantAdmin
}

// list returns the tenant's features the scope may see and change.
func (s featureScope) list(fctx contextLike) ([]features.Resolved, error) {
	all, err := s.mgr.All(fctx)
	if err != nil {
		return nil, err
	}
	out := all[:0:0]
	for _, r := range all {
		if s.editable(r.Meta) {
			out = append(out, r)
		}
	}
	return out, nil
}

// errFeatureForbidden is returned when a scope tries to change a feature it
// does not manage (a tenant admin and a SuperAdmin feature).
var errFeatureForbidden = errors.New("this setting is managed by the platform operator")

// definitionsByKey maps every registered feature by key.
func definitionsByKey(mgr *features.Manager) map[string]features.Meta {
	out := map[string]features.Meta{}
	for _, m := range mgr.Definitions() {
		out[m.Key] = m
	}
	return out
}

// check refuses the whole change when it names an unknown feature or one
// the scope does not manage, before anything is written.
func (s featureScope) check(keys []string) error {
	defs := definitionsByKey(s.mgr)
	for _, k := range keys {
		m, ok := defs[k]
		if !ok {
			return fmt.Errorf("%w: %s", features.ErrUnknownFeature, k)
		}
		if !s.editable(m) {
			return fmt.Errorf("%w: %s", errFeatureForbidden, k)
		}
	}
	return nil
}

// save writes each value the scope may change and that differs from the
// tenant's current value. It returns the validation messages by key; other
// failures (database) are returned as err. Valid values are stored even
// when others fail validation.
func (s featureScope) save(fctx contextLike, values map[string]any) (map[string]string, error) {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if err := s.check(keys); err != nil {
		return nil, err
	}
	current, err := s.mgr.All(fctx)
	if err != nil {
		return nil, err
	}
	byKey := map[string]features.Resolved{}
	for _, r := range current {
		byKey[r.Key] = r
	}
	errs := map[string]string{}
	for _, k := range keys {
		if sameFeatureValue(byKey[k], values[k]) {
			continue
		}
		if err := s.mgr.Set(fctx, k, values[k]); err != nil {
			if errors.Is(err, features.ErrInvalidValue) {
				errs[k] = invalidMessage(k, err)
				continue
			}
			return errs, err
		}
	}
	return errs, nil
}

// reset removes the tenant's value of key (after the same checks as save).
func (s featureScope) reset(fctx contextLike, key string) error {
	if err := s.check([]string{key}); err != nil {
		return err
	}
	return s.mgr.Reset(fctx, key)
}

// invalidMessage strips the "features: invalid value: <key>: " prefix.
func invalidMessage(key string, err error) string {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, features.ErrInvalidValue.Error()+": ")
	msg = strings.TrimPrefix(msg, key+": ")
	if msg == "" {
		return "This value is not allowed."
	}
	return strings.ToUpper(msg[:1]) + msg[1:] + "."
}

// sameFeatureValue reports whether in (form text or a JSON value) is the
// value r already has, so saving a page unchanged creates no overrides.
func sameFeatureValue(r features.Resolved, in any) bool {
	cur, err := json.Marshal(r.Value)
	if err != nil {
		return false
	}
	var raw []byte
	if s, ok := in.(string); ok {
		switch r.Type {
		case features.TypeString, features.TypeEnum:
			v, isStr := r.Value.(string)
			return isStr && v == s
		case features.TypeBool:
			b, err := strconv.ParseBool(strings.TrimSpace(s))
			if err != nil {
				return false
			}
			raw = []byte(strconv.FormatBool(b))
		default:
			raw = []byte(s)
		}
	} else if raw, err = json.Marshal(in); err != nil {
		return false
	}
	var a, b bytes.Buffer
	if json.Compact(&a, cur) != nil || json.Compact(&b, raw) != nil {
		return false
	}
	return bytes.Equal(a.Bytes(), b.Bytes())
}

// featureText is the value of r as form text.
func featureText(t features.Type, v any) string {
	switch t {
	case features.TypeBool:
		if b, ok := v.(bool); ok {
			return strconv.FormatBool(b)
		}
	case features.TypeString, features.TypeEnum:
		if s, ok := v.(string); ok {
			return s
		}
	case features.TypeInt:
		return fmt.Sprint(v)
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return ""
	}
	return string(out)
}

// contextLike is the request's context.Context, scoped to one tenant.
type contextLike = context.Context

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

type featurePage struct {
	title, description string
	action, resetURL   string
	items              []features.Resolved
	submitted          map[string]string
	errors             map[string]string
	notice, errMsg     string
	superAdmin         bool
	back               string
}

func (p *Panel) renderFeaturePage(ctx *Context, status int, pg featurePage) error {
	esc := html.EscapeString
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="filament-header"><div><h1 class="filament-title">%s</h1><p class="filament-description">%s</p></div>`, esc(pg.title), esc(pg.description))
	if pg.back != "" {
		fmt.Fprintf(&b, `<div><a href="%s" class="btn btn-secondary">All tenants</a></div>`, esc(pg.back))
	}
	b.WriteString(`</div>`)
	b.WriteString(alertHTML("ok", pg.notice))
	b.WriteString(alertHTML("err", pg.errMsg))
	if len(pg.items) == 0 {
		b.WriteString(`<div class="filament-card" style="padding: 1.25rem; color: var(--gray-500);">There are no settings to change.</div>`)
		return p.renderPanelPage(ctx, status, pg.title, b.String())
	}
	fmt.Fprintf(&b, `<form method="POST" action="%s" class="filament-feature-settings">%s`, esc(pg.action), p.csrfHiddenInput(ctx.Context, ""))
	group := "\x00"
	for _, r := range pg.items {
		if r.Group != group {
			if group != "\x00" {
				b.WriteString(`</div>`)
			}
			group = r.Group
			fmt.Fprintf(&b, `<div class="filament-card" data-feature-group="%s" style="padding: 1.25rem; margin-bottom: 1rem;"><h2 style="font-size: 1rem; font-weight: 600; margin: 0 0 0.5rem;">%s</h2>`,
				esc(r.Group), esc(orDefault(r.Group, "General")))
		}
		b.WriteString(p.featureFieldHTML(ctx, pg, r))
	}
	b.WriteString(`</div><div style="display: flex; justify-content: flex-end;"><button type="submit" class="btn btn-primary">Save</button></div></form>`)
	return p.renderPanelPage(ctx, status, pg.title, b.String())
}

func (p *Panel) featureFieldHTML(ctx *Context, pg featurePage, r features.Resolved) string {
	esc := html.EscapeString
	id := "feature-" + strings.NewReplacer(".", "-", "_", "-").Replace(r.Key)
	text, ok := pg.submitted[r.Key]
	if !ok {
		text = featureText(r.Type, r.Value)
	}
	label := orDefault(r.Label, r.Key)

	var control string
	switch r.Type {
	case features.TypeBool:
		checked := ""
		if b, err := strconv.ParseBool(text); err == nil && b {
			checked = " checked"
		}
		control = fmt.Sprintf(`<input type="hidden" name="%s" value="false" /><input type="checkbox" id="%s" name="%s" value="true"%s style="width: 1.25rem; height: 1.25rem; accent-color: var(--primary-600); cursor: pointer;" />`,
			esc(r.Key), id, esc(r.Key), checked)
	case features.TypeEnum:
		var opts strings.Builder
		for _, o := range r.Options {
			sel := ""
			if o == text {
				sel = " selected"
			}
			fmt.Fprintf(&opts, `<option value="%s"%s>%s</option>`, esc(o), sel, esc(o))
		}
		control = fmt.Sprintf(`<select id="%s" name="%s" class="form-control">%s</select>`, id, esc(r.Key), opts.String())
	case features.TypeInt:
		bounds := ""
		if r.Min != nil {
			bounds += fmt.Sprintf(` min="%d"`, *r.Min)
		}
		if r.Max != nil {
			bounds += fmt.Sprintf(` max="%d"`, *r.Max)
		}
		control = fmt.Sprintf(`<input type="number" step="1" id="%s" name="%s" value="%s" class="form-control"%s />`, id, esc(r.Key), esc(text), bounds)
	case features.TypeString:
		control = fmt.Sprintf(`<input type="text" id="%s" name="%s" value="%s" class="form-control" />`, id, esc(r.Key), esc(text))
	default: // json
		control = fmt.Sprintf(`<textarea id="%s" name="%s" rows="5" class="form-control" spellcheck="false" style="font-family: ui-monospace, monospace;">%s</textarea>`, id, esc(r.Key), esc(text))
	}

	badges := ""
	if r.Overridden {
		badges += ` <span class="badge badge-warning" data-overridden>Overridden</span>`
	} else {
		badges += ` <span class="badge badge-gray">Default</span>`
	}
	if pg.superAdmin && r.ManagedBy == features.SuperAdmin {
		badges += ` <span class="badge badge-info">Super admin only</span>`
	}
	helper := ""
	if r.Description != "" {
		helper = fmt.Sprintf(`<p class="form-helper">%s</p>`, esc(r.Description))
	}
	if !r.Overridden || r.Type == features.TypeJSON {
		// Nothing to add.
	} else {
		helper += fmt.Sprintf(`<p class="form-helper">Default: %s</p>`, esc(featureText(r.Type, r.Default)))
	}
	reset := ""
	if r.Overridden {
		reset = fmt.Sprintf(`<button type="submit" class="btn btn-secondary btn-sm" formaction="%s" formnovalidate name="_reset" value="%s">Reset to default</button>`,
			esc(pg.resetURL), esc(r.Key))
	}
	errHTML := ""
	cls := ""
	if msg := pg.errors[r.Key]; msg != "" {
		errHTML = fmt.Sprintf(`<p class="form-error">%s</p>`, esc(msg))
		cls = " has-error"
	}
	if r.Type == features.TypeBool {
		return fmt.Sprintf(`<div class="form-group%s" data-feature="%s" style="margin-top: 1rem;"><div style="display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap;">%s<label for="%s" style="font-size: 0.875rem; font-weight: 500; margin: 0; cursor: pointer;">%s</label>%s %s</div>%s%s</div>`,
			cls, esc(r.Key), control, id, esc(label), badges, reset, helper, errHTML)
	}
	return fmt.Sprintf(`<div class="form-group%s" data-feature="%s" style="margin-top: 1rem;"><div style="display: flex; align-items: center; gap: 0.5rem; flex-wrap: wrap;"><label class="form-label" for="%s" style="margin: 0;">%s</label>%s %s</div>%s%s%s</div>`,
		cls, esc(r.Key), id, esc(label), badges, reset, control, helper, errHTML)
}

// submittedValues reads the posted value of every listed feature (the last
// value wins, so a checked toggle beats its hidden "false").
func submittedValues(c *gcontext.Context, items []features.Resolved) (map[string]any, map[string]string) {
	_ = c.Request.ParseForm()
	values := map[string]any{}
	texts := map[string]string{}
	for _, r := range items {
		vs, ok := c.Request.PostForm[r.Key]
		if !ok || len(vs) == 0 {
			continue
		}
		v := vs[len(vs)-1]
		values[r.Key] = v
		texts[r.Key] = v
	}
	return values, texts
}

// ---------------------------------------------------------------------------
// Routes
// ---------------------------------------------------------------------------

// mountFeatureRoutes registers the web pages (web engine only).
func (p *Panel) mountFeatureRoutes(r *router.Router) {
	if p.tenantSettingsOn() {
		p.mountTenantSettings(r)
	}
	if p.featuresAdmin != nil {
		p.mountTenantFeaturesAdmin(r)
	}
}

// mountFeatureAPI registers the JSON endpoints (every engine).
func (p *Panel) mountFeatureAPI(r *router.Router) {
	if p.tenantSettingsOn() {
		p.mountTenantSettingsAPI(r)
	}
	if p.featuresAdmin != nil {
		p.mountTenantFeaturesAdminAPI(r)
	}
}

// tenantScope resolves the tenant settings request: the caller must be
// allowed and have a tenant. fctx is scoped to the panel's resolved tenant.
func (p *Panel) tenantScope(c *gcontext.Context) (*Context, contextLike, featureScope, error) {
	ctx, err := p.AuthorizeAccess(c)
	if err != nil {
		return ctx, nil, featureScope{}, err
	}
	api := p.isAPIRequest(c)
	if !p.canManageTenantSettings(ctx) {
		return ctx, nil, featureScope{}, p.featureDenied(c, api, http.StatusForbidden, "You may not change this tenant's settings.")
	}
	tenant := ctx.Tenant()
	if tenant == "" {
		return ctx, nil, featureScope{}, p.featureDenied(c, api, http.StatusForbidden, "Choose a tenant first.")
	}
	if want := c.Param("tenant"); want != "" && want != tenant {
		// A tenant in the path the user may not use: not the resolved one.
		return ctx, nil, featureScope{}, p.featureDenied(c, api, http.StatusNotFound, "Unknown tenant.")
	}
	if !api && c.Request.Method == http.MethodPost && !p.formGuard(c) {
		return ctx, nil, featureScope{}, denied(c.String(http.StatusForbidden, "403 Forbidden: your session expired"))
	}
	return ctx, features.ForTenant(ctx.GoContext(), tenant), featureScope{mgr: p.featureMgr}, nil
}

func (p *Panel) featureDenied(c *gcontext.Context, api bool, status int, msg string) error {
	if api {
		p.setCorsHeaders(c)
		return denied(c.JSON(status, map[string]string{"error": msg}))
	}
	return denied(c.String(status, "%d %s: %s", status, http.StatusText(status), msg))
}

// featureFailure answers a failed save or reset.
func (p *Panel) featureFailure(ctx *Context, api bool, err error) error {
	switch {
	case errors.Is(err, errFeatureForbidden):
		return p.featureDenied(ctx.Context, api, http.StatusForbidden, err.Error())
	case errors.Is(err, features.ErrUnknownFeature):
		return p.featureDenied(ctx.Context, api, http.StatusNotFound, "Unknown setting.")
	case errors.Is(err, privacy.ErrTenantMissing):
		return p.featureDenied(ctx.Context, api, http.StatusForbidden, "Choose a tenant first.")
	}
	slog.ErrorContext(ctx.GoContext(), "filament: saving tenant features failed", "error", err)
	return p.featureDenied(ctx.Context, api, http.StatusInternalServerError, "The settings could not be saved.")
}

var featureNotices = map[string]string{
	"saved": "Settings saved.",
	"reset": "The setting is back to its default.",
}

func (p *Panel) mountTenantSettings(r *router.Router) {
	slug := p.tenantSettings.slug()
	base := p.path + "/" + slug
	page := func(ctx *Context) featurePage {
		u := p.TenantURL(ctx, slug) // <panel>/<tenant>/<slug> with TenantPathScoping
		return featurePage{
			title:       orDefault(p.tenantSettings.Label, "Tenant settings"),
			description: "Settings for " + tenantName(ctx) + ". Fields marked Default follow the application default.",
			action:      u, resetURL: u + "/reset",
		}
	}
	show := func(ctx *Context, fctx contextLike, s featureScope, status int, pg featurePage) error {
		items, err := s.list(fctx)
		if err != nil {
			return p.featureFailure(ctx, false, err)
		}
		pg.items = items
		return p.renderFeaturePage(ctx, status, pg)
	}

	bases := []string{base}
	if p.tenantConfig != nil && p.tenantConfig.Enabled && p.tenantConfig.PathScoping {
		// The navigation links to <panel>/<tenant>/<slug>.
		bases = append(bases, p.path+"/:tenant/"+slug)
	}
	for _, base := range bases {
		p.mountTenantSettingsAt(r, base, page, show)
	}
}

func (p *Panel) mountTenantSettingsAt(r *router.Router, base string, page func(*Context) featurePage,
	show func(*Context, contextLike, featureScope, int, featurePage) error) {
	r.GET(base, func(c *gcontext.Context) error {
		ctx, fctx, s, err := p.tenantScope(c)
		if err != nil {
			return err
		}
		pg := page(ctx)
		pg.notice = featureNotices[c.Query("status")]
		return show(ctx, fctx, s, http.StatusOK, pg)
	})
	r.POST(base, func(c *gcontext.Context) error {
		ctx, fctx, s, err := p.tenantScope(c)
		if err != nil {
			return err
		}
		items, err := s.list(fctx)
		if err != nil {
			return p.featureFailure(ctx, false, err)
		}
		values, texts := submittedValues(c, items)
		errs, err := s.save(fctx, values)
		if err != nil {
			return p.featureFailure(ctx, false, err)
		}
		if len(errs) > 0 {
			pg := page(ctx)
			pg.submitted, pg.errors = texts, errs
			pg.errMsg = "Some settings were not saved. Fix the marked fields."
			return show(ctx, fctx, s, http.StatusUnprocessableEntity, pg)
		}
		return c.Redirect(http.StatusSeeOther, page(ctx).action+"?status=saved")
	})
	r.POST(base+"/reset", func(c *gcontext.Context) error {
		ctx, fctx, s, err := p.tenantScope(c)
		if err != nil {
			return err
		}
		if err := s.reset(fctx, c.FormValue("_reset")); err != nil {
			return p.featureFailure(ctx, false, err)
		}
		return c.Redirect(http.StatusSeeOther, page(ctx).action+"?status=reset")
	})

}

// mountTenantSettingsAPI is the tenant settings page's JSON API, for the
// Flutter client and headless tools.
func (p *Panel) mountTenantSettingsAPI(r *router.Router) {
	api := p.path + "/api/" + p.tenantSettings.slug()
	preflight := func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		return c.NoContent(http.StatusNoContent)
	}
	r.OPTIONS(api, preflight)
	r.OPTIONS(api+"/:key", preflight)
	list := func(c *gcontext.Context, ctx *Context, fctx contextLike, s featureScope, status int, errs map[string]string) error {
		items, err := s.list(fctx)
		if err != nil {
			return p.featureFailure(ctx, true, err)
		}
		body := map[string]any{"tenant": ctx.Tenant(), "groups": featureGroups(items)}
		if len(errs) > 0 {
			body["errors"] = errs
			body["message"] = "Some settings were not saved."
		}
		p.setCorsHeaders(c)
		return c.JSON(status, body)
	}
	r.GET(api, func(c *gcontext.Context) error {
		ctx, fctx, s, err := p.tenantScope(c)
		if err != nil {
			return err
		}
		return list(c, ctx, fctx, s, http.StatusOK, nil)
	})
	update := func(c *gcontext.Context) error {
		ctx, fctx, s, err := p.tenantScope(c)
		if err != nil {
			return err
		}
		values, err := decodeFeatureValues(c)
		if err != nil {
			return p.featureDenied(c, true, http.StatusBadRequest, err.Error())
		}
		errs, err := s.save(fctx, values)
		if err != nil {
			return p.featureFailure(ctx, true, err)
		}
		status := http.StatusOK
		if len(errs) > 0 {
			status = http.StatusUnprocessableEntity
		}
		return list(c, ctx, fctx, s, status, errs)
	}
	r.PUT(api, update)
	r.POST(api, update)
	resetAPI := func(c *gcontext.Context) error {
		ctx, fctx, s, err := p.tenantScope(c)
		if err != nil {
			return err
		}
		if err := s.reset(fctx, c.Param("key")); err != nil {
			return p.featureFailure(ctx, true, err)
		}
		return list(c, ctx, fctx, s, http.StatusOK, nil)
	}
	r.DELETE(api+"/:key", resetAPI)
	r.POST(api+"/:key/reset", resetAPI)
}

func tenantName(ctx *Context) string {
	if t := ctx.TenantRecord(); t != nil && t.GetTenantName() != "" {
		return t.GetTenantName()
	}
	return ctx.Tenant()
}

// decodeFeatureValues reads {"values": {key: value}} from a JSON body.
func decodeFeatureValues(c *gcontext.Context) (map[string]any, error) {
	var body struct {
		Values map[string]any `json:"values"`
	}
	if c.Request.Body == nil {
		return nil, errors.New("Send {\"values\": {...}}.")
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		return nil, errors.New("Invalid JSON body: send {\"values\": {...}}.")
	}
	if body.Values == nil {
		body.Values = map[string]any{}
	}
	return body.Values, nil
}

type featureGroupDTO struct {
	Name     string              `json:"name"`
	Features []features.Resolved `json:"features"`
}

func featureGroups(items []features.Resolved) []featureGroupDTO {
	out := []featureGroupDTO{}
	for _, r := range items {
		if len(out) == 0 || out[len(out)-1].Name != r.Group {
			out = append(out, featureGroupDTO{Name: r.Group})
		}
		g := &out[len(out)-1]
		g.Features = append(g.Features, r)
	}
	return out
}

// ---------------------------------------------------------------------------
// Central (super admin) screen
// ---------------------------------------------------------------------------

// adminTenant resolves the tenant named in the URL of the central screen:
// the user must be allowed, and with Tenants set, it must be listed.
func (p *Panel) adminTenant(c *gcontext.Context, key string) (*Context, Tenant, error) {
	ctx, err := p.AuthorizeAccess(c)
	if err != nil {
		return ctx, nil, err
	}
	api := p.isAPIRequest(c)
	if !p.canManageTenantFeatures(ctx) {
		return ctx, nil, p.featureDenied(c, api, http.StatusForbidden, "You may not change tenant features.")
	}
	if !api && c.Request.Method == http.MethodPost && !p.formGuard(c) {
		return ctx, nil, denied(c.String(http.StatusForbidden, "403 Forbidden: your session expired"))
	}
	if key == "" {
		return ctx, nil, nil
	}
	if p.featuresAdmin.Tenants == nil {
		return ctx, &TenantItem{Key: key, Name: key}, nil
	}
	tenants, err := p.featuresAdmin.Tenants(ctx)
	if err != nil {
		slog.ErrorContext(ctx.GoContext(), "filament: listing tenants failed", "error", err)
		return ctx, nil, p.featureDenied(c, api, http.StatusInternalServerError, "The tenants could not be listed.")
	}
	for _, t := range tenants {
		if t.GetTenantKey() == key {
			return ctx, t, nil
		}
	}
	return ctx, nil, p.featureDenied(c, api, http.StatusNotFound, "Unknown tenant.")
}

func (p *Panel) mountTenantFeaturesAdmin(r *router.Router) {
	cfg := p.featuresAdmin
	slug := cfg.slug()
	base := p.path + "/" + slug
	scope := featureScope{mgr: cfg.manager, superAdmin: true}
	title := orDefault(cfg.Label, "Tenant features")
	esc := html.EscapeString

	r.GET(base, func(c *gcontext.Context) error {
		ctx, _, err := p.adminTenant(c, "")
		if err != nil {
			return err
		}
		if t := strings.TrimSpace(c.Query("tenant")); t != "" {
			return c.Redirect(http.StatusSeeOther, base+"/"+url.PathEscape(t))
		}
		var b strings.Builder
		fmt.Fprintf(&b, `<div class="filament-header"><div><h1 class="filament-title">%s</h1><p class="filament-description">Pick a tenant to see and change its features, including those only the platform may change.</p></div></div>`, esc(title))
		if cfg.Tenants != nil {
			tenants, err := cfg.Tenants(ctx)
			if err != nil {
				return p.featureFailure(ctx, false, err)
			}
			b.WriteString(`<div class="filament-card"><table class="filament-table"><thead><tr><th>Tenant</th><th>Key</th><th></th></tr></thead><tbody>`)
			for _, t := range tenants {
				u := base + "/" + url.PathEscape(t.GetTenantKey())
				fmt.Fprintf(&b, `<tr data-tenant="%s"><td><a href="%s"><strong>%s</strong></a></td><td>%s</td><td style="text-align: end;"><a href="%s" class="btn btn-secondary btn-sm">Features</a></td></tr>`,
					esc(t.GetTenantKey()), esc(u), esc(t.GetTenantName()), esc(t.GetTenantKey()), esc(u))
			}
			if len(tenants) == 0 {
				b.WriteString(`<tr><td colspan="3" style="text-align: center; color: var(--gray-500);">No tenants.</td></tr>`)
			}
			b.WriteString(`</tbody></table></div>`)
		} else {
			fmt.Fprintf(&b, `<form method="GET" action="%s" class="filament-card" style="padding: 1.25rem; display: flex; gap: 0.5rem; align-items: end;"><div class="form-group" style="flex: 1; margin: 0;"><label class="form-label" for="tenant-key">Tenant key</label><input id="tenant-key" name="tenant" class="form-control" required /></div><button type="submit" class="btn btn-primary">Open</button></form>`, esc(base))
		}
		return p.renderPanelPage(ctx, http.StatusOK, title, b.String())
	})

	page := func(t Tenant) featurePage {
		u := base + "/" + url.PathEscape(t.GetTenantKey())
		return featurePage{
			title:       title + ": " + t.GetTenantName(),
			description: "Every feature of " + t.GetTenantName() + ", including those the tenant cannot change itself.",
			action:      u, resetURL: u + "/reset", superAdmin: true, back: base,
		}
	}
	show := func(ctx *Context, t Tenant, status int, pg featurePage) error {
		items, err := scope.list(features.ForTenant(ctx.GoContext(), t.GetTenantKey()))
		if err != nil {
			return p.featureFailure(ctx, false, err)
		}
		pg.items = items
		return p.renderFeaturePage(ctx, status, pg)
	}
	r.GET(base+"/:tenant", func(c *gcontext.Context) error {
		ctx, t, err := p.adminTenant(c, c.Param("tenant"))
		if err != nil {
			return err
		}
		pg := page(t)
		pg.notice = featureNotices[c.Query("status")]
		return show(ctx, t, http.StatusOK, pg)
	})
	r.POST(base+"/:tenant", func(c *gcontext.Context) error {
		ctx, t, err := p.adminTenant(c, c.Param("tenant"))
		if err != nil {
			return err
		}
		fctx := features.ForTenant(ctx.GoContext(), t.GetTenantKey())
		items, err := scope.list(fctx)
		if err != nil {
			return p.featureFailure(ctx, false, err)
		}
		values, texts := submittedValues(c, items)
		errs, err := scope.save(fctx, values)
		if err != nil {
			return p.featureFailure(ctx, false, err)
		}
		if len(errs) > 0 {
			pg := page(t)
			pg.submitted, pg.errors = texts, errs
			pg.errMsg = "Some features were not saved. Fix the marked fields."
			return show(ctx, t, http.StatusUnprocessableEntity, pg)
		}
		return c.Redirect(http.StatusSeeOther, page(t).action+"?status=saved")
	})
	r.POST(base+"/:tenant/reset", func(c *gcontext.Context) error {
		ctx, t, err := p.adminTenant(c, c.Param("tenant"))
		if err != nil {
			return err
		}
		if err := scope.reset(features.ForTenant(ctx.GoContext(), t.GetTenantKey()), c.FormValue("_reset")); err != nil {
			return p.featureFailure(ctx, false, err)
		}
		return c.Redirect(http.StatusSeeOther, page(t).action+"?status=reset")
	})

}

// mountTenantFeaturesAdminAPI is the central screen's JSON API.
func (p *Panel) mountTenantFeaturesAdminAPI(r *router.Router) {
	cfg := p.featuresAdmin
	scope := featureScope{mgr: cfg.manager, superAdmin: true}
	api := p.path + "/api/" + cfg.slug()
	preflight := func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		return c.NoContent(http.StatusNoContent)
	}
	r.OPTIONS(api, preflight)
	r.OPTIONS(api+"/:tenant", preflight)
	r.OPTIONS(api+"/:tenant/:key", preflight)
	// GET <panel>/api/tenant-features: the tenants to pick from. Without
	// TenantsConfig.Tenants the list is empty and free_entry is true (the
	// client asks for a tenant key).
	r.GET(api, func(c *gcontext.Context) error {
		ctx, _, err := p.adminTenant(c, "")
		if err != nil {
			return err
		}
		type tenantDTO struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		}
		out := []tenantDTO{}
		if cfg.Tenants != nil {
			tenants, err := cfg.Tenants(ctx)
			if err != nil {
				return p.featureFailure(ctx, true, err)
			}
			for _, t := range tenants {
				out = append(out, tenantDTO{Key: t.GetTenantKey(), Name: t.GetTenantName()})
			}
		}
		p.setCorsHeaders(c)
		return c.JSON(http.StatusOK, map[string]any{"tenants": out, "free_entry": cfg.Tenants == nil})
	})
	list := func(c *gcontext.Context, ctx *Context, t Tenant, status int, errs map[string]string) error {
		items, err := scope.list(features.ForTenant(ctx.GoContext(), t.GetTenantKey()))
		if err != nil {
			return p.featureFailure(ctx, true, err)
		}
		body := map[string]any{"tenant": t.GetTenantKey(), "groups": featureGroups(items)}
		if len(errs) > 0 {
			body["errors"] = errs
			body["message"] = "Some features were not saved."
		}
		p.setCorsHeaders(c)
		return c.JSON(status, body)
	}
	r.GET(api+"/:tenant", func(c *gcontext.Context) error {
		ctx, t, err := p.adminTenant(c, c.Param("tenant"))
		if err != nil {
			return err
		}
		return list(c, ctx, t, http.StatusOK, nil)
	})
	update := func(c *gcontext.Context) error {
		ctx, t, err := p.adminTenant(c, c.Param("tenant"))
		if err != nil {
			return err
		}
		values, err := decodeFeatureValues(c)
		if err != nil {
			return p.featureDenied(c, true, http.StatusBadRequest, err.Error())
		}
		errs, err := scope.save(features.ForTenant(ctx.GoContext(), t.GetTenantKey()), values)
		if err != nil {
			return p.featureFailure(ctx, true, err)
		}
		status := http.StatusOK
		if len(errs) > 0 {
			status = http.StatusUnprocessableEntity
		}
		return list(c, ctx, t, status, errs)
	}
	r.PUT(api+"/:tenant", update)
	r.POST(api+"/:tenant", update)
	resetAPI := func(c *gcontext.Context) error {
		ctx, t, err := p.adminTenant(c, c.Param("tenant"))
		if err != nil {
			return err
		}
		if err := scope.reset(features.ForTenant(ctx.GoContext(), t.GetTenantKey()), c.Param("key")); err != nil {
			return p.featureFailure(ctx, true, err)
		}
		return list(c, ctx, t, http.StatusOK, nil)
	}
	r.DELETE(api+"/:tenant/:key", resetAPI)
	r.POST(api+"/:tenant/:key/reset", resetAPI)
}
