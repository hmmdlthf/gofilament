package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sync"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/auth"
)

// demoPassword is shown on the login screens: these are public accounts.
const demoPassword = "demo"

type demoUser struct {
	email, name, role, roleLabel, description string
}

var demoUsers = []demoUser{
	{"admin@example.com", "Alex (Admin)", "admin", "Admin", "Everything: catalog, orders, customers, roles and the audit log."},
	{"warehouse@example.com", "Sam (Warehouse)", "warehouse", "Warehouse staff", "Products and orders only; can update stock and ship orders, but not delete or see customers."},
}

// roleStore is shared by both panels (web and Flutter) and reset with the data.
var roleStore = filament.NewMemoryRoleStore()

// warehousePerms is what the limited role may do.
var warehousePerms = []string{
	"products.viewAny", "products.view", "products.update",
	"orders.viewAny", "orders.view", "orders.update",
}

func resetRoles() {
	ctx := context.Background()
	roles, _ := roleStore.ListRoles(ctx)
	for _, r := range roles {
		_ = roleStore.DeleteRole(ctx, r.Name)
	}
	_ = roleStore.SaveRole(ctx, filament.RoleDef{Name: "admin", Label: "Admin", Permissions: []string{"*"}})
	_ = roleStore.SaveRole(ctx, filament.RoleDef{Name: "warehouse", Label: "Warehouse staff", Permissions: warehousePerms})
	for _, u := range demoUsers {
		_ = roleStore.AddBinding(ctx, filament.RoleBinding{UserID: u.email, Role: u.role, Panel: filament.AnyScope, Tenant: filament.AnyScope})
	}
}

func findDemoUser(email string) (demoUser, bool) {
	for _, u := range demoUsers {
		if u.email == email {
			return u, true
		}
	}
	return demoUser{}, false
}

func (u demoUser) panelUser() *filament.PanelUser {
	return &filament.PanelUser{ID: u.email, Login: u.email, Name: u.name, Email: u.email, Role: u.roleLabel}
}

// tokens maps API tokens (Flutter app sign-ins) to demo users.
var tokens sync.Map

func validateLogin(_ *filament.Context, username, password string) (auth.Authenticatable, string, error) {
	u, ok := findDemoUser(username)
	if !ok || subtle.ConstantTimeCompare([]byte(password), []byte(demoPassword)) != 1 {
		return nil, "", errors.New("use one of the demo accounts")
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return nil, "", err
	}
	tok := "demo_" + hex.EncodeToString(b)
	tokens.Store(tok, u.email)
	return u.panelUser(), tok, nil
}

func validateToken(_ *filament.Context, token string) (auth.Authenticatable, error) {
	if v, ok := tokens.Load(token); ok {
		if u, ok := findDemoUser(v.(string)); ok {
			return u.panelUser(), nil
		}
	}
	return nil, errors.New("invalid token")
}
