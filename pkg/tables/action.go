package tables

import (
	"context"
	"fmt"
	"html"

	"github.com/golaravel/gofilament/pkg/forms"
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
	visibleFunc          func(record T) bool
	urlFunc              func(record T) string
	handler              func(record T) error

	runner         ActionHandler[T]
	formBuilder    func(f *forms.Form) *forms.Form
	authorize      func(ctx context.Context, record T) bool
	successMessage string
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

func ImportAction[T any](name ...string) *TableAction[T] {
	actName := "import"
	if len(name) > 0 && name[0] != "" {
		actName = name[0]
	}
	return &TableAction[T]{
		name:  actName,
		label: "Import",
		icon:  "heroicon-arrow-up-tray",
		color: "secondary",
	}
}

func ExportAction[T any](name ...string) *TableAction[T] {
	actName := "export"
	if len(name) > 0 && name[0] != "" {
		actName = name[0]
	}
	return &TableAction[T]{
		name:  actName,
		label: "Export",
		icon:  "heroicon-arrow-down-tray",
		color: "secondary",
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
		modalDescription:     "Are you sure you want to delete this record?",
	}
}

// RestoreAction creates an action to restore a soft-deleted record back to active state.
func RestoreAction[T any]() *TableAction[T] {
	return &TableAction[T]{
		name:                 "restore",
		label:                "Restore",
		icon:                 "heroicon-arrow-uturn-left",
		color:                "success",
		requiresConfirmation: true,
		modalHeading:         "Restore Record",
		modalDescription:     "Are you sure you want to restore this record back to active status?",
	}
}

// ForceDeleteAction creates an action to permanently purge a soft-deleted record.
func ForceDeleteAction[T any]() *TableAction[T] {
	return &TableAction[T]{
		name:                 "force-delete",
		label:                "Force Delete",
		icon:                 "heroicon-trash",
		color:                "danger",
		requiresConfirmation: true,
		modalHeading:         "Permanently Delete Record",
		modalDescription:     "Are you sure you want to permanently delete this record? This action CANNOT be undone.",
	}
}

func (a *TableAction[T]) GetName() string {
	return a.name
}

func (a *TableAction[T]) GetLabel() string {
	return a.label
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

func (a *TableAction[T]) Visible(fn func(record T) bool) *TableAction[T] {
	a.visibleFunc = fn
	return a
}

func (a *TableAction[T]) IsVisible(record T) bool {
	if a.visibleFunc == nil {
		return true
	}
	return a.visibleFunc(record)
}

func (a *TableAction[T]) Render(record T, recordID string, basePath string) string {
	url := fmt.Sprintf("%s/%s/%s", basePath, recordID, a.name)
	if a.name == "restore" {
		url = fmt.Sprintf("%s/restore?id=%s", basePath, recordID)
	} else if a.name == "force-delete" {
		url = fmt.Sprintf("%s/force-delete?id=%s", basePath, recordID)
	}
	if a.urlFunc != nil {
		url = a.urlFunc(record)
	}

	btnClass := "btn-secondary"
	if a.color == "primary" {
		btnClass = "btn-primary"
	} else if a.color == "danger" {
		btnClass = "btn-danger"
	} else if a.color == "success" {
		btnClass = "btn-success"
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

// RestoreBulkAction restores all selected soft-deleted records.
func RestoreBulkAction() *BulkActionDef {
	return &BulkActionDef{
		name:                 "restore_bulk",
		label:                "Restore Selected",
		color:                "success",
		requiresConfirmation: true,
	}
}

// ForceDeleteBulkAction permanently purges all selected soft-deleted records.
func ForceDeleteBulkAction() *BulkActionDef {
	return &BulkActionDef{
		name:                 "force_delete_bulk",
		label:                "Force Delete Selected",
		color:                "danger",
		requiresConfirmation: true,
	}
}

func (b *BulkActionDef) GetName() string {
	return b.name
}

func (b *BulkActionDef) GetLabel() string {
	return b.label
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
	} else if b.color == "success" {
		btnClass = "btn-success"
	}
	confirmAttr := ""
	if b.requiresConfirmation {
		confirmAttr = `onclick="return confirm('Are you sure you want to perform this action on selected records?');"`
	}
	return fmt.Sprintf(`<button type="submit" name="bulk_action" value="%s" class="btn %s btn-sm" %s>%s</button>`,
		b.name, btnClass, confirmAttr, html.EscapeString(b.label))
}
