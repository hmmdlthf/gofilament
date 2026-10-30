package notifications

import (
	"encoding/json"
	"fmt"
	"time"
)

type NotificationType string

const (
	TypeSuccess NotificationType = "success"
	TypeInfo    NotificationType = "info"
	TypeWarning NotificationType = "warning"
	TypeDanger  NotificationType = "danger"
)

// Notification represents a toast or database alert.
type Notification struct {
	ID        string           `json:"id"`
	Type      NotificationType `json:"type"`
	Title     string           `json:"title"`
	BodyText  string           `json:"body,omitempty"`
	Duration  int              `json:"duration,omitempty"` // in ms
	CreatedAt time.Time        `json:"created_at"`
	Read      bool             `json:"read"`
}

// Make creates a new Notification builder.
func Make(title string) *Notification {
	return &Notification{
		ID:        fmt.Sprintf("notif_%d", time.Now().UnixNano()),
		Type:      TypeInfo,
		Title:     title,
		Duration:  4000,
		CreatedAt: time.Now(),
	}
}

func (n *Notification) Success() *Notification {
	n.Type = TypeSuccess
	return n
}

func (n *Notification) Warning() *Notification {
	n.Type = TypeWarning
	return n
}

func (n *Notification) Danger() *Notification {
	n.Type = TypeDanger
	return n
}

func (n *Notification) Info() *Notification {
	n.Type = TypeInfo
	return n
}

func (n *Notification) Body(body string) *Notification {
	n.BodyText = body
	return n
}

func (n *Notification) DurationMs(ms int) *Notification {
	n.Duration = ms
	return n
}

// ToJSON serializes notification for frontend client or SSE push.
func (n *Notification) ToJSON() (string, error) {
	bytes, err := json.Marshal(n)
	return string(bytes), err
}
