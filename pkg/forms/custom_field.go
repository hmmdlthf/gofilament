package forms

import (
	"context"
	"fmt"
	"html"
)

// CustomField is a form field drawn by a component of the Flutter app
// (lib/components/fields/<component>_field.dart): a signature pad, a map
// picker, a barcode scanner. Its value is whatever JSON the component sets.
// The server still applies Required / RequiredOnCreate / Disabled / Visible
// and the Rule you give it. The web engine shows the value read-only.
type CustomField struct {
	BaseField
	component string
	props     any
	rule      func(ctx context.Context, value any) string
}

// Custom declares a field drawn by the Flutter field component named
// component.
func Custom(name, component string) *CustomField {
	return &CustomField{BaseField: BaseField{name: name}, component: component}
}

// Props sets values passed to the component unchanged (usually a struct or
// map). The generator writes a typed Dart props class from them.
func (f *CustomField) Props(p any) *CustomField { f.props = p; return f }

// Rule validates the value on the server: return a message to refuse it.
func (f *CustomField) Rule(fn func(ctx context.Context, value any) string) *CustomField {
	f.rule = fn
	return f
}

func (f *CustomField) Label(l string) *CustomField      { f.BaseField.Label(l); return f }
func (f *CustomField) HelperText(h string) *CustomField { f.BaseField.HelperText(h); return f }
func (f *CustomField) Default(v any) *CustomField       { f.BaseField.Default(v); return f }
func (f *CustomField) ColumnSpan(n int) *CustomField    { f.BaseField.ColumnSpan(n); return f }
func (f *CustomField) RequiredOnCreate() *CustomField   { f.BaseField.RequiredOnCreate(); return f }
func (f *CustomField) Required(r ...bool) *CustomField  { f.BaseField.Required(r...); return f }
func (f *CustomField) Disabled(d ...bool) *CustomField  { f.BaseField.Disabled(d...); return f }

// GetComponent returns the Flutter component name.
func (f *CustomField) GetComponent() string { return f.component }

// Render shows the current value read-only: the field is edited in the app.
func (f *CustomField) Render(ctx context.Context, state map[string]any, errors map[string]string) string {
	val := ""
	if v, ok := state[f.name]; ok && v != nil {
		val = fmt.Sprint(v)
	}
	return fmt.Sprintf(`<div class="filament-field"><label class="filament-label">%s</label><div class="filament-input" style="opacity:.7">%s</div><p class="filament-helper">Edited in the app.</p></div>`,
		html.EscapeString(f.displayName()), html.EscapeString(val))
}
