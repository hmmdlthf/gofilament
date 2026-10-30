package infolists

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// Entry defines the contract for infolist entries and containers.
type Entry interface {
	GetName() string
	Render(record any) string
}

// ----------------------------------------------------
// BaseEntry
// ----------------------------------------------------

type BaseEntry struct {
	name      string
	label     string
	weight    string
	copyable  bool
	valueFunc func(record any) any
}

func (e *BaseEntry) GetName() string {
	return e.name
}

func (e *BaseEntry) Label(l string) *BaseEntry {
	e.label = l
	return e
}

func (e *BaseEntry) Weight(w string) *BaseEntry {
	e.weight = w
	return e
}

func (e *BaseEntry) Copyable(c ...bool) *BaseEntry {
	e.copyable = len(c) == 0 || c[0]
	return e
}

func (e *BaseEntry) FormatLabel() string {
	if e.label != "" {
		return e.label
	}
	words := strings.Split(e.name, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// ----------------------------------------------------
// TextEntry
// ----------------------------------------------------

type TextEntryDef struct {
	BaseEntry
	isDateTime  bool
	dateTimeFmt string
	currency    string
}

func TextEntry(name string) *TextEntryDef {
	return &TextEntryDef{
		BaseEntry: BaseEntry{name: name},
	}
}

func (e *TextEntryDef) Label(l string) *TextEntryDef {
	e.BaseEntry.Label(l)
	return e
}

func (e *TextEntryDef) Weight(w string) *TextEntryDef {
	e.BaseEntry.Weight(w)
	return e
}

func (e *TextEntryDef) Copyable(c ...bool) *TextEntryDef {
	e.BaseEntry.Copyable(c...)
	return e
}

func (e *TextEntryDef) DateTime(fmt ...string) *TextEntryDef {
	e.isDateTime = true
	if len(fmt) > 0 {
		e.dateTimeFmt = fmt[0]
	} else {
		e.dateTimeFmt = "2006-01-02 15:04"
	}
	return e
}

func (e *TextEntryDef) Money(currency string) *TextEntryDef {
	e.currency = currency
	return e
}

func (e *TextEntryDef) Render(record any) string {
	val := ""
	if m, ok := record.(map[string]any); ok {
		if raw, exists := m[e.name]; exists {
			if t, ok := raw.(time.Time); ok && e.isDateTime {
				val = t.Format(e.dateTimeFmt)
			} else if raw != nil {
				val = fmt.Sprintf("%v", raw)
			}
		}
	}

	if e.currency != "" && val != "" {
		val = fmt.Sprintf("%s %s", e.currency, val)
	}

	style := ""
	if e.weight == "bold" {
		style = "font-weight: 700;"
	}

	copyAttr := ""
	if e.copyable && val != "" {
		copyAttr = fmt.Sprintf(`onclick="navigator.clipboard.writeText('%s'); Filament.notify({type:'info', title:'Copied'});" style="cursor: pointer;" title="Click to copy"`, html.EscapeString(val))
	}

	return fmt.Sprintf(`
<div style="margin-bottom: 1rem;">
  <div style="font-size: 0.75rem; font-weight: 500; color: var(--gray-500); text-transform: uppercase;">%s</div>
  <div style="margin-top: 0.25rem; font-size: 0.9375rem; color: var(--gray-900); %s" %s>%s</div>
</div>`,
		html.EscapeString(e.FormatLabel()), style, copyAttr, html.EscapeString(val),
	)
}

// ----------------------------------------------------
// BadgeEntry
// ----------------------------------------------------

type BadgeEntryDef struct {
	BaseEntry
	colors map[string]string
}

func BadgeEntry(name string) *BadgeEntryDef {
	return &BadgeEntryDef{
		BaseEntry: BaseEntry{name: name},
		colors:    make(map[string]string),
	}
}

func (e *BadgeEntryDef) Colors(c map[string]string) *BadgeEntryDef {
	e.colors = c
	return e
}

func (e *BadgeEntryDef) Render(record any) string {
	val := ""
	if m, ok := record.(map[string]any); ok {
		if raw, exists := m[e.name]; exists && raw != nil {
			val = fmt.Sprintf("%v", raw)
		}
	}
	colorName := "gray"
	if col, ok := e.colors[val]; ok {
		colorName = col
	}

	return fmt.Sprintf(`
<div style="margin-bottom: 1rem;">
  <div style="font-size: 0.75rem; font-weight: 500; color: var(--gray-500); text-transform: uppercase;">%s</div>
  <div style="margin-top: 0.25rem;"><span class="badge badge-%s">%s</span></div>
</div>`,
		html.EscapeString(e.FormatLabel()), colorName, html.EscapeString(val),
	)
}
