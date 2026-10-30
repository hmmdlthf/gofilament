package tests

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/gofilament/pkg/widgets"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type AdvModel struct {
	ID       string
	Status   string
	Color    string
	IsActive bool
}

func TestAdvancedColumns(t *testing.T) {
	tbl := tables.New[*AdvModel]().
		Columns(
			tables.IconColumn[*AdvModel]("is_active").Boolean(),
			tables.ColorColumn[*AdvModel]("color"),
			tables.SelectColumn[*AdvModel]("status").Options(map[string]string{
				"pending": "Pending Review",
				"active":  "Active",
			}),
			tables.TextInputColumn[*AdvModel]("id"),
		)

	models := []*AdvModel{
		{ID: "001", Status: "active", Color: "#10b981", IsActive: true},
	}

	tbl.SetData(models, 1, 1, 10, "", "", "asc", nil)
	html := tbl.Render("/admin/adv")

	if !strings.Contains(html, "✓") {
		t.Errorf("Expected icon column checkmark, got:\n%s", html)
	}
	if !strings.Contains(html, "#10b981") {
		t.Errorf("Expected color column hex preview, got:\n%s", html)
	}
	if !strings.Contains(html, "Pending Review") {
		t.Errorf("Expected select column option, got:\n%s", html)
	}
}

func TestAdvancedFormComponents(t *testing.T) {
	form := forms.New().
		Schema(
			forms.Wizard(
				forms.WizardStep("Step 1").
					Description("Basic Info").
					Schema(
						forms.Radio("tier").
							Options(map[string]string{
								"pro":        "Pro Plan",
								"enterprise": "Enterprise Plan",
							}).
							Descriptions(map[string]string{
								"pro": "Best for teams",
							}).
							Default("pro"),
						forms.CheckboxList("features").
							Options(map[string]string{
								"analytics": "Advanced Analytics",
								"support":   "24/7 Priority Support",
							}),
						forms.TimePicker("start_time"),
					),
				forms.WizardStep("Step 2").
					Description("Configuration").
					Schema(
						forms.Fieldset("Advanced Settings").
							Schema(
								forms.KeyValue("metadata"),
								forms.CodeEditor("config_json").Language("json"),
								forms.MarkdownEditor("notes").Rows(6),
							),
						forms.Repeater("contacts").
							Schema(
								forms.TextInput("name"),
							),
					),
			),
		).
		Fill(map[string]any{
			"tier": "enterprise",
		})

	html := form.Render(context.Background())

	if !strings.Contains(html, "Enterprise Plan") {
		t.Errorf("Expected radio label 'Enterprise Plan'")
	}
	if !strings.Contains(html, "Advanced Analytics") {
		t.Errorf("Expected checkbox list label 'Advanced Analytics'")
	}
	if !strings.Contains(html, "Advanced Settings") {
		t.Errorf("Expected fieldset legend 'Advanced Settings'")
	}
	if !strings.Contains(html, "config_json") {
		t.Errorf("Expected code editor field 'config_json'")
	}
	if !strings.Contains(html, "+ Add Item") {
		t.Errorf("Expected repeater add item button")
	}
}

func TestGlobalSearchAndLoginAndExport(t *testing.T) {
	app := golaravel.New()
	panel := filament.NewPanel("admin").
		Path("/admin").
		GlobalSearch(true).
		Login(true)

	res := filament.NewResource[*AdvModel]("items").
		ModelName("Item").
		PluralModelName("Items").
		GloballySearchable(true).
		RecordID(func(m *AdvModel) string { return m.ID }).
		OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*AdvModel, int, error) {
			return []*AdvModel{
				{ID: "itm_100", Status: "active"},
			}, 1, nil
		})

	panel.Resource(res)
	_ = panel.Register(app)

	client := app.Test(t)

	// 1. Global Search JSON
	searchResp := client.Request(http.MethodGet, "/admin/global-search?query=itm").
		WithHeader("Accept", "application/json").
		Send()
	searchResp.AssertOK().
		AssertSee("itm_100").
		AssertSee("Items")

	// 2. Login Screen
	loginResp := client.Get("/admin/login")
	loginResp.AssertOK().
		AssertSee("Sign In").
		AssertSee("Email address")

	// 3. Export CSV
	exportResp := client.Get("/admin/items/export")
	exportResp.AssertOK().
		AssertContentType("text/csv").
		AssertHeader("Content-Disposition", "attachment; filename=\"items-export.csv\"").
		AssertSee("itm_100")
}

func TestTableWidget(t *testing.T) {
	widget := widgets.NewTableWidget("Recent Invoices").
		Columns(
			widgets.TableWidgetColumn{Name: "id", Label: "Invoice #"},
			widgets.TableWidgetColumn{Name: "amount", Label: "Amount"},
		).
		Rows(func(ctx *filament.Context) []map[string]any {
			return []map[string]any{
				{"id": "INV-001", "amount": "$450.00"},
				{"id": "INV-002", "amount": "$1,200.00"},
			}
		})

	html := widget.Render(nil)

	if !strings.Contains(html, "Recent Invoices") {
		t.Errorf("Expected widget heading 'Recent Invoices'")
	}
	if !strings.Contains(html, "INV-001") || !strings.Contains(html, "$1,200.00") {
		t.Errorf("Expected row values 'INV-001' and '$1,200.00', got:\n%s", html)
	}
}

func TestAdvancedInfolistEntries(t *testing.T) {
	infolist := infolists.New().
		Schema(
			infolists.IconEntry("is_verified").Boolean(),
			infolists.ImageEntry("avatar_url").Circular(),
			infolists.RepeatableEntry("orders").
				Columns(2).
				Schema(
					infolists.TextEntry("order_number"),
					infolists.TextEntry("total").Money("USD"),
				),
		).
		SetRecord(map[string]any{
			"is_verified": true,
			"avatar_url":  "https://example.com/photo.jpg",
			"orders": []map[string]any{
				{"order_number": "ORD-991", "total": "55.00"},
			},
		})

	html := infolist.Render()

	if !strings.Contains(html, "✓") {
		t.Errorf("Expected icon entry boolean checkmark")
	}
	if !strings.Contains(html, "https://example.com/photo.jpg") {
		t.Errorf("Expected image url in avatar")
	}
	if !strings.Contains(html, "ORD-991") {
		t.Errorf("Expected repeatable entry order number 'ORD-991'")
	}
}
