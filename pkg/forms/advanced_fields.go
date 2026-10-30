package forms

import (
	"context"
	"fmt"
	"html"
	"strings"
)

// ----------------------------------------------------
// Radio
// ----------------------------------------------------

type RadioField struct {
	BaseField
	options      map[string]string
	descriptions map[string]string
	order        []string
}

func Radio(name string) *RadioField {
	return &RadioField{
		BaseField:    BaseField{name: name},
		options:      make(map[string]string),
		descriptions: make(map[string]string),
	}
}

func (f *RadioField) Options(opts map[string]string) *RadioField {
	f.options = opts
	f.order = make([]string, 0, len(opts))
	for k := range opts {
		f.order = append(f.order, k)
	}
	return f
}

func (f *RadioField) Descriptions(descs map[string]string) *RadioField {
	f.descriptions = descs
	return f
}

func (f *RadioField) Label(l string) *RadioField {
	f.BaseField.Label(l)
	return f
}

func (f *RadioField) Default(val any) *RadioField {
	f.BaseField.Default(val)
	return f
}

func (f *RadioField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	currentVal := ""
	if v, ok := state[f.name]; ok && v != nil {
		currentVal = fmt.Sprintf("%v", v)
	} else if f.defaultValue != nil {
		currentVal = fmt.Sprintf("%v", f.defaultValue)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<div class="form-group"><label class="form-label">%s</label><div style="display: flex; flex-direction: column; gap: 0.5rem; margin-top: 0.375rem;">`, html.EscapeString(f.formatLabel())))

	for _, k := range f.order {
		label := f.options[k]
		desc := f.descriptions[k]
		checked := ""
		if k == currentVal {
			checked = "checked"
		}
		descHTML := ""
		if desc != "" {
			descHTML = fmt.Sprintf(`<span style="display: block; font-size: 0.75rem; color: var(--gray-500);">%s</span>`, html.EscapeString(desc))
		}
		sb.WriteString(fmt.Sprintf(`
<label style="display: flex; align-items: flex-start; gap: 0.625rem; cursor: pointer;">
  <input type="radio" name="%s" value="%s" %s style="margin-top: 0.25rem; accent-color: var(--primary-600);" />
  <div>
    <span style="font-size: 0.875rem; font-weight: 500;">%s</span>
    %s
  </div>
</label>`, f.name, html.EscapeString(k), checked, html.EscapeString(label), descHTML))
	}

	sb.WriteString(`</div></div>`)
	return sb.String()
}

// ----------------------------------------------------
// CheckboxList
// ----------------------------------------------------

type CheckboxListField struct {
	BaseField
	options map[string]string
	order   []string
}

func CheckboxList(name string) *CheckboxListField {
	return &CheckboxListField{
		BaseField: BaseField{name: name},
		options:   make(map[string]string),
	}
}

func (f *CheckboxListField) Options(opts map[string]string) *CheckboxListField {
	f.options = opts
	f.order = make([]string, 0, len(opts))
	for k := range opts {
		f.order = append(f.order, k)
	}
	return f
}

func (f *CheckboxListField) Label(l string) *CheckboxListField {
	f.BaseField.Label(l)
	return f
}

func (f *CheckboxListField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	selectedMap := make(map[string]bool)
	if val, ok := state[f.name]; ok {
		if slice, ok := val.([]string); ok {
			for _, item := range slice {
				selectedMap[item] = true
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<div class="form-group"><label class="form-label">%s</label><div style="display: flex; flex-direction: column; gap: 0.5rem; margin-top: 0.375rem;">`, html.EscapeString(f.formatLabel())))

	for _, k := range f.order {
		label := f.options[k]
		checked := ""
		if selectedMap[k] {
			checked = "checked"
		}
		sb.WriteString(fmt.Sprintf(`
<label style="display: flex; align-items: center; gap: 0.625rem; cursor: pointer;">
  <input type="checkbox" name="%s[]" value="%s" %s style="accent-color: var(--primary-600);" />
  <span style="font-size: 0.875rem;">%s</span>
</label>`, f.name, html.EscapeString(k), checked, html.EscapeString(label)))
	}

	sb.WriteString(`</div></div>`)
	return sb.String()
}

// ----------------------------------------------------
// TimePicker
// ----------------------------------------------------

type TimePickerField struct {
	BaseField
}

func TimePicker(name string) *TimePickerField {
	return &TimePickerField{BaseField: BaseField{name: name}}
}

func (f *TimePickerField) Label(l string) *TimePickerField {
	f.BaseField.Label(l)
	return f
}

func (f *TimePickerField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := ""
	if v, ok := state[f.name]; ok && v != nil {
		val = fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf(`
<div class="form-group">
  <label class="form-label" for="%s">%s</label>
  <input type="time" id="%s" name="%s" value="%s" class="form-control" />
</div>`, f.name, html.EscapeString(f.formatLabel()), f.name, f.name, html.EscapeString(val))
}

// ----------------------------------------------------
// MarkdownEditor & RichEditor
// ----------------------------------------------------

type MarkdownEditorField struct {
	BaseField
	rows int
}

func MarkdownEditor(name string) *MarkdownEditorField {
	return &MarkdownEditorField{
		BaseField: BaseField{name: name},
		rows:      8,
	}
}

func (f *MarkdownEditorField) Label(l string) *MarkdownEditorField {
	f.BaseField.Label(l)
	return f
}

func (f *MarkdownEditorField) Rows(r int) *MarkdownEditorField {
	f.rows = r
	return f
}

func (f *MarkdownEditorField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := ""
	if v, ok := state[f.name]; ok && v != nil {
		val = fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf(`
<div class="form-group">
  <label class="form-label" for="%s">%s</label>
  <div style="border: 1px solid var(--gray-300); border-radius: 0.5rem; overflow: hidden;">
    <div style="background-color: var(--gray-50); padding: 0.375rem 0.75rem; border-bottom: 1px solid var(--gray-200); display: flex; gap: 0.5rem; font-size: 0.8125rem;">
      <span style="font-weight: 700; cursor: pointer;">B</span>
      <span style="font-style: italic; cursor: pointer;">I</span>
      <span style="font-family: monospace; cursor: pointer;">&lt;/&gt;</span>
      <span style="cursor: pointer;">🔗</span>
      <span style="cursor: pointer;">• List</span>
    </div>
    <textarea id="%s" name="%s" rows="%d" class="form-control" style="border: none; border-radius: 0; box-shadow: none;">%s</textarea>
  </div>
</div>`, f.name, html.EscapeString(f.formatLabel()), f.name, f.name, f.rows, html.EscapeString(val))
}

// ----------------------------------------------------
// KeyValue
// ----------------------------------------------------

type KeyValueField struct {
	BaseField
	keyLabel   string
	valueLabel string
}

func KeyValue(name string) *KeyValueField {
	return &KeyValueField{
		BaseField:  BaseField{name: name},
		keyLabel:   "Key",
		valueLabel: "Value",
	}
}

func (f *KeyValueField) Label(l string) *KeyValueField {
	f.BaseField.Label(l)
	return f
}

func (f *KeyValueField) KeyLabel(l string) *KeyValueField {
	f.keyLabel = l
	return f
}

func (f *KeyValueField) ValueLabel(l string) *KeyValueField {
	f.valueLabel = l
	return f
}

func (f *KeyValueField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	return fmt.Sprintf(`
<div class="form-group">
  <label class="form-label">%s</label>
  <div style="border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 0.75rem;">
    <div style="display: flex; gap: 0.75rem; margin-bottom: 0.5rem;">
      <input type="text" placeholder="%s" class="form-control" style="font-size: 0.8125rem;" />
      <input type="text" placeholder="%s" class="form-control" style="font-size: 0.8125rem;" />
    </div>
    <button type="button" class="btn btn-secondary btn-sm">+ Add Row</button>
  </div>
</div>`, html.EscapeString(f.formatLabel()), html.EscapeString(f.keyLabel), html.EscapeString(f.valueLabel))
}

// ----------------------------------------------------
// CodeEditor
// ----------------------------------------------------

type CodeEditorField struct {
	BaseField
	language string
}

func CodeEditor(name string) *CodeEditorField {
	return &CodeEditorField{
		BaseField: BaseField{name: name},
		language:  "json",
	}
}

func (f *CodeEditorField) Language(lang string) *CodeEditorField {
	f.language = lang
	return f
}

func (f *CodeEditorField) Label(l string) *CodeEditorField {
	f.BaseField.Label(l)
	return f
}

func (f *CodeEditorField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := ""
	if v, ok := state[f.name]; ok && v != nil {
		val = fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf(`
<div class="form-group">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.375rem;">
    <label class="form-label" for="%s" style="margin-bottom: 0;">%s</label>
    <span style="font-size: 0.75rem; font-family: monospace; color: var(--gray-400);">%s</span>
  </div>
  <textarea id="%s" name="%s" rows="6" class="form-control" style="font-family: monospace; font-size: 0.8125rem; background: #0f172a; color: #f8fafc; border-color: #334155;">%s</textarea>
</div>`, f.name, html.EscapeString(f.formatLabel()), f.language, f.name, f.name, html.EscapeString(val))
}
