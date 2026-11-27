import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

Map<String, dynamic> _feature(String key, String type, Object? value,
        {bool overridden = false, List<String>? options, String? label}) =>
    {
      'key': key,
      'type': type,
      'label': label ?? key,
      'group': 'Billing',
      'managed_by': 'tenant_admin',
      'public': false,
      if (options != null) 'options': options,
      'default': value,
      'value': value,
      'overridden': overridden,
    };

void main() {
  testWidgets('a tenant admin changes, fails to save and resets settings',
      (tester) async {
    tester.view.physicalSize = const Size(1400, 1100);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    var features = [
      _feature('tax.mode', 'enum', 'inclusive',
          options: ['inclusive', 'exclusive', 'none'], label: 'Tax mode'),
      _feature('branches.max', 'int', 1, label: 'Branches'),
      _feature('ui.beta', 'bool', false, label: 'Beta'),
    ];
    Map<String, dynamic> listing() => {
          'tenant': 'acme',
          'groups': [
            {'name': 'Billing', 'features': features}
          ]
        };
    final saves = <Map<String, dynamic>>[];
    final resets = <String>[];
    final panel = FakePanel()
      ..tenantSettings = {
        'enabled': true,
        'label': 'Workspace settings',
        'path': '/tenant-settings'
      };
    panel.extraRoutes.addAll({
      'GET /tenant-settings': (_) => listing(),
      'PUT /tenant-settings': (r) {
        final values = Map<String, dynamic>.from(
            (json.decode(r.body) as Map)['values'] as Map);
        saves.add(values);
        if (values['branches.max'] == '99') {
          panel.extraStatus['PUT /tenant-settings'] = 422;
          return {
            ...listing(),
            'errors': {'branches.max': 'Must be between 1 and 50.'}
          };
        }
        panel.extraStatus.remove('PUT /tenant-settings');
        features = [
          for (final f in features)
            values.containsKey(f['key'])
                ? {...f, 'value': values[f['key']], 'overridden': true}
                : f
        ];
        return listing();
      },
      'DELETE /tenant-settings/tax.mode': (_) {
        resets.add('tax.mode');
        features = [
          for (final f in features)
            f['key'] == 'tax.mode'
                ? {...f, 'value': 'inclusive', 'overridden': false}
                : f
        ];
        return listing();
      },
    });

    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await tester.tap(find.byKey(const Key('account-menu')));
    await tester.pumpAndSettle();
    expect(find.text('Workspace settings'), findsOneWidget);
    await tester.tap(find.byKey(const Key('account-menu-tenant-settings')));
    await tester.pumpAndSettle();

    expect(find.text('Billing'), findsOneWidget);
    expect(find.text('Default'), findsNWidgets(3));
    expect(find.text('Reset to default'), findsNothing);

    // A value out of range: the error shows on the field.
    await tester.enterText(
        find.byKey(const Key('tenant-setting-branches.max')), '99');
    await tester.pump();
    await tester.tap(find.byKey(const Key('tenant-settings-save')));
    await tester.pumpAndSettle();
    expect(find.text('Must be between 1 and 50.'), findsOneWidget);

    // Fixed, plus an enum and a switch.
    await tester.enterText(
        find.byKey(const Key('tenant-setting-branches.max')), '7');
    await tester.tap(find.byKey(const Key('tenant-setting-tax.mode')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('exclusive').last);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('tenant-setting-ui.beta')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('tenant-settings-save')));
    await tester.pumpAndSettle();
    expect(saves.last,
        {'branches.max': '7', 'tax.mode': 'exclusive', 'ui.beta': true});
    expect(find.text('Must be between 1 and 50.'), findsNothing);
    expect(find.text('Overridden'), findsNWidgets(3));

    await tester.tap(find.byKey(const Key('tenant-setting-reset-tax.mode')));
    await tester.pumpAndSettle();
    expect(resets, ['tax.mode']);
    expect(find.text('Overridden'), findsNWidgets(2));
  });

  testWidgets('a super admin picks a tenant and edits its features',
      (tester) async {
    tester.view.physicalSize = const Size(1400, 1100);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);

    var recurring = {
      ..._feature('invoices.recurring', 'bool', false,
          label: 'Recurring invoices'),
      'managed_by': 'super_admin',
    };
    Map<String, dynamic> listing() => {
          'tenant': 'acme',
          'groups': [
            {
              'name': 'Billing',
              'features': [recurring]
            }
          ]
        };
    final saves = <Map<String, dynamic>>[];
    final panel = FakePanel()
      ..featuresAdmin = {
        'enabled': true,
        'label': 'Tenant features',
        'path': '/tenant-features'
      };
    panel.extraRoutes.addAll({
      'GET /tenant-features': (_) => {
            'tenants': [
              {'key': 'acme', 'name': 'Acme Inc'},
              {'key': 'globex', 'name': 'Globex'},
            ],
            'free_entry': false,
          },
      'GET /tenant-features/acme': (_) => listing(),
      'PUT /tenant-features/acme': (r) {
        final values = Map<String, dynamic>.from(
            (json.decode(r.body) as Map)['values'] as Map);
        saves.add(values);
        recurring = {...recurring, 'value': true, 'overridden': true};
        return listing();
      },
    });

    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await tester.tap(find.byKey(const Key('account-menu')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('account-menu-tenant-features')));
    await tester.pumpAndSettle();
    expect(find.text('Globex'), findsOneWidget);
    await tester.tap(find.byKey(const Key('tenant-features-acme')));
    await tester.pumpAndSettle();

    expect(find.text('Acme Inc'), findsOneWidget); // app bar title
    expect(find.text('Super admin only'), findsOneWidget);
    await tester
        .tap(find.byKey(const Key('tenant-setting-invoices.recurring')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('tenant-settings-save')));
    await tester.pumpAndSettle();
    expect(saves.single, {'invoices.recurring': true});
    expect(find.text('Overridden'), findsOneWidget);
  });

  testWidgets('no settings entry without tenant_settings in the schema',
      (tester) async {
    tester.view.physicalSize = const Size(1400, 1100);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final panel = FakePanel();
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await tester.tap(find.byKey(const Key('account-menu')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('account-menu-tenant-settings')), findsNothing);
  });
}
