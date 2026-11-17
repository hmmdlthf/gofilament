package filament

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/golaravel/golaravel/pkg/auth"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/session"
)

// Two-factor recovery codes: single-use codes a user can sign in with when
// they lose their authenticator. Only their SHA-256 hashes are stored (the
// codes carry 50 random bits each, so a fast hash is enough); the plain
// codes are shown once, when they are made.

// RecoveryCodeCount is how many recovery codes a user gets.
const RecoveryCodeCount = 8

const sessRecoveryCodes = "_filament_2fa_codes"

func (p *Panel) recoveryCodesAvailable() bool {
	return p.twoFactorAvailable() && p.accounts.RecoveryCodes != nil && p.accounts.SaveRecoveryCodes != nil
}

// NewRecoveryCodes makes n codes like "k3j9d-8x2mf" and their hashes.
func NewRecoveryCodes(n int) (codes, hashes []string, err error) {
	const alphabet = "abcdefghjkmnpqrstuvwxyz23456789" // no 0/o, 1/l/i
	for i := 0; i < n; i++ {
		buf := make([]byte, 10)
		if _, err := rand.Read(buf); err != nil {
			return nil, nil, err
		}
		var sb strings.Builder
		for j, b := range buf {
			if j == 5 {
				sb.WriteByte('-')
			}
			sb.WriteByte(alphabet[int(b)%len(alphabet)])
		}
		codes = append(codes, sb.String())
		hashes = append(hashes, HashRecoveryCode(sb.String()))
	}
	return codes, hashes, nil
}

// HashRecoveryCode is the stored form of a recovery code (case, spaces and
// dashes ignored).
func HashRecoveryCode(code string) string {
	c := strings.ToLower(strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(code)))
	sum := sha256.Sum256([]byte("gofilament-recovery:" + c))
	return hex.EncodeToString(sum[:])
}

func looksLikeRecoveryCode(code string) bool {
	c := strings.NewReplacer("-", "", " ", "").Replace(strings.TrimSpace(code))
	return len(c) == 10
}

// useRecoveryCode checks code against the user's codes and, when it
// matches, stores the rest so it cannot be used again.
func (p *Panel) useRecoveryCode(ctx *Context, user auth.Authenticatable, code string) bool {
	if !p.recoveryCodesAvailable() || !looksLikeRecoveryCode(code) {
		return false
	}
	hashes, err := p.accounts.RecoveryCodes(ctx, user)
	if err != nil {
		return false
	}
	want := HashRecoveryCode(code)
	idx := -1
	for i, h := range hashes {
		if subtle.ConstantTimeCompare([]byte(h), []byte(want)) == 1 {
			idx = i
		}
	}
	if idx < 0 {
		return false
	}
	rest := append(append([]string{}, hashes[:idx]...), hashes[idx+1:]...)
	return p.accounts.SaveRecoveryCodes(ctx, user, rest) == nil
}

// issueRecoveryCodes replaces the user's codes and keeps the plain codes in
// the session until the profile page has shown them once.
func (p *Panel) issueRecoveryCodes(c *gcontext.Context, ctx *Context, user auth.Authenticatable) error {
	if !p.recoveryCodesAvailable() {
		return nil
	}
	codes, hashes, err := NewRecoveryCodes(RecoveryCodeCount)
	if err != nil {
		return err
	}
	if err := p.accounts.SaveRecoveryCodes(ctx, user, hashes); err != nil {
		return err
	}
	if sess := session.FromContext(c); sess != nil {
		sess.Put(sessRecoveryCodes, strings.Join(codes, " "))
	}
	return nil
}

func (p *Panel) handleRecoveryCodesRegenerate(c *gcontext.Context) error {
	ctx, user, err := p.profileUser(c)
	if err != nil {
		return err
	}
	if !p.recoveryCodesAvailable() || p.userTwoFactorSecret(ctx, user) == "" {
		return c.String(http.StatusNotFound, "404 Not Found")
	}
	if !p.formGuard(c) {
		return p.renderProfile(ctx, user, http.StatusForbidden, "", "Your session expired. Please try again.", nil)
	}
	if p.accounts.PasswordHash != nil && !p.verifyPassword(ctx, user, c.FormValue("current_password")) {
		return p.renderProfile(ctx, user, http.StatusUnprocessableEntity, "", "Enter your current password to make new recovery codes.", nil)
	}
	if err := p.issueRecoveryCodes(c, ctx, user); err != nil {
		return p.renderProfile(ctx, user, http.StatusInternalServerError, "", "Recovery codes could not be made.", nil)
	}
	return c.Redirect(http.StatusSeeOther, p.path+"/profile?status=codes")
}

// recoveryCodesHTML is the recovery-code part of the profile's two-factor
// section: new codes (shown once), how many are left, and a button to
// make new ones.
func (p *Panel) recoveryCodesHTML(ctx *Context, user auth.Authenticatable, csrf, pwField string) string {
	if !p.recoveryCodesAvailable() {
		return ""
	}
	var b strings.Builder
	if sess := session.FromContext(ctx.Context); sess != nil {
		if fresh := sess.GetString(sessRecoveryCodes); fresh != "" {
			sess.Forget(sessRecoveryCodes)
			b.WriteString(`<div class="alert alert-success" style="margin: 1rem 0;" data-recovery-codes><p style="font-size: 0.875rem; margin-bottom: 0.5rem;"><strong>Save these recovery codes.</strong> Each one signs you in once if you lose your phone. They will not be shown again.</p><ul style="font-family: monospace; columns: 2; list-style: none; padding: 0; margin: 0;">`)
			for _, code := range strings.Fields(fresh) {
				fmt.Fprintf(&b, `<li>%s</li>`, html.EscapeString(code))
			}
			b.WriteString(`</ul></div>`)
		}
	}
	left := 0
	if hashes, err := p.accounts.RecoveryCodes(ctx, user); err == nil {
		left = len(hashes)
	}
	fmt.Fprintf(&b, `<p style="font-size: 0.875rem; margin: 1rem 0 0.75rem;">Recovery codes left: <strong>%d</strong></p>
<form method="POST" action="%s/profile/two-factor/recovery-codes">%s%s<button type="submit" class="btn btn-secondary">Make new recovery codes</button></form>`, left, p.path, csrf, pwField)
	return b.String()
}
