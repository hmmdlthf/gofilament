package filament

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/golaravel/gofilament/pkg/forms"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
)

var roleNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// errLockout is returned for a change that would remove the user's own
// right to manage roles.
var errLockout error = &endpointError{status: http.StatusConflict, msg: "This change would remove your own access to the roles screen."}

// SaveRoleFor creates or updates a role (renaming it from original).
// Only super admins (roles.manage everywhere) define roles.
func (p *Panel) SaveRoleFor(ctx *Context, original string, r RoleDef) error {
	if !p.canDefineRoles(ctx) {
		return ErrForbidden
	}
	r.Name = strings.TrimSpace(r.Name)
	r.Label = strings.TrimSpace(r.Label)
	errs := map[string]string{}
	if !roleNameRe.MatchString(r.Name) {
		errs["name"] = "Use lowercase letters, digits, - and _."
	}
	if r.Label == "" {
		r.Label = r.Name
	}
	known := p.AllPermissions()
	var perms []string
	seen := map[string]bool{}
	for _, perm := range r.Permissions {
		if seen[perm] {
			continue
		}
		seen[perm] = true
		if perm == "*" || known[perm] != "" {
			perms = append(perms, perm)
		} else {
			errs["permissions"] = "Unknown permission: " + perm
		}
	}
	sort.Strings(perms)
	r.Permissions = perms
	roles, err := p.roles.store.ListRoles(ctx.GoContext())
	if err != nil {
		return err
	}
	found := original == ""
	for _, x := range roles {
		if x.Name == r.Name && x.Name != original {
			errs["name"] = "A role with this name exists."
		}
		found = found || x.Name == original
	}
	if !found {
		return ErrRoleNotFound
	}
	if len(errs) > 0 {
		return &forms.ValidationError{Errors: errs}
	}
	// The roles and bindings after the change.
	var nextRoles []RoleDef
	for _, x := range roles {
		if x.Name != original {
			nextRoles = append(nextRoles, x)
		}
	}
	nextRoles = append(nextRoles, r)
	all, err := p.roles.store.ListBindings(ctx.GoContext(), BindingFilter{})
	if err != nil {
		return err
	}
	next := make([]RoleBinding, 0, len(all))
	for _, bnd := range all {
		if original != "" && bnd.Role == original {
			bnd.Role = r.Name
		}
		next = append(next, bnd)
	}
	if err := p.keepsOwnAccess(ctx, nextRoles, next); err != nil {
		return err
	}
	if err := p.roles.store.SaveRole(ctx.GoContext(), r); err != nil {
		return err
	}
	if original != "" && original != r.Name {
		for _, bnd := range all {
			if bnd.Role == original {
				moved := bnd
				moved.Role = r.Name
				if err := p.roles.store.AddBinding(ctx.GoContext(), moved); err != nil {
					return err
				}
			}
		}
		return p.roles.store.DeleteRole(ctx.GoContext(), original)
	}
	return nil
}

// DeleteRoleFor deletes a role and its bindings (super admins only).
func (p *Panel) DeleteRoleFor(ctx *Context, name string) error {
	if !p.canDefineRoles(ctx) {
		return ErrForbidden
	}
	roles, err := p.roles.store.ListRoles(ctx.GoContext())
	if err != nil {
		return err
	}
	var nextRoles []RoleDef
	for _, x := range roles {
		if x.Name != name {
			nextRoles = append(nextRoles, x)
		}
	}
	if len(nextRoles) == len(roles) {
		return ErrRoleNotFound
	}
	all, _ := p.roles.store.ListBindings(ctx.GoContext(), BindingFilter{})
	var next []RoleBinding
	for _, b := range all {
		if b.Role != name {
			next = append(next, b)
		}
	}
	if err := p.keepsOwnAccess(ctx, nextRoles, next); err != nil {
		return err
	}
	return p.roles.store.DeleteRole(ctx.GoContext(), name)
}

func (p *Panel) roleByName(ctx *Context, name string) (RoleDef, bool) {
	roles, _ := p.roles.store.ListRoles(ctx.GoContext())
	for _, r := range roles {
		if r.Name == name {
			return r, true
		}
	}
	return RoleDef{}, false
}

// GrantRole gives userID the role in every panel x tenant pair listed
// ("*" for all). All of them are checked before any is saved.
func (p *Panel) GrantRole(ctx *Context, userID, role string, panels, tenants []string) error {
	if !p.canManageRoles(ctx) {
		return ErrForbidden
	}
	r, ok := p.roleByName(ctx, role)
	if !ok {
		return &forms.ValidationError{Errors: map[string]string{"role": "Unknown role."}}
	}
	if strings.TrimSpace(userID) == "" || len(panels) == 0 || len(tenants) == 0 {
		return &forms.ValidationError{Errors: map[string]string{"scope": "Choose a user, at least one panel and at least one tenant."}}
	}
	var add []RoleBinding
	for _, pn := range panels {
		for _, tn := range tenants {
			if !p.canGrant(ctx, r, pn, tn) {
				return errCannotGrant
			}
			add = append(add, RoleBinding{UserID: userID, Role: role, Panel: pn, Tenant: tn})
		}
	}
	for _, b := range add {
		if err := p.roles.store.AddBinding(ctx.GoContext(), b); err != nil {
			return err
		}
	}
	return nil
}

// RevokeBinding removes one binding.
func (p *Panel) RevokeBinding(ctx *Context, b RoleBinding) error {
	if !p.canManageRoles(ctx) {
		return ErrForbidden
	}
	r, ok := p.roleByName(ctx, b.Role)
	if !ok || !p.canGrant(ctx, r, b.Panel, b.Tenant) {
		return errCannotGrant
	}
	roles, _ := p.roles.store.ListRoles(ctx.GoContext())
	all, _ := p.roles.store.ListBindings(ctx.GoContext(), BindingFilter{})
	var next []RoleBinding
	for _, x := range all {
		if x != b {
			next = append(next, x)
		}
	}
	if err := p.keepsOwnAccess(ctx, roles, next); err != nil {
		return err
	}
	return p.roles.store.RemoveBinding(ctx.GoContext(), b)
}

// visibleBindings: super admins see every binding; others see the ones
// that apply in the current panel and tenant.
func (p *Panel) visibleBindings(ctx *Context, userID string) []RoleBinding {
	all, _ := p.roles.store.ListBindings(ctx.GoContext(), BindingFilter{UserID: userID})
	if p.canDefineRoles(ctx) {
		return all
	}
	var out []RoleBinding
	for _, b := range all {
		if b.covers(p.id, ctx.Tenant()) {
			out = append(out, b)
		}
	}
	return out
}

// permissionGroups groups permission names by prefix, for the role form.
func (p *Panel) permissionGroups() ([]string, map[string][]string) {
	all := p.AllPermissions()
	groups := map[string][]string{}
	for perm := range all {
		g := perm
		if i := strings.Index(perm, "."); i > 0 {
			g = perm[:i]
		}
		groups[g] = append(groups[g], perm)
	}
	var names []string
	for g, ps := range groups {
		sort.Strings(ps)
		names = append(names, g)
	}
	sort.Strings(names)
	return names, groups
}

// renderPanelPage wraps content in the panel layout.
func (p *Panel) renderPanelPage(ctx *Context, status int, title, content string) error {
	page := fmt.Sprintf(`<!DOCTYPE html>
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
      <main class="filament-content %s">%s</main>
    </div>
  </div>
  %s
</body>
</html>`, ctx.Locale(), ctx.Direction(), html.EscapeString(title), p.GetBrandName(), p.path, p.path,
		p.RenderHeadHTML(ctx), p.RenderSidebar(ctx), p.RenderTopbar(ctx), p.GetMaxContentWidth(), content, p.RenderBodyEndHTML(ctx))
	return ctx.HTML(status, page)
}

func (p *Panel) renderRolesList(ctx *Context, status int, errMsg string) error {
	roles, _ := p.roles.store.ListRoles(ctx.GoContext())
	esc := html.EscapeString
	csrf := p.csrfHiddenInput(ctx.Context, "")
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="filament-header"><div><h1 class="filament-title">Roles</h1><p class="filament-description">What each role may do, and who has it.</p></div><div style="display: flex; gap: 0.5rem;">`)
	if p.roles.cfg.FindUsers != nil {
		fmt.Fprintf(&b, `<a href="%s/roles/users" class="btn btn-secondary">Users</a>`, p.path)
	}
	canDefine := p.canDefineRoles(ctx)
	if canDefine {
		fmt.Fprintf(&b, `<a href="%s/roles/new" class="btn btn-primary">New role</a>`, p.path)
	}
	fmt.Fprintf(&b, `</div></div>%s<div class="filament-card"><table class="filament-table"><thead><tr><th>Role</th><th>Permissions</th><th>Users</th><th></th></tr></thead><tbody>`, alertHTML("err", errMsg))
	for _, r := range roles {
		users, _ := p.roles.store.ListBindings(ctx.GoContext(), BindingFilter{Role: r.Name})
		perms := fmt.Sprintf("%d", len(r.Permissions))
		for _, x := range r.Permissions {
			if x == "*" {
				perms = "Everything"
			}
		}
		fmt.Fprintf(&b, `<tr data-role="%s"><td><a href="%s/roles/%s"><strong>%s</strong></a> <span style="color: var(--gray-500);">%s</span></td><td>%s</td><td>%d</td><td style="text-align: end;"><form method="POST" action="%s/roles/%s/delete" style="margin: 0;">%s<button type="submit" class="btn btn-danger btn-sm" onclick="return confirm('Delete this role?')">Delete</button></form></td></tr>`,
			esc(r.Name), p.path, esc(r.Name), esc(r.Label), esc(r.Name), perms, len(users), p.path, esc(r.Name), csrf)
	}
	if len(roles) == 0 {
		b.WriteString(`<tr><td colspan="4" style="text-align: center; color: var(--gray-500);">No roles yet.</td></tr>`)
	}
	b.WriteString(`</tbody></table></div>`)
	return p.renderPanelPage(ctx, status, "Roles", b.String())
}

func (p *Panel) renderRoleForm(ctx *Context, status int, original string, r RoleDef, errs map[string]string, errMsg string) error {
	esc := html.EscapeString
	has := map[string]bool{}
	for _, x := range r.Permissions {
		has[x] = true
	}
	labels := p.AllPermissions()
	groups, byGroup := p.permissionGroups()
	action := p.path + "/roles/new"
	title := "New role"
	if original != "" {
		action, title = p.path+"/roles/"+original, "Role: "+r.Label
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="filament-header"><div><h1 class="filament-title">%s</h1></div><a href="%s/roles" class="btn btn-secondary">Back</a></div>%s
<form method="POST" action="%s" class="filament-card">%s`, esc(title), p.path, alertHTML("err", errMsg), esc(action), p.csrfHiddenInput(ctx.Context, ""))
	field := func(name, label, value string) {
		fmt.Fprintf(&b, `<div class="form-group"><label class="form-label">%s</label><input class="form-control" name="%s" value="%s" />`, label, name, esc(value))
		if e := errs[name]; e != "" {
			fmt.Fprintf(&b, `<p style="color: #b91c1c; font-size: 0.8125rem;">%s</p>`, esc(e))
		}
		b.WriteString(`</div>`)
	}
	field("name", "Name (used in code)", r.Name)
	field("label", "Label", r.Label)
	checked := func(perm string) string {
		if has[perm] {
			return " checked"
		}
		return ""
	}
	fmt.Fprintf(&b, `<label style="display: flex; gap: 0.5rem; margin: 1rem 0;"><input type="checkbox" name="permissions" value="*"%s /> <strong>Everything</strong> (all permissions, now and later)</label>`, checked("*"))
	if e := errs["permissions"]; e != "" {
		fmt.Fprintf(&b, `<p style="color: #b91c1c;">%s</p>`, esc(e))
	}
	b.WriteString(`<div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(16rem, 1fr)); gap: 1rem;">`)
	for _, g := range groups {
		fmt.Fprintf(&b, `<fieldset style="border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 0.75rem;"><legend style="font-weight: 600; padding: 0 0.25rem;">%s</legend>`, esc(g))
		for _, perm := range byGroup[g] {
			fmt.Fprintf(&b, `<label style="display: flex; gap: 0.5rem; font-size: 0.875rem;"><input type="checkbox" name="permissions" value="%s"%s /> %s</label>`, esc(perm), checked(perm), esc(labels[perm]))
		}
		b.WriteString(`</fieldset>`)
	}
	b.WriteString(`</div><div style="margin-top: 1rem;"><button type="submit" class="btn btn-primary">Save role</button></div></form>`)
	return p.renderPanelPage(ctx, status, title, b.String())
}

func (p *Panel) scopeLabel(opts []ScopeOption, id, all string) string {
	if id == AnyScope {
		return all
	}
	for _, o := range opts {
		if o.ID == id {
			return o.Label
		}
	}
	return id
}

func (p *Panel) renderRoleUsers(ctx *Context, status int, errMsg string) error {
	esc := html.EscapeString
	q := strings.TrimSpace(ctx.Query("q"))
	users, err := p.roles.cfg.FindUsers(ctx, q)
	if err != nil {
		errMsg = "Users could not be loaded."
	}
	roles, _ := p.roles.store.ListRoles(ctx.GoContext())
	panels, tenants := p.grantScopes(ctx)
	csrf := p.csrfHiddenInput(ctx.Context, "")
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="filament-header"><div><h1 class="filament-title">Users and roles</h1><p class="filament-description">Give roles in one or more panels and tenants.</p></div><a href="%s/roles" class="btn btn-secondary">Roles</a></div>%s
<form method="GET" action="%s/roles/users" class="filament-card" style="display: flex; gap: 0.5rem;"><input class="form-control" name="q" value="%s" placeholder="Search users" /><button class="btn btn-secondary" type="submit">Search</button></form>`,
		p.path, alertHTML("err", errMsg), p.path, esc(q))
	for _, u := range users {
		fmt.Fprintf(&b, `<div class="filament-card" data-user="%s"><h3 style="font-weight: 700;">%s <span style="color: var(--gray-500); font-weight: 400;">%s</span></h3><table class="filament-table"><thead><tr><th>Role</th><th>Panel</th><th>Tenant</th><th></th></tr></thead><tbody>`, esc(u.ID), esc(u.Name), esc(u.ID))
		for _, bnd := range p.visibleBindings(ctx, u.ID) {
			remove := ""
			if r, ok := p.roleByName(ctx, bnd.Role); ok && p.canGrant(ctx, r, bnd.Panel, bnd.Tenant) {
				remove = fmt.Sprintf(`<form method="POST" action="%s/roles/users/%s/revoke" style="margin: 0;">%s<input type="hidden" name="q" value="%s" /><input type="hidden" name="role" value="%s" /><input type="hidden" name="panel" value="%s" /><input type="hidden" name="tenant" value="%s" /><button class="btn btn-danger btn-sm" type="submit">Remove</button></form>`,
					p.path, esc(u.ID), csrf, esc(q), esc(bnd.Role), esc(bnd.Panel), esc(bnd.Tenant))
			}
			fmt.Fprintf(&b, `<tr data-binding="%s|%s|%s"><td>%s</td><td>%s</td><td>%s</td><td style="text-align: end;">%s</td></tr>`,
				esc(bnd.Role), esc(bnd.Panel), esc(bnd.Tenant), esc(bnd.Role), esc(p.scopeLabel(panels, bnd.Panel, "All panels")), esc(p.scopeLabel(tenants, bnd.Tenant, "All tenants")), remove)
		}
		fmt.Fprintf(&b, `</tbody></table><form method="POST" action="%s/roles/users/%s/grant" style="display: flex; flex-wrap: wrap; gap: 1rem; align-items: flex-end; margin-top: 0.75rem;">%s<input type="hidden" name="q" value="%s" /><label>Role <select class="form-control" name="role">`, p.path, esc(u.ID), csrf, esc(q))
		for _, r := range roles {
			fmt.Fprintf(&b, `<option value="%s">%s</option>`, esc(r.Name), esc(r.Label))
		}
		b.WriteString(`</select></label><fieldset><legend>Panels</legend>`)
		for _, o := range panels {
			fmt.Fprintf(&b, `<label style="margin-inline-end: 0.75rem;"><input type="checkbox" name="panels" value="%s" /> %s</label>`, esc(o.ID), esc(o.Label))
		}
		b.WriteString(`</fieldset><fieldset><legend>Tenants</legend>`)
		for _, o := range tenants {
			fmt.Fprintf(&b, `<label style="margin-inline-end: 0.75rem;"><input type="checkbox" name="tenants" value="%s" /> %s</label>`, esc(o.ID), esc(o.Label))
		}
		b.WriteString(`</fieldset><button class="btn btn-primary btn-sm" type="submit">Give role</button></form></div>`)
	}
	return p.renderPanelPage(ctx, status, "Users and roles", b.String())
}

func roleErrorMessage(err error) (int, string, map[string]string) {
	var verr *forms.ValidationError
	if ok := errorsAs(err, &verr); ok {
		return http.StatusUnprocessableEntity, "", verr.Errors
	}
	return webStatus(err), err.Error(), nil
}

func (p *Panel) mountRoleRoutes(r *router.Router) {
	if p.roles == nil {
		return
	}
	guard := func(h func(c *gcontext.Context, ctx *Context) error) gcontext.HandlerFunc {
		return func(c *gcontext.Context) error {
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			if !p.canManageRoles(ctx) {
				return c.String(http.StatusForbidden, "403 Forbidden")
			}
			if c.Request.Method == http.MethodPost && !p.formGuard(c) {
				return c.String(http.StatusForbidden, "403 Forbidden: your session expired")
			}
			return h(c, ctx)
		}
	}
	base := p.path + "/roles"
	r.GET(base, guard(func(c *gcontext.Context, ctx *Context) error { return p.renderRolesList(ctx, http.StatusOK, "") }))
	r.GET(base+"/new", guard(func(c *gcontext.Context, ctx *Context) error {
		return p.renderRoleForm(ctx, http.StatusOK, "", RoleDef{}, nil, "")
	}))
	save := func(original string, c *gcontext.Context, ctx *Context) error {
		_ = c.Request.ParseForm()
		def := RoleDef{Name: c.Request.PostFormValue("name"), Label: c.Request.PostFormValue("label"), Permissions: c.Request.PostForm["permissions"]}
		if err := p.SaveRoleFor(ctx, original, def); err != nil {
			status, msg, errs := roleErrorMessage(err)
			return p.renderRoleForm(ctx, status, original, def, errs, msg)
		}
		return c.Redirect(http.StatusSeeOther, base)
	}
	r.POST(base+"/new", guard(func(c *gcontext.Context, ctx *Context) error { return save("", c, ctx) }))
	if p.roles.cfg.FindUsers != nil {
		r.GET(base+"/users", guard(func(c *gcontext.Context, ctx *Context) error {
			return p.renderRoleUsers(ctx, http.StatusOK, "")
		}))
		back := func(c *gcontext.Context) error {
			return c.Redirect(http.StatusSeeOther, base+"/users?q="+urlQueryEscape(c.Request.PostFormValue("q")))
		}
		r.POST(base+"/users/:user/grant", guard(func(c *gcontext.Context, ctx *Context) error {
			_ = c.Request.ParseForm()
			if err := p.GrantRole(ctx, c.Param("user"), c.Request.PostFormValue("role"), c.Request.PostForm["panels"], c.Request.PostForm["tenants"]); err != nil {
				_, msg, errs := roleErrorMessage(err)
				for _, m := range errs {
					msg = m
				}
				return p.renderRoleUsers(ctx, webStatus(err), msg)
			}
			return back(c)
		}))
		r.POST(base+"/users/:user/revoke", guard(func(c *gcontext.Context, ctx *Context) error {
			_ = c.Request.ParseForm()
			f := c.Request.PostFormValue
			if err := p.RevokeBinding(ctx, RoleBinding{UserID: c.Param("user"), Role: f("role"), Panel: f("panel"), Tenant: f("tenant")}); err != nil {
				return p.renderRoleUsers(ctx, webStatus(err), err.Error())
			}
			return back(c)
		}))
	}
	r.GET(base+"/:role", guard(func(c *gcontext.Context, ctx *Context) error {
		roles, _ := p.roles.store.ListRoles(ctx.GoContext())
		for _, x := range roles {
			if x.Name == c.Param("role") {
				return p.renderRoleForm(ctx, http.StatusOK, x.Name, x, nil, "")
			}
		}
		return c.String(http.StatusNotFound, "Role not found")
	}))
	r.POST(base+"/:role", guard(func(c *gcontext.Context, ctx *Context) error { return save(c.Param("role"), c, ctx) }))
	r.POST(base+"/:role/delete", guard(func(c *gcontext.Context, ctx *Context) error {
		if err := p.DeleteRoleFor(ctx, c.Param("role")); err != nil {
			return p.renderRolesList(ctx, webStatus(err), err.Error())
		}
		return c.Redirect(http.StatusSeeOther, base)
	}))
}

func errorsAs(err error, target any) bool { return errors.As(err, target) }

func urlQueryEscape(s string) string { return url.QueryEscape(s) }
