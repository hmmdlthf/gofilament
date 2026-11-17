package filament

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/globalsearch"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/notifications"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/cache"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func stripHTML(input string) string {
	cleaned := htmlTagRegex.ReplaceAllString(input, "")
	return strings.TrimSpace(html.UnescapeString(cleaned))
}

func normalizeHeader(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ImportResult captures the outcome of a batch or CSV import operation.
type ImportResult struct {
	SuccessCount int      `json:"imported"`
	FailedCount  int      `json:"failed"`
	Errors       []string `json:"errors,omitempty"`
	// Rows lists the rows that were not imported (at most 200).
	Rows []ImportRowError `json:"rows,omitempty"`
}

// ListHandlerFunc is the callback to query records for the data table.
type ListHandlerFunc[T any] func(ctx *Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]T, int, error)

// FindHandlerFunc is the callback to find a single record by ID.
type FindHandlerFunc[T any] func(ctx *Context, id string) (T, error)

// CreateHandlerFunc is the callback to insert a newly created record.
type CreateHandlerFunc func(ctx *Context, data map[string]any) error

// UpdateHandlerFunc is the callback to persist modifications on an existing record.
type UpdateHandlerFunc func(ctx *Context, id string, data map[string]any) error

// DeleteHandlerFunc is the callback to delete a single record.
type DeleteHandlerFunc func(ctx *Context, id string) error

// DeleteBulkHandlerFunc is the callback to delete multiple records.
type DeleteBulkHandlerFunc func(ctx *Context, ids []string) error

// RestoreHandlerFunc is the callback to restore a soft-deleted record.
type RestoreHandlerFunc func(ctx *Context, id string) error

// ForceDeleteHandlerFunc is the callback to permanently purge a soft-deleted record.
type ForceDeleteHandlerFunc func(ctx *Context, id string) error

// RestoreBulkHandlerFunc is the callback to restore multiple soft-deleted records.
type RestoreBulkHandlerFunc func(ctx *Context, ids []string) error

// ForceDeleteBulkHandlerFunc is the callback to permanently purge multiple soft-deleted records.
type ForceDeleteBulkHandlerFunc func(ctx *Context, ids []string) error

// ReorderHandlerFunc is the callback to persist a new order of record IDs.
type ReorderHandlerFunc func(ctx *Context, orderedIDs []string) error

// CellUpdateHandlerFunc is the callback to update a single cell on a record.
type CellUpdateHandlerFunc func(ctx *Context, id, column string, value any) error

// BaseResource implements ResourceInterface with generic Ent/model type T.
type BaseResource[T any] struct {
	summariesUsing            func(ctx *Context, search string, filters map[string]string) (tables.Summaries, error)
	versionOf                 func(record T) string
	slug                      string
	modelName                 string
	pluralModelName           string
	navGroup                  string
	navIcon                   string
	navSort                   int
	navBadge                  func(ctx *Context) string
	formBuilder               func(form *forms.Form) *forms.Form
	tableBuilder              func(table *tables.Table[T]) *tables.Table[T]
	defaultColumns            []string
	selectableColumns         []string
	advancedSearchableColumns []string
	infolistBuilder           func(infolist *infolists.Infolist) *infolists.Infolist
	idExtractor               func(record T) string
	relations                 []RelationManager

	// Handlers
	onList            ListHandlerFunc[T]
	onFind            FindHandlerFunc[T]
	onCreate          CreateHandlerFunc
	onUpdate          UpdateHandlerFunc
	onDelete          DeleteHandlerFunc
	onDeleteBulk      DeleteBulkHandlerFunc
	kanban            *kanbanBoard
	bulkEditFields    []string
	onBulkUpdate      BulkUpdateHandlerFunc
	historyProvider   HistoryProvider
	canSeeHistory     func(ctx *Context) bool
	onRestore         RestoreHandlerFunc
	onForceDelete     ForceDeleteHandlerFunc
	onRestoreBulk     RestoreBulkHandlerFunc
	onForceDeleteBulk ForceDeleteBulkHandlerFunc
	onReorder         ReorderHandlerFunc
	onCellUpdate      CellUpdateHandlerFunc

	// Soft Deletes
	softDeletes      bool
	softDeleteColumn string
	isRecordTrashed  func(record T) bool

	// Authorization
	canViewAny func(ctx *Context) bool
	canView    func(ctx *Context, record T) bool
	// acceptUnvalidated skips server-side form validation (AcceptUnvalidatedInput).
	acceptUnvalidated bool
	canCreate         func(ctx *Context) bool
	canEdit           func(ctx *Context, record T) bool
	canDelete         func(ctx *Context, record T) bool
	canDeleteAny      func(ctx *Context) bool
	canRestore        func(ctx *Context, id string) bool
	canForceDelete    func(ctx *Context, id string) bool

	// Global Search
	globallySearchable  bool
	searchAttributes    []string
	globalResultFunc    func(record T) GlobalSearchResult
	globalResultTitle   func(record T) string
	globalResultDetails func(record T) map[string]string
	globalResultActions func(record T) []GlobalSearchAction
	globalResultURL     func(record T) string
	globalSearchLimit   int

	// Resource Tabs (Status / Filter Segments)
	tabs        []*ResourceTab
	tabsEnabled *bool

	// Pagination & Infinite Scroll
	listMode                 ListMode
	perPage                  int
	perPageOptions           []int
	mobilePaginationPosition PaginationPosition
}

// ResourceTab represents a status or query filter tab displayed above the resource records list.
type ResourceTab struct {
	Key   string            `json:"key"`
	Label string            `json:"label"`
	Badge string            `json:"badge,omitempty"`
	Icon  string            `json:"icon,omitempty"`
	Query map[string]string `json:"query,omitempty"`
}

// Tab creates a new ResourceTab with the given label.
func Tab(label string) *ResourceTab {
	cleanKey := strings.ToLower(strings.TrimSpace(label))
	cleanKey = strings.ReplaceAll(cleanKey, " ", "_")
	cleanKey = strings.ReplaceAll(cleanKey, "-", "_")
	return &ResourceTab{
		Key:   cleanKey,
		Label: label,
		Query: make(map[string]string),
	}
}

// Key sets the unique key for the resource tab.
func (t *ResourceTab) KeyName(k string) *ResourceTab {
	t.Key = k
	return t
}

// Badge sets an optional badge label on the tab.
func (t *ResourceTab) TabBadge(b string) *ResourceTab {
	t.Badge = b
	return t
}

// Icon sets an optional icon name on the tab.
func (t *ResourceTab) TabIcon(icon string) *ResourceTab {
	t.Icon = icon
	return t
}

// QueryParam adds a query filter parameter when this tab is selected.
func (t *ResourceTab) QueryParam(k, v string) *ResourceTab {
	if t.Query == nil {
		t.Query = make(map[string]string)
	}
	t.Query[k] = v
	return t
}

// NewResource creates a new Resource with a URL slug.
func NewResource[T any](slug string) *BaseResource[T] {
	return &BaseResource[T]{
		slug:               strings.Trim(slug, "/"),
		modelName:          strings.Title(slug),
		pluralModelName:    strings.Title(slug),
		navIcon:            "heroicon-rectangle-stack",
		relations:          make([]RelationManager, 0),
		globallySearchable: true,
		listMode:           ListModePagination,
		perPage:            15,
		perPageOptions:     []int{10, 15, 25, 50, 100},
	}
}

// Pagination configures the resource for paginated list display with an optional default items-per-page.
func (r *BaseResource[T]) Pagination(perPage ...int) *BaseResource[T] {
	r.listMode = ListModePagination
	if len(perPage) > 0 && perPage[0] > 0 {
		r.perPage = perPage[0]
	}
	return r
}

// InfiniteScroll configures the resource for infinite scrolling list display with an optional chunk size.
func (r *BaseResource[T]) InfiniteScroll(perPage ...int) *BaseResource[T] {
	r.listMode = ListModeInfiniteScroll
	if len(perPage) > 0 && perPage[0] > 0 {
		r.perPage = perPage[0]
	}
	return r
}

// ListMode sets the list loading mode explicitly (ListModePagination or ListModeInfiniteScroll).
func (r *BaseResource[T]) ListMode(mode ListMode) *BaseResource[T] {
	r.listMode = mode
	return r
}

// PerPage sets the default records per page / chunk size.
func (r *BaseResource[T]) PerPage(n int) *BaseResource[T] {
	if n > 0 {
		r.perPage = n
	}
	return r
}

// PerPageOptions sets the selectable items-per-page choices.
func (r *BaseResource[T]) PerPageOptions(opts ...int) *BaseResource[T] {
	if len(opts) > 0 {
		r.perPageOptions = opts
	}
	return r
}

// GetListMode returns the configured ListMode.
func (r *BaseResource[T]) GetListMode() ListMode {
	if r.listMode == "" {
		return ListModePagination
	}
	return r.listMode
}

// GetPerPage returns the configured items per page.
func (r *BaseResource[T]) GetPerPage() int {
	if r.perPage <= 0 {
		return 15
	}
	return r.perPage
}

// GetPerPageOptions returns the configured per page choices.
func (r *BaseResource[T]) GetPerPageOptions() []int {
	if len(r.perPageOptions) == 0 {
		return []int{10, 15, 25, 50, 100}
	}
	return r.perPageOptions
}

// MobilePaginationPosition sets the location of pagination controls on mobile / cards view (PaginationPositionTop or PaginationPositionFAB).
func (r *BaseResource[T]) MobilePaginationPosition(pos PaginationPosition) *BaseResource[T] {
	r.mobilePaginationPosition = pos
	return r
}

// GetMobilePaginationPosition returns the mobile pagination position, defaulting to PaginationPositionTop ("top").
func (r *BaseResource[T]) GetMobilePaginationPosition() PaginationPosition {
	if r.mobilePaginationPosition == "" {
		return PaginationPositionTop
	}
	return r.mobilePaginationPosition
}

func (r *BaseResource[T]) GloballySearchable(enabled ...bool) *BaseResource[T] {
	if len(enabled) > 0 {
		r.globallySearchable = enabled[0]
	} else {
		r.globallySearchable = true
	}
	return r
}

func (r *BaseResource[T]) IsGloballySearchable() bool {
	return r.globallySearchable
}

func (r *BaseResource[T]) GlobalSearchAttributes(attrs ...string) *BaseResource[T] {
	r.searchAttributes = attrs
	return r
}

func (r *BaseResource[T]) GlobalSearchResult(fn func(record T) GlobalSearchResult) *BaseResource[T] {
	r.globalResultFunc = fn
	return r
}

func (r *BaseResource[T]) GlobalSearchResultTitle(fn func(record T) string) *BaseResource[T] {
	r.globalResultTitle = fn
	return r
}

func (r *BaseResource[T]) GlobalSearchResultDetails(fn func(record T) map[string]string) *BaseResource[T] {
	r.globalResultDetails = fn
	return r
}

func (r *BaseResource[T]) GlobalSearchResultActions(fn func(record T) []GlobalSearchAction) *BaseResource[T] {
	r.globalResultActions = fn
	return r
}

func (r *BaseResource[T]) GlobalSearchResultURL(fn func(record T) string) *BaseResource[T] {
	r.globalResultURL = fn
	return r
}

func (r *BaseResource[T]) GlobalSearchResultsLimit(limit int) *BaseResource[T] {
	r.globalSearchLimit = limit
	return r
}

func (r *BaseResource[T]) SearchGlobal(ctx *Context, query string) []GlobalSearchResult {
	if !r.globallySearchable || r.onList == nil {
		return nil
	}
	// Search shows record titles and details: apply the same access
	// checks as the list and view pages.
	if r.canViewAny != nil && !r.canViewAny(ctx) {
		return nil
	}

	limit := 10
	if r.globalSearchLimit > 0 {
		limit = r.globalSearchLimit
	} else if ctx != nil && ctx.Panel() != nil && ctx.Panel().GetGlobalSearchResultsLimit() > 0 {
		limit = ctx.Panel().GetGlobalSearchResultsLimit()
	}

	records, _, err := r.onList(ctx, query, "", "asc", 1, limit, nil)
	if err != nil || len(records) == 0 {
		return nil
	}

	var results []GlobalSearchResult
	basePath := ""
	if ctx != nil && ctx.Panel() != nil {
		basePath = fmt.Sprintf("%s/%s", ctx.Panel().GetPath(), r.slug)
	} else {
		basePath = "/" + r.slug
	}

	for _, rec := range records {
		if r.canView != nil && !r.canView(ctx, rec) {
			continue
		}
		if r.globalResultFunc != nil {
			res := r.globalResultFunc(rec)
			if res.Group == "" {
				res.Group = r.pluralModelName
			}
			if res.Score == 0 {
				score, _ := globalsearch.Score(query, res.Title)
				res.Score = score
			}
			results = append(results, res)
			continue
		}

		recID := ""
		if r.idExtractor != nil {
			recID = r.idExtractor(rec)
		}

		title := fmt.Sprintf("%s #%s", r.modelName, recID)
		if r.globalResultTitle != nil {
			title = r.globalResultTitle(rec)
		}

		targetURL := fmt.Sprintf("%s/%s", basePath, recID)
		if r.globalResultURL != nil {
			targetURL = r.globalResultURL(rec)
		}

		var details map[string]string
		if r.globalResultDetails != nil {
			details = r.globalResultDetails(rec)
		}

		var actions []GlobalSearchAction
		if r.globalResultActions != nil {
			actions = r.globalResultActions(rec)
		}

		score, _ := globalsearch.Score(query, title)
		if len(details) > 0 {
			for _, v := range details {
				if ds, dMatch := globalsearch.Score(query, v); dMatch && ds > score {
					score = ds
				}
			}
		}

		results = append(results, GlobalSearchResult{
			Title:   title,
			URL:     targetURL,
			Details: details,
			Group:   r.pluralModelName,
			Icon:    r.navIcon,
			Actions: actions,
			Score:   score,
		})
	}

	return results
}

// Authorization methods
func (r *BaseResource[T]) CanViewAny(fn func(ctx *Context) bool) *BaseResource[T] {
	r.canViewAny = fn
	return r
}

func (r *BaseResource[T]) CanView(fn func(ctx *Context, record T) bool) *BaseResource[T] {
	r.canView = fn
	return r
}

func (r *BaseResource[T]) CanCreate(fn func(ctx *Context) bool) *BaseResource[T] {
	r.canCreate = fn
	return r
}

func (r *BaseResource[T]) CanEdit(fn func(ctx *Context, record T) bool) *BaseResource[T] {
	r.canEdit = fn
	return r
}

func (r *BaseResource[T]) CanDelete(fn func(ctx *Context, record T) bool) *BaseResource[T] {
	r.canDelete = fn
	return r
}

func (r *BaseResource[T]) CanDeleteAny(fn func(ctx *Context) bool) *BaseResource[T] {
	r.canDeleteAny = fn
	return r
}

// Policy attaches a comprehensive Policy[T] defining rules for all CRUD actions.
func (r *BaseResource[T]) Policy(p Policy[T]) *BaseResource[T] {
	if p == nil {
		return r
	}
	r.canViewAny = func(ctx *Context) bool { return p.ViewAny(ctx) }
	r.canView = func(ctx *Context, record T) bool { return p.View(ctx, record) }
	r.canCreate = func(ctx *Context) bool { return p.Create(ctx) }
	r.canEdit = func(ctx *Context, record T) bool { return p.Update(ctx, record) }
	r.canDelete = func(ctx *Context, record T) bool { return p.Delete(ctx, record) }
	r.canDeleteAny = func(ctx *Context) bool { return p.DeleteAny(ctx) }
	return r
}

// AuthorizeWithGate binds authorization checks to GoLaravel's global auth.Gate naming conventions.
// Convention: viewAny_{name}, view_{name}, create_{name}, update_{name}, delete_{name}, deleteAny_{name}.
func (r *BaseResource[T]) AuthorizeWithGate(resourcePrefix ...string) *BaseResource[T] {
	name := r.slug
	if len(resourcePrefix) > 0 && resourcePrefix[0] != "" {
		name = resourcePrefix[0]
	}
	r.canViewAny = func(ctx *Context) bool {
		u := ctx.User()
		return u != nil && auth.Allows("viewAny_"+name, u)
	}
	r.canView = func(ctx *Context, record T) bool {
		u := ctx.User()
		return u != nil && auth.Allows("view_"+name, u, record)
	}
	r.canCreate = func(ctx *Context) bool {
		u := ctx.User()
		return u != nil && auth.Allows("create_"+name, u)
	}
	r.canEdit = func(ctx *Context, record T) bool {
		u := ctx.User()
		return u != nil && auth.Allows("update_"+name, u, record)
	}
	r.canDelete = func(ctx *Context, record T) bool {
		u := ctx.User()
		return u != nil && auth.Allows("delete_"+name, u, record)
	}
	r.canDeleteAny = func(ctx *Context) bool {
		u := ctx.User()
		return u != nil && auth.Allows("deleteAny_"+name, u)
	}
	return r
}

func (r *BaseResource[T]) ID() string {
	return r.slug
}

func (r *BaseResource[T]) Slug() string {
	return r.slug
}

func (r *BaseResource[T]) ModelName(name ...string) *BaseResource[T] {
	if len(name) > 0 {
		r.modelName = name[0]
		return r
	}
	return r
}

func (r *BaseResource[T]) GetModelName() string {
	return r.modelName
}

func (r *BaseResource[T]) PluralModelName(name ...string) *BaseResource[T] {
	if len(name) > 0 {
		r.pluralModelName = name[0]
		return r
	}
	return r
}

// CanViewAnyRecords reports whether the user may list the resource.
func (r *BaseResource[T]) CanViewAnyRecords(ctx *Context) bool {
	return r.canViewAny == nil || r.canViewAny(ctx)
}

func (r *BaseResource[T]) GetPluralModelName() string {
	return r.pluralModelName
}

func (r *BaseResource[T]) NavigationGroup(group string) *BaseResource[T] {
	r.navGroup = group
	return r
}

func (r *BaseResource[T]) NavigationIcon(icon string) *BaseResource[T] {
	r.navIcon = icon
	return r
}

func (r *BaseResource[T]) NavigationSort(sort int) *BaseResource[T] {
	r.navSort = sort
	return r
}

func (r *BaseResource[T]) NavigationBadge(fn func(ctx *Context) string) *BaseResource[T] {
	r.navBadge = fn
	return r
}

func (r *BaseResource[T]) Form(fn func(form *forms.Form) *forms.Form) *BaseResource[T] {
	r.formBuilder = fn
	return r
}

func (r *BaseResource[T]) Table(fn func(table *tables.Table[T]) *tables.Table[T]) *BaseResource[T] {
	r.tableBuilder = fn
	return r
}

func (r *BaseResource[T]) Infolist(fn func(infolist *infolists.Infolist) *infolists.Infolist) *BaseResource[T] {
	r.infolistBuilder = fn
	return r
}

func (r *BaseResource[T]) RecordID(fn func(record T) string) *BaseResource[T] {
	r.idExtractor = fn
	return r
}

func (r *BaseResource[T]) Relations(rms ...RelationManager) *BaseResource[T] {
	r.relations = append(r.relations, rms...)
	return r
}

func (r *BaseResource[T]) GetRelations() []RelationManager {
	return r.relations
}

// Tabs defines explicit status/filter tabs displayed above the records list.
func (r *BaseResource[T]) Tabs(tabs ...*ResourceTab) *BaseResource[T] {
	r.tabs = tabs
	b := true
	r.tabsEnabled = &b
	return r
}

// EnableTabs explicitly enables or disables status tabs for this resource.
func (r *BaseResource[T]) EnableTabs(enabled ...bool) *BaseResource[T] {
	b := true
	if len(enabled) > 0 {
		b = enabled[0]
	}
	r.tabsEnabled = &b
	return r
}

// GetTabs returns the configured resource tabs.
func (r *BaseResource[T]) GetTabs() []*ResourceTab {
	return r.tabs
}

// IsTabsEnabled returns true if tabs are enabled for this resource.
func (r *BaseResource[T]) IsTabsEnabled() bool {
	if r.tabsEnabled != nil {
		return *r.tabsEnabled
	}
	return false
}

func (r *BaseResource[T]) GetFilters() []tables.FilterDTO {
	if r.tableBuilder == nil {
		return nil
	}
	tbl := tables.New[T]()
	tbl = r.tableBuilder(tbl)
	withTrashedFilter(r, tbl)
	var dtos []tables.FilterDTO
	for _, f := range tbl.GetFilters() {
		dtos = append(dtos, f.ToDTO())
	}
	return dtos
}

// DefaultColumns configures which columns are shown by default for this resource.
func (r *BaseResource[T]) DefaultColumns(cols ...string) *BaseResource[T] {
	r.defaultColumns = cols
	return r
}

// SelectableColumns configures which columns are available for selection in the table UI.
func (r *BaseResource[T]) SelectableColumns(cols ...string) *BaseResource[T] {
	r.selectableColumns = cols
	return r
}

// AdvancedSearchableColumns configures which columns are enabled for user-driven advanced search.
func (r *BaseResource[T]) AdvancedSearchableColumns(cols ...string) *BaseResource[T] {
	r.advancedSearchableColumns = cols
	return r
}

// AdvancedSearch is a convenient alias for AdvancedSearchableColumns.
func (r *BaseResource[T]) AdvancedSearch(cols ...string) *BaseResource[T] {
	return r.AdvancedSearchableColumns(cols...)
}

// GetColumns returns the ColumnDTO metadata for this resource's table.
func (r *BaseResource[T]) GetColumns() []tables.ColumnDTO {
	if r.tableBuilder == nil {
		return nil
	}
	tbl := tables.New[T]()
	tbl = r.tableBuilder(tbl)
	if len(r.defaultColumns) > 0 {
		tbl.DefaultColumns(r.defaultColumns...)
	}
	if len(r.selectableColumns) > 0 {
		tbl.SelectableColumns(r.selectableColumns...)
	}
	if len(r.advancedSearchableColumns) > 0 {
		tbl.AdvancedSearchable(r.advancedSearchableColumns...)
	}
	return tbl.GetColumnDTOs()
}

// GetDefaultColumns returns the default column names for this resource.
func (r *BaseResource[T]) GetDefaultColumns() []string {
	if len(r.defaultColumns) > 0 {
		return r.defaultColumns
	}
	if r.tableBuilder == nil {
		return nil
	}
	tbl := tables.New[T]()
	tbl = r.tableBuilder(tbl)
	return tbl.GetDefaultColumns()
}

// GetSelectableColumns returns the selectable column names for this resource.
func (r *BaseResource[T]) GetSelectableColumns() []string {
	if len(r.selectableColumns) > 0 {
		return r.selectableColumns
	}
	if r.tableBuilder == nil {
		return nil
	}
	tbl := tables.New[T]()
	tbl = r.tableBuilder(tbl)
	return tbl.GetSelectableColumns()
}

// GetAdvancedSearchableColumns returns the advanced searchable column names for this resource.
func (r *BaseResource[T]) GetAdvancedSearchableColumns() []string {
	if len(r.advancedSearchableColumns) > 0 {
		return r.advancedSearchableColumns
	}
	if r.tableBuilder == nil {
		return nil
	}
	tbl := tables.New[T]()
	tbl = r.tableBuilder(tbl)
	return tbl.GetAdvancedSearchableColumns()
}

// SummariesUsing computes the aggregates of summarized columns over all
// records matching the search and filters (not just the page), shown as
// "All results" under the table and sent as "summaries" by the API. Without
// it only the page (and each group) is summarized.
func (r *BaseResource[T]) SummariesUsing(fn func(ctx *Context, search string, filters map[string]string) (tables.Summaries, error)) *BaseResource[T] {
	r.summariesUsing = fn
	return r
}

func (r *BaseResource[T]) totals(ctx *Context, search string, filters map[string]string) tables.Summaries {
	if r.summariesUsing == nil {
		return nil
	}
	s, err := r.summariesUsing(ctx, search, filters)
	if err != nil {
		return nil
	}
	return s
}

// ListTotals is used by the API.
func (r *BaseResource[T]) ListTotals(ctx *Context, search string, filters map[string]string) tables.Summaries {
	return r.totals(ctx, search, filters)
}

// GetGroups returns the table's group columns and default group.
func (r *BaseResource[T]) GetGroups() ([]string, string) {
	if r.tableBuilder == nil {
		return nil, ""
	}
	tbl := r.tableBuilder(tables.New[T]())
	return tbl.GetGroups(), tbl.GetDefaultGroup()
}

// IsAdvancedSearchEnabled reports whether advanced search is active on this resource.
func (r *BaseResource[T]) IsAdvancedSearchEnabled() bool {
	return len(r.GetAdvancedSearchableColumns()) > 0
}

// Handlers binding
func (r *BaseResource[T]) OnList(fn ListHandlerFunc[T]) *BaseResource[T] {
	r.onList = fn
	return r
}

func (r *BaseResource[T]) OnFind(fn FindHandlerFunc[T]) *BaseResource[T] {
	r.onFind = fn
	return r
}

func (r *BaseResource[T]) OnCreate(fn CreateHandlerFunc) *BaseResource[T] {
	r.onCreate = fn
	return r
}

func (r *BaseResource[T]) OnUpdate(fn UpdateHandlerFunc) *BaseResource[T] {
	r.onUpdate = fn
	return r
}

func (r *BaseResource[T]) OnDelete(fn DeleteHandlerFunc) *BaseResource[T] {
	r.onDelete = fn
	return r
}

func (r *BaseResource[T]) OnDeleteBulk(fn DeleteBulkHandlerFunc) *BaseResource[T] {
	r.onDeleteBulk = fn
	return r
}

func (r *BaseResource[T]) GetOnList() ListHandlerFunc[T] {
	return r.onList
}

func (r *BaseResource[T]) GetOnFind() FindHandlerFunc[T] {
	return r.onFind
}

func (r *BaseResource[T]) GetOnCreate() CreateHandlerFunc {
	return r.onCreate
}

func (r *BaseResource[T]) GetOnUpdate() UpdateHandlerFunc {
	return r.onUpdate
}

func (r *BaseResource[T]) OnReorder(fn ReorderHandlerFunc) *BaseResource[T] {
	r.onReorder = fn
	return r
}

func (r *BaseResource[T]) OnCellUpdate(fn CellUpdateHandlerFunc) *BaseResource[T] {
	r.onCellUpdate = fn
	return r
}

func (r *BaseResource[T]) GetOnDelete() DeleteHandlerFunc {
	return r.onDelete
}

func (r *BaseResource[T]) GetOnDeleteBulk() DeleteBulkHandlerFunc {
	return r.onDeleteBulk
}

func (r *BaseResource[T]) GetOnReorder() ReorderHandlerFunc {
	return r.onReorder
}

func (r *BaseResource[T]) GetOnCellUpdate() CellUpdateHandlerFunc {
	return r.onCellUpdate
}

// SoftDeletes enables or disables soft-delete handling on this resource.
func (r *BaseResource[T]) SoftDeletes(enabled ...bool) *BaseResource[T] {
	r.softDeletes = len(enabled) == 0 || enabled[0]
	return r
}

// SoftDeleteColumn configures the timestamp or boolean column name for soft deletes (defaults to "deleted_at").
func (r *BaseResource[T]) SoftDeleteColumn(col string) *BaseResource[T] {
	r.softDeleteColumn = col
	return r
}

// IsTrashed sets the closure to identify if a record is currently soft-deleted.
func (r *BaseResource[T]) IsTrashed(fn func(record T) bool) *BaseResource[T] {
	r.isRecordTrashed = fn
	return r
}

// OnRestore registers the callback to restore a soft-deleted record.
func (r *BaseResource[T]) OnRestore(fn RestoreHandlerFunc) *BaseResource[T] {
	r.onRestore = fn
	return r
}

// OnForceDelete registers the callback to permanently purge a record.
func (r *BaseResource[T]) OnForceDelete(fn ForceDeleteHandlerFunc) *BaseResource[T] {
	r.onForceDelete = fn
	return r
}

// OnRestoreBulk registers the callback to restore multiple soft-deleted records.
func (r *BaseResource[T]) OnRestoreBulk(fn RestoreBulkHandlerFunc) *BaseResource[T] {
	r.onRestoreBulk = fn
	return r
}

// OnForceDeleteBulk registers the callback to permanently purge multiple records.
func (r *BaseResource[T]) OnForceDeleteBulk(fn ForceDeleteBulkHandlerFunc) *BaseResource[T] {
	r.onForceDeleteBulk = fn
	return r
}

// CanRestore sets authorization rule for restoring records.
func (r *BaseResource[T]) CanRestore(fn func(ctx *Context, id string) bool) *BaseResource[T] {
	r.canRestore = fn
	return r
}

// CanForceDelete sets authorization rule for permanently purging records.
func (r *BaseResource[T]) CanForceDelete(fn func(ctx *Context, id string) bool) *BaseResource[T] {
	r.canForceDelete = fn
	return r
}

// IsSoftDeletesEnabled returns true if soft-deletes are active on this resource.
func (r *BaseResource[T]) IsSoftDeletesEnabled() bool {
	return r.softDeletes
}

// GetSoftDeleteColumn returns the column name used for soft-deletes.
func (r *BaseResource[T]) GetSoftDeleteColumn() string {
	if r.softDeleteColumn != "" {
		return r.softDeleteColumn
	}
	return "deleted_at"
}

func (r *BaseResource[T]) GetOnRestore() RestoreHandlerFunc {
	return r.onRestore
}

func (r *BaseResource[T]) GetOnForceDelete() ForceDeleteHandlerFunc {
	return r.onForceDelete
}

func (r *BaseResource[T]) GetOnRestoreBulk() RestoreBulkHandlerFunc {
	return r.onRestoreBulk
}

func (r *BaseResource[T]) GetOnForceDeleteBulk() ForceDeleteBulkHandlerFunc {
	return r.onForceDeleteBulk
}

func (r *BaseResource[T]) checkIsTrashed(record T) bool {
	if r.isRecordTrashed != nil {
		return r.isRecordTrashed(record)
	}

	colName := r.GetSoftDeleteColumn()

	if m, ok := any(record).(map[string]any); ok {
		if val, exists := m[colName]; exists && val != nil {
			return true
		}
		if val, exists := m["deleted_at"]; exists && val != nil {
			return true
		}
		return false
	}

	v := reflect.ValueOf(record)
	if v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		for _, fieldName := range []string{"DeletedAt", "Deleted", colName} {
			f := v.FieldByName(fieldName)
			if f.IsValid() {
				if f.Kind() == reflect.Pointer {
					return !f.IsNil()
				}
				if t, ok := f.Interface().(time.Time); ok {
					return !t.IsZero()
				}
				if b, ok := f.Interface().(bool); ok {
					return b
				}
			}
		}
	}
	return false
}

func (r *BaseResource[T]) NavigationItem(ctx *Context) *NavigationItem {
	if r.canViewAny != nil && !r.canViewAny(ctx) {
		return nil
	}
	badge := ""
	if r.navBadge != nil {
		badge = r.navBadge(ctx)
	}
	return &NavigationItem{
		Label: r.pluralModelName,
		Icon:  r.navIcon,
		Group: r.navGroup,
		Sort:  r.navSort,
		Badge: badge,
	}
}

// RegisterRoutes registers full CRUD endpoints for this Resource.
func (r *BaseResource[T]) RegisterRoutes(p *Panel, router *router.Router) {
	basePath := fmt.Sprintf("%s/%s", p.GetPath(), r.slug)
	p.mountSavedViewWeb(router, r, basePath)

	// 1. List Records
	router.GET(basePath, func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleList(ctx, basePath)
	})

	// 2. Static routes first (before wildcard :id)
	router.GET(basePath+"/create", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleCreateView(ctx, basePath)
	})

	router.POST(basePath+"/create", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleStore(ctx, basePath)
	})

	router.POST(basePath+"/bulk", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleBulk(ctx, basePath)
	})

	router.GET(basePath+"/export", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleExport(ctx)
	})

	router.GET(basePath+"/import", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleImportView(ctx, basePath)
	})

	if r.kanban != nil {
		router.GET(basePath+"/board", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			return r.handleKanbanPage(ctx, basePath)
		})
		router.POST(basePath+"/board/move", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			return r.handleKanbanMove(ctx)
		})
	}

	router.POST(basePath+"/bulk-edit", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleBulkEdit(ctx, basePath)
	})

	router.POST(basePath+"/import/preview", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleImportPreview(ctx, basePath)
	})

	router.POST(basePath+"/import", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleImportSubmit(ctx, basePath)
	})

	router.POST(basePath+"/live", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleLive(ctx)
	})

	// Row Reorder endpoint
	router.POST(basePath+"/reorder", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleReorder(ctx, basePath)
	})

	// Inline Cell Editing endpoint
	cellHandler := func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleCellUpdate(ctx, basePath, "")
	}
	router.POST(basePath+"/cell", cellHandler)
	router.PATCH(basePath+"/cell", cellHandler)

	// Soft delete restore static endpoint
	restoreHandler := func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.HandleRestore(ctx, basePath)
	}
	restoreHandler = p.guardUnsafeGET("Restore record", "Restore this record?", restoreHandler)
	router.GET(basePath+"/restore", restoreHandler)
	router.POST(basePath+"/restore", restoreHandler)

	// Soft delete force-delete static endpoint
	forceDeleteHandler := func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.HandleForceDelete(ctx, basePath)
	}
	forceDeleteHandler = p.guardUnsafeGET("Delete permanently", "Permanently delete this record? This cannot be undone.", forceDeleteHandler)
	router.GET(basePath+"/force-delete", forceDeleteHandler)
	router.POST(basePath+"/force-delete", forceDeleteHandler)

	// 3. Wildcard :id routes
	router.GET(basePath+"/:id", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleView(ctx, basePath, c.Param("id"))
	})

	router.GET(basePath+"/:id/edit", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleEditView(ctx, basePath, c.Param("id"))
	})

	router.POST(basePath+"/:id/edit", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleUpdate(ctx, basePath, c.Param("id"))
	})

	// Custom row actions with a Handle runner (tables.Action(name).Handle):
	// GET shows a confirm / form page, POST (CSRF-protected) runs it.
	router.GET(basePath+"/:id/actions/:action", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleActionPage(ctx, basePath, c.Param("id"), c.Param("action"), http.StatusOK, "", nil)
	})
	router.POST(basePath+"/:id/actions/:action", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleActionRun(ctx, basePath, c.Param("id"), c.Param("action"))
	})

	deleteHandler := func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleDelete(ctx, basePath, c.Param("id"))
	}
	deleteHandler = p.guardUnsafeGET("Delete record", "Delete this record?", deleteHandler)
	router.GET(basePath+"/:id/delete", deleteHandler)
	router.POST(basePath+"/:id/delete", deleteHandler)

	// 4. In-Place Relation Manager Routes
	for _, rel := range r.relations {
		relName := rel.Relationship()
		relManager := rel
		relURL := fmt.Sprintf("%s/:id/relations/%s", basePath, relName)

		// In-place Create
		router.POST(relURL+"/create", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			parentID := c.Param("id")
			_ = ctx.Request.ParseForm()
			data := make(map[string]any)
			for k, v := range ctx.Request.PostForm {
				if len(v) == 1 {
					data[k] = v[0]
				} else {
					data[k] = v
				}
			}
			if err := relManager.HandleCreate(ctx, parentID, data); err != nil {
				return ctx.String(http.StatusInternalServerError, "Relation create error: %v", err)
			}
			return ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/%s/edit", basePath, parentID))
		})

		// In-place Edit
		router.POST(relURL+"/:child_id/edit", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			parentID := c.Param("id")
			childID := c.Param("child_id")
			_ = ctx.Request.ParseForm()
			data := make(map[string]any)
			for k, v := range ctx.Request.PostForm {
				if len(v) == 1 {
					data[k] = v[0]
				} else {
					data[k] = v
				}
			}
			if err := relManager.HandleUpdate(ctx, parentID, childID, data); err != nil {
				if conflict, ok := err.(*EditConflictError); ok {
					return ctx.HTML(http.StatusConflict, fmt.Sprintf(`<p>%s</p><p><a href="%s/%s/edit">Reload</a></p>`,
						html.EscapeString(conflict.Error()), basePath, html.EscapeString(parentID)))
				}
				return ctx.String(http.StatusInternalServerError, "Relation update error: %v", err)
			}
			return ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/%s/edit", basePath, parentID))
		})

		// In-place Delete
		deleteRelHandler := func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			parentID := c.Param("id")
			childID := c.Param("child_id")
			if err := relManager.HandleDelete(ctx, parentID, childID); err != nil {
				return ctx.String(http.StatusInternalServerError, "Relation delete error: %v", err)
			}
			return ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/%s/edit", basePath, parentID))
		}
		deleteRelHandler = p.guardUnsafeGET("Delete related record", "Delete this related record?", deleteRelHandler)
		router.GET(relURL+"/:child_id/delete", deleteRelHandler)
		router.POST(relURL+"/:child_id/delete", deleteRelHandler)
	}
}

// ----------------------------------------------------
// Route Handlers Implementation
// ----------------------------------------------------

func (r *BaseResource[T]) handleList(ctx *Context, basePath string) error {
	if r.canViewAny != nil && !r.canViewAny(ctx) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}

	tbl := tables.New[T]()
	if r.tableBuilder != nil {
		tbl = r.tableBuilder(tbl)
	}
	if r.idExtractor != nil {
		tbl.RecordID(r.idExtractor)
	}
	if r.versionOf != nil {
		tbl.RecordVersion(r.versionOf)
	}
	if len(r.BulkEditFields()) > 0 {
		has := false
		for _, ba := range tbl.GetBulkActions() {
			has = has || ba.GetName() == "edit_bulk"
		}
		if !has {
			tbl.BulkActions(append([]*tables.BulkActionDef{tables.EditBulkAction()}, tbl.GetBulkActions()...)...)
		}
	}
	if r.softDeletes {
		tbl.SoftDeletes(true)
		tbl.IsTrashed(r.checkIsTrashed)
		withTrashedFilter(r, tbl)
		if len(tbl.GetBulkActions()) > 0 {
			hasRestoreBulk := false
			hasForceDelBulk := false
			for _, ba := range tbl.GetBulkActions() {
				if ba.GetName() == "restore_bulk" {
					hasRestoreBulk = true
				}
				if ba.GetName() == "force_delete_bulk" {
					hasForceDelBulk = true
				}
			}
			bulk := tbl.GetBulkActions()
			if !hasRestoreBulk {
				bulk = append(bulk, tables.RestoreBulkAction())
			}
			if !hasForceDelBulk {
				bulk = append(bulk, tables.ForceDeleteBulkAction())
			}
			tbl.BulkActions(bulk...)
		}
	}

	search := ctx.Query("search")
	sortCol := ctx.Query("sort")
	sortDir := ctx.DefaultQuery("direction", "asc")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(ctx.DefaultQuery("perPage", "15"))

	filters := tables.FilterValues(ctx.Request.URL.Query())

	// While grouped, rows are sorted by the group column so a group is
	// never split across the page.
	group := tbl.ResolveGroup(ctx.Query("group"))
	if group != "" && sortCol != group {
		sortCol, sortDir = group, "asc"
	}

	var records []T
	total := 0
	if r.onList != nil {
		var err error
		records, total, err = r.onList(ctx, search, sortCol, sortDir, page, perPage, filters)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, "Failed to load records: %v", err)
		}
	}

	tbl.SetData(records, total, page, perPage, search, sortCol, sortDir, filters)
	tbl.SetGroup(group)
	if totals := r.totals(ctx, search, filters); totals != nil {
		tbl.SetTotals(totals)
	}
	if r.canCreate == nil || r.canCreate(ctx) {
		// "New" first, keeping the table's own header actions (import, export, ...)
		tbl.HeaderActions(append([]*tables.TableAction[T]{tables.Action[T]("create").Label(fmt.Sprintf("New %s", r.modelName))}, tbl.GetHeaderActions()...)...)
	}

	content := fmt.Sprintf(`
<div class="filament-header">
  <div>
    <h1 class="filament-title">%s</h1>
    <p class="filament-description">Manage all %s in your database.</p>
  </div>%s
</div>
%s%s`,
		r.pluralModelName, strings.ToLower(r.pluralModelName), r.boardLink(basePath),
		savedViewsMenu(ctx, r, basePath), tbl.Render(basePath),
	)

	return r.renderPage(ctx, r.pluralModelName, content)
}

func (r *BaseResource[T]) handleCreateView(ctx *Context, basePath string) error {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}

	form := forms.New()
	if r.formBuilder != nil {
		form = r.formBuilder(form)
	}
	form.Action(basePath + "/create").Method("POST")

	content := fmt.Sprintf(`
<div class="filament-header">
  <div>
    <h1 class="filament-title">Create %s</h1>
    <p class="filament-description">Fill out the details below to create a new %s.</p>
  </div>
  <a href="%s" class="btn btn-secondary">Back to list</a>
</div>
<div class="filament-card">
  %s
  <div style="margin-top: 1.5rem; display: flex; justify-content: flex-end; gap: 0.75rem;">
    <a href="%s" class="btn btn-secondary">Cancel</a>
    <button type="submit" form="filament-form" class="btn btn-primary">Create Record</button>
  </div>
</div>`,
		r.modelName, strings.ToLower(r.modelName), basePath,
		form.Render(ctx.GoContext()),
		basePath,
	)

	return r.renderPage(ctx, fmt.Sprintf("Create %s", r.modelName), content)
}

func (r *BaseResource[T]) handleStore(ctx *Context, basePath string) error {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}

	_ = ctx.Request.ParseMultipartForm(32 << 20)
	if ctx.Request.PostForm == nil {
		_ = ctx.Request.ParseForm()
	}
	data := forms.ParseFormData(ctx.Request.PostForm)

	// Attach uploaded multipart filenames to data
	if ctx.Request.MultipartForm != nil && ctx.Request.MultipartForm.File != nil {
		for fileKey, fileHeaders := range ctx.Request.MultipartForm.File {
			if len(fileHeaders) == 1 {
				data[fileKey] = fileHeaders[0].Filename
			} else if len(fileHeaders) > 1 {
				names := make([]string, len(fileHeaders))
				for i, fh := range fileHeaders {
					names[i] = fh.Filename
				}
				data[fileKey] = names
			}
		}
	}

	// Fallback to existing file if no new file uploaded
	for k := range data {
		if strings.HasSuffix(k, "_existing") {
			baseKey := strings.TrimSuffix(k, "_existing")
			if cur, exists := data[baseKey]; !exists || cur == "" {
				data[baseKey] = data[k]
			}
		}
	}

	if r.onCreate != nil {
		clean, err := r.validateInput(ctx, data, true)
		if err == nil {
			err = r.onCreate(ctx, clean)
		}
		if err != nil {
			form := forms.New()
			if r.formBuilder != nil {
				form = r.formBuilder(form)
			}
			form.Fill(data)
			status := http.StatusOK
			var verr *forms.ValidationError
			if errors.As(err, &verr) {
				form.SetErrors(verr.Errors)
				status = http.StatusUnprocessableEntity
			}
			form.AddError("error", err.Error())
			content := fmt.Sprintf(`<div class="badge badge-danger mb-4">%s</div>%s`, html.EscapeString(err.Error()), form.Render(ctx.GoContext()))
			return r.renderPageStatus(ctx, status, fmt.Sprintf("Create %s", r.modelName), content)
		}
	}

	return ctx.Redirect(http.StatusSeeOther, basePath)
}

func (r *BaseResource[T]) handleView(ctx *Context, basePath, id string) error {
	if r.onFind == nil {
		return ctx.String(http.StatusNotFound, "Record not found")
	}

	record, err := r.onFind(ctx, id)
	if err != nil {
		return ctx.String(http.StatusNotFound, "Record not found: %v", err)
	}

	if r.canView != nil && !r.canView(ctx, record) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}

	infolist := infolists.New()
	if r.infolistBuilder != nil {
		infolist = r.infolistBuilder(infolist)
	}
	infolist.SetRecord(record)

	editButton := ""
	if r.canEdit == nil || r.canEdit(ctx, record) {
		editButton = fmt.Sprintf(`<a href="%s/%s/edit" class="btn btn-primary">Edit</a>`, basePath, id)
	}

	content := fmt.Sprintf(`
<div class="filament-header">
  <div>
    <h1 class="filament-title">View %s #%s</h1>
  </div>
  <div style="display: flex; gap: 0.5rem;">
    %s
    <a href="%s" class="btn btn-secondary">Back</a>
  </div>
</div>
%s
%s%s`,
		r.modelName, id, editButton, basePath, infolist.Render(),
		r.renderRelations(ctx, basePath, id), r.historyHTML(ctx, id),
	)

	return r.renderPage(ctx, fmt.Sprintf("%s #%s", r.modelName, id), content)
}

func (r *BaseResource[T]) handleEditView(ctx *Context, basePath, id string) error {
	if r.onFind == nil {
		return ctx.String(http.StatusNotFound, "Record not found")
	}

	record, err := r.onFind(ctx, id)
	if err != nil {
		return ctx.String(http.StatusNotFound, "Record not found")
	}

	if r.canEdit != nil && !r.canEdit(ctx, record) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}
	if r.softDeletes && r.checkIsTrashed(record) {
		restore := ""
		if r.onRestore != nil && r.mayRestore(ctx, id) {
			restore = fmt.Sprintf(`<form method="POST" action="%s/restore" style="margin-top: 1rem;"><input type="hidden" name="id" value="%s" /><button type="submit" class="btn btn-primary">Restore</button></form>`,
				basePath, html.EscapeString(id))
		}
		return r.renderPageStatus(ctx, http.StatusConflict, fmt.Sprintf("Edit %s #%s", r.modelName, id), fmt.Sprintf(`<div class="filament-card" data-trashed><h1 class="filament-title">%s #%s is in the trash</h1><p class="filament-description">Restore it before changing it.</p>%s<a href="%s" class="btn btn-secondary" style="margin-top: 1rem;">Back</a></div>`,
			html.EscapeString(r.modelName), html.EscapeString(id), restore, basePath))
	}

	form := forms.New()
	if r.formBuilder != nil {
		form = r.formBuilder(form)
	}
	form.Action(fmt.Sprintf("%s/%s/edit", basePath, id)).Method("POST")

	form.Fill(r.recordFormValues(ctx, record))
	formHTML := withVersionField(form.Render(ctx.GoContext()), r.RecordVersion(record))

	deleteButton := ""
	if r.canDelete == nil || r.canDelete(ctx, record) {
		deleteButton = fmt.Sprintf(`<a href="%s/%s/delete" class="btn btn-danger" onclick="return confirm('Delete this record?');">Delete</a>`, basePath, id)
	}

	content := fmt.Sprintf(`
<div class="filament-header">
  <div>
    <h1 class="filament-title">Edit %s #%s</h1>
    <p class="filament-description">Modify record fields and save changes.</p>
  </div>
  <div style="display: flex; gap: 0.5rem;">
    %s
    <a href="%s" class="btn btn-secondary">Back</a>
  </div>
</div>
<div class="filament-card">
  %s
  <div style="margin-top: 1.5rem; display: flex; justify-content: flex-end; gap: 0.75rem;">
    <a href="%s" class="btn btn-secondary">Cancel</a>
    <button type="submit" form="filament-form" class="btn btn-primary">Save Changes</button>
  </div>
</div>
%s%s`,
		r.modelName, id, deleteButton, basePath,
		formHTML,
		basePath,
		r.renderRelations(ctx, basePath, id), r.historyHTML(ctx, id),
	)

	return r.renderPage(ctx, fmt.Sprintf("Edit %s #%s", r.modelName, id), content)
}

func (r *BaseResource[T]) renderRelations(ctx *Context, basePath, id string) string {
	if len(r.relations) == 0 {
		return ""
	}

	if len(r.relations) == 1 {
		return fmt.Sprintf(`
<div class="filament-relations-section" style="margin-top: 2rem;">
  %s
</div>`, r.relations[0].Render(ctx, basePath, id))
	}

	firstRel := r.relations[0].Relationship()
	var tabButtons strings.Builder
	var tabPanes strings.Builder

	tabButtons.WriteString(`<div class="filament-tab-bar" style="display: flex; gap: 0.5rem; border-bottom: 2px solid var(--gray-200); margin-bottom: 1.5rem; padding-bottom: 0.5rem;">`)
	for _, rel := range r.relations {
		relSlug := rel.Relationship()
		title := rel.Title()
		tabButtons.WriteString(fmt.Sprintf(`
<button type="button" 
  @click="activeRelTab = '%s'" 
  :class="{ 'btn-primary': activeRelTab === '%s', 'btn-secondary': activeRelTab !== '%s' }" 
  class="btn" 
  style="font-size: 0.875rem; padding: 0.5rem 1rem;">
  %s
</button>`, relSlug, relSlug, relSlug, html.EscapeString(title)))

		tabPanes.WriteString(fmt.Sprintf(`
<div x-show="activeRelTab === '%s'" style="display: none;" :style="activeRelTab === '%s' ? 'display: block;' : 'display: none;'">
  %s
</div>`, relSlug, relSlug, rel.Render(ctx, basePath, id)))
	}
	tabButtons.WriteString(`</div>`)

	return fmt.Sprintf(`
<div class="filament-relations-section" style="margin-top: 2rem;" x-data="{ activeRelTab: '%s' }">
  %s
  %s
</div>`, firstRel, tabButtons.String(), tabPanes.String())
}

func (r *BaseResource[T]) handleUpdate(ctx *Context, basePath, id string) error {
	if err := r.authorizeRecord(ctx, id, r.canEdit); err != nil {
		return ctx.String(webStatus(err), "%s", err.Error())
	}

	_ = ctx.Request.ParseMultipartForm(32 << 20)
	if ctx.Request.PostForm == nil {
		_ = ctx.Request.ParseForm()
	}
	data := forms.ParseFormData(ctx.Request.PostForm)

	// Attach uploaded multipart filenames to data
	if ctx.Request.MultipartForm != nil && ctx.Request.MultipartForm.File != nil {
		for fileKey, fileHeaders := range ctx.Request.MultipartForm.File {
			if len(fileHeaders) == 1 {
				data[fileKey] = fileHeaders[0].Filename
			} else if len(fileHeaders) > 1 {
				names := make([]string, len(fileHeaders))
				for i, fh := range fileHeaders {
					names[i] = fh.Filename
				}
				data[fileKey] = names
			}
		}
	}

	// Fallback to existing file if no new file uploaded
	for k := range data {
		if strings.HasSuffix(k, "_existing") {
			baseKey := strings.TrimSuffix(k, "_existing")
			if cur, exists := data[baseKey]; !exists || cur == "" {
				data[baseKey] = data[k]
			}
		}
	}

	if r.onUpdate != nil {
		if err := r.checkVersion(ctx, id, data); err != nil {
			var conflict *EditConflictError
			if errors.As(err, &conflict) {
				return r.renderConflict(ctx, basePath, id, data, conflict)
			}
		}
		clean, err := r.validateInput(ctx, data, false)
		if err != nil {
			form := forms.New()
			if r.formBuilder != nil {
				form = r.formBuilder(form)
			}
			form.Fill(data)
			var verr *forms.ValidationError
			if errors.As(err, &verr) {
				form.SetErrors(verr.Errors)
			}
			content := fmt.Sprintf(`<div class="badge badge-danger mb-4">%s</div>%s`, html.EscapeString(err.Error()), form.Render(ctx.GoContext()))
			return r.renderPageStatus(ctx, http.StatusUnprocessableEntity, fmt.Sprintf("Edit %s", r.modelName), content)
		}
		if err := r.onUpdate(ctx, id, clean); err != nil {
			return ctx.String(http.StatusInternalServerError, "Update error: %v", err)
		}
	}

	return ctx.Redirect(http.StatusSeeOther, basePath)
}

func (r *BaseResource[T]) handleDelete(ctx *Context, basePath, id string) error {
	if err := r.authorizeRecord(ctx, id, r.canDelete); err != nil {
		return ctx.String(webStatus(err), "%s", err.Error())
	}

	if r.onDelete != nil {
		if err := r.onDelete(ctx, id); err != nil {
			return ctx.String(http.StatusInternalServerError, "Delete error: %v", err)
		}
	}
	return ctx.Redirect(http.StatusSeeOther, basePath)
}

func (r *BaseResource[T]) handleBulk(ctx *Context, basePath string) error {
	_ = ctx.Request.ParseForm()
	action := ctx.Request.PostFormValue("bulk_action")
	selectedIDs := ctx.Request.PostForm["selected_ids[]"]

	// Only deleting needs the delete permission (export, edit and restore
	// have their own checks).
	ownCheck := action == "edit_bulk" || action == "export_bulk" || action == "restore_bulk" || action == "force_delete_bulk"
	if !ownCheck && r.canDeleteAny != nil && !r.canDeleteAny(ctx) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}
	if action == "edit_bulk" {
		return r.handleBulkEditStart(ctx, basePath, selectedIDs)
	}

	if action == "delete_bulk" && r.onDeleteBulk != nil && len(selectedIDs) > 0 {
		if err := r.InvokeDeleteBulk(ctx, selectedIDs); err != nil {
			return ctx.String(webStatus(err), "%s", err.Error())
		}
	} else if action == "restore_bulk" && len(selectedIDs) > 0 {
		if err := r.InvokeRestore(ctx, selectedIDs...); err != nil {
			return ctx.String(webStatus(err), "Restore failed: %s", err.Error())
		}
	} else if action == "force_delete_bulk" && len(selectedIDs) > 0 {
		if err := r.InvokeForceDelete(ctx, selectedIDs...); err != nil {
			return ctx.String(webStatus(err), "Delete failed: %s", err.Error())
		}
	} else if action == "export_bulk" {
		return r.handleExport(ctx, selectedIDs...)
	}

	return ctx.Redirect(http.StatusSeeOther, basePath)
}

func (r *BaseResource[T]) HandleRestore(ctx *Context, basePath string, idParam ...string) error {
	var id string
	if len(idParam) > 0 && idParam[0] != "" {
		id = idParam[0]
	} else {
		id = ctx.Request.PostFormValue("id")
		if id == "" {
			id = ctx.Query("id")
		}
		if id == "" {
			var body struct {
				ID string `json:"id"`
			}
			_ = json.NewDecoder(ctx.Request.Body).Decode(&body)
			id = body.ID
		}
	}

	if id == "" {
		return ctx.String(http.StatusBadRequest, "Missing record ID for restore")
	}

	if err := r.InvokeRestore(ctx, id); err != nil {
		if wantsJSON(ctx, basePath) {
			return ctx.JSON(webStatus(err), map[string]string{"error": err.Error()})
		}
		return ctx.String(webStatus(err), "Restore failed: %v", err)
	}

	if wantsJSON(ctx, basePath) {
		return ctx.JSON(http.StatusOK, map[string]any{"success": true, "id": id, "restored": true})
	}
	return ctx.Redirect(http.StatusSeeOther, basePath)
}

func (r *BaseResource[T]) HandleForceDelete(ctx *Context, basePath string, idParam ...string) error {
	var id string
	if len(idParam) > 0 && idParam[0] != "" {
		id = idParam[0]
	} else {
		id = ctx.Request.PostFormValue("id")
		if id == "" {
			id = ctx.Query("id")
		}
		if id == "" {
			var body struct {
				ID string `json:"id"`
			}
			_ = json.NewDecoder(ctx.Request.Body).Decode(&body)
			id = body.ID
		}
	}

	if id == "" {
		return ctx.String(http.StatusBadRequest, "Missing record ID for force delete")
	}

	if err := r.InvokeForceDelete(ctx, id); err != nil {
		if wantsJSON(ctx, basePath) {
			return ctx.JSON(webStatus(err), map[string]string{"error": err.Error()})
		}
		return ctx.String(webStatus(err), "Delete failed: %v", err)
	}

	if wantsJSON(ctx, basePath) {
		return ctx.JSON(http.StatusOK, map[string]any{"success": true, "id": id, "force_deleted": true})
	}
	return ctx.Redirect(http.StatusSeeOther, basePath)
}

func (r *BaseResource[T]) handleExportCSV(ctx *Context, filterIDs ...string) error {
	return r.handleExport(ctx, filterIDs...)
}

func (r *BaseResource[T]) handleExport(ctx *Context, selectedIDs ...string) error {
	if r.canViewAny != nil && !r.canViewAny(ctx) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}

	format := strings.ToLower(ctx.DefaultQuery("format", "csv"))
	search := ctx.Query("search")
	sortCol := ctx.Query("sort")
	sortDir := ctx.DefaultQuery("direction", "asc")

	var filterIDs []string
	if len(selectedIDs) > 0 {
		filterIDs = selectedIDs
	} else if idsQuery := ctx.Query("ids"); idsQuery != "" {
		for _, id := range strings.Split(idsQuery, ",") {
			id = strings.TrimSpace(id)
			if id != "" {
				filterIDs = append(filterIDs, id)
			}
		}
	}

	filters := tables.FilterValues(ctx.Request.URL.Query())
	if adv := ctx.Query("advanced_search"); adv != "" {
		filters["_advanced_search"] = adv
	}

	// With background jobs, exports run as a job unless ?sync=1.
	if p := ctx.Panel(); p != nil && p.jobs != nil && ctx.Query("sync") == "" {
		if r.canViewAny != nil && !r.canViewAny(ctx) {
			return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
		}
		j, err := p.startJob(ctx, "export", "Export of "+r.pluralModelName, func(jctx *Context) (any, error) {
			return nil, r.InvokeExport(jctx, format, filterIDs, search, sortCol, sortDir, filters)
		})
		if err != nil {
			return ctx.String(http.StatusServiceUnavailable, "Export failed: %v", err)
		}
		return ctx.Redirect(http.StatusSeeOther, p.path+"/jobs/"+j.ID)
	}

	if err := r.InvokeExport(ctx, format, filterIDs, search, sortCol, sortDir, filters); err != nil {
		return ctx.String(http.StatusInternalServerError, "Export failed: %v", err)
	}
	return nil
}

func (r *BaseResource[T]) InvokeExport(ctx *Context, format string, filterIDs []string, search, sortCol, sortDir string, filters map[string]string) error {
	if r.canViewAny != nil && !r.canViewAny(ctx) {
		return ErrForbidden
	}
	if r.onList == nil {
		return fmt.Errorf("no list provider configured")
	}

	ext := "csv"
	contentType := "text/csv; charset=utf-8"
	switch format {
	case "tsv":
		ext = "tsv"
		contentType = "text/tab-separated-values; charset=utf-8"
	case "json":
		ext = "json"
		contentType = "application/json; charset=utf-8"
	case "xlsx":
		ext = "xlsx"
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	default:
		format = "csv"
		ext = "csv"
		contentType = "text/csv; charset=utf-8"
	}

	ctx.Response.Header().Set("Content-Type", contentType)
	ctx.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-export.%s\"", r.slug, ext))

	var tbl *tables.Table[T]
	var headers []string
	var columns []tables.ColumnRenderer[T]
	if r.tableBuilder != nil {
		tbl = r.tableBuilder(tables.New[T]())
		only := map[string]bool{}
		for _, c := range strings.Split(ctx.Query("columns"), ",") {
			if c = strings.TrimSpace(c); c != "" {
				only[c] = true
			}
		}
		for _, col := range tbl.GetColumns() {
			if len(only) > 0 && !only[col.GetName()] {
				continue
			}
			columns = append(columns, col)
			headers = append(headers, col.FormatLabel())
		}
	}
	if len(headers) == 0 {
		headers = []string{"ID", "Name", "Details"}
	}
	var xlsxRows [][]string

	filterIDMap := make(map[string]bool)
	for _, fid := range filterIDs {
		filterIDMap[fid] = true
	}

	page := 1
	perPage := 500
	firstRow := true
	var csvWriter *csv.Writer

	if format == "csv" || format == "tsv" {
		csvWriter = csv.NewWriter(&ctx.Response)
		if format == "tsv" {
			csvWriter.Comma = '\t'
		}
		_ = csvWriter.Write(headers)
		csvWriter.Flush()
	} else if format == "json" {
		_, _ = ctx.Response.Write([]byte("[\n"))
	}

	for {
		records, total, err := r.onList(ctx, search, sortCol, sortDir, page, perPage, filters)
		if err != nil {
			return err
		}
		if len(records) == 0 {
			break
		}

		for _, rec := range records {
			recID := ""
			if r.idExtractor != nil {
				recID = r.idExtractor(rec)
			} else if tbl != nil {
				recID = tbl.ExtractRecordID(rec)
			}
			if len(filterIDMap) > 0 && !filterIDMap[recID] {
				continue
			}

			if format == "json" {
				rowMap := make(map[string]any)
				if tbl != nil && len(columns) > 0 {
					for _, col := range columns {
						raw := tbl.ExtractFieldValue(rec, col.GetName())
						rendered := stripHTML(col.RenderCell(rec, raw))
						if rendered != "" {
							rowMap[col.GetName()] = rendered
						} else if raw != nil {
							rowMap[col.GetName()] = raw
						} else {
							rowMap[col.GetName()] = ""
						}
					}
				} else {
					rowMap["id"] = recID
					rowMap["name"] = fmt.Sprintf("%v", rec)
				}
				if recID != "" && rowMap["id"] == nil {
					rowMap["id"] = recID
				}

				rowBytes, _ := json.Marshal(rowMap)
				if !firstRow {
					_, _ = ctx.Response.Write([]byte(",\n"))
				}
				_, _ = ctx.Response.Write(rowBytes)
				firstRow = false
			} else {
				var row []string
				if tbl != nil && len(columns) > 0 {
					for _, col := range columns {
						raw := tbl.ExtractFieldValue(rec, col.GetName())
						cell := stripHTML(col.RenderCell(rec, raw))
						if cell == "" && raw != nil {
							cell = fmt.Sprintf("%v", raw)
						}
						row = append(row, cell)
					}
				} else {
					row = []string{recID, fmt.Sprintf("%v", rec), "Exported from GoFilament"}
				}
				if format == "xlsx" {
					xlsxRows = append(xlsxRows, row)
					continue
				}
				for i := range row {
					row[i] = csvSafeCell(row[i])
				}
				_ = csvWriter.Write(row)
			}
		}

		if csvWriter != nil {
			csvWriter.Flush()
		}
		if flusher, ok := ctx.Response.ResponseWriter.(http.Flusher); ok {
			flusher.Flush()
		}

		reportProgress(ctx, min(page*perPage, total), total)
		if page*perPage >= total || len(records) < perPage {
			break
		}
		page++
	}

	if format == "json" {
		_, _ = ctx.Response.Write([]byte("\n]\n"))
	}
	if format == "xlsx" {
		book, err := buildXLSX(r.pluralModelName, headers, xlsxRows)
		if err != nil {
			return err
		}
		_, _ = ctx.Response.Write(book)
	}
	return nil
}

func (r *BaseResource[T]) handleImportView(ctx *Context, basePath string) error {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}
	return r.renderImportUpload(ctx, basePath, http.StatusOK, "")
}

func (r *BaseResource[T]) handleImportSubmit(ctx *Context, basePath string) error {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}

	_ = ctx.Request.ParseMultipartForm(MaxImportBytes + 1<<20)
	mapping := importFormMapping(ctx)

	// From the preview page: the stashed file, then a result page.
	if token := ctx.Request.FormValue("token"); token != "" {
		raw, ok := r.stashedImportFile(ctx, token)
		if !ok {
			return r.renderImportUpload(ctx, basePath, http.StatusGone, "The preview expired. Upload the file again.")
		}
		if p := ctx.Panel(); p != nil && p.jobs != nil {
			_ = cache.Forget(r.importCacheKey(ctx, token))
			j, err := p.startJob(ctx, "import", "Import of "+r.pluralModelName, func(jctx *Context) (any, error) {
				return r.InvokeImportCSV(jctx, bytes.NewReader(raw), mapping)
			})
			if err != nil {
				return r.renderImportUpload(ctx, basePath, http.StatusServiceUnavailable, err.Error())
			}
			return ctx.Redirect(http.StatusSeeOther, p.path+"/jobs/"+j.ID)
		}
		result, err := r.InvokeImportCSV(ctx, bytes.NewReader(raw), mapping)
		if err != nil {
			return r.renderImportUpload(ctx, basePath, webStatus(err), err.Error())
		}
		_ = cache.Forget(r.importCacheKey(ctx, token))
		return r.renderImportResult(ctx, basePath, result)
	}

	raw, err := uploadedImportFile(ctx)
	if err != nil {
		return ctx.String(http.StatusBadRequest, "Please select a valid CSV file to import")
	}
	result, err := r.InvokeImportCSV(ctx, bytes.NewReader(raw), mapping)
	if err != nil {
		return ctx.String(webStatus(err), "Import failed: %v", err)
	}

	return ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("%s?imported=%d&failed=%d", basePath, result.SuccessCount, result.FailedCount))
}

func (r *BaseResource[T]) InvokeImportCSV(ctx *Context, rdr io.Reader, explicitMapping map[string]string) (*ImportResult, error) {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return nil, ErrForbidden
	}
	if r.onCreate == nil {
		return nil, fmt.Errorf("create handler not configured on resource")
	}
	raw, err := io.ReadAll(io.LimitReader(rdr, MaxImportBytes+1))
	if err != nil {
		return nil, err
	}
	rows, err := readImportTable(bytes.NewReader(raw))
	if err != nil {
		return nil, &forms.ValidationError{Errors: map[string]string{"file": err.Error()}}
	}
	_, result := r.runImport(ctx, rows, explicitMapping, false, bytes.HasPrefix(raw, []byte("PK\x03\x04")))
	r.notifyImport(ctx, result)
	return result, nil
}

func (r *BaseResource[T]) notifyImport(ctx *Context, result *ImportResult) {
	user := "all"
	if ctx.User() != nil {
		user = ctx.User().GetAuthIdentifier()
	}
	if result.FailedCount == 0 {
		_ = notifications.Send(user, notifications.Make("Import Completed").
			Success().
			Body(fmt.Sprintf("Successfully imported %d %s records.", result.SuccessCount, r.pluralModelName)))
	} else {
		_ = notifications.Send(user, notifications.Make("Import Completed with Warnings").
			Warning().
			Body(fmt.Sprintf("Imported %d %s records with %d failures.", result.SuccessCount, r.pluralModelName, result.FailedCount)))
	}
}

func (r *BaseResource[T]) InvokeImportRecords(ctx *Context, rows []map[string]any) (*ImportResult, error) {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return nil, ErrForbidden
	}
	if r.onCreate == nil {
		return nil, fmt.Errorf("create handler not configured on resource")
	}

	result := &ImportResult{}
	for i, row := range rows {
		rowNum := i + 1
		data, err := r.validateInput(ctx, row, true)
		if err == nil {
			err = r.onCreate(ctx, data)
		}
		if err != nil {
			result.FailedCount++
			rowErr := ImportRowError{Row: rowNum}
			var verr *forms.ValidationError
			if errors.As(err, &verr) {
				rowErr.Errors = verr.Errors
			} else {
				rowErr.Message = err.Error()
			}
			if len(result.Rows) < importMaxProblems {
				result.Rows = append(result.Rows, rowErr)
				result.Errors = append(result.Errors, fmt.Sprintf("Record %d: %s", rowNum, rowErr.describe()))
			}
		} else {
			result.SuccessCount++
		}
	}
	r.notifyImport(ctx, result)
	return result, nil
}

func (r *BaseResource[T]) handleLive(ctx *Context) error {
	_ = ctx.Request.ParseForm()
	data := make(map[string]any)
	for k, v := range ctx.Request.PostForm {
		if len(v) == 1 {
			data[k] = v[0]
		} else {
			data[k] = v
		}
	}

	form := forms.New()
	if r.formBuilder != nil {
		form = r.formBuilder(form)
	}
	form.Fill(data)

	// Trigger afterUpdated hook on the changed field if specified
	liveField := ctx.FormValue("_live_field")
	if liveField != "" {
		if comp := form.GetField(liveField); comp != nil {
			if hook, ok := comp.(interface {
				TriggerAfterUpdated(state any, f *forms.Form)
			}); ok {
				hook.TriggerAfterUpdated(data[liveField], form)
			}
		}
	}

	return ctx.HTML(http.StatusOK, form.Render(ctx.GoContext()))
}

func (r *BaseResource[T]) renderPage(ctx *Context, pageTitle, innerContent string) error {
	return r.renderPageStatus(ctx, http.StatusOK, pageTitle, innerContent)
}

func (r *BaseResource[T]) renderPageStatus(ctx *Context, status int, pageTitle, innerContent string) error {
	p := ctx.Panel()
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s" dir="%s">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s - %s</title>
  <link rel="stylesheet" href="%s/assets/filament.css">
  <script src="%s/assets/filament.js"></script>
%s
</head>
<body class="filament-body">
  <div class="filament-shell">
    %s
    <div class="filament-main">
      %s
      <main class="filament-content %s">
        %s
      </main>
    </div>
  </div>
  %s
</body>
</html>`,
		ctx.Locale(), ctx.Direction(),
		pageTitle, p.GetBrandName(),
		p.GetPath(), p.GetPath(),
		p.RenderHeadHTML(ctx),
		p.RenderSidebar(ctx),
		p.RenderTopbar(ctx),
		p.GetMaxContentWidth(),
		innerContent,
		p.RenderBodyEndHTML(ctx),
	)
	return ctx.HTML(status, html)
}

// API Invoker methods for JSON REST API and Flutter clients

func (r *BaseResource[T]) InvokeList(ctx *Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) (any, int, error) {
	if r.canViewAny != nil && !r.canViewAny(ctx) {
		return nil, 0, ErrForbidden
	}
	if r.onList == nil {
		return []T{}, 0, nil
	}
	records, total, err := r.onList(ctx, search, sortCol, sortDir, page, perPage, filters)
	return records, total, err
}

func (r *BaseResource[T]) InvokeFind(ctx *Context, id string) (any, error) {
	if r.onFind == nil {
		return nil, fmt.Errorf("record not found")
	}
	record, err := r.onFind(ctx, id)
	if err != nil {
		return nil, err
	}
	if r.canView != nil && !r.canView(ctx, record) {
		return nil, ErrForbidden
	}
	return record, nil
}

func (r *BaseResource[T]) InvokeCreate(ctx *Context, data map[string]any) error {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return ErrForbidden
	}
	if r.onCreate == nil {
		return fmt.Errorf("create not implemented")
	}
	data, err := r.validateInput(ctx, data, true)
	if err != nil {
		return err
	}
	return r.onCreate(ctx, data)
}

func (r *BaseResource[T]) InvokeUpdate(ctx *Context, id string, data map[string]any) error {
	if err := r.authorizeRecord(ctx, id, r.canEdit); err != nil {
		return err
	}
	if r.onUpdate == nil {
		return fmt.Errorf("update not implemented")
	}
	if err := r.refuseTrashed(ctx, id); err != nil {
		return err
	}
	if err := r.checkVersion(ctx, id, data); err != nil {
		return err
	}
	data, err := r.validateInput(ctx, data, false)
	if err != nil {
		return err
	}
	return r.onUpdate(ctx, id, data)
}

func (r *BaseResource[T]) InvokeDelete(ctx *Context, id string) error {
	if err := r.authorizeRecord(ctx, id, r.canDelete); err != nil {
		return err
	}
	if r.onDelete == nil {
		return fmt.Errorf("delete not implemented")
	}
	if err := r.refuseTrashed(ctx, id); err != nil {
		return err
	}
	return r.onDelete(ctx, id)
}

func (r *BaseResource[T]) handleReorder(ctx *Context, basePath string) error {
	data := parseRequestData(ctx.Context)
	var order []string
	if rawOrder, ok := data["order"]; ok {
		if slice, ok := rawOrder.([]string); ok {
			order = slice
		} else if sliceAny, ok := rawOrder.([]any); ok {
			for _, it := range sliceAny {
				order = append(order, fmt.Sprintf("%v", it))
			}
		}
	} else if rawOrderSlice, ok := data["order[]"]; ok {
		if one, ok := rawOrderSlice.(string); ok && one != "" {
			order = []string{one} // a single form value is not a slice
		} else if slice, ok := rawOrderSlice.([]string); ok {
			order = slice
		} else if sliceAny, ok := rawOrderSlice.([]any); ok {
			for _, it := range sliceAny {
				order = append(order, fmt.Sprintf("%v", it))
			}
		}
	}

	if len(order) == 0 {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"error": "No order provided"})
	}
	if err := r.authorizeReorder(ctx, order); err != nil {
		return ctx.JSON(http.StatusForbidden, map[string]any{"error": "Forbidden"})
	}

	if r.onReorder != nil {
		if err := r.onReorder(ctx, order); err != nil {
			return ctx.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
		}
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"success": true,
		"order":   order,
	})
}

func (r *BaseResource[T]) handleCellUpdate(ctx *Context, basePath, id string) error {
	data := parseRequestData(ctx.Context)
	if id == "" {
		if recID, ok := data["record_id"]; ok && recID != nil {
			id = fmt.Sprintf("%v", recID)
		} else if recID, ok := data["id"]; ok && recID != nil {
			id = fmt.Sprintf("%v", recID)
		}
	}
	if id == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"error": "Record ID is required"})
	}

	col := ""
	if c, ok := data["column"]; ok && c != nil {
		col = fmt.Sprintf("%v", c)
	}
	val := data["value"]

	if col == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]any{"error": "Column is required"})
	}

	if err := r.checkCellVersion(ctx, id, data); err != nil {
		if conflict, ok := err.(*EditConflictError); ok {
			return ctx.JSON(http.StatusConflict, conflictJSON(conflict))
		}
	}
	if err := r.InvokeCellUpdate(ctx, id, col, val); err != nil {
		return ctx.JSON(webStatus(err), map[string]any{"error": err.Error()})
	}

	out := map[string]any{
		"success": true,
		"id":      id,
		"column":  col,
		"value":   val,
	}
	if v := r.currentVersion(ctx, id); v != "" {
		out["version"] = v
	}
	return ctx.JSON(http.StatusOK, out)
}

// authorizeReorder requires edit rights on every reordered record, since
// reordering changes them.
func (r *BaseResource[T]) authorizeReorder(ctx *Context, ids []string) error {
	for _, id := range ids {
		if err := r.authorizeRecord(ctx, id, r.canEdit); err != nil {
			return err
		}
	}
	return nil
}

func (r *BaseResource[T]) InvokeReorder(ctx *Context, orderedIDs []string) error {
	if err := r.authorizeReorder(ctx, orderedIDs); err != nil {
		return err
	}
	if r.onReorder != nil {
		return r.onReorder(ctx, orderedIDs)
	}
	return nil
}

func (r *BaseResource[T]) InvokeCellUpdate(ctx *Context, id, column string, value any) error {
	if err := r.authorizeRecord(ctx, id, r.canEdit); err != nil {
		return err
	}
	if err := r.refuseTrashed(ctx, id); err != nil {
		return err
	}
	// Only editable table columns can be changed inline. A resource without
	// a table but with its own OnCellUpdate handler decides by itself.
	editable, options := r.editableColumn(column)
	if !editable && (r.tableBuilder != nil || r.onCellUpdate == nil) {
		return ErrColumnNotEditable
	}
	if len(options) > 0 {
		if _, ok := options[formatScalar(value)]; !ok {
			return &forms.ValidationError{Errors: map[string]string{column: "This is not a valid choice."}}
		}
	}
	// A column that is also a form field follows the field's rules.
	if comps := r.formComponents(); comps != nil && !r.acceptUnvalidated && forms.FindField(comps, column) != nil {
		withSelf(ctx)
		clean, err := forms.Validate(ctx.GoContext(), comps, map[string]any{column: value}, false)
		if err != nil {
			return err
		}
		v, ok := clean[column]
		if !ok {
			return ErrColumnNotEditable // disabled or hidden in the form
		}
		value = v
	}
	if r.onCellUpdate != nil {
		return r.onCellUpdate(ctx, id, column, value)
	}
	if r.onUpdate != nil {
		return r.onUpdate(ctx, id, map[string]any{column: value})
	}
	return nil
}

func (r *BaseResource[T]) InvokeDeleteBulk(ctx *Context, ids []string) error {
	if r.canDeleteAny != nil && !r.canDeleteAny(ctx) {
		return ErrForbidden
	}
	if r.onDeleteBulk == nil {
		return fmt.Errorf("bulk delete not implemented")
	}
	if r.canDelete != nil || r.canView != nil {
		// every record must pass, or nothing is deleted
		for _, id := range ids {
			if err := r.authorizeRecord(ctx, id, r.canDelete); err != nil {
				return err
			}
		}
	}
	return r.onDeleteBulk(ctx, ids)
}

// csvSafeCell stops spreadsheet formula injection: a cell starting with
// =, +, -, @, tab or carriage return is run as a formula by Excel and
// friends, so it is prefixed with a quote and read as text.
func csvSafeCell(v string) string {
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return v // a plain number (-2, +3.5) is not a formula
	}
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}
