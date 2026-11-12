package forms

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// ValidationError lists the fields of a submission that broke a form rule,
// keyed by field name ("lines.0.sku" inside repeaters). The panel API answers
// it with 422 and {"error": message, "errors": {field: message}}.
type ValidationError struct {
	Errors map[string]string
}

func (e *ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "The given data was invalid."
	}
	keys := make([]string, 0, len(e.Errors))
	for k := range e.Errors {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	msg := e.Errors[keys[0]]
	if len(keys) > 1 {
		msg += fmt.Sprintf(" (and %d more)", len(keys)-1)
	}
	return msg
}

// StatusCode makes the panel API answer 422.
func (e *ValidationError) StatusCode() int { return http.StatusUnprocessableEntity }

// basic exposes the shared field settings of every field type.
func (f *BaseField) basic() *BaseField { return f }

type basicField interface{ basic() *BaseField }

func (f *BaseField) displayName() string {
	if l := f.formatLabel(); l != "" {
		return l
	}
	return f.name
}

// Validate checks a submission against a form on the server and returns only
// the values the form accepts:
//
//   - keys that are not fields of the form, disabled fields, fields hidden by
//     Visible and fields whose Dehydrated callback returns false are dropped;
//   - required fields must be present and non-empty when creating
//     (Required and RequiredOnCreate). When updating, absent keys are left
//     alone (a partial update) but a Required field may not be emptied;
//   - text length, numeric / email / url inputs, select / radio / checkbox
//     options (resolved with ctx), dates, tag limits and repeater item counts
//     are checked, repeater items recursively.
//
// Fields of a custom Component type are kept by name without rules.
func Validate(ctx context.Context, components []Component, data map[string]any, creating bool) (map[string]any, error) {
	v := validator{ctx: ctx, creating: creating, errors: map[string]string{}}
	out := map[string]any{}
	v.walk(components, data, out, "")
	if len(v.errors) > 0 {
		return out, &ValidationError{Errors: v.errors}
	}
	return out, nil
}

type validator struct {
	ctx      context.Context
	creating bool
	errors   map[string]string
}

func (v *validator) fail(key, msg string) {
	if _, ok := v.errors[key]; !ok {
		v.errors[key] = msg
	}
}

func (v *validator) walk(components []Component, data, out map[string]any, prefix string) {
	for _, c := range components {
		switch f := c.(type) {
		case *SectionComponent:
			v.walk(f.schema, data, out, prefix)
		case *GridComponent:
			v.walk(f.schema, data, out, prefix)
		case *FieldsetComponent:
			v.walk(f.schema, data, out, prefix)
		case *SplitComponent:
			v.walk(f.schema, data, out, prefix)
		case *TabsComponent:
			for _, t := range f.tabs {
				v.walk(t.schema, data, out, prefix)
			}
		case *WizardComponent:
			for _, s := range f.steps {
				v.walk(s.schema, data, out, prefix)
			}
		case *RepeaterComponent:
			v.repeater(f, data, out, prefix)
		default:
			v.field(c, data, out, prefix)
		}
	}
}

func (v *validator) skip(b *BaseField, data map[string]any) bool {
	if b.disabled {
		return true
	}
	if b.visibleFunc != nil && !b.visibleFunc(func(key string) any { return data[key] }) {
		return true
	}
	return false
}

func (v *validator) required(b *BaseField) bool {
	return b.required || (v.creating && b.requiredOnCreate)
}

func isEmpty(val any) bool {
	switch x := val.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(x) == ""
	case []any:
		return len(x) == 0
	case []string:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}

func (v *validator) field(c Component, data, out map[string]any, prefix string) {
	name := c.GetName()
	if name == "" {
		return
	}
	bf, ok := c.(basicField)
	if !ok {
		// a custom component: keep its value, no rules
		if val, present := data[name]; present {
			out[name] = val
		}
		return
	}
	b := bf.basic()
	if v.skip(b, data) {
		return
	}
	key := prefix + name
	val, present := data[name]
	if !present || isEmpty(val) {
		if v.required(b) && (v.creating || present) {
			v.fail(key, b.displayName()+" is required.")
			return
		}
		if present {
			out[name] = val
		}
		return
	}
	if msg := v.check(c, b, val); msg != "" {
		v.fail(key, msg)
		return
	}
	if b.dehydrateFunc != nil && !b.dehydrateFunc(val) {
		return
	}
	out[name] = val
}

func toStrings(val any) ([]string, bool) {
	switch x := val.(type) {
	case []string:
		return x, true
	case []any:
		out := make([]string, len(x))
		for i, e := range x {
			out[i] = scalarString(e)
		}
		return out, true
	case string:
		if x == "" {
			return nil, true
		}
		return []string{x}, true
	}
	return nil, false
}

func scalarString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}

func isNumber(val any) bool {
	switch x := val.(type) {
	case float64, float32, int, int64, int32, uint, uint64, uint32:
		return true
	case string:
		_, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return err == nil
	}
	if n, ok := val.(interface{ Float64() (float64, error) }); ok {
		_, err := n.Float64()
		return err == nil
	}
	return false
}

func optionSet(opts map[string]string) func(string) bool {
	return func(s string) bool { _, ok := opts[s]; return ok }
}

func (v *validator) check(c Component, b *BaseField, val any) string {
	label := b.displayName()
	switch f := c.(type) {
	case *TextInputField:
		if f.inputType == "number" {
			if !isNumber(val) {
				return label + " must be a number."
			}
			return ""
		}
		s, ok := val.(string)
		if !ok {
			if isNumber(val) {
				s = scalarString(val)
			} else {
				return label + " must be text."
			}
		}
		n := utf8.RuneCountInString(s)
		if f.minLength > 0 && n < f.minLength {
			return fmt.Sprintf("%s must be at least %d characters.", label, f.minLength)
		}
		if f.maxLength > 0 && n > f.maxLength {
			return fmt.Sprintf("%s may not be longer than %d characters.", label, f.maxLength)
		}
		switch f.inputType {
		case "email":
			if a, err := mail.ParseAddress(s); err != nil || a.Address != s {
				return label + " must be a valid email address."
			}
		case "url":
			u, err := url.ParseRequestURI(s)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return label + " must be a valid URL."
			}
		}
	case *SelectField:
		if f.searchFunc != nil {
			return "" // remote options: the label loader decides what exists
		}
		allowed := map[string]bool{}
		for _, o := range f.ResolveOptions(v.ctx) {
			allowed[o.Value] = true
		}
		return checkChoices(label, val, f.multiple, func(s string) bool { return allowed[s] })
	case *RadioField:
		return checkChoices(label, val, false, optionSet(f.options))
	case *CheckboxListField:
		return checkChoices(label, val, true, optionSet(f.options))
	case *ToggleField:
		switch x := val.(type) {
		case bool:
		case string:
			switch strings.ToLower(x) {
			case "1", "0", "true", "false", "on", "off", "yes", "no":
			default:
				return label + " must be true or false."
			}
		default:
			if !isNumber(val) {
				return label + " must be true or false."
			}
		}
	case *DatePickerField:
		s, ok := val.(string)
		if !ok || !parsesAsTime(s) {
			return label + " must be a valid date."
		}
	case *TimePickerField:
		s, ok := val.(string)
		if !ok || !parsesAsClock(s) {
			return label + " must be a valid time."
		}
	case *FileUploadField:
		list, ok := toStrings(val)
		if !ok {
			return label + " must be a file reference or a list of them."
		}
		if !f.multiple && len(list) > 1 {
			return label + " takes a single file."
		}
		if f.maxFiles > 0 && len(list) > f.maxFiles {
			return fmt.Sprintf("%s may not have more than %d files.", label, f.maxFiles)
		}
	case *CustomField:
		if f.rule != nil {
			return f.rule(v.ctx, val)
		}
	case *TagsInputField:
		tags, ok := toStrings(val)
		if !ok {
			return label + " must be a list."
		}
		if len(tags) == 1 && f.separator != "" {
			tags = strings.Split(tags[0], f.separator)
		}
		if f.maxTags > 0 && len(tags) > f.maxTags {
			return fmt.Sprintf("%s may not have more than %d items.", label, f.maxTags)
		}
	}
	return ""
}

func checkChoices(label string, val any, multiple bool, allowed func(string) bool) string {
	if multiple {
		list, ok := toStrings(val)
		if !ok {
			return label + " must be a list."
		}
		for _, s := range list {
			if !allowed(s) {
				return label + " contains an invalid choice."
			}
		}
		return ""
	}
	if _, isList := val.([]any); isList {
		return label + " must be a single choice."
	}
	if !allowed(scalarString(val)) {
		return label + " is not a valid choice."
	}
	return ""
}

var timeLayouts = []string{time.RFC3339Nano, time.RFC3339, "2006-01-02", "2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02 15:04", "2006-01-02 15:04:05"}

func parsesAsTime(s string) bool {
	for _, l := range timeLayouts {
		if _, err := time.Parse(l, s); err == nil {
			return true
		}
	}
	return false
}

func parsesAsClock(s string) bool {
	for _, l := range []string{"15:04", "15:04:05"} {
		if _, err := time.Parse(l, s); err == nil {
			return true
		}
	}
	return false
}

func (v *validator) repeater(f *RepeaterComponent, data, out map[string]any, prefix string) {
	b := &f.BaseField
	if b.name == "" || v.skip(b, data) {
		return
	}
	key := prefix + b.name
	raw, present := data[b.name]
	var items []any
	switch x := raw.(type) {
	case nil:
	case []any:
		items = x
	case []map[string]any:
		for _, m := range x {
			items = append(items, m)
		}
	default:
		v.fail(key, b.displayName()+" must be a list.")
		return
	}
	if !present || len(items) == 0 {
		if v.required(b) && (v.creating || present) {
			v.fail(key, b.displayName()+" is required.")
			return
		}
	}
	if !present {
		return
	}
	if f.minItems > 0 && len(items) < f.minItems {
		v.fail(key, fmt.Sprintf("%s needs at least %d items.", b.displayName(), f.minItems))
		return
	}
	if f.maxItems > 0 && len(items) > f.maxItems {
		v.fail(key, fmt.Sprintf("%s may not have more than %d items.", b.displayName(), f.maxItems))
		return
	}
	clean := make([]any, 0, len(items))
	for i, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			v.fail(fmt.Sprintf("%s.%d", key, i), b.displayName()+" items must be objects.")
			continue
		}
		// every item is a new entry: its required fields must be filled
		item := validator{ctx: v.ctx, creating: true, errors: v.errors}
		cm := map[string]any{}
		item.walk(f.schema, m, cm, fmt.Sprintf("%s.%d.", key, i))
		clean = append(clean, cm)
	}
	out[b.name] = clean
}
