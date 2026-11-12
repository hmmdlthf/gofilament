package tables

import (
	"context"

	"github.com/golaravel/gofilament/pkg/forms"
)

// ActionHandler runs a row action. data holds the action form's values (empty
// when the action has no form). Returning an error aborts the action; its
// message is shown to the user when it is a validation-style error.
type ActionHandler[T any] func(ctx context.Context, record T, data map[string]any) error

// Handle sets what the action does when it is run from an API client.
func (a *TableAction[T]) Handle(fn ActionHandler[T]) *TableAction[T] {
	a.runner = fn
	return a
}

// Form asks for input before the action runs (shown in a modal).
func (a *TableAction[T]) Form(fn func(f *forms.Form) *forms.Form) *TableAction[T] {
	a.formBuilder = fn
	return a
}

// Authorize restricts who may run the action on a record.
func (a *TableAction[T]) Authorize(fn func(ctx context.Context, record T) bool) *TableAction[T] {
	a.authorize = fn
	return a
}

// ModalHeading / ModalDescription set the confirmation or form modal texts.
func (a *TableAction[T]) ModalHeading(s string) *TableAction[T] {
	a.modalHeading = s
	return a
}

func (a *TableAction[T]) ModalDescription(s string) *TableAction[T] {
	a.modalDescription = s
	return a
}

// SuccessMessage is shown after the action ran.
func (a *TableAction[T]) SuccessMessage(s string) *TableAction[T] {
	a.successMessage = s
	return a
}

// IsRunnable reports whether the action can be run through the API.
func (a *TableAction[T]) IsRunnable() bool { return a.runner != nil }

// IsLink reports whether the action only opens a URL (set with URL and no Handle).
func (a *TableAction[T]) IsLink() bool { return a.urlFunc != nil && a.runner == nil }

// LinkFor returns the URL a link action opens for record.
func (a *TableAction[T]) LinkFor(record T) string {
	if a.urlFunc == nil {
		return ""
	}
	return a.urlFunc(record)
}

// IsAvailable reports whether the action is visible and allowed for record.
func (a *TableAction[T]) IsAvailable(ctx context.Context, record T) bool {
	if a.visibleFunc != nil && !a.visibleFunc(record) {
		return false
	}
	return a.authorize == nil || a.authorize(ctx, record)
}

// Run executes the action.
func (a *TableAction[T]) Run(ctx context.Context, record T, data map[string]any) error {
	if data == nil {
		data = map[string]any{}
	}
	return a.runner(ctx, record, data)
}

// FormComponents returns the action's form (nil when it has none).
func (a *TableAction[T]) FormComponents() []forms.Component {
	if a.formBuilder == nil {
		return nil
	}
	return a.formBuilder(forms.New()).GetSchema()
}

// GetModalHeading / GetModalDescription return the confirmation texts.
func (a *TableAction[T]) GetModalHeading() string     { return a.modalHeading }
func (a *TableAction[T]) GetModalDescription() string { return a.modalDescription }

// GetColor returns the action's color (primary, danger, warning, secondary...).
func (a *TableAction[T]) GetColor() string { return a.color }

// GetSuccessMessage returns the message shown after the action ran.
func (a *TableAction[T]) GetSuccessMessage() string { return a.successMessage }

// FindAction returns the row action named name.
func (t *Table[T]) FindAction(name string) *TableAction[T] {
	for _, a := range t.actions {
		if a.name == name {
			return a
		}
	}
	return nil
}
