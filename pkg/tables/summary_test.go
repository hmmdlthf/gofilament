package tables

import (
	"strings"
	"testing"
)

type order struct {
	ID     string
	Status string
	Amount float64
}

func TestCompute(t *testing.T) {
	got := Compute([]any{10, "20.5", nil, "", "x", 4.5}, []Summary{Sum, Avg, Count, Min, Max})
	if got[Sum] != 35 || got[Count] != 4 || got[Min] != 4.5 || got[Max] != 20.5 {
		t.Fatalf("%v", got)
	}
	if FormatSummaryValue(got[Avg]) != "11.67" || FormatSummaryValue(35) != "35" {
		t.Fatalf("avg %v", got[Avg])
	}
	if _, ok := Compute(nil, []Summary{Avg, Min})[Avg]; ok {
		t.Fatal("avg of nothing must be absent")
	}
}

func TestGroupedTableWithSummaries(t *testing.T) {
	tbl := New[order]().
		Columns(TextColumn[order]("Status"), TextColumn[order]("Amount").Summarize(Sum, Count)).
		Groups("Status")
	rows := []order{{"1", "open", 10}, {"2", "open", 5}, {"3", "paid", 100}}
	tbl.SetData(rows, 30, 1, 3, "lamp", "Status", "asc", map[string]string{"city": "a,b"})
	tbl.SetGroup(tbl.ResolveGroup("Status"))
	tbl.SetTotals(Summaries{"Amount": {Sum: 999, Count: 30}})
	out := tbl.Render("/admin/orders")

	for _, want := range []string{
		"Status: open", "(2)", "Status: paid", "(1)",
		"<strong>15</strong>",  // open subtotal
		"<strong>115</strong>", // page sum
		"This page", "All results", "<strong>999</strong>",
		`<option value="Status" selected>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if !strings.Contains(out, "filter_city=a%2Cb") || !strings.Contains(out, "search=lamp") || !strings.Contains(out, "group=Status") {
		t.Error("pagination link drops the list state")
	}
	if tbl.ResolveGroup("Secret") != "" || tbl.ResolveGroup("none") != "" {
		t.Error("unknown group accepted")
	}
	dto := ColumnToDTO[order](TextColumn[order]("Amount").Summarize(Avg))
	if len(dto.Summaries) != 1 || dto.Summaries[0] != Avg {
		t.Error("dto summaries")
	}
}
