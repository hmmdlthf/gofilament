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

// RoleStore keeps roles and users' roles.
type RoleStore interface {
	ListRoles(ctx context.Context) ([]RoleDef, error)
	SaveRole(ctx context.Context, r RoleDef) error
	DeleteRole(ctx context.Context, name string) error
	UserRoles(ctx context.Context, userID string) ([]string, error)
	SetUserRoles(ctx context.Context, userID string, roles []string) error
	// RoleUsers lists the users holding a role.
	RoleUsers(ctx context.Context, role string) ([]string, error)
}

// ErrRoleNotFound is returned for an unknown role.
var ErrRoleNotFound = errors.New("role not found")

// MemoryRoleStore keeps roles in process memory.
type MemoryRoleStore struct {
	mu    sync.RWMutex
	roles map[string]RoleDef
	users map[string][]string
}

// NewMemoryRoleStore returns an empty in-memory role store.
func NewMemoryRoleStore() *MemoryRoleStore {
	return &MemoryRoleStore{roles: map[string]RoleDef{}, users: map[string][]string{}}
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
	for u, rs := range m.users {
		m.users[u] = without(rs, name)
	}
	return nil
}

func (m *MemoryRoleStore) UserRoles(_ context.Context, userID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]string(nil), m.users[userID]...), nil
}

func (m *MemoryRoleStore) SetUserRoles(_ context.Context, userID string, roles []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[userID] = append([]string(nil), roles...)
	return nil
}

func (m *MemoryRoleStore) RoleUsers(_ context.Context, role string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []string
	for u, rs := range m.users {
		for _, r := range rs {
			if r == role {
				out = append(out, u)
			}
		}
	}
	sort.Strings(out)
	return out, nil
}

func without(list []string, v string) []string {
	out := list[:0:0]
	for _, x := range list {
		if x != v {
			out = append(out, x)
		}
	}
	return out
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
}

type rolesState struct {
	store RoleStore
	cfg   RolesConfig
}

// PermManageRoles is the permission to use the roles screens.
const PermManageRoles = "roles.manage"

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

// userPermissions loads the user's permissions once per request.
func (p *Panel) userPermissions(ctx *Context) map[string]bool {
	if v, ok := ctx.Get(ctxPermissions); ok {
		if m, ok := v.(map[string]bool); ok {
			return m
		}
	}
	perms := map[string]bool{}
	u := ctx.User()
	if u != nil && p.roles != nil {
		names, _ := p.roles.store.UserRoles(ctx.GoContext(), u.GetAuthIdentifier())
		if len(names) > 0 {
			roles, _ := p.roles.store.ListRoles(ctx.GoContext())
			held := map[string]bool{}
			for _, n := range names {
				held[n] = true
			}
			for _, r := range roles {
				if held[r.Name] {
					for _, perm := range r.Permissions {
						perms[perm] = true
					}
				}
			}
		}
	}
	ctx.Set(ctxPermissions, perms)
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

func (p *Panel) canManageRoles(ctx *Context) bool {
	return p.roles != nil && ctx.User() != nil && p.UserCan(ctx, PermManageRoles)
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

// checkRoleChange refuses changes that leave the current user unable to
// manage roles (so nobody locks themselves out).
func (p *Panel) keepsManager(ctx *Context, roles []RoleDef, userRoles []string) bool {
	byName := map[string]RoleDef{}
	for _, r := range roles {
		byName[r.Name] = r
	}
	for _, n := range userRoles {
		for _, perm := range byName[n].Permissions {
			if perm == "*" || perm == PermManageRoles {
				return true
			}
		}
	}
	return false
}
