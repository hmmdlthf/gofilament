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
}

// TableWidget renders an embedded mini data table inside a dashboard widget card.
type TableWidget struct {
	BaseWidget
	heading string
	columns []TableWidgetColumn
	rows    func(ctx *filament.Context) []map[string]any
}

func NewTableWidget(heading string) *TableWidget {
	w := &TableWidget{
		heading: heading,
		columns: make([]TableWidgetColumn, 0),
	}
	w.columnSpan = 3
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
