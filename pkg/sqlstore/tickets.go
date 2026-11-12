package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	gsupport "github.com/golaravel/golaravel/pkg/support"
)

// Tickets implements golaravel's support.Store. A ticket and its messages
// are one JSON document; the columns used for filtering are kept next to it.
type Tickets struct{ s *Store }

// Tickets returns the support ticket store (support.SetDefaultStore).
func (s *Store) Tickets() *Tickets { return &Tickets{s: s} }

func notFound(id string) error { return fmt.Errorf("ticket %s not found", id) }

func ticketSearch(t *gsupport.Ticket) string {
	return strings.ToLower(strings.Join([]string{t.Subject, t.UserName, t.UserEmail, t.UserKey, t.ID}, " "))
}

func (k *Tickets) CreateTicket(ctx context.Context, t *gsupport.Ticket) error {
	if t.ID == "" {
		t.ID = gsupport.GenerateTicketID()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	t.UpdatedAt = time.Now()
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	s := k.s
	_, err = s.db.ExecContext(ctx, s.q(`INSERT INTO `+s.table("tickets")+` (id, tenant_id, panel_id, category, mode, status, user_key, department, search, created_at, updated_at, data) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		t.ID, t.TenantID, t.PanelID, t.Category, t.Mode, t.Status, t.UserKey, t.Department, ticketSearch(t), sortableTS(t.CreatedAt), sortableTS(t.UpdatedAt), string(data))
	return err
}

func (k *Tickets) GetTicket(ctx context.Context, id string) (*gsupport.Ticket, error) {
	s := k.s
	var data string
	err := s.db.QueryRowContext(ctx, s.q(`SELECT data FROM `+s.table("tickets")+` WHERE id = ?`), id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notFound(id)
	}
	if err != nil {
		return nil, err
	}
	var t gsupport.Ticket
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// ListTickets filters like the memory store (a ticket without a tenant or
// panel matches every tenant / panel filter), newest activity first.
func (k *Tickets) ListTickets(ctx context.Context, f gsupport.TicketFilter) ([]*gsupport.Ticket, error) {
	s := k.s
	where, args := []string{"1 = 1"}, []any{}
	if f.TenantID != "" {
		where = append(where, "(tenant_id = ? OR tenant_id = '' OR tenant_id IS NULL)")
		args = append(args, f.TenantID)
	}
	if f.PanelID != "" {
		where = append(where, "(panel_id = ? OR panel_id = '' OR panel_id IS NULL)")
		args = append(args, f.PanelID)
	}
	for col, v := range map[string]string{"category": f.Category, "mode": f.Mode, "status": f.Status, "user_key": f.UserKey, "department": f.Department} {
		if v != "" {
			where = append(where, col+" = ?")
			args = append(args, v)
		}
	}
	if q := strings.ToLower(strings.TrimSpace(f.Search)); q != "" {
		where = append(where, "search LIKE ? ESCAPE '!'")
		args = append(args, "%"+likeEscape(q)+"%")
	}
	rows, err := s.db.QueryContext(ctx, s.q(`SELECT data FROM `+s.table("tickets")+` WHERE `+strings.Join(where, " AND ")+` ORDER BY updated_at DESC, id`), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*gsupport.Ticket{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var t gsupport.Ticket
		if json.Unmarshal([]byte(data), &t) == nil {
			out = append(out, &t)
		}
	}
	return out, rows.Err()
}

// change loads a ticket, applies fn and writes it back in one transaction.
func (k *Tickets) change(ctx context.Context, id string, fn func(t *gsupport.Ticket)) error {
	s := k.s
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	lock := ""
	if s.dialect != SQLite {
		lock = " FOR UPDATE"
	}
	var data string
	err = tx.QueryRowContext(ctx, s.q(`SELECT data FROM `+s.table("tickets")+` WHERE id = ?`+lock), id).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return notFound(id)
	}
	if err != nil {
		return err
	}
	var t gsupport.Ticket
	if err := json.Unmarshal([]byte(data), &t); err != nil {
		return err
	}
	fn(&t)
	t.UpdatedAt = time.Now()
	b, err := json.Marshal(&t)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, s.q(`UPDATE `+s.table("tickets")+` SET tenant_id = ?, panel_id = ?, category = ?, mode = ?, status = ?, user_key = ?, department = ?, search = ?, updated_at = ?, data = ? WHERE id = ?`),
		t.TenantID, t.PanelID, t.Category, t.Mode, t.Status, t.UserKey, t.Department, ticketSearch(&t), sortableTS(t.UpdatedAt), string(b), id); err != nil {
		return err
	}
	return tx.Commit()
}

func (k *Tickets) AddMessage(ctx context.Context, ticketID string, msg *gsupport.SupportMessage) error {
	if msg.ID == "" {
		msg.ID = gsupport.GenerateMessageID()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	msg.TicketID = ticketID
	return k.change(ctx, ticketID, func(t *gsupport.Ticket) { t.Messages = append(t.Messages, *msg) })
}

func (k *Tickets) UpdateTicket(ctx context.Context, t *gsupport.Ticket) error {
	return k.change(ctx, t.ID, func(cur *gsupport.Ticket) {
		created := cur.CreatedAt
		*cur = *t
		cur.CreatedAt = created
	})
}

func (k *Tickets) UpdateTicketStatus(ctx context.Context, id, status string) error {
	return k.change(ctx, id, func(t *gsupport.Ticket) { t.Status = status })
}

func (k *Tickets) UpdateTicketMode(ctx context.Context, id, mode string) error {
	return k.change(ctx, id, func(t *gsupport.Ticket) { t.Mode = mode })
}

func (k *Tickets) EscalateTicket(ctx context.Context, id, targetPanel, reason string) error {
	return k.change(ctx, id, func(t *gsupport.Ticket) {
		now := time.Now()
		t.Status = gsupport.StatusEscalated
		t.Mode = gsupport.ModeLive
		t.TargetPanel = targetPanel
		t.EscalatedAt = &now
		t.EscalatedReason = reason
	})
}

var _ gsupport.Store = (*Tickets)(nil)
