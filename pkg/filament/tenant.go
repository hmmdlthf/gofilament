package filament

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/golaravel/golaravel/pkg/privacy"
)

// Tenant represents an organization, team, or workspace entity in a multi-tenant application.
type Tenant interface {
	GetTenantKey() string
	GetTenantName() string
	GetTenantAvatar() string
}

// TenantItem is a convenient standard struct implementation of the Tenant interface.
type TenantItem struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
	Slug   string `json:"slug,omitempty"`
}

func (t TenantItem) GetTenantKey() string {
	if t.Key != "" {
		return t.Key
	}
	return t.Slug
}

func (t TenantItem) GetTenantName() string {
	if t.Name != "" {
		return t.Name
	}
	return t.Key
}

func (t TenantItem) GetTenantAvatar() string {
	return t.Avatar
}

// TenantConfig configures multi-tenant isolation, resolution, and switcher UI for a Panel.
type TenantConfig struct {
	Enabled         bool
	PathScoping     bool
	DomainScoping   string
	CookieName      string
	HeaderName      string
	RegistrationURL string
	ProfileURL      string
	TenantLoader    func(ctx *Context) ([]Tenant, error)
	CurrentTenant   func(ctx *Context) (Tenant, error)
	OnTenantSwitch  func(ctx *Context, newTenant Tenant) error
}

// DefaultTenantConfig returns sensible defaults for panel multi-tenancy.
func DefaultTenantConfig() *TenantConfig {
	return &TenantConfig{
		Enabled:    true,
		CookieName: "filament_tenant",
		HeaderName: "X-Tenant-ID",
	}
}

// Tenancy enables multi-tenancy on the panel with optional configuration.
func (p *Panel) Tenancy(config ...*TenantConfig) *Panel {
	if len(config) > 0 && config[0] != nil {
		p.tenantConfig = config[0]
		p.tenantConfig.Enabled = true
		if p.tenantConfig.CookieName == "" {
			p.tenantConfig.CookieName = "filament_tenant"
		}
		if p.tenantConfig.HeaderName == "" {
			p.tenantConfig.HeaderName = "X-Tenant-ID"
		}
	} else {
		p.tenantConfig = DefaultTenantConfig()
	}
	p.hasTenantScoping = true
	return p
}

// TenantLoader configures the callback to load all tenants available to the current user.
func (p *Panel) TenantLoader(fn func(ctx *Context) ([]Tenant, error)) *Panel {
	if p.tenantConfig == nil {
		p.Tenancy()
	}
	p.tenantConfig.TenantLoader = fn
	return p
}

// CurrentTenant sets a custom callback to resolve the active tenant from context.
func (p *Panel) CurrentTenant(fn func(ctx *Context) (Tenant, error)) *Panel {
	if p.tenantConfig == nil {
		p.Tenancy()
	}
	p.tenantConfig.CurrentTenant = fn
	return p
}

// TenantPathScoping enables or disables path-based tenant prefixes in URLs (e.g. /admin/{tenant}/...).
func (p *Panel) TenantPathScoping(enabled ...bool) *Panel {
	if p.tenantConfig == nil {
		p.Tenancy()
	}
	if len(enabled) > 0 {
		p.tenantConfig.PathScoping = enabled[0]
	} else {
		p.tenantConfig.PathScoping = true
	}
	return p
}

// TenantDomainScoping scopes tenancy via subdomains (e.g. {tenant}.domain.com/admin).
func (p *Panel) TenantDomainScoping(baseDomain string) *Panel {
	if p.tenantConfig == nil {
		p.Tenancy()
	}
	p.tenantConfig.DomainScoping = baseDomain
	return p
}

// TenantRegistration sets the URL for creating a new tenant organization.
func (p *Panel) TenantRegistration(url string) *Panel {
	if p.tenantConfig == nil {
		p.Tenancy()
	}
	p.tenantConfig.RegistrationURL = url
	return p
}

// TenantProfile sets the URL for viewing/editing active tenant settings.
func (p *Panel) TenantProfile(url string) *Panel {
	if p.tenantConfig == nil {
		p.Tenancy()
	}
	p.tenantConfig.ProfileURL = url
	return p
}

// GetTenantConfig returns the active TenantConfig for the panel.
func (p *Panel) GetTenantConfig() *TenantConfig {
	return p.tenantConfig
}

// TenantURL generates a tenant-scoped URL if PathScoping is enabled, otherwise standard panel path.
func (p *Panel) TenantURL(ctx *Context, subpath string) string {
	subpath = "/" + strings.TrimPrefix(subpath, "/")
	if p.tenantConfig != nil && p.tenantConfig.Enabled && p.tenantConfig.PathScoping {
		tenantKey := ctx.Tenant()
		if tenantKey != "" {
			return fmt.Sprintf("%s/%s%s", p.path, tenantKey, subpath)
		}
	}
	return fmt.Sprintf("%s%s", p.path, subpath)
}

// ResolveTenant extracts and sets the active tenant for the request context.
func (p *Panel) ResolveTenant(ctx *Context) Tenant {
	if p.tenantConfig == nil || !p.tenantConfig.Enabled {
		return nil
	}

	// 1. Explicit resolver callback
	if p.tenantConfig.CurrentTenant != nil {
		if t, err := p.tenantConfig.CurrentTenant(ctx); err == nil && t != nil {
			ctx.setTenantInstance(t)
			return t
		}
	}

	// 2. Check route param :tenant
	var tenantKey string
	if p.tenantConfig.PathScoping {
		tenantKey = ctx.Param("tenant")
	}

	// 3. Check Cookie
	if tenantKey == "" && p.tenantConfig.CookieName != "" && ctx.Request != nil {
		if cookie, err := ctx.Request.Cookie(p.tenantConfig.CookieName); err == nil && cookie.Value != "" {
			tenantKey = cookie.Value
		}
	}

	// 4. Check HTTP Header
	if tenantKey == "" && p.tenantConfig.HeaderName != "" && ctx.Request != nil {
		tenantKey = ctx.Header(p.tenantConfig.HeaderName)
	}

	// 5. Check Subdomain
	if tenantKey == "" && p.tenantConfig.DomainScoping != "" && ctx.Request != nil {
		host := ctx.Request.Host
		if idx := strings.IndexByte(host, ':'); idx != -1 {
			host = host[:idx]
		}
		base := strings.ToLower(strings.TrimPrefix(p.tenantConfig.DomainScoping, "."))
		if strings.HasSuffix(strings.ToLower(host), "."+base) {
			sub := strings.TrimSuffix(strings.ToLower(host), "."+base)
			if sub != "" && sub != "www" {
				tenantKey = sub
			}
		}
	}

	// 6. Check existing context / GoLaravel privacy context
	if tenantKey == "" && ctx.Request != nil {
		if tid, ok := privacy.TenantFromContext(ctx.Request.Context()); ok && tid != "" {
			tenantKey = tid
		}
	}

	// 7. If we have a tenantLoader, validate or select first default tenant
	if p.tenantConfig.TenantLoader != nil {
		tenants, err := p.tenantConfig.TenantLoader(ctx)
		if err == nil && len(tenants) > 0 {
			if tenantKey != "" {
				for _, t := range tenants {
					if t.GetTenantKey() == tenantKey {
						ctx.setTenantInstance(t)
						return t
					}
				}
			}
			// Default to first tenant
			first := tenants[0]
			ctx.setTenantInstance(first)
			return first
		}
	}

	// 8. If tenantKey was identified without a loader, construct standard TenantItem
	if tenantKey != "" {
		item := &TenantItem{Key: tenantKey, Name: tenantKey}
		ctx.setTenantInstance(item)
		return item
	}

	return nil
}

// renderTenantSwitcherHTML renders the dropdown UI component for the Topbar.
func (p *Panel) renderTenantSwitcherHTML(ctx *Context) string {
	if p.tenantConfig == nil || !p.tenantConfig.Enabled {
		return ""
	}

	currentTenant := ctx.TenantRecord()
	tenantName := "Select Organization"
	tenantKey := ""
	avatarHTML := `<span style="font-size: 0.75rem; font-weight: 700;">🏢</span>`

	if currentTenant != nil {
		tenantName = currentTenant.GetTenantName()
		tenantKey = currentTenant.GetTenantKey()
		if currentTenant.GetTenantAvatar() != "" {
			avatarHTML = fmt.Sprintf(`<img src="%s" alt="%s" style="width: 1.5rem; height: 1.5rem; border-radius: 0.375rem; object-fit: cover;" />`,
				currentTenant.GetTenantAvatar(), tenantName)
		} else {
			initials := getInitials(tenantName)
			avatarHTML = fmt.Sprintf(`<span style="font-size: 0.6875rem; font-weight: 700; color: #fff;">%s</span>`, initials)
		}
	}

	var allTenants []Tenant
	if p.tenantConfig.TenantLoader != nil {
		if list, err := p.tenantConfig.TenantLoader(ctx); err == nil {
			allTenants = list
		}
	}

	var sb strings.Builder
	sb.WriteString(`<div class="filament-tenant-switcher" style="position: relative;" x-data="{ open: false }">`)
	sb.WriteString(fmt.Sprintf(`
  <button type="button" @click="open = !open" class="btn btn-secondary btn-sm" style="display: flex; align-items: center; gap: 0.5rem; border-radius: 0.5rem; padding: 0.25rem 0.625rem; font-weight: 600; cursor: pointer; border: 1px solid var(--gray-200); background: var(--gray-50);">
    <div style="width: 1.5rem; height: 1.5rem; border-radius: 0.375rem; background-color: var(--primary-600); display: flex; align-items: center; justify-content: center; flex-shrink: 0;">
      %s
    </div>
    <span style="max-width: 9rem; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 0.8125rem;">%s</span>
    <svg style="width: 0.875rem; height: 0.875rem; color: var(--gray-400); flex-shrink: 0;" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
    </svg>
  </button>`, avatarHTML, tenantName))

	sb.WriteString(`
  <div x-show="open" @click.outside="open = false" style="display: none; position: absolute; left: 0; top: 100%; margin-top: 0.375rem; width: 15rem; background: #fff; border: 1px solid var(--gray-200); border-radius: 0.5rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1); z-index: 50; overflow: hidden;">
    <div style="padding: 0.5rem 0.75rem; font-size: 0.6875rem; font-weight: 700; text-transform: uppercase; color: var(--gray-400); border-bottom: 1px solid var(--gray-100); display: flex; justify-content: space-between; align-items: center;">
      <span>Tenants</span>
      <span style="font-weight: 500;">` + fmt.Sprintf("%d", len(allTenants)) + `</span>
    </div>
    <div style="max-height: 14rem; overflow-y: auto;">`)

	for _, t := range allTenants {
		isActive := t.GetTenantKey() == tenantKey
		activeBg := ""
		checkIcon := ""
		if isActive {
			activeBg = "background-color: var(--primary-50);"
			checkIcon = `<span style="color: var(--primary-600); font-weight: 700; font-size: 0.875rem;">✓</span>`
		}

		tInitials := getInitials(t.GetTenantName())
		switchURL := fmt.Sprintf("%s/tenant/switch?tenant=%s", p.path, url.QueryEscape(t.GetTenantKey()))

		sb.WriteString(fmt.Sprintf(`
      <a href="%s" style="display: flex; align-items: center; justify-content: space-between; padding: 0.5rem 0.75rem; text-decoration: none; color: inherit; font-size: 0.8125rem; %s" onmouseover="if(!%t) this.style.backgroundColor='var(--gray-50)'" onmouseout="if(!%t) this.style.backgroundColor='transparent'">
        <div style="display: flex; align-items: center; gap: 0.5rem; overflow: hidden;">
          <div style="width: 1.375rem; height: 1.375rem; border-radius: 0.25rem; background-color: var(--gray-200); display: flex; align-items: center; justify-content: center; font-size: 0.625rem; font-weight: 700; flex-shrink: 0;">
            %s
          </div>
          <span style="overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">%s</span>
        </div>
        %s
      </a>`, switchURL, activeBg, isActive, isActive, tInitials, t.GetTenantName(), checkIcon))
	}

	// Action links (Profile & New Organization)
	hasBottomActions := p.tenantConfig.ProfileURL != "" || p.tenantConfig.RegistrationURL != ""
	if hasBottomActions {
		sb.WriteString(`<div style="border-top: 1px solid var(--gray-100); padding: 0.375rem; background: var(--gray-50);">`)
		if p.tenantConfig.ProfileURL != "" {
			profileURL := p.tenantConfig.ProfileURL
			if tenantKey != "" {
				profileURL = strings.ReplaceAll(profileURL, "{tenant}", tenantKey)
			}
			sb.WriteString(fmt.Sprintf(`
        <a href="%s" style="display: flex; align-items: center; gap: 0.5rem; padding: 0.375rem 0.5rem; border-radius: 0.375rem; font-size: 0.75rem; color: var(--gray-600); text-decoration: none;" onmouseover="this.style.backgroundColor='var(--gray-100)'" onmouseout="this.style.backgroundColor='transparent'">
          <span>⚙️</span>
          <span>Tenant Settings</span>
        </a>`, profileURL))
		}
		if p.tenantConfig.RegistrationURL != "" {
			sb.WriteString(fmt.Sprintf(`
        <a href="%s" style="display: flex; align-items: center; gap: 0.5rem; padding: 0.375rem 0.5rem; border-radius: 0.375rem; font-size: 0.75rem; color: var(--primary-600); font-weight: 600; text-decoration: none;" onmouseover="this.style.backgroundColor='var(--gray-100)'" onmouseout="this.style.backgroundColor='transparent'">
          <span>➕</span>
          <span>New Tenant</span>
        </a>`, p.tenantConfig.RegistrationURL))
		}
		sb.WriteString(`</div>`)
	}

	sb.WriteString(`
    </div>
  </div>
</div>`)

	return sb.String()
}

func getInitials(name string) string {
	parts := strings.Fields(name)
	if len(parts) == 0 {
		return "T"
	}
	if len(parts) == 1 {
		runes := []rune(parts[0])
		if len(runes) >= 2 {
			return strings.ToUpper(string(runes[:2]))
		}
		return strings.ToUpper(string(runes[:1]))
	}
	r1 := []rune(parts[0])
	r2 := []rune(parts[1])
	return strings.ToUpper(string(r1[:1]) + string(r2[:1]))
}
