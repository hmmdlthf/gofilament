package filament

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"

	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
	"github.com/golaravel/golaravel/pkg/session"
)

// Impersonation: an admin signs in as another user to see the panel as
// they do, with a banner to switch back. It works on the web engine with
// golaravel sessions (Panel.Sessions); API tokens are not issued for other
// users.

// Impersonation configures "sign in as".
type Impersonation struct {
	// CanImpersonate decides whether the signed-in user may sign in as
	// target. Required: without it nobody can impersonate.
	CanImpersonate func(ctx *Context, target auth.Authenticatable) bool
	// OnImpersonate is told when impersonation starts (leaving=false) and
	// ends (leaving=true), for example to write an audit log entry.
	OnImpersonate func(ctx *Context, impersonator, target auth.Authenticatable, leaving bool)
}

const sessImpersonator = "_filament_impersonator"

// Impersonation turns on "sign in as". Needs Panel.Sessions.
func (p *Panel) Impersonation(cfg Impersonation) *Panel {
	p.impersonation = &cfg
	return p
}

// ImpersonatorID returns the id of the admin who is signed in as the
// current user ("" when nobody is impersonating).
func ImpersonatorID(ctx *Context) string {
	if sess := session.FromContext(ctx.Context); sess != nil {
		return sess.GetString(sessImpersonator)
	}
	return ""
}

// ImpersonateURL is the link that starts impersonating userID (use it in a
// table action: tables.Action("impersonate").URL(...)).
func (p *Panel) ImpersonateURL(userID string) string {
	return p.path + "/impersonate/" + userID
}

func (p *Panel) mountImpersonationRoutes(r *router.Router) {
	if p.impersonation == nil || p.impersonation.CanImpersonate == nil {
		return
	}
	start := func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		me := ctx.User()
		sess := session.FromContext(c)
		if me == nil || sess == nil || p.sessionUsers == nil {
			return c.String(http.StatusNotFound, "Not found")
		}
		if ImpersonatorID(ctx) != "" {
			return c.String(http.StatusConflict, "Return to your own account before signing in as someone else.")
		}
		targetID := c.Param("user")
		if targetID == me.GetAuthIdentifier() {
			return c.String(http.StatusUnprocessableEntity, "You are already signed in as this user.")
		}
		target, err := p.sessionUsers(ctx, targetID)
		if err != nil || target == nil || !p.impersonation.CanImpersonate(ctx, target) {
			return c.String(http.StatusForbidden, "403 Forbidden")
		}
		if err := p.sessionLogin(c, target); err != nil {
			return c.String(http.StatusServiceUnavailable, "Sign-in is unavailable right now.")
		}
		sess.Put(sessImpersonator, me.GetAuthIdentifier())
		log.Printf("filament: %s is signed in as %s", me.GetAuthIdentifier(), targetID)
		if p.impersonation.OnImpersonate != nil {
			p.impersonation.OnImpersonate(ctx, me, target, false)
		}
		return c.Redirect(http.StatusSeeOther, p.path)
	}
	start = p.guardUnsafeGET("Sign in as this user", "Sign in as this user? You can switch back at any time.", start)
	r.GET(p.path+"/impersonate/:user", start)
	r.POST(p.path+"/impersonate/:user", start)

	leave := func(c *gcontext.Context) error {
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		origID := ImpersonatorID(ctx)
		if origID == "" {
			return c.Redirect(http.StatusSeeOther, p.path)
		}
		target := ctx.User()
		orig, err := p.sessionUsers(ctx, origID)
		if err != nil || orig == nil {
			// The admin's account is gone: end the session instead.
			_ = auth.Logout(c)
			return c.Redirect(http.StatusSeeOther, p.path+"/login")
		}
		if err := p.sessionLogin(c, orig); err != nil {
			return c.String(http.StatusServiceUnavailable, "Sign-in is unavailable right now.")
		}
		if sess := session.FromContext(c); sess != nil {
			sess.Forget(sessImpersonator)
		}
		log.Printf("filament: %s returned from %v", origID, idOf(target))
		if p.impersonation.OnImpersonate != nil && target != nil {
			p.impersonation.OnImpersonate(ctx, orig, target, true)
		}
		return c.Redirect(http.StatusSeeOther, p.path)
	}
	leave = p.guardUnsafeGET("Return to your account", "Stop signing in as this user?", leave)
	r.GET(p.path+"/impersonate-leave", leave)
	r.POST(p.path+"/impersonate-leave", leave)
}

func idOf(u auth.Authenticatable) string {
	if u == nil {
		return ""
	}
	return u.GetAuthIdentifier()
}

// impersonationBannerHTML is shown on every page while impersonating.
func (p *Panel) impersonationBannerHTML(ctx *Context) string {
	if p.impersonation == nil || ImpersonatorID(ctx) == "" {
		return ""
	}
	return fmt.Sprintf(`<div class="filament-impersonation-banner" role="status" style="display: flex; gap: 0.75rem; align-items: center; padding: 0.375rem 0.75rem; background: #fef3c7; color: #92400e; border-radius: 0.5rem; font-size: 0.8125rem;">Signed in as <strong>%s</strong> <form method="POST" action="%s/impersonate-leave" style="margin: 0;">%s<button type="submit" class="btn btn-secondary btn-sm">Return to my account</button></form></div>`,
		html.EscapeString(idOf(ctx.User())), p.path, p.csrfHiddenInput(ctx.Context, ""))
}

// blockedWhileImpersonating lists account changes an impersonator may not
// make for the user (password, two-factor, profile email).
func (p *Panel) blockedWhileImpersonating(ctx *Context) bool {
	if ImpersonatorID(ctx) == "" || ctx.Request == nil {
		return false
	}
	path := strings.TrimPrefix(ctx.Request.URL.Path, strings.TrimRight(p.path, "/"))
	if ctx.Request.Method == http.MethodGet {
		return false
	}
	for _, pre := range []string{"/profile", "/api/account"} {
		if strings.HasPrefix(path, pre) {
			return true
		}
	}
	return false
}
