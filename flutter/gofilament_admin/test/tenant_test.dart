import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

void main() {
  testWidgets('the tenant menu switches tenant and keeps it', (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final panel = FakePanel()
      ..tenants = [
        {'key': 'acme', 'name': 'Acme'},
        {'key': 'globex', 'name': 'Globex'},
      ];
    final api = panel.api();
    await tester.pumpWidget(GoFilamentAdmin(api: api));
    await tester.pumpAndSettle();
    await signIn(tester);
    expect(find.text('Acme'), findsOneWidget);

    await tester.tap(find.byKey(const Key('tenant-menu')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('tenant-globex')));
    await tester.pumpAndSettle();
    expect(api.tenant, 'globex');
    expect(panel.requests.last.headers['X-Tenant-ID'], 'globex');
    expect(find.byKey(const Key('tenant-current')), findsOneWidget);
    expect(find.text('Globex'), findsOneWidget);
    expect(await api.readPreference('tenant'), 'globex');
  });

  testWidgets('no tenant menu without tenancy', (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: FakePanel().api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    expect(find.byKey(const Key('tenant-menu')), findsNothing);
  });
}
