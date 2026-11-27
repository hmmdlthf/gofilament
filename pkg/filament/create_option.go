package filament

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/golaravel/gofilament/pkg/forms"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
)

// Creating a select's missing option without leaving the form
// (forms.SelectField.CreateOptionFrom / CreateOptionForm):
//
//	GET  .../resources/{slug}/form/options/create?field=<name>  the create form
//	POST .../resources/{slug}/form/options/create?field=<name>  create it
//
// POST answers 201 {"option": {"value", "label"}}; the client adds the option
// and selects it.

// resourceBySlug returns the panel's resource with this slug.
func (p *Panel) resourceBySlug(slug string) ResourceInterface {
	for _, r := range p.resources {
		if r.Slug() == slug {
			return r
		}
	}
	return nil
}

// canCreateIn reports whether the user may create records of the resource.
func (p *Panel) canCreateIn(ctx *Context, slug string) bool {
	target, ok := p.resourceBySlug(slug).(interface {
		Meta(ctx *Context) ResourceMeta
	})
	return ok && gateAllows(ctx, target) && target.Meta(ctx).Abilities["create"]
}

// filterCreateOptions drops the create entry of selects whose target
// resource is missing or that the user may not create records in.
func (p *Panel) filterCreateOptions(ctx *Context, nodes []forms.Node) []forms.Node {
	for i := range nodes {
		n := &nodes[i]
		if n.CreateOption != nil && n.CreateOption.Resource != "" && (p == nil || !p.canCreateIn(ctx, n.CreateOption.Resource)) {
			n.CreateOption = nil
		}
		if len(n.Children) > 0 {
			n.Children = p.filterCreateOptions(ctx, n.Children)
		}
	}
	return nodes
}

var errNoCreatedID = errors.New("the create handler did not report the new record's id: call filament.SetCreatedID(ctx, id) or set data[\"id\"] in OnCreate")

func (p *Panel) registerCreateOptionRoutes(r *router.Router, res ResourceInterface, resPath string, preflight gcontext.HandlerFunc) {
	path := resPath + "/form/options/create"

	// field resolves the creatable select named by ?field= after checking
	// that the user may see this resource.
	field := func(c *gcontext.Context) (*Context, *forms.SelectField, error) {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return nil, nil, err
		}
		finder, ok := res.(interface {
			FormField(name string) forms.Component
		})
		if !ok || res.NavigationItem(ctx) == nil {
			return ctx, nil, p.apiError(c, ErrForbidden)
		}
		sel, ok := finder.FormField(c.Query("field")).(*forms.SelectField)
		if !ok || !sel.CanCreateOption() {
			return ctx, nil, c.JSON(http.StatusNotFound, map[string]string{"error": "This field has no create form"})
		}
		if slug := sel.CreateOptionResource(); slug != "" && !p.canCreateIn(ctx, slug) {
			return ctx, nil, p.apiError(c, ErrForbidden)
		}
		return ctx, sel, nil
	}

	r.OPTIONS(path, preflight)
	r.GET(path, func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, sel, err := field(c)
		if sel == nil {
			return err
		}
		var schema []forms.Node
		if slug := sel.CreateOptionResource(); slug != "" {
			describer, ok := p.resourceBySlug(slug).(interface {
				DescribeForm(ctx *Context) []forms.Node
			})
			if !ok {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "Unknown resource " + slug})
			}
			schema = describer.DescribeForm(ctx)
		} else {
			withSelf(ctx)
			schema = p.filterCreateOptions(ctx, forms.Describe(ctx.GoContext(), sel.CreateOptionComponents()))
		}
		return c.JSON(http.StatusOK, map[string]any{
			"mode":      "create",
			"schema":    schema,
			"state":     map[string]any{},
			"abilities": map[string]bool{"create": true},
		})
	})

	r.POST(path, func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, sel, err := field(c)
		if sel == nil {
			return err
		}
		data := parseRequestData(c)
		var opt forms.Option
		if slug := sel.CreateOptionResource(); slug != "" {
			creator, ok := p.resourceBySlug(slug).(interface {
				InvokeCreate(ctx *Context, data map[string]any) error
			})
			if !ok {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "Unknown resource " + slug})
			}
			if err := creator.InvokeCreate(ctx, data); err != nil {
				return p.apiErrorStatus(c, err, http.StatusBadRequest)
			}
			id := CreatedID(ctx)
			if id == "" {
				return p.apiError(c, errNoCreatedID)
			}
			withSelf(ctx)
			opt = forms.Option{Value: id, Label: sel.OptionLabel(ctx.GoContext(), id)}
			if opt.Label == "" || opt.Label == id {
				opt.Label = guessLabel(data, id)
			}
		} else {
			withSelf(ctx)
			clean, err := forms.Validate(ctx.GoContext(), sel.CreateOptionComponents(), data, true)
			if err != nil {
				return p.apiErrorStatus(c, err, http.StatusUnprocessableEntity)
			}
			opt, err = sel.CreateOptionHandler()(ctx.GoContext(), clean)
			if err != nil {
				return p.apiErrorStatus(c, err, http.StatusBadRequest)
			}
			if opt.Label == "" {
				opt.Label = opt.Value
			}
		}
		return c.JSON(http.StatusCreated, map[string]any{"option": opt})
	})
}

// guessLabel names a new option when the field cannot: a "name", "title" or
// "label" in the submitted data, else the id.
func guessLabel(data map[string]any, id string) string {
	for _, k := range []string{"name", "title", "label"} {
		if v, ok := data[k]; ok && v != nil && fmt.Sprint(v) != "" {
			return fmt.Sprint(v)
		}
	}
	return id
}
