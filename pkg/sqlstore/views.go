package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/golaravel/gofilament/pkg/filament"
)

// Views implements filament.SavedViewStore.
type Views struct{ s *Store }

// Views returns the saved list views store (Panel.SavedViews).
func (s *Store) Views() *Views { return &Views{s: s} }

var _ filament.SavedViewStore = (*Views)(nil)

func (v *Views) ListViews(ctx context.Context, panel, resource, ownerID string) ([]filament.SavedView, error) {
	rows, err := v.s.db.QueryContext(ctx, v.s.q(`SELECT data FROM `+v.s.table("saved_views")+
		` WHERE panel = ? AND resource = ? AND (owner_id = ? OR shared = 1) ORDER BY name`), panel, resource, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []filament.SavedView
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var sv filament.SavedView
		if err := json.Unmarshal([]byte(data), &sv); err != nil {
			return nil, err
		}
		out = append(out, sv)
	}
	return out, rows.Err()
}

func (v *Views) GetView(ctx context.Context, id string) (*filament.SavedView, error) {
	var data string
	err := v.s.db.QueryRowContext(ctx, v.s.q(`SELECT data FROM `+v.s.table("saved_views")+` WHERE id = ?`), id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, filament.ErrViewNotFound
	}
	if err != nil {
		return nil, err
	}
	var sv filament.SavedView
	if err := json.Unmarshal([]byte(data), &sv); err != nil {
		return nil, err
	}
	return &sv, nil
}

func (v *Views) SaveView(ctx context.Context, sv *filament.SavedView) error {
	data, err := json.Marshal(sv)
	if err != nil {
		return err
	}
	t := v.s.table("saved_views")
	tx, err := v.s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, v.s.q(`DELETE FROM `+t+` WHERE id = ?`), sv.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, v.s.q(`INSERT INTO `+t+` (id, panel, resource, owner_id, shared, name, created_at, data) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`),
		sv.ID, sv.Panel, sv.Resource, sv.OwnerID, boolInt(sv.Shared), sv.Name, sortableTS(sv.CreatedAt), string(data)); err != nil {
		return err
	}
	return tx.Commit()
}

func (v *Views) DeleteView(ctx context.Context, id string) error {
	_, err := v.s.db.ExecContext(ctx, v.s.q(`DELETE FROM `+v.s.table("saved_views")+` WHERE id = ?`), id)
	return err
}
