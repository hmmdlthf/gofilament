package filament

import (
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/golaravel/golaravel/pkg/features"
	"github.com/golaravel/golaravel/pkg/privacy"
)

// ---------------------------------------------------------------------------
// Per-tenant features (github.com/golaravel/golaravel/pkg/features)
// ---------------------------------------------------------------------------

// Condition decides, for one request, whether something is shown and
// allowed. The tenant comes from the request context (privacy.WithTenantID),
// which the panel sets for every request once it has resolved the tenant.
//
// Conditions built by FeatureEnabled, FeatureIs, FeatureIn and
// FeatureDisabled fail closed: when the value cannot be read (no tenant,
// database down, feature not registered) they return false, so a broken
// lookup hides a feature rather than showing it.
//
// The same Condition gates every kind of panel item through a method named
// RequiresFeature:
//
//	resource.RequiresFeature(filament.FeatureEnabled(appfeatures.RecurringInvoices))
//	flutterPage.RequiresFeature(...)
//	widget.RequiresFeature(...)                 // widgets.BaseWidget
//	tables.Action[T]("repeat").RequiresFeature(...)
//	tables.TextColumn[T]("interval").RequiresFeature(...)
//	forms.Select("interval").RequiresFeature(...)
//	&filament.NavigationItem{..., Feature: ...}
//
// (VisibleWhen already means something else on forms and widgets: a
// client-side field condition and a user check.)
type Condition func(ctx context.Context) bool

// FeatureEnabled holds when the bool feature f is on for the request's
// tenant.
func FeatureEnabled(f *features.Feature[bool]) Condition {
	return func(ctx context.Context) bool { return ctx != nil && features.Enabled(ctx, f) }
}

// FeatureDisabled holds when the bool feature f is known to be off for the
// request's tenant. Like the others it fails closed: a failed lookup is
// neither enabled nor disabled.
func FeatureDisabled(f *features.Feature[bool]) Condition {
	return FeatureIs(f, false)
}

// FeatureIs holds when f's value for the request's tenant equals want:
//
//	filament.FeatureIs(appfeatures.TaxMode, "exclusive")
func FeatureIs[T comparable](f *features.Feature[T], want T) Condition {
	return func(ctx context.Context) bool { return ctx != nil && features.Is(ctx, f, want) }
}

// FeatureIn holds when f's value for the request's tenant is one of values.
func FeatureIn[T comparable](f *features.Feature[T], values ...T) Condition {
	return func(ctx context.Context) bool {
		if ctx == nil {
			return false
		}
		v, err := f.Get(ctx)
		return err == nil && slices.Contains(values, v)
	}
}

// AllOf holds when every condition holds.
func AllOf(conds ...Condition) Condition {
	return func(ctx context.Context) bool {
		for _, c := range conds {
			if c == nil || !c(ctx) {
				return false
			}
		}
		return true
	}
}

// AnyOf holds when at least one condition holds.
func AnyOf(conds ...Condition) Condition {
	return func(ctx context.Context) bool {
		for _, c := range conds {
			if c != nil && c(ctx) {
				return true
			}
		}
		return false
	}
}

// Allows reports whether the condition holds for the panel request. A nil
// Condition always holds.
func (c Condition) Allows(ctx *Context) bool {
	if c == nil {
		return true
	}
	if ctx == nil {
		return false
	}
	return c(ctx.GoContext())
}

// FeatureGated is implemented by resources and pages that are only
// available while a Condition holds (BaseResource and FlutterPageDef do,
// through RequiresFeature). A custom PageInterface can implement it too.
// The panel then leaves the item out of the navigation, the API schema and
// global search, and answers 404 to every request under its URL, web and
// API alike.
type FeatureGated interface {
	FeatureGate() Condition
}

// gateAllows applies v's feature gate, when it has one.
func gateAllows(ctx *Context, v any) bool {
	g, ok := v.(FeatureGated)
	if !ok {
		return true
	}
	return g.FeatureGate().Allows(ctx)
}

// Features gives the panel a feature manager. It adds the tenant settings
// page (<panel>/tenant-settings, see TenantSettings) and its API, and sends
// the tenant's Public features to API clients as "features" in
// <panel>/api/schema. A central panel uses TenantFeaturesAdmin instead to
// edit any tenant's features.
//
// Gating with FeatureEnabled and friends works without it: features know
// their manager from Register.
func (p *Panel) Features(mgr *features.Manager) *Panel {
	p.featureMgr = mgr
	if p.tenantSettings == nil {
		p.tenantSettings = &TenantSettingsConfig{}
	}
	return p
}

// FeatureManager returns the manager set with Features or
// TenantFeaturesAdmin (nil if none).
func (p *Panel) FeatureManager() *features.Manager { return p.featureMgr }

// featureBlocked reports whether reqPath belongs to a resource or page
// whose feature gate fails for this request.
func (p *Panel) featureBlocked(ctx *Context, reqPath string) bool {
	base := strings.TrimRight(p.path, "/")
	if base != "" && reqPath != base && !strings.HasPrefix(reqPath, base+"/") {
		return false
	}
	rest := strings.TrimPrefix(reqPath, base)
	under := func(prefix string) bool {
		return rest == prefix || strings.HasPrefix(rest, prefix+"/")
	}
	for _, res := range p.resources {
		if _, ok := res.(FeatureGated); !ok {
			continue
		}
		slug := res.Slug()
		if (under("/"+slug) || under("/api/resources/"+slug)) && !gateAllows(ctx, res) {
			return true
		}
	}
	for _, pg := range p.pages {
		if _, ok := pg.(FeatureGated); !ok {
			continue
		}
		slug := pg.Slug()
		if (under("/"+slug) || under("/api/pages/"+slug)) && !gateAllows(ctx, pg) {
			return true
		}
	}
	return false
}

// refuseFeature answers a request to a feature-gated item that is off.
func (p *Panel) refuseFeature(ctx *Context) error {
	if p.isAPIRequest(ctx.Context) {
		p.setCorsHeaders(ctx.Context)
		return denied(ctx.JSON(http.StatusNotFound, map[string]string{"error": "Not Found"}))
	}
	return denied(ctx.String(http.StatusNotFound, "404 Not Found"))
}

// verifiedTenant is the request's tenant as the panel resolved and checked
// it (or as trusted middleware set it in the privacy context). Unlike
// Context.Tenant it never falls back to an unchecked :tenant path segment.
func verifiedTenant(ctx *Context) string {
	if ctx == nil {
		return ""
	}
	if t := ctx.TenantRecord(); t != nil {
		return t.GetTenantKey()
	}
	if t, ok := privacy.TenantFromContext(ctx.GoContext()); ok {
		return t
	}
	return ""
}

// featureSchema adds the tenant's Public features and the settings screens
// to <panel>/api/schema, for the Flutter client.
func (p *Panel) featureSchema(ctx *Context, schema map[string]any) {
	if p.featureMgr == nil {
		return
	}
	pub := map[string]any{}
	if verifiedTenant(ctx) != "" {
		if v, err := p.featureMgr.Public(features.ForTenant(ctx.GoContext(), verifiedTenant(ctx))); err == nil {
			pub = v
		}
	}
	schema["features"] = pub
	ts := map[string]any{"enabled": false}
	if p.canManageTenantSettings(ctx) && verifiedTenant(ctx) != "" {
		ts = map[string]any{
			"enabled":  true,
			"label":    orDefault(p.tenantSettings.Label, "Tenant settings"),
			"endpoint": p.path + "/api/" + p.tenantSettings.slug(),
			"path":     "/" + p.tenantSettings.slug(), // relative to <panel>/api
		}
	}
	schema["tenant_settings"] = ts
	if p.canManageTenantFeatures(ctx) {
		schema["tenant_features_admin"] = map[string]any{
			"enabled":  true,
			"label":    orDefault(p.featuresAdmin.Label, "Tenant features"),
			"endpoint": p.path + "/api/" + p.featuresAdmin.slug(),
			"path":     "/" + p.featuresAdmin.slug(),
		}
	}
}
