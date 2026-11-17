package filament

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
)

// Saved views: a user names the current state of a resource list (search,
// sort, filters, advanced search, columns, grouping, page size) to come
// back to it, and can share it with everyone who can see the resource.
//
//	panel.SavedViews(nil)                 // in memory
//	panel.SavedViews(sqlStore.Views())    // in the database (pkg/sqlstore)

// ViewState is the list state a saved view restores.
type ViewState struct {
	Search         string            `json:"search,omitempty"`
	Sort           string            `json:"sort,omitempty"`
	Direction      string            `json:"direction,omitempty"`
	Filters        map[string]string `json:"filters,omitempty"`
	AdvancedSearch string            `json:"advanced_search,omitempty"`
	Columns        []string          `json:"columns,omitempty"`
	Group          string            `json:"group,omitempty"`
	PerPage        int               `json:"per_page,omitempty"`
}

// SavedView is a named list state of one resource.
type SavedView struct {
	ID        string    `json:"id"`
	Panel     string    `json:"panel"`
	Resource  string    `json:"resource"`
	Name      string    `json:"name"`
	OwnerID   string    `json:"owner_id"`
	Shared    bool      `json:"shared"`
	State     ViewState `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

// SavedViewStore keeps saved views.
type SavedViewStore interface {
	// ListViews returns ownerID's views of the resource plus the shared ones.
	ListViews(ctx context.Context, panel, resource, ownerID string) ([]SavedView, error)
	GetView(ctx context.Context, id string) (*SavedView, error)
	SaveView(ctx context.Context, v *SavedView) error
	DeleteView(ctx context.Context, id string) error
}

// ErrViewNotFound is returned for an unknown view.
var ErrViewNotFound = errors.New("view not found")

// MemoryViewStore keeps views in process memory.
type MemoryViewStore struct {
	mu    sync.RWMutex
	views map[string]SavedView
}

// NewMemoryViewStore returns an empty in-memory view store.
func NewMemoryViewStore() *MemoryViewStore {
	return &MemoryViewStore{views: map[string]SavedView{}}
}

func (m *MemoryViewStore) ListViews(_ context.Context, panel, resource, ownerID string) ([]SavedView, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []SavedView
	for _, v := range m.views {
		if v.Panel == panel && v.Resource == resource && (v.OwnerID == ownerID || v.Shared) {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

func (m *MemoryViewStore) GetView(_ context.Context, id string) (*SavedView, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.views[id]
	if !ok {
		return nil, ErrViewNotFound
	}
	return &v, nil
}

func (m *MemoryViewStore) SaveView(_ context.Context, v *SavedView) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.views[v.ID] = *v
	return nil
}

func (m *MemoryViewStore) DeleteView(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.views, id)
	return nil
}

// SavedViews turns on saved views, kept in store (nil: in memory).
func (p *Panel) SavedViews(store SavedViewStore) *Panel {
	if store == nil {
		store = NewMemoryViewStore()
	}
	p.viewStore = store
	return p
}

// SavedViewsShareable decides who may share views with everyone (default:
// any user who can see the resource) and delete other users' shared views
// (default: nobody but the owner).
func (p *Panel) SavedViewsShareable(canShare, canManageShared func(ctx *Context) bool) *Panel {
	p.canShareViews, p.canManageViews = canShare, canManageShared
	return p
}

func (p *Panel) mayShareViews(ctx *Context) bool {
	return p.canShareViews == nil || p.canShareViews(ctx)
}

func (p *Panel) mayDeleteView(ctx *Context, v *SavedView) bool {
	u := ctx.User()
	if u != nil && v.OwnerID == u.GetAuthIdentifier() {
		return true
	}
	return v.Shared && p.canManageViews != nil && p.canManageViews(ctx)
}

func viewOwner(ctx *Context) string {
	if u := ctx.User(); u != nil {
		return u.GetAuthIdentifier()
	}
	return ""
}

func newViewID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "view_" + hex.EncodeToString(b)
}

// viewStateFromQuery reads a list state from list query parameters.
func viewStateFromQuery(q url.Values) ViewState {
	s := ViewState{
		Search:         strings.TrimSpace(q.Get("search")),
		Sort:           q.Get("sort"),
		Direction:      q.Get("direction"),
		AdvancedSearch: q.Get("advanced_search"),
		Group:          q.Get("group"),
	}
	if s.Sort == "" {
		s.Direction = ""
	}
	for k, v := range q {
		if strings.HasPrefix(k, "filter_") && len(v) > 0 && v[0] != "" {
			if s.Filters == nil {
				s.Filters = map[string]string{}
			}
			s.Filters[strings.TrimPrefix(k, "filter_")] = strings.Join(v, ",")
		}
	}
	if c := q.Get("columns"); c != "" {
		s.Columns = strings.Split(c, ",")
	}
	if n, err := strconv.Atoi(q.Get("perPage")); err == nil && n > 0 {
		s.PerPage = n
	}
	return s
}

// Query returns the list URL query that shows the view.
func (s ViewState) Query() url.Values {
	q := url.Values{}
	set := func(k, v string) {
		if v != "" {
			q.Set(k, v)
		}
	}
	set("search", s.Search)
	set("sort", s.Sort)
	if s.Sort != "" {
		set("direction", s.Direction)
	}
	set("advanced_search", s.AdvancedSearch)
	set("group", s.Group)
	for k, v := range s.Filters {
		set("filter_"+k, v)
	}
	if len(s.Columns) > 0 {
		q.Set("columns", strings.Join(s.Columns, ","))
	}
	if s.PerPage > 0 {
		q.Set("perPage", strconv.Itoa(s.PerPage))
	}
	return q
}

func cleanViewState(s ViewState) ViewState {
	if len(s.Search) > 200 {
		s.Search = s.Search[:200]
	}
	if len(s.AdvancedSearch) > 8000 {
		s.AdvancedSearch = ""
	}
	if len(s.Filters) > 50 {
		s.Filters = nil
	}
	if len(s.Columns) > 100 {
		s.Columns = nil
	}
	if s.Direction != "" && s.Direction != "asc" && s.Direction != "desc" {
		s.Direction = "asc"
	}
	if s.PerPage < 0 || s.PerPage > 1000 {
		s.PerPage = 0
	}
	return s
}

// createView validates and stores a new view for the signed-in user.
func (p *Panel) createView(ctx *Context, res ResourceInterface, name string, shared bool, state ViewState) (*SavedView, string, int) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return nil, "Give the view a name (up to 80 characters).", http.StatusUnprocessableEntity
	}
	owner := viewOwner(ctx)
	if owner == "" {
		return nil, "Sign in to save views.", http.StatusUnauthorized
	}
	if shared && !p.mayShareViews(ctx) {
		return nil, "You cannot share views.", http.StatusForbidden
	}
	existing, err := p.viewStore.ListViews(ctx.GoContext(), p.id, res.Slug(), owner)
	if err != nil {
		return nil, "The view could not be saved.", http.StatusInternalServerError
	}
	mine := 0
	for _, v := range existing {
		if v.OwnerID == owner {
			mine++
			if strings.EqualFold(v.Name, name) {
				return nil, "You already have a view with this name.", http.StatusUnprocessableEntity
			}
		}
	}
	if mine >= 100 {
		return nil, "You have too many saved views; delete some first.", http.StatusUnprocessableEntity
	}
	v := &SavedView{ID: newViewID(), Panel: p.id, Resource: res.Slug(), Name: name, OwnerID: owner,
		Shared: shared, State: cleanViewState(state), CreatedAt: time.Now().UTC()}
	if err := p.viewStore.SaveView(ctx.GoContext(), v); err != nil {
		return nil, "The view could not be saved.", http.StatusInternalServerError
	}
	return v, "", http.StatusCreated
}

// deleteView removes a view the user may delete.
func (p *Panel) deleteView(ctx *Context, res ResourceInterface, id string) int {
	v, err := p.viewStore.GetView(ctx.GoContext(), id)
	if err != nil || v.Panel != p.id || v.Resource != res.Slug() {
		return http.StatusNotFound
	}
	if !p.mayDeleteView(ctx, v) {
		return http.StatusForbidden
	}
	if err := p.viewStore.DeleteView(ctx.GoContext(), id); err != nil {
		return http.StatusInternalServerError
	}
	return http.StatusOK
}

func (p *Panel) viewJSON(ctx *Context, v SavedView) map[string]any {
	owner := viewOwner(ctx)
	return map[string]any{
		"id": v.ID, "name": v.Name, "shared": v.Shared, "state": v.State,
		"mine": v.OwnerID == owner, "can_delete": p.mayDeleteView(ctx, &v),
	}
}

// canSeeResource is the list permission of a resource.
func canSeeResource(ctx *Context, res ResourceInterface) bool {
	if g, ok := res.(interface{ CanViewAnyRecords(ctx *Context) bool }); ok {
		return g.CanViewAnyRecords(ctx)
	}
	return true
}

func (p *Panel) mountSavedViewAPI(r *router.Router, res ResourceInterface, resPath string, preflight func(c *gcontext.Context) error) {
	if p.viewStore == nil {
		return
	}
	r.OPTIONS(resPath+"/views", preflight)
	r.OPTIONS(resPath+"/views/:view", preflight)
	r.GET(resPath+"/views", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		if !canSeeResource(ctx, res) {
			return viewFail(c, http.StatusForbidden, "Forbidden")
		}
		views, err := p.viewStore.ListViews(ctx.GoContext(), p.id, res.Slug(), viewOwner(ctx))
		if err != nil {
			return viewFail(c, http.StatusInternalServerError, "Views could not be loaded.")
		}
		out := []map[string]any{}
		for _, v := range views {
			out = append(out, p.viewJSON(ctx, v))
		}
		return c.JSON(http.StatusOK, map[string]any{"views": out, "can_share": p.mayShareViews(ctx)})
	})
	r.POST(resPath+"/views", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		if !canSeeResource(ctx, res) {
			return viewFail(c, http.StatusForbidden, "Forbidden")
		}
		var in struct {
			Name   string    `json:"name"`
			Shared bool      `json:"shared"`
			State  ViewState `json:"state"`
		}
		if err := c.BindJSON(&in); err != nil {
			return viewFail(c, http.StatusBadRequest, "Invalid request payload")
		}
		v, msg, status := p.createView(ctx, res, in.Name, in.Shared, in.State)
		if v == nil {
			return viewFail(c, status, msg)
		}
		return c.JSON(http.StatusCreated, p.viewJSON(ctx, *v))
	})
	r.DELETE(resPath+"/views/:view", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		switch status := p.deleteView(ctx, res, c.Param("view")); status {
		case http.StatusOK:
			return c.JSON(http.StatusOK, map[string]any{"success": true})
		case http.StatusForbidden:
			return viewFail(c, status, "You cannot delete this view.")
		case http.StatusNotFound:
			return viewFail(c, status, "View not found.")
		default:
			return viewFail(c, status, "The view could not be deleted.")
		}
	})
}

func (p *Panel) mountSavedViewWeb(r *router.Router, res ResourceInterface, basePath string) {
	if p.viewStore == nil {
		return
	}
	r.POST(basePath+"/views", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		if !p.formGuard(c) || !canSeeResource(ctx, res) {
			return c.String(http.StatusForbidden, "403 Forbidden")
		}
		q, _ := url.ParseQuery(c.FormValue("state"))
		v, msg, status := p.createView(ctx, res, c.FormValue("name"), c.FormValue("shared") != "", viewStateFromQuery(q))
		if v == nil {
			return c.String(status, "%s", msg)
		}
		return c.Redirect(http.StatusSeeOther, basePath+"?"+withView(v.State.Query(), v.ID).Encode())
	})
	r.POST(basePath+"/views/:view/delete", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		if !p.formGuard(c) {
			return c.String(http.StatusForbidden, "403 Forbidden")
		}
		if status := p.deleteView(ctx, res, c.Param("view")); status != http.StatusOK {
			return c.String(status, "The view could not be deleted.")
		}
		return c.Redirect(http.StatusSeeOther, basePath)
	})
}

func withView(q url.Values, id string) url.Values {
	q.Set("view", id)
	return q
}

// savedViewsHTML renders the Views menu of a web list.
func (p *Panel) savedViewsHTML(ctx *Context, res ResourceInterface, basePath string) string {
	if p.viewStore == nil || ctx.User() == nil {
		return ""
	}
	views, _ := p.viewStore.ListViews(ctx.GoContext(), p.id, res.Slug(), viewOwner(ctx))
	current := ctx.Query("view")
	label := "Views"
	var b strings.Builder
	esc := html.EscapeString
	csrf := p.csrfHiddenInput(ctx.Context, "")
	for _, v := range views {
		if v.ID == current {
			label = "View: " + v.Name
		}
	}
	fmt.Fprintf(&b, `<details class="filament-saved-views" style="position: relative; display: inline-block; margin-bottom: 0.75rem;"><summary class="btn btn-secondary btn-sm" style="list-style: none; cursor: pointer;">%s ▾</summary>
<div style="position: absolute; z-index: 40; min-width: 18rem; margin-top: 0.25rem; background: var(--white, #fff); border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 0.5rem; box-shadow: 0 8px 24px rgba(0,0,0,.08);">`, esc(label))
	fmt.Fprintf(&b, `<a href="%s" style="display: block; padding: 0.375rem 0.5rem; font-size: 0.875rem;">All %s</a>`, basePath, esc(strings.ToLower(res.GetPluralModelName())))
	for _, v := range views {
		shared := ""
		if v.Shared {
			shared = ` <span style="font-size: 0.75rem; color: var(--gray-500);">(shared)</span>`
		}
		fmt.Fprintf(&b, `<div style="display: flex; align-items: center; justify-content: space-between; gap: 0.5rem;" data-view="%s"><a href="%s?%s" style="padding: 0.375rem 0.5rem; font-size: 0.875rem;">%s%s</a>`,
			esc(v.ID), basePath, esc(withView(v.State.Query(), v.ID).Encode()), esc(v.Name), shared)
		if p.mayDeleteView(ctx, &v) {
			fmt.Fprintf(&b, `<form method="POST" action="%s/views/%s/delete" style="margin: 0;">%s<button type="submit" class="btn btn-secondary btn-sm" title="Delete view" onclick="return confirm('Delete this view?')">&times;</button></form>`,
				basePath, esc(v.ID), csrf)
		}
		b.WriteString(`</div>`)
	}
	state := viewStateFromQuery(ctx.Request.URL.Query())
	shareBox := ""
	if p.mayShareViews(ctx) {
		shareBox = `<label style="display: flex; gap: 0.375rem; align-items: center; font-size: 0.8125rem; margin: 0.375rem 0;"><input type="checkbox" name="shared" value="1" /> Share with everyone</label>`
	}
	fmt.Fprintf(&b, `<form method="POST" action="%s/views" style="border-top: 1px solid var(--gray-200); margin-top: 0.5rem; padding-top: 0.5rem;">%s<input type="hidden" name="state" value="%s" />
<input type="text" name="name" required maxlength="80" placeholder="Name this view" class="form-control" />%s
<button type="submit" class="btn btn-primary btn-sm" style="width: 100%%;">Save current view</button></form></div></details>`,
		basePath, csrf, esc(state.Query().Encode()), shareBox)
	return b.String()
}

func viewFail(c *gcontext.Context, status int, msg string) error {
	return c.JSON(status, map[string]string{"error": msg})
}

func savedViewsMenu(ctx *Context, res ResourceInterface, basePath string) string {
	if p := ctx.Panel(); p != nil {
		return p.savedViewsHTML(ctx, res, basePath)
	}
	return ""
}
