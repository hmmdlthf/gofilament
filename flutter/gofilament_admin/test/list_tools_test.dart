import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';
import 'package:gofilament_admin/src/list_tools.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

Future<void> _open(WidgetTester tester, FakePanel panel) async {
  tester.view.physicalSize = const Size(1400, 900);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
  await tester.pumpAndSettle();
  await signIn(tester);
  await openItems(tester);
}

Map<String, String> _lastListQuery(FakePanel panel) => panel.requests
    .lastWhere(
        (r) => r.method == 'GET' && r.url.path.endsWith('/resources/items'))
    .url
    .queryParameters;

void main() {
  testWidgets('a refined filter sends the checked values', (tester) async {
    final panel = FakePanel()
      ..extraFilters = [
        {
          'name': 'city',
          'label': 'City',
          'type': 'refined',
          'column': 'city',
          'multiple': true,
          'options': [
            {'value': 'colombo', 'label': 'Colombo', 'count': 4},
            {'value': 'kandy', 'label': 'Kandy', 'count': 2},
            {'value': 'galle', 'label': 'Galle'},
          ],
        }
      ];
    await _open(tester, panel);

    await tester.tap(find.byKey(const Key('filter-city')));
    await tester.pumpAndSettle();
    expect(find.text('4'), findsOneWidget); // live count
    await tester.tap(find.byKey(const Key('refined-option-colombo')));
    await tester.tap(find.byKey(const Key('refined-option-kandy')));
    await tester.tap(find.byKey(const Key('refined-apply')));
    await tester.pumpAndSettle();
    expect(_lastListQuery(panel)['filter_city'], 'colombo,kandy');
    expect(find.text('City: Colombo, Kandy'), findsOneWidget);

    await tester.tap(find.byKey(const Key('filter-city')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('refined-clear')));
    await tester.pumpAndSettle();
    expect(_lastListQuery(panel).containsKey('filter_city'), isFalse);
  });

  testWidgets('advanced search builds rules and sends them as JSON',
      (tester) async {
    final panel = FakePanel()
      ..itemsResourceExtras = {
        'advanced_search_enabled': true,
        'advanced_search_columns': ['name', 'status'],
      };
    await _open(tester, panel);

    await tester.tap(find.byKey(const Key('list-advanced-search')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('adv-value-0')), 'lamp');
    await tester.tap(find.byKey(const Key('adv-add')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('adv-op-1')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('is one of').last);
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('adv-value-1')), 'live, draft');
    await tester.tap(find.byKey(const Key('adv-apply')));
    await tester.pumpAndSettle();

    final sent = json.decode(_lastListQuery(panel)['advanced_search']!) as Map;
    expect(sent['conjunction'], 'AND');
    final rules = sent['rules'] as List;
    expect(
        rules[0], {'field': 'name', 'operator': 'contains', 'value': 'lamp'});
    expect(rules[1]['operator'], 'in');
    expect(rules[1]['values'], ['live', 'draft']);

    await tester.tap(find.byKey(const Key('list-advanced-search')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('adv-clear')));
    await tester.pumpAndSettle();
    expect(_lastListQuery(panel).containsKey('advanced_search'), isFalse);
  });

  testWidgets('no advanced search button unless the resource enables it',
      (tester) async {
    await _open(tester, FakePanel());
    expect(find.byKey(const Key('list-advanced-search')), findsNothing);
  });

  testWidgets('dragging a header resizes the column and is remembered',
      (tester) async {
    final panel = FakePanel();
    final api = panel.api();
    tester.view.physicalSize = const Size(1400, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: api));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);

    final handle = find.byKey(const Key('column-resize-name'));
    expect(handle, findsOneWidget);
    final before = tester.getTopLeft(handle).dx;
    await tester.drag(handle, const Offset(120, 0));
    await tester.pumpAndSettle();
    expect(tester.getTopLeft(handle).dx, greaterThan(before + 60));

    final saved = await api.readPreference('widths:items');
    expect(saved, isNotNull);
    expect((json.decode(saved!) as Map)['name'], isA<num>());

    // Double tap goes back to automatic width.
    await tester.tap(handle);
    await tester.pump(const Duration(milliseconds: 50));
    await tester.tap(handle);
    await tester.pumpAndSettle();
    expect(
        (json.decode((await api.readPreference('widths:items'))!) as Map)
            .containsKey('name'),
        isFalse);
  });

  testWidgets('a save that hits a newer change offers reload or overwrite',
      (tester) async {
    final panel = FakePanel()..itemVersion = 1;
    await _open(tester, panel);
    await tester.tap(find.byKey(const Key('row-edit-1')));
    await tester.pumpAndSettle();
    panel.itemVersion = 2; // someone else saved meanwhile

    await tester.tap(find.byKey(const Key('form-save')));
    await tester.pumpAndSettle();
    expect(panel.putBodies.last['_version'], '1');
    expect(find.textContaining('changed by someone else'), findsOneWidget);
    expect(find.byKey(const Key('conflict-reload')), findsOneWidget);

    await tester.tap(find.byKey(const Key('conflict-overwrite')));
    await tester.pumpAndSettle();
    expect(panel.putBodies.last['_version'], '2');
    expect(panel.itemVersion, 3);
    expect(find.byKey(const Key('form-save')), findsNothing); // closed
  });

  testWidgets('grouping adds group headers, subtotals and totals',
      (tester) async {
    final panel = FakePanel()
      ..extraColumns = [
        {
          'name': 'id',
          'label': 'Ref',
          'type': 'text',
          'summaries': ['sum', 'count']
        },
      ]
      ..itemsResourceExtras = {
        'groups': ['status']
      }
      ..listSummaries = {
        'id': {'sum': 99, 'count': 40}
      };
    await _open(tester, panel);

    // Summary rows without grouping.
    expect(find.text('This page'), findsOneWidget);
    expect(find.text('All results'), findsOneWidget);
    expect(find.text('Sum 99', findRichText: true), findsOneWidget);
    expect(find.text('Sum 3', findRichText: true),
        findsOneWidget); // page 1: ids 1 + 2

    await tester.tap(find.byKey(const Key('list-group')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('group-status')));
    await tester.pumpAndSettle();
    expect(_lastListQuery(panel)['sort'], 'status');
    // Page 1 sorted by status: Chains (draft), Rings (live).
    expect(find.textContaining('draft (1)'), findsOneWidget);
    expect(find.textContaining('live (1)'), findsOneWidget);
    expect(find.text('Subtotal'), findsNWidgets(2));

    await tester.tap(find.byKey(const Key('list-group')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('group-none')));
    await tester.pumpAndSettle();
    expect(find.text('Subtotal'), findsNothing);
  });

  test('computeSummaries matches the server', () {
    final s = computeSummaries([10, '20.5', null, '', 'x', 4.5],
        ['sum', 'avg', 'count', 'min', 'max']);
    expect(s['sum'], 35);
    expect(s['count'], 4);
    expect(formatSummary(s['avg']!), '11.67');
    expect(s['min'], 4.5);
    expect(s['max'], 20.5);
  });

  testWidgets('a number range filter sends "min,max"', (tester) async {
    final panel = FakePanel()
      ..extraFilters = [
        {
          'name': 'price',
          'label': 'Price',
          'type': 'number_range',
          'column': 'price'
        }
      ];
    await _open(tester, panel);
    await tester.tap(find.byKey(const Key('filter-price')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('range-from')), '10');
    await tester.tap(find.byKey(const Key('range-apply')));
    await tester.pumpAndSettle();
    expect(_lastListQuery(panel)['filter_price'], '10,');
    expect(find.text('Price: ≥ 10'), findsOneWidget);

    await tester.tap(find.byKey(const Key('filter-price')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('range-to')), '99.5');
    await tester.tap(find.byKey(const Key('range-apply')));
    await tester.pumpAndSettle();
    expect(_lastListQuery(panel)['filter_price'], '10,99.5');

    await tester.tap(find.byKey(const Key('filter-price')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('range-clear')));
    await tester.pumpAndSettle();
    expect(_lastListQuery(panel).containsKey('filter_price'), isFalse);
  });

  testWidgets('a date range filter picks dates from the calendar',
      (tester) async {
    final panel = FakePanel()
      ..extraFilters = [
        {
          'name': 'created_at',
          'label': 'Created',
          'type': 'date_range',
          'column': 'created_at'
        }
      ];
    await _open(tester, panel);
    await tester.tap(find.byKey(const Key('filter-created_at')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('range-from')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('15'));
    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('range-apply')));
    await tester.pumpAndSettle();
    final v = _lastListQuery(panel)['filter_created_at']!;
    expect(RegExp(r'^\d{4}-\d{2}-15,$').hasMatch(v), isTrue, reason: v);
  });
}
