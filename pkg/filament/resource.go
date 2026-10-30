package filament

import (
	"encoding/csv"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
)

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
	// Authorization
	canViewAny   func(ctx *Context) bool
	canView      func(ctx *Context, record T) bool
	canCreate    func(ctx *Context) bool
	canEdit      func(ctx *Context, record T) bool
	canDelete    func(ctx *Context, record T) bool
	canDeleteAny func(ctx *Context) bool

	// Global Search
	globallySearchable bool
	searchAttributes   []string
	globalResultFunc   func(record T) GlobalSearchResult
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

func (r *BaseResource[T]) SearchGlobal(ctx *Context, query string) []GlobalSearchResult {
	if !r.globallySearchable || r.onList == nil {
		return nil
	}

	records, _, err := r.onList(ctx, query, "", "asc", 1, 5, nil)
	if err != nil || len(records) == 0 {
		return nil
	}

	var results []GlobalSearchResult
	basePath := fmt.Sprintf("%s/%s", ctx.Panel().GetPath(), r.slug)

	for _, rec := range records {
		if r.globalResultFunc != nil {
			res := r.globalResultFunc(rec)
			if res.Group == "" {
				res.Group = r.pluralModelName
			}
			results = append(results, res)
			continue
		}

		recID := ""
		if r.idExtractor != nil {
			recID = r.idExtractor(rec)
		}
		title := fmt.Sprintf("%s #%s", r.modelName, recID)

		results = append(results, GlobalSearchResult{
			Title: title,
			URL:   fmt.Sprintf("%s/%s", basePath, recID),
			Group: r.pluralModelName,
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

func (r *BaseResource[T]) GetOnDelete() DeleteHandlerFunc {
	return r.onDelete
}

func (r *BaseResource[T]) GetOnDeleteBulk() DeleteBulkHandlerFunc {
	return r.onDeleteBulk
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
		return r.handleExportCSV(ctx)
	})

	router.POST(basePath+"/live", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return r.handleLive(ctx)
	})

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

	_ = ctx.Request.ParseForm()
	data := make(map[string]any)
	for k, v := range ctx.Request.PostForm {
		if len(v) == 1 {
			data[k] = v[0]
		} else {
			data[k] = v
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

	_ = ctx.Request.ParseForm()
	data := make(map[string]any)
	for k, v := range ctx.Request.PostForm {
		if len(v) == 1 {
			data[k] = v[0]
		} else {
			data[k] = v
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
		return r.handleExportCSV(ctx, selectedIDs...)
	}

	return ctx.Redirect(http.StatusSeeOther, basePath)
}

func (r *BaseResource[T]) handleExportCSV(ctx *Context, filterIDs ...string) error {
	if r.onList == nil {
		return ctx.String(http.StatusInternalServerError, "No list provider configured")
	}

	records, _, err := r.onList(ctx, "", "", "asc", 1, 10000, nil)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, "Failed to load records for export: %v", err)
	}

	ctx.Response.Header().Set("Content-Type", "text/csv")
	ctx.Response.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-export.csv\"", r.slug))

	writer := csv.NewWriter(&ctx.Response)
	defer writer.Flush()

	// Write Header
	_ = writer.Write([]string{"ID", "Name", "Details"})

	for _, rec := range records {
		recID := ""
		if r.idExtractor != nil {
			recID = r.idExtractor(rec)
		}
		// If specific IDs selected, filter
		if len(filterIDs) > 0 {
			matched := false
			for _, fid := range filterIDs {
				if fid == recID {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		_ = writer.Write([]string{recID, fmt.Sprintf("%v", rec), "Exported from GoFilament"})
	}

	return nil
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
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s - %s</title>
  <link rel="stylesheet" href="%s/assets/filament.css">
  <script src="%s/assets/filament.js"></script>
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
</body>
</html>`,
		pageTitle, p.GetBrandName(),
		p.GetPath(), p.GetPath(),
		p.RenderSidebar(ctx),
		p.RenderTopbar(ctx),
		p.GetMaxContentWidth(),
		innerContent,
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

func (r *BaseResource[T]) InvokeDeleteBulk(ctx *Context, ids []string) error {
	if r.canDeleteAny != nil && !r.canDeleteAny(ctx) {
		return fmt.Errorf("403 Forbidden: Access denied")
	}
	if r.onDeleteBulk == nil {
		return fmt.Errorf("bulk delete not implemented")
	}
	return r.onDeleteBulk(ctx, ids)
}
