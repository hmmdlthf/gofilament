package colors

import (
	"fmt"
	"strings"
)

// Standard color role names
const (
	RolePrimary = "primary"
	RoleGray    = "gray"
	RoleDanger  = "danger"
	RoleWarning = "warning"
	RoleSuccess = "success"
	RoleInfo    = "info"
)

// Shades lists the standard 11 Tailwind/Filament color shade steps.
var Shades = []int{50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950}

// Palette represents an 11-step color palette mapping shade levels (50..950) to hex codes.
type Palette map[int]string

// Shade returns the hex color for the specified shade, or empty string if not found.
func (p Palette) Shade(shade int) string {
	if p == nil {
		return ""
	}
	return p[shade]
}

// Get is an alias for Shade.
func (p Palette) Get(shade int) string {
	return p.Shade(shade)
}

// Hex returns the primary mid-tone shade (defaults to 500 or 600).
func (p Palette) Hex() string {
	if p == nil {
		return ""
	}
	if v, ok := p[500]; ok && v != "" {
		return v
	}
	if v, ok := p[600]; ok && v != "" {
		return v
	}
	for _, s := range Shades {
		if v, ok := p[s]; ok && v != "" {
			return v
		}
	}
	return ""
}

// CSSVariables formats the palette as CSS variable declarations with the given prefix.
// E.g. p.CSSVariables("primary") generates:
//
//	--primary-50: #...;
//	--primary-100: #...;
func (p Palette) CSSVariables(prefix string) string {
	if p == nil || prefix == "" {
		return ""
	}
	var sb strings.Builder
	for _, shade := range Shades {
		if val, ok := p[shade]; ok && val != "" {
			sb.WriteString(fmt.Sprintf("--%s-%d: %s;\n", prefix, shade, val))
		}
	}
	return sb.String()
}

// Clone creates a shallow copy of the palette.
func (p Palette) Clone() Palette {
	if p == nil {
		return nil
	}
	cp := make(Palette, len(p))
	for k, v := range p {
		cp[k] = v
	}
	return cp
}
