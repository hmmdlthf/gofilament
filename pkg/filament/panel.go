package filament

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/golaravel/gofilament/pkg/assets"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/router"
)

type ContentWidth string

const (
	WidthFull ContentWidth = "max-w-full"
	Width7xl  ContentWidth = "max-w-7xl"
	Width6xl  ContentWidth = "max-w-6xl"
)

// NavigationGroup aggregates items under a named section in the sidebar.
type NavigationGroup struct {
	Name  string
	Items []*NavigationItem
}

// Panel represents an isolated administrative portal.
type Panel struct {
	id                  string
	path                string
	brandName           string
	brandLogo           string
	favicon             string
	homeURL             string
	colors              ColorPalette
	darkMode            bool
	sidebarCollapsible  bool
	topNavigation       bool
	maxContentWidth     ContentWidth
	navigationGroups    []string
	userMenu            *UserMenu
	hasNotifications    bool
	hasGlobalSearch     bool
	hasTenantScoping    bool
	tenantConfig        *TenantConfig
	hasLogin            bool
	hasRegistration     bool
	hasPasswordReset    bool
	hasProfile          bool
	authGuard           any
	resources           []ResourceInterface
	pages               []PageInterface
	widgets             []WidgetInterface
	dashboardRenderFunc func(ctx *Context) error
}

// NewPanel creates a new Panel with sensible defaults.
func NewPanel(id string) *Panel {
	p := &Panel{
		id:                  id,
		path:                "/" + strings.Trim(id, "/"),
		brandName:           "GoFilament",
		homeURL:             "/" + strings.Trim(id, "/"),
		colors:              DefaultColorPalette(),
		darkMode:            true,
		sidebarCollapsible:  true,
		maxContentWidth:     Width7xl,
		navigationGroups:    make([]string, 0),
		userMenu:            NewUserMenu(),
		resources:           make([]ResourceInterface, 0),
		pages:               make([]PageInterface, 0),
		widgets:             make([]WidgetInterface, 0),
	}

	// Default user menu
	p.userMenu.Item("Profile", p.path+"/profile").Icon("heroicon-user")
	p.userMenu.Item("Sign Out", p.path+"/logout").Icon("heroicon-arrow-left-on-rectangle").Color("danger")

	return p
}

// ID returns the unique identifier of the panel.
func (p *Panel) ID() string {
	return p.id
}

// GetPath returns the root URL path for this panel.
func (p *Panel) GetPath() string {
	return p.path
}

// Path sets the root URL path prefix.
func (p *Panel) Path(path string) *Panel {
	p.path = "/" + strings.Trim(path, "/")
	if p.homeURL == "" || p.homeURL == "/admin" {
		p.homeURL = p.path
	}
	return p
}

// BrandName sets the text branding displayed in the topbar and sidebar.
func (p *Panel) BrandName(name string) *Panel {
	p.brandName = name
	return p
}

// GetBrandName returns the panel brand name.
func (p *Panel) GetBrandName() string {
	return p.brandName
}

// BrandLogo sets an image URL for the brand logo.
func (p *Panel) BrandLogo(logo string) *Panel {
	p.brandLogo = logo
	return p
}

// Favicon sets the panel favicon icon path.
func (p *Panel) Favicon(fav string) *Panel {
	p.favicon = fav
	return p
}

// HomeURL sets the redirect URL when clicking the brand logo or home icon.
func (p *Panel) HomeURL(url string) *Panel {
	p.homeURL = url
	return p
}

// GetHomeURL returns the configured home URL.
func (p *Panel) GetHomeURL() string {
	return p.homeURL
}

// Colors overrides the default color palette.
func (p *Panel) Colors(palette ColorPalette) *Panel {
	p.colors = palette
	return p
}

// DarkMode toggles dark mode availability.
func (p *Panel) DarkMode(enabled bool) *Panel {
	p.darkMode = enabled
	return p
}

// IsDarkMode returns whether dark mode is supported.
func (p *Panel) IsDarkMode() bool {
	return p.darkMode
}

// SidebarCollapsibleOnDesktop configures if desktop sidebar can collapse to mini icon mode.
func (p *Panel) SidebarCollapsibleOnDesktop(enabled ...bool) *Panel {
	if len(enabled) > 0 {
		p.sidebarCollapsible = enabled[0]
	} else {
		p.sidebarCollapsible = true
	}
	return p
}

// TopNavigation switches sidebar navigation to topbar navigation mode.
func (p *Panel) TopNavigation(enabled ...bool) *Panel {
	if len(enabled) > 0 {
		p.topNavigation = enabled[0]
	} else {
		p.topNavigation = true
	}
	return p
}

// MaxContentWidth sets max container width class.
func (p *Panel) MaxContentWidth(w ContentWidth) *Panel {
	p.maxContentWidth = w
	return p
}

// NavigationGroups defines the preferred group ordering in the navigation sidebar.
func (p *Panel) NavigationGroups(groups []string) *Panel {
	p.navigationGroups = groups
	return p
}

// UserMenu customizes the profile dropdown items.
func (p *Panel) UserMenu(fn func(m *UserMenu)) *Panel {
	fn(p.userMenu)
	return p
}

// GetUserMenu returns the user menu.
func (p *Panel) GetUserMenu() *UserMenu {
	return p.userMenu
}

// DatabaseNotifications enables database notification bell & drawer.
func (p *Panel) DatabaseNotifications(enabled ...bool) *Panel {
	if len(enabled) > 0 {
		p.hasNotifications = enabled[0]
	} else {
		p.hasNotifications = true
	}
	return p
}

// GlobalSearch enables cross-resource debounced search in the topbar.
func (p *Panel) GlobalSearch(enabled ...bool) *Panel {
	if len(enabled) > 0 {
		p.hasGlobalSearch = enabled[0]
	} else {
		p.hasGlobalSearch = true
	}
	return p
}

// TenantScoping enables multi-tenant scoping.
func (p *Panel) TenantScoping(enabled ...bool) *Panel {
	if len(enabled) > 0 {
		p.hasTenantScoping = enabled[0]
	} else {
		p.hasTenantScoping = true
	}
	return p
}

// AuthGuard specifies an authentication guard.
func (p *Panel) AuthGuard(guard any) *Panel {
	p.authGuard = guard
	return p
}

// Login enables default login screen.
func (p *Panel) Login(enabled ...bool) *Panel {
	p.hasLogin = len(enabled) == 0 || enabled[0]
	return p
}

// Registration enables self-registration screen.
func (p *Panel) Registration(enabled ...bool) *Panel {
	p.hasRegistration = len(enabled) == 0 || enabled[0]
	return p
}

// PasswordReset enables forgot/reset password screen.
func (p *Panel) PasswordReset(enabled ...bool) *Panel {
	p.hasPasswordReset = len(enabled) == 0 || enabled[0]
	return p
}

// Profile enables self-service user profile screen.
func (p *Panel) Profile(enabled ...bool) *Panel {
	p.hasProfile = len(enabled) == 0 || enabled[0]
	return p
}

// Resource registers a single CRUD resource.
func (p *Panel) Resource(r ResourceInterface) *Panel {
	p.resources = append(p.resources, r)
	return p
}

// Resources registers multiple CRUD resources.
func (p *Panel) Resources(rs ...ResourceInterface) *Panel {
	p.resources = append(p.resources, rs...)
	return p
}

// Page registers a custom panel page.
func (p *Panel) Page(page PageInterface) *Panel {
	p.pages = append(p.pages, page)
	return p
}

// Pages registers multiple custom panel pages.
func (p *Panel) Pages(pages ...PageInterface) *Panel {
	p.pages = append(p.pages, pages...)
	return p
}

// Widget registers a dashboard widget.
func (p *Panel) Widget(w WidgetInterface) *Panel {
	p.widgets = append(p.widgets, w)
	return p
}

// Widgets registers multiple dashboard widgets.
func (p *Panel) Widgets(ws ...WidgetInterface) *Panel {
	p.widgets = append(p.widgets, ws...)
	return p
}

// GetWidgets returns all registered dashboard widgets.
func (p *Panel) GetWidgets() []WidgetInterface {
	return p.widgets
}

// CustomDashboard sets a custom dashboard rendering function.
func (p *Panel) CustomDashboard(fn func(ctx *Context) error) *Panel {
	p.dashboardRenderFunc = fn
	return p
}

// BuildNavigation constructs the grouped navigation structure for this panel.
func (p *Panel) BuildNavigation(ctx *Context) []NavigationGroup {
	var allItems []*NavigationItem

	basePrefix := p.path
	if p.tenantConfig != nil && p.tenantConfig.PathScoping && ctx.Tenant() != "" {
		basePrefix = fmt.Sprintf("%s/%s", p.path, ctx.Tenant())
	}

	// 1. Dashboard item
	currentPath := ""
	if ctx.Request != nil && ctx.Request.URL != nil {
		currentPath = ctx.Request.URL.Path
	}
	dashURL := basePrefix
	allItems = append(allItems, &NavigationItem{
		Label:  "Dashboard",
		URL:    dashURL,
		Icon:   "heroicon-home",
		Sort:   -100,
		Active: currentPath == dashURL || currentPath == dashURL+"/",
	})

	// 2. Resource items
	for _, res := range p.resources {
		item := res.NavigationItem(ctx)
		if item != nil {
			if item.Visible == nil || item.Visible(ctx) {
				if item.URL == "" {
					item.URL = fmt.Sprintf("%s/%s", basePrefix, res.Slug())
				}
				item.Active = strings.HasPrefix(currentPath, item.URL)
				allItems = append(allItems, item)
			}
		}
	}

	// 3. Custom page items
	for _, pg := range p.pages {
		item := pg.NavigationItem(ctx)
		if item != nil {
			if item.Visible == nil || item.Visible(ctx) {
				if item.URL == "" {
					item.URL = fmt.Sprintf("%s/%s", basePrefix, pg.Slug())
				}
				item.Active = strings.HasPrefix(currentPath, item.URL)
				allItems = append(allItems, item)
			}
		}
	}

	// Group items
	groupMap := make(map[string][]*NavigationItem)
	for _, item := range allItems {
		groupName := item.Group
		groupMap[groupName] = append(groupMap[groupName], item)
	}

	// Sort items in each group by Sort
	for _, items := range groupMap {
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].Sort < items[j].Sort
		})
	}

	var result []NavigationGroup
	// Add configured groups first in order
	usedGroups := make(map[string]bool)

	// Ungrouped items first (or dashboard)
	if items, ok := groupMap[""]; ok {
		result = append(result, NavigationGroup{Name: "", Items: items})
		usedGroups[""] = true
	}

	for _, g := range p.navigationGroups {
		if items, ok := groupMap[g]; ok {
			result = append(result, NavigationGroup{Name: g, Items: items})
			usedGroups[g] = true
		}
	}

	// Remaining groups
	for g, items := range groupMap {
		if !usedGroups[g] {
			result = append(result, NavigationGroup{Name: g, Items: items})
		}
	}

	return result
}

// Mount attaches all routes for this panel into the given GoLaravel router.
func (p *Panel) Mount(r *router.Router) {
	// 1. Static Assets
	assets.RegisterRoutes(r, p.path+"/assets")

	// Global Search Route
	if p.hasGlobalSearch {
		r.GET(p.path+"/global-search", func(c *gcontext.Context) error {
			ctx := NewContext(c, p)
			query := ctx.Query("query")
			var allResults []GlobalSearchResult
			if len(query) >= 2 {
				for _, res := range p.resources {
					if res.IsGloballySearchable() {
						matches := res.SearchGlobal(ctx, query)
						allResults = append(allResults, matches...)
					}
				}
			}

			// If client accepts JSON
			if strings.Contains(ctx.Header("Accept"), "application/json") {
				return ctx.JSON(http.StatusOK, allResults)
			}

			// Otherwise render dropdown HTML
			var sb strings.Builder
			if len(allResults) == 0 {
				sb.WriteString(`<div style="padding: 0.75rem 1rem; font-size: 0.8125rem; color: var(--gray-400);">No matching records found.</div>`)
			} else {
				for _, match := range allResults {
					sb.WriteString(fmt.Sprintf(`
<a href="%s" class="global-search-item" style="display: block; padding: 0.625rem 1rem; border-bottom: 1px solid var(--gray-100); text-decoration: none; color: inherit;">
  <span style="font-size: 0.75rem; font-weight: 600; text-transform: uppercase; color: var(--primary-600);">%s</span>
  <div style="font-size: 0.875rem; font-weight: 500;">%s</div>
</a>`, match.URL, match.Group, match.Title))
				}
			}
			return ctx.HTML(http.StatusOK, sb.String())
		})
	}

	// 2. Tenant Switch Route
	if p.tenantConfig != nil && p.tenantConfig.Enabled {
		tenantSwitchHandler := func(c *gcontext.Context) error {
			ctx := NewContext(c, p)
			targetKey := ctx.Query("tenant")
			if targetKey == "" {
				targetKey = ctx.FormValue("tenant")
			}

			if targetKey != "" {
				var targetTenant Tenant
				if p.tenantConfig.TenantLoader != nil {
					tenants, _ := p.tenantConfig.TenantLoader(ctx)
					for _, t := range tenants {
						if t.GetTenantKey() == targetKey {
							targetTenant = t
							break
						}
					}
				}
				if targetTenant == nil {
					targetTenant = &TenantItem{Key: targetKey, Name: targetKey}
				}

				c.SetCookie(p.tenantConfig.CookieName, targetKey, 86400*30, "/", "", false, true)

				ctx.setTenantInstance(targetTenant)

				if p.tenantConfig.OnTenantSwitch != nil {
					_ = p.tenantConfig.OnTenantSwitch(ctx, targetTenant)
				}
			}

			redirectURL := p.path
			if p.tenantConfig.PathScoping && targetKey != "" {
				redirectURL = fmt.Sprintf("%s/%s", p.path, targetKey)
			} else if ref := c.Header("Referer"); ref != "" {
				redirectURL = ref
			}

			return c.Redirect(http.StatusSeeOther, redirectURL)
		}

		r.GET(p.path+"/tenant/switch", tenantSwitchHandler)
		r.POST(p.path+"/tenant/switch", tenantSwitchHandler)
	}

	// 3. Auth Routes (Login, Logout, Profile)
	if p.hasLogin {
		r.GET(p.path+"/login", func(c *gcontext.Context) error {
			ctx := NewContext(c, p)
			return p.renderLoginScreen(ctx)
		})
		r.POST(p.path+"/login", func(c *gcontext.Context) error {
			// Redirect back to admin dashboard
			return c.Redirect(http.StatusSeeOther, p.path)
		})
		r.GET(p.path+"/logout", func(c *gcontext.Context) error {
			return c.Redirect(http.StatusSeeOther, p.path+"/login")
		})
	}

	// 4. Dashboard Route
	r.GET(p.path, func(c *gcontext.Context) error {
		ctx := NewContext(c, p)
		if p.dashboardRenderFunc != nil {
			return p.dashboardRenderFunc(ctx)
		}
		return p.renderDefaultDashboard(ctx)
	})

	// Also support trailing slash
	r.GET(p.path+"/", func(c *gcontext.Context) error {
		return c.Redirect(http.StatusMovedPermanently, p.path)
	})

	// 3. Register Resources
	for _, res := range p.resources {
		res.RegisterRoutes(p, r)
	}

	// 4. Register Pages
	for _, pg := range p.pages {
		pg.RegisterRoutes(p, r)
	}
}

// Register mounts the panel onto a GoLaravel Engine.
func (p *Panel) Register(app *golaravel.Engine) error {
	p.Mount(app.Router)
	return nil
}

// Default Dashboard fallback
func (p *Panel) renderDefaultDashboard(ctx *Context) error {
	title := "Dashboard"
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
        <div class="filament-header">
          <div>
            <h1 class="filament-title">%s</h1>
            <p class="filament-description">Welcome to your %s administration workspace.</p>
          </div>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div class="stat-card filament-card">
            <span class="text-sm font-medium text-gray-500">System Status</span>
            <span class="stat-value text-emerald-600 mt-2">Operational</span>
            <span class="text-xs text-gray-400 mt-1">GoLaravel Kernel Active</span>
          </div>
          <div class="stat-card filament-card">
            <span class="text-sm font-medium text-gray-500">Active Resources</span>
            <span class="stat-value mt-2">%d</span>
            <span class="text-xs text-gray-400 mt-1">Configured Ent entities</span>
          </div>
          <div class="stat-card filament-card">
            <span class="text-sm font-medium text-gray-500">Panel Architecture</span>
            <span class="stat-value mt-2">Compiled SSR</span>
            <span class="text-xs text-gray-400 mt-1">Zero-allocation Radix Router</span>
          </div>
        </div>
      </main>
    </div>
  </div>
</body>
</html>`,
		title, p.brandName,
		p.path, p.path,
		p.renderSidebarHTML(ctx),
		p.renderTopbarHTML(ctx),
		p.maxContentWidth,
		title, p.brandName,
		len(p.resources),
	)
	return ctx.HTML(http.StatusOK, html)
}

// GetMaxContentWidth returns the max width CSS class.
func (p *Panel) GetMaxContentWidth() string {
	return string(p.maxContentWidth)
}

func (p *Panel) RenderSidebar(ctx *Context) string {
	return p.renderSidebarHTML(ctx)
}

func (p *Panel) RenderTopbar(ctx *Context) string {
	return p.renderTopbarHTML(ctx)
}

func (p *Panel) renderSidebarHTML(ctx *Context) string {
	var sb strings.Builder
	sb.WriteString(`<aside class="filament-sidebar">`)
	sb.WriteString(fmt.Sprintf(`<div style="padding: 1.25rem 1.5rem; display: flex; align-items: center; gap: 0.75rem; border-bottom: 1px solid var(--gray-200);">
		<span style="font-weight: 700; font-size: 1.125rem; color: var(--primary-600);">%s</span>
	</div>`, p.brandName))
	sb.WriteString(`<nav style="padding: 1rem 0; flex: 1; overflow-y: auto;">`)

	groups := p.BuildNavigation(ctx)
	for _, g := range groups {
		if g.Name != "" {
			sb.WriteString(fmt.Sprintf(`<div class="nav-group-title">%s</div>`, g.Name))
		}
		for _, item := range g.Items {
			activeClass := ""
			if item.Active {
				activeClass = "active"
			}
			badgeHTML := ""
			if item.Badge != "" {
				badgeHTML = fmt.Sprintf(`<span class="nav-badge">%s</span>`, item.Badge)
			}
			sb.WriteString(fmt.Sprintf(`<a href="%s" class="nav-link %s"><span>%s</span>%s</a>`,
				item.URL, activeClass, item.Label, badgeHTML))
		}
	}

	sb.WriteString(`</nav></aside>`)
	return sb.String()
}

func (p *Panel) renderTopbarHTML(ctx *Context) string {
	var sb strings.Builder
	sb.WriteString(`<header class="filament-topbar">`)

	sb.WriteString(`<div style="display: flex; align-items: center; gap: 1rem; flex: 1;">`)
	sb.WriteString(`<button class="btn btn-secondary btn-sm" onclick="Filament.theme.toggle()" title="Toggle Dark/Light Mode">🌓</button>`)

	// Tenant Switcher dropdown
	if p.tenantConfig != nil && p.tenantConfig.Enabled {
		sb.WriteString(p.renderTenantSwitcherHTML(ctx))
	}

	// Global Search bar
	if p.hasGlobalSearch {
		sb.WriteString(fmt.Sprintf(`
<div style="position: relative; max-width: 24rem; width: 100%%;">
  <input type="text" placeholder="Search across %s..." class="form-control" style="font-size: 0.8125rem; padding: 0.375rem 0.75rem;" 
    oninput="
      const q = this.value;
      const dd = document.getElementById('global-search-results');
      if (q.length < 2) { dd.style.display = 'none'; return; }
      fetch('%s/global-search?query=' + encodeURIComponent(q))
        .then(r => r.text())
        .then(html => { dd.innerHTML = html; dd.style.display = 'block'; });
    "
    onblur="setTimeout(() => { document.getElementById('global-search-results').style.display = 'none'; }, 200);"
  />
  <div id="global-search-results" style="display: none; position: absolute; top: 100%%; left: 0; right: 0; background: #fff; border: 1px solid var(--gray-200); border-radius: 0.5rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1); margin-top: 0.25rem; z-index: 50; max-height: 18rem; overflow-y: auto;"></div>
</div>`, p.brandName, p.path))
	}
	sb.WriteString(`</div>`)

	sb.WriteString(`<div style="display: flex; align-items: center; gap: 1rem;">`)

	// Notifications bell icon
	if p.hasNotifications {
		sb.WriteString(`
<button type="button" class="btn btn-secondary btn-sm" style="position: relative;" onclick="Filament.notify({title:'Notifications', body:'You are all caught up!'})">
  🔔
  <span style="position: absolute; top: -0.25rem; right: -0.25rem; width: 0.625rem; height: 0.625rem; border-radius: 9999px; background-color: var(--primary-500);"></span>
</button>`)
	}

	// User menu
	sb.WriteString(`<div style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.875rem; font-weight: 500;">`)
	sb.WriteString(`<span>Administrator</span>`)
	sb.WriteString(`</div>`)
	sb.WriteString(`</div>`)

	sb.WriteString(`</header>`)
	return sb.String()
}

func (p *Panel) renderLoginScreen(ctx *Context) error {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Sign In - %s</title>
  <link rel="stylesheet" href="%s/assets/filament.css">
  <script src="%s/assets/filament.js"></script>
</head>
<body style="min-height: 100vh; display: flex; align-items: center; justify-content: center; background-color: var(--gray-50);">
  <div class="filament-card" style="width: 100%%; max-width: 26rem; padding: 2rem;">
    <div style="text-align: center; margin-bottom: 1.5rem;">
      <h2 style="font-size: 1.5rem; font-weight: 700; color: var(--primary-600);">%s</h2>
      <p style="font-size: 0.875rem; color: var(--gray-500); margin-top: 0.25rem;">Sign in to your administration panel</p>
    </div>
    <form method="POST" action="%s/login">
      <div class="form-group">
        <label class="form-label" for="email">Email address</label>
        <input type="email" id="email" name="email" required placeholder="admin@example.com" class="form-control" />
      </div>
      <div class="form-group">
        <label class="form-label" for="password">Password</label>
        <input type="password" id="password" name="password" required placeholder="••••••••" class="form-control" />
      </div>
      <div style="display: flex; align-items: center; justify-content: space-between; margin-bottom: 1.25rem;">
        <label style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.8125rem; cursor: pointer;">
          <input type="checkbox" name="remember" value="1" /> Remember me
        </label>
      </div>
      <button type="submit" class="btn btn-primary" style="width: 100%%;">Sign In</button>
    </form>
  </div>
</body>
</html>`, p.brandName, p.path, p.path, p.brandName, p.path)

	return ctx.HTML(http.StatusOK, html)
}
