package filament

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/cache"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/hash"
	"github.com/golaravel/golaravel/pkg/router"
)

// The account screens over the JSON API (Flutter and other bearer-token
// clients): profile, password change, two-factor setup and recovery codes,
// forgot password, and the two-factor step of API sign-in.
//
// Two-factor sign-in through the API: when the user has two-factor on,
// POST /login answers {"two_factor": true, "challenge": "..."} instead of a
// token. The token the AuthValidator issued is parked in golaravel's cache
// and handed out by POST /login/two-factor {challenge, code} once a valid
// authenticator or recovery code is given. Use a shared cache store when the
// panel runs on several instances.

const apiTwoFactorTTL = 10 * time.Minute

type apiLoginChallenge struct {
	UserID    string         `json:"user_id"`
	Token     string         `json:"token"`
	User      map[string]any `json:"user"`
	ExpiresAt string         `json:"expires_at"`
	ExpiresIn int            `json:"expires_in"`
}

func (p *Panel) cacheKey(parts ...string) string {
	return "gofilament:" + p.id + ":" + strings.Join(parts, ":")
}

func hashKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:16])
}

// apiAccountsFeatures describes which account screens the API offers.
func (p *Panel) apiAccountsFeatures() map[string]bool {
	return map[string]bool{
		"profile":        p.accounts != nil && p.accounts.Profile != nil,
		"update_profile": p.accounts != nil && p.accounts.UpdateProfile != nil,
		"password":       p.canChangePassword(),
		"password_reset": p.canResetPassword(),
		"two_factor":     p.twoFactorAvailable(),
		"recovery_codes": p.recoveryCodesAvailable(),
	}
}

// apiBeginTwoFactor parks an issued token until the second step, when the
// user has two-factor sign-in on. It returns the challenge id ("" when no
// second step is needed).
func (p *Panel) apiBeginTwoFactor(ctx *Context, user auth.Authenticatable, login apiLoginChallenge) (string, error) {
	if p.userTwoFactorSecret(ctx, user) == "" {
		return "", nil
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	challenge := hex.EncodeToString(buf)
	raw, err := json.Marshal(login)
	if err != nil {
		return "", err
	}
	if err := cache.Put(p.cacheKey("api-2fa", hashKey(challenge)), string(raw), apiTwoFactorTTL); err != nil {
		return "", err
	}
	return challenge, nil
}

// userByID loads a user for the API two-factor step.
func (p *Panel) userByID(ctx *Context, id string) (auth.Authenticatable, error) {
	if p.sessionUsers != nil {
		return p.sessionUsers(ctx, id)
	}
	return &PanelUser{ID: id, Login: id}, nil
}

func (p *Panel) handleAPILoginTwoFactor(c *gcontext.Context) error {
	p.setCorsHeaders(c)
	var in struct {
		Challenge string `json:"challenge"`
		Code      string `json:"code"`
	}
	if err := c.BindJSON(&in); err != nil || in.Challenge == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}
	key := p.cacheKey("api-2fa", hashKey(in.Challenge))
	raw := cache.GetString(key)
	var login apiLoginChallenge
	if raw == "" || json.Unmarshal([]byte(raw), &login) != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Your sign-in expired. Please sign in again."})
	}
	limitKey := "2fa:" + login.UserID
	if blocked, wait := p.loginBlocked(c, limitKey); blocked {
		setRetryAfter(c, wait)
		return c.JSON(statusTooManyRequests, map[string]string{"error": throttleMessage(wait)})
	}
	ctx := NewContext(c, p)
	user, err := p.userByID(ctx, login.UserID)
	if err != nil || user == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Your sign-in expired. Please sign in again."})
	}
	secret := p.userTwoFactorSecret(ctx, user)
	code := strings.TrimSpace(in.Code)
	ok, step := p.totp.VerifyCodeStep(secret, code)
	valid := secret != "" && ok && p.acceptTOTPStep(login.UserID, secret, step)
	if !valid && secret != "" {
		valid = p.useRecoveryCode(ctx, user, code)
	}
	if !valid {
		p.loginFailed(c, limitKey)
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "The code is not valid."})
	}
	// Hand the token out once, even if two requests race.
	if first, err := cache.Add(key+":used", 1, apiTwoFactorTTL); err != nil || !first {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Your sign-in expired. Please sign in again."})
	}
	_ = cache.Forget(key)
	p.loginSucceeded(c, limitKey)
	return c.JSON(http.StatusOK, map[string]any{
		"status":     "success",
		"token":      login.Token,
		"user":       login.User,
		"expires_at": login.ExpiresAt,
		"expires_in": login.ExpiresIn,
	})
}

// ------------------------------------------------------------ account API

func (p *Panel) apiAccountUser(c *gcontext.Context) (*Context, auth.Authenticatable, bool) {
	p.setCorsHeaders(c)
	ctx, err := p.AuthorizeAccess(c)
	if err != nil {
		_ = c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthenticated"})
		return nil, nil, false
	}
	u := ctx.User()
	if u == nil || p.accounts == nil {
		_ = c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthenticated"})
		return nil, nil, false
	}
	return ctx, u, true
}

func apiFail(c *gcontext.Context, status int, msg string) error {
	return c.JSON(status, map[string]string{"error": msg})
}

func (p *Panel) apiAccountState(ctx *Context, user auth.Authenticatable) map[string]any {
	out := map[string]any{
		"id":       user.GetAuthIdentifier(),
		"features": p.apiAccountsFeatures(),
	}
	if p.accounts.Profile != nil {
		if name, email, err := p.accounts.Profile(ctx, user); err == nil {
			out["name"], out["email"] = name, email
		}
	}
	if p.twoFactorAvailable() {
		out["two_factor_enabled"] = p.userTwoFactorSecret(ctx, user) != ""
		if p.recoveryCodesAvailable() {
			if hashes, err := p.accounts.RecoveryCodes(ctx, user); err == nil {
				out["recovery_codes_left"] = len(hashes)
			}
		}
	}
	return out
}

func (p *Panel) mountAccountAPIRoutes(r *router.Router, prefix string, preflight func(c *gcontext.Context) error) {
	r.OPTIONS(prefix+"/login/two-factor", preflight)
	r.POST(prefix+"/login/two-factor", p.handleAPILoginTwoFactor)
	if p.accounts == nil {
		return
	}
	for _, path := range []string{"/account", "/account/password", "/account/two-factor/enable", "/account/two-factor/confirm",
		"/account/two-factor/disable", "/account/two-factor/recovery-codes", "/forgot-password"} {
		r.OPTIONS(prefix+path, preflight)
	}

	r.GET(prefix+"/account", func(c *gcontext.Context) error {
		ctx, user, ok := p.apiAccountUser(c)
		if !ok {
			return nil
		}
		return c.JSON(http.StatusOK, p.apiAccountState(ctx, user))
	})

	r.PUT(prefix+"/account", func(c *gcontext.Context) error {
		ctx, user, ok := p.apiAccountUser(c)
		if !ok {
			return nil
		}
		if p.accounts.UpdateProfile == nil {
			return apiFail(c, http.StatusNotFound, "Not found")
		}
		data := parseRequestData(c)
		name := strings.TrimSpace(fmt.Sprint(data["name"]))
		email := strings.TrimSpace(fmt.Sprint(data["email"]))
		if data["name"] == nil || data["email"] == nil || name == "" || !strings.Contains(email, "@") {
			return c.JSON(http.StatusUnprocessableEntity, map[string]any{"error": "Enter a name and a valid email address.",
				"errors": map[string]string{"name": "Required.", "email": "Enter a valid email address."}})
		}
		if err := p.accounts.UpdateProfile(ctx, user, name, email); err != nil {
			return apiFail(c, http.StatusUnprocessableEntity, err.Error())
		}
		return c.JSON(http.StatusOK, p.apiAccountState(ctx, user))
	})

	r.POST(prefix+"/account/password", func(c *gcontext.Context) error {
		ctx, user, ok := p.apiAccountUser(c)
		if !ok {
			return nil
		}
		if !p.canChangePassword() {
			return apiFail(c, http.StatusNotFound, "Not found")
		}
		data := parseRequestData(c)
		key := "password:" + user.GetAuthIdentifier()
		if blocked, wait := p.loginBlocked(c, key); blocked {
			setRetryAfter(c, wait)
			return apiFail(c, statusTooManyRequests, throttleMessage(wait))
		}
		if !p.verifyPassword(ctx, user, formString(data, "current_password")) {
			p.loginFailed(c, key)
			return apiFail(c, http.StatusUnprocessableEntity, "The current password is not correct.")
		}
		p.loginSucceeded(c, key)
		pw := formString(data, "password")
		if msg := checkNewPassword(pw, formString(data, "password_confirmation")); msg != "" {
			return apiFail(c, http.StatusUnprocessableEntity, msg)
		}
		h, err := hash.Make(pw)
		if err == nil {
			err = p.accounts.UpdatePassword(ctx, user, h)
		}
		if err != nil {
			return apiFail(c, http.StatusInternalServerError, "The password could not be changed.")
		}
		// Browser sessions of the user are signed out; bearer tokens stay
		// valid until they expire (revoke them in the app if it keeps them).
		if p.sessionManager != nil {
			_ = p.sessionManager.DestroyUserSessions(user.GetAuthIdentifier())
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "Your password was changed."})
	})

	setupKey := func(user auth.Authenticatable) string {
		return p.cacheKey("api-2fa-setup", hashKey(user.GetAuthIdentifier()))
	}

	r.POST(prefix+"/account/two-factor/enable", func(c *gcontext.Context) error {
		ctx, user, ok := p.apiAccountUser(c)
		if !ok {
			return nil
		}
		if !p.twoFactorAvailable() {
			return apiFail(c, http.StatusNotFound, "Not found")
		}
		if p.accounts.PasswordHash != nil && !p.verifyPassword(ctx, user, formString(parseRequestData(c), "current_password")) {
			return apiFail(c, http.StatusUnprocessableEntity, "The current password is not correct.")
		}
		secret, err := p.totp.GenerateSecret()
		if err == nil {
			err = cache.Put(setupKey(user), secret, 15*time.Minute)
		}
		if err != nil {
			return apiFail(c, http.StatusInternalServerError, "Two-factor sign-in could not be set up.")
		}
		return c.JSON(http.StatusOK, map[string]string{
			"secret":      secret,
			"otpauth_uri": p.totp.ProvisioningURI(user.GetAuthIdentifier(), secret),
		})
	})

	r.POST(prefix+"/account/two-factor/confirm", func(c *gcontext.Context) error {
		ctx, user, ok := p.apiAccountUser(c)
		if !ok {
			return nil
		}
		if !p.twoFactorAvailable() {
			return apiFail(c, http.StatusNotFound, "Not found")
		}
		secret := cache.GetString(setupKey(user))
		if secret == "" {
			return apiFail(c, http.StatusUnprocessableEntity, "Start the setup again.")
		}
		code := strings.TrimSpace(formString(parseRequestData(c), "code"))
		if ok, step := p.totp.VerifyCodeStep(secret, code); !ok || !p.acceptTOTPStep(user.GetAuthIdentifier(), secret, step) {
			return apiFail(c, http.StatusUnprocessableEntity, "The code is not valid. Check the time on your phone and try again.")
		}
		if err := p.accounts.SaveTwoFactorSecret(ctx, user, secret); err != nil {
			return apiFail(c, http.StatusInternalServerError, "Two-factor sign-in could not be turned on.")
		}
		_ = cache.Forget(setupKey(user))
		out := p.apiAccountState(ctx, user)
		if p.recoveryCodesAvailable() {
			codes, hashes, err := NewRecoveryCodes(RecoveryCodeCount)
			if err == nil {
				err = p.accounts.SaveRecoveryCodes(ctx, user, hashes)
			}
			if err == nil {
				out["recovery_codes"] = codes
				out["recovery_codes_left"] = len(codes)
			}
		}
		return c.JSON(http.StatusOK, out)
	})

	r.POST(prefix+"/account/two-factor/disable", func(c *gcontext.Context) error {
		ctx, user, ok := p.apiAccountUser(c)
		if !ok {
			return nil
		}
		if !p.twoFactorAvailable() {
			return apiFail(c, http.StatusNotFound, "Not found")
		}
		data := parseRequestData(c)
		if !p.verifyPassword(ctx, user, formString(data, "current_password")) {
			secret := p.userTwoFactorSecret(ctx, user)
			ok, step := p.totp.VerifyCodeStep(secret, strings.TrimSpace(formString(data, "code")))
			if secret == "" || !ok || !p.acceptTOTPStep(user.GetAuthIdentifier(), secret, step) {
				return apiFail(c, http.StatusUnprocessableEntity, "Enter your current password (or a code) to turn two-factor sign-in off.")
			}
		}
		if err := p.accounts.SaveTwoFactorSecret(ctx, user, ""); err != nil {
			return apiFail(c, http.StatusInternalServerError, "Two-factor sign-in could not be turned off.")
		}
		if p.recoveryCodesAvailable() {
			_ = p.accounts.SaveRecoveryCodes(ctx, user, nil)
		}
		return c.JSON(http.StatusOK, p.apiAccountState(ctx, user))
	})

	r.POST(prefix+"/account/two-factor/recovery-codes", func(c *gcontext.Context) error {
		ctx, user, ok := p.apiAccountUser(c)
		if !ok {
			return nil
		}
		if !p.recoveryCodesAvailable() || p.userTwoFactorSecret(ctx, user) == "" {
			return apiFail(c, http.StatusNotFound, "Not found")
		}
		if p.accounts.PasswordHash != nil && !p.verifyPassword(ctx, user, formString(parseRequestData(c), "current_password")) {
			return apiFail(c, http.StatusUnprocessableEntity, "Enter your current password to make new recovery codes.")
		}
		codes, hashes, err := NewRecoveryCodes(RecoveryCodeCount)
		if err == nil {
			err = p.accounts.SaveRecoveryCodes(ctx, user, hashes)
		}
		if err != nil {
			return apiFail(c, http.StatusInternalServerError, "Recovery codes could not be made.")
		}
		out := p.apiAccountState(ctx, user)
		out["recovery_codes"] = codes
		return c.JSON(http.StatusOK, out)
	})

	// Forgot password: mails a link to the web reset page.
	r.POST(prefix+"/forgot-password", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		if !p.canResetPassword() {
			return apiFail(c, http.StatusNotFound, "Not found")
		}
		ctx := NewContext(c, p)
		email := strings.ToLower(strings.TrimSpace(formString(parseRequestData(c), "email")))
		if !strings.Contains(email, "@") {
			return apiFail(c, http.StatusUnprocessableEntity, "Enter your email address.")
		}
		key := "reset:" + email
		if blocked, wait := p.loginBlocked(c, key); blocked {
			setRetryAfter(c, wait)
			return apiFail(c, statusTooManyRequests, throttleMessage(wait))
		}
		p.loginFailed(c, key)
		p.sendResetLink(c, ctx, email)
		return c.JSON(http.StatusOK, map[string]string{"message": "If an account uses that address, we have emailed a link to reset the password."})
	})
}

func formString(data map[string]any, key string) string {
	if v, ok := data[key]; ok && v != nil {
		return fmt.Sprint(v)
	}
	return ""
}
