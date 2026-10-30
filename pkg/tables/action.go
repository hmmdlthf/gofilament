package tables

import (
	"fmt"
	"html"
)

// TableAction represents an interactive operation on a single row or header.
type TableAction[T any] struct {
	name                 string
	label                string
	icon                 string
	color                string // primary, danger, warning, secondary
	requiresConfirmation bool
	modalHeading         string
	modalDescription     string
	urlFunc              func(record T) string
	handler              func(record T) error
}

func Action[T any](name string) *TableAction[T] {
	return &TableAction[T]{
		name:  name,
		label: name,
		color: "secondary",
	}
}

func ViewAction[T any]() *TableAction[T] {
	return &TableAction[T]{
		name:  "view",
		label: "View",
		icon:  "heroicon-eye",
		color: "secondary",
	}
}

func EditAction[T any]() *TableAction[T] {
	return &TableAction[T]{
		name:  "edit",
		label: "Edit",
		icon:  "heroicon-pencil-square",
		color: "primary",
	}
}

func DeleteAction[T any]() *TableAction[T] {
	return &TableAction[T]{
		name:                 "delete",
		label:                "Delete",
		icon:                 "heroicon-trash",
		color:                "danger",
		requiresConfirmation: true,
		modalHeading:         "Delete Record",
		modalDescription:     "Are you sure you want to delete this record? This action cannot be undone.",
	}
}

func (a *TableAction[T]) Label(l string) *TableAction[T] {
	a.label = l
	return a
}

func (a *TableAction[T]) Icon(i string) *TableAction[T] {
	a.icon = i
	return a
}

func (a *TableAction[T]) Color(c string) *TableAction[T] {
	a.color = c
	return a
}

func (a *TableAction[T]) RequiresConfirmation(req ...bool) *TableAction[T] {
	a.requiresConfirmation = len(req) == 0 || req[0]
	return a
}

func (a *TableAction[T]) URL(fn func(record T) string) *TableAction[T] {
	a.urlFunc = fn
	return a
}

func (a *TableAction[T]) Action(fn func(record T) error) *TableAction[T] {
	a.handler = fn
	return a
}

func (a *TableAction[T]) Render(record T, recordID string, basePath string) string {
	url := fmt.Sprintf("%s/%s/%s", basePath, recordID, a.name)
	if a.urlFunc != nil {
		url = a.urlFunc(record)
	}

	btnClass := "btn-secondary"
	if a.color == "primary" {
		btnClass = "btn-primary"
	} else if a.color == "danger" {
		btnClass = "btn-danger"
	}

	confirmAttr := ""
	if a.requiresConfirmation {
		confirmAttr = fmt.Sprintf(`onclick="return confirm('%s');"`, html.EscapeString(a.modalDescription))
	}

	return fmt.Sprintf(`<a href="%s" class="btn %s btn-sm" %s>%s</a>`,
		url, btnClass, confirmAttr, html.EscapeString(a.label))
}

// ----------------------------------------------------
// Bulk Actions
// ----------------------------------------------------

type BulkActionDef struct {
	name                 string
	label                string
	icon                 string
	color                string
	requiresConfirmation bool
	formats              []string
}

func BulkAction(name string) *BulkActionDef {
	return &BulkActionDef{
		name:  name,
		label: name,
		color: "secondary",
	}
}

func DeleteBulkAction() *BulkActionDef {
	return &BulkActionDef{
		name:                 "delete_bulk",
		label:                "Delete Selected",
		color:                "danger",
		requiresConfirmation: true,
	}
}

func ExportBulkAction() *BulkActionDef {
	return &BulkActionDef{
		name:  "export_bulk",
		label: "Export Selected",
		color: "secondary",
	}
}

func (b *BulkActionDef) Formats(formats ...string) *BulkActionDef {
	b.formats = formats
	return b
}

func (b *BulkActionDef) Label(l string) *BulkActionDef {
	b.label = l
	return b
}

func (b *BulkActionDef) Render() string {
	btnClass := "btn-secondary"
	if b.color == "danger" {
		btnClass = "btn-danger"
	}
	confirmAttr := ""
	if b.requiresConfirmation {
		confirmAttr = `onclick="return confirm('Are you sure you want to perform this action on selected records?');"`
	}
	return fmt.Sprintf(`<button type="submit" name="bulk_action" value="%s" class="btn %s btn-sm" %s>%s</button>`,
		b.name, btnClass, confirmAttr, html.EscapeString(b.label))
}
