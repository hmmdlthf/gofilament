package filament

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/golaravel/gofilament/pkg/forms"
)

// VersionField is the form / request field carrying the version of the
// record the editor started from.
const VersionField = "_version"

const ctxExpectedVersion = "filament_expected_version"

// EditConflictError is returned when a record changed after the editor
// loaded it (HTTP 409).
type EditConflictError struct {
	// Current is the record's version now.
	Current string
}

func (e *EditConflictError) Error() string {
	return "This record was changed by someone else after you opened it. Reload it to see their changes, or save again to overwrite them."
}

// StatusCode is 409 Conflict.
func (e *EditConflictError) StatusCode() int { return http.StatusConflict }

// Versioned turns on edit-conflict detection: version returns a token that
// changes on every save (an updated_at timestamp or a version counter).
// Edit forms carry the token they were opened with, and saving is refused
// with 409 when the record's token changed meanwhile, so one admin cannot
// silently overwrite another's changes. Clients that send no version (older
// API clients) are not checked.
//
// The check reads the record just before saving; to close the remaining
// race, make the write itself conditional with ExpectedVersion(ctx), for
// example UPDATE ... WHERE id = ? AND updated_at = ?.
func (r *BaseResource[T]) Versioned(version func(record T) string) *BaseResource[T] {
	r.versionOf = version
	return r
}

// RecordVersion returns the version token of a record ("" when the
// resource is not Versioned).
func (r *BaseResource[T]) RecordVersion(record any) string {
	if r.versionOf == nil {
		return ""
	}
	if rec, ok := record.(T); ok {
		return r.versionOf(rec)
	}
	return ""
}

// ExpectedVersion returns the version the editor started from, during an
// update of a Versioned resource ("" when none was sent).
func ExpectedVersion(ctx *Context) string {
	if v, ok := ctx.Get(ctxExpectedVersion); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// checkVersion takes the submitted version out of data and compares it with
// the record's current version.
func (r *BaseResource[T]) checkVersion(ctx *Context, id string, data map[string]any) error {
	raw, sent := data[VersionField]
	delete(data, VersionField)
	if r.versionOf == nil || !sent || r.onFind == nil {
		return nil
	}
	want := fmt.Sprint(raw)
	if want == "" {
		return nil
	}
	ctx.Set(ctxExpectedVersion, want)
	rec, err := r.onFind(ctx, id)
	if err != nil {
		return nil // the update itself reports a missing record
	}
	if cur := r.versionOf(rec); cur != want {
		return &EditConflictError{Current: cur}
	}
	return nil
}

// withVersionField adds the hidden version input to a rendered form.
func withVersionField(formHTML, version string) string {
	if version == "" {
		return formHTML
	}
	input := `<input type="hidden" name="` + VersionField + `" value="` + html.EscapeString(version) + `" />`
	if i := strings.LastIndex(formHTML, "</form>"); i >= 0 {
		return formHTML[:i] + input + formHTML[i:]
	}
	return formHTML + input
}

// renderConflict shows the edit form again with the user's input, a
// warning, and the record's current version: saving again overwrites the
// other change on purpose.
func (r *BaseResource[T]) renderConflict(ctx *Context, basePath, id string, data map[string]any, conflict *EditConflictError) error {
	form := forms.New()
	if r.formBuilder != nil {
		form = r.formBuilder(form)
	}
	form.Action(fmt.Sprintf("%s/%s/edit", basePath, id)).Method("POST")
	form.Fill(data)
	content := fmt.Sprintf(`
<div class="filament-header"><div><h1 class="filament-title">Edit %s #%s</h1></div></div>
<div class="filament-alert" role="alert" style="margin-bottom: 1rem; padding: 0.75rem 1rem; border-radius: 0.5rem; background: #fffbeb; color: #92400e; font-size: 0.875rem;">
  <strong>Someone else changed this record after you opened it.</strong>
  Your changes are below and were not saved. <a href="%s/%s/edit">Reload the record</a> to see the other changes,
  or save again to overwrite them.
</div>
<div class="filament-card">
  %s
  <div style="margin-top: 1.5rem; display: flex; justify-content: flex-end; gap: 0.75rem;">
    <a href="%s/%s/edit" class="btn btn-secondary">Reload</a>
    <button type="submit" form="filament-form" class="btn btn-danger">Overwrite</button>
  </div>
</div>`, html.EscapeString(r.modelName), html.EscapeString(id), basePath, html.EscapeString(id),
		withVersionField(form.Render(ctx.GoContext()), conflict.Current), basePath, html.EscapeString(id))
	return r.renderPageStatus(ctx, http.StatusConflict, fmt.Sprintf("Edit %s #%s", r.modelName, id), content)
}
