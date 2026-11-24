// Apex Store admin: the GoFilament live demo (demo/main.go serves it at
// /admin with the panel API). Everything on screen comes from the panel; this
// file only adds the demo accounts and the "data resets" strip.
import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:gofilament_admin/gofilament_admin.dart';
import 'package:http/http.dart' as http;

void main() => runApp(const DemoApp());

/// Public demo accounts (also listed on the demo's start page).
const demoAccounts = [
  DemoLogin(
      email: 'admin@example.com',
      password: 'demo',
      label: 'Sign in as Admin',
      debugOnly: false,
      submit: true),
  DemoLogin(
      email: 'warehouse@example.com',
      password: 'demo',
      label: 'Sign in as Warehouse staff',
      debugOnly: false,
      submit: true),
];

class DemoApp extends StatelessWidget {
  const DemoApp({super.key});

  @override
  Widget build(BuildContext context) {
    return const Directionality(
      textDirection: TextDirection.ltr,
      child: Column(children: [
        Expanded(child: GoFilamentAdmin(demoLogins: demoAccounts)),
        ResetStrip(),
      ]),
    );
  }
}

/// A one-line strip under the app: this is a shared demo whose data resets.
class ResetStrip extends StatefulWidget {
  const ResetStrip({super.key});

  @override
  State<ResetStrip> createState() => _ResetStripState();
}

class _ResetStripState extends State<ResetStrip> {
  int? _minutes;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _load();
    _timer = Timer.periodic(const Duration(minutes: 1), (_) => _load());
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final res = await http.get(Uri.base.resolve('/demo/status'));
      final m = jsonDecode(res.body) as Map<String, dynamic>;
      if (mounted) setState(() => _minutes = m['next_reset_minutes'] as int?);
    } catch (_) {
      // the strip still shows without the countdown
    }
  }

  @override
  Widget build(BuildContext context) {
    final when = _minutes == null ? '' : ' (next reset in $_minutes min)';
    return Container(
      key: const Key('demo-reset-strip'),
      width: double.infinity,
      color: const Color(0xFF064E3B),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: Text(
        'Live demo: changes are shared and the sample data resets every hour$when',
        textAlign: TextAlign.center,
        style: const TextStyle(
            color: Color(0xFFECFDF5),
            fontSize: 12.5,
            decoration: TextDecoration.none),
      ),
    );
  }
}
