package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golaravel/gofilament/pkg/activitylog"
	"github.com/golaravel/gofilament/pkg/colors"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/widgets"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func seedActivityLogs() {
	ctx := context.Background()
	_ = activitylog.GetDefaultStore().Clear(ctx)

	// --- Seed Order #1001 (ORD-8401) Audit Trail ---
	_, _ = activitylog.Log("Order ORD-8401 placed by Sarah Connor").
		Subject("Order", "1001").
		Causer("Customer", "1", "Sarah Connor").
		Event("created").
		New(map[string]any{
			"order_no":           "ORD-8401",
			"customer_name":      "Sarah Connor",
			"customer_email":     "sarah.connor@cyberdyne.io",
			"subtotal":           2344.00,
			"shipping_fee":       0.00,
			"tax":                187.52,
			"total":              2531.52,
			"payment_status":     "pending",
			"fulfillment_status": "processing",
			"shipping_address":   "742 Evergreen Terrace, Los Angeles, CA 90001",
		}).
		CreatedAt(time.Now().Add(-72 * time.Hour)).
		Save(ctx)

	_, _ = activitylog.Log("Payment of $2531.52 authorized & captured via Stripe").
		Subject("Order", "1001").
		Causer("Payment Gateway", "stripe_webhook", "Stripe").
		Event("updated").
		WithDiff(
			map[string]any{"payment_status": "pending"},
			map[string]any{"payment_status": "paid"},
		).
		Property("gateway", "stripe").
		Property("charge_id", "ch_3M4oPQ2eZvKYlo2C1rD4e3fX").
		Property("amount", 2531.52).
		CreatedAt(time.Now().Add(-48 * time.Hour)).
		Save(ctx)

	_, _ = activitylog.Log("Order fulfilled and dispatched with FedEx Express").
		Subject("Order", "1001").
		Causer("Staff", "usr-88", "Warehouse Fulfillment").
		Event("updated").
		WithDiff(
			map[string]any{
				"fulfillment_status": "processing",
				"tracking_number":    "",
			},
			map[string]any{
				"fulfillment_status": "delivered",
				"tracking_number":    "FEDEX-892187319",
			},
		).
		Property("carrier", "FedEx Express Priority").
		Property("estimated_delivery", "2026-11-03").
		CreatedAt(time.Now().Add(-24 * time.Hour)).
		Save(ctx)

	// --- Seed Order #1002 (ORD-8402) Audit Trail ---
	_, _ = activitylog.Log("Order ORD-8402 placed by John Doe").
		Subject("Order", "1002").
		Causer("Customer", "2", "John Doe").
		Event("created").
		New(map[string]any{
			"order_no":           "ORD-8402",
			"customer_name":      "John Doe",
			"customer_email":     "john.doe@acme.corp",
			"total":              99.99,
			"payment_status":     "pending",
			"fulfillment_status": "processing",
		}).
		CreatedAt(time.Now().Add(-40 * time.Hour)).
		Save(ctx)

	_, _ = activitylog.Log("Payment confirmed via PayPal").
		Subject("Order", "1002").
		Causer("System", "paypal_ipn", "PayPal Gateway").
		Event("updated").
		WithDiff(
			map[string]any{"payment_status": "pending"},
			map[string]any{"payment_status": "paid"},
		).
		CreatedAt(time.Now().Add(-30 * time.Hour)).
		Save(ctx)

	_, _ = activitylog.Log("Package shipped by DHL Express").
		Subject("Order", "1002").
		Causer("Courier", "dhl", "DHL Express").
		Event("updated").
		WithDiff(
			map[string]any{"fulfillment_status": "processing"},
			map[string]any{"fulfillment_status": "shipped"},
		).
		CreatedAt(time.Now().Add(-6 * time.Hour)).
		Save(ctx)

	// --- Seed Product #1 Catalog Audit Log ---
	_, _ = activitylog.Log("Catalog item 'Apple AirPods Max' created with multi-step wizard").
		Subject("Product", "1").
		Causer("Admin", "admin-1", "Store Manager").
		Event("created").
		New(map[string]any{
			"title": "Apple AirPods Max",
			"sku":   "APL-APM-SLV",
			"price": 549.00,
			"stock": 42,
		}).
		CreatedAt(time.Now().Add(-120 * time.Hour)).
		Save(ctx)

	_, _ = activitylog.Log("Stock inventory adjusted (-3 units reserved for ORD-2026-1001)").
		Subject("Product", "1").
		Causer("Automation", "inv-bot", "Inventory Syncer").
		Event("updated").
		WithDiff(
			map[string]any{"stock": 45},
			map[string]any{"stock": 42},
		).
		CreatedAt(time.Now().Add(-12 * time.Hour)).
		Save(ctx)
}

func main() {
	// 1. Pre-populate mock database & audit logs
	seedDatabase()
	seedActivityLogs()

	// 2. Initialize GoLaravel Engine
	app := golaravel.Default()

	// 3. Build GoFilament Admin Panel
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

	// 4. Register Executive KPI Dashboard Widgets
	adminPanel.Widget(widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
		dataMu.RLock()
		totalProducts := len(products)
		totalOrders := len(orders)
		totalCustomers := len(customers)

		var totalRev float64
		for _, o := range orders {
			if o.PaymentStatus == "paid" {
				totalRev += o.Total
			}
		}
		dataMu.RUnlock()

		avgOrderVal := 0.0
		if totalOrders > 0 {
			avgOrderVal = totalRev / float64(totalOrders)
		}

		return []widgets.Stat{
			*widgets.NewStat("Total Revenue", fmt.Sprintf("$%.2f", 124580.00+totalRev)).
				Description("+18.4% vs last month").
				Color("success").
				Chart([]int{45, 52, 58, 70, 84, 98, 124}),

			*widgets.NewStat("Orders Fulfilled", fmt.Sprintf("%d", 1482+totalOrders)).
				Description("+8.2% new transactions").
				Color("primary").
				Chart([]int{120, 140, 135, 180, 210, 240, 265}).
				URL("/admin/orders"),

			*widgets.NewStat("Average Order Value", fmt.Sprintf("$%.2f", avgOrderVal+84.07)).
				Description("-1.2% seasonal shift").
				Color("warning").
				Chart([]int{86, 85, 87, 85, 83, 84}),

			*widgets.NewStat("Active Customers", fmt.Sprintf("%d", 3890+totalCustomers)).
				Description(fmt.Sprintf("%d catalog items in stock", totalProducts)).
				Color("success").
				Chart([]int{280, 310, 350, 420, 480, 520, 560}).
				URL("/admin/customers"),
		}
	}))

	// 5. Register Resources with Full Automated CRUD Auditing
	productResource := activitylog.EnableAuditLog(buildProductResource())
	orderResource := activitylog.EnableAuditLog(buildOrderResource())
	customerResource := activitylog.EnableAuditLog(buildCustomerResource())
	activityResource := activitylog.NewActivityResource().NavigationGroup("System & Auditing")

	adminPanel.Resource(productResource)
	adminPanel.Resource(orderResource)
	adminPanel.Resource(customerResource)
	adminPanel.Resource(activityResource)

	// 6. Mount Panel into GoLaravel
	if err := adminPanel.Register(app); err != nil {
		log.Fatalf("Failed to mount admin panel: %v", err)
	}

	// 7. Route / to /admin
	app.GET("/", func(c *gcontext.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/admin")
	})

	// 8. Start HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("\n✨ Apex Store Admin Panel running at http://localhost:%s/admin\n\n", port)
	if err := app.Run(":" + port); err != nil {
		log.Fatalf("Server stopped: %v", err)
	}
}
