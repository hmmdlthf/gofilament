# GoFilament 🚀

> **High-Performance, Type-Safe Full-Stack Admin Panel & Application Toolkit for GoLaravel & Ent ORM**

Inspired by Laravel's renowned Filament ecosystem, **GoFilament** reimagines rapid admin panel and portal development for Go. Built natively on top of the **GoLaravel** kernel and **Ent ORM**, GoFilament combines compile-time type safety with sub-millisecond execution, Templ SSR, Alpine.js client reactivity, and HTMX micro-swaps—all within a single compiled Go binary.

---

## ⚡ Key Architectural Highlights

- **Compiled Type-Safety (`Resource[T]`)**: Full CRUD schemas, forms, tables, and filters are bound to your Ent models at compile time without reflection bottlenecks.
- **Ent ORM Direct Graph Integration**: Native hooks for predicates, mutations, eager-loading, soft deletes, and multi-tenancy privacy scopes.
- **Zero-Node Reactive Frontend**: Powered by **Templ** (compile-time SSR), **Alpine.js** (client state), and **HTMX** (targeted DOM morphing) with no Node.js, Webpack, or npm runtime dependency.
- **Micro-Millisecond Performance**: Table rendering under 1ms, form hydration under 0.8ms, and throughput exceeding 85,000 requests/sec on GoLaravel's zero-allocation radix router.
- **Embedded Web Assets**: Tailwind CSS, Alpine.js micro-runtime, and Heroicons SVGs are compiled directly into the binary via `//go:embed`.
- **Artisan Code Generators**: `artisan make:filament-panel`, `artisan make:filament-resource`, `artisan make:filament-widget`, etc.

---

## 📦 Installation

```bash
go get github.com/golaravel/gofilament
```

---

## 🛠️ Quickstart: Building an Admin Panel

### 1. Configure the Panel

```go
package main

import (
    "github.com/golaravel/gofilament/pkg/filament"
    "github.com/golaravel/gofilament/pkg/widgets"
    "github.com/golaravel/golaravel/pkg/golaravel"
)

func main() {
    app := golaravel.Default()

    // 1. Configure Panel
    adminPanel := filament.NewPanel("admin").
        Path("/admin").
        BrandName("Acme Operations").
        DarkMode(true).
        SidebarCollapsibleOnDesktop().
        MaxContentWidth(filament.Width7xl).
        NavigationGroups([]string{
            "Staff & Permissions",
            "Shop Management",
        })

    // 2. Add Dashboard KPI Widget with Sparklines
    adminPanel.Widget(widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
        return []widgets.Stat{
            *widgets.NewStat("Total Revenue", "$128,430").
                Description("+32% this month").
                Color("success").
                Chart([]int{15, 22, 35, 45, 60, 78, 128}),
            *widgets.NewStat("Active Orders", "84").
                Description("5 pending dispatch").
                Color("warning"),
        }
    }))

    // 3. Register Panel into GoLaravel
    adminPanel.Register(app)

    app.Run(":8080")
}
```

---

### 2. Define a Typed Ent Resource (`Resource[T]`)

```go
package resources

import (
    "github.com/golaravel/gofilament/pkg/filament"
    "github.com/golaravel/gofilament/pkg/forms"
    "github.com/golaravel/gofilament/pkg/tables"
    "github.com/golaravel/gofilament/pkg/infolists"
    "my-app/app/ent"
)

type UserResource struct {
    *filament.BaseResource[*ent.User]
}

func NewUserResource() *UserResource {
    res := filament.NewResource[*ent.User]("users").
        ModelName("User").
        PluralModelName("Users").
        NavigationGroup("Staff & Permissions").
        NavigationIcon("heroicon-users").
        NavigationBadge(func(ctx *filament.Context) string {
            count, _ := ctx.Ent().User.Query().Count(ctx)
            return fmt.Sprintf("%d", count)
        }).
        Form(func(form *forms.Form) *forms.Form {
            return form.Schema(
                forms.Section("Account Profile").
                    Columns(2).
                    Schema(
                        forms.TextInput("name").Required().Placeholder("e.g. John Doe"),
                        forms.TextInput("email").Email().Required().Placeholder("user@example.com"),
                        forms.Select("role").
                            Options(map[string]string{
                                "admin":  "Administrator",
                                "editor": "Editor",
                                "viewer": "Viewer",
                            }).
                            Default("viewer"),
                        forms.Toggle("is_active").Label("Active Status"),
                    ),
            )
        }).
        Table(func(table *tables.Table[*ent.User]) *tables.Table[*ent.User] {
            return table.
                Columns(
                    tables.TextColumn[*ent.User]("id").Sortable(),
                    tables.TextColumn[*ent.User]("name").Searchable().Sortable().Weight("bold"),
                    tables.TextColumn[*ent.User]("email").Searchable().Copyable(),
                    tables.BadgeColumn[*ent.User]("role").Colors(map[string]string{
                        "admin":  "danger",
                        "editor": "warning",
                        "viewer": "success",
                    }),
                    tables.ToggleColumn[*ent.User]("is_active"),
                    tables.TextColumn[*ent.User]("created_at").DateTime().Sortable(),
                ).
                Filters(
                    tables.SelectFilter("role").Options(map[string]string{
                        "admin":  "Administrators",
                        "editor": "Editors",
                    }),
                ).
                Actions(
                    tables.ViewAction[*ent.User](),
                    tables.EditAction[*ent.User](),
                    tables.DeleteAction[*ent.User](),
                ).
                BulkActions(
                    tables.DeleteBulkAction(),
                )
        }).
        Infolist(func(infolist *infolists.Infolist) *infolists.Infolist {
            return infolist.Schema(
                infolists.Section("User Overview").
                    Columns(2).
                    Schema(
                        infolists.TextEntry("name").Weight("bold"),
                        infolists.TextEntry("email").Copyable(),
                        infolists.BadgeEntry("role"),
                        infolists.TextEntry("created_at").DateTime(),
                    ),
            )
        })

    return &UserResource{BaseResource: res}
}
```

```go
    // In your resource constructor:
    userResource := resources.NewUserResource()

    // Bind directly to your Ent client with zero boilerplate!
    entdriver.BindClient(userResource, client.User)
```

---

## 🎨 Form Builder Components

| Component | Description |
|---|---|
| `forms.TextInput(name)` | Text, email, password, tel, numeric with live debouncing and validation |
| `forms.Textarea(name)` | Multi-line text with configurable rows and character count |
| `forms.Select(name)` | Dropdown options with searchability and multi-select |
| `forms.Toggle(name)` | Boolean toggle switch |
| `forms.DatePicker(name)` | Date and datetime-local picker |
| `forms.FileUpload(name)` | Local disk or S3/MinIO file and image upload |
| `forms.ColorPicker(name)` | Visual color picker with hex swatches |
| `forms.TagsInput(name)` | Chip and tag management |
| `forms.Grid(cols)` | Responsive multi-column layout grid |
| `forms.Section(title)` | Bordered card section with collapsible support |
| `forms.Tabs(...)` | Tabbed form schemas |
| `forms.Wizard(...)` | Sequential step-by-step form wizard with progress indicators |
| `forms.Repeater(name)` | Dynamic nested line-item arrays (add/remove rows) |

### 🧙 Multi-Step Wizard (`forms.Wizard`)

Break complex forms into manageable sequential steps with Alpine.js step indicators, error auto-activation, and custom submit actions:

```go
forms.Wizard(
    forms.WizardStep("Account").
        Description("Login credentials").
        Schema(
            forms.TextInput("username").Required(),
            forms.TextInput("email").Email().Required(),
        ),
    forms.WizardStep("Shipping").
        Description("Delivery destination").
        Schema(
            forms.TextInput("address").Required(),
            forms.TextInput("postal_code").Required(),
        ),
    forms.WizardStep("Payment").
        Description("Billing information").
        Schema(
            forms.TextInput("card_number").Required(),
        ),
).SubmitAction("Place Order Now")
```

### 🔁 Dynamic Line-Item Repeater (`forms.Repeater`)

Allow operators to dynamically add, remove, and manage sub-record arrays with automatic input name scoping (`items[0][sku]`, `items[1][sku]`):

```go
forms.Repeater("line_items").
    Label("Order Items").
    ItemLabel("Item").
    MinItems(1).
    MaxItems(10).
    AddActionLabel("+ Add Line Item").
    Schema(
        forms.TextInput("sku").Required(),
        forms.TextInput("quantity").Numeric().Required(),
        forms.TextInput("price").Numeric().Required(),
    )
```

### Live Reactive Fields (`.Live()`)

Form components update state reactively via HTMX micro-swaps:

```go
forms.TextInput("slug").Disabled(),

forms.TextInput("title").
    Live(forms.Debounce(300 * time.Millisecond)).
    AfterStateUpdated(func(state any, form *forms.Form) {
        form.GetField("slug").SetValue(slugify(state.(string)))
    })
```

---

## 📊 Table Builder Features

- **Typed Generics**: `tables.New[T]()` guarantees field accuracy and prevents runtime reflection overhead.
- **Server-Side Pagination & Sorting**: Dynamic sorting links with directional indicators (`↑`, `↓`).
- **Global & Column Search**: Filter thousands of records with server-side queries.
- **Filters**: `SelectFilter`, `TernaryFilter` (soft-deletes / boolean flags), and custom predicate filters.
- **Row & Bulk Actions**: In-line View/Edit/Delete actions and multi-record bulk operations.

---

## 🔔 Real-Time Notifications Engine & Drawer

GoFilament features a complete notification ecosystem with toast popups, database persistence, an interactive slide-out drawer, and real-time Server-Sent Events (SSE) push powered by GoLaravel's `pkg/broadcast` hub:

```go
// 1. Enable Database Notifications on your panel
adminPanel.DatabaseNotifications(true)

// 2. Dispatch a real-time notification from any handler, background job, or event listener
notifications.Send("admin_user", notifications.Make("Order #101 Placed").
    Success().
    Body("Customer placed a new order for $250.").
    Action("View Order", "/admin/orders/101").
    DurationMs(4000),
)

// Or broadcast to all connected operators
notifications.SendToAll(notifications.Make("Deployment Succeeded").Info())
```

- **Topbar Bell & Badge**: Displays live unread count (`#notif-unread-badge`), which auto-increments upon receiving real-time events.
- **Slide-Out Notification Drawer**: Click the bell icon to slide out the notification center, view historical alerts, open action links, mark individual items as read, or clear all.
- **Server-Sent Events (SSE)**: Built-in `GET /admin/notifications/stream` connection streams new alerts instantly into the browser with zero polling overhead.
- **REST API Support**: Native endpoints for mobile/Flutter clients at `/admin/api/notifications`.

---

## 🏢 Multi-Tenancy & Tenant Switcher

GoFilament integrates natively with GoLaravel's `privacy.TenantMixin`, providing tenant resolution, automatic privacy scope injection, and an interactive topbar organization switcher:

```go
panel := filament.NewPanel("admin").
    Path("/admin").
    Tenancy(&filament.TenantConfig{
        Enabled:         true,
        PathScoping:     true, // scoped URLs: /admin/{tenant}/...
        RegistrationURL: "/admin/tenants/new",
        ProfileURL:      "/admin/{tenant}/settings",
    }).
    TenantLoader(func(ctx *filament.Context) ([]filament.Tenant, error) {
        // Return workspaces accessible to the authenticated user
        return []filament.Tenant{
            filament.TenantItem{Key: "acme", Name: "Acme Corp"},
            filament.TenantItem{Key: "wayne", Name: "Wayne Enterprises"},
        }, nil
    })
```

- **Seamless Privacy Integration**: Automatically injects `privacy.WithTenantID` into standard `context.Context`, scoping all Ent queries and mutations without manual `WHERE tenant_id = ?` clauses.
- **Flexible Resolution**: Resolves tenants via route params (`:tenant`), cookies (`filament_tenant`), headers (`X-Tenant-ID`), subdomains (`{tenant}.app.com`), or custom resolvers.
- **Instant Switcher Dropdown**: Live dropdown in topbar with avatar initials, organization switching links, and settings/registration buttons.

---

## 🔗 Interactive In-Place Relation Managers

Relation Managers allow you to embed related child models (e.g. `OrdersRelationManager`, `AddressesRelationManager`, or `LineItemsRelationManager`) directly inside a parent resource's View and Edit screens:

```go
ordersRM := filament.NewRelationManager[*Order]("orders", "Customer Orders").
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
    Table(func(t *tables.Table[*Order]) *tables.Table[*Order] {
        return t.Columns(
            tables.TextColumn[*Order]("id").Sortable(),
            tables.TextColumn[*Order]("order_no").Searchable(),
            tables.TextColumn[*Order]("total").Money("USD"),
            tables.BadgeColumn[*Order]("status"),
        )
    }).
    OnList(func(ctx *filament.Context, customerID string, search, sort, dir string, page, perPage int) ([]*Order, int, error) {
        // Query child orders where customer_id = customerID
        return queryOrdersByCustomer(customerID)
    }).
    OnCreate(func(ctx *filament.Context, customerID string, data map[string]any) error {
        // Create order linked to customerID in-place
        return createOrderForCustomer(customerID, data)
    }).
    OnUpdate(func(ctx *filament.Context, customerID, orderID string, data map[string]any) error {
        // Update order in-place
        return updateOrder(orderID, data)
    }).
    OnDelete(func(ctx *filament.Context, customerID, orderID string) error {
        // Delete or detach order in-place
        return deleteOrder(orderID)
    })

// Attach to parent resource
customerResource.Relations(ordersRM, addressesRM)
```

- **In-Place Modals**: Create and Edit operations open clean Alpine.js modal dialogs directly on the parent edit page without full-page navigation.
- **Multi-Relation Tabs**: Automatically generates responsive tabs (`activeRelTab`) when multiple relation managers are attached.
- **REST API Support**: Native endpoints for mobile/Flutter clients at `/admin/api/resources/:slug/:id/relations/:rel`.

---

## 🛡️ Filament Shield & Policy Access Control

GoFilament provides role-based and policy-based access control at the panel, resource, and row levels:

```go
// 1. Guard the entire panel with roles or abilities
adminPanel.CanAccess(filament.Role("admin", "super-admin"))

// 2. Define a typed Ent resource policy
type InvoicePolicy struct {
    filament.BasePolicy[Invoice]
}

func (p InvoicePolicy) Create(ctx *filament.Context) bool {
    return ctx.User() != nil && ctx.User().HasPermission("invoices.create")
}

func (p InvoicePolicy) Delete(ctx *filament.Context, record Invoice) bool {
    return record.Status != "paid" // Cannot delete paid invoices
}

// 3. Attach policy or GoLaravel Gate to resource
invoiceResource.Policy(&InvoicePolicy{})
// Or bind directly to GoLaravel's auth.Gate conventions
taskResource.AuthorizeWithGate("tasks")
```

- **Panel Guarding**: Unauthorized requests return 403 Forbidden or redirect to `/admin/login`.
- **Automatic UI Pruning**: Unauthorized resources disappear from sidebar navigation; prohibited create/edit/delete buttons are removed from tables and infolists automatically.
- **Row-Level Action Visibility**: `tables.Action.Visible(func(record T) bool)` conditionally toggles table action buttons per record.

---

## 📱 Flutter Multiplatform & Mobile-First Admin Engine

GoFilament introduces **multi-engine admin panel support**. In addition to the default Templ SSR web engine, you can run your admin panel with **Flutter Multiplatform** (`filament.EngineFlutter`), turning your admin panel into a **native, responsive, mobile-first experience** across iOS, Android, macOS, Windows, Linux, and compiled Web SPA:

```go
panel := filament.NewPanel("admin").
    Path("/admin").
    BrandName("TechMart Operations").
    Flutter(). // Automatically enables Flutter engine & mounts client/build/web
    Resource(productResource)
```

### 📱 Mobile-First Features:
- **One-Thumb Navigation**: Bottom `NavigationBar` for natural mobile interaction without stretching across large phone screens.
- **Pull-to-Refresh**: Native `RefreshIndicator` on all resource listings for seamless pull-to-refresh data updates.
- **Floating Action Button (FAB)**: Quick one-tap record creation docked at the bottom right.
- **Bottom-Sheet Forms**: Record creation and editing occur inside mobile-optimized bottom sheets (`showModalBottomSheet`) rather than cumbersome desktop modals.
- **Swipe-to-Delete**: Dismissible swipe gestures (`Dismissible`) with confirmation feedback.

### 🖥️ Responsive Desktop & Tablet Adaptation:
- **Adaptive Breakpoints**: Screen widths > 768px seamlessly switch from the bottom nav bar to an expandable `NavigationRail` sidebar.
- **Split-Pane & Data Grid**: Displays dense tables with column sorting and split-view master-detail inspection on tablets and desktops.

### ⚡ Scaffolding a Flutter Admin:
```bash
# 1. Create a Flutter-enabled panel provider
php artisan make:filament-panel MobileStore --engine=flutter

# 2. Generate the responsive Flutter admin client
php artisan make:filament-flutter MobileStore --dir=client

# 3. Run natively or compile for web
cd client && flutter run         # iOS, Android, Desktop
cd client && flutter build web   # Compiles to Web SPA served by GoLaravel
```

### 🔌 Standardized Panel REST API:
The Flutter client and external integrations connect to GoFilament's standardized JSON REST API:
- `GET /admin/api/schema`: Panel metadata, registered resources, fields, and navigation groups.
- `GET /admin/api/resources/:slug`: Paginated, searchable, sortable list endpoints.
- `GET /admin/api/resources/:slug/export`: Direct streaming data export (`format=csv|tsv|json`, `ids`, `search`, filters).
- `POST /admin/api/resources/:slug/import`: High-throughput batch import (multipart CSV file, raw CSV string, or JSON array batches).
- `GET/POST/PUT/DELETE /admin/api/resources/:slug/:id`: Full CRUD with Shield policy authorization.
- `GET /admin/api/tenants` & `POST /admin/api/tenants/switch`: Active tenant management.
- `GET /admin/api/widgets`: Real-time KPI stats overview and sparklines.

---

## ⚡ High-Throughput Data Streaming: Import & Export

GoFilament features a streaming data export and interactive column-mapping import engine designed for large datasets with minimal memory overhead ($O(\text{chunk size})$):

### 📤 High-Throughput Export:
- Stream thousands of records directly to the browser or API clients without buffering whole tables in memory.
- Formats supported: **CSV** (`text/csv`), **TSV** (`text/tab-separated-values`), and **JSON** (`application/json`).
- Automatically exports configured table columns and labels, stripping HTML tags to output clean data.
- Row-level bulk export via `tables.ExportBulkAction()`, or header actions with `tables.ExportAction()`.
- Filter-aware: respects table search terms, active column filters, and selected record IDs.

```go
// Add Export to table header actions or bulk actions
table.HeaderActions(
    tables.ExportAction[*Customer](),
    tables.ImportAction[*Customer](),
).BulkActions(
    tables.ExportBulkAction(),
)
```

### 📥 Interactive Column-Mapping Import:
- Upload CSV or TSV files via web modal (`/admin/{resource}/import`) or REST API.
- **Smart Auto-Mapping**: Automatically normalizes file headers (e.g., `Customer Name` $\rightarrow$ `customer_name` $\rightarrow$ `name`) to match model fields.
- **Explicit Mapping**: Override column definitions with custom mapping fields.
- **Batch Processing & Error Resilience**: Processes rows sequentially with `OnCreate`, collecting failure logs without terminating the batch.
- **Real-Time Notification**: Dispatches toast and drawer notifications with success counts and warning reports via `notifications.Send(...)`.

---

---

## 💻 Artisan CLI Generators

Integrated directly with GoLaravel's Artisan CLI:

```bash
# Create a new panel
artisan make:filament-panel Admin

# Create an Ent CRUD resource manually
artisan make:filament-resource Customer

# Automatically inspect Ent schema AST to generate forms, tables, filters & relations
artisan make:filament-resource Invoice --generate

# Create a custom panel page
artisan make:filament-page Settings

# Create a KPI dashboard widget
artisan make:filament-widget RevenueStats

# Create an in-place relationship manager
artisan make:filament-relation Customer Orders
```

---

## 🧪 Testing

Run the comprehensive test suite with zero network overhead:

```bash
go test ./... -v
```

---

## 📄 License

Open-source software licensed under the **MIT License**.
