package tables

import (
	"fmt"
	"html"
	"strings"
)

// FilterType categorizes the filter UI/behavior.
type FilterType string

const (
	FilterTypeSelect  FilterType = "select"
	FilterTypeTernary FilterType = "ternary"
	FilterTypeRefined FilterType = "refined"
	FilterTypeCustom  FilterType = "custom"
)

// FilterOption represents an individual selectable value with optional count.
type FilterOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Count int    `json:"count,omitempty"`
}

// FilterDTO describes filter metadata for JSON serialization across REST and Flutter.
type FilterDTO struct {
	Name     string         `json:"name"`
	Label    string         `json:"label"`
	Type     FilterType     `json:"type"`
	Column   string         `json:"column"`
	Multiple bool           `json:"multiple"`
	Options  []FilterOption `json:"options"`
}

// FilterInterface defines the table filter contract.
type FilterInterface interface {
	GetName() string
	GetLabel() string
	GetType() FilterType
	GetColumn() string
	IsMultiple() bool
	GetOptions() []FilterOption
	Render(currentValue string) string
	ToDTO() FilterDTO
}

// ----------------------------------------------------
// Base Filter (Custom)
// ----------------------------------------------------

type FilterDef struct {
	name     string
	label    string
	column   string
	multiple bool
	options  []FilterOption
}

func Filter(name string) *FilterDef {
	return &FilterDef{
		name:   name,
		column: name,
	}
}

func (f *FilterDef) GetName() string       { return f.name }
func (f *FilterDef) GetType() FilterType   { return FilterTypeCustom }
func (f *FilterDef) GetColumn() string     { return f.column }
func (f *FilterDef) IsMultiple() bool      { return f.multiple }
func (f *FilterDef) GetOptions() []FilterOption { return f.options }

func (f *FilterDef) Label(l string) *FilterDef {
	f.label = l
	return f
}

func (f *FilterDef) Column(c string) *FilterDef {
	f.column = c
	return f
}

func (f *FilterDef) Multiple(m ...bool) *FilterDef {
	f.multiple = len(m) == 0 || m[0]
	return f
}

func (f *FilterDef) GetLabel() string {
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

func (f *FilterDef) Render(currentValue string) string {
	return fmt.Sprintf(`<div class="form-group" style="min-width: 8rem; margin-bottom: 0;">`+
		`<label class="form-label" style="font-size: 0.75rem;">%s</label>`+
		`<input type="text" name="filter_%s" value="%s" class="form-control" onchange="this.form.submit()" />`+
		`</div>`, html.EscapeString(f.GetLabel()), f.name, html.EscapeString(currentValue))
}

func (f *FilterDef) ToDTO() FilterDTO {
	return FilterDTO{
		Name:     f.GetName(),
		Label:    f.GetLabel(),
		Type:     f.GetType(),
		Column:   f.GetColumn(),
		Multiple: f.IsMultiple(),
		Options:  f.GetOptions(),
	}
}

// ----------------------------------------------------
// SelectFilter
// ----------------------------------------------------

type SelectFilterDef struct {
	name     string
	label    string
	column   string
	multiple bool
	options  map[string]string
	order    []string
}

func SelectFilter(name string) *SelectFilterDef {
	return &SelectFilterDef{
		name:    name,
		column:  name,
		options: make(map[string]string),
	}
}

func (f *SelectFilterDef) GetName() string       { return f.name }
func (f *SelectFilterDef) GetType() FilterType   { return FilterTypeSelect }
func (f *SelectFilterDef) GetColumn() string     { return f.column }
func (f *SelectFilterDef) IsMultiple() bool      { return f.multiple }

func (f *SelectFilterDef) Column(c string) *SelectFilterDef {
	f.column = c
	return f
}

func (f *SelectFilterDef) Multiple(m ...bool) *SelectFilterDef {
	f.multiple = len(m) == 0 || m[0]
	return f
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

func (f *SelectFilterDef) GetOptions() []FilterOption {
	var list []FilterOption
	for _, k := range f.order {
		list = append(list, FilterOption{Value: k, Label: f.options[k]})
	}
	return list
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

func (f *SelectFilterDef) ToDTO() FilterDTO {
	return FilterDTO{
		Name:     f.GetName(),
		Label:    f.GetLabel(),
		Type:     f.GetType(),
		Column:   f.GetColumn(),
		Multiple: f.IsMultiple(),
		Options:  f.GetOptions(),
	}
}

// TrashedFilter creates a specialized select filter for soft-deleted/trashed records.
func TrashedFilter(name ...string) *SelectFilterDef {
	filterName := "trashed"
	if len(name) > 0 && name[0] != "" {
		filterName = name[0]
	}
	return &SelectFilterDef{
		name:   filterName,
		label:  "Trashed Records",
		column: "deleted_at",
		options: map[string]string{
			"without_trashed": "Without Trashed",
			"with_trashed":    "With Trashed",
			"only_trashed":    "Only Trashed",
		},
		order: []string{"without_trashed", "with_trashed", "only_trashed"},
	}
}

// ----------------------------------------------------
// RefinedFilter (Refined Search / Faceted Search Filter)
// Features: Multi-select Checkboxes, Option Counts, Auto-Filtering
// ----------------------------------------------------

type RefinedFilterDef struct {
	name     string
	label    string
	column   string
	multiple bool
	options  map[string]string
	order    []string
	counts   map[string]int
}

// RefinedFilter creates an advanced faceted checkbox filter with live counts.
func RefinedFilter(name string) *RefinedFilterDef {
	return &RefinedFilterDef{
		name:     name,
		column:   name,
		multiple: true, // Checkboxes allow multiple selections by default
		options:  make(map[string]string),
		counts:   make(map[string]int),
	}
}

func (f *RefinedFilterDef) GetName() string       { return f.name }
func (f *RefinedFilterDef) GetType() FilterType   { return FilterTypeRefined }
func (f *RefinedFilterDef) GetColumn() string     { return f.column }
func (f *RefinedFilterDef) IsMultiple() bool      { return f.multiple }

func (f *RefinedFilterDef) Label(l string) *RefinedFilterDef {
	f.label = l
	return f
}

func (f *RefinedFilterDef) Column(c string) *RefinedFilterDef {
	f.column = c
	return f
}

func (f *RefinedFilterDef) Multiple(m ...bool) *RefinedFilterDef {
	f.multiple = len(m) == 0 || m[0]
	return f
}

func (f *RefinedFilterDef) Options(opts map[string]string) *RefinedFilterDef {
	f.options = opts
	f.order = make([]string, 0, len(opts))
	for k := range opts {
		f.order = append(f.order, k)
	}
	return f
}

func (f *RefinedFilterDef) Counts(counts map[string]int) *RefinedFilterDef {
	f.counts = counts
	return f
}

func (f *RefinedFilterDef) GetLabel() string {
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

func (f *RefinedFilterDef) GetOptions() []FilterOption {
	var list []FilterOption
	for _, k := range f.order {
		c := 0
		if f.counts != nil {
			c = f.counts[k]
		}
		list = append(list, FilterOption{
			Value: k,
			Label: f.options[k],
			Count: c,
		})
	}
	return list
}

func (f *RefinedFilterDef) Render(currentValue string) string {
	selectedMap := make(map[string]bool)
	for _, v := range strings.Split(currentValue, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			selectedMap[v] = true
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<div class="refined-filter-group" style="min-width: 12rem; margin-bottom: 0.5rem;">`))
	sb.WriteString(fmt.Sprintf(`<div class="form-label" style="font-size: 0.75rem; font-weight: 700; text-transform: uppercase; margin-bottom: 4px;">%s</div>`, html.EscapeString(f.GetLabel())))
	sb.WriteString(`<div class="refined-options-list" style="max-height: 140px; overflow-y: auto; display: flex; flex-direction: column; gap: 4px;">`)

	for _, k := range f.order {
		lbl := f.options[k]
		checked := ""
		if selectedMap[k] {
			checked = "checked"
		}
		countStr := ""
		if f.counts != nil {
			if cnt, exists := f.counts[k]; exists {
				countStr = fmt.Sprintf(` <span class="badge" style="font-size: 0.7rem; opacity: 0.8; margin-left: 4px;">(%d)</span>`, cnt)
			}
		}

		sb.WriteString(fmt.Sprintf(
			`<label class="form-check" style="display: flex; align-items: center; justify-content: space-between; font-size: 0.825rem; margin: 0; cursor: pointer;">`+
				`<span><input type="checkbox" name="filter_%s" value="%s" %s onchange="this.form.submit()" style="margin-right: 6px;" /> %s</span>`+
				`%s`+
				`</label>`,
			f.name, html.EscapeString(k), checked, html.EscapeString(lbl), countStr,
		))
	}

	sb.WriteString(`</div></div>`)
	return sb.String()
}

func (f *RefinedFilterDef) ToDTO() FilterDTO {
	return FilterDTO{
		Name:     f.GetName(),
		Label:    f.GetLabel(),
		Type:     f.GetType(),
		Column:   f.GetColumn(),
		Multiple: f.IsMultiple(),
		Options:  f.GetOptions(),
	}
}

// ----------------------------------------------------
// TernaryFilter
// ----------------------------------------------------

type TernaryFilterDef struct {
	name       string
	label      string
	column     string
	trueLabel  string
	falseLabel string
}

func TernaryFilter(name string) *TernaryFilterDef {
	return &TernaryFilterDef{
		name:       name,
		column:     name,
		trueLabel:  "Yes",
		falseLabel: "No",
	}
}

func (f *TernaryFilterDef) GetName() string       { return f.name }
func (f *TernaryFilterDef) GetType() FilterType   { return FilterTypeTernary }
func (f *TernaryFilterDef) GetColumn() string     { return f.column }
func (f *TernaryFilterDef) IsMultiple() bool      { return false }

func (f *TernaryFilterDef) Column(c string) *TernaryFilterDef {
	f.column = c
	return f
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

func (f *TernaryFilterDef) GetOptions() []FilterOption {
	return []FilterOption{
		{Value: "true", Label: f.trueLabel},
		{Value: "false", Label: f.falseLabel},
	}
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

func (f *TernaryFilterDef) ToDTO() FilterDTO {
	return FilterDTO{
		Name:     f.GetName(),
		Label:    f.GetLabel(),
		Type:     f.GetType(),
		Column:   f.GetColumn(),
		Multiple: f.IsMultiple(),
		Options:  f.GetOptions(),
	}
}

