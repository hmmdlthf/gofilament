import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'admin_test.dart' show pumpApp, signIn, openItems;
import 'fake_panel.dart';

void main() {
  testWidgets('relation managers show a table on wide screens',
      (tester) async {
    await pumpApp(tester, FakePanel());
    await signIn(tester);
    await openItems(tester);
    await tester.tap(find.byKey(const Key('row-edit-1')));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('rel-variants-table')), findsOneWidget);
    expect(find.byKey(const Key('rel-variants-cards')), findsNothing);
  });

  testWidgets('relation managers switch to cards on a phone, like the list',
      (tester) async {
    await pumpApp(tester, FakePanel(), size: const Size(420, 900));
    await signIn(tester);
    await tester.tap(find.byTooltip('Menu'));
    await tester.pumpAndSettle();
    await openItems(tester);
    await tester.tap(find.text('Rings'));
    await tester.pumpAndSettle();

    final cards = find.byKey(const Key('rel-variants-cards'));
    await tester.scrollUntilVisible(cards, 200,
        scrollable: find.byType(Scrollable).last);
    expect(cards, findsOneWidget);
    expect(find.byKey(const Key('rel-variants-table')), findsNothing);
    // a card per record: the first column as its title, the others labelled
    expect(find.byKey(const Key('rel-variants-card-10')), findsOneWidget);
    expect(find.text('RING-S'), findsOneWidget);
    expect(find.byKey(const Key('rel-variants-edit-10')), findsOneWidget);
    expect(tester.takeException(), isNull); // no overflow
  });
}
