package entdriver

import (
	"context"
	"fmt"
	"reflect"
)

// ExecuteQuery executes pagination and count on an Ent query builder using reflection.
func ExecuteQuery[T any](ctx context.Context, query any, limit, offset int) ([]T, int, error) {
	val := reflect.ValueOf(query)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return nil, 0, fmt.Errorf("query must be a non-nil pointer to an Ent query builder")
	}

	// 1. Call Count(ctx)
	countMethod := val.MethodByName("Count")
	if !countMethod.IsValid() {
		return nil, 0, fmt.Errorf("query builder does not have Count(context.Context) method")
	}

	countResults := countMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(countResults) < 2 {
		return nil, 0, fmt.Errorf("unexpected Count return values")
	}
	if !countResults[1].IsNil() {
		err := countResults[1].Interface().(error)
		return nil, 0, fmt.Errorf("failed to execute Count: %w", err)
	}
	total := int(countResults[0].Int())

	// 2. Call Limit(limit) if limit > 0
	if limit > 0 {
		limitMethod := val.MethodByName("Limit")
		if limitMethod.IsValid() {
			val = limitMethod.Call([]reflect.Value{reflect.ValueOf(limit)})[0]
		}
	}

	// 3. Call Offset(offset) if offset > 0
	if offset > 0 {
		offsetMethod := val.MethodByName("Offset")
		if offsetMethod.IsValid() {
			val = offsetMethod.Call([]reflect.Value{reflect.ValueOf(offset)})[0]
		}
	}

	// 4. Call All(ctx)
	allMethod := val.MethodByName("All")
	if !allMethod.IsValid() {
		return nil, 0, fmt.Errorf("query builder does not have All(context.Context) method")
	}

	allResults := allMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(allResults) < 2 {
		return nil, 0, fmt.Errorf("unexpected All return values")
	}
	if !allResults[1].IsNil() {
		err := allResults[1].Interface().(error)
		return nil, 0, fmt.Errorf("failed to execute All: %w", err)
	}

	rawSlice := allResults[0]
	sliceLen := rawSlice.Len()
	results := make([]T, sliceLen)

	for i := 0; i < sliceLen; i++ {
		elem := rawSlice.Index(i).Interface()
		if typed, ok := elem.(T); ok {
			results[i] = typed
		} else {
			return nil, 0, fmt.Errorf("element %d cannot be converted to type %T", i, results[i])
		}
	}

	return results, total, nil
}
