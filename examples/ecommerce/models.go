package main

import (
	"fmt"
	"sync"
	"time"
)

// Product represents a merchandise item in the store catalog.
type Product struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	SKU             string     `json:"sku"`
	Brand           string     `json:"brand"`
	Price           float64    `json:"price"`
	CompareAtPrice  float64    `json:"compare_at_price"`
	CostPrice       float64    `json:"cost_price"`
	Stock           int        `json:"stock"`
	Categories      []string   `json:"categories"`
	ColorVariant    string     `json:"color_variant"`
	Description     string     `json:"description"`
	IsPublished     bool       `json:"is_published"`
	InventoryStatus string     `json:"inventory_status"` // in_stock, low_stock, out_of_stock
	SortOrder       int        `json:"sort_order"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// OrderItem represents a line item within an order.
type OrderItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	Total       float64 `json:"total"`
}

// Order captures a customer purchase transaction.
type Order struct {
	ID                string      `json:"id"`
	OrderNo           string      `json:"order_no"`
	CustomerID        string      `json:"customer_id"`
	CustomerName      string      `json:"customer_name"`
	CustomerEmail     string      `json:"customer_email"`
	Items             []OrderItem `json:"items"`
	Subtotal          float64     `json:"subtotal"`
	Tax               float64     `json:"tax"`
	ShippingFee       float64     `json:"shipping_fee"`
	Total             float64     `json:"total"`
	PaymentStatus     string      `json:"payment_status"`     // paid, pending, refunded
	FulfillmentStatus string      `json:"fulfillment_status"` // processing, shipped, delivered, cancelled
	ShippingAddress   string      `json:"shipping_address"`
	TrackingNumber    string      `json:"tracking_number"`
	Notes             string      `json:"notes"`
	CreatedAt         time.Time   `json:"created_at"`
}

// Customer represents a buyer in the eCommerce store.
type Customer struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Avatar      string     `json:"avatar"`
	City        string     `json:"city"`
	Country     string     `json:"country"`
	TotalSpent  float64    `json:"total_spent"`
	OrdersCount int        `json:"orders_count"`
	IsVIP       bool       `json:"is_vip"`
	Tags        []string   `json:"tags"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Thread-safe in-memory databases
var (
	dataMu    sync.RWMutex
	products  = make(map[string]*Product)
	orders    = make(map[string]*Order)
	customers = make(map[string]*Customer)
)

func seedDatabase() {
	dataMu.Lock()
	defer dataMu.Unlock()

	products = make(map[string]*Product)
	orders = make(map[string]*Order)
	customers = make(map[string]*Customer)

	now := time.Now()
	trashedDate := now.Add(-48 * time.Hour)

	// Seed Products
	prodList := []*Product{
		{
			ID:              "1",
			Title:           "Apple AirPods Max - Space Gray",
			Slug:            "apple-airpods-max-space-gray",
			SKU:             "APL-AIRPODS-MAX-GRY",
			Brand:           "Apple",
			Price:           549.00,
			CompareAtPrice:  599.00,
			CostPrice:       380.00,
			Stock:           45,
			Categories:      []string{"Audio", "Accessories", "Electronics"},
			ColorVariant:    "#4b5563",
			Description:     "High-fidelity audio with Active Noise Cancellation and spatial audio.",
			IsPublished:     true,
			InventoryStatus: "in_stock",
			SortOrder:       1,
			CreatedAt:       now.Add(-10 * 24 * time.Hour),
		},
		{
			ID:              "2",
			Title:           "Keychron Q1 Pro Wireless Mechanical Keyboard",
			Slug:            "keychron-q1-pro-wireless",
			SKU:             "KYC-Q1P-RGB-RED",
			Brand:           "Keychron",
			Price:           199.99,
			CompareAtPrice:  219.99,
			CostPrice:       120.00,
			Stock:           18,
			Categories:      []string{"Peripherals", "Keyboards"},
			ColorVariant:    "#1f2937",
			Description:     "Custom full-metal wireless mechanical keyboard with QMK/VIA support.",
			IsPublished:     true,
			InventoryStatus: "in_stock",
			SortOrder:       2,
			CreatedAt:       now.Add(-8 * 24 * time.Hour),
		},
		{
			ID:              "3",
			Title:           "Sony WH-1000XM5 Wireless Headphones",
			Slug:            "sony-wh-1000xm5-wireless",
			SKU:             "SNY-WH1000XM5-BLK",
			Brand:           "Sony",
			Price:           398.00,
			CompareAtPrice:  448.00,
			CostPrice:       260.00,
			Stock:           5,
			Categories:      []string{"Audio", "Noise-Cancelling"},
			ColorVariant:    "#111827",
			Description:     "Industry-leading noise cancellation optimized with two processors and 8 microphones.",
			IsPublished:     true,
			InventoryStatus: "low_stock",
			SortOrder:       3,
			CreatedAt:       now.Add(-6 * 24 * time.Hour),
		},
		{
			ID:              "4",
			Title:           "Logitech MX Master 3S Ergonomic Mouse",
			Slug:            "logitech-mx-master-3s",
			SKU:             "LOG-MXM3S-PGR",
			Brand:           "Logitech",
			Price:           99.99,
			CompareAtPrice:  109.99,
			CostPrice:       60.00,
			Stock:           0,
			Categories:      []string{"Peripherals", "Productivity"},
			ColorVariant:    "#374151",
			Description:     "Quiet clicks with 8000 DPI track-on-glass sensor and MagSpeed scrolling.",
			IsPublished:     false,
			InventoryStatus: "out_of_stock",
			SortOrder:       4,
			CreatedAt:       now.Add(-4 * 24 * time.Hour),
		},
		{
			ID:              "5",
			Title:           "Herman Miller Embody Ergonomic Chair",
			Slug:            "herman-miller-embody-chair",
			SKU:             "HML-EMBODY-CYAN",
			Brand:           "Herman Miller",
			Price:           1795.00,
			CompareAtPrice:  1995.00,
			CostPrice:       1100.00,
			Stock:           12,
			Categories:      []string{"Furniture", "Office"},
			ColorVariant:    "#0284c7",
			Description:     "Ergonomic health-positive seating supporting natural blood flow and posture.",
			IsPublished:     true,
			InventoryStatus: "in_stock",
			SortOrder:       5,
			CreatedAt:       now.Add(-2 * 24 * time.Hour),
		},
		{
			ID:              "6",
			Title:           "Discontinued Wireless Charger Gen 1",
			Slug:            "discontinued-wireless-charger-v1",
			SKU:             "OLD-CHG-10W",
			Brand:           "Generic",
			Price:           19.99,
			CompareAtPrice:  29.99,
			CostPrice:       8.00,
			Stock:           0,
			Categories:      []string{"Accessories"},
			ColorVariant:    "#9ca3af",
			Description:     "Old generation wireless charging pad.",
			IsPublished:     false,
			InventoryStatus: "out_of_stock",
			SortOrder:       6,
			DeletedAt:       &trashedDate, // Trashed item to showcase Trash Recovery Bin!
			CreatedAt:       now.Add(-60 * 24 * time.Hour),
		},
	}

	for _, p := range prodList {
		products[p.ID] = p
	}

	// Seed Customers
	custList := []*Customer{
		{
			ID:          "1",
			Name:        "Sarah Connor",
			Email:       "sarah.connor@cyberdyne.io",
			Phone:       "+1 (555) 382-9912",
			City:        "Los Angeles",
			Country:     "United States",
			TotalSpent:  2344.00,
			OrdersCount: 3,
			IsVIP:       true,
			Tags:        []string{"VIP", "Tech Enthusiast", "Early Adopter"},
			CreatedAt:   now.Add(-120 * 24 * time.Hour),
		},
		{
			ID:          "2",
			Name:        "John Doe",
			Email:       "john.doe@acme.corp",
			Phone:       "+1 (555) 782-1200",
			City:        "Seattle",
			Country:     "United States",
			TotalSpent:  549.00,
			OrdersCount: 1,
			IsVIP:       false,
			Tags:        []string{"New Customer"},
			CreatedAt:   now.Add(-40 * 24 * time.Hour),
		},
		{
			ID:          "3",
			Name:        "Elena Rostova",
			Email:       "elena.r@nordictech.fi",
			Phone:       "+358 40 1234567",
			City:        "Helsinki",
			Country:     "Finland",
			TotalSpent:  1894.99,
			OrdersCount: 2,
			IsVIP:       true,
			Tags:        []string{"VIP", "B2B Buyer"},
			CreatedAt:   now.Add(-85 * 24 * time.Hour),
		},
		{
			ID:          "4",
			Name:        "Spam Account Inactive",
			Email:       "spammer@botnet.xyz",
			Phone:       "+0 (000) 000-0000",
			City:        "Unknown",
			Country:     "Unknown",
			TotalSpent:  0.00,
			OrdersCount: 0,
			IsVIP:       false,
			Tags:        []string{"Spam"},
			DeletedAt:   &trashedDate, // Trashed customer
			CreatedAt:   now.Add(-30 * 24 * time.Hour),
		},
	}

	for _, c := range custList {
		customers[c.ID] = c
	}

	// Seed Orders
	ordersList := []*Order{
		{
			ID:            "1001",
			OrderNo:       "ORD-8401",
			CustomerID:    "1",
			CustomerName:  "Sarah Connor",
			CustomerEmail: "sarah.connor@cyberdyne.io",
			Items: []OrderItem{
				{ProductID: "5", ProductName: "Herman Miller Embody Chair", Quantity: 1, UnitPrice: 1795.00, Total: 1795.00},
				{ProductID: "1", ProductName: "AirPods Max - Space Gray", Quantity: 1, UnitPrice: 549.00, Total: 549.00},
			},
			Subtotal:          2344.00,
			Tax:               187.52,
			ShippingFee:       0.00,
			Total:             2531.52,
			PaymentStatus:     "paid",
			FulfillmentStatus: "delivered",
			ShippingAddress:   "742 Evergreen Terrace, Los Angeles, CA 90001",
			TrackingNumber:    "FEDEX-892187319",
			Notes:             "Signature required on delivery.",
			CreatedAt:         now.Add(-48 * time.Hour),
		},
		{
			ID:            "1002",
			OrderNo:       "ORD-8402",
			CustomerID:    "2",
			CustomerName:  "John Doe",
			CustomerEmail: "john.doe@acme.corp",
			Items: []OrderItem{
				{ProductID: "1", ProductName: "AirPods Max - Space Gray", Quantity: 1, UnitPrice: 549.00, Total: 549.00},
			},
			Subtotal:          549.00,
			Tax:               43.92,
			ShippingFee:       15.00,
			Total:             607.92,
			PaymentStatus:     "paid",
			FulfillmentStatus: "shipped",
			ShippingAddress:   "100 Pike Street, Suite 400, Seattle, WA 98101",
			TrackingNumber:    "UPS-1Z9999999999",
			Notes:             "Leave at front desk.",
			CreatedAt:         now.Add(-12 * time.Hour),
		},
		{
			ID:            "1003",
			OrderNo:       "ORD-8403",
			CustomerID:    "3",
			CustomerName:  "Elena Rostova",
			CustomerEmail: "elena.r@nordictech.fi",
			Items: []OrderItem{
				{ProductID: "2", ProductName: "Keychron Q1 Pro Wireless Keyboard", Quantity: 2, UnitPrice: 199.99, Total: 399.98},
				{ProductID: "3", ProductName: "Sony WH-1000XM5 Headphones", Quantity: 1, UnitPrice: 398.00, Total: 398.00},
			},
			Subtotal:          797.98,
			Tax:               0.00,
			ShippingFee:       35.00,
			Total:             832.98,
			PaymentStatus:     "paid",
			FulfillmentStatus: "processing",
			ShippingAddress:   "Mannerheimintie 10, 00100 Helsinki, Finland",
			TrackingNumber:    "",
			Notes:             "International express shipping.",
			CreatedAt:         now.Add(-2 * time.Hour),
		},
		{
			ID:            "1004",
			OrderNo:       "ORD-8404",
			CustomerID:    "2",
			CustomerName:  "John Doe",
			CustomerEmail: "john.doe@acme.corp",
			Items: []OrderItem{
				{ProductID: "2", ProductName: "Keychron Q1 Pro Wireless Keyboard", Quantity: 1, UnitPrice: 199.99, Total: 199.99},
			},
			Subtotal:          199.99,
			Tax:               16.00,
			ShippingFee:       10.00,
			Total:             225.99,
			PaymentStatus:     "pending",
			FulfillmentStatus: "processing",
			ShippingAddress:   "100 Pike Street, Suite 400, Seattle, WA 98101",
			TrackingNumber:    "",
			Notes:             "Awaiting wire transfer confirmation.",
			CreatedAt:         now.Add(-30 * time.Minute),
		},
	}

	for _, o := range ordersList {
		orders[o.ID] = o
	}

	fmt.Println("🛍️  eCommerce seed data loaded: 6 products, 4 customers, 4 orders.")
}
