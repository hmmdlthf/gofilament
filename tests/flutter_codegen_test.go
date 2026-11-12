package tests

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golaravel/golaravel/pkg/golaravel"

	"github.com/golaravel/gofilament/pkg/commands"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/flutterclient"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

var updateGolden = flag.Bool("update", false, "rewrite golden files")

type cgAddress struct {
	City    string `json:"city"`
	Country string `json:"country,omitempty"`
}

type cgAudit struct {
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type cgLine struct {
	ID    int     `json:"id"`
	SKU   string  `json:"sku"`
	Qty   int     `json:"qty"`
	Price float64 `json:"price"`
}

type cgOrder struct {
	cgAudit
	ID       int64          `json:"id"`
	Number   string         `json:"number"`
	Paid     bool           `json:"paid"`
	Total    float64        `json:"total"`
	Note     *string        `json:"note,omitempty"`
	Tags     []string       `json:"tags"`
	Lines    []cgLine       `json:"lines,omitempty"`
	Ship     cgAddress      `json:"ship"`
	Meta     map[string]any `json:"meta,omitempty"`
	Extra    any            `json:"extra"`
	Secret   string         `json:"-"`
	internal int
}

type cgSummaryQuery struct {
	From  string `json:"from"`
	Limit int    `json:"limit"`
}

type cgSummary struct {
	Till   string   `json:"till"`
	Count  int      `json:"count"`
	Total  float64  `json:"total"`
	Recent []cgLine `json:"recent"`
}

type cgCloseTill struct {
	Counted float64 `json:"counted"`
	Note    string  `json:"note,omitempty"`
}

// newCodegenPanel is the action test panel plus a resource with a rich form
// and a relation, two Flutter pages and two typed endpoints.
func newCodegenPanel(t *testing.T) *filament.Panel {
	t.Helper()
	p, _, _ := newActionPanel(t)

	note := "first"
	orders := []*cgOrder{{ID: 1, Number: "SO-1", Note: &note, Tags: []string{"web"}, Ship: cgAddress{City: "Kandy"}}}

	lines := filament.NewRelationManager[*cgLine]("lines", "Order lines")
	lines.Table(func(tb *tables.Table[*cgLine]) *tables.Table[*cgLine] {
		return tb.Columns(tables.TextColumn[*cgLine]("sku"), tables.TextColumn[*cgLine]("qty"))
	})
	lines.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(forms.TextInput("sku").Required(), forms.TextInput("qty").Numeric())
	})
	lines.OnList(func(ctx *filament.Context, parentID, search, sort, dir string, page, perPage int) ([]*cgLine, int, error) {
		return nil, 0, nil
	})
	lines.OnCreate(func(ctx *filament.Context, parentID string, data map[string]any) error { return nil })

	res := filament.NewResource[*cgOrder]("sales-orders")
	res.ModelName("Sales Order").PluralModelName("Sales Orders").NavigationGroup("Sales")
	res.RecordID(func(o *cgOrder) string { return fmt.Sprint(o.ID) })
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Section("Order").Schema(
				forms.TextInput("number").Required(),
				forms.TextInput("total").Numeric(),
				forms.Toggle("paid"),
				forms.Select("status").OptionList(
					forms.Option{Value: "draft", Label: "Draft"},
					forms.Option{Value: "sent", Label: "Sent"},
				),
				forms.Select("customer_id").OptionsUsing(func(ctx context.Context) []forms.Option {
					return []forms.Option{{Value: "c1", Label: "A customer from the database"}}
				}),
				forms.DatePicker("due_on"),
				forms.DateTimePicker("shipped_at"),
				forms.TagsInput("tags"),
				forms.KeyValue("meta"),
				forms.Custom("signature", "signature_pad").Label("Customer signature").
					Props(map[string]any{"pen_color": "#112233", "height": 160}).
					Rule(func(ctx context.Context, v any) string {
						if s, _ := v.(string); !strings.HasPrefix(s, "data:image/png") {
							return "Sign in the box."
						}
						return ""
					}),
			),
			forms.Repeater("lines").Schema(
				forms.TextInput("sku").Required(),
				forms.TextInput("qty").Numeric(),
			),
		)
	})
	res.Table(func(tb *tables.Table[*cgOrder]) *tables.Table[*cgOrder] {
		return tb.Columns(
			tables.TextColumn[*cgOrder]("number"),
			tables.TextColumn[*cgOrder]("total").Money("LKR"),
			tables.CustomColumn[*cgOrder]("paid", "paid_badge").Label("Paid").Props(struct {
				Tone string `json:"tone"`
			}{Tone: "success"}),
		).Actions(
			tables.Action[*cgOrder]("mark_paid").Label("Mark paid").
				Handle(func(ctx context.Context, o *cgOrder, data map[string]any) error { return nil }),
			tables.Action[*cgOrder]("refund").Label("Refund").
				Form(func(f *forms.Form) *forms.Form {
					return f.Schema(forms.TextInput("amount").Numeric().Required(), forms.Textarea("reason"),
						forms.Custom("approval", "signature_pad").Required().Props(map[string]any{"pen_color": "#ff0000", "height": 80, "clearable": true}))
				}).
				Handle(func(ctx context.Context, o *cgOrder, data map[string]any) error { return nil }),
		)
	})
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*cgOrder, int, error) {
		return orders, len(orders), nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*cgOrder, error) {
		if id == "1" {
			return orders[0], nil
		}
		return nil, errors.New("record not found")
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		t.Error("the generator must not create records")
		return nil
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error { return nil })
	res.OnDelete(func(ctx *filament.Context, id string) error { return nil })
	res.Relations(lines)

	p.Resource(res).
		Pages(
			filament.FlutterPage("pos", "Point of sale").Icon("heroicon-o-shopping-cart").Fullscreen(),
			filament.FlutterPage("stock-take", "Stock take").Group("Sales").
				Visible(func(ctx *filament.Context) bool { return ctx.Query("no_stock") == "" }),
		).
		Widget(filament.FlutterWidget("sales", "sales_total").Heading("Sales today").
			Props(map[string]any{"currency": "LKR"}).
			Data(func(ctx *filament.Context) (any, error) { return map[string]any{"total": 1250.5}, nil }).
			Visible(func(ctx *filament.Context) bool { return ctx.Query("no_sales") == "" })).
		Endpoints(
			filament.Endpoint("salesSummary", "GET", "/pos/tills/:till/summary",
				func(ctx *filament.Context, q cgSummaryQuery) (cgSummary, error) {
					return cgSummary{Till: ctx.Param("till"), Count: q.Limit, Total: 12.5}, nil
				}).Describe("Totals of one till since [cgSummaryQuery.from]."),
			filament.Endpoint("closeTill", "POST", "/pos/tills/:till/close",
				func(ctx *filament.Context, in cgCloseTill) (struct{}, error) {
					if in.Counted < 0 {
						return struct{}{}, errors.New("counted must be positive")
					}
					return struct{}{}, nil
				}).Authorize(func(ctx *filament.Context) bool { return ctx.Query("deny") == "" }),
		)
	return p
}

func TestFlutterCodegenGolden(t *testing.T) {
	files, err := flutterclient.RenderFromPanel(newCodegenPanel(t), flutterclient.GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	again, err := flutterclient.RenderFromPanel(newCodegenPanel(t), flutterclient.GenerateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 5 {
		t.Fatalf("expected 5 files, got %d", len(files))
	}
	for name, content := range files {
		if string(again[name]) != string(content) {
			t.Fatalf("%s: output is not deterministic", name)
		}
		if !strings.HasPrefix(string(content), "// GENERATED CODE - DO NOT EDIT.") {
			t.Fatalf("%s: missing DO NOT EDIT header", name)
		}
		golden := filepath.Join("testdata", "flutter_codegen", filepath.Base(name))
		if *updateGolden {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, content, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatalf("%v (run go test ./tests -run TestFlutterCodegenGolden -update)", err)
		}
		if string(want) != string(content) {
			t.Errorf("%s differs from %s; run go test ./tests -run TestFlutterCodegenGolden -update and review the diff", name, golden)
		}
	}

	schema := string(files["lib/generated/schema.g.dart"])
	for _, leak := range []string{"A customer from the database", "'abilities'", "\n  'widgets'", "\n  'currency'", "\n  'tenants'"} {
		if strings.Contains(schema, leak) {
			t.Errorf("schema snapshot must not contain %s", leak)
		}
	}
}

func TestFlutterCodegenWritesAndChecks(t *testing.T) {
	dir := t.TempDir()
	// a page written by hand before the first generation
	pos := filepath.Join(dir, "lib", "pages", "pos_page.dart")
	_ = os.MkdirAll(filepath.Dir(pos), 0o755)
	_ = os.WriteFile(pos, []byte("// hand-written\nclass PosPage {}\n"), 0o644)

	p := newCodegenPanel(t)
	stale, err := flutterclient.CheckFromPanel(p, dir, flutterclient.GenerateOptions{})
	want := []string{
		"lib/components/cells/paid_badge_cell.dart (missing)",
		"lib/components/fields/signature_pad_field.dart (missing)",
		"lib/components/widgets/sales_total_widget.dart (missing)",
		"lib/pages/stock_take_page.dart (missing)",
	}
	if err != nil || len(stale) != 9 || fmt.Sprint(stale[:4]) != fmt.Sprint(want) {
		t.Fatalf("before generating: 5 generated files, 3 components and the stock-take page are missing: %v %v", stale, err)
	}
	if err := flutterclient.GenerateFromPanel(p, dir, flutterclient.GenerateOptions{}); err != nil {
		t.Fatal(err)
	}
	if stale, err := flutterclient.CheckFromPanel(newCodegenPanel(t), dir, flutterclient.GenerateOptions{}); err != nil || len(stale) != 0 {
		t.Fatalf("after generating nothing is stale: %v %v", stale, err)
	}
	if b, _ := os.ReadFile(pos); string(b) != "// hand-written\nclass PosPage {}\n" {
		t.Fatal("the generator touched a hand-written page")
	}
	stub, err := os.ReadFile(filepath.Join(dir, "lib", "pages", "stock_take_page.dart"))
	if err != nil || !strings.Contains(string(stub), "class StockTakePage extends StatelessWidget") {
		t.Fatalf("missing pages get a stub: %v\n%s", err, stub)
	}
	// a stub, once edited, is the user's
	_ = os.WriteFile(filepath.Join(dir, "lib", "pages", "stock_take_page.dart"), []byte("class StockTakePage {} // mine\n"), 0o644)
	if err := flutterclient.GenerateFromPanel(newCodegenPanel(t), dir, flutterclient.GenerateOptions{}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "lib", "pages", "stock_take_page.dart")); !strings.Contains(string(b), "// mine") {
		t.Fatal("an existing page file was overwritten")
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "lib", "generated"))
	if len(entries) != 5 {
		t.Fatalf("expected exactly the 5 generated files, got %d", len(entries))
	}

	if b, err := os.ReadFile(filepath.Join(dir, "lib", "components", "fields", "signature_pad_field.dart")); err != nil ||
		!strings.Contains(string(b), "class SignaturePadField extends StatelessWidget") ||
		!strings.Contains(string(b), "SignaturePadFieldProps get props") {
		t.Fatalf("field component stub: %v\n%s", err, b)
	}
	_ = os.WriteFile(filepath.Join(dir, "lib", "components", "cells", "paid_badge_cell.dart"), []byte("class Other {}\n"), 0o644)
	if err := flutterclient.GenerateFromPanel(newCodegenPanel(t), dir, flutterclient.GenerateOptions{}); err == nil || !strings.Contains(err.Error(), "must define class PaidBadgeCell") {
		t.Fatalf("expected a missing-class error for a component, got %v", err)
	}
	_ = os.Remove(filepath.Join(dir, "lib", "components", "cells", "paid_badge_cell.dart"))

	// a page file without its class is an error, not a broken build
	_ = os.WriteFile(pos, []byte("class Till {}\n"), 0o644)
	if err := flutterclient.GenerateFromPanel(newCodegenPanel(t), dir, flutterclient.GenerateOptions{}); err == nil || !strings.Contains(err.Error(), "must define class PosPage") {
		t.Fatalf("expected a missing-class error, got %v", err)
	}
	if _, err := flutterclient.CheckFromPanel(newCodegenPanel(t), dir, flutterclient.GenerateOptions{}); err == nil {
		t.Fatal("check must fail on a page file without its class")
	}
}

func TestFlutterPageCommands(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "resources", "flutter")
	if err := commands.MakeFilamentFlutterAppCommand().Run([]string{"--dir=" + app, "--name=shop_admin", "--panel-path=/shop"}); err != nil {
		t.Fatal(err)
	}
	main, err := os.ReadFile(filepath.Join(app, "lib", "main.dart"))
	if err != nil || !strings.Contains(string(main), "pages: panelAppPages") || !strings.Contains(string(main), "'/shop'") {
		t.Fatalf("typed main.dart: %v\n%s", err, main)
	}
	if err := commands.MakeFilamentFlutterPageCommand().Run([]string{"stock-take", "--dir=" + app, "--fullscreen"}); err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(app, "lib", "pages", "stock_take_page.dart"))
	if err != nil || !strings.Contains(string(page), "class StockTakePage") || !strings.Contains(string(page), "'Stock take'") || !strings.Contains(string(page), "Scaffold(") {
		t.Fatalf("page stub: %v\n%s", err, page)
	}
	_ = os.WriteFile(filepath.Join(app, "lib", "pages", "stock_take_page.dart"), []byte("class StockTakePage {}\n"), 0o644)
	if err := commands.MakeFilamentFlutterPageCommand().Run([]string{"stock-take", "--dir=" + app}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(app, "lib", "pages", "stock_take_page.dart")); string(b) != "class StockTakePage {}\n" {
		t.Fatal("make:filament-flutter-page must not overwrite a page")
	}
	if err := commands.MakeFilamentFlutterPageCommand().Run(nil); err == nil {
		t.Fatal("a page id is required")
	}
}

func TestFlutterCodegenNeedsVisibleResources(t *testing.T) {
	// A user the panel refuses gets an error, not an empty client.
	p := newCodegenPanel(t)
	p.CanAccess(func(ctx *filament.Context) bool { return false })
	if _, err := flutterclient.RenderFromPanel(p, flutterclient.GenerateOptions{}); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("expected a 403 error, got %v", err)
	}
}

func TestTypedEndpointsAndFlutterPages(t *testing.T) {
	p := newCodegenPanel(t)
	app := golaravel.New()
	if err := p.Register(app); err != nil {
		t.Fatal(err)
	}

	if code, _ := relDo(t, app, "GET", "/admin/api/pos/tills/t1/summary?limit=3", "", nil); code != 401 {
		t.Fatalf("endpoints need auth, got %d", code)
	}
	code, out := relDo(t, app, "GET", "/admin/api/pos/tills/t1/summary?limit=3&from=2026-01-01", "good", nil)
	if code != 200 || out["till"] != "t1" || out["count"] != float64(3) || out["total"] != 12.5 {
		t.Fatalf("summary: %d %v", code, out)
	}
	if code, _ := relDo(t, app, "GET", "/admin/api/pos/tills/t1/summary?limit=many", "good", nil); code != 400 {
		t.Fatalf("a bad query value is a 400, got %d", code)
	}
	if code, out := relDo(t, app, "POST", "/admin/api/pos/tills/t1/close", "good", map[string]any{"counted": 10}); code != 200 || out["success"] != true {
		t.Fatalf("close: %d %v", code, out)
	}
	if code, _ := relDo(t, app, "POST", "/admin/api/pos/tills/t1/close?deny=1", "good", map[string]any{"counted": 10}); code != 403 {
		t.Fatalf("Authorize should refuse with 403, got %d", code)
	}
	if code, out := relDo(t, app, "POST", "/admin/api/pos/tills/t1/close", "good", map[string]any{"counted": -1}); code == 200 || out["error"] != "counted must be positive" {
		t.Fatalf("handler errors are returned: %d %v", code, out)
	}

	_, schema := relDo(t, app, "GET", "/admin/api/schema", "good", nil)
	if got := fmt.Sprint(schema["client_pages"]); !strings.Contains(got, "pos") || !strings.Contains(got, "stock-take") {
		t.Fatalf("client_pages: %v", got)
	}
	_, schema = relDo(t, app, "GET", "/admin/api/schema?no_stock=1", "good", nil)
	if got := fmt.Sprint(schema["client_pages"]); strings.Contains(got, "stock-take") {
		t.Fatalf("a hidden Flutter page must not be listed: %v", got)
	}

	// custom components: DTOs, server rules and per-user widget data
	_, schema = relDo(t, app, "GET", "/admin/api/schema", "good", nil)
	if got := fmt.Sprint(schema["widgets"]); !strings.Contains(got, "sales_total") || !strings.Contains(got, "1250.5") {
		t.Fatalf("custom widget DTO: %v", got)
	}
	_, w := relDo(t, app, "GET", "/admin/api/widgets?no_sales=1", "good", nil)
	if strings.Contains(fmt.Sprint(w["widgets"]), "sales_total") {
		t.Fatalf("a hidden custom widget must not be sent: %v", w)
	}
	_, form := relDo(t, app, "GET", "/admin/api/resources/sales-orders/form", "good", nil)
	if got := fmt.Sprint(form["schema"]); !strings.Contains(got, "component:signature_pad") || !strings.Contains(got, "pen_color") {
		t.Fatalf("custom field node: %v", got)
	}
	if code, out := relDo(t, app, "POST", "/admin/api/resources/sales-orders", "good", map[string]any{"number": "SO-9", "signature": "scribble"}); code != 422 || out["errors"].(map[string]any)["signature"] != "Sign in the box." {
		t.Fatalf("custom field Rule: %d %v", code, out)
	}

	// The generator's in-process identity is not reachable over HTTP.
	if code, _ := relDo(t, app, "GET", "/admin/api/schema", "", nil); code != 401 {
		t.Fatalf("schema without a token: %d", code)
	}
}

// TestFlutterCodegenSampleApp builds a sample app from the generated code and
// runs `flutter analyze` on it. It needs Flutter and the package cache, so it
// only runs with GOFILAMENT_FLUTTER=1.
func TestFlutterCodegenSampleApp(t *testing.T) {
	if os.Getenv("GOFILAMENT_FLUTTER") == "" {
		t.Skip("set GOFILAMENT_FLUTTER=1 to analyze a generated Flutter app")
	}
	flutter, err := exec.LookPath("flutter")
	if err != nil {
		t.Skip("flutter is not installed")
	}
	pkg, _ := filepath.Abs("../flutter/gofilament_admin")
	dir := t.TempDir()
	if err := flutterclient.GenerateAdminApp(dir, flutterclient.AdminAppOptions{Name: "codegen_sample", PackagePath: pkg, Typed: true}); err != nil {
		t.Fatal(err)
	}
	// the POS page is written by hand; the stock-take page is left to the stub
	_ = os.MkdirAll(filepath.Join(dir, "lib", "pages"), 0o755)
	if err := os.WriteFile(filepath.Join(dir, "lib", "pages", "pos_page.dart"), []byte(samplePosPage), 0o644); err != nil {
		t.Fatal(err)
	}
	field := filepath.Join(dir, "lib", "components", "fields", "signature_pad_field.dart")
	_ = os.MkdirAll(filepath.Dir(field), 0o755)
	if err := os.WriteFile(field, []byte(sampleSignatureField), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := flutterclient.GenerateFromPanel(newCodegenPanel(t), dir, flutterclient.GenerateOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "analysis_options.yaml"), []byte(sampleAnalysisOptions), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(dir, "test"), 0o755)
	if err := os.WriteFile(filepath.Join(dir, "test", "generated_test.dart"), []byte(sampleTest), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"pub", "get", "--offline"}, {"analyze", "--fatal-infos"}, {"test"}} {
		cmd := exec.Command(flutter, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("flutter %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
}

const sampleSignatureField = `import 'package:flutter/material.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import '../../generated/models.g.dart';

class SignaturePadField extends StatelessWidget {
  const SignaturePadField({super.key, required this.field});

  final AdminFieldContext field;

  SignaturePadFieldProps get props => SignaturePadFieldProps.fromJson(field.props);

  @override
  Widget build(BuildContext context) => SizedBox(
        height: props.height.toDouble(),
        child: OutlinedButton(
          onPressed: field.enabled
              ? () => field.onChanged('data:image/png;base64,AAAA')
              : null,
          child: Text(field.value == null ? 'Sign (${props.penColor})' : 'Signed'),
        ),
      );
}
`

const sampleTest = `import 'package:codegen_sample/generated/models.g.dart';
import 'package:codegen_sample/generated/pages.g.dart';
import 'package:codegen_sample/generated/registry.g.dart';
import 'package:codegen_sample/generated/schema.g.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('records round-trip and keep unknown fields', () {
    final order = SalesOrder.fromJson(<String, dynamic>{
      'id': 7,
      'number': 'SO-7',
      'total': '12.5',
      'tags': <dynamic>['a'],
      'lines': <dynamic>[
        <String, dynamic>{'id': 1, 'sku': 'X', 'qty': 2, 'price': 3}
      ],
      'ship': <String, dynamic>{'city': 'Kandy'},
      'created_at': '2026-02-03T04:05:06Z',
      'future_field': true,
    });
    expect(order.id, 7);
    expect(order.total, 12.5);
    expect(order.lines!.single.price, 3.0);
    expect(order.ship!.city, 'Kandy');
    expect(order.createdAt!.day, 3);
    expect(order.toJson()['future_field'], isTrue);
    expect(order.toJson()['created_at'], '2026-02-03T04:05:06.000Z');
  });

  test('inputs send only what is set', () {
    final input = SalesOrderInput(
        number: 'SO-8',
        dueOn: DateTime(2026, 1, 31),
        lines: const [SalesOrderInputLinesItem(sku: 'A')],
        extra: const {'note': null});
    expect(input.toJson(), {
      'number': 'SO-8',
      'due_on': '2026-01-31',
      'lines': [
        {'sku': 'A'}
      ],
      'note': null,
    });
  });

  test('snapshot and pages', () {
    expect(panelSchema['snapshot'], isTrue);
    expect((panelForms['sales-orders'] as Map)['actions'], contains('refund'));
    final pages = panelPages(PanelPageBuilders(
        pos: (_, __) => const SizedBox(), stockTake: (_, __) => const SizedBox()));
    expect(pages.map((p) => p.id), ['pos', 'stock-take']);
    expect(panelAppPages.map((p) => p.label), ['Point of sale', 'Stock take']);
    expect(panelComponents.fields.keys, ['signature_pad']);
    expect(panelComponents.cells.keys, ['paid_badge']);
    expect(panelComponents.widgets.keys, ['sales_total']);
    final props = SignaturePadFieldProps.fromJson(<String, dynamic>{'pen_color': '#000', 'height': 90});
    expect(props.height, 90);
    expect(props.clearable, isNull);
    expect(pages.every((p) => p.serverControlled), isTrue);
  });
}
`

// The generated files silence lints (type=lint); the sample app turns the
// strict ones on for its own code and for the generated code via an override.
const sampleAnalysisOptions = `include: package:flutter_lints/flutter.yaml
analyzer:
  language:
    strict-casts: true
    strict-raw-types: true
`

const samplePosPage = `import 'package:flutter/material.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import '../generated/client.g.dart';
import '../generated/models.g.dart';

class PosPage extends StatelessWidget {
  const PosPage({super.key, required this.admin});

  final AdminController admin;

  PanelClient get client => PanelClient(admin.api);

  Future<String> _demo() async {
    final page = await client.salesOrders.list(search: 'SO');
    final SalesOrder first = page.items.first;
    final created = first.createdAt?.toIso8601String() ?? '';
    await client.salesOrders.create(SalesOrderInput(
      number: 'SO-2',
      total: 10,
      dueOn: DateTime(2026, 1, 31),
      lines: const [SalesOrderInputLinesItem(sku: 'A', qty: 1)],
    ));
    await client.salesOrders.runRefund('1', const SalesOrderRefundInput(amount: 5));
    await client.salesOrders.lines('1').create(const SalesOrderLineInput(sku: 'B'));
    final summary = await client.salesSummary(
        till: 't1', request: const CgSummaryQuery(limit: 3));
    await client.closeTill(till: 't1', request: const CgCloseTill(counted: 10));
    await client.docs.runDiscount('2', const DocDiscountInput(percent: 10));
    return '${first.number} ${first.ship?.city} ${summary.total} $created';
  }

  @override
  Widget build(BuildContext context) => FutureBuilder<String>(
      future: _demo(), builder: (context, s) => Text(s.data ?? ''));
}
`

func TestFlutterGenerateCommand(t *testing.T) {
	dir := t.TempDir()
	build := func() *filament.Panel { return newCodegenPanel(t) }
	cmd := commands.FlutterGenerateCommand(build, dir, flutterclient.GenerateOptions{})
	if cmd.Name != "filament:flutter-generate" {
		t.Fatalf("name: %s", cmd.Name)
	}
	if err := cmd.Run([]string{"--check"}); err == nil || !strings.Contains(err.Error(), "out of date") {
		t.Fatalf("--check before generating should fail, got %v", err)
	}
	if err := cmd.Run(nil); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run([]string{"--check"}); err != nil {
		t.Fatalf("--check after generating: %v", err)
	}
	other := t.TempDir()
	if err := cmd.Run([]string{"--out=" + other}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(other, "lib", "generated", "client.g.dart")); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run([]string{"--bogus"}); err == nil {
		t.Fatal("unknown flags are refused")
	}
}
