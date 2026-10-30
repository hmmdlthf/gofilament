package actions

import (
	"fmt"
	"html"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
)

// ActionHandler defines the execution logic for an action.
type ActionHandler func(ctx *filament.Context, record any, data map[string]any) error

// Action represents an interactive executable operation.
type Action struct {
	name                 string
	label                string
	icon                 string
	color                string
	slideOver            bool
	requiresConfirmation bool
	modalHeading         string
	modalDescription     string
	form                 *forms.Form
	authorizer           func(ctx *filament.Context) bool
	handler              ActionHandler
}

func New(name string) *Action {
	return &Action{
		name:  name,
		label: name,
		color: "primary",
	}
}

func (a *Action) Label(l string) *Action {
	a.label = l
	return a
}

func (a *Action) Icon(i string) *Action {
	a.icon = i
	return a
}

func (a *Action) Color(c string) *Action {
	a.color = c
	return a
}

func (a *Action) SlideOver(s ...bool) *Action {
	a.slideOver = len(s) == 0 || s[0]
	return a
}

func (a *Action) RequiresConfirmation(req ...bool) *Action {
	a.requiresConfirmation = len(req) == 0 || req[0]
	return a
}

func (a *Action) ModalHeading(h string) *Action {
	a.modalHeading = h
	return a
}

func (a *Action) ModalDescription(d string) *Action {
	a.modalDescription = d
	return a
}

func (a *Action) Form(components ...forms.Component) *Action {
	a.form = forms.New().Schema(components...)
	return a
}

func (a *Action) Authorize(fn func(ctx *filament.Context) bool) *Action {
	a.authorizer = fn
	return a
}

func (a *Action) Action(handler ActionHandler) *Action {
	a.handler = handler
	return a
}

func (a *Action) IsVisible(ctx *filament.Context) bool {
	if a.authorizer != nil {
		return a.authorizer(ctx)
	}
	return true
}

func (a *Action) RenderButton(targetURL string) string {
	btnClass := "btn-secondary"
	if a.color == "primary" {
		btnClass = "btn-primary"
	} else if a.color == "danger" {
		btnClass = "btn-danger"
	}

	clickAttr := ""
	if a.form != nil {
		clickAttr = fmt.Sprintf(`onclick="Filament.openModal('modal_%s'); return false;"`, a.name)
	} else if a.requiresConfirmation {
		confirmMsg := a.modalDescription
		if confirmMsg == "" {
			confirmMsg = "Are you sure you want to proceed?"
		}
		clickAttr = fmt.Sprintf(`onclick="return confirm('%s');"`, html.EscapeString(confirmMsg))
	}

	return fmt.Sprintf(`<a href="%s" class="btn %s" %s>%s</a>`,
		targetURL, btnClass, clickAttr, html.EscapeString(a.label))
}

func (a *Action) RenderModal(ctx *filament.Context, submitURL string) string {
	if a.form == nil {
		return ""
	}

	heading := a.modalHeading
	if heading == "" {
		heading = a.label
	}

	a.form.Action(submitURL).Method("POST")

	return fmt.Sprintf(`
<div id="modal_%s" class="modal-backdrop" style="display: none;">
  <div class="modal-content">
    <div class="modal-header">
      <h3 style="font-size: 1.125rem; font-weight: 600;">%s</h3>
      <button type="button" onclick="Filament.closeModal('modal_%s')" style="background:none; border:none; font-size: 1.25rem; cursor: pointer;">&times;</button>
    </div>
    <div class="modal-body">
      %s
    </div>
    <div class="modal-footer">
      <button type="button" class="btn btn-secondary" onclick="Filament.closeModal('modal_%s')">Cancel</button>
      <button type="submit" form="filament-form" class="btn btn-primary">Submit</button>
    </div>
  </div>
</div>`,
		a.name,
		html.EscapeString(heading),
		a.name,
		a.form.Render(ctx.GoContext()),
		a.name,
	)
}
