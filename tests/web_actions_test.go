package tests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func newWebActionApp(t *testing.T) (*golaravel.Engine, map[string]*actDoc) {
	t.Helper()
	docs := map[string]*actDoc{
		"1": {ID: 1, Number: "PUR-1", Total: 100},
		"2": {ID: 2, Number: "PUR-2", Posted: true, Total: 50},
		"3": {ID: 3, Number: "PUR-3", Total: 0},
	}
	res := filament.NewResource[*actDoc]("docs")
	res.ModelName("Doc").PluralModelName("Docs")
	res.RecordID(func(d *actDoc) string { return fmt.Sprint(d.ID) })
	res.Table(func(tb *tables.Table[*actDoc]) *tables.Table[*actDoc] {
		return tb.
			Columns(tables.TextColumn[*actDoc]("number")).
			Actions(
				tables.Action[*actDoc]("post").Label("Post").RequiresConfirmation().
					ModalDescription("Post this document?").
					Visible(func(d *actDoc) bool { return !d.Posted }).
					Handle(func(ctx context.Context, d *actDoc, data map[string]any) error {
						if d.Total <= 0 {
							return errors.New("nothing to post")
						}
						d.Posted = true
						return nil
					}),
				tables.Action[*actDoc]("discount").Label("Discount").
					Form(func(f *forms.Form) *forms.Form {
						return f.Schema(forms.TextInput("percent").Numeric().Required())
					}).
					Handle(func(ctx context.Context, d *actDoc, data map[string]any) error {
						var pct float64
						fmt.Sscan(fmt.Sprint(data["percent"]), &pct)
						d.Total = d.Total * (100 - pct) / 100
						return nil
					}),
			)
	})
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*actDoc, int, error) {
		return []*actDoc{docs["1"], docs["2"], docs["3"]}, 3, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*actDoc, error) {
		if d, ok := docs[id]; ok {
			return d, nil
		}
		return nil, fmt.Errorf("record not found")
	})
	app := golaravel.New()
	if err := filament.NewPanel("admin").Path("/admin").Resource(res).Register(app); err != nil {
		t.Fatal(err)
	}
	return app, docs
}

func TestWebRowActionLinkConfirmAndRun(t *testing.T) {
	app, docs := newWebActionApp(t)
	w := newWebClient(t, app)

	list := w.do("GET", "/admin/docs", nil).Body.String()
	if !strings.Contains(list, `href="/admin/docs/1/actions/post"`) {
		t.Fatalf("list should link the runnable action to its confirm page:\n%s", list)
	}

	page := w.do("GET", "/admin/docs/1/actions/post", nil)
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Post this document?") ||
		!strings.Contains(page.Body.String(), `action="/admin/docs/1/actions/post"`) {
		t.Fatalf("confirm page: %d\n%s", page.Code, page.Body.String())
	}
	token := w.csrfFrom(page.Body.String())

	if rec := w.do("POST", "/admin/docs/1/actions/post", url.Values{}); rec.Code != http.StatusForbidden || docs["1"].Posted {
		t.Fatalf("POST without CSRF token must be refused: %d posted=%v", rec.Code, docs["1"].Posted)
	}
	rec := w.do("POST", "/admin/docs/1/actions/post", url.Values{"_token": {token}})
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/admin/docs" || !docs["1"].Posted {
		t.Fatalf("run action: %d %s posted=%v", rec.Code, rec.Header().Get("Location"), docs["1"].Posted)
	}

	// Hidden for this record now.
	if rec := w.do("GET", "/admin/docs/1/actions/post", nil); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("hidden action page: %d", rec.Code)
	}
	if rec := w.do("POST", "/admin/docs/2/actions/post", url.Values{"_token": {token}}); rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("hidden action run: %d", rec.Code)
	}
	// Handler errors re-render the page with the message.
	if rec := w.do("POST", "/admin/docs/3/actions/post", url.Values{"_token": {token}}); rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "nothing to post") {
		t.Fatalf("handler error: %d", rec.Code)
	}
	if rec := w.do("GET", "/admin/docs/9/actions/post", nil); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown record: %d", rec.Code)
	}
}

func TestWebRowActionWithForm(t *testing.T) {
	app, docs := newWebActionApp(t)
	w := newWebClient(t, app)
	page := w.do("GET", "/admin/docs/1/actions/discount", nil)
	if !strings.Contains(page.Body.String(), `name="percent"`) {
		t.Fatalf("action form fields missing:\n%s", page.Body.String())
	}
	token := w.csrfFrom(page.Body.String())
	if rec := w.do("POST", "/admin/docs/1/actions/discount", url.Values{"_token": {token}, "percent": {"10"}}); rec.Code != http.StatusSeeOther {
		t.Fatalf("discount: %d %s", rec.Code, rec.Body.String())
	}
	if docs["1"].Total != 90 {
		t.Fatalf("discount not applied: %v", docs["1"].Total)
	}
}
