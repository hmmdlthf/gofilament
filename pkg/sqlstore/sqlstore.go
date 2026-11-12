// Package sqlstore keeps GoFilament's runtime state in a SQL database
// instead of process memory, so it survives restarts and is shared between
// processes: web-engine sessions, database notifications, the activity log
// and support tickets.
//
//	db, _ := database.OpenDB(cfg) // or sql.Open
//	s := sqlstore.New(db, sqlstore.Postgres)
//	if err := s.Migrate(ctx); err != nil { ... }
//
//	notifications.SetDefaultStore(s.Notifications())
//	activitylog.SetDefaultStore(s.Activities())
//	gsupport.SetDefaultStore(s.Tickets())
//	panel.WebSessionsUsing(s.WebSessions(), loadUser)
//
// Tables are prefixed with "gofilament_" (see Store.Prefix). Times are stored
// in UTC as RFC 3339 text, so ordering works the same in every database.
package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Dialect selects placeholder style and column types.
type Dialect string

const (
	SQLite   Dialect = "sqlite"
	MySQL    Dialect = "mysql"
	Postgres Dialect = "postgres"
)

// Store is a set of SQL-backed stores sharing one database handle.
type Store struct {
	db      *sql.DB
	dialect Dialect
	// Prefix is prepended to every table name (default "gofilament_").
	Prefix string
}

// New returns stores on db. Call Migrate once at start-up.
func New(db *sql.DB, dialect Dialect) *Store {
	return &Store{db: db, dialect: dialect, Prefix: "gofilament_"}
}

func (s *Store) table(name string) string { return s.Prefix + name }

// q rewrites ? placeholders for the dialect.
func (s *Store) q(query string) string {
	if s.dialect != Postgres {
		return query
	}
	var b strings.Builder
	n := 0
	for _, r := range query {
		if r == '?' {
			n++
			fmt.Fprintf(&b, "$%d", n)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// key is a type for indexed text columns (MySQL cannot index TEXT).
func (s *Store) key() string {
	if s.dialect == MySQL {
		return "VARCHAR(191)"
	}
	return "TEXT"
}

func (s *Store) longText() string {
	if s.dialect == MySQL {
		return "LONGTEXT"
	}
	return "TEXT"
}

// Migrate creates the tables and indexes that do not exist yet.
func (s *Store) Migrate(ctx context.Context) error {
	k, t := s.key(), s.longText()
	stmts := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (id %s PRIMARY KEY, user_id %s NOT NULL, expires_at %s NOT NULL)`, s.table("sessions"), k, k, k),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (id %s PRIMARY KEY, recipient %s NOT NULL, created_at %s NOT NULL, is_read INTEGER NOT NULL DEFAULT 0, data %s NOT NULL)`, s.table("notifications"), k, k, k, t),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (notification_id %s NOT NULL, recipient %s NOT NULL, PRIMARY KEY (notification_id, recipient))`, s.table("notification_reads"), k, k),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (id %s PRIMARY KEY, log_name %s, subject_type %s, subject_id %s, causer_type %s, causer_id %s, event %s, search %s, created_at %s NOT NULL, data %s NOT NULL)`,
			s.table("activities"), k, k, k, k, k, k, k, t, k, t),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (id %s PRIMARY KEY, tenant_id %s, panel_id %s, category %s, mode %s, status %s, user_key %s, department %s, search %s, created_at %s NOT NULL, updated_at %s NOT NULL, data %s NOT NULL)`,
			s.table("tickets"), k, k, k, k, k, k, k, k, t, k, k, t),
	}
	indexes := [][2]string{
		{"sessions", "expires_at"},
		{"notifications", "recipient, created_at"},
		{"activities", "subject_type, subject_id"},
		{"activities", "causer_type, causer_id"},
		{"activities", "created_at"},
		{"tickets", "updated_at"},
	}
	for _, ix := range indexes {
		name := s.table(ix[0]) + "_" + strings.NewReplacer(", ", "_").Replace(ix[1]) + "_idx"
		if s.dialect == MySQL {
			// MySQL has no CREATE INDEX IF NOT EXISTS; errors for an
			// existing index are ignored below.
			stmts = append(stmts, fmt.Sprintf(`CREATE INDEX %s ON %s (%s)`, name, s.table(ix[0]), ix[1]))
		} else {
			stmts = append(stmts, fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s (%s)`, name, s.table(ix[0]), ix[1]))
		}
	}
	for _, st := range stmts {
		if _, err := s.db.ExecContext(ctx, st); err != nil {
			if s.dialect == MySQL && strings.HasPrefix(st, "CREATE INDEX") && strings.Contains(err.Error(), "Duplicate key name") {
				continue
			}
			return fmt.Errorf("sqlstore: migrate: %w", err)
		}
	}
	return nil
}

// sortableTS pads fractional seconds so text order is time order.
func sortableTS(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000000000Z07:00") }

func parseTS(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}
