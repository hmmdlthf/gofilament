package forms

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Form manages form state, validation error bags, and reactive component trees.
type Form struct {
	schema    []Component
	state     map[string]any
	errors    map[string]string
	action    string
	method    string
	multipart bool
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

// Multipart forces the form to render with enctype="multipart/form-data".
func (f *Form) Multipart(enabled ...bool) *Form {
	if len(enabled) > 0 {
		f.multipart = enabled[0]
	} else {
		f.multipart = true
	}
	return f
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
		// Search in fieldsets
		if fs, ok := c.(*FieldsetComponent); ok {
			if match := findFieldInComponents(fs.schema, name); match != nil {
				return match
			}
		}
		// Search in splits
		if sp, ok := c.(*SplitComponent); ok {
			if match := findFieldInComponents(sp.schema, name); match != nil {
				return match
			}
		}
		// Search in wizards
		if w, ok := c.(*WizardComponent); ok {
			for _, step := range w.steps {
				if match := findFieldInComponents(step.schema, name); match != nil {
					return match
				}
			}
		}
		// Search in repeaters
		if r, ok := c.(*RepeaterComponent); ok {
			if match := findFieldInComponents(r.schema, name); match != nil {
				return match
			}
		}
	}
	return nil
}

func hasUploadsInComponents(components []Component) bool {
	for _, c := range components {
		if _, ok := c.(*FileUploadField); ok {
			return true
		}
		if s, ok := c.(*SectionComponent); ok {
			if hasUploadsInComponents(s.schema) {
				return true
			}
		}
		if g, ok := c.(*GridComponent); ok {
			if hasUploadsInComponents(g.schema) {
				return true
			}
		}
		if fs, ok := c.(*FieldsetComponent); ok {
			if hasUploadsInComponents(fs.schema) {
				return true
			}
		}
		if sp, ok := c.(*SplitComponent); ok {
			if hasUploadsInComponents(sp.schema) {
				return true
			}
		}
		if w, ok := c.(*WizardComponent); ok {
			for _, step := range w.steps {
				if hasUploadsInComponents(step.schema) {
					return true
				}
			}
		}
		if r, ok := c.(*RepeaterComponent); ok {
			if hasUploadsInComponents(r.schema) {
				return true
			}
		}
	}
	return false
}

// HasUploads returns whether any field in the form is a FileUploadField.
func (f *Form) HasUploads() bool {
	return f.multipart || hasUploadsInComponents(f.schema)
}

// Render compiles the form into HTML with current state and validation errors.
func (f *Form) Render(ctx context.Context) string {
	var sb strings.Builder
	enctypeAttr := ""
	if f.HasUploads() {
		enctypeAttr = ` enctype="multipart/form-data"`
	}

	sb.WriteString(fmt.Sprintf(`<form id="filament-form" method="%s" action="%s"%s>`, f.method, f.action, enctypeAttr))

	for _, comp := range f.schema {
		sb.WriteString(comp.Render(ctx, f.state, f.errors))
	}

	sb.WriteString(`</form>`)
	return sb.String()
}

var (
	repeaterKeyRegex = regexp.MustCompile(`^([a-zA-Z0-9_\-]+)\[([0-9]+)\]\[([a-zA-Z0-9_\-]+)\]$`)
	arrayKeyRegex    = regexp.MustCompile(`^([a-zA-Z0-9_\-]+)\[\]$`)
)

// ParseFormData parses URL PostForm values, converting bracket notation
// (e.g. items[0][name], items[0][price]) into structured []map[string]any slices,
// and handling array values (e.g. tags[]).
func ParseFormData(values map[string][]string) map[string]any {
	result := make(map[string]any)

	// Temporary structure for repeaters: map[parentKey]map[index]map[field]any
	repeaters := make(map[string]map[int]map[string]any)

	for key, valSlice := range values {
		// Ignore internal csrf or button tokens if needed, but keep all standard fields
		if m := repeaterKeyRegex.FindStringSubmatch(key); len(m) == 4 {
			parent := m[1]
			idx, _ := strconv.Atoi(m[2])
			subfield := m[3]

			if _, ok := repeaters[parent]; !ok {
				repeaters[parent] = make(map[int]map[string]any)
			}
			if _, ok := repeaters[parent][idx]; !ok {
				repeaters[parent][idx] = make(map[string]any)
			}

			if len(valSlice) == 1 {
				repeaters[parent][idx][subfield] = valSlice[0]
			} else {
				repeaters[parent][idx][subfield] = valSlice
			}
			continue
		}

		if m := arrayKeyRegex.FindStringSubmatch(key); len(m) == 2 {
			baseName := m[1]
			result[baseName] = valSlice
			continue
		}

		// Standard single or multiple value
		if len(valSlice) == 1 {
			result[key] = valSlice[0]
		} else {
			result[key] = valSlice
		}
	}

	// Collapse repeaters into ordered slices of maps
	for parent, indexMap := range repeaters {
		indices := make([]int, 0, len(indexMap))
		for idx := range indexMap {
			indices = append(indices, idx)
		}
		sort.Ints(indices)

		items := make([]map[string]any, 0, len(indices))
		for _, idx := range indices {
			items = append(items, indexMap[idx])
		}
		result[parent] = items
	}

	return result
}
