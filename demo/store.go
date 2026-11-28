package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// The demo keeps everything in memory behind one lock. resetStore (seed.go)
// replaces it all with fresh sample data every hour.
var (
	mu        sync.RWMutex
	products  = newTable[*Product]()
	variants  = newTable[*Variant]()
	barcodes  = newTable[*Barcode]()
	customers = newTable[*Customer]()
	orders    = newTable[*Order]()
	items     = newTable[*OrderItem]()
	cats      = newTable[*Category]()
)

// maxRows caps every table, so a public sandbox cannot be filled up between
// resets (imports and creates past it are refused).
const maxRows = 2000

var errDemoLimit = errors.New("the demo holds at most 2,000 records per table; it resets every hour")

// Product is a catalog item; its variants and barcodes are relation managers.
type Product struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	SKU      string  `json:"sku"`
	Brand    string  `json:"brand"`
	Category string  `json:"category"`
	Price    float64 `json:"price"`
	Cost     float64 `json:"cost"`
	Stock    int     `json:"stock"`
	Status   string  `json:"status"` // active, draft, archived
	Featured bool    `json:"featured"`
	Rating   float64 `json:"rating"`
	Summary  string  `json:"summary"`
	// Images are photo paths on the demo-public disk; SpecSheet is a PDF on
	// the private disk, SpecSheetName its original file name.
	Images        []string   `json:"images"`
	SpecSheet     string     `json:"spec_sheet"`
	SpecSheetName string     `json:"spec_sheet_name"`
	CreatedAt     time.Time  `json:"created_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

// Category groups products. Its id is its name, which products store.
type Category struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Products    int       `json:"products"`
	CreatedAt   time.Time `json:"created_at"`
}

// Variant is a size/colour option of a product.
type Variant struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	SKU       string  `json:"sku"`
	Price     float64 `json:"price"`
	Stock     int     `json:"stock"`
	Swatch    string  `json:"swatch"` // colour swatch image on demo-public
}

// Barcode is a scannable code attached to a product.
type Barcode struct {
	ID        string `json:"id"`
	ProductID string `json:"product_id"`
	Code      string `json:"code"`
	Kind      string `json:"kind"` // ean13, upca, qr
	Note      string `json:"note"`
}

// Customer buys things.
type Customer struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	City        string     `json:"city"`
	Country     string     `json:"country"`
	VIP         bool       `json:"vip"`
	OrdersCount int        `json:"orders_count"`
	TotalSpent  float64    `json:"total_spent"`
	Avatar      string     `json:"avatar"`
	CreatedAt   time.Time  `json:"created_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

// Order is a purchase; its lines are OrderItems.
type Order struct {
	ID           string    `json:"id"`
	Number       string    `json:"number"`
	CustomerID   string    `json:"customer_id"`
	CustomerName string    `json:"customer_name"`
	Status       string    `json:"status"`  // pending, paid, shipped, delivered, cancelled, refunded
	Payment      string    `json:"payment"` // card, paypal, bank
	Channel      string    `json:"channel"` // web, mobile, pos
	ItemsCount   int       `json:"items_count"`
	Subtotal     float64   `json:"subtotal"`
	Shipping     float64   `json:"shipping"`
	Total        float64   `json:"total"`
	Tracking     string    `json:"tracking"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
}

// OrderItem is one line of an order.
type OrderItem struct {
	ID        string  `json:"id"`
	OrderID   string  `json:"order_id"`
	Product   string  `json:"product"`
	SKU       string  `json:"sku"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Total     float64 `json:"total"`
}

// ---------------------------------------------------------------- tables

// table is a tiny in-memory table: rows by id, plus an auto-increment.
// Callers hold mu.
type table[T any] struct {
	rows map[string]T
	seq  int
}

func newTable[T any]() *table[T] { return &table[T]{rows: map[string]T{}} }

func (t *table[T]) nextID() string {
	t.seq++
	return strconv.Itoa(t.seq)
}

func (t *table[T]) get(id string) (T, error) {
	if r, ok := t.rows[id]; ok {
		return r, nil
	}
	var zero T
	return zero, fmt.Errorf("record %s not found", id)
}

func (t *table[T]) all() []T {
	out := make([]T, 0, len(t.rows))
	for _, r := range t.rows {
		out = append(out, r)
	}
	return out
}

// listQuery is what a table screen asks for.
type listQuery struct {
	search   string
	fields   []string // searched fields (json names)
	sortCol  string
	sortDir  string
	page     int
	perPage  int
	filters  map[string]string
	trashed  func(m map[string]any) bool // nil: no soft deletes
	defaults string                      // default sort column, newest first
}

// list filters, searches, sorts and pages rows through their JSON form, so
// one implementation serves every resource and relation manager.
func list[T any](rows []T, q listQuery) ([]T, int) {
	type row struct {
		rec T
		m   map[string]any
	}
	needle := strings.ToLower(strings.TrimSpace(q.search))
	var kept []row
	for _, r := range rows {
		m := toMap(r)
		if q.trashed != nil {
			gone := q.trashed(m)
			switch q.filters["trashed"] {
			case "only", "only_trashed":
				if !gone {
					continue
				}
			case "with", "with_trashed":
			default:
				if gone {
					continue
				}
			}
		}
		if !matchFilters(m, q.filters) {
			continue
		}
		if needle != "" && !matchSearch(m, q.fields, needle) {
			continue
		}
		kept = append(kept, row{r, m})
	}
	col, desc := q.sortCol, q.sortDir == "desc"
	if col == "" {
		col, desc = q.defaults, true
		if col == "" {
			col = "id"
		}
	}
	sort.SliceStable(kept, func(i, j int) bool {
		less := lessValue(kept[i].m[col], kept[j].m[col])
		if desc {
			return lessValue(kept[j].m[col], kept[i].m[col])
		}
		return less
	})
	total := len(kept)
	if q.perPage <= 0 {
		q.perPage = 25
	}
	if q.page <= 0 {
		q.page = 1
	}
	start := (q.page - 1) * q.perPage
	if start > total {
		start = total
	}
	end := min(start+q.perPage, total)
	out := make([]T, 0, end-start)
	for _, r := range kept[start:end] {
		out = append(out, r.rec)
	}
	return out, total
}

func matchFilters(m map[string]any, filters map[string]string) bool {
	for k, want := range filters {
		if want == "" || k == "trashed" {
			continue
		}
		v, ok := m[k]
		if !ok {
			continue // not a column of this table (e.g. a UI-only filter)
		}
		got := fmt.Sprint(v)
		if b, isBool := v.(bool); isBool {
			wb := want == "1" || want == "true" || want == "yes"
			if b != wb {
				return false
			}
			continue
		}
		// refined filters send several accepted values: "paid,shipped"
		match := false
		for _, w := range strings.Split(want, ",") {
			if strings.EqualFold(got, strings.TrimSpace(w)) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

func matchSearch(m map[string]any, fields []string, needle string) bool {
	for _, f := range fields {
		if strings.Contains(strings.ToLower(fmt.Sprint(m[f])), needle) {
			return true
		}
	}
	return false
}

func lessValue(a, b any) bool {
	fa, aNum := a.(float64)
	fb, bNum := b.(float64)
	if aNum && bNum {
		return fa < fb
	}
	sa, sb := fmt.Sprint(a), fmt.Sprint(b)
	// numeric ids sort as numbers
	if ia, err := strconv.Atoi(sa); err == nil {
		if ib, err := strconv.Atoi(sb); err == nil {
			return ia < ib
		}
	}
	return sa < sb
}

func toMap(v any) map[string]any {
	b, _ := json.Marshal(v)
	m := map[string]any{}
	_ = json.Unmarshal(b, &m)
	return m
}

// apply copies form data onto rec, converting each value to the type the
// field already has (forms send strings). Unknown keys are ignored.
func apply[T any](rec T, data map[string]any) error {
	cur := toMap(rec)
	for k, v := range data {
		old, ok := cur[k]
		if !ok || k == "id" {
			continue
		}
		cur[k] = convert(old, v)
	}
	b, err := json.Marshal(cur)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, rec)
}

func convert(old, v any) any {
	s := strings.TrimSpace(fmt.Sprint(v))
	if v == nil {
		s = ""
	}
	switch old.(type) {
	case float64:
		if f, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64); err == nil {
			return f
		}
		return old
	case bool:
		return s == "1" || s == "true" || s == "on" || s == "yes"
	case string, nil:
		switch v.(type) {
		case []string, []any, map[string]string:
			if old == nil { // empty list or map field (file fields)
				return v
			}
		}
		if v == nil {
			return ""
		}
		if _, isMap := v.(map[string]any); isMap {
			return old
		}
		return s
	default:
		return v
	}
}

func money(f float64) float64 { return float64(int(f*100+0.5)) / 100 }
