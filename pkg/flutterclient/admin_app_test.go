package flutterclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestGenerateAdminAppBaseHrefAndGitignore(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateAdminApp(dir, AdminAppOptions{Name: "staff_app"}); err != nil {
		t.Fatal(err)
	}
	if idx := readFile(t, filepath.Join(dir, "web/index.html")); !strings.Contains(idx, `<base href="$FLUTTER_BASE_HREF">`) {
		t.Fatalf("index.html must use the flutter base-href placeholder:\n%s", idx)
	}
	if gi := readFile(t, filepath.Join(dir, ".gitignore")); !strings.Contains(gi, ".flutter-plugins-dependencies") {
		t.Fatalf(".gitignore missing .flutter-plugins-dependencies:\n%s", gi)
	}
}

func TestGenerateAdminAppKeepsExistingFiles(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateAdminApp(dir, AdminAppOptions{}); err != nil {
		t.Fatal(err)
	}
	mainPath := filepath.Join(dir, "lib/main.dart")
	giPath := filepath.Join(dir, ".gitignore")
	_ = os.WriteFile(mainPath, []byte("// custom pages\n"), 0o644)
	_ = os.WriteFile(giPath, []byte("custom\n"), 0o644)
	_ = os.Remove(filepath.Join(dir, "analysis_options.yaml"))

	if err := GenerateAdminApp(dir, AdminAppOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, mainPath); got != "// custom pages\n" {
		t.Fatalf("lib/main.dart overwritten: %q", got)
	}
	if got := readFile(t, giPath); got != "custom\n" {
		t.Fatalf(".gitignore overwritten: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "analysis_options.yaml")); err != nil {
		t.Fatalf("missing file not recreated: %v", err)
	}

	if err := GenerateAdminApp(dir, AdminAppOptions{Overwrite: true}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, mainPath); !strings.Contains(got, "GoFilamentAdmin") {
		t.Fatalf("Overwrite did not regenerate lib/main.dart: %q", got)
	}
}
