package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/notifications"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type CustomerRecord struct {
	ID    string
	Name  string
	Email string
	Role  string
}

func setupImportExportApp() (*golaravel.Engine, map[string]*CustomerRecord, *sync.RWMutex) {
	app := golaravel.New()
	panel := filament.NewPanel("admin").Path("/admin")

	var mu sync.RWMutex
	store := map[string]*CustomerRecord{
		"1": {ID: "1", Name: "Alice Cooper", Email: "alice@example.com", Role: "admin"},
		"2": {ID: "2", Name: "Bob Dylan", Email: "bob@example.com", Role: "editor"},
		"3": {ID: "3", Name: "Charlie Brown", Email: "charlie@example.com", Role: "viewer"},
	}
	nextID := 4

	customerResource := filament.NewResource[*CustomerRecord]("customers").
		ModelName("Customer").
		PluralModelName("Customers").
		Table(func(table *tables.Table[*CustomerRecord]) *tables.Table[*CustomerRecord] {
			return table.
				Columns(
					tables.TextColumn[*CustomerRecord]("id").Label("Customer ID"),
					tables.TextColumn[*CustomerRecord]("name").Label("Full Name"),
					tables.TextColumn[*CustomerRecord]("email").Label("Email Address"),
					tables.BadgeColumn[*CustomerRecord]("role").Label("System Role"),
				).
				HeaderActions(
					tables.ImportAction[*CustomerRecord](),
					tables.ExportAction[*CustomerRecord](),
				).
				BulkActions(
					tables.ExportBulkAction(),
				)
		}).
		RecordID(func(record *CustomerRecord) string {
			return record.ID
		}).
		OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*CustomerRecord, int, error) {
			mu.RLock()
			defer mu.RUnlock()
			var list []*CustomerRecord
			for _, c := range store {
				if search != "" && !strings.Contains(strings.ToLower(c.Name), strings.ToLower(search)) {
					continue
				}
				list = append(list, c)
			}
			return list, len(list), nil
		}).
		OnCreate(func(ctx *filament.Context, data map[string]any) error {
			mu.Lock()
			defer mu.Unlock()

			name, _ := data["name"].(string)
			if name == "" {
				name, _ = data["full_name"].(string)
			}
			email, _ := data["email"].(string)
			if email == "" {
				email, _ = data["email_address"].(string)
			}
			role, _ := data["role"].(string)
			if role == "" {
				role, _ = data["system_role"].(string)
			}

			if strings.Contains(email, "invalid") {
				return fmt.Errorf("invalid email address: %s", email)
			}

			id := fmt.Sprintf("%d", nextID)
			nextID++
			store[id] = &CustomerRecord{
				ID:    id,
				Name:  name,
				Email: email,
				Role:  role,
			}
			return nil
		})

	panel.Resource(customerResource)
	_ = panel.Register(app)

	return app, store, &mu
}

func TestStreamingExportCSV(t *testing.T) {
	app, _, _ := setupImportExportApp()

	req := httptest.NewRequest("GET", "/admin/customers/export", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/csv") {
		t.Errorf("Expected text/csv Content-Type, got %s", contentType)
	}

	contentDisposition := rec.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "customers-export.csv") {
		t.Errorf("Expected customers-export.csv in Content-Disposition, got %s", contentDisposition)
	}

	body := rec.Body.String()
	// Should have table column labels as header
	if !strings.Contains(body, "Customer ID") || !strings.Contains(body, "Full Name") || !strings.Contains(body, "Email Address") {
		t.Errorf("Expected table header labels in CSV, got:\n%s", body)
	}

	// Should contain data rows
	if !strings.Contains(body, "Alice Cooper") || !strings.Contains(body, "bob@example.com") {
		t.Errorf("Expected records in CSV body, got:\n%s", body)
	}
}

func TestStreamingExportTSV(t *testing.T) {
	app, _, _ := setupImportExportApp()

	req := httptest.NewRequest("GET", "/admin/customers/export?format=tsv", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", rec.Code)
	}

	if !strings.Contains(rec.Header().Get("Content-Type"), "text/tab-separated-values") {
		t.Errorf("Expected TSV Content-Type, got %s", rec.Header().Get("Content-Type"))
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Customer ID\tFull Name\tEmail Address\tSystem Role") {
		t.Errorf("Expected tab-separated headers, got:\n%s", body)
	}
	if !strings.Contains(body, "Alice Cooper\talice@example.com") {
		t.Errorf("Expected tab-separated row values, got:\n%s", body)
	}
}

func TestStreamingExportJSON(t *testing.T) {
	app, _, _ := setupImportExportApp()

	req := httptest.NewRequest("GET", "/admin/customers/export?format=json", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", rec.Code)
	}

	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Errorf("Expected JSON Content-Type, got %s", rec.Header().Get("Content-Type"))
	}

	var records []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &records); err != nil {
		t.Fatalf("Failed to parse JSON export output: %v. Body was:\n%s", err, rec.Body.String())
	}

	if len(records) != 3 {
		t.Fatalf("Expected 3 exported records, got %d", len(records))
	}

	foundAlice := false
	for _, r := range records {
		if r["name"] == "Alice Cooper" {
			foundAlice = true
			if r["email"] != "alice@example.com" {
				t.Errorf("Expected alice@example.com, got %v", r["email"])
			}
		}
	}
	if !foundAlice {
		t.Errorf("Alice Cooper not found in exported JSON records")
	}
}

func TestExportFilteredIDsAndBulk(t *testing.T) {
	app, _, _ := setupImportExportApp()

	// 1. Export with query IDs: ?ids=1,3
	req := httptest.NewRequest("GET", "/admin/customers/export?ids=1,3", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Alice Cooper") || !strings.Contains(body, "Charlie Brown") {
		t.Errorf("Expected IDs 1 and 3 in CSV, got:\n%s", body)
	}
	if strings.Contains(body, "Bob Dylan") {
		t.Errorf("Expected Bob Dylan (ID 2) to be excluded, but found in:\n%s", body)
	}

	// 2. Export via bulk POST
	form := "bulk_action=export_bulk&selected_ids[]=2"
	postReq := httptest.NewRequest("POST", "/admin/customers/bulk", strings.NewReader(form))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postRec := httptest.NewRecorder()
	app.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for bulk export, got %d", postRec.Code)
	}

	bulkBody := postRec.Body.String()
	if !strings.Contains(bulkBody, "Bob Dylan") {
		t.Errorf("Expected Bob Dylan in bulk export, got:\n%s", bulkBody)
	}
	if strings.Contains(bulkBody, "Alice Cooper") {
		t.Errorf("Expected Alice Cooper to be excluded from selected bulk export")
	}
}

func TestInteractiveCSVImportWeb(t *testing.T) {
	app, store, mu := setupImportExportApp()

	// 1. Render import view
	viewReq := httptest.NewRequest("GET", "/admin/customers/import", nil)
	viewRec := httptest.NewRecorder()
	app.ServeHTTP(viewRec, viewReq)

	if viewRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for import view, got %d", viewRec.Code)
	}
	viewHTML := viewRec.Body.String()
	if !strings.Contains(viewHTML, "Import Customers") || !strings.Contains(viewHTML, "Download CSV Template") {
		t.Errorf("Expected Import view with template link, got:\n%s", viewHTML)
	}

	// 2. Submit CSV file upload with normalized column headers
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", "customers.csv")
	if err != nil {
		t.Fatalf("Failed to create multipart file part: %v", err)
	}

	csvData := `Full Name,Email Address,System Role
David Bowie,david@example.com,admin
Freddie Mercury,freddie@example.com,editor
`
	part.Write([]byte(csvData))
	writer.Close()

	uploadReq := httptest.NewRequest("POST", "/admin/customers/import", &buf)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRec := httptest.NewRecorder()
	app.ServeHTTP(uploadRec, uploadReq)

	if uploadRec.Code != http.StatusSeeOther {
		t.Fatalf("Expected 303 Redirect after import, got %d. Body: %s", uploadRec.Code, uploadRec.Body.String())
	}

	loc := uploadRec.Header().Get("Location")
	if !strings.Contains(loc, "imported=2") {
		t.Errorf("Expected redirect with imported=2, got %s", loc)
	}

	// Verify records were inserted in store
	mu.RLock()
	defer mu.RUnlock()
	foundDavid := false
	foundFreddie := false
	for _, c := range store {
		if c.Name == "David Bowie" && c.Email == "david@example.com" {
			foundDavid = true
		}
		if c.Name == "Freddie Mercury" && c.Email == "freddie@example.com" {
			foundFreddie = true
		}
	}
	if !foundDavid || !foundFreddie {
		t.Errorf("Imported records not found in store")
	}

	// Verify notification was sent
	unreadCount, _ := notifications.DefaultStore().CountUnread("all")
	if unreadCount == 0 {
		t.Errorf("Expected notification in store after CSV import")
	}
}

func TestCSVImportWithValidationErrors(t *testing.T) {
	app, _, _ := setupImportExportApp()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "customers.csv")
	csvData := `Name,Email,Role
Grace Hopper,grace@example.com,admin
Invalid User,invalid_email_no_at,editor
Ada Lovelace,ada@example.com,viewer
`
	part.Write([]byte(csvData))
	writer.Close()

	uploadReq := httptest.NewRequest("POST", "/admin/customers/import", &buf)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRec := httptest.NewRecorder()
	app.ServeHTTP(uploadRec, uploadReq)

	if uploadRec.Code != http.StatusSeeOther {
		t.Fatalf("Expected 303 Redirect, got %d", uploadRec.Code)
	}

	loc := uploadRec.Header().Get("Location")
	if !strings.Contains(loc, "imported=2") || !strings.Contains(loc, "failed=1") {
		t.Errorf("Expected imported=2 and failed=1 in redirect URL, got %s", loc)
	}
}

func TestRESTAPIExportAndImport(t *testing.T) {
	app, store, mu := setupImportExportApp()

	// 1. GET /admin/api/resources/customers/export?format=json
	exportReq := httptest.NewRequest("GET", "/admin/api/resources/customers/export?format=json", nil)
	exportRec := httptest.NewRecorder()
	app.ServeHTTP(exportRec, exportReq)

	if exportRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from REST API export, got %d", exportRec.Code)
	}

	var jsonRecords []map[string]any
	if err := json.Unmarshal(exportRec.Body.Bytes(), &jsonRecords); err != nil {
		t.Fatalf("Failed to parse REST API JSON export: %v", err)
	}
	if len(jsonRecords) < 3 {
		t.Fatalf("Expected at least 3 records in REST API export, got %d", len(jsonRecords))
	}

	// 2. POST /admin/api/resources/customers/import with JSON batch array
	batchPayload := map[string]any{
		"data": []map[string]any{
			{"name": "Mick Jagger", "email": "mick@example.com", "role": "admin"},
			{"name": "Keith Richards", "email": "keith@example.com", "role": "editor"},
		},
	}
	batchBytes, _ := json.Marshal(batchPayload)
	importReq := httptest.NewRequest("POST", "/admin/api/resources/customers/import", bytes.NewReader(batchBytes))
	importReq.Header.Set("Content-Type", "application/json")
	importRec := httptest.NewRecorder()
	app.ServeHTTP(importRec, importReq)

	if importRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from REST API batch import, got %d: %s", importRec.Code, importRec.Body.String())
	}

	var importResp filament.ImportResult
	if err := json.Unmarshal(importRec.Body.Bytes(), &importResp); err != nil {
		t.Fatalf("Failed to parse import response: %v", err)
	}
	if importResp.SuccessCount != 2 || importResp.FailedCount != 0 {
		t.Errorf("Expected 2 imported, 0 failed, got %+v", importResp)
	}

	// Verify records in store
	mu.RLock()
	foundMick := false
	for _, c := range store {
		if c.Name == "Mick Jagger" {
			foundMick = true
			break
		}
	}
	mu.RUnlock()

	if !foundMick {
		t.Errorf("Expected Mick Jagger to be present in store")
	}

	// 3. POST /admin/api/resources/customers/import with raw CSV string in JSON
	csvStringPayload := map[string]any{
		"csv": "Name,Email,Role\nBrian May,brian@example.com,guitarist\nRoger Taylor,roger@example.com,drummer\n",
	}
	csvBytes, _ := json.Marshal(csvStringPayload)
	csvReq := httptest.NewRequest("POST", "/admin/api/resources/customers/import", bytes.NewReader(csvBytes))
	csvReq.Header.Set("Content-Type", "application/json")
	csvRec := httptest.NewRecorder()
	app.ServeHTTP(csvRec, csvReq)

	if csvRec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK from CSV string import, got %d: %s", csvRec.Code, csvRec.Body.String())
	}

	var csvResp filament.ImportResult
	_ = json.Unmarshal(csvRec.Body.Bytes(), &csvResp)
	if csvResp.SuccessCount != 2 {
		t.Errorf("Expected 2 imported from CSV string, got %+v", csvResp)
	}
}

func TestTableActionHeaderRendering(t *testing.T) {
	tbl := tables.New[*CustomerRecord]().
		Columns(tables.TextColumn[*CustomerRecord]("name")).
		HeaderActions(
			tables.ImportAction[*CustomerRecord](),
			tables.ExportAction[*CustomerRecord](),
		)

	html := tbl.Render("/admin/customers")
	if !strings.Contains(html, `href="/admin/customers/import"`) {
		t.Errorf("Expected /admin/customers/import link in table header actions, got:\n%s", html)
	}
	if !strings.Contains(html, `href="/admin/customers/export"`) {
		t.Errorf("Expected /admin/customers/export link in table header actions, got:\n%s", html)
	}
}
