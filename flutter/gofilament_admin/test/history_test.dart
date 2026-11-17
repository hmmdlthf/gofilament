import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

Future<void> _openEdit(WidgetTester tester, FakePanel panel) async {
  tester.view.physicalSize = const Size(1400, 1000);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
  await tester.pumpAndSettle();
  await signIn(tester);
  await openItems(tester);
  await tester.tap(find.byKey(const Key('row-edit-1')));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('the record form shows who changed what', (tester) async {
    final panel = FakePanel()
      ..history = true
      ..extraRoutes['GET /resources/items/1/history'] = (_) => {
            'entries': [
              {
                'event': 'updated',
                'causer_name': 'Olivia',
                'at': '2026-11-17T09:30:00Z',
                'changes': [
                  {'field': 'price', 'label': 'Price', 'old': '10', 'new': '12'}
                ],
              },
              {
                'event': 'created',
                'causer_name': '',
                'at': '2026-11-01T08:00:00Z'
              },
            ]
          };
    await _openEdit(tester, panel);
    await tester.tap(find.byKey(const Key('form-history')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('history-entry-0')), findsOneWidget);
    expect(find.textContaining('Olivia', findRichText: true), findsOneWidget);
    expect(
        find.textContaining('Price: 10', findRichText: true), findsOneWidget);
    expect(find.textContaining('System', findRichText: true), findsOneWidget);
    expect(find.textContaining('created this record', findRichText: true),
        findsOneWidget);
  });

  testWidgets('no history button without the ability', (tester) async {
    await _openEdit(tester, FakePanel());
    expect(find.byKey(const Key('form-history')), findsNothing);
  });
}
