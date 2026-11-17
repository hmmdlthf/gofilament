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
	if _, err := tx.ExecContext(ctx, r.s.q(`DELETE FROM `+r.s.table("role_bindings")+` WHERE role = ?`), name); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, r.s.q(`DELETE FROM `+r.s.table("roles")+` WHERE name = ?`), name); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Roles) ListBindings(ctx context.Context, f filament.BindingFilter) ([]filament.RoleBinding, error) {
	q := `SELECT user_id, role, panel, tenant FROM ` + r.s.table("role_bindings") + ` WHERE 1 = 1`
	var args []any
	if f.UserID != "" {
		q += ` AND user_id = ?`
		args = append(args, f.UserID)
	}
	if f.Role != "" {
		q += ` AND role = ?`
		args = append(args, f.Role)
	}
	rows, err := r.s.db.QueryContext(ctx, r.s.q(q+` ORDER BY user_id, role, panel, tenant`), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []filament.RoleBinding
	for rows.Next() {
		var b filament.RoleBinding
		if err := rows.Scan(&b.UserID, &b.Role, &b.Panel, &b.Tenant); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (r *Roles) AddBinding(ctx context.Context, b filament.RoleBinding) error {
	t := r.s.table("role_bindings")
	q := `INSERT INTO ` + t + ` (user_id, role, panel, tenant) VALUES (?, ?, ?, ?)`
	switch r.s.dialect {
	case MySQL:
		q = `INSERT IGNORE INTO ` + t + ` (user_id, role, panel, tenant) VALUES (?, ?, ?, ?)`
	case Postgres:
		q += ` ON CONFLICT DO NOTHING`
	default:
		q = `INSERT OR IGNORE INTO ` + t + ` (user_id, role, panel, tenant) VALUES (?, ?, ?, ?)`
	}
	_, err := r.s.db.ExecContext(ctx, r.s.q(q), b.UserID, b.Role, b.Panel, b.Tenant)
	return err
}

func (r *Roles) RemoveBinding(ctx context.Context, b filament.RoleBinding) error {
	_, err := r.s.db.ExecContext(ctx, r.s.q(`DELETE FROM `+r.s.table("role_bindings")+` WHERE user_id = ? AND role = ? AND panel = ? AND tenant = ?`), b.UserID, b.Role, b.Panel, b.Tenant)
	return err
}
