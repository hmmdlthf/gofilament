package tables

import (
	"database/sql"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/golaravel/golaravel/pkg/database"
)

// Every operator runs against a real SQLite database with "?" placeholders.
func TestAdvancedSearchRunsOnSQLite(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "a.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE bills (id INTEGER PRIMARY KEY, number TEXT, amount INTEGER, status TEXT, note TEXT)`); err != nil {
		t.Fatal(err)
	}
	for _, r := range [][]any{
		{1, "INV-100", 40, "paid", "50% off"},
		{2, "INV-101", 75, "open", ""},
		{3, "CR-7", 120, "open", nil},
		{4, "INV_9", 10, "void", "5000 off"},
	} {
		if _, err := db.Exec(`INSERT INTO bills VALUES (?, ?, ?, ?, ?)`, r...); err != nil {
			t.Fatal(err)
		}
	}
	cols := map[string]string{"number": "number", "amount": "amount", "status": "status", "note": "note"}

	cases := []struct {
		json string
		want string
	}{
		{`{"rules":[{"field":"number","operator":"contains","value":"inv"}]}`, "1,2,4"},
		{`{"rules":[{"field":"number","operator":"starts_with","value":"INV_"}]}`, "4"}, // "_" is literal
		{`{"rules":[{"field":"note","operator":"contains","value":"50%"}]}`, "1"},       // "%" is literal
		{`{"rules":[{"field":"number","operator":"ends_with","value":"7"}]}`, "3"},
		{`{"rules":[{"field":"number","operator":"not_contains","value":"inv"}]}`, "3"},
		{`{"rules":[{"field":"status","operator":"equals","value":"open"}]}`, "2,3"},
		{`{"rules":[{"field":"status","operator":"not_equals","value":"open"}]}`, "1,4"},
		{`{"rules":[{"field":"amount","operator":"greater_than","value":"75"}]}`, "3"},
		{`{"rules":[{"field":"amount","operator":"less_or_equal","value":"40"}]}`, "1,4"},
		{`{"rules":[{"field":"amount","operator":"between","value":"20, 100"}]}`, "1,2"},
		{`{"rules":[{"field":"status","operator":"in","values":["paid","void"]}]}`, "1,4"},
		{`{"rules":[{"field":"status","operator":"not_in","value":"paid, void"}]}`, "2,3"},
		{`{"rules":[{"field":"note","operator":"is_empty"}]}`, "2,3"},
		{`{"rules":[{"field":"note","operator":"is_not_empty"}]}`, "1,4"},
		{`{"conjunction":"OR","rules":[{"field":"status","operator":"equals","value":"void"},{"field":"amount","operator":"gt","value":"100"}]}`, "3,4"},
		{`{"rules":[{"field":"status","operator":"equals","value":"open"},{"field":"amount","operator":"lt","value":"100"}]}`, "2"},
		{`{"rules":[{"field":"status","operator":"equals","value":"open","not":true}]}`, "1,4"},
		{`{"rules":[{"field":"password","operator":"equals","value":"x"}]}`, "1,2,3,4"}, // unknown field ignored
	}
	for _, c := range cases {
		cond, err := ParseAdvancedSearch(c.json)
		if err != nil {
			t.Fatal(err)
		}
		where, args := BuildAdvancedSearchSQLFor(cond, cols, database.SQLite)
		if strings.Contains(where, "$") {
			t.Fatalf("SQLite query has $ placeholders: %s", where)
		}
		q := "SELECT id FROM bills"
		if where != "" {
			q += " WHERE " + where
		}
		rows, err := db.Query(q, args...)
		if err != nil {
			t.Fatalf("%s\n%s: %v", c.json, q, err)
		}
		var ids []string
		for rows.Next() {
			var id string
			_ = rows.Scan(&id)
			ids = append(ids, id)
		}
		rows.Close()
		sort.Strings(ids)
		if got := strings.Join(ids, ","); got != c.want {
			t.Errorf("%s\n%s %v\n got %s, want %s", c.json, where, args, got, c.want)
		}
	}
}

func TestAdvancedSearchDialectSyntax(t *testing.T) {
	cond, _ := ParseAdvancedSearch(`{"rules":[{"field":"n","operator":"contains","value":"a"},{"field":"n","operator":"in","values":["x","y"]}]}`)
	cols := map[string]string{"n": "name"}
	my, _ := BuildAdvancedSearchSQLFor(cond, cols, database.MySQL)
	if want := "(LOWER(CAST(name AS CHAR)) LIKE LOWER(?) ESCAPE '!' AND name IN (?, ?))"; my != want {
		t.Errorf("mysql: %s", my)
	}
	pg, _ := BuildAdvancedSearchSQLFor(cond, cols, database.Postgres)
	if want := "(LOWER(CAST(name AS TEXT)) LIKE LOWER($1) ESCAPE '!' AND name IN ($2, $3))"; pg != want {
		t.Errorf("postgres: %s", pg)
	}
}
