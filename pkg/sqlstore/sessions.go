package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// WebSessions implements filament.WebSessionStore.
type WebSessions struct{ s *Store }

// WebSessions returns the web-engine session store (panel.WebSessionsUsing).
func (s *Store) WebSessions() *WebSessions { return &WebSessions{s: s} }

func (w *WebSessions) PutSession(ctx context.Context, id, userID string, expires time.Time) error {
	s := w.s
	// Opportunistically drop expired sessions.
	_, _ = s.db.ExecContext(ctx, s.q(`DELETE FROM `+s.table("sessions")+` WHERE expires_at < ?`), sortableTS(time.Now()))
	_, err := s.db.ExecContext(ctx, s.q(`INSERT INTO `+s.table("sessions")+` (id, user_id, expires_at) VALUES (?, ?, ?)`), id, userID, sortableTS(expires))
	return err
}

func (w *WebSessions) GetSession(ctx context.Context, id string) (string, bool, error) {
	s := w.s
	var userID, expires string
	err := s.db.QueryRowContext(ctx, s.q(`SELECT user_id, expires_at FROM `+s.table("sessions")+` WHERE id = ?`), id).Scan(&userID, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !parseTS(expires).After(time.Now()) {
		_ = w.DeleteSession(ctx, id)
		return "", false, nil
	}
	return userID, true, nil
}

func (w *WebSessions) DeleteSession(ctx context.Context, id string) error {
	_, err := w.s.db.ExecContext(ctx, w.s.q(`DELETE FROM `+w.s.table("sessions")+` WHERE id = ?`), id)
	return err
}

// DeleteUserSessions signs a user out everywhere (for example after a
// password change).
func (w *WebSessions) DeleteUserSessions(ctx context.Context, userID string) error {
	_, err := w.s.db.ExecContext(ctx, w.s.q(`DELETE FROM `+w.s.table("sessions")+` WHERE user_id = ?`), userID)
	return err
}
