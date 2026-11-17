import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

const _features = {
  'profile': true,
  'update_profile': true,
  'password': true,
  'password_reset': true,
  'two_factor': true,
  'recovery_codes': true,
};

Future<void> _boot(WidgetTester tester, FakePanel panel) async {
  tester.view.physicalSize = const Size(1400, 1100);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
  await tester.pumpAndSettle();
}

/// A fake account endpoint set: state kept in [state].
void _accountRoutes(FakePanel panel, Map<String, dynamic> state,
    List<Map<String, dynamic>> calls) {
  Map<String, dynamic> body(r) =>
      Map<String, dynamic>.from(json.decode(r.body) as Map);
  Map<String, dynamic> view() => {...state, 'features': _features};
  panel.extraRoutes.addAll({
    'GET /account': (_) => view(),
    'PUT /account': (r) {
      calls.add({'put': body(r)});
      state.addAll(body(r));
      return view();
    },
    'POST /account/password': (r) {
      calls.add({'password': body(r)});
      return {'message': 'ok'};
    },
    'POST /account/two-factor/enable': (r) {
      calls.add({'enable': body(r)});
      return {'secret': 'JBSWY3DPEHPK3PXP', 'otpauth_uri': 'otpauth://totp/x'};
    },
    'POST /account/two-factor/confirm': (r) {
      calls.add({'confirm': body(r)});
      state['two_factor_enabled'] = true;
      state['recovery_codes_left'] = 2;
      return {
        ...view(),
        'recovery_codes': ['aaaaa-bbbbb', 'ccccc-ddddd']
      };
    },
    'POST /account/two-factor/disable': (r) {
      calls.add({'disable': body(r)});
      state['two_factor_enabled'] = false;
      return view();
    },
  });
}

void main() {
  testWidgets('sign-in asks for the two-factor code', (tester) async {
    final panel = FakePanel()
      ..twoFactorLogin = true
      ..accountFeatures = Map.of(_features);
    await _boot(tester, panel);
    await signIn(tester);
    expect(find.byKey(const Key('login-code')), findsOneWidget);
    expect(find.textContaining('recovery codes'), findsOneWidget);

    await tester.enterText(find.byKey(const Key('login-code')), '000000');
    await tester.tap(find.byKey(const Key('login-code-submit')));
    await tester.pumpAndSettle();
    expect(find.text('The code is not valid.'), findsOneWidget);

    await tester.enterText(find.byKey(const Key('login-code')), '123456');
    await tester.tap(find.byKey(const Key('login-code-submit')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('login-code')), findsNothing);
    expect(find.byKey(const Key('account-menu')), findsOneWidget);
  });

  testWidgets('forgot password sends the email', (tester) async {
    final panel = FakePanel()..accountFeatures = Map.of(_features);
    await _boot(tester, panel);
    await tester.tap(find.byKey(const Key('login-forgot')));
    await tester.pumpAndSettle();
    await tester.enterText(
        find.byKey(const Key('forgot-email')), 'o@acme.test');
    await tester.tap(find.byKey(const Key('forgot-send')));
    await tester.pumpAndSettle();
    expect(panel.resetRequests, ['o@acme.test']);
    expect(find.textContaining('we sent a link'), findsOneWidget);
  });

  testWidgets('no forgot link or account item without the features',
      (tester) async {
    final panel = FakePanel();
    await _boot(tester, panel);
    expect(find.byKey(const Key('login-forgot')), findsNothing);
    await signIn(tester);
    await tester.tap(find.byKey(const Key('account-menu')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('account-menu-profile')), findsNothing);
  });

  testWidgets('the account screen edits the profile, password and 2FA',
      (tester) async {
    final panel = FakePanel()..accountFeatures = Map.of(_features);
    final state = <String, dynamic>{
      'name': 'Olivia Owner',
      'email': 'o@acme.test',
      'two_factor_enabled': false,
    };
    final calls = <Map<String, dynamic>>[];
    _accountRoutes(panel, state, calls);
    await _boot(tester, panel);
    await signIn(tester);
    await tester.tap(find.byKey(const Key('account-menu')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('account-menu-profile')));
    await tester.pumpAndSettle();

    // Profile
    expect(find.text('Olivia Owner'), findsWidgets);
    await tester.enterText(find.byKey(const Key('account-name')), 'Liv');
    await tester.tap(find.byKey(const Key('account-save')));
    await tester.pumpAndSettle();
    expect(calls.last['put'], {'name': 'Liv', 'email': 'o@acme.test'});

    // Password
    await tester.enterText(
        find.byKey(const Key('account-current-password')), 'old');
    await tester.enterText(
        find.byKey(const Key('account-new-password')), 'new-pass-1');
    await tester.enterText(
        find.byKey(const Key('account-confirm-password')), 'new-pass-1');
    await tester.tap(find.byKey(const Key('account-change-password')));
    await tester.pumpAndSettle();
    expect(calls.last['password']['current_password'], 'old');

    // Two-factor: set up (asks the password), confirm, see recovery codes.
    await tester.ensureVisible(find.byKey(const Key('two-factor-setup')));
    await tester.tap(find.byKey(const Key('two-factor-setup')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('confirm-password')), 'pw');
    await tester.tap(find.byKey(const Key('confirm-password-ok')));
    await tester.pumpAndSettle();
    expect(calls.last['enable'], {'current_password': 'pw'});
    expect(find.text('JBSWY3DPEHPK3PXP'), findsOneWidget);
    await tester.enterText(find.byKey(const Key('two-factor-code')), '654321');
    await tester.ensureVisible(find.byKey(const Key('two-factor-confirm')));
    await tester.tap(find.byKey(const Key('two-factor-confirm')));
    await tester.pumpAndSettle();
    expect(calls.last['confirm'], {'code': '654321'});
    expect(find.byKey(const Key('recovery-codes')), findsOneWidget);
    expect(find.textContaining('aaaaa-bbbbb'), findsOneWidget);
    expect(find.text('Recovery codes left: 2'), findsOneWidget);

    // Turn it off.
    await tester.ensureVisible(find.byKey(const Key('two-factor-disable')));
    await tester.tap(find.byKey(const Key('two-factor-disable')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('confirm-password')), 'pw');
    await tester.tap(find.byKey(const Key('confirm-password-ok')));
    await tester.pumpAndSettle();
    expect(calls.last.containsKey('disable'), isTrue);
    expect(find.byKey(const Key('two-factor-setup')), findsOneWidget);
    expect(find.byKey(const Key('recovery-codes')), findsNothing);
  });

  testWidgets('sign-up creates the account and waits for verification',
      (tester) async {
    final panel = FakePanel()
      ..accountFeatures = Map.of(_features)
      ..accountFeatures['registration'] = true
      ..emailUnverified = true;
    await _boot(tester, panel);
    await tester.tap(find.byKey(const Key('login-register')));
    await tester.pumpAndSettle();

    Future<void> fill(String email) async {
      await tester.enterText(find.byKey(const Key('register-name')), 'Nia');
      await tester.enterText(find.byKey(const Key('register-email')), email);
      await tester.enterText(
          find.byKey(const Key('register-password')), 'long-password');
      await tester.enterText(
          find.byKey(const Key('register-confirm')), 'long-password');
      await tester.tap(find.byKey(const Key('register-submit')));
      await tester.pumpAndSettle();
    }

    await fill('o@acme.test');
    expect(find.text('An account with this email address already exists.'),
        findsOneWidget);

    await fill('nia@acme.test');
    expect(panel.registrations.last['email'], 'nia@acme.test');
    expect(find.byKey(const Key('register-submit')), findsNothing);
    expect(find.text('Verify your email'), findsOneWidget);

    await tester.tap(find.byKey(const Key('verify-resend')));
    await tester.pumpAndSettle();
    expect(panel.verificationMails, 1);
    expect(find.byKey(const Key('verify-message')), findsOneWidget);

    panel.emailUnverified = false;
    await tester.tap(find.byKey(const Key('verify-continue')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('account-menu')), findsOneWidget);
  });

  testWidgets('no sign-up link without registration', (tester) async {
    final panel = FakePanel()..accountFeatures = Map.of(_features);
    await _boot(tester, panel);
    expect(find.byKey(const Key('login-register')), findsNothing);
  });
}
