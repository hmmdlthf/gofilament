package filament

import (
	"fmt"

	"github.com/golaravel/gofilament/pkg/tables"
)

// Live counts for refined (faceted) filters.
//
// Every resource list response (GET /api/resources/{slug}) carries
// "facets": {column: {value: count}} for the table's RefinedFilters. A
// column's counts apply the search and every other active filter but not the
// column's own filter, so each option shows how many records choosing it
// would add. Options a filter does not list come from these counts.

// FacetCountsFunc counts the values of column among the records matching
// search and filters (the column's own filter is already left out). Use it
// for large tables, for example with a SQL GROUP BY.
type FacetCountsFunc func(ctx *Context, column, search string, filters map[string]string) (map[string]int, error)

// MaxFacetScan is how many records the built-in counter reads per refined
// filter through OnList. When more records match, that column's counts are
// left out; register FacetCounts to count large tables.
const MaxFacetScan = 5000

// FacetCounts counts refined filter values with fn instead of reading the
// matching records through OnList.
func (r *BaseResource[T]) FacetCounts(fn FacetCountsFunc) *BaseResource[T] {
	r.facetCounts = fn
	return r
}

// ListFacets returns the live counts of each refined filter: column ->
// value -> count (used by the API's list response).
func (r *BaseResource[T]) ListFacets(ctx *Context, search string, filters map[string]string) map[string]map[string]int {
	if r.tableBuilder == nil {
		return nil
	}
	tbl := r.tableBuilder(tables.New[T]())
	var out map[string]map[string]int
	for _, f := range tbl.GetFilters() {
		if f.GetType() != tables.FilterTypeRefined {
			continue
		}
		col := f.GetColumn()
		others := make(map[string]string, len(filters))
		for k, v := range filters {
			if k != col {
				others[k] = v
			}
		}
		counts, ok := r.countFacet(ctx, tbl, col, search, others)
		if !ok {
			continue
		}
		if out == nil {
			out = map[string]map[string]int{}
		}
		out[col] = counts
	}
	return out
}

func (r *BaseResource[T]) countFacet(ctx *Context, tbl *tables.Table[T], col, search string, filters map[string]string) (map[string]int, bool) {
	if r.facetCounts != nil {
		counts, err := r.facetCounts(ctx, col, search, filters)
		return counts, err == nil
	}
	if r.onList == nil {
		return nil, false
	}
	rows, total, err := r.onList(ctx, search, "", "", 1, MaxFacetScan, filters)
	if err != nil || total > MaxFacetScan {
		return nil, false
	}
	counts := map[string]int{}
	for _, rec := range rows {
		addFacetValue(counts, tbl.ExtractFieldValue(rec, col))
	}
	return counts, true
}

// addFacetValue counts one record's value; list values count each element.
func addFacetValue(counts map[string]int, v any) {
	switch x := v.(type) {
	case nil:
	case []string:
		for _, s := range x {
			addFacetValue(counts, s)
		}
	case []any:
		for _, s := range x {
			addFacetValue(counts, s)
		}
	default:
		if s := fmt.Sprint(x); s != "" {
			counts[s]++
		}
	}
}
