package sqlstore_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gsupport "github.com/golaravel/golaravel/pkg/support"
	_ "github.com/mattn/go-sqlite3"

	"github.com/golaravel/gofilament/pkg/activitylog"
	"github.com/golaravel/gofilament/pkg/notifications"
	"github.com/golaravel/gofilament/pkg/sqlstore"
)

func open(t *testing.T) (*sqlstore.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s := sqlstore.New(db, sqlstore.SQLite)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("Migrate must be repeatable: %v", err)
	}
	return s, path
}

// reopen simulates a restart: a new handle on the same file.
func reopen(t *testing.T, path string) *sqlstore.Store {
	t.Helper()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return sqlstore.New(db, sqlstore.SQLite)
}

func TestWebSessions(t *testing.T) {
	s, path := open(t)
	ctx := context.Background()
	w := s.WebSessions()
	if err := w.PutSession(ctx, "h1", "u1", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	_ = w.PutSession(ctx, "old", "u1", time.Now().Add(-time.Minute))
	if id, ok, err := reopen(t, path).WebSessions().GetSession(ctx, "h1"); err != nil || !ok || id != "u1" {
		t.Fatalf("sessions survive a restart: %q %v %v", id, ok, err)
	}
	if _, ok, _ := w.GetSession(ctx, "old"); ok {
		t.Fatal("expired sessions are not returned")
	}
	_ = w.DeleteUserSessions(ctx, "u1")
	if _, ok, _ := w.GetSession(ctx, "h1"); ok {
		t.Fatal("DeleteUserSessions signs the user out")
	}
}

func TestNotifications(t *testing.T) {
	s, path := open(t)
	n := s.Notifications()
	old := notifications.Make("Old")
	old.ID, old.CreatedAt = "n1", time.Now().Add(-time.Hour)
	_ = n.Save("ann", old)
	news := notifications.Make("New")
	news.ID = "n2"
	_ = n.Save("ann", news)
	all := notifications.Make("Maintenance tonight")
	all.ID = "b1"
	_ = n.Save("all", all)
	other := notifications.Make("Bob only")
	other.ID = "n3"
	_ = n.Save("bob", other)

	n2 := reopen(t, path).Notifications()
	list, err := n2.List("ann", 0)
	if err != nil {
		t.Fatal(err)
	}
	var titles []string
	for _, x := range list {
		titles = append(titles, x.Title)
	}
	if strings.Join(titles, ",") != "Maintenance tonight,New,Old" {
		t.Fatalf("ann sees her own and broadcast notifications, newest first: %v", titles)
	}
	if c, _ := n2.CountUnread("ann"); c != 3 {
		t.Fatalf("unread: %d", c)
	}
	if err := n2.MarkAsRead("ann", "b1"); err != nil {
		t.Fatal(err)
	}
	if c, _ := n2.CountUnread("ann"); c != 2 {
		t.Fatalf("a read broadcast counts as read for ann: %d", c)
	}
	if c, _ := n2.CountUnread("bob"); c != 2 {
		t.Fatalf("...but not for bob: %d", c)
	}
	if err := n2.MarkAsRead("ann", "n3"); err == nil {
		t.Fatal("ann cannot mark bob's notification")
	}
	_ = n2.MarkAllAsRead("ann")
	if c, _ := n2.CountUnread("ann"); c != 0 {
		t.Fatalf("mark all: %d", c)
	}
	if got, _ := n2.Get("ann", "n2"); got == nil || !got.Read || got.Title != "New" {
		t.Fatalf("get: %+v", got)
	}
	_ = n2.Clear("ann")
	if list, _ := n2.List("ann", 10); len(list) != 1 || list[0].ID != "b1" {
		t.Fatalf("clear keeps broadcasts: %v", list)
	}
	if list, _ := n2.List("bob", 1); len(list) != 1 {
		t.Fatalf("limit: %v", list)
	}
}

func TestActivities(t *testing.T) {
	s, path := open(t)
	ctx := context.Background()
	a := s.Activities()
	for i, ev := range []string{"created", "updated", "deleted"} {
		_ = a.Log(ctx, &activitylog.Activity{
			LogName: "default", SubjectType: "Order", SubjectID: "7", CauserType: "User", CauserID: "u1",
			CauserName: "Ann Lee", Event: ev, Description: "Order 7 " + ev,
			NewValues: map[string]any{"total": i}, CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
		})
	}
	_ = a.Log(ctx, &activitylog.Activity{SubjectType: "Invoice", SubjectID: "1", Event: "created", Description: "100% paid_in_full"})

	a2 := reopen(t, path).Activities()
	got, _ := a2.GetBySubject(ctx, "order", "7", 2)
	if len(got) != 2 || got[0].Event != "deleted" || got[1].Event != "updated" {
		t.Fatalf("by subject, newest first, limited: %+v", got)
	}
	if got, _ := a2.GetByCauser(ctx, "User", "u1"); len(got) != 3 {
		t.Fatalf("by causer: %d", len(got))
	}
	list, total, err := a2.Query(ctx, activitylog.Filter{SubjectType: "ORDER", Page: 2, PerPage: 2})
	if err != nil || total != 3 || len(list) != 1 || list[0].Event != "created" {
		t.Fatalf("query page 2: %v %d %+v", err, total, list)
	}
	if list, total, _ := a2.Query(ctx, activitylog.Filter{Search: "ann lee"}); total != 3 || len(list) != 3 {
		t.Fatalf("search by causer name: %d", total)
	}
	if _, total, _ := a2.Query(ctx, activitylog.Filter{Search: "0% p"}); total != 1 {
		t.Fatalf("LIKE wildcards in the search are literal: %d", total)
	}
	if _, total, _ := a2.Query(ctx, activitylog.Filter{Search: "_"}); total != 1 {
		t.Fatalf("underscore is literal: %d", total)
	}
	first := got[0]
	if x, err := a2.GetByID(ctx, first.ID); err != nil || x.NewValues["total"] != float64(2) {
		t.Fatalf("get by id keeps values: %v %+v", err, x)
	}
	if _, err := a2.GetByID(ctx, "nope"); err != activitylog.ErrActivityNotFound {
		t.Fatalf("missing: %v", err)
	}
	_ = a2.Clear(ctx)
	if _, total, _ := a2.Query(ctx, activitylog.Filter{}); total != 0 {
		t.Fatal("clear")
	}
}

func TestTickets(t *testing.T) {
	s, path := open(t)
	ctx := context.Background()
	k := s.Tickets()
	t1 := &gsupport.Ticket{TenantID: "acme", PanelID: "client", UserKey: "u1", UserName: "Ann", Subject: "Printer broken", Category: "support", Mode: "ai", Status: "open"}
	if err := k.CreateTicket(ctx, t1); err != nil || t1.ID == "" {
		t.Fatalf("create: %v", err)
	}
	t2 := &gsupport.Ticket{UserKey: "u2", Subject: "Dark mode please", Category: "feature_request", Mode: "ai", Status: "open"}
	_ = k.CreateTicket(ctx, t2)
	_ = k.CreateTicket(ctx, &gsupport.Ticket{TenantID: "other", Subject: "Not mine", Status: "open"})

	k2 := reopen(t, path).Tickets()
	if err := k2.AddMessage(ctx, t1.ID, &gsupport.SupportMessage{SenderType: "user", Content: "It jams"}); err != nil {
		t.Fatal(err)
	}
	if err := k2.EscalateTicket(ctx, t1.ID, "super_admin", "needs a human"); err != nil {
		t.Fatal(err)
	}
	got, err := k2.GetTicket(ctx, t1.ID)
	if err != nil || len(got.Messages) != 1 || got.Messages[0].TicketID != t1.ID || got.Status != gsupport.StatusEscalated || got.Mode != gsupport.ModeLive || got.EscalatedAt == nil {
		t.Fatalf("ticket after message and escalation: %v %+v", err, got)
	}
	list, _ := k2.ListTickets(ctx, gsupport.TicketFilter{TenantID: "acme"})
	if len(list) != 2 || list[0].ID != t1.ID {
		t.Fatalf("tenant filter keeps unscoped tickets, newest activity first: %d", len(list))
	}
	if list, _ := k2.ListTickets(ctx, gsupport.TicketFilter{Status: "escalated"}); len(list) != 1 {
		t.Fatalf("status filter: %d", len(list))
	}
	if list, _ := k2.ListTickets(ctx, gsupport.TicketFilter{Search: "dark"}); len(list) != 1 || list[0].ID != t2.ID {
		t.Fatalf("search: %v", list)
	}
	_ = k2.UpdateTicketStatus(ctx, t2.ID, "resolved")
	_ = k2.UpdateTicketMode(ctx, t2.ID, "live")
	if g, _ := k2.GetTicket(ctx, t2.ID); g.Status != "resolved" || g.Mode != "live" {
		t.Fatalf("status/mode: %+v", g)
	}
	g, _ := k2.GetTicket(ctx, t2.ID)
	g.Priority = "high"
	_ = k2.UpdateTicket(ctx, g)
	if g2, _ := k2.GetTicket(ctx, t2.ID); g2.Priority != "high" || !g2.CreatedAt.Equal(g.CreatedAt) {
		t.Fatalf("update: %+v", g2)
	}
	if err := k2.AddMessage(ctx, "missing", &gsupport.SupportMessage{}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing ticket: %v", err)
	}
}
