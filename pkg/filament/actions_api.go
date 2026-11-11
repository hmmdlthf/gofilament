package filament

import (
	"context"
	"errors"
	"net/http"

	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"

	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

type filamentCtxKey struct{}

// FromContext returns the panel request context carried by a standard
// context, so code that only receives a context.Context (action handlers,
// option loaders) can reach the signed-in user and the panel.
func FromContext(ctx context.Context) *Context {
	c, _ := ctx.Value(filamentCtxKey{}).(*Context)
	return c
}

// withSelf makes ctx.GoContext() carry ctx itself.
func withSelf(ctx *Context) {
	if ctx.Request != nil && FromContext(ctx.Request.Context()) == nil {
		ctx.Request = ctx.Request.WithContext(context.WithValue(ctx.Request.Context(), filamentCtxKey{}, ctx))
	}
}

// ErrActionUnavailable is returned when an action does not exist, is hidden
// for the record or the user may not run it.
var ErrActionUnavailable = errors.New("This action is not available for this record")

func (r *BaseResource[T]) table() *tables.Table[T] {
	if r.tableBuilder == nil {
		return nil
	}
	return r.tableBuilder(tables.New[T]())
}

// RowMeta reports, for each record of a list page, which edit / delete and
// custom actions the user can use on it.
func (r *BaseResource[T]) RowMeta(ctx *Context, records any) map[string]map[string]any {
	list, ok := records.([]T)
	if !ok || len(list) == 0 {
		return nil
	}
	withSelf(ctx)
	tbl := r.table()
	var custom []*tables.TableAction[T]
	if tbl != nil {
		for _, a := range tbl.GetActions() {
			if a.IsRunnable() || a.IsLink() {
				custom = append(custom, a)
			}
		}
	}
	out := make(map[string]map[string]any, len(list))
	for _, rec := range list {
		id := ""
		if r.idExtractor != nil {
			id = r.idExtractor(rec)
		} else if tbl != nil {
			id = tbl.ExtractRecordID(rec)
		}
		if id == "" {
			continue
		}
		actions := []string{}
		var links map[string]string
		for _, a := range custom {
			if !a.IsAvailable(ctx.GoContext(), rec) {
				continue
			}
			if a.IsLink() {
				u := a.LinkFor(rec)
				if u == "" {
					continue
				}
				if links == nil {
					links = map[string]string{}
				}
				links[a.GetName()] = u
			}
			actions = append(actions, a.GetName())
		}
		meta := map[string]any{
			"update":  r.onUpdate != nil && (r.canEdit == nil || r.canEdit(ctx, rec)),
			"delete":  r.onDelete != nil && (r.canDelete == nil || r.canDelete(ctx, rec)),
			"actions": actions,
		}
		if links != nil {
			meta["links"] = links
		}
		out[id] = meta
	}
	return out
}

func (r *BaseResource[T]) findForAction(ctx *Context, id, name string) (T, *tables.TableAction[T], error) {
	var zero T
	withSelf(ctx)
	tbl := r.table()
	if tbl == nil {
		return zero, nil, ErrActionUnavailable
	}
	action := tbl.FindAction(name)
	if action == nil || !action.IsRunnable() {
		return zero, nil, ErrActionUnavailable
	}
	if r.onFind == nil {
		return zero, nil, ErrActionUnavailable
	}
	record, err := r.onFind(ctx, id)
	if err != nil {
		return zero, nil, err
	}
	if r.canView != nil && !r.canView(ctx, record) {
		return zero, nil, ErrForbidden
	}
	if !action.IsAvailable(ctx.GoContext(), record) {
		return zero, nil, ErrActionUnavailable
	}
	return record, action, nil
}

// DescribeActionForm returns the form of a row action for a record.
func (r *BaseResource[T]) DescribeActionForm(ctx *Context, id, name string) (map[string]any, error) {
	_, action, err := r.findForAction(ctx, id, name)
	if err != nil {
		return nil, err
	}
	dto := action.ToDTO()
	return map[string]any{
		"mode":      "action",
		"action":    dto,
		"schema":    forms.Describe(ctx.GoContext(), action.FormComponents()),
		"state":     map[string]any{},
		"abilities": map[string]bool{"run": true},
	}, nil
}

// InvokeAction runs a row action on a record.
func (r *BaseResource[T]) InvokeAction(ctx *Context, id, name string, data map[string]any) (string, error) {
	record, action, err := r.findForAction(ctx, id, name)
	if err != nil {
		return "", err
	}
	if err := action.Run(ctx.GoContext(), record, data); err != nil {
		return "", err
	}
	msg := action.GetSuccessMessage()
	if msg == "" {
		msg = action.GetLabel() + " done"
	}
	return msg, nil
}

type actionInvoker interface {
	DescribeActionForm(ctx *Context, id, name string) (map[string]any, error)
	InvokeAction(ctx *Context, id, name string, data map[string]any) (string, error)
}

func (p *Panel) actionStatus(err error) int {
	if errors.Is(err, ErrActionUnavailable) {
		return http.StatusUnprocessableEntity
	}
	return http.StatusNotFound
}

// registerActionRoutes mounts the row action endpoints of a resource.
func (p *Panel) registerActionRoutes(r *router.Router, res ResourceInterface, resPath string, preflight gcontext.HandlerFunc) {
	invoker, ok := res.(actionInvoker)
	if !ok {
		return
	}
	path := resPath + "/:id/actions/:action"

	// GET .../{id}/actions/{action}/form
	r.OPTIONS(path+"/form", preflight)
	r.GET(path+"/form", func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		payload, err := invoker.DescribeActionForm(ctx, c.Param("id"), c.Param("action"))
		if err != nil {
			return p.apiErrorStatus(c, err, p.actionStatus(err))
		}
		return c.JSON(http.StatusOK, payload)
	})

	// POST .../{id}/actions/{action}
	r.OPTIONS(path, preflight)
	r.POST(path, func(c *gcontext.Context) error {
		setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		message, err := invoker.InvokeAction(ctx, c.Param("id"), c.Param("action"), parseRequestData(c))
		if err != nil {
			return p.apiErrorStatus(c, err, p.actionStatus(err))
		}
		return c.JSON(http.StatusOK, map[string]any{"success": true, "message": message})
	})
}

// CurrencyUsing sets the currency code shown on money columns that do not
// name one (for example the tenant's currency).
func (p *Panel) CurrencyUsing(fn func(ctx *Context) string) *Panel {
	p.currencyResolver = fn
	return p
}
