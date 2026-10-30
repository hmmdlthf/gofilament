package forms

import (
	"context"
	"fmt"
	"html"
	"strings"
)

// ----------------------------------------------------
// Fieldset
// ----------------------------------------------------

type FieldsetComponent struct {
	legend string
	schema []Component
}

func Fieldset(legend string) *FieldsetComponent {
	return &FieldsetComponent{
		legend: legend,
		schema: make([]Component, 0),
	}
}

func (f *FieldsetComponent) GetName() string {
	return ""
}

func (f *FieldsetComponent) Schema(components ...Component) *FieldsetComponent {
	f.schema = components
	return f
}

func (f *FieldsetComponent) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	var body strings.Builder
	for _, comp := range f.schema {
		body.WriteString(comp.Render(ctx, state, errors))
	}
	return fmt.Sprintf(`
<fieldset style="border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 1.25rem; margin-bottom: 1.25rem;">
  <legend style="padding: 0 0.5rem; font-weight: 600; font-size: 0.875rem; color: var(--gray-700);">%s</legend>
  %s
</fieldset>`, html.EscapeString(f.legend), body.String())
}

// ----------------------------------------------------
// Split
// ----------------------------------------------------

type SplitComponent struct {
	schema []Component
}

func Split(components ...Component) *SplitComponent {
	return &SplitComponent{schema: components}
}

func (s *SplitComponent) GetName() string {
	return ""
}

func (s *SplitComponent) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	var body strings.Builder
	body.WriteString(`<div style="display: flex; gap: 1.25rem; flex-wrap: wrap;">`)
	for _, comp := range s.schema {
		body.WriteString(fmt.Sprintf(`<div style="flex: 1; min-width: 16rem;">%s</div>`, comp.Render(ctx, state, errors)))
	}
	body.WriteString(`</div>`)
	return body.String()
}

// ----------------------------------------------------
// Wizard & WizardStep
// ----------------------------------------------------

type WizardStepComponent struct {
	title       string
	description string
	schema      []Component
}

func WizardStep(title string) *WizardStepComponent {
	return &WizardStepComponent{
		title:  title,
		schema: make([]Component, 0),
	}
}

func (w *WizardStepComponent) Description(desc string) *WizardStepComponent {
	w.description = desc
	return w
}

func (w *WizardStepComponent) Schema(components ...Component) *WizardStepComponent {
	w.schema = components
	return w
}

type WizardComponent struct {
	steps []*WizardStepComponent
}

func Wizard(steps ...*WizardStepComponent) *WizardComponent {
	return &WizardComponent{steps: steps}
}

func (w *WizardComponent) GetName() string {
	return ""
}

func (w *WizardComponent) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	if len(w.steps) == 0 {
		return ""
	}

	var header strings.Builder
	var content strings.Builder

	header.WriteString(`<div style="display: flex; justify-content: space-between; border-bottom: 1px solid var(--gray-200); padding-bottom: 1rem; margin-bottom: 1.5rem;">`)
	for i, step := range w.steps {
		numColor := "background-color: var(--primary-600); color: #fff;"
		if i != 0 {
			numColor = "background-color: var(--gray-200); color: var(--gray-600);"
		}
		header.WriteString(fmt.Sprintf(`
<div style="display: flex; align-items: center; gap: 0.5rem;">
  <span style="width: 1.75rem; height: 1.75rem; border-radius: 9999px; display: inline-flex; align-items: center; justify-content: center; font-size: 0.75rem; font-weight: 700; %s">%d</span>
  <div>
    <div style="font-size: 0.875rem; font-weight: 600;">%s</div>
    %s
  </div>
</div>`, numColor, i+1, html.EscapeString(step.title),
			func() string {
				if step.description != "" {
					return fmt.Sprintf(`<div style="font-size: 0.75rem; color: var(--gray-400);">%s</div>`, html.EscapeString(step.description))
				}
				return ""
			}()))

		var stepBody strings.Builder
		for _, comp := range step.schema {
			stepBody.WriteString(comp.Render(ctx, state, errors))
		}
		display := "block"
		if i != 0 {
			display = "none"
		}
		content.WriteString(fmt.Sprintf(`<div style="display: %s;">%s</div>`, display, stepBody.String()))
	}
	header.WriteString(`</div>`)

	return fmt.Sprintf(`
<div class="filament-card">
  %s
  %s
</div>`, header.String(), content.String())
}

// ----------------------------------------------------
// Repeater
// ----------------------------------------------------

type RepeaterComponent struct {
	BaseField
	schema []Component
}

func Repeater(name string) *RepeaterComponent {
	return &RepeaterComponent{
		BaseField: BaseField{name: name},
		schema:    make([]Component, 0),
	}
}

func (r *RepeaterComponent) Label(l string) *RepeaterComponent {
	r.BaseField.Label(l)
	return r
}

func (r *RepeaterComponent) Schema(components ...Component) *RepeaterComponent {
	r.schema = components
	return r
}

func (r *RepeaterComponent) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	var itemBody strings.Builder
	for _, comp := range r.schema {
		itemBody.WriteString(comp.Render(ctx, state, errors))
	}

	return fmt.Sprintf(`
<div class="form-group">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.5rem;">
    <label class="form-label" style="margin-bottom: 0;">%s</label>
    <button type="button" class="btn btn-secondary btn-sm">+ Add Item</button>
  </div>
  <div style="border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 1rem; background-color: var(--gray-50); margin-bottom: 0.5rem;">
    <div style="display: flex; justify-content: flex-end; margin-bottom: 0.5rem;">
      <button type="button" style="background: none; border: none; color: #ef4444; font-size: 0.75rem; cursor: pointer;">Remove</button>
    </div>
    %s
  </div>
</div>`, html.EscapeString(r.formatLabel()), itemBody.String())
}
