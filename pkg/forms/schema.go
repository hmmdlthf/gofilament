package forms

import (
	"context"
	"sort"
	"strings"
)

// Option is one choice of a select, radio or checkbox list.
type Option struct {
	Value string `json:"value"`
	Label string `json:"label"`
	// Description is optional secondary text (shown under the label where supported).
	Description string `json:"description,omitempty"`
}

// OptionsFunc loads the options of a field for the current request, e.g. the
// categories of the active tenant.
type OptionsFunc func(ctx context.Context) []Option

// OptionSearchFunc returns the options matching a search text. Use it for
// large lists (products, customers) that should not be sent in full.
type OptionSearchFunc func(ctx context.Context, query string, limit int) []Option

// OptionLabelFunc resolves the label of an already selected value of a
// searchable remote select.
type OptionLabelFunc func(ctx context.Context, value string) string

// Node is the JSON description of one form component. Clients that render
// forms themselves (the Flutter engine, mobile apps, headless integrations)
// build their widgets from a tree of nodes.
type Node struct {
	// Kind is "field" for inputs or the container kind: section, grid, tabs,
	// tab, fieldset, split, wizard, step, repeater.
	Kind string `json:"kind"`
	// Type is the input type of a field: text, email, password, number, tel,
	// url, textarea, select, toggle, date, datetime, time, file, image, tags,
	// color, hidden, radio, checkbox_list, rich_text, markdown, key_value, code.
	Type string `json:"type,omitempty"`

	Name             string `json:"name,omitempty"`
	Label            string `json:"label,omitempty"`
	Placeholder      string `json:"placeholder,omitempty"`
	HelperText       string `json:"helper_text,omitempty"`
	Default          any    `json:"default,omitempty"`
	Required         bool   `json:"required,omitempty"`
	RequiredOnCreate bool   `json:"required_on_create,omitempty"`
	Disabled         bool   `json:"disabled,omitempty"`
	ColumnSpan       int    `json:"column_span,omitempty"`
	Live             bool   `json:"live,omitempty"`

	// text
	MinLength int    `json:"min_length,omitempty"`
	MaxLength int    `json:"max_length,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
	Suffix    string `json:"suffix,omitempty"`
	Rows      int    `json:"rows,omitempty"`

	// choices
	Options    []Option `json:"options,omitempty"`
	Searchable bool     `json:"searchable,omitempty"`
	Multiple   bool     `json:"multiple,omitempty"`
	// Remote is true when options must be fetched from the options endpoint
	// with a search text instead of being listed in Options.
	Remote bool `json:"remote,omitempty"`

	// files
	Accept   []string `json:"accept,omitempty"`
	MaxFiles int      `json:"max_files,omitempty"`
	MaxSize  int64    `json:"max_size,omitempty"`

	// misc field settings
	Language   string   `json:"language,omitempty"`
	KeyLabel   string   `json:"key_label,omitempty"`
	ValueLabel string   `json:"value_label,omitempty"`
	Presets    []string `json:"presets,omitempty"`

	// containers
	Heading     string `json:"heading,omitempty"`
	Description string `json:"description,omitempty"`
	Columns     int    `json:"columns,omitempty"`
	Collapsible bool   `json:"collapsible,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Badge       string `json:"badge,omitempty"`
	MinItems    int    `json:"min_items,omitempty"`
	MaxItems    int    `json:"max_items,omitempty"`
	Children    []Node `json:"children,omitempty"`
}

// Describe converts a component tree into its JSON description. Dynamic
// options (OptionsFunc) are resolved with ctx, so they reflect the request.
func Describe(ctx context.Context, components []Component) []Node {
	nodes := make([]Node, 0, len(components))
	for _, c := range components {
		if n, ok := describe(ctx, c); ok {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func baseNode(f *BaseField, typ string) Node {
	return Node{
		Kind:             "field",
		Type:             typ,
		Name:             f.name,
		Label:            f.formatLabel(),
		Placeholder:      f.placeholder,
		HelperText:       f.helperText,
		Default:          f.defaultValue,
		Required:         f.required,
		RequiredOnCreate: f.requiredOnCreate,
		Disabled:         f.disabled,
		ColumnSpan:       f.columnSpan,
		Live:             f.live,
	}
}

func sortedOptions(options map[string]string, order []string, descriptions map[string]string) []Option {
	out := make([]Option, 0, len(options))
	for k, v := range options {
		out = append(out, Option{Value: k, Label: v, Description: descriptions[k]})
	}
	// Options built from a Go map have no stable order: sort by label so every
	// request and every client shows the same list.
	sort.SliceStable(out, func(i, j int) bool {
		li, lj := strings.ToLower(out[i].Label), strings.ToLower(out[j].Label)
		if li != lj {
			return li < lj
		}
		return out[i].Value < out[j].Value
	})
	_ = order
	return out
}

func describe(ctx context.Context, c Component) (Node, bool) {
	switch f := c.(type) {
	case *TextInputField:
		typ := f.inputType
		if typ == "" {
			typ = "text"
		}
		n := baseNode(&f.BaseField, typ)
		n.MinLength, n.MaxLength, n.Prefix, n.Suffix = f.minLength, f.maxLength, f.prefix, f.suffix
		return n, true
	case *TextareaField:
		n := baseNode(&f.BaseField, "textarea")
		n.Rows = f.rows
		return n, true
	case *SelectField:
		n := baseNode(&f.BaseField, "select")
		n.Searchable, n.Multiple = f.searchable, f.multiple
		if f.searchFunc != nil {
			n.Remote, n.Searchable = true, true
		} else {
			n.Options = f.ResolveOptions(ctx)
		}
		return n, true
	case *ToggleField:
		return baseNode(&f.BaseField, "toggle"), true
	case *DatePickerField:
		if f.includeTime {
			return baseNode(&f.BaseField, "datetime"), true
		}
		return baseNode(&f.BaseField, "date"), true
	case *FileUploadField:
		typ := "file"
		if f.isImage || f.isAvatar {
			typ = "image"
		}
		n := baseNode(&f.BaseField, typ)
		n.Accept, n.MaxFiles, n.MaxSize, n.Multiple = f.accepted, f.maxFiles, f.maxSize, f.multiple
		return n, true
	case *TagsInputField:
		n := baseNode(&f.BaseField, "tags")
		for _, s := range f.suggestions {
			n.Options = append(n.Options, Option{Value: s, Label: s})
		}
		n.MaxItems = f.maxTags
		return n, true
	case *ColorPickerField:
		n := baseNode(&f.BaseField, "color")
		n.Presets = f.presets
		return n, true
	case *HiddenField:
		return baseNode(&f.BaseField, "hidden"), true
	case *RadioField:
		n := baseNode(&f.BaseField, "radio")
		n.Options = sortedOptions(f.options, f.order, f.descriptions)
		return n, true
	case *CheckboxListField:
		n := baseNode(&f.BaseField, "checkbox_list")
		n.Options = sortedOptions(f.options, f.order, nil)
		n.Multiple = true
		return n, true
	case *TimePickerField:
		return baseNode(&f.BaseField, "time"), true
	case *RichEditorField:
		return baseNode(&f.BaseField, "rich_text"), true
	case *MarkdownEditorField:
		n := baseNode(&f.BaseField, "markdown")
		n.Rows = f.rows
		return n, true
	case *KeyValueField:
		n := baseNode(&f.BaseField, "key_value")
		n.KeyLabel, n.ValueLabel = f.keyLabel, f.valueLabel
		return n, true
	case *CodeEditorField:
		n := baseNode(&f.BaseField, "code")
		n.Language = f.language
		return n, true

	case *SectionComponent:
		return Node{Kind: "section", Heading: f.heading, Description: f.description,
			Columns: f.columns, Collapsible: f.collapsible, Children: Describe(ctx, f.schema)}, true
	case *GridComponent:
		return Node{Kind: "grid", Columns: f.columns, Children: Describe(ctx, f.schema)}, true
	case *FieldsetComponent:
		return Node{Kind: "fieldset", Heading: f.legend, Children: Describe(ctx, f.schema)}, true
	case *SplitComponent:
		return Node{Kind: "split", Children: Describe(ctx, f.schema)}, true
	case *TabsComponent:
		n := Node{Kind: "tabs"}
		for _, t := range f.tabs {
			label := t.label
			if label == "" {
				label = t.name
			}
			n.Children = append(n.Children, Node{Kind: "tab", Name: t.name, Label: label,
				Badge: t.badge, Children: Describe(ctx, t.schema)})
		}
		return n, true
	case *WizardComponent:
		n := Node{Kind: "wizard"}
		for _, s := range f.steps {
			n.Children = append(n.Children, Node{Kind: "step", Label: s.title, Description: s.description,
				Icon: s.icon, Badge: s.badge, Children: Describe(ctx, s.schema)})
		}
		return n, true
	case *RepeaterComponent:
		n := baseNode(&f.BaseField, "")
		n.Kind = "repeater"
		n.MinItems, n.MaxItems, n.Columns, n.Collapsible = f.minItems, f.maxItems, f.gridColumns, f.collapsible
		n.Children = Describe(ctx, f.schema)
		return n, true
	}
	return Node{}, false
}

// FindField returns the field named name anywhere in the component tree.
func FindField(components []Component, name string) Component {
	for _, c := range components {
		var children []Component
		switch f := c.(type) {
		case *SectionComponent:
			children = f.schema
		case *GridComponent:
			children = f.schema
		case *FieldsetComponent:
			children = f.schema
		case *SplitComponent:
			children = f.schema
		case *TabsComponent:
			for _, t := range f.tabs {
				if found := FindField(t.schema, name); found != nil {
					return found
				}
			}
			continue
		case *WizardComponent:
			for _, s := range f.steps {
				if found := FindField(s.schema, name); found != nil {
					return found
				}
			}
			continue
		case *RepeaterComponent:
			if f.name == name {
				return c
			}
			children = f.schema
		default:
			if c.GetName() == name {
				return c
			}
			continue
		}
		if found := FindField(children, name); found != nil {
			return found
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Select: ordered and dynamic options
// ---------------------------------------------------------------------------

// OptionList sets the options in the given order (Options takes a map, whose
// order is not defined).
func (f *SelectField) OptionList(opts ...Option) *SelectField {
	f.options = make(map[string]string, len(opts))
	f.order = make([]string, 0, len(opts))
	for _, o := range opts {
		f.options[o.Value] = o.Label
		f.order = append(f.order, o.Value)
	}
	f.ordered = true
	return f
}

// OptionsUsing loads the options on every request, e.g. from the database for
// the active tenant. Suitable for short lists (up to a few hundred rows).
func (f *SelectField) OptionsUsing(fn OptionsFunc) *SelectField {
	f.optionsFunc = fn
	return f
}

// SearchUsing makes the select fetch matching options as the user types.
// label resolves the text of the current value when editing a record.
func (f *SelectField) SearchUsing(search OptionSearchFunc, label OptionLabelFunc) *SelectField {
	f.searchFunc = search
	f.labelFunc = label
	f.searchable = true
	return f
}

// ResolveOptions returns the options of the select for this request.
func (f *SelectField) ResolveOptions(ctx context.Context) []Option {
	if f.optionsFunc != nil {
		return f.optionsFunc(ctx)
	}
	if f.ordered {
		out := make([]Option, 0, len(f.order))
		for _, k := range f.order {
			out = append(out, Option{Value: k, Label: f.options[k]})
		}
		return out
	}
	return sortedOptions(f.options, f.order, nil)
}

// SearchOptions runs the remote search of the select (nil when not remote).
func (f *SelectField) SearchOptions(ctx context.Context, query string, limit int) []Option {
	if f.searchFunc != nil {
		return f.searchFunc(ctx, query, limit)
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var out []Option
	for _, o := range f.ResolveOptions(ctx) {
		if q == "" || strings.Contains(strings.ToLower(o.Label), q) {
			out = append(out, o)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out
}

// OptionLabel resolves the label of value (for remote selects showing a saved value).
func (f *SelectField) OptionLabel(ctx context.Context, value string) string {
	if f.labelFunc != nil {
		return f.labelFunc(ctx, value)
	}
	for _, o := range f.ResolveOptions(ctx) {
		if o.Value == value {
			return o.Label
		}
	}
	return ""
}
