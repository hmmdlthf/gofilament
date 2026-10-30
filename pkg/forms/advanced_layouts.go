package forms

import (
	"context"
	"fmt"
	"html"
	"regexp"
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

func (f *FieldsetComponent) GetSchema() []Component {
	return f.schema
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

func (s *SplitComponent) GetSchema() []Component {
	return s.schema
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
	icon        string
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

func (w *WizardStepComponent) Icon(icon string) *WizardStepComponent {
	w.icon = icon
	return w
}

func (w *WizardStepComponent) Schema(components ...Component) *WizardStepComponent {
	w.schema = components
	return w
}

func (w *WizardStepComponent) GetSchema() []Component {
	return w.schema
}

func (w *WizardStepComponent) GetTitle() string {
	return w.title
}

func (w *WizardStepComponent) GetDescription() string {
	return w.description
}

type WizardComponent struct {
	steps        []*WizardStepComponent
	submitAction string
	startStep    int
}

func Wizard(steps ...*WizardStepComponent) *WizardComponent {
	return &WizardComponent{
		steps:        steps,
		submitAction: "Submit",
		startStep:    1,
	}
}

func (w *WizardComponent) GetName() string {
	return ""
}

func (w *WizardComponent) SubmitAction(label string) *WizardComponent {
	w.submitAction = label
	return w
}

func (w *WizardComponent) StartStep(step int) *WizardComponent {
	w.startStep = step
	return w
}

func (w *WizardComponent) GetSteps() []*WizardStepComponent {
	return w.steps
}

func (w *WizardComponent) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	if len(w.steps) == 0 {
		return ""
	}

	// Detect if any step has validation errors to auto-activate that step
	activeStep := w.startStep
	if activeStep < 1 || activeStep > len(w.steps) {
		activeStep = 1
	}
	if len(errors) > 0 {
		for i, step := range w.steps {
			for _, comp := range step.schema {
				if name := comp.GetName(); name != "" && errors[name] != "" {
					activeStep = i + 1
					break
				}
			}
		}
	}

	totalSteps := len(w.steps)

	// Build Step Header Bar
	var header strings.Builder
	header.WriteString(`<div class="wizard-header" style="display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--gray-200); padding-bottom: 1.25rem; margin-bottom: 1.5rem; gap: 0.5rem; overflow-x: auto;">`)

	for i, step := range w.steps {
		stepNum := i + 1
		header.WriteString(fmt.Sprintf(`
<div class="wizard-step-pill" @click="currentStep = %d" style="display: flex; align-items: center; gap: 0.5rem; cursor: pointer; flex: 1; min-width: 8rem;">
  <span class="wizard-step-num" 
    :style="currentStep === %d ? 'background-color: var(--primary-600); color: #fff;' : (currentStep > %d ? 'background-color: #10b981; color: #fff;' : 'background-color: var(--gray-200); color: var(--gray-600);')"
    style="width: 1.875rem; height: 1.875rem; border-radius: 9999px; display: inline-flex; align-items: center; justify-content: center; font-size: 0.75rem; font-weight: 700; transition: all 0.2s ease;">
    <template x-if="currentStep > %d"><span>✓</span></template>
    <template x-if="currentStep <= %d"><span>%d</span></template>
  </span>
  <div>
    <div style="font-size: 0.875rem; font-weight: 600; color: var(--gray-900);" :style="currentStep === %d ? 'color: var(--primary-600);' : ''">%s</div>
    %s
  </div>
</div>`,
			stepNum,
			stepNum, stepNum,
			stepNum,
			stepNum, stepNum,
			stepNum, html.EscapeString(step.title),
			func() string {
				if step.description != "" {
					return fmt.Sprintf(`<div style="font-size: 0.75rem; color: var(--gray-400);">%s</div>`, html.EscapeString(step.description))
				}
				return ""
			}(),
		))

		// Separator line between steps
		if i < totalSteps-1 {
			header.WriteString(`<div style="height: 2px; flex: 1; background-color: var(--gray-200); margin: 0 0.5rem;" :style="currentStep > ` + fmt.Sprintf("%d", stepNum) + ` ? 'background-color: #10b981;' : 'background-color: var(--gray-200);'"></div>`)
		}
	}
	header.WriteString(`</div>`)

	// Build Step Panes
	var content strings.Builder
	for i, step := range w.steps {
		stepNum := i + 1
		var stepBody strings.Builder
		for _, comp := range step.schema {
			stepBody.WriteString(comp.Render(ctx, state, errors))
		}

		content.WriteString(fmt.Sprintf(`
<div class="wizard-step-pane" x-show="currentStep === %d" style="display: none;" :style="currentStep === %d ? 'display: block;' : 'display: none;'">
  %s
</div>`, stepNum, stepNum, stepBody.String()))
	}

	// Build Navigation Footer
	footer := fmt.Sprintf(`
<div class="wizard-footer" style="display: flex; justify-content: space-between; align-items: center; margin-top: 2rem; border-top: 1px solid var(--gray-200); padding-top: 1.25rem;">
  <div>
    <button type="button" class="btn btn-secondary wizard-btn-prev" x-show="currentStep > 1" @click="currentStep--">← Previous</button>
  </div>
  <div style="display: flex; gap: 0.75rem;">
    <button type="button" class="btn btn-primary wizard-btn-next" x-show="currentStep < totalSteps" @click="currentStep++">Next →</button>
    <button type="submit" form="filament-form" class="btn btn-primary wizard-btn-submit" x-show="currentStep === totalSteps">%s</button>
  </div>
</div>`, html.EscapeString(w.submitAction))

	return fmt.Sprintf(`
<div class="filament-card wizard-container" x-data="{ currentStep: %d, totalSteps: %d }">
  %s
  %s
  %s
</div>`, activeStep, totalSteps, header.String(), content.String(), footer)
}

// ----------------------------------------------------
// Repeater
// ----------------------------------------------------

type RepeaterComponent struct {
	BaseField
	schema         []Component
	minItems       int
	maxItems       int
	collapsible    bool
	addActionLabel string
	itemLabel      string
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

func (r *RepeaterComponent) MinItems(min int) *RepeaterComponent {
	r.minItems = min
	return r
}

func (r *RepeaterComponent) MaxItems(max int) *RepeaterComponent {
	r.maxItems = max
	return r
}

func (r *RepeaterComponent) Collapsible(enabled ...bool) *RepeaterComponent {
	if len(enabled) > 0 {
		r.collapsible = enabled[0]
	} else {
		r.collapsible = true
	}
	return r
}

func (r *RepeaterComponent) AddActionLabel(label string) *RepeaterComponent {
	r.addActionLabel = label
	return r
}

func (r *RepeaterComponent) ItemLabel(label string) *RepeaterComponent {
	r.itemLabel = label
	return r
}

func (r *RepeaterComponent) GetSchema() []Component {
	return r.schema
}

var (
	nameAttrRegex = regexp.MustCompile(`name="([^"\[\]]+)"`)
	idAttrRegex   = regexp.MustCompile(`id="([^"\[\]]+)"`)
)

func scopeFieldNames(content, parentName, indexStr string) string {
	res := nameAttrRegex.ReplaceAllString(content, fmt.Sprintf(`name="%s[%s][$1]"`, parentName, indexStr))
	res = idAttrRegex.ReplaceAllString(res, fmt.Sprintf(`id="%s_%s_$1"`, parentName, indexStr))
	return res
}

func (r *RepeaterComponent) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	// Extract existing items from state
	var items []map[string]any
	if raw, exists := state[r.name]; exists && raw != nil {
		if sliceMap, ok := raw.([]map[string]any); ok {
			items = sliceMap
		} else if sliceAny, ok := raw.([]any); ok {
			for _, it := range sliceAny {
				if m, ok := it.(map[string]any); ok {
					items = append(items, m)
				}
			}
		}
	}

	// Ensure minItems are present
	if len(items) < r.minItems {
		for len(items) < r.minItems {
			items = append(items, make(map[string]any))
		}
	}

	initialCount := len(items)
	addLabel := r.addActionLabel
	if addLabel == "" {
		addLabel = "+ Add Item"
	}

	// Render existing items
	var rowsHTML strings.Builder
	for i, itemData := range items {
		var rowBody strings.Builder
		for _, comp := range r.schema {
			rowBody.WriteString(comp.Render(ctx, itemData, errors))
		}

		scopedBody := scopeFieldNames(rowBody.String(), r.name, fmt.Sprintf("%d", i))
		itemTitle := fmt.Sprintf("#%d", i+1)
		if r.itemLabel != "" {
			itemTitle = fmt.Sprintf("%s #%d", r.itemLabel, i+1)
		}

		rowsHTML.WriteString(fmt.Sprintf(`
<div class="repeater-item-row" style="border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 1.25rem; background-color: var(--gray-50); margin-bottom: 0.75rem; position: relative;">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; border-bottom: 1px solid var(--gray-200); padding-bottom: 0.5rem;">
    <span style="font-weight: 600; font-size: 0.8125rem; color: var(--gray-700);">%s</span>
    <button type="button" class="repeater-btn-remove" onclick="this.closest('.repeater-item-row').remove();" style="background: none; border: none; color: #ef4444; font-size: 0.75rem; font-weight: 600; cursor: pointer;">Remove</button>
  </div>
  %s
</div>`, html.EscapeString(itemTitle), scopedBody))
	}

	// Build template for dynamically added rows
	var tmplBody strings.Builder
	for _, comp := range r.schema {
		tmplBody.WriteString(comp.Render(ctx, make(map[string]any), nil))
	}
	scopedTmpl := scopeFieldNames(tmplBody.String(), r.name, "__INDEX__")
	tmplTitle := "#__DISPLAY_INDEX__"
	if r.itemLabel != "" {
		tmplTitle = fmt.Sprintf("%s #__DISPLAY_INDEX__", r.itemLabel)
	}

	tmplRow := fmt.Sprintf(`
<div class="repeater-item-row" style="border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 1.25rem; background-color: var(--gray-50); margin-bottom: 0.75rem; position: relative;">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; border-bottom: 1px solid var(--gray-200); padding-bottom: 0.5rem;">
    <span style="font-weight: 600; font-size: 0.8125rem; color: var(--gray-700);">%s</span>
    <button type="button" class="repeater-btn-remove" onclick="this.closest('.repeater-item-row').remove();" style="background: none; border: none; color: #ef4444; font-size: 0.75rem; font-weight: 600; cursor: pointer;">Remove</button>
  </div>
  %s
</div>`, tmplTitle, scopedTmpl)

	repeaterID := fmt.Sprintf("repeater_%s", r.name)
	tmplID := fmt.Sprintf("tmpl_%s", r.name)
	listID := fmt.Sprintf("list_%s", r.name)

	return fmt.Sprintf(`
<div id="%s" class="form-group repeater-wrapper" x-data="{ count: %d, min: %d, max: %d }">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem;">
    <label class="form-label" style="margin-bottom: 0; font-weight: 600;">%s</label>
    <button type="button" class="btn btn-secondary btn-sm repeater-btn-add"
      @click="
        if (max > 0 && count >= max) return;
        const tmpl = document.getElementById('%s');
        const nextIdx = count;
        const displayIdx = nextIdx + 1;
        const row = tmpl.innerHTML.replace(/__INDEX__/g, nextIdx).replace(/__DISPLAY_INDEX__/g, displayIdx);
        document.getElementById('%s').insertAdjacentHTML('beforeend', row);
        count++;
      "
      :style="max > 0 && count >= max ? 'display: none;' : ''">%s</button>
  </div>
  <div id="%s" class="repeater-list">
    %s
  </div>
  <template id="%s">%s</template>
</div>`,
		repeaterID, initialCount, r.minItems, r.maxItems,
		html.EscapeString(r.formatLabel()),
		tmplID,
		listID,
		html.EscapeString(addLabel),
		listID,
		rowsHTML.String(),
		tmplID, tmplRow,
	)
}
