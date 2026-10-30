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
