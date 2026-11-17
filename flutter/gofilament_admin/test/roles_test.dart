import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

Future<void> _openRoles(WidgetTester tester, FakePanel panel) async {
  tester.view.physicalSize = const Size(1400, 1100);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
  await tester.pumpAndSettle();
  await signIn(tester);
  await tester.tap(find.byKey(const Key('account-menu')));
  await tester.pumpAndSettle();
  await tester.tap(find.byKey(const Key('account-menu-roles')));
  await tester.pumpAndSettle();
}

Map<String, dynamic> _meta(List<Map<String, dynamic>> roles, bool canDefine) =>
    {
      'roles': roles,
      'permissions': {
        'items.viewAny': 'List items',
        'items.create': 'Create items'
      },
      'can_find_users': true,
      'can_define': canDefine,
      'panels': [
        if (canDefine) {'id': '*', 'label': 'All panels'},
        {'id': 'admin', 'label': 'Admin'},
        if (canDefine) {'id': 'app', 'label': 'App'},
      ],
      'tenants': [
        if (canDefine) {'id': '*', 'label': 'All tenants'},
        {'id': 'acme', 'label': 'Acme'},
        if (canDefine) {'id': 'globex', 'label': 'Globex'},
      ],
    };

void main() {
  testWidgets('a super admin defines a role and gives it in two panels',
      (tester) async {
    final roles = <Map<String, dynamic>>[
      {
        'name': 'admin',
        'label': 'Admin',
        'permissions': ['*'],
        'users': 1
      },
    ];
    final bindings = <Map<String, dynamic>>[];
    final saved = <Map<String, dynamic>>[];
    final panel = FakePanel()
      ..manageRoles = true
      ..extraRoutes.addAll({
        'GET /roles': (_) => _meta(roles, true),
        'POST /roles': (r) {
          final b = Map<String, dynamic>.from(json.decode(r.body) as Map);
          saved.add(b);
          roles.add({...b, 'users': 0});
          return _meta(roles, true);
        },
        'GET /roles/users': (_) => {
              'users': [
                {'id': 'bob', 'name': 'Bob', 'bindings': bindings}
              ]
            },
        'POST /roles/bindings': (r) {
          final b = json.decode(r.body) as Map;
          for (final p in b['panels'] as List) {
            for (final t in b['tenants'] as List) {
              bindings.add({
                'role': b['role'],
                'panel': p,
                'tenant': t,
                'can_remove': true
              });
            }
          }
          return {'success': true};
        },
        'DELETE /roles/bindings': (r) {
          final b = json.decode(r.body) as Map;
          bindings.removeWhere((x) =>
              x['panel'] == b['panel'] &&
              x['tenant'] == b['tenant'] &&
              x['role'] == b['role']);
          return {'success': true};
        },
      });
    await _openRoles(tester, panel);

    await tester.tap(find.byKey(const Key('role-new')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('role-name')), 'clerk');
    await tester.enterText(find.byKey(const Key('role-label')), 'Clerk');
    await tester.tap(find.byKey(const Key('perm-items.viewAny')));
    await tester.tap(find.byKey(const Key('role-save')));
    await tester.pumpAndSettle();
    expect(saved.single['permissions'], ['items.viewAny']);

    await tester.tap(find.byKey(const Key('grant-bob')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('grant-role')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Clerk').last);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('grant-panel-admin')));
    await tester.tap(find.byKey(const Key('grant-panel-app')));
    await tester.tap(find.byKey(const Key('grant-tenant-acme')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('grant-save')));
    await tester.pumpAndSettle();
    expect(bindings.length, 2);
    expect(find.byKey(const Key('binding-bob-clerk-app-acme')), findsOneWidget);

    await tester.tap(find.descendant(
        of: find.byKey(const Key('binding-bob-clerk-app-acme')),
        matching: find.byIcon(Icons.close)));
    await tester.pumpAndSettle();
    expect(bindings.length, 1);
  });

  testWidgets('a tenant admin cannot define roles', (tester) async {
    final panel = FakePanel()
      ..manageRoles = true
      ..extraRoutes.addAll({
        'GET /roles': (_) => _meta([
              {
                'name': 'editor',
                'label': 'Editor',
                'permissions': ['items.viewAny'],
                'users': 2
              },
            ], false),
        'GET /roles/users': (_) => {'users': []},
      });
    await _openRoles(tester, panel);
    expect(
        tester.widget<IconButton>(find.byKey(const Key('role-new'))).onPressed,
        isNull);
    expect(find.byKey(const Key('role-delete-editor')), findsNothing);
  });

  testWidgets('no Roles item without the permission', (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: FakePanel().api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await tester.tap(find.byKey(const Key('account-menu')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('account-menu-roles')), findsNothing);
  });
}
