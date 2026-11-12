package filament

import (
	"errors"
	"net/http"

	"github.com/golaravel/gofilament/pkg/forms"
)

// ErrRecordNotFound is returned (404) when a write names a record the
// resource cannot load. A write is never run when its record cannot be
// checked against the authorization callbacks.
var ErrRecordNotFound error = &endpointError{status: http.StatusNotFound, msg: "Record not found"}

// ErrColumnNotEditable is returned (422) for inline edits of a column that is
// not editable in the table.
var ErrColumnNotEditable error = &endpointError{status: http.StatusUnprocessableEntity, msg: "This column cannot be edited"}

// AcceptUnvalidatedInput turns off server-side form validation for this
// resource: create and update handlers receive the request data as sent,
// including keys that are not form fields. Only use it when your handlers
// validate and whitelist the data themselves.
func (r *BaseResource[T]) AcceptUnvalidatedInput() *BaseResource[T] {
	r.acceptUnvalidated = true
	return r
}

// formComponents returns the resource form's components (nil without a form).
func (r *BaseResource[T]) formComponents() []forms.Component {
	if r.formBuilder == nil {
		return nil
	}
	return r.formBuilder(forms.New()).GetSchema()
}

// validateInput applies forms.Validate with the resource form. Resources
// without a form (API-only handlers) get the data unchanged.
func (r *BaseResource[T]) validateInput(ctx *Context, data map[string]any, creating bool) (map[string]any, error) {
	comps := r.formComponents()
	if comps == nil || r.acceptUnvalidated {
		return data, nil
	}
	withSelf(ctx)
	return forms.Validate(ctx.GoContext(), comps, data, creating)
}

// authorizeRecord loads the record a write targets and checks canView and
// check against it. Lookup errors block the write. Resources without a
// finder cannot be checked per record and are only gated by their
// resource-level callbacks.
func (r *BaseResource[T]) authorizeRecord(ctx *Context, id string, check func(ctx *Context, record T) bool) error {
	if r.onFind == nil {
		return nil
	}
	record, err := r.onFind(ctx, id)
	if err != nil {
		return ErrRecordNotFound
	}
	if r.canView != nil && !r.canView(ctx, record) {
		return ErrForbidden
	}
	if check != nil && !check(ctx, record) {
		return ErrForbidden
	}
	return nil
}

// editableColumn reports whether name is an editable column of the table and
// returns its options (select columns).
func (r *BaseResource[T]) editableColumn(name string) (bool, map[string]string) {
	for _, c := range r.GetColumns() {
		if c.Name == name && c.Editable {
			return true, c.Options
		}
	}
	return false, nil
}

// webStatus maps a guard error to an HTTP status for the web engine.
func webStatus(err error) int {
	var coded interface{ StatusCode() int }
	switch {
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.As(err, &coded):
		return coded.StatusCode()
	}
	return http.StatusInternalServerError
}
