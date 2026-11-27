package main

import (
	"context"
	"log"

	"github.com/golaravel/gofilament/pkg/filament"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/features"
	"github.com/golaravel/golaravel/pkg/privacy"
)

// The demo is one shop, so every request runs as this tenant. A real app
// resolves the tenant with Panel.Tenancy (or privacy.TenantMiddleware).
const demoTenant = "apex"

// Per-tenant features (golaravel pkg/features). The admin changes the
// TenantAdmin ones in the Flutter app under Account → Tenant settings.
var (
	shipFeature = features.Bool("orders.shipping", true).
			Label("Ship from the panel").Group("Orders").
			Description("Shows the Ship action on paid orders.")

	invoiceFeature = features.Bool("orders.invoices", true).
			Label("Printable invoices").Group("Orders").
			Description("Shows the Invoice action on orders.")

	// SuperAdmin: only the platform operator may change it (from a central
	// panel with TenantFeaturesAdmin), so it never shows on the shop's own
	// settings page.
	auditFeature = features.Bool("audit.log", true).
			Label("Audit log").Group("System").ManagedBy(features.SuperAdmin)

	featureMgr = newFeatureManager()
)

func newFeatureManager() *features.Manager {
	mgr := features.NewInMemory(features.Options{})
	if err := mgr.Register(shipFeature, invoiceFeature, auditFeature); err != nil {
		log.Fatalf("demo features: %v", err)
	}
	return mgr
}

// withDemoTenant puts the demo tenant into every panel request.
func withDemoTenant(c *gcontext.Context) error {
	c.Request = c.Request.WithContext(privacy.WithTenantID(c.Request.Context(), demoTenant))
	return nil
}

// resetFeatures puts every feature back to its default (with the data).
func resetFeatures() {
	ctx := features.ForTenant(context.Background(), demoTenant)
	for _, m := range featureMgr.Definitions() {
		_ = featureMgr.Reset(ctx, m.Key)
	}
}

var (
	canShip    = filament.FeatureEnabled(shipFeature)
	canInvoice = filament.FeatureEnabled(invoiceFeature)
	hasAudit   = filament.FeatureEnabled(auditFeature)
)
