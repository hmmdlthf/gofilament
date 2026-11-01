package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/colors"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func createTestFlutterAdminApp(t *testing.T) *golaravel.Engine {
	seedDatabase()

	app := golaravel.Default()

	tmpDir, err := os.MkdirTemp("", "test_flutter_admin_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	webDist := filepath.Join(tmpDir, "build", "web")
	_ = os.MkdirAll(webDist, 0755)
	_ = os.WriteFile(filepath.Join(webDist, "index.html"), []byte(`<!DOCTYPE html><html><head><base href="/admin/"></head><body><h1>Pulse Fleet - Flutter Multiplatform Admin</h1></body></html>`), 0644)

	adminPanel := filament.NewPanel("admin").
		Path("/admin").
		BrandName("Pulse Fleet").
		Engine(filament.EngineFlutter).
		Flutter(webDist).
		Colors(map[string]any{
			colors.RolePrimary: colors.Amber,
			colors.RoleGray:    colors.Slate,
		})

	adminPanel.Resource(buildDeviceResource())
	adminPanel.Resource(buildTelemetryResource())
	adminPanel.Resource(buildTicketResource())
	adminPanel.Resource(buildFirmwareResource())
	adminPanel.Resource(buildFleetUserResource())

	if err := adminPanel.Register(app); err != nil {
		t.Fatalf("Failed to mount flutter panel: %v", err)
	}

	return app
}

func TestFlutterAdmin_EngineAndWebServing(t *testing.T) {
	app := createTestFlutterAdminApp(t)

	// 1. Root /admin/ serves Flutter Web SPA
	req := httptest.NewRequest("GET", "/admin/", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Pulse Fleet - Flutter Multiplatform Admin") {
		t.Errorf("expected Flutter SPA title in served index.html")
	}
}

func TestFlutterAdmin_JSONAPISchema(t *testing.T) {
	app := createTestFlutterAdminApp(t)

	// 2. GET /admin/api/schema serves dynamic metadata for Flutter client
	req := httptest.NewRequest("GET", "/admin/api/schema", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var schema map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &schema); err != nil {
		t.Fatalf("invalid json schema: %v", err)
	}

	if schema["brand_name"] != "Pulse Fleet" {
		t.Errorf("expected brand_name 'Pulse Fleet', got %v", schema["brand_name"])
	}
	if schema["account_nav_placement"] != "header" {
		t.Errorf("expected account_nav_placement 'header', got %v", schema["account_nav_placement"])
	}
	if schema["engine"] != "flutter" {
		t.Errorf("expected engine 'flutter', got %v", schema["engine"])
	}

	resources, ok := schema["resources"].([]any)
	if !ok || len(resources) < 5 {
		t.Fatalf("expected 5 resources in schema (devices, telemetry, tickets, firmware, users), got %v", schema["resources"])
	}

	// Verify CORS headers are present on schema response
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Errorf("expected Access-Control-Allow-Origin header on schema response")
	}
}

func TestFlutterAdmin_ResourcesJSONAPI(t *testing.T) {
	app := createTestFlutterAdminApp(t)

	// 3. GET /admin/api/resources/devices
	req := httptest.NewRequest("GET", "/admin/api/resources/devices", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resData map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resData); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	items, ok := resData["data"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected devices in data array, got %v", resData)
	}

	// Verify device fields
	firstDev := items[0].(map[string]any)
	if firstDev["name"] == "" || firstDev["status"] == "" {
		t.Errorf("expected device to have name and status: %v", firstDev)
	}

	// 4. GET /admin/api/resources/users
	reqUsers := httptest.NewRequest("GET", "/admin/api/resources/users", nil)
	wUsers := httptest.NewRecorder()
	app.ServeHTTP(wUsers, reqUsers)

	if wUsers.Code != http.StatusOK {
		t.Fatalf("expected status 200 for users, got %d", wUsers.Code)
	}

	var usersData map[string]any
	_ = json.Unmarshal(wUsers.Body.Bytes(), &usersData)
	userItems := usersData["data"].([]any)
	if len(userItems) == 0 {
		t.Errorf("expected fleet users in data array")
	}

	// 5. POST /admin/api/resources/devices - Create Device
	newDevPayload := `{"name":"Edge-Unit-99","serial_no":"SN-9999-X","model":"EdgeCore-Pro","location":"Paris DC","status":"online","battery_level":95}`
	reqCreate := httptest.NewRequest("POST", "/admin/api/resources/devices", strings.NewReader(newDevPayload))
	reqCreate.Header.Set("Content-Type", "application/json")
	wCreate := httptest.NewRecorder()
	app.ServeHTTP(wCreate, reqCreate)

	if wCreate.Code != http.StatusCreated && wCreate.Code != http.StatusOK {
		t.Fatalf("expected status 201/200 on device create, got %d: %s", wCreate.Code, wCreate.Body.String())
	}

	// Verify newly created device exists in list
	reqCheck := httptest.NewRequest("GET", "/admin/api/resources/devices?search=Edge-Unit-99", nil)
	wCheck := httptest.NewRecorder()
	app.ServeHTTP(wCheck, reqCheck)
	if !strings.Contains(wCheck.Body.String(), "Edge-Unit-99") {
		t.Errorf("expected created device in search response: %s", wCheck.Body.String())
	}

	// 6. DELETE /admin/api/resources/devices/1 - Delete Device
	reqDel := httptest.NewRequest("DELETE", "/admin/api/resources/devices/1", nil)
	wDel := httptest.NewRecorder()
	app.ServeHTTP(wDel, reqDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("expected status 200 on delete, got %d", wDel.Code)
	}

	// 7. Test Telemetry, Tickets, and Firmware endpoints
	for _, slug := range []string{"telemetry", "tickets", "firmware"} {
		reqSlug := httptest.NewRequest("GET", "/admin/api/resources/"+slug, nil)
		wSlug := httptest.NewRecorder()
		app.ServeHTTP(wSlug, reqSlug)
		if wSlug.Code != http.StatusOK {
			t.Fatalf("expected status 200 for %s, got %d", slug, wSlug.Code)
		}
		if wSlug.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Errorf("expected Access-Control-Allow-Origin on %s endpoint", slug)
		}
	}

	// 8. Test OPTIONS preflight request
	reqOptions := httptest.NewRequest("OPTIONS", "/admin/api/resources/devices", nil)
	wOptions := httptest.NewRecorder()
	app.ServeHTTP(wOptions, reqOptions)
	if wOptions.Code != http.StatusNoContent && wOptions.Code != http.StatusOK {
		t.Fatalf("expected 204 or 200 on OPTIONS preflight, got %d", wOptions.Code)
	}
	if wOptions.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Errorf("expected Access-Control-Allow-Origin on preflight response")
	}
}

func TestEnsureFlutterClientReady(t *testing.T) {
	clientDir := "client"
	err := ensureFlutterClientReady(clientDir, "Pulse", "http://localhost:8081/admin")
	if err != nil {
		t.Fatalf("ensureFlutterClientReady failed: %v", err)
	}

	distIndexPath := filepath.Join(clientDir, "build", "web", "index.html")
	content, err := os.ReadFile(distIndexPath)
	if err != nil {
		t.Fatalf("failed to read generated index.html: %v", err)
	}

	if !strings.Contains(string(content), "account-btn") {
		t.Errorf("expected index.html to contain account-btn")
	}
	if !strings.Contains(string(content), "floating-account-sheet") {
		t.Errorf("expected index.html to contain floating-account-sheet")
	}
	if !strings.Contains(string(content), "modal-settings") {
		t.Errorf("expected index.html to contain modal-settings")
	}
	if !strings.Contains(string(content), "setting-account-placement") {
		t.Errorf("expected index.html to contain setting-account-placement")
	}
	if !strings.Contains(string(content), "pane-account") {
		t.Errorf("expected index.html to contain pane-account")
	}
}
