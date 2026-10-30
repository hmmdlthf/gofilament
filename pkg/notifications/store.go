package notifications

import (
	"errors"
	"sync"
	"time"

	"github.com/golaravel/golaravel/pkg/broadcast"
)

// NotificationStore defines the interface for persisting and retrieving database notifications.
type NotificationStore interface {
	Get(user string, id string) (*Notification, error)
	List(user string, limit int) ([]*Notification, error)
	CountUnread(user string) (int, error)
	Save(user string, n *Notification) error
	MarkAsRead(user string, id string) error
	MarkAllAsRead(user string) error
	Clear(user string) error
}

// MemoryNotificationStore is a concurrent-safe in-memory store for notifications.
type MemoryNotificationStore struct {
	mu            sync.RWMutex
	notifications map[string][]*Notification
}

// NewMemoryStore creates an initialized in-memory notification store.
func NewMemoryStore() *MemoryNotificationStore {
	return &MemoryNotificationStore{
		notifications: make(map[string][]*Notification),
	}
}

func (s *MemoryNotificationStore) Get(user string, id string) (*Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, n := range s.notifications[user] {
		if n.ID == id {
			return n, nil
		}
	}
	return nil, errors.New("notification not found")
}

func (s *MemoryNotificationStore) List(user string, limit int) ([]*Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.notifications[user]
	// Also include broadcast items sent to "all"
	if user != "all" && len(s.notifications["all"]) > 0 {
		combined := make([]*Notification, 0, len(items)+len(s.notifications["all"]))
		combined = append(combined, items...)
		combined = append(combined, s.notifications["all"]...)
		items = combined
	}

	if limit <= 0 || limit > len(items) {
		limit = len(items)
	}

	result := make([]*Notification, limit)
	copy(result, items[:limit])
	return result, nil
}

func (s *MemoryNotificationStore) CountUnread(user string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, n := range s.notifications[user] {
		if !n.Read {
			count++
		}
	}
	if user != "all" {
		for _, n := range s.notifications["all"] {
			if !n.Read {
				count++
			}
		}
	}
	return count, nil
}

func (s *MemoryNotificationStore) Save(user string, n *Notification) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}

	// Prepend for newest-first ordering
	s.notifications[user] = append([]*Notification{n}, s.notifications[user]...)
	return nil
}

func (s *MemoryNotificationStore) MarkAsRead(user string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, n := range s.notifications[user] {
		if n.ID == id {
			n.Read = true
			return nil
		}
	}
	if user != "all" {
		for _, n := range s.notifications["all"] {
			if n.ID == id {
				n.Read = true
				return nil
			}
		}
	}
	return errors.New("notification not found")
}

func (s *MemoryNotificationStore) MarkAllAsRead(user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, n := range s.notifications[user] {
		n.Read = true
	}
	if user != "all" {
		for _, n := range s.notifications["all"] {
			n.Read = true
		}
	}
	return nil
}

func (s *MemoryNotificationStore) Clear(user string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.notifications, user)
	return nil
}

// Global Default Notification Store
var (
	defaultStoreMu sync.RWMutex
	defaultStore   NotificationStore = NewMemoryStore()
)

// DefaultStore returns the active notification store.
func DefaultStore() NotificationStore {
	defaultStoreMu.RLock()
	defer defaultStoreMu.RUnlock()
	return defaultStore
}

// SetDefaultStore configures a custom notification store.
func SetDefaultStore(s NotificationStore) {
	defaultStoreMu.Lock()
	defer defaultStoreMu.Unlock()
	defaultStore = s
}

// Send persists a notification for a target user and broadcasts it in real-time.
func Send(user string, n *Notification) error {
	if user == "" {
		user = "all"
	}

	store := DefaultStore()
	if err := store.Save(user, n); err != nil {
		return err
	}

	// Broadcast via GoLaravel realtime hub
	payload := map[string]any{
		"id":         n.ID,
		"type":       string(n.Type),
		"title":      n.Title,
		"body":       n.BodyText,
		"duration":   n.Duration,
		"action_url": n.ActionURL,
		"read":       n.Read,
		"created_at": n.CreatedAt.Format(time.RFC3339),
	}

	channel := "filament:notifications:" + user
	broadcast.Broadcast(channel, "notification", payload)
	if user != "all" {
		broadcast.Broadcast("filament:notifications:all", "notification", payload)
	}

	return nil
}

// SendToAll broadcasts a notification to all connected users.
func SendToAll(n *Notification) error {
	return Send("all", n)
}
