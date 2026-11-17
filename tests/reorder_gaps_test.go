package tests

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func newReorderApp(t *testing.T, withHandler bool) (*golaravel.Engine, *[]string, *int) {
	t.Helper()
	var got []string
	offset := -1
	res := filament.NewResource[*jobRow]("slides").PluralModelName("Slides")
	res.Table(func(tb *tables.Table[*jobRow]) *tables.Table[*jobRow] {
		return tb.Columns(tables.TextColumn[*jobRow]("name")).Reorderable("position")
	})
	res.RecordID(func(x *jobRow) string { return fmt.Sprint(x.ID) })
	res.OnList(func(ctx *filament.Context, s, so, d string, p, pp int, f map[string]string) ([]*jobRow, int, error) {
		return []*jobRow{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}, 40, nil
	})
	if withHandler {
		res.OnReorder(func(ctx *filament.Context, ids []string) error {
			got, offset = ids, filament.ReorderOffset(ctx)
			return nil
		})
	}
	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: "1", Login: "a"}, nil
		}).Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, &got, &offset
}

func TestReorderOnlyOnThePlainList(t *testing.T) {
	app, _, _ := newReorderApp(t, true)
	page := bulkReq(app, "GET", "/admin/slides?page=3&perPage=10", "", "").Body.String()
	if !strings.Contains(page, `data-reorder-offset="20"`) || !strings.Contains(page, "table-row-draggable") {
		t.Fatal("plain list is not reorderable with its offset")
	}
	for _, q := range []string{"search=a", "sort=name", "filter_status=x"} {
		if strings.Contains(bulkReq(app, "GET", "/admin/slides?"+q, "", "").Body.String(), "table-row-draggable") {
			t.Fatalf("reorderable with %s", q)
		}
	}
	if !strings.Contains(bulkReq(app, "GET", "/admin/slides?sort=position", "", "").Body.String(), "table-row-draggable") {
		t.Fatal("not reorderable when sorted by position")
	}
}

func TestReorderOffsetAndMissingHandler(t *testing.T) {
	app, got, offset := newReorderApp(t, true)
	rec := bulkReq(app, "POST", "/admin/api/resources/slides/reorder", "application/json", `{"order":["2","1"],"offset":20}`)
	if rec.Code != 200 || strings.Join(*got, ",") != "2,1" || *offset != 20 {
		t.Fatalf("reorder: %d %v %d", rec.Code, *got, *offset)
	}
	none, _, _ := newReorderApp(t, false)
	if rec := bulkReq(none, "POST", "/admin/slides/reorder", "application/json", `{"order":["1"]}`); rec.Code != http.StatusNotFound {
		t.Fatalf("no handler: %d", rec.Code)
	}
}
