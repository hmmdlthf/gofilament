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
// FileUpload & ImageUpload
// ----------------------------------------------------

type FileUploadField struct {
	BaseField
	disk              string
	directory         string
	maxSize           int64 // in KB
	accepted          []string
	isImage           bool
	isAvatar          bool
	multiple          bool
	maxFiles          int
	imagePreview      bool
	preserveFilenames bool
}

func FileUpload(name string) *FileUploadField {
	return &FileUploadField{
		BaseField:    BaseField{name: name},
		disk:         "public",
		directory:    "uploads",
		imagePreview: true,
	}
}

func ImageUpload(name string) *FileUploadField {
	return &FileUploadField{
		BaseField:    BaseField{name: name},
		disk:         "public",
		directory:    "images",
		isImage:      true,
		imagePreview: true,
		accepted:     []string{"image/png", "image/jpeg", "image/webp", "image/gif", "image/svg+xml"},
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

func (f *FileUploadField) Image(enabled ...bool) *FileUploadField {
	if len(enabled) > 0 {
		f.isImage = enabled[0]
	} else {
		f.isImage = true
	}
	if f.isImage && len(f.accepted) == 0 {
		f.accepted = []string{"image/png", "image/jpeg", "image/webp", "image/gif", "image/svg+xml"}
	}
	return f
}

func (f *FileUploadField) Avatar(enabled ...bool) *FileUploadField {
	if len(enabled) > 0 {
		f.isAvatar = enabled[0]
	} else {
		f.isAvatar = true
	}
	if f.isAvatar {
		f.isImage = true
		if len(f.accepted) == 0 {
			f.accepted = []string{"image/png", "image/jpeg", "image/webp", "image/gif"}
		}
	}
	return f
}

func (f *FileUploadField) Multiple(enabled ...bool) *FileUploadField {
	if len(enabled) > 0 {
		f.multiple = enabled[0]
	} else {
		f.multiple = true
	}
	return f
}

func (f *FileUploadField) MaxFiles(n int) *FileUploadField {
	f.maxFiles = n
	return f
}

func (f *FileUploadField) MaxSize(kb int64) *FileUploadField {
	f.maxSize = kb
	return f
}

func (f *FileUploadField) ImagePreview(enabled ...bool) *FileUploadField {
	if len(enabled) > 0 {
		f.imagePreview = enabled[0]
	} else {
		f.imagePreview = true
	}
	return f
}

func (f *FileUploadField) PreserveFilenames(enabled ...bool) *FileUploadField {
	if len(enabled) > 0 {
		f.preserveFilenames = enabled[0]
	} else {
		f.preserveFilenames = true
	}
	return f
}

func (f *FileUploadField) Label(l string) *FileUploadField {
	f.BaseField.Label(l)
	return f
}

func (f *FileUploadField) Placeholder(p string) *FileUploadField {
	f.BaseField.Placeholder(p)
	return f
}

func (f *FileUploadField) HelperText(h string) *FileUploadField {
	f.BaseField.HelperText(h)
	return f
}

func (f *FileUploadField) Required(req ...bool) *FileUploadField {
	f.BaseField.Required(req...)
	return f
}

func (f *FileUploadField) Disabled(d ...bool) *FileUploadField {
	f.BaseField.Disabled(d...)
	return f
}

func (f *FileUploadField) ColumnSpan(span int) *FileUploadField {
	f.BaseField.ColumnSpan(span)
	return f
}

func (f *FileUploadField) IsImage() bool {
	return f.isImage
}

func (f *FileUploadField) IsAvatar() bool {
	return f.isAvatar
}

func (f *FileUploadField) IsMultiple() bool {
	return f.multiple
}

func (f *FileUploadField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	acceptAttr := ""
	if len(f.accepted) > 0 {
		acceptAttr = fmt.Sprintf(`accept="%s"`, strings.Join(f.accepted, ","))
	}

	multipleAttr := ""
	nameAttr := f.name
	if f.multiple {
		multipleAttr = "multiple"
		if !strings.HasSuffix(nameAttr, "[]") {
			nameAttr = f.name + "[]"
		}
	}

	existingVal := ""
	if v, ok := state[f.name]; ok && v != nil {
		existingVal = fmt.Sprintf("%v", v)
	}

	requiredBadge := ""
	if f.required {
		requiredBadge = ` <span style="color: #ef4444;">*</span>`
	}

	errHTML := ""
	if err, hasErr := errors[f.name]; hasErr && err != "" {
		errHTML = fmt.Sprintf(`<span style="display: block; font-size: 0.75rem; color: #ef4444; margin-top: 0.25rem;">%s</span>`, html.EscapeString(err))
	}

	helperHTML := ""
	if f.helperText != "" {
		helperHTML = fmt.Sprintf(`<span style="display: block; font-size: 0.75rem; color: var(--gray-500); margin-top: 0.25rem;">%s</span>`, html.EscapeString(f.helperText))
	}

	// Avatar circular mode
	if f.isAvatar {
		avatarSrc := existingVal
		hasAvatar := avatarSrc != ""
		imgDisplay := "none"
		if hasAvatar {
			imgDisplay = "block"
		}
		placeholderDisplay := "block"
		if hasAvatar {
			placeholderDisplay = "none"
		}

		return fmt.Sprintf(`
<div class="form-group file-upload-avatar-wrapper" x-data="{ preview: '%s' }">
  <label class="form-label">%s%s</label>
  <div style="display: flex; align-items: center; gap: 1.25rem; margin-top: 0.5rem;">
    <div style="position: relative; width: 5.5rem; height: 5.5rem; border-radius: 9999px; overflow: hidden; border: 2px dashed var(--gray-300); background: var(--gray-100); display: flex; align-items: center; justify-content: center; cursor: pointer;"
         onclick="document.getElementById('%s_input').click()">
      <img :src="preview" x-show="preview" src="%s" alt="Avatar" style="width: 100%%; height: 100%%; object-fit: cover; display: %s;" />
      <div x-show="!preview" style="display: %s; color: var(--gray-400); text-align: center;">
        <svg style="width: 2rem; height: 2rem; margin: 0 auto;" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
        </svg>
      </div>
      <div style="position: absolute; inset: 0; background: rgba(0,0,0,0.35); opacity: 0; hover:opacity: 1; transition: opacity 0.2s; display: flex; align-items: center; justify-content: center; color: white;"
           onmouseover="this.style.opacity='1'" onmouseout="this.style.opacity='0'">
        <svg style="width: 1.25rem; height: 1.25rem;" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 9a2 2 0 012-2h.93a2 2 0 001.664-.89l.812-1.22A2 2 0 0110.07 4h3.86a2 2 0 011.664.89l.812 1.22A2 2 0 0018.07 7H19a2 2 0 012 2v9a2 2 0 01-2 2H5a2 2 0 01-2-2V9z" />
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 13a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      </div>
    </div>
    <div>
      <input type="file" id="%s_input" name="%s" %s class="sr-only" style="display: none;"
             @change="const file = $event.target.files[0]; if (file) { preview = URL.createObjectURL(file); }" />
      <input type="hidden" name="%s_existing" value="%s" />
      <button type="button" class="btn btn-secondary btn-sm" onclick="document.getElementById('%s_input').click()">Choose Avatar</button>
      <button type="button" class="btn btn-outline-danger btn-sm" x-show="preview" style="margin-left: 0.5rem;"
              @click="preview = ''; document.getElementById('%s_input').value = ''; document.querySelector('input[name=%s_existing]').value = '';">Remove</button>
      %s
    </div>
  </div>
  %s
</div>`,
			html.EscapeString(existingVal),
			html.EscapeString(f.formatLabel()), requiredBadge,
			f.name, html.EscapeString(avatarSrc), imgDisplay, placeholderDisplay,
			f.name, nameAttr, acceptAttr,
			f.name, html.EscapeString(existingVal),
			f.name,
			f.name, f.name,
			helperHTML,
			errHTML,
		)
	}

	// Standard File / Image Dropzone
	typeHint := "Any file"
	if f.isImage {
		typeHint = "PNG, JPG, WEBP, GIF, SVG"
	} else if len(f.accepted) > 0 {
		typeHint = strings.ToUpper(strings.Join(f.accepted, ", "))
	}
	if f.maxSize > 0 {
		typeHint += fmt.Sprintf(" (up to %d KB)", f.maxSize)
	}

	return fmt.Sprintf(`
<div class="form-group file-upload-wrapper" x-data="{ isDragging: false, preview: '%s', fileName: '%s' }">
  <label class="form-label" for="%s_input">%s%s</label>
  <div class="file-upload-dropzone"
       :style="isDragging ? 'border-color: var(--primary-600); background-color: var(--primary-50, rgba(99, 102, 241, 0.05));' : 'border-color: var(--gray-300); background-color: var(--gray-50);'"
       style="border: 2px dashed var(--gray-300); border-radius: 0.75rem; padding: 1.5rem 1rem; text-align: center; cursor: pointer; transition: all 0.2s; position: relative;"
       @dragover.prevent="isDragging = true"
       @dragleave.prevent="isDragging = false"
       @drop.prevent="isDragging = false; const files = $event.dataTransfer.files; if (files.length) { document.getElementById('%s_input').files = files; const f = files[0]; fileName = f.name; if (f.type.startsWith('image/')) { preview = URL.createObjectURL(f); } else { preview = ''; } }"
       onclick="document.getElementById('%s_input').click()">
    <input type="file" id="%s_input" name="%s" %s %s style="display: none;"
           @change="const f = $event.target.files[0]; if (f) { fileName = f.name; if (f.type.startsWith('image/')) { preview = URL.createObjectURL(f); } else { preview = ''; } }" />
    <input type="hidden" name="%s_existing" value="%s" />

    <!-- Default Dropzone Prompt -->
    <div x-show="!preview && !fileName" style="display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 0.5rem;">
      <svg style="width: 2.25rem; height: 2.25rem; color: var(--gray-400);" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
      </svg>
      <div>
        <span style="font-size: 0.875rem; font-weight: 600; color: var(--primary-600);">Click to upload</span>
        <span style="font-size: 0.875rem; color: var(--gray-500);"> or drag and drop</span>
      </div>
      <span style="font-size: 0.75rem; color: var(--gray-400);">%s</span>
    </div>

    <!-- Active Preview Card -->
    <div x-show="preview || fileName" style="display: flex; align-items: center; justify-content: center; gap: 1rem;" @click.stop>
      <template x-if="preview">
        <img :src="preview" alt="Preview" style="max-height: 5rem; border-radius: 0.375rem; box-shadow: 0 1px 3px rgba(0,0,0,0.1); object-fit: contain;" />
      </template>
      <div style="text-align: left;">
        <span x-text="fileName || preview" style="font-size: 0.875rem; font-weight: 500; color: var(--gray-800); display: block; max-width: 15rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;"></span>
        <button type="button" class="btn btn-outline-danger btn-sm" style="margin-top: 0.375rem; font-size: 0.75rem; padding: 0.125rem 0.5rem;"
                @click="preview = ''; fileName = ''; document.getElementById('%s_input').value = ''; document.querySelector('input[name=%s_existing]').value = '';">Remove</button>
      </div>
    </div>
  </div>
  %s
  %s
</div>`,
		html.EscapeString(existingVal), html.EscapeString(existingVal),
		f.name, html.EscapeString(f.formatLabel()), requiredBadge,
		f.name,
		f.name,
		f.name, nameAttr, acceptAttr, multipleAttr,
		f.name, html.EscapeString(existingVal),
		html.EscapeString(typeHint),
		f.name, f.name,
		helperHTML,
		errHTML,
	)
}

// ----------------------------------------------------
// TagsInput & ColorPicker & Hidden
// ----------------------------------------------------

type TagsInputField struct {
	BaseField
	suggestions []string
	maxTags     int
	separator   string
}

func TagsInput(name string) *TagsInputField {
	return &TagsInputField{
		BaseField: BaseField{name: name},
		separator: ",",
	}
}

func (f *TagsInputField) Suggestions(suggestions ...string) *TagsInputField {
	f.suggestions = suggestions
	return f
}

func (f *TagsInputField) MaxTags(max int) *TagsInputField {
	f.maxTags = max
	return f
}

func (f *TagsInputField) Separator(sep string) *TagsInputField {
	f.separator = sep
	return f
}

func (f *TagsInputField) Label(l string) *TagsInputField {
	f.BaseField.Label(l)
	return f
}

func (f *TagsInputField) Placeholder(p string) *TagsInputField {
	f.BaseField.Placeholder(p)
	return f
}

func (f *TagsInputField) HelperText(h string) *TagsInputField {
	f.BaseField.HelperText(h)
	return f
}

func (f *TagsInputField) Required(req ...bool) *TagsInputField {
	f.BaseField.Required(req...)
	return f
}

func (f *TagsInputField) Disabled(d ...bool) *TagsInputField {
	f.BaseField.Disabled(d...)
	return f
}

func (f *TagsInputField) ColumnSpan(span int) *TagsInputField {
	f.BaseField.ColumnSpan(span)
	return f
}

func (f *TagsInputField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	initialTags := make([]string, 0)
	if val, ok := state[f.name]; ok && val != nil {
		if slice, ok := val.([]string); ok {
			initialTags = slice
		} else if str, ok := val.(string); ok && str != "" {
			parts := strings.Split(str, f.separator)
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					initialTags = append(initialTags, p)
				}
			}
		}
	}

	tagsJSONParts := make([]string, len(initialTags))
	for i, t := range initialTags {
		tagsJSONParts[i] = fmt.Sprintf(`"%s"`, html.EscapeString(t))
	}
	tagsJSON := fmt.Sprintf("[%s]", strings.Join(tagsJSONParts, ","))

	suggJSONParts := make([]string, len(f.suggestions))
	for i, s := range f.suggestions {
		suggJSONParts[i] = fmt.Sprintf(`"%s"`, html.EscapeString(s))
	}
	suggJSON := fmt.Sprintf("[%s]", strings.Join(suggJSONParts, ","))

	placeholder := f.placeholder
	if placeholder == "" {
		placeholder = "Type tag and press Enter..."
	}

	requiredBadge := ""
	if f.required {
		requiredBadge = ` <span style="color: #ef4444;">*</span>`
	}

	errHTML := ""
	if err, hasErr := errors[f.name]; hasErr && err != "" {
		errHTML = fmt.Sprintf(`<span style="display: block; font-size: 0.75rem; color: #ef4444; margin-top: 0.25rem;">%s</span>`, html.EscapeString(err))
	}

	return fmt.Sprintf(`
<div class="form-group tags-input-wrapper" x-data="{
  tags: %s,
  input: '',
  suggestions: %s,
  max: %d,
  sep: '%s',
  addTag(val) {
    val = (val || this.input).trim();
    if (!val) return;
    if (this.max > 0 && this.tags.length >= this.max) return;
    if (!this.tags.includes(val)) {
      this.tags.push(val);
    }
    this.input = '';
  },
  removeTag(idx) {
    this.tags.splice(idx, 1);
  }
}">
  <label class="form-label" for="%s">%s%s</label>
  <div style="display: flex; flex-wrap: wrap; align-items: center; gap: 0.375rem; padding: 0.375rem 0.5rem; border: 1px solid var(--gray-300); border-radius: 0.5rem; background: white; min-height: 2.5rem; focus-within: border-color: var(--primary-600);">
    <template x-for="(tag, idx) in tags" :key="idx">
      <span style="display: inline-flex; align-items: center; gap: 0.25rem; padding: 0.125rem 0.5rem; background: var(--primary-50, #eef2ff); color: var(--primary-700, #4338ca); border: 1px solid var(--primary-200, #c7d2fe); border-radius: 9999px; font-size: 0.75rem; font-weight: 500;">
        <span x-text="tag"></span>
        <button type="button" @click="removeTag(idx)" style="background: none; border: none; padding: 0; cursor: pointer; color: var(--primary-500); line-height: 1;">&times;</button>
      </span>
    </template>
    <input type="text" id="%s" x-model="input" placeholder="%s"
           @keydown.enter.prevent="addTag()"
           @keydown.comma.prevent="addTag()"
           @keydown.backspace="if (input === '' && tags.length > 0) removeTag(tags.length - 1)"
           style="border: none; outline: none; flex: 1; min-width: 8rem; font-size: 0.875rem; padding: 0.25rem; background: transparent;" />
  </div>
  <input type="hidden" name="%s" :value="tags.join(sep)" />
  <template x-if="suggestions.length > 0">
    <div style="display: flex; flex-wrap: wrap; gap: 0.375rem; margin-top: 0.375rem;">
      <span style="font-size: 0.75rem; color: var(--gray-500); margin-right: 0.25rem;">Suggested:</span>
      <template x-for="sugg in suggestions" :key="sugg">
        <button type="button" @click="addTag(sugg)" x-show="!tags.includes(sugg)"
                style="border: 1px dashed var(--gray-300); background: var(--gray-50); border-radius: 9999px; font-size: 0.6875rem; padding: 0.0625rem 0.375rem; cursor: pointer; color: var(--gray-600);"
                x-text="'+ ' + sugg"></button>
      </template>
    </div>
  </template>
  %s
</div>`,
		tagsJSON, suggJSON, f.maxTags, f.separator,
		f.name, html.EscapeString(f.formatLabel()), requiredBadge,
		f.name, html.EscapeString(placeholder),
		f.name,
		errHTML,
	)
}

type ColorPickerField struct {
	BaseField
	presets []string
}

func ColorPicker(name string) *ColorPickerField {
	return &ColorPickerField{
		BaseField: BaseField{name: name},
		presets:   []string{"#6366f1", "#10b981", "#f59e0b", "#ef4444", "#3b82f6", "#8b5cf6", "#ec4899", "#64748b"},
	}
}

func (f *ColorPickerField) Presets(presets ...string) *ColorPickerField {
	f.presets = presets
	return f
}

func (f *ColorPickerField) Label(l string) *ColorPickerField {
	f.BaseField.Label(l)
	return f
}

func (f *ColorPickerField) Placeholder(p string) *ColorPickerField {
	f.BaseField.Placeholder(p)
	return f
}

func (f *ColorPickerField) HelperText(h string) *ColorPickerField {
	f.BaseField.HelperText(h)
	return f
}

func (f *ColorPickerField) Required(req ...bool) *ColorPickerField {
	f.BaseField.Required(req...)
	return f
}

func (f *ColorPickerField) Disabled(d ...bool) *ColorPickerField {
	f.BaseField.Disabled(d...)
	return f
}

func (f *ColorPickerField) ColumnSpan(span int) *ColorPickerField {
	f.BaseField.ColumnSpan(span)
	return f
}

func (f *ColorPickerField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := "#6366f1"
	if v, ok := state[f.name]; ok && v != nil && fmt.Sprintf("%v", v) != "" {
		val = fmt.Sprintf("%v", v)
	}

	requiredBadge := ""
	if f.required {
		requiredBadge = ` <span style="color: #ef4444;">*</span>`
	}

	errHTML := ""
	if err, hasErr := errors[f.name]; hasErr && err != "" {
		errHTML = fmt.Sprintf(`<span style="display: block; font-size: 0.75rem; color: #ef4444; margin-top: 0.25rem;">%s</span>`, html.EscapeString(err))
	}

	var presetsHTML strings.Builder
	for _, p := range f.presets {
		presetsHTML.WriteString(fmt.Sprintf(`
<button type="button" @click="color = '%s'"
        style="width: 1.5rem; height: 1.5rem; border-radius: 9999px; background-color: %s; border: 2px solid white; box-shadow: 0 0 0 1px var(--gray-300); cursor: pointer; transition: transform 0.1s;"
        onmouseover="this.style.transform='scale(1.1)'" onmouseout="this.style.transform='scale(1)'"
        title="%s"></button>`, p, p, p))
	}

	return fmt.Sprintf(`
<div class="form-group color-picker-wrapper" x-data="{ color: '%s' }">
  <label class="form-label" for="%s">%s%s</label>
  <div style="display: flex; flex-direction: column; gap: 0.5rem; margin-top: 0.25rem;">
    <div style="display: flex; gap: 0.75rem; align-items: center;">
      <input type="color" id="%s" x-model="color" style="width: 2.75rem; height: 2.25rem; border-radius: 0.375rem; border: 1px solid var(--gray-300); cursor: pointer; padding: 0.125rem;" />
      <input type="text" name="%s" x-model="color" maxlength="7" class="form-control" style="width: 6.5rem; font-family: monospace; font-size: 0.875rem; text-transform: uppercase;" />
      <div style="width: 1.75rem; height: 1.75rem; border-radius: 0.375rem; border: 1px solid var(--gray-300);" :style="'background-color: ' + color"></div>
    </div>
    <div style="display: flex; gap: 0.375rem; align-items: center; margin-top: 0.125rem;">
      %s
    </div>
  </div>
  %s
</div>`,
		val,
		f.name, html.EscapeString(f.formatLabel()), requiredBadge,
		f.name,
		f.name,
		presetsHTML.String(),
		errHTML,
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
