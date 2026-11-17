package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/notifications"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/storage"
)

type jobRow struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func newJobsApp(t *testing.T) (*golaravel.Engine, *[]*jobRow) {
	t.Helper()
	disk, err := storage.NewLocalDriver(t.TempDir(), "/files")
	if err != nil {
		t.Fatal(err)
	}
	storage.RegisterDisk("jobs-test", disk)
	rows := []*jobRow{}
	for i := 1; i <= 1200; i++ {
		rows = append(rows, &jobRow{ID: i, Name: fmt.Sprintf("Row %d", i)})
	}
	res := filament.NewResource[*jobRow]("rows").PluralModelName("Rows")
	res.Form(func(f *forms.Form) *forms.Form { return f.Schema(forms.TextInput("name").Required()) })
	res.Table(func(tb *tables.Table[*jobRow]) *tables.Table[*jobRow] {
		return tb.Columns(tables.TextColumn[*jobRow]("name").Label("Name"))
	})
	res.RecordID(func(x *jobRow) string { return fmt.Sprint(x.ID) })
	res.OnList(func(ctx *filament.Context, s, so, d string, page, perPage int, f map[string]string) ([]*jobRow, int, error) {
		start := (page - 1) * perPage
		if start >= len(rows) {
			return nil, len(rows), nil
		}
		return rows[start:min(start+perPage, len(rows))], len(rows), nil
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		rows = append(rows, &jobRow{ID: len(rows) + 1, Name: fmt.Sprint(data["name"])})
		return nil
	})
	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: token, Login: token}, nil
		}).
		BackgroundJobs(filament.JobsConfig{Disk: "jobs-test"}).
		Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, &rows
}

func jobReq(app http.Handler, method, path, user, ct string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "http://example.com"+path, bytes.NewReader(body))
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	req.Header.Set("Authorization", "Bearer "+user)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func waitJob(t *testing.T, app http.Handler, id, user string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		var j map[string]any
		_ = json.Unmarshal(jobReq(app, "GET", "/admin/api/jobs/"+id, user, "", nil).Body.Bytes(), &j)
		if j["status"] == "done" || j["status"] == "failed" {
			return j
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("job did not finish")
	return nil
}

func TestBackgroundExport(t *testing.T) {
	app, _ := newJobsApp(t)
	rec := jobReq(app, "GET", "/admin/api/resources/rows/export?format=csv&background=1", "ann", "", nil)
	var j map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &j)
	if rec.Code != http.StatusAccepted || j["id"] == nil {
		t.Fatalf("start: %d %s", rec.Code, rec.Body.String())
	}
	id := j["id"].(string)
	done := waitJob(t, app, id, "ann")
	if done["status"] != "done" || done["progress"] != float64(100) || done["total"] != float64(1200) || done["file_name"] != "rows-export.csv" {
		t.Fatalf("job: %v", done)
	}
	if rec := jobReq(app, "GET", "/admin/api/jobs/"+id, "bob", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("another user saw the job: %d", rec.Code)
	}
	if rec := jobReq(app, "GET", "/admin/api/jobs/"+id+"/download", "bob", "", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("another user downloaded: %d", rec.Code)
	}
	dl := jobReq(app, "GET", "/admin/api/jobs/"+id+"/download", "ann", "", nil)
	if dl.Code != 200 || strings.Count(dl.Body.String(), "Row ") != 1200 || !strings.Contains(dl.Header().Get("Content-Disposition"), "rows-export.csv") {
		t.Fatalf("download: %d %d rows", dl.Code, strings.Count(dl.Body.String(), "Row "))
	}
	var list map[string]any
	_ = json.Unmarshal(jobReq(app, "GET", "/admin/api/jobs", "ann", "", nil).Body.Bytes(), &list)
	if len(list["jobs"].([]any)) != 1 {
		t.Fatalf("jobs list: %v", list)
	}
	ns, _ := notifications.DefaultStore().List("ann", 10)
	found := false
	for _, n := range ns {
		found = found || strings.Contains(n.Title, "Export of Rows")
	}
	if !found {
		t.Fatal("no notification")
	}
}

func TestWebExportRunsAsJob(t *testing.T) {
	app, _ := newJobsApp(t)
	rec := jobReq(app, "GET", "/admin/rows/export?format=xlsx", "ann", "", nil)
	loc := rec.Header().Get("Location")
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(loc, "/admin/jobs/job_") {
		t.Fatalf("web export: %d %s", rec.Code, loc)
	}
	id := strings.TrimPrefix(loc, "/admin/jobs/")
	waitJob(t, app, id, "ann")
	page := jobReq(app, "GET", loc, "ann", "", nil).Body.String()
	if !strings.Contains(page, `data-job-status="done"`) || !strings.Contains(page, "rows-export.xlsx") {
		t.Fatalf("job page:\n%s", page)
	}
	if rec := jobReq(app, "GET", "/admin/rows/export?sync=1", "ann", "", nil); rec.Code != 200 {
		t.Fatalf("sync export: %d", rec.Code)
	}
}

func TestBackgroundImport(t *testing.T) {
	app, rows := newJobsApp(t)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "r.csv")
	_, _ = part.Write([]byte("Name\nNew one\n\nNew two\n"))
	_ = w.WriteField("background", "1")
	_ = w.Close()
	rec := jobReq(app, "POST", "/admin/api/resources/rows/import", "ann", w.FormDataContentType(), buf.Bytes())
	var j map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &j)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("start: %d %s", rec.Code, rec.Body.String())
	}
	done := waitJob(t, app, j["id"].(string), "ann")
	if done["status"] != "done" || done["result"].(map[string]any)["imported"] != float64(2) || len(*rows) != 1202 {
		t.Fatalf("import job: %v", done)
	}
}
