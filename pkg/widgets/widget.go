package widgets

import (
	"github.com/golaravel/gofilament/pkg/filament"
)

// BaseWidget provides common sorting and grid layout properties.
type BaseWidget struct {
	id         string
	sort       int
	columnSpan int
	polling    string
}

func (w *BaseWidget) ID() string {
	return w.id
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

// WidgetRenderer is an interface for widgets that can render to HTML.
type WidgetRenderer interface {
	filament.WidgetInterface
	Render(ctx *filament.Context) string
}
