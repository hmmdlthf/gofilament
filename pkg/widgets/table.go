package widgets

import (
	"fmt"
	"html"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
)

// TableWidgetColumn defines a simple column for the dashboard mini-table.
type TableWidgetColumn struct {
	Name  string
	Label string
	// Format: "" (text), "number", "money" or "percent".
	Format string
	// Align: "" (left; numbers default to right), "right" or "center".
	Align string
}

// RowURL is the row key that makes a row open a panel URL when tapped.
const RowURL = "_url"

// TableWidget renders an embedded mini data table inside a dashboard widget card.
type TableWidget struct {
	BaseWidget
	heading     string
	description string
	emptyText   string
	columns     []TableWidgetColumn
	rows        func(ctx *filament.Context) []map[string]any
}

func NewTableWidget(heading string) *TableWidget {
	w := &TableWidget{
		heading: heading,
		columns: make([]TableWidgetColumn, 0),
	}
	w.columnSpan = 3
	return w
}

// Description is shown under the heading.
func (w *TableWidget) Description(d string) *TableWidget {
	w.description = d
	return w
}

// EmptyText is shown when there are no rows.
func (w *TableWidget) EmptyText(t string) *TableWidget {
	w.emptyText = t
	return w
}

func (w *TableWidget) Columns(cols ...TableWidgetColumn) *TableWidget {
	w.columns = cols
	return w
}

func (w *TableWidget) Rows(provider func(ctx *filament.Context) []map[string]any) *TableWidget {
	w.rows = provider
	return w
}

func (w *TableWidget) GetHeading() string {
	return w.heading
}

func (w *TableWidget) GetColumns() []TableWidgetColumn {
	return w.columns
}

func (w *TableWidget) EvaluateRows(ctx *filament.Context) []map[string]any {
	if w.rows != nil {
		return w.rows(ctx)
	}
	return nil
}

func (w *TableWidget) WidgetType() string {
	return "table"
}

func (w *TableWidget) ToWidgetDTO(ctx *filament.Context) map[string]any {
	var cols []map[string]string
	for _, c := range w.columns {
		cols = append(cols, map[string]string{
			"name":   c.Name,
			"label":  c.Label,
			"format": c.Format,
			"align":  c.Align,
		})
	}
	rows := w.EvaluateRows(ctx)
	if rows == nil {
		rows = []map[string]any{}
	}
	return map[string]any{
		"type":        "table",
		"heading":     w.heading,
		"description": w.description,
		"empty_text":  w.emptyText,
		"columns":     cols,
		"rows":        rows,
	}
}

func (w *TableWidget) Render(ctx *filament.Context) string {
	var rowData []map[string]any
	if w.rows != nil {
		rowData = w.rows(ctx)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`
<div class="filament-card">
  <div style="margin-bottom: 1rem;">
    <h3 style="font-size: 1.125rem; font-weight: 600;">%s</h3>
  </div>
  <div class="filament-table-wrapper">
    <table class="filament-table">
      <thead><tr>`, html.EscapeString(w.heading)))

	for _, col := range w.columns {
		sb.WriteString(fmt.Sprintf(`<th>%s</th>`, html.EscapeString(col.Label)))
	}
	sb.WriteString(`</tr></thead><tbody>`)

	if len(rowData) == 0 {
		sb.WriteString(fmt.Sprintf(`<tr><td colspan="%d" style="text-align: center; color: var(--gray-400); padding: 2rem;">No data available.</td></tr>`, len(w.columns)))
	} else {
		for _, row := range rowData {
			sb.WriteString(`<tr>`)
			for _, col := range w.columns {
				val := fmt.Sprintf("%v", row[col.Name])
				sb.WriteString(fmt.Sprintf(`<td>%s</td>`, html.EscapeString(val)))
			}
			sb.WriteString(`</tr>`)
		}
	}

	sb.WriteString(`</tbody></table></div></div>`)
	return sb.String()
}
