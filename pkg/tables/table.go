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

// CellContextRenderer is an optional interface for columns requiring record ID and basePath context.
type CellContextRenderer[T any] interface {
	RenderCellWithContext(record T, recordID, basePath string, rawValue any) string
}

// ColumnToggleable is an optional interface for columns that can be shown or hidden.
type ColumnToggleable interface {
	IsToggleable() bool
	IsToggledHiddenByDefault() bool
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
	toggleable    bool
	reorderable   bool
	reorderCol    string
	softDeletes   bool
	isTrashed     func(record T) bool
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

// Toggleable enables or disables the column show/hide toggler.
func (t *Table[T]) Toggleable(enabled ...bool) *Table[T] {
	t.toggleable = len(enabled) == 0 || enabled[0]
	return t
}

// IsToggleable returns whether column toggling is enabled.
func (t *Table[T]) IsToggleable() bool {
	return t.toggleable
}

// Reorderable enables drag-and-drop row reordering.
func (t *Table[T]) Reorderable(col ...string) *Table[T] {
	t.reorderable = true
	if len(col) > 0 {
		t.reorderCol = col[0]
	} else {
		t.reorderCol = "sort_order"
	}
	return t
}

// IsReorderable returns whether row reordering is enabled.
func (t *Table[T]) IsReorderable() bool {
	return t.reorderable
}

// GetReorderColumn returns the sort order column name.
func (t *Table[T]) GetReorderColumn() string {
	return t.reorderCol
}

// SoftDeletes enables soft-delete awareness and trashed row presentation.
func (t *Table[T]) SoftDeletes(enabled ...bool) *Table[T] {
	t.softDeletes = len(enabled) == 0 || enabled[0]
	return t
}

// IsTrashed sets the closure to identify if a record is soft-deleted.
func (t *Table[T]) IsTrashed(fn func(record T) bool) *Table[T] {
	t.isTrashed = fn
	return t
}

// IsSoftDeletesEnabled returns true if soft-deletes are active on this table.
func (t *Table[T]) IsSoftDeletesEnabled() bool {
	return t.softDeletes
}

// HasFilter checks if a filter with the given name is registered.
func (t *Table[T]) HasFilter(name string) bool {
	for _, f := range t.filters {
		if f.GetName() == name {
			return true
		}
	}
	return false
}

// ResolveFilters combines explicitly registered filters with auto-generated column filters.
func (t *Table[T]) ResolveFilters() []FilterInterface {
	allFilters := make([]FilterInterface, 0, len(t.filters))
	existing := make(map[string]bool)

	// Add explicit filters first
	for _, f := range t.filters {
		allFilters = append(allFilters, f)
		existing[f.GetName()] = true
		if f.GetColumn() != "" {
			existing[f.GetColumn()] = true
		}
	}

	// Auto-create filters for columns marked Filterable() or RefinedFilter()
	for _, col := range t.columns {
		if c, ok := any(col).(interface {
			GetName() string
			GetLabel() string
			IsFilterable() bool
			IsRefinedFilter() bool
			GetFilterOptions() map[string]string
		}); ok {
			colName := c.GetName()
			if c.IsFilterable() && !existing[colName] {
				label := c.GetLabel()
				if c.IsRefinedFilter() {
					rf := RefinedFilter(colName).Label(label).Column(colName)
					if opts := c.GetFilterOptions(); len(opts) > 0 {
						rf.Options(opts)
					}
					allFilters = append(allFilters, rf)
				} else {
					sf := SelectFilter(colName).Label(label).Column(colName)
					if opts := c.GetFilterOptions(); len(opts) > 0 {
						sf.Options(opts)
					}
					allFilters = append(allFilters, sf)
				}
				existing[colName] = true
			}
		}
	}

	return allFilters
}

// GetFilters returns registered and column-resolved filters.
func (t *Table[T]) GetFilters() []FilterInterface {
	return t.ResolveFilters()
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

// GetColumns returns the configured columns.
func (t *Table[T]) GetColumns() []ColumnRenderer[T] {
	return t.columns
}

// GetHeaderActions returns the configured header actions.
func (t *Table[T]) GetHeaderActions() []*TableAction[T] {
	return t.headerActions
}

// GetActions returns the row-level actions.
func (t *Table[T]) GetActions() []*TableAction[T] {
	return t.actions
}

// GetBulkActions returns the bulk actions.
func (t *Table[T]) GetBulkActions() []*BulkActionDef {
	return t.bulkActions
}

// ExtractFieldValue extracts a field value by name using reflection or closures.
func (t *Table[T]) ExtractFieldValue(record T, name string) any {
	return t.extractFieldValue(record, name)
}

// ExtractRecordID returns the record ID.
func (t *Table[T]) ExtractRecordID(record T) string {
	return t.extractRecordID(record)
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

	storageKey := fmt.Sprintf("filament_cols_%s", strings.ReplaceAll(basePath, "/", "_"))

	// Setup Alpine x-data for column toggling
	alpineData := ""
	if t.toggleable {
		var colPairs []string
		for _, col := range t.columns {
			hidden := false
			if ct, ok := col.(ColumnToggleable); ok && ct.IsToggledHiddenByDefault() {
				hidden = true
			}
			colPairs = append(colPairs, fmt.Sprintf(`"%s": %t`, col.GetName(), !hidden))
		}
		initialJSON := fmt.Sprintf("{%s}", strings.Join(colPairs, ", "))
		alpineData = fmt.Sprintf(`x-data="{
  visibleCols: %s,
  toggleCol(name) {
    this.visibleCols[name] = !this.visibleCols[name];
    try { localStorage.setItem('%s', JSON.stringify(this.visibleCols)); } catch(e){}
  },
  init() {
    try {
      const saved = localStorage.getItem('%s');
      if (saved) { Object.assign(this.visibleCols, JSON.parse(saved)); }
    } catch(e){}
  }
}"`, initialJSON, storageKey, storageKey)
	}

	sb.WriteString(fmt.Sprintf(`<div class="filament-table-root" %s>`, alpineData))

	// Top controls: Search, Filters, Column Toggle, Header Actions
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

	// Right toolbar controls: Column Toggler & Header Actions
	sb.WriteString(`<div style="display: flex; gap: 0.5rem; align-items: center;">`)

	// Column Toggler Popover
	if t.toggleable {
		sb.WriteString(`<div class="filament-dropdown-wrapper" style="position: relative;" x-data="{ open: false }">`)
		sb.WriteString(`<button type="button" @click="open = !open" class="btn btn-secondary btn-sm" style="display: inline-flex; align-items: center; gap: 0.375rem;">`)
		sb.WriteString(`<svg style="width: 1rem; height: 1rem;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7m0 10a2 2 0 002 2h2a2 2 0 002-2V7a2 2 0 00-2-2h-2a2 2 0 00-2 2" /></svg>`)
		sb.WriteString(`<span>Columns</span></button>`)
		sb.WriteString(`<div x-show="open" @click.outside="open = false" style="display: none; position: absolute; right: 0; margin-top: 0.375rem; width: 14rem; background: white; border: 1px solid var(--gray-200); border-radius: 0.5rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1); padding: 0.5rem; z-index: 50;" :style="open ? 'display: block;' : 'display: none;'">`)
		sb.WriteString(`<div style="font-size: 0.75rem; font-weight: 600; color: var(--gray-500); padding: 0.25rem 0.5rem; text-transform: uppercase;">Toggle Columns</div>`)
		for _, col := range t.columns {
			isTogg := true
			if ct, ok := col.(ColumnToggleable); ok && !ct.IsToggleable() {
				isTogg = false
			}
			if isTogg {
				sb.WriteString(fmt.Sprintf(`
<label style="display: flex; align-items: center; gap: 0.5rem; padding: 0.375rem 0.5rem; font-size: 0.8125rem; cursor: pointer; border-radius: 0.25rem;" onmouseover="this.style.background='var(--gray-50)'" onmouseout="this.style.background='transparent'">
  <input type="checkbox" :checked="visibleCols['%s']" @change="toggleCol('%s')" style="accent-color: var(--primary-600);" />
  <span>%s</span>
</label>`, col.GetName(), col.GetName(), html.EscapeString(col.FormatLabel())))
			}
		}
		sb.WriteString(`</div></div>`)
	}

	// Header Actions
	if len(t.headerActions) > 0 {
		for _, ha := range t.headerActions {
			btnClass := "btn-primary"
			if ha.color == "secondary" {
				btnClass = "btn-secondary"
			} else if ha.color == "danger" {
				btnClass = "btn-danger"
			}
			sb.WriteString(fmt.Sprintf(`<a href="%s/%s" class="btn %s">%s</a>`, basePath, ha.name, btnClass, html.EscapeString(ha.label)))
		}
	}
	sb.WriteString(`</div>`)

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

	// Reorder grip header
	if t.reorderable {
		sb.WriteString(`<th style="width: 2.5rem; text-align: center;" title="Drag to reorder">&#8942;&#8942;</th>`)
	}

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
		colStyle := ""
		if t.toggleable {
			colStyle = fmt.Sprintf(`:style="!visibleCols['%s'] ? 'display: none;' : ''"`, col.GetName())
		}
		sb.WriteString(fmt.Sprintf(`<th data-col="%s" %s><a href="%s" style="color: inherit; text-decoration: none;">%s%s</a></th>`,
			col.GetName(), colStyle, sortLink, html.EscapeString(col.FormatLabel()), arrow))
	}

	// Actions column header
	if len(t.actions) > 0 || t.softDeletes {
		sb.WriteString(`<th style="text-align: right;">Actions</th>`)
	}
	sb.WriteString(`</tr></thead><tbody>`)

	if len(t.records) == 0 {
		colspan := len(t.columns)
		if len(t.bulkActions) > 0 {
			colspan++
		}
		if t.reorderable {
			colspan++
		}
		if len(t.actions) > 0 || t.softDeletes {
			colspan++
		}
		sb.WriteString(fmt.Sprintf(`<tr class="empty-row"><td colspan="%d" style="text-align: center; padding: 3rem 1rem; color: var(--gray-400);">No records found.</td></tr>`, colspan))
	} else {
		for _, rec := range t.records {
			recID := t.extractRecordID(rec)
			isRecTrashed := t.softDeletes && t.isTrashed != nil && t.isTrashed(rec)
			trAttrs := ""
			if t.reorderable {
				trAttrs = fmt.Sprintf(`data-id="%s" class="table-row-draggable%s" draggable="true" ondragstart="event.dataTransfer.setData('text/plain', this.dataset.id); this.classList.add('table-row-dragging');" ondragend="this.classList.remove('table-row-dragging');" ondragover="event.preventDefault(); const dragging = document.querySelector('.table-row-draggable.table-row-dragging'); if (dragging && dragging !== this) { const rect = this.getBoundingClientRect(); const next = (event.clientY - rect.top)/(rect.bottom - rect.top) > 0.5; this.parentNode.insertBefore(dragging, next ? this.nextSibling : this); }" ondrop="event.preventDefault(); Filament.reorderRows('%s/reorder', this.closest('table'));"`, recID, func() string { if isRecTrashed { return " table-row-trashed" }; return "" }(), basePath)
			} else {
				trAttrs = fmt.Sprintf(`data-id="%s"%s`, recID, func() string { if isRecTrashed { return ` class="table-row-trashed" style="background-color: rgba(254, 242, 242, 0.4);"` }; return "" }())
			}

			sb.WriteString(fmt.Sprintf(`<tr %s>`, trAttrs))

			// Reorder handle cell
			if t.reorderable {
				sb.WriteString(`<td class="drag-handle" data-label="Reorder" style="width: 2.5rem; text-align: center; cursor: grab; color: var(--gray-400); user-select: none;">&#8942;&#8942;</td>`)
			}

			// Bulk checkbox
			if len(t.bulkActions) > 0 {
				sb.WriteString(fmt.Sprintf(`<td class="table-cell-checkbox" data-label="Select" style="text-align: center;"><label class="table-checkbox-label"><input type="checkbox" name="selected_ids[]" value="%s" class="row-select-checkbox" /><span class="table-checkbox-text">Select</span></label></td>`, recID))
			}

			// Render columns
			for _, col := range t.columns {
				raw := t.extractFieldValue(rec, col.GetName())
				colStyle := ""
				if t.toggleable {
					colStyle = fmt.Sprintf(`:style="!visibleCols['%s'] ? 'display: none;' : ''"`, col.GetName())
				}

				cellHTML := ""
				if ccr, ok := col.(CellContextRenderer[T]); ok {
					cellHTML = ccr.RenderCellWithContext(rec, recID, basePath, raw)
				} else {
					cellHTML = col.RenderCell(rec, raw)
				}

				sb.WriteString(fmt.Sprintf(`<td data-col="%s" data-label="%s" %s><span class="table-cell-value">%s</span></td>`, col.GetName(), html.EscapeString(col.FormatLabel()), colStyle, cellHTML))
			}

			if len(t.actions) > 0 || isRecTrashed {
				sb.WriteString(`<td class="table-cell-actions" data-label="Actions" style="text-align: right;"><div style="display: inline-flex; gap: 0.375rem; align-items: center;">`)
				if isRecTrashed {
					sb.WriteString(`<span class="badge badge-danger" style="font-size: 0.6875rem; margin-right: 0.25rem;">Trashed</span>`)
					restoreAct := RestoreAction[T]()
					forceDelAct := ForceDeleteAction[T]()
					sb.WriteString(restoreAct.Render(rec, recID, basePath))
					sb.WriteString(forceDelAct.Render(rec, recID, basePath))
				} else {
					for _, act := range t.actions {
						if act.IsVisible(rec) {
							sb.WriteString(act.Render(rec, recID, basePath))
						}
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

	sb.WriteString(`</div>`) // Close filament-table-root

	return sb.String()
}
