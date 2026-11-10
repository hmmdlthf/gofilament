package entdriver

import (
	"encoding"
	"fmt"
	"math"
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

var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

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

	// JSON numbers arrive as float64: format integers without an exponent
	// (fmt prints 1000000.0 as "1e+06", which strconv.Atoi rejects).
	switch n := raw.(type) {
	case float64:
		if n == math.Trunc(n) && math.Abs(n) < 1e15 {
			strVal = strconv.FormatInt(int64(n), 10)
		} else {
			strVal = strconv.FormatFloat(n, 'f', -1, 64)
		}
	case float32:
		strVal = strconv.FormatFloat(float64(n), 'f', -1, 32)
	}

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

	// Types that parse themselves from text (uuid.UUID, net.IP, custom enums, ...)
	if targetType.Kind() != reflect.String && reflect.PointerTo(targetType).Implements(textUnmarshalerType) {
		ptr := reflect.New(targetType)
		if err := ptr.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(strVal)); err != nil {
			return reflect.Value{}, err
		}
		return ptr.Elem(), nil
	}

	switch targetType.Kind() {
	case reflect.String:
		return reflect.ValueOf(strVal).Convert(targetType), nil

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
