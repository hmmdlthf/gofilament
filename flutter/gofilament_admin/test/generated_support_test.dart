import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'fake_panel.dart';

/// A snapshot like the generated `panelSchema`: no abilities, marked as such.
const snapshot = <String, dynamic>{
  'snapshot': true,
  'brand': <String, dynamic>{'name': 'Acme Snapshot'},
  'brand_name': 'Acme Snapshot',
  'navigation': <dynamic>[],
  'client_pages': <dynamic>[
    <String, dynamic>{'id': 'pos', 'label': 'Till'},
  ],
  'resources': <dynamic>[
    <String, dynamic>{
      'slug': 'items',
      'model_name': 'Item',
      'plural_model_name': 'Items',
      'group': 'Catalog',
      'columns': <dynamic>[],
    },
  ],
  'resource_meta': <String, dynamic>{
    'items': <String, dynamic>{'has_form': true},
  },
};

void main() {
  group('GfJson', () {
    test('reads leniently', () {
      expect(GfJson.integer('12'), 12);
      expect(GfJson.integer(12.7), 12);
      expect(GfJson.decimal('1.5'), 1.5);
      expect(GfJson.string(5), '5');
      expect(GfJson.boolean('1'), isTrue);
      expect(GfJson.boolean('maybe'), isNull);
      expect(GfJson.dateTime('0001-01-01T00:00:00Z'), isNull);
      expect(GfJson.dateTime('2026-03-04T05:06:07Z')!.month, 3);
      expect(GfJson.list(<dynamic>['1', 2], (e) => GfJson.integer(e) ?? 0),
          [1, 2]);
      expect(GfJson.object(<String, dynamic>{'a': 1}, (m) => m['a']), 1);
      expect(GfJson.object('x', (m) => m), isNull);
    });

    test('writes dates and queries', () {
      expect(GfJson.date(DateTime(2026, 1, 5)), '2026-01-05');
      expect(
          GfJson.query({
            'a': 1,
            'b': null,
            'c': ['x', 'y']
          }),
          {'a': '1', 'c': 'x,y'});
      expect(GfJson.withQuery('/p', {'a': '1 2'}), '/p?a=1+2');
      expect(GfJson.withQuery('/p?x=1', {'a': '2'}), '/p?x=1&a=2');
      expect(GfJson.withQuery('/p', {}), '/p');
    });

    test('TypedPage decodes rows and exposes row abilities', () {
      final page = ListPage(rows: [
        {'id': 1, 'name': 'A'}
      ], total: 1, page: 1, perPage: 25)
        ..rowMeta = {
          '1': {
            'update': true,
            'actions': ['post']
          }
        };
      final typed = TypedPage.from(page, (m) => m['name'] as String);
      expect(typed.items, ['A']);
      expect(typed.total, 1);
      expect(typed.canRun('1', 'post'), isTrue);
      expect(typed.canRun('1', 'refund'), isFalse);
      expect(typed.rowMeta('9'), isEmpty);
    });
  });

  group('initial schema', () {
    Future<(AdminController, Completer<void>, FakePanel)> start(
        WidgetTester tester,
        {String location = ''}) async {
      final panel = FakePanel()..clientPages = ['pos'];
      final store = MemorySessionStore();
      // a stored session, as after an earlier sign-in
      await AdminApi(
              baseUrl: 'http://acme.test/app',
              client: MockClient(panel.handle),
              store: store)
          .login('owner', 'secret-pass');

      final gate = Completer<void>();
      final api = AdminApi(
          baseUrl: 'http://acme.test/app',
          store: store,
          client: MockClient((http.Request req) async {
            if (req.url.path.endsWith('/schema')) await gate.future;
            return panel.handle(req);
          }));
      final admin = AdminController(
        api: api,
        initialSchema: snapshot,
        pages: [
          AdminPage(
              id: 'pos',
              label: 'Till',
              serverControlled: true,
              builder: (_, __) => const Text('TILL')),
          AdminPage(
              id: 'secret',
              label: 'Secret',
              serverControlled: true,
              builder: (_, __) => const Text('SECRET')),
        ],
      )..initialLocation = location;
      tester.view.physicalSize = const Size(1400, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(GoFilamentAdmin(controller: admin));
      return (admin, gate, panel);
    }

    testWidgets('draws the shell from the snapshot, then uses the live schema',
        (tester) async {
      final (admin, gate, _) = await start(tester);
      await tester.pump();
      await tester.pump();

      // first frames: snapshot, no permissions, no server-controlled pages
      expect(admin.stage, AdminStage.signedIn);
      expect(admin.schemaIsLive, isFalse);
      expect(find.byKey(const Key('nav-resource:items')), findsOneWidget);
      expect(admin.can('items', 'create'), isFalse);
      expect(find.byKey(const Key('nav-page:pos')), findsNothing);

      gate.complete();
      await tester.pumpAndSettle();
      expect(admin.schemaIsLive, isTrue);
      expect(admin.can('items', 'create'), isTrue);
      expect(find.text('Acme Stores'), findsWidgets);
      // the live schema lists pos but not secret
      expect(find.byKey(const Key('nav-page:pos')), findsOneWidget);
      expect(find.byKey(const Key('nav-page:secret')), findsNothing);
    });

    testWidgets('a deep link to a page waits for the live schema',
        (tester) async {
      final (_, gate, _) = await start(tester, location: '/pages/pos');
      await tester.pump();
      await tester.pump();
      expect(find.text('TILL'), findsNothing);
      gate.complete();
      await tester.pumpAndSettle();
      expect(find.text('TILL'), findsOneWidget);
    });

    testWidgets('a deep link to a page the server does not list is ignored',
        (tester) async {
      final (_, gate, _) = await start(tester, location: '/pages/secret');
      gate.complete();
      await tester.pumpAndSettle();
      expect(find.text('SECRET'), findsNothing);
    });

    testWidgets('without a stored session the login screen is drawn',
        (tester) async {
      final panel = FakePanel();
      final admin = AdminController(api: panel.api(), initialSchema: snapshot);
      tester.view.physicalSize = const Size(1400, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(GoFilamentAdmin(controller: admin));
      await tester.pumpAndSettle();
      expect(admin.stage, AdminStage.signedOut);
      expect(find.byKey(const Key('login-submit')), findsOneWidget);
    });
  });
}
