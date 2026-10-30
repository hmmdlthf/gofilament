package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/commands"
)

func TestArtisanFilamentGenerators(t *testing.T) {
	tempDir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(tempDir)
	defer os.Chdir(origDir)

	// 1. make:filament-panel
	panelCmd := commands.MakeFilamentPanelCommand()
	if err := panelCmd.Run([]string{"App"}); err != nil {
		t.Fatalf("make:filament-panel failed: %v", err)
	}
	panelFile := filepath.Join("app", "filament", "panels", "app_panel.go")
	panelContent, err := os.ReadFile(panelFile)
	if err != nil {
		t.Fatalf("Expected panel file %s to exist: %v", panelFile, err)
	}
	if !strings.Contains(string(panelContent), `filament.NewPanel("app")`) {
		t.Errorf("Expected generated panel to define panel 'app'")
	}

	// 2. make:filament-resource
	resCmd := commands.MakeFilamentResourceCommand()
	if err := resCmd.Run([]string{"Customer"}); err != nil {
		t.Fatalf("make:filament-resource failed: %v", err)
	}
	resFile := filepath.Join("app", "filament", "resources", "customer_resource.go")
	resContent, err := os.ReadFile(resFile)
	if err != nil {
		t.Fatalf("Expected resource file %s to exist: %v", resFile, err)
	}
	if !strings.Contains(string(resContent), `type CustomerResource struct`) {
		t.Errorf("Expected CustomerResource struct definition in generated file")
	}

	// 3. make:filament-page
	pageCmd := commands.MakeFilamentPageCommand()
	if err := pageCmd.Run([]string{"Settings"}); err != nil {
		t.Fatalf("make:filament-page failed: %v", err)
	}
	pageFile := filepath.Join("app", "filament", "pages", "settings.go")
	pageContent, err := os.ReadFile(pageFile)
	if err != nil {
		t.Fatalf("Expected page file %s to exist: %v", pageFile, err)
	}
	if !strings.Contains(string(pageContent), `type SettingsPage struct`) {
		t.Errorf("Expected SettingsPage struct in generated file")
	}

	// 4. make:filament-widget
	widgetCmd := commands.MakeFilamentWidgetCommand()
	if err := widgetCmd.Run([]string{"Revenue"}); err != nil {
		t.Fatalf("make:filament-widget failed: %v", err)
	}
	widgetFile := filepath.Join("app", "filament", "widgets", "revenue_widget.go")
	widgetContent, err := os.ReadFile(widgetFile)
	if err != nil {
		t.Fatalf("Expected widget file %s to exist: %v", widgetFile, err)
	}
	if !strings.Contains(string(widgetContent), `type RevenueWidget struct`) {
		t.Errorf("Expected RevenueWidget struct in generated file")
	}

	// 5. make:filament-relation
	relCmd := commands.MakeFilamentRelationCommand()
	if err := relCmd.Run([]string{"Customer", "Orders"}); err != nil {
		t.Fatalf("make:filament-relation failed: %v", err)
	}
	relFile := filepath.Join("app", "filament", "resources", "customer", "relations", "orders_relation_manager.go")
	relContent, err := os.ReadFile(relFile)
	if err != nil {
		t.Fatalf("Expected relation file %s to exist: %v", relFile, err)
	}
	if !strings.Contains(string(relContent), `type OrdersRelationManager struct`) {
		t.Errorf("Expected OrdersRelationManager struct in generated file")
	}
}
