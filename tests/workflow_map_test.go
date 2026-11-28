package tests

import (
	"fmt"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/golaravel/pkg/auth"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type mapRow struct {
	ID int `json:"id"`
}

// mapResource is a minimal resource for map tests; canCreate and canView
// decide its abilities for the "good" user.
func mapResource(slug, plural, icon string, canCreate, canView bool) *filament.BaseResource[*mapRow] {
	res := filament.NewResource[*mapRow](slug)
	res.ModelName(plural).PluralModelName(plural).NavigationGroup("Books").NavigationIcon(icon)
	res.RecordID(func(r *mapRow) string { return fmt.Sprint(r.ID) })
	res.OnList(func(ctx *filament.Context, _, _, _ string, _, _ int, _ map[string]string) ([]*mapRow, int, error) {
		return nil, 0, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*mapRow, error) { return nil, fmt.Errorf("not found") })
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error { return nil })
	res.CanCreate(func(ctx *filament.Context) bool { return canCreate })
	res.CanViewAny(func(ctx *filament.Context) bool { return canView })
	return res
}

func mapPanel(t *testing.T, build func(p *filament.Panel)) (*golaravel.Engine, error) {
	t.Helper()
	p := filament.NewPanel("admin").Path("/admin").BrandName("Books").
		Engine(filament.EngineFlutter).Flutter(t.TempDir()).
		TokenValidator(func(ctx *filament.Context, token string) (auth.Authenticatable, error) {
			if token == "good" {
				return &filament.PanelUser{ID: "1", Login: "admin", Name: "Admin"}, nil
			}
			return nil, fmt.Errorf("bad token")
		})
	build(p)
	app := golaravel.New()
	return app, p.Register(app)
}

func TestWorkflowMapShowsWhatTheUserMayOpen(t *testing.T) {
	invoices := mapResource("invoices", "Invoices", "document", true, true)
	receipts := mapResource("receipts", "Receipts", "cash", false, true) // may view, not create
	payroll := mapResource("payroll", "Payroll", "users", true, false)   // hidden from this user
	deposits := mapResource("deposits", "Deposits", "bank", true, true)

	invoices.OnMap("home", "customers").Label("Create invoices").Opens(filament.MapCreate).Sort(1).
		LeadsTo("receipts", "statements")
	invoices.OnMap("home", "customers").ID("statements").Label("Statements").Sort(2)
	receipts.OnMap("home", "customers").Label("Receive payments").Opens(filament.MapCreate).Sort(3).Cell(2, 0).
		LeadsTo("deposits")
	payroll.OnMap("home", "employees").Label("Pay employees").Feeds("invoices")
	deposits.OnMap("home", "banking").Label("Record deposits").
		Badge(func(ctx *filament.Context) string { return "2" })

	app, err := mapPanel(t, func(p *filament.Panel) {
		p.Resources(invoices, receipts, payroll, deposits)
		p.Page(filament.FlutterPage("pos", "Till").Icon("cart"))
		p.WorkflowMap(filament.NewWorkflowMap("home").Label("Workflow").OnDashboard().
			Block(filament.MapBlock("customers", "Customers").At(0, 0).Span(3, 1).Icon("users")).
			Block(filament.MapBlock("employees", "Employees").At(0, 1).Span(3, 1)).
			Block(filament.MapBlock("banking", "Banking").At(3, 0).Span(1, 2)).
			Block(filament.MapBlock("help", "Help")).
			Task(filament.NewMapTask("till", "customers", "Sale at the till").Page("pos").Sort(0).LeadsTo("deposits")).
			Task(filament.NewMapTask("guide", "help", "Guide").URL("https://example.com/guide")).
			Link("statements", "receipts"))
	})
	if err != nil {
		t.Fatal(err)
	}

	code, schema := clientGet(t, app, "example.com", "/admin/api/schema", "good")
	maps, _ := schema["workflow_maps"].([]any)
	if code != 200 || len(maps) != 1 || maps[0].(map[string]any)["on_dashboard"] != true {
		t.Fatalf("schema lists the map: %d %v", code, schema["workflow_maps"])
	}

	code, m := clientGet(t, app, "example.com", "/admin/api/maps/home", "good")
	if code != 200 {
		t.Fatalf("map: %d %v", code, m)
	}
	tasks := map[string]map[string]any{}
	var order []string
	for _, x := range m["tasks"].([]any) {
		task := x.(map[string]any)
		tasks[fmt.Sprint(task["id"])] = task
		order = append(order, fmt.Sprint(task["id"]))
	}
	if _, hidden := tasks["payroll"]; hidden {
		t.Fatal("a task whose screen the user cannot open is left out")
	}
	if strings.Join(order, ",") != "till,invoices,statements,receipts,deposits,guide" {
		t.Fatalf("tasks in block then sort order: %v", order)
	}
	open := func(id string) map[string]any { return tasks[id]["open"].(map[string]any) }
	if open("invoices")["url"] != "/admin/invoices/new" || open("invoices")["action"] != "create" {
		t.Fatalf("create task: %v", open("invoices"))
	}
	if open("receipts")["url"] != "/admin/receipts" || open("receipts")["action"] != "list" {
		t.Fatalf("a user who may not create gets the list: %v", open("receipts"))
	}
	if open("statements")["url"] != "/admin/invoices" || tasks["statements"]["icon"] != "document" {
		t.Fatalf("second task of a resource: %v", tasks["statements"])
	}
	if open("till")["kind"] != "page" || open("till")["url"] != "/admin/pos" || tasks["till"]["icon"] != "cart" {
		t.Fatalf("page task: %v", tasks["till"])
	}
	if open("guide")["url"] != "https://example.com/guide" {
		t.Fatalf("link task: %v", tasks["guide"])
	}
	if tasks["deposits"]["badge"] != "2" || tasks["receipts"]["col"] != 2.0 || tasks["receipts"]["row"] != 0.0 {
		t.Fatalf("badge and cell: %v %v", tasks["deposits"], tasks["receipts"])
	}

	var links []string
	for _, x := range m["links"].([]any) {
		l := x.(map[string]any)
		links = append(links, fmt.Sprintf("%v>%v:%v", l["from"], l["to"], l["kind"]))
	}
	want := "till>deposits:leads,invoices>receipts:leads,invoices>statements:leads,receipts>deposits:leads,statements>receipts:leads"
	if strings.Join(links, ",") != want {
		t.Fatalf("links (none to the hidden payroll task):\n got %v\nwant %v", strings.Join(links, ","), want)
	}

	var blocks []string
	for _, x := range m["blocks"].([]any) {
		b := x.(map[string]any)
		blocks = append(blocks, fmt.Sprintf("%v@%v,%v", b["id"], b["col"], b["row"]))
	}
	if strings.Join(blocks, " ") != "customers@0,0 banking@3,0 help@0,2" {
		t.Fatalf("blocks (employees has nothing left; help goes under the placed ones): %v", blocks)
	}

	if code, _ := clientGet(t, app, "example.com", "/admin/api/maps/nope", "good"); code != 404 {
		t.Fatalf("unknown map: %d", code)
	}
	if code, _ := clientGet(t, app, "example.com", "/admin/api/maps/home", ""); code == 200 {
		t.Fatal("the map needs a signed-in user")
	}
}

func TestWorkflowMapHiddenWhenNothingToOpen(t *testing.T) {
	payroll := mapResource("payroll", "Payroll", "users", true, false)
	payroll.OnMap("home", "employees")
	app, err := mapPanel(t, func(p *filament.Panel) {
		p.Resources(payroll)
		p.WorkflowMap(filament.NewWorkflowMap("home").Block(filament.MapBlock("employees", "Employees")))
	})
	if err != nil {
		t.Fatal(err)
	}
	_, schema := clientGet(t, app, "example.com", "/admin/api/schema", "good")
	if maps, _ := schema["workflow_maps"].([]any); len(maps) != 0 {
		t.Fatalf("a map with nothing to open is not listed: %v", maps)
	}
	if code, _ := clientGet(t, app, "example.com", "/admin/api/maps/home", "good"); code != 404 {
		t.Fatalf("empty map: %d", code)
	}
}

func TestWorkflowMapDeclarationMistakesStopStartUp(t *testing.T) {
	cases := map[string]func(p *filament.Panel){
		"unknown block": func(p *filament.Panel) {
			r := mapResource("a", "A", "", true, true)
			r.OnMap("home", "nowhere")
			p.Resources(r).WorkflowMap(filament.NewWorkflowMap("home").Block(filament.MapBlock("b", "B")))
		},
		"goes to unknown task": func(p *filament.Panel) {
			r := mapResource("a", "A", "", true, true)
			r.OnMap("home", "b").LeadsTo("ghost")
			p.Resources(r).WorkflowMap(filament.NewWorkflowMap("home").Block(filament.MapBlock("b", "B")))
		},
		"declared twice": func(p *filament.Panel) {
			r := mapResource("a", "A", "", true, true)
			r.OnMap("home", "b")
			r.OnMap("home", "b")
			p.Resources(r).WorkflowMap(filament.NewWorkflowMap("home").Block(filament.MapBlock("b", "B")))
		},
		"unknown map": func(p *filament.Panel) {
			r := mapResource("a", "A", "", true, true)
			r.OnMap("other", "b")
			p.Resources(r).WorkflowMap(filament.NewWorkflowMap("home").Block(filament.MapBlock("b", "B")))
		},
		"unknown page": func(p *filament.Panel) {
			p.WorkflowMap(filament.NewWorkflowMap("home").Block(filament.MapBlock("b", "B")).
				Task(filament.NewMapTask("t", "b", "T").Page("missing")))
		},
		"opens nothing": func(p *filament.Panel) {
			p.WorkflowMap(filament.NewWorkflowMap("home").Block(filament.MapBlock("b", "B")).
				Task(filament.NewMapTask("t", "b", "T")))
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := mapPanel(t, build); err == nil || !strings.Contains(err.Error(), "workflow maps") {
				t.Fatalf("want a start-up error, got %v", err)
			}
		})
	}
}
