package tables

import (
	"encoding/json"
	"fmt"
	"strings"
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

// BuildAdvancedSearchSQL generates a parameterized SQL snippet and matching argument slice.
// dbColumnMapping maps JSON field names to database column expressions (e.g. "bill_number" => "bill_number").
// startArgIndex specifies the initial $N placeholder index (PostgreSQL style: $1, $2...).
func BuildAdvancedSearchSQL(cond *AdvancedSearchCondition, dbColumnMapping map[string]string, startArgIndex int) (string, []any) {
	if cond == nil || len(cond.Rules) == 0 {
		return "", nil
	}

	var clauses []string
	var args []any
	argIdx := startArgIndex
	if argIdx < 1 {
		argIdx = 1
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

		var clause string
		switch op {
		case "contains":
			clause = fmt.Sprintf("LOWER(CAST(%s AS TEXT)) LIKE LOWER($%d)", dbCol, argIdx)
			args = append(args, "%"+val+"%")
			argIdx++

		case "not_contains", "does_not_contain":
			clause = fmt.Sprintf("LOWER(CAST(%s AS TEXT)) NOT LIKE LOWER($%d)", dbCol, argIdx)
			args = append(args, "%"+val+"%")
			argIdx++

		case "starts_with":
			clause = fmt.Sprintf("LOWER(CAST(%s AS TEXT)) LIKE LOWER($%d)", dbCol, argIdx)
			args = append(args, val+"%")
			argIdx++

		case "ends_with":
			clause = fmt.Sprintf("LOWER(CAST(%s AS TEXT)) LIKE LOWER($%d)", dbCol, argIdx)
			args = append(args, "%"+val)
			argIdx++

		case "equals", "eq":
			clause = fmt.Sprintf("%s = $%d", dbCol, argIdx)
			args = append(args, val)
			argIdx++

		case "not_equals", "neq":
			clause = fmt.Sprintf("%s != $%d", dbCol, argIdx)
			args = append(args, val)
			argIdx++

		case "greater_than", "gt":
			clause = fmt.Sprintf("%s > $%d", dbCol, argIdx)
			args = append(args, val)
			argIdx++

		case "greater_or_equal", "gte":
			clause = fmt.Sprintf("%s >= $%d", dbCol, argIdx)
			args = append(args, val)
			argIdx++

		case "less_than", "lt":
			clause = fmt.Sprintf("%s < $%d", dbCol, argIdx)
			args = append(args, val)
			argIdx++

		case "less_or_equal", "lte":
			clause = fmt.Sprintf("%s <= $%d", dbCol, argIdx)
			args = append(args, val)
			argIdx++

		case "between":
			minVal := val
			maxVal := ""
			if len(rule.Values) >= 2 {
				minVal = strings.TrimSpace(rule.Values[0])
				maxVal = strings.TrimSpace(rule.Values[1])
			} else if strings.Contains(val, ",") {
				parts := strings.SplitN(val, ",", 2)
				minVal = strings.TrimSpace(parts[0])
				maxVal = strings.TrimSpace(parts[1])
			}
			if minVal != "" && maxVal != "" {
				clause = fmt.Sprintf("%s BETWEEN $%d AND $%d", dbCol, argIdx, argIdx+1)
				args = append(args, minVal, maxVal)
				argIdx += 2
			}

		case "is_empty":
			clause = fmt.Sprintf("(%s IS NULL OR CAST(%s AS TEXT) = '')", dbCol, dbCol)

		case "is_not_empty":
			clause = fmt.Sprintf("(%s IS NOT NULL AND CAST(%s AS TEXT) != '')", dbCol, dbCol)

		case "in":
			var inVals []string
			if len(rule.Values) > 0 {
				inVals = rule.Values
			} else if val != "" {
				for _, p := range strings.Split(val, ",") {
					if tp := strings.TrimSpace(p); tp != "" {
						inVals = append(inVals, tp)
					}
				}
			}
			if len(inVals) > 0 {
				var placeholders []string
				for _, iv := range inVals {
					placeholders = append(placeholders, fmt.Sprintf("$%d", argIdx))
					args = append(args, iv)
					argIdx++
				}
				clause = fmt.Sprintf("%s IN (%s)", dbCol, strings.Join(placeholders, ", "))
			}

		case "not_in":
			var inVals []string
			if len(rule.Values) > 0 {
				inVals = rule.Values
			} else if val != "" {
				for _, p := range strings.Split(val, ",") {
					if tp := strings.TrimSpace(p); tp != "" {
						inVals = append(inVals, tp)
					}
				}
			}
			if len(inVals) > 0 {
				var placeholders []string
				for _, iv := range inVals {
					placeholders = append(placeholders, fmt.Sprintf("$%d", argIdx))
					args = append(args, iv)
					argIdx++
				}
				clause = fmt.Sprintf("%s NOT IN (%s)", dbCol, strings.Join(placeholders, ", "))
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
