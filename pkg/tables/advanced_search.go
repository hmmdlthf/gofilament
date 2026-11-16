package tables

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golaravel/golaravel/pkg/database"
)

// AdvancedSearchCondition defines the root container for user-constructed advanced search filters.
type AdvancedSearchCondition struct {
	Conjunction string               `json:"conjunction"` // "AND" or "OR" (default: "AND")
	Rules       []AdvancedSearchRule `json:"rules"`
}

// AdvancedSearchRule defines a single comparison predicate.
type AdvancedSearchRule struct {
	Field    string   `json:"field"`            // e.g. "bill_number", "amount", "status"
	Operator string   `json:"operator"`         // "contains", "not_contains", "equals", "not_equals", "starts_with", "ends_with", "greater_than", "greater_or_equal", "less_than", "less_or_equal", "between", "is_empty", "is_not_empty", "in", "not_in"
	Value    string   `json:"value"`            // target comparison value
	Values   []string `json:"values,omitempty"` // for range (between) or multiple values (in)
	Not      bool     `json:"not,omitempty"`    // negation flag
}

// ParseAdvancedSearch parses a JSON string into an AdvancedSearchCondition.
func ParseAdvancedSearch(rawJSON string) (*AdvancedSearchCondition, error) {
	if strings.TrimSpace(rawJSON) == "" {
		return nil, nil
	}
	var cond AdvancedSearchCondition
	if err := json.Unmarshal([]byte(rawJSON), &cond); err != nil {
		return nil, err
	}
	if cond.Conjunction == "" {
		cond.Conjunction = "AND"
	}
	cond.Conjunction = strings.ToUpper(cond.Conjunction)
	if cond.Conjunction != "OR" {
		cond.Conjunction = "AND"
	}
	return &cond, nil
}

// GetAdvancedSearch extracts and parses the advanced search condition from a filters map.
func GetAdvancedSearch(filters map[string]string) *AdvancedSearchCondition {
	if filters == nil {
		return nil
	}
	raw := filters["_advanced_search"]
	if raw == "" {
		raw = filters["advanced_search"]
	}
	if raw == "" {
		return nil
	}
	cond, _ := ParseAdvancedSearch(raw)
	return cond
}

// BuildAdvancedSearchSQL generates a parameterized SQL snippet and matching
// argument slice with PostgreSQL placeholders ($N, starting at
// startArgIndex). For SQLite or MySQL use BuildAdvancedSearchSQLFor.
// dbColumnMapping maps JSON field names to database column expressions
// (e.g. "bill_number" => "bill_number"); other fields are ignored.
func BuildAdvancedSearchSQL(cond *AdvancedSearchCondition, dbColumnMapping map[string]string, startArgIndex int) (string, []any) {
	return buildAdvancedSearch(cond, dbColumnMapping, database.Postgres, startArgIndex)
}

// BuildAdvancedSearchSQLFor is BuildAdvancedSearchSQL for any dialect:
// SQLite and MySQL get "?" placeholders (and MySQL CAST(... AS CHAR)),
// Postgres gets $1, $2, ... starting at 1. Append it to a WHERE clause
// whose earlier placeholders are already "?" (or, for Postgres, rebind
// afterwards with database.Postgres.Rebind when you build with "?").
func BuildAdvancedSearchSQLFor(cond *AdvancedSearchCondition, dbColumnMapping map[string]string, dialect database.Dialect) (string, []any) {
	return buildAdvancedSearch(cond, dbColumnMapping, dialect, 1)
}

// likeEscaper escapes LIKE wildcards in user input (with ESCAPE '!'), so
// searching for "50%" does not match everything.
var likeEscaper = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")

func buildAdvancedSearch(cond *AdvancedSearchCondition, dbColumnMapping map[string]string, dialect database.Dialect, startArgIndex int) (string, []any) {
	if cond == nil || len(cond.Rules) == 0 {
		return "", nil
	}

	var clauses []string
	var args []any
	argIdx := startArgIndex
	if argIdx < 1 {
		argIdx = 1
	}
	ph := func(v any) string {
		args = append(args, v)
		if dialect == database.Postgres {
			argIdx++
			return fmt.Sprintf("$%d", argIdx-1)
		}
		return "?"
	}
	textType := "TEXT"
	if dialect == database.MySQL {
		textType = "CHAR"
	}
	text := func(col string) string { return fmt.Sprintf("CAST(%s AS %s)", col, textType) }
	like := func(col, pattern string, not bool) string {
		op := "LIKE"
		if not {
			op = "NOT LIKE"
		}
		return fmt.Sprintf("LOWER(%s) %s LOWER(%s) ESCAPE '!'", text(col), op, ph(pattern))
	}
	list := func(rule AdvancedSearchRule, val string) []string {
		var out []string
		src := rule.Values
		if len(src) == 0 && val != "" {
			src = strings.Split(val, ",")
		}
		for _, p := range src {
			if tp := strings.TrimSpace(p); tp != "" {
				out = append(out, tp)
			}
		}
		return out
	}

	conjunction := " AND "
	if cond.Conjunction == "OR" {
		conjunction = " OR "
	}

	for _, rule := range cond.Rules {
		dbCol, ok := dbColumnMapping[rule.Field]
		if !ok || dbCol == "" {
			// Skip unrecognized or disallowed fields for security
			continue
		}

		op := strings.ToLower(strings.TrimSpace(rule.Operator))
		val := strings.TrimSpace(rule.Value)
		esc := likeEscaper.Replace(val)

		var clause string
		switch op {
		case "contains":
			clause = like(dbCol, "%"+esc+"%", false)
		case "not_contains", "does_not_contain":
			clause = like(dbCol, "%"+esc+"%", true)
		case "starts_with":
			clause = like(dbCol, esc+"%", false)
		case "ends_with":
			clause = like(dbCol, "%"+esc, false)
		case "equals", "eq":
			clause = fmt.Sprintf("%s = %s", dbCol, ph(val))
		case "not_equals", "neq":
			clause = fmt.Sprintf("%s != %s", dbCol, ph(val))
		case "greater_than", "gt":
			clause = fmt.Sprintf("%s > %s", dbCol, ph(val))
		case "greater_or_equal", "gte":
			clause = fmt.Sprintf("%s >= %s", dbCol, ph(val))
		case "less_than", "lt":
			clause = fmt.Sprintf("%s < %s", dbCol, ph(val))
		case "less_or_equal", "lte":
			clause = fmt.Sprintf("%s <= %s", dbCol, ph(val))
		case "between":
			if vals := list(rule, val); len(vals) >= 2 {
				lo := ph(vals[0])
				clause = fmt.Sprintf("%s BETWEEN %s AND %s", dbCol, lo, ph(vals[1]))
			}
		case "is_empty":
			clause = fmt.Sprintf("(%s IS NULL OR %s = '')", dbCol, text(dbCol))
		case "is_not_empty":
			clause = fmt.Sprintf("(%s IS NOT NULL AND %s != '')", dbCol, text(dbCol))
		case "in", "not_in":
			if vals := list(rule, val); len(vals) > 0 {
				phs := make([]string, len(vals))
				for i, v := range vals {
					phs[i] = ph(v)
				}
				kw := "IN"
				if op == "not_in" {
					kw = "NOT IN"
				}
				clause = fmt.Sprintf("%s %s (%s)", dbCol, kw, strings.Join(phs, ", "))
			}
		}

		if clause != "" {
			if rule.Not {
				clause = fmt.Sprintf("NOT (%s)", clause)
			}
			clauses = append(clauses, clause)
		}
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return "(" + strings.Join(clauses, conjunction) + ")", args
}
