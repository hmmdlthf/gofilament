package filament

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var placeholder = regexp.MustCompile(`\{[a-z]+\}`)

func placeholders(s string) string {
	p := placeholder.FindAllString(s, -1)
	sort.Strings(p)
	return strings.Join(p, ",")
}

// Every built-in translation keeps the English text's placeholders.
func TestUITranslationsKeepPlaceholders(t *testing.T) {
	for loc, table := range defaultUITranslations {
		for en, tr := range table {
			if placeholders(en) != placeholders(tr) {
				t.Errorf("%s: %q -> %q changes the placeholders", loc, en, tr)
			}
			if strings.TrimSpace(tr) == "" {
				t.Errorf("%s: %q is empty", loc, en)
			}
		}
	}
}

func TestUITranslationsMerge(t *testing.T) {
	p := NewPanel("x").UITranslations("de", map[string]string{"Sign in": "Einloggen"}).UITranslations("pt", map[string]string{"Sign in": "Entrar"})
	de := p.GetUITranslations("de-AT")
	if de["Sign in"] != "Einloggen" || de["Sign out"] != "Abmelden" {
		t.Fatalf("de: %q %q", de["Sign in"], de["Sign out"])
	}
	if p.GetUITranslations("pt")["Sign in"] != "Entrar" || len(p.GetUITranslations("en")) != 0 {
		t.Fatal("pt / en")
	}
}

// Every tr('...') string of the Flutter app has a built-in translation, so
// new words cannot ship English-only by accident.
func TestUITranslationsCoverTheFlutterApp(t *testing.T) {
	files, _ := filepath.Glob("../../flutter/gofilament_admin/lib/src/*.dart")
	if len(files) == 0 {
		t.Skip("Flutter sources not found")
	}
	call := regexp.MustCompile(`tr\(\s*'((?:[^'\\]|\\.)*)'`)
	for _, f := range files {
		src, _ := os.ReadFile(f)
		for _, m := range call.FindAllStringSubmatch(string(src), -1) {
			key := strings.ReplaceAll(m[1], `\'`, "'")
			for loc, table := range defaultUITranslations {
				if _, ok := table[key]; !ok {
					t.Errorf("%s: no %s translation for %q", filepath.Base(f), loc, key)
				}
			}
		}
	}
}
