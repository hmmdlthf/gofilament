package activitylog

import (
	"context"
	"time"
)

// Builder facilitates fluent construction and logging of Activity records.
type Builder struct {
	activity *Activity
	store    Store
}

// Log initiates a new activity builder with an optional description.
func Log(description ...string) *Builder {
	desc := ""
	if len(description) > 0 {
		desc = description[0]
	}
	return &Builder{
		activity: &Activity{
			LogName:     "default",
			Description: desc,
			Event:       "custom",
			CreatedAt:   time.Now(),
			OldValues:   make(map[string]any),
			NewValues:   make(map[string]any),
			Properties:  make(map[string]any),
		},
		store: GetDefaultStore(),
	}
}

// LogName assigns a category or channel name to the activity log.
func (b *Builder) LogName(name string) *Builder {
	b.activity.LogName = name
	return b
}

// Subject sets the target entity type and identifier.
func (b *Builder) Subject(subjectType, subjectID string) *Builder {
	b.activity.SubjectType = subjectType
	b.activity.SubjectID = subjectID
	return b
}

// Causer sets the acting entity or user triggering the event.
func (b *Builder) Causer(causerType, causerID string, causerName ...string) *Builder {
	b.activity.CauserType = causerType
	b.activity.CauserID = causerID
	if len(causerName) > 0 {
		b.activity.CauserName = causerName[0]
	}
	return b
}

// CauserAvatar sets an avatar image URL for the causer.
func (b *Builder) CauserAvatar(url string) *Builder {
	b.activity.CauserAvatar = url
	return b
}

// Event sets the lifecycle action keyword (e.g. "created", "updated", "deleted", "approved").
func (b *Builder) Event(event string) *Builder {
	b.activity.Event = event
	return b
}

// Description sets the human-readable explanation of the action.
func (b *Builder) Description(desc string) *Builder {
	b.activity.Description = desc
	return b
}

// Old sets the baseline state before changes.
func (b *Builder) Old(old map[string]any) *Builder {
	b.activity.OldValues = old
	return b
}

// New sets the resulting state after changes.
func (b *Builder) New(new map[string]any) *Builder {
	b.activity.NewValues = new
	return b
}

// WithDiff stores both pre-change and post-change attributes for diff tracking.
func (b *Builder) WithDiff(old, new map[string]any) *Builder {
	b.activity.OldValues = old
	b.activity.NewValues = new
	return b
}

// Property attaches a custom key-value pair to properties metadata.
func (b *Builder) Property(key string, val any) *Builder {
	if b.activity.Properties == nil {
		b.activity.Properties = make(map[string]any)
	}
	b.activity.Properties[key] = val
	return b
}

// Properties replaces or sets the properties metadata map.
func (b *Builder) Properties(props map[string]any) *Builder {
	b.activity.Properties = props
	return b
}

// IP sets the client IP address.
func (b *Builder) IP(ip string) *Builder {
	b.activity.IPAddress = ip
	return b
}

// UserAgent sets the client User-Agent header.
func (b *Builder) UserAgent(ua string) *Builder {
	b.activity.UserAgent = ua
	return b
}

// CreatedAt overrides the event timestamp.
func (b *Builder) CreatedAt(t time.Time) *Builder {
	b.activity.CreatedAt = t
	return b
}

// Store specifies a custom Store instance for persisting this activity.
func (b *Builder) Store(s Store) *Builder {
	if s != nil {
		b.store = s
	}
	return b
}

// Build finalizes and returns the built Activity struct without persisting.
func (b *Builder) Build() *Activity {
	return b.activity
}

// Save records the activity into the configured store.
func (b *Builder) Save(ctx ...context.Context) (*Activity, error) {
	c := context.Background()
	if len(ctx) > 0 && ctx[0] != nil {
		c = ctx[0]
	}
	err := b.store.Log(c, b.activity)
	if err != nil {
		return nil, err
	}
	return b.activity, nil
}
