package flutterclient

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
)

type generator struct {
	snap      *panelSnapshot
	o         GenerateOptions
	reg       *typeRegistry
	resources []*resGen
	endpoints []*endpointGen
	client    string // name of the panel client class
}

type resGen struct {
	rs         resourceSnapshot
	record     *dartClass
	input      *dartClass // nil without a form
	actions    []*actGen
	relations  []*relGen
	clientName string
	member     string
}

type actGen struct {
	a      actionSnapshot
	input  *dartClass // nil without a form
	method string
}

type relGen struct {
	r          relationSnapshot
	record     *dartClass
	input      *dartClass
	clientName string
	member     string
}

type pathParam struct{ segment, name string }

type endpointGen struct {
	e      *filament.ClientEndpoint
	req    *dartType
	res    *dartType // nil: no response body
	params []pathParam
}

func newGenerator(snap *panelSnapshot, o GenerateOptions) *generator {
	return &generator{snap: snap, o: o, reg: newTypeRegistry(), client: "PanelClient"}
}

func (g *generator) render() (map[string][]byte, error) {
	if err := g.plan(); err != nil {
		return nil, err
	}
	files := map[string][]byte{
		GeneratedDir + "/schema.g.dart":   []byte(g.schemaFile()),
		GeneratedDir + "/models.g.dart":   []byte(g.modelsFile()),
		GeneratedDir + "/client.g.dart":   []byte(g.clientFile()),
		GeneratedDir + "/pages.g.dart":    []byte(g.pagesFile()),
		GeneratedDir + "/registry.g.dart": []byte(g.registryFile()),
	}
	return files, nil
}

// plan assigns every class and member name, in panel order.
func (g *generator) plan() error {
	clientNames := nameSet{g.client: true}
	members := nameSet{}
	for _, rs := range g.snap.resources {
		model := pascal(rs.model)
		rg := &resGen{rs: rs}
		goName := "record"
		if rs.recordType != nil {
			goName = "`" + rs.recordType.String() + "`"
		}
		rg.record = g.reg.recordClass(rs.recordType, model, fmt.Sprintf("A `%s` record (Go type %s).", rs.slug, goName))
		if rg.record.doc == "" {
			rg.record.doc = fmt.Sprintf("A record (Go type %s). Also used by the `%s` resource.", goName, rs.slug)
		}
		if rs.hasForm {
			rg.input = g.reg.inputClass(model+"Input", fmt.Sprintf("The values of the `%s` form (create and edit).\n\nOnly non-null fields are sent; put explicit nulls and extra keys in [extra].", rs.slug), rs.form)
		}
		methods := nameSet{"list": true, "find": true, "create": true, "update": true, "delete": true, "deleteMany": true, "slug": true, "columns": true}
		for _, a := range rs.actions {
			ag := &actGen{a: a, method: methods.unique("run" + pascal(a.name))}
			if a.hasForm {
				ag.input = g.reg.inputClass(model+pascal(a.name)+"Input", fmt.Sprintf("The input of the `%s` row action (%s) of `%s`.", a.name, a.label, rs.slug), a.form)
			}
			rg.actions = append(rg.actions, ag)
		}
		for _, r := range rs.relations {
			relName := model + pascal(singular(pascal(r.name)))
			rel := &relGen{r: r, clientName: clientNames.unique(pascal(rs.slug) + pascal(r.name) + "Client"), member: methods.unique(camel(r.name))}
			name := relName
			if r.recordType != nil {
				if t := derefType(r.recordType); t.Name() != "" {
					name = pascal(goTypeName(t))
				}
			}
			rel.record = g.reg.recordClass(r.recordType, name, fmt.Sprintf("A `%s` related record of `%s`.", r.name, rs.slug))
			if r.hasForm {
				rel.input = g.reg.inputClass(relName+"Input", fmt.Sprintf("The values of the `%s` relation form of `%s`.\n\nOnly non-null fields are sent; put explicit nulls and extra keys in [extra].", r.name, rs.slug), r.form)
			}
			rg.relations = append(rg.relations, rel)
		}
		rg.clientName = clientNames.unique(pascal(rs.slug) + "Client")
		rg.member = members.unique(camel(rs.slug))
		g.resources = append(g.resources, rg)
	}

	for _, e := range g.snap.endpoints {
		eg := &endpointGen{e: e}
		name := camel(e.Name())
		if members[name] {
			return fmt.Errorf("flutterclient: endpoint %q clashes with another client member", e.Name())
		}
		members[name] = true
		for _, seg := range strings.Split(e.Path(), "/") {
			if strings.HasPrefix(seg, ":") || strings.HasPrefix(seg, "*") {
				eg.params = append(eg.params, pathParam{segment: seg, name: camel(seg[1:])})
			}
		}
		if rt := derefType(e.RequestType()); !isEmptyStruct(rt) {
			t, _ := g.reg.goType(e.RequestType(), pascal(e.Name())+"Request")
			if (e.Method() == http.MethodGet || e.Method() == http.MethodDelete) && rt.Kind() != reflect.Struct {
				return fmt.Errorf("flutterclient: endpoint %q: a %s request must be a struct (it is sent as query parameters)", e.Name(), e.Method())
			}
			if rt.Kind() == reflect.Struct && rt.Name() == "" {
				// anonymous structs get a name from the endpoint
				t = dartClassType(g.reg.recordClass(rt, pascal(e.Name())+"Request", "").name)
			}
			eg.req = &t
		}
		if rt := derefType(e.ResponseType()); !isEmptyStruct(rt) {
			t, _ := g.reg.goType(e.ResponseType(), pascal(e.Name())+"Response")
			if rt.Kind() == reflect.Struct && rt.Name() == "" {
				t = dartClassType(g.reg.recordClass(rt, pascal(e.Name())+"Response", "").name)
			}
			eg.res = &t
		}
		g.endpoints = append(g.endpoints, eg)
	}
	return nil
}

func derefType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func isEmptyStruct(t reflect.Type) bool {
	return t == nil || (t.Kind() == reflect.Struct && t.NumField() == 0)
}

func (g *generator) header(what string) string {
	return fmt.Sprintf(`// GENERATED CODE - DO NOT EDIT.
//
// %s of the %s panel (%s), written by gofilament's
// flutterclient.GenerateFromPanel. Change the panel in Go and regenerate;
// keep hand-written code in other files.

// ignore_for_file: type=lint
`, what, dartString(g.snap.id), g.snap.path)
}

// ---------------------------------------------------------------- schema

func (g *generator) schemaFile() string {
	var b strings.Builder
	b.WriteString(g.header("Schema snapshot"))
	b.WriteString(`
/// The panel schema as the generator saw it, without per-user data:
/// permissions, dashboard values and tenants are left out. Pass it to
/// ` + "`GoFilamentAdmin(initialSchema: panelSchema)`" + ` to draw the shell before the
/// live schema arrives; the live schema still decides what the user may do.
const Map<String, dynamic> panelSchema = `)
	dartLiteral(&b, g.snap.schema, "")
	b.WriteString(";\n\n")

	forms := map[string]any{}
	for _, rs := range g.snap.resources {
		entry := map[string]any{}
		if rs.hasForm {
			entry["form"] = sanitizeForm(rs.form)
		}
		actions := map[string]any{}
		for _, a := range rs.actions {
			if a.hasForm {
				actions[a.name] = sanitizeForm(a.form)
			}
		}
		if len(actions) > 0 {
			entry["actions"] = actions
		}
		rels := map[string]any{}
		for _, r := range rs.relations {
			if r.hasForm {
				rels[r.name] = sanitizeForm(r.form)
			}
		}
		if len(rels) > 0 {
			entry["relations"] = rels
		}
		if len(entry) > 0 {
			forms[rs.slug] = entry
		}
	}
	b.WriteString(`/// Form layouts by resource slug: ` + "`{form, actions: {name: ...}, relations: {name: ...}}`" + `.
/// Options loaded from data are left out; the live form endpoints return them.
const Map<String, dynamic> panelForms = `)
	dartLiteral(&b, forms, "")
	b.WriteString(";\n")
	return b.String()
}

// ---------------------------------------------------------------- models

func (g *generator) modelsFile() string {
	var b strings.Builder
	b.WriteString(g.header("Models"))
	b.WriteString("\nimport '" + g.o.PackageImport + "';\n")
	for _, c := range g.reg.classes {
		b.WriteString("\n")
		writeClass(&b, c)
	}
	return b.String()
}

func writeClass(b *strings.Builder, c *dartClass) {
	b.WriteString(docComment("", c.doc))
	fmt.Fprintf(b, "class %s {\n", c.name)

	// constructor
	if len(c.fields) == 0 && !c.input {
		fmt.Fprintf(b, "  const %s({this.raw = const <String, dynamic>{}});\n\n", c.name)
	} else {
		fmt.Fprintf(b, "  const %s({\n", c.name)
		for _, f := range c.fields {
			if !f.nullable && !f.typ.dynamic {
				fmt.Fprintf(b, "    this.%s = %s,\n", f.name, f.typ.zero)
			} else {
				fmt.Fprintf(b, "    this.%s,\n", f.name)
			}
		}
		if c.input {
			b.WriteString("    this.extra = const <String, dynamic>{},\n")
		} else {
			b.WriteString("    this.raw = const <String, dynamic>{},\n")
		}
		b.WriteString("  });\n\n")
	}

	// fromJson
	fmt.Fprintf(b, "  factory %s.fromJson(Map<String, dynamic> json) => %s(\n", c.name, c.name)
	for _, f := range c.fields {
		read := f.typ.read("json[" + dartString(f.key) + "]")
		if !f.nullable && !f.typ.dynamic {
			read += " ?? " + f.typ.zero
		}
		fmt.Fprintf(b, "        %s: %s,\n", f.name, read)
	}
	if !c.input {
		b.WriteString("        raw: json,\n")
	}
	b.WriteString("      );\n\n")

	// fields
	for _, f := range c.fields {
		b.WriteString(docComment("  ", f.doc))
		fmt.Fprintf(b, "  final %s %s;\n", f.declType(), f.name)
	}
	if c.input {
		b.WriteString("\n  /// Sent as well, after the fields above (overrides them).\n  final Map<String, dynamic> extra;\n\n")
	} else {
		b.WriteString("\n  /// The JSON the record was read from, including fields not modelled here.\n  final Map<String, dynamic> raw;\n\n")
	}

	// toJson
	b.WriteString("  Map<String, dynamic> toJson() => <String, dynamic>{\n")
	if c.input {
		for _, f := range c.fields {
			fmt.Fprintf(b, "        if (%s != null) %s: %s,\n", f.name, dartString(f.key), f.typ.write(f.name, true))
		}
		b.WriteString("        ...extra,\n")
	} else {
		b.WriteString("        ...raw,\n")
		for _, f := range c.fields {
			fmt.Fprintf(b, "        %s: %s,\n", dartString(f.key), f.typ.write(f.name, f.nullable || f.typ.dynamic))
		}
	}
	b.WriteString("      };\n}\n")
}

// ---------------------------------------------------------------- client

func (g *generator) clientFile() string {
	var b strings.Builder
	b.WriteString(g.header("Typed API client"))
	b.WriteString("\nimport '" + g.o.PackageImport + "';\n\nimport 'models.g.dart';\n\n")

	fmt.Fprintf(&b, `/// Typed client of the %s panel API. Every call goes through [api], so it
/// uses the signed-in session; the server checks permissions on each request.
///
/// In a custom page: `+"`final client = %s(admin.api);`"+`
class %s {
  %s(this.api);

  final AdminApi api;
`, dartString(g.snap.id), g.client, g.client, g.client)
	for _, rg := range g.resources {
		fmt.Fprintf(&b, "\n  /// The `%s` resource (%s).\n  late final %s %s = %s(api);\n", rg.rs.slug, rg.rs.plural, rg.clientName, rg.member, rg.clientName)
	}
	for _, eg := range g.endpoints {
		b.WriteString("\n")
		g.writeEndpoint(&b, eg)
	}
	b.WriteString("}\n")

	for _, rg := range g.resources {
		b.WriteString("\n")
		g.writeResourceClient(&b, rg)
		for _, rel := range rg.relations {
			b.WriteString("\n")
			g.writeRelationClient(&b, rg, rel)
		}
	}
	return b.String()
}

func dartStringList(items []string) string {
	if len(items) == 0 {
		return "<String>[]"
	}
	q := make([]string, len(items))
	for i, s := range items {
		q[i] = dartString(s)
	}
	return "<String>[" + strings.Join(q, ", ") + "]"
}

func (g *generator) writeResourceClient(b *strings.Builder, rg *resGen) {
	rec := rg.record.name
	fmt.Fprintf(b, `/// Typed calls for the `+"`%s`"+` resource.
class %s {
  const %s(this.api);

  final AdminApi api;

  static const String slug = %s;

  /// Table columns, in display order.
  static const List<String> columns = %s;

  /// One page of records with the per-row abilities the server granted.
  Future<TypedPage<%s>> list({
    int page = 1,
    int perPage = 25,
    String search = '',
    String sort = '',
    bool descending = false,
    Map<String, String> filters = const {},
  }) async =>
      TypedPage.from(
          await api.list(slug,
              page: page,
              perPage: perPage,
              search: search,
              sort: sort,
              descending: descending,
              filters: filters),
          %s.fromJson);

  Future<%s> find(String id) async => %s.fromJson(await api.find(slug, id));
`, rg.rs.slug, rg.clientName, rg.clientName, dartString(rg.rs.slug), dartStringList(rg.rs.columns), rec, rec, rec, rec)
	if rg.input != nil {
		fmt.Fprintf(b, `
  Future<void> create(%s input) => api.create(slug, input.toJson());

  Future<void> update(String id, %s input) =>
      api.update(slug, id, input.toJson());
`, rg.input.name, rg.input.name)
	}
	b.WriteString(`
  Future<void> delete(String id) => api.delete(slug, id);

  Future<void> deleteMany(List<String> ids) => api.deleteMany(slug, ids);
`)
	for _, ag := range rg.actions {
		fmt.Fprintf(b, "\n  /// Runs the `%s` row action (%s); returns the server's message.\n", ag.a.name, ag.a.label)
		if ag.input != nil {
			fmt.Fprintf(b, "  Future<String> %s(String id, %s input) =>\n      api.runAction(slug, id, %s, input.toJson());\n", ag.method, ag.input.name, dartString(ag.a.name))
		} else {
			fmt.Fprintf(b, "  Future<String> %s(String id) => api.runAction(slug, id, %s);\n", ag.method, dartString(ag.a.name))
		}
	}
	for _, rel := range rg.relations {
		fmt.Fprintf(b, "\n  /// The `%s` relation (%s) of record [id].\n  %s %s(String id) => %s(api, id);\n", rel.r.name, rel.r.title, rel.clientName, rel.member, rel.clientName)
	}
	b.WriteString("}\n")
}

func (g *generator) writeRelationClient(b *strings.Builder, rg *resGen, rel *relGen) {
	rec := rel.record.name
	fmt.Fprintf(b, `/// Typed calls for the `+"`%s`"+` relation of one `+"`%s`"+` record.
class %s {
  const %s(this.api, this.parentId);

  final AdminApi api;
  final String parentId;

  static const String relationship = %s;

  Future<TypedPage<%s>> list(
          {int page = 1, int perPage = 50, String search = ''}) async =>
      TypedPage.from(
          await api.relationList(%s.slug, parentId, relationship,
              page: page, perPage: perPage, search: search),
          %s.fromJson);
`, rel.r.name, rg.rs.slug, rel.clientName, rel.clientName, dartString(rel.r.name), rec, rg.clientName, rec)
	if rel.input != nil {
		fmt.Fprintf(b, `
  Future<void> create(%s input) =>
      api.relationCreate(%s.slug, parentId, relationship, input.toJson());

  Future<void> update(String childId, %s input) => api.relationUpdate(
      %s.slug, parentId, relationship, childId, input.toJson());
`, rel.input.name, rg.clientName, rel.input.name, rg.clientName)
	}
	fmt.Fprintf(b, `
  Future<void> delete(String childId) =>
      api.relationDelete(%s.slug, parentId, relationship, childId);
}
`, rg.clientName)
}

func (g *generator) writeEndpoint(b *strings.Builder, eg *endpointGen) {
	e := eg.e
	doc := e.Description()
	if doc == "" {
		doc = fmt.Sprintf("Calls the `%s` endpoint.", e.Name())
	}
	doc += fmt.Sprintf("\n\n`%s %s`", e.Method(), e.Path())
	b.WriteString(docComment("  ", doc))

	query := e.Method() == http.MethodGet || e.Method() == http.MethodDelete
	var params []string
	for _, p := range eg.params {
		params = append(params, "required String "+p.name)
	}
	if eg.req != nil {
		if query {
			params = append(params, eg.req.decl+"? request")
		} else {
			params = append(params, "required "+eg.req.decl+" request")
		}
	}

	ret := "void"
	if eg.res != nil {
		ret = eg.res.decl
		if eg.res.dynamic {
			ret = "dynamic"
		} else if eg.res.zero == "" {
			ret += "?"
		}
	}
	sig := ""
	if len(params) > 0 {
		sig = "{" + strings.Join(params, ", ") + "}"
	}
	fmt.Fprintf(b, "  Future<%s> %s(%s) async {\n", ret, camel(e.Name()), sig)

	path := e.Path()
	for _, p := range eg.params {
		path = strings.Replace(path, p.segment, "${Uri.encodeComponent("+p.name+")}", 1)
	}
	pathExpr := dartString(path)
	pathExpr = strings.ReplaceAll(pathExpr, `\${Uri.encodeComponent(`, "${Uri.encodeComponent(")

	var callExpr string
	switch {
	case query:
		q := "const <String, String>{}"
		if eg.req != nil {
			q = "GfJson.query(" + eg.req.write("request", true) + ")"
		}
		if e.Method() == http.MethodGet {
			callExpr = "api.get(" + pathExpr + ", " + q + ")"
		} else {
			callExpr = "api.send('DELETE', GfJson.withQuery(" + pathExpr + ", " + q + "))"
		}
	default:
		body := "null"
		if eg.req != nil {
			body = eg.req.write("request", false)
		}
		callExpr = "api.send(" + dartString(e.Method()) + ", " + pathExpr + ", " + body + ")"
	}

	switch {
	case eg.res == nil:
		fmt.Fprintf(b, "    await %s;\n", callExpr)
	case eg.res.dynamic:
		fmt.Fprintf(b, "    return %s;\n", callExpr)
	default:
		read := eg.res.read("await " + callExpr)
		if eg.res.zero != "" {
			read += " ?? " + eg.res.zero
		}
		fmt.Fprintf(b, "    return %s;\n", read)
	}
	b.WriteString("  }\n")
}

// ---------------------------------------------------------------- pages

func (g *generator) pagesFile() string {
	var b strings.Builder
	b.WriteString(g.header("Flutter pages"))
	b.WriteString("\nimport 'package:flutter/material.dart';\nimport '" + g.o.PackageImport + "';\n\n")
	b.WriteString("/// Draws one page.\ntypedef PanelPageBuilder = Widget Function(\n    BuildContext context, AdminController admin);\n\n")

	type pg struct {
		info   filament.FlutterPageInfo
		member string
	}
	var pages []pg
	members := pageMembers(g.snap.pages)
	for i, p := range g.snap.pages {
		pages = append(pages, pg{info: p, member: members[i]})
	}

	b.WriteString(`/// The builders of the pages the panel declares with filament.FlutterPage.
/// Write them in your own files; adding a page in Go adds a required
/// parameter here, so the app does not build until the page exists.
class PanelPageBuilders {
`)
	if len(pages) == 0 {
		b.WriteString("  const PanelPageBuilders();\n")
	} else {
		b.WriteString("  const PanelPageBuilders({\n")
		for _, p := range pages {
			fmt.Fprintf(&b, "    required this.%s,\n", p.member)
		}
		b.WriteString("  });\n")
		for _, p := range pages {
			fmt.Fprintf(&b, "\n  /// %s (`%s`).\n  final PanelPageBuilder %s;\n", p.info.Label, p.info.ID, p.member)
		}
	}
	b.WriteString("}\n\n")

	b.WriteString(`/// The panel's Flutter pages for ` + "`GoFilamentAdmin(pages: ...)`" + `. The server decides
/// who sees each one: a page is shown only when the live schema lists it.
List<AdminPage> panelPages(PanelPageBuilders builders) => <AdminPage>[
`)
	for _, p := range pages {
		fmt.Fprintf(&b, `      AdminPage(
        id: %s,
        label: %s,
        icon: adminIcon(%s, fallback: Icons.widgets_outlined),
        group: %s,
        fullscreen: %t,
        serverControlled: true,
        builder: builders.%s,
      ),
`, dartString(p.info.ID), dartString(p.info.Label), dartString(p.info.Icon), dartString(p.info.Group), p.info.Fullscreen, p.member)
	}
	b.WriteString("    ];\n")
	return b.String()
}
