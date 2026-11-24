package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/golaravel/gofilament/pkg/activitylog"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

// Every panel gets its own resource values (web at /admin, Flutter at /app);
// they share the data through the package tables.

var (
	productStatus = map[string]string{"active": "Active", "draft": "Draft", "archived": "Archived"}
	orderStatus   = map[string]string{"pending": "Pending", "paid": "Paid", "shipped": "Shipped", "delivered": "Delivered", "cancelled": "Cancelled", "refunded": "Refunded"}
	payments      = map[string]string{"card": "Card", "paypal": "PayPal", "bank": "Bank transfer"}
	channels      = map[string]string{"web": "Web store", "mobile": "Mobile app", "pos": "In store"}
	barcodeKinds  = map[string]string{"ean13": "EAN-13", "upca": "UPC-A", "qr": "QR code"}
	categories    = func() map[string]string {
		m := map[string]string{}
		for _, c := range catalog {
			m[c.category] = c.category
		}
		return m
	}()
)

func trashed(m map[string]any) bool { return m["deleted_at"] != nil }

// ---------------------------------------------------------------- products

func productResource() *filament.BaseResource[*Product] {
	res := filament.NewResource[*Product]("products").
		ModelName("Product").
		NavigationGroup("Catalog").
		NavigationIcon("heroicon-shopping-bag").
		NavigationSort(1).
		SoftDeletes(true).
		IsTrashed(func(p *Product) bool { return p != nil && p.DeletedAt != nil }).
		RecordID(func(p *Product) string { return p.ID }).
		GloballySearchable().
		GlobalSearchAttributes("name", "sku", "brand").
		GlobalSearchResultTitle(func(p *Product) string { return p.Name }).
		GlobalSearchResultDetails(func(p *Product) map[string]string {
			return map[string]string{"SKU": p.SKU, "Stock": fmt.Sprint(p.Stock)}
		}).
		NavigationBadge(func(ctx *filament.Context) string {
			mu.RLock()
			defer mu.RUnlock()
			n := 0
			for _, p := range products.rows {
				if p.DeletedAt == nil && p.Status == "active" && p.Stock < 10 {
					n++
				}
			}
			if n == 0 {
				return ""
			}
			return fmt.Sprint(n)
		})

	res.Table(func(t *tables.Table[*Product]) *tables.Table[*Product] {
		return t.Toggleable(true).
			Columns(
				tables.TextColumn[*Product]("name").Label("Product").Weight("bold").Searchable().Sortable(),
				tables.TextColumn[*Product]("sku").Label("SKU").Copyable().Searchable(),
				tables.BadgeColumn[*Product]("category").Label("Category").Sortable(),
				tables.TextColumn[*Product]("brand").Label("Brand").Searchable().ToggledHiddenByDefault(true),
				tables.TextColumn[*Product]("price").Label("Price").Money("USD").Sortable(),
				tables.TextColumn[*Product]("stock").Label("Stock").Sortable(),
				tables.BadgeColumn[*Product]("status").Label("Status").Labels(productStatus).
					Colors(map[string]string{"active": "success", "draft": "warning", "archived": "gray"}),
				tables.ToggleColumn[*Product]("featured").Label("Featured"),
				tables.TextColumn[*Product]("rating").Label("Rating").Sortable().ToggledHiddenByDefault(true),
			).
			Filters(
				tables.SelectFilter("status").Label("Status").Options(productStatus),
				tables.SelectFilter("category").Label("Category").Options(categories),
				tables.TernaryFilter("featured").Label("Featured").TrueLabel("Featured only").FalseLabel("Not featured"),
				tables.TrashedFilter(),
			).
			HeaderActions(tables.ImportAction[*Product](), tables.ExportAction[*Product]()).
			Actions(tables.ViewAction[*Product](), tables.EditAction[*Product](), tables.DeleteAction[*Product](),
				tables.RestoreAction[*Product](), tables.ForceDeleteAction[*Product]()).
			BulkActions(tables.EditBulkAction(), tables.ExportBulkAction(), tables.DeleteBulkAction())
	})

	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Section("Details").Schema(
				forms.Grid(2).Schema(
					forms.TextInput("name").Label("Name").Required(),
					forms.TextInput("sku").Label("SKU").Required(),
					forms.TextInput("brand").Label("Brand"),
					forms.Select("category").Label("Category").Options(categories).Required(),
				),
				forms.Textarea("summary").Label("Summary"),
			),
			forms.Section("Pricing and stock").Schema(
				forms.Grid(3).Schema(
					forms.TextInput("price").Label("Price").Numeric().Prefix("$").Required(),
					forms.TextInput("cost").Label("Cost").Numeric().Prefix("$"),
					forms.TextInput("stock").Label("Stock").Numeric(),
				),
				forms.Grid(2).Schema(
					forms.Select("status").Label("Status").Options(productStatus).Default("draft"),
					forms.Toggle("featured").Label("Featured on the storefront"),
				),
			),
		)
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Product, int, error) {
		mu.RLock()
		defer mu.RUnlock()
		rows, total := list(products.all(), listQuery{search: search, fields: []string{"name", "sku", "brand", "category"},
			sortCol: sortCol, sortDir: sortDir, page: page, perPage: perPage, filters: filters, trashed: trashed, defaults: "created_at"})
		return rows, total, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*Product, error) {
		mu.RLock()
		defer mu.RUnlock()
		return products.get(id)
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		if len(products.rows) >= maxRows {
			return errDemoLimit
		}
		p := &Product{Status: "draft", CreatedAt: time.Now()}
		if err := apply(p, data); err != nil {
			return err
		}
		p.ID = products.nextID()
		data["id"] = p.ID
		products.rows[p.ID] = p
		return nil
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		p, err := products.get(id)
		if err != nil {
			return err
		}
		return apply(p, data)
	})
	res.OnCellUpdate(func(ctx *filament.Context, id, column string, value any) error {
		mu.Lock()
		defer mu.Unlock()
		p, err := products.get(id)
		if err != nil {
			return err
		}
		return apply(p, map[string]any{column: value})
	})
	res.OnDelete(func(ctx *filament.Context, id string) error {
		mu.Lock()
		defer mu.Unlock()
		if p, err := products.get(id); err == nil {
			p.DeletedAt = ptr(time.Now())
		}
		return nil
	})
	res.OnRestore(func(ctx *filament.Context, id string) error {
		mu.Lock()
		defer mu.Unlock()
		if p, err := products.get(id); err == nil {
			p.DeletedAt = nil
		}
		return nil
	})
	res.OnForceDelete(func(ctx *filament.Context, id string) error {
		mu.Lock()
		defer mu.Unlock()
		delete(products.rows, id)
		for vid, v := range variants.rows {
			if v.ProductID == id {
				delete(variants.rows, vid)
			}
		}
		for bid, b := range barcodes.rows {
			if b.ProductID == id {
				delete(barcodes.rows, bid)
			}
		}
		return nil
	})

	res.BulkEditable("status", "category", "featured")
	res.Relations(variantsManager(), barcodesManager())
	return res
}

func variantsManager() filament.RelationManager {
	rm := filament.NewRelationManager[*Variant]("variants", "Variants").
		Icon("heroicon-squares-2x2").
		RecordTitleAttribute("name").
		RecordID(func(v *Variant) string { return v.ID })
	rm.Table(func(t *tables.Table[*Variant]) *tables.Table[*Variant] {
		return t.Columns(
			tables.TextColumn[*Variant]("name").Label("Option").Weight("bold").Searchable().Sortable(),
			tables.TextColumn[*Variant]("sku").Label("SKU").Copyable(),
			tables.TextColumn[*Variant]("price").Label("Price").Money("USD").Sortable(),
			tables.TextColumn[*Variant]("stock").Label("Stock").Sortable(),
		)
	})
	rm.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(forms.Grid(2).Schema(
			forms.TextInput("name").Label("Option").Required(),
			forms.TextInput("sku").Label("SKU").Required(),
			forms.TextInput("price").Label("Price").Numeric().Prefix("$").Required(),
			forms.TextInput("stock").Label("Stock").Numeric(),
		))
	})
	rm.OnList(func(ctx *filament.Context, parentID, search, sortCol, dir string, page, perPage int) ([]*Variant, int, error) {
		mu.RLock()
		defer mu.RUnlock()
		var rows []*Variant
		for _, v := range variants.rows {
			if v.ProductID == parentID {
				rows = append(rows, v)
			}
		}
		out, total := list(rows, listQuery{search: search, fields: []string{"name", "sku"}, sortCol: sortCol, sortDir: dir, page: page, perPage: perPage})
		return out, total, nil
	})
	rm.OnFind(func(ctx *filament.Context, parentID, childID string) (*Variant, error) {
		mu.RLock()
		defer mu.RUnlock()
		v, err := variants.get(childID)
		if err != nil || v.ProductID != parentID {
			return nil, fmt.Errorf("variant %s not found", childID)
		}
		return v, nil
	})
	rm.OnCreate(func(ctx *filament.Context, parentID string, data map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		if len(variants.rows) >= maxRows {
			return errDemoLimit
		}
		v := &Variant{ProductID: parentID}
		if err := apply(v, data); err != nil {
			return err
		}
		v.ID = variants.nextID()
		variants.rows[v.ID] = v
		syncStock(parentID)
		return nil
	})
	rm.OnUpdate(func(ctx *filament.Context, parentID, childID string, data map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		v, err := variants.get(childID)
		if err != nil || v.ProductID != parentID {
			return fmt.Errorf("variant %s not found", childID)
		}
		if err := apply(v, data); err != nil {
			return err
		}
		syncStock(parentID)
		return nil
	})
	rm.OnDelete(func(ctx *filament.Context, parentID, childID string) error {
		mu.Lock()
		defer mu.Unlock()
		if v, err := variants.get(childID); err == nil && v.ProductID == parentID {
			delete(variants.rows, childID)
			syncStock(parentID)
		}
		return nil
	})
	return rm
}

// syncStock keeps a product's stock equal to its variants' total. Callers hold mu.
func syncStock(productID string) {
	p, err := products.get(productID)
	if err != nil {
		return
	}
	n := 0
	for _, v := range variants.rows {
		if v.ProductID == productID {
			n += v.Stock
		}
	}
	p.Stock = n
}

func barcodesManager() filament.RelationManager {
	rm := filament.NewRelationManager[*Barcode]("barcodes", "Barcodes").
		Icon("heroicon-qr-code").
		RecordTitleAttribute("code").
		RecordID(func(b *Barcode) string { return b.ID })
	rm.Table(func(t *tables.Table[*Barcode]) *tables.Table[*Barcode] {
		return t.Columns(
			tables.TextColumn[*Barcode]("code").Label("Barcode").Weight("bold").Copyable().Searchable(),
			tables.BadgeColumn[*Barcode]("kind").Label("Type").Labels(barcodeKinds),
			tables.TextColumn[*Barcode]("note").Label("Note"),
		)
	})
	rm.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(forms.Grid(2).Schema(
			forms.TextInput("code").Label("Barcode").Required(),
			forms.Select("kind").Label("Type").Options(barcodeKinds).Default("ean13").Required(),
			forms.TextInput("note").Label("Note"),
		))
	})
	rm.OnList(func(ctx *filament.Context, parentID, search, sortCol, dir string, page, perPage int) ([]*Barcode, int, error) {
		mu.RLock()
		defer mu.RUnlock()
		var rows []*Barcode
		for _, b := range barcodes.rows {
			if b.ProductID == parentID {
				rows = append(rows, b)
			}
		}
		out, total := list(rows, listQuery{search: search, fields: []string{"code", "note"}, sortCol: sortCol, sortDir: dir, page: page, perPage: perPage})
		return out, total, nil
	})
	rm.OnFind(func(ctx *filament.Context, parentID, childID string) (*Barcode, error) {
		mu.RLock()
		defer mu.RUnlock()
		b, err := barcodes.get(childID)
		if err != nil || b.ProductID != parentID {
			return nil, fmt.Errorf("barcode %s not found", childID)
		}
		return b, nil
	})
	rm.OnCreate(func(ctx *filament.Context, parentID string, data map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		if len(barcodes.rows) >= maxRows {
			return errDemoLimit
		}
		b := &Barcode{ProductID: parentID, Kind: "ean13"}
		if err := apply(b, data); err != nil {
			return err
		}
		b.ID = barcodes.nextID()
		barcodes.rows[b.ID] = b
		return nil
	})
	rm.OnUpdate(func(ctx *filament.Context, parentID, childID string, data map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		b, err := barcodes.get(childID)
		if err != nil || b.ProductID != parentID {
			return fmt.Errorf("barcode %s not found", childID)
		}
		return apply(b, data)
	})
	rm.OnDelete(func(ctx *filament.Context, parentID, childID string) error {
		mu.Lock()
		defer mu.Unlock()
		if b, err := barcodes.get(childID); err == nil && b.ProductID == parentID {
			delete(barcodes.rows, childID)
		}
		return nil
	})
	return rm
}

// ---------------------------------------------------------------- orders

func orderResource() *filament.BaseResource[*Order] {
	res := filament.NewResource[*Order]("orders").
		ModelName("Order").
		NavigationGroup("Sales").
		NavigationIcon("heroicon-shopping-cart").
		NavigationSort(1).
		RecordID(func(o *Order) string { return o.ID }).
		GloballySearchable().
		GlobalSearchAttributes("number", "customer_name", "tracking").
		GlobalSearchResultTitle(func(o *Order) string { return o.Number + " · " + o.CustomerName }).
		GlobalSearchResultDetails(func(o *Order) map[string]string {
			return map[string]string{"Status": orderStatus[o.Status], "Total": fmt.Sprintf("$%.2f", o.Total)}
		}).
		NavigationBadge(func(ctx *filament.Context) string {
			mu.RLock()
			defer mu.RUnlock()
			n := 0
			for _, o := range orders.rows {
				if o.Status == "paid" {
					n++
				}
			}
			if n == 0 {
				return ""
			}
			return fmt.Sprint(n)
		})

	res.Table(func(t *tables.Table[*Order]) *tables.Table[*Order] {
		return t.Toggleable(true).
			Columns(
				tables.TextColumn[*Order]("number").Label("Order").Weight("bold").Copyable().Searchable().Sortable(),
				tables.TextColumn[*Order]("customer_name").Label("Customer").Searchable().Sortable(),
				tables.BadgeColumn[*Order]("status").Label("Status").Labels(orderStatus).
					Colors(map[string]string{"pending": "gray", "paid": "warning", "shipped": "info", "delivered": "success", "cancelled": "danger", "refunded": "danger"}),
				tables.BadgeColumn[*Order]("payment").Label("Payment").Labels(payments).ToggledHiddenByDefault(true),
				tables.BadgeColumn[*Order]("channel").Label("Channel").Labels(channels),
				tables.TextColumn[*Order]("items_count").Label("Items").Sortable(),
				tables.TextColumn[*Order]("total").Label("Total").Money("USD").Sortable(),
				tables.TextColumn[*Order]("tracking").Label("Tracking").Copyable().ToggledHiddenByDefault(true),
				tables.TextColumn[*Order]("created_at").Label("Placed").DateTime("Jan 2, 15:04").Sortable(),
			).
			Filters(
				tables.SelectFilter("status").Label("Status").Options(orderStatus),
				tables.SelectFilter("payment").Label("Payment").Options(payments),
				tables.SelectFilter("channel").Label("Channel").Options(channels),
			).
			HeaderActions(tables.ExportAction[*Order]()).
			Actions(
				tables.ViewAction[*Order](),
				tables.Action[*Order]("ship").Label("Ship").Icon("heroicon-truck").Color("success").RequiresConfirmation().
					Visible(func(o *Order) bool { return o.Status == "paid" }).
					Action(func(o *Order) error { shipOrders([]string{o.ID}, "Ship action"); return nil }),
				tables.EditAction[*Order](),
			).
			BulkActions(tables.EditBulkAction(), tables.ExportBulkAction())
	})

	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Section("Fulfilment").Schema(
				forms.Grid(2).Schema(
					forms.Select("status").Label("Status").Options(orderStatus).Required(),
					forms.TextInput("tracking").Label("Tracking number"),
				),
				forms.Textarea("notes").Label("Internal notes"),
			),
		)
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Order, int, error) {
		mu.RLock()
		defer mu.RUnlock()
		rows, total := list(orders.all(), listQuery{search: search, fields: []string{"number", "customer_name", "tracking"},
			sortCol: sortCol, sortDir: sortDir, page: page, perPage: perPage, filters: filters, defaults: "created_at"})
		return rows, total, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*Order, error) {
		mu.RLock()
		defer mu.RUnlock()
		return orders.get(id)
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		o, err := orders.get(id)
		if err != nil {
			return err
		}
		// Only fulfilment fields are editable here.
		return apply(o, map[string]any{"status": data["status"], "tracking": data["tracking"], "notes": data["notes"]})
	})
	res.BulkEditable("status")
	res.Relations(orderItemsManager())
	return res
}

// shipOrders marks paid orders as shipped and logs it to the audit trail.
func shipOrders(ids []string, via string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	type shipped struct{ id, number, tracking string }
	var done []shipped
	mu.Lock()
	for _, id := range ids {
		o, err := orders.get(id)
		if err != nil || (o.Status != "paid" && o.Status != "pending") {
			continue
		}
		o.Status = "shipped"
		o.Tracking = fmt.Sprintf("1Z%06d%04d", r.Intn(1000000), r.Intn(10000))
		done = append(done, shipped{o.ID, o.Number, o.Tracking})
	}
	mu.Unlock()
	for _, s := range done {
		_, _ = activitylog.Log("Shipped ("+via+")").Subject("Order", s.id).Event("updated").
			WithDiff(map[string]any{"tracking": ""}, map[string]any{"status": "shipped", "tracking": s.tracking}).
			Save(context.Background())
	}
}

func orderItemsManager() filament.RelationManager {
	rm := filament.NewRelationManager[*OrderItem]("items", "Items").
		Icon("heroicon-list-bullet").
		RecordTitleAttribute("product").
		RecordID(func(i *OrderItem) string { return i.ID })
	rm.Table(func(t *tables.Table[*OrderItem]) *tables.Table[*OrderItem] {
		return t.Columns(
			tables.TextColumn[*OrderItem]("product").Label("Product").Weight("bold"),
			tables.TextColumn[*OrderItem]("sku").Label("SKU").Copyable(),
			tables.TextColumn[*OrderItem]("quantity").Label("Qty"),
			tables.TextColumn[*OrderItem]("unit_price").Label("Unit price").Money("USD"),
			tables.TextColumn[*OrderItem]("total").Label("Total").Money("USD"),
		)
	})
	rm.OnList(func(ctx *filament.Context, parentID, search, sortCol, dir string, page, perPage int) ([]*OrderItem, int, error) {
		mu.RLock()
		defer mu.RUnlock()
		var rows []*OrderItem
		for _, it := range items.rows {
			if it.OrderID == parentID {
				rows = append(rows, it)
			}
		}
		out, total := list(rows, listQuery{search: search, fields: []string{"product", "sku"}, sortCol: sortCol, sortDir: dir, page: page, perPage: perPage})
		return out, total, nil
	})
	return rm
}

// ---------------------------------------------------------------- customers

func customerResource() *filament.BaseResource[*Customer] {
	res := filament.NewResource[*Customer]("customers").
		ModelName("Customer").
		NavigationGroup("Sales").
		NavigationIcon("heroicon-users").
		NavigationSort(2).
		SoftDeletes(true).
		IsTrashed(func(c *Customer) bool { return c != nil && c.DeletedAt != nil }).
		RecordID(func(c *Customer) string { return c.ID }).
		GloballySearchable().
		GlobalSearchAttributes("name", "email", "city").
		GlobalSearchResultTitle(func(c *Customer) string { return c.Name }).
		GlobalSearchResultDetails(func(c *Customer) map[string]string {
			return map[string]string{"Email": c.Email, "City": c.City}
		})

	res.Table(func(t *tables.Table[*Customer]) *tables.Table[*Customer] {
		return t.Toggleable(true).
			Columns(
				tables.TextColumn[*Customer]("name").Label("Name").Weight("bold").Searchable().Sortable(),
				tables.TextColumn[*Customer]("email").Label("Email").Copyable().Searchable(),
				tables.TextColumn[*Customer]("city").Label("City").Sortable(),
				tables.TextColumn[*Customer]("country").Label("Country").Sortable(),
				tables.TextColumn[*Customer]("orders_count").Label("Orders").Sortable(),
				tables.TextColumn[*Customer]("total_spent").Label("Lifetime value").Money("USD").Sortable(),
				tables.ToggleColumn[*Customer]("vip").Label("VIP"),
				tables.TextColumn[*Customer]("phone").Label("Phone").ToggledHiddenByDefault(true),
			).
			Filters(
				tables.TernaryFilter("vip").Label("VIP").TrueLabel("VIP only").FalseLabel("Regular"),
				tables.SelectFilter("country").Label("Country").Options(countries()),
				tables.TrashedFilter(),
			).
			HeaderActions(tables.ImportAction[*Customer](), tables.ExportAction[*Customer]()).
			Actions(tables.ViewAction[*Customer](), tables.EditAction[*Customer](), tables.DeleteAction[*Customer](), tables.RestoreAction[*Customer]()).
			BulkActions(tables.ExportBulkAction(), tables.DeleteBulkAction())
	})
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(forms.Section("Customer").Schema(forms.Grid(2).Schema(
			forms.TextInput("name").Label("Name").Required(),
			forms.TextInput("email").Label("Email").Email().Required(),
			forms.TextInput("phone").Label("Phone"),
			forms.TextInput("city").Label("City"),
			forms.TextInput("country").Label("Country"),
			forms.Toggle("vip").Label("VIP"),
		)))
	})
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Customer, int, error) {
		mu.RLock()
		defer mu.RUnlock()
		rows, total := list(customers.all(), listQuery{search: search, fields: []string{"name", "email", "city", "country"},
			sortCol: sortCol, sortDir: sortDir, page: page, perPage: perPage, filters: filters, trashed: trashed, defaults: "total_spent"})
		return rows, total, nil
	})
	res.OnFind(func(ctx *filament.Context, id string) (*Customer, error) {
		mu.RLock()
		defer mu.RUnlock()
		return customers.get(id)
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		if len(customers.rows) >= maxRows {
			return errDemoLimit
		}
		c := &Customer{CreatedAt: time.Now()}
		if err := apply(c, data); err != nil {
			return err
		}
		c.ID = customers.nextID()
		data["id"] = c.ID
		customers.rows[c.ID] = c
		return nil
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		mu.Lock()
		defer mu.Unlock()
		c, err := customers.get(id)
		if err != nil {
			return err
		}
		return apply(c, data)
	})
	res.OnCellUpdate(func(ctx *filament.Context, id, column string, value any) error {
		mu.Lock()
		defer mu.Unlock()
		c, err := customers.get(id)
		if err != nil {
			return err
		}
		return apply(c, map[string]any{column: value})
	})
	res.OnDelete(func(ctx *filament.Context, id string) error {
		mu.Lock()
		defer mu.Unlock()
		if c, err := customers.get(id); err == nil {
			c.DeletedAt = ptr(time.Now())
		}
		return nil
	})
	res.OnRestore(func(ctx *filament.Context, id string) error {
		mu.Lock()
		defer mu.Unlock()
		if c, err := customers.get(id); err == nil {
			c.DeletedAt = nil
		}
		return nil
	})
	return res
}

func countries() map[string]string {
	m := map[string]string{}
	for _, p := range places {
		m[p[1]] = p[1]
	}
	return m
}
