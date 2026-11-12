package filament

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"

	"github.com/golaravel/gofilament/pkg/forms"
)

// webActionURL is the web-engine URL of a row action: GET shows the
// confirmation (and the action's form), POST runs its Handle runner.
func webActionURL(basePath, id, name string) string {
	return fmt.Sprintf("%s/%s/actions/%s", basePath, url.PathEscape(id), url.PathEscape(name))
}

// handleActionPage renders the confirm / form page of a runnable row action.
func (r *BaseResource[T]) handleActionPage(ctx *Context, basePath, id, name string, status int, errMsg string, state map[string]any) error {
	_, action, err := r.findForAction(ctx, id, name)
	if err != nil {
		return r.actionPageError(ctx, err)
	}
	heading := action.GetModalHeading()
	if heading == "" {
		heading = action.GetLabel()
	}
	desc := action.GetModalDescription()
	if desc == "" {
		desc = fmt.Sprintf("Run “%s” on this %s?", action.GetLabel(), r.modelName)
	}
	errHTML := ""
	if errMsg != "" {
		errHTML = fmt.Sprintf(`<div class="filament-alert filament-alert-danger" role="alert" style="margin-bottom: 1rem; padding: 0.75rem; border-radius: 0.5rem; background: #fef2f2; color: #b91c1c;">%s</div>`, html.EscapeString(errMsg))
	}
	actionURL := webActionURL(basePath, id, name)
	form := forms.New().Schema(action.FormComponents()...).Action(html.EscapeString(actionURL)).Method("POST")
	if state != nil {
		form.Fill(state)
	}
	btnClass := "btn-primary"
	if c := action.GetColor(); c == "danger" || c == "success" {
		btnClass = "btn-" + c
	}
	content := fmt.Sprintf(`
<div class="filament-header">
  <div>
    <h1 class="filament-title">%s</h1>
    <p class="filament-description">%s</p>
  </div>
  <a href="%s" class="btn btn-secondary">Back to list</a>
</div>
<div class="filament-card">
  %s
  %s
  %s
  <div style="margin-top: 1.5rem; display: flex; justify-content: flex-end; gap: 0.75rem;">
    <a href="%s" class="btn btn-secondary">Cancel</a>
    <button type="submit" form="filament-form" class="btn %s">%s</button>
  </div>
</div>`,
		html.EscapeString(heading), html.EscapeString(desc), basePath,
		errHTML,
		form.Render(ctx.GoContext()),
		ctx.Panel().csrfHiddenInput(ctx.Context, "filament-form"),
		basePath, btnClass, html.EscapeString(action.GetLabel()),
	)
	return r.renderPageStatus(ctx, status, heading, content)
}

// handleActionRun runs a row action submitted from its confirm page.
func (r *BaseResource[T]) handleActionRun(ctx *Context, basePath, id, name string) error {
	if !ctx.Panel().VerifyCSRF(ctx.Context) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: CSRF token mismatch")
	}
	_ = ctx.Request.ParseMultipartForm(32 << 20)
	data := forms.ParseFormData(ctx.Request.PostForm)
	delete(data, CSRFFormField)
	if _, err := r.InvokeAction(ctx, id, name, data); err != nil {
		if errors.Is(err, ErrActionUnavailable) || errors.Is(err, ErrForbidden) {
			return r.actionPageError(ctx, err)
		}
		return r.handleActionPage(ctx, basePath, id, name, http.StatusUnprocessableEntity, err.Error(), data)
	}
	return ctx.Redirect(http.StatusSeeOther, basePath)
}

func (r *BaseResource[T]) actionPageError(ctx *Context, err error) error {
	switch {
	case errors.Is(err, ErrForbidden):
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	case errors.Is(err, ErrActionUnavailable):
		return ctx.String(http.StatusUnprocessableEntity, "422: %s", err.Error())
	default:
		return ctx.String(http.StatusNotFound, "404 Not Found: %v", err)
	}
}
