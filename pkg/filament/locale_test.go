package filament

import (
	"strings"
	"testing"
)

func TestRequiredKeysNotEmpty(t *testing.T) {
	if len(RequiredTranslationKeys) == 0 {
		t.Fatal("RequiredTranslationKeys must not be empty")
	}

	essential := []string{
		"dashboard", "sign_in_title", "sign_in_subtitle", "remember_me",
		"search_records", "bulk_actions", "bulk_delete", "relations",
		"show_relations", "hide_relations", "toggle_relations", "direction",
	}

	keySet := make(map[string]bool)
	for _, k := range RequiredTranslationKeys {
		keySet[k] = true
	}

	for _, k := range essential {
		if !keySet[k] {
			t.Errorf("Essential key %q is missing from RequiredTranslationKeys", k)
		}
	}
}

func TestAllDefaultLocalesHaveRequiredKeys(t *testing.T) {
	locales := []string{"en", "ar", "es", "de", "fr"}
	for _, loc := range locales {
		dict, ok := defaultAdminTranslations[loc]
		if !ok {
			t.Fatalf("Default translations missing for locale %q", loc)
		}

		missing := ValidateLocaleTranslations(loc, dict)
		if len(missing) > 0 {
			t.Errorf("Locale %q is missing %d required translation key(s):\n  %s",
				loc, len(missing), strings.Join(missing, ", "))
		}
	}
}

func TestPanelValidateTranslationsPasses(t *testing.T) {
	p := NewPanel("admin").Locales("en", "ar", "fr", "es", "de")
	if err := p.ValidateTranslations(); err != nil {
		t.Fatalf("ValidateTranslations() unexpectedly failed for supported default locales: %v", err)
	}
}

func TestPanelValidateTranslationsFailsOnMissingKey(t *testing.T) {
	// Custom language missing required keys
	p := NewPanel("test_panel").Locales("it").Translations("it", map[string]string{
		"dashboard": "Pannello di controllo",
	})

	err := p.ValidateTranslations()
	if err == nil {
		t.Fatal("Expected ValidateTranslations() to fail for incomplete 'it' locale, but got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "sign_in_title") {
		t.Errorf("Expected error to mention missing key 'sign_in_title', got:\n%s", errMsg)
	}
	if !strings.Contains(errMsg, "bulk_delete") {
		t.Errorf("Expected error to mention missing key 'bulk_delete', got:\n%s", errMsg)
	}
}
