package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/golaravel/gofilament/pkg/tables"
)

type TestUser struct {
	ID        int
	Name      string
	Email     string
	Role      string
	IsActive  bool
	CreatedAt time.Time
}

func TestTableBuilderRendering(t *testing.T) {
	tbl := tables.New[*TestUser]().
		Columns(
			tables.TextColumn[*TestUser]("id").Sortable(),
			tables.TextColumn[*TestUser]("name").Searchable().Sortable().Weight("bold"),
			tables.TextColumn[*TestUser]("email").Copyable(),
			tables.BadgeColumn[*TestUser]("role").Colors(map[string]string{
				"admin":  "danger",
				"editor": "warning",
			}),
			tables.ToggleColumn[*TestUser]("is_active"),
			tables.TextColumn[*TestUser]("created_at").DateTime("2006-01-02"),
		).
		Filters(
			tables.SelectFilter("role").Options(map[string]string{
				"admin":  "Administrator",
				"editor": "Editor",
			}),
			tables.TernaryFilter("is_active").Label("Active Status"),
		).
		Actions(
			tables.ViewAction[*TestUser](),
			tables.EditAction[*TestUser](),
			tables.DeleteAction[*TestUser](),
		).
		BulkActions(
			tables.DeleteBulkAction(),
		)

	users := []*TestUser{
		{
			ID:        1,
			Name:      "Alice Admin",
			Email:     "alice@example.com",
			Role:      "admin",
			IsActive:  true,
			CreatedAt: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			Name:      "Bob Editor",
			Email:     "bob@example.com",
			Role:      "editor",
			IsActive:  false,
			CreatedAt: time.Date(2026, 2, 20, 14, 30, 0, 0, time.UTC),
		},
	}

	tbl.SetData(users, 45, 1, 10, "alice", "name", "asc", map[string]string{"role": "admin"})

	html := tbl.Render("/admin/users")

	// 1. Assert Column Values
	if !strings.Contains(html, "Alice Admin") {
		t.Errorf("Expected table to display 'Alice Admin'")
	}
	if !strings.Contains(html, "bob@example.com") {
		t.Errorf("Expected table to display 'bob@example.com'")
	}
	if !strings.Contains(html, `<span class="badge badge-danger">admin</span>`) {
		t.Errorf("Expected badge with class 'badge-danger' for admin")
	}
	if !strings.Contains(html, "2026-01-15") {
		t.Errorf("Expected formatted date '2026-01-15'")
	}

	// 2. Assert Actions
	if !strings.Contains(html, `/admin/users/1/edit`) {
		t.Errorf("Expected edit action URL '/admin/users/1/edit'")
	}
	if !strings.Contains(html, `/admin/users/1/delete`) {
		t.Errorf("Expected delete action URL '/admin/users/1/delete'")
	}

	// 3. Assert Pagination
	if !strings.Contains(html, "Showing page 1 of 5 (45 total records)") {
		t.Errorf("Expected pagination text 'Showing page 1 of 5 (45 total records)', got:\n%s", html)
	}

	// 4. Assert Bulk Action Bar
	if !strings.Contains(html, "Delete Selected") {
		t.Errorf("Expected bulk action button 'Delete Selected'")
	}
}

func TestRefinedFilterAndColumnAutoResolve(t *testing.T) {
	// 1. RefinedFilter with options and live counts
	rf := tables.RefinedFilter("type").
		Label("Partner Type").
		Options(map[string]string{
			"station": "Station Partner",
			"fleet":   "Fleet Company",
			"dealer":  "Fuel Dealer",
		}).
		Counts(map[string]int{
			"station": 24,
			"fleet":   18,
			"dealer":  5,
		})

	if rf.GetType() != tables.FilterTypeRefined {
		t.Fatalf("expected FilterTypeRefined, got %s", rf.GetType())
	}
	if !rf.IsMultiple() {
		t.Fatalf("expected RefinedFilter to be multiple-selection by default")
	}

	dto := rf.ToDTO()
	if dto.Name != "type" || dto.Label != "Partner Type" {
		t.Errorf("unexpected DTO: %+v", dto)
	}
	if len(dto.Options) != 3 {
		t.Fatalf("expected 3 options in DTO, got %d", len(dto.Options))
	}
	// Check count in options
	var stationOpt *tables.FilterOption
	for i := range dto.Options {
		if dto.Options[i].Value == "station" {
			stationOpt = &dto.Options[i]
		}
	}
	if stationOpt == nil || stationOpt.Count != 24 {
		t.Errorf("expected station option count 24, got %+v", stationOpt)
	}

	rendered := rf.Render("station,dealer")
	if !strings.Contains(rendered, "checked") {
		t.Errorf("expected checked checkbox in rendered HTML, got: %s", rendered)
	}
	if !strings.Contains(rendered, "(24)") || !strings.Contains(rendered, "(18)") {
		t.Errorf("expected count badges (24) and (18) in rendered HTML, got: %s", rendered)
	}

	// 2. Test Column auto-resolve into table filters
	tbl := tables.New[*TestUser]().
		Columns(
			tables.TextColumn[*TestUser]("name").Searchable(),
			tables.TextColumn[*TestUser]("category").Filterable(),
			tables.TextColumn[*TestUser]("type").RefinedFilter(),
		)

	filters := tbl.GetFilters()
	if len(filters) != 2 {
		t.Fatalf("expected 2 auto-resolved filters from columns, got %d", len(filters))
	}

	if filters[0].GetName() != "category" || filters[0].GetType() != tables.FilterTypeSelect {
		t.Errorf("expected select filter for category, got: %+v", filters[0])
	}
	if filters[1].GetName() != "type" || filters[1].GetType() != tables.FilterTypeRefined {
		t.Errorf("expected refined filter for type, got: %+v", filters[1])
	}
}
