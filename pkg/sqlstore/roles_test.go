package sqlstore_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
)

func TestRoleBindings(t *testing.T) {
	s, path := open(t)
	ctx := context.Background()
	r := s.Roles()
	_ = r.SaveRole(ctx, filament.RoleDef{Name: "editor", Label: "Editor", Permissions: []string{"posts.update"}})
	_ = r.SaveRole(ctx, filament.RoleDef{Name: "admin", Label: "Admin", Permissions: []string{"*"}})
	for _, b := range []filament.RoleBinding{
		{UserID: "ann", Role: "admin", Panel: "*", Tenant: "*"},
		{UserID: "bob", Role: "editor", Panel: "app", Tenant: "acme"},
		{UserID: "bob", Role: "editor", Panel: "app", Tenant: "acme"}, // again: no duplicate
		{UserID: "bob", Role: "editor", Panel: "admin", Tenant: "*"},
	} {
		if err := r.AddBinding(ctx, b); err != nil {
			t.Fatal(err)
		}
	}
	r2 := reopen(t, path).Roles()
	if bs, _ := r2.ListBindings(ctx, filament.BindingFilter{UserID: "bob"}); len(bs) != 2 || bs[0].Panel != "admin" {
		t.Fatalf("bob: %+v", bs)
	}
	if bs, _ := r2.ListBindings(ctx, filament.BindingFilter{Role: "editor"}); len(bs) != 2 {
		t.Fatalf("editors: %+v", bs)
	}
	_ = r2.RemoveBinding(ctx, filament.RoleBinding{UserID: "bob", Role: "editor", Panel: "admin", Tenant: "*"})
	_ = r2.DeleteRole(ctx, "admin")
	if bs, _ := r2.ListBindings(ctx, filament.BindingFilter{}); len(bs) != 1 || bs[0].Tenant != "acme" {
		t.Fatalf("after remove/delete: %+v", bs)
	}
}

// v1.38 role assignments become bindings valid everywhere.
func TestUserRolesMigrateToBindings(t *testing.T) {
	s, path := open(t)
	db := rawDB(t, path)
	if _, err := db.Exec(`INSERT INTO gofilament_user_roles (user_id, role) VALUES ('ann', 'admin')`); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	bs, _ := s.Roles().ListBindings(context.Background(), filament.BindingFilter{UserID: "ann"})
	if len(bs) != 1 || bs[0].Panel != "*" || bs[0].Tenant != "*" {
		t.Fatalf("migrated: %+v", bs)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if bs, _ := s.Roles().ListBindings(context.Background(), filament.BindingFilter{UserID: "ann"}); len(bs) != 1 {
		t.Fatalf("migrated twice: %+v", bs)
	}
}

func rawDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
