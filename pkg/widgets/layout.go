package widgets

import "github.com/golaravel/gofilament/pkg/filament"

// Dashboard layout. The dashboard is a grid three columns wide (one column on
// phones): a widget's span is how many of the three columns it takes, so two
// span-1 charts and a span-1 table share a row, and a span-3 widget takes a
// row of its own. Widgets are placed in Sort order, filling each row from the
// start.

// Span sets how many of the dashboard's three columns the widget takes.
func (w *StatsOverviewWidget) Span(columns int) *StatsOverviewWidget {
	w.columnSpan = columns
	return w
}

// Span sets how many of the dashboard's three columns the widget takes.
func (w *ChartWidget) Span(columns int) *ChartWidget { w.columnSpan = columns; return w }

// Span sets how many of the dashboard's three columns the widget takes.
func (w *TableWidget) Span(columns int) *TableWidget { w.columnSpan = columns; return w }

// Span sets how many of the dashboard's three columns the widget takes.
func (w *BreakdownWidget) Span(columns int) *BreakdownWidget { w.columnSpan = columns; return w }

// Span sets how many of the dashboard's three columns the widget takes.
func (w *ProgressWidget) Span(columns int) *ProgressWidget { w.columnSpan = columns; return w }

// Span sets how many of the dashboard's three columns the widget takes.
func (w *QuickActionsWidget) Span(columns int) *QuickActionsWidget { w.columnSpan = columns; return w }

// Span sets how many of the dashboard's three columns the widget takes.
func (w *BannerWidget) Span(columns int) *BannerWidget { w.columnSpan = columns; return w }

// Span sets how many of the dashboard's three columns the widget takes.
func (w *TimelineWidget) Span(columns int) *TimelineWidget { w.columnSpan = columns; return w }

// Order sets the widget's place on the dashboard (lower first).
func (w *ChartWidget) Order(n int) *ChartWidget { w.sort = n; return w }

// Order sets the widget's place on the dashboard (lower first).
func (w *StatsOverviewWidget) Order(n int) *StatsOverviewWidget { w.sort = n; return w }

// Order sets the widget's place on the dashboard (lower first).
func (w *TableWidget) Order(n int) *TableWidget { w.sort = n; return w }

// Order sets the widget's place on the dashboard (lower first).
func (w *BreakdownWidget) Order(n int) *BreakdownWidget { w.sort = n; return w }

// Order sets the widget's place on the dashboard (lower first).
func (w *ProgressWidget) Order(n int) *ProgressWidget { w.sort = n; return w }

// Order sets the widget's place on the dashboard (lower first).
func (w *QuickActionsWidget) Order(n int) *QuickActionsWidget { w.sort = n; return w }

// Order sets the widget's place on the dashboard (lower first).
func (w *BannerWidget) Order(n int) *BannerWidget { w.sort = n; return w }

// Order sets the widget's place on the dashboard (lower first).
func (w *TimelineWidget) Order(n int) *TimelineWidget { w.sort = n; return w }

// Key gives the widget a fixed id (its data URL is /widgets/<id>/data).
func (w *StatsOverviewWidget) Key(id string) *StatsOverviewWidget { w.id = id; return w }

// Key gives the widget a fixed id.
func (w *TableWidget) Key(id string) *TableWidget { w.id = id; return w }

// Key gives the widget a fixed id.
func (w *BreakdownWidget) Key(id string) *BreakdownWidget { w.id = id; return w }

// Key gives the widget a fixed id.
func (w *ProgressWidget) Key(id string) *ProgressWidget { w.id = id; return w }

// Key gives the widget a fixed id.
func (w *QuickActionsWidget) Key(id string) *QuickActionsWidget { w.id = id; return w }

// Key gives the widget a fixed id.
func (w *BannerWidget) Key(id string) *BannerWidget { w.id = id; return w }

// Visible shows the widget only to users for whom fn returns true.
func (w *StatsOverviewWidget) Visible(fn func(ctx *Ctx) bool) *StatsOverviewWidget {
	w.visible = fn
	return w
}

// Visible shows the widget only to users for whom fn returns true.
func (w *ChartWidget) Visible(fn func(ctx *Ctx) bool) *ChartWidget { w.visible = fn; return w }

// Visible shows the widget only to users for whom fn returns true.
func (w *TableWidget) Visible(fn func(ctx *Ctx) bool) *TableWidget { w.visible = fn; return w }

// Visible shows the widget only to users for whom fn returns true.
func (w *BreakdownWidget) Visible(fn func(ctx *Ctx) bool) *BreakdownWidget { w.visible = fn; return w }

// Visible shows the widget only to users for whom fn returns true.
func (w *ProgressWidget) Visible(fn func(ctx *Ctx) bool) *ProgressWidget { w.visible = fn; return w }

// Visible shows the widget only to users for whom fn returns true.
func (w *QuickActionsWidget) Visible(fn func(ctx *Ctx) bool) *QuickActionsWidget {
	w.visible = fn
	return w
}

// Visible shows the widget only to users for whom fn returns true.
func (w *BannerWidget) Visible(fn func(ctx *Ctx) bool) *BannerWidget { w.visible = fn; return w }

// Ctx is the request context widget callbacks receive.
type Ctx = filament.Context
