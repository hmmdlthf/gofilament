package tables

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/golaravel/golaravel/pkg/database"
)

func TestFilterValues(t *testing.T) {
	q := url.Values{
		"filter_city":         {"colombo", "kandy"},
		"filter_created_from": {"2026-01-01"},
		"filter_created_to":   {""},
		"filter_amount_from":  {""},
		"filter_amount_to":    {""},
		"filter_status":       {"open"},
		"search":              {"x"},
	}
	got := FilterValues(q)
	want := map[string]string{"city": "colombo,kandy", "created": "2026-01-01,", "created_from": "2026-01-01", "status": "open"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
}

func TestDateRangeRules(t *testing.T) {
	f := DateRangeFilter("created_at")
	if from, to := f.Bounds("2026-02-30,nope"); from != "" || to != "" {
		t.Fatalf("invalid dates kept: %q %q", from, to)
	}
	sql, args := BuildAdvancedSearchSQLFor(f.Condition("2026-01-01,2026-01-31"), map[string]string{"created_at": "created_at"}, database.SQLite)
	if sql != "(created_at >= ? AND created_at < ?)" || !reflect.DeepEqual(args, []any{"2026-01-01", "2026-02-01"}) {
		t.Fatalf("%s %v", sql, args)
	}
	if f.Condition(",") != nil {
		t.Fatal("empty range must give no condition")
	}
	day := time.Date(2026, 1, 31, 23, 0, 0, 0, time.UTC)
	if !f.Matches("2026-01-01,2026-01-31", day) || f.Matches("2026-02-01,", day) || !f.Matches("", day) {
		t.Fatal("Matches")
	}
	if !strings.Contains(f.Render("2026-01-01,"), `name="filter_created_at_from" value="2026-01-01"`) {
		t.Fatal("render")
	}
}

func TestNumberRangeRules(t *testing.T) {
	f := NumberRangeFilter("amount")
	rules := f.Rules("10,")
	if len(rules) != 1 || rules[0].Operator != "greater_or_equal" || rules[0].Value != "10" {
		t.Fatalf("%+v", rules)
	}
	if !f.Matches("10,20", 15) || f.Matches("10,20", "25") || f.Matches("abc,", 5) == false {
		t.Fatal("Matches")
	}
	if f.ToDTO().Type != FilterTypeNumberRange {
		t.Fatal("dto type")
	}
}
