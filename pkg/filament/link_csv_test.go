package filament

import "testing"

func TestSafeLinkURL(t *testing.T) {
	for raw, want := range map[string]bool{
		"/admin/orders/1":         true,
		"https://example.com/x":   true,
		"http://example.com":      true,
		"javascript:alert(1)":     false,
		" JavaScript:alert(1)":    false,
		"data:text/html,<b>x</b>": false,
		"vbscript:msgbox(1)":      false,
	} {
		if got := safeLinkURL(raw); got != want {
			t.Errorf("safeLinkURL(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestCSVSafeCell(t *testing.T) {
	for in, want := range map[string]string{
		"=HYPERLINK(\"http://evil\")": "'=HYPERLINK(\"http://evil\")",
		"+1+1":                        "'+1+1",
		"-2":                          "-2", "-2+3": "'-2+3",
		"@SUM(A1)": "'@SUM(A1)",
		"plain":    "plain",
		"":         "",
	} {
		if got := csvSafeCell(in); got != want {
			t.Errorf("csvSafeCell(%q) = %q, want %q", in, got, want)
		}
	}
}
