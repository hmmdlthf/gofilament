import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

void main() {
  testWidgets('rows can be reordered on the plain list', (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    Map<String, dynamic>? sent;
    final panel = FakePanel()
      ..reorderable = true
      ..extraRoutes['POST /resources/items/reorder'] = (r) {
        sent = Map<String, dynamic>.from(json.decode(r.body) as Map);
        return {'success': true};
      };
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);

    await tester.tap(find.byKey(const Key('list-reorder')));
    await tester.pumpAndSettle();
    final first = find.byKey(const Key('reorder-handle-1'));
    await tester.drag(first, const Offset(0, 120));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('reorder-save')));
    await tester.pumpAndSettle();
    expect(sent?['offset'], 0);
    expect((sent?['order'] as List).first, isNot('1'));

    // Not while searching.
    await tester.enterText(find.byKey(const Key('list-search')), 'ring');
    await tester.pump(const Duration(seconds: 1));
    await tester.pumpAndSettle();
    expect(
        tester
            .widget<IconButton>(find.byKey(const Key('list-reorder')))
            .onPressed,
        isNull);
  });
}
