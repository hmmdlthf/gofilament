package filament

import (
	"crypto/sha256"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/cache"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/hash"
	"github.com/golaravel/golaravel/pkg/router"
	"github.com/golaravel/golaravel/pkg/session"
)

// Accounts connects the panel's account screens to the application's user
// storage. Each group of callbacks turns on one feature; leave a group nil
// to keep that feature off.
//
//   - Profile page (Panel.Profile): Profile, and optionally UpdateProfile.
//   - Change password (on the profile page): PasswordHash + UpdatePassword.
//   - Forgot / reset password (Panel.PasswordReset): FindByEmail +
//     SendResetLink + UpdatePassword.
//   - Two-factor sign-in with an authenticator app: TwoFactorSecret +
//     SaveTwoFactorSecret. Users turn it on from the profile page; the
//     sign-in then asks for a code. Requires Panel.Sessions.
//
// Users are identified by auth.Authenticatable.GetAuthIdentifier, and the
// panel's session user loader (Panel.Sessions) loads them back.
type Accounts struct {
	// Profile returns the name and email shown on the profile page.
	Profile func(ctx *Context, user auth.Authenticatable) (name, email string, err error)
	// UpdateProfile saves an edited name and email. Validate the email
	// (uniqueness, verification) here; a returned error is shown to the user.
	UpdateProfile func(ctx *Context, user auth.Authenticatable, name, email string) error

	// PasswordHash returns the user's stored password hash (bcrypt or
	// argon2id, as golaravel's hash package makes them).
	PasswordHash func(ctx *Context, user auth.Authenticatable) (string, error)
	// UpdatePassword stores a new hash made with golaravel's hash.Make.
	UpdatePassword func(ctx *Context, user auth.Authenticatable, newHash string) error

	// FindByEmail finds the user a reset link is for. Return an error or
	// nil for unknown addresses; the page says the same thing either way.
	FindByEmail func(ctx *Context, email string) (auth.Authenticatable, error)
	// SendResetLink mails link (absolute, valid for ResetTTL) to email.
	SendResetLink func(ctx *Context, user auth.Authenticatable, email, link string) error
	// ResetTokens stores reset tokens; default in memory. Use
	// auth.NewSQLResetTokenStore or auth.CacheResetTokenStore when the
	// panel runs on several instances or must survive restarts.
	ResetTokens auth.ResetTokenStore
	// ResetTTL is how long a reset link works (default 60 minutes).
	ResetTTL time.Duration

	// TwoFactorSecret returns the user's TOTP secret, or "" when two-factor
	// sign-in is off for them.
	TwoFactorSecret func(ctx *Context, user auth.Authenticatable) (string, error)
	// SaveTwoFactorSecret stores a confirmed secret, or "" to turn it off.
	// Store it encrypted if you can.
	SaveTwoFactorSecret func(ctx *Context, user auth.Authenticatable, secret string) error
}

// Accounts configures the profile, password and two-factor screens.
func (p *Panel) Accounts(a Accounts) *Panel {
	p.accounts = &a
	if a.ResetTokens == nil {
		a.ResetTokens = auth.NewMemoryResetTokenStore()
	}
	ttl := a.ResetTTL
	if ttl <= 0 {
		ttl = 60 * time.Minute
	}
	p.accounts.ResetTokens = a.ResetTokens
	p.resetBroker = auth.NewPasswordResetBrokerWithStore(a.ResetTokens, ttl)
	p.totp = auth.NewTOTPManager(auth.TOTPOptions{Issuer: p.brandName})
	if a.Profile != nil {
		p.hasProfile = true
	}
	if a.FindByEmail != nil && a.SendResetLink != nil && a.UpdatePassword != nil {
		p.hasPasswordReset = true
	}
	return p
}

func (p *Panel) canChangePassword() bool {
	return p.accounts != nil && p.accounts.PasswordHash != nil && p.accounts.UpdatePassword != nil
}

func (p *Panel) canResetPassword() bool {
	return p.hasPasswordReset && p.accounts != nil && p.accounts.FindByEmail != nil &&
		p.accounts.SendResetLink != nil && p.accounts.UpdatePassword != nil
}

func (p *Panel) twoFactorAvailable() bool {
	return p.accounts != nil && p.accounts.TwoFactorSecret != nil && p.accounts.SaveTwoFactorSecret != nil
}

func (p *Panel) profileAvailable() bool {
	return p.hasProfile && p.accounts != nil && (p.accounts.Profile != nil || p.canChangePassword() || p.twoFactorAvailable())
}

// session keys of the account screens
const (
	sessTwoFactorPending  = "_filament_2fa_user"
	sessTwoFactorSince    = "_filament_2fa_since"
	sessTwoFactorRemember = "_filament_2fa_remember"
	sessTwoFactorSetup    = "_filament_2fa_setup"
)

// password rules shared by change and reset
func checkNewPassword(pw, confirm string) string {
	switch {
	case len(pw) < 8:
		return "The new password must be at least 8 characters."
	case len(pw) > 72:
		return "The new password may not be longer than 72 bytes."
	case pw != confirm:
		return "The password confirmation does not match."
	}
	return ""
}

// ---------------------------------------------------------------- routes

func (p *Panel) mountAccountRoutes(r *router.Router) {
	if p.accounts == nil {
		return
	}
	if p.profileAvailable() {
		r.GET(p.path+"/profile", p.handleProfile)
		r.POST(p.path+"/profile", p.handleProfileDetails)
		r.POST(p.path+"/profile/password", p.handleProfilePassword)
		r.POST(p.path+"/profile/two-factor/enable", p.handleTwoFactorEnable)
		r.POST(p.path+"/profile/two-factor/confirm", p.handleTwoFactorConfirm)
		r.POST(p.path+"/profile/two-factor/disable", p.handleTwoFactorDisable)
	}
	if p.canResetPassword() {
		r.GET(p.path+"/forgot-password", func(c *gcontext.Context) error {
			return p.renderForgotPassword(NewContext(c, p), http.StatusOK, "", "")
		})
		r.POST(p.path+"/forgot-password", p.handleForgotPassword)
		r.GET(p.path+"/reset-password", func(c *gcontext.Context) error {
			return p.renderResetPassword(NewContext(c, p), http.StatusOK, "", c.Query("email"), c.Query("token"))
		})
		r.POST(p.path+"/reset-password", p.handleResetPassword)
	}
	if p.twoFactorAvailable() {
		r.GET(p.path+"/two-factor-challenge", func(c *gcontext.Context) error {
			if pendingTwoFactorUser(c) == "" {
				return c.Redirect(http.StatusSeeOther, p.path+"/login")
			}
			return p.renderTwoFactorChallenge(NewContext(c, p), http.StatusOK, "")
		})
		r.POST(p.path+"/two-factor-challenge", p.handleTwoFactorChallenge)
	}
}

// formGuard checks the origin and CSRF token of a POST.
func (p *Panel) formGuard(c *gcontext.Context) bool {
	return p.checkCrossOrigin(c) && p.VerifyCSRF(c)
}

// --------------------------------------------------------------- profile

var profileNotices = map[string]string{
	"saved":        "Your profile was saved.",
	"password":     "Your password was changed. Other devices were signed out.",
	"password-one": "Your password was changed.",
	"2fa-on":       "Two-factor sign-in is on.",
	"2fa-off":      "Two-factor sign-in is off.",
}

func (p *Panel) profileUser(c *gcontext.Context) (*Context, auth.Authenticatable, error) {
	ctx, err := p.AuthorizeAccess(c)
	if err != nil {
		return nil, nil, err
	}
	u := ctx.User()
	if u == nil {
		return nil, nil, ErrAccessDenied
	}
	return ctx, u, nil
}

func (p *Panel) handleProfile(c *gcontext.Context) error {
	ctx, user, err := p.profileUser(c)
	if err != nil {
		return err
	}
	return p.renderProfile(ctx, user, http.StatusOK, profileNotices[c.Query("status")], "", nil)
}

func (p *Panel) handleProfileDetails(c *gcontext.Context) error {
	ctx, user, err := p.profileUser(c)
	if err != nil {
		return err
	}
	if !p.formGuard(c) {
		return p.renderProfile(ctx, user, http.StatusForbidden, "", "Your session expired. Please try again.", nil)
	}
	if p.accounts.UpdateProfile == nil {
		return c.String(http.StatusNotFound, "404 Not Found")
	}
	name := strings.TrimSpace(c.FormValue("name"))
	email := strings.TrimSpace(c.FormValue("email"))
	if name == "" || email == "" || !strings.Contains(email, "@") {
		return p.renderProfile(ctx, user, http.StatusUnprocessableEntity, "", "Enter a name and a valid email address.",
			map[string]string{"name": name, "email": email})
	}
	if err := p.accounts.UpdateProfile(ctx, user, name, email); err != nil {
		return p.renderProfile(ctx, user, http.StatusUnprocessableEntity, "", err.Error(), map[string]string{"name": name, "email": email})
	}
	return c.Redirect(http.StatusSeeOther, p.path+"/profile?status=saved")
}

// verifyPassword checks the user's current password.
func (p *Panel) verifyPassword(ctx *Context, user auth.Authenticatable, pw string) bool {
	if p.accounts.PasswordHash == nil || pw == "" {
		return false
	}
	h, err := p.accounts.PasswordHash(ctx, user)
	return err == nil && h != "" && hash.Check(pw, h)
}

func (p *Panel) handleProfilePassword(c *gcontext.Context) error {
	ctx, user, err := p.profileUser(c)
	if err != nil {
		return err
	}
	if !p.canChangePassword() {
		return c.String(http.StatusNotFound, "404 Not Found")
	}
	if !p.formGuard(c) {
		return p.renderProfile(ctx, user, http.StatusForbidden, "", "Your session expired. Please try again.", nil)
	}
	key := "password:" + user.GetAuthIdentifier()
	if blocked, wait := p.loginBlocked(c, key); blocked {
		setRetryAfter(c, wait)
		return p.renderProfile(ctx, user, statusTooManyRequests, "", throttleMessage(wait), nil)
	}
	if !p.verifyPassword(ctx, user, c.FormValue("current_password")) {
		p.loginFailed(c, key)
		return p.renderProfile(ctx, user, http.StatusUnprocessableEntity, "", "The current password is not correct.", nil)
	}
	p.loginSucceeded(c, key)
	pw := c.FormValue("password")
	if msg := checkNewPassword(pw, c.FormValue("password_confirmation")); msg != "" {
		return p.renderProfile(ctx, user, http.StatusUnprocessableEntity, "", msg, nil)
	}
	h, err := hash.Make(pw)
	if err != nil {
		return p.renderProfile(ctx, user, http.StatusInternalServerError, "", "The password could not be changed.", nil)
	}
	if err := p.accounts.UpdatePassword(ctx, user, h); err != nil {
		return p.renderProfile(ctx, user, http.StatusInternalServerError, "", "The password could not be changed.", nil)
	}
	status := "password-one"
	if p.sessionManager != nil {
		if err := auth.LogoutOtherDevices(c); err == nil {
			status = "password"
		}
	}
	return c.Redirect(http.StatusSeeOther, p.path+"/profile?status="+status)
}

// ------------------------------------------------------------ two-factor

func (p *Panel) userTwoFactorSecret(ctx *Context, user auth.Authenticatable) string {
	if !p.twoFactorAvailable() {
		return ""
	}
	s, err := p.accounts.TwoFactorSecret(ctx, user)
	if err != nil {
		return ""
	}
	return s
}

func (p *Panel) handleTwoFactorEnable(c *gcontext.Context) error {
	ctx, user, err := p.profileUser(c)
	if err != nil {
		return err
	}
	if !p.twoFactorAvailable() {
		return c.String(http.StatusNotFound, "404 Not Found")
	}
	sess := session.FromContext(c)
	if !p.formGuard(c) || sess == nil {
		return p.renderProfile(ctx, user, http.StatusForbidden, "", "Your session expired. Please try again.", nil)
	}
	if !p.verifyPassword(ctx, user, c.FormValue("current_password")) && p.accounts.PasswordHash != nil {
		return p.renderProfile(ctx, user, http.StatusUnprocessableEntity, "", "The current password is not correct.", nil)
	}
	secret, err := p.totp.GenerateSecret()
	if err != nil {
		return p.renderProfile(ctx, user, http.StatusInternalServerError, "", "Two-factor sign-in could not be set up.", nil)
	}
	sess.Put(sessTwoFactorSetup, secret)
	return p.renderProfile(ctx, user, http.StatusOK, "", "", nil)
}

func (p *Panel) handleTwoFactorConfirm(c *gcontext.Context) error {
	ctx, user, err := p.profileUser(c)
	if err != nil {
		return err
	}
	sess := session.FromContext(c)
	if !p.twoFactorAvailable() || sess == nil {
		return c.String(http.StatusNotFound, "404 Not Found")
	}
	if !p.formGuard(c) {
		return p.renderProfile(ctx, user, http.StatusForbidden, "", "Your session expired. Please try again.", nil)
	}
	secret := sess.GetString(sessTwoFactorSetup)
	if secret == "" {
		return c.Redirect(http.StatusSeeOther, p.path+"/profile")
	}
	if ok, step := p.totp.VerifyCodeStep(secret, strings.TrimSpace(c.FormValue("code"))); !ok || !p.acceptTOTPStep(user.GetAuthIdentifier(), secret, step) {
		return p.renderProfile(ctx, user, http.StatusUnprocessableEntity, "", "The code is not valid. Check the time on your phone and try again.", nil)
	}
	if err := p.accounts.SaveTwoFactorSecret(ctx, user, secret); err != nil {
		return p.renderProfile(ctx, user, http.StatusInternalServerError, "", "Two-factor sign-in could not be turned on.", nil)
	}
	sess.Forget(sessTwoFactorSetup)
	return c.Redirect(http.StatusSeeOther, p.path+"/profile?status=2fa-on")
}

func (p *Panel) handleTwoFactorDisable(c *gcontext.Context) error {
	ctx, user, err := p.profileUser(c)
	if err != nil {
		return err
	}
	if !p.twoFactorAvailable() {
		return c.String(http.StatusNotFound, "404 Not Found")
	}
	if !p.formGuard(c) {
		return p.renderProfile(ctx, user, http.StatusForbidden, "", "Your session expired. Please try again.", nil)
	}
	// Turning it off needs the password, or a current code when the app
	// keeps no passwords.
	okPw := p.verifyPassword(ctx, user, c.FormValue("current_password"))
	if !okPw {
		secret := p.userTwoFactorSecret(ctx, user)
		ok, step := p.totp.VerifyCodeStep(secret, strings.TrimSpace(c.FormValue("code")))
		if secret == "" || !ok || !p.acceptTOTPStep(user.GetAuthIdentifier(), secret, step) {
			return p.renderProfile(ctx, user, http.StatusUnprocessableEntity, "", "Enter your current password (or a code) to turn two-factor sign-in off.", nil)
		}
	}
	if err := p.accounts.SaveTwoFactorSecret(ctx, user, ""); err != nil {
		return p.renderProfile(ctx, user, http.StatusInternalServerError, "", "Two-factor sign-in could not be turned off.", nil)
	}
	return c.Redirect(http.StatusSeeOther, p.path+"/profile?status=2fa-off")
}

// acceptTOTPStep makes each code single-use: the step is remembered in
// golaravel's cache (shared between instances when the cache is).
// A new secret starts a new series of codes, so the key includes it (hashed).
func (p *Panel) acceptTOTPStep(userID, secret string, step int64) bool {
	sum := sha256.Sum256([]byte(secret))
	ok, err := cache.Add(fmt.Sprintf("gofilament:%s:totp:%s:%x:%d", p.id, userID, sum[:8], step), 1, 3*time.Minute)
	return err == nil && ok
}

// beginTwoFactor parks a password-checked login until the code is given.
// It reports whether a challenge is needed.
func (p *Panel) beginTwoFactor(c *gcontext.Context, ctx *Context, user auth.Authenticatable, remember bool) (bool, error) {
	if p.userTwoFactorSecret(ctx, user) == "" {
		return false, nil
	}
	sess := session.FromContext(c)
	if sess == nil {
		return true, errTwoFactorNeedsSessions
	}
	if err := session.ManagerFrom(c).Regenerate(sess); err != nil {
		return true, err
	}
	sess.Put(sessTwoFactorPending, user.GetAuthIdentifier())
	sess.Put(sessTwoFactorSince, time.Now().Unix())
	sess.Put(sessTwoFactorRemember, remember)
	return true, nil
}

var errTwoFactorNeedsSessions = fmt.Errorf("filament: two-factor sign-in needs Panel.Sessions")

func pendingTwoFactorUser(c *gcontext.Context) string {
	sess := session.FromContext(c)
	if sess == nil {
		return ""
	}
	uid := sess.GetString(sessTwoFactorPending)
	since := int64(sess.GetInt(sessTwoFactorSince))
	if uid == "" || time.Since(time.Unix(since, 0)) > 10*time.Minute {
		return ""
	}
	return uid
}

func (p *Panel) handleTwoFactorChallenge(c *gcontext.Context) error {
	ctx := NewContext(c, p)
	uid := pendingTwoFactorUser(c)
	if uid == "" {
		return c.Redirect(http.StatusSeeOther, p.path+"/login")
	}
	if !p.formGuard(c) {
		return p.renderTwoFactorChallenge(ctx, http.StatusForbidden, "Your session expired. Please try again.")
	}
	key := "2fa:" + uid
	if blocked, wait := p.loginBlocked(c, key); blocked {
		setRetryAfter(c, wait)
		return p.renderTwoFactorChallenge(ctx, statusTooManyRequests, throttleMessage(wait))
	}
	if p.sessionUsers == nil {
		return p.renderTwoFactorChallenge(ctx, http.StatusServiceUnavailable, "Sign-in is unavailable right now.")
	}
	user, err := p.sessionUsers(ctx, uid)
	if err != nil || user == nil {
		return c.Redirect(http.StatusSeeOther, p.path+"/login")
	}
	secret := p.userTwoFactorSecret(ctx, user)
	ok, step := p.totp.VerifyCodeStep(secret, strings.TrimSpace(c.FormValue("code")))
	if secret == "" || !ok || !p.acceptTOTPStep(uid, secret, step) {
		p.loginFailed(c, key)
		return p.renderTwoFactorChallenge(ctx, http.StatusUnauthorized, "The code is not valid.")
	}
	p.loginSucceeded(c, key)
	sess := session.FromContext(c)
	sess.Forget(sessTwoFactorPending, sessTwoFactorSince, sessTwoFactorRemember)
	if err := p.sessionLogin(c, user); err != nil {
		return p.renderTwoFactorChallenge(ctx, http.StatusServiceUnavailable, "Sign-in is unavailable right now.")
	}
	return c.Redirect(http.StatusSeeOther, p.path)
}

// ------------------------------------------------------- password reset

func (p *Panel) handleForgotPassword(c *gcontext.Context) error {
	ctx := NewContext(c, p)
	if !p.formGuard(c) {
		return p.renderForgotPassword(ctx, http.StatusForbidden, "Your session expired. Please try again.", "")
	}
	email := strings.ToLower(strings.TrimSpace(c.FormValue("email")))
	if email == "" || !strings.Contains(email, "@") {
		return p.renderForgotPassword(ctx, http.StatusUnprocessableEntity, "Enter your email address.", "")
	}
	key := "reset:" + email
	if blocked, wait := p.loginBlocked(c, key); blocked {
		setRetryAfter(c, wait)
		return p.renderForgotPassword(ctx, statusTooManyRequests, throttleMessage(wait), "")
	}
	p.loginFailed(c, key) // every request counts against the limit
	if user, err := p.accounts.FindByEmail(ctx, email); err == nil && user != nil {
		if token, err := p.resetBroker.CreateToken(email); err == nil {
			q := url.Values{"token": {token}, "email": {email}}
			link := p.absoluteURL(c, p.path+"/reset-password?"+q.Encode())
			_ = p.accounts.SendResetLink(ctx, user, email, link)
		}
	}
	// The same answer whether or not the address has an account.
	return p.renderForgotPassword(ctx, http.StatusOK, "", "If an account uses that address, we have emailed a link to reset the password.")
}

func (p *Panel) absoluteURL(c *gcontext.Context, path string) string {
	scheme := "http"
	if isSecureRequest(c) {
		scheme = "https"
	}
	host := ""
	if c.Request != nil {
		host = c.Request.Host
	}
	return scheme + "://" + host + path
}

func (p *Panel) handleResetPassword(c *gcontext.Context) error {
	ctx := NewContext(c, p)
	email := strings.ToLower(strings.TrimSpace(c.FormValue("email")))
	token := c.FormValue("token")
	if !p.formGuard(c) {
		return p.renderResetPassword(ctx, http.StatusForbidden, "Your session expired. Please try again.", email, token)
	}
	if !p.resetBroker.ValidateToken(email, token) {
		return p.renderResetPassword(ctx, http.StatusUnprocessableEntity, "This reset link is invalid or has expired. Ask for a new one.", email, "")
	}
	pw := c.FormValue("password")
	if msg := checkNewPassword(pw, c.FormValue("password_confirmation")); msg != "" {
		return p.renderResetPassword(ctx, http.StatusUnprocessableEntity, msg, email, token)
	}
	user, err := p.accounts.FindByEmail(ctx, email)
	if err != nil || user == nil {
		return p.renderResetPassword(ctx, http.StatusUnprocessableEntity, "This reset link is invalid or has expired. Ask for a new one.", email, "")
	}
	h, err := hash.Make(pw)
	if err == nil {
		err = p.accounts.UpdatePassword(ctx, user, h)
	}
	if err != nil {
		return p.renderResetPassword(ctx, http.StatusInternalServerError, "The password could not be reset.", email, token)
	}
	p.resetBroker.DeleteToken(email)
	if p.sessionManager != nil {
		_ = p.sessionManager.DestroyUserSessions(user.GetAuthIdentifier())
	}
	return c.Redirect(http.StatusSeeOther, p.path+"/login?status=reset")
}

// ---------------------------------------------------------------- pages

func alertHTML(kind, msg string) string {
	if msg == "" {
		return ""
	}
	bg, fg := "#fef2f2", "#b91c1c"
	if kind == "ok" {
		bg, fg = "#f0fdf4", "#15803d"
	}
	return fmt.Sprintf(`<div class="filament-alert" role="%s" style="margin-bottom: 1rem; padding: 0.75rem; border-radius: 0.5rem; background: %s; color: %s; font-size: 0.875rem;">%s</div>`,
		map[bool]string{true: "status", false: "alert"}[kind == "ok"], bg, fg, html.EscapeString(msg))
}

// renderCardPage renders a centred card page in the panel's style.
func (p *Panel) renderCardPage(ctx *Context, status int, title, subtitle, body string, wide bool) error {
	width := "26rem"
	if wide {
		width = "40rem"
	}
	page := fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s" dir="%s">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>%s - %s</title>
  <link rel="stylesheet" href="%s/assets/filament.css">
  <script src="%s/assets/filament.js"></script>
%s
</head>
<body style="min-height: 100vh; display: flex; align-items: center; justify-content: center; background-color: var(--gray-50); padding: 1rem;">
  <div class="filament-card" style="width: 100%%; max-width: %s; padding: 2rem;">
    <div style="text-align: center; margin-bottom: 1.5rem;">
      <h2 style="font-size: 1.5rem; font-weight: 700; color: var(--primary-600);">%s</h2>
      <p style="font-size: 0.875rem; color: var(--gray-500); margin-top: 0.25rem;">%s</p>
    </div>
    %s
  </div>
  %s
</body>
</html>`, ctx.Locale(), ctx.Direction(), html.EscapeString(title), html.EscapeString(p.brandName), p.path, p.path,
		p.RenderHeadHTML(ctx), width, html.EscapeString(title), html.EscapeString(subtitle), body, p.RenderBodyEndHTML(ctx))
	return ctx.HTML(status, page)
}

func (p *Panel) field(name, label, typ, value, extra string) string {
	return fmt.Sprintf(`<div class="form-group"><label class="form-label" for="%s">%s</label><input type="%s" id="%s" name="%s" value="%s" class="form-control" %s /></div>`,
		name, html.EscapeString(label), typ, name, name, html.EscapeString(value), extra)
}

func (p *Panel) renderForgotPassword(ctx *Context, status int, errMsg, okMsg string) error {
	body := alertHTML("err", errMsg) + alertHTML("ok", okMsg) + fmt.Sprintf(`<form method="POST" action="%s/forgot-password">%s%s
<button type="submit" class="btn btn-primary" style="width: 100%%;">Email me a reset link</button></form>
<p style="text-align: center; margin-top: 1rem; font-size: 0.8125rem;"><a href="%s/login">Back to sign in</a></p>`,
		p.path, p.csrfHiddenInput(ctx.Context, ""), p.field("email", "Email address", "email", "", `required autocomplete="email"`), p.path)
	return p.renderCardPage(ctx, status, "Forgot password", "We will email you a link to choose a new password.", body, false)
}

func (p *Panel) renderResetPassword(ctx *Context, status int, errMsg, email, token string) error {
	body := alertHTML("err", errMsg)
	if token == "" {
		body += fmt.Sprintf(`<p style="text-align: center; font-size: 0.875rem;"><a href="%s/forgot-password">Ask for a new reset link</a></p>`, p.path)
	} else {
		body += fmt.Sprintf(`<form method="POST" action="%s/reset-password">%s
<input type="hidden" name="token" value="%s" /><input type="hidden" name="email" value="%s" />
%s%s<button type="submit" class="btn btn-primary" style="width: 100%%;">Reset password</button></form>`,
			p.path, p.csrfHiddenInput(ctx.Context, ""), html.EscapeString(token), html.EscapeString(email),
			p.field("password", "New password", "password", "", `required minlength="8" autocomplete="new-password"`),
			p.field("password_confirmation", "Confirm new password", "password", "", `required autocomplete="new-password"`))
	}
	return p.renderCardPage(ctx, status, "Reset password", email, body, false)
}

func (p *Panel) renderTwoFactorChallenge(ctx *Context, status int, errMsg string) error {
	body := alertHTML("err", errMsg) + fmt.Sprintf(`<form method="POST" action="%s/two-factor-challenge">%s%s
<button type="submit" class="btn btn-primary" style="width: 100%%;">Verify</button></form>
<p style="text-align: center; margin-top: 1rem; font-size: 0.8125rem;"><a href="%s/login">Back to sign in</a></p>`,
		p.path, p.csrfHiddenInput(ctx.Context, ""),
		p.field("code", "Authentication code", "text", "", `required inputmode="numeric" pattern="[0-9]{6}" maxlength="6" autocomplete="one-time-code" autofocus`), p.path)
	return p.renderCardPage(ctx, status, "Two-factor sign-in", "Enter the 6-digit code from your authenticator app.", body, false)
}

func (p *Panel) renderProfile(ctx *Context, user auth.Authenticatable, status int, okMsg, errMsg string, old map[string]string) error {
	var b strings.Builder
	b.WriteString(alertHTML("ok", okMsg))
	b.WriteString(alertHTML("err", errMsg))
	section := func(title string) {
		fmt.Fprintf(&b, `<h3 style="font-size: 1rem; font-weight: 700; margin: 1.5rem 0 0.75rem;">%s</h3>`, html.EscapeString(title))
	}
	csrf := p.csrfHiddenInput(ctx.Context, "")

	if p.accounts.Profile != nil {
		name, email, _ := p.accounts.Profile(ctx, user)
		if old != nil {
			name, email = old["name"], old["email"]
		}
		section("Profile")
		if p.accounts.UpdateProfile != nil {
			fmt.Fprintf(&b, `<form method="POST" action="%s/profile">%s%s%s<button type="submit" class="btn btn-primary">Save profile</button></form>`,
				p.path, csrf, p.field("name", "Name", "text", name, `required autocomplete="name"`), p.field("email", "Email address", "email", email, `required autocomplete="email"`))
		} else {
			fmt.Fprintf(&b, `<p style="font-size: 0.875rem;">%s<br><span style="color: var(--gray-500);">%s</span></p>`, html.EscapeString(name), html.EscapeString(email))
		}
	}
	if p.canChangePassword() {
		section("Change password")
		fmt.Fprintf(&b, `<form method="POST" action="%s/profile/password">%s%s%s%s<button type="submit" class="btn btn-primary">Change password</button></form>`,
			p.path, csrf,
			p.field("current_password", "Current password", "password", "", `required autocomplete="current-password"`),
			p.field("password", "New password", "password", "", `required minlength="8" autocomplete="new-password"`),
			p.field("password_confirmation", "Confirm new password", "password", "", `required autocomplete="new-password"`))
	}
	if p.twoFactorAvailable() {
		section("Two-factor sign-in")
		pending := ""
		if sess := session.FromContext(ctx.Context); sess != nil {
			pending = sess.GetString(sessTwoFactorSetup)
		}
		pwField := ""
		if p.accounts.PasswordHash != nil {
			pwField = p.field("current_password", "Current password", "password", "", `required autocomplete="current-password"`)
		}
		switch {
		case p.userTwoFactorSecret(ctx, user) != "":
			fmt.Fprintf(&b, `<p style="font-size: 0.875rem; margin-bottom: 0.75rem;">Two-factor sign-in is <strong>on</strong>.</p>
<form method="POST" action="%s/profile/two-factor/disable">%s%s<button type="submit" class="btn btn-danger">Turn off</button></form>`, p.path, csrf, pwField)
		case pending != "" && session.FromContext(ctx.Context) != nil:
			uri := p.totp.ProvisioningURI(user.GetAuthIdentifier(), pending)
			fmt.Fprintf(&b, `<p style="font-size: 0.875rem; margin-bottom: 0.5rem;">Add this key to your authenticator app, then enter the code it shows.</p>
<p style="font-family: monospace; font-size: 1rem; letter-spacing: 0.1em; word-break: break-all; padding: 0.5rem; background: var(--gray-100); border-radius: 0.375rem;" data-otpauth="%s">%s</p>
<p style="font-size: 0.75rem; margin: 0.5rem 0 1rem;"><a href="%s">Open in an authenticator app</a></p>
<form method="POST" action="%s/profile/two-factor/confirm">%s%s<button type="submit" class="btn btn-primary">Turn on</button></form>`,
				html.EscapeString(uri), html.EscapeString(pending), html.EscapeString(uri), p.path, csrf,
				p.field("code", "Code", "text", "", `required inputmode="numeric" maxlength="6" autocomplete="one-time-code"`))
		case p.sessionManager == nil:
			b.WriteString(`<p style="font-size: 0.875rem; color: var(--gray-500);">Two-factor sign-in needs the panel to run on golaravel sessions (Panel.Sessions).</p>`)
		default:
			fmt.Fprintf(&b, `<p style="font-size: 0.875rem; margin-bottom: 0.75rem;">Ask for a code from an authenticator app when you sign in.</p>
<form method="POST" action="%s/profile/two-factor/enable">%s%s<button type="submit" class="btn btn-primary">Set up</button></form>`, p.path, csrf, pwField)
		}
	}
	fmt.Fprintf(&b, `<p style="text-align: center; margin-top: 1.5rem; font-size: 0.8125rem;"><a href="%s">Back to the dashboard</a></p>`, p.path)
	return p.renderCardPage(ctx, status, "Your account", user.GetAuthIdentifier(), b.String(), true)
}

// loginNotice is shown above the sign-in form after a redirect.
func loginNotice(c *gcontext.Context) string {
	switch c.Query("status") {
	case "reset":
		return "Your password was reset. Sign in with the new password."
	}
	return ""
}

// renderUserMenuHTML renders the signed-in user's name with the user menu
// (Profile, Sign Out and items added with Panel.UserMenu).
func (p *Panel) renderUserMenuHTML(ctx *Context) string {
	name := "Administrator"
	if u := ctx.User(); u != nil {
		name = u.GetAuthIdentifier()
		if p.accounts != nil && p.accounts.Profile != nil {
			if n, _, err := p.accounts.Profile(ctx, u); err == nil && n != "" {
				name = n
			}
		}
	}
	var sb strings.Builder
	sb.WriteString(`<details class="filament-user-menu" style="position: relative; font-size: 0.875rem; font-weight: 500;">`)
	fmt.Fprintf(&sb, `<summary style="cursor: pointer; list-style: none;">%s ▾</summary>`, html.EscapeString(name))
	sb.WriteString(`<div style="position: absolute; inset-inline-end: 0; top: 1.75rem; min-width: 11rem; z-index: 40; background: var(--white, #fff); border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 0.25rem; box-shadow: 0 8px 24px rgba(0,0,0,.08);">`)
	for _, it := range p.userMenu.Items() {
		href := it.URL
		if strings.HasPrefix(href, "/") && !strings.HasPrefix(href, p.path+"/") {
			href = p.path + href
		}
		switch href {
		case p.path + "/profile":
			if !p.profileAvailable() {
				continue
			}
		case p.path + "/logout":
			if !p.hasLogin {
				continue
			}
		}
		color := ""
		if it.Color == "danger" {
			color = "color: var(--danger-600, #dc2626);"
		}
		fmt.Fprintf(&sb, `<a href="%s" style="display: block; padding: 0.5rem 0.75rem; border-radius: 0.375rem; %s">%s</a>`,
			html.EscapeString(href), color, html.EscapeString(it.Label))
	}
	sb.WriteString(`</div></details>`)
	return sb.String()
}
