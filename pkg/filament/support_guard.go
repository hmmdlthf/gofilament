package filament

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	gcontext "github.com/golaravel/golaravel/pkg/context"
	gosupport "github.com/golaravel/golaravel/pkg/support"
)

// guestSupportKey owns tickets opened by visitors who are not signed in.
// Such a ticket is reachable only by its (unguessable) ID.
const guestSupportKey = "guest"

// newSupportTicketID returns a ticket ID with 128 bits of randomness. The
// ID is the only credential an anonymous visitor has for their ticket, so
// it must not be guessable; clients never choose it.
func newSupportTicketID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "TKT-" + hex.EncodeToString(b)
}

// SupportStaff decides which signed-in users act as support staff (may
// read, answer and escalate every ticket in their scope). The default is
// any signed-in user who passes the panel's CanAccess check.
func (p *Panel) SupportStaff(fn func(ctx *Context) bool) *Panel {
	p.supportStaff = fn
	return p
}

type supportCaller struct {
	ctx   *Context
	key   string // user id, or guestSupportKey
	staff bool
}

func (p *Panel) supportCallerFor(c *gcontext.Context) supportCaller {
	ctx := p.resolveUser(c)
	sc := supportCaller{ctx: ctx, key: guestSupportKey}
	if u := ctx.User(); u != nil {
		sc.key = u.GetAuthIdentifier()
		switch {
		case p.supportStaff != nil:
			sc.staff = p.supportStaff(ctx)
		default:
			sc.staff = p.canAccess == nil || p.canAccess(ctx)
		}
	}
	return sc
}

// canSee: staff within their tenant scope, or the ticket's owner (a guest
// ticket is owned by whoever holds its ID).
func (p *Panel) supportCanSee(sc supportCaller, t *gosupport.Ticket) bool {
	if t == nil {
		return false
	}
	if sc.staff {
		if (p.GetSupportScope() == "tenant_admin" || p.hasTenantScoping) && t.TenantID != "" {
			return t.TenantID == sc.ctx.Tenant()
		}
		return true
	}
	return t.UserKey == sc.key
}

// loadSupportTicket loads a ticket the caller may see, answering 404
// otherwise (the same answer for "missing" and "not yours", so IDs cannot
// be probed).
func (p *Panel) loadSupportTicket(c *gcontext.Context, id string) (*gosupport.Ticket, supportCaller, bool) {
	sc := p.supportCallerFor(c)
	if id == "" {
		_ = c.JSON(http.StatusBadRequest, map[string]string{"error": "Ticket ID is required"})
		return nil, sc, false
	}
	t, err := gosupport.GetDefaultStore().GetTicket(c.Request.Context(), id)
	if err != nil || !p.supportCanSee(sc, t) {
		_ = c.JSON(http.StatusNotFound, map[string]string{"error": "Ticket not found"})
		return nil, sc, false
	}
	return t, sc, true
}
