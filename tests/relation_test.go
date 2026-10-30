package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/commands"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type MockCustomer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type MockOrder struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	OrderNo    string `json:"order_no"`
	Total      int    `json:"total"`
	Status     string `json:"status"`
}

type MockAddress struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Street     string `json:"street"`
	City       string `json:"city"`
}

func setupCustomerAndOrdersApp(t *testing.T) (*golaravel.Engine, map[string]*MockCustomer, map[string]*MockOrder, map[string]*MockAddress) {
	app := golaravel.New()

	customers := map[string]*MockCustomer{
		"1": {ID: "1", Name: "Alice Smith", Email: "alice@example.com"},
		"2": {ID: "2", Name: "Bob Jones", Email: "bob@example.com"},
	}

	orders := map[string]*MockOrder{
		"101": {ID: "101", CustomerID: "1", OrderNo: "ORD-101", Total: 250, Status: "completed"},
		"102": {ID: "102", CustomerID: "1", OrderNo: "ORD-102", Total: 80, Status: "pending"},
		"201": {ID: "201", CustomerID: "2", OrderNo: "ORD-201", Total: 500, Status: "completed"},
	}

	addresses := map[string]*MockAddress{
		"1": {ID: "1", CustomerID: "1", Street: "123 Market St", City: "San Francisco"},
	}

	ordersRM := filament.NewRelationManager[*MockOrder]("orders", "Customer Orders").
		RecordTitleAttribute("order_no").
		Form(func(f *forms.Form) *forms.Form {
			return f.Schema(
				forms.TextInput("order_no").Required(),
				forms.TextInput("total").Numeric().Required(),
				forms.Select("status").Options(map[string]string{
					"pending":   "Pending",
					"completed": "Completed",
				}),
			)
		}).
		Table(func(t *tables.Table[*MockOrder]) *tables.Table[*MockOrder] {
			return t.Columns(
				tables.TextColumn[*MockOrder]("id").Sortable(),
				tables.TextColumn[*MockOrder]("order_no").Searchable(),
				tables.TextColumn[*MockOrder]("total"),
				tables.BadgeColumn[*MockOrder]("status"),
			)
		}).
		OnList(func(ctx *filament.Context, customerID string, search, sort, dir string, page, perPage int) ([]*MockOrder, int, error) {
			var list []*MockOrder
			for _, o := range orders {
				if o.CustomerID == customerID {
					list = append(list, o)
				}
			}
			return list, len(list), nil
		}).
		OnCreate(func(ctx *filament.Context, customerID string, data map[string]any) error {
			id := fmt.Sprintf("%d", 100+len(orders)+1)
			total := 0
			if rawTotal, ok := data["total"]; ok {
				fmt.Sscanf(fmt.Sprintf("%v", rawTotal), "%d", &total)
			}
			orders[id] = &MockOrder{
				ID:         id,
				CustomerID: customerID,
				OrderNo:    fmt.Sprintf("%v", data["order_no"]),
				Total:      total,
				Status:     fmt.Sprintf("%v", data["status"]),
			}
			return nil
		}).
		OnUpdate(func(ctx *filament.Context, customerID, orderID string, data map[string]any) error {
			if o, ok := orders[orderID]; ok && o.CustomerID == customerID {
				if orderNo, exists := data["order_no"]; exists {
					o.OrderNo = fmt.Sprintf("%v", orderNo)
				}
				if status, exists := data["status"]; exists {
					o.Status = fmt.Sprintf("%v", status)
				}
			}
			return nil
		}).
		OnDelete(func(ctx *filament.Context, customerID, orderID string) error {
			if o, ok := orders[orderID]; ok && o.CustomerID == customerID {
				delete(orders, orderID)
			}
			return nil
		})

	addressesRM := filament.NewRelationManager[*MockAddress]("addresses", "Addresses").
		RecordTitleAttribute("street").
		Table(func(t *tables.Table[*MockAddress]) *tables.Table[*MockAddress] {
			return t.Columns(
				tables.TextColumn[*MockAddress]("id"),
				tables.TextColumn[*MockAddress]("street"),
				tables.TextColumn[*MockAddress]("city"),
			)
		}).
		OnList(func(ctx *filament.Context, customerID string, search, sort, dir string, page, perPage int) ([]*MockAddress, int, error) {
			var list []*MockAddress
			for _, a := range addresses {
				if a.CustomerID == customerID {
					list = append(list, a)
				}
			}
			return list, len(list), nil
		})

	customerRes := filament.NewResource[*MockCustomer]("customers").
		ModelName("Customer").
		Relations(ordersRM, addressesRM).
		OnList(func(ctx *filament.Context, search, sort, dir string, page, perPage int, filters map[string]string) ([]*MockCustomer, int, error) {
			var list []*MockCustomer
			for _, c := range customers {
				list = append(list, c)
			}
			return list, len(list), nil
		}).
		OnFind(func(ctx *filament.Context, id string) (*MockCustomer, error) {
			if c, ok := customers[id]; ok {
				return c, nil
			}
			return nil, os.ErrNotExist
		})

	panel := filament.NewPanel("admin").
		Path("/admin").
		BrandName("Shop Admin").
		Resource(customerRes)

	_ = panel.Register(app)
	return app, customers, orders, addresses
}

func TestRelationManager_WebRenderingAndTabs(t *testing.T) {
	app, _, _, _ := setupCustomerAndOrdersApp(t)
	client := app.Test(t)

	// 1. GET /admin/customers/1/edit renders in-place relation manager
	resp := client.Get("/admin/customers/1/edit")
	resp.AssertOK()

	// Verifies tab bar rendered for multiple relations
	resp.AssertSee("activeRelTab = 'orders'")
	resp.AssertSee("Customer Orders")
	resp.AssertSee("Addresses")

	// Verifies in-place "+ New Customer Orders" button and Create Modal
	resp.AssertSee("+ New Customer Orders")
	resp.AssertSee("modal_rel_orders_create")
	resp.AssertSee("/admin/customers/1/relations/orders/create")

	// Verifies child records in table
	resp.AssertSee("ORD-101")
	resp.AssertSee("ORD-102")

	// 2. GET /admin/customers/1 (View page) also renders relations
	viewResp := client.Get("/admin/customers/1")
	viewResp.AssertOK()
	viewResp.AssertSee("Customer Orders")
	viewResp.AssertSee("ORD-101")
}

func TestRelationManager_WebInPlaceLifecycle(t *testing.T) {
	app, _, orders, _ := setupCustomerAndOrdersApp(t)
	client := app.Test(t)

	// 1. In-place Create related record
	createResp := client.PostForm("/admin/customers/1/relations/orders/create", map[string]string{
		"order_no": "ORD-103",
		"total":    "340",
		"status":   "pending",
	})
	createResp.AssertRedirect("/admin/customers/1/edit")

	// Verify order created in store linked to customer 1
	var createdOrder *MockOrder
	for _, o := range orders {
		if o.OrderNo == "ORD-103" {
			createdOrder = o
			break
		}
	}
	if createdOrder == nil || createdOrder.CustomerID != "1" || createdOrder.Total != 340 {
		t.Fatalf("expected order 103 to be created for customer 1, got %+v", createdOrder)
	}

	// 2. In-place Edit related record
	editResp := client.PostForm(fmt.Sprintf("/admin/customers/1/relations/orders/%s/edit", createdOrder.ID), map[string]string{
		"order_no": "ORD-103-UPDATED",
		"status":   "completed",
	})
	editResp.AssertRedirect("/admin/customers/1/edit")

	if orders[createdOrder.ID].OrderNo != "ORD-103-UPDATED" || orders[createdOrder.ID].Status != "completed" {
		t.Fatalf("expected updated order, got %+v", orders[createdOrder.ID])
	}

	// 3. In-place Delete related record
	delResp := client.PostForm(fmt.Sprintf("/admin/customers/1/relations/orders/%s/delete", createdOrder.ID), nil)
	delResp.AssertRedirect("/admin/customers/1/edit")

	if _, exists := orders[createdOrder.ID]; exists {
		t.Fatalf("expected order %s to be deleted", createdOrder.ID)
	}
}

func TestRelationManager_REST_API(t *testing.T) {
	app, _, orders, _ := setupCustomerAndOrdersApp(t)
	client := app.Test(t)

	// 1. GET /admin/api/resources/customers/1/relations - metadata discovery
	metaResp := client.Get("/admin/api/resources/customers/1/relations")
	metaResp.AssertOK()

	var metaData map[string]any
	if err := json.Unmarshal(metaResp.Body(), &metaData); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	rels := metaData["relations"].([]any)
	if len(rels) != 2 {
		t.Fatalf("expected 2 relation managers, got %d", len(rels))
	}

	// 2. GET /admin/api/resources/customers/1/relations/orders - list related records
	listResp := client.Get("/admin/api/resources/customers/1/relations/orders")
	listResp.AssertOK().AssertSee("ORD-101").AssertSee("ORD-102")

	var listData map[string]any
	_ = json.Unmarshal(listResp.Body(), &listData)
	if int(listData["total"].(float64)) != 2 {
		t.Fatalf("expected 2 orders for customer 1, got %v", listData["total"])
	}

	// 3. POST /admin/api/resources/customers/1/relations/orders - create via API
	createResp := client.PostJSON("/admin/api/resources/customers/1/relations/orders", map[string]any{
		"order_no": "ORD-555",
		"total":    999,
		"status":   "pending",
	})
	createResp.AssertStatus(http.StatusCreated)

	var foundID string
	for id, o := range orders {
		if o.OrderNo == "ORD-555" {
			foundID = id
			break
		}
	}
	if foundID == "" {
		t.Fatalf("expected order ORD-555 to be created")
	}

	// 4. PUT /admin/api/resources/customers/1/relations/orders/:id - update via API
	updateResp := client.Request(http.MethodPut, fmt.Sprintf("/admin/api/resources/customers/1/relations/orders/%s", foundID)).
		WithJSON(map[string]any{"order_no": "ORD-555-PAID", "status": "completed"}).
		Send()
	updateResp.AssertOK()

	if orders[foundID].OrderNo != "ORD-555-PAID" || orders[foundID].Status != "completed" {
		t.Fatalf("expected order updated via API, got %+v", orders[foundID])
	}

	// 5. DELETE /admin/api/resources/customers/1/relations/orders/:id - delete via API
	delResp := client.Request(http.MethodDelete, fmt.Sprintf("/admin/api/resources/customers/1/relations/orders/%s", foundID)).Send()
	delResp.AssertOK()

	if _, exists := orders[foundID]; exists {
		t.Fatalf("expected order %s to be deleted via API", foundID)
	}
}

func TestArtisanMakeFilamentRelationCommand(t *testing.T) {
	cmd := commands.MakeFilamentRelationCommand()
	err := cmd.Run([]string{"Customer", "Orders"})
	if err != nil {
		t.Fatalf("artisan make:filament-relation failed: %v", err)
	}

	relPath := filepath.Join("app", "filament", "resources", "customer", "relations", "orders_relation_manager.go")
	defer os.RemoveAll(filepath.Join("app", "filament", "resources", "customer"))

	content, err := os.ReadFile(relPath)
	if err != nil {
		t.Fatalf("failed to read generated relation manager: %v", err)
	}
	str := string(content)
	if !strings.Contains(str, "OrdersRelationManager") {
		t.Fatalf("expected OrdersRelationManager in generated file")
	}
	if !strings.Contains(str, "NewRelationManager") {
		t.Fatalf("expected NewRelationManager in generated file")
	}
}
