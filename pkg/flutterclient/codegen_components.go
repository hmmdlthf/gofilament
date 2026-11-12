package flutterclient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ComponentKind is one of the kinds of custom Flutter components.
type ComponentKind string

const (
	// FieldComponent is a form field: forms.Custom(name, component).
	FieldComponent ComponentKind = "field"
	// CellComponent is a table cell: tables.CustomColumn(name, component).
	CellComponent ComponentKind = "cell"
	// WidgetComponent is a dashboard widget: filament.FlutterWidget(id, component).
	WidgetComponent ComponentKind = "widget"
)

// ComponentsDir is where hand-written components live, relative to the app
// root: lib/components/{fields,cells,widgets}.
const ComponentsDir = "lib/components"

var componentKinds = []ComponentKind{FieldComponent, CellComponent, WidgetComponent}

// ComponentFile is the path of a component's hand-written file.
func ComponentFile(kind ComponentKind, name string) string {
	return fmt.Sprintf("%s/%ss/%s_%s.dart", ComponentsDir, kind, snake(name), kind)
}

// ComponentClass is the widget class a component's file must define.
func ComponentClass(kind ComponentKind, name string) string {
	return pascal(name) + pascal(string(kind))
}

// componentParam is the constructor parameter that receives the context.
func componentParam(kind ComponentKind) (param, typ string) {
	switch kind {
	case FieldComponent:
		return "field", "AdminFieldContext"
	case CellComponent:
		return "cell", "AdminCellContext"
	}
	return "dashboard", "AdminWidgetContext"
}

// componentUse is one custom component the panel references.
type componentUse struct {
	kind     ComponentKind
	name     string
	props    []any // the props of each place it is used (decoded JSON)
	propsCls *dartClass
}

// collectComponents lists the components in a deterministic order: by kind,
// then by first use.
func (s *panelSnapshot) collectComponents() []*componentUse {
	byKey := map[string]*componentUse{}
	var order []*componentUse
	add := func(kind ComponentKind, name string, props any) {
		if name == "" {
			return
		}
		key := string(kind) + ":" + name
		u, ok := byKey[key]
		if !ok {
			u = &componentUse{kind: kind, name: name}
			byKey[key] = u
			order = append(order, u)
		}
		u.props = append(u.props, props)
	}
	var walk func(v any)
	walk = func(v any) {
		for _, n := range asList(v) {
			m, _ := n.(map[string]any)
			if m == nil {
				continue
			}
			if m["type"] == "custom" {
				add(FieldComponent, fmt.Sprint(m["component"]), m["props"])
			}
			walk(m["children"])
		}
	}
	for _, rs := range s.resources {
		walk(toGeneric(rs.form))
		for _, a := range rs.actions {
			walk(toGeneric(a.form))
		}
		for _, r := range rs.relations {
			walk(toGeneric(r.form))
		}
	}
	for _, r := range asList(s.schema["resources"]) {
		m, _ := r.(map[string]any)
		for _, c := range asList(m["columns"]) {
			if cm, ok := c.(map[string]any); ok && cm["type"] == "custom" {
				add(CellComponent, fmt.Sprint(cm["component"]), cm["props"])
			}
		}
	}
	for _, cols := range s.relationColumns {
		for _, c := range asList(cols) {
			if cm, ok := c.(map[string]any); ok && cm["type"] == "custom" {
				add(CellComponent, fmt.Sprint(cm["component"]), cm["props"])
			}
		}
	}
	for _, w := range s.widgets {
		add(WidgetComponent, w.component, toGeneric(w.props))
	}
	sort.SliceStable(order, func(i, j int) bool {
		return kindIndex(order[i].kind) < kindIndex(order[j].kind)
	})
	return order
}

func kindIndex(k ComponentKind) int {
	for i, c := range componentKinds {
		if c == k {
			return i
		}
	}
	return len(componentKinds)
}

// inferPropsClass builds a Dart class from the props values set in Go. A
// key keeps a concrete type only when every use agrees on it, and is
// non-null only when every use sets it.
func (r *typeRegistry) inferPropsClass(u *componentUse) *dartClass {
	type info struct {
		typ   string
		count int
	}
	keys := map[string]*info{}
	var order []string
	maps := 0
	for _, p := range u.props {
		m, ok := p.(map[string]any)
		if !ok {
			continue
		}
		maps++
		names := make([]string, 0, len(m))
		for k := range m {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, k := range names {
			t := jsonKind(m[k])
			in, ok := keys[k]
			if !ok {
				in = &info{typ: t}
				keys[k] = in
				order = append(order, k)
			} else if in.typ != t {
				in.typ = "dynamic"
			}
			in.count++
		}
	}
	if maps == 0 {
		return nil
	}
	c := r.add(pascal(u.name)+pascal(string(u.kind))+"Props",
		fmt.Sprintf("The props of the `%s` %s component, inferred from the values set in Go.", u.name, u.kind))
	members := nameSet{}
	for _, k := range order {
		in := keys[k]
		var t dartType
		switch in.typ {
		case "string":
			t = dartStringT
		case "int":
			t = dartIntT
		case "number":
			t = dartNumT
		case "bool":
			t = dartBoolT
		case "list":
			t = dartList(dartDynamic)
		case "map":
			t = dartMapT
		default:
			t = dartDynamic
		}
		nullable := in.count < maps || t.zero == "" || in.typ == "list" || in.typ == "map"
		c.fields = append(c.fields, dartField{key: k, name: members.unique(camel(k)), typ: t, nullable: nullable})
	}
	return c
}

func jsonKind(v any) string {
	switch x := v.(type) {
	case string:
		return "string"
	case bool:
		return "bool"
	case json.Number:
		if _, err := x.Int64(); err == nil {
			return "int"
		}
		return "number"
	case []any:
		return "list"
	case map[string]any:
		return "map"
	}
	return "dynamic"
}

// ComponentStub returns the starting content of a component's file.
func componentStub(u *componentUse, packageImport string) string {
	class := ComponentClass(u.kind, u.name)
	param, typ := componentParam(u.kind)
	var imports, propsGetter string
	if u.propsCls != nil {
		imports = "\nimport '../../generated/models.g.dart';\n"
		propsGetter = fmt.Sprintf("\n  %s get props => %s.fromJson(%s.props);\n", u.propsCls.name, u.propsCls.name, param)
	}
	var doc, body string
	switch u.kind {
	case FieldComponent:
		doc = fmt.Sprintf(`/// The %s form field component: forms.Custom(name, %s) in Go.
///
/// Call field.onChanged(value) with the new JSON value. The server checks it
/// (Required, Rule) when the form is saved.`, dartString(u.name), dartString(u.name))
		body = `InputDecorator(
        decoration: InputDecoration(
          labelText: field.label,
          helperText: field.helperText,
          errorText: field.errorText,
          enabled: field.enabled,
        ),
        child: Text(field.value == null ? '' : '${field.value}'),
      )`
	case CellComponent:
		doc = fmt.Sprintf(`/// The %s table cell component: tables.CustomColumn(name, %s) in Go.
/// cell.row is the whole record.`, dartString(u.name), dartString(u.name))
		body = `Text(cell.value == null ? '' : '${cell.value}')`
	default:
		doc = fmt.Sprintf(`/// The %s dashboard widget component: filament.FlutterWidget(id, %s) in Go.
/// dashboard.data is what the widget's Data function returned for this user.`, dartString(u.name), dartString(u.name))
		body = `Text('${dashboard.data ?? ''}')`
	}
	return fmt.Sprintf(`import 'package:flutter/material.dart';
import '%s';
%s
%s
///
/// This file is yours: the generator created it once and never changes it.
class %s extends StatelessWidget {
  const %s({super.key, required this.%s});

  final %s %s;
%s
  @override
  Widget build(BuildContext context) => %s;
}
`, packageImport, imports, doc, class, class, param, typ, param, propsGetter, body)
}

// componentStubs lists the stubs to create in appDir, or the first component
// file that cannot be used.
func componentStubs(uses []*componentUse, appDir, packageImport string) (map[string]string, error) {
	stubs := map[string]string{}
	seen := map[string]string{}
	for _, u := range uses {
		file := ComponentFile(u.kind, u.name)
		if other, ok := seen[file]; ok {
			return nil, fmt.Errorf("flutterclient: components %q and %q both map to %s", other, u.name, file)
		}
		seen[file] = u.name
		b, err := os.ReadFile(filepath.Join(appDir, file))
		if os.IsNotExist(err) {
			stubs[file] = componentStub(u, packageImport)
			continue
		}
		if err != nil {
			return nil, err
		}
		class := ComponentClass(u.kind, u.name)
		if !regexp.MustCompile(`\bclass\s+` + class + `\b`).Match(b) {
			param, _ := componentParam(u.kind)
			return nil, fmt.Errorf("flutterclient: %s must define class %s (the generated registry builds it with %s(%s: ...))", file, class, class, param)
		}
	}
	return stubs, nil
}

// componentsRegistry writes panelComponents into the registry file.
func (g *generator) componentsRegistry(b *strings.Builder) {
	b.WriteString("\n/// The app's custom components, built from your files in lib/components:\n/// `GoFilamentAdmin(components: panelComponents)`.\nconst AdminComponents panelComponents = AdminComponents(")
	if len(g.components) == 0 {
		b.WriteString(");\n")
		return
	}
	b.WriteString("\n")
	for _, kind := range componentKinds {
		var entries []string
		for _, u := range g.components {
			if u.kind == kind {
				entries = append(entries, fmt.Sprintf("    %s: _%s%s,\n", dartString(u.name), kind, pascal(u.name)))
			}
		}
		if len(entries) == 0 {
			continue
		}
		fmt.Fprintf(b, "  %ss: {\n%s  },\n", kind, strings.Join(entries, ""))
	}
	b.WriteString(");\n")
	for _, u := range g.components {
		param, typ := componentParam(u.kind)
		fmt.Fprintf(b, "\nWidget _%s%s(BuildContext context, %s %s) =>\n    c%d.%s(%s: %s);\n",
			u.kind, pascal(u.name), typ, param, g.componentIndex(u), ComponentClass(u.kind, u.name), param, param)
	}
}

func (g *generator) componentIndex(u *componentUse) int {
	for i, c := range g.components {
		if c == u {
			return i
		}
	}
	return -1
}
