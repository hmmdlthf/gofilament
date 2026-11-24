import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';
import 'form_editors_test.dart' show tapVisible;

void main() {
  testWidgets('visible_when shows a section only for its value and drops hidden values',
      (tester) async {
    final panel = FakePanel()
      ..extraFormFields = [
        {'kind': 'field', 'type': 'toggle', 'name': 'gift', 'label': 'Gift'},
        {
          'kind': 'section',
          'heading': 'Gift wrapping',
          'visible_when': {
            'field': 'gift',
            'values': ['true']
          },
          'children': [
            {
              'kind': 'field',
              'type': 'text',
              'name': 'gift_note',
              'label': 'Gift note',
              'required': true
            },
          ],
        },
      ];
    tester.view.physicalSize = const Size(1400, 2400);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);
    await tester.tap(find.byKey(const Key('list-create')));
    await tester.pumpAndSettle();

    expect(find.text('Gift wrapping'), findsNothing);
    await tester.enterText(find.byKey(const Key('field-code')), 'BOX');
    await tester.enterText(find.byKey(const Key('field-name')), 'Box');

    await tapVisible(tester, find.byKey(const Key('field-gift')));
    expect(find.text('Gift wrapping'), findsOneWidget);
    await tester.enterText(find.byKey(const Key('field-gift_note')), 'Happy day');

    // switched off again: the hidden required field neither blocks nor is sent
    await tapVisible(tester, find.byKey(const Key('field-gift')));
    expect(find.text('Gift wrapping'), findsNothing);
    await tapVisible(tester, find.byKey(const Key('form-save')));
    final saved = panel.items.last;
    expect(saved['code'], 'BOX');
    expect(saved.containsKey('gift_note'), isFalse);
  });
}
