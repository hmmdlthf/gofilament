package entdriver

import (
	"context"
	"fmt"
	"reflect"

	"github.com/golaravel/gofilament/pkg/filament"
)

// Config configures an Ent entity client binding.
type Config[T any] struct {
	// Client is the Ent entity client (e.g. client.User, client.Invoice)
	Client any

	// Query is an optional custom query builder factory (defaults to Client.Query())
	Query func(ctx context.Context) any

	// ID is an optional custom ID extractor closure
	ID func(record T) string
}

// BindClient binds an Ent entity client (e.g. client.User, client.Invoice) directly with sensible defaults.
func BindClient[T any](res *filament.BaseResource[T], client any) *filament.BaseResource[T] {
	return Bind(res, Config[T]{Client: client})
}
func Bind[T any](res *filament.BaseResource[T], cfg Config[T]) *filament.BaseResource[T] {
	clientVal := reflect.ValueOf(cfg.Client)
	if clientVal.Kind() != reflect.Pointer || clientVal.IsNil() {
		panic("entdriver.Bind: Client must be a non-nil pointer to an Ent entity client")
	}

	// 1. Record ID extractor
	if cfg.ID != nil {
		res.RecordID(cfg.ID)
	} else {
		res.RecordID(func(record T) string {
			v := reflect.ValueOf(record)
			if v.Kind() == reflect.Pointer && !v.IsNil() {
				v = v.Elem()
			}
			if v.Kind() == reflect.Struct {
				f := v.FieldByName("ID")
				if f.IsValid() {
					return fmt.Sprintf("%v", f.Interface())
				}
			}
			return ""
		})
	}

	// 2. OnList Handler
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]T, int, error) {
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
		return ExecuteQuery[T](ctx.GoContext(), q, perPage, offset)
	})

	// 3. OnFind Handler
	res.OnFind(func(ctx *filament.Context, id string) (T, error) {
		var zero T
		getMethod := clientVal.MethodByName("Get")
		if !getMethod.IsValid() {
			return zero, fmt.Errorf("client does not have a Get(context.Context, id) method")
		}

		targetIDType := getMethod.Type().In(1)
		convID, err := ConvertValue(targetIDType, id)
		if err != nil {
			return zero, fmt.Errorf("failed to parse id %q: %w", id, err)
		}

		results := getMethod.Call([]reflect.Value{
			reflect.ValueOf(ctx.GoContext()),
			convID,
		})

		if !results[1].IsNil() {
			return zero, results[1].Interface().(error)
		}

		return results[0].Interface().(T), nil
	})

	// 4. OnCreate Handler
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		createMethod := clientVal.MethodByName("Create")
		if !createMethod.IsValid() {
			return fmt.Errorf("client does not have a Create() method")
		}

		builder := createMethod.Call(nil)[0].Interface()
		if err := ApplyFormData(builder, data); err != nil {
			return err
		}

		_, err := ExecuteSave(ctx.GoContext(), builder)
		return err
	})

	// 5. OnUpdate Handler
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		updateMethod := clientVal.MethodByName("UpdateOneID")
		if !updateMethod.IsValid() {
			return fmt.Errorf("client does not have an UpdateOneID(id) method")
		}

		targetIDType := updateMethod.Type().In(0)
		convID, err := ConvertValue(targetIDType, id)
		if err != nil {
			return fmt.Errorf("failed to parse id %q: %w", id, err)
		}

		builder := updateMethod.Call([]reflect.Value{convID})[0].Interface()
		if err := ApplyFormData(builder, data); err != nil {
			return err
		}

		_, err = ExecuteSave(ctx.GoContext(), builder)
		return err
	})

	// 6. OnDelete Handler
	res.OnDelete(func(ctx *filament.Context, id string) error {
		return ExecuteDeleteID(ctx.GoContext(), cfg.Client, id)
	})

	// 7. OnDeleteBulk Handler
	res.OnDeleteBulk(func(ctx *filament.Context, ids []string) error {
		for _, id := range ids {
			if err := ExecuteDeleteID(ctx.GoContext(), cfg.Client, id); err != nil {
				return err
			}
		}
		return nil
	})

	return res
}
