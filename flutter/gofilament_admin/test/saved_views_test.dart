import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';
import 'package:gofilament_admin/src/list_tools.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

Map<String, String> _lastListQuery(FakePanel panel) => panel.requests
    .lastWhere(
        (r) => r.url.path.endsWith('/resources/items') && r.method == 'GET')
    .url
    .queryParameters;

void main() {
  testWidgets('saved views apply, save and delete list states', (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final views = <Map<String, dynamic>>[
      {
        'id': 'v1',
        'name': 'Drafts by name',
        'shared': true,
        'mine': false,
        'can_delete': false,
        'state': {
          'search': 'ch',
          'sort': 'name',
          'direction': 'desc',
          'filters': {'status': 'draft'},
          'per_page': 10,
        },
      },
    ];
    final saved = <Map<String, dynamic>>[];
    final panel = FakePanel()
      ..savedViews = true
      ..extraRoutes.addAll({
        'GET /resources/items/views': (_) =>
            {'views': views, 'can_share': true},
        'POST /resources/items/views': (r) {
          final body = Map<String, dynamic>.from(json.decode(r.body) as Map);
          saved.add(body);
          final v = {
            'id': 'v2',
            'name': body['name'],
            'shared': body['shared'],
            'mine': true,
            'can_delete': true,
            'state': body['state'],
          };
          views.add(v);
          return v;
        },
        'DELETE /resources/items/views/v2': (_) {
          views.removeWhere((v) => v['id'] == 'v2');
          return {'success': true};
        },
      });
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);

    // Apply the shared view.
    await tester.tap(find.byKey(const Key('list-views')));
    await tester.pumpAndSettle();
    expect(find.text('Drafts by name (shared)'), findsOneWidget);
    expect(find.byKey(const Key('view-delete')), findsNothing);
    await tester.tap(find.byKey(const Key('view-v1')));
    await tester.pumpAndSettle();
    final q = _lastListQuery(panel);
    expect(q['search'], 'ch');
    expect(q['sort'], 'name');
    expect(q['direction'], 'desc');
    expect(q['filter_status'], 'draft');
    expect(q['per_page'], '10');
    expect(find.byKey(const Key('view-current')), findsOneWidget);

    // Save it under a new name, shared.
    await tester.tap(find.byKey(const Key('list-views')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('view-save')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('view-name')), 'Mine');
    await tester.tap(find.byKey(const Key('view-shared')));
    await tester.tap(find.byKey(const Key('view-save-ok')));
    await tester.pumpAndSettle();
    expect(saved.single['name'], 'Mine');
    expect(saved.single['shared'], true);
    expect(saved.single['state']['filters'], {'status': 'draft'});
    expect(find.text('Mine'), findsOneWidget); // current view label

    // Delete it, then back to all records.
    await tester.tap(find.byKey(const Key('list-views')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('view-delete')));
    await tester.pumpAndSettle();
    expect(views.length, 1);
    await tester.tap(find.byKey(const Key('list-views')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('view-all')));
    await tester.pumpAndSettle();
    expect(_lastListQuery(panel).containsKey('filter_status'), isFalse);
    expect(_lastListQuery(panel).containsKey('search'), isFalse);
  });

  testWidgets('no views menu when the panel has none', (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final panel = FakePanel();
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);
    expect(find.byKey(const Key('list-views')), findsNothing);
  });

  test('AdvancedSearch.decode reads encode back', () {
    final s = AdvancedSearch(conjunction: 'OR', rules: [
      AdvancedRule('name', 'contains', 'ring'),
      AdvancedRule('price', 'between', '10, 20'),
    ]);
    final back = AdvancedSearch.decode(s.encode());
    expect(back.encode(), s.encode());
    expect(AdvancedSearch.decode('nope').isEmpty, isTrue);
  });
}
