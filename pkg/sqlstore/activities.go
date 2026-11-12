package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/activitylog"
)

// Activities implements activitylog.Store.
type Activities struct{ s *Store }

// Activities returns the activity log store.
func (s *Store) Activities() *Activities { return &Activities{s: s} }

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func (a *Activities) Log(ctx context.Context, x *activitylog.Activity) error {
	if x == nil {
		return errors.New("cannot log nil activity")
	}
	if x.ID == "" {
		x.ID = newID("act_")
	}
	if x.CreatedAt.IsZero() {
		x.CreatedAt = time.Now()
	}
	data, err := json.Marshal(x)
	if err != nil {
		return err
	}
	search := strings.ToLower(strings.Join([]string{x.Description, x.CauserName, x.SubjectType, x.SubjectID, x.IPAddress, x.Event}, " "))
	s := a.s
	_, err = s.db.ExecContext(ctx, s.q(`INSERT INTO `+s.table("activities")+` (id, log_name, subject_type, subject_id, causer_type, causer_id, event, search, created_at, data) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		x.ID, strings.ToLower(x.LogName), strings.ToLower(x.SubjectType), x.SubjectID, strings.ToLower(x.CauserType), x.CauserID, strings.ToLower(x.Event), search, sortableTS(x.CreatedAt), string(data))
	return err
}

func (a *Activities) scan(rows *sql.Rows) ([]*activitylog.Activity, error) {
	defer rows.Close()
	out := []*activitylog.Activity{}
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var x activitylog.Activity
		if json.Unmarshal([]byte(data), &x) == nil {
			out = append(out, &x)
		}
	}
	return out, rows.Err()
}

func (a *Activities) GetByID(ctx context.Context, id string) (*activitylog.Activity, error) {
	s := a.s
	rows, err := s.db.QueryContext(ctx, s.q(`SELECT data FROM `+s.table("activities")+` WHERE id = ?`), id)
	if err != nil {
		return nil, err
	}
	list, err := a.scan(rows)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, activitylog.ErrActivityNotFound
	}
	return list[0], nil
}

func limitOf(limit []int) int {
	if len(limit) > 0 && limit[0] > 0 {
		return limit[0]
	}
	return 50
}

func (a *Activities) by(ctx context.Context, typeCol, idCol, typ, id string, limit int) ([]*activitylog.Activity, error) {
	s := a.s
	rows, err := s.db.QueryContext(ctx, s.q(`SELECT data FROM `+s.table("activities")+` WHERE `+typeCol+` = ? AND `+idCol+` = ? ORDER BY created_at DESC, id DESC LIMIT ?`),
		strings.ToLower(typ), id, limit)
	if err != nil {
		return nil, err
	}
	return a.scan(rows)
}

func (a *Activities) GetBySubject(ctx context.Context, subjectType, subjectID string, limit ...int) ([]*activitylog.Activity, error) {
	return a.by(ctx, "subject_type", "subject_id", subjectType, subjectID, limitOf(limit))
}

func (a *Activities) GetByCauser(ctx context.Context, causerType, causerID string, limit ...int) ([]*activitylog.Activity, error) {
	return a.by(ctx, "causer_type", "causer_id", causerType, causerID, limitOf(limit))
}

// Query filters (case-insensitively for names and types, like the memory
// store), searches and paginates, newest first.
func (a *Activities) Query(ctx context.Context, f activitylog.Filter) ([]*activitylog.Activity, int, error) {
	s := a.s
	where, args := []string{"1 = 1"}, []any{}
	eq := func(col, v string, lower bool) {
		if v == "" {
			return
		}
		if lower {
			v = strings.ToLower(v)
		}
		where = append(where, col+" = ?")
		args = append(args, v)
	}
	eq("log_name", f.LogName, true)
	eq("subject_type", f.SubjectType, true)
	eq("subject_id", f.SubjectID, false)
	eq("causer_type", f.CauserType, true)
	eq("causer_id", f.CauserID, false)
	eq("event", f.Event, true)
	if q := strings.ToLower(strings.TrimSpace(f.Search)); q != "" {
		where = append(where, "search LIKE ? ESCAPE '!'")
		args = append(args, "%"+likeEscape(q)+"%")
	}
	cond := strings.Join(where, " AND ")
	var total int
	if err := s.db.QueryRowContext(ctx, s.q(`SELECT COUNT(*) FROM `+s.table("activities")+` WHERE `+cond), args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	page, per := f.Page, f.PerPage
	if page < 1 {
		page = 1
	}
	if per <= 0 {
		per = 25
	}
	rows, err := s.db.QueryContext(ctx, s.q(`SELECT data FROM `+s.table("activities")+` WHERE `+cond+` ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`),
		append(args, per, (page-1)*per)...)
	if err != nil {
		return nil, 0, err
	}
	list, err := a.scan(rows)
	return list, total, err
}

func likeEscape(s string) string {
	return strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(s)
}

func (a *Activities) Clear(ctx context.Context) error {
	_, err := a.s.db.ExecContext(ctx, `DELETE FROM `+a.s.table("activities"))
	return err
}
