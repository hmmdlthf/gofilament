package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type member struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Joined string `json:"joined"`
}

func newMemberApp(t *testing.T) (*golaravel.Engine, *[]*member) {
	t.Helper()
	members := []*member{{ID: 1, Name: "Ann", Email: "ann@example.com", Joined: "2026-01-02"}}
	res := filament.NewResource[*member]("members").PluralModelName("Members")
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.TextInput("name").Label("Full name").Required(),
			forms.TextInput("email").Label("Email").Email().Required(),
			forms.DatePicker("joined").Label("Joined"),
		)
	})
	res.Table(func(tb *tables.Table[*member]) *tables.Table[*member] {
		return tb.Columns(
			tables.TextColumn[*member]("name").Label("Full name"),
			tables.TextColumn[*member]("email").Label("Email"),
			tables.TextColumn[*member]("joined").Label("Joined"),
		).HeaderActions(tables.ImportAction[*member](), tables.ExportAction[*member]())
	})
	res.RecordID(func(m *member) string { return fmt.Sprint(m.ID) })
	res.OnList(func(ctx *filament.Context, search, sort, dir string, page, perPage int, filters map[string]string) ([]*member, int, error) {
		var out []*member
		for _, m := range members {
			if search == "" || strings.Contains(m.Name, search) {
				out = append(out, m)
			}
		}
		return out, len(out), nil
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		members = append(members, &member{ID: len(members) + 1, Name: fmt.Sprint(data["name"]), Email: fmt.Sprint(data["email"]), Joined: fmt.Sprint(data["joined"])})
		return nil
	})
	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: "1", Login: "admin"}, nil
		}).Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, &members
}

func send(app http.Handler, method, path, contentType string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://example.com"+path, bytes.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func multipartFile(name string, content []byte, fields map[string]string) ([]byte, string) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", name)
	_, _ = part.Write(content)
	for k, v := range fields {
		_ = w.WriteField(k, v)
	}
	_ = w.Close()
	return buf.Bytes(), w.FormDataContentType()
}

const memberCSV = "Full name,E-mail address,Notes\nBea,bea@example.com,x\nCal,not-an-email,y\n,dan@example.com,z\n"

func TestImportValidatesRowsWithTheForm(t *testing.T) {
	app, members := newMemberApp(t)
	body, ct := multipartFile("m.csv", []byte(memberCSV), map[string]string{"mapping[E-mail address]": "email"})
	rec := send(app, "POST", "/admin/api/resources/members/import", ct, body)
	var res filament.ImportResult
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if rec.Code != 200 || res.SuccessCount != 1 || res.FailedCount != 2 || len(*members) != 2 {
		t.Fatalf("import: %d %+v (members %d)", rec.Code, res, len(*members))
	}
	if res.Rows[0].Row != 3 || res.Rows[0].Errors["email"] == "" || res.Rows[1].Errors["name"] == "" {
		t.Fatalf("row errors: %+v", res.Rows)
	}
}

func TestAPIImportDryRun(t *testing.T) {
	app, members := newMemberApp(t)
	body, ct := multipartFile("m.csv", []byte(memberCSV), map[string]string{"dry_run": "1"})
	rec := send(app, "POST", "/admin/api/resources/members/import", ct, body)
	var pv filament.ImportPreview
	_ = json.Unmarshal(rec.Body.Bytes(), &pv)
	if rec.Code != 200 || len(*members) != 1 {
		t.Fatalf("dry run saved or failed: %d members=%d", rec.Code, len(*members))
	}
	// "E-mail address" is not a field name or label, so it is not mapped.
	if pv.Mapping["Full name"] != "name" || pv.Mapping["E-mail address"] != "" || pv.Mapping["Notes"] != "" {
		t.Fatalf("mapping: %v", pv.Mapping)
	}
	if pv.Total != 3 || pv.Valid != 0 || len(pv.Fields) != 3 || !pv.Fields[1].Required {
		t.Fatalf("preview: %+v", pv)
	}
	// JSON with an explicit mapping.
	raw, _ := json.Marshal(map[string]any{"csv": memberCSV, "dry_run": true, "mapping": map[string]string{"E-mail address": "email"}})
	rec = send(app, "POST", "/admin/api/resources/members/import", "application/json", raw)
	pv = filament.ImportPreview{}
	_ = json.Unmarshal(rec.Body.Bytes(), &pv)
	if pv.Valid != 1 || pv.Invalid != 2 || len(pv.Problems) != 2 || pv.Rows[1].Errors["email"] == "" || len(*members) != 1 {
		t.Fatalf("mapped preview: %+v", pv)
	}
}

var tokenRe = regexp.MustCompile(`name="token" value="([0-9a-f]+)"`)

func TestWebImportPreviewFlow(t *testing.T) {
	app, members := newMemberApp(t)
	body, ct := multipartFile("m.csv", []byte(memberCSV), nil)
	rec := send(app, "POST", "/admin/members/import/preview", ct, body)
	page := rec.Body.String()
	m := tokenRe.FindStringSubmatch(page)
	if rec.Code != 200 || m == nil || !strings.Contains(page, `name="mapping[E-mail address]"`) || !strings.Contains(page, "Import 0 rows") {
		t.Fatalf("preview page: %d\n%s", rec.Code, page)
	}
	token := m[1]
	// Map the email column and check again: one row is ready.
	form := url.Values{"token": {token}, "mapping[Full name]": {"name"}, "mapping[E-mail address]": {"email"}, "mapping[Notes]": {""}}
	rec = send(app, "POST", "/admin/members/import/preview", "application/x-www-form-urlencoded", []byte(form.Encode()))
	page = rec.Body.String()
	if !strings.Contains(page, "Import 1 rows") || !strings.Contains(page, "data-import-invalid") || len(*members) != 1 {
		t.Fatalf("re-check:\n%s", page)
	}
	rec = send(app, "POST", "/admin/members/import", "application/x-www-form-urlencoded", []byte(form.Encode()))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "1 rows imported.") || !strings.Contains(rec.Body.String(), "2 rows were skipped.") || len(*members) != 2 {
		t.Fatalf("import: %d members=%d\n%s", rec.Code, len(*members), rec.Body.String())
	}
	// The stashed file is gone.
	rec = send(app, "POST", "/admin/members/import", "application/x-www-form-urlencoded", []byte(form.Encode()))
	if rec.Code != http.StatusGone {
		t.Fatalf("reused token: %d", rec.Code)
	}
}

func TestExcelExportImportRoundTrip(t *testing.T) {
	app, members := newMemberApp(t)
	rec := send(app, "GET", "/admin/members/export?format=xlsx", "", nil)
	if rec.Code != 200 || !strings.HasPrefix(rec.Body.String(), "PK") || !strings.Contains(rec.Header().Get("Content-Type"), "spreadsheetml") {
		t.Fatalf("xlsx export: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	body, ct := multipartFile("m.xlsx", rec.Body.Bytes(), map[string]string{"dry_run": "1"})
	rec = send(app, "POST", "/admin/api/resources/members/import", ct, body)
	var pv filament.ImportPreview
	_ = json.Unmarshal(rec.Body.Bytes(), &pv)
	if rec.Code != 200 || pv.Valid != 1 || pv.Rows[0].Values["email"] != "ann@example.com" || pv.Rows[0].Values["joined"] != "2026-01-02" {
		t.Fatalf("xlsx import preview: %d %+v", rec.Code, pv)
	}
	_ = members
}

func TestExportLinksKeepTheListState(t *testing.T) {
	app, _ := newMemberApp(t)
	page := send(app, "GET", "/admin/members?search=Ann", "", nil).Body.String()
	if !strings.Contains(page, `href="/admin/members/export?search=Ann"`) || !strings.Contains(page, `href="/admin/members/export?format=xlsx&amp;search=Ann"`) {
		t.Fatalf("export links:\n%s", page)
	}
	csv := send(app, "GET", "/admin/members/export?columns=email", "", nil).Body.String()
	if strings.Contains(csv, "Full name") || !strings.Contains(csv, "Email") {
		t.Fatalf("columns=email export:\n%s", csv)
	}
}
