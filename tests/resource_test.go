package tests

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type ResourceUser struct {
	ID    string
	Name  string
	Email string
	Role  string
}

func TestResourceFullCRUDLifecycle(t *testing.T) {
	app := golaravel.New()
	panel := filament.NewPanel("admin").Path("/admin")

	// In-memory mock store
	var mu sync.RWMutex
	store := map[string]*ResourceUser{
		"1": {ID: "1", Name: "Super Admin", Email: "admin@example.com", Role: "admin"},
		"2": {ID: "2", Name: "Jane Doe", Email: "jane@example.com", Role: "editor"},
	}
	nextID := 3

	userResource := filament.NewResource[*ResourceUser]("users").
		ModelName("User").
		PluralModelName("Users").
		Form(func(form *forms.Form) *forms.Form {
			return form.Schema(
				forms.TextInput("name").Required(),
				forms.TextInput("email").Email().Required(),
				forms.Select("role").Options(map[string]string{
					"admin":  "Admin",
					"editor": "Editor",
				}),
			)
		}).
		Table(func(table *tables.Table[*ResourceUser]) *tables.Table[*ResourceUser] {
			return table.
				Columns(
					tables.TextColumn[*ResourceUser]("id").Sortable(),
					tables.TextColumn[*ResourceUser]("name").Searchable().Sortable(),
					tables.TextColumn[*ResourceUser]("email").Copyable(),
					tables.BadgeColumn[*ResourceUser]("role"),
				).
				Actions(
					tables.ViewAction[*ResourceUser](),
					tables.EditAction[*ResourceUser](),
					tables.DeleteAction[*ResourceUser](),
				)
		}).
		Infolist(func(infolist *infolists.Infolist) *infolists.Infolist {
			return infolist.Schema(
				infolists.Section("User Information").
					Columns(2).
					Schema(
						infolists.TextEntry("name").Weight("bold"),
						infolists.TextEntry("email").Copyable(),
						infolists.BadgeEntry("role"),
					),
			)
		}).
		RecordID(func(record *ResourceUser) string {
			return record.ID
		}).
		OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*ResourceUser, int, error) {
			mu.RLock()
			defer mu.RUnlock()
			var list []*ResourceUser
			for _, u := range store {
				list = append(list, u)
			}
			return list, len(list), nil
		}).
		OnFind(func(ctx *filament.Context, id string) (*ResourceUser, error) {
			mu.RLock()
			defer mu.RUnlock()
			if u, ok := store[id]; ok {
				return u, nil
			}
			return nil, fmt.Errorf("user not found")
		}).
		OnCreate(func(ctx *filament.Context, data map[string]any) error {
			mu.Lock()
			defer mu.Unlock()
			id := strconv.Itoa(nextID)
			nextID++
			store[id] = &ResourceUser{
				ID:    id,
				Name:  fmt.Sprintf("%v", data["name"]),
				Email: fmt.Sprintf("%v", data["email"]),
				Role:  fmt.Sprintf("%v", data["role"]),
			}
			return nil
		}).
		OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
			mu.Lock()
			defer mu.Unlock()
			if u, ok := store[id]; ok {
				if v, exists := data["name"]; exists {
					u.Name = fmt.Sprintf("%v", v)
				}
				if v, exists := data["email"]; exists {
					u.Email = fmt.Sprintf("%v", v)
				}
				if v, exists := data["role"]; exists {
					u.Role = fmt.Sprintf("%v", v)
				}
			}
			return nil
		}).
		OnDelete(func(ctx *filament.Context, id string) error {
			mu.Lock()
			defer mu.Unlock()
			delete(store, id)
			return nil
		}).
		OnDeleteBulk(func(ctx *filament.Context, ids []string) error {
			mu.Lock()
			defer mu.Unlock()
			for _, id := range ids {
				delete(store, id)
			}
			return nil
		})

	panel.Resource(userResource)
	_ = panel.Register(app)

	client := app.Test(t)

	// 1. List records
	client.Get("/admin/users").
		AssertOK().
		AssertSee("Super Admin").
		AssertSee("Jane Doe")

	// 2. View create form
	client.Get("/admin/users/create").
		AssertOK().
		AssertSee("Create User").
		AssertSee("Create Record")

	// 3. Store new record
	createResp := client.PostForm("/admin/users/create", map[string]string{
		"name":  "Charlie Developer",
		"email": "charlie@example.com",
		"role":  "editor",
	})
	createResp.AssertStatus(http.StatusSeeOther)

	// Verify newly created record appears in list
	client.Get("/admin/users").
		AssertOK().
		AssertSee("Charlie Developer")

	// 4. View record infolist
	client.Get("/admin/users/3").
		AssertOK().
		AssertSee("View User #3")

	// 5. View edit form
	client.Get("/admin/users/3/edit").
		AssertOK().
		AssertSee("Edit User #3")

	// 6. Update record
	updateResp := client.PostForm("/admin/users/3/edit", map[string]string{
		"name":  "Charlie Senior Dev",
		"email": "charlie.sr@example.com",
		"role":  "admin",
	})
	updateResp.AssertStatus(http.StatusSeeOther)

	// Verify updated name
	client.Get("/admin/users").
		AssertOK().
		AssertSee("Charlie Senior Dev")

	// 7. Delete record
	delResp := client.Get("/admin/users/3/delete")
	delResp.AssertStatus(http.StatusSeeOther)

	// Verify record 3 was removed
	client.Get("/admin/users").
		AssertOK().
		AssertDontSee("Charlie Senior Dev")

	// 8. Bulk delete
	bulkResp := client.PostForm("/admin/users/bulk", map[string]string{
		"bulk_action":     "delete_bulk",
		"selected_ids[]": "1",
	})
	bulkResp.AssertStatus(http.StatusSeeOther)

	// Verify record 1 was removed
	client.Get("/admin/users").
		AssertOK().
		AssertDontSee("Super Admin").
		AssertSee("Jane Doe")
}
