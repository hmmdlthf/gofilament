package filament

// SidebarMode is how the desktop sidebar is shown.
type SidebarMode string

const (
	// SidebarExpanded shows icons and labels (the default).
	SidebarExpanded SidebarMode = "expanded"
	// SidebarIcons shows a narrow rail of icons; labels appear as tooltips.
	SidebarIcons SidebarMode = "icons"
	// SidebarHidden hides the sidebar; the top bar's menu button opens it.
	SidebarHidden SidebarMode = "hidden"
)

// SidebarCollapseModes sets which collapsed forms users can switch the
// desktop sidebar to (default: both SidebarIcons and SidebarHidden). The
// expanded sidebar is always available. Needs SidebarCollapsibleOnDesktop
// (on by default).
func (p *Panel) SidebarCollapseModes(modes ...SidebarMode) *Panel {
	p.sidebarModes = nil
	for _, m := range modes {
		if m == SidebarIcons || m == SidebarHidden {
			p.sidebarModes = append(p.sidebarModes, m)
		}
	}
	return p
}

// SidebarDefaultMode sets how the desktop sidebar starts before a user picks
// a mode (each device then remembers the user's choice).
func (p *Panel) SidebarDefaultMode(mode SidebarMode) *Panel {
	p.sidebarDefault = mode
	return p
}

// SidebarModes returns the modes the desktop sidebar can be switched
// between, expanded first.
func (p *Panel) SidebarModes() []SidebarMode {
	modes := []SidebarMode{SidebarExpanded}
	if !p.sidebarCollapsible {
		return modes
	}
	if p.sidebarModes == nil {
		return append(modes, SidebarIcons, SidebarHidden)
	}
	return append(modes, p.sidebarModes...)
}

// SidebarStartMode returns the mode the desktop sidebar starts in.
func (p *Panel) SidebarStartMode() SidebarMode {
	for _, m := range p.SidebarModes() {
		if m == p.sidebarDefault {
			return m
		}
	}
	return SidebarExpanded
}

func (p *Panel) sidebarDTO() map[string]any {
	modes := []string{}
	for _, m := range p.SidebarModes() {
		modes = append(modes, string(m))
	}
	return map[string]any{
		"collapsible": len(modes) > 1,
		"modes":       modes,
		"default":     string(p.SidebarStartMode()),
	}
}
