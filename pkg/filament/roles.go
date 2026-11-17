package filament

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
)

// Roles and permissions kept by the panel: admins create roles, tick their
// permissions and give roles to users from the panel (Panel.Roles).
//
// Permission names: "<resource>.viewAny", ".view", ".create", ".update",
// ".delete" for every resource, "roles.manage" for the roles screens, and
// any added with RolesConfig.Permissions. A role with "*" may do anything.

// RoleDef is a role and its permissions.
type RoleDef struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Permissions []string `json:"permissions"`
}

// RoleBinding gives a user a role in a place: a panel (by id) and a
// tenant (by key). "*" stands for every panel or every tenant.
type RoleBinding struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Panel  string `json:"panel"`
	Tenant string `json:"tenant"`
}

// AnyScope matches every panel or every tenant in a RoleBinding.
const AnyScope = "*"

// covers reports whether a binding's place includes panel and tenant
// ("*" asked for: the binding must be "*" too).
func (b RoleBinding) covers(panel, tenant string) bool {
	return (b.Panel == AnyScope || b.Panel == panel) && (b.Tenant == AnyScope || b.Tenant == tenant)
}

// BindingFilter selects bindings; empty fields match anything.
type BindingFilter struct {
	UserID string
	Role   string
}

// RoleStore keeps roles and who has them where.
type RoleStore interface {
	ListRoles(ctx context.Context) ([]RoleDef, error)
	SaveRole(ctx context.Context, r RoleDef) error
	// DeleteRole removes the role and its bindings.
	DeleteRole(ctx context.Context, name string) error
	ListBindings(ctx context.Context, f BindingFilter) ([]RoleBinding, error)
	// AddBinding is a no-op when the binding exists.
	AddBinding(ctx context.Context, b RoleBinding) error
	RemoveBinding(ctx context.Context, b RoleBinding) error
}

// ErrRoleNotFound is returned for an unknown role.
var ErrRoleNotFound = errors.New("role not found")

// MemoryRoleStore keeps roles in process memory.
type MemoryRoleStore struct {
	mu       sync.RWMutex
	roles    map[string]RoleDef
	bindings map[RoleBinding]bool
}

// NewMemoryRoleStore returns an empty in-memory role store.
func NewMemoryRoleStore() *MemoryRoleStore {
	return &MemoryRoleStore{roles: map[string]RoleDef{}, bindings: map[RoleBinding]bool{}}
}

func (m *MemoryRoleStore) ListRoles(context.Context) ([]RoleDef, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]RoleDef, 0, len(m.roles))
	for _, r := range m.roles {
		r.Permissions = append([]string(nil), r.Permissions...)
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *MemoryRoleStore) SaveRole(_ context.Context, r RoleDef) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r.Permissions = append([]string(nil), r.Permissions...)
	m.roles[r.Name] = r
	return nil
}

func (m *MemoryRoleStore) DeleteRole(_ context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.roles, name)
	for b := range m.bindings {
		if b.Role == name {
			delete(m.bindings, b)
		}
	}
	return nil
}

func (m *MemoryRoleStore) ListBindings(_ context.Context, f BindingFilter) ([]RoleBinding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []RoleBinding
	for b := range m.bindings {
		if (f.UserID == "" || b.UserID == f.UserID) && (f.Role == "" || b.Role == f.Role) {
			out = append(out, b)
		}
	}
	sortBindings(out)
	return out, nil
}

func (m *MemoryRoleStore) AddBinding(_ context.Context, b RoleBinding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bindings[b] = true
	return nil
}

func (m *MemoryRoleStore) RemoveBinding(_ context.Context, b RoleBinding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.bindings, b)
	return nil
}

func sortBindings(bs []RoleBinding) {
	sort.Slice(bs, func(i, j int) bool {
		a, b := bs[i], bs[j]
		if a.UserID != b.UserID {
			return a.UserID < b.UserID
		}
		if a.Role != b.Role {
			return a.Role < b.Role
		}
		if a.Panel != b.Panel {
			return a.Panel < b.Panel
		}
		return a.Tenant < b.Tenant
	})
}

// UserRef is a user shown on the roles screens.
type UserRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// RolesConfig configures Panel.Roles.
type RolesConfig struct {
	// Permissions adds app permissions to the list (name -> label).
	Permissions map[string]string
	// FindUsers searches users to give roles to (needed for the user
	// screen); an empty query lists the first users.
	FindUsers func(ctx *Context, query string) ([]UserRef, error)
	// EnforceResources checks the resource permissions on every resource
	// that has no Can* callback of its own (default true).
	EnforceResources *bool
	// RequirePanelAccess lets in only users with a role binding in this
	// panel (default true).
	RequirePanelAccess *bool
	// Panels names the panels sharing this role store (id -> label), so
	// roles can be given in several at once. Default: this panel.
	Panels map[string]string
	// Tenants lists the tenants roles can be given in. Default: the
	// panel's TenantLoader for the signed-in user.
	Tenants func(ctx *Context) ([]Tenant, error)
}

type rolesState struct {
	store RoleStore
	cfg   RolesConfig
}

// PermManageRoles defines roles and gives any role (where held).
const PermManageRoles = "roles.manage"

// PermAssignRoles gives roles, but only roles whose permissions the giver
// has, and only where the giver holds it (for example a tenant admin).
const PermAssignRoles = "roles.assign"

// Roles turns on roles and permissions kept in store (nil: in memory).
func (p *Panel) Roles(store RoleStore, cfg ...RolesConfig) *Panel {
	if store == nil {
		store = NewMemoryRoleStore()
	}
	c := RolesConfig{}
	if len(cfg) > 0 {
		c = cfg[0]
	}
	p.roles = &rolesState{store: store, cfg: c}
	return p
}

// RoleStore returns the panel's role store (nil without Panel.Roles).
func (p *Panel) RoleStore() RoleStore {
	if p.roles == nil {
		return nil
	}
	return p.roles.store
}

const ctxPermissions = "filament_permissions"

// userPermissions loads the user's permissions in the current panel and
// tenant once per request.
func (p *Panel) userPermissions(ctx *Context) map[string]bool {
	if v, ok := ctx.Get(ctxPermissions); ok {
		if m, ok := v.(map[string]bool); ok {
			return m
		}
	}
	perms := map[string]bool{}
	if u := ctx.User(); u != nil && p.roles != nil {
		perms = p.permissionsAt(ctx, u.GetAuthIdentifier(), p.id, ctx.Tenant())
	}
	ctx.Set(ctxPermissions, perms)
	return perms
}

// permissionsAt is the union of the permissions of a user's roles bound in
// a place covering panel and tenant (which may be "*").
func (p *Panel) permissionsAt(ctx *Context, userID, panel, tenant string) map[string]bool {
	perms := map[string]bool{}
	bindings, _ := p.roles.store.ListBindings(ctx.GoContext(), BindingFilter{UserID: userID})
	if len(bindings) == 0 {
		return perms
	}
	roles, _ := p.roles.store.ListRoles(ctx.GoContext())
	return permsOf(roles, bindings, panel, tenant)
}

func permsOf(roles []RoleDef, bindings []RoleBinding, panel, tenant string) map[string]bool {
	byName := map[string]RoleDef{}
	for _, r := range roles {
		byName[r.Name] = r
	}
	perms := map[string]bool{}
	for _, b := range bindings {
		if b.covers(panel, tenant) {
			for _, perm := range byName[b.Role].Permissions {
				perms[perm] = true
			}
		}
	}
	return perms
}

// UserCan reports whether the signed-in user has permission perm through
// the panel's roles (a "*" permission allows everything).
func (p *Panel) UserCan(ctx *Context, perm string) bool {
	if p.roles == nil {
		return false
	}
	perms := p.userPermissions(ctx)
	return perms["*"] || perms[perm]
}

// AllPermissions lists the permissions roles can have (name -> label),
// grouped by their prefix.
func (p *Panel) AllPermissions() map[string]string {
	out := map[string]string{PermManageRoles: "Manage roles"}
	for _, res := range p.resources {
		name := res.GetPluralModelName()
		for _, a := range []struct{ key, label string }{
			{"viewAny", "List"}, {"view", "View"}, {"create", "Create"}, {"update", "Edit"}, {"delete", "Delete"},
		} {
			out[res.Slug()+"."+a.key] = a.label + " " + strings.ToLower(name)
		}
	}
	if p.roles != nil {
		for k, v := range p.roles.cfg.Permissions {
			out[k] = v
		}
	}
	return out
}

// canDefineRoles: roles are shared by every panel and tenant, so only
// someone with roles.manage everywhere (a super admin) defines them.
func (p *Panel) canDefineRoles(ctx *Context) bool {
	if p.roles == nil || ctx.User() == nil {
		return false
	}
	perms := p.permissionsAt(ctx, ctx.User().GetAuthIdentifier(), AnyScope, AnyScope)
	return perms["*"] || perms[PermManageRoles]
}

// canManageRoles: the roles screens are open to whoever may give roles in
// the current panel and tenant.
func (p *Panel) canManageRoles(ctx *Context) bool {
	return p.roles != nil && ctx.User() != nil &&
		(p.UserCan(ctx, PermManageRoles) || p.UserCan(ctx, PermAssignRoles))
}

// usePanelPermissions is implemented by resources that can take their
// missing Can* checks from the panel's roles.
type usePanelPermissions interface {
	applyRolePermissions(p *Panel)
}

func (r *BaseResource[T]) applyRolePermissions(p *Panel) {
	can := func(action string) func(ctx *Context) bool {
		perm := r.slug + "." + action
		return func(ctx *Context) bool { return p.UserCan(ctx, perm) }
	}
	if r.canViewAny == nil {
		r.canViewAny = can("viewAny")
	}
	if r.canView == nil {
		f := can("view")
		r.canView = func(ctx *Context, _ T) bool { return f(ctx) }
	}
	if r.canCreate == nil {
		r.canCreate = can("create")
	}
	if r.canEdit == nil {
		f := can("update")
		r.canEdit = func(ctx *Context, _ T) bool { return f(ctx) }
	}
	if r.canDelete == nil {
		f := can("delete")
		r.canDelete = func(ctx *Context, _ T) bool { return f(ctx) }
	}
	if r.canDeleteAny == nil {
		r.canDeleteAny = can("delete")
	}
}

// enforceRoles fills in the resources' missing Can* checks.
func (p *Panel) enforceRoles() {
	if p.roles == nil || (p.roles.cfg.EnforceResources != nil && !*p.roles.cfg.EnforceResources) {
		return
	}
	for _, res := range p.resources {
		if u, ok := res.(usePanelPermissions); ok {
			u.applyRolePermissions(p)
		}
	}
}

