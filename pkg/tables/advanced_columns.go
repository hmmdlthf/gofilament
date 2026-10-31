package tables

import (
	"fmt"
	"html"
	"strings"
)

// ----------------------------------------------------
// IconColumn
// ----------------------------------------------------

type IconColumnDef[T any] struct {
	BaseColumn[T]
	icons  map[any]string
	colors map[any]string
}

func IconColumn[T any](name string) *IconColumnDef[T] {
	return &IconColumnDef[T]{
		BaseColumn: BaseColumn[T]{name: name},
		icons:      make(map[any]string),
		colors:     make(map[any]string),
	}
}

func (c *IconColumnDef[T]) Icons(m map[any]string) *IconColumnDef[T] {
	c.icons = m
	return c
}

func (c *IconColumnDef[T]) Colors(m map[any]string) *IconColumnDef[T] {
	c.colors = m
	return c
}

func (c *IconColumnDef[T]) Boolean() *IconColumnDef[T] {
	c.icons[true] = "heroicon-check-circle"
	c.icons[false] = "heroicon-x-circle"
	c.colors[true] = "success"
	c.colors[false] = "danger"
	return c
}

func (c *IconColumnDef[T]) RenderCell(record T, rawValue any) string {
	if c.valueFunc != nil {
		rawValue = c.valueFunc(record)
	}

	icon := "•"
	color := "#64748b"

	if b, ok := rawValue.(bool); ok {
		if b {
			icon = "✓"
			color = "#10b981"
		} else {
			icon = "✕"
			color = "#ef4444"
		}
	} else if ic, ok := c.icons[rawValue]; ok {
		icon = ic
	}

	return fmt.Sprintf(`<span style="color: %s; font-weight: bold; font-size: 1.125rem;">%s</span>`, color, html.EscapeString(icon))
}

// ----------------------------------------------------
// ColorColumn
// ----------------------------------------------------

type ColorColumnDef[T any] struct {
	BaseColumn[T]
	copyable bool
}

func ColorColumn[T any](name string) *ColorColumnDef[T] {
	return &ColorColumnDef[T]{
		BaseColumn: BaseColumn[T]{name: name},
	}
}

func (c *ColorColumnDef[T]) RenderCell(record T, rawValue any) string {
	if c.valueFunc != nil {
		rawValue = c.valueFunc(record)
	}
	hex := "#000000"
	if rawValue != nil {
		hex = fmt.Sprintf("%v", rawValue)
	}

	return fmt.Sprintf(`
<div style="display: flex; align-items: center; gap: 0.5rem;">
  <span style="display: inline-block; width: 1.25rem; height: 1.25rem; border-radius: 9999px; background-color: %s; border: 1px solid rgba(0,0,0,0.1);"></span>
  <span style="font-family: monospace; font-size: 0.75rem;">%s</span>
</div>`, html.EscapeString(hex), html.EscapeString(hex))
}

// ----------------------------------------------------
// SelectColumn
// ----------------------------------------------------

type SelectColumnDef[T any] struct {
	BaseColumn[T]
	options map[string]string
}

func SelectColumn[T any](name string) *SelectColumnDef[T] {
	return &SelectColumnDef[T]{
		BaseColumn: BaseColumn[T]{name: name},
		options:    make(map[string]string),
	}
}

func (c *SelectColumnDef[T]) Options(opts map[string]string) *SelectColumnDef[T] {
	c.options = opts
	return c
}

func (c *SelectColumnDef[T]) RenderCell(record T, rawValue any) string {
	if c.valueFunc != nil {
		rawValue = c.valueFunc(record)
	}
	current := fmt.Sprintf("%v", rawValue)

	var optHTML string
	for k, v := range c.options {
		selected := ""
		if k == current {
			selected = "selected"
		}
		optHTML += fmt.Sprintf(`<option value="%s" %s>%s</option>`, html.EscapeString(k), selected, html.EscapeString(v))
	}

	return fmt.Sprintf(`
<select class="form-control" style="padding: 0.25rem 0.5rem; font-size: 0.8125rem; width: auto;" onchange="Filament.notify({title:'Updated', body:'Status saved'});">
  %s
</select>`, optHTML)
}

func (c *SelectColumnDef[T]) RenderCellWithContext(record T, recordID, basePath string, rawValue any) string {
	if c.valueFunc != nil {
		rawValue = c.valueFunc(record)
	}
	current := fmt.Sprintf("%v", rawValue)

	var optHTML strings.Builder
	for k, v := range c.options {
		selected := ""
		if k == current {
			selected = "selected"
		}
		optHTML.WriteString(fmt.Sprintf(`<option value="%s" %s>%s</option>`, html.EscapeString(k), selected, html.EscapeString(v)))
	}

	return fmt.Sprintf(`
<select class="form-control form-control-sm" style="padding: 0.25rem 0.5rem; font-size: 0.8125rem; width: auto;"
        onchange="Filament.updateCell('%s/cell', '%s', '%s', this.value, this);">
  %s
</select>`, basePath, recordID, c.name, optHTML.String())
}

// ----------------------------------------------------
// TextInputColumn
// ----------------------------------------------------

type TextInputColumnDef[T any] struct {
	BaseColumn[T]
}

func TextInputColumn[T any](name string) *TextInputColumnDef[T] {
	return &TextInputColumnDef[T]{
		BaseColumn: BaseColumn[T]{name: name},
	}
}

func (c *TextInputColumnDef[T]) RenderCell(record T, rawValue any) string {
	if c.valueFunc != nil {
		rawValue = c.valueFunc(record)
	}
	val := fmt.Sprintf("%v", rawValue)
	return fmt.Sprintf(`<input type="text" value="%s" class="form-control" style="padding: 0.25rem 0.5rem; font-size: 0.8125rem; max-width: 12rem;" onchange="Filament.notify({title:'Saved', body:'Field value updated'});" />`, html.EscapeString(val))
}

func (c *TextInputColumnDef[T]) RenderCellWithContext(record T, recordID, basePath string, rawValue any) string {
	if c.valueFunc != nil {
		rawValue = c.valueFunc(record)
	}
	val := fmt.Sprintf("%v", rawValue)
	if rawValue == nil {
		val = ""
	}
	return fmt.Sprintf(`
<input type="text" value="%s" class="form-control form-control-sm"
       style="padding: 0.25rem 0.5rem; font-size: 0.8125rem; max-width: 12rem; border-color: var(--gray-300);"
       onchange="Filament.updateCell('%s/cell', '%s', '%s', this.value, this);" />`,
		html.EscapeString(val), basePath, recordID, c.name)
}
