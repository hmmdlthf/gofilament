package main

import (
	"fmt"
	"html"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Protections for a public sandbox, around the whole app:
//   - request bodies are capped (imports included);
//   - each client IP gets a request budget per minute, smaller for writes;
//   - nothing is indexed by search engines.
//
// There are no uploads (no file fields), no mail (no password reset or
// sign-up) and no payment gateway; data resets every hour (seed.go).

const (
	maxBody       = 1 << 20 // 1 MB per request
	readsPerMin   = 300
	writesPerMin  = 40
	limiterWindow = time.Minute
)

func sandbox(next http.Handler) http.Handler {
	lim := &limiter{hits: map[string]*bucket{}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
		write := r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions
		if !lim.allow(clientIP(r), write, time.Now()) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too many requests from your address. The demo allows "+fmt.Sprint(readsPerMin)+" requests a minute; try again shortly.", http.StatusTooManyRequests)
			return
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP is the address goplesk's proxy saw: the last X-Forwarded-For
// entry (the one the proxy appended), else the connection's address.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type bucket struct {
	start         time.Time
	reads, writes int
}

type limiter struct {
	mu    sync.Mutex
	hits  map[string]*bucket
	swept time.Time
}

func (l *limiter) allow(ip string, write bool, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.swept) > limiterWindow {
		for k, b := range l.hits {
			if now.Sub(b.start) > limiterWindow {
				delete(l.hits, k)
			}
		}
		l.swept = now
	}
	b := l.hits[ip]
	if b == nil || now.Sub(b.start) > limiterWindow {
		b = &bucket{start: now}
		l.hits[ip] = b
	}
	b.reads++
	if write {
		b.writes++
	}
	return b.reads <= readsPerMin && b.writes <= writesPerMin
}

func homePage(built bool) string {
	open := `<a class="btn" href="/admin">Open the admin →</a>`
	if !built {
		open = `<p class="muted">The Flutter app is not built into this binary.</p>`
	}
	var accounts strings.Builder
	for _, u := range demoUsers {
		fmt.Fprintf(&accounts, `<li><strong>%s</strong>: %s<br><code>%s</code> / <code>%s</code></li>`,
			html.EscapeString(u.roleLabel), html.EscapeString(u.description), html.EscapeString(u.email), demoPassword)
	}
	return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + brand + ` · GoFilament live demo</title><meta name="robots" content="noindex">
<link rel="icon" href="/admin/favicon.png">
<style>
:root{color-scheme:light dark;--bg:#f7f8fa;--card:#fff;--text:#111827;--muted:#4b5563;--line:#e5e7eb;--accent:#059669}
@media (prefers-color-scheme:dark){:root{--bg:#0b0f17;--card:#141a26;--text:#e8ebf2;--muted:#9aa3b5;--line:#232a38;--accent:#34d399}}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:16px/1.6 system-ui,-apple-system,"Segoe UI",sans-serif}
main{max-width:760px;margin:0 auto;padding:56px 16px}h1{font-size:clamp(28px,5vw,44px);line-height:1.15;margin:8px 0 12px;text-wrap:balance}
.lead{color:var(--muted);margin:0 0 28px}.tag{font-size:12px;font-weight:600;letter-spacing:.04em;text-transform:uppercase;color:var(--accent)}
.btn{display:inline-block;padding:12px 22px;border-radius:10px;background:var(--accent);color:#fff;font-weight:600;text-decoration:none}
.muted{color:var(--muted)}.note{margin-top:36px;padding:16px 20px;border-radius:12px;border:1px solid var(--line);background:var(--card);color:var(--muted);font-size:14px}
.note ul{margin:8px 0 0;padding-left:20px}.note li{margin-bottom:6px}code{font-size:13px}a{color:var(--accent)}
</style></head><body><main>
<span class="tag">GoFilament live demo</span>
<h1>` + brand + `: a store back office in Flutter, driven by Go</h1>
<p class="lead">Products with variants and barcodes, orders with an audit trail, customers, a dashboard with charts, roles, filters, bulk edits, import and export, global search and dark mode. The screens are the gofilament_admin Flutter package; everything on them comes from the Go panel's API, so the same app runs on the web, iOS, Android and desktop.</p>
` + open + `
<div class="note">Sign in with a demo account (one click on the login screen):<ul>` + accounts.String() + `</ul>
<p>Changes are shared with other visitors and go back to the sample data every hour. No email is sent and there are no payments.</p>
<p><a href="https://gofilament.elridhma.com/">GoFilament</a> · <a href="https://gofilament.elridhma.com/docs/latest/">Docs</a> · <a href="https://github.com/hmmdlthf/gofilament">GitHub</a></p></div>
</main></body></html>`
}
