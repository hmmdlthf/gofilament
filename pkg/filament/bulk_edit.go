package filament

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/golaravel/gofilament/pkg/forms"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
)

// Bulk edit: set one form field to the same value on many selected
// records. Each record goes through the field's validation, CanEdit and the
// trashed check, then OnUpdate with just that field (or OnBulkUpdate once).

// MaxBulkEdit is the most records one bulk edit changes.
const MaxBulkEdit = 1000

// BulkEditResult reports a bulk edit.
type BulkEditResult struct {
	Updated int               `json:"updated"`
	Failed  int               `json:"failed"`
	Errors  map[string]string `json:"errors,omitempty"` // record id -> problem
}

// BulkUpdateHandlerFunc saves one field on many records at once.
type BulkUpdateHandlerFunc func(ctx *Context, ids []string, data map[string]any) error

// BulkEditable lists the form fields users may change on many records at
// once (none by default).
func (r *BaseResource[T]) BulkEditable(fields ...string) *BaseResource[T] {
	r.bulkEditFields = fields
	return r
}

// OnBulkUpdate saves a bulk edit in one call (default: OnUpdate per record).
func (r *BaseResource[T]) OnBulkUpdate(fn BulkUpdateHandlerFunc) *BaseResource[T] {
	r.onBulkUpdate = fn
	return r
}

// BulkEditFields returns the bulk-editable fields the form has.
func (r *BaseResource[T]) BulkEditFields() []string {
	if r.onUpdate == nil && r.onBulkUpdate == nil {
		return nil
	}
	comps := r.formComponents()
	var out []string
	for _, f := range r.bulkEditFields {
		if forms.FindField(comps, f) != nil {
			out = append(out, f)
		}
	}
	return out
}

// InvokeBulkUpdate sets field to value on the records ids.
func (r *BaseResource[T]) InvokeBulkUpdate(ctx *Context, ids []string, field string, value any) (*BulkEditResult, error) {
	allowed := false
	for _, f := range r.BulkEditFields() {
		allowed = allowed || f == field
	}
	if !allowed {
		return nil, ErrColumnNotEditable
	}
	if len(ids) == 0 {
		return nil, &forms.ValidationError{Errors: map[string]string{"ids": "Select at least one record."}}
	}
	if len(ids) > MaxBulkEdit {
		return nil, &forms.ValidationError{Errors: map[string]string{"ids": fmt.Sprintf("Select at most %d records.", MaxBulkEdit)}}
	}
	withSelf(ctx)
	clean, err := forms.Validate(ctx.GoContext(), r.formComponents(), map[string]any{field: value}, false)
	if err != nil {
		return nil, err
	}
	v, ok := clean[field]
	if !ok {
		return nil, ErrColumnNotEditable
	}
	data := map[string]any{field: v}

	res := &BulkEditResult{Errors: map[string]string{}}
	var okIDs []string
	for _, id := range ids {
		err := r.authorizeRecord(ctx, id, r.canEdit)
		if err == nil {
			err = r.refuseTrashed(ctx, id)
		}
		if err == nil && r.onBulkUpdate == nil {
			err = r.onUpdate(ctx, id, map[string]any{field: v})
		}
		if err != nil {
			res.Failed++
			res.Errors[id] = err.Error()
			continue
		}
		okIDs = append(okIDs, id)
	}
	if r.onBulkUpdate != nil && len(okIDs) > 0 {
		if err := r.onBulkUpdate(ctx, okIDs, data); err != nil {
			return nil, err
		}
	}
	res.Updated = len(okIDs)
	if len(res.Errors) == 0 {
		res.Errors = nil
	}
	return res, nil
}

func hiddenIDs(ids []string) string {
	var b strings.Builder
	for _, id := range ids {
		fmt.Fprintf(&b, `<input type="hidden" name="selected_ids[]" value="%s" />`, html.EscapeString(id))
	}
	return b.String()
}

// handleBulkEditStart shows the field choice for the selected records.
func (r *BaseResource[T]) handleBulkEditStart(ctx *Context, basePath string, ids []string) error {
	fields := r.BulkEditFields()
	if len(fields) == 0 {
		return ctx.String(http.StatusNotFound, "Bulk edit is not available")
	}
	if len(ids) == 0 {
		return ctx.Redirect(http.StatusSeeOther, basePath)
	}
	labels := r.formFieldLabelsAll(ctx)
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="filament-header"><div><h1 class="filament-title">Change %d %s</h1><p class="filament-description">Choose the field to set on all selected records.</p></div><a href="%s" class="btn btn-secondary">Cancel</a></div><div class="filament-card" style="display: flex; flex-wrap: wrap; gap: 0.5rem;">`,
		len(ids), html.EscapeString(strings.ToLower(r.pluralModelName)), basePath)
	for _, f := range fields {
		fmt.Fprintf(&b, `<form method="POST" action="%s/bulk-edit">%s<input type="hidden" name="field" value="%s" /><button type="submit" class="btn btn-secondary" data-bulk-field="%s">%s</button></form>`,
			basePath, hiddenIDs(ids), html.EscapeString(f), html.EscapeString(f), html.EscapeString(labels[f]))
	}
	b.WriteString(`</div>`)
	return r.renderPage(ctx, "Change "+r.pluralModelName, b.String())
}

// handleBulkEdit shows the chosen field's input, or saves it.
func (r *BaseResource[T]) handleBulkEdit(ctx *Context, basePath string) error {
	_ = ctx.Request.ParseForm()
	ids := ctx.Request.PostForm["selected_ids[]"]
	field := ctx.Request.PostFormValue("field")
	comp := forms.FindField(r.formComponents(), field)
	allowed := false
	for _, f := range r.BulkEditFields() {
		allowed = allowed || f == field
	}
	if !allowed || comp == nil {
		return ctx.String(http.StatusUnprocessableEntity, "This field cannot be changed in bulk")
	}
	renderInput := func(status int, errs map[string]string, value any) error {
		form := forms.New().Schema(comp).Action(basePath + "/bulk-edit").Method("POST")
		if value != nil {
			form.Fill(map[string]any{field: value})
		}
		if errs != nil {
			form.SetErrors(errs)
		}
		formHTML := form.Render(ctx.GoContext())
		extra := hiddenIDs(ids) + fmt.Sprintf(`<input type="hidden" name="field" value="%s" /><input type="hidden" name="apply" value="1" />`, html.EscapeString(field))
		if i := strings.LastIndex(formHTML, "</form>"); i >= 0 {
			formHTML = formHTML[:i] + extra + formHTML[i:]
		}
		content := fmt.Sprintf(`<div class="filament-header"><div><h1 class="filament-title">Change %d %s</h1></div><a href="%s" class="btn btn-secondary">Cancel</a></div><div class="filament-card">%s<div style="margin-top: 1rem; display: flex; justify-content: flex-end;"><button type="submit" form="filament-form" class="btn btn-primary">Apply to %d records</button></div></div>`,
			len(ids), html.EscapeString(strings.ToLower(r.pluralModelName)), basePath, formHTML, len(ids))
		return r.renderPageStatus(ctx, status, "Change "+r.pluralModelName, content)
	}
	if ctx.Request.PostFormValue("apply") == "" {
		return renderInput(http.StatusOK, nil, nil)
	}
	var value any = ctx.Request.PostFormValue(field)
	if vs := ctx.Request.PostForm[field]; len(vs) > 1 {
		value = vs
	}
	res, err := r.InvokeBulkUpdate(ctx, ids, field, value)
	if err != nil {
		var verr *forms.ValidationError
		if errors.As(err, &verr) {
			return renderInput(http.StatusUnprocessableEntity, verr.Errors, value)
		}
		return ctx.String(webStatus(err), "%s", err.Error())
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="filament-card">%s`, alertHTML("ok", fmt.Sprintf("%d records changed.", res.Updated)))
	if res.Failed > 0 {
		b.WriteString(alertHTML("err", fmt.Sprintf("%d records were not changed.", res.Failed)))
		b.WriteString(`<ul style="font-size: 0.875rem; color: #b91c1c;">`)
		for id, msg := range res.Errors {
			fmt.Fprintf(&b, `<li>#%s: %s</li>`, html.EscapeString(id), html.EscapeString(msg))
		}
		b.WriteString(`</ul>`)
	}
	fmt.Fprintf(&b, `<a href="%s" class="btn btn-primary">Back to %s</a></div>`, basePath, html.EscapeString(r.pluralModelName))
	return r.renderPage(ctx, "Change "+r.pluralModelName, b.String())
}

// formFieldLabelsAll maps every form field to its label (or its name).
func (r *BaseResource[T]) formFieldLabelsAll(ctx *Context) map[string]string {
	out := map[string]string{}
	var walk func(ns []forms.Node)
	walk = func(ns []forms.Node) {
		for _, n := range ns {
			if n.Kind == "field" && n.Name != "" {
				out[n.Name] = n.Label
				if n.Label == "" {
					out[n.Name] = humanizeField(n.Name)
				}
			}
			walk(n.Children)
		}
	}
	walk(r.DescribeForm(ctx))
	return out
}

func (p *Panel) mountBulkEditAPI(r *router.Router, res ResourceInterface, resPath string, preflight func(c *gcontext.Context) error) {
	be, ok := res.(interface {
		InvokeBulkUpdate(ctx *Context, ids []string, field string, value any) (*BulkEditResult, error)
	})
	if !ok {
		return
	}
	r.OPTIONS(resPath+"/bulk-update", preflight)
	r.POST(resPath+"/bulk-update", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		var in struct {
			IDs   []string `json:"ids"`
			Field string   `json:"field"`
			Value any      `json:"value"`
		}
		if err := c.BindJSON(&in); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}
		out, err := be.InvokeBulkUpdate(ctx, in.IDs, in.Field, in.Value)
		if err != nil {
			return p.apiError(c, err)
		}
		return c.JSON(http.StatusOK, out)
	})
}
