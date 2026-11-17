import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

void main() {
  testWidgets('bulk edit sets a field on the selected rows', (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final sent = <Map<String, dynamic>>[];
    final panel = FakePanel()
      ..bulkEdit = ['category_id']
      ..extraRoutes['POST /resources/items/bulk-update'] = (r) {
        sent.add(Map<String, dynamic>.from(json.decode(r.body) as Map));
        return {
          'updated': 1,
          'failed': 1,
          'errors': {'2': 'Forbidden'}
        };
      };
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);
    expect(find.byKey(const Key('bulk-edit')), findsNothing);
    await tester.tap(find.byType(Checkbox).first);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('bulk-edit')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('bulk-value')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Jewellery').last);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('bulk-apply')));
    await tester.pumpAndSettle();
    expect(sent.single['field'], 'category_id');
    expect(sent.single['value'], '2');
    expect((sent.single['ids'] as List).length, 2);
    expect(find.text('1 changed, 1 not changed.'), findsOneWidget);
    expect(
        find.byKey(const Key('bulk-edit')), findsNothing); // selection cleared
  });
}
