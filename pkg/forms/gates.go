package forms

import "context"

// A gate shows a component only when a server-side check on the request
// passes, for example a per-tenant feature (filament.FeatureEnabled). Unlike
// VisibleWhen, which the browser and the Flutter client evaluate as the
// user types, a gate is decided once per request on the server:
//
//   - Render leaves a gated-out component out of the HTML;
//   - Describe leaves it out of the JSON schema sent to the Flutter client;
//   - Validate drops its values, so a client cannot write a hidden field.
//
// A gate receives the request context. It must fail closed: return false
// when it cannot decide (the filament.Feature* helpers do).
type gated interface {
	featureGate() func(ctx context.Context) bool
}

// gateAllows reports whether c passes its gate for ctx. Components without
// a gate always pass; a gate with a nil ctx never does.
func gateAllows(ctx context.Context, c Component) bool {
	g, ok := c.(gated)
	if !ok {
		return true
	}
	fn := g.featureGate()
	if fn == nil {
		return true
	}
	return ctx != nil && fn(ctx)
}

// RequiresFeature shows the field only for requests where fn returns true,
// typically filament.FeatureEnabled(appfeatures.X). Hidden fields are not
// rendered, not described to the Flutter client, and their submitted values
// are dropped by Validate.
func (f *BaseField) RequiresFeature(fn func(ctx context.Context) bool) *BaseField {
	f.gate = fn
	return f
}

func (f *BaseField) featureGate() func(ctx context.Context) bool { return f.gate }

// RequiresFeature shows the text input only for requests where fn returns true.
func (f *TextInputField) RequiresFeature(fn func(ctx context.Context) bool) *TextInputField {
	f.BaseField.RequiresFeature(fn)
	return f
}

// RequiresFeature shows the textarea only for requests where fn returns true.
func (f *TextareaField) RequiresFeature(fn func(ctx context.Context) bool) *TextareaField {
	f.BaseField.RequiresFeature(fn)
	return f
}

// RequiresFeature shows the select only for requests where fn returns true.
func (f *SelectField) RequiresFeature(fn func(ctx context.Context) bool) *SelectField {
	f.BaseField.RequiresFeature(fn)
	return f
}

// RequiresFeature shows the toggle only for requests where fn returns true.
func (f *ToggleField) RequiresFeature(fn func(ctx context.Context) bool) *ToggleField {
	f.BaseField.RequiresFeature(fn)
	return f
}

// RequiresFeature shows the section (and every field in it) only for
// requests where fn returns true.
func (s *SectionComponent) RequiresFeature(fn func(ctx context.Context) bool) *SectionComponent {
	s.gate = fn
	return s
}

func (s *SectionComponent) featureGate() func(ctx context.Context) bool { return s.gate }

// RequiresFeature shows the grid (and every field in it) only for requests
// where fn returns true.
func (g *GridComponent) RequiresFeature(fn func(ctx context.Context) bool) *GridComponent {
	g.gate = fn
	return g
}

func (g *GridComponent) featureGate() func(ctx context.Context) bool { return g.gate }
