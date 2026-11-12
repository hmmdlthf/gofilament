import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

void main() {
  testWidgets('trashed rows offer restore and delete for good', (tester) async {
    final panel = FakePanel();
    panel.extraRowMeta['1'] = {
      'trashed': true,
      'update': false,
      'delete': false,
      'actions': <String>[],
      'restore': true,
      'force_delete': true,
    };
    panel.extraRoutes['POST /resources/items/restore'] =
        (req) => {'success': true, 'id': (json.decode(req.body) as Map)['id']};
    panel.extraRoutes['POST /resources/items/force-delete'] =
        (req) => {'success': true};
    tester.view.physicalSize = const Size(1400, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);

    expect(find.byKey(const Key('row-edit-1')), findsNothing);
    expect(find.byKey(const Key('row-delete-1')), findsNothing);
    expect(find.byKey(const Key('row-restore-2')), findsNothing);

    await tester.tap(find.byKey(const Key('row-restore-1')));
    await tester.pumpAndSettle();
    expect(find.text('Restored'), findsOneWidget);
    final restore =
        panel.requests.lastWhere((r) => r.url.path.endsWith('/restore'));
    expect(json.decode(restore.body), {'id': '1'});

    await tester.tap(find.byKey(const Key('row-force-delete-1')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('confirm-delete')));
    await tester.pumpAndSettle();
    expect(panel.requests.any((r) => r.url.path.endsWith('/force-delete')),
        isTrue);
  });
}
