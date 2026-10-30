package forms

import (
	"context"
	"time"
)

// Get is a function signature for reading form state reactively.
type Get func(key string) any

// Set is a function signature for writing form state reactively.
type Set func(key string, val any)

// Component defines the common contract for all form fields and layout containers.
type Component interface {
	GetName() string
	Render(ctx context.Context, state map[string]any, errors map[string]string) string
}

// LiveOption configures reactive live debouncing.
type LiveOption struct {
	Debounce time.Duration
}

// Debounce creates a live debounce option.
func Debounce(d time.Duration) LiveOption {
	return LiveOption{Debounce: d}
}
