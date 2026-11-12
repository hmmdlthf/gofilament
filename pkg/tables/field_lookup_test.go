package tables

import "testing"

type entLike struct {
	ID         int    `json:"id,omitempty"`
	InstallID  string `json:"install_id,omitempty"`
	RemoteIP   string `json:"remote_ip,omitempty"`
	CustomerID int    `json:"customer_id,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	Renamed    string `json:"label"`
	Edges      struct{ Owner string }
}

func TestExtractFieldValueGoInitialismsAndJSONTags(t *testing.T) {
	tbl := New[*entLike]()
	rec := &entLike{ID: 3, InstallID: "inst-1", RemoteIP: "10.0.0.1", CustomerID: 42, CreatedAt: "today", Renamed: "x"}
	for name, want := range map[string]any{
		"install_id":  "inst-1",
		"remote_ip":   "10.0.0.1",
		"customer_id": 42,
		"created_at":  "today",
		"id":          3,
		"label":       "x",
		"InstallID":   "inst-1",
	} {
		if got := tbl.ExtractFieldValue(rec, name); got != want {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
	if got := tbl.ExtractFieldValue(rec, "missing"); got != nil {
		t.Errorf("missing field: got %v", got)
	}
}
