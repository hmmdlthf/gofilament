package filament

import (
	"io/fs"
	"errors"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/assets"
	"github.com/golaravel/gofilament/pkg/colors"
	"github.com/golaravel/gofilament/pkg/globalsearch"
	"github.com/golaravel/gofilament/pkg/notifications"
	"github.com/golaravel/golaravel/pkg/appupdate"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/broadcast"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/flutter"
	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/router"
)

type ContentWidth string

const (
	WidthFull ContentWidth = "max-w-full"
	Width7xl  ContentWidth = "max-w-7xl"
	Width6xl  ContentWidth = "max-w-6xl"
)

// PanelEngine specifies the UI delivery engine for the admin panel.
type PanelEngine string

const (
	EngineWeb     PanelEngine = "web"     // Default: Templ SSR + Alpine.js + HTMX
	EngineFlutter PanelEngine = "flutter" // Flutter Multiplatform (Mobile-First Web, iOS, Android, Desktop)
)

// AccountNavPlacement specifies where the operator account entrypoint is displayed.
type AccountNavPlacement string

const (
	AccountNavHeader    AccountNavPlacement = "header"     // Circular avatar button in top header/AppBar (opens floating drawer)
	AccountNavBottomBar AccountNavPlacement = "bottom_bar" // Dedicated account tab in mobile bottom navigation bar (opens account page)
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
	engine              PanelEngine
	flutterDistDir      string
	flutterFS           fs.FS
	accountNavPlacement AccountNavPlacement
	brandName           string
	brandLogo           string
	favicon             string
	homeURL             string
	colors              map[string]colors.Palette
	fontFamily          string
	fontURL             string
	customStylesheets   []string
	customScripts       []string
	renderHooks         map[string][]func(ctx *Context) string
	darkMode            bool
	sidebarCollapsible  bool
	topNavigation       bool
	maxContentWidth     ContentWidth
	navigationGroups    []string
	userMenu            *UserMenu
	hasNotifications         bool
	hasGlobalSearch          bool
	globalSearchKeyBinding   string
	globalSearchResultsLimit int
	globalSearchFuzzy        bool
	hasSupport               bool
	supportScope             string
	supportSuperAdminPanelID string
	hasTenantScoping         bool
	tenantConfig        *TenantConfig
	hasLogin            bool
	hasRegistration     bool
	hasPasswordReset    bool
	hasProfile          bool
	requiresAuth        bool
	sessionDuration     time.Duration
	rememberMeDuration  time.Duration
	tabsEnabled         bool
	defaultListMode                   ListMode
	defaultPerPage                    int
	defaultPerPageOptions             []int
	defaultMobilePaginationPosition   PaginationPosition
	mobileBottomNavShowLabels         *bool
	mobileBottomNavItems              []string
	flutterLoadingConfig              flutter.LoadingConfig
	version                           string
	buildID                           string
	forceReloadOnUpdate               bool
	appUpdateManager                  *appupdate.Manager
	authGuard           any
	authValidator       AuthValidatorFunc
	tokenValidator      TokenValidatorFunc
	apiErrorMapper      APIErrorMapper
	brandResolver       BrandResolver
	maxPerPage          int
	currencyResolver    func(ctx *Context) string
	exportMaxRows       func(ctx *Context) int
	canAccess           func(ctx *Context) bool
	middlewares         []gcontext.HandlerFunc
	corsAllowedOrigins  []string
	webSessions         webSessionStore
	corsAllowOriginFunc func(origin string) bool
	resources           []ResourceInterface
	pages               []PageInterface
	endpoints           []*ClientEndpoint
	loginLimit          *loginThrottle
	uploadHandler       UploadHandler
	sessionStore        WebSessionStore
	sessionUsers        UserLoader
	widgets             []WidgetInterface
	dashboardRenderFunc func(ctx *Context) error
	locales             []string
	defaultLocale       string
	localeLabels        map[string]string
	localeFlags         map[string]string
	translations        map[string]map[string]string
}

// NewPanel creates a new Panel with sensible defaults.
func NewPanel(id string) *Panel {
	p := &Panel{
		id:                  id,
		path:                "/" + strings.Trim(id, "/"),
		engine:              EngineWeb,
		flutterDistDir:      "client/build/web",
		brandName:           "GoFilament",
		homeURL:             "/" + strings.Trim(id, "/"),
		colors: map[string]colors.Palette{
			colors.RolePrimary: colors.Emerald,
			colors.RoleGray:    colors.Slate,
			colors.RoleSuccess: colors.Emerald,
			colors.RoleWarning: colors.Amber,
			colors.RoleDanger:  colors.Red,
			colors.RoleInfo:    colors.Sky,
		},
		customStylesheets:   make([]string, 0),
		customScripts:       make([]string, 0),
		renderHooks:         make(map[string][]func(ctx *Context) string),
		darkMode:            true,
		sidebarCollapsible:  true,
		maxContentWidth:     Width7xl,
		navigationGroups:    make([]string, 0),
		userMenu:            NewUserMenu(),
		resources:           make([]ResourceInterface, 0),
		pages:               make([]PageInterface, 0),
		widgets:             make([]WidgetInterface, 0),
		locales:             make([]string, 0),
		defaultLocale:            "en",
		localeLabels:             make(map[string]string),
		localeFlags:              make(map[string]string),
		translations:             make(map[string]map[string]string),
		globalSearchKeyBinding:   "k",
		globalSearchResultsLimit: 10,
		globalSearchFuzzy:        true,
		sessionDuration:          24 * time.Hour,
		rememberMeDuration:       30 * 24 * time.Hour,
		tabsEnabled:              false,
		supportScope:             "standard",
		defaultListMode:                  ListModePagination,
		defaultPerPage:                   15,
		defaultPerPageOptions:            []int{10, 15, 25, 50, 100},
		defaultMobilePaginationPosition:  PaginationPositionTop,
		flutterLoadingConfig:             flutter.DefaultLoadingConfig(),
	}

	// Default user menu
	p.userMenu.Item("Profile", p.path+"/profile").Icon("heroicon-user")
	p.userMenu.Item("Sign Out", p.path+"/logout").Icon("heroicon-arrow-left-on-rectangle").Color("danger")

	return p
}

// DefaultListMode sets the default list mode across all resources in the panel.
func (p *Panel) DefaultListMode(mode ListMode) *Panel {
	p.defaultListMode = mode
	return p
}

// DefaultPerPage sets the panel-wide default records per page.
func (p *Panel) DefaultPerPage(n int) *Panel {
	if n > 0 {
		p.defaultPerPage = n
	}
	return p
}

// DefaultPerPageOptions sets the panel-wide selectable items per page choices.
func (p *Panel) DefaultPerPageOptions(opts ...int) *Panel {
	if len(opts) > 0 {
		p.defaultPerPageOptions = opts
	}
	return p
}

// GetDefaultListMode returns the panel-wide default ListMode.
func (p *Panel) GetDefaultListMode() ListMode {
	if p.defaultListMode == "" {
		return ListModePagination
	}
	return p.defaultListMode
}

// GetDefaultPerPage returns the panel-wide default items per page.
func (p *Panel) GetDefaultPerPage() int {
	if p.defaultPerPage <= 0 {
		return 15
	}
	return p.defaultPerPage
}

// GetDefaultPerPageOptions returns the panel-wide default items per page choices.
func (p *Panel) GetDefaultPerPageOptions() []int {
	if len(p.defaultPerPageOptions) == 0 {
		return []int{10, 15, 25, 50, 100}
	}
	return p.defaultPerPageOptions
}

// DefaultMobilePaginationPosition sets the panel-wide default position of pagination controls on mobile ("top" or "fab").
func (p *Panel) DefaultMobilePaginationPosition(pos PaginationPosition) *Panel {
	p.defaultMobilePaginationPosition = pos
	return p
}

// GetDefaultMobilePaginationPosition returns the panel-wide default mobile pagination position, defaulting to PaginationPositionTop ("top").
func (p *Panel) GetDefaultMobilePaginationPosition() PaginationPosition {
	if p.defaultMobilePaginationPosition == "" {
		return PaginationPositionTop
	}
	return p.defaultMobilePaginationPosition
}

// MobileBottomNavShowLabels sets whether text labels are visible in the mobile bottom navigation bar (default: false / icon-only).
func (p *Panel) MobileBottomNavShowLabels(show bool) *Panel {
	p.mobileBottomNavShowLabels = &show
	return p
}

// SetShowBottomNavLabels is an alias for MobileBottomNavShowLabels.
func (p *Panel) SetShowBottomNavLabels(show bool) *Panel {
	return p.MobileBottomNavShowLabels(show)
}

// IsBottomNavLabelsVisible returns whether mobile bottom navigation labels are visible (default: false).
func (p *Panel) IsBottomNavLabelsVisible() bool {
	if p.mobileBottomNavShowLabels != nil {
		return *p.mobileBottomNavShowLabels
	}
	return false
}

// MobileBottomNavItems sets the default 4 pinned destinations (slugs or special keys: "dashboard", "entities", "account") in the mobile bottom bar.
func (p *Panel) MobileBottomNavItems(items ...string) *Panel {
	if len(items) > 0 {
		p.mobileBottomNavItems = items
	}
	return p
}

// SetDefaultBottomNavItems is an alias for MobileBottomNavItems.
func (p *Panel) SetDefaultBottomNavItems(items ...string) *Panel {
	return p.MobileBottomNavItems(items...)
}

// GetMobileBottomNavItems returns the configured default mobile bottom navigation items.
func (p *Panel) GetMobileBottomNavItems() []string {
	if len(p.mobileBottomNavItems) == 0 {
		return nil
	}
	return p.mobileBottomNavItems
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

// Engine configures the UI delivery engine (EngineWeb or EngineFlutter).
func (p *Panel) Engine(engine PanelEngine) *Panel {
	p.engine = engine
	return p
}

// Flutter configures the panel to run in Flutter Multiplatform mode.
func (p *Panel) Flutter(distDir ...string) *Panel {
	p.engine = EngineFlutter
	if len(distDir) > 0 && distDir[0] != "" {
		p.flutterDistDir = distDir[0]
	}
	return p
}

// GetEngine returns the active PanelEngine.
func (p *Panel) GetEngine() PanelEngine {
	if p.engine == "" {
		return EngineWeb
	}
	return p.engine
}

// IsFlutter returns true if Flutter engine is selected.
func (p *Panel) IsFlutter() bool {
	return p.GetEngine() == EngineFlutter
}

// GetFlutterDistDir returns the Flutter build directory path.
func (p *Panel) GetFlutterDistDir() string {
	if p.flutterDistDir == "" {
		return "client/build/web"
	}
	return p.flutterDistDir
}

// FlutterLoading configures the visual pre-Flutter HTML/CSS loading screen.
func (p *Panel) FlutterLoading(fn func(cfg *flutter.LoadingConfig)) *Panel {
	if fn != nil {
		fn(&p.flutterLoadingConfig)
	}
	return p
}

// FlutterLoadingStyle sets the pre-Flutter loading animation style (spinner, pulse, bar, dots, minimal).
func (p *Panel) FlutterLoadingStyle(style flutter.LoadingStyle) *Panel {
	p.flutterLoadingConfig.Style = style
	return p
}

// FlutterLoadingSubtitle sets the subtext displayed under the brand title on the loading screen.
func (p *Panel) FlutterLoadingSubtitle(subtitle string) *Panel {
	p.flutterLoadingConfig.Subtitle = subtitle
	return p
}

// FlutterLoadingColors configures the background, primary accent, and text colors of the loading screen.
func (p *Panel) FlutterLoadingColors(bg, primary, text string) *Panel {
	if bg != "" {
		p.flutterLoadingConfig.BackgroundColor = bg
	}
	if primary != "" {
		p.flutterLoadingConfig.PrimaryColor = primary
	}
	if text != "" {
		p.flutterLoadingConfig.TextColor = text
	}
	return p
}

// GetFlutterLoadingConfig returns the loading screen configuration.
func (p *Panel) GetFlutterLoadingConfig() flutter.LoadingConfig {
	cfg := p.flutterLoadingConfig
	if cfg.BrandName == "" {
		cfg.BrandName = p.brandName
	}
	if cfg.LogoURL == "" && p.brandLogo != "" {
		cfg.LogoURL = p.brandLogo
	}
	return cfg
}

// Version sets the panel's semantic version (e.g. "1.4.0").
func (p *Panel) Version(v string) *Panel {
	p.version = v
	return p
}

// GetVersion returns the configured panel semantic version.
func (p *Panel) GetVersion() string {
	return p.version
}

// BuildID sets the build identifier (e.g. git hash or release timestamp).
func (p *Panel) BuildID(id string) *Panel {
	p.buildID = id
	return p
}

// GetBuildID returns the configured build identifier.
func (p *Panel) GetBuildID() string {
	return p.buildID
}

// RequireReloadOnUpdate specifies whether clients must reload immediately upon update detection.
func (p *Panel) RequireReloadOnUpdate(force bool) *Panel {
	p.forceReloadOnUpdate = force
	return p
}

// IsForceReloadOnUpdate reports whether reload is required on update.
func (p *Panel) IsForceReloadOnUpdate() bool {
	return p.forceReloadOnUpdate
}

// AppUpdate attaches an appupdate.Manager to the panel.
func (p *Panel) AppUpdate(mgr *appupdate.Manager) *Panel {
	p.appUpdateManager = mgr
	return p
}

// GetAppUpdateManager returns the configured appupdate.Manager, if any.
func (p *Panel) GetAppUpdateManager() *appupdate.Manager {
	return p.appUpdateManager
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

// AccountNavPlacement configures whether the operator account entrypoint is placed
// in the top header circle (AccountNavHeader) or in the bottom navigation bar (AccountNavBottomBar).
// This enforces an either-or design so both entrypoints are not shown at once.
func (p *Panel) AccountNavPlacement(placement AccountNavPlacement) *Panel {
	p.accountNavPlacement = placement
	return p
}

// GetAccountNavPlacement returns the configured account navigation placement, defaulting to AccountNavHeader.
func (p *Panel) GetAccountNavPlacement() AccountNavPlacement {
	if p.accountNavPlacement == "" {
		return AccountNavHeader
	}
	return p.accountNavPlacement
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

// GlobalSearchKeyBinding sets the keyboard shortcut trigger (defaults to "k", meaning ⌘K / Ctrl+K).
func (p *Panel) GlobalSearchKeyBinding(key string) *Panel {
	p.globalSearchKeyBinding = key
	return p
}

// GlobalSearchResultsLimit sets the maximum number of results to fetch per resource.
func (p *Panel) GlobalSearchResultsLimit(limit int) *Panel {
	p.globalSearchResultsLimit = limit
	return p
}

// GlobalSearchFuzzy enables or disables typo-tolerant fuzzy matching.
func (p *Panel) GlobalSearchFuzzy(enabled ...bool) *Panel {
	p.globalSearchFuzzy = len(enabled) == 0 || enabled[0]
	return p
}

// GetGlobalSearchKeyBinding returns the configured keyboard shortcut key.
func (p *Panel) GetGlobalSearchKeyBinding() string {
	if p.globalSearchKeyBinding != "" {
		return p.globalSearchKeyBinding
	}
	return "k"
}

// GetGlobalSearchResultsLimit returns the results limit per resource.
func (p *Panel) GetGlobalSearchResultsLimit() int {
	if p.globalSearchResultsLimit > 0 {
		return p.globalSearchResultsLimit
	}
	return 10
}

// GetGlobalSearchFuzzy returns whether fuzzy matching is active.
func (p *Panel) GetGlobalSearchFuzzy() bool {
	return p.globalSearchFuzzy
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

// EnableSupport enables the built-in AI concierge, multimodal feature requests, and live support escalation.
func (p *Panel) EnableSupport(enabled ...bool) *Panel {
	if len(enabled) > 0 {
		p.hasSupport = enabled[0]
	} else {
		p.hasSupport = true
	}
	return p
}

// SuperAdminSupport configures this panel as the Super Admin platform support hub, granting visibility across all tenants and tickets.
func (p *Panel) SuperAdminSupport(enabled ...bool) *Panel {
	p.supportScope = "super_admin"
	p.hasSupport = len(enabled) == 0 || enabled[0]
	return p
}

// TenantAdminSupport configures this panel as a Tenant Admin portal, scoping tickets to the active tenant and allowing escalation up to the platform super admin panel.
func (p *Panel) TenantAdminSupport(superAdminPanelID ...string) *Panel {
	p.supportScope = "tenant_admin"
	p.hasSupport = true
	if len(superAdminPanelID) > 0 && superAdminPanelID[0] != "" {
		p.supportSuperAdminPanelID = superAdminPanelID[0]
	}
	return p
}

// SupportScope explicitly sets the support hierarchy tier ("super_admin", "tenant_admin", "standard").
func (p *Panel) SupportScope(scope string) *Panel {
	p.supportScope = scope
	p.hasSupport = true
	return p
}

// IsSupportEnabled returns whether support concierge is enabled for this panel.
func (p *Panel) IsSupportEnabled() bool {
	return p.hasSupport
}

// GetSupportScope returns the configured support hierarchy tier ("super_admin", "tenant_admin", "standard").
func (p *Panel) GetSupportScope() string {
	if p.supportScope == "" {
		return "standard"
	}
	return p.supportScope
}

// GetSupportSuperAdminPanelID returns the upstream Super Admin panel ID for escalation.
func (p *Panel) GetSupportSuperAdminPanelID() string {
	return p.supportSuperAdminPanelID
}

// AuthValidatorFunc authenticates credentials and returns the user, token, and optional error.
type AuthValidatorFunc func(ctx *Context, username, password string) (auth.Authenticatable, string, error)

// TokenValidatorFunc verifies a Bearer token and returns the authenticatable user.
type TokenValidatorFunc func(ctx *Context, token string) (auth.Authenticatable, error)

// PanelUser represents a standardized user for Filament panels.
type PanelUser struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (u *PanelUser) GetAuthIdentifier() string {
	return u.Login
}

func (u *PanelUser) ToMap() map[string]any {
	return map[string]any{
		"id":    u.ID,
		"login": u.Login,
		"name":  u.Name,
		"email": u.Email,
		"role":  u.Role,
	}
}

// RequiresAuth enforces authentication for this panel.
func (p *Panel) RequiresAuth(enabled ...bool) *Panel {
	if len(enabled) > 0 {
		p.requiresAuth = enabled[0]
	} else {
		p.requiresAuth = true
	}
	return p
}

// IsAuthRequired returns whether authentication is enforced on this panel.
func (p *Panel) IsAuthRequired() bool {
	return p.requiresAuth
}

// AuthValidator registers a credential authenticator function.
func (p *Panel) AuthValidator(fn AuthValidatorFunc) *Panel {
	p.authValidator = fn
	p.requiresAuth = true
	return p
}

// TokenValidator registers a token validator function.
func (p *Panel) TokenValidator(fn TokenValidatorFunc) *Panel {
	p.tokenValidator = fn
	p.requiresAuth = true
	return p
}

// SessionDuration configures the standard session lifespan (when Remember Me is unchecked).
// Useful for banking or high-security applications where short timeouts (e.g. 5m, 15m) are required.
func (p *Panel) SessionDuration(d time.Duration) *Panel {
	p.sessionDuration = d
	return p
}

// RememberMeDuration configures the extended lifespan when "Remember Me" is selected.
func (p *Panel) RememberMeDuration(d time.Duration) *Panel {
	p.rememberMeDuration = d
	return p
}

// Tabs enables or disables resource status filter tabs globally across this panel.
func (p *Panel) Tabs(enabled ...bool) *Panel {
	if len(enabled) > 0 {
		p.tabsEnabled = enabled[0]
	} else {
		p.tabsEnabled = true
	}
	return p
}

// GetSessionDuration returns the panel session duration (default 24h).
func (p *Panel) GetSessionDuration() time.Duration {
	if p.sessionDuration <= 0 {
		return 24 * time.Hour
	}
	return p.sessionDuration
}

// GetRememberMeDuration returns the panel remember me duration (default 30d).
func (p *Panel) GetRememberMeDuration() time.Duration {
	if p.rememberMeDuration <= 0 {
		return 30 * 24 * time.Hour
	}
	return p.rememberMeDuration
}

// ResolveTokenDuration calculates the effective token duration based on remember-me flag.
func (p *Panel) ResolveTokenDuration(remember bool) time.Duration {
	if remember {
		return p.GetRememberMeDuration()
	}
	return p.GetSessionDuration()
}

// IsTabsEnabled returns true if status tabs are enabled for the panel.
func (p *Panel) IsTabsEnabled() bool {
	return p.tabsEnabled
}

// AuthGuard specifies an authentication guard.
func (p *Panel) AuthGuard(guard any) *Panel {
	p.authGuard = guard
	return p
}

// CanAccess configures an authorization guard for the entire panel.
func (p *Panel) CanAccess(fn func(ctx *Context) bool) *Panel {
	p.canAccess = fn
	return p
}

// Middleware appends router middleware to all non-asset routes on this panel,
// including the JSON API under <panel>/api. It must be called before the
// panel is mounted (Register / Mount): the router snapshots middleware per
// route at registration time.
//
// The middleware runs for requests whose path is the panel path or below it,
// except static assets (<panel>/assets/..., and the compiled Flutter bundle
// for the Flutter engine, where only <panel>/api/... is covered) and CORS
// preflight (OPTIONS) requests. Routes the application itself registers under
// the panel path after Mount are covered too.
func (p *Panel) Middleware(mw ...gcontext.HandlerFunc) *Panel {
	p.middlewares = append(p.middlewares, mw...)
	return p
}

// middlewareApplies reports whether panel middleware runs for a request.
func (p *Panel) middlewareApplies(method, reqPath string) bool {
	if method == http.MethodOptions {
		return false
	}
	base := strings.TrimRight(p.path, "/")
	if base != "" && reqPath != base && !strings.HasPrefix(reqPath, base+"/") {
		return false
	}
	rest := strings.TrimPrefix(reqPath, base)
	if strings.HasPrefix(rest, "/api/") || rest == "/api" {
		return true
	}
	if p.IsFlutter() {
		// Everything outside the API is the static Flutter bundle.
		return false
	}
	return !strings.HasPrefix(rest, "/assets/")
}

// installMiddleware attaches the panel middleware to the router so every
// route registered afterwards runs it for requests inside this panel.
func (p *Panel) installMiddleware(r *router.Router) {
	for _, mw := range p.middlewares {
		mw := mw
		r.Use(func(c *gcontext.Context) error {
			if c.Request == nil || c.Request.URL == nil || !p.middlewareApplies(c.Request.Method, c.Request.URL.Path) {
				return nil
			}
			return mw(c)
		})
	}
}

// ErrAccessDenied is returned by AuthorizeAccess after it has written the
// 401 / 403 response (or the redirect to the login page). Handlers must stop
// and return it: the request is already answered.
var ErrAccessDenied = errors.New("filament: access denied")

func denied(err error) error {
	if err != nil {
		return err
	}
	return ErrAccessDenied
}

// resolveUser builds the request context and authenticates a bearer token
// when one is sent, or, outside <panel>/api, the web-engine login cookie set
// by POST <panel>/login. It never writes a response.
func (p *Panel) resolveUser(c *gcontext.Context) *Context {
	ctx := NewContext(c, p)
	if u := clientGeneratorUser(c.Request); u != nil {
		c.Set(auth.ContextUserKey, u)
		return ctx
	}
	authHeader := c.Header("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") && p.tokenValidator != nil {
		if u, err := p.tokenValidator(ctx, strings.TrimPrefix(authHeader, "Bearer ")); err == nil && u != nil {
			c.Set(auth.ContextUserKey, u)
		}
	}
	if ctx.User() == nil && !p.isAPIRequest(c) {
		if u := p.userFromSessionCookie(ctx); u != nil {
			c.Set(auth.ContextUserKey, u)
		}
	}
	return ctx
}

// isAPIRequest reports whether the request targets <panel>/api.
func (p *Panel) isAPIRequest(c *gcontext.Context) bool {
	if c.Request == nil || c.Request.URL == nil {
		return false
	}
	rest := strings.TrimPrefix(c.Request.URL.Path, strings.TrimRight(p.path, "/"))
	return rest == "/api" || strings.HasPrefix(rest, "/api/")
}

// AuthorizeAccess verifies panel authorization for the incoming request.
// When access is refused it writes the response and returns ErrAccessDenied
// (or the write error): callers must return immediately on any error.
func (p *Panel) AuthorizeAccess(c *gcontext.Context) (*Context, error) {
	ctx := p.resolveUser(c)

	reqPath := ""
	if c.Request != nil && c.Request.URL != nil {
		reqPath = c.Request.URL.Path
	}
	if p.requiresAuth && ctx.User() == nil {
		if strings.Contains(reqPath, "/api/") {
			p.setCorsHeaders(c)
			return ctx, denied(c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized: Authentication required"}))
		}
		if p.hasLogin {
			return ctx, denied(c.Redirect(http.StatusSeeOther, p.path+"/login"))
		}
		return ctx, denied(c.String(http.StatusUnauthorized, "401 Unauthorized: Authentication required"))
	}

	if p.canAccess != nil && !p.canAccess(ctx) {
		if strings.Contains(reqPath, "/api/") {
			p.setCorsHeaders(c)
			return ctx, denied(c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden: Panel access denied"}))
		}
		if p.hasLogin && ctx.User() == nil {
			return ctx, denied(c.Redirect(http.StatusSeeOther, p.path+"/login"))
		}
		return ctx, denied(c.String(http.StatusForbidden, "403 Forbidden: Panel access denied"))
	}

	return ctx, nil
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
	if err := p.ValidateTranslations(); err != nil {
		panic(err)
	}

	// 0. Panel middleware (must precede every route registration below).
	p.installMiddleware(r)

	// 1. Standardized Panel REST API (powers Flutter, mobile apps, and headless integrations)
	p.RegisterAPIRoutes(r)

	// 2. If Flutter engine is selected, mount the Flutter Web application (handles assets & SPA)
	if p.IsFlutter() {
		p.mountFlutter(r)
		return
	}

	// 3. Static Assets for Web engine
	assets.RegisterRoutes(r, p.path+"/assets")

	// Global Search Route
	if p.hasGlobalSearch {
		r.GET(p.path+"/global-search", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			query := ctx.Query("query")
			if query == "" {
				query = ctx.Query("q")
			}

			groups, allResults := p.searchAll(ctx, query)

			// If client accepts JSON
			if strings.Contains(ctx.Header("Accept"), "application/json") {
				return ctx.JSON(http.StatusOK, allResults)
			}

			// Render Spotlight HTML
			var dtos []globalsearch.GroupDTO
			for _, g := range groups {
				var items []globalsearch.ItemDTO
				for _, it := range g.Items {
					var actions []globalsearch.ActionDTO
					for _, a := range it.Actions {
						actions = append(actions, globalsearch.ActionDTO{
							Label: a.Label,
							URL:   a.URL,
							Icon:  a.Icon,
						})
					}
					items = append(items, globalsearch.ItemDTO{
						Title:   it.Title,
						URL:     it.URL,
						Details: it.Details,
						Group:   it.Group,
						Icon:    it.Icon,
						Badge:   it.Badge,
						Actions: actions,
					})
				}
				dtos = append(dtos, globalsearch.GroupDTO{
					Name:  g.Name,
					Count: g.Count,
					Items: items,
				})
			}

			return ctx.HTML(http.StatusOK, globalsearch.RenderResultsHTML(dtos, query))
		})
	}

	// 2. Tenant Switch Route
	if p.tenantConfig != nil && p.tenantConfig.Enabled {
		tenantSwitchHandler := func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
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

	// 2b. Locale Switch Routes
	localeSwitchHandler := func(c *gcontext.Context) error {
		targetLang := c.Query("lang")
		if targetLang == "" {
			targetLang = c.Query("locale")
		}
		if targetLang == "" {
			targetLang = c.FormValue("lang")
		}
		if targetLang == "" {
			targetLang = c.FormValue("locale")
		}

		if targetLang != "" {
			// Validate if configured locales are defined
			if len(p.locales) > 0 {
				matched := false
				for _, loc := range p.locales {
					if strings.EqualFold(loc, targetLang) {
						targetLang = loc
						matched = true
						break
					}
				}
				if !matched {
					if p.defaultLocale != "" {
						targetLang = p.defaultLocale
					} else {
						targetLang = p.locales[0]
					}
				}
			}

			c.SetCookie("locale", targetLang, 86400*365, "/", "", false, false)
			c.SetLocale(targetLang)
		}

		redirectURL := p.path
		if ref := c.Header("Referer"); ref != "" {
			redirectURL = ref
		}
		return c.Redirect(http.StatusSeeOther, redirectURL)
	}

	r.GET(p.path+"/locale/switch", localeSwitchHandler)
	r.POST(p.path+"/locale/switch", localeSwitchHandler)

	// 3. Auth Routes (Login, Logout, Profile)
	if p.hasLogin {
		r.GET(p.path+"/login", func(c *gcontext.Context) error {
			ctx := p.resolveUser(c)
			if ctx.User() != nil {
				return c.Redirect(http.StatusSeeOther, p.path)
			}
			return p.renderLoginScreen(ctx, http.StatusOK, "", "")
		})
		// Authenticates with the AuthValidator and starts a browser session
		// (HttpOnly, SameSite=Lax cookie; see SessionCookieName).
		r.POST(p.path+"/login", p.handleWebLogin)
		logout := func(c *gcontext.Context) error {
			p.endWebSession(c)
			return c.Redirect(http.StatusSeeOther, p.path+"/login")
		}
		r.GET(p.path+"/logout", logout)
		r.POST(p.path+"/logout", logout)
	}

	// 4. Dashboard Route
	r.GET(p.path, func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		if p.dashboardRenderFunc != nil {
			return p.dashboardRenderFunc(ctx)
		}
		return p.renderDefaultDashboard(ctx)
	})

	// Also support trailing slash
	r.GET(p.path+"/", func(c *gcontext.Context) error {
		return c.Redirect(http.StatusMovedPermanently, p.path)
	})

	// 5. Dynamic Widget Data Endpoint (AJAX filter updates & live metric polling)
	r.GET(p.path+"/widgets/:id/data", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		widgetID := c.Param("id")
		filter := ctx.Query("filter")
		if filter == "" {
			filter = ctx.DefaultQuery("period", "")
		}

		for _, w := range p.widgets {
			if w.ID() == widgetID {
				if chartWidget, ok := w.(interface {
					EvaluateChartData(ctx *Context, filter string) any
				}); ok {
					data := chartWidget.EvaluateChartData(ctx, filter)
					return c.JSON(http.StatusOK, data)
				}
			}
		}
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Widget not found"})
	})

	// 6. Database Notifications Routes
	if p.hasNotifications {
		r.GET(p.path+"/notifications", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			user := "all"
			if ctx.User() != nil {
				user = ctx.User().GetAuthIdentifier()
			}
			items, _ := notifications.DefaultStore().List(user, 50)
			unreadCount, _ := notifications.DefaultStore().CountUnread(user)

			if strings.Contains(c.Header("Accept"), "application/json") {
				return c.JSON(http.StatusOK, map[string]any{
					"unread_count": unreadCount,
					"items":        items,
				})
			}
			return c.HTML(http.StatusOK, p.renderNotificationsDrawerHTML(ctx))
		})

		r.POST(p.path+"/notifications/mark-read", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			user := "all"
			if ctx.User() != nil {
				user = ctx.User().GetAuthIdentifier()
			}
			id := c.Query("id")
			if id == "" {
				id = c.FormValue("id")
			}
			if id == "" {
				var body map[string]string
				_ = json.NewDecoder(c.Request.Body).Decode(&body)
				id = body["id"]
			}
			_ = notifications.DefaultStore().MarkAsRead(user, id)
			return c.JSON(http.StatusOK, map[string]any{"success": true})
		})

		r.POST(p.path+"/notifications/mark-all-read", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			user := "all"
			if ctx.User() != nil {
				user = ctx.User().GetAuthIdentifier()
			}
			_ = notifications.DefaultStore().MarkAllAsRead(user)
			return c.JSON(http.StatusOK, map[string]any{"success": true})
		})

		r.POST(p.path+"/notifications/clear", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			user := "all"
			if ctx.User() != nil {
				user = ctx.User().GetAuthIdentifier()
			}
			_ = notifications.DefaultStore().Clear(user)
			return c.JSON(http.StatusOK, map[string]any{"success": true})
		})

		r.GET(p.path+"/notifications/stream", func(c *gcontext.Context) error {
			user := "all"
			if authUser := c.Header("X-User-ID"); authUser != "" {
				user = authUser
			}
			channel := "filament:notifications:" + user
			c.Request.URL.RawQuery = "channel=" + channel
			return broadcast.SSEHandler()(c)
		})
	}

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
	if err := p.ValidateTranslations(); err != nil {
		return err
	}
	p.Mount(app.Router)
	return nil
}

// Default Dashboard fallback
func (p *Panel) renderDefaultDashboard(ctx *Context) error {
	title := "Dashboard"

	var dashboardContent string
	if len(p.widgets) > 0 {
		sortedWidgets := make([]WidgetInterface, len(p.widgets))
		copy(sortedWidgets, p.widgets)
		sort.Slice(sortedWidgets, func(i, j int) bool {
			return sortedWidgets[i].Sort() < sortedWidgets[j].Sort()
		})

		var sb strings.Builder
		sb.WriteString(`<div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">`)
		for _, w := range sortedWidgets {
			spanClass := "col-span-1"
			if w.ColumnSpan() >= 3 {
				spanClass = "col-span-1 md:col-span-3"
			} else if w.ColumnSpan() == 2 {
				spanClass = "col-span-1 md:col-span-2"
			}
			if renderer, ok := w.(interface{ Render(ctx *Context) string }); ok {
				sb.WriteString(fmt.Sprintf(`<div class="%s">%s</div>`, spanClass, renderer.Render(ctx)))
			}
		}
		sb.WriteString(`</div>`)
		dashboardContent = sb.String()
	} else {
		dashboardContent = fmt.Sprintf(`
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
        </div>`, len(p.resources))
	}

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
        <div class="filament-header">
          <div>
            <h1 class="filament-title">%s</h1>
            <p class="filament-description">Welcome to your %s administration workspace.</p>
          </div>
        </div>
        %s
      </main>
    </div>
  </div>
  %s
</body>
</html>`,
		ctx.Locale(), ctx.Direction(),
		title, p.brandName,
		p.path, p.path,
		p.RenderHeadHTML(ctx),
		p.renderSidebarHTML(ctx),
		p.renderTopbarHTML(ctx),
		p.maxContentWidth,
		title, p.brandName,
		dashboardContent,
		p.RenderBodyEndHTML(ctx),
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
	sb.WriteString(`<aside class="filament-sidebar" id="filament-sidebar">`)
	sb.WriteString(fmt.Sprintf(`<div style="padding: 1.25rem 1.5rem; display: flex; align-items: center; justify-content: space-between; border-bottom: 1px solid var(--gray-200);">
		<span style="font-weight: 700; font-size: 1.125rem; color: var(--primary-600);">%s</span>
		<button type="button" class="filament-sidebar-close-btn" onclick="Filament.sidebar.close()" aria-label="Close Sidebar">&times;</button>
	</div>`, p.brandName))
	sb.WriteString(`<nav style="padding: 1rem 0; flex: 1; overflow-y: auto;">`)

	if hook := p.renderHook(PanelsRenderHookSidebarNavStart, ctx); hook != "" {
		sb.WriteString(hook)
	}

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

	if hook := p.renderHook(PanelsRenderHookSidebarNavEnd, ctx); hook != "" {
		sb.WriteString(hook)
	}

	sb.WriteString(`</nav></aside>`)
	return sb.String()
}

func (p *Panel) renderTopbarHTML(ctx *Context) string {
	var sb strings.Builder
	sb.WriteString(`<header class="filament-topbar">`)

	if hook := p.renderHook(PanelsRenderHookTopbarStart, ctx); hook != "" {
		sb.WriteString(hook)
	}

	sb.WriteString(`<div style="display: flex; align-items: center; gap: 0.75rem; flex: 1;">`)
	sb.WriteString(`<button type="button" class="btn btn-secondary btn-sm filament-mobile-menu-btn" onclick="Filament.sidebar.toggle()" title="Toggle Navigation Menu" aria-label="Toggle Navigation"><svg style="width: 1.25rem; height: 1.25rem;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" /></svg></button>`)
	sb.WriteString(`<button class="btn btn-secondary btn-sm" onclick="Filament.theme.toggle()" title="Toggle Dark/Light Mode">🌓</button>`)

	// Tenant Switcher dropdown
	if p.tenantConfig != nil && p.tenantConfig.Enabled {
		sb.WriteString(p.renderTenantSwitcherHTML(ctx))
	}

	// Global Search Command Bar Trigger
	if p.hasGlobalSearch {
		sb.WriteString(globalsearch.RenderTriggerHTML(p.brandName, p.path, p.GetGlobalSearchKeyBinding()))
	}
	sb.WriteString(`</div>`)

	sb.WriteString(`<div style="display: flex; align-items: center; gap: 1rem;">`)

	// Notifications bell icon
	if p.hasNotifications {
		user := "all"
		if ctx.User() != nil {
			user = ctx.User().GetAuthIdentifier()
		}
		unreadCount, _ := notifications.DefaultStore().CountUnread(user)
		badgeDisplay := "display: none;"
		if unreadCount > 0 {
			badgeDisplay = "display: inline-flex;"
		}
		sb.WriteString(fmt.Sprintf(`
<button type="button" class="btn btn-secondary btn-sm" style="position: relative;" onclick="Filament.openNotificationDrawer()" title="Notifications">
  🔔
  <span id="notif-unread-badge" class="badge-count" style="%s">%d</span>
</button>`, badgeDisplay, unreadCount))
	}

	// Language / Locale switcher dropdown
	if len(p.locales) > 1 {
		sb.WriteString(p.renderLocaleSwitcherHTML(ctx))
	}

	// User menu
	sb.WriteString(`<div style="display: flex; align-items: center; gap: 0.5rem; font-size: 0.875rem; font-weight: 500;">`)
	sb.WriteString(`<span>Administrator</span>`)
	sb.WriteString(`</div>`)
	sb.WriteString(`</div>`)

	if hook := p.renderHook(PanelsRenderHookTopbarEnd, ctx); hook != "" {
		sb.WriteString(hook)
	}

	sb.WriteString(`</header>`)

	// Append notification drawer and SSE script if notifications enabled
	if p.hasNotifications {
		sb.WriteString(p.renderNotificationsDrawerHTML(ctx))
	}

	return sb.String()
}

// searchAll executes cross-resource and navigation fuzzy global search.
func (p *Panel) searchAll(ctx *Context, query string) ([]GlobalSearchGroup, []GlobalSearchResult) {
	query = strings.TrimSpace(query)
	if len(query) < 2 {
		return nil, nil
	}

	var allResults []GlobalSearchResult
	var navResults []GlobalSearchResult

	// 1. Search Navigation items
	navGroups := p.BuildNavigation(ctx)
	for _, grp := range navGroups {
		for _, item := range grp.Items {
			score, matched := globalsearch.Score(query, item.Label)
			if !matched && p.globalSearchFuzzy {
				if gScore, gMatched := globalsearch.Score(query, grp.Name); gMatched {
					score = gScore * 0.8
					matched = true
				}
			}
			if matched {
				res := GlobalSearchResult{
					Title: item.Label,
					URL:   item.URL,
					Group: "Navigation",
					Icon:  item.Icon,
					Badge: item.Badge,
					Score: score,
				}
				navResults = append(navResults, res)
			}
		}
	}

	// Also search Custom Pages
	for _, page := range p.pages {
		score, matched := globalsearch.Score(query, page.Title())
		if matched {
			res := GlobalSearchResult{
				Title: page.Title(),
				URL:   p.path + "/" + page.Slug(),
				Group: "Navigation",
				Score: score,
			}
			navResults = append(navResults, res)
		}
	}

	if len(navResults) > 0 {
		sort.SliceStable(navResults, func(i, j int) bool {
			return navResults[i].Score > navResults[j].Score
		})
		allResults = append(allResults, navResults...)
	}

	// 2. Search Resources
	var resourceGroups []GlobalSearchGroup
	for _, res := range p.resources {
		if res.IsGloballySearchable() {
			matches := res.SearchGlobal(ctx, query)
			if len(matches) > 0 {
				if p.globalSearchFuzzy {
					sort.SliceStable(matches, func(i, j int) bool {
						return matches[i].Score > matches[j].Score
					})
				}
				allResults = append(allResults, matches...)
				resourceGroups = append(resourceGroups, GlobalSearchGroup{
					Name:  res.GetPluralModelName(),
					Count: len(matches),
					Items: matches,
				})
			}
		}
	}

	var groups []GlobalSearchGroup
	if len(navResults) > 0 {
		groups = append(groups, GlobalSearchGroup{
			Name:  "Navigation",
			Count: len(navResults),
			Items: navResults,
		})
	}
	groups = append(groups, resourceGroups...)

	return groups, allResults
}

func (p *Panel) renderNotificationsDrawerHTML(ctx *Context) string {
	user := "all"
	if ctx.User() != nil {
		user = ctx.User().GetAuthIdentifier()
	}

	unreadCount, _ := notifications.DefaultStore().CountUnread(user)
	items, _ := notifications.DefaultStore().List(user, 50)

	var listHTML strings.Builder
	if len(items) == 0 {
		listHTML.WriteString(`<div class="empty-notif-msg" style="text-align: center; color: var(--gray-400); padding: 3rem 1rem;">No notifications yet.</div>`)
	} else {
		for _, it := range items {
			unreadClass := ""
			if !it.Read {
				unreadClass = "unread"
			}
			actionBtn := ""
			if it.ActionURL != "" {
				lbl := it.ActionLabel
				if lbl == "" {
					lbl = "View"
				}
				actionBtn = fmt.Sprintf(`<a href="%s" class="btn btn-primary btn-xs" style="margin-top: 0.5rem; text-decoration: none; display: inline-block;">%s</a>`, html.EscapeString(it.ActionURL), html.EscapeString(lbl))
			}
			markReadBtn := ""
			if !it.Read {
				markReadBtn = fmt.Sprintf(`<button type="button" class="btn btn-secondary btn-xs" onclick="Filament.markNotificationRead('%s/notifications/mark-read', '%s')" style="margin-top: 0.5rem;">Mark as read</button>`, p.path, it.ID)
			}
			timeStr := it.CreatedAt.Format("Jan 02, 15:04")

			bodyHTML := ""
			if it.BodyText != "" {
				bodyHTML = fmt.Sprintf(`<div style="font-size: 0.8125rem; color: var(--gray-500); margin-top: 0.25rem;">%s</div>`, html.EscapeString(it.BodyText))
			}

			listHTML.WriteString(fmt.Sprintf(`
<div id="notif-item-%s" class="notification-item %s">
  <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 0.5rem;">
    <div style="font-weight: 600; font-size: 0.875rem;">%s</div>
    <span style="font-size: 0.6875rem; color: var(--gray-400);">%s</span>
  </div>
  %s
  <div style="display: flex; gap: 0.5rem; align-items: center;">
    %s
    %s
  </div>
</div>`,
				it.ID, unreadClass,
				html.EscapeString(it.Title),
				timeStr,
				bodyHTML,
				actionBtn,
				markReadBtn,
			))
		}
	}

	unreadBadge := fmt.Sprintf("%d unread", unreadCount)
	return fmt.Sprintf(`
<div id="filament-notifications-drawer" class="notifications-drawer-backdrop" style="display: none;" onclick="if(event.target === this) Filament.closeNotificationDrawer();">
  <div class="notifications-drawer">
    <div class="notifications-drawer-header">
      <div style="display: flex; align-items: center; gap: 0.5rem;">
        <span style="font-size: 1.125rem;">🔔</span>
        <h3 style="font-size: 1rem; font-weight: 600;">Notifications</h3>
        <span id="drawer-unread-count" class="badge badge-info">%s</span>
      </div>
      <div style="display: flex; gap: 0.375rem; align-items: center;">
        <button type="button" class="btn btn-secondary btn-xs" onclick="Filament.markAllNotificationsRead('%s/notifications/mark-all-read')">Mark all read</button>
        <button type="button" class="btn btn-danger btn-xs" onclick="Filament.clearAllNotifications('%s/notifications/clear')">Clear all</button>
        <button type="button" onclick="Filament.closeNotificationDrawer()" style="background: none; border: none; font-size: 1.25rem; cursor: pointer; color: var(--gray-400); margin-left: 0.25rem;">&times;</button>
      </div>
    </div>
    <div id="notifications-drawer-list" class="notifications-drawer-list">
      %s
    </div>
  </div>
</div>
<script>
  if (window.Filament && window.Filament.initSSE) {
    window.Filament.initSSE('%s/notifications/stream');
  }
</script>`,
		unreadBadge,
		p.path, p.path,
		listHTML.String(),
		p.path,
	)
}

func (p *Panel) renderLoginScreen(ctx *Context, status int, errMsg, identifier string) error {
	errHTML := ""
	if errMsg != "" {
		errHTML = fmt.Sprintf(`<div class="filament-alert filament-alert-danger" role="alert" style="margin-bottom: 1rem; padding: 0.75rem; border-radius: 0.5rem; background: #fef2f2; color: #b91c1c; font-size: 0.875rem;">%s</div>`, html.EscapeString(errMsg))
	}
	page := fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s" dir="%s">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Sign In - %s</title>
  <link rel="stylesheet" href="%s/assets/filament.css">
  <script src="%s/assets/filament.js"></script>
%s
</head>
<body style="min-height: 100vh; display: flex; align-items: center; justify-content: center; background-color: var(--gray-50);">
  <div class="filament-card" style="width: 100%%; max-width: 26rem; padding: 2rem;">
    <div style="text-align: center; margin-bottom: 1.5rem;">
      <h2 style="font-size: 1.5rem; font-weight: 700; color: var(--primary-600);">%s</h2>
      <p style="font-size: 0.875rem; color: var(--gray-500); margin-top: 0.25rem;">Sign in to your administration panel</p>
    </div>
    %s
    <form method="POST" action="%s/login">
      %s
      <div class="form-group">
        <label class="form-label" for="email">Email address</label>
        <input type="email" id="email" name="email" required placeholder="admin@example.com" value="%s" class="form-control" />
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
  %s
</body>
</html>`, ctx.Locale(), ctx.Direction(), p.brandName, p.path, p.path, p.RenderHeadHTML(ctx), p.brandName, errHTML, p.path, p.csrfHiddenInput(ctx.Context, ""), html.EscapeString(identifier), p.RenderBodyEndHTML(ctx))

	return ctx.HTML(status, page)
}

func (p *Panel) mountFlutter(r *router.Router) {
	if p.flutterFS != nil {
		p.serveFlutterFS(r)
		return
	}
	distDir := p.GetFlutterDistDir()
	indexPath := filepath.Join(distDir, "index.html")

	if _, err := os.Stat(indexPath); err == nil {
		// Serve compiled Flutter Web SPA with customizable pre-loader and version meta
		flutter.ServeWeb(r, flutter.Options{
			MountPath:     p.path,
			DistDir:       distDir,
			FallbackSPA:   true,
			LoadingConfig: p.GetFlutterLoadingConfig(),
			Version:       p.GetVersion(),
			BuildID:       p.GetBuildID(),
		})
		return
	}

	// If Flutter Web hasn't been built yet, show guidance page
	r.GET(p.path, func(c *gcontext.Context) error {
		ctx := NewContext(c, p)
		clientDir := filepath.Dir(distDir)
		if strings.HasSuffix(distDir, "web") {
			clientDir = filepath.Dir(clientDir)
		}
		html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s" dir="%s">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s - Flutter Mobile-First Admin Panel</title>
  <link rel="stylesheet" href="%s/assets/filament.css">
%s
</head>
<body style="min-height: 100vh; display: flex; align-items: center; justify-content: center; background: #0f172a; color: #f8fafc; font-family: sans-serif; padding: 1rem;">
  <div style="max-width: 32rem; width: 100%%; padding: 2.5rem; background: #1e293b; border-radius: 1rem; border: 1px solid #334155; text-align: center; box-shadow: 0 25px 50px -12px rgba(0,0,0,0.5);">
    <div style="font-size: 3rem; margin-bottom: 1rem;">📱⚡</div>
    <h1 style="font-size: 1.5rem; font-weight: 700; margin-bottom: 0.5rem; color: #38bdf8;">%s Flutter Admin Active</h1>
    <p style="font-size: 0.875rem; color: #94a3b8; line-height: 1.5; margin-bottom: 1.5rem;">
      Your admin panel is running in <strong>Flutter Multiplatform</strong> mode (Mobile-First Web, iOS, Android, Desktop).
    </p>
    <div style="background: #090d16; padding: 1rem; border-radius: 0.5rem; text-align: left; font-family: monospace; font-size: 0.8125rem; color: #38bdf8; margin-bottom: 1.5rem;">
      <div style="color: #64748b;"># 1. Compile Flutter Web client:</div>
      <div style="color: #4ade80;">cd %s && flutter build web</div>
      <div style="color: #64748b; margin-top: 0.5rem;"># 2. Or run natively on iOS / Android:</div>
      <div style="color: #4ade80;">cd %s && flutter run</div>
    </div>
    <div style="display: flex; gap: 0.75rem; justify-content: center;">
      <a href="%s/api/schema" style="display: inline-block; padding: 0.5rem 1rem; border-radius: 0.375rem; background: #38bdf8; color: #0f172a; font-weight: 600; text-decoration: none; font-size: 0.875rem;">View JSON Schema</a>
    </div>
  </div>
  %s
</body>
</html>`, ctx.Locale(), ctx.Direction(), p.brandName, p.path, p.RenderHeadHTML(ctx), p.brandName, clientDir, clientDir, p.path, p.RenderBodyEndHTML(ctx))
		return c.HTML(http.StatusOK, html)
	})

	r.GET(p.path+"/", func(c *gcontext.Context) error {
		return c.Redirect(http.StatusMovedPermanently, p.path)
	})
}
