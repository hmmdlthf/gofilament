package flutterclient

import (
	"encoding"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/forms"
)

// dartType is how a value of one Dart type is read from and written to JSON.
type dartType struct {
	decl string // the non-null Dart type
	// read turns a `dynamic` JSON expression into a nullable decl expression.
	read func(v string) string
	// write turns a decl expression into a JSON expression (nullable: the
	// expression may be null and so may the result).
	write func(v string, nullable bool) string
	// zero is a const default for non-null fields ("" when there is none).
	zero string
	// dynamic types are declared as `dynamic`, never with `?`.
	dynamic bool
}

func identity(v string, _ bool) string { return v }

func call(fn string) func(string) string {
	return func(v string) string { return fn + "(" + v + ")" }
}

var (
	dartDynamic  = dartType{decl: "dynamic", read: func(v string) string { return v }, write: identity, dynamic: true}
	dartStringT  = dartType{decl: "String", read: call("GfJson.string"), write: identity, zero: "''"}
	dartIntT     = dartType{decl: "int", read: call("GfJson.integer"), write: identity, zero: "0"}
	dartDoubleT  = dartType{decl: "double", read: call("GfJson.decimal"), write: identity, zero: "0.0"}
	dartNumT     = dartType{decl: "num", read: call("GfJson.number"), write: identity, zero: "0"}
	dartBoolT    = dartType{decl: "bool", read: call("GfJson.boolean"), write: identity, zero: "false"}
	dartMapT     = dartType{decl: "Map<String, dynamic>", read: call("GfJson.map"), write: identity, zero: "const <String, dynamic>{}"}
	dartDateTime = dartType{decl: "DateTime", read: call("GfJson.dateTime"), write: func(v string, nullable bool) string {
		if nullable {
			return v + "?.toUtc().toIso8601String()"
		}
		return v + ".toUtc().toIso8601String()"
	}}
	// dartDate is a calendar date sent as yyyy-mm-dd (date form fields).
	dartDate = dartType{decl: "DateTime", read: call("GfJson.dateTime"), write: func(v string, _ bool) string {
		return "GfJson.date(" + v + ")"
	}}
)

func dartList(elem dartType) dartType {
	elemNullable := elem.zero == "" && !elem.dynamic
	elemDecl := elem.decl
	if elemNullable {
		elemDecl += "?"
	}
	item := elem.read("e")
	if elem.zero != "" {
		item += " ?? " + elem.zero
	}
	return dartType{
		decl: "List<" + elemDecl + ">",
		read: func(v string) string { return "GfJson.list(" + v + ", (e) => " + item + ")" },
		write: func(v string, nullable bool) string {
			w := elem.write("e", elemNullable)
			if w == "e" {
				return v
			}
			op := "."
			if nullable {
				op = "?."
			}
			return v + op + "map((e) => " + w + ").toList()"
		},
		zero: "const <" + elemDecl + ">[]",
	}
}

func dartClassType(name string) dartType {
	return dartType{
		decl: name,
		read: func(v string) string { return "GfJson.object(" + v + ", " + name + ".fromJson)" },
		write: func(v string, nullable bool) string {
			if nullable {
				return v + "?.toJson()"
			}
			return v + ".toJson()"
		},
		zero: name + ".fromJson(const <String, dynamic>{})",
	}
}

type dartField struct {
	key      string // JSON key
	name     string // Dart member
	typ      dartType
	nullable bool
	doc      string
}

func (f dartField) declType() string {
	if f.typ.dynamic {
		return "dynamic"
	}
	if f.nullable {
		return f.typ.decl + "?"
	}
	return f.typ.decl
}

type dartClass struct {
	name   string
	doc    string
	input  bool // form input: nulls are omitted, extra values are merged
	fields []dartField
}

// typeRegistry assigns one Dart class to each Go struct type and to each
// form, in a deterministic order.
type typeRegistry struct {
	names   nameSet
	byType  map[reflect.Type]*dartClass
	classes []*dartClass
}

func newTypeRegistry() *typeRegistry {
	return &typeRegistry{names: nameSet{}, byType: map[reflect.Type]*dartClass{}}
}

func (r *typeRegistry) add(name, doc string) *dartClass {
	c := &dartClass{name: r.names.unique(name), doc: doc}
	r.classes = append(r.classes, c)
	return c
}

var (
	timeType          = reflect.TypeFor[time.Time]()
	jsonMarshalerType = reflect.TypeFor[json.Marshaler]()
	textMarshalerType = reflect.TypeFor[encoding.TextMarshaler]()
)

func implements(t, iface reflect.Type) bool {
	return t.Implements(iface) || (t.Kind() != reflect.Pointer && reflect.PointerTo(t).Implements(iface))
}

// goTypeName is a Go type's name without package or type arguments.
func goTypeName(t reflect.Type) string {
	n := t.Name()
	if i := strings.IndexByte(n, '['); i >= 0 {
		n = n[:i]
	}
	return n
}

// recordClass returns the class for a Go record type. name is used when the
// type has no class yet (a resource's model name); otherwise the Go type
// name is.
func (r *typeRegistry) recordClass(t reflect.Type, name, doc string) *dartClass {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct || implements(t, jsonMarshalerType) {
		return r.add(name, doc)
	}
	if c, ok := r.byType[t]; ok {
		return c
	}
	if name == "" {
		name = pascal(goTypeName(t))
	}
	if doc == "" && t.Name() != "" {
		doc = "Go type `" + t.String() + "`."
	}
	c := r.add(name, doc)
	r.byType[t] = c
	members := nameSet{}
	seen := map[string]bool{}
	c.fields = r.structFields(t, c.name, seen, members)
	return c
}

// structFields lists the JSON fields of a struct the way encoding/json
// writes them: direct fields first, then those of embedded structs.
func (r *typeRegistry) structFields(t reflect.Type, owner string, seen map[string]bool, members nameSet) []dartField {
	var out []dartField
	var embedded []reflect.Type
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if f.Anonymous && name == "" {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				embedded = append(embedded, ft)
				continue
			}
		}
		if !f.IsExported() {
			continue
		}
		key := name
		if key == "" {
			key = f.Name
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		typ, nullable := r.goType(f.Type, owner+pascal(f.Name))
		if strings.Contains(","+opts+",", ",string,") {
			typ, nullable = dartDynamic, true
		}
		if strings.Contains(","+opts+",", ",omitempty,") || strings.Contains(","+opts+",", ",omitzero,") {
			nullable = true
		}
		out = append(out, dartField{key: key, name: members.unique(camel(key)), typ: typ, nullable: nullable})
	}
	for _, et := range embedded {
		out = append(out, r.structFields(et, owner, seen, members)...)
	}
	return out
}

// goType maps a Go type to a Dart type. hint names an anonymous struct.
func (r *typeRegistry) goType(t reflect.Type, hint string) (dartType, bool) {
	if t == timeType {
		return dartDateTime, true
	}
	if t.Kind() == reflect.Pointer {
		inner, _ := r.goType(t.Elem(), hint)
		return inner, true
	}
	if implements(t, jsonMarshalerType) {
		return dartDynamic, true
	}
	if implements(t, textMarshalerType) {
		return dartStringT, false
	}
	switch t.Kind() {
	case reflect.String:
		return dartStringT, false
	case reflect.Bool:
		return dartBoolT, false
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return dartIntT, false
	case reflect.Float32, reflect.Float64:
		return dartDoubleT, false
	case reflect.Struct:
		name := ""
		if t.Name() == "" {
			name = hint
		}
		return dartClassType(r.recordClass(t, name, "").name), true
	case reflect.Slice, reflect.Array:
		if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
			return dartStringT, true // base64
		}
		elem, _ := r.goType(t.Elem(), hint+"Item")
		return dartList(elem), true
	case reflect.Map:
		return dartMapT, true
	}
	return dartDynamic, true
}

// inputClass builds the class of a form's values.
func (r *typeRegistry) inputClass(name, doc string, nodes []forms.Node) *dartClass {
	c := r.add(name, doc)
	c.input = true
	members := nameSet{}
	seen := map[string]bool{}
	var walk func(ns []forms.Node)
	walk = func(ns []forms.Node) {
		for _, n := range ns {
			switch {
			case n.Kind == "field" && n.Name != "":
				if seen[n.Name] {
					continue
				}
				seen[n.Name] = true
				c.fields = append(c.fields, dartField{key: n.Name, name: members.unique(camel(n.Name)), typ: formFieldType(n), nullable: true, doc: fieldDoc(n)})
			case n.Kind == "repeater" && n.Name != "":
				if seen[n.Name] {
					continue
				}
				seen[n.Name] = true
				item := r.inputClass(c.name+pascal(n.Name)+"Item", "One entry of "+dartRef(c.name)+"."+camel(n.Name)+".", n.Children)
				c.fields = append(c.fields, dartField{key: n.Name, name: members.unique(camel(n.Name)), typ: dartList(dartClassType(item.name)), nullable: true, doc: fieldDoc(n)})
			default:
				walk(n.Children)
			}
		}
	}
	walk(nodes)
	return c
}

func dartRef(name string) string { return "[" + name + "]" }

func formFieldType(n forms.Node) dartType {
	switch n.Type {
	case "number":
		return dartNumT
	case "toggle":
		return dartBoolT
	case "select":
		if n.Multiple {
			return dartList(dartStringT)
		}
		return dartStringT
	case "tags", "checkbox_list":
		return dartList(dartStringT)
	case "date":
		return dartDate
	case "datetime":
		return dartDateTime
	case "key_value":
		return dartMapT
	case "file", "image", "hidden":
		return dartDynamic
	}
	return dartStringT
}

func fieldDoc(n forms.Node) string {
	doc := n.Label
	if doc == "" {
		doc = n.Name
	}
	var notes []string
	if n.Required {
		notes = append(notes, "required")
	} else if n.RequiredOnCreate {
		notes = append(notes, "required on create")
	}
	if n.Type != "" && n.Type != "text" {
		notes = append(notes, n.Type)
	}
	if len(n.Options) > 0 && !n.DynamicOptions {
		vals := make([]string, 0, len(n.Options))
		for _, o := range n.Options {
			vals = append(vals, "`"+o.Value+"`")
		}
		if len(vals) <= 12 {
			notes = append(notes, "one of "+strings.Join(vals, ", "))
		}
	}
	if len(notes) > 0 {
		doc += " (" + strings.Join(notes, "; ") + ")"
	}
	return doc + "."
}
