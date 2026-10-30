package tables

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// BaseColumn defines shared attributes for table columns.
type BaseColumn[T any] struct {
	name       string
	label      string
	sortable   bool
	searchable bool
	weight     string // bold, medium, normal
	icon       string
	copyable   bool
	align      string // left, center, right
	valueFunc  func(record T) any
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
