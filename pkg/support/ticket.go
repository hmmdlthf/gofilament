package support

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// TicketMode defines the conversation mode: AI automated bot or Human live agent.
const (
	ModeAI   = "ai"
	ModeLive = "live"
)

// TicketStatus defines current lifecycle status.
const (
	StatusOpen     = "open"
	StatusPending  = "pending"
	StatusResolved = "resolved"
	StatusClosed   = "closed"
)

// SupportMessage represents a message exchanged in a support session.
type SupportMessage struct {
	ID         string    `json:"id"`
	TicketID   string    `json:"ticket_id"`
	SenderType string    `json:"sender_type"` // "user", "ai", "agent", "system"
	SenderName string    `json:"sender_name"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

// Ticket represents a customer support inquiry or live chat session.
type Ticket struct {
	ID         string           `json:"id"`
	UserKey    string           `json:"user_key"`
	UserName   string           `json:"user_name"`
	UserEmail  string           `json:"user_email,omitempty"`
	Subject    string           `json:"subject"`
	Mode       string           `json:"mode"`       // "ai" or "live"
	Status     string           `json:"status"`     // "open", "pending", "resolved", "closed"
	Priority   string           `json:"priority"`   // "normal", "high", "urgent"
	Department string           `json:"department"` // "Shariah Matrimony", "Wali Supervision", "Technical & Billing"
	Messages   []SupportMessage `json:"messages"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// TimeAgo returns a human-friendly duration since ticket creation.
func (t *Ticket) TimeAgo() string {
	diff := time.Since(t.CreatedAt)
	if diff < time.Minute {
		return "Just now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
}

// LastMessagePreview returns a snippet of the latest message.
func (t *Ticket) LastMessagePreview() string {
	if len(t.Messages) == 0 {
		return "No messages"
	}
	last := t.Messages[len(t.Messages)-1]
	prefix := ""
	if last.SenderType == "ai" {
		prefix = "🤖 "
	} else if last.SenderType == "agent" {
		prefix = "👨‍💼 "
	}
	txt := last.Content
	if len(txt) > 60 {
		txt = txt[:57] + "..."
	}
	return prefix + txt
}

// Store defines repository storage for support tickets and chat transcripts.
type Store interface {
	CreateTicket(ctx context.Context, t *Ticket) error
	GetTicket(ctx context.Context, id string) (*Ticket, error)
	ListTickets(ctx context.Context) ([]*Ticket, error)
	AddMessage(ctx context.Context, ticketID string, msg *SupportMessage) error
	UpdateTicketStatus(ctx context.Context, id string, status string) error
	UpdateTicketMode(ctx context.Context, id string, mode string) error
}

// MemorySupportStore provides an in-memory thread-safe support ticket repository.
type MemorySupportStore struct {
	mu      sync.RWMutex
	tickets map[string]*Ticket
}

var (
	defaultStoreMu sync.RWMutex
	defaultStore   Store
)

// GetDefaultStore returns the globally registered support store.
func GetDefaultStore() Store {
	defaultStoreMu.RLock()
	defer defaultStoreMu.RUnlock()
	if defaultStore == nil {
		defaultStore = NewMemorySupportStore()
	}
	return defaultStore
}

// SetDefaultStore registers the global support store.
func SetDefaultStore(s Store) {
	defaultStoreMu.Lock()
	defer defaultStoreMu.Unlock()
	defaultStore = s
}

// NewMemorySupportStore creates an in-memory store seeded with realistic initial support conversations.
func NewMemorySupportStore() *MemorySupportStore {
	s := &MemorySupportStore{
		tickets: make(map[string]*Ticket),
	}
	s.seedDefaultTickets()
	return s
}

func (s *MemorySupportStore) CreateTicket(ctx context.Context, t *Ticket) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t.ID == "" {
		t.ID = generateTicketID()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}
	t.UpdatedAt = time.Now()
	s.tickets[t.ID] = t
	return nil
}

func (s *MemorySupportStore) GetTicket(ctx context.Context, id string) (*Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, exists := s.tickets[id]
	if !exists {
		return nil, fmt.Errorf("ticket %s not found", id)
	}
	return t, nil
}

func (s *MemorySupportStore) ListTickets(ctx context.Context) ([]*Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*Ticket, 0, len(s.tickets))
	for _, t := range s.tickets {
		list = append(list, t)
	}
	return list, nil
}

func (s *MemorySupportStore) AddMessage(ctx context.Context, ticketID string, msg *SupportMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, exists := s.tickets[ticketID]
	if !exists {
		return fmt.Errorf("ticket %s not found", ticketID)
	}
	if msg.ID == "" {
		msg.ID = generateTicketID()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	t.Messages = append(t.Messages, *msg)
	t.UpdatedAt = time.Now()
	return nil
}

func (s *MemorySupportStore) UpdateTicketStatus(ctx context.Context, id string, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, exists := s.tickets[id]
	if !exists {
		return fmt.Errorf("ticket %s not found", id)
	}
	t.Status = status
	t.UpdatedAt = time.Now()
	return nil
}

func (s *MemorySupportStore) UpdateTicketMode(ctx context.Context, id string, mode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, exists := s.tickets[id]
	if !exists {
		return fmt.Errorf("ticket %s not found", id)
	}
	t.Mode = mode
	t.UpdatedAt = time.Now()
	return nil
}

func generateTicketID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("TKT-%s", hex.EncodeToString(b))
}

func (s *MemorySupportStore) seedDefaultTickets() {
	now := time.Now()

	t1 := &Ticket{
		ID:         "TKT-8841",
		UserKey:    "zahra",
		UserName:   "Zahra Al-Mansoor",
		UserEmail:  "zahra@twodeen.com",
		Subject:    "Wali Invitation & Verification Process",
		Mode:       ModeLive,
		Status:     StatusOpen,
		Priority:   "high",
		Department: "Wali Supervision",
		CreatedAt:  now.Add(-45 * time.Minute),
		UpdatedAt:  now.Add(-10 * time.Minute),
		Messages: []SupportMessage{
			{
				ID:         "MSG-001",
				TicketID:   "TKT-8841",
				SenderType: "user",
				SenderName: "Zahra Al-Mansoor",
				Content:    "Assalamu Alaikum. How do I invite my father as my Wali to oversee chat requests from suitors?",
				CreatedAt:  now.Add(-45 * time.Minute),
			},
			{
				ID:         "MSG-002",
				TicketID:   "TKT-8841",
				SenderType: "agent",
				SenderName: "Ustadh Hamza (Support Concierge)",
				Content:    "Wa Alaikum Assalam sister Zahra. You can send an invitation SMS/email directly from your Profile > Wali Settings. He will receive an enclave access link with full Shariah chat oversight.",
				CreatedAt:  now.Add(-25 * time.Minute),
			},
			{
				ID:         "MSG-003",
				TicketID:   "TKT-8841",
				SenderType: "user",
				SenderName: "Zahra Al-Mansoor",
				Content:    "JazakAllah Khair! Does he get notified before or after stage 2 photo release?",
				CreatedAt:  now.Add(-10 * time.Minute),
			},
		},
	}

	t2 := &Ticket{
		ID:         "TKT-9023",
		UserKey:    "althaf",
		UserName:   "Althaf Hussain",
		UserEmail:  "althaf@twodeen.com",
		Subject:    "Inquiry on Staged Privacy Vault and Mahr Guidelines",
		Mode:       ModeAI,
		Status:     StatusResolved,
		Priority:   "normal",
		Department: "Shariah Matrimony",
		CreatedAt:  now.Add(-2 * time.Hour),
		UpdatedAt:  now.Add(-90 * time.Minute),
		Messages: []SupportMessage{
			{
				ID:         "MSG-010",
				TicketID:   "TKT-9023",
				SenderType: "user",
				SenderName: "Althaf Hussain",
				Content:    "What are the requirements for unlocking Stage 3 in the privacy vault?",
				CreatedAt:  now.Add(-2 * time.Hour),
			},
			{
				ID:         "MSG-011",
				TicketID:   "TKT-9023",
				SenderType: "ai",
				SenderName: "TwoDeen Islamic AI Advisor",
				Content:    "Stage 3 (Full Halal Biodata & Family Verification) requires: 1) Mutual consent from both candidates, 2) Active Guardian/Wali endorsement, and 3) ID verification on file.",
				CreatedAt:  now.Add(-119 * time.Minute),
			},
		},
	}

	s.tickets[t1.ID] = t1
	s.tickets[t2.ID] = t2
}
