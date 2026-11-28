import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

void main() {
  tearDown(() {
    adminPickFiles = _noFiles;
    adminCropImage = _noCrop;
    adminTakePhoto = _noPhoto;
    openLink = _noLink;
  });

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

  Future<void> openEdit(WidgetTester tester, FakePanel panel,
      {Size size = const Size(1400, 2000)}) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    if (size.width < 600) {
      await tester.tap(find.byTooltip('Menu'));
      await tester.pumpAndSettle();
    }
    await openItems(tester);
    await tester.tap(find.byKey(const Key('row-edit-1')).first);
    await tester.pumpAndSettle();
  }

  testWidgets('stored files show their names and links from the server',
      (tester) async {
    final opened = <String>[];
    openLink = (url) async {
      opened.add(url);
      return true;
    };
    final panel = FakePanel()
      ..items.first['contract'] = 'docs/9f2a.pdf'
      ..formFiles = {
        'contract': [
          {
            'value': 'docs/9f2a.pdf',
            'name': 'NDA 2026.pdf',
            'url': '/storage/docs/9f2a.pdf?expires=1&signature=x'
          }
        ]
      }
      ..extraFormFields = [
        {
          'kind': 'field',
          'type': 'file',
          'name': 'contract',
          'label': 'Contract',
          'downloadable': true,
          'openable': true,
        },
      ];
    await openEdit(tester, panel);
    expect(find.text('NDA 2026.pdf'), findsOneWidget);
    await tester.ensureVisible(find.byKey(const Key('contract-download-0')));
    await tester.tap(find.byKey(const Key('contract-download-0')));
    await tester.tap(find.byKey(const Key('contract-open-0')));
    expect(opened, [
      '/storage/docs/9f2a.pdf?expires=1&signature=x&download=NDA+2026.pdf',
      '/storage/docs/9f2a.pdf?expires=1&signature=x',
    ]);
  });

  testWidgets('uploads can be cancelled, and failed ones retried',
      (tester) async {
    final panel = FakePanel()
      ..extraFormFields = [
        {
          'kind': 'field',
          'type': 'file',
          'name': 'manuals',
          'label': 'Manuals',
          'multiple': true,
        },
      ];
    final gate = Completer<void>();
    var calls = 0;
    panel.extraRoutes['POST /resources/items/uploads'] = (req) async {
      calls++;
      if (calls == 1) {
        await gate.future; // stays in flight until cancelled
        return {'value': 'never'};
      }
      if (calls == 2) throw Exception('network down');
      return {'value': 'tmp:abc/b.pdf', 'name': 'b.pdf'};
    };
    adminPickFiles =
        ({bool multiple = false, List<String> extensions = const []}) async => [
              (name: 'b.pdf', bytes: Uint8List.fromList([37, 80, 68, 70]))
            ];
    await openEdit(tester, panel);

    await tester.ensureVisible(find.byKey(const Key('manuals-pick')));
    await tester.tap(find.byKey(const Key('manuals-pick')));
    await tester.pump();
    await tester.pump();
    expect(find.byType(LinearProgressIndicator), findsOneWidget);
    await tester.tap(find.byKey(const Key('manuals-cancel-0')));
    await tester.pumpAndSettle();
    expect(find.text('b.pdf'), findsNothing,
        reason: 'cancelled upload removed');
    gate.complete();

    await tester.tap(find.byKey(const Key('manuals-pick')));
    await tester.pumpAndSettle();
    expect(find.textContaining('network down'), findsOneWidget);
    await tester.ensureVisible(find.byKey(const Key('form-save')));
    await tester.tap(find.byKey(const Key('form-save')));
    await tester.pumpAndSettle();
    expect(find.text('Remove or retry the files that failed'), findsOneWidget);

    await tester.ensureVisible(find.byKey(const Key('manuals-retry-0')));
    await tester.tap(find.byKey(const Key('manuals-retry-0')));
    await tester.pumpAndSettle();
    expect(find.textContaining('network down'), findsNothing);
    await tester.ensureVisible(find.byKey(const Key('form-save')));
    await tester.tap(find.byKey(const Key('form-save')));
    await tester.pumpAndSettle();
    expect(panel.putBodies.last['manuals'], ['tmp:abc/b.pdf']);
  });

  testWidgets('files over the size limit are refused before uploading',
      (tester) async {
    final panel = FakePanel()
      ..extraFormFields = [
        {
          'kind': 'field',
          'type': 'file',
          'name': 'manual',
          'label': 'Manual',
          'max_size': 1,
        },
      ];
    adminPickFiles = (
            {bool multiple = false,
            List<String> extensions = const []}) async =>
        [(name: 'big.pdf', bytes: Uint8List(4096))];
    await openEdit(tester, panel);
    await tester.ensureVisible(find.byKey(const Key('manual-pick')));
    await tester.tap(find.byKey(const Key('manual-pick')));
    await tester.pumpAndSettle();
    expect(find.text('Larger than 1 KB'), findsOneWidget);
    expect(find.byKey(const Key('manual-retry-0')), findsNothing);
    expect(
        panel.requests.where((r) => r.url.path.endsWith('/uploads')), isEmpty);
  });

  testWidgets('images are cropped to the field ratio before upload',
      (tester) async {
    final panel = FakePanel()
      ..extraFormFields = [
        {
          'kind': 'field',
          'type': 'image',
          'name': 'cover',
          'label': 'Cover',
          'crop_aspect_ratio': '16:9',
        },
      ];
    double? askedRatio;
    final png = Uint8List.fromList([0x89, 0x50, 0x4E, 0x47, 0, 0, 0, 1]);
    adminCropImage = (context, image, ratio) async {
      askedRatio = ratio;
      return png;
    };
    adminPickFiles =
        ({bool multiple = false, List<String> extensions = const []}) async => [
              (name: 'photo.jpg', bytes: Uint8List.fromList([0xFF, 0xD8, 1]))
            ];
    String? sentName;
    panel.extraRoutes['POST /resources/items/uploads'] = (req) {
      sentName = RegExp(r'filename="([^"]+)"')
          .firstMatch(latin1.decode(req.bodyBytes))
          ?.group(1);
      return {'value': 'tmp:k.png/photo.png', 'name': 'photo.png'};
    };
    await openEdit(tester, panel);
    await tester.ensureVisible(find.byKey(const Key('cover-pick')));
    await tester.tap(find.byKey(const Key('cover-pick')));
    await tester.pumpAndSettle();
    expect(askedRatio, closeTo(16 / 9, 0.001));
    expect(sentName, 'photo.png', reason: 'renamed to match the cropped PNG');
    expect(find.byKey(const Key('cover-file-0')), findsOneWidget);
  });

  testWidgets('phones can take a photo for an image field', (tester) async {
    final panel = FakePanel()
      ..extraFormFields = [
        {
          'kind': 'field',
          'type': 'image',
          'name': 'avatar',
          'label': 'Avatar',
          'avatar': true,
        },
      ];
    panel.extraRoutes['POST /resources/items/uploads'] =
        (req) => {'value': 'tmp:k.jpg/cam.jpg', 'name': 'cam.jpg'};
    adminCropImage = (context, image, ratio) async => image;
    adminTakePhoto = () async =>
        (name: 'cam.jpg', bytes: Uint8List.fromList([0xFF, 0xD8, 1]));
    await openEdit(tester, panel, size: const Size(390, 900));
    await tester.ensureVisible(find.byKey(const Key('avatar-camera')));
    await tester.tap(find.byKey(const Key('avatar-camera')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('avatar-file-remove-0')), findsOneWidget);
    expect(panel.requests.where((r) => r.url.path.endsWith('/uploads')),
        hasLength(1));
  });

  testWidgets('image cells show the link the server sent', (tester) async {
    final panel = FakePanel()
      ..extraColumns = [
        {'name': 'photo', 'label': 'Photo', 'type': 'image'}
      ]
      ..extraRowMeta = {
        '1': {
          'images': {'photo': 'https://cdn.test/photos/1.png'}
        }
      };
    panel.items.first['photo'] = 'photos/1.png';
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);
    final urls = tester
        .widgetList<Image>(find.byType(Image))
        .map((i) => i.image)
        .whereType<NetworkImage>()
        .map((n) => n.url);
    expect(urls, contains('https://cdn.test/photos/1.png'));
  });

  testWidgets('relation forms upload to the relation endpoint', (tester) async {
    final panel = FakePanel()
      ..extraVariantFields = [
        {
          'kind': 'field',
          'type': 'image',
          'name': 'swatch',
          'label': 'Swatch',
        },
      ];
    panel.extraRoutes['POST /resources/items/1/relations/variants/uploads'] =
        (req) => {'value': 'tmp:k.png/s.png', 'name': 's.png'};
    adminPickFiles =
        ({bool multiple = false, List<String> extensions = const []}) async => [
              (name: 's.png', bytes: Uint8List.fromList([0x89, 0x50, 1]))
            ];
    await openEdit(tester, panel);
    await tester.ensureVisible(find.byKey(const Key('rel-variants-add')));
    await tester.tap(find.byKey(const Key('rel-variants-add')));
    await tester.pumpAndSettle();
    await tester.enterText(find.byKey(const Key('field-sku')).last, 'RING-M');
    await tester.ensureVisible(find.byKey(const Key('swatch-pick')));
    await tester.tap(find.byKey(const Key('swatch-pick')));
    await tester.pumpAndSettle();
    expect(
        panel.requests.where(
            (r) => r.url.path.endsWith('/items/1/relations/variants/uploads')),
        hasLength(1));
    await tester.tap(find.byKey(const Key('form-save')).last);
    await tester.pumpAndSettle();
    expect(panel.variants.last['swatch'], 'tmp:k.png/s.png');
  });

  testWidgets('file fields work inside repeater rows', (tester) async {
    final panel = FakePanel()
      ..extraFormFields = [
        {
          'kind': 'repeater',
          'name': 'gallery',
          'label': 'Gallery',
          'children': [
            {'kind': 'field', 'type': 'text', 'name': 'caption'},
            {'kind': 'field', 'type': 'image', 'name': 'photo'},
          ],
        },
      ]
      ..formFiles = {
        'gallery.photo': [
          {'value': 'g/1.png', 'name': 'beach.png', 'url': '/s/g/1.png'}
        ]
      };
    panel.items.first['gallery'] = [
      {'caption': 'Beach', 'photo': 'g/1.png'}
    ];
    String? uploadedField;
    panel.extraRoutes['POST /resources/items/uploads'] = (req) {
      uploadedField = req.url.queryParameters['field'];
      return {'value': 'tmp:k.png/new.png', 'name': 'new.png'};
    };
    adminPickFiles =
        ({bool multiple = false, List<String> extensions = const []}) async => [
              (name: 'new.png', bytes: Uint8List.fromList([0x89, 0x50, 1]))
            ];
    await openEdit(tester, panel);
    // the stored row shows its picture from the server's links
    expect(find.byKey(const Key('gallery.0.photo-file-0')), findsOneWidget);

    // these sit under the form dialog's sticky footer in a test window
    tester
        .widget<ButtonStyleButton>(find.byKey(const Key('gallery-add')))
        .onPressed!();
    await tester.pumpAndSettle();
    tester
        .widget<ButtonStyleButton>(
            find.byKey(const Key('gallery.1.photo-pick')))
        .onPressed!();
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.byKey(const Key('form-save')));
    await tester.tap(find.byKey(const Key('form-save')));
    await tester.pumpAndSettle();
    expect(uploadedField, 'photo');
    expect(panel.requests.where((r) => r.url.path.endsWith('/uploads')),
        hasLength(1));
    final rows = panel.putBodies.last['gallery'] as List;
    expect(rows.map((r) => r['photo']), ['g/1.png', 'tmp:k.png/new.png']);
  });

  testWidgets('phone cards show the row picture next to the title',
      (tester) async {
    final panel = FakePanel()
      ..extraColumns = [
        {'name': 'photo', 'label': 'Photo', 'type': 'image'}
      ]
      ..extraRowMeta = {
        '1': {
          'images': {'photo': 'https://cdn.test/photos/1.png'}
        }
      };
    tester.view.physicalSize = const Size(390, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await tester.tap(find.byTooltip('Menu'));
    await tester.pumpAndSettle();
    await openItems(tester);
    final pic = find.byKey(const Key('card-picture-1'));
    expect(pic, findsOneWidget);
    final img = tester
        .widget<Image>(find.descendant(of: pic, matching: find.byType(Image)));
    expect((img.image as NetworkImage).url, 'https://cdn.test/photos/1.png');
    // rows without a picture keep a placeholder, and no image text is shown
    expect(find.byKey(const Key('card-picture-2')), findsOneWidget);
    expect(find.text('Photo: '), findsNothing);
  });
}

Future<List<PickedFile>> _noFiles(
        {bool multiple = false, List<String> extensions = const []}) async =>
    const [];

Future<Uint8List?> _noCrop(BuildContext c, Uint8List b, double r) async => b;

Future<PickedFile?> _noPhoto() async => null;

Future<bool> _noLink(String url) async => false;
