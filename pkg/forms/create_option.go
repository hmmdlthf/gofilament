package forms

import "context"

// Creating a missing option from a select field (the Flutter admin shows a
// "Create" entry in the field's picker), so users never leave the form they
// are filling in to add, say, a new category.
//
//	forms.Select("category_id").OptionsUsing(categoryOptions).
//	    CreateOptionFrom("categories") // the Categories resource's own form
//
//	forms.Select("tag").OptionsUsing(tagOptions).
//	    CreateOptionForm(forms.TextInput("name").Required()).
//	    CreateOptionUsing(func(ctx context.Context, data map[string]any) (forms.Option, error) {
//	        id, err := tags.Create(ctx, data["name"].(string))
//	        return forms.Option{Value: id, Label: data["name"].(string)}, err
//	    })

// CreateOptionFunc creates an option from the create form's validated data
// and returns it (its value is then selected).
type CreateOptionFunc func(ctx context.Context, data map[string]any) (Option, error)

// CreateOption is the schema's description of a creatable select.
type CreateOption struct {
	// Label is the create entry's text, e.g. "Create category".
	Label string `json:"label"`
	// Resource is the slug of the resource whose form creates the option
	// (CreateOptionFrom); empty for a custom form.
	Resource string `json:"resource,omitempty"`
}

type createOption struct {
	label      string
	resource   string
	components []Component
	create     CreateOptionFunc
}

// CreateOptionFrom lets users create a missing option with the form of the
// resource with this slug. That resource's validation, create handler and
// create permission apply; the new record's id becomes the selected value.
// Its label comes from the field's options (OptionsUsing / OptionLabel).
func (f *SelectField) CreateOptionFrom(resourceSlug string) *SelectField {
	f.creatable().resource = resourceSlug
	return f
}

// CreateOptionForm sets the fields of a custom create form (with
// CreateOptionUsing).
func (f *SelectField) CreateOptionForm(components ...Component) *SelectField {
	f.creatable().components = components
	return f
}

// CreateOptionUsing creates the option from the custom form's data.
func (f *SelectField) CreateOptionUsing(fn CreateOptionFunc) *SelectField {
	f.creatable().create = fn
	return f
}

// CreateOptionLabel sets the create entry's text (default "Create <label>").
func (f *SelectField) CreateOptionLabel(label string) *SelectField {
	f.creatable().label = label
	return f
}

func (f *SelectField) creatable() *createOption {
	if f.createOpt == nil {
		f.createOpt = &createOption{}
	}
	return f.createOpt
}

// CanCreateOption reports whether options can be created from the field.
func (f *SelectField) CanCreateOption() bool {
	c := f.createOpt
	return c != nil && (c.resource != "" || (c.create != nil && len(c.components) > 0))
}

// CreateOptionResource is the resource slug of CreateOptionFrom ("" for a
// custom form).
func (f *SelectField) CreateOptionResource() string {
	if f.createOpt == nil {
		return ""
	}
	return f.createOpt.resource
}

// CreateOptionComponents are the custom create form's fields.
func (f *SelectField) CreateOptionComponents() []Component {
	if f.createOpt == nil {
		return nil
	}
	return f.createOpt.components
}

// CreateOptionHandler is the custom create function (nil with
// CreateOptionFrom).
func (f *SelectField) CreateOptionHandler() CreateOptionFunc {
	if f.createOpt == nil {
		return nil
	}
	return f.createOpt.create
}

func (f *SelectField) describeCreateOption() *CreateOption {
	if !f.CanCreateOption() {
		return nil
	}
	label := f.createOpt.label
	if label == "" {
		label = "Create " + lowerFirst(f.formatLabel())
	}
	return &CreateOption{Label: label, Resource: f.createOpt.resource}
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if len(r) > 1 && r[1] >= 'A' && r[1] <= 'Z' {
		return s // an acronym such as "SKU"
	}
	if r[0] >= 'A' && r[0] <= 'Z' {
		r[0] += 'a' - 'A'
	}
	return string(r)
}
