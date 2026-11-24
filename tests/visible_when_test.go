package tests

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/forms"
)

func visibleWhenForm() *forms.Form {
	return forms.New().Schema(
		forms.Select("driver").Options(map[string]string{"stripe": "Stripe", "payhere": "PayHere"}).Required(),
		forms.Section("Stripe").VisibleWhen("driver", "stripe").Schema(
			forms.TextInput("stripe_key").Required(),
		),
		forms.Section("PayHere").VisibleWhen("driver", "payhere").Schema(
			forms.TextInput("merchant_id").Required(),
		),
		forms.TextInput("note").VisibleWhen("driver", "payhere"),
	)
}

func TestVisibleWhenValidation(t *testing.T) {
	f := visibleWhenForm()
	out, err := forms.Validate(context.Background(), f.GetSchema(),
		map[string]any{"driver": "stripe", "stripe_key": "sk", "merchant_id": "leak", "note": "x"}, true)
	if err != nil {
		t.Fatalf("hidden required field blocked the save: %v", err)
	}
	if _, ok := out["merchant_id"]; ok {
		t.Fatal("hidden section value kept")
	}
	if _, ok := out["note"]; ok {
		t.Fatal("hidden field value kept")
	}
	if out["stripe_key"] != "sk" {
		t.Fatalf("visible value lost: %v", out)
	}
	if _, err := forms.Validate(context.Background(), f.GetSchema(), map[string]any{"driver": "payhere"}, true); err == nil {
		t.Fatal("visible required field not enforced")
	}
}

func TestVisibleWhenDescribeAndRender(t *testing.T) {
	f := visibleWhenForm()
	b, _ := json.Marshal(forms.Describe(context.Background(), f.GetSchema()))
	if !strings.Contains(string(b), `"visible_when":{"field":"driver","values":["stripe"]}`) {
		t.Fatalf("describe: %s", b)
	}
	f.Fill(map[string]any{"driver": "stripe"})
	html := f.Render(context.Background())
	if !strings.Contains(html, `data-visible-when="driver" data-visible-values="stripe">`) {
		t.Fatalf("stripe section should render shown:\n%s", html)
	}
	if !strings.Contains(html, `data-visible-values="payhere" style="display:none"`) {
		t.Fatalf("payhere section should render hidden:\n%s", html)
	}
}
