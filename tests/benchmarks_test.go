package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

func BenchmarkFormRendering(b *testing.B) {
	form := forms.New().
		Schema(
			forms.Section("Account Information").
				Columns(2).
				Schema(
					forms.TextInput("name").Required(),
					forms.TextInput("email").Email().Required(),
					forms.TextInput("password").Password().Required(),
					forms.Select("role").Options(map[string]string{
						"admin":  "Administrator",
						"editor": "Editor",
						"viewer": "Viewer",
					}),
					forms.Toggle("is_active"),
					forms.DatePicker("birthday"),
					forms.FileUpload("avatar"),
					forms.ColorPicker("theme_color"),
					forms.TagsInput("skills"),
					forms.CodeEditor("metadata"),
				),
		).
		Fill(map[string]any{
			"name":        "Benchmark User",
			"email":       "bench@example.com",
			"role":        "admin",
			"is_active":   true,
			"theme_color": "#10b981",
		})

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = form.Render(ctx)
	}
}

func BenchmarkTableRendering(b *testing.B) {
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
		)

	var users []*TestUser
	for i := 1; i <= 25; i++ {
		users = append(users, &TestUser{
			ID:        i,
			Name:      fmt.Sprintf("User %d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			Role:      "editor",
			IsActive:  true,
			CreatedAt: time.Now(),
		})
	}

	tbl.SetData(users, 500, 1, 25, "", "name", "asc", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tbl.Render("/admin/users")
	}
}
