package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/activitylog"
	"github.com/golaravel/gofilament/pkg/colors"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/widgets"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func createTestEcommerceApp(t *testing.T) *golaravel.Engine {
	seedDatabase()
	seedActivityLogs()

	app := golaravel.Default()

	adminPanel := filament.NewPanel("admin").
		Path("/admin").
		BrandName("Apex Store").
		Colors(map[string]any{
			colors.RolePrimary: colors.Emerald,
			colors.RoleGray:    colors.Zinc,
		}).
		DarkMode(true).
		SidebarCollapsibleOnDesktop().
		MaxContentWidth(filament.WidthFull).
		NavigationGroups([]string{
			"Sales & Fulfillment",
			"Store Catalog",
			"Customers & CRM",
			"System & Auditing",
		})

	adminPanel.Widget(widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
		return []widgets.Stat{
			*widgets.NewStat("Total Revenue", "$124,580.00").
				Description("+18.4% vs last month").
				Color("success").
				Chart([]int{45, 52, 58, 70, 84, 98, 124}),
			*widgets.NewStat("Orders Fulfilled", "1,482").
				Description("+8.2% new transactions").
				Color("primary").
				URL("/admin/orders"),
			*widgets.NewStat("Average Order Value", "$84.07").
				Description("-1.2% seasonal shift").
				Color("warning"),
			*widgets.NewStat("Active Customers", "3,890").
				Description("catalog items in stock").
				Color("success").
				URL("/admin/customers"),
		}
	}))

	adminPanel.Resource(activitylog.EnableAuditLog(buildProductResource()))
	adminPanel.Resource(activitylog.EnableAuditLog(buildOrderResource()))
	adminPanel.Resource(activitylog.EnableAuditLog(buildCustomerResource()))
	adminPanel.Resource(activitylog.NewActivityResource().NavigationGroup("System & Auditing"))

	if err := adminPanel.Register(app); err != nil {
		t.Fatalf("Failed to mount panel: %v", err)
	}

	return app
}

func TestEcommerce_DashboardAndKPIWidgets(t *testing.T) {
	app := createTestEcommerceApp(t)

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Apex Store") {
		t.Errorf("expected brand name 'Apex Store' in dashboard HTML")
	}
	if !strings.Contains(body, "Total Revenue") || !strings.Contains(body, "$124,580.00") {
		t.Errorf("expected KPI stat 'Total Revenue' in dashboard")
	}
	if !strings.Contains(body, "Orders Fulfilled") {
		t.Errorf("expected KPI stat 'Orders Fulfilled' in dashboard")
	}
	if !strings.Contains(body, "Average Order Value") {
		t.Errorf("expected KPI stat 'Average Order Value' in dashboard")
	}
}

func TestEcommerce_ProductResource_TableAndWizard(t *testing.T) {
	app := createTestEcommerceApp(t)

	// 1. Table view: verify columns, products, inline editing
	req := httptest.NewRequest("GET", "/admin/products", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Apple AirPods Max") {
		t.Errorf("expected Apple AirPods Max in products list")
	}
	if !strings.Contains(body, "Keychron Q1 Pro Wireless Mechanical Keyboard") {
		t.Errorf("expected Keychron Q1 Pro in products list")
	}
	// Check inline edit cell endpoints present
	if !strings.Contains(body, "/admin/products/cell") {
		t.Errorf("expected inline cell editing endpoint in product table")
	}

	// 2. Form view: verify Wizard multi-step form rendering
	reqWizard := httptest.NewRequest("GET", "/admin/products/create", nil)
	wWizard := httptest.NewRecorder()
	app.ServeHTTP(wWizard, reqWizard)

	if wWizard.Code != http.StatusOK {
		t.Fatalf("expected status 200 for product create wizard, got %d", wWizard.Code)
	}

	wizardBody := wWizard.Body.String()
	if !strings.Contains(wizardBody, "Basic Info") {
		t.Errorf("expected wizard step 'Basic Info' in create product form")
	}
	if !strings.Contains(wizardBody, "Pricing & Stock") {
		t.Errorf("expected wizard step 'Pricing & Stock' in create product form")
	}
	if !strings.Contains(wizardBody, "Media & Description") {
		t.Errorf("expected wizard step 'Media & Description' in create product form")
	}
}

func TestEcommerce_OrderResource_InfolistAndTimeline(t *testing.T) {
	app := createTestEcommerceApp(t)

	// 1. Orders table
	req := httptest.NewRequest("GET", "/admin/orders", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for orders list, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "ORD-8401") {
		t.Errorf("expected ORD-8401 in orders table")
	}

	// 2. Order infolist view with timeline entries
	reqView := httptest.NewRequest("GET", "/admin/orders/1001", nil)
	wView := httptest.NewRecorder()
	app.ServeHTTP(wView, reqView)

	if wView.Code != http.StatusOK {
		t.Fatalf("expected status 200 for order 1001 view, got %d", wView.Code)
	}

	viewBody := wView.Body.String()
	if !strings.Contains(viewBody, "Sarah Connor") {
		t.Errorf("expected customer Sarah Connor in order view")
	}
	if !strings.Contains(viewBody, "FEDEX-892187319") {
		t.Errorf("expected FedEx tracking number in order view")
	}
	// Check audit timeline rendered
	if !strings.Contains(viewBody, "Audit Trail &amp; Order Timeline") {
		t.Errorf("expected audit timeline section in order view")
	}
	if !strings.Contains(viewBody, "Order fulfilled and dispatched with FedEx Express") {
		t.Errorf("expected timeline activity for fulfillment dispatch")
	}
}

func TestEcommerce_CustomerResource_TagsAndSoftDeletes(t *testing.T) {
	app := createTestEcommerceApp(t)

	// 1. Customers list
	req := httptest.NewRequest("GET", "/admin/customers", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for customers list, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Sarah Connor") {
		t.Errorf("expected Sarah Connor in customers list")
	}
	if !strings.Contains(body, "VIP") {
		t.Errorf("expected VIP badge or tag in customers list")
	}

	// 2. Audit logs admin resource
	reqAudit := httptest.NewRequest("GET", "/admin/activity-logs", nil)
	wAudit := httptest.NewRecorder()
	app.ServeHTTP(wAudit, reqAudit)

	if wAudit.Code != http.StatusOK {
		t.Fatalf("expected status 200 for activity logs, got %d", wAudit.Code)
	}

	auditBody := wAudit.Body.String()
	if !strings.Contains(auditBody, "Audit Logs") {
		t.Errorf("expected Audit Logs heading")
	}
	if !strings.Contains(auditBody, "Sarah Connor") {
		t.Errorf("expected activity causer Sarah Connor in audit log list")
	}
}
