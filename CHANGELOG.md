# Changelog 🚀

All notable changes to the **GoFilament** framework will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v1.14.0] - 2026-10-31

### Added
- **Advanced Form Fields: Rich Text Editor, File & Image Upload, Dynamic Repeater, TagsInput & ColorPicker (`pkg/forms`, `pkg/filament/resource.go`)**:
  - **Interactive File & Image Upload Field (`forms.FileUpload` & `forms.ImageUpload`)**:
    - Modern drag-and-drop dropzone with drop target highlights, upload icons, and file type/size constraint indicators.
    - Avatar circular layout mode (`avatar.Avatar()`) with hover camera action, circular preview, and custom sizing.
    - Live image preview cards for newly selected files via instant client data URLs.
    - Existing file preservation (`_existing`) with filename display, download/replace controls, and remove button.
    - Multi-file upload support (`.Multiple()`, `.MaxFiles(n)`), accepted MIME types (`.AcceptedFileTypes(...)`), and max size validation (`.MaxSize(kb)`).
    - Automatic multipart detection: forms containing `FileUpload` automatically render with `enctype="multipart/form-data"`.
  - **Full-Featured Rich Text WYSIWYG & Markdown Editor (`forms.RichEditor` & `forms.MarkdownEditor`)**:
    - **`RichEditor`**: Full WYSIWYG editing toolbar supporting Bold, Italic, Underline, Strikethrough, Headings (H1/H2/H3/P), Unordered/Ordered lists, Blockquotes, Code blocks, Hyperlinks with dialog prompt, Clear formatting, and Undo/Redo. Real-time content synchronization into hidden form inputs.
    - **`MarkdownEditor`**: Dual-mode Write and Preview tabs. Write mode offers quick markdown token insertion buttons; Preview mode renders formatted HTML in real-time.
  - **Production-Grade Dynamic Repeater Component (`forms.Repeater`)**:
    - Addable and deletable array items with unique client-side index tracking and field name scoping (`items[0][field]`).
    - Reorderable items with Move Up (`↑`) and Move Down (`↓`) controls.
    - Collapsible item cards with collapse/expand toggle (`▼` / `►`), dynamic item titles, and badge indicators.
    - Multi-column grid support inside each repeater item (`.Grid(columns)`).
    - Server-side bracket notation parser (`forms.ParseFormData`) converting incoming POST form data into structured `[]map[string]any` slices automatically in `BaseResource` handlers.
  - **Interactive `TagsInput` & `ColorPicker`**:
    - **`TagsInput`**: Interactive pill tags with remove (`×`) buttons, keyboard triggers (Enter, Comma, Backspace), autocomplete suggestions popover (`.Suggestions(...)`), and max tags enforcement.
    - **`ColorPicker`**: Brand color swatches row for quick 1-click preset selection, native HTML5 color wheel input, and hex text input with bidirectional synchronization.

---

## [v1.13.0] - 2026-10-31

### Added
- **Advanced Global Search with Typo-Tolerant Fuzzy Search & Spotlight Modal (`pkg/globalsearch`, `pkg/filament/panel.go`, `pkg/filament/resource.go`, `pkg/filament/api.go`, `pkg/assets/dist`)**:
  - **Spotlight Command Palette Modal (`⌘K` / `Ctrl+K`)**:
    - Keyboard-first command palette opening with `Cmd+K` / `Ctrl+K` or clicking the topbar command trigger.
    - Smooth backdrop blur (`backdrop-filter: blur(4px)`), centered animated modal container, and autofocus search input.
    - Full keyboard navigation: `↑` / `↓` cycle through matches with auto-scrolling highlight, `↵` navigates to the active record, `ESC` dismisses the modal.
    - Clear query button and shortcut hints in the spotlight footer.
  - **High-Performance Typo-Tolerant Fuzzy Search Engine (`pkg/globalsearch/fuzzy.go`)**:
    - Multi-tiered scoring algorithm evaluating exact matches, prefix matches, word-boundary/acronym subsequences, and Levenshtein edit distance for typo tolerance (e.g. searching "custmer" matches "Customer", "ordr" matches "Order").
    - Search highlight formatting (`globalsearch.Highlight`) wrapping matched query letters with `<mark class="spotlight-highlight">`.
  - **Cross-Resource & Navigation Search**:
    - Searches all panel navigation items and custom pages alongside database records.
    - Categorized group headers with record count indicators (e.g. `NAVIGATION (2)`, `CUSTOMERS (4)`, `ORDERS (3)`).
    - Rich record display with key-value metadata tags (`GlobalSearchResultDetails`), direct record action buttons (`GlobalSearchResultActions`), and custom URLs (`GlobalSearchResultURL`).
  - **Panel Configuration**:
    - `panel.GlobalSearch(true)`: Enables global search and spotlight modal.
    - `panel.GlobalSearchKeyBinding("k")`: Configurable keyboard shortcut trigger.
    - `panel.GlobalSearchResultsLimit(15)`: Configurable result limit per resource.
    - `panel.GlobalSearchFuzzy(true)`: Toggles typo-tolerant fuzzy ranking.
  - **Standardized REST API Endpoint**:
    - `GET /admin/api/global-search?query=...`: Returns JSON with total count, categorized groups, items, details, and actions for headless, Flutter, and mobile integrations.
    - `GET /admin/api/schema`: Includes `global_search` metadata (enabled, key binding, limit, fuzzy).

---

## [v1.12.0] - 2026-10-31

### Added
- **Custom Theme Builder & Brand Color Palettes (`pkg/colors`, `pkg/filament/theme.go`, `pkg/filament/hooks.go`, `pkg/assets/dist/filament.css`)**:
  - **Full-Spectrum Color Palette Engine (`pkg/colors`)**:
    - **11-Shade Mathematical Palette Generator**: `colors.FromHex(hex)` and `colors.FromRGB(r, g, b)` automatically derive full 11-step Tailwind/Filament scales (`50`, `100`, `200`, `300`, `400`, `500`, `600`, `700`, `800`, `900`, `950`) via HSL color space lightness/saturation interpolation.
    - **22 Built-in Presets**: Pre-configured battle-tested palettes (`Amber`, `Blue`, `Cyan`, `Emerald`, `Fuchsia`, `Gray`, `Green`, `Indigo`, `Lime`, `Neutral`, `Orange`, `Pink`, `Purple`, `Red`, `Rose`, `Sky`, `Slate`, `Stone`, `Teal`, `Violet`, `Yellow`, `Zinc`).
    - CSS variable generation: `palette.CSSVariables(role)` formats declarations for `:root`.
  - **Panel Theme & Brand Color Configuration**:
    - `panel.Colors(map[string]any{"primary": colors.Indigo, "gray": colors.Zinc, "danger": "#ef4444"})`: Registers brand color palettes with auto-resolution from presets, palettes, or raw hex codes.
    - `panel.Color("primary", "#6366f1")`: Fluent single-role color override.
  - **Typography & Font Management**:
    - `panel.Font("Poppins")`: Auto-generates Google Fonts `<link>` and injects `--font-sans` CSS variable.
    - `panel.Font("Inter", customUrl)`: Custom font CDN stylesheet integration.
  - **Custom Stylesheets, Scripts & Render Hooks**:
    - `panel.Theme("custom.css")` / `panel.Stylesheet("extra.css")` and `panel.Script("analytics.js")`.
    - **Filament Render Hooks**: Injects custom HTML at standard extension points (`PanelsRenderHookHeadStart`, `PanelsRenderHookHeadEnd`, `PanelsRenderHookBodyStart`, `PanelsRenderHookBodyEnd`, `PanelsRenderHookTopbarStart`, `PanelsRenderHookTopbarEnd`, `PanelsRenderHookSidebarNavStart`, `PanelsRenderHookSidebarNavEnd`).
  - **Dynamic Theme CSS Variables Injection**:
    - All admin layouts automatically inject `<style id="filament-theme-vars">` containing customized CSS variables into `<head>`, instantly updating primary, gray, danger, warning, success, info, and typography.
  - **REST API Schema Theme Metadata**:
    - `GET /admin/api/schema` returns `"theme"` object with `"font_family"`, `"font_url"`, `"colors"`, and `"stylesheets"` for native mobile/Flutter theme alignment.

---

## [v1.11.0] - 2026-10-31

### Added
- **Multi-Language Localization & Right-to-Left (RTL) Support (`pkg/filament/locale.go`, `pkg/filament/context.go`, `pkg/filament/panel.go`, `pkg/filament/api.go`, `pkg/assets/dist/filament.css`)**:
  - **GoLaravel v1.1.0 Integration**:
    - Direct dependency upgrade to GoLaravel v1.1.0 translation architecture (`pkg/translation`, `pkg/context`).
    - Context helper methods: `c.Locale()`, `c.Direction()`, `c.IsRTL()`, `c.Trans(key, replace...)`, `c.TransChoice(key, number, replace...)`.
  - **Panel Locale Configuration**:
    - `p.Locales("en", "ar", "es", ...)`: Configures supported admin panel languages.
    - `p.DefaultLocale("en")`: Configures fallback locale.
    - `p.LocaleLabels(map[string]string{...})`: Configures custom display names for language selectors.
    - `p.LocaleFlags(map[string]string{...})`: Configures custom emoji/icon flags.
    - Built-in default dictionary supporting 19+ languages with native labels and national flag emojis.
  - **Automated Locale Resolution**:
    - Multi-tiered resolution: URL query params (`?lang=` / `?locale=`) > Session cookie (`locale`) > GoLaravel Context locale > Configured panel defaults.
    - Strict validation fallback to default locale if an unsupported code is requested.
    - Automatically sets `Content-Language` header on HTTP responses.
  - **Interactive Topbar Language Switcher**:
    - Accessible Alpine.js dropdown pill rendered automatically in the panel topbar whenever `len(locales) > 1`.
    - Active language highlight with checkmark, flag icons, and native display labels.
    - Direction-aware popup positioning (left-aligned for LTR, right-aligned for RTL).
  - **Language Switch Endpoints**:
    - Web: `GET /admin/locale/switch?lang={code}` & `POST /admin/locale/switch` (sets persistent 1-year cookie and redirects back to `Referer` or panel base).
    - REST API: `POST /admin/api/locale/switch` (supports JSON body `{"locale": "ar"}` or query params, returns `{success, locale, direction, is_rtl, label, flag}`).
    - Schema Metadata: `GET /admin/api/schema` exposes `locale`, `direction`, `is_rtl`, `supported_locales`, `default_locale`, and `locale_labels`.
  - **RTL Layout & CSS Styling**:
    - Layout templates render dynamic `<html lang="%s" dir="%s">` across dashboard, resource CRUD pages, and login screen.
    - Comprehensive RTL CSS rules: reversed sidebar borders, notifications drawer sliding in from the left, mirrored unread notification markers, and right-aligned tables.

---

## [v1.10.0] - 2026-10-31

### Added
- **Dynamic Chart Drilldowns & Interactive Metric Dashboards (`pkg/widgets/chart.go`, `pkg/widgets/stats.go`, `pkg/filament/panel.go`, `pkg/filament/api.go`)**:
  - **Dynamic Chart Filters & Presets**:
    - Filter dropdown selector (`w.Filters(map[string]string{...})`) with default filter support (`w.DefaultFilter("7d")`).
    - Dynamic data provider `w.FilterData(func(ctx *filament.Context, filter string) ChartData)` evaluated on-demand.
    - Interactive Date-Range Picker (`w.DateRangePicker(true)`): native From/Until date inputs firing live client-side dataset updates.
  - **Click-Through Chart Drilldowns**:
    - `w.Drilldown("/admin/orders?status={label}&dataset={dataset}&value={value}")`: Segment click interception on bars, lines, and slices with automatic parameter interpolation and resource navigation.
    - Visual feedback with pointer cursor on hoverable chart elements.
  - **Live Metric Polling & Asynchronous Refresh**:
    - Configurable polling intervals (`w.Polling("10s")` / `w.Polling("1m")`).
    - Client-side chart controller (`window.FilamentCharts.update(id, filter)` & `dateRangeChange(id)`) updating datasets in place without full page reloads.
  - **Clickable Stat Cards**:
    - `Stat.URL(targetUrl)`: Wraps KPI stat overview cards in interactive links with hover shadows for instant drilldowns to filtered resource tables.
  - **Dynamic Dashboard Grid Layout**:
    - `renderDefaultDashboard` automatically arranges registered panel widgets in a responsive CSS grid respecting `Sort()` and `ColumnSpan()`.
  - **Widget Dynamic Data Endpoints & REST API**:
    - Web AJAX endpoint: `GET /admin/widgets/:id/data?filter=...&from=...&until=...`
    - REST API single widget data: `GET /admin/api/widgets/:id/data?filter=...`
    - REST API metadata & evaluation: `GET /admin/api/widgets` with rich metadata, chart types, active filters, and evaluated datasets.

---

## [v1.9.0] - 2026-10-31

### Added
- **High-Throughput Data Streaming (Export Engine)** (`pkg/filament/resource.go`, `pkg/filament/api.go`, `pkg/tables`):
  - Direct HTTP chunked streaming for large datasets ($O(\text{chunk size})$ memory footprint) with periodic `http.Flusher` flushes.
  - Multi-format support: **CSV** (`text/csv`), **TSV** (`text/tab-separated-values`), and **JSON** (`application/json`).
  - Dynamic inspection of configured table columns and labels, extracting raw and formatted values with automated HTML tag stripping for clean data.
  - Comprehensive query filtering: query strings (`search`, `sort`, `direction`), custom filters (`filter_*`), and explicit IDs (`?ids=1,2,3`).
  - Bulk action integration: `tables.ExportBulkAction()` natively hooks into row selection checkboxes.
- **Interactive Column-Mapping Import Engine** (`pkg/filament/resource.go`, `pkg/filament/api.go`, `pkg/tables`):
  - Web import page (`GET /admin/{resource}/import`) with sample CSV download link, file selector, and auto-mapping overview.
  - Auto-normalization of column headers (case-insensitive, whitespace, hyphen, and underscore canonicalization).
  - Explicit column mapping overrides (`mapping[source]=target`).
  - Row-by-row batch insertion via `r.onCreate` with failure tracking and error reporting.
  - Real-time notification dispatch via `notifications.Send(...)` for completion and warnings.
  - REST API import & export endpoints:
    - `GET /admin/api/resources/:slug/export` (supports `format=csv|tsv|json`, `ids`, `search`, filters).
    - `POST /admin/api/resources/:slug/import` (supports multipart file upload, JSON array batches `{"data": [...]}`, and raw CSV string payloads `{"csv": "..."}`).
- **Table DSL Extensions**:
  - `tables.ImportAction[T]()` and `tables.ExportAction[T]()` for table header action bars.
  - `.Label(string)` chaining methods for `BadgeColumnDef`, `ImageColumnDef`, and `ToggleColumnDef`.
  - Column and action accessors on `tables.Table[T]` (`GetColumns()`, `GetHeaderActions()`, `GetActions()`, `GetBulkActions()`, `ExtractFieldValue()`, `ExtractRecordID()`).

---

## [v1.8.0] - 2026-10-31

### Added
- **Advanced Form Components: Wizard & Repeater (`pkg/forms/advanced_layouts.go`, `pkg/forms/form.go`)**:
  - **Sequential Multi-Step Wizard (`forms.Wizard([*forms.WizardStep])`)**:
    - Stepper navigation bar with interactive step pills and connecting progress bars.
    - Completed step checkmarks (`✓`), active step highlight, and pending steps.
    - Automatic validation error detection: auto-activates the step containing failing fields.
    - Built-in Next, Previous, and customizable Submit (`SubmitAction`) buttons.
  - **Dynamic Line-Item Repeater (`forms.Repeater(name)`)**:
    - In-place dynamic row additions with template cloning (`<template id="tmpl_...">`).
    - Client-side row removals and counter updates.
    - Scoped array field names: automatically formats child inputs as `items[0][field]`, `items[1][field]`, etc.
    - Constraints: `MinItems`, `MaxItems`, and customizable `AddActionLabel` / `ItemLabel`.
  - **Recursive Nested Field Resolution**:
    - Enhanced `form.GetField(...)` to resolve fields deeply nested inside sections, grids, fieldsets, splits, wizards, and repeaters.

---

## [v1.7.0] - 2026-10-31

### Added
- **Real-Time Database Notifications Drawer & SSE Push (`pkg/notifications`, `pkg/filament/panel.go`, `pkg/filament/api.go`)**:
  - `NotificationStore` interface and `MemoryNotificationStore` thread-safe in-memory database notification persistence.
  - `notifications.Send(user, notif)` and `notifications.SendToAll(notif)` dispatchers with automatic real-time broadcast over GoLaravel's `pkg/broadcast` hub.
  - **Topbar Notification Bell & Live Badge**:
    - Animated bell button displaying unread badge count (`#notif-unread-badge`).
    - Real-time increment and decrement on new alerts and read actions.
  - **Slide-Out Notification Drawer**:
    - Animated slide-in right drawer (`#filament-notifications-drawer`) accessible from anywhere in the admin panel.
    - View notification title, formatted body text, timestamp, and optional call-to-action link.
    - Inline "Mark as read" button on unread notification items.
    - Drawer header controls: "Mark all read", "Clear all", and close drawer.
  - **Server-Sent Events (SSE) Real-Time Stream**:
    - `GET /admin/notifications/stream`: Real-time SSE endpoint streaming new alerts directly into connected browsers.
    - Auto-injected client listener (`Filament.initSSE(...)`) displaying toast popups and updating drawer state without page refreshes.
  - **Web Endpoints & REST API**:
    - `GET /admin/notifications` & `GET /admin/api/notifications`
    - `POST /admin/notifications/mark-read` & `POST /admin/api/notifications/mark-read`
    - `POST /admin/notifications/mark-all-read` & `POST /admin/api/notifications/mark-all-read`
    - `POST /admin/notifications/clear` & `POST /admin/api/notifications/clear`

---

## [v1.6.0] - 2026-10-31

### Added
- **Interactive In-Place Relation Managers (`pkg/filament/relation.go`, `pkg/filament/resource.go`, `pkg/filament/api.go`)**:
  - `RelationManager` interface & `BaseRelationManager[T any]` generic implementation for embedding related child models directly into parent resource View and Edit screens.
  - **In-Place Web Operations (Modals & AJAX)**:
    - Dedicated child data table with columns, sorting, pagination, and search.
    - Header action `+ New [Relation]` opening in-place Alpine.js Create Modal without leaving the parent page.
    - Row action `Edit` opening in-place Edit Modal pre-filled with child record data.
    - Row action `Delete` with confirmation modal deleting/detaching the related record in-place.
    - Automatic redirection back to the parent edit screen (`/:id/edit`).
  - **Interactive Multi-Relation Tab Bar**:
    - Automatically generates an Alpine.js responsive tab bar (`activeRelTab`) when multiple relation managers are attached to a resource (e.g. Orders, Invoices, Addresses).
  - **Standardized REST API Endpoints for Nested Relations**:
    - `GET /admin/api/resources/:slug/:id/relations`: Metadata discovery of all attached relation managers.
    - `GET /admin/api/resources/:slug/:id/relations/:rel`: Paginated JSON listing of related child records.
    - `POST /admin/api/resources/:slug/:id/relations/:rel`: In-place record creation via REST.
    - `PUT /admin/api/resources/:slug/:id/relations/:rel/:child_id`: In-place record update via REST.
    - `DELETE /admin/api/resources/:slug/:id/relations/:rel/:child_id`: In-place record deletion via REST.
  - **Artisan Code Generator**:
    - `php artisan make:filament-relation <Resource> <Relation>` scaffolding type-safe relation managers.

---

## [v1.5.0] - 2026-10-31

### Added
- **Multi-Engine Admin Panels & Mobile-First Flutter Engine (`pkg/filament/panel.go`, `pkg/filament/api.go`, `pkg/flutterclient`)**:
  - `PanelEngine` selection (`filament.EngineWeb` SSR default or `filament.EngineFlutter`).
  - `panel.Flutter(distDir...)` fluent method enabling compiled Flutter Web SPA serving and cross-platform native clients.
  - **Standardized Panel REST API (`pkg/filament/api.go`)**:
    - `GET /admin/api/schema`: Complete panel schema discovery (navigation groups, resource definitions, tenant configuration, engine mode, brand metadata).
    - `GET /admin/api/resources/:slug`: Paginated resource listings with search, sort, and filters.
    - `GET /admin/api/resources/:slug/:id`: Single record retrieval.
    - `POST /admin/api/resources/:slug`: Record creation with policy validation.
    - `PUT /admin/api/resources/:slug/:id`: Record update with policy validation.
    - `DELETE /admin/api/resources/:slug/:id`: Record deletion with policy validation.
    - `POST /admin/api/resources/:slug/bulk`: Bulk operations.
    - `GET /admin/api/tenants` & `POST /admin/api/tenants/switch`: Tenancy management for mobile/SPA clients.
    - `GET /admin/api/widgets`: Real-time dashboard stats and charts.
  - **Mobile-First Responsive Flutter Client Generator (`pkg/flutterclient/scaffold.go`)**:
    - Material 3 theme with Dark & Light mode toggle.
    - Adaptive `ResponsiveLayout` supporting mobile phones, tablets, and desktop workstations.
    - **Mobile-First Experience**:
      - One-thumb bottom `NavigationBar` with icons, badges, and smooth animations.
      - Pull-to-refresh (`RefreshIndicator`) for instant table/feed updates.
      - Floating action buttons (`FloatingActionButton`) for one-tap record creation.
      - Bottom-sheet modal forms (`showModalBottomSheet`) optimized for single-hand mobile inputs.
      - Swipe-to-delete dismissible list cards (`Dismissible`) with confirmation feedback.
    - **Tablet & Desktop Experience**:
      - Expandable `NavigationRail` sidebar.
      - High-density data grid tables with column sorting.
      - Split-pane master-detail inspect view.
    - Cross-compiles to Web SPA (`flutter build web`), iOS, Android, macOS, Linux, and Windows (`flutter run`).
  - **Artisan Generator Commands (`pkg/commands/make_filament.go`)**:
    - `php artisan make:filament-panel [name] --engine=flutter`: Scaffolds a Flutter-ready panel provider.
    - `php artisan make:filament-flutter [panel] [--dir=client]`: Scaffolds a ready-to-run mobile-first Flutter admin client connected to GoFilament.

---

## [v1.4.0] - 2026-10-31

### Added
- **Filament Shield & Policy-Based Access Control (`pkg/filament/shield.go`)**:
  - `ShieldUser` interface with `HasRole` and `HasPermission` methods.
  - `SimpleShieldUser` standard model with multi-role hierarchy (`super-admin` > `admin` > user roles).
  - `Policy[T]` interface and `BasePolicy[T]` default allow-all implementation for complete CRUD control (`ViewAny`, `View`, `Create`, `Update`, `Delete`, `DeleteAny`).
  - `filament.Role(roles...)`, `filament.Permission(perm)`, and `filament.Gate(ability, args...)` authorization predicates.
  - **Panel-Level Authorization**:
    - `panel.CanAccess(func(ctx *filament.Context) bool)` guarding dashboard, global search, resources, and custom pages.
    - Automatic redirection to `/admin/login` for unauthenticated requests, and `403 Forbidden` for unauthorized users.
  - **Resource-Level Granular Guards**:
    - `CanViewAny`: Hides resource from sidebar navigation and blocks `/admin/{resource}` with 403.
    - `CanCreate`: Omit "New [Model]" button from table header actions and blocks `GET/POST /admin/{resource}/create` with 403.
    - `CanView`: Gates `GET /admin/{resource}/:id`.
    - `CanEdit`: Omit edit buttons from infolists and blocks `GET/POST /admin/{resource}/:id/edit` with 403.
    - `CanDelete`: Omit delete button from edit views and blocks `GET/POST /admin/{resource}/:id/delete` with 403.
    - `CanDeleteAny`: Blocks bulk delete action on `POST /admin/{resource}/bulk` with 403.
  - **GoLaravel Global Gate Integration**:
    - `res.AuthorizeWithGate("prefix")` automatically wiring GoLaravel's `auth.Gate` (`viewAny_{name}`, `create_{name}`, `update_{name}`, etc.).
  - **Row-Level Action Visibility**:
    - `tableAction.Visible(func(record T) bool)` conditionally displaying actions (e.g. Refund, Approve) on specific table rows.

---

## [v1.3.0] - 2026-10-31

### Added
- **Multi-Tenancy & Tenant Switcher Engine (`pkg/filament/tenant.go`)**:
  - `Tenant` interface and `TenantItem` model for workspace/organization isolation.
  - Automatic tenant resolution pipeline via:
    1. Custom callback (`panel.CurrentTenant(...)`)
    2. Route parameter (`:tenant` with `panel.TenantPathScoping(true)`)
    3. HTTP Cookie (`filament_tenant`)
    4. HTTP Header (`X-Tenant-ID`)
    5. Host subdomain (`acme.domain.com` with `panel.TenantDomainScoping(...)`)
    6. User tenant list loader (`panel.TenantLoader(...)`)
  - **GoLaravel Privacy Layer Integration**:
    - Automatically injects `privacy.WithTenantID` into standard `context.Context` and `c.Request.Context()`.
    - Seamless zero-boilerplate integration with GoLaravel's Ent `TenantScope` and `TenantMutationHook` preventing cross-tenant data leaks.
  - **Topbar Tenant Switcher Dropdown**:
    - Interactive Alpine.js dropdown switcher displaying tenant avatar/initials, name, and total organization count.
    - Tenant list with active checkmark and instant switch URLs (`/admin/tenant/switch?tenant={key}`).
    - Built-in links for Tenant Profile (`panel.TenantProfile(...)`) and New Tenant onboarding (`panel.TenantRegistration(...)`).
  - **Path Scoping Support (`panel.TenantPathScoping(true)`)**:
    - Tenant-scoped navigation links and `panel.TenantURL(ctx, subpath)` generating `/admin/{tenant}/...` routes.

---

## [v1.2.0] - 2026-10-31

### Added
- **Zero-Boilerplate Ent Driver Adapter (`pkg/entdriver`)**:
  - `entdriver.BindClient(resource, client.User)` or `gofilament.BindEntClient(resource, client.User)`.
  - Automatic query pagination (`Count`, `Limit`, `Offset`, `All`) on any Ent query builder.
  - Automatic `Create` and `Update` form data reflection mapper supporting typed setters (`SetName`, `SetEmail`, etc.).
  - Automatic string ID type conversion to model primary key type (`int`, `int64`, `string`, `uuid.UUID`).
  - Automatic `DeleteOneID(id)` and bulk delete execution.

---

## [v1.1.0] - 2026-10-31

### Added
- **Ent Auto-Scaffolding Generator (`artisan make:filament-resource Model --generate`)**:
  - AST schema inspector (`pkg/codegen`) scanning `app/schema/*.go` and `schema/*.go`.
  - Automatic Ent field mapping:
    - `field.String` (emails) $\rightarrow$ `forms.TextInput().Email().Required()`, `tables.TextColumn().Copyable()`
    - `field.String` (passwords/sensitive) $\rightarrow$ `forms.TextInput().Password().RequiredOnCreate().Hash()`
    - `field.Enum` $\rightarrow$ `forms.Select().Options()`, `tables.BadgeColumn()`, `tables.SelectFilter()`
    - `field.Bool` $\rightarrow$ `forms.Toggle()`, `tables.IconColumn().Boolean()`, `tables.TernaryFilter()`
    - `field.Time` $\rightarrow$ `forms.DateTimePicker()`, `tables.TextColumn().DateTime()`
    - `field.Float / Int` (money) $\rightarrow$ `forms.TextInput().Numeric().Prefix("$")`, `tables.TextColumn().Money("USD")`
  - Automatic `RelationManager` generation for graph edges (`edge.To("items", LineItem.Type)` $\rightarrow$ `relations/items_relation_manager.go`).

---

## [v1.0.0] - 2026-10-31

### Initial Production Release 🎉

GoFilament is a premier, batteries-included administrative toolkit and rapid application framework built natively on top of **GoLaravel** and **Ent ORM**.

---

### ⚡ Panel Builder Architecture (`pkg/filament`)
- **Fluent Panel Configuration**: Multi-panel support (`admin`, `app`, `portal`), configurable base path, brand name, logo, favicon, dark mode, collapsible desktop sidebar, and navigation grouping.
- **Global Search**: Cross-resource search with debounced autocomplete query dropdown at `/admin/global-search` supporting both JSON and HTML modes.
- **Built-In Authentication**: Pre-configured login screen (`/admin/login`), session handling, and topbar notification drawer.
- **Context Management (`filament.Context`)**: High-performance request context wrapper providing panel metadata, active tenant, and authenticated user.

---

### 📝 Form Builder Engine (`pkg/forms`)
- **15+ Input Fields**: `TextInput`, `Textarea`, `Select`, `Toggle`, `Checkbox`, `CheckboxList`, `Radio`, `DatePicker`, `TimePicker`, `FileUpload`, `ColorPicker`, `TagsInput`, `KeyValue`, `CodeEditor`, `MarkdownEditor`, and `Hidden`.
- **Layout Containers**: `Section` (with headers and descriptions), `Grid` (multi-column responsive), `Tabs`, `Fieldset`, `Split`, `Wizard` (multi-step progress flow), and `Repeater` (dynamic nested arrays).
- **Server-Driven Reactivity (`.Live()`)**: Micro-swaps powered by HTMX (`POST /admin/:resource/live`) and dynamic `.AfterStateUpdated()` callbacks with sub-millisecond execution (10.8 µs form rendering).

---

### 📊 Table Builder Engine (`pkg/tables`)
- **Compile-Time Generics (`Table[T]`)**: Zero reflection overhead for data rendering.
- **Columns**: `TextColumn` (with date/money formatting and copyable tooltips), `BadgeColumn` (dynamic status colors), `IconColumn` (boolean checkmarks and state-mapped SVGs), `ColorColumn` (hex preview swatches), `ToggleColumn`, `SelectColumn`, and `TextInputColumn`.
- **Filtering & Search**: `SelectFilter`, `TernaryFilter`, column-specific search, and global table query matching.
- **Actions & Bulk Operations**: Inline row actions (`ViewAction`, `EditAction`, `DeleteAction`), header actions, multi-record bulk delete, and native streaming CSV export.

---

### 🗂️ Ent Resource Engine (`Resource[T]`)
- **CRUD Lifecycle**: Automatically generates `ListRecords`, `CreateRecord`, `ViewRecord`, `EditRecord`, `DeleteRecord`, and `BulkActions`.
- **Relation Managers (`RelationManager`)**: Nested parent-child relationship tables embedded within resource pages.

---

### 📈 Infolists, Widgets & Actions
- **Infolists (`pkg/infolists`)**: Read-only schemas with `TextEntry`, `BadgeEntry`, `IconEntry`, `ImageEntry`, `RepeatableEntry`, and `Section`.
- **Dashboard Studio (`pkg/widgets`)**: `StatsOverviewWidget` with dynamic SVG sparklines, `ChartWidget` (Chart.js line, bar, pie), and `TableWidget`.
- **Action Engine (`pkg/actions`)**: Modal forms, slide-overs, confirmation dialogs, and policy authorization checks.
- **Notifications Engine (`pkg/notifications`)**: Toast alerts with success, warning, danger, and info levels.

---

### 🛠️ Artisan CLI Generators (`pkg/commands`)
- `artisan make:filament-panel <Name>`
- `artisan make:filament-resource <Name>`
- `artisan make:filament-page <Name>`
- `artisan make:filament-widget <Name>`
- `artisan make:filament-relation <Resource> <Relation>`
