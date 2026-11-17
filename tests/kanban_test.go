package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func newBoardApp(t *testing.T) (*golaravel.Engine, map[string]*bulkTask) {
	t.Helper()
	db := map[string]*bulkTask{"1": {1, "Write docs", "open"}, "2": {2, "Ship", "done"}, "3": {3, "Locked", "open"}}
	res := filament.NewResource[*bulkTask]("tasks").PluralModelName("Tasks")
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(forms.TextInput("title"), forms.Select("status").Options(map[string]string{"open": "Open", "done": "Done"}))
	})
	res.Table(func(tb *tables.Table[*bulkTask]) *tables.Table[*bulkTask] {
		return tb.Columns(tables.TextColumn[*bulkTask]("title"))
	})
	res.RecordID(func(x *bulkTask) string { return fmt.Sprint(x.ID) })
	res.OnList(func(ctx *filament.Context, s, so, d string, p, pp int, f map[string]string) ([]*bulkTask, int, error) {
		var out []*bulkTask
		for _, id := range []string{"1", "2", "3"} {
			if f["status"] == "" || db[id].Status == f["status"] {
				out = append(out, db[id])
			}
		}
		return out, len(out), nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*bulkTask, error) {
		if x, ok := db[id]; ok {
			cp := *x
			return &cp, nil
		}
		return nil, fmt.Errorf("not found")
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		db[id].Status = fmt.Sprint(data["status"])
		return nil
	})
	res.CanEdit(func(ctx *filament.Context, x *bulkTask) bool { return x.Title != "Locked" })
	res.Kanban(filament.KanbanConfig[*bulkTask]{
		Field:   "status",
		Columns: []filament.KanbanColumn{{Value: "open", Label: "Open"}, {Value: "done", Label: "Done"}},
		Title:   func(x *bulkTask) string { return x.Title },
	})
	p := filament.NewPanel("admin").Path("/admin").
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			return &filament.PanelUser{ID: "1", Login: "a"}, nil
		}).Resource(res)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}
	return app, db
}

func TestKanbanBoard(t *testing.T) {
	app, db := newBoardApp(t)
	var board struct {
		Columns []filament.KanbanColumnData `json:"columns"`
	}
	_ = json.Unmarshal(bulkReq(app, "GET", "/admin/api/resources/tasks/board", "", "").Body.Bytes(), &board)
	if len(board.Columns) != 2 || board.Columns[0].Total != 2 || board.Columns[1].Cards[0].Title != "Ship" {
		t.Fatalf("board: %+v", board)
	}
	move := func(path, id, value string) int {
		return bulkReq(app, "POST", path, "application/json", fmt.Sprintf(`{"id":%q,"value":%q}`, id, value)).Code
	}
	if c := move("/admin/api/resources/tasks/board/move", "1", "done"); c != 200 || db["1"].Status != "done" {
		t.Fatalf("move: %d", c)
	}
	if c := move("/admin/api/resources/tasks/board/move", "1", "nowhere"); c != http.StatusUnprocessableEntity {
		t.Fatalf("unknown column: %d", c)
	}
	if c := move("/admin/tasks/board/move", "3", "done"); c != http.StatusForbidden || db["3"].Status != "open" {
		t.Fatalf("locked card moved: %d", c)
	}
	page := bulkReq(app, "GET", "/admin/tasks/board", "", "").Body.String()
	if !strings.Contains(page, `data-column="done"`) || !strings.Contains(page, "Write docs") {
		t.Fatalf("board page:\n%s", page)
	}
	if !strings.Contains(bulkReq(app, "GET", "/admin/tasks", "", "").Body.String(), "data-board-link") {
		t.Fatal("no Board link on the list")
	}
}
