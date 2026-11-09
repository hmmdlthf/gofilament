package support

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Service provides high-level support assistance and AI responses.
type Service struct {
	store Store
}

// NewService creates a support service instance.
func NewService(store Store) *Service {
	if store == nil {
		store = GetDefaultStore()
	}
	return &Service{store: store}
}

// Store returns the underlying ticket store.
func (s *Service) Store() Store {
	return s.store
}

// SendUserMessage submits a user question or reply. If the ticket is in AI mode, generates an intelligent answer.
func (s *Service) SendUserMessage(ctx context.Context, ticketID, userKey, userName, content string) (*SupportMessage, *SupportMessage, error) {
	ticket, err := s.store.GetTicket(ctx, ticketID)
	if err != nil {
		// Auto-create ticket if not exists
		ticket = &Ticket{
			ID:         ticketID,
			UserKey:    userKey,
			UserName:   userName,
			Subject:    deriveSubject(content),
			Mode:       ModeAI,
			Status:     StatusOpen,
			Priority:   "normal",
			Department: "Shariah Matrimony",
		}
		if err := s.store.CreateTicket(ctx, ticket); err != nil {
			return nil, nil, fmt.Errorf("failed creating ticket: %w", err)
		}
	}

	userMsg := &SupportMessage{
		TicketID:   ticket.ID,
		SenderType: "user",
		SenderName: userName,
		Content:    content,
		CreatedAt:  time.Now(),
	}
	if err := s.store.AddMessage(ctx, ticket.ID, userMsg); err != nil {
		return nil, nil, err
	}

	// If in AI mode, automatically generate a helpful Halal response
	var botMsg *SupportMessage
	if ticket.Mode == ModeAI {
		replyContent := GenerateAIResponse(content)
		botMsg = &SupportMessage{
			TicketID:   ticket.ID,
			SenderType: "ai",
			SenderName: "TwoDeen Islamic AI Advisor",
			Content:    replyContent,
			CreatedAt:  time.Now().Add(500 * time.Millisecond),
		}
		_ = s.store.AddMessage(ctx, ticket.ID, botMsg)
	}

	return userMsg, botMsg, nil
}

// SwitchMode switches a ticket between AI automated mode and Live Human agent mode.
func (s *Service) SwitchMode(ctx context.Context, ticketID, mode string) error {
	if mode != ModeAI && mode != ModeLive {
		return fmt.Errorf("invalid mode %s; must be 'ai' or 'live'", mode)
	}
	return s.store.UpdateTicketMode(ctx, ticketID, mode)
}

// GenerateAIResponse generates intelligent matrimonial and platform guidance.
func GenerateAIResponse(query string) string {
	q := strings.ToLower(query)

	if strings.Contains(q, "wali") || strings.Contains(q, "guardian") || strings.Contains(q, "father") {
		return "In TwoDeen, Wali (Guardian) involvement is a foundational Shariah pillar. Sisters can invite their Wali via their profile, granting him full visibility into suitor inquiries, communication logs, and stage approval requests before any personal contact."
	}

	if strings.Contains(q, "photo") || strings.Contains(q, "picture") || strings.Contains(q, "see") || strings.Contains(q, "face") {
		return "To uphold Islamic modesty (Haya), candidate photos are never public. Profiles display dignified modest silhouettes. Blurred or verified photos are only unlocked in Stage 2 with explicit mutual candidate and Wali consent."
	}

	if strings.Contains(q, "stage") || strings.Contains(q, "privacy") || strings.Contains(q, "vault") {
		return "TwoDeen operates a 4-Stage Shariah Privacy Vault: Stage 1 (Modest Biodata & Deen Assessment), Stage 2 (Mutual Photo Unveiling with Wali approval), Stage 3 (Family Introduction & Contact Sharing), and Stage 4 (Formal Khatbah / Nikah Finalization)."
	}

	if strings.Contains(q, "name") || strings.Contains(q, "hidden") || strings.Contains(q, "real name") {
		return "By default on TwoDeen, candidate real names are strictly hidden to safeguard privacy. Profiles are identified by their unique Member ID (e.g. USR-20143). Candidates may optionally choose to reveal their name in their profile settings."
	}

	if strings.Contains(q, "divorce") || strings.Contains(q, "talaq") || strings.Contains(q, "khula") || strings.Contains(q, "iddah") {
		return "TwoDeen supports transparent and compassionate Islamic divorce handling. We track Fiqh dissolution types (Talaq, Khula, Faskh, Nikah-Only Unconsummated) and verified Iddah completion, ensuring honest compatibility without stigma."
	}

	if strings.Contains(q, "price") || strings.Contains(q, "cost") || strings.Contains(q, "vip") || strings.Contains(q, "subscription") || strings.Contains(q, "plan") {
		return "TwoDeen offers Basic Halal Browsing complimentary. VIP Concierge and Wali Assisted Family packages provide verified background checks, dedicated Islamic relationship advisors, and unlimited privacy stage unlocks."
	}

	if strings.Contains(q, "agent") || strings.Contains(q, "human") || strings.Contains(q, "talk to someone") || strings.Contains(q, "support") {
		return "You can switch to 'Live Chat' at any time using the mode toggle above. A human TwoDeen Concierge agent will join your session to assist you directly."
	}

	return "Assalamu Alaikum! I am your TwoDeen Islamic Matrimonial Assistant. I can help guide you through Shariah-compliant matching, Wali invitations, 4-stage privacy vaults, and profile verification. How may I assist you today?"
}

func deriveSubject(content string) string {
	content = strings.TrimSpace(content)
	if len(content) > 40 {
		return content[:37] + "..."
	}
	if content == "" {
		return "Support Inquiry"
	}
	return content
}
