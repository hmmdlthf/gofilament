package tests

import (
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/golaravel/golaravel/pkg/golaravel"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/flutterclient"
)

func get(t *testing.T, app *golaravel.Engine, path string) (int, string, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	b, _ := io.ReadAll(rec.Body)
	return rec.Code, string(b), rec.Header().Get("Cache-Control")
}

func TestFlutterFSServesAnEmbeddedBuild(t *testing.T) {
	build := fstest.MapFS{
		"index.html":           {Data: []byte(`<html><head><base href="$FLUTTER_BASE_HREF"></head><body><script src="flutter_bootstrap.js"></script></body></html>`)},
		"flutter_bootstrap.js": {Data: []byte(`_flutter.loader.load({serviceWorkerSettings: {}, "mainJsPath":"main.dart.js"});`)},
		"main.dart.js":         {Data: []byte(`console.log("app")`)},
		"assets/logo.png":      {Data: pngBytes},
	}
	p := filament.NewPanel("emb").Path("/admin").FlutterFS(build).BuildID("b42")
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}

	code, body, cache := get(t, app, "/admin/")
	if code != 200 || !strings.Contains(body, `<base href="/admin/">`) || !strings.Contains(body, `flutter_bootstrap.js?v=b42`) || !strings.Contains(cache, "no-cache") {
		t.Fatalf("index: %d %q %q", code, body, cache)
	}
	if code, body, _ := get(t, app, "/admin/resources/items/3"); code != 200 || !strings.Contains(body, `<base href="/admin/">`) {
		t.Fatalf("client routes get index.html: %d", code)
	}
	if code, body, _ := get(t, app, "/admin/flutter_bootstrap.js"); code != 200 || !strings.Contains(body, `main.dart.js?v=b42`) || strings.Contains(body, "serviceWorkerSettings:") {
		t.Fatalf("bootstrap: %d %q", code, body)
	}
	if code, body, _ := get(t, app, "/admin/main.dart.js"); code != 200 || body != `console.log("app")` {
		t.Fatalf("assets: %d %q", code, body)
	}
	if code, _, _ := get(t, app, "/admin/assets/logo.png"); code != 200 {
		t.Fatalf("nested assets: %d", code)
	}
	if code, body, _ := get(t, app, "/admin/../../etc/passwd"); code != 200 || !strings.Contains(body, "<base") {
		t.Fatalf("paths are confined to the build: %d", code)
	}
	if code, _, _ := get(t, app, "/admin/api/boot"); code != 200 {
		t.Fatalf("the API still answers next to the app: %d", code)
	}
}

func TestBuildWebEmbedsTheBuild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a shell script as flutter")
	}
	app := t.TempDir()
	fake := filepath.Join(t.TempDir(), "flutter")
	script := "#!/bin/sh\necho \"flutter $@\"\nmkdir -p build/web/assets\necho '<html>app</html>' > build/web/index.html\necho 'js' > build/web/main.dart.js\necho 'x' > build/web/assets/a.txt\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	embed := filepath.Join(t.TempDir(), "admin-web")
	var out strings.Builder
	err := flutterclient.BuildWeb(app, flutterclient.BuildOptions{Flutter: fake, Args: []string{"--wasm"}, EmbedDir: embed, Output: &out})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "flutter build web --release --wasm") {
		t.Fatalf("flutter arguments: %q", out.String())
	}
	for _, f := range []string{"dist/index.html", "dist/main.dart.js", "dist/assets/a.txt", "dist/.gofilament-dist"} {
		if _, err := os.Stat(filepath.Join(embed, f)); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
	}
	src, _ := os.ReadFile(filepath.Join(embed, "embed.go"))
	if !strings.Contains(string(src), "package admin_web") || !strings.Contains(string(src), "//go:embed all:dist") {
		t.Fatalf("embed.go:\n%s", src)
	}
	// rebuilding replaces dist; stale files go away
	_ = os.WriteFile(filepath.Join(embed, "dist", "stale.js"), []byte("old"), 0o644)
	if err := flutterclient.BuildWeb(app, flutterclient.BuildOptions{Flutter: fake, EmbedDir: embed, Output: io.Discard}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(embed, "dist", "stale.js")); !os.IsNotExist(err) {
		t.Fatal("a rebuild must replace dist")
	}
	// a dist directory it did not write is never removed
	other := t.TempDir()
	_ = os.MkdirAll(filepath.Join(other, "dist"), 0o755)
	_ = os.WriteFile(filepath.Join(other, "dist", "precious.txt"), []byte("keep"), 0o644)
	if err := flutterclient.BuildWeb(app, flutterclient.BuildOptions{Flutter: fake, EmbedDir: other, Output: io.Discard}); err == nil {
		t.Fatal("expected a refusal to replace a foreign dist directory")
	}
	if _, err := os.Stat(filepath.Join(other, "dist", "precious.txt")); err != nil {
		t.Fatal("a foreign dist directory was touched")
	}
}
