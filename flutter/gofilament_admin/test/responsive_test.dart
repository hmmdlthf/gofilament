import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

const _phone = Size(390, 844);
const _tablet = Size(820, 1180);
const _desktop = Size(1440, 900);

FakePanel _panel() => FakePanel()
  ..tenants = [
    {'key': 'acme', 'name': 'Acme'},
    {'key': 'globex', 'name': 'Globex'},
  ]
  ..itemsResourceExtras = {
    'groups': ['status']
  };

Future<void> _pump(WidgetTester tester, FakePanel panel, Size size) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
  await tester.pumpAndSettle();
  await signIn(tester);
}

Future<void> _openItems(WidgetTester tester) async {
  if (find
      .byKey(const Key('nav-resource:items'))
      .hitTestable()
      .evaluate()
      .isEmpty) {
    await tester.tap(find.byTooltip('Menu'));
    await tester.pumpAndSettle();
  }
  await tester.tap(find.byKey(const Key('nav-resource:items')));
  await tester.pumpAndSettle();
}

double _topBarHeight(WidgetTester tester) => tester
    .getSize(find
        .ancestor(
            of: find.byKey(const Key('topbar-title')),
            matching: find.byType(Container))
        .first)
    .height;

void main() {
  test('form factors follow the window width', () {
    expect(AdminFormFactor.forWidth(390), AdminFormFactor.phone);
    expect(AdminFormFactor.forWidth(599), AdminFormFactor.phone);
    expect(AdminFormFactor.forWidth(600), AdminFormFactor.tablet);
    expect(AdminFormFactor.forWidth(1023), AdminFormFactor.tablet);
    expect(AdminFormFactor.forWidth(1024), AdminFormFactor.desktop);
  });

  group('phone', () {
    testWidgets('slim top bar; the avatar opens the account hub as a page',
        (tester) async {
      await _pump(tester, _panel(), _phone);
      expect(_topBarHeight(tester), 56);
      // tenant lives in the hub on a phone
      expect(find.byKey(const Key('tenant-menu')), findsNothing);

      await tester.tap(find.byKey(const Key('account-menu')));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('account-hub')), findsOneWidget);
      expect(find.byType(Dialog), findsNothing);
      expect(tester.getSize(find.byKey(const Key('account-hub'))).width,
          _phone.width);
      expect(find.byKey(const Key('hub-tenant-globex')), findsOneWidget);
      expect(find.byKey(const Key('hub-theme')), findsOneWidget);

      // pages open over the hub, back returns to it
      await tester.tap(find.byKey(const Key('account-menu-settings')));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('settings-theme')), findsOneWidget);
      await tester.pageBack();
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('account-hub')), findsOneWidget);
    });

    testWidgets('list: compact search, actions under ⋮, floating New',
        (tester) async {
      final panel = _panel();
      await _pump(tester, panel, _phone);
      await _openItems(tester);

      final search = tester.getSize(find.byKey(const Key('list-search')));
      expect(search.height, 40);
      expect(search.width, greaterThan(_phone.width * 0.6));
      expect(find.byKey(const Key('list-refresh')), findsNothing);
      expect(find.byKey(const Key('list-group')), findsNothing);
      expect(find.byType(FloatingActionButton), findsOneWidget);

      // an action with its own menu opens it from ⋮
      await tester.tap(find.byKey(const Key('list-more')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('list-more-group')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('group-status')));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('group-card-Status: draft (1)')),
          findsOneWidget);
    });

    testWidgets('signing out asks first', (tester) async {
      await _pump(tester, _panel(), _phone);
      await tester.tap(find.byKey(const Key('account-menu')));
      await tester.pumpAndSettle();
      await tester.scrollUntilVisible(
          find.byKey(const Key('account-menu-logout')), 200);
      await tester.tap(find.byKey(const Key('account-menu-logout')));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('account-hub')), findsOneWidget);

      await tester.tap(find.byKey(const Key('account-menu-logout')));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('confirm-sign-out')));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('login-submit')), findsOneWidget);
    });
  });

  group('tablet', () {
    testWidgets('the account hub is a dialog; the tenant is an icon',
        (tester) async {
      await _pump(tester, _panel(), _tablet);
      expect(_topBarHeight(tester), 60);
      expect(find.byKey(const Key('tenant-menu')), findsOneWidget);
      expect(find.byKey(const Key('tenant-current')), findsNothing);

      await tester.tap(find.byKey(const Key('account-menu')));
      await tester.pumpAndSettle();
      expect(find.byType(Dialog), findsOneWidget);
      expect(tester.getSize(find.byKey(const Key('account-hub'))).width,
          lessThanOrEqualTo(440));
      await tester.tap(find.byType(CloseButton));
      await tester.pumpAndSettle();
      expect(find.byKey(const Key('account-hub')), findsNothing);
    });

    testWidgets('list keeps a New button and fits its actions', (tester) async {
      await _pump(tester, _panel(), _tablet);
      await _openItems(tester);
      expect(find.byType(FloatingActionButton), findsNothing);
      expect(find.byKey(const Key('list-create')), findsOneWidget);
      expect(tester.getSize(find.byKey(const Key('list-search'))).height, 40);
      expect(find.byKey(const Key('list-group')), findsOneWidget);
    });
  });

  group('desktop', () {
    testWidgets('sidebar, theme toggle and full tenant menu', (tester) async {
      await _pump(tester, _panel(), _desktop);
      expect(find.byKey(const Key('sidebar-expanded')), findsOneWidget);
      expect(find.byKey(const Key('tenant-current')), findsOneWidget);
      expect(find.byTooltip('Dark mode'), findsOneWidget);
      await _openItems(tester);
      expect(find.byKey(const Key('list-more')), findsNothing);
      expect(find.byKey(const Key('list-refresh')), findsOneWidget);
    });
  });

  testWidgets('filter dialogs become bottom sheets on a phone', (tester) async {
    final panel = _panel()
      ..extraFilters = [
        {'name': 'price', 'label': 'Price', 'type': 'number_range'},
      ];
    await _pump(tester, panel, _phone);
    await _openItems(tester);
    await tester.tap(find.byKey(const Key('filter-price')));
    await tester.pumpAndSettle();
    expect(find.byType(BottomSheet), findsOneWidget);
    expect(find.byType(AlertDialog), findsNothing);
    expect(find.byKey(const Key('range-from')), findsOneWidget);
  });
}
