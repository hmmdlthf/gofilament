package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/gofilament/pkg/widgets"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

// User represents an administrative user record.
type User struct {
	ID        string
	Name      string
	Email     string
	Role      string
	IsActive  bool
	CreatedAt time.Time
}

var (
	storeMu sync.RWMutex
	store   = map[string]*User{
		"1": {
			ID:        "1",
			Name:      "Taylor Otwell",
			Email:     "taylor@laravel.com",
			Role:      "admin",
			IsActive:  true,
			CreatedAt: time.Now().Add(-72 * time.Hour),
		},
		"2": {
			ID:        "2",
			Name:      "Dan Harrin",
			Email:     "dan@filamentphp.com",
			Role:      "editor",
			IsActive:  true,
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
		"3": {
			ID:        "3",
			Name:      "Ahmad Developer",
			Email:     "ahmad@golaravel.com",
			Role:      "admin",
			IsActive:  true,
			CreatedAt: time.Now(),
		},
	}
	nextID = 4
)

func main() {
	app := golaravel.Default()

	// 1. Build the Admin Panel
	adminPanel := filament.NewPanel("admin").
		Path("/admin").
		BrandName("GoLaravel Filament").
		DarkMode(true).
		SidebarCollapsibleOnDesktop().
		MaxContentWidth(filament.Width7xl).
		NavigationGroups([]string{
			"Access Control & Staff",
			"System Settings",
		})

	// 2. Dashboard KPI Stats Widget
	adminPanel.Widget(widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
		storeMu.RLock()
		userCount := len(store)
		storeMu.RUnlock()

		return []widgets.Stat{
			*widgets.NewStat("Total System Users", strconv.Itoa(userCount)).
				Description("+25% from last week").
				Color("success").
				Chart([]int{1, 2, 2, 3, 3, userCount}),
			*widgets.NewStat("Active Engine Port", "8080").
				Description("GoLaravel Radix Kernel").
				Color("primary").
				Chart([]int{8080, 8080, 8080}),
			*widgets.NewStat("Render Overhead", "< 0.8 ms").
				Description("Zero-allocation radix matching").
				Color("success").
				Chart([]int{5, 3, 2, 1, 1}),
		}
	}))

	// 3. Register User Resource
	userResource := filament.NewResource[*User]("users").
		ModelName("User").
		PluralModelName("Users").
		NavigationGroup("Access Control & Staff").
		NavigationIcon("heroicon-users").
		NavigationSort(1).
		NavigationBadge(func(ctx *filament.Context) string {
			storeMu.RLock()
			defer storeMu.RUnlock()
			return strconv.Itoa(len(store))
		}).
		Form(func(form *forms.Form) *forms.Form {
			return form.Schema(
				forms.Section("Personal Credentials").
					Description("Enter primary identification and authentication info.").
					Columns(2).
					Schema(
						forms.TextInput("name").
							Required().
							Placeholder("e.g. John Doe"),

						forms.TextInput("email").
							Email().
							Required().
							Placeholder("john@example.com"),

						forms.Select("role").
							Options(map[string]string{
								"admin":  "Administrator",
								"editor": "Content Editor",
								"viewer": "Read-Only Viewer",
							}).
							Default("viewer"),

						forms.Toggle("is_active").
							Label("Active Account Status"),
					),
			)
		}).
		Table(func(table *tables.Table[*User]) *tables.Table[*User] {
			return table.
				Columns(
					tables.TextColumn[*User]("id").Sortable(),
					tables.TextColumn[*User]("name").Searchable().Sortable().Weight("bold"),
					tables.TextColumn[*User]("email").Searchable().Copyable(),
					tables.BadgeColumn[*User]("role").Colors(map[string]string{
						"admin":  "danger",
						"editor": "warning",
						"viewer": "success",
					}),
					tables.ToggleColumn[*User]("is_active"),
					tables.TextColumn[*User]("created_at").DateTime().Sortable(),
				).
				Filters(
					tables.SelectFilter("role").Options(map[string]string{
						"admin":  "Administrators",
						"editor": "Content Editors",
						"viewer": "Viewers",
					}),
				).
				Actions(
					tables.ViewAction[*User](),
					tables.EditAction[*User](),
					tables.DeleteAction[*User](),
				).
				BulkActions(
					tables.DeleteBulkAction(),
				)
		}).
		Infolist(func(infolist *infolists.Infolist) *infolists.Infolist {
			return infolist.Schema(
				infolists.Section("User Overview").
					Columns(2).
					Schema(
						infolists.TextEntry("name").Weight("bold"),
						infolists.TextEntry("email").Copyable(),
						infolists.BadgeEntry("role"),
						infolists.TextEntry("created_at").DateTime(),
					),
			)
		}).
		RecordID(func(u *User) string {
			return u.ID
		}).
		OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*User, int, error) {
			storeMu.RLock()
			defer storeMu.RUnlock()
			var list []*User
			for _, u := range store {
				if search != "" && !containsIgnoreCase(u.Name, search) && !containsIgnoreCase(u.Email, search) {
					continue
				}
				if roleFilter, exists := filters["role"]; exists && roleFilter != "" && u.Role != roleFilter {
					continue
				}
				list = append(list, u)
			}
			return list, len(list), nil
		}).
		OnFind(func(ctx *filament.Context, id string) (*User, error) {
			storeMu.RLock()
			defer storeMu.RUnlock()
			if u, ok := store[id]; ok {
				return u, nil
			}
			return nil, fmt.Errorf("user #%s not found", id)
		}).
		OnCreate(func(ctx *filament.Context, data map[string]any) error {
			storeMu.Lock()
			defer storeMu.Unlock()
			id := strconv.Itoa(nextID)
			nextID++
			isActive := false
			if v, ok := data["is_active"]; ok && (v == "1" || v == "true" || v == "on") {
				isActive = true
			}
			store[id] = &User{
				ID:        id,
				Name:      fmt.Sprintf("%v", data["name"]),
				Email:     fmt.Sprintf("%v", data["email"]),
				Role:      fmt.Sprintf("%v", data["role"]),
				IsActive:  isActive,
				CreatedAt: time.Now(),
			}
			return nil
		}).
		OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
			storeMu.Lock()
			defer storeMu.Unlock()
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
				if v, exists := data["is_active"]; exists {
					u.IsActive = (v == "1" || v == "true" || v == "on")
				}
			}
			return nil
		}).
		OnDelete(func(ctx *filament.Context, id string) error {
			storeMu.Lock()
			defer storeMu.Unlock()
			delete(store, id)
			return nil
		}).
		OnDeleteBulk(func(ctx *filament.Context, ids []string) error {
			storeMu.Lock()
			defer storeMu.Unlock()
			for _, id := range ids {
				delete(store, id)
			}
			return nil
		})

	adminPanel.Resource(userResource)

	// Mount GoFilament into GoLaravel
	if err := adminPanel.Register(app); err != nil {
		log.Fatalf("Failed to register admin panel: %v", err)
	}

	// Redirect root to admin panel
	app.GET("/", func(c *gcontext.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/admin")
	})

	fmt.Println("🚀 GoFilament server starting on http://localhost:8080/admin")
	if err := app.Run(":8080"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func containsIgnoreCase(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && stringContains(s, substr))
}

func stringContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			c1 := s[i+j]
			c2 := sub[j]
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 32
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 32
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
