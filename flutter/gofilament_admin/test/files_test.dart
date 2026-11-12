import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

void main() {
  tearDown(() => adminPickFiles = _noFiles);

  testWidgets('file fields upload the chosen files and save their values',
      (tester) async {
    final panel = FakePanel()
      ..extraFormFields = [
        {
          'kind': 'field',
          'type': 'file',
          'name': 'manuals',
          'label': 'Manuals',
          'multiple': true,
          'accept': ['application/pdf'],
        },
        {
          'kind': 'field',
          'type': 'image',
          'name': 'photo',
          'label': 'Photo',
          'required': true,
        },
      ];
    var n = 0;
    panel.extraRoutes['POST /resources/items/uploads'] = (req) => {
          'value': 'up/${req.url.queryParameters['field']}-${++n}',
          'name': 'file$n',
        };
    List<String>? askedExtensions;
    adminPickFiles =
        ({bool multiple = false, List<String> extensions = const []}) async {
      askedExtensions = extensions;
      return [
        (name: 'a.pdf', bytes: Uint8List.fromList([37, 80, 68, 70])),
        if (multiple) (name: 'b.pdf', bytes: Uint8List.fromList([37, 80])),
      ];
    };

    tester.view.physicalSize = const Size(1400, 2000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);
    await tester.tap(find.byKey(const Key('list-create')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('field-code')), 'DOC');
    await tester.enterText(find.byKey(const Key('field-name')), 'Doc');

    await tester.ensureVisible(find.byKey(const Key('form-save')));
    await tester.tap(find.byKey(const Key('form-save')));
    await tester.pumpAndSettle();
    expect(find.text('Photo is required'), findsOneWidget);

    await tester.ensureVisible(find.byKey(const Key('manuals-pick')));
    await tester.tap(find.byKey(const Key('manuals-pick')));
    await tester.pumpAndSettle();
    expect(askedExtensions, ['pdf']);
    expect(find.text('file1'), findsOneWidget);
    expect(find.text('file2'), findsOneWidget);
    await tester.tap(find.byKey(const Key('manuals-file-remove-0')));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.byKey(const Key('photo-pick')));
    await tester.tap(find.byKey(const Key('photo-pick')));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.byKey(const Key('form-save')));
    await tester.tap(find.byKey(const Key('form-save')));
    await tester.pumpAndSettle();
    final saved = panel.items.last;
    expect(saved['manuals'], ['up/manuals-2']);
    expect(saved['photo'], 'up/photo-3');
    final uploads =
        panel.requests.where((r) => r.url.path.endsWith('/uploads')).toList();
    expect(uploads.length, 3);
    expect(uploads.first.headers['content-type'],
        startsWith('multipart/form-data'));
  });
}

Future<List<PickedFile>> _noFiles(
        {bool multiple = false, List<String> extensions = const []}) async =>
    const [];
