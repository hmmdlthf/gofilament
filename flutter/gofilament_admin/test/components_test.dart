import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

/// A toy "rating" field: taps set 1..3.
Widget ratingField(BuildContext context, AdminFieldContext field) => Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('${field.label} (max ${field.props['max']})'),
        Row(children: [
          for (var i = 1; i <= 3; i++)
            IconButton(
              key: Key('star-$i'),
              onPressed: field.enabled ? () => field.onChanged(i) : null,
              icon: Icon((field.value is int && field.value as int >= i)
                  ? Icons.star
                  : Icons.star_border),
            ),
        ]),
        if (field.errorText != null) Text(field.errorText!),
      ],
    );

Future<FakePanel> pump(WidgetTester tester,
    {AdminComponents components = const AdminComponents()}) async {
  final panel = FakePanel()
    ..extraFormFields = [
      {
        'kind': 'field',
        'type': 'custom',
        'component': 'rating',
        'name': 'rating',
        'label': 'Rating',
        'required': true,
        'props': {'max': 3},
      }
    ]
    ..extraColumns = [
      {
        'name': 'code',
        'label': 'Badge',
        'type': 'custom',
        'component': 'code_badge',
        'props': {'prefix': '#'},
      }
    ]
    ..extraWidgets = [
      {
        'id': 'sales',
        'type': 'custom',
        'component': 'sales_total',
        'heading': 'Sales',
        'data': {'total': 1234},
      }
    ];
  tester.view.physicalSize = const Size(1400, 900);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester
      .pumpWidget(GoFilamentAdmin(api: panel.api(), components: components));
  await tester.pumpAndSettle();
  await signIn(tester);
  return panel;
}

const components = AdminComponents(
  fields: {'rating': ratingField},
  cells: {'code_badge': _badge},
  widgets: {'sales_total': _sales},
);

Widget _badge(BuildContext context, AdminCellContext cell) =>
    Text('${cell.props['prefix']}${cell.value}', key: const Key('badge'));

Widget _sales(BuildContext context, AdminWidgetContext w) =>
    Text('Total ${(w.data as Map)['total']}');

void main() {
  testWidgets('custom widget and cell components render server data',
      (tester) async {
    await pump(tester, components: components);
    expect(find.text('Sales'), findsOneWidget);
    expect(find.text('Total 1234'), findsOneWidget);

    await openItems(tester);
    expect(find.text('#RING'), findsWidgets);
  });

  testWidgets('a custom field sets the value that is saved, and is validated',
      (tester) async {
    final panel = await pump(tester, components: components);
    await openItems(tester);
    await tester.tap(find.byKey(const Key('list-create')));
    await tester.pumpAndSettle();
    expect(find.text('Rating (max 3)'), findsOneWidget);

    await tester.enterText(find.byKey(const Key('field-code')), 'NEW');
    await tester.enterText(find.byKey(const Key('field-name')), 'New item');
    await tester.tap(find.byKey(const Key('form-save')));
    await tester.pumpAndSettle();
    expect(find.text('Rating is required'), findsOneWidget);
    expect(panel.items.length, 3);

    await tester.ensureVisible(find.byKey(const Key('star-2')));
    await tester.tap(find.byKey(const Key('star-2')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('form-save')));
    await tester.pumpAndSettle();
    expect(panel.items.last['rating'], 2);
    expect(panel.items.last['name'], 'New item');
  });

  testWidgets('missing components fall back without breaking the page',
      (tester) async {
    await pump(tester);
    expect(
        find.text('This widget is not available in this app.'), findsOneWidget);
    await openItems(tester);
    expect(find.text('RING'), findsWidgets); // plain value
    await tester.tap(find.byKey(const Key('list-create')));
    await tester.pumpAndSettle();
    expect(
        find.text('This field is not available in this app.'), findsOneWidget);
  });
}
