import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

void main() {
  testWidgets('the column picker shows and hides toggleable columns',
      (tester) async {
    final panel = FakePanel()
      ..extraColumns = [
        {
          'name': 'status',
          'label': 'Stage',
          'type': 'text',
          'toggleable': true,
        },
        {
          'name': 'cost_price',
          'label': 'Cost',
          'type': 'text',
          'toggleable': true,
          'toggled_hidden_by_default': true,
        },
      ];
    tester.view.physicalSize = const Size(1400, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);

    expect(find.text('STAGE'), findsOneWidget);
    expect(find.text('COST'), findsNothing);

    await tester.tap(find.byKey(const Key('list-columns')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('column-toggle-cost_price')));
    await tester.pumpAndSettle();
    expect(find.text('COST'), findsOneWidget);

    await tester.tap(find.byKey(const Key('list-columns')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('column-toggle-status')));
    await tester.pumpAndSettle();
    expect(find.text('STAGE'), findsNothing);
    expect(find.text('COST'), findsOneWidget);
  });
}
