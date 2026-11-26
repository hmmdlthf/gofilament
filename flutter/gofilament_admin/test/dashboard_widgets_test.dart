import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

Map<String, dynamic> _chart(String id, {int span = 1, int sort = 0}) => {
      'id': id,
      'type': 'chart',
      'heading': 'Chart $id',
      'chart_type': 'line',
      'column_span': span,
      'sort': sort,
      'data': {
        'labels': ['Mon', 'Tue'],
        'datasets': [
          {
            'label': 'Sales',
            'data': [1, 2]
          }
        ],
      },
    };

Future<FakePanel> _open(WidgetTester tester, List<Map<String, dynamic>> widgets,
    {Size size = const Size(1400, 2400)}) async {
  final panel = FakePanel()..extraWidgets = widgets;
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
  await tester.pumpAndSettle();
  await signIn(tester);
  return panel;
}

void main() {
  testWidgets('widgets share rows by their span and stack on phones',
      (tester) async {
    await _open(tester, [
      _chart('c', sort: 3),
      _chart('a', sort: 1),
      _chart('b', sort: 2),
      _chart('wide', span: 2, sort: 4),
    ]);
    final a = tester.getRect(find.byKey(const Key('chart-a')));
    final b = tester.getRect(find.byKey(const Key('chart-b')));
    final c = tester.getRect(find.byKey(const Key('chart-c')));
    final wide = tester.getRect(find.byKey(const Key('chart-wide')));
    // a, b, c (sorted) share the first row, left to right, a third each
    expect(a.top, b.top);
    expect(b.top, c.top);
    expect(a.left < b.left && b.left < c.left, isTrue);
    expect((a.width - b.width).abs() < 1, isTrue);
    // the span-2 chart starts the next row, twice as wide
    expect(wide.top > a.bottom, isTrue);
    expect(wide.width > a.width * 1.9, isTrue);
    // the stats row (no span) takes the whole width
    final stat = tester.getRect(find.byKey(const Key('stat-0')));
    expect(stat.top < a.top, isTrue);
  });

  testWidgets('on a phone every widget has its own row', (tester) async {
    await _open(tester, [_chart('a'), _chart('b')],
        size: const Size(420, 2400));
    final a = tester.getRect(find.byKey(const Key('chart-a')));
    final b = tester.getRect(find.byKey(const Key('chart-b')));
    expect(b.top > a.bottom, isTrue);
    expect((a.width - b.width).abs() < 1, isTrue);
  });

  testWidgets('stat cards show the change, the trend and open their link',
      (tester) async {
    final panel = await _open(tester, [
      {
        'id': 'kpi',
        'type': 'stats_overview',
        'column_span': 3,
        'stats': [
          {
            'label': 'Sales',
            'value': '1,200.00',
            'change': 12.5,
            'change_caption': 'vs last week',
            'chart': [1, 3, 2, 5],
            'url': '/app/resources/items?filter_status=live',
          },
          {
            'label': 'Returns',
            'value': '40.00',
            'change': 8,
            'lower_is_better': true,
          },
        ],
      },
    ]);
    expect(find.text('+12.5%'), findsOneWidget);
    expect(find.text('vs last week'), findsOneWidget);
    expect(find.text('+8.0%'), findsOneWidget);
    expect(find.byKey(const Key('stat-sparkline')), findsOneWidget);
    // the lower-is-better rise is drawn as bad news (a different colour)
    final up = tester.widget<Text>(find.text('+12.5%')).style?.color;
    final bad = tester.widget<Text>(find.text('+8.0%')).style?.color;
    expect(up, isNot(bad));

    await tester.tap(find.text('Sales'));
    await tester.pumpAndSettle();
    expect(
        panel.requests.any((r) =>
            r.url.path == '/app/api/resources/items' &&
            r.url.queryParameters['filter_status'] == 'live'),
        isTrue,
        reason: 'the list opens with the link\'s filter');
  });

  testWidgets('tables, breakdowns, progress, quick actions and banners draw',
      (tester) async {
    final panel = await _open(tester, [
      {
        'id': 'top',
        'type': 'table',
        'heading': 'Top products',
        'column_span': 1,
        'columns': [
          {'name': 'name', 'label': 'Product'},
          {'name': 'revenue', 'label': 'Revenue', 'format': 'money'},
          {'name': 'qty', 'label': 'Qty', 'format': 'number'},
        ],
        'rows': [
          {
            'name': 'Rice 5kg',
            'revenue': 125000.5,
            'qty': 40,
            '_url': '/app/resources/items/2'
          },
          {'name': 'Sugar', 'revenue': 9000, 'qty': 12},
        ],
      },
      {
        'id': 'empty',
        'type': 'table',
        'heading': 'Nothing here',
        'empty_text': 'No sales yet',
        'columns': [
          {'name': 'name', 'label': 'Name'}
        ],
        'rows': [],
      },
      {
        'id': 'mix',
        'type': 'breakdown',
        'heading': 'Payment mix',
        'items': [
          {
            'label': 'Cash',
            'value': 75,
            'formatted': '75,000.00',
            'percentage': 75
          },
          {'label': 'Card', 'value': 25, 'percentage': 25, 'color': 'amber'},
        ],
      },
      {
        'id': 'target',
        'type': 'progress',
        'heading': 'Targets',
        'items': [
          {
            'label': 'Monthly sales',
            'current': 600,
            'max': 1000,
            'percentage': 60,
            'unit': 'k'
          },
        ],
      },
      {
        'id': 'todo',
        'type': 'quick_actions',
        'heading': 'Waiting',
        'actions': [
          {
            'label': 'Draft items',
            'icon': 'pencil',
            'url': '/app/resources/items?filter_status=draft',
            'badge': '3'
          },
        ],
      },
      {
        'id': 'warn',
        'type': 'banner',
        'title': 'Journal needs attention',
        'subtitle': '2 entries do not balance',
        'color': 'warning',
        'dismissible': true,
      },
    ]);
    expect(find.text('Rice 5kg'), findsOneWidget);
    expect(find.text('125,000.50'), findsOneWidget);
    expect(find.text('40'), findsOneWidget);
    expect(find.text('No sales yet'), findsOneWidget);
    expect(find.text('Payment mix'), findsOneWidget);
    expect(find.text('75,000.00'), findsOneWidget);
    expect(find.text('25.0%'), findsOneWidget);
    expect(find.text('600 / 1,000 k'), findsOneWidget);
    expect(find.text('3'), findsWidgets);
    expect(find.text('Journal needs attention'), findsOneWidget);

    await tester.tap(find.byKey(const Key('banner-dismiss')));
    await tester.pumpAndSettle();
    expect(find.text('Journal needs attention'), findsNothing);

    await tester.tap(find.byKey(const Key('quick-action-0')));
    await tester.pumpAndSettle();
    expect(
        panel.requests.any((r) =>
            r.url.path == '/app/api/resources/items' &&
            r.url.queryParameters['filter_status'] == 'draft'),
        isTrue);
  });

  testWidgets('a table row with a link opens it', (tester) async {
    final panel = await _open(tester, [
      {
        'id': 'top',
        'type': 'table',
        'heading': 'Top',
        'columns': [
          {'name': 'name', 'label': 'Name'}
        ],
        'rows': [
          {'name': 'Rice', '_url': '/app/resources/items?search=Rice'},
        ],
      },
    ]);
    await tester.tap(find.byKey(const Key('widget-top-row-0')));
    await tester.pumpAndSettle();
    expect(
        panel.requests.any((r) =>
            r.url.path == '/app/api/resources/items' &&
            r.url.queryParameters['search'] == 'Rice'),
        isTrue);
  });

  testWidgets(
      'charts order their periods, format values and show the point under the pointer',
      (tester) async {
    await _open(tester, [
      {
        'id': 'pl',
        'type': 'chart',
        'heading': 'Profit and loss',
        'description': 'Last 3 months',
        'chart_type': 'bar',
        'column_span': 3,
        'value_format': 'money',
        'filters': {'7d': '7 days', '30d': '30 days', '12m': '12 months'},
        'filter_order': ['7d', '30d', '12m'],
        'default_filter': '30d',
        'data': {
          'labels': ['Sep', 'Oct', 'Nov'],
          'datasets': [
            {
              'label': 'Income',
              'data': [1000, 2000, 3000]
            },
            {
              'label': 'Expenses',
              'data': [500, 800, 900]
            },
            {
              'label': 'Net',
              'type': 'line',
              'data': [500, 1200, 2100]
            },
          ],
        },
      },
    ]);
    expect(find.text('Last 3 months'), findsOneWidget);
    final x7 = tester.getTopLeft(find.byKey(const Key('chart-filter-7d'))).dx;
    final x30 = tester.getTopLeft(find.byKey(const Key('chart-filter-30d'))).dx;
    final x12 = tester.getTopLeft(find.byKey(const Key('chart-filter-12m'))).dx;
    expect(x7 < x30 && x30 < x12, isTrue,
        reason: 'periods in filter_order, not sorted by key');

    final canvas = tester.getRect(find.byKey(const Key('chart-canvas-pl')));
    // the last third of the plot is November
    await tester.tapAt(Offset(canvas.right - 20, canvas.center.dy));
    await tester.pumpAndSettle();
    expect(find.text('Nov · Income 3,000.00 · Expenses 900.00 · Net 2,100.00'),
        findsOneWidget);
  });

  testWidgets('a doughnut shows the slice under the pointer with its share',
      (tester) async {
    await _open(tester, [
      {
        'id': 'pay',
        'type': 'chart',
        'heading': 'Payments',
        'chart_type': 'doughnut',
        'column_span': 3,
        'value_format': 'money',
        'data': {
          'labels': ['Cash', 'Card'],
          'datasets': [
            {
              'label': 'Paid',
              'data': [300, 100]
            }
          ],
        },
      },
    ]);
    final canvas = tester.getRect(find.byKey(const Key('chart-canvas-pay')));
    // the first slice (Cash, 75%) runs clockwise from 12 o'clock: tap right of the top
    await tester.tapAt(
        canvas.center + Offset(canvas.height * 0.3, -canvas.height * 0.2));
    await tester.pumpAndSettle();
    expect(find.text('Cash: 300.00 (75.0%)'), findsOneWidget);
  });

  test('links carry filters as filter_x or filters[x]', () {
    expect(
        linkFilters({
          'filter_status': 'paid',
          'filters[brand]': 'acme',
          'search': 'x',
          'page': '2'
        }),
        {'status': 'paid', 'brand': 'acme'});
  });
}
