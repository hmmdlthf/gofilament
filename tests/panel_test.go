package tests

import (
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func TestPanelMountAndDashboard(t *testing.T) {
	app := golaravel.New()

	panel := filament.NewPanel("admin").
		Path("/admin").
		BrandName("GoLaravel Cloud").
		DarkMode(true).
		NavigationGroups([]string{
			"Shop Management",
			"System & Settings",
		})

	err := panel.Register(app)
	if err != nil {
		t.Fatalf("Failed to register panel: %v", err)
	}

	client := app.Test(t)

	// Test GET /admin dashboard
	resp := client.Get("/admin")
	resp.AssertOK().
		AssertSee("GoLaravel Cloud").
		AssertSee("Dashboard").
		AssertSee("filament-sidebar")
}

func TestPanelEmbeddedAssets(t *testing.T) {
	app := golaravel.New()
	panel := filament.NewPanel("admin").Path("/admin")
	_ = panel.Register(app)

	client := app.Test(t)

	// 1. Test CSS asset
	client.Get("/admin/assets/filament.css").
		AssertOK().
		AssertSee("filament-shell")

	// 2. Test JS asset
	client.Get("/admin/assets/filament.js").
		AssertOK().
		AssertSee("Filament")
}
