import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

void main() {
  testWidgets('switching to Arabic lays the app out right to left',
      (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final panel = FakePanel()..bilingual = true;
    final api = panel.api();
    await tester.pumpWidget(GoFilamentAdmin(api: api));
    await tester.pumpAndSettle();
    await signIn(tester);
    TextDirection dir() => Directionality.of(
        tester.element(find.byKey(const Key('account-menu'))));
    expect(dir(), TextDirection.ltr);

    await tester.tap(find.byKey(const Key('account-menu')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('language-ar')));
    await tester.pumpAndSettle();
    expect(api.locale, 'ar');
    expect(panel.requests.last.headers['Accept-Language'], 'ar');
    expect(dir(), TextDirection.rtl);
    expect(await api.readPreference('locale'), 'ar');
  });

  testWidgets('no language menu with one language', (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: FakePanel().api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await tester.tap(find.byKey(const Key('account-menu')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('language-ar')), findsNothing);
  });
}
