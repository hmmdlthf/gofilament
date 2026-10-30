package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golaravel/gofilament/pkg/forms"
)

func TestFormSchemaRendering(t *testing.T) {
	form := forms.New().
		Schema(
			forms.Section("Profile Details").
				Description("User credentials and display information").
				Columns(2).
				Schema(
					forms.TextInput("name").
						Required().
						Placeholder("John Doe"),

					forms.TextInput("email").
						Email().
						Required().
						Placeholder("john@example.com"),

					forms.Select("role").
						Options(map[string]string{
							"admin":  "Administrator",
							"editor": "Editor",
						}).
						Default("editor"),

					forms.Toggle("is_active").
						Label("Account Active"),

					forms.DatePicker("joined_at").
						Label("Join Date"),
				),
		).
		Fill(map[string]any{
			"name":      "Alice Smith",
			"email":     "alice@example.com",
			"role":      "admin",
			"is_active": true,
		})

	html := form.Render(context.Background())

	// Assert fields and filled state
	if !strings.Contains(html, "Profile Details") {
		t.Errorf("Expected section heading 'Profile Details'")
	}
	if !strings.Contains(html, `value="Alice Smith"`) {
		t.Errorf("Expected filled name 'Alice Smith', got:\n%s", html)
	}
	if !strings.Contains(html, `value="alice@example.com"`) {
		t.Errorf("Expected filled email 'alice@example.com'")
	}
	if !strings.Contains(html, `selected>Administrator</option>`) {
		t.Errorf("Expected selected option 'Administrator'")
	}
	if !strings.Contains(html, `checked`) {
		t.Errorf("Expected toggle to be checked")
	}
}

func TestFormValidationErrors(t *testing.T) {
	form := forms.New().
		Schema(
			forms.TextInput("email").Required(),
		).
		AddError("email", "The email field must be a valid email address.")

	html := form.Render(context.Background())

	if !strings.Contains(html, "The email field must be a valid email address.") {
		t.Errorf("Expected validation error message in rendered HTML, got:\n%s", html)
	}
}

func TestFormLiveReactivityAttributes(t *testing.T) {
	form := forms.New().
		Schema(
			forms.TextInput("title").
				Live(forms.Debounce(500 * time.Millisecond)),
			forms.Select("country").
				Options(map[string]string{"us": "United States"}).
				Live(),
		)

	html := form.Render(context.Background())

	if !strings.Contains(html, `oninput="Filament.liveUpdate('filament-form', 'title')"`) {
		t.Errorf("Expected live oninput attribute on title field, got:\n%s", html)
	}
	if !strings.Contains(html, `onchange="Filament.liveUpdate('filament-form', 'country')"`) {
		t.Errorf("Expected live onchange attribute on country field, got:\n%s", html)
	}
}
