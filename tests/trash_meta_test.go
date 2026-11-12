package tests

import (
	"fmt"
	"testing"

	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
)

func TestRowMetaReportsTrashedRecords(t *testing.T) {
	items := []*guardItem{{ID: 1, Name: "Live"}, {ID: 2, Name: "Binned", Locked: true}, {ID: 3, Name: "Kept", Locked: true}}
	res := filament.NewResource[*guardItem]("items")
	res.RecordID(func(i *guardItem) string { return fmt.Sprint(i.ID) })
	res.Table(func(tb *tables.Table[*guardItem]) *tables.Table[*guardItem] {
		return tb.Columns(tables.TextColumn[*guardItem]("name"))
	})
	res.SoftDeletes().IsTrashed(func(i *guardItem) bool { return i.Locked })
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*guardItem, int, error) {
		return items, len(items), nil
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error { return nil })
	res.OnDelete(func(ctx *filament.Context, id string) error { return nil })
	res.OnRestore(func(ctx *filament.Context, id string) error { return nil })
	res.OnForceDelete(func(ctx *filament.Context, id string) error { return nil })
	res.CanForceDelete(func(ctx *filament.Context, id string) bool { return id != "3" })

	p := filament.NewPanel("trash").Path("/admin").Engine(filament.EngineFlutter).Flutter(t.TempDir()).
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: "u1"}, nil
		}).Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	_, list := relDo(t, app, "GET", "/admin/api/resources/items", "good", nil)
	meta := list["row_meta"].(map[string]any)
	live, binned, kept := meta["1"].(map[string]any), meta["2"].(map[string]any), meta["3"].(map[string]any)
	if live["trashed"] != false || live["restore"] != nil || live["update"] != true {
		t.Fatalf("live record: %v", live)
	}
	if binned["trashed"] != true || binned["restore"] != true || binned["force_delete"] != true || binned["update"] != false || binned["delete"] != false {
		t.Fatalf("trashed record: %v", binned)
	}
	if kept["force_delete"] != false {
		t.Fatalf("CanForceDelete applies per record: %v", kept)
	}
}
