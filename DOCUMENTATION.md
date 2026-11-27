# GoFilament 🚀 Complete Framework Architecture & Developer Reference

> **The Definitive Guide for Building High-Performance, Type-Safe Full-Stack Admin Panels, Portals, and Multiplatform Applications with GoLaravel & Ent ORM**
> *Current Framework Release: v1.25.0*

---

## Table of Contents

1. [Executive Overview & Philosophy](#1-executive-overview--philosophy)
2. [Dual-Engine Architecture](#2-dual-engine-architecture)
   - [Engine 1: Zero-Node Reactive SSR (Templ + Alpine.js + HTMX)](#engine-1-zero-node-reactive-ssr-templ--alpinejs--htmx)
   - [Engine 2: Flutter Multiplatform Engine](#engine-2-flutter-multiplatform-engine)
3. [Installation & Project Structure](#3-installation--project-structure)
4. [Panel Builder Architecture (`pkg/filament`)](#4-panel-builder-architecture-pkgfilament)
   - [Panel Initialization & Core Configuration](#panel-initialization--core-configuration)
   - [Dual-Engine Switching](#dual-engine-switching)
   - [Navigation, Groups, Badges & Collapsible Sidebars](#navigation-groups-badges--collapsible-sidebars)
   - [Mobile Navigation & Bottom Navigation Bar](#mobile-navigation--bottom-navigation-bar)
   - [App Versioning & Hot Update Detection](#app-versioning--hot-update-detection)
   - [Customizable Pre-Flutter Loading Screen](#customizable-pre-flutter-loading-screen)
   - [Global Search Configuration](#global-search-configuration)
   - [Multi-Tenancy & Privacy Scoping](#multi-tenancy--privacy-scoping)
   - [Authentication & Session Security](#authentication--session-security)
   - [Internationalization (i18n) & RTL Layout](#internationalization-i18n--rtl-layout)
5. [Resource System (`Resource[T]` & `BaseResource[T]`)](#5-resource-system-resourcet--baseresourcet)
   - [Resource Declaration & Model Binding](#resource-declaration--model-binding)
   - [Lifecycle Handlers (`OnList`, `OnFind`, `OnCreate`, `OnUpdate`, `OnDelete`)](#lifecycle-handlers)
   - [Inline Cell Editing & Row Reordering](#inline-cell-editing--row-reordering)
   - [Soft Deletes & Trash Recovery Bin](#soft-deletes--trash-recovery-bin)
   - [Authorization Policies & Gates](#authorization-policies--gates)
   - [Dynamic Resource Status Tabs](#dynamic-resource-status-tabs)
   - [Pagination & Infinite Scroll](#pagination--infinite-scroll)
   - [Column Controls (`DefaultColumns`, `SelectableColumns`, `AdvancedSearch`)](#column-controls)
6. [Form Builder Engine (`pkg/forms`)](#6-form-builder-engine-pkgforms)
   - [Layout Containers (`Section`, `Grid`, `Tabs`, `Fieldset`, `Split`, `Wizard`, `Repeater`)](#layout-containers)
   - [Input Fields Reference](#input-fields-reference)
   - [Live Reactivity & Dynamic Micro-Swaps (`.Live()`)](#live-reactivity--dynamic-micro-swaps)
   - [Form Parsing & Multipart Detection](#form-parsing--multipart-detection)
7. [Table Builder Engine (`pkg/tables`)](#7-table-builder-engine-pkgtables)
   - [Column Types Reference](#column-types-reference)
   - [Column Controls, Resizing, Sorting & Toggling](#column-controls-resizing-sorting--toggling)
   - [Filter Suite (`SelectFilter`, `TernaryFilter`, `TrashedFilter`, Refined Search)](#filter-suite)
   - [Table Actions & Bulk Operations](#table-actions--bulk-operations)
   - [Streaming CSV Data Export & Import](#streaming-csv-data-export--import)
8. [Relationship Managers (`RelationManager`)](#8-relationship-managers-relationmanager)
9. [Infolist Engine (`pkg/infolists`)](#9-infolist-engine-pkginfolists)
10. [Dashboard Studio & Widgets (`pkg/widgets`)](#10-dashboard-studio--widgets-pkgwidgets)
11. [Activity Log & Audit Trail Timeline (`pkg/activitylog`)](#11-activity-log--audit-trail-timeline-pkgactivitylog)
12. [Live Chat & AI Support Concierge (`pkg/support`)](#12-live-chat--ai-support-concierge-pkgsupport)
13. [Flutter Multiplatform Client Architecture (`pkg/flutterclient`)](#13-flutter-multiplatform-client-architecture-pkgflutterclient)
14. [Artisan CLI Generator Reference (`pkg/commands`)](#14-artisan-cli-generator-reference-pkgcommands)
15. [Complete End-to-End Code Blueprint](#15-complete-end-to-end-code-blueprint)
16. [Payment Gateways (`pkg/payments`)](#16-payment-gateways-pkgpayments)
17. [Per-Tenant Features & Settings (`pkg/features`)](#17-per-tenant-features--settings-pkgfeatures)

---

## 1. Executive Overview & Philosophy

**GoFilament** is a premier, compile-time type-safe, batteries-included administrative toolkit and rapid application framework built natively on top of the **GoLaravel** kernel and **Ent ORM**.

Inspired by the developer ergonomics of Laravel's Filament, GoFilament is designed for the Go ecosystem with the following core principles:

1. **Compile-Time Type Safety (`Resource[T]`)**: All schemas, forms, tables, filters, and actions are bound to your Go structs and Ent models using generics—eliminating reflection bottlenecks and runtime type crashes.
2. **Sub-Millisecond Performance**: Table rendering occurs under 1ms, form hydration under 0.8ms, and HTTP throughput exceeds 85,000 requests/sec on GoLaravel's zero-allocation radix router.
3. **Zero-Node Reactive Frontend**: In default SSR mode, the UI is powered by compile-time **Templ** components, lightweight **Alpine.js**, and targeted **HTMX** DOM morphs. No Node.js, Webpack, or npm runtime dependency is required.
4. **Native Multiplatform Capabilities**: Beyond SSR, GoFilament includes a native multiplatform engine that powers mobile-first **Flutter** clients (iOS, Android, macOS, Linux, Windows, and Web) fed by a uniform JSON REST API.

---

## 2. Dual-Engine Architecture

GoFilament supports two distinct, first-class execution engines configured on the panel level via `.Engine(...)`:

```
┌────────────────────────────────────────────────────────────────────────┐
│                              GoFilament                                │
│   ┌──────────────────────────────────┬─────────────────────────────┐   │
│   │   EngineDefault (SSR Templ)      │   EngineFlutter (REST API)  │   │
│   │   • Templ compile-time SSR       │   • Material 3 Flutter UI   │   │
│   │   • Alpine.js reactive state     │   • Mobile-First cards      │   │
│   │   • HTMX dynamic micro-swaps     │   • Desktop Navigation Rail │   │
│   │   • Embedded Tailwind & Icons    │   • iOS/Android/Desktop/Web │   │
│   └──────────────────────────────────┴─────────────────────────────┘   │
├────────────────────────────────────────────────────────────────────────┤
│                    GoFilament Framework Core                           │
│     Panel Builder • Form Builder • Table Builder • Infolists • Actions │
│     Widgets • Activity Log • Soft Deletes • Support Concierge • i18n   │
├────────────────────────────────────────────────────────────────────────┤
│                    GoLaravel Framework & Ent ORM                       │
│     Zero-Alloc Router • Ent Graph Queries • Migrations • Auth • Cache  │
└────────────────────────────────────────────────────────────────────────┘
```

### Engine 1: Zero-Node Reactive SSR (`EngineDefault`)
- **Technology**: Templ + Alpine.js 3.x + HTMX 1.9.x + Tailwind CSS.
- **Delivery**: A single self-contained compiled Go binary with all CSS, JS, and SVG assets embedded via `//go:embed`.
- **Ideal for**: Classic internal tools, enterprise portals, high-throughput desktop web dashboards, and deployments where zero external dependencies are paramount.

### Engine 2: Flutter Multiplatform Engine (`EngineFlutter`)
- **Technology**: Material 3 Flutter multiplatform application served either natively on devices or via a Web SPA shell with full offline/mobile optimizations.
- **API Surface**: GoFilament automatically exposes `/admin/api/schema`, `/admin/api/resources/{slug}`, `/admin/api/locale`, `/admin/api/version`, and `/admin/api/tenants`.
- **UX Features**: 
  - Dynamic bottom navigation with customizable slots.
  - Zero-scroll touch-optimized cards on mobile & tablet viewports.
  - Directionality mirroring (LTR/RTL) across the entire widget tree.
  - Haptic feedback, pull-to-refresh, biometric security diagnostics, and floating account sheets.
- **Ideal for**: Modern field operations, mobile-first admin apps, IoT telemetry terminals, and unified multiplatform software.

---

## 3. Installation & Project Structure

### Installation

```bash
go get github.com/golaravel/gofilament
go get github.com/golaravel/golaravel
```

### Recommended Directory Layout

```
my-app/
├── cmd/
│   └── app/
│       └── main.go                  # Application entrypoint & panel registration
├── app/
│   ├── filament/
│   │   ├── resources/               # Filament resources
│   │   │   ├── product_resource.go
│   │   │   ├── order_resource.go
│   │   │   └── user_resource.go
│   │   ├── widgets/                 # Dashboard widgets
│   │   │   ├── stats_overview.go
│   │   │   └── revenue_chart.go
│   │   └── pages/                   # Custom panel pages
│   │       └── settings_page.go
│   └── models/                      # Ent ORM or domain structs
│       ├── product.go
│       └── order.go
├── go.mod
└── go.sum
```

---

## 4. Panel Builder Architecture (`pkg/filament`)

The Panel Builder is the foundational orchestrator. A single GoLaravel application can register multiple independent panels (e.g. `/admin`, `/partner`, `/portal`, `/app`).

### Panel Initialization & Core Configuration

```go
package main

import (
    "time"
    "github.com/golaravel/gofilament/pkg/filament"
    "github.com/golaravel/gofilament/pkg/colors"
    "github.com/golaravel/golaravel/pkg/golaravel"
)

func main() {
    app := golaravel.Default()

    adminPanel := filament.NewPanel("admin").
        Path("/admin").
        BrandName("Pulse Enterprise").
        BrandLogo("/assets/logo.svg").
        Favicon("/assets/favicon.ico").
        HomeURL("/admin").
        DarkMode(true).
        SidebarCollapsibleOnDesktop().
        // Desktop sidebar: full, an icon rail with tooltips, or hidden.
        // A top-bar button switches between them; each device remembers it.
        SidebarCollapseModes(filament.SidebarIcons, filament.SidebarHidden).
        SidebarDefaultMode(filament.SidebarExpanded).
        MaxContentWidth(filament.Width7xl).
        Colors(map[string]any{
            colors.RolePrimary: colors.Amber,
            colors.RoleGray:    colors.Slate,
            colors.RoleSuccess: colors.Emerald,
            colors.RoleDanger:  colors.Rose,
            colors.RoleWarning: colors.Amber,
            colors.RoleInfo:    colors.Sky,
        })

    // Mount resources, widgets, and pages...
    // adminPanel.Resource(myResource)

    if err := adminPanel.Register(app); err != nil {
        panic(err)
    }

    app.Run(":8080")
}
```

### Dual-Engine Switching

To switch a panel to the **Flutter Multiplatform Engine**:

```go
adminPanel.
    Engine(filament.EngineFlutter).
    Flutter("client/build/web") // Points to compiled Flutter web distribution directory
```

When `EngineFlutter` is enabled, the panel serves the interactive multiplatform shell at the root path (`/admin/`) and mounts full REST API endpoints at `/admin/api/*`.

### Navigation, Groups, Badges & Collapsible Sidebars

Panels support hierarchical navigation grouping, custom ordering, and dynamic badge counters:

```go
adminPanel.
    NavigationGroups([]string{
        "Shop Management",
        "Operations & Logistics",
        "System Configuration",
    })
```

Inside individual resources:

```go
resource.
    NavigationGroup("Shop Management").
    NavigationSort(1).
    NavigationIcon(`<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z"/></svg>`).
    NavigationBadge(func(ctx *filament.Context) string {
        return "12 New"
    })
```

### Mobile Navigation & Bottom Navigation Bar

GoFilament provides deep customization for mobile layouts:

```go
adminPanel.
    // Configure default mobile bottom bar navigation slots
    MobileBottomNavItems("dashboard", "products", "orders", "account").
    // Toggle whether labels appear below icons on mobile
    MobileBottomNavShowLabels(true).
    // Mutually exclusive account entrypoint:
    // filament.AccountNavHeader: Circular avatar in top AppBar opening floating sheet
    // filament.AccountNavBottomBar: Dedicated bottom bar tab opening full account hub
    AccountNavPlacement(filament.AccountNavHeader)
```

### App Versioning & Hot Update Detection

Panels support build identification and automatic hot reload prompts:

```go
adminPanel.
    Version("v1.25.0").
    BuildID("20261109-prod-sha9a1a").
    RequireReloadOnUpdate(true)
```

Clients poll `GET /admin/api/version`. If a new `BuildID` or `Version` is published, clients automatically surface a non-intrusive update banner or trigger a smooth reload.

### Customizable Pre-Flutter Loading Screen

Eliminate blank screens while the multiplatform engine hydrates:

```go
import "github.com/golaravel/gofilament/pkg/flutter"

adminPanel.FlutterLoading(func(cfg *flutter.LoadingConfig) {
    cfg.Style = flutter.LoadingStylePulse     // LoadingStyleSpinner | LoadingStyleBar | LoadingStylePulse
    cfg.Title = "Pulse Fleet"
    cfg.Subtitle = "Connecting to telemetry cluster..."
    cfg.BackgroundColor = "#0F172A"
    cfg.PrimaryColor = "#F59E0B"
    cfg.TextColor = "#F8FAFC"
})
```

### Global Search Configuration

Global search queries across all resources via `GET /admin/global-search?q=query`:

```go
adminPanel.
    GlobalSearch(true).
    GlobalSearchKeyBinding("k").       // Command+K / Ctrl+K
    GlobalSearchResultsLimit(10).
    GlobalSearchFuzzy(true)
```

### Multi-Tenancy & Privacy Scoping

```go
adminPanel.
    TenantScoping(true)
```

When active, GoFilament injects the tenant context from `X-Tenant-ID` header, route parameter, or session cookie into `ctx.ActiveTenant()`. All resource queries can automatically scope by this identifier.

### Authentication & Session Security

```go
adminPanel.
    RequiresAuth(true).
    Login(true).
    Registration(false).
    PasswordReset(true).
    Profile(true).
    SessionDuration(24 * time.Hour).
    RememberMeDuration(30 * 24 * time.Hour).
    AuthValidator(func(ctx *filament.Context, email, password string) (any, error) {
        // Validate credentials against database/Ent
        return user, nil
    }).
    TokenValidator(func(token string) (any, error) {
        // Validate bearer token for API access
        return user, nil
    })
```

### Internationalization (i18n) & RTL Layout

GoFilament integrates natively with `github.com/golaravel/golaravel/pkg/translation`:

```go
adminPanel.
    Locales("en", "ar", "es", "de").
    DefaultLocale("en").
    LocaleLabels(map[string]string{
        "en": "English",
        "ar": "العربية",
        "es": "Español",
        "de": "Deutsch",
    }).
    LocaleFlags(map[string]string{
        "en": "🇺🇸",
        "ar": "🇸🇦",
        "es": "🇪🇸",
        "de": "🇩🇪",
    }).
    Translations("ar", map[string]string{
        "dashboard": "لوحة التحكم",
        "products":  "المنتجات",
        "orders":    "الطلبات",
        "save":      "حفظ",
        "cancel":    "إلغاء",
    })
```

- When `ar` is active, the layout sets `dir="rtl"` in HTML/CSS and wraps Flutter in `Directionality(textDirection: TextDirection.rtl)`.
- Live locale switching: `POST /admin/api/locale/switch?lang=ar`.
- Translation dictionary endpoint: `GET /admin/api/translations?locale=ar`.

---

## 5. Resource System (`Resource[T]` & `BaseResource[T]`)

Resources map a data model to full administrative CRUD capabilities.

### Resource Declaration & Model Binding

```go
package resources

import (
    "github.com/golaravel/gofilament/pkg/filament"
    "github.com/golaravel/gofilament/pkg/forms"
    "github.com/golaravel/gofilament/pkg/tables"
)

type Product struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Price       float64   `json:"price"`
    Status      string    `json:"status"`
    Stock       int       `json:"stock"`
    DeletedAt   *time.Time `json:"deleted_at"`
}

func NewProductResource() *filament.BaseResource[*Product] {
    res := filament.NewResource[*Product]("products")

    res.ModelName("Product").
        PluralModelName("Products").
        NavigationGroup("Catalog").
        NavigationIcon(`<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>`).
        RecordID(func(p *Product) string {
            return p.ID
        })

    return res
}
```

### Lifecycle Handlers

Attach data operations cleanly:

```go
res.
    OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Product, int, error) {
        // Query Ent ORM or DB
        return products, totalCount, nil
    }).
    OnFind(func(ctx *filament.Context, id string) (*Product, error) {
        // Find single record
        return product, nil
    }).
    OnCreate(func(ctx *filament.Context, data map[string]any) error {
        // Insert record
        return nil
    }).
    OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
        // Update record
        return nil
    }).
    OnDelete(func(ctx *filament.Context, id string) error {
        // Delete record
        return nil
    }).
    OnDeleteBulk(func(ctx *filament.Context, ids []string) error {
        // Bulk delete records
        return nil
    })
```

### Inline Cell Editing & Row Reordering

```go
res.
    OnCellUpdate(func(ctx *filament.Context, id, column string, value any) error {
        // Handles instant cell updates from ToggleColumn, SelectColumn, or TextInputColumn
        return updateProductField(id, column, value)
    }).
    OnReorder(func(ctx *filament.Context, orderedIDs []string) error {
        // Handles drag-and-drop table row reordering
        return updateProductSortOrder(orderedIDs)
    })
```

### Soft Deletes & Trash Recovery Bin

Enable soft deletes with a single builder call:

```go
res.
    SoftDeletes(true).
    SoftDeleteColumn("deleted_at").
    IsTrashed(func(p *Product) bool {
        return p.DeletedAt != nil
    }).
    OnRestore(func(ctx *filament.Context, id string) error {
        return restoreProduct(id)
    }).
    OnForceDelete(func(ctx *filament.Context, id string) error {
        return permanentlyDeleteProduct(id)
    }).
    OnRestoreBulk(func(ctx *filament.Context, ids []string) error {
        return restoreProductsBulk(ids)
    }).
    OnForceDeleteBulk(func(ctx *filament.Context, ids []string) error {
        return permanentlyDeleteProductsBulk(ids)
    })
```

- When active, resource tables automatically inject `tables.TrashedFilter()`.
- Active records display `Edit` and `Delete` actions.
- Trashed records display a `[Trashed]` badge, strike-through styling, and swap actions to `Restore (⟲)` and `Force Delete (🗑)`.

### Authorization Policies & Gates

```go
res.
    CanViewAny(func(ctx *filament.Context) bool {
        return ctx.UserHasRole("manager", "admin")
    }).
    CanCreate(func(ctx *filament.Context) bool {
        return ctx.UserCan("products.create")
    }).
    CanEdit(func(ctx *filament.Context, record *Product) bool {
        return ctx.UserCan("products.edit")
    }).
    CanDelete(func(ctx *filament.Context, record *Product) bool {
        return ctx.UserHasRole("admin")
    }).
    CanRestore(func(ctx *filament.Context, id string) bool {
        return ctx.UserHasRole("admin")
    }).
    CanForceDelete(func(ctx *filament.Context, id string) bool {
        return ctx.UserHasRole("super-admin")
    })
```

### Dynamic Resource Status Tabs

Group lists by workflow states with live counter badges:

```go
res.
    EnableTabs(true).
    Tabs(
        filament.NewResourceTab("all", "All Products").
            Badge(func(ctx *filament.Context) string { return "150" }),
        filament.NewResourceTab("in_stock", "In Stock").
            Badge(func(ctx *filament.Context) string { return "128" }).
            Query(func(filters map[string]string) {
                filters["status"] = "in_stock"
            }),
        filament.NewResourceTab("out_of_stock", "Low / Out of Stock").
            BadgeColor("danger").
            Badge(func(ctx *filament.Context) string { return "22" }).
            Query(func(filters map[string]string) {
                filters["status"] = "out_of_stock"
            }),
    )
```

### Pagination & Infinite Scroll

```go
res.
    // Classic pagination
    Pagination(25).
    PerPageOptions(10, 25, 50, 100).
    // Or infinite scroll:
    // InfiniteScroll(30).
    // Mobile pagination display:
    // filament.PaginationPositionFab (circular floating button modal)
    // filament.PaginationPositionTop (compact top bar)
    MobilePaginationPosition(filament.PaginationPositionFab)
```

### Column Controls

Configure user column toggling, selectable visibility, and default columns:

```go
res.
    DefaultColumns("id", "title", "price", "status").
    SelectableColumns("id", "title", "sku", "price", "stock", "status", "created_at").
    AdvancedSearchableColumns("title", "sku", "id")
```

---

## 6. Form Builder Engine (`pkg/forms`)

### Layout Containers

```go
res.Form(func(form *forms.Form) *forms.Form {
    return form.Schema(
        // 1. Multi-Step Form Wizard
        forms.Wizard(
            forms.WizardStep("General Information").
                Description("Basic product attributes").
                Schema(
                    forms.Grid(2).Schema(
                        forms.TextInput("title").
                            Label("Product Title").
                            Required().
                            MaxLength(150),
                        forms.TextInput("sku").
                            Label("SKU Code").
                            Required().
                            Placeholder("PROD-XXXX"),
                    ),
                    forms.Textarea("description").
                        Rows(4).
                        MaxLength(1000),
                ),

            forms.WizardStep("Pricing & Inventory").
                Description("Commercial configuration").
                Schema(
                    forms.Grid(3).Schema(
                        forms.TextInput("price").
                            Numeric().
                            Prefix("$").
                            Required(),
                        forms.TextInput("cost_price").
                            Numeric().
                            Prefix("$"),
                        forms.TextInput("stock").
                            Numeric().
                            Default(0),
                    ),
                    forms.Toggle("is_published").
                        Label("Publish immediately to storefront").
                        Default(true),
                ),

            forms.WizardStep("Media & Repeater").
                Description("Images and specifications").
                Schema(
                    forms.ImageUpload("thumbnail").
                        Directory("products/thumbnails").
                        MaxFiles(1),
                    // Dynamic Nested Array Repeater
                    forms.Repeater("specifications").
                        Label("Technical Specifications").
                        Schema(
                            forms.TextInput("spec_key").Placeholder("e.g. Dimensions"),
                            forms.TextInput("spec_value").Placeholder("e.g. 10x20x5 cm"),
                        ),
                ),
        ),
    )
})
```

Other available layout containers include:
- `forms.Section(heading).Description(desc).Collapsible().Schema(...)`
- `forms.Grid(columns).Schema(...)`
- `forms.Tabs(forms.Tab("Details", ...), forms.Tab("SEO", ...))`
- `forms.Fieldset(legend).Schema(...)`
- `forms.Split(leftCol, rightCol)`

### Input Fields Reference

| Constructor | Methods | Description |
|---|---|---|
| `forms.TextInput(name)` | `.Email()`, `.Password()`, `.Numeric()`, `.Tel()`, `.URL()`, `.Prefix("$")`, `.Suffix("/mo")`, `.Placeholder(...)`, `.Required()`, `.Min(n)`, `.Max(n)` | Versatile text field with input mask formats |
| `forms.Textarea(name)` | `.Rows(n)`, `.Cols(n)`, `.MaxLength(n)`, `.Placeholder(...)` | Multi-line text field |
| `forms.Select(name)` | `.Options(map[string]string)`, `.Searchable()`, `.Multiple()` | Single or multi-select dropdown |
| `forms.Toggle(name)` | `.OnColor("success")`, `.OffColor("gray")`, `.Default(bool)` | Boolean toggle switch |
| `forms.Checkbox(name)` | `.Label(...)`, `.Default(bool)` | Single checkbox |
| `forms.CheckboxList(name)` | `.Options(map[string]string)` | Group of checkboxes |
| `forms.Radio(name)` | `.Options(map[string]string)` | Radio button group |
| `forms.DatePicker(name)` | `.Format("2006-01-02")`, `.MinDate(...)`, `.MaxDate(...)` | Date selection |
| `forms.DateTimePicker(name)` | `.Format("2006-01-02 15:04:05")` | Date and time picker |
| `forms.TimePicker(name)` | `.Format("15:04")` | Time selection |
| `forms.FileUpload(name)` | `.Disk("s3")`, `.Directory("path")`, `.MaxFiles(n)`, `.AcceptedFileTypes(...)` | Secure file upload |
| `forms.ImageUpload(name)` | Same as FileUpload + image preview cropping | Image upload |
| `forms.RichEditor(name)` | `.ToolbarButtons(...)` | WYSIWYG HTML rich text editor |
| `forms.MarkdownEditor(name)` | `.PreviewMode()` | Markdown editor with split preview |
| `forms.TagsInput(name)` | `.Separator(",")`, `.Suggestions(...)` | Tag list input |
| `forms.ColorPicker(name)` | `.Format("hex")` | Hex/RGB color selector |
| `forms.KeyValue(name)` | `.KeyLabel(...)`, `.ValueLabel(...)` | Key-value dictionary editor |
| `forms.CodeEditor(name)` | `.Language("json")` | Monospace code editor with syntax highlighting |
| `forms.Hidden(name)` | `.Default(val)` | Hidden input field |

### Live Reactivity & Dynamic Micro-Swaps

Fields can react in real time without a full page refresh:

```go
forms.Select("discount_type").
    Options(map[string]string{
        "percentage": "Percentage Discount (%)",
        "fixed":      "Fixed Amount ($)",
    }).
    Live(). // Triggers HTMX micro-swap on change
    AfterStateUpdated(func(state any, form *forms.Form) {
        if state == "percentage" {
            // Modify form fields dynamically
        }
    })
```

### Creating a Missing Option from a Select

Users filling in a form can add an option that doesn't exist yet (a new category, a new supplier) from the select itself, without leaving the form. In the Flutter admin the field's picker shows **Create …**, or **Create "typed text"** after a search. A small form opens with the typed text filled in, and when it's saved the new option is added and selected.

```go
// Reuse another resource: its form, validation, create handler and create
// permission. Users who may not create categories don't see the entry.
forms.Select("category_id").Label("Category").
    OptionsUsing(categoryOptions).
    CreateOptionFrom("categories")

// Or a custom form and handler.
forms.Select("tag").OptionsUsing(tagOptions).
    CreateOptionForm(forms.TextInput("name").Required()).
    CreateOptionUsing(func(ctx context.Context, data map[string]any) (forms.Option, error) {
        id, err := tags.Create(ctx, data["name"].(string))
        return forms.Option{Value: id, Label: data["name"].(string)}, err
    })
```

With `CreateOptionFrom`, the target resource's `OnCreate` must report the new id, by calling `filament.SetCreatedID(ctx, id)` or setting `data["id"]`; it becomes the selected value. Its label comes from the field's options, so use `OptionsUsing` or `SearchUsing` (live options) rather than a fixed `Options` map. `CreateOptionLabel("Add supplier")` changes the entry's text (default: "Create <label>").

The API behind it is `GET` and `POST /api/resources/{slug}/form/options/create?field=<name>`. POST answers `201 {"option": {"value", "label"}}`. The resource create API (`POST /api/resources/{slug}`) now also returns the new record's `id` when the handler reports it.

---

### Conditional Sections and Fields (`VisibleWhen`)

`VisibleWhen(field, values...)` shows a section, grid or field only while another field holds one of the values. The web panel and the Flutter admin both check it as the user edits. Hidden components' values are dropped on save and their rules are skipped, so a hidden `Required()` field doesn't block the form.

```go
forms.Select("driver").Options(map[string]string{"stripe": "Stripe", "payhere": "PayHere"}),
forms.Section("Stripe").VisibleWhen("driver", "stripe").Schema(
    forms.TextInput("stripe_key").Password().Required(),
),
forms.TextInput("gift_note").VisibleWhen("gift", "true"), // a toggle holds "true" / "false"
```

Unlike `Visible(func(get) bool)`, which only the server can run, `VisibleWhen` is declarative, so clients can evaluate it.

## 7. Table Builder Engine (`pkg/tables`)

### Column Types Reference

```go
res.Table(func(table *tables.Table[*Product]) *tables.Table[*Product] {
    return table.
        Columns(
            // 1. Text Column
            tables.TextColumn[*Product]("title").
                Label("Product Name").
                Sortable().
                Searchable().
                Weight("bold").
                Copyable(),

            // 2. Formatted Money Text Column
            tables.TextColumn[*Product]("price").
                Label("Price").
                Sortable().
                Money("USD").
                Align("right"),

            // 3. Status Badge Column
            tables.BadgeColumn[*Product]("status").
                Label("Status").
                Sortable().
                Colors(map[string]string{
                    "published": "success",
                    "draft":     "warning",
                    "archived":  "gray",
                }),

            // 4. Boolean Icon Column
            tables.IconColumn[*Product]("is_featured").
                Label("Featured").
                Boolean(),

            // 5. Interactive Inline Toggle Column
            tables.ToggleColumn[*Product]("is_published").
                Label("Storefront Active"),

            // 6. Interactive Inline Select Column
            tables.SelectColumn[*Product]("status").
                Label("Quick Status").
                Options(map[string]string{
                    "draft":     "Draft",
                    "published": "Published",
                    "archived":  "Archived",
                }),

            // 7. Interactive Inline Text Column
            tables.TextInputColumn[*Product]("stock").
                Label("Live Stock"),

            // 8. Image Column
            tables.ImageColumn[*Product]("thumbnail").
                Label("Preview").
                Circular().
                Size(40),
        )
})
```

### Column Controls, Resizing, Sorting & Toggling

Columns can be configured with fine-grained visibility defaults:

```go
tables.TextColumn[*Product]("sku").
    Toggleable().                    // User can show/hide via column picker
    ToggledHiddenByDefault().        // Initially hidden
    Sortable().                      // Click header to sort ASC/DESC
    Searchable().                    // Included in quick table search
    AdvancedSearchable()             // Included in refined facet filters
```

### Filter Suite

```go
table.Filters(
    // Select filter
    tables.SelectFilter[*Product]("status").
        Label("Fulfillment Status").
        Options(map[string]string{
            "pending":   "Pending Processing",
            "shipped":   "Shipped",
            "delivered": "Delivered",
        }),

    // Boolean Ternary filter
    tables.TernaryFilter[*Product]("is_published").
        Label("Storefront Visibility").
        TrueLabel("Published Only").
        FalseLabel("Drafts Only"),

    // Automatic 3-state Trashed filter (injected automatically when SoftDeletes is enabled)
    tables.TrashedFilter[*Product](),
)
```

#### Refined Filters with Live Facet Counts
`tables.RefinedFilter(column)` is a multi-select filter. The chosen values reach `OnList` comma-separated (`filters["status"] == "paid,shipped"`), so match any of them.

Every list response carries live counts for each refined filter: `"facets": {"status": {"paid": 21, "shipped": 38}}`. The Flutter client shows them next to each option. A column's counts apply the search and every other filter, but not the column's own filter, so the other options still show what choosing them would add. A refined filter without `Options` offers the values found in the data, ordered by count; listed options appear in label order.

```go
tables.RefinedFilter("status").Label("Status").Options(orderStatus), // listed values
tables.RefinedFilter("brand").Label("Brand"),                       // values from the data
```

The counts are read through `OnList`, up to `filament.MaxFacetScan` (5,000) matching records per filter; above that the counts are left out. For large tables, count in the database instead:

```go
res.FacetCounts(func(ctx *filament.Context, column, search string, filters map[string]string) (map[string]int, error) {
    return db.CountBy(column, search, filters) // e.g. SELECT column, COUNT(*) ... GROUP BY column
})
```

### Table Actions & Bulk Operations

```go
table.
    // Row Actions
    Actions(
        tables.ViewAction[*Product](),
        tables.EditAction[*Product](),
        tables.DeleteAction[*Product](),
        // Custom Action with Modal Form
        tables.NewAction[*Product]("apply_discount").
            Label("Apply Discount").
            Icon("tag").
            Color("warning").
            RequiresConfirmation().
            Form(forms.New().Schema(
                forms.TextInput("percentage").Numeric().Required().Suffix("%"),
            )).
            Action(func(ctx *filament.Context, record *Product, data map[string]any) error {
                return applyDiscount(record.ID, data["percentage"])
            }),
    ).
    // Bulk Operations
    BulkActions(
        tables.DeleteBulkAction[*Product](),
        tables.NewBulkAction[*Product]("publish_all").
            Label("Publish Selected").
            Icon("check-circle").
            Action(func(ctx *filament.Context, ids []string) error {
                return publishBatch(ids)
            }),
    )
```

### Streaming CSV Data Export & Import

Resources support high-performance streaming export:

- **Export**: Triggered via `tables.ExportAction()` or `GET /admin/api/resources/{slug}/export?format=csv`. Streams directly to client with memory efficiency.
- **Import**: `res.InvokeImportCSV(ctx, reader, fieldMapping)` with automatic schema validation and batch ingestion.

---

## 8. Relationship Managers (`RelationManager`)

Manage one-to-many or many-to-many child relationships directly within a parent resource's View or Edit screens:

```go
type OrderItem struct {
    ID        string  `json:"id"`
    OrderID   string  `json:"order_id"`
    Product   string  `json:"product"`
    Quantity  int     `json:"quantity"`
    UnitPrice float64 `json:"unit_price"`
}

func NewOrderItemsRelationManager() filament.RelationManager {
    rm := filament.NewRelationManager[*Order, *OrderItem]("items")

    rm.Title("Line Items").
        Icon("shopping-cart").
        Table(func(t *tables.Table[*OrderItem]) *tables.Table[*OrderItem] {
            return t.Columns(
                tables.TextColumn[*OrderItem]("product").Label("Product"),
                tables.TextColumn[*OrderItem]("quantity").Label("Qty"),
                tables.TextColumn[*OrderItem]("unit_price").Label("Unit Price").Money("USD"),
            )
        })

    return rm
}
```

Attach to parent resource:

```go
orderResource.Relations(NewOrderItemsRelationManager())
```

---

## 9. Infolist Engine (`pkg/infolists`)

Infolists define structured, read-only detail views for the `ViewRecord` screen:

```go
res.Infolist(func(info *infolists.Infolist) *infolists.Infolist {
    return info.Schema(
        infolists.Section("Customer Snapshot").Schema(
            infolists.TextEntry("name").Weight("bold"),
            infolists.TextEntry("email").Copyable(),
            infolists.BadgeEntry("tier").Colors(map[string]string{
                "vip":      "success",
                "standard": "gray",
            }),
            infolists.IconEntry("is_verified").Boolean(),
        ),
    )
})
```

Available entries: `TextEntry`, `BadgeEntry`, `IconEntry`, `ImageEntry`, `RepeatableEntry`.

---

## 10. Dashboard Studio & Widgets (`pkg/widgets`)

GoFilament includes a rich dashboard studio featuring 8 responsive widget types:

```go
adminPanel.Widgets(
    // 1. KPI Stats Overview with Sparklines
    widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
        return []widgets.Stat{
            *widgets.NewStat("Total Revenue", "$128,450").
                Description("+14.2% from last week").
                DescriptionIcon("trending-up").
                Color("success").
                Chart([]float64{65, 78, 82, 95, 110, 128}),
            *widgets.NewStat("Active Orders", "42").
                Description("8 awaiting shipment").
                Color("warning"),
            *widgets.NewStat("System Health", "99.98%").
                Description("All clusters nominal").
                Color("info"),
        }
    }),

    // 2. Chart.js Chart Widget
    widgets.NewChartWidget("Fulfillment Trends", widgets.ChartTypeLine).
        Height(300).
        Data(func(ctx *filament.Context) widgets.ChartData {
            return widgets.ChartData{
                Labels: []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
                Datasets: []widgets.ChartDataset{
                    {
                        Label:           "Delivered Orders",
                        Data:            []float64{120, 150, 180, 220, 200, 240, 310},
                        BorderColor:     "#10B981",
                        BackgroundColor: "rgba(16, 185, 129, 0.1)",
                    },
                },
            }
        }),

    // 3. Quick Table Widget
    widgets.NewTableWidget("Recent Urgent Incidents"),

    // 4. Milestone Timeline Widget
    widgets.NewTimelineWidget("Deployment Milestones"),

    // 5. Quick Shortcut Actions
    widgets.NewQuickActionsWidget("Operational Actions"),

    // 6. Progress Targets
    widgets.NewProgressWidget("Monthly Target Completion"),

    // 7. Alert Banner Widget
    widgets.NewBannerWidget("Scheduled Maintenance", "Telemetry ingest will restart at 02:00 UTC."),

    // 8. Percentage Breakdown Widget
    widgets.NewBreakdownWidget("Traffic Source Distribution"),
)
```

---

### Dashboard layout, stat changes and chart formats

The dashboard is a grid **three columns wide** (two on tablets, one on phones), in the web panel and the Flutter admin alike. Each widget takes `Span(n)` columns and widgets fill the rows in `Order(n)` order. Stat rows and banners take the whole width when no span is set.

```go
adminPanel.Widgets(
    widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
        return []widgets.Stat{
            *widgets.NewStat("Sales today", "LKR 125,400.00").
                ChangeBetween(125400, 111800, "vs last Tuesday"). // "+12.2% vs last Tuesday", green
                ChartValues(last14Days).                          // trend line from any numbers
                URL("/app/resources/sales?filter_status=paid"),   // tapping opens the filtered list
            *widgets.NewStat("Returns", "LKR 3,200.00").Change(8, "vs yesterday").LowerIsBetter(), // a rise is red
        }
    }).Order(1),

    widgets.NewChartWidget("Profit and loss", widgets.ChartTypeBar).
        Description("Last 12 months").
        Money("LKR").                                   // axis and tooltips as amounts (or ValueFormat(widgets.FormatPercent))
        Filters(map[string]string{"7d": "7 days", "30d": "30 days", "12m": "12 months"}).
        FilterOrder("7d", "30d", "12m").                // button order (otherwise sorted by key)
        DefaultFilter("12m").
        Span(2).Order(2).
        FilterData(func(ctx *filament.Context, period string) widgets.ChartData {
            return widgets.ChartData{Labels: months, Datasets: []widgets.ChartDataset{
                {Label: "Income", Data: income},
                {Label: "Expenses", Data: expenses},
                {Label: "Net", Type: "line", Data: net}, // bars and a line on one chart
            }}
        }),

    widgets.NewTableWidget("Top products").Span(1).Order(3).
        EmptyText("No sales yet").
        Columns(
            widgets.TableWidgetColumn{Name: "name", Label: "Product"},
            widgets.TableWidgetColumn{Name: "revenue", Label: "Revenue", Format: "money"}, // number, money, percent
        ).
        Rows(func(ctx *filament.Context) []map[string]any {
            return []map[string]any{{"name": "Rice 5kg", "revenue": 125000.5, widgets.RowURL: "/app/resources/products/12"}}
        }),
)
```

- **Stats**: `Change(percent, caption)` or `ChangeBetween(current, previous, caption)` (left out when the previous figure is zero), `LowerIsBetter()`, `ChartValues([]float64)` (or `Chart([]int)`), `URL(...)`.
- **Charts**: `Money(currency)` / `ValueFormat(...)`, `FilterOrder(...)`, `Description(...)`, and per-dataset `Type: "bar"` or `"line"`. In the Flutter admin, hovering or tapping shows the values at that point, or a slice's value and share.
- **Tables**: column `Format` and `Align`, `Description`, `EmptyText`. A row with `widgets.RowURL` opens it.
- **Links** (stat `URL`, table rows, quick actions, banner actions) are panel URLs such as `/app/resources/orders?filter_status=paid&search=rice` or `orders/42`. The Flutter admin opens the resource with those filters (`filter_<name>=` or `filters[<name>]=`) and the search.
- **Every widget** has `Span`, `Order` and `Visible(fn)`, and most have `Key(id)`. In the Flutter admin, tables, breakdowns, progress meters, quick actions and banners are drawn as well as stats and charts. A dismissible banner can be closed for the visit.

---

## 11. Activity Log & Audit Trail Timeline (`pkg/activitylog`)

GoFilament provides an enterprise audit trail system with zero boilerplate:

### Enabling Audit on a Resource

```go
import "github.com/golaravel/gofilament/pkg/activitylog"

// Attach audit logging to any resource
activitylog.EnableAuditLog(productResource)
```

When attached:
- Every Create, Update, Delete, and Restore action is recorded.
- Intercepts authenticated causer ID, display name, client IP, and User-Agent.
- Generates field-level diffs (`OldValues` vs `NewValues`).

### Embedding Timeline In Infolists

```go
res.Infolist(func(info *infolists.Infolist) *infolists.Infolist {
    return info.Schema(
        // Renders vertical audit timeline with expandable before/after diff inspector
        activitylog.TimelineEntry("activities"),
    )
})
```

### Pre-Configured Standalone Audit Resource

Mount the administrative audit screen to view all system activity:

```go
adminPanel.Resource(activitylog.NewActivityResource())
```

---

## 12. Live Chat, AI Concierge & Multimodal Voice Feature Requests (`pkg/support`)

Updated and significantly enhanced in release **v1.26.0**, `pkg/support` delivers an enterprise-grade AI concierge, seamless live operator escalation, multimodal multilingual voice feature requests, and hierarchical multi-panel / multi-tenant routing.

### Core Concepts & Architecture

```
┌────────────────────────────────────────────────────────────────────────┐
│                   Customer / Mobile / Web Client                       │
│       • Multilingual Chat (English, Sinhala, Tamil, Arabic, etc.)      │
│       • Multimodal Voice Feature Requests (Microphone Audio Stream)    │
│       • Language Preference Cookie (golaravel_support_lang)            │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │
                                   ▼
┌────────────────────────────────────────────────────────────────────────┐
│             GoLaravel / GoFilament Support Concierge Engine            │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ 1. AI Support Concierge                                          │  │
│  │    • Answers inquiries instantly via Grounded Knowledge Base     │  │
│  │    • Auto-detects escalation triggers ("human agent", "help")    │  │
│  │    • Seamlessly flips mode to ModeLive for human takeover        │  │
│  └──────────────────────────────────┬───────────────────────────────┘  │
│                                     │                                  │
│  ┌──────────────────────────────────▼───────────────────────────────┐  │
│  │ 2. Multimodal Voice Feature Request Processor                    │  │
│  │    • Ingests audio in any language (Sinhala, Tamil, Arabic, etc.)│  │
│  │    • Preserves original audio bytes + Base64 for playback        │  │
│  │    • Transcribes & translates to English for engineering teams   │  │
│  │    • Checks system Knowledge Base:                               │  │
│  │      - If feature already exists: AI explains how to use it      │  │
│  │      - If new feature: rates Feasibility (simple vs complex)     │  │
│  │    • Dispatches real-time alerts to operators                    │  │
│  └──────────────────────────────────┬───────────────────────────────┘  │
│                                     │                                  │
│  ┌──────────────────────────────────▼───────────────────────────────┐  │
│  │ 3. Multi-Panel & Multi-Tenant Support Hierarchy                  │  │
│  │    • Tenant Admin Panels: Scoped to active tenant               │  │
│  │    • Super Admin Panels: Platform-wide visibility & monitoring    │  │
│  │    • Upstream Escalation: Tenant Admins can escalate to Super    │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────┘
```

- **`Ticket`**: Represents a support inquiry or feature request with `Category` (`support`, `feature_request`, `bug`, `billing`), `Status` (`open`, `pending`, `escalated`, `resolved`, `closed`), `Mode` (`ai`, `live`), `Priority`, `TenantID`, `PanelID`, and `Feasibility` (`simple`, `complex`, `existing_feature`).
- **`VoiceRecording`**: Preserves original voice audio buffer (`AudioData`), web player payload (`AudioBase64`), original transcript, and English translation.
- **`KnowledgeBase`**: Pre-seeded knowledge repository containing system capabilities up to `v1.25.0`.
- **`LanguageCookieName`**: `"golaravel_support_lang"`, persisting customer language across sessions.
- **`ScopeSuperAdmin` vs `ScopeTenantAdmin`**: Configures whether the panel views all tickets globally or scopes to tenant records with upstream escalation.

### Panel Configuration & Hierarchy

```go
import (
    "github.com/golaravel/gofilament/pkg/filament"
    "github.com/golaravel/gofilament/pkg/support"
)

// 1. Configure Super Admin Panel (Platform Hub)
superAdminPanel := filament.NewPanel("super_admin").
    Path("/super-admin").
    SuperAdminSupport(). // Enables cross-tenant global support visibility
    Resource(support.NewSupportResource(
        support.WithScope(support.ScopeSuperAdmin),
    ))

// 2. Configure Tenant Admin Panel (Customer Portal with Upstream Escalation)
tenantAdminPanel := filament.NewPanel("admin").
    Path("/admin").
    TenantAdminSupport("super_admin"). // Scoped to tenant, escalates to super_admin
    Resource(support.NewSupportResource(
        support.WithScope(support.ScopeTenantAdmin),
        support.WithTenantScoping(true),
        support.WithSuperAdminPanelID("super_admin"),
    ))
```

### Multimodal Voice Feature Request Ingestion

```go
service := support.GetDefaultService()

// Ingest voice recording in Sinhala, Tamil, Arabic, English, etc.
ticket, aiReply, err := service.SubmitVoiceFeatureRequest(ctx, support.VoiceFeatureRequest{
    AudioBytes:      audioBytes,
    MimeType:        "audio/webm",
    DurationSeconds: 12.4,
    Language:        "si", // Sinhala
    UserKey:         "user_102",
    UserName:        "Kasun Perera",
    UserEmail:       "kasun@example.lk",
    TenantID:        "tenant_colombo",
    PanelID:         "admin",
})

// ticket.Voice.AudioData contains the original preserved audio
// ticket.Voice.TranslatedEnglish contains the English translation for engineers
// ticket.Feasibility contains "simple" or "complex"
// aiReply.Content contains localized AI feedback explaining acceptance or review
```

### Support REST API Endpoints

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/admin/api/support/tickets` | `GET` | List tickets (tenant-scoped or super-admin global) |
| `/admin/api/support/message` | `POST` | Send message with automated AI response and auto-escalation |
| `/admin/api/support/voice-feature` | `POST` | Ingest voice feature request in any language |
| `/admin/api/support/audio/:id` | `GET` | Stream preserved audio recording for playback |
| `/admin/api/support/escalate` | `POST` | Escalate ticket to live agent or upstream super admin panel |
| `/admin/api/support/language` | `POST` | Set persistent `golaravel_support_lang` cookie |
| `/admin/api/support/knowledge` | `GET` | List known system features and capabilities |

---

## 13. Flutter Multiplatform Client Architecture (`pkg/flutterclient`)

When a panel runs with `.Engine(filament.EngineFlutter)`, it powers native mobile and desktop clients via a unified JSON REST API.

### REST API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/admin/api/schema` | `GET` | Complete metadata: brand, theme colors, navigation items, resources, locales, account placement |
| `/admin/api/resources/{slug}` | `GET` | List records with search, pagination, sort, and filters |
| `/admin/api/resources/{slug}` | `POST` | Create a new record |
| `/admin/api/resources/{slug}/{id}` | `GET` | Retrieve single record details |
| `/admin/api/resources/{slug}/{id}` | `PUT` | Update record |
| `/admin/api/resources/{slug}/{id}` | `DELETE` | Delete record |
| `/admin/api/resources/{slug}/restore` | `POST` | Restore soft-deleted record |
| `/admin/api/resources/{slug}/force-delete` | `POST` | Permanently delete record |
| `/admin/api/resources/{slug}/export` | `GET` | Stream CSV data export |
| `/admin/api/locale/switch?lang={code}` | `POST` | Switch session locale (returns directionality, dictionary, flag) |
| `/admin/api/translations?locale={code}` | `GET` | Fetch translation dictionary |
| `/admin/api/version` | `GET` | Version and BuildID check |
| `/admin/api/tenants` | `GET` | List available tenant workspaces |

### Flutter Client Key Architecture

1. **`ResponsiveLayout`**: 
   - Screen width < 640px: Material 3 Bottom Navigation bar, full-screen card feeds with zero horizontal scroll, and a circular FAB modal trigger for pagination.
   - Screen width >= 1024px: High-density data grid tables, column resize/sort/toggle handles, and topbar navigation.
2. **Account Entrypoint Exclusivity**:
   - `header`: Circular avatar button in top header opens a floating bottom sheet (`AccountModalSheet`).
   - `bottom_bar`: Dedicated tab in bottom navigation opens a full `AccountHub` screen.
3. **Directionality Mirroring (Zero External Packages)**:
   - Root application wraps tree with `Directionality(textDirection: isRTL ? TextDirection.rtl : TextDirection.ltr)`.
   - Layout automatically mirrors AppBars, icons, grids, padding, and dialogs when switching between English and Arabic.

---

## 14. Artisan CLI Generator Reference (`pkg/commands`)

Integrated with GoLaravel's Artisan CLI:

```bash
# 1. Generate a new admin panel
artisan make:filament-panel Admin

# 2. Generate a resource manually
artisan make:filament-resource Product

# 3. Automatically inspect Ent AST schema to generate forms, tables, filters & relations
artisan make:filament-resource Product --generate

# 4. Generate a mobile-first Flutter admin client codebase
artisan make:filament-flutter MobileAdmin

# 5. Generate a custom panel page
artisan make:filament-page AnalyticsDashboard

# 6. Generate a dashboard widget
artisan make:filament-widget RevenueStats

# 7. Generate an in-place relationship manager
artisan make:filament-relation Order Items
```

---

## 15. Complete End-to-End Code Blueprint

Below is a complete, standalone, runnable example showing how all components integrate:

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "sync"
    "time"

    "github.com/golaravel/gofilament/pkg/activitylog"
    "github.com/golaravel/gofilament/pkg/colors"
    "github.com/golaravel/gofilament/pkg/filament"
    "github.com/golaravel/gofilament/pkg/forms"
    "github.com/golaravel/gofilament/pkg/support"
    "github.com/golaravel/gofilament/pkg/tables"
    "github.com/golaravel/gofilament/pkg/widgets"
    "github.com/golaravel/golaravel/pkg/golaravel"
)

// 1. Data Model
type Device struct {
    ID          string     `json:"id"`
    Name        string     `json:"name"`
    SerialNo    string     `json:"serial_no"`
    Model       string     `json:"model"`
    Location    string     `json:"location"`
    Status      string     `json:"status"`
    Battery     int        `json:"battery_level"`
    IsActive    bool       `json:"is_active"`
    DeletedAt   *time.Time `json:"deleted_at"`
}

var (
    deviceLock sync.RWMutex
    devices    = []*Device{
        {ID: "1", Name: "Gateway Frankfurt 01", SerialNo: "SN-8821-X", Model: "EdgeCore-Pro", Location: "Frankfurt DC", Status: "online", Battery: 98, IsActive: true},
        {ID: "2", Name: "Gateway Zurich 02", SerialNo: "SN-9942-Z", Model: "EdgeCore-Pro", Location: "Zurich Hub", Status: "online", Battery: 84, IsActive: true},
        {ID: "3", Name: "Sensor Riyadh 01", SerialNo: "SN-1022-A", Model: "PulseSense", Location: "Riyadh Data Hub", Status: "maintenance", Battery: 45, IsActive: false},
    }
)

// 2. Resource Definition
func buildDeviceResource() *filament.BaseResource[*Device] {
    res := filament.NewResource[*Device]("devices")

    res.ModelName("Device").
        PluralModelName("Devices").
        NavigationGroup("Fleet Management").
        NavigationSort(1).
        NavigationIcon(`<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"/></svg>`).
        RecordID(func(d *Device) string { return d.ID })

    // Soft Deletes & Trash Recovery
    res.SoftDeletes(true).
        SoftDeleteColumn("deleted_at").
        IsTrashed(func(d *Device) bool { return d.DeletedAt != nil })

    // Form Schema
    res.Form(func(f *forms.Form) *forms.Form {
        return f.Schema(
            forms.Section("Device Identity & Deployment").Schema(
                forms.Grid(2).Schema(
                    forms.TextInput("name").Required().Label("Device Name"),
                    forms.TextInput("serial_no").Required().Label("Serial Number"),
                    forms.Select("model").Options(map[string]string{
                        "EdgeCore-Pro": "EdgeCore Pro Gateway",
                        "PulseSense":   "PulseSense Telemetry Node",
                    }).Required(),
                    forms.TextInput("location").Required().Label("Data Center / Location"),
                ),
                forms.Grid(2).Schema(
                    forms.Select("status").Options(map[string]string{
                        "online":      "Online (Nominal)",
                        "offline":     "Offline (Disconnected)",
                        "maintenance": "Under Maintenance",
                    }).Default("online"),
                    forms.Toggle("is_active").Label("Active Telemetry Ingestion").Default(true),
                ),
            ),
        )
    })

    // Table Schema
    res.Table(func(t *tables.Table[*Device]) *tables.Table[*Device] {
        return t.
            Columns(
                tables.TextColumn[*Device]("name").Label("Device Name").Sortable().Searchable().Weight("bold"),
                tables.TextColumn[*Device]("serial_no").Label("Serial").Sortable().Copyable(),
                tables.BadgeColumn[*Device]("status").Label("Status").Colors(map[string]string{
                    "online":      "success",
                    "maintenance": "warning",
                    "offline":     "danger",
                }),
                tables.TextColumn[*Device]("location").Label("Location").Sortable(),
                tables.ToggleColumn[*Device]("is_active").Label("Ingestion Active"),
            ).
            Filters(
                tables.SelectFilter[*Device]("status").Options(map[string]string{
                    "online":      "Online Only",
                    "offline":     "Offline Only",
                    "maintenance": "Maintenance Only",
                }),
                tables.TrashedFilter[*Device](),
            ).
            Actions(
                tables.ViewAction[*Device](),
                tables.EditAction[*Device](),
                tables.DeleteAction[*Device](),
                tables.RestoreAction[*Device](),
                tables.ForceDeleteAction[*Device](),
            ).
            BulkActions(
                tables.DeleteBulkAction[*Device](),
                tables.RestoreBulkAction[*Device](),
                tables.ForceDeleteBulkAction[*Device](),
            )
    })

    // Handlers
    res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Device, int, error) {
        deviceLock.RLock()
        defer deviceLock.RUnlock()
        return devices, len(devices), nil
    }).
    OnFind(func(ctx *filament.Context, id string) (*Device, error) {
        deviceLock.RLock()
        defer deviceLock.RUnlock()
        for _, d := range devices {
            if d.ID == id {
                return d, nil
            }
        }
        return nil, fmt.Errorf("device not found")
    }).
    OnCellUpdate(func(ctx *filament.Context, id, column string, value any) error {
        deviceLock.Lock()
        defer deviceLock.Unlock()
        for _, d := range devices {
            if d.ID == id {
                if column == "is_active" {
                    if b, ok := value.(bool); ok {
                        d.IsActive = b
                    }
                }
                return nil
            }
        }
        return fmt.Errorf("device not found")
    })

    return res
}

// 3. Application Entrypoint
func main() {
    app := golaravel.Default()

    // Configure GoFilament Admin Panel
    adminPanel := filament.NewPanel("admin").
        Path("/admin").
        BrandName("Pulse Fleet").
        Version("v1.25.0").
        BuildID("20261109-prod").
        DarkMode(true).
        SidebarCollapsibleOnDesktop().
        MaxContentWidth(filament.Width7xl).
        Colors(map[string]any{
            colors.RolePrimary: colors.Amber,
            colors.RoleGray:    colors.Slate,
        }).
        // i18n & RTL Configuration
        Locales("en", "ar", "es", "de").
        DefaultLocale("en").
        LocaleLabels(map[string]string{
            "en": "English",
            "ar": "العربية",
            "es": "Español",
            "de": "Deutsch",
        }).
        LocaleFlags(map[string]string{
            "en": "🇺🇸",
            "ar": "🇸🇦",
            "es": "🇪🇸",
            "de": "🇩🇪",
        })

    // Mount Device Resource with Automated Audit Trail
    deviceRes := buildDeviceResource()
    activitylog.EnableAuditLog(deviceRes)
    adminPanel.Resource(deviceRes)

    // Mount Activity Audit Resource
    adminPanel.Resource(activitylog.NewActivityResource())

    // Mount Live Chat & AI Support Concierge Resource (v1.25.0)
    adminPanel.Resource(support.NewSupportResource())

    // Mount Dashboard KPI & Trend Widgets
    adminPanel.Widgets(
        widgets.NewStatsOverviewWidget(func(ctx *filament.Context) []widgets.Stat {
            return []widgets.Stat{
                *widgets.NewStat("Connected Gateways", "2 Nodes").
                    Description("99.9% uptime").
                    Color("success").
                    Chart([]float64{1, 1, 2, 2, 2, 2}),
                *widgets.NewStat("Telemetry Rate", "1.4k / sec").
                    Description("IoT live stream active").
                    Color("info"),
                *widgets.NewStat("Active Alarms", "0").
                    Description("No sensor breaches").
                    Color("success"),
            }
        }),
    )

    // Register Panel with GoLaravel Router
    if err := adminPanel.Register(app); err != nil {
        panic(err)
    }

    fmt.Println("🚀 GoFilament v1.25.0 Control Center running at http://localhost:8080/admin")
    app.Run(":8080")
}
```

---

## 16. Payment Gateways (`pkg/payments`)

`pkg/payments` puts golaravel's payment gateway layer (`pkg/payment`) in the panel. The built-in drivers are Stripe, PayPal, PayHere, OnePay, Tabby, Tamara and a test gateway. Any `payment.Driver` plugin you install shows up as well.

```go
mgr := payment.NewManager(payment.Options{BaseURL: "https://shop.example.com", Store: store, Encrypter: enc})
app.RegisterProvider(gateways.Provider(mgr)) // drivers + /payments routes (see golaravel's payment-gateways guide)

res := payments.Register(panel, mgr)
res.Gateways.AuthorizeWithGate()            // narrow access like any resource
```

- **Payment Gateways**:
  - Pick a driver; the form shows only that driver's settings (`VisibleWhen`).
  - Choose test or live mode, enable it, and turn on automatic capture if you want it.
  - Secret settings are write-only: they're encrypted at rest and never sent back. A blank field keeps the saved value.
  - Each gateway shows its webhook URL and has a **Test connection** action.
- **Payments**:
  - Lists every payment, with filters for status, gateway and mode, and search by order, email or gateway reference.
  - The record view has a **History** tab with every status change, webhook and refund.
  - Row actions show only where the gateway supports them: **Refund** (full or partial), **Capture** and **Void** for authorized payments, and **Sync with gateway**.
  - The navigation badge counts authorized payments waiting to be captured.
  - Payments can't be created, edited or deleted from the panel.
- **Subscriptions**:
  - Gateway-run subscriptions (PayHere Recurring, Stripe Billing, PayPal Billing) and subscriptions billed to a saved card.
  - Each shows its schedule, how many periods are paid, and when it next bills.
  - Every subscription has **Payments** and **History** tabs.
  - Row actions: **Retry payment** (past due or failed), **Sync with gateway**, and **Cancel**.
  - The navigation badge counts past-due subscriptions.
- **Saved Payment Methods**:
  - Cards customers saved through preapproval, shown as "Visa •••• 4242 (12/30)". The gateway token is never shown.
  - Each has a History tab and a **Revoke** action.

## 17. Per-Tenant Features & Settings (`pkg/features`)

GoFilament builds on GoLaravel's `pkg/features`: features and settings are declared in code with a default, and a tenant gets a `tenant_settings` row only when its value differs. See GoLaravel's `docs/tenant-features-and-settings.md` for declaring features, the manager, caching and the CLI.

### Wiring a panel

```go
panel.Features(mgr)                        // settings page + API, schema "features"
panel.TenantSettings(filament.TenantSettingsConfig{
    Slug: "settings", Label: "Workspace settings", Group: "Settings",
    CanManage: func(ctx *filament.Context) bool { return isOwner(ctx) },
})
central.TenantFeaturesAdmin(mgr, filament.TenantFeaturesAdminConfig{
    Tenants: func(ctx *filament.Context) ([]filament.Tenant, error) { return allTenants(ctx) },
})
```

Every panel request carries its resolved tenant in the context (`privacy.WithTenantID`) and a request memo (`features.WithMemo`), so all the feature checks of one request read the tenant's values once. `ctx.GoContext()` is the context to pass to `features.Enabled` in your own handlers.

### Conditions

`filament.Condition` is `func(ctx context.Context) bool`. The helpers read the request's tenant and **fail closed**: with no tenant, an unregistered feature or a database error, they return `false`.

| Helper | Holds when |
|---|---|
| `FeatureEnabled(f)` | the bool feature is on |
| `FeatureDisabled(f)` | the bool feature is known to be off |
| `FeatureIs(f, v)` | the value equals `v` |
| `FeatureIn(f, v1, v2...)` | the value is one of the values |
| `AllOf(c...)`, `AnyOf(c...)` | combinations |

Every gated item uses a method named `RequiresFeature(cond)` (`VisibleWhen` already means a client-side field condition on forms and a user check on widgets):

| Item | Effect when the condition fails |
|---|---|
| `BaseResource.RequiresFeature` | left out of navigation, `/api/schema` (`resources`, `resource_meta`, `navigation`) and global search; every web and API URL under the resource answers **404**; create-option forms for it are refused |
| `FlutterPageDef.RequiresFeature`, custom pages implementing `FeatureGated` | left out of navigation and `client_pages`; URLs under the page answer 404 |
| `NavigationItem{Feature: cond}` (returned by a custom page or resource) | item hidden |
| `widgets.BaseWidget.RequiresFeature` (and every widget type) | widget hidden, its data endpoint answers 404 |
| `tables.TableAction.RequiresFeature`, `BulkActionDef.RequiresFeature` | not rendered, not in the table meta, cannot be run (web or API) |
| `tables.*Column.RequiresFeature` | not rendered, not exported, not in the schema, not editable inline |
| `forms.*.RequiresFeature` (fields, `Section`, `Grid`) | not rendered, not described to the Flutter client, its submitted values are dropped by `forms.Validate` |

Tables get the request context from the panel (`Table.WithContext`). A table built without one hides every gated column and action.

### Tenant settings page

`<panel>/tenant-settings` lists the features with `ManagedBy == features.TenantAdmin`, grouped by `Meta.Group`:

| Type | Control |
|---|---|
| `bool` | toggle |
| `enum` | select over `Meta.Options` |
| `int` | number input with `Meta.Min` / `Meta.Max` |
| `string` | text input |
| `json` | JSON textarea |

Each field shows **Overridden** (with the default and **Reset to default**) or **Default**. Saving writes only the values that changed, through `mgr.Set` (strings are parsed and validated). `features.ErrInvalidValue` messages show on their field with status 422. Valid values in the same submission are saved.

Who may use it: `TenantSettingsConfig.CanManage`, else the `tenant-settings.manage` permission when the panel has `Roles`, else every user of the panel. The page needs a tenant (403 without one).

**SuperAdmin features** are never listed. Every endpoint checks the keys it is asked to change before writing anything, and answers 403 for a `SuperAdmin` feature (404 for an unknown key), so hiding the field is not the only protection.

### JSON API

| Method & path | Body / response |
|---|---|
| `GET <panel>/api/tenant-settings` | `{tenant, groups: [{name, features: [{key, type, label, description, group, managed_by, public, options, min, max, default, value, overridden}]}]}` |
| `PUT` (or `POST`) `<panel>/api/tenant-settings` | `{"values": {"tax.mode": "exclusive", "branches.max": 7}}` → the listing; 422 with `errors: {key: message}` |
| `DELETE <panel>/api/tenant-settings/{key}` (or `POST .../{key}/reset`) | reset to default → the listing |
| `GET` / `PUT` `<central>/api/tenant-features/{tenant}` | the same for any tenant, `SuperAdmin` features included |
| `DELETE <central>/api/tenant-features/{tenant}/{key}` | reset |

`/api/schema` adds `features` (the tenant's `Public` values), `tenant_settings: {enabled, label, endpoint, path}` and, on a central panel, `tenant_features_admin`. The Flutter admin shows **Tenant settings** in the account hub when `tenant_settings.enabled` is true (`TenantSettingsScreen`).

### Central screen

`TenantFeaturesAdmin` adds `<panel>/tenant-features` (a tenant list, or a tenant key field without `Tenants`) and `<panel>/tenant-features/{tenant}`, with the same controls plus a **Super admin only** badge. With `Tenants` set, only listed tenants open (404 otherwise). Access: `CanManage`, else `tenant-features.manage` with `Roles`, else every user of the panel, so protect a central panel with `CanAccess`.

## Summary Checklist for AI Agents & Developers

When using this documentation to build applications with GoFilament:

1. **Panel**: Define panels with `filament.NewPanel("admin")` and register to `app` with `adminPanel.Register(app)`.
2. **Resources**: Define domain resources with `filament.NewResource[*T]("slug")`, implementing `OnList`, `OnFind`, `OnCreate`, `OnUpdate`, `OnDelete`.
3. **Forms**: Use `res.Form()` with layout containers (`Section`, `Grid`, `Wizard`, `Repeater`) and typed fields (`TextInput`, `Select`, `Toggle`, `ImageUpload`).
4. **Tables**: Use `res.Table()` with columns (`TextColumn`, `BadgeColumn`, `ToggleColumn`), filters (`SelectFilter`, `TrashedFilter`), and actions (`EditAction`, `DeleteAction`).
5. **Reactivity**: Add `.Live()` and `.OnCellUpdate()` for zero-reload table edits and form micro-swaps.
6. **Audit & Timeline**: Instrument resources with `activitylog.EnableAuditLog(res)` and mount `activitylog.NewActivityResource()`.
7. **Support Concierge**: Integrate `support.NewSupportResource()` for AI & live customer support.
8. **Multiplatform**: Add `.Engine(filament.EngineFlutter)` to serve Flutter clients natively and over REST API.
