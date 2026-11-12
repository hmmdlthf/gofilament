package filament

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/golaravel/golaravel/pkg/auth"
)

// WebSessionStore keeps web-engine sessions somewhere that survives restarts
// and is shared between processes (sqlstore.WebSessions is one). Session ids
// reach the store already hashed, so the store never holds a usable cookie.
type WebSessionStore interface {
	PutSession(ctx context.Context, id, userID string, expires time.Time) error
	// GetSession returns ok=false for unknown and expired sessions.
	GetSession(ctx context.Context, id string) (userID string, ok bool, err error)
	DeleteSession(ctx context.Context, id string) error
}

// UserLoader loads a signed-in user by the identifier the session stored
// (auth.Authenticatable.GetAuthIdentifier). Return an error or nil for users
// that no longer exist or may no longer sign in.
type UserLoader func(ctx *Context, id string) (auth.Authenticatable, error)

// WebSessionsUsing stores web-engine sessions (used when the panel has no
// TokenValidator) in store instead of process memory; users loads the user
// on each request, so disabling a user ends their sessions.
func (p *Panel) WebSessionsUsing(store WebSessionStore, users UserLoader) *Panel {
	p.sessionStore, p.sessionUsers = store, users
	return p
}

func hashSessionID(id string) string {
	sum := sha256.Sum256([]byte("gofilament-session:" + id))
	return hex.EncodeToString(sum[:])
}

func (p *Panel) storedSessionUser(ctx *Context, id string) (auth.Authenticatable, bool) {
	userID, ok, err := p.sessionStore.GetSession(ctx.GoContext(), hashSessionID(id))
	if err != nil || !ok || userID == "" || p.sessionUsers == nil {
		return nil, false
	}
	u, err := p.sessionUsers(ctx, userID)
	if err != nil || u == nil {
		return nil, false
	}
	return u, true
}
