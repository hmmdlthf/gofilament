package tables

import (
	"fmt"
	"html"
	"math"
	"net/url"
	"strconv"
	"strings"
)

// Summary is an aggregate shown under a column (and under each group).
type Summary string

const (
	Sum   Summary = "sum"
	Avg   Summary = "avg"
	Count Summary = "count"
	Min   Summary = "min"
	Max   Summary = "max"
)

var summaryLabels = map[Summary]string{Sum: "Sum", Avg: "Average", Count: "Count", Min: "Min", Max: "Max"}

// Summarize shows the given aggregates under the column: for the rows on
// the page (and for each group), and for all matching records when the
// resource provides SummariesUsing.
func (c *BaseColumn[T]) Summarize(s ...Summary) *BaseColumn[T] {
	c.summaries = append(c.summaries, s...)
	return c
}

// GetSummaries returns the column's aggregates.
func (c *BaseColumn[T]) GetSummaries() []Summary { return c.summaries }

// Summarize shows the given aggregates under the column.
func (c *TextColumnDef[T]) Summarize(s ...Summary) *TextColumnDef[T] {
	c.BaseColumn.Summarize(s...)
	return c
}

// Summaries holds computed aggregates: column -> summary -> value.
type Summaries map[string]map[Summary]float64

// Compute aggregates values (numbers or numeric strings; others are
// skipped, except for Count, which counts every non-empty value).
func Compute(values []any, kinds []Summary) map[Summary]float64 {
	out := map[Summary]float64{}
	var nums []float64
	count := 0
	for _, v := range values {
		if v == nil || v == "" {
			continue
		}
		count++
		if n, ok := toFloat(v); ok && !math.IsNaN(n) {
			nums = append(nums, n)
		}
	}
	for _, k := range kinds {
		switch k {
		case Count:
			out[k] = float64(count)
		case Sum, Avg:
			s := 0.0
			for _, n := range nums {
				s += n
			}
			if k == Avg {
				if len(nums) == 0 {
					continue
				}
				s /= float64(len(nums))
			}
			out[k] = s
		case Min, Max:
			if len(nums) == 0 {
				continue
			}
			m := nums[0]
			for _, n := range nums[1:] {
				if (k == Min && n < m) || (k == Max && n > m) {
					m = n
				}
			}
			out[k] = m
		}
	}
	return out
}

// FormatSummaryValue prints an aggregate with at most two decimals.
func FormatSummaryValue(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
}

// --------------------------------------------------------------- grouping

// Groups lets users group the table's rows by one of these columns (a
// header row per value, with subtotals of summarized columns). The list is
// sorted by the group column first, so a group is never split by sorting.
func (t *Table[T]) Groups(cols ...string) *Table[T] {
	t.groups = append(t.groups, cols...)
	return t
}

// DefaultGroup groups the rows by col until the user picks another group
// (or none). col is added to Groups.
func (t *Table[T]) DefaultGroup(col string) *Table[T] {
	t.defaultGroup = col
	if !t.isGroupable(col) {
		t.groups = append(t.groups, col)
	}
	return t
}

// GetGroups returns the columns rows can be grouped by.
func (t *Table[T]) GetGroups() []string { return t.groups }

// GetDefaultGroup returns the default group column ("" for none).
func (t *Table[T]) GetDefaultGroup() string { return t.defaultGroup }

func (t *Table[T]) isGroupable(col string) bool {
	for _, g := range t.groups {
		if g == col {
			return true
		}
	}
	return false
}

// ResolveGroup returns the group column for a request's "group" parameter:
// the requested column when allowed, "" for "none", else the default.
func (t *Table[T]) ResolveGroup(requested string) string {
	switch {
	case requested == "none":
		return ""
	case requested != "" && t.isGroupable(requested):
		return requested
	}
	return t.defaultGroup
}

// SetGroup sets the active group column (see ResolveGroup).
func (t *Table[T]) SetGroup(col string) *Table[T] { t.activeGroup = col; return t }

// SetTotals sets aggregates over all matching records (not just the page).
func (t *Table[T]) SetTotals(s Summaries) *Table[T] { t.totals = s; return t }

// SummarizedColumns returns the columns with aggregates.
func (t *Table[T]) SummarizedColumns() map[string][]Summary {
	out := map[string][]Summary{}
	for _, col := range t.columns {
		if s, ok := any(col).(interface{ GetSummaries() []Summary }); ok && len(s.GetSummaries()) > 0 {
			out[col.GetName()] = s.GetSummaries()
		}
	}
	return out
}

func (t *Table[T]) summarize(records []T) Summaries {
	out := Summaries{}
	for name, kinds := range t.SummarizedColumns() {
		vals := make([]any, 0, len(records))
		for _, r := range records {
			vals = append(vals, t.extractFieldValue(r, name))
		}
		out[name] = Compute(vals, kinds)
	}
	return out
}

func groupKey(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// summaryRowHTML renders one row of aggregates under the data columns.
func (t *Table[T]) summaryRowHTML(label string, s Summaries, lead, trail int, class string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, `<tr class="%s" style="background: var(--gray-50); font-size: 0.8125rem;">`, class)
	for i := 0; i < lead; i++ {
		sb.WriteString(`<td></td>`)
	}
	for i, col := range t.columns {
		cell := ""
		if kinds, ok := t.SummarizedColumns()[col.GetName()]; ok {
			var parts []string
			for _, k := range kinds {
				if v, ok := s[col.GetName()][k]; ok {
					parts = append(parts, fmt.Sprintf(`<div><span style="color: var(--gray-500);">%s</span> <strong>%s</strong></div>`, summaryLabels[k], FormatSummaryValue(v)))
				}
			}
			cell = strings.Join(parts, "")
		} else if i == 0 {
			cell = fmt.Sprintf(`<span style="color: var(--gray-500); font-weight: 600;">%s</span>`, html.EscapeString(label))
		}
		colStyle := ""
		if t.toggleable {
			colStyle = fmt.Sprintf(` :style="!visibleCols['%s'] ? 'display: none;' : ''"`, col.GetName())
		}
		fmt.Fprintf(&sb, `<td data-col="%s" data-label="%s"%s>%s</td>`, col.GetName(), html.EscapeString(col.FormatLabel()), colStyle, cell)
	}
	for i := 0; i < trail; i++ {
		sb.WriteString(`<td></td>`)
	}
	sb.WriteString(`</tr>`)
	return sb.String()
}

func (t *Table[T]) columnLabel(name string) string {
	for _, c := range t.columns {
		if c.GetName() == name {
			return c.FormatLabel()
		}
	}
	return humanizeName(name)
}

// groupSelectHTML renders the "Group by" select of the toolbar form.
func (t *Table[T]) groupSelectHTML() string {
	if len(t.groups) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(`<label style="display: inline-flex; align-items: center; gap: 0.375rem; font-size: 0.8125rem;">Group by <select name="group" class="form-control" style="width: auto;" onchange="this.form.submit()">`)
	sel := func(v string) string {
		if v == t.activeGroup || (v == "none" && t.activeGroup == "") {
			return " selected"
		}
		return ""
	}
	fmt.Fprintf(&sb, `<option value="none"%s>None</option>`, sel("none"))
	for _, g := range t.groups {
		fmt.Fprintf(&sb, `<option value="%s"%s>%s</option>`, html.EscapeString(g), sel(g), html.EscapeString(t.columnLabel(g)))
	}
	sb.WriteString(`</select></label>`)
	return sb.String()
}

// pageURL links to another page of the list, keeping the search, sort,
// page size, filters and group.
func (t *Table[T]) pageURL(basePath string, page int) string {
	q := t.listQuery()
	q.Set("page", strconv.Itoa(page))
	if t.perPage > 0 {
		q.Set("perPage", strconv.Itoa(t.perPage))
	}
	if len(t.groups) > 0 {
		g := t.activeGroup
		if g == "" {
			g = "none"
		}
		q.Set("group", g)
	}
	return basePath + "?" + q.Encode()
}

// listQuery is the list's search, sort and filters as query parameters.
func (t *Table[T]) listQuery() url.Values {
	q := url.Values{}
	if t.searchQuery != "" {
		q.Set("search", t.searchQuery)
	}
	if t.sortCol != "" {
		q.Set("sort", t.sortCol)
		q.Set("direction", t.sortDir)
	}
	for k, v := range t.activeFilters {
		if v != "" && !strings.HasPrefix(k, "_") {
			q.Set("filter_"+k, v)
		}
	}
	if adv := t.activeFilters["_advanced_search"]; adv != "" {
		q.Set("advanced_search", adv)
	}
	return q
}

// exportLinks renders the Export header action: CSV and Excel downloads of
// the list as filtered and sorted now.
func (t *Table[T]) exportLinks(basePath, name, label string) string {
	q := t.listQuery()
	csvURL := basePath + "/" + name
	if len(q) > 0 {
		csvURL += "?" + q.Encode()
	}
	q.Set("format", "xlsx")
	return fmt.Sprintf(`<a href="%s" class="btn btn-secondary" data-export="csv">%s CSV</a><a href="%s" class="btn btn-secondary" data-export="xlsx">Excel</a>`,
		html.EscapeString(csvURL), html.EscapeString(label), html.EscapeString(basePath+"/"+name+"?"+q.Encode()))
}
