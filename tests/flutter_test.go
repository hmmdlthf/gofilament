package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/commands"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/flutterclient"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type MockProduct struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"`
}

func TestPanelEngineConfiguration(t *testing.T) {
	// 1. Default is Web
	pWeb := filament.NewPanel("admin")
	if pWeb.GetEngine() != filament.EngineWeb {
		t.Fatalf("expected EngineWeb by default, got %s", pWeb.GetEngine())
	}
	if pWeb.IsFlutter() {
		t.Fatalf("expected IsFlutter to be false for default panel")
	}

	// 2. Flutter mode configuration
	pFlutter := filament.NewPanel("admin").
		Flutter("custom/client/build/web")

	if pFlutter.GetEngine() != filament.EngineFlutter {
		t.Fatalf("expected EngineFlutter, got %s", pFlutter.GetEngine())
	}
	if !pFlutter.IsFlutter() {
		t.Fatalf("expected IsFlutter to be true")
	}
	if pFlutter.GetFlutterDistDir() != "custom/client/build/web" {
		t.Fatalf("expected custom flutter dist dir, got %s", pFlutter.GetFlutterDistDir())
	}
}

func TestPanelAPISchemaAndResourceCRUD(t *testing.T) {
	app := golaravel.New()

	productStore := map[string]*MockProduct{
		"1": {ID: "1", Name: "Mechanical Keyboard", Price: 150},
		"2": {ID: "2", Name: "Ergonomic Mouse", Price: 80},
	}

	productRes := filament.NewResource[*MockProduct]("products").
		ModelName("Product").
		OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*MockProduct, int, error) {
			var list []*MockProduct
			for _, p := range productStore {
				if search == "" || strings.Contains(strings.ToLower(p.Name), strings.ToLower(search)) {
					list = append(list, p)
				}
			}
			return list, len(list), nil
		}).
		OnFind(func(ctx *filament.Context, id string) (*MockProduct, error) {
			if p, ok := productStore[id]; ok {
				return p, nil
			}
			return nil, os.ErrNotExist
		}).
		OnCreate(func(ctx *filament.Context, data map[string]any) error {
			id := "3"
			name := fmt.Sprintf("%v", data["name"])
			productStore[id] = &MockProduct{ID: id, Name: name, Price: 200}
			return nil
		}).
		OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
			if p, ok := productStore[id]; ok {
				if name, exists := data["name"]; exists {
					p.Name = fmt.Sprintf("%v", name)
				}
			}
			return nil
		}).
		OnDelete(func(ctx *filament.Context, id string) error {
			delete(productStore, id)
			return nil
		})

	panel := filament.NewPanel("admin").
		Path("/admin").
		Flutter().
		BrandName("TechMart Admin").
		Tenancy().
		TenantLoader(func(ctx *filament.Context) ([]filament.Tenant, error) {
			return []filament.Tenant{
				filament.TenantItem{Key: "us_store", Name: "US Store"},
				filament.TenantItem{Key: "eu_store", Name: "EU Store"},
			}, nil
		}).
		Resource(productRes)

	_ = panel.Register(app)
	client := app.Test(t)

	// 1. GET /admin/api/schema
	schemaResp := client.Get("/admin/api/schema")
	schemaResp.AssertOK()

	var schema map[string]any
	if err := json.Unmarshal(schemaResp.Body(), &schema); err != nil {
		t.Fatalf("failed to decode schema JSON: %v", err)
	}
	if schema["brand_name"] != "TechMart Admin" {
		t.Fatalf("expected TechMart Admin, got %v", schema["brand_name"])
	}
	if schema["engine"] != "flutter" {
		t.Fatalf("expected engine flutter, got %v", schema["engine"])
	}
	if schema["active_tenant"] != "us_store" {
		t.Fatalf("expected us_store active tenant, got %v", schema["active_tenant"])
	}

	// 2. GET /admin/api/resources/products (List)
	listResp := client.Get("/admin/api/resources/products")
	listResp.AssertOK()
	var listData map[string]any
	_ = json.Unmarshal(listResp.Body(), &listData)
	if int(listData["total"].(float64)) != 2 {
		t.Fatalf("expected 2 products, got %v", listData["total"])
	}

	// 3. GET /admin/api/resources/products/1 (Find)
	findResp := client.Get("/admin/api/resources/products/1")
	findResp.AssertOK().AssertSee("Mechanical Keyboard")

	// 4. POST /admin/api/resources/products (Create)
	createResp := client.PostJSON("/admin/api/resources/products", map[string]any{
		"name":  "UltraWide Monitor",
		"price": 600,
	})
	createResp.AssertStatus(http.StatusCreated)

	if len(productStore) != 3 || productStore["3"] == nil || productStore["3"].Name != "UltraWide Monitor" {
		t.Fatalf("expected product 3 to be created, store: %v", productStore)
	}

	// 5. PUT /admin/api/resources/products/3 (Update)
	updateResp := client.Request(http.MethodPut, "/admin/api/resources/products/3").
		WithJSON(map[string]any{"name": "UltraWide Gaming Monitor"}).
		Send()
	updateResp.AssertOK()
	if productStore["3"].Name != "UltraWide Gaming Monitor" {
		t.Fatalf("expected updated name, got %s", productStore["3"].Name)
	}

	// 6. DELETE /admin/api/resources/products/3 (Delete)
	delResp := client.Request(http.MethodDelete, "/admin/api/resources/products/3").Send()
	delResp.AssertOK()
	if _, exists := productStore["3"]; exists {
		t.Fatalf("expected product 3 to be deleted")
	}

	// 7. GET /admin/api/tenants & Switch
	tenantsResp := client.Get("/admin/api/tenants")
	tenantsResp.AssertOK().AssertSee("US Store").AssertSee("EU Store")

	switchResp := client.PostJSON("/admin/api/tenants/switch", map[string]string{
		"tenant": "eu_store",
	})
	switchResp.AssertOK().AssertSee("eu_store")
}

func TestFlutterWebServingFallback(t *testing.T) {
	app := golaravel.New()

	panel := filament.NewPanel("admin").
		Path("/admin").
		Flutter("non_existent_flutter_path/build/web").
		BrandName("CloudPanel")

	_ = panel.Register(app)
	client := app.Test(t)

	// Since Flutter build web is not present, serves guidance landing page
	resp := client.Get("/admin")
	resp.AssertOK().
		AssertSee("CloudPanel Flutter Admin Active").
		AssertSee("flutter build web").
		AssertSee("Mobile-First")
}

func TestFlutterWebCompiledServing(t *testing.T) {
	app := golaravel.New()

	tmpDist, err := os.MkdirTemp("", "flutter_web_dist_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDist)

	indexPath := filepath.Join(tmpDist, "index.html")
	_ = os.WriteFile(indexPath, []byte(`<!DOCTYPE html><html><head><base href="/"></head><body><h1>Flutter Compiled SPA</h1></body></html>`), 0644)

	panel := filament.NewPanel("admin").
		Path("/admin").
		Flutter(tmpDist).
		BrandName("CompiledApp")

	_ = panel.Register(app)
	client := app.Test(t)

	// Serves compiled Flutter Web SPA
	resp := client.Get("/admin/")
	resp.AssertOK().
		AssertSee("Flutter Compiled SPA").
		AssertSee(`<base href="/admin/">`)

	// Schema API still accessible alongside Flutter SPA
	apiResp := client.Get("/admin/api/schema")
	apiResp.AssertOK().AssertSee("CompiledApp")
}

func TestFlutterClientScaffoldGenerator(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "filament_flutter_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = flutterclient.GenerateFlutterAdmin(tmpDir, "Operations", "/admin")
	if err != nil {
		t.Fatalf("failed to generate Flutter admin client: %v", err)
	}

	// Verify required Flutter project files exist
	pubspecPath := filepath.Join(tmpDir, "pubspec.yaml")
	if _, err := os.Stat(pubspecPath); err != nil {
		t.Fatalf("expected pubspec.yaml to exist: %v", err)
	}

	mainDartPath := filepath.Join(tmpDir, "lib", "main.dart")
	mainContent, err := os.ReadFile(mainDartPath)
	if err != nil {
		t.Fatalf("expected lib/main.dart to exist: %v", err)
	}

	mainStr := string(mainContent)
	// Check Mobile-First Responsive widgets and components
	if !strings.Contains(mainStr, "ResponsiveLayout") {
		t.Fatalf("expected ResponsiveLayout in main.dart")
	}
	if !strings.Contains(mainStr, "NavigationBar") {
		t.Fatalf("expected NavigationBar (mobile bottom nav) in main.dart")
	}
	if !strings.Contains(mainStr, "RefreshIndicator") {
		t.Fatalf("expected pull-to-refresh RefreshIndicator in main.dart")
	}
	if !strings.Contains(mainStr, "FloatingActionButton") {
		t.Fatalf("expected FAB for mobile record creation in main.dart")
	}
	if !strings.Contains(mainStr, "showModalBottomSheet") {
		t.Fatalf("expected mobile bottom sheet forms in main.dart")
	}
	if !strings.Contains(mainStr, "_buildDesktopSidebar") {
		t.Fatalf("expected desktop _buildDesktopSidebar in main.dart")
	}

	apiDartPath := filepath.Join(tmpDir, "lib", "api.dart")
	if _, err := os.Stat(apiDartPath); err != nil {
		t.Fatalf("expected lib/api.dart to exist: %v", err)
	}
}

func TestArtisanFlutterCommands(t *testing.T) {
	// 1. Test make:filament-panel with --engine=flutter
	cmdPanel := commands.MakeFilamentPanelCommand()
	err := cmdPanel.Run([]string{"MobileStore", "--engine=flutter"})
	if err != nil {
		t.Fatalf("make:filament-panel failed: %v", err)
	}

	panelPath := filepath.Join("app", "filament", "panels", "mobilestore_panel.go")
	defer os.Remove(panelPath)

	panelContent, err := os.ReadFile(panelPath)
	if err != nil {
		t.Fatalf("failed to read generated panel: %v", err)
	}
	if !strings.Contains(string(panelContent), "Engine(filament.EngineFlutter)") {
		t.Fatalf("expected Engine(filament.EngineFlutter) in generated panel, got: %s", string(panelContent))
	}

	// 2. Test make:filament-flutter
	cmdFlutter := commands.MakeFilamentFlutterCommand()
	tmpClientDir, _ := os.MkdirTemp("", "test_client_*")
	defer os.RemoveAll(tmpClientDir)

	err = cmdFlutter.Run([]string{"MobileStore", "--dir=" + tmpClientDir})
	if err != nil {
		t.Fatalf("make:filament-flutter failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpClientDir, "lib", "main.dart")); err != nil {
		t.Fatalf("expected generated Flutter client in %s: %v", tmpClientDir, err)
	}
}
