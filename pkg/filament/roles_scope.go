package filament

import (
	"net/http"
	"strings"
)

// Granting roles (Kubernetes-style escalation rules):
//   - you give or take a role only in a place where you hold roles.manage
//     or roles.assign;
//   - with roles.assign you only give roles whose permissions you have
//     there yourself (roles.manage or "*" may give any role);
//   - nobody's change may take away their own right to manage roles.

// errCannotGrant is returned for a binding the user may not give or take.
var errCannotGrant error = &endpointError{status: http.StatusForbidden, msg: "You cannot give or remove this role there."}

// canGrant reports whether the signed-in user may add or remove a binding
// of role in panel / tenant.
func (p *Panel) canGrant(ctx *Context, role RoleDef, panel, tenant string) bool {
	u := ctx.User()
	if p.roles == nil || u == nil {
		return false
	}
	mine := p.permissionsAt(ctx, u.GetAuthIdentifier(), panel, tenant)
	if mine["*"] || mine[PermManageRoles] {
		return true
	}
	if !mine[PermAssignRoles] {
		return false
	}
	for _, perm := range role.Permissions {
		if perm == "*" || perm == PermManageRoles || !mine[perm] {
			return false
		}
	}
	return true
}

// managerEverywhere reports whether userID can manage roles in every panel
// and tenant, given roles and bindings.
func managerEverywhere(roles []RoleDef, bindings []RoleBinding, userID string) bool {
	var mine []RoleBinding
	for _, b := range bindings {
		if b.UserID == userID {
			mine = append(mine, b)
		}
	}
	perms := permsOf(roles, mine, AnyScope, AnyScope)
	return perms["*"] || perms[PermManageRoles]
}

// keepsOwnAccess refuses a change (the roles and bindings after it) that
// removes the signed-in user's own right to manage roles, when they had it.
func (p *Panel) keepsOwnAccess(ctx *Context, roles []RoleDef, bindings []RoleBinding) error {
	uid := ctx.User().GetAuthIdentifier()
	before, _ := p.roles.store.ListBindings(ctx.GoContext(), BindingFilter{UserID: uid})
	oldRoles, _ := p.roles.store.ListRoles(ctx.GoContext())
	if managerEverywhere(oldRoles, before, uid) && !managerEverywhere(roles, bindings, uid) {
		return errLockout
	}
	// A manager of just this panel / tenant keeps that too.
	had := permsOf(oldRoles, before, p.id, ctx.Tenant())
	now := permsOf(roles, filterUser(bindings, uid), p.id, ctx.Tenant())
	for _, perm := range []string{"*", PermManageRoles, PermAssignRoles} {
		if had[perm] && !(now["*"] || now[PermManageRoles] || now[PermAssignRoles]) {
			return errLockout
		}
	}
	return nil
}

func filterUser(bs []RoleBinding, uid string) []RoleBinding {
	var out []RoleBinding
	for _, b := range bs {
		if b.UserID == uid {
			out = append(out, b)
		}
	}
	return out
}

// hasPanelAccess reports whether the user has a role binding in this panel.
func (p *Panel) hasPanelAccess(ctx *Context) bool {
	u := ctx.User()
	if u == nil {
		return true // sign-in is checked elsewhere
	}
	bindings, _ := p.roles.store.ListBindings(ctx.GoContext(), BindingFilter{UserID: u.GetAuthIdentifier()})
	for _, b := range bindings {
		if b.Panel == AnyScope || b.Panel == p.id {
			return true
		}
	}
	return false
}

// panelAccessRequired: with roles, users need a binding in the panel.
func (p *Panel) panelAccessRequired(path string) bool {
	if p.roles == nil || (p.roles.cfg.RequirePanelAccess != nil && !*p.roles.cfg.RequirePanelAccess) {
		return false
	}
	rest := strings.TrimPrefix(path, strings.TrimRight(p.path, "/"))
	for _, ok := range []string{"/logout", "/api/logout", "/api/me", "/api/boot", "/assets/", "/login", "/api/login"} {
		if rest == ok || strings.HasPrefix(rest, ok) {
			return false
		}
	}
	return true
}

// ScopeOption is a panel or tenant offered on the roles screens.
type ScopeOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// grantScopes lists the panels and tenants the user may give roles in
// ("*" first when they may give roles everywhere in that dimension).
func (p *Panel) grantScopes(ctx *Context) (panels, tenants []ScopeOption) {
	uid := ctx.User().GetAuthIdentifier()
	can := func(panel, tenant string) bool {
		perms := p.permissionsAt(ctx, uid, panel, tenant)
		return perms["*"] || perms[PermManageRoles] || perms[PermAssignRoles]
	}
	known := p.roles.cfg.Panels
	if len(known) == 0 {
		known = map[string]string{p.id: p.GetBrandName()}
	}
	if can(AnyScope, AnyScope) {
		panels = append(panels, ScopeOption{ID: AnyScope, Label: "All panels"})
	}
	ids := make([]string, 0, len(known))
	for id := range known {
		ids = append(ids, id)
	}
	sortStrings(ids)
	for _, id := range ids {
		if can(id, ctx.Tenant()) || can(id, AnyScope) {
			panels = append(panels, ScopeOption{ID: id, Label: known[id]})
		}
	}
	if p.tenantConfig != nil && p.tenantConfig.Enabled {
		if can(p.id, AnyScope) {
			tenants = append(tenants, ScopeOption{ID: AnyScope, Label: "All tenants"})
			var list []Tenant
			if p.roles.cfg.Tenants != nil {
				list, _ = p.roles.cfg.Tenants(ctx)
			} else if p.tenantConfig.TenantLoader != nil {
				list, _ = p.tenantConfig.TenantLoader(ctx)
			}
			for _, t := range list {
				tenants = append(tenants, ScopeOption{ID: t.GetTenantKey(), Label: t.GetTenantName()})
			}
		} else if t := ctx.TenantRecord(); t != nil {
			tenants = append(tenants, ScopeOption{ID: t.GetTenantKey(), Label: t.GetTenantName()})
		}
	} else {
		tenants = []ScopeOption{{ID: AnyScope, Label: "All tenants"}}
	}
	return panels, tenants
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
