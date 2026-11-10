import 'dart:convert';

import 'package:archive/archive.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'fake_panel.dart';

Future<void> pumpApp(WidgetTester tester, FakePanel panel,
    {Size size = const Size(1400, 900),
    List<AdminPage> pages = const []}) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(GoFilamentAdmin(api: panel.api(), pages: pages));
  await tester.pumpAndSettle();
}

Future<void> signIn(WidgetTester tester) async {
  await tester.enterText(find.byKey(const Key('login-username')), 'owner');
  await tester.enterText(
      find.byKey(const Key('login-password')), 'secret-pass');
  await tester.tap(find.byKey(const Key('login-submit')));
  await tester.pumpAndSettle();
}

Future<void> openItems(WidgetTester tester) async {
  await tester.tap(find.byKey(const Key('nav-resource:items')));
  await tester.pumpAndSettle();
}

void main() {
  group('login', () {
    testWidgets('shows the brand from the public boot payload', (tester) async {
      final panel = FakePanel();
      await pumpApp(tester, panel);

      expect(find.text('Acme Stores'), findsOneWidget);
      expect(find.text('Retail made simple'), findsOneWidget);
      expect(
          find.text('Powered by GoFilament'), findsNothing); // hide_powered_by
      expect(panel.requests.first.url.path, '/app/api/boot');
      expect(
          panel.requests.first.headers.containsKey('Authorization'), isFalse);
    });

    testWidgets('validates, rejects wrong credentials, then signs in',
        (tester) async {
      final panel = FakePanel();
      await pumpApp(tester, panel);

      await tester.tap(find.byKey(const Key('login-submit')));
      await tester.pumpAndSettle();
      expect(find.text('Enter your username or email'), findsOneWidget);

      await tester.enterText(find.byKey(const Key('login-username')), 'owner');
      await tester.enterText(find.byKey(const Key('login-password')), 'wrong');
      await tester.tap(find.byKey(const Key('login-submit')));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('login-error')), findsOneWidget);
      expect(
          find.text('The username or password is incorrect.'), findsOneWidget);

      await signIn(tester);
      expect(find.text('Welcome back, Olivia'), findsOneWidget);
      expect(find.text('12,500.00'), findsOneWidget); // dashboard stat
    });

    testWidgets('signing out returns to the login screen', (tester) async {
      final panel = FakePanel();
      await pumpApp(tester, panel);
      await signIn(tester);

      await tester.tap(find.byKey(const Key('account-menu')));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Sign out'));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('login-submit')), findsOneWidget);
    });
  });

  group('shell', () {
    testWidgets('groups navigation and applies the server palette',
        (tester) async {
      final panel = FakePanel();
      await pumpApp(tester, panel);
      await signIn(tester);

      expect(find.text('CATALOGUE'), findsOneWidget);
      expect(find.text('Items'), findsOneWidget);
      final theme = Theme.of(tester.element(find.text('Items')));
      expect(theme.extension<AdminPalette>()!.color('primary', 600),
          const Color(0xFF4F46E5));
    });

    testWidgets(
        'custom pages appear in navigation; fullscreen pages take the window',
        (tester) async {
      final panel = FakePanel();
      await pumpApp(tester, panel, pages: [
        AdminPage(
            id: 'reports',
            label: 'Reports',
            group: 'Catalogue',
            builder: (_, __) => const Center(child: Text('REPORTS BODY'))),
        AdminPage(
          id: 'pos',
          label: 'Point of Sale',
          fullscreen: true,
          builder: (ctx, admin) =>
              Scaffold(body: Center(child: Text('POS for ${admin.brandName}'))),
        ),
      ]);
      await signIn(tester);

      await tester.tap(find.byKey(const Key('nav-page:reports')));
      await tester.pumpAndSettle();
      expect(find.text('REPORTS BODY'), findsOneWidget);
      expect(find.byKey(const Key('account-menu')), findsOneWidget);

      await tester.tap(find.byKey(const Key('nav-page:pos')));
      await tester.pumpAndSettle();
      expect(find.text('POS for Acme Stores'), findsOneWidget);
      expect(find.byKey(const Key('account-menu')).hitTestable(), findsNothing);
    });

    testWidgets('narrow screens use a drawer', (tester) async {
      final panel = FakePanel();
      await pumpApp(tester, panel, size: const Size(420, 800));
      await signIn(tester);

      expect(find.byKey(const Key('nav-resource:items')), findsNothing);
      await tester.tap(find.byTooltip('Menu'));
      await tester.pumpAndSettle();
      await openItems(tester);
      expect(find.text('Rings'), findsOneWidget); // card list
    });
  });

  group('resource list', () {
    testWidgets('renders typed cells, pages, sorts, filters and searches',
        (tester) async {
      final panel = FakePanel();
      await pumpApp(tester, panel);
      await signIn(tester);
      await openItems(tester);

      // per_page = 2 from the schema
      expect(find.text('Rings'), findsOneWidget);
      expect(find.text('Chains'), findsOneWidget);
      expect(find.text('Bangles'), findsNothing);
      expect(find.text('1–2 of 3'), findsOneWidget);
      expect(find.text('Rs. 1,500.50'), findsOneWidget); // money
      expect(find.text('live'), findsOneWidget); // badge
      expect(find.text('HIDDEN'), findsNothing); // hidden-by-default column

      await tester.tap(find.byKey(const Key('list-next')));
      await tester.pumpAndSettle();
      expect(find.text('Bangles'), findsOneWidget);
      expect(find.text('3–3 of 3'), findsOneWidget);

      await tester.tap(find.text('NAME'));
      await tester.pumpAndSettle();
      expect(panel.requests.last.url.queryParameters['sort'], 'name');
      expect(panel.requests.last.url.queryParameters['page'], '1');

      await tester.tap(find.byKey(const Key('filter-status')));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Draft').last);
      await tester.pumpAndSettle();
      expect(panel.requests.last.url.queryParameters['filter_status'], 'draft');
      expect(find.text('Status: Draft'), findsOneWidget);
      expect(find.text('1–1 of 1'), findsOneWidget);

      await tester.enterText(find.byKey(const Key('list-search')), 'zzz');
      await tester.pump(const Duration(milliseconds: 400));
      await tester.pumpAndSettle();
      expect(panel.requests.last.url.queryParameters['search'], 'zzz');
      expect(find.text('No records match your search.'), findsOneWidget);
    });

    testWidgets('inline toggle updates the cell through the API',
        (tester) async {
      final panel = FakePanel();
      await pumpApp(tester, panel);
      await signIn(tester);
      await openItems(tester);

      await tester.tap(find.byKey(const Key('cell-is_active-2')));
      await tester.pumpAndSettle();
      expect(panel.items[1]['is_active'], isTrue);
      expect(panel.requests.last.url.path, '/app/api/resources/items/2/cell');
    });

    testWidgets('deletes one record after confirmation and several in bulk',
        (tester) async {
      final panel = FakePanel();
      await pumpApp(tester, panel);
      await signIn(tester);
      await openItems(tester);

      await tester.tap(find.byKey(const Key('row-delete-1')));
      await tester.pumpAndSettle();
      expect(panel.items.length, 3); // nothing until confirmed
      await tester.tap(find.byKey(const Key('confirm-delete')));
      await tester.pumpAndSettle();
      expect(panel.items.map((i) => i['id']), [2, 3]);

      await tester.tap(find.byType(Checkbox).first); // select all on the page
      await tester.pumpAndSettle();
      expect(find.text('2 selected'), findsOneWidget);
      await tester.tap(find.byKey(const Key('bulk-delete')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('confirm-delete')));
      await tester.pumpAndSettle();
      expect(panel.items, isEmpty);
      expect(find.text('No records yet.'), findsOneWidget);
    });

    testWidgets('hides actions the user is not allowed to use', (tester) async {
      final panel = FakePanel()
        ..canCreate = false
        ..canDelete = false;
      await pumpApp(tester, panel);
      await signIn(tester);
      await openItems(tester);

      expect(find.byKey(const Key('list-create')), findsNothing);
      expect(find.byKey(const Key('row-delete-1')), findsNothing);
      expect(find.byKey(const Key('row-edit-1')), findsOneWidget);
    });
  });

  group('resource form', () {
    testWidgets('create: renders the schema, validates and posts typed values',
        (tester) async {
      final panel = FakePanel();
      await pumpApp(tester, panel);
      await signIn(tester);
      await openItems(tester);

      await tester.tap(find.byKey(const Key('list-create')));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('form-close')), findsOneWidget);
      expect(find.text('Basic details'), findsOneWidget);
      expect(find.text('Code *'), findsOneWidget);

      await tester.tap(find.byKey(const Key('form-save')));
      await tester.pumpAndSettle();
      expect(find.text('Code is required'), findsOneWidget);
      expect(panel.items.length, 3);

      await tester.enterText(
          find.byKey(const Key('field-code')), 'EARRING-TOO-LONG');
      await tester.enterText(find.byKey(const Key('field-name')), 'Earrings');
      await tester.enterText(find.byKey(const Key('field-price')), '99.5');

      // short local select -> dropdown
      await tester.tap(find.byKey(const Key('field-category_id')));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Jewellery').last);
      await tester.pumpAndSettle();

      // remote select -> searchable picker
      await tester.tap(find.byKey(const Key('field-supplier_id')));
      await tester.pumpAndSettle();
      await tester.enterText(find.byKey(const Key('picker-search')), 'bright');
      await tester.pump(const Duration(milliseconds: 350));
      await tester.pumpAndSettle();
      expect(find.text('Agasthi Traders'), findsNothing);
      await tester.tap(find.byKey(const Key('option-41')));
      await tester.pumpAndSettle();
      expect(find.text('Bright Metals'), findsOneWidget);

      await tester.tap(find.byKey(const Key('form-save')));
      await tester.pumpAndSettle();

      final created = panel.items.last;
      expect(created['code'], 'EARRING-TO'); // max_length 10 enforced
      expect(created['name'], 'Earrings');
      expect(created['price'], 99.5); // number, not string
      expect(created['category_id'], '2');
      expect(created['supplier_id'], '41');
      expect(created['is_active'], isTrue); // toggle default
      expect(
          find.byKey(const Key('form-close')), findsNothing); // dialog closed
    });

    testWidgets('edit: prefills the record, shows remote labels and saves',
        (tester) async {
      final panel = FakePanel();
      panel.items[0]['supplier_id'] = 40;
      await pumpApp(tester, panel);
      await signIn(tester);
      await openItems(tester);

      await tester.tap(find.byKey(const Key('row-edit-1')));
      await tester.pumpAndSettle();
      expect(find.text('Edit Item'), findsOneWidget);
      expect(
          tester
              .widget<TextFormField>(find.byKey(const Key('field-name')))
              .controller!
              .text,
          'Rings');
      expect(find.text('Agasthi Traders'),
          findsOneWidget); // label of the saved remote value
      expect(find.text('Jewellery'), findsOneWidget); // category 2

      await tester.enterText(find.byKey(const Key('field-name')), 'Gold Rings');
      await tester.tap(find.byKey(const Key('form-save')));
      await tester.pumpAndSettle();
      expect(panel.items[0]['name'], 'Gold Rings');
      expect(
          panel.requests.last.url.path.startsWith('/app/api/resources/items'),
          isTrue);
    });

    testWidgets(
        'shows the server message when saving fails and keeps the form open',
        (tester) async {
      final panel = FakePanel()
        ..failNextSaveWith = 'A record with the same code already exists';
      await pumpApp(tester, panel);
      await signIn(tester);
      await openItems(tester);

      await tester.tap(find.byKey(const Key('list-create')));
      await tester.pumpAndSettle();
      await tester.enterText(find.byKey(const Key('field-code')), 'RING');
      await tester.enterText(find.byKey(const Key('field-name')), 'Dup');
      await tester.tap(find.byKey(const Key('form-save')));
      await tester.pumpAndSettle();

      expect(find.byKey(const Key('form-error')), findsOneWidget);
      expect(find.text('A record with the same code already exists'),
          findsOneWidget);
      expect(find.byKey(const Key('form-close')), findsOneWidget);
    });

    testWidgets('view mode is read only', (tester) async {
      final panel = FakePanel();
      await pumpApp(tester, panel);
      await signIn(tester);
      await openItems(tester);

      await tester.tap(find.byKey(const Key('row-view-1')));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('form-save')), findsNothing);
      expect(
          tester
              .widget<TextFormField>(find.byKey(const Key('field-name')))
              .enabled,
          isFalse);
    });
  });

  group('deep links', () {
    testWidgets('a link to a record opens its form after sign-in',
        (tester) async {
      final panel = FakePanel();
      final admin = AdminController(api: panel.api())
        ..initialLocation = '/resources/items/2';
      tester.view.physicalSize = const Size(1400, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(GoFilamentAdmin(controller: admin));
      await tester.pumpAndSettle();
      await signIn(tester);

      expect(find.text('Edit Item'), findsOneWidget);
      expect(
          tester
              .widget<TextFormField>(find.byKey(const Key('field-name')))
              .controller!
              .text,
          'Chains');
      expect(admin.initialLocation, isEmpty);
    });

    testWidgets('a link to "new" opens the create form', (tester) async {
      final panel = FakePanel();
      final admin = AdminController(api: panel.api())
        ..initialLocation = '/resources/items/new';
      tester.view.physicalSize = const Size(1400, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(GoFilamentAdmin(controller: admin));
      await tester.pumpAndSettle();
      await signIn(tester);

      expect(find.byKey(const Key('form-save')), findsOneWidget);
      expect(find.text('Create'), findsOneWidget);
    });
  });

  group('session', () {
    testWidgets('an expired session sends the user back to sign in',
        (tester) async {
      final panel = FakePanel();
      final api = panel.api();
      tester.view.physicalSize = const Size(1400, 900);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      await tester.pumpWidget(GoFilamentAdmin(api: api));
      await tester.pumpAndSettle();
      await signIn(tester);

      api.token = 'revoked';
      await openItems(tester);
      expect(find.byKey(const Key('login-submit')), findsOneWidget);
    });
  });

  group('export', () {
    final export = TableExport(headers: [
      'Code',
      'Name',
      'Price'
    ], rows: [
      ['RING', 'Rings, "gold"', '1500.50'],
      ['X', '=HYPERLINK("http://evil")', '-5'],
    ]);

    test('CSV quotes fields and neutralises formulas', () {
      final csv = utf8.decode(export.toCsv().sublist(3));
      expect(export.toCsv().sublist(0, 3), [0xEF, 0xBB, 0xBF]);
      final lines = const LineSplitter().convert(csv);
      expect(lines[0], 'Code,Name,Price');
      expect(lines[1], 'RING,"Rings, ""gold""",1500.50');
      expect(lines[2].startsWith("X,\"'=HYPERLINK"), isTrue);
      expect(lines[2].endsWith("'-5"), isTrue);
    });

    test('XLSX is a valid workbook with numbers stored as numbers', () {
      final archive =
          ZipDecoder().decodeBytes(export.toXlsx(sheetName: 'Items'));
      final names = archive.files.map((f) => f.name).toSet();
      expect(
          names,
          containsAll([
            '[Content_Types].xml',
            'xl/workbook.xml',
            'xl/worksheets/sheet1.xml',
            'xl/styles.xml'
          ]));
      final sheet = utf8.decode(
          archive.findFile('xl/worksheets/sheet1.xml')!.content as List<int>);
      expect(sheet, contains('<c r="C2"><v>1500.50</v></c>'));
      expect(sheet, contains('Rings, &quot;gold&quot;'));
      expect(
          utf8.decode(
              archive.findFile('xl/workbook.xml')!.content as List<int>),
          contains('name="Items"'));
    });
  });

  group('helpers', () {
    test('palette parses colours and falls back to the nearest shade', () {
      final palette = AdminPalette.fromJson(FakePanel.colors);
      expect(palette.color('primary', 600), const Color(0xFF4F46E5));
      expect(palette.color('primary', 700), const Color(0xFF4F46E5)); // nearest
      expect(palette.color('unknown'), isA<Color>());
      expect(parseColor('#fff'), const Color(0xFFFFFFFF));
    });

    test('formatting', () {
      expect(formatNumber(1234567.5), '1,234,567.50');
      expect(formatNumber(-950), '-950.00');
      expect(formatMoney('250', 'Rs.'), 'Rs. 250.00');
      expect(humanize('created_at'), 'Created At');
      expect(displayValue(7.0), '7');
      expect(displayValue(true), 'Yes');
    });
  });
}
