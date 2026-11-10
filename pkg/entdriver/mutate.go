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

		// Convert "first_name" -> "FirstName", "category_id" -> "CategoryID"
		pascal := snakeToPascal(k)
		method := val.MethodByName("Set" + pascal)
		if !method.IsValid() || method.Type().NumIn() != 1 {
			continue
		}
		targetType := method.Type().In(0)

		// An empty input on an optional field means "no value": clear it on
		// update builders (Clear<Field>) and leave it unset on create builders.
		// Required string fields keep the empty string so validators can reject it.
		if isEmptyInput(rawVal) {
			if clear := val.MethodByName("Clear" + pascal); clear.IsValid() && clear.Type().NumIn() == 0 {
				clear.Call(nil)
				continue
			}
			if targetType.Kind() != reflect.String || val.MethodByName("SetNillable"+pascal).IsValid() {
				continue
			}
		}

		convertedVal, err := ConvertValue(targetType, rawVal)
		if err != nil {
			return fmt.Errorf("field %s conversion error: %w", k, err)
		}
		method.Call([]reflect.Value{convertedVal})
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

// initialisms mirrors the acronyms Ent upper-cases in generated method names
// (entc/gen: category_id -> SetCategoryID, image_url -> SetImageURL).
var initialisms = map[string]string{}

func init() {
	for _, w := range []string{
		"ACL", "API", "ASCII", "AWS", "CPU", "CSS", "DNS", "EOF", "GB", "GUID",
		"HCL", "HTML", "HTTP", "HTTPS", "ID", "IP", "JSON", "KB", "LHS", "MAC",
		"MB", "QPS", "RAM", "RHS", "RPC", "SLA", "SMTP", "SQL", "SSH", "SSO",
		"TCP", "TLS", "TTL", "UDP", "UI", "UID", "URI", "URL", "UTF8", "UUID",
		"VM", "XML", "XMPP", "XSRF", "XSS",
	} {
		initialisms[strings.ToLower(w)] = w
	}
}

func isEmptyInput(v any) bool {
	if v == nil {
		return true
	}
	s, ok := v.(string)
	return ok && strings.TrimSpace(s) == ""
}

func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	var sb strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		if up, ok := initialisms[p]; ok {
			sb.WriteString(up)
			continue
		}
		sb.WriteString(strings.ToUpper(p[:1]))
		sb.WriteString(p[1:])
	}
	return sb.String()
}
