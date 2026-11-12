package filament

// FlutterWidgetDef is a dashboard widget drawn by a component of the Flutter
// app (lib/components/widgets/<component>_widget.dart). Its data is computed
// on the server for each request; the web engine does not show it.
type FlutterWidgetDef struct {
	id         string
	component  string
	heading    string
	sort       int
	columnSpan int
	props      any
	data       func(ctx *Context) (any, error)
	visible    func(ctx *Context) bool
}

// FlutterWidget declares a dashboard widget drawn by the Flutter widget
// component named component.
//
//	panel.Widget(filament.FlutterWidget("sales", "sales_chart").Heading("Sales").
//		Data(func(ctx *filament.Context) (any, error) { return salesByDay(ctx) }))
func FlutterWidget(id, component string) *FlutterWidgetDef {
	return &FlutterWidgetDef{id: id, component: component, columnSpan: 1}
}

func (w *FlutterWidgetDef) Heading(h string) *FlutterWidgetDef { w.heading = h; return w }

// SortBy sets the dashboard order.
func (w *FlutterWidgetDef) SortBy(n int) *FlutterWidgetDef { w.sort = n; return w }
func (w *FlutterWidgetDef) Span(n int) *FlutterWidgetDef   { w.columnSpan = n; return w }

// Props sets static values passed to the component (the generator writes a
// typed Dart class from them).
func (w *FlutterWidgetDef) Props(p any) *FlutterWidgetDef { w.props = p; return w }

// Data computes the widget's data for the signed-in user.
func (w *FlutterWidgetDef) Data(fn func(ctx *Context) (any, error)) *FlutterWidgetDef {
	w.data = fn
	return w
}

// Visible hides the widget (and skips Data) for users fn rejects.
func (w *FlutterWidgetDef) Visible(fn func(ctx *Context) bool) *FlutterWidgetDef {
	w.visible = fn
	return w
}

func (w *FlutterWidgetDef) ID() string { return w.id }

// Sort returns the dashboard order.
func (w *FlutterWidgetDef) Sort() int { return w.sort }

func (w *FlutterWidgetDef) ColumnSpan() int { return w.columnSpan }

// GetComponent returns the Flutter component name.
func (w *FlutterWidgetDef) GetComponent() string { return w.component }

// GetProps returns the static props.
func (w *FlutterWidgetDef) GetProps() any { return w.props }

// CanView reports whether the widget is shown to the user.
func (w *FlutterWidgetDef) CanView(ctx *Context) bool { return w.visible == nil || w.visible(ctx) }

// ToWidgetDTO describes the widget and its data for the current user.
func (w *FlutterWidgetDef) ToWidgetDTO(ctx *Context) map[string]any {
	dto := map[string]any{"type": "custom", "component": w.component, "heading": w.heading}
	if w.props != nil {
		dto["props"] = w.props
	}
	if w.data != nil {
		data, err := w.data(ctx)
		if err != nil {
			dto["error"] = err.Error()
		} else {
			dto["data"] = data
		}
	}
	return dto
}
