import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

void main() {
  tearDown(() => adminPickFiles = (
          {bool multiple = false, List<String> extensions = const []}) async =>
      const []);

  testWidgets('import previews the file, maps columns and imports',
      (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final panel = FakePanel()..canImport = true;
    List<String>? asked;
    adminPickFiles =
        ({bool multiple = false, List<String> extensions = const []}) async {
      asked = extensions;
      return [
        (
          name: 'people.csv',
          bytes: Uint8List.fromList(
              utf8.encode('Name,Mail\nBea,bea@x.io\nCal,nope\n'))
        )
      ];
    };
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);

    await tester.tap(find.byKey(const Key('list-import')));
    await tester.pumpAndSettle();
    expect(asked, contains('xlsx'));
    expect(panel.importRequests.single, contains('name="dry_run"'));
    expect(find.text('2 rows: 0 ready to import, 2 with problems.'),
        findsOneWidget);
    expect(
        tester
            .widget<FilledButton>(find.byKey(const Key('import-run')))
            .onPressed,
        isNull);

    // Map "Mail" to Email and check again.
    await tester.tap(find.byKey(const Key('import-map-Mail')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Email *').last);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('import-recheck')));
    await tester.pumpAndSettle();
    expect(panel.importRequests.last, contains('"Mail":"email"'));
    expect(find.text('2 rows: 1 ready to import, 1 with problems.'),
        findsOneWidget);
    expect(find.textContaining('Enter a valid email.'), findsWidgets);

    await tester.tap(find.byKey(const Key('import-run')));
    await tester.pumpAndSettle();
    expect(panel.importRequests.last, isNot(contains('name="dry_run"')));
    expect(find.byKey(const Key('import-result')), findsOneWidget);
    expect(find.text('1 rows were skipped.'), findsOneWidget);
    final lists = panel.requests
        .where(
            (r) => r.url.path.endsWith('/resources/items') && r.method == 'GET')
        .length;
    await tester.tap(find.byKey(const Key('import-done')));
    await tester.pumpAndSettle();
    expect(
        panel.requests
            .where((r) =>
                r.url.path.endsWith('/resources/items') && r.method == 'GET')
            .length,
        greaterThan(lists));
  });

  testWidgets('no import button without the ability', (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final panel = FakePanel();
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);
    expect(find.byKey(const Key('list-import')), findsNothing);
  });
}
