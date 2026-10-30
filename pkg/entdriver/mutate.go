package entdriver

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// ApplyFormData applies fields from form data map onto an Ent Create or Update builder.
func ApplyFormData(builder any, data map[string]any) error {
	val := reflect.ValueOf(builder)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return fmt.Errorf("builder must be a non-nil pointer")
	}

	for k, rawVal := range data {
		// Ignore internal form fields like csrf token or _live_field
		if strings.HasPrefix(k, "_") || k == "id" {
			continue
		}

		// Convert "first_name" -> "FirstName"
		methodName := "Set" + snakeToPascal(k)
		method := val.MethodByName(methodName)

		if method.IsValid() && method.Type().NumIn() == 1 {
			targetType := method.Type().In(0)
			convertedVal, err := ConvertValue(targetType, rawVal)
			if err != nil {
				return fmt.Errorf("field %s conversion error: %w", k, err)
			}
			method.Call([]reflect.Value{convertedVal})
		}
	}

	return nil
}

// ExecuteSave calls Save(ctx) on an Ent Create or UpdateOne builder.
func ExecuteSave(ctx context.Context, builder any) (any, error) {
	val := reflect.ValueOf(builder)
	saveMethod := val.MethodByName("Save")
	if !saveMethod.IsValid() {
		return nil, fmt.Errorf("builder does not have a Save(context.Context) method")
	}

	results := saveMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(results) < 2 {
		return nil, fmt.Errorf("unexpected Save return values")
	}

	if !results[1].IsNil() {
		err := results[1].Interface().(error)
		return nil, err
	}

	return results[0].Interface(), nil
}

// ExecuteDeleteID calls DeleteOneID(id).Exec(ctx) on an Ent client.
func ExecuteDeleteID(ctx context.Context, client any, id string) error {
	val := reflect.ValueOf(client)
	deleteMethod := val.MethodByName("DeleteOneID")
	if !deleteMethod.IsValid() {
		return fmt.Errorf("client does not have DeleteOneID method")
	}

	// Determine expected ID parameter type (int, int64, string, etc.)
	targetType := deleteMethod.Type().In(0)
	convID, err := ConvertValue(targetType, id)
	if err != nil {
		return fmt.Errorf("failed to convert id %q: %w", id, err)
	}

	builder := deleteMethod.Call([]reflect.Value{convID})[0]
	execMethod := builder.MethodByName("Exec")
	if !execMethod.IsValid() {
		return fmt.Errorf("delete builder does not have Exec method")
	}

	results := execMethod.Call([]reflect.Value{reflect.ValueOf(ctx)})
	if len(results) > 0 && !results[0].IsNil() {
		return results[0].Interface().(error)
	}

	return nil
}

func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	var sb strings.Builder
	for _, p := range parts {
		if len(p) > 0 {
			sb.WriteString(strings.ToUpper(p[:1]))
			sb.WriteString(p[1:])
		}
	}
	return sb.String()
}
