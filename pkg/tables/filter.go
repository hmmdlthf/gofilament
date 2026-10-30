package tables

import (
	"fmt"
	"html"
	"strings"
)

// FilterInterface defines the table filter contract.
type FilterInterface interface {
	GetName() string
	GetLabel() string
	Render(currentValue string) string
}

// ----------------------------------------------------
// SelectFilter
// ----------------------------------------------------

type SelectFilterDef struct {
	name    string
	label   string
	options map[string]string
	order   []string
}

func SelectFilter(name string) *SelectFilterDef {
	return &SelectFilterDef{
		name:    name,
		options: make(map[string]string),
	}
}

func (f *SelectFilterDef) GetName() string {
	return f.name
}

func (f *SelectFilterDef) Label(l string) *SelectFilterDef {
	f.label = l
	return f
}

func (f *SelectFilterDef) GetLabel() string {
	if f.label != "" {
		return f.label
	}
	words := strings.Split(f.name, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func (f *SelectFilterDef) Options(opts map[string]string) *SelectFilterDef {
	f.options = opts
	f.order = make([]string, 0, len(opts))
	for k := range opts {
		f.order = append(f.order, k)
	}
	return f
}

func (f *SelectFilterDef) Render(currentValue string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<div class="form-group" style="min-width: 10rem; margin-bottom: 0;">`))
	sb.WriteString(fmt.Sprintf(`<label class="form-label" style="font-size: 0.75rem;">%s</label>`, html.EscapeString(f.GetLabel())))
	sb.WriteString(fmt.Sprintf(`<select name="filter_%s" class="form-control" onchange="this.form.submit()">`, f.name))
	sb.WriteString(`<option value="">All</option>`)
	for _, k := range f.order {
		selected := ""
		if k == currentValue {
			selected = "selected"
		}
		sb.WriteString(fmt.Sprintf(`<option value="%s" %s>%s</option>`, html.EscapeString(k), selected, html.EscapeString(f.options[k])))
	}
	sb.WriteString(`</select></div>`)
	return sb.String()
}

// ----------------------------------------------------
// TernaryFilter
// ----------------------------------------------------

type TernaryFilterDef struct {
	name       string
	label      string
	trueLabel  string
	falseLabel string
}

func TernaryFilter(name string) *TernaryFilterDef {
	return &TernaryFilterDef{
		name:       name,
		trueLabel:  "Yes",
		falseLabel: "No",
	}
}

func (f *TernaryFilterDef) GetName() string {
	return f.name
}

func (f *TernaryFilterDef) Label(l string) *TernaryFilterDef {
	f.label = l
	return f
}

func (f *TernaryFilterDef) GetLabel() string {
	if f.label != "" {
		return f.label
	}
	return f.name
}

func (f *TernaryFilterDef) TrueLabel(l string) *TernaryFilterDef {
	f.trueLabel = l
	return f
}

func (f *TernaryFilterDef) FalseLabel(l string) *TernaryFilterDef {
	f.falseLabel = l
	return f
}

func (f *TernaryFilterDef) Render(currentValue string) string {
	return fmt.Sprintf(`
<div class="form-group" style="min-width: 8rem; margin-bottom: 0;">
  <label class="form-label" style="font-size: 0.75rem;">%s</label>
  <select name="filter_%s" class="form-control" onchange="this.form.submit()">
    <option value="">All</option>
    <option value="true" %s>%s</option>
    <option value="false" %s>%s</option>
  </select>
</div>`,
		html.EscapeString(f.GetLabel()),
		f.name,
		func() string { if currentValue == "true" { return "selected" }; return "" }(),
		html.EscapeString(f.trueLabel),
		func() string { if currentValue == "false" { return "selected" }; return "" }(),
		html.EscapeString(f.falseLabel),
	)
}
