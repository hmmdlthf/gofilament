package entdriver

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"entgo.io/ent/dialect/sql"
)

// ExecuteQuery executes pagination and count on an Ent query builder using reflection.
func ExecuteQuery[T any](ctx context.Context, query any, limit, offset int) ([]T, int, error) {
	return executePage[T](ctx, query, limit, offset, true)
}

func executePage[T any](ctx context.Context, query any, limit, offset int, withCount bool) ([]T, int, error) {
	val := reflect.ValueOf(query)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return nil, 0, fmt.Errorf("query must be a non-nil pointer to an Ent query builder")
	}

	// 1. Call Count(ctx)
	total := 0
	if withCount {
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
		total = int(countResults[0].Int())
	}

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

// ListOptions describes the search, sort and filter state of a resource table.
type ListOptions struct {
	// Search is the quick-search text; SearchColumns are matched with a
	// case-insensitive "contains" and OR-ed together.
	Search        string
	SearchColumns []string

	// SortColumn / SortDirection come from the table header. SortColumns is an
	// optional allowlist; when empty any plain column identifier is accepted.
	SortColumn    string
	SortDirection string
	SortColumns   []string

	// DefaultSort ("column" or "column desc") applies when no sort is requested.
	DefaultSort string

	// Filters are equality filters (column -> value). Keys starting with "_"
	// are reserved for the framework and skipped. FilterColumns is an optional
	// allowlist; when empty any plain column identifier is accepted.
	Filters       map[string]string
	FilterColumns []string
}

var identRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func allowedColumn(col string, allow []string) bool {
	if !identRe.MatchString(col) {
		return false
	}
	if len(allow) == 0 {
		return true
	}
	for _, a := range allow {
		if a == col {
			return true
		}
	}
	return false
}

// ApplyListOptions adds WHERE and ORDER BY clauses to an Ent query builder.
// It relies only on the generated Where / Order methods, so it works with any
// entity. Builders without those methods are returned unchanged.
func ApplyListOptions(query any, o ListOptions) any {
	return ApplyListOrder(ApplyListFilters(query, o), o)
}

// ExecuteListQuery filters, counts, orders and pages an Ent query builder. The
// total is counted before ORDER BY / LIMIT are added.
func ExecuteListQuery[T any](ctx context.Context, query any, o ListOptions, limit, offset int) ([]T, int, error) {
	query = ApplyListFilters(query, o)

	val := reflect.ValueOf(query)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return nil, 0, fmt.Errorf("query must be a non-nil pointer to an Ent query builder")
	}
	total := -1
	if clone := val.MethodByName("Clone"); clone.IsValid() && clone.Type().NumIn() == 0 {
		// count on a clone so the ordered, paged query stays untouched
		countQ := clone.Call(nil)[0]
		if count := countQ.MethodByName("Count"); count.IsValid() {
			res := count.Call([]reflect.Value{reflect.ValueOf(ctx)})
			if len(res) == 2 {
				if !res[1].IsNil() {
					return nil, 0, fmt.Errorf("failed to execute Count: %w", res[1].Interface().(error))
				}
				total = int(res[0].Int())
			}
		}
	}

	query = ApplyListOrder(query, o)
	records, counted, err := executePage[T](ctx, query, limit, offset, total < 0)
	if err != nil {
		return nil, 0, err
	}
	if total < 0 {
		total = counted
	}
	return records, total, nil
}

// ApplyListFilters adds the search and equality filter predicates of o.
func ApplyListFilters(query any, o ListOptions) any {
	val := reflect.ValueOf(query)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return query
	}

	// quick search
	if s := strings.TrimSpace(o.Search); s != "" && len(o.SearchColumns) > 0 {
		cols := make([]string, 0, len(o.SearchColumns))
		for _, c := range o.SearchColumns {
			if identRe.MatchString(c) {
				cols = append(cols, c)
			}
		}
		if len(cols) > 0 {
			val = callSelectorMethod(val, "Where", func(sel *sql.Selector) {
				preds := make([]*sql.Predicate, 0, len(cols))
				for _, c := range cols {
					preds = append(preds, sql.ContainsFold(sel.C(c), s))
				}
				sel.Where(sql.Or(preds...))
			})
		}
	}

	// equality filters
	keys := make([]string, 0, len(o.Filters))
	for k := range o.Filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := o.Filters[k]
		if v == "" || strings.HasPrefix(k, "_") || !allowedColumn(k, o.FilterColumns) {
			continue
		}
		col, value := k, v
		val = callSelectorMethod(val, "Where", func(sel *sql.Selector) {
			switch strings.ToLower(value) {
			case "true":
				sel.Where(sql.EQ(sel.C(col), true))
			case "false":
				sel.Where(sql.EQ(sel.C(col), false))
			default:
				sel.Where(sql.EQ(sel.C(col), value))
			}
		})
	}

	return val.Interface()
}

// ApplyListOrder adds the ORDER BY clause of o.
func ApplyListOrder(query any, o ListOptions) any {
	val := reflect.ValueOf(query)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return query
	}

	sortCol, sortDir := o.SortColumn, o.SortDirection
	if sortCol == "" || !allowedColumn(sortCol, o.SortColumns) {
		sortCol, sortDir = "", ""
		if parts := strings.Fields(o.DefaultSort); len(parts) > 0 && identRe.MatchString(parts[0]) {
			sortCol = parts[0]
			if len(parts) > 1 {
				sortDir = parts[1]
			}
		}
	}
	if sortCol != "" {
		desc := strings.EqualFold(sortDir, "desc")
		val = callSelectorMethod(val, "Order", func(sel *sql.Selector) {
			if desc {
				sel.OrderBy(sql.Desc(sel.C(sortCol)))
			} else {
				sel.OrderBy(sql.Asc(sel.C(sortCol)))
			}
		})
	}

	return val.Interface()
}

// callSelectorMethod invokes a variadic builder method whose element type is a
// named func(*sql.Selector) (Ent predicates and order options).
func callSelectorMethod(builder reflect.Value, name string, fn func(*sql.Selector)) reflect.Value {
	m := builder.MethodByName(name)
	if !m.IsValid() {
		return builder
	}
	mt := m.Type()
	if !mt.IsVariadic() || mt.NumIn() != 1 {
		return builder
	}
	elem := mt.In(0).Elem()
	fnVal := reflect.ValueOf(fn)
	if !fnVal.Type().ConvertibleTo(elem) {
		return builder
	}
	out := m.Call([]reflect.Value{fnVal.Convert(elem)})
	if len(out) == 1 && out[0].Kind() == reflect.Pointer && !out[0].IsNil() {
		return out[0]
	}
	return builder
}
