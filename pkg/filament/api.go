package filament

import (
	"errors"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/notifications"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/appupdate"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
	gosupport "github.com/golaravel/golaravel/pkg/support"
	"github.com/golaravel/golaravel/pkg/translation"
)

func setCorsHeaders(c *gcontext.Context) {
	origin := c.Header("Origin")
	if origin == "" {
		origin = "*"
	}
	c.SetHeader("Access-Control-Allow-Origin", origin)
	c.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
	c.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Tenant-ID, Accept, Origin, X-Requested-With")
	c.SetHeader("Access-Control-Allow-Credentials", "true")
}

// RegisterAPIRoutes mounts the standardized Panel JSON REST API.
func (p *Panel) RegisterAPIRoutes(r *router.Router) {
	apiPrefix := p.path + "/api"

	corsPreflight := func(c *gcontext.Context) error {
		setCorsHeaders(c)
		return c.NoContent(http.StatusNoContent)
	}

	// Register CORS preflight handlers
	r.OPTIONS(apiPrefix+"/schema", corsPreflight)
	r.OPTIONS(apiPrefix+"/version", corsPreflight)
	r.OPTIONS(apiPrefix+"/global-search", corsPreflight)
	r.OPTIONS(apiPrefix+"/tenants", corsPreflight)
	r.OPTIONS(apiPrefix+"/notifications", corsPreflight)
	r.OPTIONS(apiPrefix+"/login", corsPreflight)
	r.OPTIONS(apiPrefix+"/logout", corsPreflight)
	r.OPTIONS(apiPrefix+"/me", corsPreflight)
	r.OPTIONS(apiPrefix+"/support/tickets", corsPreflight)
	r.OPTIONS(apiPrefix+"/support/message", corsPreflight)
	r.OPTIONS(apiPrefix+"/support/voice-feature", corsPreflight)
	r.OPTIONS(apiPrefix+"/support/escalate", corsPreflight)
	r.OPTIONS(apiPrefix+"/support/language", corsPreflight)
	r.OPTIONS(apiPrefix+"/support/knowledge", corsPreflight)

	// Panel Auth Login Endpoint (Flutter & Headless Clients)
	r.POST(apiPrefix+"/login", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		var creds struct {
			Username string `json:"username"`
			Login    string `json:"login"`
			Email    string `json:"email"`
			Password string `json:"password"`
			Remember bool   `json:"remember"`
		}
		if err := c.BindJSON(&creds); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}
		identifier := creds.Username
		if identifier == "" {
			identifier = creds.Login
		}
		if identifier == "" {
			identifier = creds.Email
		}
		if identifier == "" || creds.Password == "" {
			return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "Username and password are required"})
		}
		if p.authValidator == nil {
			return c.JSON(http.StatusNotImplemented, map[string]string{"error": "Panel authentication not configured"})
		}

		ctx := NewContext(c, p)
		tokenDuration := p.ResolveTokenDuration(creds.Remember)
		ctx.Set("remember", creds.Remember)
		ctx.Set("token_duration", tokenDuration)

		authUser, tokenStr, err := p.authValidator(ctx, identifier, creds.Password)
		if err != nil || authUser == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		}

		expiresAt := time.Now().Add(tokenDuration)

		var userInfo map[string]any
		if pu, ok := authUser.(interface{ ToMap() map[string]any }); ok {
			userInfo = pu.ToMap()
		} else {
			userInfo = map[string]any{
				"id":    authUser.GetAuthIdentifier(),
				"login": authUser.GetAuthIdentifier(),
				"name":  authUser.GetAuthIdentifier(),
			}
		}

		return c.JSON(http.StatusOK, map[string]any{
			"status":     "success",
			"token":      tokenStr,
			"user":       userInfo,
			"expires_at": expiresAt.Format(time.RFC3339),
			"expires_in": int(tokenDuration.Seconds()),
		})
	})

	// Panel Auth Logout Endpoint
	r.POST(apiPrefix+"/logout", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		return c.JSON(http.StatusOK, map[string]string{"status": "logged_out"})
	})

	// Panel Current User Endpoint
	r.GET(apiPrefix+"/me", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		u := ctx.User()
		if u == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthenticated"})
		}
		if pu, ok := u.(interface{ ToMap() map[string]any }); ok {
			return c.JSON(http.StatusOK, pu.ToMap())
		}
		return c.JSON(http.StatusOK, map[string]any{
			"id":    u.GetAuthIdentifier(),
			"login": u.GetAuthIdentifier(),
		})
	})

	// 1. Panel Schema Endpoint
	r.GET(apiPrefix+"/schema", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}

		navGroups := p.BuildNavigation(ctx)
		type navItemDTO struct {
			Label  string `json:"label"`
			URL    string `json:"url"`
			Icon   string `json:"icon"`
			Badge  string `json:"badge,omitempty"`
			Active bool   `json:"active"`
		}
		type navGroupDTO struct {
			Name  string       `json:"name"`
			Items []navItemDTO `json:"items"`
		}

		var navDTOs []navGroupDTO
		for _, g := range navGroups {
			groupDTO := navGroupDTO{Name: g.Name}
			for _, it := range g.Items {
				groupDTO.Items = append(groupDTO.Items, navItemDTO{
					Label:  it.Label,
					URL:    it.URL,
					Icon:   it.Icon,
					Badge:  it.Badge,
					Active: it.Active,
				})
			}
			navDTOs = append(navDTOs, groupDTO)
		}

		type relationMetaDTO struct {
			Relationship string `json:"relationship"`
			Title        string `json:"title"`
			Icon         string `json:"icon"`
		}

		type resourceDTO struct {
			Slug                     string              `json:"slug"`
			ModelName                string              `json:"model_name"`
			PluralModelName          string              `json:"plural_model_name"`
			NavIcon                  string              `json:"icon"`
			NavGroup                 string              `json:"group"`
			Relations                []relationMetaDTO   `json:"relations,omitempty"`
			Filters                  []tables.FilterDTO  `json:"filters,omitempty"`
			Columns                  []tables.ColumnDTO  `json:"columns,omitempty"`
			DefaultColumns           []string            `json:"default_columns,omitempty"`
			SelectableColumns        []string            `json:"selectable_columns,omitempty"`
			AdvancedSearchEnabled    bool                `json:"advanced_search_enabled"`
			AdvancedSearchColumns    []string            `json:"advanced_search_columns,omitempty"`
			Tabs                     []*ResourceTab      `json:"tabs,omitempty"`
			TabsEnabled              bool                `json:"tabs_enabled"`
			ListMode                 string              `json:"list_mode"`
			PerPage                  int                 `json:"per_page"`
			PerPageOptions           []int               `json:"per_page_options,omitempty"`
			MobilePaginationPosition string              `json:"mobile_pagination_position"`
		}

		var resourceDTOs []resourceDTO
		for _, res := range p.resources {
			item := res.NavigationItem(ctx)
			if item != nil {
				var relsDTO []relationMetaDTO
				for _, rel := range res.GetRelations() {
					relsDTO = append(relsDTO, relationMetaDTO{
						Relationship: rel.Relationship(),
						Title:        rel.Title(),
						Icon:         rel.GetIcon(),
					})
				}
				var filtersDTO []tables.FilterDTO
				if hf, ok := res.(interface{ GetFilters() []tables.FilterDTO }); ok {
					filtersDTO = hf.GetFilters()
				}
				var columnsDTO []tables.ColumnDTO
				if hc, ok := res.(interface{ GetColumns() []tables.ColumnDTO }); ok {
					columnsDTO = hc.GetColumns()
				}
				var defaultCols []string
				if hdc, ok := res.(interface{ GetDefaultColumns() []string }); ok {
					defaultCols = hdc.GetDefaultColumns()
				}
				var selectableCols []string
				if hsc, ok := res.(interface{ GetSelectableColumns() []string }); ok {
					selectableCols = hsc.GetSelectableColumns()
				}
				var advSearchEnabled bool
				var advSearchCols []string
				if hasc, ok := res.(interface {
					IsAdvancedSearchEnabled() bool
					GetAdvancedSearchableColumns() []string
				}); ok {
					advSearchEnabled = hasc.IsAdvancedSearchEnabled()
					advSearchCols = hasc.GetAdvancedSearchableColumns()
				}
				tabsEnabled := p.IsTabsEnabled() || res.IsTabsEnabled()
				var tabsDTO []*ResourceTab = res.GetTabs()
				if len(tabsDTO) == 0 && tabsEnabled {
					tabsDTO = []*ResourceTab{
						{Key: "all", Label: "All"},
						{Key: "active", Label: "Active", Query: map[string]string{"status": "active"}},
						{Key: "inactive", Label: "Inactive", Query: map[string]string{"status": "inactive"}},
					}
				}
				listMode := string(res.GetListMode())
				if listMode == "" {
					listMode = string(p.GetDefaultListMode())
				}
				perPage := res.GetPerPage()
				if perPage <= 0 {
					perPage = p.GetDefaultPerPage()
				}
				perPageOpts := res.GetPerPageOptions()
				if len(perPageOpts) == 0 {
					perPageOpts = p.GetDefaultPerPageOptions()
				}

				mobPos := res.GetMobilePaginationPosition()
				if mobPos == "" {
					mobPos = p.GetDefaultMobilePaginationPosition()
				}
				if mobPos == "" {
					mobPos = PaginationPositionTop
				}

				resourceDTOs = append(resourceDTOs, resourceDTO{
					Slug:                     res.Slug(),
					ModelName:                item.Label,
					PluralModelName:          item.Label,
					NavIcon:                  item.Icon,
					NavGroup:                 item.Group,
					Relations:                relsDTO,
					Filters:                  filtersDTO,
					Columns:                  columnsDTO,
					DefaultColumns:           defaultCols,
					SelectableColumns:        selectableCols,
					AdvancedSearchEnabled:    advSearchEnabled,
					AdvancedSearchColumns:    advSearchCols,
					Tabs:                     tabsDTO,
					TabsEnabled:              tabsEnabled,
					ListMode:                 listMode,
					PerPage:                  perPage,
					PerPageOptions:           perPageOpts,
					MobilePaginationPosition: string(mobPos),
				})
			}
		}

		type tenantDTO struct {
			Key    string `json:"key"`
			Name   string `json:"name"`
			Avatar string `json:"avatar,omitempty"`
			Active bool   `json:"active"`
		}

		var tenantsDTO []tenantDTO
		activeTenantKey := ctx.Tenant()
		if p.tenantConfig != nil && p.tenantConfig.Enabled && p.tenantConfig.TenantLoader != nil {
			if list, err := p.tenantConfig.TenantLoader(ctx); err == nil {
				for _, t := range list {
					tenantsDTO = append(tenantsDTO, tenantDTO{
						Key:    t.GetTenantKey(),
						Name:   t.GetTenantName(),
						Avatar: t.GetTenantAvatar(),
						Active: t.GetTenantKey() == activeTenantKey,
					})
				}
			}
		}

		colorsDTO := make(map[string]map[string]string)
		for role, pal := range p.colors {
			if pal != nil {
				palMap := make(map[string]string)
				for shade, hex := range pal {
					palMap[strconv.Itoa(shade)] = hex
				}
				colorsDTO[role] = palMap
			}
		}

		themeDTO := map[string]any{
			"font_family": p.fontFamily,
			"font_url":    p.fontURL,
			"colors":      colorsDTO,
			"stylesheets": p.customStylesheets,
			"scripts":     p.customScripts,
		}

		schema := map[string]any{
			"id":                p.id,
			"brand_name":        p.brandName,
			"path":              p.path,
			"engine":            string(p.GetEngine()),
			"dark_mode":         p.darkMode,
			"theme":             themeDTO,
			"global_search": map[string]any{
				"enabled":     p.hasGlobalSearch,
				"key_binding": p.GetGlobalSearchKeyBinding(),
				"limit":       p.GetGlobalSearchResultsLimit(),
				"fuzzy":       p.GetGlobalSearchFuzzy(),
			},
			"locale":                ctx.Locale(),
			"direction":             ctx.Direction(),
			"is_rtl":                ctx.IsRTL(),
			"supported_locales":     p.locales,
			"default_locale":        p.GetDefaultLocale(),
			"locale_labels":         p.GetLocaleLabels(),
			"locale_flags":          p.GetLocaleFlags(),
			"translations":          p.GetTranslations(ctx.Locale()),
			"active_tenant":         activeTenantKey,
			"tenants":               tenantsDTO,
			"navigation":            navDTOs,
			"resources":             resourceDTOs,
			"widgets_count":         len(p.widgets),
			"widgets":               p.EvaluateWidgetDTOs(ctx),
			"account_nav_placement": string(p.GetAccountNavPlacement()),
			"requires_auth":         p.requiresAuth,
			"session_duration":      int(p.GetSessionDuration().Seconds()),
			"remember_me_duration":  int(p.GetRememberMeDuration().Seconds()),
			"tabs_enabled":          p.IsTabsEnabled(),
			"default_list_mode":                  string(p.GetDefaultListMode()),
			"default_per_page":                   p.GetDefaultPerPage(),
			"default_per_page_options":           p.GetDefaultPerPageOptions(),
			"default_mobile_pagination_position": string(p.GetDefaultMobilePaginationPosition()),
			"bottom_nav_show_labels":             p.IsBottomNavLabelsVisible(),
			"default_bottom_nav_items":           p.GetMobileBottomNavItems(),
			"app_update": map[string]any{
				"enabled":                p.GetVersion() != "" || p.GetBuildID() != "" || p.GetAppUpdateManager() != nil,
				"version":                p.GetVersion(),
				"build_id":               p.GetBuildID(),
				"force_reload":           p.IsForceReloadOnUpdate(),
				"check_interval_seconds": 300,
				"title":                  "System Update Available",
				"message":                "A new version of the admin panel is available. Please reload the page to apply the latest updates.",
			},
			"support": map[string]any{
				"enabled":              p.IsSupportEnabled(),
				"scope":                p.GetSupportScope(),
				"super_admin_panel_id": p.GetSupportSuperAdminPanelID(),
				"supported_languages":  []string{"en", "si", "ta", "ar", "es", "de"},
			},
		}

		return c.JSON(http.StatusOK, schema)
	})

	// Version / Update Check Endpoint
	r.GET(apiPrefix+"/version", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		clientVersion := c.Query("version")
		if clientVersion == "" {
			clientVersion = c.Header("X-App-Version")
		}
		clientBuildID := c.Query("build_id")
		if clientBuildID == "" {
			clientBuildID = c.Header("X-App-Build-ID")
		}

		serverVersion := p.GetVersion()
		serverBuildID := p.GetBuildID()
		forceReload := p.IsForceReloadOnUpdate()

		updateAvailable := false
		if clientBuildID != "" && serverBuildID != "" {
			updateAvailable = (clientBuildID != serverBuildID)
		} else if clientVersion != "" && serverVersion != "" {
			updateAvailable = (appupdate.CompareVersions(clientVersion, serverVersion) < 0)
		}

		res := map[string]any{
			"update_available":       updateAvailable,
			"force_reload":           forceReload,
			"version":                serverVersion,
			"build_id":               serverBuildID,
			"current_version":        clientVersion,
			"current_build_id":       clientBuildID,
			"check_interval_seconds": 300,
			"title":                  "System Update Available",
			"message":                "A new version of the admin panel is available. Please reload the page to apply the latest updates.",
		}

		if updateAvailable {
			c.SetHeader("X-App-Update-Available", "true")
			if forceReload {
				c.SetHeader("X-App-Force-Update", "true")
			}
		}

		return c.JSON(http.StatusOK, res)
	})

	// Global Search REST API Endpoint
	r.GET(apiPrefix+"/global-search", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		query := c.Query("query")
		if query == "" {
			query = c.Query("q")
		}

		groups, allResults := p.searchAll(ctx, query)
		if groups == nil {
			groups = []GlobalSearchGroup{}
		}
		if allResults == nil {
			allResults = []GlobalSearchResult{}
		}

		return c.JSON(http.StatusOK, map[string]any{
			"query":   query,
			"total":   len(allResults),
			"groups":  groups,
			"results": allResults,
		})
	})

	// Locale switch REST endpoint
	r.POST(apiPrefix+"/locale/switch", func(c *gcontext.Context) error {
		ctx := NewContext(c, p)

		targetLocale := c.Query("locale")
		if targetLocale == "" {
			targetLocale = c.Query("lang")
		}
		if targetLocale == "" {
			targetLocale = c.FormValue("locale")
		}
		if targetLocale == "" {
			targetLocale = c.FormValue("lang")
		}

		// Also support JSON body: {"locale": "ar"} or {"lang": "ar"}
		if targetLocale == "" && strings.Contains(c.Header("Content-Type"), "application/json") {
			var body struct {
				Locale string `json:"locale"`
				Lang   string `json:"lang"`
			}
			if err := json.NewDecoder(c.Request.Body).Decode(&body); err == nil {
				if body.Locale != "" {
					targetLocale = body.Locale
				} else if body.Lang != "" {
					targetLocale = body.Lang
				}
			}
		}

		if targetLocale == "" {
			return c.JSON(http.StatusBadRequest, map[string]any{
				"error": "Locale code is required (param 'locale' or 'lang')",
			})
		}

		// Validate against supported locales if configured
		if len(p.locales) > 0 {
			matched := false
			for _, loc := range p.locales {
				if strings.EqualFold(loc, targetLocale) {
					targetLocale = loc
					matched = true
					break
				}
			}
			if !matched {
				return c.JSON(http.StatusBadRequest, map[string]any{
					"error":             fmt.Sprintf("Locale '%s' is not supported", targetLocale),
					"supported_locales": p.locales,
				})
			}
		}

		c.SetCookie("locale", targetLocale, 86400*365, "/", "", false, false)
		ctx.SetLocale(targetLocale)

		return c.JSON(http.StatusOK, map[string]any{
			"success":      true,
			"locale":       ctx.Locale(),
			"direction":    ctx.Direction(),
			"is_rtl":       ctx.IsRTL(),
			"label":        p.GetLocaleLabel(ctx.Locale()),
			"flag":         p.GetLocaleFlag(ctx.Locale()),
			"translations": p.GetTranslations(ctx.Locale()),
		})
	})

	// Translations REST API endpoint
	r.GET(apiPrefix+"/translations", func(c *gcontext.Context) error {
		ctx := NewContext(c, p)
		loc := c.Query("locale")
		if loc == "" {
			loc = c.Query("lang")
		}
		if loc == "" {
			loc = ctx.Locale()
		}
		return c.JSON(http.StatusOK, map[string]any{
			"locale":       loc,
			"direction":    translation.Direction(loc),
			"is_rtl":       translation.IsRTL(loc),
			"label":        p.GetLocaleLabel(loc),
			"flag":         p.GetLocaleFlag(loc),
			"translations": p.GetTranslations(loc),
		})
	})

	// 2. Multi-Tenant API Endpoints
	if p.tenantConfig != nil && p.tenantConfig.Enabled {
		r.GET(apiPrefix+"/tenants", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			var list []Tenant
			if p.tenantConfig.TenantLoader != nil {
				list, _ = p.tenantConfig.TenantLoader(ctx)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"active_tenant": ctx.Tenant(),
				"tenants":       list,
			})
		})

		r.POST(apiPrefix+"/tenants/switch", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			targetKey := c.Query("tenant")
			if targetKey == "" {
				targetKey = c.FormValue("tenant")
			}
			if targetKey == "" {
				var body map[string]string
				if err := json.NewDecoder(c.Request.Body).Decode(&body); err == nil {
					targetKey = body["tenant"]
				}
			}

			if targetKey != "" {
				c.SetCookie(p.tenantConfig.CookieName, targetKey, 86400*30, "/", "", false, true)
				ctx.SetTenant(targetKey)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"success":       true,
				"active_tenant": targetKey,
			})
		})
	}

	// 3. Widgets Evaluation API
	r.GET(apiPrefix+"/widgets", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}

		widgetData := p.EvaluateWidgetDTOs(ctx)
		return c.JSON(http.StatusOK, map[string]any{
			"widgets": widgetData,
		})
	})

	r.GET(apiPrefix+"/widgets/:id/data", func(c *gcontext.Context) error {
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

	// 4. Notifications REST API
	if p.hasNotifications {
		r.GET(apiPrefix+"/notifications", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			user := "all"
			if ctx.User() != nil {
				user = ctx.User().GetAuthIdentifier()
			}
			items, _ := notifications.DefaultStore().List(user, 50)
			unread, _ := notifications.DefaultStore().CountUnread(user)
			return c.JSON(http.StatusOK, map[string]any{
				"unread_count":  unread,
				"notifications": items,
			})
		})

		r.POST(apiPrefix+"/notifications/mark-read", func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			user := "all"
			if ctx.User() != nil {
				user = ctx.User().GetAuthIdentifier()
			}
			payload := parseRequestData(c)
			id := fmt.Sprintf("%v", payload["id"])
			_ = notifications.DefaultStore().MarkAsRead(user, id)
			return c.JSON(http.StatusOK, map[string]any{"success": true})
		})

		r.POST(apiPrefix+"/notifications/mark-all-read", func(c *gcontext.Context) error {
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

		r.POST(apiPrefix+"/notifications/clear", func(c *gcontext.Context) error {
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
	}

	// 5. Support Concierge & Voice Feature Request API
	// GET /support/tickets - List support tickets scoped to tenant / super admin
	r.GET(apiPrefix+"/support/tickets", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		filter := gosupport.TicketFilter{
			Status:     c.Query("status"),
			Category:   c.Query("category"),
			Department: c.Query("department"),
		}
		if p.GetSupportScope() == "tenant_admin" || p.hasTenantScoping {
			if tid := ctx.Tenant(); tid != "" {
				filter.TenantID = tid
			}
		}
		if reqTenant := c.Query("tenant_id"); reqTenant != "" && (p.GetSupportScope() == "super_admin" || filter.TenantID == "") {
			filter.TenantID = reqTenant
		}

		tickets, err := gosupport.GetDefaultStore().ListTickets(c.Request.Context(), filter)
		if err != nil {
			return p.apiError(c, err)
		}
		return c.JSON(http.StatusOK, map[string]any{
			"tickets": tickets,
			"total":   len(tickets),
			"scope":   p.GetSupportScope(),
		})
	})

	// POST /support/message - Send customer message & get instant AI response / auto-escalation
	r.POST(apiPrefix+"/support/message", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, _ := p.AuthorizeAccess(c)
		var req struct {
			TicketID  string `json:"ticket_id"`
			TenantID  string `json:"tenant_id"`
			UserID    string `json:"user_id"`
			UserName  string `json:"user_name"`
			UserEmail string `json:"user_email"`
			Content   string `json:"content"`
			Category  string `json:"category"`
			Language  string `json:"language"`
		}
		_ = c.BindJSON(&req)
		if req.Content == "" {
			payload := parseRequestData(c)
			if v, ok := payload["ticket_id"].(string); ok {
				req.TicketID = v
			}
			if v, ok := payload["tenant_id"].(string); ok {
				req.TenantID = v
			}
			if v, ok := payload["user_id"].(string); ok {
				req.UserID = v
			}
			if v, ok := payload["user_name"].(string); ok {
				req.UserName = v
			}
			if v, ok := payload["user_email"].(string); ok {
				req.UserEmail = v
			}
			if v, ok := payload["content"].(string); ok {
				req.Content = v
			}
			if v, ok := payload["category"].(string); ok {
				req.Category = v
			}
			if v, ok := payload["language"].(string); ok {
				req.Language = v
			}
		}
		if req.Content == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Message content is required"})
		}
		if req.Language == "" {
			req.Language = gosupport.ResolveSupportLanguage(c.Request)
		}
		if req.TenantID == "" && ctx != nil {
			req.TenantID = ctx.Tenant()
		}
		if req.UserID == "" && ctx != nil && ctx.User() != nil {
			req.UserID = ctx.User().GetAuthIdentifier()
			req.UserName = ctx.User().GetAuthIdentifier()
		}
		if req.Category == "" {
			req.Category = gosupport.CategorySupport
		}

		userMsg, aiMsg, err := gosupport.GetDefaultService().SendUserMessage(
			c.Request.Context(),
			req.TicketID,
			req.UserID,
			req.UserName,
			req.Content,
			req.Language,
		)
		if err != nil {
			return p.apiError(c, err)
		}

		ticket, _ := gosupport.GetDefaultStore().GetTicket(c.Request.Context(), userMsg.TicketID)
		autoEscalated := ticket != nil && (ticket.Status == gosupport.StatusEscalated || ticket.Mode == gosupport.ModeLive)

		return c.JSON(http.StatusOK, map[string]any{
			"ticket":         ticket,
			"user_message":   userMsg,
			"ai_message":     aiMsg,
			"auto_escalated": autoEscalated,
		})
	})

	// POST /support/voice-feature - Submit multimodal voice feature request in any language
	r.POST(apiPrefix+"/support/voice-feature", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, _ := p.AuthorizeAccess(c)

		var (
			ticketID   string
			tenantID   string
			userID     string
			userName   string
			userEmail  string
			language   string
			audioBytes []byte
			mimeType   = "audio/webm"
			duration   float64
		)

		if strings.Contains(c.Header("Content-Type"), "application/json") {
			var body struct {
				TicketID    string  `json:"ticket_id"`
				TenantID    string  `json:"tenant_id"`
				UserID      string  `json:"user_id"`
				UserName    string  `json:"user_name"`
				UserEmail   string  `json:"user_email"`
				Language    string  `json:"language"`
				AudioBase64 string  `json:"audio_base64"`
				MimeType    string  `json:"mime_type"`
				DurationSec float64 `json:"duration_seconds"`
			}
			if err := c.BindJSON(&body); err == nil {
				ticketID = body.TicketID
				tenantID = body.TenantID
				userID = body.UserID
				userName = body.UserName
				userEmail = body.UserEmail
				language = body.Language
				duration = body.DurationSec
				if body.MimeType != "" {
					mimeType = body.MimeType
				}
				if body.AudioBase64 != "" {
					data, decErr := base64.StdEncoding.DecodeString(body.AudioBase64)
					if decErr == nil {
						audioBytes = data
					}
				}
			}
		} else {
			fh, fileErr := c.FormFile("audio")
			if fileErr != nil {
				fh, fileErr = c.FormFile("voice")
			}
			if fileErr == nil && fh != nil {
				f, openErr := fh.Open()
				if openErr == nil {
					defer f.Close()
					audioBytes, _ = io.ReadAll(f)
					if ct := fh.Header.Get("Content-Type"); ct != "" {
						mimeType = ct
					}
				}
			}
			ticketID = c.FormValue("ticket_id")
			tenantID = c.FormValue("tenant_id")
			userID = c.FormValue("user_id")
			userName = c.FormValue("user_name")
			userEmail = c.FormValue("user_email")
			language = c.FormValue("language")
		}

		if len(audioBytes) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Audio recording data is required (either 'audio_base64' JSON or 'audio' multipart file)"})
		}

		if language == "" {
			language = gosupport.ResolveSupportLanguage(c.Request)
		}
		if tenantID == "" && ctx != nil {
			tenantID = ctx.Tenant()
		}
		if userID == "" && ctx != nil && ctx.User() != nil {
			userID = ctx.User().GetAuthIdentifier()
			userName = ctx.User().GetAuthIdentifier()
		}

		ticket, aiReply, err := gosupport.GetDefaultService().SubmitVoiceFeatureRequest(
			c.Request.Context(),
			gosupport.VoiceFeatureRequest{
				TicketID:        ticketID,
				AudioBytes:      audioBytes,
				MimeType:        mimeType,
				DurationSeconds: duration,
				Language:        language,
				UserKey:         userID,
				UserName:        userName,
				UserEmail:       userEmail,
				TenantID:        tenantID,
				PanelID:         p.ID(),
			},
		)
		if err != nil {
			return p.apiError(c, err)
		}

		return c.JSON(http.StatusOK, map[string]any{
			"ticket":              ticket,
			"ai_reply":            aiReply,
			"feasibility":         ticket.Feasibility,
			"is_existing_feature": ticket.Status == gosupport.StatusResolved,
			"audio_url":           fmt.Sprintf("%s/support/audio/%s", apiPrefix, ticket.ID),
		})
	})

	// GET /support/audio/:id - Stream preserved voice recording
	r.GET(apiPrefix+"/support/audio/:id", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ticketID := c.Param("id")
		ticket, err := gosupport.GetDefaultStore().GetTicket(c.Request.Context(), ticketID)
		if err != nil || ticket == nil || ticket.Voice == nil {
			return c.NoContent(http.StatusNotFound)
		}
		c.SetHeader("Content-Type", ticket.Voice.MimeType)
		c.SetHeader("Content-Disposition", fmt.Sprintf(`inline; filename="feature_request_%s.webm"`, ticket.ID))
		return c.Bytes(http.StatusOK, ticket.Voice.MimeType, ticket.Voice.AudioData)
	})

	// POST /support/escalate - Escalate to Live Support Agent or Super Admin Panel
	r.POST(apiPrefix+"/support/escalate", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		var req struct {
			TicketID string `json:"ticket_id"`
			Target   string `json:"target"`
			Reason   string `json:"reason"`
		}
		_ = c.BindJSON(&req)
		if req.TicketID == "" {
			payload := parseRequestData(c)
			if v, ok := payload["ticket_id"].(string); ok {
				req.TicketID = v
			}
			if v, ok := payload["target"].(string); ok {
				req.Target = v
			}
			if v, ok := payload["reason"].(string); ok {
				req.Reason = v
			}
		}
		if req.TicketID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Ticket ID is required"})
		}

		if req.Target == "super_admin" {
			superAdminID := p.GetSupportSuperAdminPanelID()
			if err := gosupport.GetDefaultService().EscalateToSuperAdmin(c.Request.Context(), req.TicketID, superAdminID, req.Reason); err != nil {
				return p.apiError(c, err)
			}
		} else {
			if err := gosupport.GetDefaultService().EscalateToLive(c.Request.Context(), req.TicketID, req.Reason); err != nil {
				return p.apiError(c, err)
			}
		}

		ticket, _ := gosupport.GetDefaultStore().GetTicket(c.Request.Context(), req.TicketID)
		return c.JSON(http.StatusOK, map[string]any{
			"status": "escalated",
			"ticket": ticket,
		})
	})

	// POST /support/language - Set language preference (persisted in golaravel_support_lang cookie)
	r.POST(apiPrefix+"/support/language", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		var req struct {
			Language string `json:"language"`
		}
		_ = c.BindJSON(&req)
		targetLang := req.Language
		if targetLang == "" {
			targetLang = c.Query("lang")
		}
		if targetLang == "" {
			payload := parseRequestData(c)
			if v, ok := payload["language"].(string); ok {
				targetLang = v
			}
		}
		if targetLang == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Language code is required"})
		}
		targetLang = gosupport.NormalizeLanguage(targetLang)
		c.SetCookie(gosupport.LanguageCookieName, targetLang, 86400*365, "/", "", false, false)
		gosupport.SetSupportLanguageCookie(&c.Response, targetLang)

		return c.JSON(http.StatusOK, map[string]any{
			"status":   "success",
			"language": targetLang,
		})
	})

	// GET /support/knowledge - Return system features known up to latest release
	r.GET(apiPrefix+"/support/knowledge", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		return c.JSON(http.StatusOK, map[string]any{
			"features": gosupport.GetDefaultKnowledgeBase().All(),
			"count":    len(gosupport.GetDefaultKnowledgeBase().All()),
		})
	})

	// 6. Register REST Endpoints for each resource
	for _, res := range p.resources {
		p.registerResourceAPI(res, apiPrefix, r)
	}

	// Mount App Update Manager routes under this panel if configured
	if p.appUpdateManager != nil {
		p.appUpdateManager.RegisterRoutes(r, apiPrefix+"/app-update")
	}
}

// EvaluateWidgetDTOs serializes all panel dashboard widgets for headless REST clients and Flutter UI.
func (p *Panel) EvaluateWidgetDTOs(ctx *Context) []map[string]any {
	var widgetData []map[string]any
	for _, w := range p.widgets {
		dto := map[string]any{
			"id":          w.ID(),
			"sort":        w.Sort(),
			"column_span": w.ColumnSpan(),
		}
		if wp, ok := w.(interface{ ToWidgetDTO(ctx *Context) map[string]any }); ok {
			customDTO := wp.ToWidgetDTO(ctx)
			for k, v := range customDTO {
				dto[k] = v
			}
		} else if chart, ok := w.(interface {
			GetHeading() string
			GetChartTypeString() string
			GetFilters() map[string]string
			GetDefaultFilter() string
			GetDrilldownPattern() string
			GetPolling() string
			EvaluateChartData(ctx *Context, filter string) any
		}); ok {
			dto["type"] = "chart"
			dto["heading"] = chart.GetHeading()
			dto["chart_type"] = chart.GetChartTypeString()
			dto["filters"] = chart.GetFilters()
			dto["default_filter"] = chart.GetDefaultFilter()
			dto["drilldown_url"] = chart.GetDrilldownPattern()
			dto["polling"] = chart.GetPolling()
			dto["data"] = chart.EvaluateChartData(ctx, "")
		} else if stats, ok := w.(interface {
			EvaluateStatsMap(ctx *Context) []map[string]any
		}); ok {
			dto["type"] = "stats_overview"
			dto["stats"] = stats.EvaluateStatsMap(ctx)
		}
		widgetData = append(widgetData, dto)
	}
	return widgetData
}

// registerResourceAPI binds JSON endpoints for a specific Resource.
func (p *Panel) registerResourceAPI(res ResourceInterface, apiPrefix string, r *router.Router) {
	slug := res.Slug()
	resPath := fmt.Sprintf("%s/resources/%s", apiPrefix, slug)

	corsPreflight := func(c *gcontext.Context) error {
		setCorsHeaders(c)
		return c.NoContent(http.StatusNoContent)
	}
	r.OPTIONS(resPath, corsPreflight)
	r.OPTIONS(resPath+"/:id", corsPreflight)
	r.OPTIONS(resPath+"/:id/cell", corsPreflight)
	r.OPTIONS(resPath+"/export", corsPreflight)
	r.OPTIONS(resPath+"/import", corsPreflight)
	r.OPTIONS(resPath+"/bulk", corsPreflight)
	r.OPTIONS(resPath+"/reorder", corsPreflight)

	// GET /admin/api/resources/:slug - List
	r.GET(resPath, func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}

		search := ctx.Query("search")
		sortCol := ctx.Query("sort")
		sortDir := ctx.DefaultQuery("direction", "asc")
		page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
		perPageStr := ctx.Query("perPage")
		if perPageStr == "" {
			perPageStr = ctx.Query("per_page")
		}
		defaultPerPage := res.GetPerPage()
		if defaultPerPage <= 0 {
			defaultPerPage = 15
		}
		perPage := defaultPerPage
		if perPageStr != "" {
			if pVal, err := strconv.Atoi(perPageStr); err == nil && pVal > 0 {
				perPage = pVal
			}
		}

		filters := make(map[string]string)
		for k, v := range ctx.Request.URL.Query() {
			if strings.HasPrefix(k, "filter_") && len(v) > 0 {
				filters[strings.TrimPrefix(k, "filter_")] = v[0]
			}
		}
		if adv := ctx.Query("advanced_search"); adv != "" {
			filters["_advanced_search"] = adv
		}

		// Generic handler invoker
		if invoker, ok := res.(interface {
			InvokeList(ctx *Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) (any, int, error)
		}); ok {
			records, total, err := invoker.InvokeList(ctx, search, sortCol, sortDir, page, perPage, filters)
			if err != nil {
				return p.apiError(c, err)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"data":     records,
				"total":    total,
				"page":     page,
				"per_page": perPage,
			})
		}

		return c.JSON(http.StatusOK, map[string]any{
			"data":     []any{},
			"total":    0,
			"page":     page,
			"per_page": perPage,
		})
	})

	// GET /admin/api/resources/:slug/export - Streaming Export
	r.GET(resPath+"/export", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}

		if exporter, ok := res.(interface {
			InvokeExport(ctx *Context, format string, filterIDs []string, search, sortCol, sortDir string, filters map[string]string) error
		}); ok {
			format := strings.ToLower(ctx.DefaultQuery("format", "csv"))
			search := ctx.Query("search")
			sortCol := ctx.Query("sort")
			sortDir := ctx.DefaultQuery("direction", "asc")
			var filterIDs []string
			if idsQuery := ctx.Query("ids"); idsQuery != "" {
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
			if adv := ctx.Query("advanced_search"); adv != "" {
				filters["_advanced_search"] = adv
			}

			return exporter.InvokeExport(ctx, format, filterIDs, search, sortCol, sortDir, filters)
		}

		return c.JSON(http.StatusNotImplemented, map[string]string{"error": "Export not implemented"})
	})

	// POST /admin/api/resources/:slug/import - Batch & CSV Import
	r.POST(resPath+"/import", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}

		// 1. Check multipart form file upload
		contentType := c.Header("Content-Type")
		if strings.HasPrefix(contentType, "multipart/form-data") {
			_ = c.Request.ParseMultipartForm(32 << 20)
			file, _, err := c.Request.FormFile("file")
			if err != nil {
				file, _, err = c.Request.FormFile("csv_file")
			}
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing file in upload"})
			}
			defer file.Close()

			mapping := make(map[string]string)
			for k, v := range c.Request.PostForm {
				if strings.HasPrefix(k, "mapping[") && strings.HasSuffix(k, "]") && len(v) > 0 {
					src := strings.TrimSuffix(strings.TrimPrefix(k, "mapping["), "]")
					mapping[src] = v[0]
				}
			}

			if importer, ok := res.(interface {
				InvokeImportCSV(ctx *Context, rdr io.Reader, explicitMapping map[string]string) (*ImportResult, error)
			}); ok {
				result, err := importer.InvokeImportCSV(ctx, file, mapping)
				if err != nil {
					return p.apiError(c, err)
				}
				return c.JSON(http.StatusOK, result)
			}
			return c.JSON(http.StatusNotImplemented, map[string]string{"error": "CSV import not implemented"})
		}

		// 2. Check JSON payload ({"data": [ {...}, ... ]} or {"csv": "..."})
		var jsonBody struct {
			Data []map[string]any  `json:"data"`
			CSV  string            `json:"csv"`
			Map  map[string]string `json:"mapping"`
		}
		if err := json.NewDecoder(c.Request.Body).Decode(&jsonBody); err == nil {
			if len(jsonBody.Data) > 0 {
				if importer, ok := res.(interface {
					InvokeImportRecords(ctx *Context, rows []map[string]any) (*ImportResult, error)
				}); ok {
					result, err := importer.InvokeImportRecords(ctx, jsonBody.Data)
					if err != nil {
						return p.apiError(c, err)
					}
					return c.JSON(http.StatusOK, result)
				}
			} else if jsonBody.CSV != "" {
				if importer, ok := res.(interface {
					InvokeImportCSV(ctx *Context, rdr io.Reader, explicitMapping map[string]string) (*ImportResult, error)
				}); ok {
					result, err := importer.InvokeImportCSV(ctx, strings.NewReader(jsonBody.CSV), jsonBody.Map)
					if err != nil {
						return p.apiError(c, err)
					}
					return c.JSON(http.StatusOK, result)
				}
			}
		}

		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid import request payload"})
	})

	// POST /admin/api/resources/:slug/bulk - Bulk Action
	r.POST(resPath+"/bulk", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}

		data := parseRequestData(c)
		action := fmt.Sprintf("%v", data["bulk_action"])

		var ids []string
		if rawIDs, ok := data["selected_ids[]"]; ok {
			if slice, ok := rawIDs.([]string); ok {
				ids = slice
			} else if sliceAny, ok := rawIDs.([]any); ok {
				for _, it := range sliceAny {
					ids = append(ids, fmt.Sprintf("%v", it))
				}
			}
		} else if rawIDs, ok := data["ids"]; ok {
			if slice, ok := rawIDs.([]string); ok {
				ids = slice
			} else if sliceAny, ok := rawIDs.([]any); ok {
				for _, it := range sliceAny {
					ids = append(ids, fmt.Sprintf("%v", it))
				}
			}
		}

		if action == "delete_bulk" && len(ids) > 0 {
			if invoker, ok := res.(interface {
				InvokeDeleteBulk(ctx *Context, ids []string) error
			}); ok {
				if err := invoker.InvokeDeleteBulk(ctx, ids); err != nil {
					return p.apiError(c, err)
				}
				return c.JSON(http.StatusOK, map[string]any{
					"success": true,
					"message": fmt.Sprintf("%d records deleted successfully", len(ids)),
				})
			} else if singleInvoker, ok := res.(interface {
				InvokeDelete(ctx *Context, id string) error
			}); ok {
				for _, id := range ids {
					_ = singleInvoker.InvokeDelete(ctx, id)
				}
				return c.JSON(http.StatusOK, map[string]any{
					"success": true,
					"message": fmt.Sprintf("%d records deleted successfully", len(ids)),
				})
			}
		}

		return c.JSON(http.StatusOK, map[string]any{
			"success": true,
		})
	})

	// POST /admin/api/resources/:slug/reorder - Row Reorder
	r.POST(resPath+"/reorder", func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}

		data := parseRequestData(c)
		var order []string
		if rawOrder, ok := data["order"]; ok {
			if slice, ok := rawOrder.([]string); ok {
				order = slice
			} else if sliceAny, ok := rawOrder.([]any); ok {
				for _, it := range sliceAny {
					order = append(order, fmt.Sprintf("%v", it))
				}
			}
		}

		if len(order) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "No order array provided"})
		}

		if invoker, ok := res.(interface {
			InvokeReorder(ctx *Context, orderedIDs []string) error
		}); ok {
			if err := invoker.InvokeReorder(ctx, order); err != nil {
				return p.apiError(c, err)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"success": true,
				"order":   order,
				"message": "Rows reordered successfully",
			})
		}

		return c.JSON(http.StatusNotImplemented, map[string]string{"error": "Reorder not implemented"})
	})

	// Soft Delete REST API endpoints
	type softDeletableAPI interface {
		HandleRestore(ctx *Context, basePath string, idParam ...string) error
		HandleForceDelete(ctx *Context, basePath string, idParam ...string) error
	}

	if sdr, ok := res.(softDeletableAPI); ok {
		restoreAPIHandler := func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			return sdr.HandleRestore(ctx, resPath)
		}
		r.POST(resPath+"/restore", restoreAPIHandler)

		forceDeleteAPIHandler := func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			return sdr.HandleForceDelete(ctx, resPath)
		}
		r.POST(resPath+"/force-delete", forceDeleteAPIHandler)
		r.DELETE(resPath+"/force-delete", forceDeleteAPIHandler)
	}

	// GET /admin/api/resources/:slug/:id - Find
	r.GET(resPath+"/:id", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		id := c.Param("id")

		if invoker, ok := res.(interface {
			InvokeFind(ctx *Context, id string) (any, error)
		}); ok {
			record, err := invoker.InvokeFind(ctx, id)
			if err != nil {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "Record not found"})
			}
			var relationsMeta []map[string]any
			for _, rel := range res.GetRelations() {
				relationsMeta = append(relationsMeta, rel.GetAPISchema())
			}
			return c.JSON(http.StatusOK, map[string]any{
				"data":      record,
				"relations": relationsMeta,
			})
		}

		return c.JSON(http.StatusNotFound, map[string]string{"error": "Record not found"})
	})

	// POST /admin/api/resources/:slug - Create
	r.POST(resPath, func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}

		data := parseRequestData(c)

		if invoker, ok := res.(interface {
			InvokeCreate(ctx *Context, data map[string]any) error
		}); ok {
			if err := invoker.InvokeCreate(ctx, data); err != nil {
				return p.apiErrorStatus(c, err, http.StatusBadRequest)
			}
			return c.JSON(http.StatusCreated, map[string]any{
				"success": true,
				"message": "Record created successfully",
			})
		}

		return c.JSON(http.StatusNotImplemented, map[string]string{"error": "Create not implemented"})
	})

	// PUT /admin/api/resources/:slug/:id - Update
	updateHandler := func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		id := c.Param("id")
		data := parseRequestData(c)

		if invoker, ok := res.(interface {
			InvokeUpdate(ctx *Context, id string, data map[string]any) error
		}); ok {
			if err := invoker.InvokeUpdate(ctx, id, data); err != nil {
				return p.apiErrorStatus(c, err, http.StatusBadRequest)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"success": true,
				"message": "Record updated successfully",
			})
		}

		return c.JSON(http.StatusNotImplemented, map[string]string{"error": "Update not implemented"})
	}
	r.PUT(resPath+"/:id", updateHandler)
	r.POST(resPath+"/:id", updateHandler)

	// DELETE /admin/api/resources/:slug/:id - Delete
	deleteHandler := func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		id := c.Param("id")

		if invoker, ok := res.(interface {
			InvokeDelete(ctx *Context, id string) error
		}); ok {
			if err := invoker.InvokeDelete(ctx, id); err != nil {
				return p.apiError(c, err)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"success": true,
				"message": "Record deleted successfully",
			})
		}

		return c.JSON(http.StatusNotImplemented, map[string]string{"error": "Delete not implemented"})
	}
	r.DELETE(resPath+"/:id", deleteHandler)
	r.POST(resPath+"/:id/delete", deleteHandler)

	// POST & PATCH /admin/api/resources/:slug/:id/cell - Inline Cell Edit
	cellUpdateAPIHandler := func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		id := c.Param("id")
		data := parseRequestData(c)
		col := fmt.Sprintf("%v", data["column"])
		val := data["value"]

		if col == "" || col == "<nil>" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Column is required"})
		}

		if invoker, ok := res.(interface {
			InvokeCellUpdate(ctx *Context, id, column string, value any) error
		}); ok {
			if err := invoker.InvokeCellUpdate(ctx, id, col, val); err != nil {
				return p.apiError(c, err)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"success": true,
				"id":      id,
				"column":  col,
				"value":   val,
				"message": "Cell updated successfully",
			})
		}

		return c.JSON(http.StatusNotImplemented, map[string]string{"error": "Cell update not implemented"})
	}
	r.POST(resPath+"/:id/cell", cellUpdateAPIHandler)
	r.PATCH(resPath+"/:id/cell", cellUpdateAPIHandler)

	// ----------------------------------------------------
	// Nested Relation Manager Endpoints
	// ----------------------------------------------------

	// GET /admin/api/resources/:slug/:id/relations
	r.GET(resPath+"/:id/relations", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		_ = ctx

		var relationsMeta []map[string]any
		for _, rel := range res.GetRelations() {
			relationsMeta = append(relationsMeta, rel.GetAPISchema())
		}
		return c.JSON(http.StatusOK, map[string]any{
			"resource":  slug,
			"parent_id": c.Param("id"),
			"relations": relationsMeta,
		})
	})

	// Register CRUD for each relation manager on this resource
	for _, rel := range res.GetRelations() {
		relName := rel.Relationship()
		relManager := rel
		relPath := fmt.Sprintf("%s/:id/relations/%s", resPath, relName)

		// GET /admin/api/resources/:slug/:id/relations/:rel - List
		r.GET(relPath, func(c *gcontext.Context) error {
			setCorsHeaders(c)
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			parentID := c.Param("id")
			search := ctx.Query("search")
			sortCol := ctx.Query("sort")
			sortDir := ctx.DefaultQuery("direction", "asc")
			page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
			perPage, _ := strconv.Atoi(ctx.DefaultQuery("perPage", "15"))

			data, total, err := relManager.HandleList(ctx, parentID, search, sortCol, sortDir, page, perPage)
			if err != nil {
				return p.apiError(c, err)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"relationship": relName,
				"parent_id":    parentID,
				"data":         data,
				"total":        total,
				"page":         page,
				"per_page":     perPage,
			})
		})

		// POST /admin/api/resources/:slug/:id/relations/:rel - Create
		r.POST(relPath, func(c *gcontext.Context) error {
			setCorsHeaders(c)
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			parentID := c.Param("id")
			payload := parseRequestData(c)

			if err := relManager.HandleCreate(ctx, parentID, payload); err != nil {
				return p.apiError(c, err)
			}
			return c.JSON(http.StatusCreated, map[string]any{
				"success": true,
				"message": fmt.Sprintf("Related %s record created successfully", relName),
			})
		})

		// PUT /admin/api/resources/:slug/:id/relations/:rel/:child_id - Update
		updateRelHandler := func(c *gcontext.Context) error {
			setCorsHeaders(c)
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			parentID := c.Param("id")
			childID := c.Param("child_id")
			payload := parseRequestData(c)

			if err := relManager.HandleUpdate(ctx, parentID, childID, payload); err != nil {
				return p.apiError(c, err)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"success": true,
				"message": fmt.Sprintf("Related %s record updated successfully", relName),
			})
		}
		r.PUT(relPath+"/:child_id", updateRelHandler)
		r.POST(relPath+"/:child_id", updateRelHandler)

		// DELETE /admin/api/resources/:slug/:id/relations/:rel/:child_id - Delete
		deleteRelHandler := func(c *gcontext.Context) error {
			setCorsHeaders(c)
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			parentID := c.Param("id")
			childID := c.Param("child_id")

			if err := relManager.HandleDelete(ctx, parentID, childID); err != nil {
				return p.apiError(c, err)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"success": true,
				"message": fmt.Sprintf("Related %s record deleted successfully", relName),
			})
		}
		r.DELETE(relPath+"/:child_id", deleteRelHandler)
		r.POST(relPath+"/:child_id/delete", deleteRelHandler)
	}
}

func parseRequestData(c *gcontext.Context) map[string]any {
	data := make(map[string]any)
	if strings.Contains(c.Header("Content-Type"), "application/json") && c.Request.Body != nil {
		_ = json.NewDecoder(c.Request.Body).Decode(&data)
		return data
	}
	_ = c.Request.ParseForm()
	for k, v := range c.Request.PostForm {
		if len(v) == 1 {
			data[k] = v[0]
		} else {
			data[k] = v
		}
	}
	return data
}

// ErrForbidden is returned by the Invoke* resource methods when an
// authorization callback denies the operation. The API answers it with 403.
var ErrForbidden = errors.New("403 Forbidden: Access denied")

// APIErrorMapper translates a handler error into the HTTP status and message
// sent to API clients. Return handled=false to fall back to the default
// (403 for ErrForbidden, the error's own StatusCode() if it has one, else 500).
type APIErrorMapper func(err error) (status int, message string, handled bool)

// APIErrorHandler installs a mapper for errors returned by resource handlers,
// e.g. to turn unique-constraint violations into a friendly 409 and to keep
// raw database messages out of responses.
func (p *Panel) APIErrorHandler(fn APIErrorMapper) *Panel {
	p.apiErrorMapper = fn
	return p
}

// apiError writes the JSON error response for err (500 unless mapped).
func (p *Panel) apiError(c *gcontext.Context, err error) error {
	return p.apiErrorStatus(c, err, http.StatusInternalServerError)
}

// apiErrorStatus writes the JSON error response for err, using fallback as the
// status when neither the error itself nor the panel's mapper decides one.
func (p *Panel) apiErrorStatus(c *gcontext.Context, err error, fallback int) error {
	status, message := fallback, err.Error()
	var coded interface{ StatusCode() int }
	switch {
	case errors.Is(err, ErrForbidden):
		status = http.StatusForbidden
	case errors.As(err, &coded):
		if sc := coded.StatusCode(); sc >= 400 && sc < 600 {
			status = sc
		}
	}
	if p.apiErrorMapper != nil {
		if s, m, ok := p.apiErrorMapper(err); ok {
			status, message = s, m
		}
	}
	return c.JSON(status, map[string]string{"error": message})
}
