# Changelog 🚀

All notable changes to the **GoFilament** framework will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added
- **Reorder in Flutter**: reorderable tables get a Reorder button in the Flutter list (a drag list of the page's rows). `filament.ReorderOffset(ctx)` gives `OnReorder` the position of the first id (web and API send the page's start).
- **Widget visibility**: every built-in widget has `VisibleWhen(fn)` (and `CanView`), like `FlutterWidget`.
- **Flutter language and RTL**: the app lays itself out right to left when the panel's language is RTL, sends the chosen language as `Accept-Language`, and offers a language menu (account menu) when the panel has several `Locales`; the choice is remembered on the device.
- **Accept-Language**: panels with `Locales` pick the first supported language from the `Accept-Language` header when no `?lang` or cookie chooses one (API clients).
### Fixed
- **Reordering pages and filtered lists**: only the current page's ids were sent, with nothing telling where the page starts, so saving positions on page 2 collided with page 1. Drag handles also showed while searching, filtering, grouping or sorting by another column, where the visible order is not the real one; they now show only on the plain list (or sorted by the reorder column).
- **Reorder without a handler** answered success; it is now 404.
- **Security: tenant could be chosen by the request**: without a `TenantLoader`, the tenant named by the `X-Tenant-ID` header, the tenant cookie, the path or the subdomain was trusted as is, so a user could act in another tenant. With a loader that returned no tenants, the raw key was used too, and a tenant guessed before the user was known stayed in place. Now only tenants from `TenantLoader`, or keys allowed by the new `TenantConfig.CanAccessTenant(ctx, key)`, are used. **Upgrade:** panels using tenancy without a `TenantLoader` must set `CanAccessTenant`, otherwise no tenant is selected.
- **API tenant switch** accepted any tenant key and did not call `OnTenantSwitch`; it now checks membership like the web switch (403 otherwise), calls the hook and sets a Secure cookie on HTTPS.
- **Hidden widgets leaked data**: the web dashboard ignored a widget's `CanView`, and `/widgets/:id/data` (web and API) served any widget's chart data by id. Hidden widgets are now left out everywhere and their data answers 404.
- Removed a stray `pkg/filament/api.go.orig` committed by accident.


## [v1.36.0] - 2026-11-17

### Added
- **Bulk edit**: `Resource.BulkEditable(fields...)` and `OnBulkUpdate` set one field on many selected records (web *Change Selected* bulk action, Flutter *Change* button, `POST /resources/:slug/bulk-update`), with field validation and per-record `CanEdit` / trashed checks; failures are reported per record. `tables.EditBulkAction()`.
- **Background jobs**: `Panel.BackgroundJobs(JobsConfig)` runs exports (web; API with `background=1`) and imports (web preview flow; API with `background=1`) after the request, with progress, a notification and a download link when done. `/admin/jobs/:id` page, API `/jobs`, `/jobs/:id`, `/jobs/:id/download` (owner only). Flutter exports use server jobs when the panel has them; `adminSaveFile` can be replaced.
- **Impersonation**: `Panel.Impersonation{CanImpersonate, OnImpersonate}` lets allowed users sign in as another user (web, with `Panel.Sessions`) via `ImpersonateURL(id)`, shows a banner with *Return to my account*, blocks account changes and nested impersonation while impersonating; `filament.ImpersonatorID(ctx)`.
- **Flutter notification bell**: with `Panel.DatabaseNotifications`-style notifications on (`schema.notifications`), the Flutter topbar shows a bell with the unread count (checked every minute) and a list to read, open (action links; background-job downloads go through the API), mark all read and clear.

### Fixed
- **Web bulk actions needed delete rights**: every bulk action (export, restore, ...) was refused without `CanDeleteAny`; only bulk delete (and unknown actions) need it now.
- **API bulk delete hid errors**: without a bulk delete handler, per-record delete errors were ignored and the request reported success.
- **Broadcast notifications (memory store)**: one user marking a notification sent to everyone as read marked it read for every user; reads are now per user, as in the SQL store. `List` returns copies sorted newest first (the user's own and broadcast notifications were not interleaved by time).

## [v1.35.0] - 2026-11-17

### Added
- **Edit conflicts for inline cells and related records**: rows of a `Versioned` resource carry their version (`data-version` on the web, `row_meta.version` in the API); inline cell saves send it, get 409 when the record changed meanwhile, and return the new version. `RelationManager.Versioned(fn)` checks related-record edits (web modal and API, version in the relation form). The version is only compared after edit rights are checked.
- **Self-registration and email verification**: `Accounts.Register` turns on `/register` (web) and `POST /api/register` (with `Panel.Registration()`); `Accounts.EmailVerified` / `MarkEmailVerified` / `SendVerificationLink` send a signed verification link and keep unverified users on a "verify your email" page (web) or a 403 `email_unverified` (API). Flutter gets a sign-up screen and a verify-email screen (`AdminStage.verifyEmail`).
- **Account screens in Flutter and the API**: `/account` (profile), `/account/password`, `/account/two-factor/*` (setup, confirm, turn off, new recovery codes) and `/forgot-password` for bearer-token clients; `boot.auth.accounts` lists what is available. The Flutter app adds a *Your account* screen, a *Forgot your password?* dialog and the two-factor code step at sign-in.
- **Two-factor recovery codes**: with `Accounts.RecoveryCodes` / `SaveRecoveryCodes`, turning two-factor on makes 8 single-use codes (shown once, stored as SHA-256 hashes) that work at the sign-in challenge in place of an authenticator code. The profile shows how many are left and can make a new set; turning two-factor off removes them.
- **Flutter phone layout**: the card list now shows group headers, group subtotals and the "This page" / "All results" summaries, like the table.
- **Import preview and Excel**: imports (web, API, new Flutter Import dialog) now preview the file first: columns are matched to form fields (editable), every row is validated, and the problems are listed before the valid rows are imported. `.xlsx` files can be imported; exports gain `format=xlsx` and `columns=`. API: `dry_run`, `mapping`, `file_base64`; results include per-row field errors (`rows`). Resources report an `import` ability to API clients.
- **Saved views**: `Panel.SavedViews(store)` lets users save the current list state (search, sort, filters, advanced search, columns, grouping, page size) under a name and share it; a *Views* menu on web and Flutter lists, applies and deletes them. `SavedViewsShareable` controls sharing; `sqlstore.Views()` keeps them in the database. API under `/resources/:slug/views`.
- **Record history**: `activitylog.EnableAuditLog` now gives a resource a history provider (`Resource.History`): the web view/edit pages show a *History* section, `GET /resources/:slug/:id/history` serves it, and the Flutter record form has a History button. `HistoryVisible`, `EnableAuditLogWith(res, AuditOptions{Except, NoHistory})`, `filament.SetCreatedID`. Inline cell edits and bulk deletes are logged.

### Fixed
- **API sign-in skipped two-factor**: users with two-factor sign-in on could get an API token (and sign in to the Flutter app) with the password alone. `POST /login` now returns a challenge, and the token is only handed out by `POST /login/two-factor` after a valid code.
- **Imports skipped form validation**: imported rows went straight to `OnCreate`, without the create form's rules. They are now validated like the create form.
- **Web Export ignored the list's filters**: the Export header action always exported every record as CSV. It now offers CSV and Excel of the current search, filters, advanced search and sort.
- **Header actions hidden**: when a resource allowed creating records, the list replaced the table's header actions (Import, Export, custom ones) with the New button.
- **Audit log diffs**: updates compared the whole stored record with the submitted form, so untouched fields showed as removed and differently named keys (`FullName` / `full_name`) as changed; only real changes are recorded now, and no-op saves are not logged.
- **Audit log stored secrets**: password values were stored in created / updated entries. Password fields and `*password*` / `*secret*` / `*token*` fields are now hidden.
- **Web edit form empty for struct records**: the edit page only filled the form when records were maps. Struct records now fill it too (loose name matching); password fields are never filled.
- **Soft deletes**: the "trashed" filter is now part of the API schema, so the Flutter app can show and restore trashed records. Trashed records can no longer be edited, inline-edited or deleted again (409; the web edit page offers Restore). Restore and force delete fall back to the resource's delete permission (`CanDeleteAny`) when `CanRestore` / `CanForceDelete` are not set, answer 404 on resources without soft deletes or handlers, and report handler errors instead of claiming success (single and bulk).

## [v1.34.0] - 2026-11-17

### Added
- **Account screens (web engine)**: `Panel.Accounts(...)` adds a profile page (name and email), change password, forgot / reset password and two-factor sign-in with an authenticator app (set up and turned off from the profile page). Built on golaravel's hash, reset-token broker, TOTP, cache and sessions; reset and two-factor requests are throttled, codes and reset links are single-use, and a password change signs out the user's other devices.
- **Range filters (`pkg/tables`)**: `DateRangeFilter` and `NumberRangeFilter`, with web inputs and a Flutter range dialog (date picker for dates). The value is `"from,to"`; `Condition` / `Rules` turn it into advanced-search rules for `BuildAdvancedSearchSQLFor`, and `Matches` checks in-memory values. New `tables.FilterValues(query)` collects a request's filter values.
- **Table summaries and grouping**: `Summarize(...)` on columns (sum, average, count, min, max) for the page and each group, `Resource.SummariesUsing` for totals over all matching records (also sent as `summaries` by the API), and `Table.Groups` / `DefaultGroup` for grouped rows with headers and subtotals. Web tables and the Flutter list.
- **Edit-conflict detection**: `Resource.Versioned(fn)` makes edit forms (web and Flutter) send the version they were opened with (`_version`); a save after someone else's change is refused with 409 and offers Reload or Overwrite. The form and record API responses include `version`; `filament.ExpectedVersion(ctx)` gives `OnUpdate` the expected version for a conditional write. Inline cell edits are not checked.

### Fixed
- **Topbar user menu**: the web topbar showed a fixed "Administrator" label with no way to sign out, and the user menu (`Panel.UserMenu`) was never rendered. It now shows the user's name with Profile (when available), Sign Out and custom items.
- `Profile()` and `PasswordReset()` were flags with no screens behind them; with `Accounts` configured they now have pages.
- **Web checkbox filters**: only the first checked value of a refined filter reached the list handler. Repeated `filter_*` values are now joined with `,`, as the API and Flutter already sent them.
- **Web pagination links** dropped the search, sort, page size and filters; they now keep the list state.

## [v1.33.1] - 2026-11-17

### Fixed
- **Advanced search SQL (`pkg/tables`)**: `BuildAdvancedSearchSQL` only produced Postgres `$N` placeholders, so the query failed on SQLite and MySQL. New `BuildAdvancedSearchSQLFor(cond, columns, dialect)` uses `?` placeholders for SQLite and MySQL (and `CAST(... AS CHAR)` on MySQL). `BuildAdvancedSearchSQL` keeps its Postgres output.
- **LIKE wildcards**: `%` and `_` typed into contains / starts with / ends with rules are now matched literally (`ESCAPE '!'`) instead of matching everything.

## [v1.33.0] - 2026-11-17

### Added
- **Flutter lists (`flutter/gofilament_admin`)**:
  - **Resizable columns**: drag a header's handle to set a column's width, double-tap it to go back to automatic. Widths are remembered per resource, like the column picker.
  - **Refined filters**: `RefinedFilter` filters open a checkbox list with the server's live counts and send the checked values (`filter_<column>=a,b`). Before, they showed as single-choice menus.
  - **Advanced search**: resources with `AdvancedSearch(cols...)` get a rule builder (field, operator, value, match all or any) sent as `advanced_search`, which `tables.GetAdvancedSearch` parses. Export uses the same rules.

## [v1.32.0] - 2026-11-17

### Added
- **Panel login on golaravel sessions**: `Panel.Sessions(mgr, users)` runs the Templ engine's sign-in, sign-out and CSRF on golaravel's session manager (any driver: SQL, Redis, file, memory, cookie), using `auth.Login` / `auth.Logout`. Sessions survive restarts with a persistent driver, are linked to the user (`mgr.DestroyUserSessions`), and `auth.User(c)` works in app handlers. Requires golaravel v1.17.0.
- **`Panel.BearerGuard(guard)`**: authenticate API / Flutter bearer tokens with a golaravel guard such as `auth.BearerGuardFunc(jwt, provider)`.
- **Opt-in demo account button (`flutter/gofilament_admin`)**: `GoFilamentAdmin(demoLogin: DemoLogin(email:, password:, label:, debugOnly:))` shows a "fill in the demo account" button on the sign-in screen. Without the option there's no button, the package contains no credentials, and by default (`debugOnly: true`) the button only appears in debug builds.

### Security
- **Legacy scaffold demo shortcuts (`flutterclient.GenerateFlutterAdmin`)**: the generated sign-in screen showed three "quick demo" chips that fill in `admin` / `admin123`, including in release builds. They are now inside a `kDebugMode` check, so release builds of generated apps never offer an account.

### Deprecated
- `Panel.WebSessionsUsing` and the in-memory web session fallback: use `Panel.Sessions`. Using the `TokenValidator` token as the web login cookie is superseded by `Panel.Sessions` too (it stays supported for bearer tokens).

## [v1.31.0] - 2026-11-16

### Breaking
- **Tenant switch:** a GET to `<panel>/tenant/switch` from outside the panel (scripts, bookmarks) shows a confirmation page; programmatic callers should POST.
- **Support API:** clients can no longer choose ticket IDs or claim `user_id` / `user_email` / `tenant_id`. Posting to a ticket that doesn't exist answers 404 instead of creating it. By default, support staff are signed-in users who pass `CanAccess`; use `Panel.SupportStaff` to narrow that.
- **GET delete / restore / logout from outside the panel** (scripts, bookmarks, old browsers without `Sec-Fetch-Site`) now gets a confirmation page instead of acting. Use POST for programmatic calls.

### Security
- **Cross-site requests**: every state-changing web (non-API) panel request now goes through `net/http.CrossOriginProtection` (Sec-Fetch-Site / Origin), so a form on another site can no longer create, edit, delete, bulk-act or run actions. Non-browser clients that send neither header are unaffected. `Panel.TrustedOrigins(...)` allows extra origins.
- **GET deletes and logout**: delete, restore, force-delete, relation-delete and logout links act on GET only when the browser reports `Sec-Fetch-Site: same-origin`. Any other GET shows a confirmation page that POSTs, because SameSite=Lax cookies are sent on cross-site top-level GETs.
- **Notification stream**: `GET <panel>/notifications/stream` required no access check and took the user from a client-sent `X-User-ID` header, so anyone could read anyone's notifications. It now requires panel access and uses the authenticated user.
- **Tenancy**: the tenant was resolved before the user, so a `TenantLoader` that filters by membership saw no user, and any tenant key in a cookie or header was accepted. It is now resolved again once the user is known. With a `TenantLoader`, switching to a tenant it does not list answers 403. The tenant cookie is `Secure` on HTTPS.
- **Open redirects**: locale and tenant switches redirected to any `Referer`. They now only return to a path on the same host.
- **Support API: tickets were open to anyone.**
  - **Reading and writing any ticket:** `POST /support/message` took any `ticket_id`, returned the whole ticket (every message, the customer), and created a ticket under a client-chosen ID when it didn't exist.
  - **No access check at all:** `GET /support/audio/:id` served voice recordings, and `POST /support/escalate` changed tickets.
  - **Claimed identities:** `user_id`, `user_email` and `tenant_id` from the request body were trusted.
  - **The fix:** identity now comes from the session (visitors are `guest`), new tickets get a 128-bit server-generated ID, and every ticket operation needs the ticket's owner or support staff in scope (`Panel.SupportStaff`). "Not yours" answers 404. Escalating to the super admin panel is staff-only, customers list only their own tickets, and recordings are only ever served as audio.
- **Global search** listed records from every searchable resource without `CanViewAny` / `CanView`; both checks now apply.
- **Reordering** (web and API) only checked panel access; it now needs edit rights on every reordered record.
- **CSV export** wrote cells starting with `=`, `+`, `-` or `@` as-is (spreadsheet formula injection); such cells, other than plain numbers, are prefixed with a quote.
- **Notification links** allow only `http(s)` and relative URLs, so a `javascript:` link can't run script.
- **Tenant switch from another site**: `GET <panel>/tenant/switch` now switches only when the browser reports `Sec-Fetch-Site: same-origin` (a click inside the panel). Any other GET shows a confirmation page that POSTs, so a link or image on another site can't silently change the tenant a signed-in user is working in.
- **In-memory web sessions** log a one-time warning on the first web login without a persistent store (`Panel.WebSessionsUsing`, `pkg/sqlstore`) or a `TokenValidator`, since such sessions are lost on restart and not shared between instances.

### Changed
- **GoLaravel v1.16.0**: 128-bit support ticket IDs (`support.GenerateTicketID`).
- **GoLaravel v1.15.0**: fixes a 2FA bypass in its auth workflows (challenge tokens worked as access tokens), throttles logins and 2FA, and recovers panics in scheduled tasks, event listeners and queue jobs. GoFilament doesn't use those workflows.
- **GoLaravel v1.14.0**: security pass, covering broadcast channel authorization and the WebSocket origin check, a random storage signing key, a tenant `Authorize` option, required STARTTLS, and limits on HTTP responses and metrics. GoFilament's notification stream uses a public SSE channel that it authorizes itself, so it is unaffected.
- **GoLaravel v1.13.0**: form method spoofing (`_method` / `X-HTTP-Method-Override`) now routes POST forms to PUT, PATCH and DELETE handlers; `.env` parsing fixes (export lines, inline comments, escapes).
- **GoLaravel v1.12.0**: request body read timeout (2 minutes by default, per-route override); cache stampede protection and atomic `Pull`; new validation rules (`Password()`, `exclude_*`, `required_array_keys`, `dimensions`); a working `make:oauth` scaffold.
- **GoLaravel v1.11.0**: JWTs signed with public/private keys (RS256, ES256, EdDSA) with JWKS and key rotation; `middleware.RateLimiter` delegates to `pkg/ratelimit`; `ratelimit.ThrottleMax` routes no longer share one counter.
- **GoLaravel v1.10.0**: CI with real MySQL / Postgres / Redis integration tests; one shared Redis client; atomic Redis rate limiting (bursts can no longer exceed the limit); delayed jobs on any queue; fixes for SQL cache increments on Postgres, SQL cache locks on MySQL and a container singleton data race.
- **GoLaravel v1.9.0**: adds a Laravel-style validation rule engine (`validation.Make`, `Rules()` on form requests, `ValidateRequest`).
- **GoLaravel v1.8.0**: adds SQL and Redis stores for sessions, the cache and password reset tokens (`session.NewSQLDriver` / `NewRedisDriver`, `cache.NewSQLStore` / `NewRedisStore`, `auth.CacheResetTokenStore`).
- **GoLaravel v1.7.0**: fixes mail header injection and adds a telemetry `Authorize` option, Content-Security-Policy options in `SecureHeaders`, and JWT issuer / audience / key rotation.
- **GoLaravel v1.6.0**: secure defaults for sessions, CORS, request body size (32 MiB), the queue dashboard and JWT. See the GoLaravel changelog for its breaking changes. None of the changed APIs are used by GoFilament.
- **GoLaravel v1.5.0**: `go.mod` now pins the released GoLaravel (`github.com/hmmdlthf/golaravel v1.5.0`) instead of a `../golaravel` checkout, so GoFilament builds anywhere without a sibling folder. v1.5.0 brings trusted-proxy handling for `ClientIP()`. Building needs access to the private repository (`GOPRIVATE`, see the README). For local GoLaravel work, use a `go.work` workspace.

## [v1.30.0] - 2026-11-13

### Breaking
- **Server-side form validation (`pkg/filament`, `pkg/forms`)**: create, update, inline cell edits, relation forms and row-action forms are validated against the declared form, and handlers receive only the form's fields (unknown keys, disabled fields, fields hidden by `Visible` and non-dehydrated fields are dropped). Invalid input answers 422 with `{"error", "errors"}`. Updates are partial: absent keys are left alone. Handlers that relied on extra keys must declare them as fields (for example `forms.Hidden`) or opt out with `AcceptUnvalidatedInput()`.
- **Inline cell edits** only reach editable table columns (and select columns only accept their options); the fallback to `OnUpdate({column: value})` can no longer write arbitrary fields. A resource without a table but with its own `OnCellUpdate` still decides by itself.
- **Writes need a loadable, visible record**: update, delete, inline edit and bulk delete answer 404 when the record cannot be loaded and 403 when `CanView` refuses it, instead of running the handler.
- **Sign-in is throttled** by default (5 failures per client and name per minute, 20 per client): `Panel.LoginThrottle(n, window)`; `0` disables it.
- **CORS (`pkg/filament`)**: the panel API no longer reflects every `Origin` with `Access-Control-Allow-Credentials: true`. By default only same-origin requests get CORS headers. Allow other origins with `Panel.CORSAllowedOrigins("https://app.example.com")` (reflected with credentials), `CORSAllowedOrigins("*")` (any origin, no credentials) or `CORSAllowOriginFunc(fn)` (for example `http://localhost:*` during development). Native apps and server-to-server clients send no `Origin` and are unaffected. A Flutter web client served from another origin (such as `flutter run -d chrome`) needs one of these.
- **`flutterclient.GenerateAdminApp`**: existing files are no longer overwritten on re-runs, so edits to `lib/main.dart`, `pubspec.yaml`, `.gitignore`, etc. are kept. Set `AdminAppOptions.Overwrite: true` for the old behaviour.

### Security
- Update / delete / cell update used to skip `CanEdit` / `CanDelete` when `OnFind` returned an error, and never checked `CanView`; bulk delete only checked `CanDeleteAny`, and bulk restore / force delete skipped `CanRestore` / `CanForceDelete`. All of them now check every record.
- Form input was passed to handlers unchecked (mass assignment through any key, including disabled fields and, via inline edit, any column). See Breaking.
- The web create page printed handler error messages unescaped; they are escaped now.
- **Web login (`pkg/filament`)**: the login form and row action forms carry a CSRF token (`_token`, double-submit cookie, compatible with golaravel's `middleware.CSRF`). New `Panel.CSRFToken` / `VerifyCSRF` for custom pages.

### Added
- `forms.Validate(ctx, components, data, creating)` and `forms.ValidationError`; `BaseResource.AcceptUnvalidatedInput`, `BaseRelationManager.AcceptUnvalidatedInput`; `filament.ErrRecordNotFound`, `filament.ErrColumnNotEditable`; `Panel.LoginThrottle`. `Visible` and `Dehydrated` on fields are now applied on the server.
- **Typed Dart from a panel (`flutterclient.GenerateFromPanel`)**: boots the panel in-process (httptest, no network, no writes) and writes `lib/generated/{schema,models,client,pages}.g.dart` into a Flutter app: a schema and form snapshot, a model class per resource record (from the Go record type) and per form / action / relation input, a typed `PanelClient`, and the `AdminPage` list of the panel's Flutter pages. Output is deterministic; `CheckFromPanel` and `RenderFromPanel` support CI. New artisan command `commands.FlutterGenerateCommand` (`filament:flutter-generate [--out=dir] [--check]`).
- **Persistent runtime state (`pkg/sqlstore`)**: SQL-backed stores for web-engine sessions (`filament.WebSessionStore`, `Panel.WebSessionsUsing(store, userLoader)`; session ids stored hashed; `DeleteUserSessions`), notifications (`notifications.NotificationStore`, per-user read state for broadcasts), the activity log (`activitylog.Store`, filters, literal search, pagination) and support tickets (golaravel `support.Store`, transactional updates). SQLite, MySQL and Postgres; `Migrate` creates `gofilament_*` tables. The web login now reports a failure to store the session instead of signing in without one.
- **Build and embed**: `Panel.FlutterFS(fs.FS)` serves the Flutter web build from any file system (for example `go:embed`) with the same behaviour as the directory server. `flutterclient.BuildWeb` runs `flutter build web --release` and, with `EmbedDir`, writes an embeddable package (`dist/` + generated `embed.go` exposing `FS`). Artisan `commands.FlutterBuildCommand` (`filament:flutter-build [--dir] [--embed=dir] [--no-generate] [-- flutter args]`) regenerates the typed code first.
- **gofilament_admin renderer**: repeaters, key-value fields, real tabs and wizard steps; file and image fields (upload on pick, `adminPickFiles`, new `file_picker` dependency); trash / restore / delete-for-good on soft-deleted rows; a remembered column picker; chart widgets with period filters; global search; exports saved through a save dialog on native platforms. `AdminApi.upload`, `restoreRecord`, `forceDelete`.
- **Uploads API**: `POST <panel>/api/resources/<slug>/uploads?field=` stores a file for a form's file field after checking `MaxSize` and `AcceptedFileTypes` against the sniffed content; files go to the field's storage disk or `Panel.UploadsUsing(fn)`. `FileUploadField.CheckUpload`, `GetDisk`, `GetDirectory`, `PreservesFilenames`; the server also checks the number of files a file field receives.
- **Row meta for soft deletes**: `trashed`, and for trashed rows `restore` / `force_delete` (per record, `CanRestore` / `CanForceDelete`), with update, delete and custom actions turned off.
- **Custom Flutter components**: `forms.Custom(name, component)` (with `.Props`, `.Rule` checked on the server), `tables.CustomColumn[T](name, component)` and `filament.FlutterWidget(id, component)` (`.Data` computed per request, `.Visible`). gofilament_admin renders them through `GoFilamentAdmin(components: AdminComponents(fields:, cells:, widgets:))` with `AdminFieldContext`, `AdminCellContext`, `AdminWidgetContext`, and falls back safely when one is missing. The generator creates `lib/components/{fields,cells,widgets}/<name>_<kind>.dart` stubs once, typed `<Name><Kind>Props` classes inferred from the Go props, and `panelComponents` in `registry.g.dart`. `forms.Node` and `tables.ColumnDTO` gain `component` / `props`; widgets with `CanView(ctx) bool` are left out of `/api/widgets` for users they reject.
- **`resources/flutter` convention**: hand-written Flutter pages live in `lib/pages/<id>_page.dart` (class `<Id>Page`); `GenerateFromPanel` also writes `registry.g.dart` (`panelAppPages`) and creates a stub for each declared page without a file (never overwriting one). New commands `make:filament-flutter-app` (typed app in `resources/flutter`, `AdminAppOptions.Typed`) and `make:filament-flutter-page <id>`; `flutterclient.PageFile`, `PageClass`, `PageStub`, `commands.DefaultFlutterDir`.
- **`filament.FlutterPage(id, label)`**: a page drawn only by the Flutter client (`FlutterPager` for your own page types). The schema lists the pages the user may open in `client_pages`.
- **`filament.Endpoint[Req, Res](name, method, path, fn)`** + `Panel.Endpoints(...)`: typed JSON endpoints under `<panel>/api` behind the panel's authentication (`.Authorize`, `.Describe`); the generator writes a typed Dart method for each.
- `filament.WithClientGenerator` (in-process identity for the generator; not reachable over HTTP), `Panel.GetResources`, `Panel.GetPages`, `Panel.GetEndpoints`, `BaseResource.RecordType` / `DescribeActionForms`, `BaseRelationManager.RecordType`.
- `forms.Node.DynamicOptions` (`dynamic_options`): marks select options resolved per request (`OptionsUsing`).
- **gofilament_admin**: `GoFilamentAdmin(initialSchema: ...)` / `AdminController.initialSchema` draw the shell from a compiled-in snapshot when a session is stored, then switch to the live schema (`schemaIsLive`); `AdminPage.serverControlled` shows a page only when the live schema lists it; `AdminApi.find`; `GfJson` and `TypedPage` for generated code.
- **Web engine login**: `POST <panel>/login` now calls the `AuthValidator` and starts a session in an HttpOnly, SameSite=Lax cookie scoped to the panel path (`Panel.SessionCookieName()`); `GET`/`POST <panel>/logout` ends it. With a `TokenValidator` the cookie carries the validator's token and is re-validated on every request; without one, an opaque id points at an in-memory session store (lost on restart, single process). The cookie is only read outside `<panel>/api`, which stays bearer-token only.
- **Web engine row actions**: a `tables.Action(name).Handle(...)` now links to `<base>/<id>/actions/<name>`, a confirm page (showing the action's form, if any) that POSTs, CSRF-checked, to run it. Visibility, `Authorize` and view permissions are checked as in the API. Actions without a runner keep their old link.
- `TableAction.GetModalHeading`, `GetModalDescription`, `GetColor`.

### Deprecated
- `flutterclient.GenerateFlutterAdmin` (the legacy scaffold): use `GenerateAdminApp` plus `GenerateFromPanel`.

### Fixed
- **`Panel.Middleware`** was stored but never applied. It now runs on every panel route, including `/api/*`, except static assets, the Flutter web bundle and CORS preflight. Call it before `Register` / `Mount`.
- **Tables / infolists**: column names resolve through the struct's `json` tag and Go initialisms (`install_id` -> `InstallID`, `remote_ip` -> `RemoteIP`, `customer_id` -> `CustomerID`), so Ent entities show those fields.
- **`flutterclient.GenerateAdminApp`**: `web/index.html` uses `<base href="$FLUTTER_BASE_HREF">`, so `flutter build web --base-href` works. The generated `.gitignore` also ignores `.flutter-plugins` and `.flutter-plugins-dependencies`.

## [v1.29.0] - 2026-11-11

### Added
- **Link row actions**: a `tables.Action` with `URL(...)` and no `Handle` is now a link action. The schema marks it `link: true`, and each visible row's metadata carries its URL under `links`. Hidden rows never receive the URL. Link actions are never run on the server.
- **Flutter admin (`flutter/gofilament_admin`)**: link actions open their URL in a new browser tab. `openLink` is exported so host apps (or tests) on other platforms can replace it, for example with url_launcher. When a URL cannot be opened, it is copied to the clipboard instead.

## [v1.28.1] - 2026-11-11

### Added
- **Flutter admin (`flutter/gofilament_admin`)**: `AdminApi.getText` for panel endpoints that return HTML or CSV (for example printable receipts).

### Changed
- **Flutter admin deep links**: `#/pages/<id>` now also opens full-screen pages, so a device such as a till terminal can bookmark a full-screen page directly.

## [v1.28.0] - 2026-11-11

### Security
- **Access checks (`pkg/filament`)**: `AuthorizeAccess` wrote the 401 / 403 response but returned a nil error, so handlers continued running for refused requests (an unauthenticated request could run a create, update or row action). It now returns `ErrAccessDenied`; every caller stops on it.
- **Relation endpoints**: previously had no permission check. Reading related rows now needs view rights on the parent record; changing them needs edit rights.

### Added
- **Reusable Flutter admin (`flutter/gofilament_admin`)**: schema-driven admin client (login, grouped navigation, dashboard stats, lists, forms, relation panels, row actions, client-side CSV / Excel export, deep links, custom page slots). `flutterclient.GenerateAdminApp` scaffolds a thin app on it.
- **Client API**: form schema JSON (`forms.Describe`), `GET /api/boot`, `/resources/{slug}/form` and `/form/options`, relation form endpoints, row action endpoints, `row_meta` in list responses, `resource_meta` and `export` in the schema.
- **Forms**: `SelectField.OptionList`, `OptionsUsing`, `SearchUsing`; `ToggleField.Default`.
- **Panels**: `BrandUsing` (per-request brand), `CurrencyUsing`, `MaxPerPage`, `ExportMaxRows` / `ExportMaxRowsUsing`, `APIErrorHandler`; `FromContext`, `RelationParentID`.
- **Tables**: `TableAction.Handle` / `Form` / `Authorize` / `SuccessMessage`, `BadgeColumn.Labels`, `Money("")` uses the panel currency, richer column metadata.
- **Ent adapter**: list search / sort / filter, `BindRelation`, eager-loading `Query` hooks.

### Fixed
- Ent adapter form mapping (initialism setters such as `SetCategoryID`, empty values, JSON numbers, `TextUnmarshaler` targets).
- Generated Flutter client compiled with errors (duplicate methods, missing import).

## [v1.27.0] - 2026-11-10

### Added
- **Ent Adapter List Options (`pkg/entdriver`)**:
  - Resource lists bound with `entdriver.Bind` now apply quick search, equality filters and ordering. Configure with `Config.SearchColumns`, `SortColumns`, `FilterColumns` and `DefaultSort`.
  - New `ListOptions`, `ApplyListFilters`, `ApplyListOrder`, `ApplyListOptions` and `ExecuteListQuery`; the total is counted before `ORDER BY` / `LIMIT`.
- **API Error Mapping (`pkg/filament`)**:
  - `ErrForbidden`: operations denied by an authorization callback now answer `403` instead of `500` / `400`.
  - `Panel.APIErrorHandler(fn)`: map handler errors to an HTTP status and a client-safe message (for example unique-constraint violations to `409`).

### Fixed
- **Ent Adapter Form Mapping (`pkg/entdriver`)**:
  - Form keys are mapped to Ent setters using Go initialisms (`category_id` -> `SetCategoryID`, `image_url` -> `SetImageURL`); such fields were silently skipped before.
  - Empty input clears optional fields on update and is skipped for non-string fields on create.
  - JSON numbers convert without exponent notation; `encoding.TextUnmarshaler` targets such as `uuid.UUID` are supported.
- **Flutter Client Generator (`pkg/flutterclient`)**:
  - The generated client compiles again (duplicate `saveLocale` / `restoreLocale`, missing `shared_preferences` import).
  - On web the API base is derived from the page origin and first path segment, so one build serves several panels and any host.

## [v1.26.0] - 2026-11-09

### Added
- **AI Support Concierge & Multimodal Voice Feature Request System (`pkg/support`, `github.com/golaravel/golaravel/pkg/support`)**:
  - **Dual Engine Tier-1 AI Concierge & Seamless Live Escalation**:
    - AI conversational support agent grounded with knowledge of all framework features up to `v1.25.0`.
    - Auto-escalation trigger detection (e.g. phrases such as "speak to human", "live agent", "human representative", "unresolved") that seamlessly escalates the session to live human operators.
    - Explicit escalation actions (`EscalateToLive`, `EscalateToSuperAdmin`, `ResolveTicket`).
  - **Multimodal Voice Feature Requests in Any Language**:
    - Real-time voice audio recording submission in Sinhala (`si`), Tamil (`ta`), Arabic (`ar`), English (`en`), Spanish (`es`), German (`de`), and more.
    - Multilingual speech processor preserves original raw audio (`AudioData`, `AudioBase64`) for administrative review and audio playback.
    - Automatic speech-to-text transcription and English translation for engineering teams.
    - Automated Knowledge Base checking:
      - If requested feature already exists in the system: AI explains its presence and how to use it in the customer's native language.
      - If it is a new capability: AI evaluates complexity (simple vs complex/under review), replies in the customer's language, creates the ticket, and assigns feasibility ratings (`simple`, `complex`).
  - **Multi-Panel & Multi-Tenant Support Hierarchy**:
    - **Super Admin Panels (`panel.SuperAdminSupport()`, `support.WithScope(support.ScopeSuperAdmin)`)**: Platform-wide visibility across all tenant tickets, central escalation monitoring, and cross-panel support capabilities.
    - **Tenant Admin Panels (`panel.TenantAdminSupport("super_admin")`, `support.WithScope(support.ScopeTenantAdmin)`)**: Scoped to the active tenant, providing customer service downstream while maintaining upstream escalation to platform super admins.
  - **Persistent Support Language Cookie (`golaravel_support_lang`)**:
    - Automatic language resolution checking query (`?lang=`), cookie (`golaravel_support_lang`), or `Accept-Language` headers.
    - Seamless switching via `POST /admin/api/support/language` with 1-year persistence.
  - **Standardized Panel REST API Endpoints**:
    - `GET /admin/api/support/tickets`: List support tickets scoped to tenant or super-admin.
    - `POST /admin/api/support/message`: Send user/agent chat message with auto-escalation detection.
    - `POST /admin/api/support/voice-feature`: Ingest voice feature recordings (JSON base64 or multipart).
    - `GET /admin/api/support/audio/:id`: Stream preserved original audio files.
    - `POST /admin/api/support/escalate`: Escalate ticket to live agent or upstream super-admin panel.
    - `POST /admin/api/support/language`: Set persistent support language cookie.
    - `GET /admin/api/support/knowledge`: Query system capabilities knowledge base.
  - **Administrative Support Resource (`support.NewSupportResource`)**:
    - Pre-configured Filament Resource with inline HTML5 audio player (`RenderVoicePlayerHTML`), language flags, feasibility badges, mode indicators, and quick actions.

### Added
- **Live Chat & AI Support Concierge (`pkg/support`, `support.NewSupportResource`, `support.NewService`)**:
  - **Support Ticket Engine (`pkg/support/ticket.go`)**:
    - Complete `Ticket` domain model with `Status` (`open`, `in_progress`, `resolved`, `closed`), `Priority` (`low`, `normal`, `high`, `urgent`), `Department`, and `UserKey`.
    - Message streams with `SupportMessage` timestamps, sender types (`user`, `agent`, `ai`), and read statuses.
    - Thread-safe storage layer with `MemoryStore` supporting atomic sequences, department queries, and user history lookups.
  - **Dual Mode AI / Live Concierge Service (`pkg/support/service.go`)**:
    - `ModeAI`: Automated Tier-1 AI Concierge answers user questions based on knowledge base policies and FAQ matching.
    - `ModeLive`: Live human agent takeover with immediate socket/SSE notification and mode switching (`service.SwitchMode`).
  - **Administrative Support Resource (`pkg/support/resource.go`)**:
    - Pre-configured Filament Resource with ticket status badges, mode indicators (`AI Chat` vs `Live Chat`), priority styling, and chat timeline inspection.
- **Advanced Search Engine & Refined Filter Facets (`pkg/tables/advanced_search.go`, `res.AdvancedSearchableColumns`)**:
  - Multi-attribute refined facet searching with dynamic count calculations (`[x] In Stock (42)`).
  - Schema API metadata export for client facet rendering.

---

## [v1.24.1] - 2026-11-08

### Fixed
- **App Update Route Registration**:
  - Mounted `/version` and update check endpoints at the top panel level instead of inside the resource iteration loop.

---

## [v1.24.0] - 2026-11-08

### Added
- **Panel Versioning & Hot Update Detection (`panel.Version()`, `panel.BuildID()`, `panel.RequireReloadOnUpdate()`)**:
  - Configurable application version string and deployment build identifier.
  - Automatic `GET /admin/api/version` JSON endpoint returning version and force-reload status.
  - Web and Flutter client update banners prompting users to reload when a newer build is deployed.

---

## [v1.23.0] - 2026-11-07

### Added
- **Customizable Pre-Flutter Loading Screen (`panel.FlutterLoading()`)**:
  - Pure HTML/CSS splash loading integration rendering immediately while Flutter engine hydrates.
  - Configurable loading styles: `LoadingStyleSpinner`, `LoadingStyleBar`, and `LoadingStylePulse`.
  - Configurable branding text, subtitle, background color, primary accent color, and text color.

---

## [v1.22.0] - 2026-11-06

### Added
- **Selectable & Default Column Configuration (`res.SelectableColumns()`, `res.DefaultColumns()`)**:
  - Declarative control over which columns are visible by default and which can be toggled by end users.
  - Full metadata export in `/admin/api/schema` enabling clients to render dynamic column pickers.
  - Locale-aware column header labeling and translations.

---

## [v1.21.0] - 2026-11-05

### Added
- **Customizable Mobile Bottom Navigation (`panel.MobileBottomNavItems()`, `panel.MobileBottomNavShowLabels()`)**:
  - Dynamic bottom navigation bar slots with customizable resource IDs.
  - Label visibility toggle (`MobileBottomNavShowLabels`) for compact or detailed icon navigation.
  - Mobile pagination options: circular Floating Action Button (FAB) modal trigger (`PaginationPositionFab`) vs compact top bar (`PaginationPositionTop`).

---

## [v1.20.0] - 2026-11-03

### Added
- **Authentication Suite & Session Persistence**:
  - Panel authentication endpoints: `/admin/login`, `/admin/logout`, and token validation.
  - Configurable session duration (`SessionDuration`) and remember-me TTL (`RememberMeDuration`).
  - Scaffolding for native Flutter login screens with validation and error toasts.

---

## [v1.19.0] - 2026-11-01

### Added
- **Internationalization (i18n) & RTL Layout Mirroring**:
  - Full Arabic (`ar`), Spanish (`es`), and German (`de`) localization dictionaries.
  - Zero-dependency RTL directionality flipping across HTML/CSS (`dir="rtl"`) and Flutter (`Directionality`).
  - Interactive language switcher buttons and modal sheets.
  - Dynamic translation dictionary API (`GET /admin/api/translations`, `POST /admin/api/locale/switch`).

---

## [v1.18.0] - 2026-10-31

### Added
- **Soft Deletes & Trash Recovery Bin (`res.SoftDeletes()`, `pkg/tables.TrashedFilter`, `pkg/tables.RestoreAction`, `pkg/tables.ForceDeleteAction`)**:
  - **Resource Configuration (`res.SoftDeletes()`, `res.SoftDeleteColumn()`, `res.IsTrashed()`)**:
    - `.SoftDeletes(true)`: Enables soft-delete awareness across the entire resource with automatic trash scoping.
    - Automatic reflection and dynamic discovery of `DeletedAt` / `deleted_at` fields on models and maps.
    - Lifecycle hooks: `OnRestore(fn)`, `OnForceDelete(fn)`, `OnRestoreBulk(fn)`, `OnForceDeleteBulk(fn)`, and authorization checks `CanRestore(fn)`, `CanForceDelete(fn)`.
  - **Automatic Trashed Filter Scoping (`tables.TrashedFilter()`)**:
    - Pre-configured 3-state filter automatically injected into resource table toolbars:
      - `without_trashed` (default): Only active records (`deleted_at == nil`).
      - `with_trashed`: Shows both active and soft-deleted records.
      - `only_trashed`: Isolates soft-deleted items in the Trash Recovery Bin.
  - **Dynamic Action Swapping & Visual Styling (`pkg/tables/table.go`, `pkg/tables/action.go`)**:
    - Trashed records automatically display `[Trashed]` badge and `.table-row-trashed` highlight styling.
    - Row-level actions automatically swap: active records show `Edit` and `Delete`, while trashed records display `Restore` (`⟲`) and `Force Delete` (`🗑`) with confirmation prompts.
    - Bulk operations: Added `tables.RestoreBulkAction()` and `tables.ForceDeleteBulkAction()` with confirmation prompts.
  - **Web Endpoints & JSON REST API (`pkg/filament/resource.go`, `pkg/filament/api.go`)**:
    - Collision-free static endpoints: `GET/POST /admin/{slug}/restore` and `GET/POST /admin/{slug}/force-delete` supporting both form POSTs and query params.
    - REST API endpoints: `POST /admin/api/resources/{slug}/restore` and `POST /admin/api/resources/{slug}/force-delete`.
    - Bulk endpoints: `bulk_action=restore_bulk` and `bulk_action=force_delete_bulk` via `/admin/{slug}/bulk` and API.
  - **Activity Log Audit Trail Integration (`pkg/activitylog/observer.go`)**:
    - When `activitylog.EnableAuditLog` is attached to a soft-deletable resource, restoring automatically records `restored` events, and force-deleting records `deleted` events with `force_deleted: true`.

---

## [v1.17.0] - 2026-10-31

### Added
- **Activity Log & Audit Trail Timeline (`pkg/activitylog`, `gofilament.LogActivity`, `gofilament.EnableAuditLog`, `gofilament.NewActivityResource`)**:
  - **Core Activity & Changeset Diff Engine (`pkg/activitylog/activity.go`)**:
    - Comprehensive `Activity` model capturing `Subject` (entity type & ID), `Causer` (actor ID, name, avatar, type), `Event` (`created`, `updated`, `deleted`, `restored`, `approved`, `rejected`, `login`, custom), `Description`, `OldValues`, `NewValues`, `Properties`, `IPAddress`, `UserAgent`, and timestamps.
    - Field-level `Diff` engine with deterministic sorting, computing changed, added, and removed attributes with human-readable value representations.
    - Helper formatters: `TimeAgo` relative time ("just now", "5 minutes ago", "yesterday"), `EventColor` palettes, and `GetInitials` avatar fallbacks.
  - **Thread-Safe Storage Layer (`pkg/activitylog/store.go`)**:
    - Pluggable `Store` interface supporting `Log`, `GetByID`, `GetBySubject`, `GetByCauser`, `Query` (with multi-criteria filtering, search, and pagination), and `Clear`.
    - High-performance, concurrent-safe `MemoryStore` with atomic ID sequence generation.
    - Global default store registry (`SetDefaultStore`, `GetDefaultStore`).
  - **Fluent Activity Logger (`pkg/activitylog/logger.go`)**:
    - Chainable builder API: `activitylog.Log("description").Subject(...).Causer(...).WithDiff(old, new).Property(...).IP(...).Save()`.
  - **Vertical Audit Timeline & Infolist Component (`pkg/activitylog/timeline.go`)**:
    - Modern vertical audit timeline with continuous spine line, event-specific node icons, and colored badge chips.
    - Causer pill with circular initials badge or avatar image, display name, and causer type.
    - Relative timestamp with RFC3339 tooltip and IP/UserAgent badges.
    - **Interactive Alpine.js Diff Inspector**: Expandable changeset accordion toggle (`▾ Changes (N)`) displaying structured before/after comparison tables with red strikethrough for previous values and green pills for updated values.
    - Live client-side search input and event filter dropdown (`All Events`, `created`, `updated`, etc.) via Alpine.js (`.Filterable(true)`).
    - `infolists.Entry` integration via `activitylog.TimelineEntry("activities")` for seamless embedding inside Resource View and Edit schemas.
  - **Automated Resource CRUD Auditing (`pkg/activitylog/observer.go`)**:
    - `activitylog.EnableAuditLog(resource, store)` (or `gofilament.EnableAuditLog(res)`): Instruments any Filament Resource with zero boilerplate.
    - Intercepts `OnCreate`, `OnUpdate`, and `OnDelete` handlers, automatically extracting authenticated causers from `ctx.User()`, client IP, user agent, and computing field-level diffs against pre-update records.
  - **Pre-Configured Administrative Audit Resource (`pkg/activitylog/resource.go`)**:
    - `activitylog.NewActivityResource()` (or `gofilament.NewActivityResource()`): Ready-to-mount Filament Resource in the "System" navigation group.
    - Features event badge columns, subject/causer columns, description, IP, relative timestamps, event filters, and detailed view timeline.
  - **Table Improvements (`pkg/tables/column.go`)**:
    - Added fluent `.Value(fn)` method on `tables.BadgeColumnDef[T]`.

---

## [v1.16.0] - 2026-10-31

### Added
- **Enhanced Multi-Step Form Wizard (`forms.Wizard` & `forms.WizardStep` in `pkg/forms/advanced_layouts.go`)**:
  - **Client-Side Step Validation Gate**:
    - Automatically checks `checkValidity()` and executes `reportValidity()` on all form inputs (`input`, `select`, `textarea`) in the current step pane before advancing forward.
    - Prevents progression when invalid fields are encountered, displaying native and custom browser validation callouts directly at the offending element.
  - **Skippable & Navigation Controls (`.Skippable(bool)`)**:
    - `.Skippable(true)` allows users to click and jump freely between any steps in any sequence.
    - When `.Skippable(false)` (default), forward jumps are strictly gated behind current step validation or previously visited steps (`maxStep`), while backward navigation is always freely accessible.
  - **Dynamic Progress Bar (`.ProgressBar(bool)`)**:
    - Added high-visibility animated progress bar indicator displaying current completion percentage `((currentStep / totalSteps) * 100)%`.
    - Configurable and toggleable via `.ProgressBar(false)`.
  - **Step Badges & Status Indicators (`.Badge(string)`, `.BadgeColor(string)`)**:
    - Added contextual step badges (e.g. "Step 1", "Optional", "Final") with automatic color schemes (`primary`, `success`, `warning`, `danger`, `info`).
  - **Step Icons & Indicators (`.Icon(string)`)**:
    - Custom icon rendering supporting both raw SVG markup (`<svg...`) and icon classes (`heroicon-o-user`).
    - Completed step checkmarks (`✓`) with smooth color transition upon step completion.
  - **Customizable Actions & Buttons**:
    - `.NextActionLabel(string)` (default: `"Next →"`).
    - `.PreviousActionLabel(string)` (default: `"← Previous"`).
    - `.SubmitAction(string)` (default: `"Submit"`).
    - `.CancelAction(url string, label ...string)` for clean navigation away from the wizard workflow.
    - `.StartStep(int)` to configure custom default active steps.
  - **Responsive Mobile Counter**:
    - Compact step summary bar (`Step X of Y: Step Title`) to ensure seamless usability on mobile and compact screens without horizontal overflow.
  - **Error Auto-Activation**:
    - Automatically analyzes server-side validation error maps on form submission and pre-activates the first step containing invalid fields.

---

## [v1.15.0] - 2026-10-31

### Added
- **Advanced Table Features: Inline Cell Editing, Column Toggling & Drag-and-Drop Row Reordering (`pkg/tables`, `pkg/filament/resource.go`, `pkg/filament/api.go`, `pkg/assets/dist`)**:
  - **Inline Cell Editing (`pkg/tables/column.go`, `pkg/tables/advanced_columns.go`)**:
    - `CellContextRenderer[T]`: Interface enabling columns to access record IDs, base paths, and contextual metadata.
    - `TextInputColumn[T]`: Direct inline editing of text cells with real-time blur/enter synchronization to the backend.
    - `SelectColumn[T]`: Inline dropdown selector updating model values immediately on option change.
    - `ToggleColumn[T]`: Interactive toggle switch button with optimistic visual animation and server-side state persistence.
    - `TextColumn[T].Editable()`: Click-to-edit inline text fields with quick cancel (`Esc`) and save (`Enter`/blur).
    - Web & REST API: `POST /admin/{resource}/cell` and `POST/PATCH /admin/api/resources/{slug}/:id/cell` with custom handler support (`r.OnCellUpdate(func(ctx, id, column, value) error)`).
  - **Dynamic Column Toggling (Show/Hide) (`pkg/tables/table.go`)**:
    - `table.Toggleable(true)`: Adds a sleek "Columns" dropdown menu to table toolbars.
    - Column control methods: `.Toggleable(bool)` and `.ToggledHiddenByDefault(bool)` to configure default visibility.
    - Instant client-side show/hide via Alpine.js `:style="!visibleCols['name'] ? 'display: none;' : ''"` without full-page reloads.
    - Persistent settings: Automatically caches column visibility choices in browser `localStorage` keyed per resource path.
  - **Drag-and-Drop Row Reordering (`pkg/tables/table.go`, `pkg/filament/resource.go`, `pkg/assets/dist`)**:
    - `table.Reorderable("sort_order")`: Adds grip handle headers (`⋮⋮`) and drag handle table cells.
    - Drag & drop DOM interaction with visual drop indicator styling (`.table-row-dragging`).
    - Asynchronous reorder dispatch: Sends new record ID sequences (`POST /admin/{resource}/reorder` and `POST /admin/api/resources/{slug}/reorder`).
    - Custom callback integration: `r.OnReorder(func(ctx *filament.Context, orderedIDs []string) error)`.
  - **Client Runtime & Styling Updates (`pkg/assets/dist/filament.js`, `pkg/assets/dist/filament.css`)**:
    - Added `Filament.updateCell` and `Filament.reorderRows` helper functions with toast notifications and optimistic state indicators.
    - Added CSS for drag-and-drop handles, dragging state elevation, toggle switches, and cell-saved animations.

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
