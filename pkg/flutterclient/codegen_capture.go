package flutterclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"

	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/golaravel"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
)

// panelSnapshot is everything the generator reads from a panel.
type panelSnapshot struct {
	id        string
	path      string
	schema    map[string]any // sanitized /api/schema
	resources []resourceSnapshot
	pages     []filament.FlutterPageInfo
	endpoints []*filament.ClientEndpoint
	// relationColumns are the column DTOs of every relation (decoded JSON).
	relationColumns []any
	widgets         []widgetSnapshot
}

type widgetSnapshot struct {
	component string
	props     any
}

type resourceSnapshot struct {
	slug       string
	model      string
	plural     string
	recordType reflect.Type
	columns    []string
	hasForm    bool
	form       []forms.Node
	actions    []actionSnapshot
	relations  []relationSnapshot
}

type actionSnapshot struct {
	name, label string
	form        []forms.Node // nil: no input
	hasForm     bool
}

type relationSnapshot struct {
	name, title string
	recordType  reflect.Type
	hasForm     bool
	form        []forms.Node
}

type recordTyper interface{ RecordType() reflect.Type }

// capturePanel boots p on a private engine and reads it as user.
func capturePanel(p *filament.Panel, user auth.Authenticatable) (*panelSnapshot, error) {
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		return nil, fmt.Errorf("flutterclient: register panel: %w", err)
	}

	newRequest := func(method, path string) *http.Request {
		req := httptest.NewRequest(method, "http://gofilament.codegen"+path, nil)
		req.Header.Set("Accept", "application/json")
		return req.WithContext(filament.WithClientGenerator(req.Context(), user))
	}

	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, newRequest(http.MethodGet, p.GetPath()+"/api/schema"))
	if rec.Code != http.StatusOK {
		return nil, fmt.Errorf("flutterclient: GET %s/api/schema answered %d: %s", p.GetPath(), rec.Code, bytes.TrimSpace(rec.Body.Bytes()))
	}
	schema, err := decodeJSON(rec.Body.Bytes())
	if err != nil {
		return nil, fmt.Errorf("flutterclient: decode schema: %w", err)
	}
	schemaMap, _ := schema.(map[string]any)
	if schemaMap == nil {
		return nil, fmt.Errorf("flutterclient: schema is not an object")
	}

	// A panel context for the describers, signed in as user.
	gc := gcontext.NewContext()
	gc.Reset(httptest.NewRecorder(), newRequest(http.MethodGet, p.GetPath()+"/api/schema"))
	gc.Set(auth.ContextUserKey, user)
	ctx := filament.NewContext(gc, p)

	snap := &panelSnapshot{id: p.ID(), path: p.GetPath(), endpoints: p.GetEndpoints()}

	visible := map[string]map[string]any{}
	for _, r := range asList(schemaMap["resources"]) {
		if m, ok := r.(map[string]any); ok {
			visible[fmt.Sprint(m["slug"])] = m
		}
	}
	meta, _ := schemaMap["resource_meta"].(map[string]any)

	for _, res := range p.GetResources() {
		dto, ok := visible[res.Slug()]
		if !ok {
			continue // hidden from the generator's user
		}
		rs := resourceSnapshot{slug: res.Slug(), model: res.GetModelName(), plural: fmt.Sprint(dto["plural_model_name"])}
		if rt, ok := res.(recordTyper); ok {
			rs.recordType = rt.RecordType()
		}
		for _, c := range asList(dto["columns"]) {
			if m, ok := c.(map[string]any); ok {
				rs.columns = append(rs.columns, fmt.Sprint(m["name"]))
			}
		}
		rm, _ := meta[res.Slug()].(map[string]any)
		rs.hasForm, _ = rm["has_form"].(bool)
		if d, ok := res.(interface {
			DescribeForm(ctx *filament.Context) []forms.Node
		}); ok && rs.hasForm {
			rs.form = d.DescribeForm(ctx)
		}
		var actionForms map[string][]forms.Node
		if d, ok := res.(interface {
			DescribeActionForms(ctx *filament.Context) map[string][]forms.Node
		}); ok {
			actionForms = d.DescribeActionForms(ctx)
		}
		table, _ := rm["table"].(map[string]any)
		for _, a := range asList(table["actions"]) {
			m, _ := a.(map[string]any)
			// Only actions that run on the server get a client method.
			if m == nil || m["custom"] != true || m["link"] == true {
				continue
			}
			name := fmt.Sprint(m["name"])
			hasForm, _ := m["has_form"].(bool)
			rs.actions = append(rs.actions, actionSnapshot{name: name, label: fmt.Sprint(m["label"]), hasForm: hasForm, form: actionForms[name]})
		}
		for _, rel := range res.GetRelations() {
			r := relationSnapshot{name: rel.Relationship(), title: rel.Title()}
			if rt, ok := rel.(recordTyper); ok {
				r.recordType = rt.RecordType()
			}
			apiSchema := rel.GetAPISchema()
			r.hasForm, _ = apiSchema["has_form"].(bool)
			if cols, ok := apiSchema["columns"]; ok {
				snap.relationColumns = append(snap.relationColumns, toGeneric(cols))
			}
			if d, ok := rel.(interface {
				DescribeForm(ctx *filament.Context) []forms.Node
			}); ok && r.hasForm {
				r.form = d.DescribeForm(ctx)
			}
			rs.relations = append(rs.relations, r)
		}
		snap.resources = append(snap.resources, rs)
	}

	for _, w := range p.GetWidgets() {
		if fw, ok := w.(interface {
			GetComponent() string
			GetProps() any
		}); ok {
			snap.widgets = append(snap.widgets, widgetSnapshot{component: fw.GetComponent(), props: fw.GetProps()})
		}
	}

	for _, pg := range p.GetPages() {
		if fp, ok := pg.(filament.FlutterPager); ok {
			snap.pages = append(snap.pages, fp.FlutterPage())
		}
	}

	snap.schema = sanitizeSchema(schemaMap)
	return snap, nil
}

func decodeJSON(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	err := dec.Decode(&v)
	return v, err
}

// toGeneric converts a value to the JSON object model (maps, lists, numbers).
func toGeneric(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	out, _ := decodeJSON(b)
	return out
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}

// Request- or data-dependent parts of the schema that must not be baked into
// generated code: they change between requests or users, and the live schema
// always provides them.
var volatileSchemaKeys = []string{
	"widgets", "widgets_count", "tenants", "active_tenant", "app_update",
	"currency", "export",
}

// sanitizeSchema strips per-user and per-request data from a schema payload.
// Permissions (abilities) are removed so that nothing is allowed until the
// live schema says so.
func sanitizeSchema(schema map[string]any) map[string]any {
	for _, k := range volatileSchemaKeys {
		delete(schema, k)
	}
	if meta, ok := schema["resource_meta"].(map[string]any); ok {
		for _, m := range meta {
			if mm, ok := m.(map[string]any); ok {
				delete(mm, "abilities")
			}
		}
	}
	for _, g := range asList(schema["navigation"]) {
		gm, _ := g.(map[string]any)
		for _, it := range asList(gm["items"]) {
			if im, ok := it.(map[string]any); ok {
				delete(im, "badge")
				delete(im, "active")
			}
		}
	}
	schema["snapshot"] = true
	return schema
}

// sanitizeForm drops options that were resolved from data (OptionsUsing):
// they belong to the live form, not to source code.
func sanitizeForm(nodes []forms.Node) any {
	g := toGeneric(nodes)
	var walk func(v any)
	walk = func(v any) {
		for _, n := range asList(v) {
			m, _ := n.(map[string]any)
			if m == nil {
				continue
			}
			if m["dynamic_options"] == true || m["remote"] == true {
				delete(m, "options")
			}
			walk(m["children"])
		}
	}
	walk(g)
	if g == nil {
		return []any{}
	}
	return g
}
