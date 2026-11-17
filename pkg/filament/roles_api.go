package filament

import (
	"net/http"

	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
)

func (p *Panel) mountRoleAPI(r *router.Router, prefix string, preflight func(c *gcontext.Context) error) {
	if p.roles == nil {
		return
	}
	for _, path := range []string{"/roles", "/roles/:role", "/roles/users", "/roles/users/:user"} {
		r.OPTIONS(prefix+path, preflight)
	}
	guard := func(h func(c *gcontext.Context, ctx *Context) error) gcontext.HandlerFunc {
		return func(c *gcontext.Context) error {
			p.setCorsHeaders(c)
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			if !p.canManageRoles(ctx) {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Forbidden"})
			}
			return h(c, ctx)
		}
	}
	list := func(c *gcontext.Context, ctx *Context) error {
		roles, err := p.roles.store.ListRoles(ctx.GoContext())
		if err != nil {
			return p.apiError(c, err)
		}
		out := []map[string]any{}
		for _, role := range roles {
			users, _ := p.roles.store.RoleUsers(ctx.GoContext(), role.Name)
			perms := role.Permissions
			if perms == nil {
				perms = []string{}
			}
			out = append(out, map[string]any{"name": role.Name, "label": role.Label, "permissions": perms, "users": len(users)})
		}
		return c.JSON(http.StatusOK, map[string]any{
			"roles": out, "permissions": p.AllPermissions(), "can_find_users": p.roles.cfg.FindUsers != nil,
		})
	}
	r.GET(prefix+"/roles", guard(list))
	save := func(original string) func(c *gcontext.Context, ctx *Context) error {
		return func(c *gcontext.Context, ctx *Context) error {
			var in RoleDef
			if err := c.BindJSON(&in); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
			}
			orig := original
			if orig == ":param" {
				orig = c.Param("role")
			}
			if err := p.SaveRoleFor(ctx, orig, in); err != nil {
				return p.apiError(c, err)
			}
			return list(c, ctx)
		}
	}
	r.POST(prefix+"/roles", guard(save("")))
	r.GET(prefix+"/roles/users", guard(func(c *gcontext.Context, ctx *Context) error {
		if p.roles.cfg.FindUsers == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Not found"})
		}
		users, err := p.roles.cfg.FindUsers(ctx, c.Query("q"))
		if err != nil {
			return p.apiError(c, err)
		}
		out := []map[string]any{}
		for _, u := range users {
			rs, _ := p.roles.store.UserRoles(ctx.GoContext(), u.ID)
			if rs == nil {
				rs = []string{}
			}
			out = append(out, map[string]any{"id": u.ID, "name": u.Name, "roles": rs})
		}
		return c.JSON(http.StatusOK, map[string]any{"users": out})
	}))
	r.PUT(prefix+"/roles/users/:user", guard(func(c *gcontext.Context, ctx *Context) error {
		var in struct {
			Roles []string `json:"roles"`
		}
		if err := c.BindJSON(&in); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}
		if err := p.SetUserRolesFor(ctx, c.Param("user"), in.Roles); err != nil {
			return p.apiError(c, err)
		}
		return c.JSON(http.StatusOK, map[string]any{"id": c.Param("user"), "roles": in.Roles})
	}))
	r.PUT(prefix+"/roles/:role", guard(save(":param")))
	r.DELETE(prefix+"/roles/:role", guard(func(c *gcontext.Context, ctx *Context) error {
		if err := p.DeleteRoleFor(ctx, c.Param("role")); err != nil {
			return p.apiError(c, err)
		}
		return list(c, ctx)
	}))
}
