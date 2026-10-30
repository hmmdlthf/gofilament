# Changelog 🚀

All notable changes to the **GoFilament** framework will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
