package widgets

import (
	"github.com/golaravel/gofilament/pkg/filament"
)

// BaseWidget provides common sorting and grid layout properties.
type BaseWidget struct {
	visible func(ctx *filament.Context) bool
	id         string
	sort       int
	columnSpan int
	polling    string
}

// VisibleWhen shows the widget (and serves its data) only to users for
// whom fn returns true.
func (w *BaseWidget) VisibleWhen(fn func(ctx *filament.Context) bool) *BaseWidget {
	w.visible = fn
	return w
}

// CanView reports whether the widget is shown to the request's user.
func (w *BaseWidget) CanView(ctx *filament.Context) bool {
	return w.visible == nil || w.visible(ctx)
}

func (w *BaseWidget) ID() string {
	return w.id
}

func (w *BaseWidget) SetID(id string) *BaseWidget {
	w.id = id
	return w
}

// SetDefaultID gives the widget an id when it has none (Panel.Widget does
// this, so widgets without an explicit id still get their own data URL).
func (w *BaseWidget) SetDefaultID(id string) {
	if w.id == "" {
		w.id = id
	}
}

func (w *BaseWidget) Sort() int {
	return w.sort
}

func (w *BaseWidget) ColumnSpan() int {
	if w.columnSpan <= 0 {
		return 1
	}
	return w.columnSpan
}

func (w *BaseWidget) SetSort(s int) *BaseWidget {
	w.sort = s
	return w
}

func (w *BaseWidget) SetColumnSpan(span int) *BaseWidget {
	w.columnSpan = span
	return w
}

func (w *BaseWidget) PollingInterval(interval string) *BaseWidget {
	w.polling = interval
	return w
}

func (w *BaseWidget) GetPolling() string {
	return w.polling
}

// WidgetRenderer is an interface for widgets that can render to HTML.
type WidgetRenderer interface {
	filament.WidgetInterface
	Render(ctx *filament.Context) string
}
