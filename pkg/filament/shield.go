package filament

import (
	"github.com/golaravel/golaravel/pkg/auth"
)

// ShieldUser extends auth.Authenticatable with role and permission checking.
type ShieldUser interface {
	auth.Authenticatable
	HasRole(role string) bool
	HasPermission(permission string) bool
}

// SimpleShieldUser provides a ready-to-use ShieldUser implementation.
type SimpleShieldUser struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

func (u *SimpleShieldUser) GetAuthIdentifier() string {
	return u.ID
}

func (u *SimpleShieldUser) HasRole(role string) bool {
	for _, r := range u.Roles {
		if r == role || r == "super-admin" || (r == "admin" && role != "super-admin") {
			return true
		}
	}
	return false
}

func (u *SimpleShieldUser) HasPermission(permission string) bool {
	for _, p := range u.Permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}

// Policy defines full authorization callbacks for an Ent entity or model type T.
type Policy[T any] interface {
	ViewAny(ctx *Context) bool
	View(ctx *Context, record T) bool
	Create(ctx *Context) bool
	Update(ctx *Context, record T) bool
	Delete(ctx *Context, record T) bool
	DeleteAny(ctx *Context) bool
}

// BasePolicy provides default allow-all implementations of Policy[T].
// Embed this in your custom resource policies and override only required rules.
type BasePolicy[T any] struct{}

func (BasePolicy[T]) ViewAny(ctx *Context) bool         { return true }
func (BasePolicy[T]) View(ctx *Context, record T) bool  { return true }
func (BasePolicy[T]) Create(ctx *Context) bool          { return true }
func (BasePolicy[T]) Update(ctx *Context, record T) bool { return true }
func (BasePolicy[T]) Delete(ctx *Context, record T) bool { return true }
func (BasePolicy[T]) DeleteAny(ctx *Context) bool       { return true }

// Role returns an authorization predicate requiring the user to have at least one of the specified roles.
func Role(roles ...string) func(ctx *Context) bool {
	return func(ctx *Context) bool {
		u := ctx.User()
		if u == nil {
			return false
		}
		if su, ok := u.(ShieldUser); ok {
			for _, r := range roles {
				if su.HasRole(r) {
					return true
				}
			}
			return false
		}
		return false
	}
}

// Permission returns an authorization predicate requiring the user to possess a specific permission.
func Permission(perm string) func(ctx *Context) bool {
	return func(ctx *Context) bool {
		u := ctx.User()
		if u == nil {
			return false
		}
		if su, ok := u.(ShieldUser); ok {
			return su.HasPermission(perm)
		}
		return false
	}
}

// Gate returns an authorization predicate verifying ability against GoLaravel's global auth.Gate.
func Gate(ability string, args ...any) func(ctx *Context) bool {
	return func(ctx *Context) bool {
		u := ctx.User()
		if u == nil {
			return false
		}
		return auth.Allows(ability, u, args...)
	}
}
