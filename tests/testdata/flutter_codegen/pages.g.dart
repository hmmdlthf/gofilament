// GENERATED CODE - DO NOT EDIT.
//
// Flutter pages of the 'admin' panel (/admin), written by gofilament's
// flutterclient.GenerateFromPanel. Change the panel in Go and regenerate;
// keep hand-written code in other files.

// ignore_for_file: type=lint

import 'package:flutter/material.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

/// Draws one page.
typedef PanelPageBuilder = Widget Function(
    BuildContext context, AdminController admin);

/// The builders of the pages the panel declares with filament.FlutterPage.
/// Write them in your own files; adding a page in Go adds a required
/// parameter here, so the app does not build until the page exists.
class PanelPageBuilders {
  const PanelPageBuilders({
    required this.pos,
    required this.stockTake,
  });

  /// Point of sale (`pos`).
  final PanelPageBuilder pos;

  /// Stock take (`stock-take`).
  final PanelPageBuilder stockTake;
}

/// The panel's Flutter pages for `GoFilamentAdmin(pages: ...)`. The server decides
/// who sees each one: a page is shown only when the live schema lists it.
List<AdminPage> panelPages(PanelPageBuilders builders) => <AdminPage>[
      AdminPage(
        id: 'pos',
        label: 'Point of sale',
        icon: adminIcon('heroicon-o-shopping-cart', fallback: Icons.widgets_outlined),
        group: '',
        fullscreen: true,
        serverControlled: true,
        builder: builders.pos,
      ),
      AdminPage(
        id: 'stock-take',
        label: 'Stock take',
        icon: adminIcon('', fallback: Icons.widgets_outlined),
        group: 'Sales',
        fullscreen: false,
        serverControlled: true,
        builder: builders.stockTake,
      ),
    ];
