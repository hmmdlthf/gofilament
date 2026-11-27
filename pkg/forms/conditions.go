package forms

import (
	"context"
	"fmt"
	"html"
	"strings"
)

// Condition shows a component only while another field of the form holds
// one of some values, for example the settings of the payment driver that
// is selected. Unlike Visible (a Go callback the server alone can run), a
// Condition is declarative: the web panel and the Flutter admin evaluate it
// as the user types, and Validate drops the values of hidden components.
type Condition struct {
	Field  string   `json:"field"`
	Values []string `json:"values"`
}

// Matches reports whether the condition holds for a form's state.
func (c *Condition) Matches(get Get) bool {
	if c == nil {
		return true
	}
	v := get(c.Field)
	var cur []string
	switch x := v.(type) {
	case nil:
	case []string:
		cur = x
	case []any:
		for _, e := range x {
			cur = append(cur, fmt.Sprint(e))
		}
	case bool:
		cur = []string{fmt.Sprint(x)}
	default:
		cur = []string{fmt.Sprint(x)}
	}
	for _, have := range cur {
		for _, want := range c.Values {
			if have == want {
				return true
			}
		}
	}
	return false
}

// VisibleWhen shows the field only while field holds one of values. For a
// toggle, use "true" or "false".
func (f *BaseField) VisibleWhen(field string, values ...string) *BaseField {
	f.visibleWhen = &Condition{Field: field, Values: values}
	return f
}

func (f *BaseField) visibilityCondition() *Condition { return f.visibleWhen }

// VisibleWhen shows the text input only while field holds one of values.
func (f *TextInputField) VisibleWhen(field string, values ...string) *TextInputField {
	f.BaseField.VisibleWhen(field, values...)
	return f
}

// VisibleWhen shows the textarea only while field holds one of values.
func (f *TextareaField) VisibleWhen(field string, values ...string) *TextareaField {
	f.BaseField.VisibleWhen(field, values...)
	return f
}

// VisibleWhen shows the select only while field holds one of values.
func (f *SelectField) VisibleWhen(field string, values ...string) *SelectField {
	f.BaseField.VisibleWhen(field, values...)
	return f
}

// VisibleWhen shows the toggle only while field holds one of values.
func (f *ToggleField) VisibleWhen(field string, values ...string) *ToggleField {
	f.BaseField.VisibleWhen(field, values...)
	return f
}

// VisibleWhen shows the section only while field holds one of values.
func (s *SectionComponent) VisibleWhen(field string, values ...string) *SectionComponent {
	s.visibleWhen = &Condition{Field: field, Values: values}
	return s
}

func (s *SectionComponent) visibilityCondition() *Condition { return s.visibleWhen }

// VisibleWhen shows the grid only while field holds one of values.
func (g *GridComponent) VisibleWhen(field string, values ...string) *GridComponent {
	g.visibleWhen = &Condition{Field: field, Values: values}
	return g
}

func (g *GridComponent) visibilityCondition() *Condition { return g.visibleWhen }

type conditional interface{ visibilityCondition() *Condition }

func conditionOf(c Component) *Condition {
	if cc, ok := c.(conditional); ok {
		return cc.visibilityCondition()
	}
	return nil
}

func stateGetter(state map[string]any) Get {
	return func(key string) any { return state[key] }
}

// renderChild renders a component, wrapped so the panel script can show
// and hide it when it has a Condition.
func renderChild(ctx context.Context, c Component, state map[string]any, errors map[string]string) string {
	if !gateAllows(ctx, c) {
		return ""
	}
	out := c.Render(ctx, state, errors)
	cond := conditionOf(c)
	if cond == nil {
		return out
	}
	style := ""
	if !cond.Matches(stateGetter(state)) {
		style = ` style="display:none"`
	}
	return fmt.Sprintf(`<div data-visible-when="%s" data-visible-values="%s"%s>%s</div>`,
		html.EscapeString(cond.Field), html.EscapeString(strings.Join(cond.Values, "\x1f")), style, out)
}
