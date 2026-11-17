package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/hash"
)

func (a *accountApp) api(method, path, token string, body map[string]any) (int, map[string]any) {
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(method, "http://example.com/admin/api"+path, strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	a.handler.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestAPIAccountProfileAndPassword(t *testing.T) {
	a := newAccountApp(t)
	tok := "tok-ann@example.com"
	if code, _ := a.api("GET", "/account", "", nil); code != http.StatusUnauthorized {
		t.Fatalf("anonymous account: %d", code)
	}
	code, acc := a.api("GET", "/account", tok, nil)
	feats, _ := acc["features"].(map[string]any)
	if code != 200 || acc["name"] != "Ann" || feats["two_factor"] != true || feats["recovery_codes"] != true || acc["two_factor_enabled"] != false {
		t.Fatalf("account: %d %v", code, acc)
	}
	if code, _ := a.api("PUT", "/account", tok, map[string]any{"name": "", "email": "x"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid profile: %d", code)
	}
	if code, acc := a.api("PUT", "/account", tok, map[string]any{"name": "Annie", "email": "ann@example.com"}); code != 200 || acc["name"] != "Annie" {
		t.Fatalf("profile: %d %v", code, acc)
	}
	if code, _ := a.api("POST", "/account/password", tok, map[string]any{"current_password": "nope", "password": "new-password-1", "password_confirmation": "new-password-1"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("wrong current password: %d", code)
	}
	if code, _ := a.api("POST", "/account/password", tok, map[string]any{"current_password": "secret-pass", "password": "new-password-1", "password_confirmation": "new-password-1"}); code != 200 || !hash.Check("new-password-1", a.users["ann@example.com"].pwHash) {
		t.Fatalf("change password: %d", code)
	}
	// Boot tells the app which screens exist.
	_, boot := a.api("GET", "/boot", "", nil)
	if authInfo, _ := boot["auth"].(map[string]any); authInfo["accounts"] == nil {
		t.Fatalf("boot has no accounts features: %v", boot["auth"])
	}
}

func TestAPIForgotPassword(t *testing.T) {
	a := newAccountApp(t)
	for _, email := range []string{"ann@example.com", "nobody@example.com"} {
		code, out := a.api("POST", "/forgot-password", "", map[string]any{"email": email})
		if code != 200 || !strings.Contains(out["message"].(string), "If an account") {
			t.Fatalf("%s: %d %v", email, code, out)
		}
	}
	if len(a.mails) != 1 || !strings.Contains(a.mails[0], "/admin/reset-password?") {
		t.Fatalf("mails: %v", a.mails)
	}
}

func TestAPITwoFactorSetupAndSignIn(t *testing.T) {
	a := newAccountApp(t)
	tok := "tok-ann@example.com"
	totp := auth.NewTOTPManager()

	code, setup := a.api("POST", "/account/two-factor/enable", tok, map[string]any{"current_password": "secret-pass"})
	secret, _ := setup["secret"].(string)
	if code != 200 || secret == "" || !strings.HasPrefix(setup["otpauth_uri"].(string), "otpauth://") {
		t.Fatalf("enable: %d %v", code, setup)
	}
	if code, _ := a.api("POST", "/account/two-factor/confirm", tok, map[string]any{"code": "000000"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("wrong code: %d", code)
	}
	c1, _ := totp.GenerateCode(secret)
	code, done := a.api("POST", "/account/two-factor/confirm", tok, map[string]any{"code": c1})
	codes, _ := done["recovery_codes"].([]any)
	if code != 200 || done["two_factor_enabled"] != true || len(codes) != 8 || a.users["ann@example.com"].secret != secret {
		t.Fatalf("confirm: %d %v", code, done)
	}

	// API sign-in now has a second step; no token before it.
	code, login := a.api("POST", "/login", "", map[string]any{"username": "ann@example.com", "password": "secret-pass"})
	challenge, _ := login["challenge"].(string)
	if code != 200 || login["two_factor"] != true || challenge == "" || login["token"] != nil {
		t.Fatalf("login: %d %v", code, login)
	}
	if code, _ := a.api("POST", "/login/two-factor", "", map[string]any{"challenge": challenge, "code": "123456"}); code != http.StatusUnauthorized {
		t.Fatalf("bad code: %d", code)
	}
	code, out := a.api("POST", "/login/two-factor", "", map[string]any{"challenge": challenge, "code": codes[0]})
	if code != 200 || out["token"] != tok {
		t.Fatalf("recovery code step: %d %v", code, out)
	}
	if code, _ := a.api("POST", "/login/two-factor", "", map[string]any{"challenge": challenge, "code": codes[1]}); code != http.StatusUnauthorized {
		t.Fatalf("challenge reused: %d", code)
	}
	if n := len(a.users["ann@example.com"].codes); n != 7 {
		t.Fatalf("recovery code not used up: %d left", n)
	}

	// New codes, then turning it off.
	if code, out := a.api("POST", "/account/two-factor/recovery-codes", tok, map[string]any{"current_password": "secret-pass"}); code != 200 || len(out["recovery_codes"].([]any)) != 8 {
		t.Fatalf("regenerate: %d", code)
	}
	if code, _ := a.api("POST", "/account/two-factor/disable", tok, map[string]any{"current_password": "bad"}); code != http.StatusUnprocessableEntity {
		t.Fatalf("disable without password: %d", code)
	}
	if code, out := a.api("POST", "/account/two-factor/disable", tok, map[string]any{"current_password": "secret-pass"}); code != 200 || out["two_factor_enabled"] != false {
		t.Fatalf("disable: %d", code)
	}
	if code, login := a.api("POST", "/login", "", map[string]any{"username": "ann@example.com", "password": "secret-pass"}); code != 200 || login["token"] != tok {
		t.Fatalf("login after disable: %d %v", code, login)
	}
}
