package filament

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/golaravel/gofilament/pkg/notifications"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
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
	r.OPTIONS(apiPrefix+"/global-search", corsPreflight)
	r.OPTIONS(apiPrefix+"/tenants", corsPreflight)
	r.OPTIONS(apiPrefix+"/notifications", corsPreflight)

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

		type resourceDTO struct {
			Slug            string `json:"slug"`
			ModelName       string `json:"model_name"`
			PluralModelName string `json:"plural_model_name"`
			NavIcon         string `json:"icon"`
			NavGroup        string `json:"group"`
		}

		var resourceDTOs []resourceDTO
		for _, res := range p.resources {
			item := res.NavigationItem(ctx)
			if item != nil {
				resourceDTOs = append(resourceDTOs, resourceDTO{
					Slug:            res.Slug(),
					ModelName:       item.Label,
					PluralModelName: item.Label,
					NavIcon:         item.Icon,
					NavGroup:        item.Group,
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
			"locale":            ctx.Locale(),
			"direction":         ctx.Direction(),
			"is_rtl":            ctx.IsRTL(),
			"supported_locales": p.locales,
			"default_locale":    p.GetDefaultLocale(),
			"locale_labels":     p.localeLabels,
			"active_tenant":     activeTenantKey,
			"tenants":           tenantsDTO,
			"navigation":        navDTOs,
			"resources":         resourceDTOs,
			"widgets_count":         len(p.widgets),
			"account_nav_placement": string(p.GetAccountNavPlacement()),
		}

		return c.JSON(http.StatusOK, schema)
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
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}

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
			"success":   true,
			"locale":    ctx.Locale(),
			"direction": ctx.Direction(),
			"is_rtl":    ctx.IsRTL(),
			"label":     p.GetLocaleLabel(ctx.Locale()),
			"flag":      p.GetLocaleFlag(ctx.Locale()),
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

		var widgetData []map[string]any
		for _, w := range p.widgets {
			dto := map[string]any{
				"id":          w.ID(),
				"sort":        w.Sort(),
				"column_span": w.ColumnSpan(),
			}
			if chart, ok := w.(interface {
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

	// 5. Register REST Endpoints for each resource
	for _, res := range p.resources {
		p.registerResourceAPI(res, apiPrefix, r)
	}
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
		perPage, _ := strconv.Atoi(ctx.DefaultQuery("perPage", "15"))

		filters := make(map[string]string)
		for k, v := range ctx.Request.URL.Query() {
			if strings.HasPrefix(k, "filter_") && len(v) > 0 {
				filters[strings.TrimPrefix(k, "filter_")] = v[0]
			}
		}

		// Generic handler invoker
		if invoker, ok := res.(interface {
			InvokeList(ctx *Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) (any, int, error)
		}); ok {
			records, total, err := invoker.InvokeList(ctx, search, sortCol, sortDir, page, perPage, filters)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
						return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
					}
					return c.JSON(http.StatusOK, result)
				}
			} else if jsonBody.CSV != "" {
				if importer, ok := res.(interface {
					InvokeImportCSV(ctx *Context, rdr io.Reader, explicitMapping map[string]string) (*ImportResult, error)
				}); ok {
					result, err := importer.InvokeImportCSV(ctx, strings.NewReader(jsonBody.CSV), jsonBody.Map)
					if err != nil {
						return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
		}

		if action == "delete_bulk" && len(ids) > 0 {
			if invoker, ok := res.(interface {
				InvokeDeleteBulk(ctx *Context, ids []string) error
			}); ok {
				if err := invoker.InvokeDeleteBulk(ctx, ids); err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
			return c.JSON(http.StatusOK, map[string]any{
				"data": record,
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
				return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
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
				return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
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
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			parentID := c.Param("id")
			payload := parseRequestData(c)

			if err := relManager.HandleCreate(ctx, parentID, payload); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			}
			return c.JSON(http.StatusCreated, map[string]any{
				"success": true,
				"message": fmt.Sprintf("Related %s record created successfully", relName),
			})
		})

		// PUT /admin/api/resources/:slug/:id/relations/:rel/:child_id - Update
		updateRelHandler := func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			parentID := c.Param("id")
			childID := c.Param("child_id")
			payload := parseRequestData(c)

			if err := relManager.HandleUpdate(ctx, parentID, childID, payload); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			parentID := c.Param("id")
			childID := c.Param("child_id")

			if err := relManager.HandleDelete(ctx, parentID, childID); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
