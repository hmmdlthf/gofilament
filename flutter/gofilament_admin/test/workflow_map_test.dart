import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

/// A map like a home page: invoices lead to receipts, which lead to a
/// deposit in another block; the guide is an external link.
Map<String, dynamic> _map() => {
      'id': 'home',
      'label': 'Workflow',
      'blocks': [
        {
          'id': 'customers',
          'label': 'Customers',
          'col': 0,
          'row': 0,
          'cols': 3,
          'rows': 1
        },
        {
          'id': 'banking',
          'label': 'Banking',
          'col': 3,
          'row': 0,
          'cols': 1,
          'rows': 2
        },
        {
          'id': 'help',
          'label': 'Help',
          'col': 0,
          'row': 1,
          'cols': 3,
          'rows': 1
        },
      ],
      'tasks': [
        {
          'id': 'invoices',
          'block': 'customers',
          'label': 'Create invoices',
          'icon': 'document',
          'open': {
            'kind': 'resource',
            'slug': 'items',
            'action': 'create',
            'url': '/app/items/new'
          }
        },
        {
          'id': 'receipts',
          'block': 'customers',
          'label': 'Receive payments',
          'badge': '2',
          'open': {
            'kind': 'resource',
            'slug': 'items',
            'action': 'list',
            'url': '/app/items'
          }
        },
        {
          'id': 'deposits',
          'block': 'banking',
          'label': 'Record deposits',
          'open': {
            'kind': 'resource',
            'slug': 'items',
            'action': 'list',
            'url': '/app/items'
          }
        },
        {
          'id': 'guide',
          'block': 'help',
          'label': 'Guide',
          'open': {'kind': 'url', 'url': 'https://example.com/guide'}
        },
      ],
      'links': [
        {'from': 'invoices', 'to': 'receipts', 'kind': 'leads'},
        {'from': 'receipts', 'to': 'deposits', 'kind': 'leads'},
        {'from': 'guide', 'to': 'invoices', 'kind': 'feeds'},
      ],
    };

FakePanel _panel({bool dashboard = false}) {
  final panel = FakePanel()
    ..workflowMaps = [
      {
        'id': 'home',
        'label': 'Workflow',
        'icon': 'route',
        'group': '',
        'sort': 0,
        'on_dashboard': dashboard
      }
    ];
  panel.extraRoutes['GET /maps/home'] = (_) => _map();
  return panel;
}

Future<void> _pump(WidgetTester tester, FakePanel panel, Size size) async {
  tester.view.physicalSize = size;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.reset);
  await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
  await tester.pumpAndSettle();
  await signIn(tester);
}

void main() {
  testWidgets(
      'the map is in the menu, drawn with arrows, and tasks open their screens',
      (tester) async {
    final panel = _panel();
    final opened = <String>[];
    final opener = openLink;
    addTearDown(() => openLink = opener);
    openLink = (url) async {
      opened.add(url);
      return true;
    };
    await _pump(tester, panel, const Size(1400, 900));

    await tester.tap(find.byKey(const Key('nav-map:home')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('workflow-map-canvas')), findsOneWidget);
    expect(find.byKey(const Key('workflow-block-banking')), findsOneWidget);
    expect(find.text('Receive payments'), findsOneWidget);
    expect(find.text('2'), findsOneWidget, reason: 'the badge');

    // tiles sit inside their blocks; banking is to the right of customers
    final customers =
        tester.getRect(find.byKey(const Key('workflow-block-customers')));
    final banking =
        tester.getRect(find.byKey(const Key('workflow-block-banking')));
    final invoice =
        tester.getRect(find.byKey(const Key('workflow-task-invoices')));
    expect(customers.contains(invoice.center), isTrue);
    expect(banking.left, greaterThan(customers.right));

    // screen readers hear what comes next
    expect(find.bySemanticsLabel('Create invoices. Then: Receive payments'),
        findsOneWidget);

    // an external link opens outside the panel
    await tester.tap(find.byKey(const Key('workflow-task-guide')));
    await tester.pumpAndSettle();
    expect(opened, ['https://example.com/guide']);

    // a create task opens the resource's new form
    await tester.tap(find.byKey(const Key('workflow-task-invoices')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('workflow-map-canvas')), findsNothing);
    expect(find.byKey(const Key('nav-resource:items')), findsOneWidget);
  });

  testWidgets(
      'hovering a task highlights it and dims tasks it is not joined to',
      (tester) async {
    await _pump(tester, _panel(), const Size(1400, 900));
    await tester.tap(find.byKey(const Key('nav-map:home')));
    await tester.pumpAndSettle();

    double opacity(String id) => tester
        .widget<AnimatedOpacity>(find.ancestor(
            of: find.byKey(Key('workflow-task-$id')),
            matching: find.byType(AnimatedOpacity)))
        .opacity;
    final mouse = await tester.createGesture(kind: PointerDeviceKind.mouse);
    addTearDown(mouse.removePointer);
    await mouse.addPointer(location: Offset.zero);
    await mouse.moveTo(
        tester.getCenter(find.byKey(const Key('workflow-task-deposits'))));
    await tester.pumpAndSettle();
    expect(opacity('deposits'), 1);
    expect(opacity('receipts'), 1, reason: 'joined by an arrow');
    expect(opacity('guide'), lessThan(1));
  });

  testWidgets('on a phone each block is a list with Then chips',
      (tester) async {
    await _pump(tester, _panel(), const Size(390, 844));
    await tester.tap(find.byTooltip('Menu'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('nav-map:home')));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('workflow-map-list')), findsOneWidget);
    expect(find.byKey(const Key('workflow-map-canvas')), findsNothing);
    expect(find.byKey(const Key('workflow-next-invoices-receipts')),
        findsOneWidget);
    await tester.tap(find.byKey(const Key('workflow-next-receipts-deposits')));
    await tester.pumpAndSettle();
    expect(find.byKey(const Key('workflow-map-list')), findsNothing,
        reason: 'the chip opens the next task');
  });

  testWidgets('a map marked for the dashboard is shown there', (tester) async {
    await _pump(tester, _panel(dashboard: true), const Size(1400, 900));
    expect(find.byKey(const Key('workflow-map-card-home')), findsOneWidget);
    expect(find.text('Record deposits'), findsOneWidget);
  });

  testWidgets('a deep link opens the map', (tester) async {
    final panel = _panel();
    tester.view.physicalSize = const Size(1400, 900);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final admin = AdminController(api: panel.api())
      ..initialLocation = '/maps/home';
    await tester
        .pumpWidget(GoFilamentAdmin(api: panel.api(), controller: admin));
    await tester.pumpAndSettle();
    await signIn(tester);
    expect(find.byKey(const Key('workflow-map-canvas')), findsOneWidget);
  });
}
