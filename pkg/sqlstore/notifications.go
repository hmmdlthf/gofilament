package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/golaravel/gofilament/pkg/notifications"
)

// Notifications implements notifications.NotificationStore. Notifications
// sent to "all" are broadcasts: every user sees them, and each user marks
// them read for themselves.
type Notifications struct{ s *Store }

// Notifications returns the notification store (notifications.SetDefaultStore).
func (s *Store) Notifications() *Notifications { return &Notifications{s: s} }

var errNotificationNotFound = errors.New("notification not found")

const broadcast = "all"

func (n *Notifications) Save(user string, x *notifications.Notification) error {
	if user == "" {
		user = broadcast
	}
	if x.CreatedAt.IsZero() {
		x.CreatedAt = time.Now()
	}
	data, err := json.Marshal(x)
	if err != nil {
		return err
	}
	s := n.s
	_, err = s.db.ExecContext(context.Background(), s.q(`INSERT INTO `+s.table("notifications")+` (id, recipient, created_at, is_read, data) VALUES (?, ?, ?, ?, ?)`),
		x.ID, user, sortableTS(x.CreatedAt), boolInt(x.Read), string(data))
	return err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// rows selects the user's own notifications and the broadcasts, newest
// first, with the read state as this user sees it.
func (n *Notifications) rows(ctx context.Context, user, extra string, args ...any) ([]*notifications.Notification, error) {
	s := n.s
	query := `SELECT n.data, n.is_read, r.recipient FROM ` + s.table("notifications") + ` n LEFT JOIN ` + s.table("notification_reads") +
		` r ON r.notification_id = n.id AND r.recipient = ? WHERE (n.recipient = ? OR n.recipient = ?)` + extra + ` ORDER BY n.created_at DESC`
	all := append([]any{user, user, broadcast}, args...)
	res, err := s.db.QueryContext(ctx, s.q(query), all...)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	var out []*notifications.Notification
	for res.Next() {
		var data string
		var read int
		var readBy sql.NullString
		if err := res.Scan(&data, &read, &readBy); err != nil {
			return nil, err
		}
		var x notifications.Notification
		if err := json.Unmarshal([]byte(data), &x); err != nil {
			continue
		}
		x.Read = read == 1 || readBy.Valid
		out = append(out, &x)
	}
	return out, res.Err()
}

func (n *Notifications) Get(user, id string) (*notifications.Notification, error) {
	list, err := n.rows(context.Background(), user, ` AND n.id = ?`, id)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errNotificationNotFound
	}
	return list[0], nil
}

func (n *Notifications) List(user string, limit int) ([]*notifications.Notification, error) {
	list, err := n.rows(context.Background(), user, "")
	if err != nil {
		return nil, err
	}
	if limit > 0 && limit < len(list) {
		list = list[:limit]
	}
	if list == nil {
		list = []*notifications.Notification{}
	}
	return list, nil
}

func (n *Notifications) CountUnread(user string) (int, error) {
	list, err := n.rows(context.Background(), user, "")
	if err != nil {
		return 0, err
	}
	c := 0
	for _, x := range list {
		if !x.Read {
			c++
		}
	}
	return c, nil
}

func (n *Notifications) MarkAsRead(user, id string) error {
	s := n.s
	var recipient string
	err := s.db.QueryRow(s.q(`SELECT recipient FROM `+s.table("notifications")+` WHERE id = ? AND (recipient = ? OR recipient = ?)`), id, user, broadcast).Scan(&recipient)
	if errors.Is(err, sql.ErrNoRows) {
		return errNotificationNotFound
	}
	if err != nil {
		return err
	}
	if recipient == broadcast && user != broadcast {
		return n.markBroadcastRead(user, id)
	}
	_, err = s.db.Exec(s.q(`UPDATE `+s.table("notifications")+` SET is_read = 1 WHERE id = ?`), id)
	return err
}

func (n *Notifications) markBroadcastRead(user, id string) error {
	s := n.s
	var exists int
	err := s.db.QueryRow(s.q(`SELECT 1 FROM `+s.table("notification_reads")+` WHERE notification_id = ? AND recipient = ?`), id, user).Scan(&exists)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = s.db.Exec(s.q(`INSERT INTO `+s.table("notification_reads")+` (notification_id, recipient) VALUES (?, ?)`), id, user)
	return err
}

func (n *Notifications) MarkAllAsRead(user string) error {
	s := n.s
	if _, err := s.db.Exec(s.q(`UPDATE `+s.table("notifications")+` SET is_read = 1 WHERE recipient = ?`), user); err != nil {
		return err
	}
	if user == broadcast {
		return nil
	}
	list, err := n.rows(context.Background(), user, ` AND n.recipient = ?`, broadcast)
	if err != nil {
		return err
	}
	for _, x := range list {
		if !x.Read {
			if err := n.markBroadcastRead(user, x.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// Clear deletes the user's own notifications (broadcasts stay).
func (n *Notifications) Clear(user string) error {
	s := n.s
	_, err := s.db.Exec(s.q(`DELETE FROM `+s.table("notifications")+` WHERE recipient = ?`), user)
	return err
}
