import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

void main() {
  testWidgets('global search opens the chosen record', (tester) async {
    final panel = FakePanel()
      ..globalSearch = true
      ..extraRoutes['GET /global-search'] = (req) => {
            'query': req.url.queryParameters['query'],
            'groups': [
              {
                'name': 'Items',
                'count': 1,
                'items': [
                  {
                    'title': 'Chains',
                    'url': '/app/items/2',
                    'details': {'Code': 'CHAIN'},
                  }
                ],
              }
            ],
          };
    tester.view.physicalSize = const Size(1400, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);

    await tester.tap(find.byKey(const Key('global-search')));
    await tester.pumpAndSettle();
    await tester.enterText(
        find.byKey(const Key('global-search-input')), 'chain');
    await tester.pump(const Duration(milliseconds: 300));
    await tester.pumpAndSettle();
    expect(find.text('Code: CHAIN'), findsOneWidget);

    await tester.tap(find.byKey(const Key('search-result-/app/items/2')));
    await tester.pumpAndSettle();
    // the record's form opens over the items list
    expect(find.byKey(const Key('form-close')), findsOneWidget);
    expect(
        panel.requests.any((r) =>
            r.url.path == '/app/api/resources/items/form' &&
            r.url.queryParameters['id'] == '2'),
        isTrue);
  });

  testWidgets('no search button when the panel has no global search',
      (tester) async {
    final panel = FakePanel();
    tester.view.physicalSize = const Size(1400, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    expect(find.byKey(const Key('global-search')), findsNothing);
  });
}
