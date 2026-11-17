package sqlstore_test

import (
	"context"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
)

func TestRoles(t *testing.T) {
	s, path := open(t)
	ctx := context.Background()
	r := s.Roles()
	_ = r.SaveRole(ctx, filament.RoleDef{Name: "editor", Label: "Editor", Permissions: []string{"posts.update"}})
	_ = r.SaveRole(ctx, filament.RoleDef{Name: "admin", Label: "Admin", Permissions: []string{"*"}})
	_ = r.SetUserRoles(ctx, "ann", []string{"editor", "admin"})
	_ = r.SetUserRoles(ctx, "bob", []string{"editor"})

	r2 := reopen(t, path).Roles()
	roles, _ := r2.ListRoles(ctx)
	if len(roles) != 2 || roles[1].Name != "editor" || roles[1].Permissions[0] != "posts.update" {
		t.Fatalf("roles: %+v", roles)
	}
	if got, _ := r2.UserRoles(ctx, "ann"); len(got) != 2 {
		t.Fatalf("ann: %v", got)
	}
	if got, _ := r2.RoleUsers(ctx, "editor"); len(got) != 2 {
		t.Fatalf("editors: %v", got)
	}
	_ = r2.DeleteRole(ctx, "editor")
	if got, _ := r2.UserRoles(ctx, "bob"); len(got) != 0 {
		t.Fatalf("bob kept a deleted role: %v", got)
	}
}
