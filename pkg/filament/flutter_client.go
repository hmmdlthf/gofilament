package filament

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	json "github.com/goccy/go-json"
	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"

	"github.com/golaravel/gofilament/pkg/forms"
)

// ---------------------------------------------------------------------------
// In-process client generation
// ---------------------------------------------------------------------------

type clientGeneratorKey struct{}

// WithClientGenerator returns a context that signs a request in as u without
// a token. Only Go code in the same process can set it (an HTTP client cannot),
// so it is how flutterclient.GenerateFromPanel reads the panel through its own
// API with httptest. Every authorization callback still runs for u.
func WithClientGenerator(ctx context.Context, u auth.Authenticatable) context.Context {
	return context.WithValue(ctx, clientGeneratorKey{}, u)
}

func clientGeneratorUser(r *http.Request) auth.Authenticatable {
	if r == nil {
		return nil
	}
	u, _ := r.Context().Value(clientGeneratorKey{}).(auth.Authenticatable)
	return u
}

// GetResources returns the registered resources.
func (p *Panel) GetResources() []ResourceInterface { return p.resources }

// GetPages returns the registered custom pages.
func (p *Panel) GetPages() []PageInterface { return p.pages }

// RecordType returns the Go type of the resource's records.
func (r *BaseResource[T]) RecordType() reflect.Type { return reflect.TypeFor[T]() }

// RecordType returns the Go type of the relation's records.
func (rm *BaseRelationManager[T]) RecordType() reflect.Type { return reflect.TypeFor[T]() }

// DescribeActionForms returns the input forms of the row actions that have
// one, keyed by action name. Unlike the action form endpoint it needs no
// record, so a client generator can read it.
func (r *BaseResource[T]) DescribeActionForms(ctx *Context) map[string][]forms.Node {
	withSelf(ctx)
	tbl := r.table()
	if tbl == nil {
		return nil
	}
	out := map[string][]forms.Node{}
	for _, a := range tbl.GetActions() {
		if !a.IsRunnable() {
			continue
		}
		if comps := a.FormComponents(); comps != nil {
			out[a.GetName()] = forms.Describe(ctx.GoContext(), comps)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Flutter pages
// ---------------------------------------------------------------------------

// FlutterPageInfo describes an application page that a Flutter client
// implements (a point of sale screen, a scanner). The generated client has
// one required builder per page, so the app does not compile until each page
// is written.
type FlutterPageInfo struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	Icon       string `json:"icon,omitempty"`
	Group      string `json:"group,omitempty"`
	Fullscreen bool   `json:"fullscreen,omitempty"`
}

// FlutterPager is implemented by panel pages that have a Flutter screen. Any
// PageInterface can implement it; FlutterPage builds one.
type FlutterPager interface {
	FlutterPage() FlutterPageInfo
}

// FlutterPageDef is a panel page that only exists in the Flutter client.
type FlutterPageDef struct {
	info    FlutterPageInfo
	sort    int
	visible func(ctx *Context) bool
}

// FlutterPage declares a Flutter-only page. The server decides who sees it
// (Visible); the generated client hides it for everyone else.
func FlutterPage(id, label string) *FlutterPageDef {
	return &FlutterPageDef{info: FlutterPageInfo{ID: id, Label: label}}
}

func (f *FlutterPageDef) Icon(icon string) *FlutterPageDef   { f.info.Icon = icon; return f }
func (f *FlutterPageDef) Group(group string) *FlutterPageDef { f.info.Group = group; return f }
func (f *FlutterPageDef) Sort(n int) *FlutterPageDef         { f.sort = n; return f }

// Fullscreen opens the page over the whole window (no sidebar or top bar).
func (f *FlutterPageDef) Fullscreen(on ...bool) *FlutterPageDef {
	f.info.Fullscreen = len(on) == 0 || on[0]
	return f
}

// Visible restricts the page to the users fn accepts.
func (f *FlutterPageDef) Visible(fn func(ctx *Context) bool) *FlutterPageDef {
	f.visible = fn
	return f
}

func (f *FlutterPageDef) ID() string                   { return f.info.ID }
func (f *FlutterPageDef) Title() string                { return f.info.Label }
func (f *FlutterPageDef) Slug() string                 { return f.info.ID }
func (f *FlutterPageDef) FlutterPage() FlutterPageInfo { return f.info }

// NavigationItem returns nil for users who may not see the page.
func (f *FlutterPageDef) NavigationItem(ctx *Context) *NavigationItem {
	if f.visible != nil && !f.visible(ctx) {
		return nil
	}
	return &NavigationItem{Label: f.info.Label, Icon: f.info.Icon, Group: f.info.Group, Sort: f.sort}
}

// RegisterRoutes does nothing: the page is drawn by the Flutter client.
func (f *FlutterPageDef) RegisterRoutes(p *Panel, r *router.Router) {}

// clientPages lists the Flutter pages the current user may open.
func (p *Panel) clientPages(ctx *Context) []FlutterPageInfo {
	out := []FlutterPageInfo{}
	for _, pg := range p.pages {
		fp, ok := pg.(FlutterPager)
		if !ok {
			continue
		}
		item := pg.NavigationItem(ctx)
		if item == nil || (item.Visible != nil && !item.Visible(ctx)) {
			continue
		}
		out = append(out, fp.FlutterPage())
	}
	return out
}

// ---------------------------------------------------------------------------
// Typed client endpoints
// ---------------------------------------------------------------------------

// ClientEndpoint is a custom JSON endpoint under <panel>/api with Go request
// and response types, so the generated Flutter client gets a typed method for
// it. It runs behind the panel's authentication like every API route.
type ClientEndpoint struct {
	name        string
	method      string
	path        string
	description string
	reqType     reflect.Type
	resType     reflect.Type
	authorize   func(ctx *Context) bool
	handle      func(ctx *Context, c *gcontext.Context) (any, error)
}

// Endpoint declares a typed endpoint. name is the Dart method name
// (lowerCamelCase), path is relative to <panel>/api and may contain :params,
// which the handler reads with ctx.Param. GET and DELETE requests decode Req
// from the query string, other methods from the JSON body. Use struct{} for
// no request or no response body.
//
//	filament.Endpoint("salesSummary", "GET", "/pos/summary",
//		func(ctx *filament.Context, q SummaryQuery) (Summary, error) { ... })
func Endpoint[Req, Res any](name, method, path string, fn func(ctx *Context, req Req) (Res, error)) *ClientEndpoint {
	method = strings.ToUpper(method)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return &ClientEndpoint{
		name: name, method: method, path: path,
		reqType: reflect.TypeFor[Req](), resType: reflect.TypeFor[Res](),
		handle: func(ctx *Context, c *gcontext.Context) (any, error) {
			var req Req
			if method == http.MethodGet || method == http.MethodDelete {
				if err := decodeQuery(c.Request.URL.Query(), &req); err != nil {
					return nil, err
				}
			} else if c.Request.Body != nil {
				dec := json.NewDecoder(c.Request.Body)
				if err := dec.Decode(&req); err != nil && err.Error() != "EOF" {
					return nil, &endpointError{status: http.StatusBadRequest, msg: "Invalid request payload"}
				}
			}
			return fn(ctx, req)
		},
	}
}

type endpointError struct {
	status int
	msg    string
}

func (e *endpointError) Error() string   { return e.msg }
func (e *endpointError) StatusCode() int { return e.status }

// Describe sets the documentation comment of the generated Dart method.
func (e *ClientEndpoint) Describe(s string) *ClientEndpoint { e.description = s; return e }

// Authorize refuses the endpoint (403) for users fn rejects.
func (e *ClientEndpoint) Authorize(fn func(ctx *Context) bool) *ClientEndpoint {
	e.authorize = fn
	return e
}

func (e *ClientEndpoint) Name() string               { return e.name }
func (e *ClientEndpoint) Method() string             { return e.method }
func (e *ClientEndpoint) Path() string               { return e.path }
func (e *ClientEndpoint) Description() string        { return e.description }
func (e *ClientEndpoint) RequestType() reflect.Type  { return e.reqType }
func (e *ClientEndpoint) ResponseType() reflect.Type { return e.resType }

// Endpoints registers typed client endpoints.
func (p *Panel) Endpoints(es ...*ClientEndpoint) *Panel {
	p.endpoints = append(p.endpoints, es...)
	return p
}

// GetEndpoints returns the typed client endpoints.
func (p *Panel) GetEndpoints() []*ClientEndpoint { return p.endpoints }

func (p *Panel) registerEndpoints(r *router.Router, preflight gcontext.HandlerFunc) {
	for _, e := range p.endpoints {
		e := e
		path := p.path + "/api" + e.path
		r.OPTIONS(path, preflight)
		r.Handle(e.method, path, func(c *gcontext.Context) error {
			p.setCorsHeaders(c)
			ctx, err := p.AuthorizeAccess(c)
			if err != nil {
				return err
			}
			if e.authorize != nil && !e.authorize(ctx) {
				return p.apiError(c, ErrForbidden)
			}
			withSelf(ctx)
			out, err := e.handle(ctx, c)
			if err != nil {
				return p.apiError(c, err)
			}
			if e.resType.Kind() == reflect.Struct && e.resType.NumField() == 0 {
				return c.JSON(http.StatusOK, map[string]any{"success": true})
			}
			return c.JSON(http.StatusOK, out)
		})
	}
}

// decodeQuery fills the exported fields of a struct from query parameters,
// matched by their json names.
func decodeQuery(q map[string][]string, dst any) error {
	v := reflect.ValueOf(dst).Elem()
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name := f.Name
		if tag := f.Tag.Get("json"); tag != "" {
			if tag == "-" {
				continue
			}
			if n := strings.Split(tag, ",")[0]; n != "" {
				name = n
			}
		}
		vals, ok := q[name]
		if !ok || len(vals) == 0 {
			continue
		}
		if err := setFromString(v.Field(i), vals); err != nil {
			return &endpointError{status: http.StatusBadRequest, msg: "Invalid value for " + name}
		}
	}
	return nil
}

func setFromString(fv reflect.Value, vals []string) error {
	if fv.Kind() == reflect.Pointer {
		ptr := reflect.New(fv.Type().Elem())
		if err := setFromString(ptr.Elem(), vals); err != nil {
			return err
		}
		fv.Set(ptr)
		return nil
	}
	s := vals[0]
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(s)
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		if err != nil {
			return err
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return err
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		fv.SetFloat(n)
	case reflect.Slice:
		if fv.Type().Elem().Kind() != reflect.String {
			return nil
		}
		list := []string{}
		for _, v := range vals {
			list = append(list, strings.Split(v, ",")...)
		}
		fv.Set(reflect.ValueOf(list).Convert(fv.Type()))
	}
	return nil
}
