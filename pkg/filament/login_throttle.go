package filament

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/ratelimit"
)

// loginThrottle limits failed sign-in attempts. Counters live in golaravel's
// rate limit store (in memory by default; use ratelimit.SetStore with a Redis
// store when the panel runs on several processes).
type loginThrottle struct {
	maxAttempts int           // per client IP and identifier
	window      time.Duration // how long failures are remembered
	disabled    bool
}

func (p *Panel) throttle() loginThrottle {
	if p.loginLimit == nil {
		return loginThrottle{maxAttempts: 5, window: time.Minute}
	}
	return *p.loginLimit
}

// LoginThrottle sets how many failed sign-ins a client may make for one
// login name within window (default 5 per minute) before it is answered 429.
// A client is also limited to 4x that many failures across all names, so
// trying many accounts is slowed down as well. maxAttempts <= 0 disables it.
// Applies to POST <panel>/api/login and the web engine login form.
func (p *Panel) LoginThrottle(maxAttempts int, window time.Duration) *Panel {
	p.loginLimit = &loginThrottle{maxAttempts: maxAttempts, window: window, disabled: maxAttempts <= 0}
	return p
}

func (p *Panel) loginKeys(c *gcontext.Context, identifier string) (perName, perIP string) {
	ip := c.ClientIP()
	base := "gofilament:login:" + p.id + ":" + ip
	return base + ":" + strings.ToLower(strings.TrimSpace(identifier)), base
}

// loginBlocked reports whether the client must wait, and for how long.
func (p *Panel) loginBlocked(c *gcontext.Context, identifier string) (bool, time.Duration) {
	t := p.throttle()
	if t.disabled {
		return false, 0
	}
	perName, perIP := p.loginKeys(c, identifier)
	for _, k := range []struct {
		key string
		max int
	}{{perName, t.maxAttempts}, {perIP, t.maxAttempts * 4}} {
		if res, err := ratelimit.Status(k.key, k.max, t.window); err == nil && !res.Allowed {
			return true, res.RetryAfter
		}
	}
	return false, 0
}

// loginFailed records a failed attempt.
func (p *Panel) loginFailed(c *gcontext.Context, identifier string) {
	t := p.throttle()
	if t.disabled {
		return
	}
	perName, perIP := p.loginKeys(c, identifier)
	_, _ = ratelimit.Hit(perName, t.maxAttempts, t.window)
	_, _ = ratelimit.Hit(perIP, t.maxAttempts*4, t.window)
}

// loginSucceeded forgets the failures for this login name.
func (p *Panel) loginSucceeded(c *gcontext.Context, identifier string) {
	if p.throttle().disabled {
		return
	}
	perName, _ := p.loginKeys(c, identifier)
	_ = ratelimit.Reset(perName)
}

func throttleMessage(wait time.Duration) string {
	secs := int(math.Ceil(wait.Seconds()))
	if secs < 1 {
		secs = 1
	}
	return fmt.Sprintf("Too many sign-in attempts. Please try again in %d seconds.", secs)
}

func setRetryAfter(c *gcontext.Context, wait time.Duration) {
	secs := int(math.Ceil(wait.Seconds()))
	if secs < 1 {
		secs = 1
	}
	c.SetHeader("Retry-After", fmt.Sprint(secs))
}

const statusTooManyRequests = http.StatusTooManyRequests
