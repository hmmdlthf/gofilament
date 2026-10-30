package tests

import (
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/actions"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
)

func TestActionConfirmationButton(t *testing.T) {
	action := actions.New("delete").
		Label("Delete Account").
		Color("danger").
		RequiresConfirmation().
		ModalDescription("Are you sure you want to delete your account?")

	btnHTML := action.RenderButton("/admin/account/delete")

	if !strings.Contains(btnHTML, "btn-danger") {
		t.Errorf("Expected button to have danger class, got: %s", btnHTML)
	}
	if !strings.Contains(btnHTML, "Are you sure you want to delete your account?") {
		t.Errorf("Expected confirmation popup text in button onclick, got: %s", btnHTML)
	}
}

func TestActionModalWithForm(t *testing.T) {
	action := actions.New("refund").
		Label("Issue Refund").
		Form(
			forms.TextInput("amount").Required().Numeric(),
			forms.Textarea("reason").Required(),
		)

	// In modal mode, button triggers Filament.openModal
	btnHTML := action.RenderButton("#")
	if !strings.Contains(btnHTML, "Filament.openModal('modal_refund')") {
		t.Errorf("Expected modal trigger on action button, got: %s", btnHTML)
	}

	dummyCtx := filament.NewContext(nil, filament.NewPanel("admin"))
	modalHTML := action.RenderModal(dummyCtx, "/admin/orders/1/refund")

	if !strings.Contains(modalHTML, `id="modal_refund"`) {
		t.Errorf("Expected modal backdrop with id 'modal_refund'")
	}
	if !strings.Contains(modalHTML, `Issue Refund`) {
		t.Errorf("Expected modal heading 'Issue Refund'")
	}
	if !strings.Contains(modalHTML, `name="amount"`) {
		t.Errorf("Expected amount field in modal form")
	}
	if !strings.Contains(modalHTML, `name="reason"`) {
		t.Errorf("Expected reason field in modal form")
	}
}

func TestActionAuthorization(t *testing.T) {
	action := actions.New("admin_only").
		Authorize(func(ctx *filament.Context) bool {
			return false
		})

	if action.IsVisible(nil) {
		t.Errorf("Expected action to be hidden for unauthorized context")
	}
}
