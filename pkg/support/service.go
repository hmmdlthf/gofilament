package support

import (
	gsupport "github.com/golaravel/golaravel/pkg/support"
)

// Service provides high-level support assistance, live agent escalation, and voice feature requests.
type Service = gsupport.Service

// NewService creates a support service instance.
func NewService(store Store, kb ...*KnowledgeBase) *Service {
	return gsupport.NewService(store, kb...)
}

// GetDefaultService returns the global support concierge service instance.
func GetDefaultService() *Service {
	return gsupport.GetDefaultService()
}

// SetDefaultService configures the global support concierge service instance.
func SetDefaultService(svc *Service) {
	gsupport.SetDefaultService(svc)
}

// NormalizeLanguage trims and lowercases a language code.
func NormalizeLanguage(lang string) string {
	return gsupport.NormalizeLanguage(lang)
}

// IsValidLanguage checks whether a language code is supported.
func IsValidLanguage(lang string) bool {
	return gsupport.IsValidLanguage(lang)
}
