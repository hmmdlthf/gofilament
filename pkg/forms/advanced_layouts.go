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
	badge       string
	badgeColor  string
	schema      []Component
}

func WizardStep(title string) *WizardStepComponent {
	return &WizardStepComponent{
		title:      title,
		badgeColor: "primary",
		schema:     make([]Component, 0),
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

func (w *WizardStepComponent) Badge(badge string) *WizardStepComponent {
	w.badge = badge
	return w
}

func (w *WizardStepComponent) BadgeColor(color string) *WizardStepComponent {
	w.badgeColor = color
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

func (w *WizardStepComponent) GetIcon() string {
	return w.icon
}

func (w *WizardStepComponent) GetBadge() string {
	return w.badge
}

func (w *WizardStepComponent) GetBadgeColor() string {
	return w.badgeColor
}

type WizardComponent struct {
	steps               []*WizardStepComponent
	submitAction        string
	previousActionLabel string
	nextActionLabel     string
	cancelUrl           string
	cancelLabel         string
	startStep           int
	skippable           bool
	hasProgressBar      bool
}

func Wizard(steps ...*WizardStepComponent) *WizardComponent {
	return &WizardComponent{
		steps:               steps,
		submitAction:        "Submit",
		previousActionLabel: "← Previous",
		nextActionLabel:     "Next →",
		cancelLabel:         "Cancel",
		startStep:           1,
		skippable:           false,
		hasProgressBar:      true,
	}
}

func (w *WizardComponent) GetName() string {
	return ""
}

func (w *WizardComponent) SubmitAction(label string) *WizardComponent {
	w.submitAction = label
	return w
}

func (w *WizardComponent) PreviousActionLabel(label string) *WizardComponent {
	w.previousActionLabel = label
	return w
}

func (w *WizardComponent) NextActionLabel(label string) *WizardComponent {
	w.nextActionLabel = label
	return w
}

func (w *WizardComponent) CancelAction(url string, label ...string) *WizardComponent {
	w.cancelUrl = url
	if len(label) > 0 && label[0] != "" {
		w.cancelLabel = label[0]
	}
	return w
}

func (w *WizardComponent) StartStep(step int) *WizardComponent {
	w.startStep = step
	return w
}

func (w *WizardComponent) Skippable(enabled ...bool) *WizardComponent {
	if len(enabled) > 0 {
		w.skippable = enabled[0]
	} else {
		w.skippable = true
	}
	return w
}

func (w *WizardComponent) ProgressBar(enabled ...bool) *WizardComponent {
	if len(enabled) > 0 {
		w.hasProgressBar = enabled[0]
	} else {
		w.hasProgressBar = true
	}
	return w
}

func (w *WizardComponent) GetSteps() []*WizardStepComponent {
	return w.steps
}

func (w *WizardComponent) GetSubmitAction() string {
	return w.submitAction
}

func (w *WizardComponent) GetPreviousActionLabel() string {
	return w.previousActionLabel
}

func (w *WizardComponent) GetNextActionLabel() string {
	return w.nextActionLabel
}

func (w *WizardComponent) GetCancelUrl() string {
	return w.cancelUrl
}

func (w *WizardComponent) GetCancelLabel() string {
	return w.cancelLabel
}

func (w *WizardComponent) GetStartStep() int {
	return w.startStep
}

func (w *WizardComponent) IsSkippable() bool {
	return w.skippable
}

func (w *WizardComponent) HasProgressBar() bool {
	return w.hasProgressBar
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

	// Build titles array for Alpine
	titlesJS := make([]string, len(w.steps))
	for i, s := range w.steps {
		titlesJS[i] = fmt.Sprintf("%q", s.title)
	}
	titlesListStr := strings.Join(titlesJS, ", ")

	// Optional Progress Bar
	var progressBarHTML string
	if w.hasProgressBar {
		progressBarHTML = fmt.Sprintf(`
<div class="wizard-progress-bar-container" style="background-color: var(--gray-200); height: 4px; border-radius: 9999px; overflow: hidden; margin-bottom: 1.5rem;">
  <div class="wizard-progress-bar" style="background-color: var(--primary-600); height: 100%%; transition: width 0.3s ease;" :style="'width: ' + ((currentStep / totalSteps) * 100) + '%%'"></div>
</div>`)
	}

	// Mobile step header
	mobileHeader := fmt.Sprintf(`
<div class="wizard-mobile-header" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; padding-bottom: 0.75rem; border-bottom: 1px solid var(--gray-100); font-size: 0.875rem;">
  <span style="font-weight: 600; color: var(--gray-700);">Step <span x-text="currentStep"></span> of %d</span>
  <span style="font-weight: 600; color: var(--primary-600);" x-text="stepTitles[currentStep - 1]"></span>
</div>`, totalSteps)

	// Build Step Header Bar
	var header strings.Builder
	header.WriteString(`<div class="wizard-header" style="display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid var(--gray-200); padding-bottom: 1.25rem; margin-bottom: 1.5rem; gap: 0.5rem; overflow-x: auto;">`)

	for i, step := range w.steps {
		stepNum := i + 1

		// Badge styling
		var badgeHTML string
		if step.badge != "" {
			var badgeBg, badgeColor string
			switch step.badgeColor {
			case "success":
				badgeBg = "#d1fae5"
				badgeColor = "#065f46"
			case "warning":
				badgeBg = "#fef3c7"
				badgeColor = "#92400e"
			case "danger":
				badgeBg = "#fee2e2"
				badgeColor = "#991b1b"
			case "info":
				badgeBg = "#e0f2fe"
				badgeColor = "#075985"
			default:
				badgeBg = "var(--primary-100, #e0e7ff)"
				badgeColor = "var(--primary-800, #3730a3)"
			}
			badgeHTML = fmt.Sprintf(`<span class="wizard-step-badge" style="font-size: 0.6875rem; font-weight: 600; padding: 0.125rem 0.375rem; border-radius: 9999px; background-color: %s; color: %s; margin-left: 0.375rem;">%s</span>`, badgeBg, badgeColor, html.EscapeString(step.badge))
		}

		// Icon / Indicator inside circle
		indicatorHTML := fmt.Sprintf(`<span>%d</span>`, stepNum)
		if step.icon != "" {
			if strings.HasPrefix(strings.TrimSpace(step.icon), "<svg") {
				indicatorHTML = fmt.Sprintf(`<span class="wizard-step-icon" style="display: inline-flex; align-items: center; justify-content: center; width: 1.125rem; height: 1.125rem;">%s</span>`, step.icon)
			} else {
				indicatorHTML = fmt.Sprintf(`<i class="%s wizard-step-icon"></i>`, html.EscapeString(step.icon))
			}
		}

		header.WriteString(fmt.Sprintf(`
<div class="wizard-step-pill" @click="goToStep(%d)" style="display: flex; align-items: center; gap: 0.5rem; cursor: pointer; flex: 1; min-width: 8rem;">
  <span class="wizard-step-num" 
    :style="currentStep === %d ? 'background-color: var(--primary-600); color: #fff;' : (currentStep > %d ? 'background-color: #10b981; color: #fff;' : 'background-color: var(--gray-200); color: var(--gray-600);')"
    style="width: 1.875rem; height: 1.875rem; border-radius: 9999px; display: inline-flex; align-items: center; justify-content: center; font-size: 0.75rem; font-weight: 700; transition: all 0.2s ease;">
    <template x-if="currentStep > %d"><span>✓</span></template>
    <template x-if="currentStep <= %d">%s</template>
  </span>
  <div>
    <div style="font-size: 0.875rem; font-weight: 600; color: var(--gray-900); display: flex; align-items: center;" :style="currentStep === %d ? 'color: var(--primary-600);' : ''">%s%s</div>
    %s
  </div>
</div>`,
			stepNum,
			stepNum, stepNum,
			stepNum,
			stepNum, indicatorHTML,
			stepNum, html.EscapeString(step.title), badgeHTML,
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
<div id="wizard_step_%d" class="wizard-step-pane" x-show="currentStep === %d" style="display: none;" :style="currentStep === %d ? 'display: block;' : 'display: none;'">
  %s
</div>`, stepNum, stepNum, stepNum, stepBody.String()))
	}

	// Cancel button if configured
	var cancelBtnHTML string
	if w.cancelUrl != "" {
		cancelBtnHTML = fmt.Sprintf(`<a href="%s" class="btn btn-secondary wizard-btn-cancel" style="text-decoration: none;">%s</a>`, html.EscapeString(w.cancelUrl), html.EscapeString(w.cancelLabel))
	}

	// Build Navigation Footer
	footer := fmt.Sprintf(`
<div class="wizard-footer" style="display: flex; justify-content: space-between; align-items: center; margin-top: 2rem; border-top: 1px solid var(--gray-200); padding-top: 1.25rem;">
  <div style="display: flex; gap: 0.5rem; align-items: center;">
    %s
    <button type="button" class="btn btn-secondary wizard-btn-prev" x-show="currentStep > 1" @click="prevStep()">%s</button>
  </div>
  <div style="display: flex; gap: 0.75rem; align-items: center;">
    <button type="button" class="btn btn-primary wizard-btn-next" x-show="currentStep < totalSteps" @click="nextStep()">%s</button>
    <button type="submit" form="filament-form" class="btn btn-primary wizard-btn-submit" x-show="currentStep === totalSteps">%s</button>
  </div>
</div>`, cancelBtnHTML, html.EscapeString(w.previousActionLabel), html.EscapeString(w.nextActionLabel), html.EscapeString(w.submitAction))

	return fmt.Sprintf(`
<div class="filament-card wizard-container" x-data="{
  currentStep: %d,
  totalSteps: %d,
  maxStep: %d,
  skippable: %t,
  stepTitles: [%s],
  validateCurrentStep() {
    const pane = document.getElementById('wizard_step_' + this.currentStep);
    if (!pane) return true;
    const inputs = pane.querySelectorAll('input, select, textarea');
    for (let el of inputs) {
      if (!el.checkValidity()) {
        el.reportValidity();
        return false;
      }
    }
    return true;
  },
  goToStep(step) {
    if (this.skippable || step < this.currentStep || step <= this.maxStep) {
      this.currentStep = step;
      return;
    }
    if (step === this.currentStep + 1) {
      if (this.validateCurrentStep()) {
        this.currentStep = step;
        if (this.currentStep > this.maxStep) this.maxStep = this.currentStep;
      }
    }
  },
  nextStep() {
    if (this.skippable || this.validateCurrentStep()) {
      if (this.currentStep < this.totalSteps) {
        this.currentStep++;
        if (this.currentStep > this.maxStep) this.maxStep = this.currentStep;
      }
    }
  },
  prevStep() {
    if (this.currentStep > 1) {
      this.currentStep--;
    }
  }
}">
  %s
  %s
  %s
  %s
  %s
</div>`, activeStep, totalSteps, activeStep, w.skippable, titlesListStr, progressBarHTML, mobileHeader, header.String(), content.String(), footer)
}

// ----------------------------------------------------
// Repeater
// ----------------------------------------------------

type RepeaterComponent struct {
	BaseField
	schema          []Component
	minItems        int
	maxItems        int
	collapsible     bool
	collapsed       bool
	reorderable     bool
	addable         bool
	deletable       bool
	gridColumns     int
	addActionLabel  string
	itemLabel       string
	itemLabelFunc   func(item map[string]any, index int) string
}

func Repeater(name string) *RepeaterComponent {
	return &RepeaterComponent{
		BaseField: BaseField{name: name},
		schema:    make([]Component, 0),
		addable:   true,
		deletable: true,
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

func (r *RepeaterComponent) Collapsed(enabled ...bool) *RepeaterComponent {
	if len(enabled) > 0 {
		r.collapsed = enabled[0]
	} else {
		r.collapsed = true
	}
	if r.collapsed {
		r.collapsible = true
	}
	return r
}

func (r *RepeaterComponent) Reorderable(enabled ...bool) *RepeaterComponent {
	if len(enabled) > 0 {
		r.reorderable = enabled[0]
	} else {
		r.reorderable = true
	}
	return r
}

func (r *RepeaterComponent) Addable(enabled ...bool) *RepeaterComponent {
	if len(enabled) > 0 {
		r.addable = enabled[0]
	} else {
		r.addable = true
	}
	return r
}

func (r *RepeaterComponent) Deletable(enabled ...bool) *RepeaterComponent {
	if len(enabled) > 0 {
		r.deletable = enabled[0]
	} else {
		r.deletable = true
	}
	return r
}

func (r *RepeaterComponent) Grid(columns int) *RepeaterComponent {
	r.gridColumns = columns
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

func (r *RepeaterComponent) ItemLabelCallback(fn func(item map[string]any, index int) string) *RepeaterComponent {
	r.itemLabelFunc = fn
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

	gridStyle := ""
	if r.gridColumns > 1 {
		gridStyle = fmt.Sprintf(`style="display: grid; grid-template-columns: repeat(%d, minmax(0, 1fr)); gap: 1rem;"`, r.gridColumns)
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
		if r.itemLabelFunc != nil {
			itemTitle = r.itemLabelFunc(itemData, i)
		} else if r.itemLabel != "" {
			itemTitle = fmt.Sprintf("%s #%d", r.itemLabel, i+1)
		}

		reorderControls := ""
		if r.reorderable {
			reorderControls = `
<div style="display: flex; gap: 0.25rem; margin-right: 0.5rem;">
  <button type="button" title="Move Up" style="background: none; border: 1px solid var(--gray-300); border-radius: 0.25rem; padding: 0.125rem 0.375rem; font-size: 0.75rem; cursor: pointer; color: var(--gray-600);"
          onclick="const row = this.closest('.repeater-item-row'); const prev = row.previousElementSibling; if (prev && prev.classList.contains('repeater-item-row')) { row.parentNode.insertBefore(row, prev); }">&uarr;</button>
  <button type="button" title="Move Down" style="background: none; border: 1px solid var(--gray-300); border-radius: 0.25rem; padding: 0.125rem 0.375rem; font-size: 0.75rem; cursor: pointer; color: var(--gray-600);"
          onclick="const row = this.closest('.repeater-item-row'); const next = row.nextElementSibling; if (next && next.classList.contains('repeater-item-row')) { row.parentNode.insertBefore(next, row); }">&darr;</button>
</div>`
		}

		deleteButton := ""
		if r.deletable {
			deleteButton = `<button type="button" class="repeater-btn-remove" onclick="this.closest('.repeater-item-row').remove();" style="background: none; border: none; color: #ef4444; font-size: 0.75rem; font-weight: 600; cursor: pointer; display: flex; align-items: center; gap: 0.25rem;">&times; Remove</button>`
		}

		collapseButton := ""
		collapseDirective := ""
		if r.collapsible {
			collapseButton = `<button type="button" @click="collapsed = !collapsed" style="background: none; border: none; font-size: 0.75rem; cursor: pointer; color: var(--gray-500); margin-right: 0.375rem;" x-text="collapsed ? '▶' : '▼'"></button>`
			collapseDirective = `x-show="!collapsed"`
		}

		rowsHTML.WriteString(fmt.Sprintf(`
<div class="repeater-item-row" x-data="{ collapsed: %v }" style="border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 1.25rem; background-color: var(--gray-50); margin-bottom: 0.75rem; position: relative; box-shadow: 0 1px 2px rgba(0,0,0,0.02);">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; border-bottom: 1px solid var(--gray-200); padding-bottom: 0.5rem;">
    <div style="display: flex; align-items: center;">
      %s
      <span style="font-weight: 600; font-size: 0.8125rem; color: var(--gray-700);">%s</span>
    </div>
    <div style="display: flex; align-items: center;">
      %s
      %s
    </div>
  </div>
  <div class="repeater-item-body" %s %s>
    %s
  </div>
</div>`, r.collapsed, collapseButton, html.EscapeString(itemTitle), reorderControls, deleteButton, collapseDirective, gridStyle, scopedBody))
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

	reorderTmpl := ""
	if r.reorderable {
		reorderTmpl = `
<div style="display: flex; gap: 0.25rem; margin-right: 0.5rem;">
  <button type="button" title="Move Up" style="background: none; border: 1px solid var(--gray-300); border-radius: 0.25rem; padding: 0.125rem 0.375rem; font-size: 0.75rem; cursor: pointer; color: var(--gray-600);"
          onclick="const row = this.closest('.repeater-item-row'); const prev = row.previousElementSibling; if (prev && prev.classList.contains('repeater-item-row')) { row.parentNode.insertBefore(row, prev); }">&uarr;</button>
  <button type="button" title="Move Down" style="background: none; border: 1px solid var(--gray-300); border-radius: 0.25rem; padding: 0.125rem 0.375rem; font-size: 0.75rem; cursor: pointer; color: var(--gray-600);"
          onclick="const row = this.closest('.repeater-item-row'); const next = row.nextElementSibling; if (next && next.classList.contains('repeater-item-row')) { row.parentNode.insertBefore(next, row); }">&darr;</button>
</div>`
	}

	deleteTmpl := ""
	if r.deletable {
		deleteTmpl = `<button type="button" class="repeater-btn-remove" onclick="this.closest('.repeater-item-row').remove();" style="background: none; border: none; color: #ef4444; font-size: 0.75rem; font-weight: 600; cursor: pointer; display: flex; align-items: center; gap: 0.25rem;">&times; Remove</button>`
	}

	collapseTmplButton := ""
	collapseTmplDirective := ""
	if r.collapsible {
		collapseTmplButton = `<button type="button" @click="collapsed = !collapsed" style="background: none; border: none; font-size: 0.75rem; cursor: pointer; color: var(--gray-500); margin-right: 0.375rem;" x-text="collapsed ? '▶' : '▼'"></button>`
		collapseTmplDirective = `x-show="!collapsed"`
	}

	tmplRow := fmt.Sprintf(`
<div class="repeater-item-row" x-data="{ collapsed: false }" style="border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 1.25rem; background-color: var(--gray-50); margin-bottom: 0.75rem; position: relative; box-shadow: 0 1px 2px rgba(0,0,0,0.02);">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; border-bottom: 1px solid var(--gray-200); padding-bottom: 0.5rem;">
    <div style="display: flex; align-items: center;">
      %s
      <span style="font-weight: 600; font-size: 0.8125rem; color: var(--gray-700);">%s</span>
    </div>
    <div style="display: flex; align-items: center;">
      %s
      %s
    </div>
  </div>
  <div class="repeater-item-body" %s %s>
    %s
  </div>
</div>`, collapseTmplButton, tmplTitle, reorderTmpl, deleteTmpl, collapseTmplDirective, gridStyle, scopedTmpl)

	repeaterID := fmt.Sprintf("repeater_%s", r.name)
	tmplID := fmt.Sprintf("tmpl_%s", r.name)
	listID := fmt.Sprintf("list_%s", r.name)

	addButtonHTML := ""
	if r.addable {
		addButtonHTML = fmt.Sprintf(`
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
      :style="max > 0 && count >= max ? 'display: none;' : ''">%s</button>`, tmplID, listID, html.EscapeString(addLabel))
	}

	return fmt.Sprintf(`
<div id="%s" class="form-group repeater-wrapper" x-data="{ count: %d, min: %d, max: %d }">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem;">
    <label class="form-label" style="margin-bottom: 0; font-weight: 600;">%s</label>
    %s
  </div>
  <div id="%s" class="repeater-list">
    %s
  </div>
  <template id="%s">%s</template>
</div>`,
		repeaterID, initialCount, r.minItems, r.maxItems,
		html.EscapeString(r.formatLabel()),
		addButtonHTML,
		listID,
		rowsHTML.String(),
		tmplID, tmplRow,
	)
}
