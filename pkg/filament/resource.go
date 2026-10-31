package filament

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/globalsearch"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/notifications"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
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

// ReorderHandlerFunc is the callback to persist a new order of record IDs.
type ReorderHandlerFunc func(ctx *Context, orderedIDs []string) error

// CellUpdateHandlerFunc is the callback to update a single cell on a record.
type CellUpdateHandlerFunc func(ctx *Context, id, column string, value any) error

// BaseResource implements ResourceInterface with generic Ent/model type T.
type BaseResource[T any] struct {
	slug            string
	modelName       string
	pluralModelName string
	navGroup        string
	navIcon         string
	navSort         int
	navBadge        func(ctx *Context) string
	formBuilder     func(form *forms.Form) *forms.Form
	tableBuilder    func(table *tables.Table[T]) *tables.Table[T]
	infolistBuilder func(infolist *infolists.Infolist) *infolists.Infolist
	idExtractor     func(record T) string
	relations       []RelationManager

	// Handlers
	onList       ListHandlerFunc[T]
	onFind       FindHandlerFunc[T]
	onCreate     CreateHandlerFunc
	onUpdate     UpdateHandlerFunc
	onDelete     DeleteHandlerFunc
	onDeleteBulk DeleteBulkHandlerFunc
	onReorder    ReorderHandlerFunc
	onCellUpdate CellUpdateHandlerFunc
	// Authorization
	canViewAny   func(ctx *Context) bool
	canView      func(ctx *Context, record T) bool
	canCreate    func(ctx *Context) bool
	canEdit      func(ctx *Context, record T) bool
	canDelete    func(ctx *Context, record T) bool
	canDeleteAny func(ctx *Context) bool

	// Global Search
	globallySearchable  bool
	searchAttributes    []string
	globalResultFunc    func(record T) GlobalSearchResult
	globalResultTitle   func(record T) string
	globalResultDetails func(record T) map[string]string
	globalResultActions func(record T) []GlobalSearchAction
	globalResultURL     func(record T) string
	globalSearchLimit   int
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
	}
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

	deleteHandler := func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleDelete(ctx, basePath, c.Param("id"))
	}
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

	search := ctx.Query("search")
	sortCol := ctx.Query("sort")
	sortDir := ctx.DefaultQuery("direction", "asc")
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(ctx.DefaultQuery("perPage", "15"))

	filters := make(map[string]string)
	for k, v := range ctx.Request.URL.Query() {
		if strings.HasPrefix(k, "filter_") && len(v) > 0 {
			filters[strings.TrimPrefix(k, "filter_")] = v[0]
		}
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
	if r.canCreate == nil || r.canCreate(ctx) {
		tbl.HeaderActions(tables.Action[T]("create").Label(fmt.Sprintf("New %s", r.modelName)))
	}

	content := fmt.Sprintf(`
<div class="filament-header">
  <div>
    <h1 class="filament-title">%s</h1>
    <p class="filament-description">Manage all %s in your database.</p>
  </div>
</div>
%s`,
		r.pluralModelName, strings.ToLower(r.pluralModelName),
		tbl.Render(basePath),
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
		if err := r.onCreate(ctx, data); err != nil {
			form := forms.New()
			if r.formBuilder != nil {
				form = r.formBuilder(form)
			}
			form.Fill(data)
			form.AddError("error", err.Error())
			content := fmt.Sprintf(`<div class="badge badge-danger mb-4">%s</div>%s`, err.Error(), form.Render(ctx.GoContext()))
			return r.renderPage(ctx, fmt.Sprintf("Create %s", r.modelName), content)
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
%s`,
		r.modelName, id, editButton, basePath, infolist.Render(),
		r.renderRelations(ctx, basePath, id),
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

	form := forms.New()
	if r.formBuilder != nil {
		form = r.formBuilder(form)
	}
	form.Action(fmt.Sprintf("%s/%s/edit", basePath, id)).Method("POST")

	if m, ok := any(record).(map[string]any); ok {
		form.Fill(m)
	}

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
%s`,
		r.modelName, id, deleteButton, basePath,
		form.Render(ctx.GoContext()),
		basePath,
		r.renderRelations(ctx, basePath, id),
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
	if r.onFind != nil && r.canEdit != nil {
		record, err := r.onFind(ctx, id)
		if err == nil && !r.canEdit(ctx, record) {
			return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
		}
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
		if err := r.onUpdate(ctx, id, data); err != nil {
			return ctx.String(http.StatusInternalServerError, "Update error: %v", err)
		}
	}

	return ctx.Redirect(http.StatusSeeOther, basePath)
}

func (r *BaseResource[T]) handleDelete(ctx *Context, basePath, id string) error {
	if r.onFind != nil && r.canDelete != nil {
		record, err := r.onFind(ctx, id)
		if err == nil && !r.canDelete(ctx, record) {
			return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
		}
	}

	if r.onDelete != nil {
		if err := r.onDelete(ctx, id); err != nil {
			return ctx.String(http.StatusInternalServerError, "Delete error: %v", err)
		}
	}
	return ctx.Redirect(http.StatusSeeOther, basePath)
}

func (r *BaseResource[T]) handleBulk(ctx *Context, basePath string) error {
	if r.canDeleteAny != nil && !r.canDeleteAny(ctx) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}

	_ = ctx.Request.ParseForm()
	action := ctx.Request.PostFormValue("bulk_action")
	selectedIDs := ctx.Request.PostForm["selected_ids[]"]

	if action == "delete_bulk" && r.onDeleteBulk != nil && len(selectedIDs) > 0 {
		_ = r.onDeleteBulk(ctx, selectedIDs)
	} else if action == "export_bulk" {
		return r.handleExport(ctx, selectedIDs...)
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

	filters := make(map[string]string)
	for k, v := range ctx.Request.URL.Query() {
		if strings.HasPrefix(k, "filter_") && len(v) > 0 {
			filters[strings.TrimPrefix(k, "filter_")] = v[0]
		}
	}

	if err := r.InvokeExport(ctx, format, filterIDs, search, sortCol, sortDir, filters); err != nil {
		return ctx.String(http.StatusInternalServerError, "Export failed: %v", err)
	}
	return nil
}

func (r *BaseResource[T]) InvokeExport(ctx *Context, format string, filterIDs []string, search, sortCol, sortDir string, filters map[string]string) error {
	if r.canViewAny != nil && !r.canViewAny(ctx) {
		return fmt.Errorf("403 Forbidden: Access denied")
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
	default:
		format = "csv"
		ext = "csv"
		contentType = "text/csv; charset=utf-8"
	}

	ctx.Response.Header().Set("Content-Type", contentType)
	ctx.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-export.%s\"", r.slug, ext))

	var tbl *tables.Table[T]
	var headers []string
	if r.tableBuilder != nil {
		tbl = r.tableBuilder(tables.New[T]())
		for _, col := range tbl.GetColumns() {
			headers = append(headers, col.FormatLabel())
		}
	}
	if len(headers) == 0 {
		headers = []string{"ID", "Name", "Details"}
	}

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
				if tbl != nil && len(tbl.GetColumns()) > 0 {
					for _, col := range tbl.GetColumns() {
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
				if tbl != nil && len(tbl.GetColumns()) > 0 {
					for _, col := range tbl.GetColumns() {
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
				_ = csvWriter.Write(row)
			}
		}

		if csvWriter != nil {
			csvWriter.Flush()
		}
		if flusher, ok := ctx.Response.ResponseWriter.(http.Flusher); ok {
			flusher.Flush()
		}

		if page*perPage >= total || len(records) < perPage {
			break
		}
		page++
	}

	if format == "json" {
		_, _ = ctx.Response.Write([]byte("\n]\n"))
	}
	return nil
}

func (r *BaseResource[T]) handleImportView(ctx *Context, basePath string) error {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}

	content := fmt.Sprintf(`
<div style="max-width: 48rem; margin: 0 auto; padding: 2rem 1rem;">
  <div style="margin-bottom: 1.5rem;">
    <a href="%s" style="font-size: 0.875rem; color: var(--primary-600); text-decoration: none;">&larr; Back to %s</a>
    <h1 style="font-size: 1.875rem; font-weight: 700; color: var(--gray-900); margin-top: 0.5rem;">Import %s</h1>
    <p style="font-size: 0.875rem; color: var(--gray-500); margin-top: 0.25rem;">
      Upload a CSV or TSV file to bulk import records. Column headers will be automatically mapped to resource fields.
    </p>
  </div>

  <div class="filament-card" style="padding: 2rem;">
    <form method="POST" action="%s/import" enctype="multipart/form-data">
      <div style="margin-bottom: 1.5rem;">
        <label style="display: block; font-size: 0.875rem; font-weight: 500; color: var(--gray-700); margin-bottom: 0.5rem;">
          CSV or TSV Data File
        </label>
        <input type="file" name="file" accept=".csv,.tsv,.txt" required class="form-control" style="padding: 0.5rem;" />
        <p style="font-size: 0.75rem; color: var(--gray-500); margin-top: 0.375rem;">
          Supported formats: .csv, .tsv. Need a sample format? <a href="%s/export?format=csv" style="color: var(--primary-600);">Download CSV Template</a>.
        </p>
      </div>

      <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 2rem;">
        <a href="%s" class="btn btn-secondary">Cancel</a>
        <button type="submit" class="btn btn-primary">Start Import</button>
      </div>
    </form>
  </div>
</div>`, basePath, html.EscapeString(r.pluralModelName), html.EscapeString(r.pluralModelName), basePath, basePath, basePath)

	return r.renderPage(ctx, "Import "+r.pluralModelName, content)
}

func (r *BaseResource[T]) handleImportSubmit(ctx *Context, basePath string) error {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}

	_ = ctx.Request.ParseMultipartForm(32 << 20)
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		file, _, err = ctx.Request.FormFile("csv_file")
	}
	if err != nil {
		return ctx.String(http.StatusBadRequest, "Please select a valid CSV file to import")
	}
	defer file.Close()

	mapping := make(map[string]string)
	for k, v := range ctx.Request.PostForm {
		if strings.HasPrefix(k, "mapping[") && strings.HasSuffix(k, "]") && len(v) > 0 {
			src := strings.TrimSuffix(strings.TrimPrefix(k, "mapping["), "]")
			mapping[src] = v[0]
		}
	}

	result, err := r.InvokeImportCSV(ctx, file, mapping)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "Import failed: %v", err)
	}

	return ctx.Redirect(http.StatusSeeOther, fmt.Sprintf("%s?imported=%d&failed=%d", basePath, result.SuccessCount, result.FailedCount))
}

func (r *BaseResource[T]) InvokeImportCSV(ctx *Context, rdr io.Reader, explicitMapping map[string]string) (*ImportResult, error) {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return nil, fmt.Errorf("403 Forbidden: Access denied")
	}
	if r.onCreate == nil {
		return nil, fmt.Errorf("create handler not configured on resource")
	}

	csvReader := csv.NewReader(rdr)
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = -1

	headers, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}

	var mappedFields []string
	for _, h := range headers {
		norm := normalizeHeader(h)
		if field, ok := explicitMapping[norm]; ok && field != "" {
			mappedFields = append(mappedFields, field)
		} else if field, ok := explicitMapping[h]; ok && field != "" {
			mappedFields = append(mappedFields, field)
		} else {
			mappedFields = append(mappedFields, norm)
		}
	}

	result := &ImportResult{}
	rowNum := 1
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		rowNum++
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: parse error: %v", rowNum, err))
			continue
		}

		data := make(map[string]any)
		for i, val := range record {
			if i < len(mappedFields) {
				col := mappedFields[i]
				if col != "" {
					data[col] = strings.TrimSpace(val)
				}
			}
		}
		if len(data) == 0 {
			continue
		}

		if err := r.onCreate(ctx, data); err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("Row %d: %v", rowNum, err))
		} else {
			result.SuccessCount++
		}
	}

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

	return result, nil
}

func (r *BaseResource[T]) InvokeImportRecords(ctx *Context, rows []map[string]any) (*ImportResult, error) {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return nil, fmt.Errorf("403 Forbidden: Access denied")
	}
	if r.onCreate == nil {
		return nil, fmt.Errorf("create handler not configured on resource")
	}

	result := &ImportResult{}
	for i, row := range rows {
		rowNum := i + 1
		if err := r.onCreate(ctx, row); err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("Record %d: %v", rowNum, err))
		} else {
			result.SuccessCount++
		}
	}

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
			if hook, ok := comp.(interface{ TriggerAfterUpdated(state any, f *forms.Form) }); ok {
				hook.TriggerAfterUpdated(data[liveField], form)
			}
		}
	}

	return ctx.HTML(http.StatusOK, form.Render(ctx.GoContext()))
}

func (r *BaseResource[T]) renderPage(ctx *Context, pageTitle, innerContent string) error {
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
	return ctx.HTML(http.StatusOK, html)
}

// API Invoker methods for JSON REST API and Flutter clients

func (r *BaseResource[T]) InvokeList(ctx *Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) (any, int, error) {
	if r.canViewAny != nil && !r.canViewAny(ctx) {
		return nil, 0, fmt.Errorf("403 Forbidden: Access denied")
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
		return nil, fmt.Errorf("403 Forbidden: Access denied")
	}
	return record, nil
}

func (r *BaseResource[T]) InvokeCreate(ctx *Context, data map[string]any) error {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return fmt.Errorf("403 Forbidden: Access denied")
	}
	if r.onCreate == nil {
		return fmt.Errorf("create not implemented")
	}
	return r.onCreate(ctx, data)
}

func (r *BaseResource[T]) InvokeUpdate(ctx *Context, id string, data map[string]any) error {
	if r.onFind != nil && r.canEdit != nil {
		record, err := r.onFind(ctx, id)
		if err == nil && !r.canEdit(ctx, record) {
			return fmt.Errorf("403 Forbidden: Access denied")
		}
	}
	if r.onUpdate == nil {
		return fmt.Errorf("update not implemented")
	}
	return r.onUpdate(ctx, id, data)
}

func (r *BaseResource[T]) InvokeDelete(ctx *Context, id string) error {
	if r.onFind != nil && r.canDelete != nil {
		record, err := r.onFind(ctx, id)
		if err == nil && !r.canDelete(ctx, record) {
			return fmt.Errorf("403 Forbidden: Access denied")
		}
	}
	if r.onDelete == nil {
		return fmt.Errorf("delete not implemented")
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
		if slice, ok := rawOrderSlice.([]string); ok {
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

	if r.onFind != nil && r.canEdit != nil {
		rec, err := r.onFind(ctx, id)
		if err == nil && !r.canEdit(ctx, rec) {
			return ctx.JSON(http.StatusForbidden, map[string]any{"error": "403 Forbidden: Access denied"})
		}
	}

	if r.onCellUpdate != nil {
		if err := r.onCellUpdate(ctx, id, col, val); err != nil {
			return ctx.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
		}
	} else if r.onUpdate != nil {
		if err := r.onUpdate(ctx, id, map[string]any{col: val}); err != nil {
			return ctx.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
		}
	}

	return ctx.JSON(http.StatusOK, map[string]any{
		"success": true,
		"id":      id,
		"column":  col,
		"value":   val,
	})
}

func (r *BaseResource[T]) InvokeReorder(ctx *Context, orderedIDs []string) error {
	if r.onReorder != nil {
		return r.onReorder(ctx, orderedIDs)
	}
	return nil
}

func (r *BaseResource[T]) InvokeCellUpdate(ctx *Context, id, column string, value any) error {
	if r.onFind != nil && r.canEdit != nil {
		record, err := r.onFind(ctx, id)
		if err == nil && !r.canEdit(ctx, record) {
			return fmt.Errorf("403 Forbidden: Access denied")
		}
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
		return fmt.Errorf("403 Forbidden: Access denied")
	}
	if r.onDeleteBulk == nil {
		return fmt.Errorf("bulk delete not implemented")
	}
	return r.onDeleteBulk(ctx, ids)
}
