package payments_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/payments"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
	"github.com/golaravel/golaravel/pkg/payment"
	"github.com/golaravel/golaravel/pkg/payment/gateways"
	"github.com/golaravel/golaravel/pkg/session"
)

func newApp(t *testing.T) (http.Handler, *payment.Manager) {
	t.Helper()
	reg := payment.NewRegistry()
	gateways.Register(reg)
	m := payment.NewManager(payment.Options{BaseURL: "https://shop.test", Registry: reg,
		Encrypter: session.NewEncrypter("k")})
	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: "1", Login: "a"}, nil
		})
	payments.Register(p, m)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, m
}

func call(t *testing.T, app http.Handler, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, "http://example.com/admin/api/resources/"+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestGatewayForm(t *testing.T) {
	app, _ := newApp(t)
	code, out := call(t, app, "GET", "payment-gateways/form", nil)
	if code != 200 {
		t.Fatalf("form: %d %v", code, out)
	}
	raw, _ := json.Marshal(out)
	s := string(raw)
	for _, want := range []string{`"name":"stripe__secret_key"`, `"name":"tamara__notification_token"`,
		`"visible_when":{"field":"driver","values":["payhere"]}`, `"type":"password"`} {
		if !strings.Contains(s, want) {
			t.Errorf("form lacks %s", want)
		}
	}
}

func TestGatewayCRUD(t *testing.T) {
	app, m := newApp(t)
	ctx := context.Background()
	// other drivers' fields in the submission are dropped, not saved
	code, out := call(t, app, "POST", "payment-gateways", map[string]any{
		"driver": "stripe", "id": "stripe", "name": "Card", "mode": "test", "enabled": true,
		"stripe__secret_key": "sk_test_1", "stripe__webhook_secret": "whsec_1", "stripe__capture": "automatic",
		"tabby__secret_key": "leak",
	})
	if code >= 300 {
		t.Fatalf("create: %d %v", code, out)
	}
	raw, _ := m.Store().Gateway(ctx, "stripe")
	if !strings.HasPrefix(raw.Settings["secret_key"], "enc:") || len(raw.Settings) > 5 {
		t.Fatalf("stored settings %v", raw.Settings)
	}

	// secrets never come back
	code, out = call(t, app, "GET", "payment-gateways/stripe", nil)
	data, _ := json.Marshal(out)
	if code != 200 || strings.Contains(string(data), "sk_test_1") || strings.Contains(string(data), "whsec_1") {
		t.Fatalf("find leaked a secret: %d %s", code, data)
	}
	if !strings.Contains(string(data), `"webhook_url":"https://shop.test/payments/webhooks/stripe"`) {
		t.Fatalf("no webhook url: %s", data)
	}

	// editing with blank secrets keeps them
	code, out = call(t, app, "PUT", "payment-gateways/stripe", map[string]any{"name": "Cards", "stripe__secret_key": ""})
	if code >= 300 {
		t.Fatalf("update: %d %v", code, out)
	}
	g, _ := m.GatewayConfig(ctx, "stripe")
	if g.Name != "Cards" || g.Settings["secret_key"] != payment.MaskedValue {
		t.Fatalf("after update %+v", g)
	}

	// a live gateway with a test key is refused, against a field
	code, out = call(t, app, "POST", "payment-gateways", map[string]any{
		"driver": "stripe", "id": "live", "mode": "live",
		"stripe__secret_key": "sk_test_1", "stripe__webhook_secret": "w"})
	if code != 422 {
		t.Fatalf("bad live key: %d %v", code, out)
	}
	// the key cannot change
	if code, _ = call(t, app, "PUT", "payment-gateways/stripe", map[string]any{"id": "other"}); code != 422 {
		t.Fatalf("renamed key: %d", code)
	}
	if code, out = call(t, app, "POST", "payment-gateways", map[string]any{"driver": "stripe", "id": "stripe",
		"stripe__secret_key": "sk_test_2", "stripe__webhook_secret": "w"}); code != 422 {
		t.Fatalf("duplicate key: %d %v", code, out)
	}
	if code, _ = call(t, app, "DELETE", "payment-gateways/stripe", nil); code >= 300 {
		t.Fatalf("delete: %d", code)
	}
}

func TestPaymentsListViewAndRefund(t *testing.T) {
	app, m := newApp(t)
	ctx := context.Background()
	if err := m.SaveGateway(ctx, &payment.GatewayConfig{ID: "test", Driver: "test", Name: "Test card", Enabled: true,
		Settings: map[string]string{"signing_secret": "s"}}); err != nil {
		t.Fatal(err)
	}
	p, err := m.Checkout(ctx, "test", payment.CheckoutRequest{Reference: "ORD-9", Amount: 4200, Currency: "LKR",
		Customer: payment.Customer{FirstName: "Kamal", Email: "kamal@example.com"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.HandleWebhook(ctx, "test", httptest.NewRequest("POST", "/", nil),
		payment.TestWebhookBody("s", p.GatewayRef, "pay")); err != nil {
		t.Fatal(err)
	}

	code, out := call(t, app, "GET", "payments?filter[status]=paid", nil)
	raw, _ := json.Marshal(out)
	if code != 200 || !strings.Contains(string(raw), `"money":"LKR 42.00"`) || !strings.Contains(string(raw), `"gateway_name":"Test card"`) {
		t.Fatalf("list: %d %s", code, raw)
	}
	if code, _ := call(t, app, "POST", "payments", map[string]any{"reference": "x"}); code < 400 {
		t.Fatalf("payments can be created from the panel: %d", code)
	}

	code, out = call(t, app, "POST", "payments/"+p.ID+"/actions/refund", map[string]any{"amount": "10.50", "reason": "late"})
	if code != 200 {
		t.Fatalf("refund: %d %v", code, out)
	}
	np, _ := m.Payment(ctx, p.ID)
	if np.Status != payment.StatusPartiallyRefunded || np.Refunded != 1050 {
		t.Fatalf("after refund %s %d", np.Status, np.Refunded)
	}
	if code, _ = call(t, app, "POST", "payments/"+p.ID+"/actions/refund", map[string]any{"amount": "99"}); code == 200 {
		t.Fatal("over-refund accepted")
	}
	if code, _ = call(t, app, "POST", "payments/"+p.ID+"/actions/capture", nil); code == 200 {
		t.Fatal("captured a paid payment")
	}

	code, out = call(t, app, "GET", "payments/"+p.ID+"/relations/history", nil)
	raw, _ = json.Marshal(out)
	if code != 200 || !strings.Contains(string(raw), "Paid → Partially refunded") || !strings.Contains(string(raw), "Pending → Paid") {
		t.Fatalf("history: %d %s", code, raw)
	}
}

func TestSubscriptionsAndMethods(t *testing.T) {
	app, m := newApp(t)
	ctx := context.Background()
	if err := m.SaveGateway(ctx, &payment.GatewayConfig{ID: "test", Driver: "test", Name: "Test card", Enabled: true,
		Settings: map[string]string{"signing_secret": "s"}}); err != nil {
		t.Fatal(err)
	}
	pm, _ := m.SavePaymentMethod(ctx, "test", payment.PreapprovalRequest{CustomerRef: "c-9", Currency: "LKR",
		Customer: payment.Customer{FirstName: "Sunil", Email: "sunil@example.com"}})
	if _, err := m.HandleWebhook(ctx, "test", httptest.NewRequest("POST", "/", nil),
		payment.TestWebhookBody("s", "test_"+pm.ID, "pay")); err != nil {
		t.Fatal(err)
	}
	// .13 is declined by the test gateway: the subscription starts past due
	sub, err := m.Subscribe(ctx, "", payment.SubscriptionRequest{Reference: "PLAN-GOLD", Amount: 99913,
		Interval: payment.Month, Cycles: 6, PaymentMethodID: pm.ID})
	if err != nil {
		t.Fatal(err)
	}

	code, out := call(t, app, "GET", "payment-methods", nil)
	raw, _ := json.Marshal(out)
	if code != 200 || !strings.Contains(string(raw), "Visa •••• 4242") || strings.Contains(string(raw), "tok_") {
		t.Fatalf("methods list: %d %s", code, raw)
	}
	code, out = call(t, app, "GET", "subscriptions?filter[status]=past_due", nil)
	raw, _ = json.Marshal(out)
	if code != 200 || !strings.Contains(string(raw), `"schedule":"every month, 6 times"`) || !strings.Contains(string(raw), `"progress":"0 of 6 paid"`) {
		t.Fatalf("subscriptions list: %d %s", code, raw)
	}
	code, out = call(t, app, "GET", "subscriptions/"+sub.ID+"/relations/payments", nil)
	raw, _ = json.Marshal(out)
	if code != 200 || !strings.Contains(string(raw), "PLAN-GOLD-1") || !strings.Contains(string(raw), `"status":"failed"`) {
		t.Fatalf("period payments: %d %s", code, raw)
	}
	code, out = call(t, app, "GET", "subscriptions/"+sub.ID+"/relations/history", nil)
	raw, _ = json.Marshal(out)
	if code != 200 || !strings.Contains(string(raw), "Active → Past due") {
		t.Fatalf("subscription history: %d %s", code, raw)
	}

	// still declined: the retry action reports the subscription as it is
	if code, out = call(t, app, "POST", "subscriptions/"+sub.ID+"/actions/retry", nil); code != 200 {
		t.Fatalf("retry: %d %v", code, out)
	}
	if code, out = call(t, app, "POST", "subscriptions/"+sub.ID+"/actions/cancel", nil); code != 200 {
		t.Fatalf("cancel: %d %v", code, out)
	}
	if s, _ := m.Subscription(ctx, sub.ID); s.Status != payment.SubCanceled {
		t.Fatalf("after cancel %s", s.Status)
	}
	if code, _ = call(t, app, "POST", "subscriptions/"+sub.ID+"/actions/cancel", nil); code == 200 {
		t.Fatal("canceled twice")
	}

	if code, out = call(t, app, "POST", "payment-methods/"+pm.ID+"/actions/revoke", nil); code != 200 {
		t.Fatalf("revoke: %d %v", code, out)
	}
	if got, _ := m.PaymentMethod(ctx, pm.ID); got.Status != payment.MethodRevoked {
		t.Fatalf("after revoke %s", got.Status)
	}
	if code, _ = call(t, app, "POST", "subscriptions", map[string]any{"reference": "x"}); code < 400 {
		t.Fatalf("subscriptions can be created from the panel: %d", code)
	}
}
