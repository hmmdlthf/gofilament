package filament

import (
	"errors"

	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/session"
)

// Sessions runs the Templ (web) engine login on golaravel's session
// manager: the session cookie, its driver (memory, file, cookie, SQL,
// Redis), lifetime and key all come from mgr (nil: session.DefaultManager,
// keyed by APP_KEY). Sign-in uses auth.Login (new session id, new CSRF
// token), sign-out auth.Logout, CSRF the session token, and users loads the
// signed-in user on every request, so disabling a user ends their access.
//
// The credential check stays the panel's AuthValidator; its token is not
// used for the web engine. The session lifetime is the manager's Lifetime;
// SessionDuration / RememberMeDuration apply to API tokens only.
//
// If the application already runs session.StartSession on the panel's
// routes, that session is used as is.
func (p *Panel) Sessions(mgr *session.Manager, users UserLoader) *Panel {
	if mgr == nil {
		mgr = session.DefaultManager()
	}
	p.sessionManager, p.sessionUsers = mgr, users
	p.requiresAuth = true
	return p
}

// UsesGolaravelSessions reports whether Sessions was configured.
func (p *Panel) UsesGolaravelSessions() bool { return p.sessionManager != nil }

// sessionMiddleware starts the golaravel session on web-engine requests
// (not the bearer-token API or static assets).
func (p *Panel) sessionMiddleware() gcontext.HandlerFunc {
	start := session.StartSession(p.sessionManager)
	return func(c *gcontext.Context) error {
		if session.FromContext(c) != nil || p.isAPIRequest(c) || !p.middlewareApplies(c.Request.Method, c.Request.URL.Path) {
			return nil
		}
		return start(c)
	}
}

func (p *Panel) sessionUser(ctx *Context) auth.Authenticatable {
	uid := auth.SessionUserID(ctx.Context)
	if uid == "" || p.sessionUsers == nil {
		return nil
	}
	u, err := p.sessionUsers(ctx, uid)
	if err != nil || u == nil {
		_ = auth.Logout(ctx.Context)
		return nil
	}
	return u
}

var errNoPanelSession = errors.New("filament: the golaravel session did not start for this request")

func (p *Panel) sessionLogin(c *gcontext.Context, user auth.Authenticatable) error {
	if session.FromContext(c) == nil {
		return errNoPanelSession
	}
	return auth.Login(c, user)
}

// BearerGuard authenticates API (and Flutter) requests that send
// "Authorization: Bearer <token>" with a golaravel guard, for example
// auth.BearerGuardFunc(jwt, users), which also refuses 2FA challenge
// tokens. It is tried before TokenValidator.
func (p *Panel) BearerGuard(guard auth.GuardFunc) *Panel {
	p.bearerGuard = guard
	p.requiresAuth = true
	return p
}
