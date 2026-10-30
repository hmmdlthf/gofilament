package infolists

import (
	"fmt"
	"html"
	"strings"
)

// ----------------------------------------------------
// IconEntry
// ----------------------------------------------------

type IconEntryDef struct {
	BaseEntry
	isBoolean bool
}

func IconEntry(name string) *IconEntryDef {
	return &IconEntryDef{BaseEntry: BaseEntry{name: name}}
}

func (e *IconEntryDef) Boolean() *IconEntryDef {
	e.isBoolean = true
	return e
}

func (e *IconEntryDef) Label(l string) *IconEntryDef {
	e.BaseEntry.Label(l)
	return e
}

func (e *IconEntryDef) Render(record any) string {
	val := ""
	if m, ok := record.(map[string]any); ok {
		if raw, exists := m[e.name]; exists && raw != nil {
			val = fmt.Sprintf("%v", raw)
		}
	}

	icon := "✓"
	color := "#10b981"
	if val == "false" || val == "0" || val == "" {
		icon = "✕"
		color = "#ef4444"
	}

	return fmt.Sprintf(`
<div style="margin-bottom: 1rem;">
  <div style="font-size: 0.75rem; font-weight: 500; color: var(--gray-500); text-transform: uppercase;">%s</div>
  <div style="margin-top: 0.25rem; font-size: 1.125rem; font-weight: bold; color: %s;">%s</div>
</div>`, html.EscapeString(e.FormatLabel()), color, html.EscapeString(icon))
}

// ----------------------------------------------------
// ImageEntry
// ----------------------------------------------------

type ImageEntryDef struct {
	BaseEntry
	circular bool
	size     string
}

func ImageEntry(name string) *ImageEntryDef {
	return &ImageEntryDef{
		BaseEntry: BaseEntry{name: name},
		size:      "3rem",
	}
}

func (e *ImageEntryDef) Circular(circ ...bool) *ImageEntryDef {
	e.circular = len(circ) == 0 || circ[0]
	return e
}

func (e *ImageEntryDef) Render(record any) string {
	url := ""
	if m, ok := record.(map[string]any); ok {
		if raw, exists := m[e.name]; exists && raw != nil {
			url = fmt.Sprintf("%v", raw)
		}
	}

	radius := "0.375rem"
	if e.circular {
		radius = "9999px"
	}

	return fmt.Sprintf(`
<div style="margin-bottom: 1rem;">
  <div style="font-size: 0.75rem; font-weight: 500; color: var(--gray-500); text-transform: uppercase;">%s</div>
  <div style="margin-top: 0.25rem;">
    <img src="%s" alt="image" style="width: %s; height: %s; border-radius: %s; object-fit: cover;" />
  </div>
</div>`, html.EscapeString(e.FormatLabel()), html.EscapeString(url), e.size, e.size, radius)
}

// ----------------------------------------------------
// RepeatableEntry
// ----------------------------------------------------

type RepeatableEntryDef struct {
	BaseEntry
	columns int
	schema  []Entry
}

func RepeatableEntry(name string) *RepeatableEntryDef {
	return &RepeatableEntryDef{
		BaseEntry: BaseEntry{name: name},
		columns:   1,
		schema:    make([]Entry, 0),
	}
}

func (e *RepeatableEntryDef) Columns(c int) *RepeatableEntryDef {
	e.columns = c
	return e
}

func (e *RepeatableEntryDef) Schema(entries ...Entry) *RepeatableEntryDef {
	e.schema = entries
	return e
}

func (e *RepeatableEntryDef) Render(record any) string {
	var items []map[string]any
	if m, ok := record.(map[string]any); ok {
		if raw, exists := m[e.name]; exists {
			if slice, ok := raw.([]map[string]any); ok {
				items = slice
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<div style="margin-bottom: 1.5rem;"><div style="font-size: 0.75rem; font-weight: 600; text-transform: uppercase; color: var(--gray-500); margin-bottom: 0.5rem;">%s</div>`, html.EscapeString(e.FormatLabel())))

	for _, item := range items {
		colClass := fmt.Sprintf("grid-cols-%d", e.columns)
		sb.WriteString(fmt.Sprintf(`<div class="grid %s" style="border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 0.75rem; margin-bottom: 0.5rem; background-color: var(--gray-50);">`, colClass))
		for _, sub := range e.schema {
			sb.WriteString(sub.Render(item))
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</div>`)
	return sb.String()
}
