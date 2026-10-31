package filament

import (
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"

	"github.com/golaravel/gofilament/pkg/colors"
)

var presetPalettes = map[string]colors.Palette{
	"amber":   colors.Amber,
	"blue":    colors.Blue,
	"cyan":    colors.Cyan,
	"emerald": colors.Emerald,
	"fuchsia": colors.Fuchsia,
	"gray":    colors.Gray,
	"green":   colors.Green,
	"indigo":  colors.Indigo,
	"lime":    colors.Lime,
	"neutral": colors.Neutral,
	"orange":  colors.Orange,
	"pink":    colors.Pink,
	"purple":  colors.Purple,
	"red":     colors.Red,
	"rose":    colors.Rose,
	"sky":     colors.Sky,
	"slate":   colors.Slate,
	"stone":   colors.Stone,
	"teal":    colors.Teal,
	"violet":  colors.Violet,
	"yellow":  colors.Yellow,
	"zinc":    colors.Zinc,
}

// Colors registers color palettes for standard UI roles (primary, gray, danger, warning, success, info).
// Values can be map[string]any, map[string]colors.Palette, or ColorPalette struct for backwards compatibility.
func (p *Panel) Colors(palettes any) *Panel {
	if p.colors == nil {
		p.colors = make(map[string]colors.Palette)
	}
	switch v := palettes.(type) {
	case map[string]any:
		for role, val := range v {
			p.Color(role, val)
		}
	case map[string]colors.Palette:
		for role, pal := range v {
			p.Color(role, pal)
		}
	case ColorPalette:
		if v.Primary != "" {
			p.Color(colors.RolePrimary, string(v.Primary))
		}
		if v.Gray != "" {
			p.Color(colors.RoleGray, string(v.Gray))
		}
		if v.Success != "" {
			p.Color(colors.RoleSuccess, string(v.Success))
		}
		if v.Warning != "" {
			p.Color(colors.RoleWarning, string(v.Warning))
		}
		if v.Danger != "" {
			p.Color(colors.RoleDanger, string(v.Danger))
		}
		if v.Info != "" {
			p.Color(colors.RoleInfo, string(v.Info))
		}
	}
	return p
}

// Color sets a single color palette for a role (e.g. "primary", colors.Indigo or "#6366f1").
func (p *Panel) Color(role string, val any) *Panel {
	if p.colors == nil {
		p.colors = make(map[string]colors.Palette)
	}
	role = strings.ToLower(strings.TrimSpace(role))
	palette := resolvePalette(val)
	if palette != nil {
		p.colors[role] = palette
	}
	return p
}

func resolvePalette(val any) colors.Palette {
	switch v := val.(type) {
	case colors.Palette:
		return v
	case map[int]string:
		return colors.Palette(v)
	case map[string]string:
		pal := make(colors.Palette)
		for k, hex := range v {
			if shade, err := strconv.Atoi(k); err == nil {
				pal[shade] = hex
			}
		}
		return pal
	case string:
		str := strings.TrimSpace(v)
		if pal, ok := presetPalettes[strings.ToLower(str)]; ok {
			return pal
		}
		return colors.FromHex(str)
	default:
		return nil
	}
}

// Font sets the primary brand typography font family and optional Google Fonts URL.
// If no URL is provided, GoFilament generates the standard Google Fonts link for the family.
func (p *Panel) Font(family string, urls ...string) *Panel {
	p.fontFamily = strings.TrimSpace(family)
	if len(urls) > 0 && urls[0] != "" {
		p.fontURL = urls[0]
	} else if p.fontFamily != "" {
		p.fontURL = fmt.Sprintf("https://fonts.googleapis.com/css2?family=%s:wght@300;400;500;600;700&display=swap", url.QueryEscape(p.fontFamily))
	}
	return p
}

// Theme registers a primary custom CSS stylesheet for the panel.
func (p *Panel) Theme(stylesheetURL string) *Panel {
	return p.Stylesheet(stylesheetURL)
}

// Stylesheet appends one or more custom stylesheet links to the panel head.
func (p *Panel) Stylesheet(urls ...string) *Panel {
	for _, u := range urls {
		if u != "" {
			p.customStylesheets = append(p.customStylesheets, u)
		}
	}
	return p
}

// Script appends one or more custom JavaScript files to the panel head.
func (p *Panel) Script(urls ...string) *Panel {
	for _, u := range urls {
		if u != "" {
			p.customScripts = append(p.customScripts, u)
		}
	}
	return p
}

// RenderHook registers a callback that injects raw HTML into named template positions.
func (p *Panel) RenderHook(name string, hook func(ctx *Context) string) *Panel {
	if p.renderHooks == nil {
		p.renderHooks = make(map[string][]func(ctx *Context) string)
	}
	p.renderHooks[name] = append(p.renderHooks[name], hook)
	return p
}

// GetFontFamily returns the configured font family.
func (p *Panel) GetFontFamily() string {
	return p.fontFamily
}

// GetFontURL returns the configured font stylesheet URL.
func (p *Panel) GetFontURL() string {
	return p.fontURL
}

// GetColors returns the configured color palettes map.
func (p *Panel) GetColors() map[string]colors.Palette {
	return p.colors
}

// GetColor returns the palette for a specific role.
func (p *Panel) GetColor(role string) colors.Palette {
	if p.colors != nil {
		if pal, ok := p.colors[strings.ToLower(role)]; ok {
			return pal
		}
	}
	switch strings.ToLower(role) {
	case colors.RolePrimary:
		return colors.Emerald
	case colors.RoleGray:
		return colors.Slate
	case colors.RoleDanger:
		return colors.Red
	case colors.RoleWarning:
		return colors.Amber
	case colors.RoleSuccess:
		return colors.Emerald
	case colors.RoleInfo:
		return colors.Sky
	default:
		return nil
	}
}

// GetStylesheets returns custom stylesheets.
func (p *Panel) GetStylesheets() []string {
	return p.customStylesheets
}

// GetScripts returns custom scripts.
func (p *Panel) GetScripts() []string {
	return p.customScripts
}

func (p *Panel) renderHook(name string, ctx *Context) string {
	if p.renderHooks == nil {
		return ""
	}
	hooks, ok := p.renderHooks[name]
	if !ok || len(hooks) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, h := range hooks {
		if h != nil {
			sb.WriteString(h(ctx))
		}
	}
	return sb.String()
}

func (p *Panel) renderFontLinkHTML() string {
	if p.fontURL == "" {
		return ""
	}
	return fmt.Sprintf(`  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link rel="stylesheet" href="%s">`, html.EscapeString(p.fontURL))
}

// RenderThemeCSS generates the <style> block containing dynamic CSS variables and typography overrides.
func (p *Panel) RenderThemeCSS(ctx *Context) string {
	var vars strings.Builder

	if p.fontFamily != "" {
		vars.WriteString(fmt.Sprintf("    --font-sans: '%s', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;\n", html.EscapeString(p.fontFamily)))
	}

	for role, palette := range p.colors {
		if palette == nil {
			continue
		}
		for _, shade := range colors.Shades {
			if val, ok := palette[shade]; ok && val != "" {
				vars.WriteString(fmt.Sprintf("    --%s-%d: %s;\n", role, shade, val))
			}
		}
	}

	if vars.Len() == 0 {
		return ""
	}

	return fmt.Sprintf(`<style id="filament-theme-vars">
  :root {
%s  }
</style>`, vars.String())
}

// RenderHeadHTML generates all theme-related HTML elements for injection inside <head>.
func (p *Panel) RenderHeadHTML(ctx *Context) string {
	var sb strings.Builder

	// 1. PanelsRenderHookHeadStart
	if hook := p.renderHook(PanelsRenderHookHeadStart, ctx); hook != "" {
		sb.WriteString("  ")
		sb.WriteString(hook)
		sb.WriteString("\n")
	}

	// 2. Google Fonts
	if fontLink := p.renderFontLinkHTML(); fontLink != "" {
		sb.WriteString(fontLink)
		sb.WriteString("\n")
	}

	// 3. Dynamic Theme CSS Variables
	if themeCSS := p.RenderThemeCSS(ctx); themeCSS != "" {
		sb.WriteString("  ")
		sb.WriteString(themeCSS)
		sb.WriteString("\n")
	}

	// 4. Custom stylesheets
	for _, css := range p.customStylesheets {
		sb.WriteString(fmt.Sprintf("  <link rel=\"stylesheet\" href=\"%s\">\n", html.EscapeString(css)))
	}

	// 5. Custom scripts in head
	for _, js := range p.customScripts {
		sb.WriteString(fmt.Sprintf("  <script src=\"%s\"></script>\n", html.EscapeString(js)))
	}

	// 6. PanelsRenderHookHeadEnd
	if hook := p.renderHook(PanelsRenderHookHeadEnd, ctx); hook != "" {
		sb.WriteString("  ")
		sb.WriteString(hook)
		sb.WriteString("\n")
	}

	return strings.TrimRight(sb.String(), "\n")
}

// RenderBodyEndHTML generates elements for injection right before </body>.
func (p *Panel) RenderBodyEndHTML(ctx *Context) string {
	return p.renderHook(PanelsRenderHookBodyEnd, ctx)
}
