package sqlstore

import (
	"context"
	"encoding/json"

	"github.com/golaravel/gofilament/pkg/filament"
)

// Roles implements filament.RoleStore.
type Roles struct{ s *Store }

// Roles returns the roles and permissions store (Panel.Roles).
func (s *Store) Roles() *Roles { return &Roles{s: s} }

var _ filament.RoleStore = (*Roles)(nil)

func (r *Roles) ListRoles(ctx context.Context) ([]filament.RoleDef, error) {
	rows, err := r.s.db.QueryContext(ctx, `SELECT data FROM `+r.s.table("roles")+` ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []filament.RoleDef
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var role filament.RoleDef
		if err := json.Unmarshal([]byte(data), &role); err != nil {
			return nil, err
		}
		out = append(out, role)
	}
	return out, rows.Err()
}

func (r *Roles) SaveRole(ctx context.Context, role filament.RoleDef) error {
	data, err := json.Marshal(role)
	if err != nil {
		return err
	}
	t := r.s.table("roles")
	tx, err := r.s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, r.s.q(`DELETE FROM `+t+` WHERE name = ?`), role.Name); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, r.s.q(`INSERT INTO `+t+` (name, data) VALUES (?, ?)`), role.Name, string(data)); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Roles) DeleteRole(ctx context.Context, name string) error {
	tx, err := r.s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, r.s.q(`DELETE FROM `+r.s.table("user_roles")+` WHERE role = ?`), name); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, r.s.q(`DELETE FROM `+r.s.table("roles")+` WHERE name = ?`), name); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Roles) strings(ctx context.Context, query string, arg string) ([]string, error) {
	rows, err := r.s.db.QueryContext(ctx, r.s.q(query), arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Roles) UserRoles(ctx context.Context, userID string) ([]string, error) {
	return r.strings(ctx, `SELECT role FROM `+r.s.table("user_roles")+` WHERE user_id = ? ORDER BY role`, userID)
}

func (r *Roles) RoleUsers(ctx context.Context, role string) ([]string, error) {
	return r.strings(ctx, `SELECT user_id FROM `+r.s.table("user_roles")+` WHERE role = ? ORDER BY user_id`, role)
}

func (r *Roles) SetUserRoles(ctx context.Context, userID string, roles []string) error {
	t := r.s.table("user_roles")
	tx, err := r.s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, r.s.q(`DELETE FROM `+t+` WHERE user_id = ?`), userID); err != nil {
		return err
	}
	for _, role := range roles {
		if _, err := tx.ExecContext(ctx, r.s.q(`INSERT INTO `+t+` (user_id, role) VALUES (?, ?)`), userID, role); err != nil {
			return err
		}
	}
	return tx.Commit()
}
