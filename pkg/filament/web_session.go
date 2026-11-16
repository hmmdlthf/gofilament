package filament

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
)

// CSRFFormField is the form field carrying the CSRF token on panel forms.
// It matches golaravel's middleware.CSRFFormField, so panel forms also pass
// golaravel's global CSRF middleware when the application uses it.
const CSRFFormField = "_token"

// golaravelCSRFContextKey is where golaravel's CSRF middleware stores the
// request token (middleware.CSRFContextKey).
const golaravelCSRFContextKey = "csrf_token"

// webSession is a signed-in browser session of the Templ (web) engine.
type webSession struct {
	user    auth.Authenticatable
	expires time.Time
}

var warnMemorySessions sync.Once

// webSessionStore keeps web-engine sessions in memory. It is used when the
// panel has no TokenValidator; sessions are lost on restart and are not
// shared between processes. Configure a TokenValidator to make web sessions
// survive restarts: the cookie then carries the AuthValidator's token.
type webSessionStore struct {
	mu       sync.Mutex
	sessions map[string]webSession
}

func (s *webSessionStore) put(id string, sess webSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions == nil {
		s.sessions = make(map[string]webSession)
	}
	now := time.Now()
	for k, v := range s.sessions {
		if now.After(v.expires) {
			delete(s.sessions, k)
		}
	}
	s.sessions[id] = sess
}

func (s *webSessionStore) get(id string) (auth.Authenticatable, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return nil, false
	}
	if time.Now().After(sess.expires) {
		delete(s.sessions, id)
		return nil, false
	}
	return sess.user, true
}

func (s *webSessionStore) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
}

var cookieNameUnsafe = regexp.MustCompile(`[^A-Za-z0-9_-]`)

func (p *Panel) cookiePrefix() string {
	return "gofilament_" + cookieNameUnsafe.ReplaceAllString(p.id, "_")
}

// SessionCookieName is the cookie holding the web-engine login session.
func (p *Panel) SessionCookieName() string { return p.cookiePrefix() + "_session" }

func (p *Panel) csrfCookieName() string { return p.cookiePrefix() + "_csrf" }

func (p *Panel) cookiePath() string {
	if p.path == "" {
		return "/"
	}
	return p.path
}

func isSecureRequest(c *gcontext.Context) bool {
	if c.Request == nil {
		return false
	}
	return c.Request.TLS != nil || strings.EqualFold(c.Header("X-Forwarded-Proto"), "https")
}

func (p *Panel) setPanelCookie(c *gcontext.Context, name, value string, maxAge time.Duration) {
	ck := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     p.cookiePath(),
		HttpOnly: true,
		Secure:   isSecureRequest(c),
		SameSite: http.SameSiteLaxMode,
	}
	if maxAge < 0 {
		ck.MaxAge = -1
		ck.Expires = time.Unix(0, 0)
	} else if maxAge > 0 {
		ck.MaxAge = int(maxAge.Seconds())
		ck.Expires = time.Now().Add(maxAge)
	}
	http.SetCookie(&c.Response, ck)
}

func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func cookieValue(c *gcontext.Context, name string) string {
	if c.Request == nil {
		return ""
	}
	ck, err := c.Request.Cookie(name)
	if err != nil {
		return ""
	}
	return ck.Value
}

// CSRFToken returns the CSRF token to embed (as the "_token" form field) in
// panel forms that POST, issuing the token cookie when needed. When
// golaravel's CSRF middleware runs on the request its token is reused.
func (p *Panel) CSRFToken(c *gcontext.Context) string {
	if v, ok := c.Get(golaravelCSRFContextKey); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	if v, ok := c.Get(p.csrfCookieName()); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	token := cookieValue(c, p.csrfCookieName())
	if token == "" {
		token = randomToken()
		p.setPanelCookie(c, p.csrfCookieName(), token, 0)
	}
	c.Set(p.csrfCookieName(), token)
	return token
}

// VerifyCSRF checks the "_token" form field (or X-CSRF-TOKEN header) of a
// state-changing request against the panel's CSRF cookie (double-submit).
func (p *Panel) VerifyCSRF(c *gcontext.Context) bool {
	if c.Request == nil {
		return false
	}
	sent := c.Header("X-CSRF-TOKEN")
	if sent == "" {
		sent = c.Request.FormValue(CSRFFormField)
	}
	if sent == "" {
		return false
	}
	expected := ""
	if v, ok := c.Get(golaravelCSRFContextKey); ok {
		expected, _ = v.(string)
	}
	if expected == "" {
		expected = cookieValue(c, p.csrfCookieName())
	}
	return expected != "" && subtle.ConstantTimeCompare([]byte(sent), []byte(expected)) == 1
}

func (p *Panel) csrfHiddenInput(c *gcontext.Context, formID string) string {
	attr := ""
	if formID != "" {
		attr = ` form="` + formID + `"`
	}
	return `<input type="hidden" name="` + CSRFFormField + `" value="` + p.CSRFToken(c) + `"` + attr + ` />`
}

// userFromSessionCookie resolves the web-engine login cookie.
func (p *Panel) userFromSessionCookie(ctx *Context) auth.Authenticatable {
	val := cookieValue(ctx.Context, p.SessionCookieName())
	if val == "" {
		return nil
	}
	if p.sessionStore != nil {
		if u, ok := p.storedSessionUser(ctx, val); ok {
			return u
		}
	} else if u, ok := p.webSessions.get(val); ok {
		return u
	}
	if p.tokenValidator != nil {
		if u, err := p.tokenValidator(ctx, val); err == nil && u != nil {
			return u
		}
	}
	return nil
}

// startWebSession signs the user in on this browser.
func (p *Panel) startWebSession(c *gcontext.Context, user auth.Authenticatable, token string, d time.Duration) error {
	value := token
	if p.tokenValidator == nil || token == "" {
		value = randomToken()
		if p.sessionStore != nil {
			if err := p.sessionStore.PutSession(c.Request.Context(), hashSessionID(value), user.GetAuthIdentifier(), time.Now().Add(d)); err != nil {
				return err
			}
		} else {
			warnMemorySessions.Do(func() {
				slog.Warn("filament: web sessions are kept in memory (lost on restart, not shared between instances). Use Panel.WebSessionsUsing with a persistent store such as pkg/sqlstore, or a TokenValidator.")
			})
			p.webSessions.put(value, webSession{user: user, expires: time.Now().Add(d)})
		}
	}
	p.setPanelCookie(c, p.SessionCookieName(), value, d)
	return nil
}

// endWebSession signs the browser out.
func (p *Panel) endWebSession(c *gcontext.Context) {
	if val := cookieValue(c, p.SessionCookieName()); val != "" {
		if p.sessionStore != nil {
			_ = p.sessionStore.DeleteSession(c.Request.Context(), hashSessionID(val))
		}
		p.webSessions.delete(val)
	}
	p.setPanelCookie(c, p.SessionCookieName(), "", -1)
}

// handleWebLogin processes POST <panel>/login of the web engine.
func (p *Panel) handleWebLogin(c *gcontext.Context) error {
	ctx := NewContext(c, p)
	if !p.checkCrossOrigin(c) || !p.VerifyCSRF(c) {
		return p.renderLoginScreen(ctx, http.StatusForbidden, "Your session expired. Please try again.", "")
	}
	identifier := strings.TrimSpace(c.FormValue("email"))
	for _, k := range []string{"username", "login"} {
		if identifier == "" {
			identifier = strings.TrimSpace(c.FormValue(k))
		}
	}
	password := c.FormValue("password")
	if identifier == "" || password == "" {
		return p.renderLoginScreen(ctx, http.StatusUnprocessableEntity, "Email and password are required.", identifier)
	}
	if p.authValidator == nil {
		return p.renderLoginScreen(ctx, http.StatusNotImplemented, "Sign-in is not configured for this panel.", identifier)
	}
	if blocked, wait := p.loginBlocked(c, identifier); blocked {
		setRetryAfter(c, wait)
		return p.renderLoginScreen(ctx, statusTooManyRequests, throttleMessage(wait), identifier)
	}
	remember := c.FormValue("remember") != ""
	d := p.ResolveTokenDuration(remember)
	ctx.Set("remember", remember)
	ctx.Set("token_duration", d)

	user, token, err := p.authValidator(ctx, identifier, password)
	if err != nil || user == nil {
		p.loginFailed(c, identifier)
		return p.renderLoginScreen(ctx, http.StatusUnauthorized, "These credentials do not match our records.", identifier)
	}
	p.loginSucceeded(c, identifier)
	if err := p.startWebSession(c, user, token, d); err != nil {
		return p.renderLoginScreen(ctx, http.StatusServiceUnavailable, "Sign-in is unavailable right now. Please try again.", identifier)
	}
	return c.Redirect(http.StatusSeeOther, p.path)
}

// crossOriginProtection returns the panel's Fetch-Metadata / Origin based
// cross-site request guard (net/http.CrossOriginProtection).
func (p *Panel) crossOriginProtection() *http.CrossOriginProtection {
	p.crossOriginOnce.Do(func() {
		if p.crossOrigin == nil {
			p.crossOrigin = http.NewCrossOriginProtection()
		}
	})
	return p.crossOrigin
}

// TrustedOrigins lets the listed origins (for example "https://admin.example.com"
// when it differs from the panel host) submit the panel's web forms. All
// other cross-origin browser POST / PUT / PATCH / DELETE requests to web
// (non-API) panel routes are refused with 403. It panics on a malformed origin.
func (p *Panel) TrustedOrigins(origins ...string) *Panel {
	cop := p.crossOriginProtection()
	for _, o := range origins {
		if err := cop.AddTrustedOrigin(o); err != nil {
			panic(fmt.Sprintf("filament: TrustedOrigins(%q): %v", o, err))
		}
	}
	return p
}

// checkCrossOrigin refuses cross-site state-changing browser requests.
// Requests without Sec-Fetch-Site and Origin headers (non-browser clients)
// pass, as do safe methods.
func (p *Panel) checkCrossOrigin(c *gcontext.Context) bool {
	return c.Request == nil || p.crossOriginProtection().Check(c.Request) == nil
}

// isSameOriginNavigation reports whether the browser says the request was
// started from a page of this origin (Sec-Fetch-Site: same-origin).
func isSameOriginNavigation(c *gcontext.Context) bool {
	return c.Header("Sec-Fetch-Site") == "same-origin"
}

// guardUnsafeGET wraps a handler that changes state but is also reachable by
// GET (delete / restore links). A GET started from the panel itself runs it;
// any other GET (a link on another site, a typed or bookmarked URL, an old
// browser) gets a confirmation page that POSTs to the same URL instead.
func (p *Panel) guardUnsafeGET(title, message string, h gcontext.HandlerFunc) gcontext.HandlerFunc {
	return func(c *gcontext.Context) error {
		if c.Request.Method != http.MethodGet || isSameOriginNavigation(c) {
			return h(c)
		}
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return p.renderConfirmPage(ctx, title, message, c.Request.URL.RequestURI())
	}
}

// renderConfirmPage renders a minimal page with a POST button to actionURL.
func (p *Panel) renderConfirmPage(ctx *Context, title, message, actionURL string) error {
	back := p.safeBackURL(ctx.Context, p.path)
	page := fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s" dir="%s">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s - %s</title>
  <link rel="stylesheet" href="%s/assets/filament.css">
</head>
<body style="min-height: 100vh; display: flex; align-items: center; justify-content: center; background-color: var(--gray-50);">
  <div class="filament-card" style="width: 100%%; max-width: 26rem; padding: 2rem;">
    <h2 style="font-size: 1.25rem; font-weight: 700; margin-bottom: 0.5rem;">%s</h2>
    <p style="font-size: 0.875rem; color: var(--gray-500); margin-bottom: 1.5rem;">%s</p>
    <form method="POST" action="%s" style="display: flex; justify-content: flex-end; gap: 0.75rem;">
      %s
      <a href="%s" class="btn btn-secondary">Cancel</a>
      <button type="submit" class="btn btn-danger">Confirm</button>
    </form>
  </div>
</body>
</html>`, ctx.Locale(), ctx.Direction(), html.EscapeString(title), html.EscapeString(p.brandName), p.path,
		html.EscapeString(title), html.EscapeString(message), html.EscapeString(actionURL),
		p.csrfHiddenInput(ctx.Context, ""), html.EscapeString(back))
	return ctx.HTML(http.StatusOK, page)
}

// safeBackURL returns the Referer as a local path when it points at this
// host, else fallback. It never returns an absolute or protocol-relative URL.
func (p *Panel) safeBackURL(c *gcontext.Context, fallback string) string {
	ref := c.Header("Referer")
	if ref == "" || c.Request == nil {
		return fallback
	}
	u, err := url.Parse(ref)
	if err != nil || (u.Host != "" && !strings.EqualFold(u.Host, c.Request.Host)) {
		return fallback
	}
	local := u.RequestURI()
	if !strings.HasPrefix(local, "/") || strings.HasPrefix(local, "//") || strings.HasPrefix(local, "/\\") {
		return fallback
	}
	return local
}
