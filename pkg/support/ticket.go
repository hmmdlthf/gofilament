package support

import (
	"net/http"

	gsupport "github.com/golaravel/golaravel/pkg/support"
)

// Re-export core types from GoLaravel support package
type Ticket = gsupport.Ticket
type SupportMessage = gsupport.SupportMessage
type VoiceRecording = gsupport.VoiceRecording
type Store = gsupport.Store
type MemorySupportStore = gsupport.MemorySupportStore
type TicketFilter = gsupport.TicketFilter
type VoiceFeatureRequest = gsupport.VoiceFeatureRequest
type FeatureKnowledge = gsupport.FeatureKnowledge
type KnowledgeBase = gsupport.KnowledgeBase
type SpeechProcessor = gsupport.SpeechProcessor
type DefaultSpeechProcessor = gsupport.DefaultSpeechProcessor

// Re-export event types
type TicketCreatedEvent = gsupport.TicketCreatedEvent
type TicketEscalatedEvent = gsupport.TicketEscalatedEvent
type FeatureRequestLoggedEvent = gsupport.FeatureRequestLoggedEvent

// Re-export constants
const (
	CategorySupport        = gsupport.CategorySupport
	CategoryBug            = gsupport.CategoryBug
	CategoryFeatureRequest = gsupport.CategoryFeatureRequest
	CategoryBilling        = gsupport.CategoryBilling
	CategoryGeneral        = gsupport.CategoryGeneral

	ModeAI   = gsupport.ModeAI
	ModeLive = gsupport.ModeLive

	StatusOpen      = gsupport.StatusOpen
	StatusPending   = gsupport.StatusPending
	StatusEscalated = gsupport.StatusEscalated
	StatusResolved  = gsupport.StatusResolved
	StatusClosed    = gsupport.StatusClosed

	PriorityLow    = gsupport.PriorityLow
	PriorityNormal = gsupport.PriorityNormal
	PriorityHigh   = gsupport.PriorityHigh
	PriorityUrgent = gsupport.PriorityUrgent

	FeasibilityExisting    = gsupport.FeasibilityExisting
	FeasibilitySimple      = gsupport.FeasibilitySimple
	FeasibilityComplex     = gsupport.FeasibilityComplex
	FeasibilityUnfeasible  = gsupport.FeasibilityUnfeasible
	FeasibilityUnderReview = gsupport.FeasibilityUnderReview

	LanguageCookieName = gsupport.LanguageCookieName
)

var (
	SupportedLanguages = gsupport.SupportedLanguages
	LanguageFlags      = gsupport.LanguageFlags
)

// GetDefaultStore returns the globally registered support store.
func GetDefaultStore() Store {
	return gsupport.GetDefaultStore()
}

// SetDefaultStore registers the global support store.
func SetDefaultStore(s Store) {
	gsupport.SetDefaultStore(s)
}

// NewMemorySupportStore creates an in-memory support store.
func NewMemorySupportStore() *MemorySupportStore {
	return gsupport.NewMemorySupportStore()
}

// GetDefaultKnowledgeBase returns the system-wide feature knowledge base.
func GetDefaultKnowledgeBase() *KnowledgeBase {
	return gsupport.GetDefaultKnowledgeBase()
}

// SetDefaultKnowledgeBase sets the system-wide feature knowledge base.
func SetDefaultKnowledgeBase(kb *KnowledgeBase) {
	gsupport.SetDefaultKnowledgeBase(kb)
}

// NewKnowledgeBase initializes a knowledge base.
func NewKnowledgeBase() *KnowledgeBase {
	return gsupport.NewKnowledgeBase()
}

// ResolveSupportLanguage extracts preferred language from request cookies or headers.
func ResolveSupportLanguage(r *http.Request) string {
	return gsupport.ResolveSupportLanguage(r)
}

// SetSupportLanguageCookie sets the persistent language preference cookie.
func SetSupportLanguageCookie(w http.ResponseWriter, lang string) {
	gsupport.SetSupportLanguageCookie(w, lang)
}

// GenerateTicketID returns a unique random ticket ID.
func GenerateTicketID() string {
	return gsupport.GenerateTicketID()
}

// GenerateMessageID returns a unique random message ID.
func GenerateMessageID() string {
	return gsupport.GenerateMessageID()
}
