import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

void main() {
  testWidgets('the board moves a card to another column', (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final moves = <Map<String, dynamic>>[];
    final panel = FakePanel()
      ..board = true
      ..extraRoutes.addAll({
        'GET /resources/items/board': (_) => {
              'columns': [
                {
                  'value': 'draft',
                  'label': 'Draft',
                  'total': 1,
                  'cards': [
                    {'id': '7', 'title': 'Gold chain', 'version': '1'}
                  ]
                },
                {'value': 'live', 'label': 'Live', 'total': 0, 'cards': []},
              ]
            },
        'POST /resources/items/board/move': (r) {
          moves.add(Map<String, dynamic>.from(json.decode(r.body) as Map));
          return {'success': true, 'version': '2'};
        },
      });
    await tester.pumpWidget(GoFilamentAdmin(api: panel.api()));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);
    await tester.tap(find.byKey(const Key('list-board')));
    await tester.pumpAndSettle();
    expect(find.text('Gold chain'), findsOneWidget);

    final from = tester.getCenter(find.byKey(const Key('kanban-card-7')));
    final to = tester.getCenter(find.byKey(const Key('kanban-column-live')));
    final gesture = await tester.startGesture(from);
    await tester.pump(const Duration(milliseconds: 50));
    await gesture.moveTo(to);
    await tester.pump();
    await gesture.up();
    await tester.pumpAndSettle();
    expect(moves.single, {'id': '7', 'value': 'live', '_version': '1'});
    expect(find.text('Live  1'), findsOneWidget);
  });
}
