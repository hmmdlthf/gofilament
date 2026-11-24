import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'admin_test.dart' show pumpApp, signIn, openItems;
import 'fake_panel.dart';

Future<void> _newItem(WidgetTester tester, FakePanel panel) async {
  await pumpApp(tester, panel);
  await signIn(tester);
  await openItems(tester);
  await tester.tap(find.byKey(const Key('list-create')));
  await tester.pumpAndSettle();
  await tester.enterText(find.byKey(const Key('field-code')), 'BANGLE');
  await tester.enterText(find.byKey(const Key('field-name')), 'Bangles');
  await tester.enterText(find.byKey(const Key('field-price')), '10');
}

void main() {
  testWidgets('a missing option is created from the select and selected',
      (tester) async {
    final panel = FakePanel()..creatableCategory = true;
    await _newItem(tester, panel);

    await tester.tap(find.byKey(const Key('field-category_id')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('picker-search')), 'Bracelets');
    await tester.pumpAndSettle();
    expect(find.text('Create "Bracelets"'), findsOneWidget);
    await tester.tap(find.byKey(const Key('option-create')));
    await tester.pumpAndSettle();

    // the create form opens with the typed name filled in
    expect(find.text('Create category'), findsOneWidget);
    final name = tester.widget<TextField>(find.descendant(
        of: find.byKey(const Key('field-name')).last,
        matching: find.byType(TextField)));
    expect(name.controller!.text, 'Bracelets');
    await tester.tap(find.byKey(const Key('form-save')).last);
    await tester.pumpAndSettle();

    // back on the item form, with the new category chosen
    expect(panel.createdCategories, ['Bracelets']);
    expect(find.text('Bracelets'), findsOneWidget);
    await tester.tap(find.byKey(const Key('form-save')));
    await tester.pumpAndSettle();
    expect(panel.items.last['category_id'], '101');
    expect(panel.items.last['name'], 'Bangles'); // nothing typed was lost
  });

  testWidgets('the create form shows validation errors and can be cancelled',
      (tester) async {
    final panel = FakePanel()..creatableCategory = true;
    await _newItem(tester, panel);
    await tester.tap(find.byKey(const Key('field-category_id')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('option-create')));
    await tester.pumpAndSettle();
    expect(find.text('Create category'), findsWidgets);

    await tester.tap(find.byKey(const Key('form-close')).last);
    await tester.pumpAndSettle();
    expect(panel.createdCategories, isEmpty);
  });

  testWidgets('no create entry when the field cannot create options',
      (tester) async {
    await _newItem(tester, FakePanel());
    await tester.tap(find.byKey(const Key('field-category_id')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('option-create')), findsNothing);
  });
}
