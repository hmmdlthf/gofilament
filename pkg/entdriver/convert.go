package entdriver

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Supported time layout formats for parsing form inputs.
var timeLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
}

// ConvertValue converts a raw form input (usually string) to target reflect.Type.
func ConvertValue(targetType reflect.Type, raw any) (reflect.Value, error) {
	if raw == nil {
		return reflect.Zero(targetType), nil
	}

	rawVal := reflect.ValueOf(raw)
	if rawVal.Type().AssignableTo(targetType) {
		return rawVal, nil
	}

	strVal := fmt.Sprintf("%v", raw)

	// Pointer target
	if targetType.Kind() == reflect.Pointer {
		elemType := targetType.Elem()
		elemVal, err := ConvertValue(elemType, raw)
		if err != nil {
			return reflect.Value{}, err
		}
		ptr := reflect.New(elemType)
		ptr.Elem().Set(elemVal)
		return ptr, nil
	}

	// Handle time.Time
	if targetType == reflect.TypeOf(time.Time{}) {
		if t, ok := raw.(time.Time); ok {
			return reflect.ValueOf(t), nil
		}
		for _, layout := range timeLayouts {
			if t, err := time.Parse(layout, strVal); err == nil {
				return reflect.ValueOf(t), nil
			}
		}
		return reflect.Value{}, fmt.Errorf("cannot parse %q as time.Time", strVal)
	}

	switch targetType.Kind() {
	case reflect.String:
		return reflect.ValueOf(strVal), nil

	case reflect.Bool:
		s := strings.ToLower(strings.TrimSpace(strVal))
		b := s == "1" || s == "true" || s == "on" || s == "yes"
		return reflect.ValueOf(b), nil

	case reflect.Int:
		i, err := strconv.Atoi(strVal)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(i), nil

	case reflect.Int64:
		i, err := strconv.ParseInt(strVal, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(i), nil

	case reflect.Float64:
		f, err := strconv.ParseFloat(strVal, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(f), nil

	case reflect.Float32:
		f, err := strconv.ParseFloat(strVal, 32)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(float32(f)), nil

	default:
		return reflect.Value{}, fmt.Errorf("unsupported target conversion type: %v", targetType)
	}
}
