package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/storage"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
)

var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func newUploadApp(t *testing.T, configure func(*filament.Panel)) *golaravel.Engine {
	t.Helper()
	res := filament.NewResource[*guardItem]("items")
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.TextInput("name"),
			forms.ImageUpload("photo").Disk("uploads-test").Directory("photos").MaxSize(1),
			forms.FileUpload("contract").Disk("").AcceptedFileTypes("application/pdf", ".pdf"),
		)
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error { return nil })
	p := filament.NewPanel("uploads").Path("/admin").Engine(filament.EngineFlutter).Flutter(t.TempDir()).
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			if token == "good" {
				return &filament.PanelUser{ID: "u1"}, nil
			}
			return nil, fmt.Errorf("bad token")
		}).Resource(res)
	if configure != nil {
		configure(p)
	}
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app
}

func upload(t *testing.T, app http.Handler, field, token, name string, content []byte) (int, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	fw, _ := w.CreateFormFile("file", name)
	_, _ = fw.Write(content)
	_ = w.Close()
	req := httptest.NewRequest("POST", "/admin/api/resources/items/uploads?field="+field, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestUploadsToTheFieldsDisk(t *testing.T) {
	dir := t.TempDir()
	drv, err := storage.NewLocalDriver(dir, "/files")
	if err != nil {
		t.Fatal(err)
	}
	storage.RegisterDisk("uploads-test", drv)
	app := newUploadApp(t, nil)

	if code, _ := upload(t, app, "photo", "", "a.png", pngBytes); code != 401 {
		t.Fatalf("uploads need auth: %d", code)
	}
	code, out := upload(t, app, "photo", "good", "../../etc/a.png", pngBytes)
	if code != 201 {
		t.Fatalf("upload: %d %v", code, out)
	}
	value := fmt.Sprint(out["value"])
	if filepath.Dir(value) != "photos" || filepath.Ext(value) != ".png" {
		t.Fatalf("stored under the field's directory with a random name: %q", value)
	}
	if _, err := os.Stat(filepath.Join(dir, value)); err != nil {
		t.Fatalf("file not stored: %v", err)
	}
	if code, out := upload(t, app, "photo", "good", "a.png", []byte("just text")); code != 422 {
		t.Fatalf("an image field refuses non-images (by content): %d %v", code, out)
	}
	if code, _ := upload(t, app, "photo", "good", "big.png", append(pngBytes, make([]byte, 2048)...)); code != 422 {
		t.Fatalf("MaxSize: %d", code)
	}
	if code, _ := upload(t, app, "name", "good", "a.png", pngBytes); code != 404 {
		t.Fatalf("only file fields take uploads: %d", code)
	}
	if code, out := upload(t, app, "contract", "good", "c.pdf", []byte("%PDF-1.4 test")); code != 501 {
		t.Fatalf("a field with an empty Disk and no UploadsUsing refuses: %d %v", code, out)
	}
}

func TestUploadsUsing(t *testing.T) {
	var got filament.UploadedFile
	app := newUploadApp(t, func(p *filament.Panel) {
		p.UploadsUsing(func(ctx *filament.Context, f filament.UploadedFile) (string, string, error) {
			got = f
			return "key-123", "https://cdn.test/key-123", nil
		})
	})
	if code, _ := upload(t, app, "contract", "good", "c.exe", []byte("MZ\x90\x00")); code != 422 {
		t.Fatalf("accepted types: %d", code)
	}
	code, out := upload(t, app, "contract", "good", "c.pdf", []byte("%PDF-1.4 test"))
	if code != 201 || out["value"] != "key-123" || out["url"] != "https://cdn.test/key-123" || out["name"] != "c.pdf" {
		t.Fatalf("custom store: %d %v", code, out)
	}
	if got.Resource != "items" || got.ContentType != "application/pdf" || got.Field.GetName() != "contract" {
		t.Fatalf("handler input: %+v", got)
	}

	// the saved value is checked too: one file for a single-file field
	if code, _ := relDo(t, app, "POST", "/admin/api/resources/items", "good", map[string]any{"contract": []any{"a", "b"}}); code != 422 {
		t.Fatalf("two files for a single file field: %d", code)
	}
}
