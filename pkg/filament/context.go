package filament

import (
	"context"

	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/privacy"
)

// Context wraps GoLaravel request context with Filament-specific metadata and helpers.
type Context struct {
	*gcontext.Context
	panel  *Panel
	tenant string
}

// NewContext creates a new Filament Context wrapper.
func NewContext(gc *gcontext.Context, p *Panel) *Context {
	ctx := &Context{
		Context: gc,
		panel:   p,
	}
	if p != nil && p.tenantConfig != nil && p.tenantConfig.Enabled {
		p.ResolveTenant(ctx)
	}
	return ctx
}

// Panel returns the current active Panel.
func (c *Context) Panel() *Panel {
	return c.panel
}

// User returns the authenticated user entity from the request session/token.
func (c *Context) User() auth.Authenticatable {
	if val, ok := c.Get(auth.ContextUserKey); ok {
		if u, ok := val.(auth.Authenticatable); ok {
			return u
		}
	}
	return nil
}

// Tenant returns the current tenant identifier if multi-tenancy is active.
func (c *Context) Tenant() string {
	if c.tenant != "" {
		return c.tenant
	}
	if c.Request != nil {
		if tid, ok := privacy.TenantFromContext(c.Request.Context()); ok && tid != "" {
			c.tenant = tid
			return tid
		}
	}
	// Check query, route param or session
	if t := c.Param("tenant"); t != "" {
		return t
	}
	if t, ok := c.Get("current_tenant"); ok {
		if s, ok := t.(string); ok {
			return s
		}
	}
	return ""
}

// TenantRecord returns the active Tenant model instance.
func (c *Context) TenantRecord() Tenant {
	if val, ok := c.Get("filament_tenant_record"); ok {
		if t, ok := val.(Tenant); ok {
			return t
		}
	}
	return nil
}

func (c *Context) setTenantInstance(t Tenant) {
	if t == nil {
		return
	}
	c.tenant = t.GetTenantKey()
	c.Set("current_tenant", t.GetTenantKey())
	c.Set("filament_tenant_record", t)
	if c.Request != nil {
		c.Request = c.Request.WithContext(privacy.WithTenantID(c.Request.Context(), t.GetTenantKey()))
	}
}

// SetTenant sets the active tenant identifier and injects it into GoLaravel's privacy context.
func (c *Context) SetTenant(t string) {
	c.tenant = t
	c.Set("current_tenant", t)
	if c.Request != nil {
		c.Request = c.Request.WithContext(privacy.WithTenantID(c.Request.Context(), t))
	}
}

// GoContext returns the standard library context.Context.
func (c *Context) GoContext() context.Context {
	if c == nil || c.Context == nil || c.Request == nil {
		return context.Background()
	}
	return c.Request.Context()
}
