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

func (p *Panel) currentUserRoles(ctx *Context) []string {
	rs, _ := p.roles.store.UserRoles(ctx.GoContext(), ctx.User().GetAuthIdentifier())
	return rs
}

// SaveRoleFor creates or updates a role (renaming it from original).
func (p *Panel) SaveRoleFor(ctx *Context, original string, r RoleDef) error {
	if !p.canManageRoles(ctx) {
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
		if (perm == "*" || known[perm] != "") && !seen[perm] {
			seen[perm] = true
			perms = append(perms, perm)
		} else if !seen[perm] {
			errs["permissions"] = "Unknown permission: " + perm
		}
	}
	sort.Strings(perms)
	r.Permissions = perms
	roles, err := p.roles.store.ListRoles(ctx.GoContext())
	if err != nil {
		return err
	}
	for _, x := range roles {
		if x.Name == r.Name && x.Name != original {
			errs["name"] = "A role with this name exists."
		}
	}
	if original != "" {
		found := false
		for _, x := range roles {
			found = found || x.Name == original
		}
		if !found {
			return ErrRoleNotFound
		}
	}
	if len(errs) > 0 {
		return &forms.ValidationError{Errors: errs}
	}
	// What the roles would be after the change.
	next := []RoleDef{}
	for _, x := range roles {
		if x.Name != original {
			next = append(next, x)
		}
	}
	next = append(next, r)
	mine := p.currentUserRoles(ctx)
	if original != "" && original != r.Name {
		mine = append(without(mine, original), r.Name)
	}
	if !p.keepsManager(ctx, next, mine) {
		return errLockout
	}
	if err := p.roles.store.SaveRole(ctx.GoContext(), r); err != nil {
		return err
	}
	if original != "" && original != r.Name {
		users, _ := p.roles.store.RoleUsers(ctx.GoContext(), original)
		for _, u := range users {
			rs, _ := p.roles.store.UserRoles(ctx.GoContext(), u)
			_ = p.roles.store.SetUserRoles(ctx.GoContext(), u, append(without(rs, original), r.Name))
		}
		_ = p.roles.store.DeleteRole(ctx.GoContext(), original)
	}
	return nil
}

// DeleteRoleFor deletes a role.
func (p *Panel) DeleteRoleFor(ctx *Context, name string) error {
	if !p.canManageRoles(ctx) {
		return ErrForbidden
	}
	roles, err := p.roles.store.ListRoles(ctx.GoContext())
	if err != nil {
		return err
	}
	var next []RoleDef
	for _, x := range roles {
		if x.Name != name {
			next = append(next, x)
		}
	}
	if len(next) == len(roles) {
		return ErrRoleNotFound
	}
	if !p.keepsManager(ctx, next, without(p.currentUserRoles(ctx), name)) {
		return errLockout
	}
	return p.roles.store.DeleteRole(ctx.GoContext(), name)
}

// SetUserRolesFor gives a user exactly these roles.
func (p *Panel) SetUserRolesFor(ctx *Context, userID string, names []string) error {
	if !p.canManageRoles(ctx) {
		return ErrForbidden
	}
	roles, err := p.roles.store.ListRoles(ctx.GoContext())
	if err != nil {
		return err
	}
	exists := map[string]bool{}
	for _, r := range roles {
		exists[r.Name] = true
	}
	var clean []string
	for _, n := range names {
		if !exists[n] {
			return &forms.ValidationError{Errors: map[string]string{"roles": "Unknown role: " + n}}
		}
		clean = append(clean, n)
	}
	if userID == ctx.User().GetAuthIdentifier() && !p.keepsManager(ctx, roles, clean) {
		return errLockout
	}
	return p.roles.store.SetUserRoles(ctx.GoContext(), userID, clean)
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
	fmt.Fprintf(&b, `<a href="%s/roles/new" class="btn btn-primary">New role</a></div></div>%s<div class="filament-card"><table class="filament-table"><thead><tr><th>Role</th><th>Permissions</th><th>Users</th><th></th></tr></thead><tbody>`, p.path, alertHTML("err", errMsg))
	for _, r := range roles {
		users, _ := p.roles.store.RoleUsers(ctx.GoContext(), r.Name)
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

func (p *Panel) renderRoleUsers(ctx *Context, status int, errMsg, okMsg string) error {
	esc := html.EscapeString
	q := strings.TrimSpace(ctx.Query("q"))
	users, err := p.roles.cfg.FindUsers(ctx, q)
	if err != nil {
		errMsg = "Users could not be loaded."
	}
	roles, _ := p.roles.store.ListRoles(ctx.GoContext())
	csrf := p.csrfHiddenInput(ctx.Context, "")
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="filament-header"><div><h1 class="filament-title">Users and roles</h1></div><a href="%s/roles" class="btn btn-secondary">Roles</a></div>%s%s
<form method="GET" action="%s/roles/users" class="filament-card" style="display: flex; gap: 0.5rem;"><input class="form-control" name="q" value="%s" placeholder="Search users" /><button class="btn btn-secondary" type="submit">Search</button></form>
<div class="filament-card"><table class="filament-table"><thead><tr><th>User</th><th>Roles</th></tr></thead><tbody>`,
		p.path, alertHTML("err", errMsg), alertHTML("ok", okMsg), p.path, esc(q))
	for _, u := range users {
		mine, _ := p.roles.store.UserRoles(ctx.GoContext(), u.ID)
		has := map[string]bool{}
		for _, r := range mine {
			has[r] = true
		}
		fmt.Fprintf(&b, `<tr data-user="%s"><td><strong>%s</strong><br><span style="color: var(--gray-500);">%s</span></td><td><form method="POST" action="%s/roles/users/%s" style="display: flex; flex-wrap: wrap; gap: 0.75rem; align-items: center; margin: 0;">%s<input type="hidden" name="q" value="%s" />`,
			esc(u.ID), esc(u.Name), esc(u.ID), p.path, esc(u.ID), csrf, esc(q))
		for _, r := range roles {
			c := ""
			if has[r.Name] {
				c = " checked"
			}
			fmt.Fprintf(&b, `<label style="display: flex; gap: 0.25rem;"><input type="checkbox" name="roles" value="%s"%s /> %s</label>`, esc(r.Name), c, esc(r.Label))
		}
		b.WriteString(`<button type="submit" class="btn btn-secondary btn-sm">Save</button></form></td></tr>`)
	}
	b.WriteString(`</tbody></table></div>`)
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
			return p.renderRoleUsers(ctx, http.StatusOK, "", "")
		}))
		r.POST(base+"/users/:user", guard(func(c *gcontext.Context, ctx *Context) error {
			_ = c.Request.ParseForm()
			if err := p.SetUserRolesFor(ctx, c.Param("user"), c.Request.PostForm["roles"]); err != nil {
				_, msg, errs := roleErrorMessage(err)
				if msg == "" {
					msg = errs["roles"]
				}
				return p.renderRoleUsers(ctx, webStatus(err), msg, "")
			}
			return c.Redirect(http.StatusSeeOther, base+"/users?q="+urlQueryEscape(c.Request.PostFormValue("q")))
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
