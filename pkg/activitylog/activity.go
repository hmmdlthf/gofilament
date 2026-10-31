package activitylog

import (
	"encoding/json"
	"fmt"
	"html"
	"reflect"
	"sort"
	"strings"
	"time"
)

// FieldDiff captures the change in a single field between old and new state.
type FieldDiff struct {
	Field   string `json:"field"`
	Old     any    `json:"old"`
	New     any    `json:"new"`
	Changed bool   `json:"changed"`
}

// Activity represents an individual audit trail event.
type Activity struct {
	ID           string         `json:"id"`
	LogName      string         `json:"log_name"`
	SubjectType  string         `json:"subject_type"`
	SubjectID    string         `json:"subject_id"`
	CauserType   string         `json:"causer_type"`
	CauserID     string         `json:"causer_id"`
	CauserName   string         `json:"causer_name"`
	CauserAvatar string         `json:"causer_avatar,omitempty"`
	Event        string         `json:"event"`
	Description  string         `json:"description"`
	OldValues    map[string]any `json:"old_values,omitempty"`
	NewValues    map[string]any `json:"new_values,omitempty"`
	Properties   map[string]any `json:"properties,omitempty"`
	IPAddress    string         `json:"ip_address,omitempty"`
	UserAgent    string         `json:"user_agent,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// GetDiff calculates the field differences between OldValues and NewValues.
func (a *Activity) GetDiff() []FieldDiff {
	return Diff(a.OldValues, a.NewValues)
}

// HasDiff returns true if there are differences recorded in this activity.
func (a *Activity) HasDiff() bool {
	return len(a.GetDiff()) > 0
}

// TimeAgo returns a human-friendly relative time string.
func (a *Activity) TimeAgo() string {
	return FormatTimeAgo(a.CreatedAt)
}

// EventColor returns the visual palette color associated with the event.
func (a *Activity) EventColor() string {
	switch strings.ToLower(a.Event) {
	case "created", "added", "inserted":
		return "emerald"
	case "updated", "modified", "edited":
		return "indigo"
	case "deleted", "removed", "destroyed":
		return "rose"
	case "restored":
		return "cyan"
	case "approved", "verified", "completed":
		return "green"
	case "rejected", "declined", "cancelled":
		return "red"
	case "login", "authenticated":
		return "amber"
	default:
		return "gray"
	}
}

// Diff compares two maps and returns a slice of FieldDiff for all modified or added/removed fields.
func Diff(oldVals, newVals map[string]any) []FieldDiff {
	if len(oldVals) == 0 && len(newVals) == 0 {
		return nil
	}

	allKeys := make(map[string]bool)
	for k := range oldVals {
		allKeys[k] = true
	}
	for k := range newVals {
		allKeys[k] = true
	}

	keys := make([]string, 0, len(allKeys))
	for k := range allKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	diffs := make([]FieldDiff, 0)
	for _, k := range keys {
		oldV, hasOld := oldVals[k]
		newV, hasNew := newVals[k]

		// Skip unchanged
		if hasOld && hasNew {
			if reflect.DeepEqual(oldV, newV) {
				continue
			}
			if fmt.Sprintf("%v", oldV) == fmt.Sprintf("%v", newV) {
				continue
			}
		}

		diffs = append(diffs, FieldDiff{
			Field:   k,
			Old:     oldV,
			New:     newV,
			Changed: true,
		})
	}

	return diffs
}

// FormatTimeAgo converts a timestamp to a friendly relative representation.
func FormatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	if d < 0 {
		return "just now"
	}
	if d < 45*time.Second {
		return "just now"
	}
	if d < time.Hour {
		m := int(d.Minutes())
		if m <= 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		if h <= 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "yesterday"
	}
	if days < 30 {
		return fmt.Sprintf("%d days ago", days)
	}
	months := days / 30
	if months <= 1 {
		return "1 month ago"
	}
	if months < 12 {
		return fmt.Sprintf("%d months ago", months)
	}
	years := months / 12
	if years <= 1 {
		return "1 year ago"
	}
	return fmt.Sprintf("%d years ago", years)
}

// FormatValue returns a human-readable display string for any value in diff tables.
func FormatValue(val any) string {
	if val == nil {
		return "null"
	}
	switch v := val.(type) {
	case string:
		if v == "" {
			return `""`
		}
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case time.Time:
		return v.Format("2006-01-02 15:04:05")
	case map[string]any, []any:
		b, err := json.Marshal(v)
		if err == nil {
			return string(b)
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// GetInitials extracts a 1-2 character abbreviation for avatar fallbacks.
func GetInitials(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "?"
	}
	parts := strings.Fields(name)
	if len(parts) == 1 {
		runes := []rune(parts[0])
		if len(runes) >= 2 {
			return strings.ToUpper(string(runes[:2]))
		}
		return strings.ToUpper(string(runes[:1]))
	}
	r1 := []rune(parts[0])
	r2 := []rune(parts[len(parts)-1])
	return strings.ToUpper(string(r1[:1]) + string(r2[:1]))
}

// EscapeHTML is a convenience alias for template rendering.
func EscapeHTML(s string) string {
	return html.EscapeString(s)
}
