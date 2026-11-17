import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

import 'admin_test.dart' show signIn, openItems;
import 'fake_panel.dart';

void main() {
  testWidgets('export runs as a server job when the panel has jobs',
      (tester) async {
    tester.view.physicalSize = const Size(1400, 1000);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    jobPollInterval = const Duration(milliseconds: 10);
    final saved = <String, Uint8List>{};
    adminSaveFile = (name, bytes, mime) async {
      saved[name] = bytes;
      return true;
    };
    addTearDown(() => adminSaveFile = (a, b, c) async => false);
    var polls = 0;
    Map<String, String>? startQuery;
    final panel = FakePanel()
      ..backgroundJobs = true
      ..extraRoutes.addAll({
        'GET /resources/items/export': (r) {
          startQuery = r.url.queryParameters;
          return {'id': 'job_1', 'status': 'queued'};
        },
        'GET /jobs/job_1': (_) => ++polls < 3
            ? {'id': 'job_1', 'status': 'running', 'progress': 40}
            : {
                'id': 'job_1',
                'status': 'done',
                'total': 3,
                'file_name': 'items-export.csv'
              },
      });
    final api = panel.api();
    await tester.pumpWidget(GoFilamentAdmin(api: api));
    await tester.pumpAndSettle();
    await signIn(tester);
    await openItems(tester);
    // The download is a raw GET, answered by the panel's handler.
    panel.extraRoutes['GET /jobs/job_1/download'] = (_) => 'unused';
    await tester.tap(find.byKey(const Key('list-export')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Export CSV').last);
    for (var i = 0; i < 50 && saved.isEmpty; i++) {
      await tester.pump(const Duration(milliseconds: 20));
    }
    expect(startQuery?['background'], '1');
    expect(startQuery?['format'], 'csv');
    expect(polls, 3);
    expect(saved.keys.single, 'items-export.csv');
  });
}
