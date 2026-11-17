import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

void main() {
  testWidgets('the bell shows unread notifications and marks them read',
      (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final items = [
      {
        'id': 'n1',
        'type': 'success',
        'title': 'Export of Items',
        'body': 'It is ready.',
        'read': false
      },
      {'id': 'n2', 'type': 'info', 'title': 'Welcome', 'read': true},
    ];
    final readIds = <String>[];
    var cleared = false;
    final panel = FakePanel()
      ..notificationsOn = true
      ..extraRoutes.addAll({
        'GET /notifications': (_) => {
              'unread_count': items.where((n) => n['read'] != true).length,
              'notifications': cleared ? [] : items,
            },
        'POST /notifications/mark-read': (r) {
          readIds.add('n1');
          items[0]['read'] = true;
          return {'success': true};
        },
        'POST /notifications/clear': (_) {
          cleared = true;
          return {'success': true};
        },
      });
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    expect(
        find.descendant(
            of: find.byKey(const Key('notifications-badge')),
            matching: find.text('1')),
        findsOneWidget);

    await tester.tap(find.byKey(const Key('notifications')));
    await tester.pumpAndSettle();
    expect(find.text('Export of Items'), findsOneWidget);
    await tester.tap(find.byKey(const Key('notification-n1')));
    await tester.pumpAndSettle();
    expect(readIds, ['n1']);
    await tester.tap(find.byKey(const Key('notifications-clear')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('notifications-empty')), findsOneWidget);
    await tester.tap(find.text('Close'));
    await tester.pumpAndSettle();
    expect(
        find.descendant(
            of: find.byKey(const Key('notifications-badge')),
            matching: find.text('1')),
        findsNothing);
  });

  testWidgets('no bell when the panel has no notifications', (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: FakePanel().api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    expect(find.byKey(const Key('notifications')), findsNothing);
  });
}
