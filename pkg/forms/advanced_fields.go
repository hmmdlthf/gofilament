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

type RichEditorField struct {
	BaseField
	toolbarButtons []string
	minHeight      string
}

func RichEditor(name string) *RichEditorField {
	return &RichEditorField{
		BaseField: BaseField{name: name},
		minHeight: "220px",
	}
}

func (f *RichEditorField) ToolbarButtons(buttons ...string) *RichEditorField {
	f.toolbarButtons = buttons
	return f
}

func (f *RichEditorField) MinHeight(h string) *RichEditorField {
	f.minHeight = h
	return f
}

func (f *RichEditorField) Label(l string) *RichEditorField {
	f.BaseField.Label(l)
	return f
}

func (f *RichEditorField) Placeholder(p string) *RichEditorField {
	f.BaseField.Placeholder(p)
	return f
}

func (f *RichEditorField) HelperText(h string) *RichEditorField {
	f.BaseField.HelperText(h)
	return f
}

func (f *RichEditorField) Required(req ...bool) *RichEditorField {
	f.BaseField.Required(req...)
	return f
}

func (f *RichEditorField) Disabled(d ...bool) *RichEditorField {
	f.BaseField.Disabled(d...)
	return f
}

func (f *RichEditorField) ColumnSpan(span int) *RichEditorField {
	f.BaseField.ColumnSpan(span)
	return f
}

func (f *RichEditorField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := ""
	if v, ok := state[f.name]; ok && v != nil {
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

	helperHTML := ""
	if f.helperText != "" {
		helperHTML = fmt.Sprintf(`<span style="display: block; font-size: 0.75rem; color: var(--gray-500); margin-top: 0.25rem;">%s</span>`, html.EscapeString(f.helperText))
	}

	editorID := fmt.Sprintf("rich_editor_%s", f.name)
	inputID := fmt.Sprintf("rich_input_%s", f.name)

	return fmt.Sprintf(`
<div class="form-group rich-editor-wrapper" x-data="{
  exec(cmd, val) {
    document.execCommand(cmd, false, val || null);
    this.sync();
  },
  formatBlock(tag) {
    document.execCommand('formatBlock', false, tag);
    this.sync();
  },
  createLink() {
    const url = prompt('Enter hyperlink URL:');
    if (url) {
      document.execCommand('createLink', false, url);
      this.sync();
    }
  },
  sync() {
    const el = document.getElementById('%s');
    const input = document.getElementById('%s');
    if (el && input) {
      input.value = el.innerHTML;
    }
  }
}">
  <label class="form-label" for="%s">%s%s</label>
  <div style="border: 1px solid var(--gray-300); border-radius: 0.5rem; overflow: hidden; background: white; box-shadow: 0 1px 2px rgba(0,0,0,0.05); transition: border-color 0.2s;"
       onfocusin="this.style.borderColor='var(--primary-600)'" onfocusout="this.style.borderColor='var(--gray-300)'">
    <!-- Toolbar -->
    <div style="background-color: var(--gray-50); padding: 0.375rem 0.5rem; border-bottom: 1px solid var(--gray-200); display: flex; flex-wrap: wrap; gap: 0.25rem; align-items: center; user-select: none;">
      <button type="button" class="btn-toolbar" @click="exec('bold')" title="Bold" style="font-weight: 700; width: 1.75rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">B</button>
      <button type="button" class="btn-toolbar" @click="exec('italic')" title="Italic" style="font-style: italic; width: 1.75rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">I</button>
      <button type="button" class="btn-toolbar" @click="exec('underline')" title="Underline" style="text-decoration: underline; width: 1.75rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">U</button>
      <button type="button" class="btn-toolbar" @click="exec('strikeThrough')" title="Strikethrough" style="text-decoration: line-through; width: 1.75rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">S</button>
      <span style="width: 1px; height: 1.25rem; background: var(--gray-300); margin: 0 0.25rem;"></span>
      <button type="button" class="btn-toolbar" @click="formatBlock('h1')" title="Heading 1" style="font-size: 0.75rem; font-weight: 700; padding: 0 0.375rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">H1</button>
      <button type="button" class="btn-toolbar" @click="formatBlock('h2')" title="Heading 2" style="font-size: 0.75rem; font-weight: 700; padding: 0 0.375rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">H2</button>
      <button type="button" class="btn-toolbar" @click="formatBlock('h3')" title="Heading 3" style="font-size: 0.75rem; font-weight: 700; padding: 0 0.375rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">H3</button>
      <button type="button" class="btn-toolbar" @click="formatBlock('p')" title="Paragraph" style="font-size: 0.75rem; padding: 0 0.375rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">¶</button>
      <span style="width: 1px; height: 1.25rem; background: var(--gray-300); margin: 0 0.25rem;"></span>
      <button type="button" class="btn-toolbar" @click="exec('insertUnorderedList')" title="Bullet List" style="font-size: 0.875rem; width: 1.75rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">&bull;&equiv;</button>
      <button type="button" class="btn-toolbar" @click="exec('insertOrderedList')" title="Numbered List" style="font-size: 0.75rem; font-weight: 600; width: 1.75rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">1.&equiv;</button>
      <button type="button" class="btn-toolbar" @click="formatBlock('blockquote')" title="Quote" style="font-size: 0.875rem; width: 1.75rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">&ldquo;</button>
      <button type="button" class="btn-toolbar" @click="formatBlock('pre')" title="Code Block" style="font-family: monospace; font-size: 0.75rem; padding: 0 0.25rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">&lt;/&gt;</button>
      <span style="width: 1px; height: 1.25rem; background: var(--gray-300); margin: 0 0.25rem;"></span>
      <button type="button" class="btn-toolbar" @click="createLink()" title="Insert Link" style="width: 1.75rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">&#128279;</button>
      <button type="button" class="btn-toolbar" @click="exec('removeFormat')" title="Clear Formatting" style="font-size: 0.75rem; padding: 0 0.25rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">&cross;</button>
      <span style="width: 1px; height: 1.25rem; background: var(--gray-300); margin: 0 0.25rem;"></span>
      <button type="button" class="btn-toolbar" @click="exec('undo')" title="Undo" style="font-size: 0.875rem; width: 1.75rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">&#8630;</button>
      <button type="button" class="btn-toolbar" @click="exec('redo')" title="Redo" style="font-size: 0.875rem; width: 1.75rem; height: 1.75rem; border: 1px solid transparent; border-radius: 0.25rem; background: none; cursor: pointer;">&#8631;</button>
    </div>
    <!-- Content Editable Area -->
    <div id="%s" class="rich-editor-content" contenteditable="true"
         @input="sync()" @blur="sync()"
         style="min-height: %s; padding: 0.875rem 1rem; outline: none; font-size: 0.875rem; line-height: 1.6; color: var(--gray-800);">%s</div>
  </div>
  <input type="hidden" id="%s" name="%s" value="%s" />
  %s
  %s
</div>`,
		editorID, inputID,
		editorID, html.EscapeString(f.formatLabel()), requiredBadge,
		editorID, f.minHeight, val,
		inputID, f.name, html.EscapeString(val),
		helperHTML,
		errHTML,
	)
}

type MarkdownEditorField struct {
	BaseField
	rows           int
	previewable    bool
	toolbarButtons []string
}

func MarkdownEditor(name string) *MarkdownEditorField {
	return &MarkdownEditorField{
		BaseField:   BaseField{name: name},
		rows:        8,
		previewable: true,
	}
}

func (f *MarkdownEditorField) Label(l string) *MarkdownEditorField {
	f.BaseField.Label(l)
	return f
}

func (f *MarkdownEditorField) Placeholder(p string) *MarkdownEditorField {
	f.BaseField.Placeholder(p)
	return f
}

func (f *MarkdownEditorField) HelperText(h string) *MarkdownEditorField {
	f.BaseField.HelperText(h)
	return f
}

func (f *MarkdownEditorField) Rows(r int) *MarkdownEditorField {
	f.rows = r
	return f
}

func (f *MarkdownEditorField) Previewable(p ...bool) *MarkdownEditorField {
	if len(p) > 0 {
		f.previewable = p[0]
	} else {
		f.previewable = true
	}
	return f
}

func (f *MarkdownEditorField) ToolbarButtons(buttons ...string) *MarkdownEditorField {
	f.toolbarButtons = buttons
	return f
}

func (f *MarkdownEditorField) Required(req ...bool) *MarkdownEditorField {
	f.BaseField.Required(req...)
	return f
}

func (f *MarkdownEditorField) Disabled(d ...bool) *MarkdownEditorField {
	f.BaseField.Disabled(d...)
	return f
}

func (f *MarkdownEditorField) ColumnSpan(span int) *MarkdownEditorField {
	f.BaseField.ColumnSpan(span)
	return f
}

func (f *MarkdownEditorField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := ""
	if v, ok := state[f.name]; ok && v != nil {
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

	helperHTML := ""
	if f.helperText != "" {
		helperHTML = fmt.Sprintf(`<span style="display: block; font-size: 0.75rem; color: var(--gray-500); margin-top: 0.25rem;">%s</span>`, html.EscapeString(f.helperText))
	}

	textareaID := fmt.Sprintf("markdown_ta_%s", f.name)

	return fmt.Sprintf(`
<div class="form-group markdown-editor-wrapper" x-data="{
  tab: 'write',
  content: '%s',
  insert(prefix, suffix) {
    const ta = document.getElementById('%s');
    if (!ta) return;
    const start = ta.selectionStart;
    const end = ta.selectionEnd;
    const text = ta.value;
    const selected = text.substring(start, end) || 'text';
    const replacement = prefix + selected + suffix;
    this.content = text.substring(0, start) + replacement + text.substring(end);
    this.$nextTick(() => {
      ta.focus();
      ta.setSelectionRange(start + prefix.length, start + prefix.length + selected.length);
    });
  },
  renderMarkdown(src) {
    if (!src) return '<p style=\"color: var(--gray-400); font-style: italic;\">Nothing to preview</p>';
    let h = src
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      .replace(/^### (.*$)/gim, '<h3 style=\"font-size: 1.125rem; font-weight: 700; margin: 0.5rem 0;\">$1</h3>')
      .replace(/^## (.*$)/gim, '<h2 style=\"font-size: 1.25rem; font-weight: 700; margin: 0.75rem 0;\">$1</h2>')
      .replace(/^# (.*$)/gim, '<h1 style=\"font-size: 1.5rem; font-weight: 700; margin: 1rem 0;\">$1</h1>')
      .replace(/\*\*(.*?)\*\*/gim, '<strong>$1</strong>')
      .replace(/\*(.*?)\*/gim, '<em>$1</em>')
      .replace(/~~(.*?)~~/gim, '<del>$1</del>')
      .replace(/\x60([^\x60]+)\x60/gim, '<code style=\"background: var(--gray-100); padding: 0.125rem 0.25rem; border-radius: 0.25rem; font-family: monospace;\">$1</code>')
      .replace(/^\s*\-\s+(.*$)/gim, '<li style=\"margin-left: 1.25rem;\">$1</li>')
      .replace(/\[([^\]]+)\]\(([^)]+)\)/gim, '<a href=\"$2\" target=\"_blank\" style=\"color: var(--primary-600); text-decoration: underline;\">$1</a>')
      .replace(/\n\n+/g, '</p><p style=\"margin: 0.5rem 0;\">')
      .replace(/\n/g, '<br/>');
    return '<p style=\"margin: 0.5rem 0;\">' + h + '</p>';
  }
}">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.375rem;">
    <label class="form-label" for="%s" style="margin-bottom: 0;">%s%s</label>
    <div style="display: flex; gap: 0.25rem; background: var(--gray-100); padding: 0.125rem; border-radius: 0.375rem;">
      <button type="button" @click="tab = 'write'" :style="tab === 'write' ? 'background: white; font-weight: 600; box-shadow: 0 1px 2px rgba(0,0,0,0.05);' : 'background: transparent; color: var(--gray-600);'"
              style="border: none; border-radius: 0.25rem; padding: 0.125rem 0.5rem; font-size: 0.75rem; cursor: pointer;">Write</button>
      <button type="button" @click="tab = 'preview'" :style="tab === 'preview' ? 'background: white; font-weight: 600; box-shadow: 0 1px 2px rgba(0,0,0,0.05);' : 'background: transparent; color: var(--gray-600);'"
              style="border: none; border-radius: 0.25rem; padding: 0.125rem 0.5rem; font-size: 0.75rem; cursor: pointer;">Preview</button>
    </div>
  </div>

  <div style="border: 1px solid var(--gray-300); border-radius: 0.5rem; overflow: hidden; background: white;">
    <!-- Write mode toolbar -->
    <div x-show="tab === 'write'" style="background-color: var(--gray-50); padding: 0.375rem 0.75rem; border-bottom: 1px solid var(--gray-200); display: flex; flex-wrap: wrap; gap: 0.375rem; font-size: 0.8125rem;">
      <button type="button" @click="insert('**', '**')" title="Bold" style="font-weight: 700; background: none; border: none; cursor: pointer; padding: 0.125rem 0.375rem;">B</button>
      <button type="button" @click="insert('*', '*')" title="Italic" style="font-style: italic; background: none; border: none; cursor: pointer; padding: 0.125rem 0.375rem;">I</button>
      <button type="button" @click="insert('~~', '~~')" title="Strikethrough" style="text-decoration: line-through; background: none; border: none; cursor: pointer; padding: 0.125rem 0.375rem;">S</button>
      <button type="button" @click="insert('### ', '')" title="Heading" style="font-weight: 600; background: none; border: none; cursor: pointer; padding: 0.125rem 0.375rem;">H</button>
      <button type="button" @click="insert(String.fromCharCode(96), String.fromCharCode(96))" title="Inline Code" style="font-family: monospace; background: none; border: none; cursor: pointer; padding: 0.125rem 0.375rem;">&lt;/&gt;</button>
      <button type="button" @click="insert('[', '](https://)')" title="Link" style="background: none; border: none; cursor: pointer; padding: 0.125rem 0.375rem;">🔗</button>
      <button type="button" @click="insert('> ', '')" title="Blockquote" style="background: none; border: none; cursor: pointer; padding: 0.125rem 0.375rem;">&ldquo;</button>
      <button type="button" @click="insert('- ', '')" title="List" style="background: none; border: none; cursor: pointer; padding: 0.125rem 0.375rem;">• List</button>
    </div>

    <!-- Write Textarea -->
    <div x-show="tab === 'write'">
      <textarea id="%s" name="%s" rows="%d" x-model="content" class="form-control"
                placeholder="%s"
                style="border: none; border-radius: 0; box-shadow: none; font-family: monospace; font-size: 0.875rem; line-height: 1.5; resize: vertical;">%s</textarea>
    </div>

    <!-- Preview Tab -->
    <div x-show="tab === 'preview'" style="padding: 1rem; min-height: %drem; font-size: 0.875rem; line-height: 1.6; color: var(--gray-800);"
         x-html="renderMarkdown(content)">
    </div>
  </div>
  %s
  %s
</div>`,
		html.EscapeString(val),
		textareaID,
		textareaID, html.EscapeString(f.formatLabel()), requiredBadge,
		textareaID, f.name, f.rows,
		html.EscapeString(f.placeholder),
		html.EscapeString(val),
		f.rows*2,
		helperHTML,
		errHTML,
	)
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
