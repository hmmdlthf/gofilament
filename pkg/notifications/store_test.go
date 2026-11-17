package notifications

import (
	"testing"
	"time"
)

func TestMemoryStoreBroadcastReadsArePerUser(t *testing.T) {
	s := NewMemoryStore()
	b := Make("Maintenance tonight")
	b.ID, b.CreatedAt = "b1", time.Now().Add(-time.Hour)
	_ = s.Save("all", b)
	own := Make("Your export is ready")
	own.ID = "o1"
	_ = s.Save("ann", own)

	list, _ := s.List("ann", 10)
	if len(list) != 2 || list[0].ID != "o1" {
		t.Fatalf("newest first: %+v", list)
	}
	if err := s.MarkAsRead("ann", "b1"); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.CountUnread("ann"); n != 1 {
		t.Fatalf("ann unread: %d", n)
	}
	if n, _ := s.CountUnread("bob"); n != 1 {
		t.Fatalf("bob's broadcast was marked read by ann: %d", n)
	}
	// Changing a listed copy does not change the store.
	list, _ = s.List("bob", 10)
	list[0].Read = true
	if n, _ := s.CountUnread("bob"); n != 1 {
		t.Fatal("List returned shared objects")
	}
	_ = s.MarkAllAsRead("bob")
	_ = s.Clear("bob")
	if n, _ := s.CountUnread("ann"); n != 1 {
		t.Fatalf("bob's clear/mark-all touched ann: %d", n)
	}
	if list, _ := s.List("carl", 10); len(list) != 1 {
		t.Fatal("clear deleted the broadcast")
	}
}
