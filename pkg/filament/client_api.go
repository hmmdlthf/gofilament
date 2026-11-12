package filament

import (
	"context"
	"net/http"
	"strconv"

	json "github.com/goccy/go-json"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"

	"github.com/golaravel/gofilament/pkg/colors"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

// Brand is the identity a panel presents: name, artwork and colours. A panel
// has a static brand (BrandName, Colors, ...); BrandUsing overrides it per
// request, e.g. per tenant or per host for white-label deployments.
type Brand struct {
	Name          string `json:"name"`
	Tagline       string `json:"tagline,omitempty"`
	LogoURL       string `json:"logo_url,omitempty"`
	LogoDarkURL   string `json:"logo_dark_url,omitempty"`
	IconURL       string `json:"icon_url,omitempty"`
	FaviconURL    string `json:"favicon_url,omitempty"`
	LoginImageURL string `json:"login_image_url,omitempty"`
	// PrimaryColor / AccentColor are hex colours ("#4F46E5"). When set they
	// replace the panel's primary / accent palettes.
	PrimaryColor string `json:"primary_color,omitempty"`
	AccentColor  string `json:"accent_color,omitempty"`
	// ThemeMode forces "light" or "dark"; empty follows the user's choice.
	ThemeMode     string `json:"theme_mode,omitempty"`
	FontFamily    string `json:"font_family,omitempty"`
	SupportEmail  string `json:"support_email,omitempty"`
	SupportPhone  string `json:"support_phone,omitempty"`
	HidePoweredBy bool   `json:"hide_powered_by,omitempty"`
}

// BrandResolver returns the brand for a request. Return nil to use the panel's
// static brand.
type BrandResolver func(ctx *Context) *Brand

// BrandUsing resolves the panel brand per request (tenant, host, partner).
// The resolver runs on every boot / schema request, so cache what it reads.
func (p *Panel) BrandUsing(fn BrandResolver) *Panel {
	p.brandResolver = fn
	return p
}

// ResolveBrand returns the brand for the request.
func (p *Panel) ResolveBrand(ctx *Context) Brand {
	b := Brand{Name: p.brandName, LogoURL: p.brandLogo, FaviconURL: p.favicon, FontFamily: p.fontFamily}
	if p.brandResolver != nil {
		if o := p.brandResolver(ctx); o != nil {
			if o.Name == "" {
				o.Name = b.Name
			}
			if o.LogoURL == "" {
				o.LogoURL = b.LogoURL
			}
			if o.FaviconURL == "" {
				o.FaviconURL = b.FaviconURL
			}
			if o.FontFamily == "" {
				o.FontFamily = b.FontFamily
			}
			return *o
		}
	}
	return b
}

// themeColors returns the panel palettes with the brand's colours applied.
func (p *Panel) themeColors(b Brand) map[string]map[string]string {
	out := make(map[string]map[string]string, len(p.colors)+2)
	add := func(role string, pal colors.Palette) {
		if pal == nil {
			return
		}
		m := make(map[string]string, len(pal))
		for shade, hex := range pal {
			m[strconv.Itoa(shade)] = hex
		}
		out[role] = m
	}
	for role, pal := range p.colors {
		add(role, pal)
	}
	if b.PrimaryColor != "" {
		add(colors.RolePrimary, colors.FromHex(b.PrimaryColor))
	}
	if b.AccentColor != "" {
		add("accent", colors.FromHex(b.AccentColor))
	}
	return out
}

// MaxPerPage caps the page size API clients may request (default 200), so a
// single request can never pull an unbounded number of rows.
func (p *Panel) MaxPerPage(n int) *Panel {
	p.maxPerPage = n
	return p
}

// GetMaxPerPage returns the page size cap.
func (p *Panel) GetMaxPerPage() int {
	if p.maxPerPage <= 0 {
		return 200
	}
	return p.maxPerPage
}

// ExportMaxRows sets how many rows a client-side export may collect (default
// 5000). Exports are built by the client from normal list pages.
func (p *Panel) ExportMaxRows(n int) *Panel {
	p.exportMaxRows = func(*Context) int { return n }
	return p
}

// ExportMaxRowsUsing resolves the export row limit per request, e.g. from the
// tenant's plan. Return 0 to disable exporting.
func (p *Panel) ExportMaxRowsUsing(fn func(ctx *Context) int) *Panel {
	p.exportMaxRows = fn
	return p
}

// ResourceMeta is what an API client needs to know about a resource beyond
// its columns and filters.
type ResourceMeta struct {
	HasForm     bool             `json:"has_form"`
	SoftDeletes bool             `json:"soft_deletes"`
	Table       tables.TableMeta `json:"table"`
	Abilities   map[string]bool  `json:"abilities"`
}

// DescribeForm returns the resource form as JSON nodes with dynamic options
// resolved for the request.
func (r *BaseResource[T]) DescribeForm(ctx *Context) []forms.Node {
	withSelf(ctx)
	if r.formBuilder == nil {
		return []forms.Node{}
	}
	return forms.Describe(ctx.GoContext(), r.formBuilder(forms.New()).GetSchema())
}

// FormField returns a field of the resource form by name.
func (r *BaseResource[T]) FormField(name string) forms.Component {
	if r.formBuilder == nil {
		return nil
	}
	return forms.FindField(r.formBuilder(forms.New()).GetSchema(), name)
}

// Meta describes the resource's actions and what the current user may do.
func (r *BaseResource[T]) Meta(ctx *Context) ResourceMeta {
	m := ResourceMeta{
		HasForm:     r.formBuilder != nil,
		SoftDeletes: r.IsSoftDeletesEnabled(),
		Table:       tables.TableMeta{Actions: []tables.ActionDTO{}, BulkActions: []tables.ActionDTO{}, HeaderActions: []tables.ActionDTO{}},
		Abilities: map[string]bool{
			"create":     r.onCreate != nil && (r.canCreate == nil || r.canCreate(ctx)),
			"delete_any": r.onDelete != nil && (r.canDeleteAny == nil || r.canDeleteAny(ctx)),
		},
	}
	if r.tableBuilder != nil {
		m.Table = r.tableBuilder(tables.New[T]()).Meta()
	}
	return m
}

// RecordAbilities loads a record and reports what the current user may do with it.
func (r *BaseResource[T]) RecordAbilities(ctx *Context, id string) (any, map[string]bool, error) {
	if r.onFind == nil {
		return nil, nil, ErrForbidden
	}
	record, err := r.onFind(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if r.canView != nil && !r.canView(ctx, record) {
		return nil, nil, ErrForbidden
	}
	return record, map[string]bool{
		"view":   true,
		"update": r.onUpdate != nil && (r.canEdit == nil || r.canEdit(ctx, record)),
		"delete": r.onDelete != nil && (r.canDelete == nil || r.canDelete(ctx, record)),
		// may_edit ignores whether the resource itself has an update handler:
		// it is the permission that governs changes to related rows.
		"may_edit": r.canEdit == nil || r.canEdit(ctx, record),
	}, nil
}

// decorateSchema adds the brand, brand colours and per-resource metadata to
// the panel schema payload.
func (p *Panel) decorateSchema(ctx *Context, schema map[string]any) {
	brand := p.ResolveBrand(ctx)
	schema["brand"] = brand
	schema["brand_name"] = brand.Name
	if theme, ok := schema["theme"].(map[string]any); ok {
		theme["colors"] = p.themeColors(brand)
		if brand.FontFamily != "" {
			theme["font_family"] = brand.FontFamily
		}
	}

	meta := make(map[string]ResourceMeta, len(p.resources))
	for _, res := range p.resources {
		if res.NavigationItem(ctx) == nil {
			continue
		}
		if d, ok := res.(interface {
			Meta(ctx *Context) ResourceMeta
		}); ok {
			meta[res.Slug()] = d.Meta(ctx)
		}
	}
	schema["resource_meta"] = meta
	schema["client_pages"] = p.clientPages(ctx)
	if p.currencyResolver != nil {
		schema["currency"] = p.currencyResolver(ctx)
	}

	maxRows := 5000
	if p.exportMaxRows != nil {
		maxRows = p.exportMaxRows(ctx)
	}
	schema["export"] = map[string]any{
		"enabled":      maxRows > 0,
		"max_rows":     maxRows,
		"max_per_page": p.GetMaxPerPage(),
		"formats":      []string{"csv", "xlsx"},
	}
}

// registerClientRoutes mounts the endpoints used by clients that render the
// panel themselves (the Flutter engine and other API clients).
func (p *Panel) registerClientRoutes(r *router.Router) {
	apiPrefix := p.path + "/api"
	preflight := func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		return c.NoContent(http.StatusNoContent)
	}

	// Public boot payload: everything needed to draw the login screen in the
	// right brand before the user is known. Contains no data.
	r.OPTIONS(apiPrefix+"/boot", preflight)
	r.GET(apiPrefix+"/boot", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx := NewContext(c, p)
		brand := p.ResolveBrand(ctx)
		return c.JSON(http.StatusOK, map[string]any{
			"id":         p.id,
			"path":       p.path,
			"brand":      brand,
			"brand_name": brand.Name,
			"dark_mode":  p.darkMode,
			"theme": map[string]any{
				"font_family": brand.FontFamily,
				"font_url":    p.fontURL,
				"colors":      p.themeColors(brand),
			},
			"auth": map[string]any{
				"required":       p.requiresAuth,
				"login":          p.hasLogin,
				"registration":   p.hasRegistration,
				"password_reset": p.hasPasswordReset,
			},
			"locale":            ctx.Locale(),
			"direction":         ctx.Direction(),
			"supported_locales": p.locales,
			"locale_labels":     p.GetLocaleLabels(),
			"translations":      p.GetTranslations(ctx.Locale()),
			"version":           p.GetVersion(),
			"build_id":          p.GetBuildID(),
		})
	})

	p.registerEndpoints(r, preflight)

	for _, res := range p.resources {
		res := res
		resPath := apiPrefix + "/resources/" + res.Slug()

		// GET .../form            -> create form: schema + defaults
		// GET .../form?id=<id>    -> edit form: schema + record + abilities
		r.OPTIONS(resPath+"/form", preflight)
		r.GET(resPath+"/form", func(c *gcontext.Context) error {
			p.setCorsHeaders(c)
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			describer, ok := res.(interface {
				DescribeForm(ctx *Context) []forms.Node
				Meta(ctx *Context) ResourceMeta
				RecordAbilities(ctx *Context, id string) (any, map[string]bool, error)
				FormField(name string) forms.Component
			})
			if !ok {
				return c.JSON(http.StatusNotImplemented, map[string]string{"error": "Form not available"})
			}

			id := c.Query("id")
			if id == "" {
				if !describer.Meta(ctx).Abilities["create"] {
					return p.apiError(c, ErrForbidden)
				}
				return c.JSON(http.StatusOK, map[string]any{
					"mode":      "create",
					"schema":    describer.DescribeForm(ctx),
					"state":     map[string]any{},
					"abilities": map[string]bool{"create": true},
				})
			}

			record, abilities, err := describer.RecordAbilities(ctx, id)
			if err != nil {
				return p.apiErrorStatus(c, err, http.StatusNotFound)
			}
			nodes := describer.DescribeForm(ctx)

			// Remote selects only carry the saved value: resolve its label so the
			// client can show it without another round trip.
			state := map[string]any{}
			if raw, err := json.Marshal(record); err == nil {
				_ = json.Unmarshal(raw, &state)
			}
			labels := map[string]string{}
			var walk func(ns []forms.Node)
			walk = func(ns []forms.Node) {
				for _, n := range ns {
					if n.Kind == "field" && n.Remote {
						if v, ok := state[n.Name]; ok && v != nil {
							if sel, ok := describer.FormField(n.Name).(*forms.SelectField); ok {
								val := formatScalar(v)
								labels[n.Name] = sel.OptionLabel(ctx.GoContext(), val)
							}
						}
					}
					walk(n.Children)
				}
			}
			walk(nodes)

			return c.JSON(http.StatusOK, map[string]any{
				"mode":          "edit",
				"schema":        nodes,
				"state":         record,
				"option_labels": labels,
				"abilities":     abilities,
			})
		})

		p.registerRelationClientRoutes(r, res, resPath, preflight)
		p.registerActionRoutes(r, res, resPath, preflight)

		// GET .../form/options?field=<name>&q=<text>&limit=<n>
		r.OPTIONS(resPath+"/form/options", preflight)
		r.GET(resPath+"/form/options", func(c *gcontext.Context) error {
			p.setCorsHeaders(c)
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			finder, ok := res.(interface {
				FormField(name string) forms.Component
				Meta(ctx *Context) ResourceMeta
			})
			if !ok {
				return c.JSON(http.StatusNotImplemented, map[string]string{"error": "Form not available"})
			}
			// Options can expose other records' names: require the right to see the resource.
			if res.NavigationItem(ctx) == nil {
				return p.apiError(c, ErrForbidden)
			}
			sel, ok := finder.FormField(c.Query("field")).(*forms.SelectField)
			if !ok {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "Unknown field"})
			}
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
			if limit <= 0 || limit > 100 {
				limit = 25
			}
			opts := sel.SearchOptions(ctx.GoContext(), c.Query("q"), limit)
			if opts == nil {
				opts = []forms.Option{}
			}
			return c.JSON(http.StatusOK, map[string]any{"options": opts})
		})
	}
}

// authorizeRelation checks that the user may see (or, for writes, edit) the
// parent record before any of its related rows are read or changed.
func (p *Panel) authorizeRelation(ctx *Context, res ResourceInterface, parentID string, write bool) error {
	guard, ok := res.(interface {
		RecordAbilities(ctx *Context, id string) (any, map[string]bool, error)
	})
	if !ok {
		return nil
	}
	finder, hasFinder := res.(interface{ HasFinder() bool })
	if hasFinder && !finder.HasFinder() {
		return nil // the resource cannot load records, so there is nothing to check against
	}
	_, abilities, err := guard.RecordAbilities(ctx, parentID)
	if err != nil {
		return err
	}
	if write && !abilities["may_edit"] {
		return ErrForbidden
	}
	return nil
}

// HasFinder reports whether the resource can load a single record.
func (r *BaseResource[T]) HasFinder() bool { return r.onFind != nil }

type relationParentKey struct{}

// RelationParentID returns the id of the parent record while a relation
// manager's form or options are being resolved ("" otherwise). Option loaders
// use it to offer choices that depend on the parent.
func RelationParentID(ctx context.Context) string {
	id, _ := ctx.Value(relationParentKey{}).(string)
	return id
}

func withRelationParent(ctx *Context, parentID string) {
	if ctx.Request != nil {
		ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), relationParentKey{}, parentID))
	}
}

// relationFormer is implemented by relation managers that can describe their form.
type relationFormer interface {
	DescribeForm(ctx *Context) []forms.Node
	FormField(name string) forms.Component
	HandleFind(ctx *Context, parentID, childID string) (any, error)
}

// registerRelationClientRoutes mounts the form endpoints of a resource's relation managers.
func (p *Panel) registerRelationClientRoutes(r *router.Router, res ResourceInterface, resPath string, preflight gcontext.HandlerFunc) {
	for _, rel := range res.GetRelations() {
		rel := rel
		former, ok := rel.(relationFormer)
		if !ok {
			continue
		}
		relPath := resPath + "/:id/relations/" + rel.Relationship()

		// GET .../relations/<rel>/form[?child_id=<id>]
		r.OPTIONS(relPath+"/form", preflight)
		r.GET(relPath+"/form", func(c *gcontext.Context) error {
			p.setCorsHeaders(c)
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			parentID := c.Param("id")
			childID := c.Query("child_id")
			if err := p.authorizeRelation(ctx, res, parentID, childID == ""); err != nil {
				return p.apiErrorStatus(c, err, http.StatusNotFound)
			}
			withRelationParent(ctx, parentID)
			nodes := former.DescribeForm(ctx)
			abilities, _ := rel.GetAPISchema()["abilities"].(map[string]bool)
			if childID == "" {
				return c.JSON(http.StatusOK, map[string]any{
					"mode": "create", "schema": nodes, "state": map[string]any{}, "abilities": abilities,
				})
			}
			record, err := former.HandleFind(ctx, parentID, childID)
			if err != nil {
				return p.apiErrorStatus(c, err, http.StatusNotFound)
			}
			state := map[string]any{}
			if raw, err := json.Marshal(record); err == nil {
				_ = json.Unmarshal(raw, &state)
			}
			labels := map[string]string{}
			var walk func(ns []forms.Node)
			walk = func(ns []forms.Node) {
				for _, n := range ns {
					if n.Kind == "field" && n.Remote {
						if v, ok := state[n.Name]; ok && v != nil {
							if sel, ok := former.FormField(n.Name).(*forms.SelectField); ok {
								labels[n.Name] = sel.OptionLabel(ctx.GoContext(), formatScalar(v))
							}
						}
					}
					walk(n.Children)
				}
			}
			walk(nodes)
			parentAbilities := map[string]bool{"may_edit": true}
			if guard, ok := res.(interface {
				RecordAbilities(ctx *Context, id string) (any, map[string]bool, error)
			}); ok {
				_, parentAbilities, _ = guard.RecordAbilities(ctx, parentID)
			}
			return c.JSON(http.StatusOK, map[string]any{
				"mode": "edit", "schema": nodes, "state": record, "option_labels": labels,
				"abilities": map[string]bool{
					"view":   true,
					"update": abilities["update"] && parentAbilities["may_edit"],
					"delete": abilities["delete"] && parentAbilities["may_edit"],
				},
			})
		})

		// GET .../relations/<rel>/form/options?field=&q=
		r.OPTIONS(relPath+"/form/options", preflight)
		r.GET(relPath+"/form/options", func(c *gcontext.Context) error {
			p.setCorsHeaders(c)
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			if err := p.authorizeRelation(ctx, res, c.Param("id"), false); err != nil {
				return p.apiErrorStatus(c, err, http.StatusNotFound)
			}
			withRelationParent(ctx, c.Param("id"))
			sel, ok := former.FormField(c.Query("field")).(*forms.SelectField)
			if !ok {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "Unknown field"})
			}
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
			if limit <= 0 || limit > 100 {
				limit = 25
			}
			opts := sel.SearchOptions(ctx.GoContext(), c.Query("q"), limit)
			if opts == nil {
				opts = []forms.Option{}
			}
			return c.JSON(http.StatusOK, map[string]any{"options": opts})
		})
	}
}

func formatScalar(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	}
	b, _ := json.Marshal(v)
	return string(b)
}
