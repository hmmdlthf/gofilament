package forms

import (
	"fmt"
	"strings"

	"context"
)

// Form manages form state, validation error bags, and reactive component trees.
type Form struct {
	schema []Component
	state  map[string]any
	errors map[string]string
	action string
	method string
}

// New creates an empty Form instance.
func New() *Form {
	return &Form{
		schema: make([]Component, 0),
		state:  make(map[string]any),
		errors: make(map[string]string),
		method: "POST",
	}
}

// Schema sets top-level form components.
func (f *Form) Schema(components ...Component) *Form {
	f.schema = components
	return f
}

// GetSchema returns the configured components.
func (f *Form) GetSchema() []Component {
	return f.schema
}

// Fill pre-populates form state from a map or model attributes.
func (f *Form) Fill(data map[string]any) *Form {
	for k, v := range data {
		f.state[k] = v
	}
	return f
}

// SetState updates a specific key in the form state.
func (f *Form) SetState(key string, val any) *Form {
	f.state[key] = val
	return f
}

// GetState reads a specific key from form state.
func (f *Form) GetState(key string) any {
	return f.state[key]
}

// State returns the complete current state map.
func (f *Form) State() map[string]any {
	return f.state
}

// SetErrors sets validation error messages.
func (f *Form) SetErrors(errs map[string]string) *Form {
	f.errors = errs
	return f
}

// AddError appends an error for a given field.
func (f *Form) AddError(key, message string) *Form {
	f.errors[key] = message
	return f
}

// Errors returns all current validation errors.
func (f *Form) Errors() map[string]string {
	return f.errors
}

// Action sets the form submit action URL.
func (f *Form) Action(url string) *Form {
	f.action = url
	return f
}

// Method sets the HTTP submit method (POST, PUT, PATCH).
func (f *Form) Method(m string) *Form {
	f.method = m
	return f
}

// GetField finds a component by its field name recursively.
func (f *Form) GetField(name string) Component {
	return findFieldInComponents(f.schema, name)
}

func findFieldInComponents(components []Component, name string) Component {
	for _, c := range components {
		if c.GetName() == name {
			return c
		}
		// Search in sections
		if s, ok := c.(*SectionComponent); ok {
			if match := findFieldInComponents(s.schema, name); match != nil {
				return match
			}
		}
		// Search in grids
		if g, ok := c.(*GridComponent); ok {
			if match := findFieldInComponents(g.schema, name); match != nil {
				return match
			}
		}
	}
	return nil
}

// Render compiles the form into HTML with current state and validation errors.
func (f *Form) Render(ctx context.Context) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<form id="filament-form" method="%s" action="%s">`, f.method, f.action))

	for _, comp := range f.schema {
		sb.WriteString(comp.Render(ctx, f.state, f.errors))
	}

	sb.WriteString(`</form>`)
	return sb.String()
}
