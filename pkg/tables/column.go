package tables

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// BaseColumn defines shared attributes for table columns.
type BaseColumn[T any] struct {
	name                   string
	label                  string
	sortable               bool
	searchable             bool
	weight                 string // bold, medium, normal
	icon                   string
	copyable               bool
	align                  string // left, center, right
	editable               bool
	toggleable             bool
	toggledHiddenByDefault bool
	valueFunc              func(record T) any
}

func (c *BaseColumn[T]) GetName() string {
	return c.name
}

func (c *BaseColumn[T]) Label(l string) *BaseColumn[T] {
	c.label = l
	return c
}

func (c *BaseColumn[T]) Sortable(s ...bool) *BaseColumn[T] {
	c.sortable = len(s) == 0 || s[0]
	return c
}

func (c *BaseColumn[T]) Searchable(s ...bool) *BaseColumn[T] {
	c.searchable = len(s) == 0 || s[0]
	return c
}

func (c *BaseColumn[T]) Weight(w string) *BaseColumn[T] {
	c.weight = w
	return c
}

func (c *BaseColumn[T]) Icon(i string) *BaseColumn[T] {
	c.icon = i
	return c
}

func (c *BaseColumn[T]) Copyable(cp ...bool) *BaseColumn[T] {
	c.copyable = len(cp) == 0 || cp[0]
	return c
}

func (c *BaseColumn[T]) Align(a string) *BaseColumn[T] {
	c.align = a
	return c
}

func (c *BaseColumn[T]) Editable(e ...bool) *BaseColumn[T] {
	c.editable = len(e) == 0 || e[0]
	return c
}

func (c *BaseColumn[T]) Toggleable(t ...bool) *BaseColumn[T] {
	c.toggleable = len(t) == 0 || t[0]
	return c
}

func (c *BaseColumn[T]) ToggledHiddenByDefault(h ...bool) *BaseColumn[T] {
	c.toggledHiddenByDefault = len(h) == 0 || h[0]
	if c.toggledHiddenByDefault {
		c.toggleable = true
	}
	return c
}

func (c *BaseColumn[T]) IsEditable() bool {
	return c.editable
}

func (c *BaseColumn[T]) IsToggleable() bool {
	return c.toggleable
}

func (c *BaseColumn[T]) IsToggledHiddenByDefault() bool {
	return c.toggledHiddenByDefault
}

func (c *BaseColumn[T]) Value(fn func(record T) any) *BaseColumn[T] {
	c.valueFunc = fn
	return c
}

func (c *BaseColumn[T]) FormatLabel() string {
	if c.label != "" {
		return c.label
	}
	words := strings.Split(c.name, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// ----------------------------------------------------
// TextColumn
// ----------------------------------------------------

type TextColumnDef[T any] struct {
	BaseColumn[T]
	isDateTime   bool
	dateTimeFmt  string
	currency     string
	limit        int
}

func TextColumn[T any](name string) *TextColumnDef[T] {
	return &TextColumnDef[T]{
		BaseColumn: BaseColumn[T]{name: name},
	}
}

func (c *TextColumnDef[T]) Label(l string) *TextColumnDef[T] {
	c.BaseColumn.Label(l)
	return c
}

func (c *TextColumnDef[T]) Sortable(s ...bool) *TextColumnDef[T] {
	c.BaseColumn.Sortable(s...)
	return c
}

func (c *TextColumnDef[T]) Searchable(s ...bool) *TextColumnDef[T] {
	c.BaseColumn.Searchable(s...)
	return c
}

func (c *TextColumnDef[T]) Weight(w string) *TextColumnDef[T] {
	c.BaseColumn.Weight(w)
	return c
}

func (c *TextColumnDef[T]) Icon(i string) *TextColumnDef[T] {
	c.BaseColumn.Icon(i)
	return c
}

func (c *TextColumnDef[T]) Copyable(cp ...bool) *TextColumnDef[T] {
	c.BaseColumn.Copyable(cp...)
	return c
}

func (c *TextColumnDef[T]) DateTime(fmt ...string) *TextColumnDef[T] {
	c.isDateTime = true
	if len(fmt) > 0 {
		c.dateTimeFmt = fmt[0]
	} else {
		c.dateTimeFmt = "2006-01-02 15:04"
	}
	return c
}

func (c *TextColumnDef[T]) Money(currency string) *TextColumnDef[T] {
	c.currency = currency
	return c
}

func (c *TextColumnDef[T]) Limit(l int) *TextColumnDef[T] {
	c.limit = l
	return c
}

func (c *TextColumnDef[T]) Value(fn func(record T) any) *TextColumnDef[T] {
	c.BaseColumn.Value(fn)
	return c
}

func (c *TextColumnDef[T]) Editable(e ...bool) *TextColumnDef[T] {
	c.BaseColumn.Editable(e...)
	return c
}

func (c *TextColumnDef[T]) Toggleable(t ...bool) *TextColumnDef[T] {
	c.BaseColumn.Toggleable(t...)
	return c
}

func (c *TextColumnDef[T]) ToggledHiddenByDefault(h ...bool) *TextColumnDef[T] {
	c.BaseColumn.ToggledHiddenByDefault(h...)
	return c
}

func (c *TextColumnDef[T]) RenderCell(record T, rawValue any) string {
	val := ""
	if c.valueFunc != nil {
		rawValue = c.valueFunc(record)
	}

	if t, ok := rawValue.(time.Time); ok && c.isDateTime {
		val = t.Format(c.dateTimeFmt)
	} else if rawValue != nil {
		val = fmt.Sprintf("%v", rawValue)
	}

	if c.currency != "" && val != "" {
		val = fmt.Sprintf("%s %s", c.currency, val)
	}

	if c.limit > 0 && len(val) > c.limit {
		val = val[:c.limit] + "..."
	}

	textWeight := ""
	if c.weight == "bold" {
		textWeight = "font-weight: 700;"
	} else if c.weight == "medium" {
		textWeight = "font-weight: 500;"
	}

	copyAttr := ""
	if c.copyable {
		copyAttr = fmt.Sprintf(`onclick="navigator.clipboard.writeText('%s'); Filament.notify({type:'info', title:'Copied to clipboard'});" style="cursor: pointer;" title="Click to copy"`, html.EscapeString(val))
	}

	return fmt.Sprintf(`<span style="%s" %s>%s</span>`, textWeight, copyAttr, html.EscapeString(val))
}

func (c *TextColumnDef[T]) RenderCellWithContext(record T, recordID, basePath string, rawValue any) string {
	if !c.editable {
		return c.RenderCell(record, rawValue)
	}

	val := ""
	if c.valueFunc != nil {
		rawValue = c.valueFunc(record)
	}
	if rawValue != nil {
		val = fmt.Sprintf("%v", rawValue)
	}

	return fmt.Sprintf(`
<div class="inline-edit-wrapper" style="display: flex; align-items: center; gap: 0.375rem;">
  <span class="inline-edit-text" style="cursor: pointer; border-bottom: 1px dashed var(--gray-400); display: inline-block;"
        title="Click to edit"
        onclick="this.style.display='none'; const inp = this.nextElementSibling; inp.style.display='inline-block'; inp.focus(); inp.select();">%s</span>
  <input type="text" value="%s" class="form-control form-control-sm"
         style="display: none; width: auto; max-width: 14rem; padding: 0.125rem 0.375rem; font-size: 0.8125rem;"
         onblur="this.style.display='none'; this.previousElementSibling.textContent = this.value; this.previousElementSibling.style.display='inline-block'; Filament.updateCell('%s/cell', '%s', '%s', this.value, this);"
         onkeydown="if (event.key === 'Enter') { this.blur(); } if (event.key === 'Escape') { this.value = this.previousElementSibling.textContent; this.style.display='none'; this.previousElementSibling.style.display='inline-block'; }" />
</div>`, html.EscapeString(val), html.EscapeString(val), basePath, recordID, c.name)
}

// ----------------------------------------------------
// BadgeColumn
// ----------------------------------------------------

type BadgeColumnDef[T any] struct {
	BaseColumn[T]
	colors map[string]string // "admin": "danger", "editor": "warning"
}

func BadgeColumn[T any](name string) *BadgeColumnDef[T] {
	return &BadgeColumnDef[T]{
		BaseColumn: BaseColumn[T]{name: name},
		colors:     make(map[string]string),
	}
}

func (c *BadgeColumnDef[T]) Label(l string) *BadgeColumnDef[T] {
	c.BaseColumn.Label(l)
	return c
}

func (c *BadgeColumnDef[T]) Colors(colors map[string]string) *BadgeColumnDef[T] {
	c.colors = colors
	return c
}

func (c *BadgeColumnDef[T]) RenderCell(record T, rawValue any) string {
	val := ""
	if c.valueFunc != nil {
		rawValue = c.valueFunc(record)
	}
	if rawValue != nil {
		val = fmt.Sprintf("%v", rawValue)
	}
	colorName := "gray"
	if col, ok := c.colors[val]; ok {
		colorName = col
	}
	return fmt.Sprintf(`<span class="badge badge-%s">%s</span>`, colorName, html.EscapeString(val))
}

// ----------------------------------------------------
// ImageColumn
// ----------------------------------------------------

type ImageColumnDef[T any] struct {
	BaseColumn[T]
	isCircular bool
	size       string
}

func ImageColumn[T any](name string) *ImageColumnDef[T] {
	return &ImageColumnDef[T]{
		BaseColumn: BaseColumn[T]{name: name},
		size:       "2.25rem",
	}
}

func (c *ImageColumnDef[T]) Label(l string) *ImageColumnDef[T] {
	c.BaseColumn.Label(l)
	return c
}

func (c *ImageColumnDef[T]) Circular(circ ...bool) *ImageColumnDef[T] {
	c.isCircular = len(circ) == 0 || circ[0]
	return c
}

func (c *ImageColumnDef[T]) RenderCell(record T, rawValue any) string {
	url := ""
	if c.valueFunc != nil {
		rawValue = c.valueFunc(record)
	}
	if rawValue != nil {
		url = fmt.Sprintf("%v", rawValue)
	}
	if url == "" {
		return `<div style="width: 2.25rem; height: 2.25rem; border-radius: 9999px; background-color: var(--gray-200);"></div>`
	}
	borderRadius := "0.375rem"
	if c.isCircular {
		borderRadius = "9999px"
	}
	return fmt.Sprintf(`<img src="%s" alt="avatar" style="width: %s; height: %s; border-radius: %s; object-fit: cover;" />`,
		html.EscapeString(url), c.size, c.size, borderRadius)
}

// ----------------------------------------------------
// ToggleColumn
// ----------------------------------------------------

type ToggleColumnDef[T any] struct {
	BaseColumn[T]
}

func ToggleColumn[T any](name string) *ToggleColumnDef[T] {
	return &ToggleColumnDef[T]{
		BaseColumn: BaseColumn[T]{name: name},
	}
}

func (c *ToggleColumnDef[T]) Label(l string) *ToggleColumnDef[T] {
	c.BaseColumn.Label(l)
	return c
}

func (c *ToggleColumnDef[T]) RenderCell(record T, rawValue any) string {
	checked := ""
	if b, ok := rawValue.(bool); ok && b {
		checked = "checked"
	}
	return fmt.Sprintf(`<input type="checkbox" %s disabled style="width: 1.125rem; height: 1.125rem; accent-color: var(--primary-600);" />`, checked)
}

func (c *ToggleColumnDef[T]) RenderCellWithContext(record T, recordID, basePath string, rawValue any) string {
	checked := false
	if b, ok := rawValue.(bool); ok {
		checked = b
	} else if s, ok := rawValue.(string); ok {
		checked = s == "true" || s == "1"
	} else if i, ok := rawValue.(int); ok {
		checked = i == 1
	}

	bgColor := "var(--gray-300)"
	transform := "translateX(0.125rem)"
	checkedAttr := "false"
	if checked {
		bgColor = "var(--primary-600, #4f46e5)"
		transform = "translateX(1.125rem)"
		checkedAttr = "true"
	}

	return fmt.Sprintf(`
<button type="button" role="switch" aria-checked="%s" class="filament-toggle-switch"
        style="position: relative; display: inline-flex; height: 1.375rem; width: 2.375rem; flex-shrink: 0; cursor: pointer; border-radius: 9999px; border: 2px solid transparent; transition: background-color 0.2s; background-color: %s;"
        onclick="const cur = this.getAttribute('aria-checked') === 'true'; const next = !cur; this.setAttribute('aria-checked', next ? 'true' : 'false'); this.style.backgroundColor = next ? 'var(--primary-600, #4f46e5)' : 'var(--gray-300)'; this.firstElementChild.style.transform = next ? 'translateX(1.125rem)' : 'translateX(0.125rem)'; Filament.updateCell('%s/cell', '%s', '%s', next, this);">
  <span style="pointer-events: none; display: inline-block; height: 1.125rem; width: 1.125rem; transform: %s; border-radius: 9999px; background-color: white; box-shadow: 0 1px 2px rgba(0,0,0,0.2); transition: transform 0.2s;"></span>
</button>`, checkedAttr, bgColor, basePath, recordID, c.name, transform)
}
