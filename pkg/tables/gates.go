package tables

import "context"

// Feature gates hide columns and actions for requests where a server-side
// check fails, typically a per-tenant feature (filament.FeatureEnabled).
// The panel gives the table the request context with WithContext; until it
// does, gated columns and actions are left out (fail closed).
//
// A gated-out column is not rendered, not exported and not described to the
// Flutter client. A gated-out action is not rendered and FindAction does not
// return it, so it cannot be run through the API either.

type featureGated interface {
	featureGate() func(ctx context.Context) bool
}

// WithContext gives the table the request context its feature gates are
// checked against. The panel calls it for every table it builds.
func (t *Table[T]) WithContext(ctx context.Context) *Table[T] {
	t.ctx = ctx
	return t
}

// Context returns the request context set with WithContext (nil if none).
func (t *Table[T]) Context() context.Context { return t.ctx }

func (t *Table[T]) gateAllows(g featureGated) bool {
	fn := g.featureGate()
	if fn == nil {
		return true
	}
	return t.ctx != nil && fn(t.ctx)
}

func anyGated[E any](items []E) bool {
	for _, it := range items {
		if g, ok := any(it).(featureGated); ok && g.featureGate() != nil {
			return true
		}
	}
	return false
}

func (t *Table[T]) markGates() {
	t.hasGates = anyGated(t.columns) || anyGated(t.actions) || anyGated(t.headerActions) || anyGated(t.bulkActions)
}

func keepAllowed[E any](t interface{ gateAllows(featureGated) bool }, items []E) []E {
	out := items[:0:0]
	for _, it := range items {
		if g, ok := any(it).(featureGated); ok && !t.gateAllows(g) {
			continue
		}
		out = append(out, it)
	}
	return out
}

// applyGates drops the columns and actions whose gate fails for the
// table's context. Tables without gates are never written to.
func (t *Table[T]) applyGates() {
	if !t.hasGates {
		return
	}
	t.columns = keepAllowed(t, t.columns)
	t.actions = keepAllowed(t, t.actions)
	t.headerActions = keepAllowed(t, t.headerActions)
	t.bulkActions = keepAllowed(t, t.bulkActions)
	t.hasGates = false
}

// RequiresFeature shows the column only for requests where fn returns true.
func (c *BaseColumn[T]) RequiresFeature(fn func(ctx context.Context) bool) *BaseColumn[T] {
	c.gate = fn
	return c
}

func (c *BaseColumn[T]) featureGate() func(ctx context.Context) bool { return c.gate }

// RequiresFeature shows the action only for requests where fn returns true;
// otherwise it is not rendered and cannot be run.
func (a *TableAction[T]) RequiresFeature(fn func(ctx context.Context) bool) *TableAction[T] {
	a.gate = fn
	return a
}

func (a *TableAction[T]) featureGate() func(ctx context.Context) bool { return a.gate }

// RequiresFeature shows the bulk action only for requests where fn returns true.
func (b *BulkActionDef) RequiresFeature(fn func(ctx context.Context) bool) *BulkActionDef {
	b.gate = fn
	return b
}

func (b *BulkActionDef) featureGate() func(ctx context.Context) bool { return b.gate }

// RequiresFeature shows the column only for requests where fn returns true.
func (c *TextColumnDef[T]) RequiresFeature(fn func(ctx context.Context) bool) *TextColumnDef[T] {
	c.BaseColumn.RequiresFeature(fn)
	return c
}

// RequiresFeature shows the column only for requests where fn returns true.
func (c *BadgeColumnDef[T]) RequiresFeature(fn func(ctx context.Context) bool) *BadgeColumnDef[T] {
	c.BaseColumn.RequiresFeature(fn)
	return c
}

// RequiresFeature shows the column only for requests where fn returns true.
func (c *ImageColumnDef[T]) RequiresFeature(fn func(ctx context.Context) bool) *ImageColumnDef[T] {
	c.BaseColumn.RequiresFeature(fn)
	return c
}

// RequiresFeature shows the column only for requests where fn returns true.
func (c *ToggleColumnDef[T]) RequiresFeature(fn func(ctx context.Context) bool) *ToggleColumnDef[T] {
	c.BaseColumn.RequiresFeature(fn)
	return c
}

// RequiresFeature shows the column only for requests where fn returns true.
func (c *IconColumnDef[T]) RequiresFeature(fn func(ctx context.Context) bool) *IconColumnDef[T] {
	c.BaseColumn.RequiresFeature(fn)
	return c
}

// RequiresFeature shows the column only for requests where fn returns true.
func (c *ColorColumnDef[T]) RequiresFeature(fn func(ctx context.Context) bool) *ColorColumnDef[T] {
	c.BaseColumn.RequiresFeature(fn)
	return c
}

// RequiresFeature shows the column only for requests where fn returns true.
func (c *SelectColumnDef[T]) RequiresFeature(fn func(ctx context.Context) bool) *SelectColumnDef[T] {
	c.BaseColumn.RequiresFeature(fn)
	return c
}

// RequiresFeature shows the column only for requests where fn returns true.
func (c *TextInputColumnDef[T]) RequiresFeature(fn func(ctx context.Context) bool) *TextInputColumnDef[T] {
	c.BaseColumn.RequiresFeature(fn)
	return c
}

// RequiresFeature shows the column only for requests where fn returns true.
func (c *CustomColumnDef[T]) RequiresFeature(fn func(ctx context.Context) bool) *CustomColumnDef[T] {
	c.BaseColumn.RequiresFeature(fn)
	return c
}

// AllColumnNames returns the names of every column, feature-gated ones
// included, in order. Use GetColumns for the columns a request may see.
func (t *Table[T]) AllColumnNames() []string {
	out := make([]string, 0, len(t.columns))
	for _, c := range t.columns {
		out = append(out, c.GetName())
	}
	return out
}

// AllBulkActionNames returns the names of every bulk action, feature-gated
// ones included.
func (t *Table[T]) AllBulkActionNames() []string {
	out := make([]string, 0, len(t.bulkActions))
	for _, b := range t.bulkActions {
		out = append(out, b.GetName())
	}
	return out
}
