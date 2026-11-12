import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

Future<FakePanel> openCreate(WidgetTester tester) async {
  final panel = FakePanel()
    ..extraFormFields = [
      {
        'kind': 'tabs',
        'children': [
          {
            'kind': 'tab',
            'label': 'Shipping',
            'children': [
              {
                'kind': 'field',
                'type': 'text',
                'name': 'ship_to',
                'label': 'Ship to',
                'required': true
              },
            ],
          },
          {
            'kind': 'tab',
            'label': 'Notes',
            'children': [
              {
                'kind': 'field',
                'type': 'text',
                'name': 'note',
                'label': 'Note'
              },
            ],
          },
        ],
      },
      {
        'kind': 'repeater',
        'name': 'lines',
        'label': 'Lines',
        'min_items': 1,
        'children': [
          {
            'kind': 'field',
            'type': 'text',
            'name': 'sku',
            'label': 'SKU',
            'required': true
          },
          {'kind': 'field', 'type': 'number', 'name': 'qty', 'label': 'Qty'},
        ],
      },
      {
        'kind': 'field',
        'type': 'key_value',
        'name': 'attrs',
        'label': 'Attributes'
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
  return panel;
}

Future<void> tapVisible(WidgetTester tester, Finder f) async {
  await tester.ensureVisible(f);
  await tester.pumpAndSettle();
  await tester.tap(f);
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('tabs, repeater and key-value fields are edited and saved',
      (tester) async {
    final panel = await openCreate(tester);
    await tester.enterText(find.byKey(const Key('field-code')), 'BOX');
    await tester.enterText(find.byKey(const Key('field-name')), 'Box');

    // fields of hidden tabs still validate
    await tapVisible(tester, find.byKey(const Key('step-1')));
    await tapVisible(tester, find.byKey(const Key('form-save')));
    expect(find.text('Lines needs at least 1 items'), findsOneWidget);
    expect(panel.items.length, 3);

    await tapVisible(tester, find.byKey(const Key('step-0')));
    await tester.enterText(find.byKey(const Key('field-ship_to')), 'Kandy');
    await tapVisible(tester, find.byKey(const Key('step-1')));
    await tester.enterText(find.byKey(const Key('field-note')), 'Fragile');

    await tapVisible(tester, find.byKey(const Key('lines-add')));
    await tapVisible(tester, find.byKey(const Key('lines-add')));
    await tester.enterText(
        find.descendant(
            of: find.byKey(const Key('lines.0.sku')),
            matching: find.byType(EditableText)),
        'A-1');
    await tester.enterText(
        find.descendant(
            of: find.byKey(const Key('lines.0.qty')),
            matching: find.byType(EditableText)),
        '3');
    await tapVisible(tester, find.byKey(const Key('lines-remove-1')));

    await tapVisible(tester, find.byKey(const Key('attrs-add')));
    await tester.enterText(find.byKey(const Key('attrs-key-0')), 'colour');
    await tester.enterText(find.byKey(const Key('attrs-value-0')), 'red');

    await tapVisible(tester, find.byKey(const Key('form-save')));
    final saved = panel.items.last;
    expect(saved['ship_to'], 'Kandy');
    expect(saved['note'], 'Fragile');
    expect(saved['lines'], [
      {'sku': 'A-1', 'qty': 3}
    ]);
    expect(saved['attrs'], {'colour': 'red'});
  });

  testWidgets('a repeater item with an empty required field is refused',
      (tester) async {
    final panel = await openCreate(tester);
    await tester.enterText(find.byKey(const Key('field-code')), 'BOX');
    await tester.enterText(find.byKey(const Key('field-name')), 'Box');
    await tester.enterText(find.byKey(const Key('field-ship_to')), 'Kandy');
    await tapVisible(tester, find.byKey(const Key('lines-add')));
    await tapVisible(tester, find.byKey(const Key('form-save')));
    expect(find.text('SKU is required'), findsOneWidget);
    expect(panel.items.length, 3);
  });
}
