package filament

import (
	"encoding/json"
	"fmt"
	"html"
	"reflect"
	"strings"

	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

// RelationManager provides nested, in-place CRUD tables and modals for related models.
type RelationManager interface {
	Relationship() string
	Title() string
	GetIcon() string
	GetRecordTitleAttribute() string
	Render(ctx *Context, parentBasePath, parentID string) string

	// Direct execution handlers
	HandleList(ctx *Context, parentID string, search, sort, dir string, page, perPage int) (any, int, error)
	HandleCreate(ctx *Context, parentID string, data map[string]any) error
	HandleUpdate(ctx *Context, parentID, childID string, data map[string]any) error
	HandleDelete(ctx *Context, parentID, childID string) error

	// Schema discovery for REST API
	GetAPISchema() map[string]any
}

// BaseRelationManager provides a type-safe generic implementation of RelationManager.
type BaseRelationManager[T any] struct {
	relationship         string
	title                string
	icon                 string
	recordTitleAttribute string
	tableDef             *tables.Table[T]
	formDef              *forms.Form
	tableBuilder         func(t *tables.Table[T]) *tables.Table[T]
	formBuilder          func(f *forms.Form) *forms.Form
	idExtractor          func(record T) string

	onList   func(ctx *Context, parentID string, search, sort, dir string, page, perPage int) ([]T, int, error)
	onCreate func(ctx *Context, parentID string, data map[string]any) error
	onUpdate func(ctx *Context, parentID, childID string, data map[string]any) error
	onDelete func(ctx *Context, parentID, childID string) error
	onAttach func(ctx *Context, parentID, childID string) error
	onDetach func(ctx *Context, parentID, childID string) error
	onFind   func(ctx *Context, parentID, childID string) (T, error)

	acceptUnvalidated bool
}

// NewRelationManager creates a new typed relation manager.
func NewRelationManager[T any](relationship, title string) *BaseRelationManager[T] {
	return &BaseRelationManager[T]{
		relationship:         relationship,
		title:                title,
		recordTitleAttribute: "id",
		tableDef:             tables.New[T](),
		formDef:              forms.New(),
	}
}

func (rm *BaseRelationManager[T]) Relationship() string {
	return rm.relationship
}

func (rm *BaseRelationManager[T]) Title() string {
	return rm.title
}

func (rm *BaseRelationManager[T]) Icon(i string) *BaseRelationManager[T] {
	rm.icon = i
	return rm
}

func (rm *BaseRelationManager[T]) GetIcon() string {
	if rm.icon != "" {
		return rm.icon
	}
	return "heroicon-rectangle-stack"
}

func (rm *BaseRelationManager[T]) RecordTitleAttribute(attr ...string) *BaseRelationManager[T] {
	if len(attr) > 0 {
		rm.recordTitleAttribute = attr[0]
	}
	return rm
}

func (rm *BaseRelationManager[T]) GetRecordTitleAttribute() string {
	return rm.recordTitleAttribute
}

// Table configures the table schema.
func (rm *BaseRelationManager[T]) Table(fn ...func(t *tables.Table[T]) *tables.Table[T]) *BaseRelationManager[T] {
	if len(fn) > 0 && fn[0] != nil {
		rm.tableBuilder = fn[0]
	}
	return rm
}

// GetTable returns the configured table instance.
func (rm *BaseRelationManager[T]) GetTable() *tables.Table[T] {
	if rm.tableBuilder != nil {
		return rm.tableBuilder(rm.tableDef)
	}
	return rm.tableDef
}

// Form configures the form schema used in in-place modals.
func (rm *BaseRelationManager[T]) Form(fn func(f *forms.Form) *forms.Form) *BaseRelationManager[T] {
	rm.formBuilder = fn
	return rm
}

// RecordID sets the unique ID extractor for records.
func (rm *BaseRelationManager[T]) RecordID(fn func(record T) string) *BaseRelationManager[T] {
	rm.idExtractor = fn
	return rm
}

// OnList registers the query handler for loading related records.
func (rm *BaseRelationManager[T]) OnList(fn func(ctx *Context, parentID string, search, sort, dir string, page, perPage int) ([]T, int, error)) *BaseRelationManager[T] {
	rm.onList = fn
	return rm
}

// OnCreate registers the creation handler for in-place adding a related record.
func (rm *BaseRelationManager[T]) OnCreate(fn func(ctx *Context, parentID string, data map[string]any) error) *BaseRelationManager[T] {
	rm.onCreate = fn
	return rm
}

// OnUpdate registers the update handler for in-place editing a related record.
func (rm *BaseRelationManager[T]) OnUpdate(fn func(ctx *Context, parentID, childID string, data map[string]any) error) *BaseRelationManager[T] {
	rm.onUpdate = fn
	return rm
}

// OnDelete registers the delete handler for removing a related record.
func (rm *BaseRelationManager[T]) OnDelete(fn func(ctx *Context, parentID, childID string) error) *BaseRelationManager[T] {
	rm.onDelete = fn
	return rm
}

// OnAttach registers the attach handler for M2M associations.
func (rm *BaseRelationManager[T]) OnAttach(fn func(ctx *Context, parentID, childID string) error) *BaseRelationManager[T] {
	rm.onAttach = fn
	return rm
}

// OnDetach registers the detach handler for M2M disassociations.
func (rm *BaseRelationManager[T]) OnDetach(fn func(ctx *Context, parentID, childID string) error) *BaseRelationManager[T] {
	rm.onDetach = fn
	return rm
}

// HandleList fulfills RelationManager interface.
func (rm *BaseRelationManager[T]) HandleList(ctx *Context, parentID string, search, sort, dir string, page, perPage int) (any, int, error) {
	if rm.onList != nil {
		return rm.onList(ctx, parentID, search, sort, dir, page, perPage)
	}
	return []T{}, 0, nil
}

// HandleCreate fulfills RelationManager interface.
func (rm *BaseRelationManager[T]) HandleCreate(ctx *Context, parentID string, data map[string]any) error {
	if rm.onCreate != nil {
		data, err := rm.validateInput(ctx, parentID, data, true)
		if err != nil {
			return err
		}
		return rm.onCreate(ctx, parentID, data)
	}
	return nil
}

// HandleUpdate fulfills RelationManager interface.
func (rm *BaseRelationManager[T]) HandleUpdate(ctx *Context, parentID, childID string, data map[string]any) error {
	if rm.onUpdate != nil {
		data, err := rm.validateInput(ctx, parentID, data, false)
		if err != nil {
			return err
		}
		return rm.onUpdate(ctx, parentID, childID, data)
	}
	return nil
}

// HandleDelete fulfills RelationManager interface.
func (rm *BaseRelationManager[T]) HandleDelete(ctx *Context, parentID, childID string) error {
	if rm.onDelete != nil {
		return rm.onDelete(ctx, parentID, childID)
	}
	return nil
}

// extractID gets the string ID of record T.
func (rm *BaseRelationManager[T]) extractID(rec T) string {
	if rm.idExtractor != nil {
		return rm.idExtractor(rec)
	}
	val := reflect.ValueOf(rec)
	if val.Kind() == reflect.Pointer && !val.IsNil() {
		val = val.Elem()
	}
	if val.Kind() == reflect.Struct {
		field := val.FieldByName("ID")
		if field.IsValid() {
			return fmt.Sprintf("%v", field.Interface())
		}
		field = val.FieldByName("Id")
		if field.IsValid() {
			return fmt.Sprintf("%v", field.Interface())
		}
	} else if val.Kind() == reflect.Map {
		iter := val.MapRange()
		for iter.Next() {
			k := fmt.Sprintf("%v", iter.Key().Interface())
			if strings.EqualFold(k, "id") {
				return fmt.Sprintf("%v", iter.Value().Interface())
			}
		}
	}
	return ""
}

// Render produces the interactive in-place relation table and modals.
func (rm *BaseRelationManager[T]) Render(ctx *Context, parentBasePath, parentID string) string {
	tbl := tables.New[T]()
	if rm.tableBuilder != nil {
		tbl = rm.tableBuilder(tbl)
	}
	if rm.idExtractor != nil {
		tbl.RecordID(rm.idExtractor)
	}

	// Fetch related records
	var records []T
	total := 0
	if rm.onList != nil {
		var err error
		records, total, err = rm.onList(ctx, parentID, "", "", "asc", 1, 50)
		if err != nil {
			records = []T{}
			total = 0
		}
	}

	tbl.SetData(records, total, 1, 50, "", "", "asc", nil)

	// Configure in-place row actions on table
	relPrefix := fmt.Sprintf("%s/%s/relations/%s", parentBasePath, parentID, rm.relationship)
	createModalID := fmt.Sprintf("modal_rel_%s_create", rm.relationship)

	// In-place row actions: Edit & Delete
	tbl.Actions(
		tables.Action[T]("edit").
			Label("Edit").
			Color("primary").
			RequiresConfirmation(false),
		tables.Action[T]("delete").
			Label("Delete").
			Color("danger").
			RequiresConfirmation(true),
	)

	// Build in-place Edit modals for each loaded child record
	var editModals strings.Builder
	for _, rec := range records {
		recID := rm.extractID(rec)
		if recID == "" {
			continue
		}
		editModalID := fmt.Sprintf("modal_rel_%s_edit_%s", rm.relationship, recID)
		editActionURL := fmt.Sprintf("%s/%s/edit", relPrefix, recID)

		form := forms.New()
		if rm.formBuilder != nil {
			form = rm.formBuilder(form)
		}
		form.Action(editActionURL).Method("POST")
		form.Fill(recordToMap(rec))

		editModals.WriteString(fmt.Sprintf(`
<div id="%s" class="modal-backdrop" style="display: none;">
  <div class="modal-content">
    <div class="modal-header">
      <h3 style="font-size: 1.125rem; font-weight: 600;">Edit %s</h3>
      <button type="button" onclick="Filament.closeModal('%s')" style="background:none; border:none; font-size: 1.25rem; cursor: pointer; color: var(--gray-400);">&times;</button>
    </div>
    <div class="modal-body">
      %s
    </div>
    <div class="modal-footer">
      <button type="button" class="btn btn-secondary" onclick="Filament.closeModal('%s')">Cancel</button>
      <button type="submit" form="filament-form" class="btn btn-primary">Save Changes</button>
    </div>
  </div>
</div>`,
			editModalID,
			html.EscapeString(rm.title),
			editModalID,
			form.Render(ctx.GoContext()),
			editModalID,
		))
	}

	// Build Create Modal
	createActionURL := fmt.Sprintf("%s/create", relPrefix)
	createForm := forms.New()
	if rm.formBuilder != nil {
		createForm = rm.formBuilder(createForm)
	}
	createForm.Action(createActionURL).Method("POST")

	createModalHTML := fmt.Sprintf(`
<div id="%s" class="modal-backdrop" style="display: none;">
  <div class="modal-content">
    <div class="modal-header">
      <h3 style="font-size: 1.125rem; font-weight: 600;">New %s</h3>
      <button type="button" onclick="Filament.closeModal('%s')" style="background:none; border:none; font-size: 1.25rem; cursor: pointer; color: var(--gray-400);">&times;</button>
    </div>
    <div class="modal-body">
      %s
    </div>
    <div class="modal-footer">
      <button type="button" class="btn btn-secondary" onclick="Filament.closeModal('%s')">Cancel</button>
      <button type="submit" form="filament-form" class="btn btn-primary">Create Record</button>
    </div>
  </div>
</div>`,
		createModalID,
		html.EscapeString(rm.title),
		createModalID,
		createForm.Render(ctx.GoContext()),
		createModalID,
	)

	// Render relation card container
	return fmt.Sprintf(`
<div class="filament-card relation-manager-box" style="margin-top: 1.5rem;">
  <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; border-bottom: 1px solid var(--gray-100); padding-bottom: 0.75rem;">
    <div>
      <h3 style="font-size: 1.125rem; font-weight: 600; color: var(--gray-900);">%s</h3>
      <span style="font-size: 0.75rem; color: var(--gray-400); font-weight: 500;">%d related records</span>
    </div>
    <div>
      <button type="button" class="btn btn-primary" onclick="Filament.openModal('%s')" style="font-size: 0.8125rem; padding: 0.375rem 0.75rem;">+ New %s</button>
    </div>
  </div>
  %s
  %s
  %s
</div>`,
		html.EscapeString(rm.title),
		total,
		createModalID,
		html.EscapeString(rm.title),
		tbl.Render(relPrefix),
		createModalHTML,
		editModals.String(),
	)
}

// GetOnCreate returns the create handler (to wrap it with extra behaviour).
func (rm *BaseRelationManager[T]) GetOnCreate() func(ctx *Context, parentID string, data map[string]any) error {
	return rm.onCreate
}

// GetOnUpdate returns the update handler.
func (rm *BaseRelationManager[T]) GetOnUpdate() func(ctx *Context, parentID, childID string, data map[string]any) error {
	return rm.onUpdate
}

// GetOnDelete returns the delete handler.
func (rm *BaseRelationManager[T]) GetOnDelete() func(ctx *Context, parentID, childID string) error {
	return rm.onDelete
}

// OnFind loads one related record (used by the edit form of API clients).
func (rm *BaseRelationManager[T]) OnFind(fn func(ctx *Context, parentID, childID string) (T, error)) *BaseRelationManager[T] {
	rm.onFind = fn
	return rm
}

// HandleFind loads one related record.
func (rm *BaseRelationManager[T]) HandleFind(ctx *Context, parentID, childID string) (any, error) {
	if rm.onFind == nil {
		return nil, fmt.Errorf("record not found")
	}
	return rm.onFind(ctx, parentID, childID)
}

func (rm *BaseRelationManager[T]) formComponents() []forms.Component {
	if rm.formBuilder != nil {
		return rm.formBuilder(forms.New()).GetSchema()
	}
	if rm.formDef != nil {
		return rm.formDef.GetSchema()
	}
	return nil
}

// DescribeForm returns the relation's form as JSON nodes.
func (rm *BaseRelationManager[T]) DescribeForm(ctx *Context) []forms.Node {
	return forms.Describe(ctx.GoContext(), rm.formComponents())
}

// FormField returns a field of the relation's form by name.
func (rm *BaseRelationManager[T]) FormField(name string) forms.Component {
	return forms.FindField(rm.formComponents(), name)
}

// GetAPISchema returns schema metadata for REST/Flutter integration.
func (rm *BaseRelationManager[T]) GetAPISchema() map[string]any {
	tbl := rm.GetTable()
	columns := []tables.ColumnDTO{}
	meta := tables.TableMeta{Actions: []tables.ActionDTO{}, BulkActions: []tables.ActionDTO{}, HeaderActions: []tables.ActionDTO{}}
	if tbl != nil {
		columns = tbl.GetColumnDTOs()
		meta = tbl.Meta()
	}
	return map[string]any{
		"relationship":           rm.relationship,
		"title":                  rm.title,
		"icon":                   rm.GetIcon(),
		"record_title_attribute": rm.recordTitleAttribute,
		"columns":                columns,
		"table":                  meta,
		"has_form":               len(rm.formComponents()) > 0,
		"abilities": map[string]bool{
			"create": rm.onCreate != nil,
			"update": rm.onUpdate != nil && rm.onFind != nil,
			"delete": rm.onDelete != nil,
		},
	}
}

// recordToMap converts any struct or map into a map[string]any.
func recordToMap(rec any) map[string]any {
	if m, ok := rec.(map[string]any); ok {
		return m
	}
	bytes, err := json.Marshal(rec)
	if err == nil {
		var res map[string]any
		if err := json.Unmarshal(bytes, &res); err == nil {
			return res
		}
	}
	return make(map[string]any)
}

// validateInput applies forms.Validate with the relation form (see
// BaseResource.AcceptUnvalidatedInput for the rules).
func (rm *BaseRelationManager[T]) validateInput(ctx *Context, parentID string, data map[string]any, creating bool) (map[string]any, error) {
	comps := rm.formComponents()
	if len(comps) == 0 || rm.acceptUnvalidated {
		return data, nil
	}
	withSelf(ctx)
	withRelationParent(ctx, parentID)
	return forms.Validate(ctx.GoContext(), comps, data, creating)
}

// AcceptUnvalidatedInput turns off server-side form validation for this
// relation: handlers receive the request data as sent.
func (rm *BaseRelationManager[T]) AcceptUnvalidatedInput() *BaseRelationManager[T] {
	rm.acceptUnvalidated = true
	return rm
}
