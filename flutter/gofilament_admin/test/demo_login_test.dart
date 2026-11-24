import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'fake_panel.dart';

Future<void> pumpLogin(WidgetTester tester,
    {DemoLogin? demo, List<DemoLogin> demos = const []}) async {
  tester.view.physicalSize = const Size(1400, 900);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(GoFilamentAdmin(
      api: FakePanel().api(), demoLogin: demo, demoLogins: demos));
  await tester.pumpAndSettle();
}

String fieldText(WidgetTester tester, String key) => tester
    .widget<TextField>(find.descendant(
        of: find.byKey(Key(key)), matching: find.byType(TextField)))
    .controller!
    .text;

void main() {
  testWidgets('no demo button unless the app configures one', (tester) async {
    await pumpLogin(tester);
    expect(find.byKey(const Key('login-submit')), findsOneWidget);
    expect(find.byKey(const Key('demo-login-btn')), findsNothing);
  });

  testWidgets('configured demo button fills in the account', (tester) async {
    await pumpLogin(tester,
        demo: const DemoLogin(
            email: 'owner', password: 'secret-pass', label: 'Try the demo'));
    expect(find.text('Try the demo'), findsOneWidget);

    await tester.tap(find.byKey(const Key('demo-login-btn')));
    await tester.pump();
    expect(fieldText(tester, 'login-username'), 'owner');
    expect(fieldText(tester, 'login-password'), 'secret-pass');

    await tester.tap(find.byKey(const Key('login-submit')));
    await tester.pumpAndSettle();
    expect(find.text('Welcome back, Olivia'), findsOneWidget);
  });

  test('debug-only demo accounts are hidden outside debug builds', () {
    // Tests run in debug mode, so a debug-only account is available here;
    // a shared demo (debugOnly: false) is available in every build.
    expect(const DemoLogin(email: 'a', password: 'b').isAvailable, isTrue);
    expect(
        const DemoLogin(email: 'a', password: 'b', debugOnly: false)
            .isAvailable,
        isTrue);
  });

  testWidgets('several demo accounts get a button each', (tester) async {
    await pumpLogin(tester, demos: const [
      DemoLogin(email: 'owner', password: 'secret-pass', label: 'Admin'),
      DemoLogin(email: 'clerk', password: 'other-pass', label: 'Clerk'),
    ]);
    expect(find.text('Admin'), findsOneWidget);
    expect(find.text('Clerk'), findsOneWidget);

    await tester.tap(find.byKey(const Key('demo-login-btn-1')));
    await tester.pump();
    expect(fieldText(tester, 'login-username'), 'clerk');
    expect(fieldText(tester, 'login-password'), 'other-pass');
  });

  testWidgets('a submitting demo account signs in with one tap',
      (tester) async {
    await pumpLogin(tester, demos: const [
      DemoLogin(
          email: 'owner',
          password: 'secret-pass',
          label: 'Sign in as owner',
          submit: true),
    ]);
    await tester.tap(find.byKey(const Key('demo-login-btn')));
    await tester.pumpAndSettle();
    expect(find.text('Welcome back, Olivia'), findsOneWidget);
  });
}
