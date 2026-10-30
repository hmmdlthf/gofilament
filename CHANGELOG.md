# Changelog 🚀

All notable changes to the **GoFilament** framework will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
