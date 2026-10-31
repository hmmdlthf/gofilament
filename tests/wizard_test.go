package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/forms"
)

func TestWizard_FullFeaturesAndCustomizations(t *testing.T) {
	wizard := forms.Wizard(
		forms.WizardStep("Account Setup").
			Description("Setup your credentials and profile").
			Icon("heroicon-o-user").
			Badge("Step 1").
			BadgeColor("primary").
			Schema(
				forms.TextInput("username").Required(),
				forms.TextInput("email").Email().Required(),
			),
		forms.WizardStep("Company Details").
			Description("Tell us about your organization").
			Icon(`<svg viewBox="0 0 20 20"><path d="M10 10"/></svg>`).
			Badge("Optional").
			BadgeColor("warning").
			Schema(
				forms.TextInput("company_name"),
				forms.TextInput("vat_number"),
			),
		forms.WizardStep("Review & Complete").
			Description("Verify your configuration").
			Badge("Final").
			BadgeColor("success").
			Schema(
				forms.Toggle("agreed").Label("Confirm and agree to terms"),
			),
	).
		StartStep(2).
		Skippable(true).
		ProgressBar(true).
		NextActionLabel("Continue →").
		PreviousActionLabel("← Go Back").
		SubmitAction("Create Organization").
		CancelAction("/admin/users", "Discard Changes")

	// Verify getters
	if len(wizard.GetSteps()) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(wizard.GetSteps()))
	}
	if wizard.GetStartStep() != 2 {
		t.Fatalf("expected start step 2, got %d", wizard.GetStartStep())
	}
	if !wizard.IsSkippable() {
		t.Fatalf("expected skippable to be true")
	}
	if !wizard.HasProgressBar() {
		t.Fatalf("expected progress bar to be true")
	}
	if wizard.GetNextActionLabel() != "Continue →" {
		t.Fatalf("expected Continue →, got %s", wizard.GetNextActionLabel())
	}
	if wizard.GetPreviousActionLabel() != "← Go Back" {
		t.Fatalf("expected ← Go Back, got %s", wizard.GetPreviousActionLabel())
	}
	if wizard.GetSubmitAction() != "Create Organization" {
		t.Fatalf("expected Create Organization, got %s", wizard.GetSubmitAction())
	}
	if wizard.GetCancelUrl() != "/admin/users" || wizard.GetCancelLabel() != "Discard Changes" {
		t.Fatalf("expected CancelAction /admin/users / Discard Changes")
	}

	step1 := wizard.GetSteps()[0]
	if step1.GetTitle() != "Account Setup" || step1.GetDescription() != "Setup your credentials and profile" {
		t.Fatalf("expected step 1 title and description")
	}
	if step1.GetIcon() != "heroicon-o-user" || step1.GetBadge() != "Step 1" || step1.GetBadgeColor() != "primary" {
		t.Fatalf("expected step 1 icon, badge, badgeColor")
	}

	// Render HTML
	form := forms.New().Schema(wizard)
	html := form.Render(context.Background())

	// Verifies Alpine.js controller state
	if !strings.Contains(html, "currentStep: 2") {
		t.Fatalf("expected Alpine currentStep: 2 due to StartStep(2), got: %s", html)
	}
	if !strings.Contains(html, "skippable: true") {
		t.Fatalf("expected skippable: true in Alpine x-data")
	}
	if !strings.Contains(html, "stepTitles: [\"Account Setup\", \"Company Details\", \"Review & Complete\"]") {
		t.Fatalf("expected stepTitles array in Alpine x-data")
	}

	// Verifies Validation & Navigation logic
	if !strings.Contains(html, "validateCurrentStep()") || !strings.Contains(html, "goToStep(") {
		t.Fatalf("expected validateCurrentStep and goToStep methods in Alpine controller")
	}
	if !strings.Contains(html, `id="wizard_step_1"`) || !strings.Contains(html, `id="wizard_step_2"`) || !strings.Contains(html, `id="wizard_step_3"`) {
		t.Fatalf("expected step pane IDs wizard_step_1, wizard_step_2, wizard_step_3")
	}

	// Verifies Progress Bar
	if !strings.Contains(html, "wizard-progress-bar-container") || !strings.Contains(html, "wizard-progress-bar") {
		t.Fatalf("expected progress bar container and element in HTML")
	}

	// Verifies Mobile Step Header
	if !strings.Contains(html, "wizard-mobile-header") || !strings.Contains(html, "Step <span x-text=\"currentStep\"></span> of 3") {
		t.Fatalf("expected mobile header counter in HTML")
	}

	// Verifies Badges
	if !strings.Contains(html, "wizard-step-badge") || !strings.Contains(html, "Step 1") || !strings.Contains(html, "Optional") || !strings.Contains(html, "Final") {
		t.Fatalf("expected step badges in HTML")
	}

	// Verifies SVG and class icons
	if !strings.Contains(html, "heroicon-o-user") || !strings.Contains(html, `<svg viewBox="0 0 20 20">`) {
		t.Fatalf("expected heroicon class and raw SVG in step indicators")
	}

	// Verifies Custom Actions & Labels
	if !strings.Contains(html, "Continue →") || !strings.Contains(html, "← Go Back") || !strings.Contains(html, "Create Organization") {
		t.Fatalf("expected custom action labels in footer buttons")
	}
	if !strings.Contains(html, "wizard-btn-cancel") || !strings.Contains(html, "Discard Changes") || !strings.Contains(html, `href="/admin/users"`) {
		t.Fatalf("expected cancel button with correct link and label")
	}
}

func TestWizard_ProgressBarToggle(t *testing.T) {
	wizard := forms.Wizard(
		forms.WizardStep("Step A").Schema(forms.TextInput("a")),
		forms.WizardStep("Step B").Schema(forms.TextInput("b")),
	).ProgressBar(false)

	if wizard.HasProgressBar() {
		t.Fatalf("expected HasProgressBar to be false")
	}

	html := forms.New().Schema(wizard).Render(context.Background())
	if strings.Contains(html, "wizard-progress-bar-container") {
		t.Fatalf("did not expect progress bar container when disabled")
	}
}
