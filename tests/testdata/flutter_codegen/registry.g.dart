// GENERATED CODE - DO NOT EDIT.
//
// Page registry of the 'admin' panel (/admin), written by gofilament's
// flutterclient.GenerateFromPanel. Change the panel in Go and regenerate;
// keep hand-written code in other files.

// ignore_for_file: type=lint

import 'package:flutter/widgets.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import '../pages/pos_page.dart';
import '../pages/stock_take_page.dart';
import 'pages.g.dart';

/// The panel's Flutter pages, built from your files in lib/pages:
/// `GoFilamentAdmin(pages: panelAppPages)`.
final List<AdminPage> panelAppPages = panelPages(const PanelPageBuilders(
  pos: _pos,
  stockTake: _stockTake,
));

Widget _pos(BuildContext context, AdminController admin) =>
    PosPage(admin: admin);

Widget _stockTake(BuildContext context, AdminController admin) =>
    StockTakePage(admin: admin);
