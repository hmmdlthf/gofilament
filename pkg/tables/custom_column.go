package tables

import (
	"fmt"
	"html"
)

// CustomColumnDef is a column drawn by a component of the Flutter app
// (lib/components/cells/<component>_cell.dart). The web engine shows the
// plain value.
type CustomColumnDef[T any] struct {
	BaseColumn[T]
	component string
	props     any
}

// CustomColumn declares a column rendered by the Flutter cell component
// named component. Props (any JSON value, usually a struct or map) are sent
// to the component unchanged.
func CustomColumn[T any](name, component string) *CustomColumnDef[T] {
	return &CustomColumnDef[T]{BaseColumn: BaseColumn[T]{name: name, columnType: "custom"}, component: component}
}

func (c *CustomColumnDef[T]) Label(l string) *CustomColumnDef[T] {
	c.BaseColumn.Label(l)
	return c
}

func (c *CustomColumnDef[T]) Sortable(s ...bool) *CustomColumnDef[T] {
	c.BaseColumn.Sortable(s...)
	return c
}

func (c *CustomColumnDef[T]) Searchable(s ...bool) *CustomColumnDef[T] {
	c.BaseColumn.Searchable(s...)
	return c
}

// Props sets the values passed to the component.
func (c *CustomColumnDef[T]) Props(p any) *CustomColumnDef[T] {
	c.props = p
	return c
}

// GetComponent returns the Flutter component name.
func (c *CustomColumnDef[T]) GetComponent() string { return c.component }

// RenderCell shows the plain value in the web engine.
func (c *CustomColumnDef[T]) RenderCell(record T, rawValue any) string {
	if rawValue == nil {
		return ""
	}
	return html.EscapeString(fmt.Sprint(rawValue))
}

func (c *CustomColumnDef[T]) describeColumn(dto *ColumnDTO) {
	c.BaseColumn.describeColumn(dto)
	dto.Type = "custom"
	dto.Component = c.component
	dto.Props = c.props
}
