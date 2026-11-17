package filament

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/hash"
	"github.com/golaravel/golaravel/pkg/router"
)

// Self-registration and email verification.
//
//   - Registration (Panel.Registration): Accounts.Register creates the user
//     from a name, email and password hash. The new user is signed in.
//   - Email verification: EmailVerified + MarkEmailVerified +
//     SendVerificationLink (and Profile, for the address). New users get a
//     signed link (golaravel's EmailVerificationBroker, keyed by
//     VerificationSecret or APP_KEY); until they open it, the panel shows
//     only a "verify your email" page and the API answers 403 with
//     "email_unverified": true. EmailVerified runs on every request, so keep
//     it cheap.

// ErrEmailTaken is what Register should return for an address that already
// has an account; the form says so next to the field.
var ErrEmailTaken = errors.New("An account with this email address already exists.")

const defaultVerificationTTL = 60 * time.Minute

func (p *Panel) registrationAvailable() bool {
	return p.hasRegistration && p.accounts != nil && p.accounts.Register != nil
}

func (p *Panel) verificationAvailable() bool {
	a := p.accounts
	return a != nil && a.EmailVerified != nil && a.MarkEmailVerified != nil && a.SendVerificationLink != nil && a.Profile != nil
}

func (p *Panel) verificationBroker() *auth.EmailVerificationBroker {
	p.verifyOnce.Do(func() {
		secret := ""
		if p.accounts != nil {
			secret = p.accounts.VerificationSecret
		}
		if secret == "" {
			secret = os.Getenv("APP_KEY")
		}
		if secret == "" {
			buf := make([]byte, 32)
			_, _ = rand.Read(buf)
			secret = hex.EncodeToString(buf)
			log.Printf("filament: no VerificationSecret or APP_KEY; email verification links stop working when the process restarts")
		}
		p.verifyBroker = auth.NewEmailVerificationBroker(secret)
	})
	return p.verifyBroker
}

// emailUnverified reports whether user must still verify their address.
func (p *Panel) emailUnverified(ctx *Context, user auth.Authenticatable) bool {
	if !p.verificationAvailable() || user == nil {
		return false
	}
	ok, err := p.accounts.EmailVerified(ctx, user)
	return err == nil && !ok
}

// verificationExempt lists the routes an unverified user may still use.
func (p *Panel) verificationExempt(path string) bool {
	rest := strings.TrimPrefix(path, strings.TrimRight(p.path, "/"))
	for _, ok := range []string{"/email/", "/logout", "/api/email/", "/api/me", "/api/logout", "/api/boot", "/assets/"} {
		if strings.HasPrefix(rest, ok) || rest == strings.TrimSuffix(ok, "/") {
			return true
		}
	}
	return false
}

// sendVerification mails a fresh verification link to the user.
func (p *Panel) sendVerification(c *gcontext.Context, ctx *Context, user auth.Authenticatable) error {
	_, email, err := p.accounts.Profile(ctx, user)
	if err != nil || email == "" {
		return fmt.Errorf("filament: no email address for %s", user.GetAuthIdentifier())
	}
	ttl := p.accounts.VerificationTTL
	if ttl <= 0 {
		ttl = defaultVerificationTTL
	}
	link := p.verificationBroker().GenerateSignedURL(p.absoluteURL(c, p.path+"/email/verify/confirm"), user.GetAuthIdentifier(), strings.ToLower(email), ttl)
	return p.accounts.SendVerificationLink(ctx, user, email, link)
}

func (p *Panel) mountRegistrationRoutes(r *router.Router) {
	if p.registrationAvailable() {
		r.GET(p.path+"/register", func(c *gcontext.Context) error {
			return p.renderRegister(NewContext(c, p), http.StatusOK, "", nil)
		})
		r.POST(p.path+"/register", p.handleRegister)
	}
	if p.verificationAvailable() {
		r.GET(p.path+"/email/verify", p.handleVerifyNotice)
		r.POST(p.path+"/email/verification-notification", p.handleResendVerification)
		r.GET(p.path+"/email/verify/confirm", p.handleVerifyConfirm)
	}
}

// register validates the form and creates the user. It returns the user, or
// a message and the field it belongs to.
func (p *Panel) register(c *gcontext.Context, ctx *Context, name, email, pw, confirm string) (auth.Authenticatable, string, string, int) {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return nil, "Enter your name.", "name", http.StatusUnprocessableEntity
	case !strings.Contains(email, "@") || strings.ContainsAny(email, " \t\r\n"):
		return nil, "Enter a valid email address.", "email", http.StatusUnprocessableEntity
	}
	if msg := checkNewPassword(pw, confirm); msg != "" {
		return nil, msg, "password", http.StatusUnprocessableEntity
	}
	key := "register:" + c.ClientIP()
	if blocked, wait := p.loginBlocked(c, key); blocked {
		setRetryAfter(c, wait)
		return nil, throttleMessage(wait), "", statusTooManyRequests
	}
	p.loginFailed(c, key) // every sign-up counts against the limit
	h, err := hash.Make(pw)
	if err != nil {
		return nil, "The account could not be created.", "", http.StatusInternalServerError
	}
	user, err := p.accounts.Register(ctx, name, email, h)
	if errors.Is(err, ErrEmailTaken) {
		return nil, ErrEmailTaken.Error(), "email", http.StatusUnprocessableEntity
	}
	if err != nil || user == nil {
		var verr *forms.ValidationError
		if errors.As(err, &verr) {
			for field, m := range verr.Errors {
				return nil, m, field, http.StatusUnprocessableEntity
			}
		}
		return nil, "The account could not be created.", "", http.StatusUnprocessableEntity
	}
	if p.verificationAvailable() {
		_ = p.sendVerification(c, ctx, user)
	}
	return user, "", "", http.StatusCreated
}

func (p *Panel) handleRegister(c *gcontext.Context) error {
	ctx := NewContext(c, p)
	old := map[string]string{"name": c.FormValue("name"), "email": c.FormValue("email")}
	if !p.formGuard(c) {
		return p.renderRegister(ctx, http.StatusForbidden, "Your session expired. Please try again.", old)
	}
	user, msg, _, status := p.register(c, ctx, old["name"], old["email"], c.FormValue("password"), c.FormValue("password_confirmation"))
	if user == nil {
		return p.renderRegister(ctx, status, msg, old)
	}
	if err := p.startWebSession(c, user, "", p.ResolveTokenDuration(false)); err != nil {
		return c.Redirect(http.StatusSeeOther, p.path+"/login")
	}
	return c.Redirect(http.StatusSeeOther, p.path)
}

func (p *Panel) handleVerifyNotice(c *gcontext.Context) error {
	ctx, user, err := p.profileUser(c)
	if err != nil {
		return err
	}
	if !p.emailUnverified(ctx, user) {
		return c.Redirect(http.StatusSeeOther, p.path)
	}
	return p.renderVerifyNotice(ctx, http.StatusOK, c.Query("status") == "sent")
}

func (p *Panel) handleResendVerification(c *gcontext.Context) error {
	ctx, user, err := p.profileUser(c)
	if err != nil {
		return err
	}
	if !p.formGuard(c) {
		return p.renderVerifyNotice(ctx, http.StatusForbidden, false)
	}
	key := "verify:" + user.GetAuthIdentifier()
	if blocked, wait := p.loginBlocked(c, key); blocked {
		setRetryAfter(c, wait)
		return p.renderVerifyNotice(ctx, statusTooManyRequests, false)
	}
	p.loginFailed(c, key)
	if p.emailUnverified(ctx, user) {
		_ = p.sendVerification(c, ctx, user)
	}
	return c.Redirect(http.StatusSeeOther, p.path+"/email/verify?status=sent")
}

// verifyLink checks a verification link and marks the address verified.
func (p *Panel) verifyLink(ctx *Context, id, hashHex, expiresRaw, signature string) bool {
	expires, err := strconv.ParseInt(expiresRaw, 10, 64)
	if err != nil || id == "" {
		return false
	}
	user, err := p.userByID(ctx, id)
	if err != nil || user == nil {
		return false
	}
	_, email, err := p.accounts.Profile(ctx, user)
	if err != nil || email == "" || !p.verificationBroker().VerifySignature(id, strings.ToLower(email), expires, signature) {
		return false
	}
	_ = hashHex // the signature covers the address's hash
	if p.emailUnverified(ctx, user) {
		return p.accounts.MarkEmailVerified(ctx, user) == nil
	}
	return true
}

func (p *Panel) handleVerifyConfirm(c *gcontext.Context) error {
	ctx := NewContext(c, p)
	if !p.verifyLink(ctx, c.Query("id"), c.Query("hash"), c.Query("expires"), c.Query("signature")) {
		return p.renderCardPage(ctx, http.StatusForbidden, "Verify your email", "",
			alertHTML("err", "This verification link is invalid or has expired. Sign in to ask for a new one.")+
				fmt.Sprintf(`<p style="text-align: center; font-size: 0.8125rem;"><a href="%s/login">Sign in</a></p>`, p.path), false)
	}
	return p.renderCardPage(ctx, http.StatusOK, "Email verified", "",
		alertHTML("ok", "Thank you, your email address is verified.")+
			fmt.Sprintf(`<p style="text-align: center; font-size: 0.8125rem;"><a href="%s">Continue</a></p>`, p.path), false)
}

func (p *Panel) renderRegister(ctx *Context, status int, errMsg string, old map[string]string) error {
	if old == nil {
		old = map[string]string{}
	}
	body := alertHTML("err", errMsg) + fmt.Sprintf(`<form method="POST" action="%s/register">%s%s%s%s%s
<button type="submit" class="btn btn-primary" style="width: 100%%;">Create account</button></form>
<p style="text-align: center; margin-top: 1rem; font-size: 0.8125rem;">Already have an account? <a href="%s/login">Sign in</a></p>`,
		p.path, p.csrfHiddenInput(ctx.Context, ""),
		p.field("name", "Name", "text", old["name"], `required autocomplete="name" autofocus`),
		p.field("email", "Email address", "email", old["email"], `required autocomplete="email"`),
		p.field("password", "Password", "password", "", `required minlength="8" autocomplete="new-password"`),
		p.field("password_confirmation", "Confirm password", "password", "", `required autocomplete="new-password"`),
		p.path)
	return p.renderCardPage(ctx, status, "Create an account", "", body, false)
}

func (p *Panel) renderVerifyNotice(ctx *Context, status int, sent bool) error {
	msg := ""
	if sent {
		msg = "A new verification link was sent to your email address."
	}
	body := alertHTML("ok", msg) + fmt.Sprintf(`<p style="font-size: 0.875rem; margin-bottom: 1rem;">Before you continue, open the link we emailed you to verify your address.</p>
<form method="POST" action="%s/email/verification-notification">%s<button type="submit" class="btn btn-primary" style="width: 100%%;">Send the link again</button></form>
<p style="text-align: center; margin-top: 1rem; font-size: 0.8125rem;"><a href="%s/logout">Sign out</a></p>`,
		p.path, p.csrfHiddenInput(ctx.Context, ""), html.EscapeString(p.path))
	return p.renderCardPage(ctx, status, "Verify your email", "", body, false)
}
