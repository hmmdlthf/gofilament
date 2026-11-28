package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/storage"
	gltest "github.com/golaravel/golaravel/pkg/testing"
)

func TestDemoUploads(t *testing.T) {
	if uploadsRoot == "" {
		if err := setupUploads(); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = os.RemoveAll(uploadsRoot); uploadsRoot = "" })
	h := testApp(t)
	tok := login(t, h, "admin@example.com")

	// seeded products have sample photos, linked from the list rows
	rec := call(t, h, "GET", "/admin/api/resources/products?per_page=50", tok, "")
	if !strings.Contains(rec.Body.String(), `"images":{"images":"/storage/demo/products/sample-`) {
		t.Fatalf("row image links missing: %s", rec.Body.String()[:300])
	}

	// upload an avatar: cropped square and shrunk to 256px when picked
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, _ := w.CreateFormFile("file", "me.png")
	_, _ = part.Write(gltest.FakeImage("me.png", 600, 300).Content)
	_ = w.Close()
	req := httptest.NewRequest("POST", "/admin/api/resources/customers/uploads?field=avatar", &body)
	req.RemoteAddr = "203.0.113.9:1234"
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	up := httptest.NewRecorder()
	h.ServeHTTP(up, req)
	if up.Code != http.StatusCreated {
		t.Fatalf("upload: %d %s", up.Code, up.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(up.Body.Bytes(), &out)

	if rec := call(t, h, "PUT", "/admin/api/resources/customers/1", tok, `{"avatar":"`+out["value"].(string)+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("save: %d %s", rec.Code, rec.Body.String())
	}
	mu.RLock()
	avatar := customers.rows["1"].Avatar
	mu.RUnlock()
	if !strings.HasPrefix(avatar, "avatars/") {
		t.Fatalf("avatar stored as %q", avatar)
	}
	if _, err := os.Stat(filepath.Join(uploadsRoot, "public", avatar)); err != nil {
		t.Fatalf("avatar file: %v", err)
	}

	// relation forms upload to their own endpoint
	var vid string
	mu.RLock()
	for id, v := range variants.rows {
		if v.ProductID == "1" {
			vid = id
			break
		}
	}
	mu.RUnlock()
	var sb bytes.Buffer
	sw := multipart.NewWriter(&sb)
	sp, _ := sw.CreateFormFile("file", "red.png")
	_, _ = sp.Write(gltest.FakeImage("red.png", 64, 64).Content)
	_ = sw.Close()
	sreq := httptest.NewRequest("POST", "/admin/api/resources/products/1/relations/variants/uploads?field=swatch", &sb)
	sreq.RemoteAddr = "203.0.113.9:1234"
	sreq.Header.Set("Content-Type", sw.FormDataContentType())
	sreq.Header.Set("Authorization", "Bearer "+tok)
	srec := httptest.NewRecorder()
	h.ServeHTTP(srec, sreq)
	if srec.Code != http.StatusCreated {
		t.Fatalf("swatch upload: %d %s", srec.Code, srec.Body.String())
	}
	var sout map[string]any
	_ = json.Unmarshal(srec.Body.Bytes(), &sout)
	if rec := call(t, h, "PUT", "/admin/api/resources/products/1/relations/variants/"+vid, tok, `{"swatch":"`+sout["value"].(string)+`"}`); rec.Code != http.StatusOK {
		t.Fatalf("save variant: %d %s", rec.Code, rec.Body.String())
	}
	mu.RLock()
	swatch := variants.rows[vid].Swatch
	mu.RUnlock()
	if !strings.HasPrefix(swatch, "swatches/") {
		t.Fatalf("swatch stored as %q", swatch)
	}

	// the stored files are served, and a reset wipes them
	app := golaravel.New()
	app.ServeStorage()
	get := httptest.NewRecorder()
	app.ServeHTTP(get, httptest.NewRequest("GET", storage.Disk(publicDisk).URL(avatar), nil))
	if get.Code != http.StatusOK || get.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("serving the avatar: %d %v", get.Code, get.Header())
	}
	resetUploads()
	if _, err := os.Stat(filepath.Join(uploadsRoot, "public", avatar)); !os.IsNotExist(err) {
		t.Fatal("reset must wipe uploads")
	}
}
