package main

import (
	"context"
	"fmt"
	"image/color"
	"math/rand"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/activitylog"
)

// resetEvery is how often the sandbox goes back to the sample data.
const resetEvery = time.Hour

// lastReset is when the data was last reseeded (guarded by mu).
var lastReset time.Time

// resetStore replaces all data with a fresh copy of the sample store. The
// numbers are generated from a fixed seed, but dates are relative to now, so
// the dashboard always shows the last 90 days.
func resetStore(now time.Time) {
	r := rand.New(rand.NewSource(42))

	mu.Lock()
	products, variants, barcodes = newTable[*Product](), newTable[*Variant](), newTable[*Barcode]()
	customers, orders, items = newTable[*Customer](), newTable[*Order](), newTable[*OrderItem]()
	cats = newTable[*Category]()
	seedCatalog(r, now)
	seedCustomers(r, now)
	seedOrders(r, now)
	lastReset = now
	mu.Unlock()

	seedActivity(now)
	resetRoles()
}

// nextReset is when the data will next be reset.
func nextReset() time.Time {
	mu.RLock()
	defer mu.RUnlock()
	return lastReset.Add(resetEvery)
}

type catalogItem struct {
	name, brand, category string
	price                 float64
	options               []string
}

var catalog = []catalogItem{
	{"Aurora Wireless Headphones", "Sonance", "Audio", 249, []string{"Midnight", "Sand", "Ocean"}},
	{"Pulse Earbuds Pro", "Sonance", "Audio", 179, []string{"Black", "White"}},
	{"Studio Monitor Speakers", "Sonance", "Audio", 329, []string{"Pair"}},
	{"Atlas 14 Laptop", "Northwind", "Computers", 1299, []string{"16 GB / 512 GB", "32 GB / 1 TB"}},
	{"Atlas 16 Laptop", "Northwind", "Computers", 1799, []string{"32 GB / 1 TB", "64 GB / 2 TB"}},
	{"Featherweight Keyboard", "Keyforge", "Accessories", 129, []string{"US", "UK", "DE"}},
	{"Precision Mouse", "Keyforge", "Accessories", 79, []string{"Graphite", "Frost"}},
	{"USB-C Dock 12-in-1", "Northwind", "Accessories", 199, []string{"Standard"}},
	{"4K Creator Monitor 27\"", "Lumen", "Displays", 649, []string{"Matte", "Glossy"}},
	{"Ultrawide Monitor 34\"", "Lumen", "Displays", 899, []string{"Standard"}},
	{"Trail Runner Shoes", "Stride", "Footwear", 139, []string{"40", "41", "42", "43", "44"}},
	{"City Sneakers", "Stride", "Footwear", 109, []string{"39", "40", "41", "42"}},
	{"Hiking Boots GTX", "Stride", "Footwear", 219, []string{"41", "42", "43", "44", "45"}},
	{"Merino Crew Sweater", "Fjord", "Apparel", 119, []string{"S", "M", "L", "XL"}},
	{"Waxed Canvas Jacket", "Fjord", "Apparel", 289, []string{"S", "M", "L"}},
	{"Everyday Tee 3-Pack", "Fjord", "Apparel", 49, []string{"S", "M", "L", "XL"}},
	{"Rain Shell", "Fjord", "Apparel", 179, []string{"M", "L", "XL"}},
	{"Commuter Backpack 22L", "Carryall", "Bags", 149, []string{"Black", "Olive"}},
	{"Weekender Duffel", "Carryall", "Bags", 199, []string{"Tan", "Black"}},
	{"Sling Bag", "Carryall", "Bags", 69, []string{"Black", "Sand"}},
	{"Pour-Over Coffee Set", "Brewhaus", "Kitchen", 89, []string{"Glass", "Ceramic"}},
	{"Espresso Grinder", "Brewhaus", "Kitchen", 239, []string{"Steel"}},
	{"Cast Iron Skillet 26cm", "Hearth", "Kitchen", 59, []string{"26 cm", "30 cm"}},
	{"Chef Knife 20cm", "Hearth", "Kitchen", 129, []string{"Standard"}},
	{"Smart Desk Lamp", "Lumen", "Home", 99, []string{"White", "Black"}},
	{"Linen Bedding Set", "Nest", "Home", 189, []string{"Single", "Double", "King"}},
	{"Aroma Diffuser", "Nest", "Home", 45, []string{"Oak", "Stone"}},
	{"Yoga Mat Pro", "Flow", "Fitness", 89, []string{"Moss", "Clay", "Slate"}},
	{"Adjustable Dumbbells", "Flow", "Fitness", 349, []string{"24 kg", "40 kg"}},
	{"Fitness Tracker Band", "Pulse", "Wearables", 129, []string{"Black", "Coral"}},
	{"Smartwatch Series 5", "Pulse", "Wearables", 399, []string{"41 mm", "45 mm"}},
	{"Action Camera 4K", "Vista", "Cameras", 349, []string{"Standard", "Creator Kit"}},
	{"Mirrorless Camera Body", "Vista", "Cameras", 1499, []string{"Body", "Kit 24-70"}},
	{"Travel Tripod", "Vista", "Cameras", 159, []string{"Carbon", "Aluminium"}},
	{"E-Reader Paperlight", "Folio", "Electronics", 149, []string{"8 GB", "32 GB"}},
	{"Portable Charger 20k", "Northwind", "Electronics", 59, []string{"Black", "White"}},
	{"Bluetooth Speaker Go", "Sonance", "Audio", 89, []string{"Red", "Blue", "Black"}},
	{"Noise Machine", "Nest", "Home", 69, []string{"White"}},
	{"Insulated Bottle 750ml", "Flow", "Fitness", 35, []string{"Moss", "Clay", "Slate", "Sand"}},
	{"Plant Pot Trio", "Nest", "Home", 55, []string{"Terracotta", "White"}},
}

var (
	firstNames = []string{"Ava", "Liam", "Maya", "Noah", "Zara", "Omar", "Lena", "Yusuf", "Chloe", "Hiro", "Sofia", "Mateo", "Amara", "Ethan", "Leila", "Ravi", "Nina", "Jonas", "Aisha", "Lucas", "Mei", "Karim", "Elena", "Tariq", "Isla", "Diego", "Fatima", "Oscar", "Priya", "Felix"}
	lastNames  = []string{"Hassan", "Silva", "Kowalski", "Nakamura", "Okafor", "Müller", "Rossi", "Haddad", "Novak", "Andersen", "Patel", "Garcia", "Kim", "Dubois", "Rahman", "Fischer", "Costa", "Ali", "Jensen", "Moreau"}
	places     = [][2]string{{"Lisbon", "Portugal"}, {"Berlin", "Germany"}, {"Dubai", "UAE"}, {"Toronto", "Canada"}, {"Austin", "USA"}, {"Lyon", "France"}, {"Osaka", "Japan"}, {"Cape Town", "South Africa"}, {"Colombo", "Sri Lanka"}, {"Madrid", "Spain"}, {"Amsterdam", "Netherlands"}, {"Melbourne", "Australia"}, {"Istanbul", "Turkey"}, {"Doha", "Qatar"}, {"Seattle", "USA"}}
)

func seedCatalog(r *rand.Rand, now time.Time) {
	for _, c := range catalog {
		if _, ok := cats.rows[c.category]; !ok {
			cats.rows[c.category] = &Category{ID: c.category, Name: c.category,
				Description: c.category + " products", CreatedAt: now.AddDate(-1, 0, 0)}
		}
	}
	for i, c := range catalog {
		id := products.nextID()
		status := "active"
		switch {
		case i%13 == 7:
			status = "draft"
		case i%17 == 11:
			status = "archived"
		}
		prefix := strings.ToUpper(c.brand[:3])
		p := &Product{
			ID: id, Name: c.name, Brand: c.brand, Category: c.category,
			SKU:       fmt.Sprintf("%s-%04d", prefix, 1000+i*7),
			Price:     c.price,
			Cost:      money(c.price * (0.45 + r.Float64()*0.2)),
			Status:    status,
			Featured:  i%6 == 0,
			Rating:    float64(35+r.Intn(16)) / 10,
			Summary:   fmt.Sprintf("%s by %s. A customer favourite in %s.", c.name, c.brand, strings.ToLower(c.category)),
			CreatedAt: now.Add(-time.Duration(120+r.Intn(240)) * 24 * time.Hour),
		}
		stock := 0
		for j, opt := range c.options {
			vs := r.Intn(40)
			if (i+j)%9 == 4 {
				vs = r.Intn(4) // a few low-stock variants for the dashboard
			}
			stock += vs
			vid := variants.nextID()
			variants.rows[vid] = &Variant{
				ID: vid, ProductID: id, Name: opt,
				SKU:   fmt.Sprintf("%s-%02d", p.SKU, j+1),
				Price: p.Price + float64(j)*map[bool]float64{true: 100, false: 0}[c.price > 500],
				Stock: vs,
			}
		}
		p.Stock = stock
		p.Images = []string{}
		if i < 8 { // sample photos for the first products
			hues := []color.RGBA{{79, 70, 229, 255}, {14, 165, 233, 255}, {16, 185, 129, 255}, {245, 158, 11, 255},
				{239, 68, 68, 255}, {168, 85, 247, 255}, {236, 72, 153, 255}, {20, 184, 166, 255}}
			if photo := productPhoto(i+1, hues[i]); photo != "" {
				p.Images = []string{photo}
			}
		}
		if i == 9 {
			p.DeletedAt = ptr(now.Add(-36 * time.Hour)) // one product in the trash
		}
		products.rows[id] = p

		for k := 0; k < 1+i%2; k++ {
			bid := barcodes.nextID()
			kind, code := "ean13", ean13(r)
			if k == 1 {
				kind, code = "upca", upcA(r)
			}
			barcodes.rows[bid] = &Barcode{ID: bid, ProductID: id, Code: code, Kind: kind, Note: map[int]string{0: "Retail box", 1: "US retail"}[k]}
		}
	}
}

func seedCustomers(r *rand.Rand, now time.Time) {
	for i := 0; i < 120; i++ {
		id := customers.nextID()
		first, last := firstNames[r.Intn(len(firstNames))], lastNames[r.Intn(len(lastNames))]
		pl := places[r.Intn(len(places))]
		customers.rows[id] = &Customer{
			ID: id, Name: first + " " + last,
			Email: strings.ToLower(fmt.Sprintf("%s.%s%d@example.com", first, strings.NewReplacer("ü", "u").Replace(last), i)),
			Phone: fmt.Sprintf("+1 555 %03d %04d", r.Intn(1000), r.Intn(10000)),
			City:  pl[0], Country: pl[1],
			CreatedAt: now.Add(-time.Duration(r.Intn(400)) * 24 * time.Hour),
		}
	}
}

func seedOrders(r *rand.Rand, now time.Time) {
	catalogIDs := make([]string, 0, len(products.rows))
	for id, p := range products.rows {
		if p.Status == "active" && p.DeletedAt == nil {
			catalogIDs = append(catalogIDs, id)
		}
	}
	sortStrings(catalogIDs)
	payments := []string{"card", "card", "card", "paypal", "bank"}
	channels := []string{"web", "web", "mobile", "mobile", "pos"}
	for day := 89; day >= 0; day-- {
		// more orders on recent days and weekends
		n := 4 + r.Intn(5) + (89-day)/15
		if wd := now.AddDate(0, 0, -day).Weekday(); wd == time.Saturday || wd == time.Sunday {
			n += 3
		}
		for k := 0; k < n; k++ {
			cust := customers.rows[fmt.Sprint(1+r.Intn(len(customers.rows)))]
			created := time.Date(now.Year(), now.Month(), now.Day(), 8+r.Intn(14), r.Intn(60), 0, 0, now.Location()).AddDate(0, 0, -day)
			if created.After(now) {
				created = now.Add(-time.Duration(1+r.Intn(50)) * time.Minute)
			}
			id := orders.nextID()
			o := &Order{
				ID: id, Number: fmt.Sprintf("ORD-%05d", 10000+len(orders.rows)+1),
				CustomerID: cust.ID, CustomerName: cust.Name,
				Payment: payments[r.Intn(len(payments))], Channel: channels[r.Intn(len(channels))],
				CreatedAt: created,
			}
			lines := 1 + r.Intn(3)
			for l := 0; l < lines; l++ {
				p := products.rows[catalogIDs[r.Intn(len(catalogIDs))]]
				q := 1 + r.Intn(2)
				iid := items.nextID()
				items.rows[iid] = &OrderItem{ID: iid, OrderID: id, Product: p.Name, SKU: p.SKU, Quantity: q, UnitPrice: p.Price, Total: money(p.Price * float64(q))}
				o.Subtotal += p.Price * float64(q)
				o.ItemsCount += q
			}
			o.Subtotal = money(o.Subtotal)
			if o.Subtotal < 100 {
				o.Shipping = 9.90
			}
			o.Total = money(o.Subtotal + o.Shipping)
			age := now.Sub(created)
			switch {
			case r.Intn(40) == 0:
				o.Status = "cancelled"
			case r.Intn(60) == 0:
				o.Status = "refunded"
			case age < 6*time.Hour:
				o.Status = []string{"pending", "paid"}[r.Intn(2)]
			case age < 48*time.Hour:
				o.Status = []string{"paid", "shipped"}[r.Intn(2)]
			case age < 5*24*time.Hour:
				o.Status = "shipped"
			default:
				o.Status = "delivered"
			}
			if o.Status == "shipped" || o.Status == "delivered" {
				o.Tracking = fmt.Sprintf("1Z%06d%04d", r.Intn(1000000), r.Intn(10000))
			}
			orders.rows[id] = o
			if o.Status != "cancelled" && o.Status != "refunded" {
				cust.OrdersCount++
				cust.TotalSpent = money(cust.TotalSpent + o.Total)
			}
		}
	}
	for _, c := range customers.rows {
		c.VIP = c.TotalSpent > 2500
	}
}

// seedActivity writes an audit trail for the newest orders, so record pages
// show a History timeline right away.
func seedActivity(now time.Time) {
	store := activitylog.GetDefaultStore()
	ctx := context.Background()
	_ = store.Clear(ctx)

	mu.RLock()
	recent, _ := list(orders.all(), listQuery{defaults: "created_at", perPage: 30})
	mu.RUnlock()
	for _, o := range recent {
		at := o.CreatedAt
		_, _ = activitylog.Log("Order "+o.Number+" placed").Subject("Order", o.ID).
			Causer("Customer", o.CustomerID, o.CustomerName).Event("created").
			New(map[string]any{"number": o.Number, "total": o.Total, "status": "pending", "channel": o.Channel}).
			CreatedAt(at).Save(ctx)
		if o.Status == "pending" {
			continue
		}
		at = at.Add(4 * time.Minute)
		_, _ = activitylog.Log(fmt.Sprintf("Payment of $%.2f captured (%s)", o.Total, o.Payment)).Subject("Order", o.ID).
			Causer("System", "payments", "Payments").Event("updated").
			WithDiff(map[string]any{"status": "pending"}, map[string]any{"status": "paid"}).
			CreatedAt(at).Save(ctx)
		if o.Status == "paid" {
			continue
		}
		at = at.Add(20 * time.Hour)
		if at.After(now) {
			at = now.Add(-10 * time.Minute)
		}
		switch o.Status {
		case "shipped", "delivered":
			_, _ = activitylog.Log("Packed and shipped").Subject("Order", o.ID).
				Causer("User", "warehouse@example.com", "Sam (Warehouse)").Event("updated").
				WithDiff(map[string]any{"status": "paid", "tracking": ""}, map[string]any{"status": "shipped", "tracking": o.Tracking}).
				CreatedAt(at).Save(ctx)
		case "cancelled", "refunded":
			_, _ = activitylog.Log("Order "+o.Status).Subject("Order", o.ID).
				Causer("User", "admin@example.com", "Alex (Admin)").Event("updated").
				WithDiff(map[string]any{"status": "paid"}, map[string]any{"status": o.Status}).
				CreatedAt(at).Save(ctx)
		}
		if o.Status == "delivered" {
			at = at.Add(48 * time.Hour)
			if at.After(now) {
				at = now.Add(-5 * time.Minute)
			}
			_, _ = activitylog.Log("Delivered").Subject("Order", o.ID).
				Causer("System", "carrier", "Carrier webhook").Event("updated").
				WithDiff(map[string]any{"status": "shipped"}, map[string]any{"status": "delivered"}).
				CreatedAt(at).Save(ctx)
		}
	}
}

func ean13(r *rand.Rand) string {
	d := make([]int, 12)
	d[0], d[1], d[2] = 4, 0, 0
	for i := 3; i < 12; i++ {
		d[i] = r.Intn(10)
	}
	sum := 0
	for i, v := range d {
		if i%2 == 1 {
			sum += 3 * v
		} else {
			sum += v
		}
	}
	return digits(d) + fmt.Sprint((10-sum%10)%10)
}

func upcA(r *rand.Rand) string {
	d := make([]int, 11)
	d[0] = 0
	for i := 1; i < 11; i++ {
		d[i] = r.Intn(10)
	}
	sum := 0
	for i, v := range d {
		if i%2 == 0 {
			sum += 3 * v
		} else {
			sum += v
		}
	}
	return digits(d) + fmt.Sprint((10-sum%10)%10)
}

func digits(d []int) string {
	var b strings.Builder
	for _, v := range d {
		b.WriteByte(byte('0' + v))
	}
	return b.String()
}

func ptr[T any](v T) *T { return &v }

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && lessValue(s[j], s[j-1]); j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
