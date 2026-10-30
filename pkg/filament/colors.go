package filament

// Color represents a named or hex color.
type Color string

const (
	ColorEmerald Color = "emerald"
	ColorIndigo  Color = "indigo"
	ColorBlue    Color = "blue"
	ColorAmber   Color = "amber"
	ColorRose    Color = "rose"
	ColorSlate   Color = "slate"
	ColorGray    Color = "gray"
	ColorZinc    Color = "zinc"
)

// ColorPalette defines the active color themes for the panel.
type ColorPalette struct {
	Primary Color
	Gray    Color
	Success Color
	Warning Color
	Danger  Color
	Info    Color
}

// DefaultColorPalette returns the standard GoFilament emerald & slate palette.
func DefaultColorPalette() ColorPalette {
	return ColorPalette{
		Primary: ColorEmerald,
		Gray:    ColorSlate,
		Success: ColorEmerald,
		Warning: ColorAmber,
		Danger:  ColorRose,
		Info:    ColorBlue,
	}
}
