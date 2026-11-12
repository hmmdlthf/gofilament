# gofilament_admin

Schema-driven Flutter admin client for GoFilament panels running the Flutter engine
(`panel.Engine(filament.EngineFlutter)`). Everything on screen comes from the panel API:
brand and colours, navigation, lists, forms, permissions and dashboard widgets. There is no
per-resource Dart code.

```dart
import 'package:flutter/material.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

void main() => runApp(const GoFilamentAdmin());
```

Scaffold an app that uses it with `flutterclient.GenerateAdminApp`.

## What it renders

| Area | Source in the panel API |
|---|---|
| Login screen brand and colours | `GET /api/boot` (public) |
| Navigation, groups, permissions | `GET /api/schema` (`resources`, `navigation`, `resource_meta`) |
| Lists: search, sort, filters, paging, typed cells, inline toggles, row and bulk delete | `GET /api/resources/{slug}` + column / filter metadata |
| Forms: sections, grids, text, number, select (local, searchable, server-searched), toggle, date, textarea, checkbox list | `GET /api/resources/{slug}/form`, `.../form/options` |
| Relation managers under a record: list, add, edit, delete | `GET /api/resources/{slug}/{id}/relations` and `.../relations/{rel}/form` |
| Export to CSV and Excel, built in the client | list pages, capped by `export.max_rows` |
| Dashboard | `GET /api/widgets` (stats) |

One web build serves every panel and every host: the panel path is read from the page URL.
Column names may be dotted paths (`edges.category.name`) to show fields of eager-loaded related records.

Deep links: `#/resources/<slug>`, `#/resources/<slug>/<id>`, `#/resources/<slug>/new`, `#/pages/<id>`.

## Adding application pages

```dart
GoFilamentAdmin(
  pages: [
    AdminPage(
      id: 'pos',
      label: 'Point of Sale',
      icon: Icons.point_of_sale,
      fullscreen: true, // takes the whole window: no sidebar, no top bar
      builder: (context, admin) => PosScreen(api: admin.api),
    ),
  ],
)
```

`admin.api.get(...)` / `admin.api.send(...)` call custom endpoints of the panel with the signed-in session.

## Generated, typed code

`flutterclient.GenerateFromPanel` (Go) writes `lib/generated/*.g.dart` into an
app: `panelSchema` for `GoFilamentAdmin(initialSchema: ...)`, model classes, a
typed `PanelClient` and `panelPages(...)` for the panel's `filament.FlutterPage`s
(`AdminPage.serverControlled`: shown only when the live schema lists the page).
See the main README, "Typed Dart from the panel".

## Not covered yet

Repeaters, file / image upload, key-value, tabs and wizards as real tabs (shown as consecutive
sections), chart widgets, global search, soft-delete restore, column picker,
non-web file saving for exports.

## Development

```bash
flutter analyze
flutter test
```

Tests run against an in-memory panel (`test/fake_panel.dart`), no server needed.
