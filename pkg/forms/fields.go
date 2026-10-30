package forms

import (
	"fmt"
	"html"
	"strings"

	"context"
)

// BaseField provides common field attributes and fluent setters.
type BaseField struct {
	name             string
	label            string
	placeholder      string
	helperText       string
	defaultValue     any
	required         bool
	requiredOnCreate bool
	disabled         bool
	columnSpan       int
	live             bool
	liveDebounce     int // in ms
	afterUpdated     func(state any, form *Form)
	visibleFunc      func(get Get) bool
	dehydrateFunc    func(state any) bool
}

func (f *BaseField) GetName() string {
	return f.name
}

func (f *BaseField) Label(l string) *BaseField {
	f.label = l
	return f
}

func (f *BaseField) Placeholder(p string) *BaseField {
	f.placeholder = p
	return f
}

func (f *BaseField) HelperText(h string) *BaseField {
	f.helperText = h
	return f
}

func (f *BaseField) Default(val any) *BaseField {
	f.defaultValue = val
	return f
}

func (f *BaseField) Required(req ...bool) *BaseField {
	if len(req) > 0 {
		f.required = req[0]
	} else {
		f.required = true
	}
	return f
}

func (f *BaseField) RequiredOnCreate() *BaseField {
	f.requiredOnCreate = true
	return f
}

func (f *BaseField) Disabled(d ...bool) *BaseField {
	if len(d) > 0 {
		f.disabled = d[0]
	} else {
		f.disabled = true
	}
	return f
}

func (f *BaseField) ColumnSpan(span int) *BaseField {
	f.columnSpan = span
	return f
}

func (f *BaseField) Visible(fn func(get Get) bool) *BaseField {
	f.visibleFunc = fn
	return f
}

func (f *BaseField) Dehydrated(fn func(state any) bool) *BaseField {
	f.dehydrateFunc = fn
	return f
}

func (f *BaseField) TriggerAfterUpdated(state any, form *Form) {
	if f.afterUpdated != nil {
		f.afterUpdated(state, form)
	}
}

func (f *BaseField) formatLabel() string {
	if f.label != "" {
		return f.label
	}
	// Auto format name: "first_name" -> "First Name"
	words := strings.Split(f.name, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// ----------------------------------------------------
// TextInput
// ----------------------------------------------------

type TextInputField struct {
	BaseField
	inputType string // text, email, password, number, tel, url
	minLength int
	maxLength int
	prefix    string
	suffix    string
	hashOnSave bool
	uniqueTable string
	uniqueCol   string
}

func TextInput(name string) *TextInputField {
	f := &TextInputField{
		BaseField: BaseField{name: name},
		inputType: "text",
	}
	return f
}

func (f *TextInputField) Email() *TextInputField {
	f.inputType = "email"
	return f
}

func (f *TextInputField) Password() *TextInputField {
	f.inputType = "password"
	return f
}

func (f *TextInputField) Numeric() *TextInputField {
	f.inputType = "number"
	return f
}

func (f *TextInputField) Tel() *TextInputField {
	f.inputType = "tel"
	return f
}

func (f *TextInputField) URL() *TextInputField {
	f.inputType = "url"
	return f
}

func (f *TextInputField) MinLength(min int) *TextInputField {
	f.minLength = min
	return f
}

func (f *TextInputField) MaxLength(max int) *TextInputField {
	f.maxLength = max
	return f
}

func (f *TextInputField) Prefix(p string) *TextInputField {
	f.prefix = p
	return f
}

func (f *TextInputField) Suffix(s string) *TextInputField {
	f.suffix = s
	return f
}

func (f *TextInputField) Hash() *TextInputField {
	f.hashOnSave = true
	return f
}

func (f *TextInputField) Unique(table, col string) *TextInputField {
	f.uniqueTable = table
	f.uniqueCol = col
	return f
}

func (f *TextInputField) Label(l string) *TextInputField {
	f.BaseField.Label(l)
	return f
}

func (f *TextInputField) Placeholder(p string) *TextInputField {
	f.BaseField.Placeholder(p)
	return f
}

func (f *TextInputField) HelperText(h string) *TextInputField {
	f.BaseField.HelperText(h)
	return f
}

func (f *TextInputField) Default(val any) *TextInputField {
	f.BaseField.Default(val)
	return f
}

func (f *TextInputField) Required(req ...bool) *TextInputField {
	f.BaseField.Required(req...)
	return f
}

func (f *TextInputField) RequiredOnCreate() *TextInputField {
	f.BaseField.RequiredOnCreate()
	return f
}

func (f *TextInputField) Disabled(d ...bool) *TextInputField {
	f.BaseField.Disabled(d...)
	return f
}

func (f *TextInputField) ColumnSpan(span int) *TextInputField {
	f.BaseField.ColumnSpan(span)
	return f
}

func (f *TextInputField) Dehydrated(fn func(state any) bool) *TextInputField {
	f.BaseField.Dehydrated(fn)
	return f
}

func (f *TextInputField) Live(opt ...LiveOption) *TextInputField {
	f.live = true
	if len(opt) > 0 {
		f.liveDebounce = int(opt[0].Debounce.Milliseconds())
	} else {
		f.liveDebounce = 300
	}
	return f
}

func (f *TextInputField) AfterStateUpdated(fn func(state any, form *Form)) *TextInputField {
	f.afterUpdated = fn
	return f
}

func (f *TextInputField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := ""
	if v, ok := state[f.name]; ok && v != nil {
		val = fmt.Sprintf("%v", v)
	} else if f.defaultValue != nil {
		val = fmt.Sprintf("%v", f.defaultValue)
	}

	reqAttr := ""
	if f.required {
		reqAttr = "required"
	}
	disAttr := ""
	if f.disabled {
		disAttr = "disabled"
	}

	liveAttr := ""
	if f.live {
		liveAttr = fmt.Sprintf(`oninput="Filament.liveUpdate('filament-form', '%s')"`, f.name)
	}

	errHTML := ""
	if err, exists := errors[f.name]; exists {
		errHTML = fmt.Sprintf(`<p class="form-error">%s</p>`, html.EscapeString(err))
	}

	helperHTML := ""
	if f.helperText != "" {
		helperHTML = fmt.Sprintf(`<p class="form-helper">%s</p>`, html.EscapeString(f.helperText))
	}

	colClass := ""
	if f.columnSpan > 0 {
		colClass = fmt.Sprintf("col-span-%d", f.columnSpan)
	}

	return fmt.Sprintf(`
<div class="form-group %s">
  <label class="form-label" for="%s">%s %s</label>
  <input type="%s" id="%s" name="%s" value="%s" placeholder="%s" class="form-control" %s %s %s />
  %s
  %s
</div>`,
		colClass,
		f.name, html.EscapeString(f.formatLabel()),
		func() string { if f.required { return `<span style="color:#ef4444;">*</span>` }; return "" }(),
		f.inputType, f.name, f.name, html.EscapeString(val), html.EscapeString(f.placeholder),
		reqAttr, disAttr, liveAttr,
		errHTML, helperHTML,
	)
}

// ----------------------------------------------------
// Textarea
// ----------------------------------------------------

type TextareaField struct {
	BaseField
	rows int
	cols int
}

func Textarea(name string) *TextareaField {
	return &TextareaField{
		BaseField: BaseField{name: name},
		rows:      4,
	}
}

func (f *TextareaField) Rows(r int) *TextareaField {
	f.rows = r
	return f
}

func (f *TextareaField) Label(l string) *TextareaField {
	f.BaseField.Label(l)
	return f
}

func (f *TextareaField) Placeholder(p string) *TextareaField {
	f.BaseField.Placeholder(p)
	return f
}

func (f *TextareaField) Required(req ...bool) *TextareaField {
	f.BaseField.Required(req...)
	return f
}

func (f *TextareaField) ColumnSpan(span int) *TextareaField {
	f.BaseField.ColumnSpan(span)
	return f
}

func (f *TextareaField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := ""
	if v, ok := state[f.name]; ok && v != nil {
		val = fmt.Sprintf("%v", v)
	}
	errHTML := ""
	if err, exists := errors[f.name]; exists {
		errHTML = fmt.Sprintf(`<p class="form-error">%s</p>`, html.EscapeString(err))
	}
	colClass := ""
	if f.columnSpan > 0 {
		colClass = fmt.Sprintf("col-span-%d", f.columnSpan)
	}
	return fmt.Sprintf(`
<div class="form-group %s">
  <label class="form-label" for="%s">%s</label>
  <textarea id="%s" name="%s" rows="%d" placeholder="%s" class="form-control">%s</textarea>
  %s
</div>`,
		colClass, f.name, html.EscapeString(f.formatLabel()),
		f.name, f.name, f.rows, html.EscapeString(f.placeholder), html.EscapeString(val),
		errHTML,
	)
}

// ----------------------------------------------------
// Select
// ----------------------------------------------------

type SelectField struct {
	BaseField
	options    map[string]string
	order      []string
	searchable bool
	multiple   bool
}

func Select(name string) *SelectField {
	return &SelectField{
		BaseField: BaseField{name: name},
		options:   make(map[string]string),
	}
}

func (f *SelectField) Options(opts map[string]string) *SelectField {
	f.options = opts
	f.order = make([]string, 0, len(opts))
	for k := range opts {
		f.order = append(f.order, k)
	}
	return f
}

func (f *SelectField) SetOptions(opts map[string]string) {
	f.options = opts
}

func (f *SelectField) Searchable(s ...bool) *SelectField {
	f.searchable = len(s) == 0 || s[0]
	return f
}

func (f *SelectField) Multiple(m ...bool) *SelectField {
	f.multiple = len(m) == 0 || m[0]
	return f
}

func (f *SelectField) Label(l string) *SelectField {
	f.BaseField.Label(l)
	return f
}

func (f *SelectField) Default(val any) *SelectField {
	f.BaseField.Default(val)
	return f
}

func (f *SelectField) Required(req ...bool) *SelectField {
	f.BaseField.Required(req...)
	return f
}

func (f *SelectField) ColumnSpan(span int) *SelectField {
	f.BaseField.ColumnSpan(span)
	return f
}

func (f *SelectField) Live() *SelectField {
	f.live = true
	return f
}

func (f *SelectField) AfterStateUpdated(fn func(state any, form *Form)) *SelectField {
	f.afterUpdated = fn
	return f
}

func (f *SelectField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	currentVal := ""
	if v, ok := state[f.name]; ok && v != nil {
		currentVal = fmt.Sprintf("%v", v)
	} else if f.defaultValue != nil {
		currentVal = fmt.Sprintf("%v", f.defaultValue)
	}

	var optHTML strings.Builder
	optHTML.WriteString(`<option value="">Choose an option...</option>`)
	for _, k := range f.order {
		label := f.options[k]
		selected := ""
		if k == currentVal {
			selected = "selected"
		}
		optHTML.WriteString(fmt.Sprintf(`<option value="%s" %s>%s</option>`, html.EscapeString(k), selected, html.EscapeString(label)))
	}

	liveAttr := ""
	if f.live {
		liveAttr = fmt.Sprintf(`onchange="Filament.liveUpdate('filament-form', '%s')"`, f.name)
	}

	colClass := ""
	if f.columnSpan > 0 {
		colClass = fmt.Sprintf("col-span-%d", f.columnSpan)
	}

	return fmt.Sprintf(`
<div class="form-group %s">
  <label class="form-label" for="%s">%s</label>
  <select id="%s" name="%s" class="form-control" %s>
    %s
  </select>
</div>`,
		colClass, f.name, html.EscapeString(f.formatLabel()),
		f.name, f.name, liveAttr,
		optHTML.String(),
	)
}

// ----------------------------------------------------
// Toggle & Checkbox
// ----------------------------------------------------

type ToggleField struct {
	BaseField
}

func Toggle(name string) *ToggleField {
	return &ToggleField{
		BaseField: BaseField{name: name},
	}
}

func (f *ToggleField) Label(l string) *ToggleField {
	f.BaseField.Label(l)
	return f
}

func (f *ToggleField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	checked := ""
	if v, ok := state[f.name]; ok {
		if b, ok := v.(bool); ok && b {
			checked = "checked"
		} else if s, ok := v.(string); ok && (s == "1" || s == "true" || s == "on") {
			checked = "checked"
		}
	}
	return fmt.Sprintf(`
<div class="form-group" style="display: flex; align-items: center; gap: 0.75rem; margin-top: 1rem;">
  <input type="checkbox" id="%s" name="%s" value="1" %s style="width: 1.25rem; height: 1.25rem; accent-color: var(--primary-600); cursor: pointer;" />
  <label for="%s" style="font-size: 0.875rem; font-weight: 500; cursor: pointer;">%s</label>
</div>`,
		f.name, f.name, checked, f.name, html.EscapeString(f.formatLabel()),
	)
}

// ----------------------------------------------------
// DatePicker & DateTimePicker
// ----------------------------------------------------

type DatePickerField struct {
	BaseField
	includeTime bool
}

func DatePicker(name string) *DatePickerField {
	return &DatePickerField{
		BaseField: BaseField{name: name},
	}
}

func DateTimePicker(name string) *DatePickerField {
	return &DatePickerField{
		BaseField:   BaseField{name: name},
		includeTime: true,
	}
}

func (f *DatePickerField) Label(l string) *DatePickerField {
	f.BaseField.Label(l)
	return f
}

func (f *DatePickerField) Required(req ...bool) *DatePickerField {
	f.BaseField.Required(req...)
	return f
}

func (f *DatePickerField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	inputType := "date"
	if f.includeTime {
		inputType = "datetime-local"
	}
	val := ""
	if v, ok := state[f.name]; ok && v != nil {
		val = fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf(`
<div class="form-group">
  <label class="form-label" for="%s">%s</label>
  <input type="%s" id="%s" name="%s" value="%s" class="form-control" />
</div>`,
		f.name, html.EscapeString(f.formatLabel()), inputType, f.name, f.name, html.EscapeString(val),
	)
}

// ----------------------------------------------------
// FileUpload
// ----------------------------------------------------

type FileUploadField struct {
	BaseField
	disk      string
	directory string
	maxSize   int64
	accepted  []string
}

func FileUpload(name string) *FileUploadField {
	return &FileUploadField{
		BaseField: BaseField{name: name},
		disk:      "public",
		directory: "uploads",
	}
}

func (f *FileUploadField) Disk(d string) *FileUploadField {
	f.disk = d
	return f
}

func (f *FileUploadField) Directory(dir string) *FileUploadField {
	f.directory = dir
	return f
}

func (f *FileUploadField) AcceptedFileTypes(types ...string) *FileUploadField {
	f.accepted = types
	return f
}

func (f *FileUploadField) Label(l string) *FileUploadField {
	f.BaseField.Label(l)
	return f
}

func (f *FileUploadField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	acceptAttr := ""
	if len(f.accepted) > 0 {
		acceptAttr = fmt.Sprintf(`accept="%s"`, strings.Join(f.accepted, ","))
	}
	return fmt.Sprintf(`
<div class="form-group">
  <label class="form-label" for="%s">%s</label>
  <input type="file" id="%s" name="%s" class="form-control" %s />
</div>`,
		f.name, html.EscapeString(f.formatLabel()), f.name, f.name, acceptAttr,
	)
}

// ----------------------------------------------------
// TagsInput & ColorPicker & Hidden
// ----------------------------------------------------

type TagsInputField struct {
	BaseField
}

func TagsInput(name string) *TagsInputField {
	return &TagsInputField{BaseField: BaseField{name: name}}
}

func (f *TagsInputField) Label(l string) *TagsInputField {
	f.BaseField.Label(l)
	return f
}

func (f *TagsInputField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := ""
	if v, ok := state[f.name]; ok && v != nil {
		val = fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf(`
<div class="form-group">
  <label class="form-label" for="%s">%s</label>
  <input type="text" id="%s" name="%s" value="%s" placeholder="Add tags separated by comma..." class="form-control" />
</div>`,
		f.name, html.EscapeString(f.formatLabel()), f.name, f.name, html.EscapeString(val),
	)
}

type ColorPickerField struct {
	BaseField
}

func ColorPicker(name string) *ColorPickerField {
	return &ColorPickerField{BaseField: BaseField{name: name}}
}

func (f *ColorPickerField) Label(l string) *ColorPickerField {
	f.BaseField.Label(l)
	return f
}

func (f *ColorPickerField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := "#10b981"
	if v, ok := state[f.name]; ok && v != nil && v != "" {
		val = fmt.Sprintf("%v", v)
	}
	return fmt.Sprintf(`
<div class="form-group">
  <label class="form-label" for="%s">%s</label>
  <div style="display: flex; gap: 0.5rem; align-items: center;">
    <input type="color" id="%s" name="%s" value="%s" style="width: 3rem; height: 2.25rem; border-radius: 0.375rem; border: 1px solid var(--gray-300); cursor: pointer;" />
    <span style="font-family: monospace; font-size: 0.875rem;">%s</span>
  </div>
</div>`,
		f.name, html.EscapeString(f.formatLabel()), f.name, f.name, val, val,
	)
}

type HiddenField struct {
	BaseField
}

func Hidden(name string) *HiddenField {
	return &HiddenField{BaseField: BaseField{name: name}}
}

func (f *HiddenField) Default(v any) *HiddenField {
	f.BaseField.Default(v)
	return f
}

func (f *HiddenField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := ""
	if v, ok := state[f.name]; ok && v != nil {
		val = fmt.Sprintf("%v", v)
	} else if f.defaultValue != nil {
		val = fmt.Sprintf("%v", f.defaultValue)
	}
	return fmt.Sprintf(`<input type="hidden" id="%s" name="%s" value="%s" />`, f.name, f.name, html.EscapeString(val))
}
