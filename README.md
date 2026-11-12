# GoFilament 🚀

> **High-Performance, Type-Safe Full-Stack Admin Panel & Application Toolkit for GoLaravel & Ent ORM**
>
> 📖 **Comprehensive AI & Developer Documentation**: See [DOCUMENTATION.md](DOCUMENTATION.md) for the complete API reference, dual-engine architecture guide, and end-to-end code blueprints.

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
| `forms.RichEditor(name)` | Full WYSIWYG rich text editor with toolbar (bold, lists, headings, links, code) |
| `forms.MarkdownEditor(name)` | Markdown editor with toolbar and real-time Write / Preview tabbed interface |
| `forms.Select(name)` | Dropdown options with searchability and multi-select |
| `forms.Toggle(name)` | Boolean toggle switch |
| `forms.DatePicker(name)` | Date and datetime-local picker |
| `forms.FileUpload(name)` | Drag-and-drop file upload with live previews, multi-file and size limits |
| `forms.ImageUpload(name)` | Specialized image uploader with avatar circular mode and live preview cards |
| `forms.ColorPicker(name)` | Visual color picker with palette presets, native wheel, and hex sync |
| `forms.TagsInput(name)` | Interactive tag pills with autocomplete suggestions and key triggers |
| `forms.Grid(cols)` | Responsive multi-column layout grid |
| `forms.Section(title)` | Bordered card section with collapsible support |
| `forms.Tabs(...)` | Tabbed form schemas |
| `forms.Wizard(...)` | Sequential step-by-step form wizard with progress indicators |
| `forms.Repeater(name)` | Dynamic nested arrays with reordering (`↑`/`↓`), collapsing, and grid support |

### 🛡️ Server-side validation and input whitelisting

Every create, update, inline edit, relation form and row-action form is checked
on the server against the form you declared, in both engines and in the API
(`forms.Validate`). Your handler receives **only the form's fields**:

- keys that are not fields, `Disabled()` fields, fields hidden by `Visible(...)`
  and fields whose `Dehydrated(...)` returns false are dropped;
- `Required()` / `RequiredOnCreate()`, `MinLength` / `MaxLength`, `Numeric()`,
  `Email()`, `URL()`, select / radio / checkbox options (including
  `OptionsUsing`, resolved per request), dates and times, tag limits and
  repeater `MinItems` / `MaxItems` are enforced, repeater items recursively;
- updates are partial: absent keys are left alone, but a required field cannot
  be emptied.

A failed check answers `422` with `{"error": "...", "errors": {"field": "..."}}`
(`lines.0.sku` inside repeaters); the web engine re-renders the form with the
messages. Inline cell edits only reach columns that are editable in the table
(`ToggleColumn`, `SelectColumn` with its options, `TextInputColumn`, `.Editable()`).
Resources without a form get the data unchanged; call
`res.AcceptUnvalidatedInput()` (or the same on a relation manager) only if your
handlers validate and whitelist input themselves.

Writes also need a record the user may see: update, delete, inline edit and
bulk delete load the record first, and a lookup error (`404`), `CanView` or
`CanEdit` / `CanDelete` refusal (`403`) stops them. Bulk restore and force
delete check `CanRestore` / `CanForceDelete` for every id.

Sign-in is throttled: 5 failed attempts per client and login name per minute
(and 20 per client across names) answer `429` with `Retry-After`, on
`POST <panel>/api/login` and the web login form. Tune it with
`panel.LoginThrottle(n, window)` (`0` disables it). Counters use golaravel's
rate limit store, in memory by default: use a Redis store when the panel runs
on several processes.

### 📁 File & Image Uploads (`forms.FileUpload` & `forms.ImageUpload`)

Support for drag-and-drop file handling with instant client-side image preview and circular avatar mode:

```go
// 1. Standard Drag & Drop File Upload
forms.FileUpload("attachment").
    Label("Contract Document").
    AcceptedFileTypes("application/pdf", "image/png", "image/jpeg").
    MaxSize(10240). // 10 MB
    Required()

// 2. Mobile-First Circular Avatar Mode
forms.ImageUpload("avatar").
    Label("Profile Photo").
    Avatar().
    HelperText("Recommended: 400x400 square photo").
    Required()
```
- **Automatic Multipart Encoding**: Forms containing upload fields automatically render with `enctype="multipart/form-data"`.
- **Existing File Retention**: Retains existing file paths (`name_existing`) if no replacement file is chosen.
- **Client Previews**: Generates instant `URL.createObjectURL` thumbnail previews upon file selection without waiting for server roundtrips.

---

### 📝 Rich Text & Markdown Editors (`forms.RichEditor` & `forms.MarkdownEditor`)

```go
// 1. WYSIWYG Rich Text Editor
forms.RichEditor("content").
    Label("Article Body").
    MinHeight("280px").
    Required()

// 2. Dual-Mode Markdown Editor with Write / Preview Tabs
forms.MarkdownEditor("documentation").
    Label("API Documentation").
    Rows(12).
    Previewable(true)
```
- **RichEditor**: Full formatting suite: Bold (`B`), Italic (`I`), Underline (`U`), Strikethrough (`S`), Headings (`H1`, `H2`, `H3`, `P`), Bullet Lists, Numbered Lists, Blockquotes, Code blocks, Hyperlinks with modal prompts, Clear format, and Undo/Redo.
- **MarkdownEditor**: Real-time Write / Preview tab switching with instant client-side markdown parsing and token insertion toolbar.

---

### 🔁 Dynamic Reorderable Repeater (`forms.Repeater`)

Allow operators to dynamically add, remove, collapse, and reorder sub-record arrays with automatic input name scoping (`items[0][sku]`, `items[1][sku]`):

```go
forms.Repeater("line_items").
    Label("Order Items").
    ItemLabel("Item").
    Collapsible(true).
    Reorderable(true). // Enables Move Up (↑) and Move Down (↓) buttons
    Grid(3).           // 3-column responsive layout per repeater item
    MinItems(1).
    MaxItems(15).
    AddActionLabel("+ Add Line Item").
    Schema(
        forms.TextInput("sku").Label("SKU").Required(),
        forms.TextInput("quantity").Numeric().Label("Qty").Required(),
        forms.TextInput("price").Numeric().Label("Price ($)").Required(),
    )
```
- **Automatic Bracket Notation Parsing**: Server-side `forms.ParseFormData` automatically converts `line_items[0][sku]` into structured `[]map[string]any` slices inside resource create and update handlers.
- **Reorderable & Collapsible**: Move items up and down with instant DOM reordering controls; collapse completed cards to save screen space.

---

### 🏷️ TagsInput & ColorPicker

```go
// Interactive tag chips with autocomplete suggestions
forms.TagsInput("technologies").
    Label("Tech Stack").
    Suggestions("Go", "Ent ORM", "Laravel", "Flutter", "Tailwind", "Alpine.js").
    MaxTags(8)

// Color picker with palette swatches and 2-way hex synchronization
forms.ColorPicker("primary_color").
    Label("Brand Primary Color").
    Presets("#6366f1", "#10b981", "#f59e0b", "#ef4444", "#3b82f6", "#8b5cf6")
```

---

### 🧙 Multi-Step Form Wizard (`forms.Wizard`)

Break complex or lengthy input workflows into structured, intuitive multi-step wizards with full step validation, progress indicators, and custom actions:

```go
forms.Wizard(
    forms.WizardStep("Personal Information").
        Description("Your contact and profile details").
        Icon("heroicon-o-user").
        Badge("Step 1").
        BadgeColor("primary").
        Schema(
            forms.TextInput("name").Required(),
            forms.TextInput("email").Email().Required(),
        ),
    forms.WizardStep("Company Details").
        Description("Tell us about your organization").
        Icon(`<svg viewBox="0 0 20 20"><path d="..."/></svg>`).
        Badge("Optional").
        BadgeColor("warning").
        Schema(
            forms.TextInput("company_name"),
            forms.TextInput("vat_number"),
        ),
    forms.WizardStep("Confirmation").
        Description("Review terms and finalize").
        Badge("Final").
        BadgeColor("success").
        Schema(
            forms.Toggle("terms_agreed").Label("I accept terms & conditions").Required(),
        ),
).
    Skippable(false).                         // Gate forward navigation behind valid step completion
    ProgressBar(true).                        // Animated completion percentage bar
    StartStep(1).                             // Initial active step
    NextActionLabel("Continue →").           // Custom Next button label
    PreviousActionLabel("← Go Back").         // Custom Previous button label
    SubmitAction("Complete Registration").    // Custom final Submit button label
    CancelAction("/admin/users", "Cancel")    // Cancel action link
```

- **Client-Side Validation Gate**: Evaluates `checkValidity()` and `reportValidity()` across inputs on the active step before allowing forward progression.
- **Flexible Navigation**: With `.Skippable(false)`, users can return to any previously visited step at any time while forward progression requires valid input. With `.Skippable(true)`, users may freely jump directly to any step.
- **Dynamic Progress Bar**: Live visual indicator displaying completed percentage `((currentStep / totalSteps) * 100)%`.
- **Badges & Icons**: Contextual badges (`primary`, `success`, `warning`, `danger`, `info`) and support for SVG or font icon classes.
- **Error Auto-Activation**: If server validation fails upon form submission, the wizard automatically switches directly to the first step containing invalid fields.
- **Mobile Responsive**: Adapts seamlessly to mobile viewports with a compact step summary (`Step X of Y: Step Title`) without horizontal layout overflow.

---

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

GoFilament's Table Builder (`pkg/tables`) provides a high-performance, type-safe data grid with server-side pagination, sorting, filtering, bulk actions, and advanced client interactivity:

```go
table := tables.New[*Customer]().
    Toggleable(true).               // Enable Column Show/Hide dropdown
    Reorderable("sort_order").      // Enable Drag-and-Drop row reordering
    Columns(
        tables.TextColumn[*Customer]("id").Sortable(),
        tables.TextColumn[*Customer]("name").Editable(true).Sortable(), // Inline editable on click
        tables.SelectColumn[*Customer]("status").Options(map[string]string{
            "active":   "Active",
            "inactive": "Inactive",
            "pending":  "Pending Review",
        }), // Inline select dropdown
        tables.ToggleColumn[*Customer]("is_verified"),                 // Interactive toggle switch
        tables.TextColumn[*Customer]("email").Copyable(),
        tables.TextColumn[*Customer]("notes").ToggledHiddenByDefault(true), // Hidden until toggled
    ).
    Filters(
        tables.SelectFilter("status").Options(map[string]string{
            "active": "Active Only",
        }),
    ).
    Actions(
        tables.EditAction[*Customer](),
        tables.DeleteAction[*Customer](),
    ).
    BulkActions(
        tables.DeleteBulkAction(),
        tables.ExportBulkAction(),
    )
```

---

### 🗑️ Soft Deletes & Trash Recovery Bin (`res.SoftDeletes()`)

Protect your application from accidental data loss with full lifecycle soft-delete support and Trash Recovery Bin:

```go
res := filament.NewResource[*Document]("documents").
    SoftDeletes(true).                           // Enables soft-delete awareness and trash scoping
    OnRestore(func(ctx *filament.Context, id string) error {
        return db.RestoreDocument(ctx, id)
    }).
    OnForceDelete(func(ctx *filament.Context, id string) error {
        return db.PermanentlyDeleteDocument(ctx, id)
    })
```

- **Automatic Trashed Filter**: Injects a 3-state scope filter (`Without Trashed` (default active), `With Trashed`, and `Only Trashed` (trash bin view)) into the table toolbar.
- **Dynamic Row Action Swapping**: Active records show standard `Edit` and `Delete` actions, while trashed records automatically display `Restore` (`⟲`) and `Force Delete` (`🗑`) actions with safety confirmation modals.
- **Visual Highlight**: Trashed records automatically display `[Trashed]` badges and muted row styling (`.table-row-trashed`).
- **Bulk Operations**: Seamlessly restore or purge batches of records with `tables.RestoreBulkAction()` and `tables.ForceDeleteBulkAction()`.
- **Activity Log Audit Trail**: Intercepts restore and force delete operations, recording `restored` and `force_deleted` events in the audit trail.
- **REST API Support**: Native endpoints at `/admin/api/resources/{slug}/restore` and `/admin/api/resources/{slug}/force-delete`.

### ⚡ Advanced Table Capabilities:

- **✏️ Inline Cell Editing**: Edit values directly in the table row without opening edit modals or full pages:
  - `TextInputColumn[T](name)`: Direct inline text edits synced on blur or `Enter`.
  - `SelectColumn[T](name)`: Immediate status/role updates via interactive dropdown.
  - `ToggleColumn[T](name)`: Instant visual switch animation persisting state asynchronously.
  - `TextColumn[T](name).Editable(true)`: Click-to-edit inline field with `Esc` revert and `Enter` save.
  - Automatic backend integration: Routes `POST /admin/{resource}/cell` and `POST/PATCH /admin/api/resources/{slug}/:id/cell`, triggering `r.OnCellUpdate(func(ctx, id, column, value) error)` or model updates.
- **👁️ Dynamic Column Toggling (Show/Hide)**: Operators can customize which columns are visible via a "Columns" dropdown menu. Preferences are automatically persisted to browser `localStorage` across page visits.
- **↕️ Drag-and-Drop Row Reordering**: Reorder records with grip handles (`⋮⋮`). Dragging a row dispatches an asynchronous update to `POST /admin/{resource}/reorder` and `POST /admin/api/resources/{slug}/reorder`, calling `r.OnReorder(func(ctx, orderedIDs) error)`.
- **🚀 Server-Side Sorting & Filtering**: Multi-column sorting (`↑`, `↓`), search bars, `SelectFilter`, and `TernaryFilter` (soft-deletes).
- **📦 Streaming Bulk & Header Actions**: In-line View/Edit/Delete actions, multi-record delete, and streaming data export (`CSV`, `TSV`, `JSON`).

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

## 🌐 Multi-Language Localization & Right-to-Left (RTL)

GoFilament integrates natively with GoLaravel v1.1.0's translation architecture (`pkg/translation`), offering seamless multi-language administration with automatic Right-to-Left (RTL) layout switching:

```go
panel := filament.NewPanel("admin").
    Path("/admin").
    Locales("en", "ar", "es", "fr"). // Supported languages
    DefaultLocale("en").
    LocaleLabels(map[string]string{
        "en": "English (US)",
        "ar": "العربية الفصحى",
    }).
    LocaleFlags(map[string]string{
        "en": "🇺🇸",
        "ar": "🇸🇦",
    })
```

### 🌍 Key Localization Features:
- **Automatic Topbar Language Switcher**: An interactive Alpine.js dropdown pill in the Topbar displays current flags, localized names, and active checkmarks when more than one locale is configured.
- **Direction-Aware (RTL/LTR)**: Evaluates `ctx.IsRTL()` and `ctx.Direction()`, automatically rendering `<html lang="%s" dir="%s">` across dashboards, resource CRUD views, and login screens.
- **Dedicated RTL Stylesheet**: Reverses sidebar borders, mirrors notification drawers to slide in from the left, flips unread indicators, and aligns text appropriately.
- **Flexible Resolution**: Automatically resolves language from URL query parameters (`?lang=ar`), persistent cookies (`locale=ar`), context defaults, or panel configuration with fallback validation.
- **Web & REST Endpoints**:
  - Web: `GET /admin/locale/switch?lang={code}` & `POST /admin/locale/switch` (sets persistent cookie, redirects back to Referer).
  - REST API: `POST /admin/api/locale/switch` (JSON body `{"locale": "ar"}` returning `{success, locale, direction, is_rtl}`).
  - REST API Schema: `GET /admin/api/schema` includes `locale`, `direction`, `is_rtl`, and `supported_locales`.

---

## 🎨 Custom Theme Builder & Brand Color Palettes

GoFilament features a full-spectrum Theme Builder and Color Palette Engine (`pkg/colors`), allowing you to completely brand your admin panel with zero build tools or complex asset pipelines:

```go
import "github.com/golaravel/gofilament/pkg/colors"

panel := filament.NewPanel("admin").
    Path("/admin").
    Font("Poppins"). // Auto Google Fonts integration
    Colors(map[string]any{
        "primary": colors.Indigo,   // Built-in preset
        "gray":    colors.Zinc,     // Zinc, Slate, Stone, Neutral, Gray
        "danger":  "#e11d48",       // Hex string auto-generates 11 shades!
        "warning": colors.Amber,
        "success": colors.Emerald,
        "info":    colors.Sky,
    }).
    Theme("/css/custom-theme.css"). // Custom stylesheets
    RenderHook(filament.PanelsRenderHookHeadEnd, func(ctx *filament.Context) string {
        return `<meta name="application-name" content="My App">`
    })
```

### 🌈 Key Theme & Color Features:
- **11-Shade Mathematical Palette Generator (`pkg/colors`)**: Pass any single hex color (e.g. `"#6366f1"`) and GoFilament will automatically calculate the full 11-step scale (`50`, `100`, `200`, `300`, `400`, `500`, `600`, `700`, `800`, `900`, `950`) using perceptually balanced HSL interpolation.
- **22 Built-in Color Presets**: Includes Tailwind/Filament presets (`Amber`, `Blue`, `Cyan`, `Emerald`, `Fuchsia`, `Gray`, `Green`, `Indigo`, `Lime`, `Neutral`, `Orange`, `Pink`, `Purple`, `Red`, `Rose`, `Sky`, `Slate`, `Stone`, `Teal`, `Violet`, `Yellow`, `Zinc`).
- **Dynamic CSS Variables Injection**: Automatically injects `<style id="filament-theme-vars">` into `<head>`, updating primary buttons, active links, badges, focus outlines, and typography without full rebuilds.
- **Typography & Google Fonts**: Set `Font("Poppins")` or provide custom font CDN URLs. GoFilament automatically adds preconnect links and updates `--font-sans`.
- **Filament Render Hooks**: Inject arbitrary HTML, analytics scripts, or custom UI elements into standard lifecycle hooks (`PanelsRenderHookHeadStart`, `PanelsRenderHookHeadEnd`, `PanelsRenderHookBodyStart`, `PanelsRenderHookBodyEnd`, `PanelsRenderHookTopbarStart`, `PanelsRenderHookTopbarEnd`, `PanelsRenderHookSidebarNavStart`, `PanelsRenderHookSidebarNavEnd`).
- **REST API Schema Alignment**: `GET /admin/api/schema` exports `"theme"` (font family, font URL, and full color palettes) so Flutter and mobile admin apps render with exact brand consistency.

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

## 📜 Activity Log & Audit Trail Timeline (`pkg/activitylog`)

GoFilament features a comprehensive, production-grade **Activity Log & Audit Trail** package for tracking entity history, user actions, system events, and changesets:

### 1. Fluent Activity Logger

```go
// Log custom actions anywhere in your application
activitylog.Log("Customer credit limit increased").
    Subject("Customer", "42").
    Causer("Admin", "1", "Alice Admin").
    Event("updated").
    WithDiff(
        map[string]any{"credit_limit": 5000},
        map[string]any{"credit_limit": 15000},
    ).
    Property("approved_by_finance", true).
    IP(c.ClientIP()).
    Save(c.Request.Context())
```

### 2. Automated Resource Auditing (Zero Boilerplate)

Automatically track all Create, Update, and Delete actions with pre/post changeset diffs and authenticated causers:

```go
userResource := filament.NewResource[*ent.User]("users")

// Automatically logs all CRUD actions on the resource
activitylog.EnableAuditLog(userResource)
// Or via the top-level helper:
gofilament.EnableAuditLog(userResource)
```

### 3. Vertical Audit Timeline & Infolist Component

Embed visual audit timelines with interactive diff inspectors directly inside resource View schemas:

```go
userResource.Infolist(func(infolist *infolists.Infolist) *infolists.Infolist {
    return infolist.Schema(
        infolists.Section("User Details").Schema(
            infolists.TextEntry("name"),
            infolists.TextEntry("email"),
        ),
        infolists.Section("Audit Trail & History").Schema(
            activitylog.TimelineEntry("activities").
                Title("Recent Activity").
                Filterable(true).        // Live client-side event filtering and search
                CollapsibleDiffs(true),  // Expandable changeset diff accordions
        ),
    )
})
```

- **Interactive Changeset Diff Inspector**: Expands before/after comparison tables showing field changes with strikethrough red pills for previous values and green pills for updated values.
- **Continuous Connected Timeline Spine**: Visual status node circles with event icons (plus for created, pencil for updated, trash for deleted, check for approved, etc.).
- **Actor Badges & Fallbacks**: Renders actor avatars or automatic initials badges (`AA` for Alice Admin) with hoverable exact timestamps and relative time representation ("just now", "15 minutes ago", "yesterday").
- **Standalone Audit Resource**: Mount `gofilament.NewActivityResource()` (or `activitylog.NewActivityResource()`) to provide administrators with a complete audit browsing and search interface in the panel navigation.

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

### 🧬 Typed Dart from the panel (`flutterclient.GenerateFromPanel`)

The `gofilament_admin` package renders a panel from its API at runtime. When you
also want **static, typed Dart** for your own screens, generate it from the Go
panel instead of writing it by hand:

```go
// tools/gen-flutter/main.go — run with `go run ./tools/gen-flutter`
err := flutterclient.GenerateFromPanel(panels.Admin(), "resources/flutter", flutterclient.GenerateOptions{})
```

or register the artisan command (it needs a function that builds your panel):

```go
reg.Register(commands.FlutterGenerateCommand(panels.Admin, commands.DefaultFlutterDir, flutterclient.GenerateOptions{}))
// artisan filament:flutter-generate [--out=dir] [--check]
```

The panel is booted in-process and read through its own API with `httptest`:
no port is opened and no create / update / delete / action handler runs (list,
form and option loaders do). Five files are written to `lib/generated/`, each
marked `DO NOT EDIT`; the only other files it creates are stubs for pages that
have no file yet:

| File | Contents |
|---|---|
| `schema.g.dart` | `panelSchema` (a snapshot of `/api/schema`) and `panelForms` (resource, relation and action forms) |
| `models.g.dart` | a class per resource record, built from the Go record type (`fromJson` / `toJson`, unknown keys kept in `raw`); an `…Input` class per form, relation form and action form; classes for endpoint request / response types |
| `client.g.dart` | `PanelClient`: `list` / `find` / `create` / `update` / `delete` / `run<Action>` per resource, a client per relation, and a typed method per `filament.Endpoint` |
| `pages.g.dart` | `panelPages(PanelPageBuilders(...))`: the `AdminPage`s of every `filament.FlutterPage`, with one **required** builder each |
| `registry.g.dart` | `panelAppPages` and `panelComponents`: your pages and components from `lib/pages` and `lib/components` |

#### `resources/flutter`: your Dart lives next to your Go

Like Laravel's `resources/` folder, the panel's Flutter app sits in the Go
project, and hand-written Dart has a fixed place in it:

```
resources/flutter/
  lib/main.dart                 ← written once by make:filament-flutter-app
  lib/pages/pos_page.dart       ← yours: class PosPage, one file per filament.FlutterPage("pos", ...)
  lib/components/fields/signature_pad_field.dart    ← class SignaturePadField: forms.Custom(name, "signature_pad")
  lib/components/cells/paid_badge_cell.dart         ← class PaidBadgeCell: tables.CustomColumn(name, "paid_badge")
  lib/components/widgets/sales_total_widget.dart    ← class SalesTotalWidget: filament.FlutterWidget(id, "sales_total")
  lib/generated/*.g.dart        ← generator only (DO NOT EDIT)
```

```bash
artisan make:filament-flutter-app                 # resources/flutter, typed main.dart
artisan make:filament-flutter-page pos --label="Point of sale" --fullscreen
#   → lib/pages/pos_page.dart and the Go line: panel.Pages(filament.FlutterPage("pos", "Point of sale").Fullscreen())
artisan filament:flutter-generate                 # writes lib/generated, stubs missing pages
```

A page id maps to `lib/pages/<snake_id>_page.dart` and the class
`<PascalId>Page({super.key, required AdminController admin})`. Declaring a
page in Go and regenerating creates that file once if it is missing; the
generator never changes an existing page. A page file that does not define its
class stops generation with a clear error, and `--check` lists pages that have
no file yet.

#### Custom field, cell and dashboard components

When a built-in input or cell is not enough, name a Flutter component in Go
and write it once in Dart:

```go
forms.Custom("signature", "signature_pad").Required().
    Props(map[string]any{"pen_color": "#112233", "height": 160}).
    Rule(func(ctx context.Context, v any) string { /* server-side check */ return "" })

tables.CustomColumn[*Order]("paid", "paid_badge").Props(BadgeProps{Tone: "success"})

panel.Widget(filament.FlutterWidget("sales", "sales_total").Heading("Sales today").
    Data(func(ctx *filament.Context) (any, error) { return salesToday(ctx) }).
    Visible(func(ctx *filament.Context) bool { return isManager(ctx) }))
```

The generator creates `lib/components/<kind>s/<name>_<kind>.dart` with class
`<Name>Field` / `<Name>Cell` / `<Name>Widget` once, a typed
`<Name><Kind>Props` class in `models.g.dart` (inferred from the props values
set in Go: a key is non-null only when every use sets it, and keeps its type
only when every use agrees), and `panelComponents` in `registry.g.dart` for
`GoFilamentAdmin(components: panelComponents)`.

| Component | Receives | Server keeps |
|---|---|---|
| field | `AdminFieldContext`: `value`, `onChanged(v)`, `enabled`, `required`, `errorText`, `props` | Required / RequiredOnCreate / Disabled / Visible and your `Rule` on every save |
| cell | `AdminCellContext`: `value`, the whole `row`, `props` | what the list endpoint returns for the user |
| widget | `AdminWidgetContext`: `data` from `Data(...)` for this user, `heading`, `props` | `Visible`, and `Data` runs per request |

A component missing from the app falls back to a read-only value (cells), a
"not available in this app" note (fields, widgets). The web engine shows
custom fields and cells as read-only values and does not show Flutter widgets.
Avoid component names whose class would clash with Flutter's own (for example
`text` → `TextField`).

```dart
void main() => runApp(GoFilamentAdmin(
      initialSchema: panelSchema, // first frame without waiting for /api/schema
      pages: panelAppPages,       // or panelPages(PanelPageBuilders(pos: ...)) to wire them yourself
    ));

// in lib/pages/pos_page.dart (PanelClient get client => PanelClient(admin.api);)
final orders = await client.salesOrders.list(search: 'SO');
await client.salesOrders.runRefund('12', const SalesOrderRefundInput(amount: 5));
final summary = await client.salesSummary(till: 't1');
```

Declare Flutter pages and typed endpoints in Go:

```go
panel.
    Pages(filament.FlutterPage("pos", "Point of sale").Icon("heroicon-o-shopping-cart").Fullscreen().
        Visible(func(ctx *filament.Context) bool { return ctx.User() != nil })).
    Endpoints(filament.Endpoint("salesSummary", "GET", "/pos/tills/:till/summary",
        func(ctx *filament.Context, q SummaryQuery) (Summary, error) { ... }).
        Authorize(func(ctx *filament.Context) bool { ... }))
```

Endpoints are mounted under `<panel>/api` behind the panel's authentication;
GET and DELETE read the request struct from the query string, other methods
from the JSON body. Use `struct{}` for no request or no response.

**The server stays authoritative.** The snapshot has no permissions, dashboard
values, tenants or data-loaded select options: until the live schema arrives
nothing can be created, edited or run, and Flutter pages are shown only when
the live schema lists them in `client_pages`. Every request is still checked
on the server. Abilities, row visibility and validation never come from
generated code.

**Keep it in sync in CI.** Output is deterministic (sorted keys, stable order),
so regenerate and fail on a diff, or use `--check`:

```bash
go run ./tools/gen-flutter && git diff --exit-code resources/flutter/lib/generated
```

The generator reads the panel as a `SimpleShieldUser` with the `super-admin`
role and the `*` permission. If your authorization callbacks type-assert your
own user type, pass one in `GenerateOptions.User`. Generated files are not
`dart format`ted; leave `lib/generated` out of format checks.
`flutterclient.GenerateFlutterAdmin` (the old 12k-line scaffold) is
deprecated: use `GenerateAdminApp` for the app and `GenerateFromPanel` for its
typed code.

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

## 📊 Dynamic Chart Drilldowns & Live Metric Dashboards

Build reactive, real-time KPI and graphical dashboards with zero JavaScript build steps:

### 📈 Interactive Chart Widgets:
- **Filters & Preset Ranges**: Select from filter presets (`today`, `7d`, `30d`, `quarter`, `year`) or native date range pickers (`DateRangePicker(true)`).
- **Asynchronous In-Place Updates**: Changing a filter triggers a background fetch to `/admin/widgets/:id/data?filter=...` that updates chart datasets smoothly using Chart.js animations without reloading the page.
- **Click-Through Drilldowns**: Click on any bar, line point, or pie segment to navigate directly to filtered resource tables with token replacement:
  ```go
  chart := widgets.NewChartWidget("Revenue Analytics", widgets.ChartTypeBar).
      WidgetID("revenue_analytics").
      Filters(map[string]string{
          "7d":  "Last 7 Days",
          "30d": "Last 30 Days",
      }).
      DefaultFilter("7d").
      Drilldown("/admin/orders?status={label}&source=chart").
      DateRangePicker(true).
      Polling("15s").
      FilterData(func(ctx *filament.Context, filter string) widgets.ChartData {
          return widgets.ChartData{
              Labels: []string{"Completed", "Pending", "Failed"},
              Datasets: []widgets.ChartDataset{
                  {Label: "Orders", Data: []float64{120, 45, 8}},
              },
          }
      })
  ```
- **Live Metric Polling**: Configure `Polling("10s")` for continuous background telemetry updates.
- **Clickable Stat Overview Cards**: Wrap KPI cards in drilldown links (`Stat.URL("/admin/invoices?status=pending")`) with animated hover elevation.
- **Automatic Grid Placement**: The panel automatically arranges registered widgets in responsive 3-column CSS grids respecting `.SetColumnSpan()` and `.SetSort()`.

---

## 🔍 Advanced Global Search & Spotlight Command Palette (⌘K / Ctrl+K)

GoFilament features a macOS Spotlight / Raycast-inspired **Command Palette & Global Search Modal** powered by a zero-dependency, typo-tolerant **Fuzzy Search Algorithm** (`pkg/globalsearch`):

```go
panel := filament.NewPanel("admin").
    Path("/admin").
    GlobalSearch(true).                     // Enable or disable global search (default: true)
    GlobalSearchKeyBinding("k").            // Shortcut key: Cmd+K / Ctrl+K
    GlobalSearchResultsLimit(10).           // Maximum results per group/panel
    GlobalSearchFuzzy(true)                 // Typo tolerance and subsequence matching
```

### 🎯 Customizing Resource Global Search:

Equip any Ent resource with rich global search capabilities, secondary badge tags, quick action shortcuts, and custom URLs:

```go
customerResource := filament.NewResource[*Customer]("customers").
    ModelName("Customer").
    PluralModelName("Customers").
    GloballySearchable(true).
    GlobalSearchColumns("name", "email", "company").
    GlobalSearchResultsLimit(5).
    GlobalSearchResultTitle(func(record *Customer) string {
        return record.Name
    }).
    GlobalSearchResultDetails(func(record *Customer) map[string]string {
        return map[string]string{
            "Email":   record.Email,
            "Company": record.Company,
        }
    }).
    GlobalSearchResultActions(func(record *Customer) []filament.GlobalSearchAction {
        return []filament.GlobalSearchAction{
            {
                Label: "View Orders",
                URL:   fmt.Sprintf("/admin/orders?customer_id=%d", record.ID),
                Icon:  "heroicon-shopping-cart",
            },
        }
    }).
    GlobalSearchResultURL(func(record *Customer) string {
        return fmt.Sprintf("/admin/customers/%d/edit", record.ID)
    })
```

### ⚡ Key Spotlight & Fuzzy Search Features:
- **Keyboard-Driven Navigation**: Tap `⌘K` or `Ctrl+K` from any screen to summon the backdrop-blurred Spotlight modal. Navigate through categorized results with `ArrowUp` / `ArrowDown`, jump directly with `Enter`, or dismiss with `Escape`.
- **Typo-Tolerant Fuzzy Scoring (`pkg/globalsearch`)**: Evaluates exact matches (`1.0`), prefix matches (`0.95`), substring matches (`0.85`), word-boundary acronym subsequences (`0.70-0.84`), and Levenshtein distance edits ($\le 2$ typos) with automatic `<mark class="spotlight-highlight">` matched term highlighting.
- **Cross-Domain Discovery**: Searches across registered Ent database resources, custom panel pages (`p.pages`), and sidebar navigation menu items (`BuildNavigation`) in a unified, grouped view.
- **Rich Metadata & Inline Actions**: Results display categorized badges (`NAVIGATION`, `CUSTOMERS`), detail tag pills, and clickable action buttons alongside the primary record link.
- **Dual Web & REST API Architecture**:
  - Web: `/admin/global-search?query=...` serves instant rendered HTML spotlight items or direct JSON when `Accept: application/json` is sent.
  - Dedicated REST API: `GET /admin/api/global-search?query=...` returns structured JSON `{query, total, groups, results}` with pre-calculated relevance scores for Flutter, mobile, and external integrations.
  - Schema Metadata: `GET /admin/api/schema` exposes `global_search: {enabled, key_binding, results_limit, fuzzy}`.

---

## 🛒 Complete Example Application: Apex eCommerce Store

GoFilament ships with a complete, production-grade eCommerce store admin panel example located in [`examples/ecommerce`](./examples/ecommerce). It serves as a real-world reference implementation demonstrating all of GoFilament's major capabilities working in unison:

```bash
# Run the eCommerce demo application
go run ./examples/ecommerce

# Open your browser at
# http://localhost:8080/admin
```

### ✨ Features Showcased:
1. **Executive KPI Dashboard (`widgets.NewStatsOverviewWidget`)**:
   - Total Revenue (`$124,580.00`), Orders Fulfilled (`1,482`), Average Order Value (`$84.07`), and Active Customers (`3,890`).
   - SVG sparklines, trend comparisons (`+18.4% vs last month`), and one-click drilldown URLs into resource tables.
2. **Product Catalog (`ProductResource`)**:
   - **Multi-Step Form Wizard (`forms.Wizard`)**: Multi-step creation flow spanning *Basic Info*, *Pricing & Stock*, and *Media & Description* with dynamic progress indicators and validation autofocus.
   - **Interactive Table**: Inline cell editing (`TextInputColumn("price")`, `SelectColumn("inventory_status")`, `ToggleColumn("is_published")`), drag-and-drop row reordering (`Reorderable("sort_order")`), column toggling modal (`Toggleable(true)`), and soft-deletes Trash Recovery Bin (`SoftDeletes(true)`).
3. **Orders & Logistics (`OrderResource`)**:
   - **Dynamic Line Items**: Multi-item invoice repeater (`forms.Repeater("items")`) with automatic line item total calculations.
   - **Audit Trail & Changeset Timeline**: Infolist detail view featuring `activitylog.TimelineEntry` displaying real-time before/after diffs, fulfillment milestones, and webhook event logs.
4. **Customer CRM (`CustomerResource`)**:
   - Tag management (`forms.TagsInput("tags")`), VIP membership toggle, customer spending tracking, and soft-delete recovery.
5. **Automated Audit Logging & Standalone Audit Resource**:
   - Zero-boilerplate audit trail recording via `activitylog.EnableAuditLog(resource)`.
   - Dedicated `activitylog.NewActivityResource()` management screen with filterable event tags (`created`, `updated`, `deleted`, `restored`).

---

## 📱 Multiplatform Example: Pulse Fleet (Flutter Engine)

GoFilament supports delivering panels as **Flutter Multiplatform Applications** (`EngineFlutter`), targeting Mobile (iOS, Android), Desktop (macOS, Linux, Windows), and Web through a single codebase driven by GoFilament's JSON REST API:

```bash
# 1. Run the Go backend server (port :8081)
go run ./examples/flutter_admin

# Open browser at http://localhost:8081/admin (serves the Flutter Web shell)
# Or inspect dynamic metadata at http://localhost:8081/admin/api/schema

# 2. (Optional) Run natively on mobile simulator or desktop via Flutter SDK
cd examples/flutter_admin/client
flutter run
```

### ⚡ Flutter Engine Features:
- **Dynamic Schema Hydration**: The Flutter client queries `GET /admin/api/schema` on launch to dynamically configure brand titles, theme colors, navigation destinations, and active tenant contexts.
- **Mobile-First Responsive Shell**: Uses `ResponsiveLayout` to swap seamlessly between Material 3 Bottom Navigation on phones and a Desktop Navigation Rail on larger screens.
- **Native Touch Interactions**: Haptic feedback on tap, pull-to-refresh on entity feeds, swipe-to-delete, and fluid modal dialogs.
- **Artisan Scaffolding**: Generate customized mobile-first Flutter admin clients anytime using:
  ```bash
  artisan make:filament-flutter MobileAdmin
  ```

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
