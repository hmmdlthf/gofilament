import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

Future<void> _pump(WidgetTester tester, AdminApi api,
    {Size size = const Size(1400, 900)}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(GoFilamentAdmin(api: api));
  await tester.pumpAndSettle();
}

Future<void> _toggle(WidgetTester tester) async {
  await tester.tap(find.byKey(const Key('sidebar-toggle')));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('the desktop sidebar cycles through expanded, icons and hidden',
      (tester) async {
    final panel = FakePanel()
      ..sidebar = {
        'collapsible': true,
        'modes': ['expanded', 'icons', 'hidden'],
        'default': 'expanded'
      };
    await _pump(tester, panel.api());
    await signIn(tester);
    expect(find.byKey(const Key('sidebar-expanded')), findsOneWidget);

    await _toggle(tester);
    expect(find.byKey(const Key('sidebar-icons')), findsOneWidget);
    expect(tester.getSize(find.byKey(const Key('sidebar-icons'))).width, 72);
    // labels become tooltips, and items still navigate
    expect(find.byTooltip('Dashboard'), findsOneWidget);
    await tester.tap(find.byKey(const Key('nav-resource:items')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('sidebar-icons')), findsOneWidget);

    await _toggle(tester);
    expect(find.byKey(const Key('sidebar-hidden')), findsNothing);
    expect(find.byKey(const Key('sidebar-icons')), findsNothing);
    expect(find.byKey(const Key('nav-dashboard')), findsNothing);

    await _toggle(tester);
    expect(find.byKey(const Key('sidebar-expanded')), findsOneWidget);
  });

  testWidgets('the chosen mode is remembered on the device', (tester) async {
    final panel = FakePanel()
      ..sidebar = {
        'modes': ['expanded', 'icons', 'hidden']
      };
    final api = panel.api();
    await _pump(tester, api);
    await signIn(tester);
    await _toggle(tester); // icons
    expect(find.byKey(const Key('sidebar-icons')), findsOneWidget);

    await tester.pumpWidget(const SizedBox());
    await _pump(tester, api);
    expect(find.byKey(const Key('sidebar-icons')), findsOneWidget);
  });

  testWidgets('the panel picks the modes and the starting one',
      (tester) async {
    final panel = FakePanel()
      ..sidebar = {
        'modes': ['expanded', 'icons'],
        'default': 'icons'
      };
    await _pump(tester, panel.api());
    await signIn(tester);
    expect(find.byKey(const Key('sidebar-icons')), findsOneWidget);
    await _toggle(tester);
    expect(find.byKey(const Key('sidebar-expanded')), findsOneWidget);
    await _toggle(tester);
    expect(find.byKey(const Key('sidebar-icons')), findsOneWidget);
  });

  testWidgets('no toggle without collapse modes, nor on narrow screens',
      (tester) async {
    await _pump(tester, FakePanel().api());
    await signIn(tester);
    expect(find.byKey(const Key('sidebar-toggle')), findsNothing);
    expect(find.byKey(const Key('sidebar-expanded')), findsOneWidget);

    final narrow = FakePanel()
      ..sidebar = {
        'modes': ['expanded', 'icons', 'hidden'],
        'default': 'icons'
      };
    await tester.pumpWidget(const SizedBox());
    await _pump(tester, narrow.api(), size: const Size(700, 900));
    await signIn(tester);
    expect(find.byKey(const Key('sidebar-toggle')), findsNothing);
    expect(find.byKey(const Key('sidebar-icons')), findsNothing);
  });
}
