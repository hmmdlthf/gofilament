package tables

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"
)

// Range filter types.
const (
	FilterTypeDateRange   FilterType = "date_range"
	FilterTypeNumberRange FilterType = "number_range"
)

// RangeFilterDef filters a date or number column between two bounds. Its
// value is "from,to" (either side may be empty: "2026-01-01," is "on or
// after"). Dates are YYYY-MM-DD and the end date is inclusive.
type RangeFilterDef struct {
	name   string
	label  string
	column string
	kind   FilterType
}

// DateRangeFilter filters a date or datetime column between two dates.
func DateRangeFilter(name string) *RangeFilterDef {
	return &RangeFilterDef{name: name, column: name, kind: FilterTypeDateRange}
}

// NumberRangeFilter filters a numeric column between a minimum and maximum.
func NumberRangeFilter(name string) *RangeFilterDef {
	return &RangeFilterDef{name: name, column: name, kind: FilterTypeNumberRange}
}

func (f *RangeFilterDef) Label(l string) *RangeFilterDef  { f.label = l; return f }
func (f *RangeFilterDef) Column(c string) *RangeFilterDef { f.column = c; return f }

func (f *RangeFilterDef) GetName() string            { return f.name }
func (f *RangeFilterDef) GetType() FilterType        { return f.kind }
func (f *RangeFilterDef) GetColumn() string          { return f.column }
func (f *RangeFilterDef) IsMultiple() bool           { return false }
func (f *RangeFilterDef) GetOptions() []FilterOption { return nil }
func (f *RangeFilterDef) GetLabel() string {
	if f.label != "" {
		return f.label
	}
	return humanizeName(f.name)
}

func (f *RangeFilterDef) ToDTO() FilterDTO {
	return FilterDTO{Name: f.name, Label: f.GetLabel(), Type: f.kind, Column: f.column}
}

// Render draws two inputs, filter_<name>_from and filter_<name>_to; the
// panel joins them into the "from,to" value.
func (f *RangeFilterDef) Render(currentValue string) string {
	from, to := f.Bounds(currentValue)
	typ, step := "date", ""
	if f.kind == FilterTypeNumberRange {
		typ, step = "number", ` step="any"`
	}
	input := func(suffix, val, ph string) string {
		return fmt.Sprintf(`<input type="%s"%s name="filter_%s_%s" value="%s" placeholder="%s" aria-label="%s %s" class="form-control" style="width: 9.5rem;" onchange="this.form.submit()" />`,
			typ, step, html.EscapeString(f.name), suffix, html.EscapeString(val), ph, html.EscapeString(f.GetLabel()), ph)
	}
	return fmt.Sprintf(`<div class="range-filter" style="display: flex; align-items: center; gap: 0.375rem;"><span class="form-label" style="margin: 0; font-size: 0.75rem;">%s</span>%s<span>–</span>%s</div>`,
		html.EscapeString(f.GetLabel()), input("from", from, "from"), input("to", to, "to"))
}

// Bounds returns the valid from / to of a "from,to" value ("" for a
// missing or malformed side).
func (f *RangeFilterDef) Bounds(value string) (from, to string) {
	parts := strings.SplitN(value, ",", 2)
	from = strings.TrimSpace(parts[0])
	if len(parts) == 2 {
		to = strings.TrimSpace(parts[1])
	}
	valid := func(s string) bool {
		if s == "" {
			return false
		}
		if f.kind == FilterTypeDateRange {
			_, err := time.Parse("2006-01-02", s)
			return err == nil
		}
		_, err := strconv.ParseFloat(s, 64)
		return err == nil
	}
	if !valid(from) {
		from = ""
	}
	if !valid(to) {
		to = ""
	}
	return from, to
}

// Rules turns the filter's value into advanced-search rules on its column,
// ready for BuildAdvancedSearchSQLFor (map the column name to its SQL
// expression). The end date is inclusive: "to" becomes "before the next
// day", so datetime columns include the whole last day.
func (f *RangeFilterDef) Rules(value string) []AdvancedSearchRule {
	from, to := f.Bounds(value)
	var rules []AdvancedSearchRule
	if from != "" {
		rules = append(rules, AdvancedSearchRule{Field: f.column, Operator: "greater_or_equal", Value: from})
	}
	if to != "" {
		if f.kind == FilterTypeDateRange {
			d, _ := time.Parse("2006-01-02", to)
			rules = append(rules, AdvancedSearchRule{Field: f.column, Operator: "less_than", Value: d.AddDate(0, 0, 1).Format("2006-01-02")})
		} else {
			rules = append(rules, AdvancedSearchRule{Field: f.column, Operator: "less_or_equal", Value: to})
		}
	}
	return rules
}

// Condition is Rules wrapped in an AND condition (nil when the value is
// empty or invalid).
func (f *RangeFilterDef) Condition(value string) *AdvancedSearchCondition {
	rules := f.Rules(value)
	if len(rules) == 0 {
		return nil
	}
	return &AdvancedSearchCondition{Conjunction: "AND", Rules: rules}
}

// Matches reports whether v (a time.Time, a "2006-01-02..." string or a
// number) is inside the range, for in-memory lists.
func (f *RangeFilterDef) Matches(value string, v any) bool {
	from, to := f.Bounds(value)
	if from == "" && to == "" {
		return true
	}
	if f.kind == FilterTypeDateRange {
		var day string
		switch t := v.(type) {
		case time.Time:
			day = t.Format("2006-01-02")
		case string:
			if len(t) < 10 {
				return false
			}
			day = t[:10]
		default:
			return false
		}
		return (from == "" || day >= from) && (to == "" || day <= to)
	}
	n, ok := toFloat(v)
	if !ok {
		return false
	}
	lo, _ := strconv.ParseFloat(from, 64)
	hi, _ := strconv.ParseFloat(to, 64)
	return (from == "" || n >= lo) && (to == "" || n <= hi)
}
