package tests

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func newSupportApp(t *testing.T) http.Handler {
	t.Helper()
	p := filament.NewPanel("desk").Path("/desk").Engine(filament.EngineFlutter).Flutter(t.TempDir()).
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			switch token {
			case "alice", "bob", "staff":
				return &filament.PanelUser{ID: token, Login: token}, nil
			}
			return nil, fmt.Errorf("bad token")
		}).
		SupportStaff(func(ctx *filament.Context) bool { return ctx.User() != nil && ctx.User().GetAuthIdentifier() == "staff" })
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app
}

func supportCall(t *testing.T, app http.Handler, method, path, token string, body any) (int, map[string]any, http.Header) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, "http://example.com/desk/api"+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out, rec.Header()
}

func ticketOf(m map[string]any) map[string]any { t, _ := m["ticket"].(map[string]any); return t }

func TestSupportTicketsBelongToTheirOwner(t *testing.T) {
	app := newSupportApp(t)

	// A visitor's identity fields are ignored; the ID is server-made.
	code, out, _ := supportCall(t, app, "POST", "/support/message", "", map[string]any{
		"content": "hello", "user_id": "alice", "user_email": "alice@example.com", "tenant_id": "acme",
	})
	guest := ticketOf(out)
	if code != 200 || guest["user_key"] != "guest" || guest["tenant_id"] != nil && guest["tenant_id"] != "" {
		t.Fatalf("anonymous ticket must not take the claimed identity: %d %v", code, guest)
	}
	if id, _ := guest["id"].(string); len(id) != len("TKT-")+32 {
		t.Fatalf("ticket IDs must carry 128 bits of randomness: %q", id)
	}

	// Clients cannot open a ticket under an ID of their choosing.
	if code, _, _ := supportCall(t, app, "POST", "/support/message", "", map[string]any{"content": "x", "ticket_id": "TKT-deadbeef"}); code != 404 {
		t.Fatalf("made-up ticket ID: %d", code)
	}

	_, out, _ = supportCall(t, app, "POST", "/support/message", "alice", map[string]any{"content": "my invoice is wrong"})
	aliceID := ticketOf(out)["id"].(string)

	for _, tc := range []struct {
		who  string
		want int
	}{{"bob", 404}, {"", 404}, {"alice", 200}, {"staff", 200}} {
		code, out, _ := supportCall(t, app, "POST", "/support/message", tc.who, map[string]any{"content": "ping", "ticket_id": aliceID})
		if code != tc.want {
			t.Errorf("%q posting to alice's ticket: %d, want %d", tc.who, code, tc.want)
		}
		if tc.want == 404 && ticketOf(out) != nil {
			t.Errorf("%q received alice's ticket", tc.who)
		}
	}

	// Escalation: only visible tickets; super admin only by staff.
	if code, _, _ := supportCall(t, app, "POST", "/support/escalate", "bob", map[string]any{"ticket_id": aliceID}); code != 404 {
		t.Errorf("bob escalating alice's ticket: %d", code)
	}
	if code, _, _ := supportCall(t, app, "POST", "/support/escalate", "alice", map[string]any{"ticket_id": aliceID, "target": "super_admin"}); code != 403 {
		t.Errorf("customer escalating to super admin: %d", code)
	}
	if code, _, _ := supportCall(t, app, "POST", "/support/escalate", "alice", map[string]any{"ticket_id": aliceID}); code != 200 {
		t.Errorf("owner escalating to live support: %d", code)
	}

	// Listing: customers see only their own tickets.
	_, out, _ = supportCall(t, app, "GET", "/support/tickets", "bob", nil)
	if list, _ := out["tickets"].([]any); len(list) != 0 {
		t.Errorf("bob sees other people's tickets: %v", list)
	}
	_, out, _ = supportCall(t, app, "GET", "/support/tickets", "alice", nil)
	for _, it := range out["tickets"].([]any) {
		if it.(map[string]any)["user_key"] != "alice" {
			t.Errorf("alice sees someone else's ticket: %v", it)
		}
	}
}

func TestSupportVoiceAudioIsPrivateAndServedAsAudio(t *testing.T) {
	app := newSupportApp(t)
	html := base64.StdEncoding.EncodeToString([]byte("<script>alert(1)</script>"))
	code, out, _ := supportCall(t, app, "POST", "/support/voice-feature", "alice", map[string]any{
		"audio_base64": html, "mime_type": "text/html", "ticket_id": "TKT-chosen-by-client",
	})
	if code != 200 {
		t.Fatalf("voice request: %d %v", code, out)
	}
	id := ticketOf(out)["id"].(string)
	if id == "TKT-chosen-by-client" || !strings.HasPrefix(id, "TKT-") {
		t.Fatalf("voice ticket ID must be server-made: %q", id)
	}
	if code, _, _ := supportCall(t, app, "GET", "/support/audio/"+id, "bob", nil); code != 404 {
		t.Fatalf("another user fetched the recording: %d", code)
	}
	code, _, hdr := supportCall(t, app, "GET", "/support/audio/"+id, "alice", nil)
	if code != 200 || hdr.Get("Content-Type") == "text/html" || hdr.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("owner audio: %d type=%q", code, hdr.Get("Content-Type"))
	}
}
