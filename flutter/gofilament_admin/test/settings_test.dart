import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

Future<void> _openSettings(WidgetTester tester, AdminApi api) async {
  tester.view.physicalSize = const Size(1400, 1000);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(GoFilamentAdmin(api: api));
  await tester.pumpAndSettle();
  await signIn(tester);
  await tester.tap(find.byKey(const Key('account-menu')));
  await tester.pumpAndSettle();
  await tester.tap(find.byKey(const Key('account-menu-settings')));
  await tester.pumpAndSettle();
}

Brightness _brightness(WidgetTester tester) =>
    Theme.of(tester.element(find.byKey(const Key('settings-theme'))))
        .brightness;

void main() {
  testWidgets('settings switch the theme, including back to the system one',
      (tester) async {
    final api = FakePanel().api();
    await _openSettings(tester, api);
    expect(find.text('Settings'), findsWidgets);

    await tester.tap(find.text('Dark'));
    await tester.pumpAndSettle();
    expect(_brightness(tester), Brightness.dark);
    expect(await api.readPreference('theme'), 'dark');

    await tester.tap(find.text('Light'));
    await tester.pumpAndSettle();
    expect(_brightness(tester), Brightness.light);

    await tester.tap(find.text('System'));
    await tester.pumpAndSettle();
    expect(await api.readPreference('theme'), 'system');
  });

  testWidgets('settings pick the language', (tester) async {
    final panel = FakePanel()..bilingual = true;
    final api = panel.api();
    await _openSettings(tester, api);
    await tester.tap(find.byKey(const Key('settings-language-ar')));
    await tester.pumpAndSettle();
    expect(api.locale, 'ar');
    expect(await api.readPreference('locale'), 'ar');
  });

  testWidgets('settings set the sidebar, shared with the top bar',
      (tester) async {
    final panel = FakePanel()
      ..sidebar = {
        'modes': ['expanded', 'icons', 'hidden']
      };
    final api = panel.api();
    await _openSettings(tester, api);
    await tester.tap(find.text('Icons only'));
    await tester.pumpAndSettle();
    expect(await api.readPreference('sidebar_mode'), 'icons');

    // settings open as a dialog on a desktop
    await tester.tap(find.byType(CloseButton));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('sidebar-icons')), findsOneWidget);
  });

  testWidgets('sections the panel does not offer are left out', (tester) async {
    await _openSettings(tester, FakePanel().api());
    expect(find.byKey(const Key('settings-theme')), findsOneWidget);
    expect(find.byKey(const Key('settings-sidebar')), findsNothing);
    expect(find.byKey(const Key('settings-language-en')), findsNothing);
  });

  test('icon names in every Heroicons spelling', () {
    expect(adminIcon('heroicon-o-users'), Icons.groups_outlined);
    expect(adminIcon('heroicon-s-users'), Icons.groups_outlined);
    expect(adminIcon('heroicon-shopping-bag'), Icons.shopping_bag_outlined);
    expect(adminIcon('heroicons-qr-code'), Icons.qr_code);
    expect(adminIcon('rectangle-stack'), Icons.layers_outlined);
    expect(adminIcon('heroicon-nope'), Icons.table_rows_outlined);
  });
}
