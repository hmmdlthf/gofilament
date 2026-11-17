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
	for _, path := range []string{"/roles", "/roles/:role", "/roles/users", "/roles/bindings"} {
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
			holders, _ := p.roles.store.ListBindings(ctx.GoContext(), BindingFilter{Role: role.Name})
			perms := role.Permissions
			if perms == nil {
				perms = []string{}
			}
			out = append(out, map[string]any{"name": role.Name, "label": role.Label, "permissions": perms, "users": len(holders)})
		}
		panels, tenants := p.grantScopes(ctx)
		return c.JSON(http.StatusOK, map[string]any{
			"roles": out, "permissions": p.AllPermissions(),
			"can_find_users": p.roles.cfg.FindUsers != nil,
			"can_define":     p.canDefineRoles(ctx),
			"panels":         panels, "tenants": tenants,
		})
	}
	r.GET(prefix+"/roles", guard(list))
	save := func(fromPath bool) func(c *gcontext.Context, ctx *Context) error {
		return func(c *gcontext.Context, ctx *Context) error {
			var in RoleDef
			if err := c.BindJSON(&in); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
			}
			orig := ""
			if fromPath {
				orig = c.Param("role")
			}
			if err := p.SaveRoleFor(ctx, orig, in); err != nil {
				return p.apiError(c, err)
			}
			return list(c, ctx)
		}
	}
	r.POST(prefix+"/roles", guard(save(false)))
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
			bindings := []map[string]any{}
			for _, b := range p.visibleBindings(ctx, u.ID) {
				r, _ := p.roleByName(ctx, b.Role)
				bindings = append(bindings, map[string]any{"role": b.Role, "panel": b.Panel, "tenant": b.Tenant, "can_remove": p.canGrant(ctx, r, b.Panel, b.Tenant)})
			}
			out = append(out, map[string]any{"id": u.ID, "name": u.Name, "bindings": bindings})
		}
		return c.JSON(http.StatusOK, map[string]any{"users": out})
	}))
	// POST /roles/bindings {user, role, panels, tenants}: give a role.
	r.POST(prefix+"/roles/bindings", guard(func(c *gcontext.Context, ctx *Context) error {
		var in struct {
			User    string   `json:"user"`
			Role    string   `json:"role"`
			Panels  []string `json:"panels"`
			Tenants []string `json:"tenants"`
		}
		if err := c.BindJSON(&in); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}
		if err := p.GrantRole(ctx, in.User, in.Role, in.Panels, in.Tenants); err != nil {
			return p.apiError(c, err)
		}
		return c.JSON(http.StatusOK, map[string]any{"success": true})
	}))
	// DELETE /roles/bindings {user, role, panel, tenant}: take a role away.
	r.DELETE(prefix+"/roles/bindings", guard(func(c *gcontext.Context, ctx *Context) error {
		var in struct {
			User   string `json:"user"`
			Role   string `json:"role"`
			Panel  string `json:"panel"`
			Tenant string `json:"tenant"`
		}
		if err := c.BindJSON(&in); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
		}
		if err := p.RevokeBinding(ctx, RoleBinding{UserID: in.User, Role: in.Role, Panel: in.Panel, Tenant: in.Tenant}); err != nil {
			return p.apiError(c, err)
		}
		return c.JSON(http.StatusOK, map[string]any{"success": true})
	}))
	r.PUT(prefix+"/roles/:role", guard(save(true)))
	r.DELETE(prefix+"/roles/:role", guard(func(c *gcontext.Context, ctx *Context) error {
		if err := p.DeleteRoleFor(ctx, c.Param("role")); err != nil {
			return p.apiError(c, err)
		}
		return list(c, ctx)
	}))
}
