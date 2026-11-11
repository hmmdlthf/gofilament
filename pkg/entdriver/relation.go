package entdriver

import (
	"context"
	"fmt"
	"reflect"

	"github.com/golaravel/gofilament/pkg/filament"
)

// RelationConfig binds a relation manager to the Ent client of the child entity.
type RelationConfig[T any] struct {
	// Client is the child's Ent client (e.g. client.ProductDetail).
	Client any
	// ParentColumn is the child's foreign key column to the parent (e.g. "product_id").
	ParentColumn string

	// Query optionally builds the list query, e.g. to eager-load edges
	// (client.UserRole.Query().WithRole()). Defaults to Client.Query().
	Query func(ctx context.Context) any

	SearchColumns []string
	SortColumns   []string
	DefaultSort   string
}

// BindRelation wires a relation manager to Ent: rows are listed, created,
// changed and deleted only under the given parent. A child id that belongs to
// a different parent is treated as not found.
func BindRelation[T any](rm *filament.BaseRelationManager[T], cfg RelationConfig[T]) *filament.BaseRelationManager[T] {
	clientVal := reflect.ValueOf(cfg.Client)
	if clientVal.Kind() != reflect.Pointer || clientVal.IsNil() {
		panic("entdriver.BindRelation: Client must be a non-nil pointer to an Ent entity client")
	}
	if !identRe.MatchString(cfg.ParentColumn) {
		panic("entdriver.BindRelation: ParentColumn must be a column name")
	}
	parentField := snakeToPascal(cfg.ParentColumn)

	rm.RecordID(func(record T) string {
		v := reflect.ValueOf(record)
		if v.Kind() == reflect.Pointer && !v.IsNil() {
			v = v.Elem()
		}
		if v.Kind() == reflect.Struct {
			if f := v.FieldByName("ID"); f.IsValid() {
				return fmt.Sprintf("%v", f.Interface())
			}
		}
		return ""
	})

	// find loads the child and verifies it hangs off parentID.
	find := func(ctx *filament.Context, parentID, childID string) (T, error) {
		var zero T
		getMethod := clientVal.MethodByName("Get")
		if !getMethod.IsValid() {
			return zero, fmt.Errorf("client does not have a Get(context.Context, id) method")
		}
		convID, err := ConvertValue(getMethod.Type().In(1), childID)
		if err != nil {
			return zero, fmt.Errorf("record not found")
		}
		out := getMethod.Call([]reflect.Value{reflect.ValueOf(ctx.GoContext()), convID})
		if !out[1].IsNil() {
			return zero, out[1].Interface().(error)
		}
		record, ok := out[0].Interface().(T)
		if !ok {
			return zero, fmt.Errorf("record not found")
		}
		v := reflect.ValueOf(record)
		if v.Kind() == reflect.Pointer {
			v = v.Elem()
		}
		f := v.FieldByName(parentField)
		if !f.IsValid() {
			return zero, fmt.Errorf("entity has no %s field", parentField)
		}
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				return zero, fmt.Errorf("record not found")
			}
			f = f.Elem()
		}
		if fmt.Sprintf("%v", f.Interface()) != parentID {
			return zero, fmt.Errorf("record not found")
		}
		return record, nil
	}

	rm.OnFind(find)

	rm.OnList(func(ctx *filament.Context, parentID string, search, sort, dir string, page, perPage int) ([]T, int, error) {
		var q any
		if cfg.Query != nil {
			q = cfg.Query(ctx.GoContext())
		} else {
			queryMethod := clientVal.MethodByName("Query")
			if !queryMethod.IsValid() {
				return nil, 0, fmt.Errorf("client does not have a Query() method")
			}
			q = queryMethod.Call(nil)[0].Interface()
		}
		offset := (page - 1) * perPage
		if offset < 0 {
			offset = 0
		}
		return ExecuteListQuery[T](ctx.GoContext(), q, ListOptions{
			Search:        search,
			SearchColumns: cfg.SearchColumns,
			SortColumn:    sort,
			SortDirection: dir,
			SortColumns:   cfg.SortColumns,
			DefaultSort:   cfg.DefaultSort,
			Filters:       map[string]string{cfg.ParentColumn: parentID},
			FilterColumns: []string{cfg.ParentColumn},
		}, perPage, offset)
	})

	rm.OnCreate(func(ctx *filament.Context, parentID string, data map[string]any) error {
		createMethod := clientVal.MethodByName("Create")
		if !createMethod.IsValid() {
			return fmt.Errorf("client does not have a Create() method")
		}
		data[cfg.ParentColumn] = parentID // never trust a parent id from the payload
		builder := createMethod.Call(nil)[0].Interface()
		if err := ApplyFormData(builder, data); err != nil {
			return err
		}
		_, err := ExecuteSave(ctx.GoContext(), builder)
		return err
	})

	rm.OnUpdate(func(ctx *filament.Context, parentID, childID string, data map[string]any) error {
		if _, err := find(ctx, parentID, childID); err != nil {
			return err
		}
		updateMethod := clientVal.MethodByName("UpdateOneID")
		if !updateMethod.IsValid() {
			return fmt.Errorf("client does not have an UpdateOneID(id) method")
		}
		convID, err := ConvertValue(updateMethod.Type().In(0), childID)
		if err != nil {
			return err
		}
		delete(data, cfg.ParentColumn) // a row cannot be moved to another parent
		builder := updateMethod.Call([]reflect.Value{convID})[0].Interface()
		if err := ApplyFormData(builder, data); err != nil {
			return err
		}
		_, err = ExecuteSave(ctx.GoContext(), builder)
		return err
	})

	rm.OnDelete(func(ctx *filament.Context, parentID, childID string) error {
		if _, err := find(ctx, parentID, childID); err != nil {
			return err
		}
		return ExecuteDeleteID(ctx.GoContext(), cfg.Client, childID)
	})

	return rm
}
