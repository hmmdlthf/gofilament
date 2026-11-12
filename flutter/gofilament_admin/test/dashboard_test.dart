import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn;
import 'fake_panel.dart';

void main() {
  testWidgets('chart widgets draw and reload when a period is chosen',
      (tester) async {
    final panel = FakePanel()
      ..extraWidgets = [
        {
          'id': 'revenue',
          'type': 'chart',
          'heading': 'Revenue',
          'chart_type': 'bar',
          'filters': {'week': 'This week', 'month': 'This month'},
          'default_filter': 'week',
          'data': {
            'labels': ['Mon', 'Tue'],
            'datasets': [
              {
                'label': 'Sales',
                'data': [10, 20]
              }
            ],
          },
        },
        {
          'id': 'mix',
          'type': 'chart',
          'heading': 'Mix',
          'chart_type': 'doughnut',
          'data': {
            'labels': ['A', 'B'],
            'datasets': [
              {
                'label': 'Share',
                'data': [1, 3]
              }
            ],
          },
        },
      ]
      ..extraRoutes['GET /widgets/revenue/data'] = (req) => {
            'labels': ['W1', 'W2', 'W3', 'W4'],
            'datasets': [
              {
                'label': req.url.queryParameters['filter'],
                'data': [1, 2, 3, 4]
              },
              {
                'label': 'Returns',
                'data': [0, 1, 0, 1]
              },
            ],
          };
    tester.view.physicalSize = const Size(1400, 1800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);

    expect(find.byKey(const Key('chart-revenue')), findsOneWidget);
    expect(find.text('Revenue'), findsOneWidget);
    expect(find.text('B'), findsOneWidget); // doughnut legend

    await tester.tap(find.byKey(const Key('chart-filter-month')));
    await tester.pumpAndSettle();
    expect(find.text('month'), findsOneWidget); // new dataset label in legend
    expect(find.text('Returns'), findsOneWidget);
    expect(
        panel.requests.any((r) =>
            r.url.path == '/app/api/widgets/revenue/data' &&
            r.url.queryParameters['filter'] == 'month'),
        isTrue);
  });
}
