package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type MockShieldModel struct {
	ID    string
	Title string
	Owner string
}

func TestPanelCanAccessGuard(t *testing.T) {
	app := golaravel.New()

	panel := filament.NewPanel("admin").
		Path("/admin").
		Login().
		CanAccess(filament.Role("admin"))

	_ = panel.Register(app)
	client := app.Test(t)

	// 1. Unauthenticated request without user redirects to /admin/login
	resp := client.Get("/admin")
	resp.AssertStatus(http.StatusSeeOther)
	if !strings.HasSuffix(resp.Header().Get("Location"), "/admin/login") {
		t.Fatalf("expected redirect to /admin/login, got %s", resp.Header().Get("Location"))
	}

	// 2. Authenticated user without required role gets 403 Forbidden
	appUser := golaravel.New()
	appUser.Router.Use(func(c *gcontext.Context) error {
		c.Set(auth.ContextUserKey, &filament.SimpleShieldUser{
			ID:    "user_1",
			Email: "member@example.com",
			Roles: []string{"viewer"},
		})
		return c.Next()
	})

	panelUser := filament.NewPanel("admin").
		Path("/admin").
		CanAccess(filament.Role("admin"))
	_ = panelUser.Register(appUser)

	clientUser := appUser.Test(t)
	respUser := clientUser.Get("/admin")
	respUser.AssertStatus(http.StatusForbidden).
		AssertSee("Panel access denied")

	// 3. Authenticated admin user gets 200 OK
	appAdmin := golaravel.New()
	appAdmin.Router.Use(func(c *gcontext.Context) error {
		c.Set(auth.ContextUserKey, &filament.SimpleShieldUser{
			ID:    "admin_1",
			Email: "admin@example.com",
			Roles: []string{"admin"},
		})
		return c.Next()
	})

	panelAdmin := filament.NewPanel("admin").
		Path("/admin").
		CanAccess(filament.Role("admin"))
	_ = panelAdmin.Register(appAdmin)

	clientAdmin := appAdmin.Test(t)
	respAdmin := clientAdmin.Get("/admin")
	respAdmin.AssertOK().AssertSee("Dashboard")
}

func TestResourceViewAnyAndNavigationHiding(t *testing.T) {
	app := golaravel.New()

	res := filament.NewResource[MockShieldModel]("documents").
		CanViewAny(func(ctx *filament.Context) bool {
			return ctx.User() != nil && ctx.User().GetAuthIdentifier() == "allowed_user"
		})

	panel := filament.NewPanel("admin").
		Path("/admin").
		Resource(res)

	_ = panel.Register(app)

	// When user is not allowed
	req := httptest.NewRequest(http.MethodGet, "/admin/documents", nil)
	rec := httptest.NewRecorder()
	gc := newTestContext(rec, req)
	fCtx := filament.NewContext(gc, panel)

	// Navigation item is completely hidden
	navItems := panel.BuildNavigation(fCtx)
	for _, g := range navItems {
		for _, item := range g.Items {
			if item.Label == "Documents" {
				t.Fatalf("expected Documents to be hidden from navigation")
			}
		}
	}

	// Direct endpoint returns 403 Forbidden
	client := app.Test(t)
	resp := client.Get("/admin/documents")
	resp.AssertStatus(http.StatusForbidden).AssertSee("Access denied")
}

func TestResourceCreateAuthorization(t *testing.T) {
	app := golaravel.New()

	allowCreate := false
	res := filament.NewResource[MockShieldModel]("projects").
		OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]MockShieldModel, int, error) {
			return []MockShieldModel{{ID: "1", Title: "Project Alpha"}}, 1, nil
		}).
		CanCreate(func(ctx *filament.Context) bool {
			return allowCreate
		})

	panel := filament.NewPanel("admin").
		Path("/admin").
		Resource(res)

	_ = panel.Register(app)
	client := app.Test(t)

	// 1. In list view, "New Project" button is hidden when allowCreate is false
	respList := client.Get("/admin/projects")
	respList.AssertOK()
	if strings.Contains(respList.BodyString(), "New Project") {
		t.Fatalf("expected New Project button to be hidden")
	}

	// 2. Direct create GET returns 403
	respCreateGet := client.Get("/admin/projects/create")
	respCreateGet.AssertStatus(http.StatusForbidden)

	// 3. Direct create POST returns 403
	respCreatePost := client.Post("/admin/projects/create", strings.NewReader("title=Forbidden"))
	respCreatePost.AssertStatus(http.StatusForbidden)

	// 4. When allowCreate is true, button renders and form is accessible
	allowCreate = true
	respListAllowed := client.Get("/admin/projects")
	respListAllowed.AssertOK().AssertSee("New Project")

	respCreateGetAllowed := client.Get("/admin/projects/create")
	respCreateGetAllowed.AssertOK().AssertSee("Create Project")
}

func TestResourceViewEditDeleteGranularGuards(t *testing.T) {
	app := golaravel.New()

	mockDB := map[string]MockShieldModel{
		"101": {ID: "101", Title: "Public Doc", Owner: "john"},
		"102": {ID: "102", Title: "Secret Doc", Owner: "boss"},
	}

	res := filament.NewResource[MockShieldModel]("notes").
		OnFind(func(ctx *filament.Context, id string) (MockShieldModel, error) {
			return mockDB[id], nil
		}).
		CanView(func(ctx *filament.Context, record MockShieldModel) bool {
			return record.Owner != "boss"
		}).
		CanEdit(func(ctx *filament.Context, record MockShieldModel) bool {
			return record.Owner == "john"
		}).
		CanDelete(func(ctx *filament.Context, record MockShieldModel) bool {
			return false // Nobody can delete
		}).
		CanDeleteAny(func(ctx *filament.Context) bool {
			return false // No bulk delete
		})

	panel := filament.NewPanel("admin").
		Path("/admin").
		Resource(res)

	_ = panel.Register(app)
	client := app.Test(t)

	// 1. View allowed on record 101, forbidden on record 102
	client.Get("/admin/notes/101").AssertOK().AssertSee("View Notes #101")
	client.Get("/admin/notes/102").AssertStatus(http.StatusForbidden)

	// 2. Edit allowed on 101, forbidden on 102
	client.Get("/admin/notes/101/edit").AssertOK().AssertSee("Edit Notes #101")
	client.Get("/admin/notes/102/edit").AssertStatus(http.StatusForbidden)

	// Delete button is omitted on edit view when CanDelete returns false
	respEdit := client.Get("/admin/notes/101/edit")
	if strings.Contains(respEdit.BodyString(), `class="btn btn-danger"`) {
		t.Fatalf("expected delete button to be omitted on edit view")
	}

	// 3. Delete action forbidden
	client.Post("/admin/notes/101/delete", nil).AssertStatus(http.StatusForbidden)

	// 4. Bulk delete action forbidden
	client.Post("/admin/notes/bulk", strings.NewReader("bulk_action=delete_bulk&selected_ids[]=101")).
		AssertStatus(http.StatusForbidden)
}

// Custom Policy implementation
type MockPostPolicy struct {
	filament.BasePolicy[MockShieldModel]
	canCreate bool
}

func (p *MockPostPolicy) Create(ctx *filament.Context) bool {
	return p.canCreate
}

func (p *MockPostPolicy) Delete(ctx *filament.Context, record MockShieldModel) bool {
	return record.ID == "deletable"
}

func TestResourcePolicyBinding(t *testing.T) {
	app := golaravel.New()

	policy := &MockPostPolicy{canCreate: false}

	res := filament.NewResource[MockShieldModel]("posts").
		OnFind(func(ctx *filament.Context, id string) (MockShieldModel, error) {
			return MockShieldModel{ID: id, Title: "Test Post"}, nil
		}).
		Policy(policy)

	panel := filament.NewPanel("admin").
		Path("/admin").
		Resource(res)

	_ = panel.Register(app)
	client := app.Test(t)

	// Create is blocked by policy
	client.Get("/admin/posts/create").AssertStatus(http.StatusForbidden)

	// Delete is blocked on post 99, allowed on "deletable"
	client.Post("/admin/posts/99/delete", nil).AssertStatus(http.StatusForbidden)
	client.Post("/admin/posts/deletable/delete", nil).AssertStatus(http.StatusSeeOther)
}

func TestResourceGateBinding(t *testing.T) {
	app := golaravel.New()

	// Register gate abilities in GoLaravel's global auth.Gate
	auth.Define("viewAny_tasks", func(user auth.Authenticatable, args ...any) bool {
		return user != nil && user.GetAuthIdentifier() == "super_admin"
	})
	auth.Define("create_tasks", func(user auth.Authenticatable, args ...any) bool {
		return user != nil && user.GetAuthIdentifier() == "super_admin"
	})

	res := filament.NewResource[MockShieldModel]("tasks").
		AuthorizeWithGate("tasks")

	// 1. Regular user denied
	app.Router.Use(func(c *gcontext.Context) error {
		c.Set(auth.ContextUserKey, &filament.SimpleShieldUser{
			ID: "regular_user",
		})
		return c.Next()
	})

	panel := filament.NewPanel("admin").
		Path("/admin").
		Resource(res)

	_ = panel.Register(app)

	client := app.Test(t)
	client.Get("/admin/tasks").AssertStatus(http.StatusForbidden)
	client.Get("/admin/tasks/create").AssertStatus(http.StatusForbidden)

	// 2. Super admin allowed
	appSuper := golaravel.New()
	appSuper.Router.Use(func(c *gcontext.Context) error {
		c.Set(auth.ContextUserKey, &filament.SimpleShieldUser{
			ID: "super_admin",
		})
		return c.Next()
	})

	panelSuper := filament.NewPanel("admin").
		Path("/admin").
		Resource(res)
	_ = panelSuper.Register(appSuper)

	clientSuper := appSuper.Test(t)
	clientSuper.Get("/admin/tasks").AssertOK()
	clientSuper.Get("/admin/tasks/create").AssertOK()
}

func TestTableActionVisibilityCallback(t *testing.T) {
	table := tables.New[MockShieldModel]().
		Actions(
			tables.Action[MockShieldModel]("refund").
				Label("Refund").
				Visible(func(record MockShieldModel) bool {
					return record.Owner == "refundable"
				}),
		)

	recRefundable := MockShieldModel{ID: "1", Title: "Order 1", Owner: "refundable"}
	recNormal := MockShieldModel{ID: "2", Title: "Order 2", Owner: "normal"}

	table.SetData([]MockShieldModel{recRefundable, recNormal}, 2, 1, 10, "", "", "", nil)
	rendered := table.Render("/admin/orders")

	// Verify refund action button only appears once (for refundable record)
	count := strings.Count(rendered, "Refund")
	if count != 1 {
		t.Fatalf("expected Refund button to appear exactly once, appeared %d times", count)
	}
}
