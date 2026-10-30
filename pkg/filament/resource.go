package filament

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/tables"
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
}

// NewResource creates a new Resource with a URL slug.
func NewResource[T any](slug string) *BaseResource[T] {
	return &BaseResource[T]{
		slug:            strings.Trim(slug, "/"),
		modelName:       strings.Title(slug),
		pluralModelName: strings.Title(slug),
		navIcon:         "heroicon-rectangle-stack",
		relations:       make([]RelationManager, 0),
	}
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

func (r *BaseResource[T]) NavigationItem(ctx *Context) *NavigationItem {
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
		ctx := NewContext(c, p)
		return r.handleList(ctx, basePath)
	})

	// 2. Static routes first (before wildcard :id)
	router.GET(basePath+"/create", func(c *gcontext.Context) error {
		ctx := NewContext(c, p)
		return r.handleCreateView(ctx, basePath)
	})

	router.POST(basePath+"/create", func(c *gcontext.Context) error {
		ctx := NewContext(c, p)
		return r.handleStore(ctx, basePath)
	})

	router.POST(basePath+"/bulk", func(c *gcontext.Context) error {
		ctx := NewContext(c, p)
		return r.handleBulk(ctx, basePath)
	})

	// 3. Wildcard :id routes
	router.GET(basePath+"/:id", func(c *gcontext.Context) error {
		ctx := NewContext(c, p)
		return r.handleView(ctx, basePath, c.Param("id"))
	})

	router.GET(basePath+"/:id/edit", func(c *gcontext.Context) error {
		ctx := NewContext(c, p)
		return r.handleEditView(ctx, basePath, c.Param("id"))
	})

	router.POST(basePath+"/:id/edit", func(c *gcontext.Context) error {
		ctx := NewContext(c, p)
		return r.handleUpdate(ctx, basePath, c.Param("id"))
	})

	deleteHandler := func(c *gcontext.Context) error {
		ctx := NewContext(c, p)
		return r.handleDelete(ctx, basePath, c.Param("id"))
	}
	router.GET(basePath+"/:id/delete", deleteHandler)
	router.POST(basePath+"/:id/delete", deleteHandler)
}

// ----------------------------------------------------
// Route Handlers Implementation
// ----------------------------------------------------

func (r *BaseResource[T]) handleList(ctx *Context, basePath string) error {
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
	tbl.HeaderActions(tables.Action[T]("create").Label(fmt.Sprintf("New %s", r.modelName)))

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

	infolist := infolists.New()
	if r.infolistBuilder != nil {
		infolist = r.infolistBuilder(infolist)
	}
	infolist.SetRecord(record)

	content := fmt.Sprintf(`
<div class="filament-header">
  <div>
    <h1 class="filament-title">View %s #%s</h1>
  </div>
  <div style="display: flex; gap: 0.5rem;">
    <a href="%s/%s/edit" class="btn btn-primary">Edit</a>
    <a href="%s" class="btn btn-secondary">Back</a>
  </div>
</div>
%s`,
		r.modelName, id, basePath, id, basePath, infolist.Render(),
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

	form := forms.New()
	if r.formBuilder != nil {
		form = r.formBuilder(form)
	}
	form.Action(fmt.Sprintf("%s/%s/edit", basePath, id)).Method("POST")

	if m, ok := any(record).(map[string]any); ok {
		form.Fill(m)
	}

	content := fmt.Sprintf(`
<div class="filament-header">
  <div>
    <h1 class="filament-title">Edit %s #%s</h1>
    <p class="filament-description">Modify record fields and save changes.</p>
  </div>
  <div style="display: flex; gap: 0.5rem;">
    <a href="%s/%s/delete" class="btn btn-danger" onclick="return confirm('Delete this record?');">Delete</a>
    <a href="%s" class="btn btn-secondary">Back</a>
  </div>
</div>
<div class="filament-card">
  %s
  <div style="margin-top: 1.5rem; display: flex; justify-content: flex-end; gap: 0.75rem;">
    <a href="%s" class="btn btn-secondary">Cancel</a>
    <button type="submit" form="filament-form" class="btn btn-primary">Save Changes</button>
  </div>
</div>`,
		r.modelName, id, basePath, id, basePath,
		form.Render(ctx.GoContext()),
		basePath,
	)

	return r.renderPage(ctx, fmt.Sprintf("Edit %s #%s", r.modelName, id), content)
}

func (r *BaseResource[T]) handleUpdate(ctx *Context, basePath, id string) error {
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

	if action == "delete_bulk" && r.onDeleteBulk != nil && len(selectedIDs) > 0 {
		_ = r.onDeleteBulk(ctx, selectedIDs)
	}

	return ctx.Redirect(http.StatusSeeOther, basePath)
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
