package filament

import (
	"context"
	"strings"
	"time"

	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/privacy"
	_ "github.com/golaravel/golaravel/pkg/translation"
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
	if p != nil {
		p.resolveLocale(ctx)
		if p.tenantConfig != nil && p.tenantConfig.Enabled {
			p.ResolveTenant(ctx)
		}
	}
	return ctx
}

// Panel returns the current active Panel.
func (c *Context) Panel() *Panel {
	return c.panel
}

// RememberMe returns whether the authentication request requested persistent Remember Me.
func (c *Context) RememberMe() bool {
	if val, ok := c.Get("remember"); ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

// TokenDuration returns the resolved session duration for this authentication request.
func (c *Context) TokenDuration() time.Duration {
	if val, ok := c.Get("token_duration"); ok {
		if d, ok := val.(time.Duration); ok {
			return d
		}
	}
	if c.panel != nil {
		return c.panel.ResolveTokenDuration(c.RememberMe())
	}
	return 24 * time.Hour
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

// Locale returns the active language code for this request (e.g. "en", "ar", "es").
func (c *Context) Locale() string {
	if c == nil || c.Context == nil {
		return "en"
	}
	return c.GetLocale()
}

// Direction returns "rtl" or "ltr" based on the active locale.
func (c *Context) Direction() string {
	if c == nil || c.Context == nil {
		return "ltr"
	}
	if c.IsRTL() {
		return "rtl"
	}
	return "ltr"
}

// IsRTL returns true if the active request locale is Right-to-Left (e.g. Arabic, Persian, Urdu, Hebrew).
func (c *Context) IsRTL() bool {
	if c == nil || c.Context == nil {
		return false
	}
	if c.Context.IsRTL() {
		return true
	}
	loc := strings.ToLower(strings.TrimSpace(c.Locale()))
	prefix := loc
	if idx := strings.IndexAny(loc, "-_"); idx != -1 {
		prefix = loc[:idx]
	}
	switch prefix {
	case "ar", "fa", "he", "ur", "yi", "ps", "sd", "ug", "ckb", "syr", "dv":
		return true
	default:
		return false
	}
}

// Trans translates the given translation key for the active request locale.
func (c *Context) Trans(key string, replace ...map[string]any) string {
	if c == nil || c.Context == nil {
		return key
	}
	return c.Context.Trans(key, replace...)
}

// TransChoice translates with pluralization for the active request locale.
func (c *Context) TransChoice(key string, number int, replace ...map[string]any) string {
	if c == nil || c.Context == nil {
		return key
	}
	return c.Context.TransChoice(key, number, replace...)
}
