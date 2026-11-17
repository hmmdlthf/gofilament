import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

void main() {
  testWidgets('roles: create a role and give it to a user', (tester) async {
    tester.view.physicalSize = const Size(1400, 1100);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final roles = <Map<String, dynamic>>[
      {
        'name': 'admin',
        'label': 'Admin',
        'permissions': ['*'],
        'users': 1
      },
    ];
    final user = {'id': 'bob', 'name': 'Bob', 'roles': <String>[]};
    final saved = <Map<String, dynamic>>[];
    Map<String, dynamic> list() => {
          'roles': roles,
          'permissions': {
            'items.viewAny': 'List items',
            'items.create': 'Create items',
            'roles.manage': 'Manage roles'
          },
          'can_find_users': true,
        };
    final panel = FakePanel()
      ..manageRoles = true
      ..extraRoutes.addAll({
        'GET /roles': (_) => list(),
        'POST /roles': (r) {
          final b = Map<String, dynamic>.from(json.decode(r.body) as Map);
          saved.add(b);
          roles.add({...b, 'users': 0});
          return list();
        },
        'GET /roles/users': (_) => {
              'users': [user]
            },
        'PUT /roles/users/bob': (r) {
          user['roles'] = (json.decode(r.body) as Map)['roles'];
          return {'id': 'bob'};
        },
      });
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await tester.tap(find.byKey(const Key('account-menu')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('account-menu-roles')));
    await tester.pumpAndSettle();
    expect(find.text('Admin'), findsWidgets);

    await tester.tap(find.byKey(const Key('role-new')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('role-name')), 'clerk');
    await tester.enterText(find.byKey(const Key('role-label')), 'Clerk');
    await tester.tap(find.byKey(const Key('perm-items.viewAny')));
    await tester.tap(find.byKey(const Key('role-save')));
    await tester.pumpAndSettle();
    expect(saved.single, {
      'name': 'clerk',
      'label': 'Clerk',
      'permissions': ['items.viewAny']
    });
    expect(find.byKey(const Key('role-clerk')), findsOneWidget);

    await tester.tap(find.byKey(const Key('user-bob-role-clerk')));
    await tester.pumpAndSettle();
    expect(user['roles'], ['clerk']);
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
