package filament

// NavigationItem represents a link in the panel's sidebar or topbar navigation.
type NavigationItem struct {
	Label      string
	URL        string
	Icon       string
	Badge      string
	BadgeColor string
	Group      string
	Sort       int
	Active     bool
	Visible    func(ctx *Context) bool
}

// UserMenuItem represents an action item inside the topbar user avatar dropdown.
type UserMenuItem struct {
	Label   string
	URL     string
	Icon    string
	Color   string
	OnClick string
}

// UserMenu builder allows configuring profile dropdown links.
type UserMenu struct {
	items []UserMenuItem
}

// NewUserMenu creates an empty UserMenu.
func NewUserMenu() *UserMenu {
	return &UserMenu{
		items: make([]UserMenuItem, 0),
	}
}

// Item appends a new user menu link.
func (m *UserMenu) Item(label, url string) *UserMenuItemBuilder {
	item := UserMenuItem{Label: label, URL: url}
	m.items = append(m.items, item)
	return &UserMenuItemBuilder{item: &m.items[len(m.items)-1]}
}

// Items returns all configured menu items.
func (m *UserMenu) Items() []UserMenuItem {
	return m.items
}

// UserMenuItemBuilder is a fluent modifier for UserMenuItem.
type UserMenuItemBuilder struct {
	item *UserMenuItem
}

// Icon sets the item SVG icon name.
func (b *UserMenuItemBuilder) Icon(icon string) *UserMenuItemBuilder {
	b.item.Icon = icon
	return b
}

// Color sets the item text/highlight color.
func (b *UserMenuItemBuilder) Color(color string) *UserMenuItemBuilder {
	b.item.Color = color
	return b
}
