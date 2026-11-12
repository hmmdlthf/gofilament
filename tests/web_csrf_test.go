package tests

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func TestCrossSiteWebPostsAreRefused(t *testing.T) {
	app, docs := newWebActionApp(t)
	w := newWebClient(t, app)
	token := w.csrfFrom(w.do("GET", "/admin/docs/1/actions/post", nil).Body.String())

	// A form on another site posting into the panel, even with a token.
	w.fetchSite, w.origin = "cross-site", "https://evil.example"
	if rec := w.do("POST", "/admin/docs/1/actions/post", url.Values{"_token": {token}}); rec.Code != http.StatusForbidden || docs["1"].Posted {
		t.Fatalf("cross-site action POST: %d posted=%v", rec.Code, docs["1"].Posted)
	}
	if rec := w.do("POST", "/admin/docs/create", url.Values{"number": {"X"}}); rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site create POST: %d", rec.Code)
	}
	// Old browsers send only Origin.
	w.fetchSite = ""
	if rec := w.do("POST", "/admin/docs/bulk", url.Values{"bulk_action": {"delete_bulk"}}); rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin bulk POST (Origin only): %d", rec.Code)
	}
	// Same-origin still works.
	w.fetchSite, w.origin = "same-origin", ""
	if rec := w.do("POST", "/admin/docs/1/actions/post", url.Values{"_token": {token}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("same-origin POST: %d", rec.Code)
	}
}

func TestTrustedOriginsMayPost(t *testing.T) {
	app := golaravel.New()
	p := filament.NewPanel("admin").Path("/admin").TrustedOrigins("https://admin.example.org").DatabaseNotifications()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	w := newWebClient(t, app)
	w.fetchSite, w.origin = "cross-site", "https://admin.example.org"
	if rec := w.do("POST", "/admin/notifications/clear", url.Values{}); rec.Code != http.StatusOK {
		t.Fatalf("trusted origin POST: %d", rec.Code)
	}
}

func TestGETDeleteFromAnotherSiteNeedsConfirmation(t *testing.T) {
	deleted := ""
	res := filament.NewResource[*actDoc]("docs")
	res.OnDelete(func(ctx *filament.Context, id string) error { deleted = id; return nil })
	app := golaravel.New()
	if err := filament.NewPanel("admin").Path("/admin").Resource(res).Register(app); err != nil {
		t.Fatal(err)
	}
	w := newWebClient(t, app)

	for _, site := range []string{"cross-site", "none", ""} {
		w.fetchSite = site
		rec := w.do("GET", "/admin/docs/7/delete", nil)
		if deleted != "" || rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `method="POST" action="/admin/docs/7/delete"`) {
			t.Fatalf("GET delete with Sec-Fetch-Site=%q must show a confirm page: %d deleted=%q", site, rec.Code, deleted)
		}
	}
	w.fetchSite = "same-origin"
	if rec := w.do("POST", "/admin/docs/7/delete", url.Values{}); rec.Code != http.StatusSeeOther || deleted != "7" {
		t.Fatalf("confirmed delete: %d deleted=%q", rec.Code, deleted)
	}
	deleted = ""
	if rec := w.do("GET", "/admin/docs/8/delete", nil); rec.Code != http.StatusSeeOther || deleted != "8" {
		t.Fatalf("in-panel delete link: %d deleted=%q", rec.Code, deleted)
	}
}

func TestLogoutFromAnotherSiteNeedsConfirmation(t *testing.T) {
	app, p := newMembershipPanel(t)
	w := newWebClient(t, app)
	signIn(t, w)
	w.fetchSite = "cross-site"
	w.do("GET", "/admin/logout", nil)
	if _, ok := w.cookies[p.SessionCookieName()]; !ok {
		t.Fatal("a cross-site GET signed the user out")
	}
	w.origin = "https://evil.example"
	if rec := w.do("POST", "/admin/logout", url.Values{}); rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site logout POST: %d", rec.Code)
	}
}

func TestSwitchRedirectsStayOnSite(t *testing.T) {
	app := golaravel.New()
	if err := filament.NewPanel("admin").Path("/admin").Register(app); err != nil {
		t.Fatal(err)
	}
	for ref, want := range map[string]string{
		"https://evil.example/phish":           "/admin",
		"//evil.example/x":                     "/admin",
		"http://example.com/admin/docs?page=2": "/admin/docs?page=2",
	} {
		req, _ := http.NewRequest("GET", "http://example.com/admin/locale/switch?lang=en", nil)
		req.Header.Set("Referer", ref)
		rec := doReq(app, req)
		if got := rec.Header().Get("Location"); got != want {
			t.Errorf("Referer %q: redirect %q, want %q", ref, got, want)
		}
	}
}

func doReq(app http.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}
