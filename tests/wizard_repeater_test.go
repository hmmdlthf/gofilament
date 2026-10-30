package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/forms"
)

func TestWizardComponent_StepNavigationAndErrorAutoActivation(t *testing.T) {
	// 1. Basic 3-Step Wizard
	wizard := forms.Wizard(
		forms.WizardStep("Personal Information").
			Description("Enter your full name and email").
			Schema(
				forms.TextInput("name").Required(),
				forms.TextInput("email").Email().Required(),
			),
		forms.WizardStep("Shipping Details").
			Description("Delivery address and postal code").
			Schema(
				forms.TextInput("address").Required(),
				forms.TextInput("postal_code").Required(),
			),
		forms.WizardStep("Confirmation").
			Description("Review order details before checkout").
			Schema(
				forms.Toggle("terms_agreed").Label("I agree to terms"),
			),
	).SubmitAction("Place Order Now")

	form := forms.New().Schema(wizard)
	html := form.Render(context.Background())

	// Verifies Stepper Navigation & Alpine state
	if !strings.Contains(html, `x-data="{ currentStep: 1, totalSteps: 3 }"`) {
		t.Fatalf("expected Alpine currentStep: 1, totalSteps: 3 in wizard HTML, got: %s", html)
	}
	if !strings.Contains(html, "Personal Information") || !strings.Contains(html, "Shipping Details") || !strings.Contains(html, "Confirmation") {
		t.Fatalf("expected all 3 step titles in wizard HTML")
	}
	if !strings.Contains(html, "wizard-btn-prev") || !strings.Contains(html, "wizard-btn-next") || !strings.Contains(html, "Place Order Now") {
		t.Fatalf("expected next, previous, and custom submit buttons in wizard HTML")
	}

	// 2. Error Auto-Activation: If step 2 has an error, active step should be 2!
	formWithErrors := forms.New().
		Schema(wizard).
		AddError("postal_code", "Postal code is invalid")

	htmlWithErrors := formWithErrors.Render(context.Background())
	if !strings.Contains(htmlWithErrors, `x-data="{ currentStep: 2, totalSteps: 3 }"`) {
		t.Fatalf("expected wizard to auto-activate step 2 due to postal_code error, got: %s", htmlWithErrors)
	}
}

func TestRepeaterComponent_DynamicRowGenerationAndFieldScoping(t *testing.T) {
	repeater := forms.Repeater("line_items").
		Label("Order Line Items").
		ItemLabel("Item").
		MinItems(1).
		MaxItems(5).
		AddActionLabel("+ Add Line Item").
		Schema(
			forms.TextInput("sku").Required(),
			forms.TextInput("quantity").Numeric().Required(),
			forms.TextInput("price").Numeric().Required(),
		)

	// Pre-fill state with 2 items
	form := forms.New().
		Schema(repeater).
		Fill(map[string]any{
			"line_items": []map[string]any{
				{"sku": "LAPTOP-01", "quantity": 1, "price": 1200},
				{"sku": "MOUSE-02", "quantity": 2, "price": 50},
			},
		})

	html := form.Render(context.Background())

	// Verifies Alpine container with count, min, max
	if !strings.Contains(html, `x-data="{ count: 2, min: 1, max: 5 }"`) {
		t.Fatalf("expected count: 2, min: 1, max: 5 in repeater HTML, got: %s", html)
	}

	// Verifies scoped field names for row 0
	if !strings.Contains(html, `name="line_items[0][sku]"`) || !strings.Contains(html, `value="LAPTOP-01"`) {
		t.Fatalf("expected scoped row 0 field name and value in HTML")
	}
	if !strings.Contains(html, `name="line_items[0][quantity]"`) || !strings.Contains(html, `value="1"`) {
		t.Fatalf("expected scoped row 0 quantity in HTML")
	}

	// Verifies scoped field names for row 1
	if !strings.Contains(html, `name="line_items[1][sku]"`) || !strings.Contains(html, `value="MOUSE-02"`) {
		t.Fatalf("expected scoped row 1 field name and value in HTML")
	}

	// Verifies custom add button and template placeholder
	if !strings.Contains(html, "+ Add Line Item") {
		t.Fatalf("expected custom add button label")
	}
	if !strings.Contains(html, `<template id="tmpl_line_items">`) {
		t.Fatalf("expected row template in HTML")
	}
	if !strings.Contains(html, `name="line_items[__INDEX__][sku]"`) {
		t.Fatalf("expected __INDEX__ placeholder in template HTML")
	}
	if !strings.Contains(html, "repeater-btn-remove") {
		t.Fatalf("expected remove button for rows")
	}
}

func TestForm_RecursiveNestedFieldLookup(t *testing.T) {
	form := forms.New().Schema(
		forms.Wizard(
			forms.WizardStep("Account").Schema(
				forms.TextInput("username"),
			),
			forms.WizardStep("Orders").Schema(
				forms.Repeater("items").Schema(
					forms.TextInput("product_code"),
					forms.TextInput("qty"),
				),
			),
		),
	)

	// 1. Field inside Wizard Step
	userField := form.GetField("username")
	if userField == nil || userField.GetName() != "username" {
		t.Fatalf("expected to resolve 'username' nested inside wizard step")
	}

	// 2. Repeater itself
	itemsField := form.GetField("items")
	if itemsField == nil || itemsField.GetName() != "items" {
		t.Fatalf("expected to resolve repeater 'items'")
	}

	// 3. Field inside Repeater
	prodField := form.GetField("product_code")
	if prodField == nil || prodField.GetName() != "product_code" {
		t.Fatalf("expected to resolve 'product_code' nested inside repeater")
	}
}
