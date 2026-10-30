package tables

import (
	"fmt"
	"html"
	"reflect"
	"strings"
)

// ColumnRenderer is an internal interface for rendering typed column cells.
type ColumnRenderer[T any] interface {
	GetName() string
	FormatLabel() string
	RenderCell(record T, rawValue any) string
}

// Table represents a complete data table specification for model type T.
type Table[T any] struct {
	columns       []ColumnRenderer[T]
	filters       []FilterInterface
	actions       []*TableAction[T]
	bulkActions   []*BulkActionDef
	headerActions []*TableAction[T]
	records       []T
	totalCount    int
	page          int
	perPage       int
	searchQuery   string
	sortCol       string
	sortDir       string
	activeFilters map[string]string
	idFunc        func(record T) string
}

// New creates an empty Table instance.
func New[T any]() *Table[T] {
	return &Table[T]{
		columns:       make([]ColumnRenderer[T], 0),
		filters:       make([]FilterInterface, 0),
		actions:       make([]*TableAction[T], 0),
		bulkActions:   make([]*BulkActionDef, 0),
		headerActions: make([]*TableAction[T], 0),
		page:          1,
		perPage:       15,
		activeFilters: make(map[string]string),
	}
}

// Columns registers table columns.
func (t *Table[T]) Columns(cols ...ColumnRenderer[T]) *Table[T] {
	t.columns = cols
	return t
}

// Filters registers table filters.
func (t *Table[T]) Filters(filters ...FilterInterface) *Table[T] {
	t.filters = filters
	return t
}

// Actions registers row-level actions.
func (t *Table[T]) Actions(actions ...*TableAction[T]) *Table[T] {
	t.actions = actions
	return t
}

// BulkActions registers multi-record actions.
func (t *Table[T]) BulkActions(actions ...*BulkActionDef) *Table[T] {
	t.bulkActions = actions
	return t
}

// HeaderActions registers table header actions.
func (t *Table[T]) HeaderActions(actions ...*TableAction[T]) *Table[T] {
	t.headerActions = actions
	return t
}

// RecordID sets the closure to extract a unique string ID from record T.
func (t *Table[T]) RecordID(fn func(record T) string) *Table[T] {
	t.idFunc = fn
	return t
}

// SetData populates records and pagination state into the table.
func (t *Table[T]) SetData(records []T, total, page, perPage int, search, sortCol, sortDir string, filters map[string]string) *Table[T] {
	t.records = records
	t.totalCount = total
	t.page = page
	if t.page < 1 {
		t.page = 1
	}
	t.perPage = perPage
	if t.perPage < 1 {
		t.perPage = 15
	}
	t.searchQuery = search
	t.sortCol = sortCol
	t.sortDir = sortDir
	if filters != nil {
		t.activeFilters = filters
	}
	return t
}

// extractRecordID safely gets an ID string from a record.
func (t *Table[T]) extractRecordID(record T) string {
	if t.idFunc != nil {
		return t.idFunc(record)
	}

	val := reflect.ValueOf(record)
	if val.Kind() == reflect.Pointer && !val.IsNil() {
		val = val.Elem()
	}
	if val.Kind() == reflect.Struct {
		field := val.FieldByName("ID")
		if field.IsValid() {
			return fmt.Sprintf("%v", field.Interface())
		}
	}
	return ""
}

// extractFieldValue extracts a field by name using struct field matching.
func (t *Table[T]) extractFieldValue(record T, name string) any {
	val := reflect.ValueOf(record)
	if val.Kind() == reflect.Pointer && !val.IsNil() {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}

	// Match exact or title-cased name
	field := val.FieldByName(name)
	if !field.IsValid() {
		// Try TitleCase: "created_at" -> "CreatedAt"
		parts := strings.Split(name, "_")
		for i, p := range parts {
			if len(p) > 0 {
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
		}
		titleName := strings.Join(parts, "")
		field = val.FieldByName(titleName)
	}

	if field.IsValid() && field.CanInterface() {
		return field.Interface()
	}
	return nil
}

// Render renders the entire table HTML.
func (t *Table[T]) Render(basePath string) string {
	var sb strings.Builder

	// Top controls: Search, Filters, Header Actions
	sb.WriteString(`<div class="filament-card" style="padding: 1rem 1.25rem; margin-bottom: 1rem;">`)
	sb.WriteString(fmt.Sprintf(`<form method="GET" action="%s" style="display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 1rem;">`, basePath))

	// Search bar
	sb.WriteString(`<div style="display: flex; align-items: center; gap: 0.75rem; flex: 1; max-width: 24rem;">`)
	sb.WriteString(fmt.Sprintf(`<input type="text" name="search" value="%s" placeholder="Search records..." class="form-control" />`, html.EscapeString(t.searchQuery)))
	if t.searchQuery != "" {
		sb.WriteString(fmt.Sprintf(`<a href="%s" class="btn btn-secondary btn-sm" title="Clear search">&times;</a>`, basePath))
	}
	sb.WriteString(`</div>`)

	// Filters
	if len(t.filters) > 0 {
		sb.WriteString(`<div style="display: flex; align-items: center; gap: 0.75rem; flex-wrap: wrap;">`)
		for _, f := range t.filters {
			currentVal := t.activeFilters[f.GetName()]
			sb.WriteString(f.Render(currentVal))
		}
		sb.WriteString(`</div>`)
	}

	// Header Actions
	if len(t.headerActions) > 0 {
		sb.WriteString(`<div style="display: flex; gap: 0.5rem; align-items: center;">`)
		for _, ha := range t.headerActions {
			sb.WriteString(fmt.Sprintf(`<a href="%s/%s" class="btn btn-primary">%s</a>`, basePath, ha.name, html.EscapeString(ha.label)))
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`</form></div>`)

	// Bulk Action Form Wrapper
	sb.WriteString(fmt.Sprintf(`<form method="POST" action="%s/bulk">`, basePath))

	// Bulk action bar (if bulk actions configured)
	if len(t.bulkActions) > 0 {
		sb.WriteString(`<div style="display: flex; align-items: center; justify-content: space-between; padding: 0.75rem 1rem; background-color: var(--gray-50); border: 1px solid var(--gray-200); border-bottom: none; border-radius: 0.75rem 0.75rem 0 0;">`)
		sb.WriteString(`<span style="font-size: 0.8125rem; color: var(--gray-500);">Select rows to perform bulk operations</span>`)
		sb.WriteString(`<div style="display: flex; gap: 0.5rem;">`)
		for _, ba := range t.bulkActions {
			sb.WriteString(ba.Render())
		}
		sb.WriteString(`</div></div>`)
	}

	// Table Container
	sb.WriteString(`<div class="filament-table-wrapper">`)
	sb.WriteString(`<table class="filament-table">`)
	sb.WriteString(`<thead><tr>`)

	// Bulk checkbox header
	if len(t.bulkActions) > 0 {
		sb.WriteString(`<th style="width: 2.5rem; text-align: center;"><input type="checkbox" onclick="document.querySelectorAll('.row-select-checkbox').forEach(c => c.checked = this.checked)" /></th>`)
	}

	// Columns Header
	for _, col := range t.columns {
		sortLink := fmt.Sprintf("%s?sort=%s&direction=asc", basePath, col.GetName())
		if t.sortCol == col.GetName() && t.sortDir == "asc" {
			sortLink = fmt.Sprintf("%s?sort=%s&direction=desc", basePath, col.GetName())
		}
		arrow := ""
		if t.sortCol == col.GetName() {
			if t.sortDir == "asc" {
				arrow = " ↑"
			} else {
				arrow = " ↓"
			}
		}
		sb.WriteString(fmt.Sprintf(`<th><a href="%s" style="color: inherit; text-decoration: none;">%s%s</a></th>`,
			sortLink, html.EscapeString(col.FormatLabel()), arrow))
	}

	// Actions column header
	if len(t.actions) > 0 {
		sb.WriteString(`<th style="text-align: right;">Actions</th>`)
	}
	sb.WriteString(`</tr></thead><tbody>`)

	if len(t.records) == 0 {
		colspan := len(t.columns)
		if len(t.bulkActions) > 0 {
			colspan++
		}
		if len(t.actions) > 0 {
			colspan++
		}
		sb.WriteString(fmt.Sprintf(`<tr><td colspan="%d" style="text-align: center; padding: 3rem 1rem; color: var(--gray-400);">No records found.</td></tr>`, colspan))
	} else {
		for _, rec := range t.records {
			recID := t.extractRecordID(rec)
			sb.WriteString(`<tr>`)
			if len(t.bulkActions) > 0 {
				sb.WriteString(fmt.Sprintf(`<td style="text-align: center;"><input type="checkbox" name="selected_ids[]" value="%s" class="row-select-checkbox" /></td>`, recID))
			}
			for _, col := range t.columns {
				raw := t.extractFieldValue(rec, col.GetName())
				sb.WriteString(fmt.Sprintf(`<td>%s</td>`, col.RenderCell(rec, raw)))
			}
			if len(t.actions) > 0 {
				sb.WriteString(`<td style="text-align: right;"><div style="display: inline-flex; gap: 0.375rem;">`)
				for _, act := range t.actions {
					if act.IsVisible(rec) {
						sb.WriteString(act.Render(rec, recID, basePath))
					}
				}
				sb.WriteString(`</div></td>`)
			}
			sb.WriteString(`</tr>`)
		}
	}

	sb.WriteString(`</tbody></table></div>`)
	sb.WriteString(`</form>`)

	// Pagination
	totalPages := (t.totalCount + t.perPage - 1) / t.perPage
	if totalPages < 1 {
		totalPages = 1
	}

	sb.WriteString(`<div style="display: flex; align-items: center; justify-content: space-between; margin-top: 1rem; font-size: 0.875rem;">`)
	sb.WriteString(fmt.Sprintf(`<span style="color: var(--gray-500);">Showing page %d of %d (%d total records)</span>`, t.page, totalPages, t.totalCount))

	if totalPages > 1 {
		sb.WriteString(`<div style="display: flex; gap: 0.375rem;">`)
		if t.page > 1 {
			sb.WriteString(fmt.Sprintf(`<a href="%s?page=%d" class="btn btn-secondary btn-sm">Previous</a>`, basePath, t.page-1))
		}
		if t.page < totalPages {
			sb.WriteString(fmt.Sprintf(`<a href="%s?page=%d" class="btn btn-secondary btn-sm">Next</a>`, basePath, t.page+1))
		}
		sb.WriteString(`</div>`)
	}
	sb.WriteString(`</div>`)

	return sb.String()
}
