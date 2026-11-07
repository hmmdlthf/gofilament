package tables

import (
	"testing"
)

func TestBuildAdvancedSearchSQL(t *testing.T) {
	mapping := map[string]string{
		"bill_number": "bill_number",
		"amount":      "total_amount",
		"status":      "status",
	}

	rawJSON := `{
		"conjunction": "AND",
		"rules": [
			{"field": "bill_number", "operator": "contains", "value": "INV-10"},
			{"field": "amount", "operator": "greater_than", "value": "50"},
			{"field": "status", "operator": "equals", "value": "completed", "not": true}
		]
	}`

	cond, err := ParseAdvancedSearch(rawJSON)
	if err != nil {
		t.Fatalf("unexpected error parsing advanced search: %v", err)
	}

	sql, args := BuildAdvancedSearchSQL(cond, mapping, 1)
	expectedSQL := "(LOWER(CAST(bill_number AS TEXT)) LIKE LOWER($1) AND total_amount > $2 AND NOT (status = $3))"
	if sql != expectedSQL {
		t.Errorf("expected SQL %q, got %q", expectedSQL, sql)
	}

	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(args))
	}
	if args[0] != "%INV-10%" {
		t.Errorf("expected args[0] to be %%INV-10%%, got %v", args[0])
	}
	if args[1] != "50" {
		t.Errorf("expected args[1] to be 50, got %v", args[1])
	}
	if args[2] != "completed" {
		t.Errorf("expected args[2] to be completed, got %v", args[2])
	}
}

func TestBuildAdvancedSearchSQL_BetweenAndIn(t *testing.T) {
	mapping := map[string]string{
		"amount": "amount",
		"status": "status",
	}

	rawJSON := `{
		"conjunction": "OR",
		"rules": [
			{"field": "amount", "operator": "between", "values": ["10", "100"]},
			{"field": "status", "operator": "in", "values": ["active", "pending"]}
		]
	}`

	cond, err := ParseAdvancedSearch(rawJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sql, args := BuildAdvancedSearchSQL(cond, mapping, 5)
	expectedSQL := "(amount BETWEEN $5 AND $6 OR status IN ($7, $8))"
	if sql != expectedSQL {
		t.Errorf("expected SQL %q, got %q", expectedSQL, sql)
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}
}

func TestGetAdvancedSearchFromFilters(t *testing.T) {
	filters := map[string]string{
		"_advanced_search": `{"conjunction":"AND","rules":[{"field":"name","operator":"contains","value":"test"}]}`,
	}
	cond := GetAdvancedSearch(filters)
	if cond == nil {
		t.Fatalf("expected non-nil condition")
	}
	if len(cond.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(cond.Rules))
	}
	if cond.Rules[0].Field != "name" {
		t.Errorf("expected field name, got %s", cond.Rules[0].Field)
	}
}
