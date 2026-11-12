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
import '../components/fields/signature_pad_field.dart' as c0;
import '../components/cells/paid_badge_cell.dart' as c1;
import '../components/widgets/sales_total_widget.dart' as c2;
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

/// The app's custom components, built from your files in lib/components:
/// `GoFilamentAdmin(components: panelComponents)`.
const AdminComponents panelComponents = AdminComponents(
  fields: {
    'signature_pad': _fieldSignaturePad,
  },
  cells: {
    'paid_badge': _cellPaidBadge,
  },
  widgets: {
    'sales_total': _widgetSalesTotal,
  },
);

Widget _fieldSignaturePad(BuildContext context, AdminFieldContext field) =>
    c0.SignaturePadField(field: field);

Widget _cellPaidBadge(BuildContext context, AdminCellContext cell) =>
    c1.PaidBadgeCell(cell: cell);

Widget _widgetSalesTotal(BuildContext context, AdminWidgetContext dashboard) =>
    c2.SalesTotalWidget(dashboard: dashboard);
