package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/activitylog"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/tables"
)

// ----------------------------------------------------
// 1. Product Resource
// ----------------------------------------------------

func buildProductResource() *filament.BaseResource[*Product] {
	res := filament.NewResource[*Product]("products").
		ModelName("Product").
		PluralModelName("Products").
		NavigationGroup("Store Catalog").
		NavigationIcon(`<svg style="width: 1.25rem; height: 1.25rem;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" /></svg>`).
		SoftDeletes(true).
		IsTrashed(func(p *Product) bool {
			return p != nil && p.DeletedAt != nil
		}).
		RecordID(func(p *Product) string {
			if p == nil {
				return ""
			}
			return p.ID
		})

	// Table Configuration: Inline editing, column toggling, drag-and-drop reordering
	res.Table(func(t *tables.Table[*Product]) *tables.Table[*Product] {
		return t.
			Toggleable(true).
			Reorderable("sort_order").
			Columns(
				tables.TextColumn[*Product]("title").
					Label("Product Title").
					Weight("bold").
					Searchable().
					Sortable(),
				tables.TextColumn[*Product]("sku").
					Label("SKU").
					Copyable(),
				tables.TextInputColumn[*Product]("price").
					Label("Price ($)"),
				tables.SelectColumn[*Product]("inventory_status").
					Label("Inventory").
					Options(map[string]string{
						"in_stock":     "In Stock",
						"low_stock":    "Low Stock (≤5)",
						"out_of_stock": "Sold Out",
					}),
				tables.ToggleColumn[*Product]("is_published").
					Label("Published"),
				tables.TextColumn[*Product]("stock").
					Label("Stock Qty").
					Sortable(),
				tables.TextColumn[*Product]("brand").
					Label("Brand").
					ToggledHiddenByDefault(true),
			).
			Filters(
				tables.SelectFilter("inventory_status").
					Label("Filter Inventory").
					Options(map[string]string{
						"in_stock":     "In Stock Only",
						"low_stock":    "Low Stock Only",
						"out_of_stock": "Sold Out Only",
					}),
			).
			Actions(
				tables.EditAction[*Product](),
				tables.DeleteAction[*Product](),
			).
			BulkActions(
				tables.DeleteBulkAction(),
				tables.ExportBulkAction(),
			)
	})

	// Form Configuration: Multi-step Form Wizard (forms.Wizard)
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Wizard(
				forms.WizardStep("Basic Info").
					Description("Title, SKU, and visibility").
					Icon("heroicon-o-information-circle").
					Badge("Step 1").
					BadgeColor("primary").
					Schema(
						forms.Grid(2).Schema(
							forms.TextInput("title").
								Label("Product Title").
								Placeholder("e.g. Wireless Noise-Cancelling Headphones").
								Required(),
							forms.TextInput("slug").
								Label("URL Slug").
								Placeholder("auto-generated-slug"),
						),
						forms.Grid(2).Schema(
							forms.TextInput("sku").
								Label("Stock Keeping Unit (SKU)").
								Required(),
							forms.TextInput("brand").
								Label("Brand / Manufacturer"),
						),
						forms.Toggle("is_published").
							Label("Publish to Storefront").
							HelperText("Controls whether buyers can view and purchase this item."),
					),
				forms.WizardStep("Pricing & Stock").
					Description("Price points and inventory quantities").
					Icon("heroicon-o-currency-dollar").
					Badge("Step 2").
					BadgeColor("warning").
					Schema(
						forms.Grid(3).Schema(
							forms.TextInput("price").
								Numeric().
								Label("Retail Price ($)").
								Required(),
							forms.TextInput("compare_at_price").
								Numeric().
								Label("Compare At Price ($)"),
							forms.TextInput("cost_price").
								Numeric().
								Label("Cost per Item ($)"),
						),
						forms.Grid(2).Schema(
							forms.TextInput("stock").
								Numeric().
								Label("Available Stock Quantity").
								Required(),
							forms.Select("inventory_status").
								Label("Inventory Status").
								Options(map[string]string{
									"in_stock":     "In Stock",
									"low_stock":    "Low Stock",
									"out_of_stock": "Out of Stock",
								}),
						),
						forms.TagsInput("categories").
							Label("Product Categories").
							Suggestions("Electronics", "Audio", "Keyboards", "Peripherals", "Accessories", "Furniture"),
					),
				forms.WizardStep("Media & Description").
					Description("Color swatch and product description").
					Icon("heroicon-o-photo").
					Badge("Final").
					BadgeColor("success").
					Schema(
						forms.ColorPicker("color_variant").
							Label("Primary Color Swatch").
							Presets("#111827", "#4b5563", "#1f2937", "#0284c7", "#10b981", "#ef4444"),
						forms.MarkdownEditor("description").
							Label("Product Overview & Specs").
							Rows(8),
					),
			).
				ProgressBar(true).
				SubmitAction("Save Product"),
		)
	})

	// CRUD Handlers
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Product, int, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()

		trashedScope := filters["trashed"]
		invFilter := filters["inventory_status"]
		searchLower := strings.ToLower(strings.TrimSpace(search))

		var matched []*Product
		for _, p := range products {
			// Soft-delete trash scoping
			if trashedScope == "only_trashed" {
				if p.DeletedAt == nil {
					continue
				}
			} else if trashedScope == "with_trashed" {
				// show all
			} else {
				// without_trashed (default)
				if p.DeletedAt != nil {
					continue
				}
			}

			if invFilter != "" && p.InventoryStatus != invFilter {
				continue
			}

			if searchLower != "" {
				if !strings.Contains(strings.ToLower(p.Title), searchLower) &&
					!strings.Contains(strings.ToLower(p.SKU), searchLower) &&
					!strings.Contains(strings.ToLower(p.Brand), searchLower) {
					continue
				}
			}

			matched = append(matched, p)
		}

		// Sort by SortOrder or specified column
		sort.Slice(matched, func(i, j int) bool {
			if sortCol == "price" {
				if sortDir == "desc" {
					return matched[i].Price > matched[j].Price
				}
				return matched[i].Price < matched[j].Price
			}
			if sortCol == "stock" {
				if sortDir == "desc" {
					return matched[i].Stock > matched[j].Stock
				}
				return matched[i].Stock < matched[j].Stock
			}
			return matched[i].SortOrder < matched[j].SortOrder
		})

		total := len(matched)
		start := (page - 1) * perPage
		if start >= total {
			return []*Product{}, total, nil
		}
		end := start + perPage
		if end > total {
			end = total
		}
		return matched[start:end], total, nil
	})

	res.OnFind(func(ctx *filament.Context, id string) (*Product, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()
		if p, ok := products[id]; ok {
			return p, nil
		}
		return nil, fmt.Errorf("product %s not found", id)
	})

	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		newID := fmt.Sprintf("%d", len(products)+1)
		data["id"] = newID
		price, _ := strconv.ParseFloat(fmt.Sprintf("%v", data["price"]), 64)
		stock, _ := strconv.Atoi(fmt.Sprintf("%v", data["stock"]))

		p := &Product{
			ID:              newID,
			Title:           fmt.Sprintf("%v", data["title"]),
			Slug:            fmt.Sprintf("%v", data["slug"]),
			SKU:             fmt.Sprintf("%v", data["sku"]),
			Brand:           fmt.Sprintf("%v", data["brand"]),
			Price:           price,
			Stock:           stock,
			ColorVariant:    fmt.Sprintf("%v", data["color_variant"]),
			Description:     fmt.Sprintf("%v", data["description"]),
			IsPublished:     data["is_published"] == "true" || data["is_published"] == true || data["is_published"] == "on",
			InventoryStatus: fmt.Sprintf("%v", data["inventory_status"]),
			SortOrder:       len(products) + 1,
			CreatedAt:       time.Now(),
		}
		if p.InventoryStatus == "" || p.InventoryStatus == "<nil>" {
			p.InventoryStatus = "in_stock"
		}
		products[newID] = p
		return nil
	})

	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		p, ok := products[id]
		if !ok {
			return fmt.Errorf("product %s not found", id)
		}

		if val, exists := data["title"]; exists && val != nil {
			p.Title = fmt.Sprintf("%v", val)
		}
		if val, exists := data["price"]; exists && val != nil {
			p.Price, _ = strconv.ParseFloat(fmt.Sprintf("%v", val), 64)
		}
		if val, exists := data["stock"]; exists && val != nil {
			p.Stock, _ = strconv.Atoi(fmt.Sprintf("%v", val))
		}
		if val, exists := data["is_published"]; exists {
			p.IsPublished = val == "true" || val == true || val == "on" || val == "1"
		}
		if val, exists := data["inventory_status"]; exists && val != nil && val != "" {
			p.InventoryStatus = fmt.Sprintf("%v", val)
		}
		return nil
	})

	res.OnDelete(func(ctx *filament.Context, id string) error {
		dataMu.Lock()
		defer dataMu.Unlock()
		if p, ok := products[id]; ok {
			t := time.Now()
			p.DeletedAt = &t
		}
		return nil
	})

	res.OnRestore(func(ctx *filament.Context, id string) error {
		dataMu.Lock()
		defer dataMu.Unlock()
		if p, ok := products[id]; ok {
			p.DeletedAt = nil
		}
		return nil
	})

	res.OnForceDelete(func(ctx *filament.Context, id string) error {
		dataMu.Lock()
		defer dataMu.Unlock()
		delete(products, id)
		return nil
	})

	// Inline Cell Editing handler (handles inline Price, Status, Toggle)
	res.OnCellUpdate(func(ctx *filament.Context, id, column string, value any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		p, ok := products[id]
		if !ok {
			return fmt.Errorf("product %s not found", id)
		}

		switch column {
		case "price":
			p.Price, _ = strconv.ParseFloat(fmt.Sprintf("%v", value), 64)
		case "inventory_status":
			p.InventoryStatus = fmt.Sprintf("%v", value)
		case "is_published":
			p.IsPublished = value == "true" || value == true || value == "1" || value == 1
		}
		return nil
	})

	// Drag-and-Drop row reorder handler
	res.OnReorder(func(ctx *filament.Context, orderedIDs []string) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		for idx, id := range orderedIDs {
			if p, ok := products[id]; ok {
				p.SortOrder = idx + 1
			}
		}
		return nil
	})

	return res
}

// ----------------------------------------------------
// 2. Order Resource
// ----------------------------------------------------

func buildOrderResource() *filament.BaseResource[*Order] {
	res := filament.NewResource[*Order]("orders").
		ModelName("Order").
		PluralModelName("Orders").
		NavigationGroup("Sales & Fulfillment").
		NavigationIcon(`<svg style="width: 1.25rem; height: 1.25rem;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-3 7h3m-3 4h3m-6-4h.01M9 16h.01" /></svg>`).
		RecordID(func(o *Order) string {
			if o == nil {
				return ""
			}
			return o.ID
		})

	res.Table(func(t *tables.Table[*Order]) *tables.Table[*Order] {
		return t.
			Columns(
				tables.TextColumn[*Order]("order_no").
					Label("Order #").
					Weight("bold").
					Copyable().
					Searchable().
					Sortable(),
				tables.TextColumn[*Order]("customer_name").
					Label("Customer").
					Searchable(),
				tables.BadgeColumn[*Order]("payment_status").
					Label("Payment").
					Colors(map[string]string{
						"paid":     "success",
						"pending":  "warning",
						"refunded": "danger",
					}),
				tables.BadgeColumn[*Order]("fulfillment_status").
					Label("Fulfillment").
					Colors(map[string]string{
						"delivered":  "success",
						"shipped":    "primary",
						"processing": "warning",
						"cancelled":  "danger",
					}),
				tables.TextColumn[*Order]("total").
					Label("Total ($)").
					Sortable().
					Value(func(o *Order) any {
						return fmt.Sprintf("$%.2f", o.Total)
					}),
				tables.TextColumn[*Order]("created_at").
					Label("Placed").
					DateTime(),
			).
			Filters(
				tables.SelectFilter("payment_status").
					Label("Payment Status").
					Options(map[string]string{
						"paid":     "Paid",
						"pending":  "Pending",
						"refunded": "Refunded",
					}),
				tables.SelectFilter("fulfillment_status").
					Label("Fulfillment Status").
					Options(map[string]string{
						"delivered":  "Delivered",
						"shipped":    "Shipped",
						"processing": "Processing",
						"cancelled":  "Cancelled",
					}),
			).
			Actions(
				tables.ViewAction[*Order](),
				tables.EditAction[*Order](),
			)
	})

	// Detailed View Infolist with Timeline
	res.Infolist(func(il *infolists.Infolist) *infolists.Infolist {
		return il.Schema(
			infolists.Section("Order Overview").
				Columns(3).
				Schema(
					infolists.TextEntry("order_no").Weight("bold").Copyable(),
					infolists.TextEntry("customer_name"),
					infolists.TextEntry("customer_email").Copyable(),
					infolists.BadgeEntry("payment_status"),
					infolists.BadgeEntry("fulfillment_status"),
					infolists.TextEntry("tracking_number").Copyable(),
				),
			infolists.Section("Order Items & Totals").
				Columns(2).
				Schema(
					infolists.TextEntry("shipping_address"),
					infolists.TextEntry("total").Weight("bold"),
				),
			infolists.Section("Audit Trail & Order Timeline").
				Schema(
					activitylog.TimelineEntry("activities").
						ForSubject("Order").
						Title("Order Status History & Changesets").
						Filterable(true).
						CollapsibleDiffs(true),
				),
		)
	})

	// Form with dynamic Repeater for Order Line Items
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Section("Order & Customer Details").Columns(2).Schema(
				forms.TextInput("order_no").Label("Order Number").Required(),
				forms.TextInput("customer_name").Label("Customer Name").Required(),
				forms.TextInput("customer_email").Email().Label("Customer Email").Required(),
				forms.TextInput("shipping_address").Label("Delivery Address").Required(),
			),
			forms.Section("Status & Logistics").Columns(3).Schema(
				forms.Select("payment_status").
					Label("Payment Status").
					Options(map[string]string{
						"paid":     "Paid",
						"pending":  "Pending",
						"refunded": "Refunded",
					}),
				forms.Select("fulfillment_status").
					Label("Fulfillment Status").
					Options(map[string]string{
						"delivered":  "Delivered",
						"shipped":    "Shipped",
						"processing": "Processing",
						"cancelled":  "Cancelled",
					}),
				forms.TextInput("tracking_number").Label("Tracking Number"),
			),
			forms.Repeater("items").
				Label("Order Items").
				ItemLabel("Item").
				AddActionLabel("+ Add Product Item").
				MinItems(1).
				Grid(4).
				Schema(
					forms.TextInput("product_name").Label("Product").Required(),
					forms.TextInput("quantity").Numeric().Label("Qty").Required(),
					forms.TextInput("unit_price").Numeric().Label("Unit Price ($)").Required(),
					forms.TextInput("total").Numeric().Label("Line Total ($)").Required(),
				),
		)
	})

	// CRUD Handlers
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Order, int, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()

		searchLower := strings.ToLower(strings.TrimSpace(search))
		payFilter := filters["payment_status"]
		fulFilter := filters["fulfillment_status"]

		var matched []*Order
		for _, o := range orders {
			if payFilter != "" && o.PaymentStatus != payFilter {
				continue
			}
			if fulFilter != "" && o.FulfillmentStatus != fulFilter {
				continue
			}
			if searchLower != "" {
				if !strings.Contains(strings.ToLower(o.OrderNo), searchLower) &&
					!strings.Contains(strings.ToLower(o.CustomerName), searchLower) &&
					!strings.Contains(strings.ToLower(o.TrackingNumber), searchLower) {
					continue
				}
			}
			matched = append(matched, o)
		}

		sort.Slice(matched, func(i, j int) bool {
			return matched[i].CreatedAt.After(matched[j].CreatedAt)
		})

		total := len(matched)
		start := (page - 1) * perPage
		if start >= total {
			return []*Order{}, total, nil
		}
		end := start + perPage
		if end > total {
			end = total
		}
		return matched[start:end], total, nil
	})

	res.OnFind(func(ctx *filament.Context, id string) (*Order, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()
		if o, ok := orders[id]; ok {
			return o, nil
		}
		return nil, fmt.Errorf("order %s not found", id)
	})

	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		o, ok := orders[id]
		if !ok {
			return fmt.Errorf("order %s not found", id)
		}

		if v, ok := data["payment_status"]; ok && v != nil {
			o.PaymentStatus = fmt.Sprintf("%v", v)
		}
		if v, ok := data["fulfillment_status"]; ok && v != nil {
			o.FulfillmentStatus = fmt.Sprintf("%v", v)
		}
		if v, ok := data["tracking_number"]; ok && v != nil {
			o.TrackingNumber = fmt.Sprintf("%v", v)
		}
		if v, ok := data["shipping_address"]; ok && v != nil {
			o.ShippingAddress = fmt.Sprintf("%v", v)
		}
		if v, ok := data["notes"]; ok && v != nil {
			o.Notes = fmt.Sprintf("%v", v)
		}
		return nil
	})

	return res
}

// ----------------------------------------------------
// 3. Customer Resource
// ----------------------------------------------------

func buildCustomerResource() *filament.BaseResource[*Customer] {
	res := filament.NewResource[*Customer]("customers").
		ModelName("Customer").
		PluralModelName("Customers").
		NavigationGroup("Customers & CRM").
		NavigationIcon(`<svg style="width: 1.25rem; height: 1.25rem;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" /></svg>`).
		SoftDeletes(true).
		IsTrashed(func(c *Customer) bool {
			return c != nil && c.DeletedAt != nil
		}).
		RecordID(func(c *Customer) string {
			if c == nil {
				return ""
			}
			return c.ID
		})

	res.Table(func(t *tables.Table[*Customer]) *tables.Table[*Customer] {
		return t.
			Columns(
				tables.TextColumn[*Customer]("name").
					Label("Customer Name").
					Weight("bold").
					Searchable().
					Sortable(),
				tables.TextColumn[*Customer]("email").
					Label("Email Address").
					Copyable(),
				tables.TextColumn[*Customer]("city").
					Label("City").
					Sortable(),
				tables.ToggleColumn[*Customer]("is_vip").
					Label("VIP"),
				tables.TextColumn[*Customer]("total_spent").
					Label("Total Spent").
					Sortable().
					Value(func(c *Customer) any {
						return fmt.Sprintf("$%.2f", c.TotalSpent)
					}),
				tables.TextColumn[*Customer]("orders_count").
					Label("Orders").
					Sortable(),
			).
			Actions(
				tables.ViewAction[*Customer](),
				tables.EditAction[*Customer](),
				tables.DeleteAction[*Customer](),
			)
	})

	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Grid(2).Schema(
				forms.TextInput("name").Label("Full Name").Required(),
				forms.TextInput("email").Email().Label("Email Address").Required(),
				forms.TextInput("phone").Label("Phone Number"),
				forms.TextInput("city").Label("City"),
			),
			forms.TagsInput("tags").
				Label("Customer Tags").
				Suggestions("VIP", "Wholesale", "Influencer", "B2B", "New Buyer"),
			forms.Toggle("is_vip").
				Label("VIP Membership Status").
				HelperText("Grants free priority shipping and early product access."),
		)
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Customer, int, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()

		trashedScope := filters["trashed"]
		searchLower := strings.ToLower(strings.TrimSpace(search))

		var matched []*Customer
		for _, c := range customers {
			if trashedScope == "only_trashed" {
				if c.DeletedAt == nil {
					continue
				}
			} else if trashedScope == "with_trashed" {
				// all
			} else {
				if c.DeletedAt != nil {
					continue
				}
			}

			if searchLower != "" {
				if !strings.Contains(strings.ToLower(c.Name), searchLower) &&
					!strings.Contains(strings.ToLower(c.Email), searchLower) &&
					!strings.Contains(strings.ToLower(c.City), searchLower) {
					continue
				}
			}
			matched = append(matched, c)
		}

		total := len(matched)
		start := (page - 1) * perPage
		if start >= total {
			return []*Customer{}, total, nil
		}
		end := start + perPage
		if end > total {
			end = total
		}
		return matched[start:end], total, nil
	})

	res.OnFind(func(ctx *filament.Context, id string) (*Customer, error) {
		dataMu.RLock()
		defer dataMu.RUnlock()
		if c, ok := customers[id]; ok {
			return c, nil
		}
		return nil, fmt.Errorf("customer %s not found", id)
	})

	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		newID := fmt.Sprintf("%d", len(customers)+1)
		data["id"] = newID

		var tags []string
		if tVal, ok := data["tags"]; ok {
			switch tv := tVal.(type) {
			case []string:
				tags = tv
			case []any:
				for _, itm := range tv {
					tags = append(tags, fmt.Sprintf("%v", itm))
				}
			case string:
				for _, part := range strings.Split(tv, ",") {
					pTrim := strings.TrimSpace(part)
					if pTrim != "" {
						tags = append(tags, pTrim)
					}
				}
			}
		}

		c := &Customer{
			ID:          newID,
			Name:        fmt.Sprintf("%v", data["name"]),
			Email:       fmt.Sprintf("%v", data["email"]),
			Phone:       fmt.Sprintf("%v", data["phone"]),
			City:        fmt.Sprintf("%v", data["city"]),
			Tags:        tags,
			IsVIP:       data["is_vip"] == "true" || data["is_vip"] == true || data["is_vip"] == "on" || data["is_vip"] == "1",
			OrdersCount: 0,
			TotalSpent:  0.0,
			CreatedAt:   time.Now(),
		}
		customers[newID] = c
		return nil
	})

	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		dataMu.Lock()
		defer dataMu.Unlock()

		c, ok := customers[id]
		if !ok {
			return fmt.Errorf("customer %s not found", id)
		}

		if v, ok := data["name"]; ok && v != nil {
			c.Name = fmt.Sprintf("%v", v)
		}
		if v, ok := data["email"]; ok && v != nil {
			c.Email = fmt.Sprintf("%v", v)
		}
		if v, ok := data["phone"]; ok && v != nil {
			c.Phone = fmt.Sprintf("%v", v)
		}
		if v, ok := data["city"]; ok && v != nil {
			c.City = fmt.Sprintf("%v", v)
		}
		if v, ok := data["tags"]; ok {
			var tags []string
			switch tv := v.(type) {
			case []string:
				tags = tv
			case []any:
				for _, itm := range tv {
					tags = append(tags, fmt.Sprintf("%v", itm))
				}
			case string:
				for _, part := range strings.Split(tv, ",") {
					pTrim := strings.TrimSpace(part)
					if pTrim != "" {
						tags = append(tags, pTrim)
					}
				}
			}
			c.Tags = tags
		}
		if v, ok := data["is_vip"]; ok {
			c.IsVIP = v == "true" || v == true || v == "on" || v == "1"
		}
		return nil
	})

	res.OnDelete(func(ctx *filament.Context, id string) error {
		dataMu.Lock()
		defer dataMu.Unlock()
		if c, ok := customers[id]; ok {
			t := time.Now()
			c.DeletedAt = &t
		}
		return nil
	})

	res.OnRestore(func(ctx *filament.Context, id string) error {
		dataMu.Lock()
		defer dataMu.Unlock()
		if c, ok := customers[id]; ok {
			c.DeletedAt = nil
		}
		return nil
	})

	res.OnForceDelete(func(ctx *filament.Context, id string) error {
		dataMu.Lock()
		defer dataMu.Unlock()
		delete(customers, id)
		return nil
	})

	return res
}
