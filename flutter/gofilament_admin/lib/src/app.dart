import 'package:flutter/material.dart';

import 'api.dart';
import 'controller.dart';
import 'login.dart';
import 'shell.dart';
import 'theme.dart';

/// A complete admin application for a GoFilament panel.
///
/// ```dart
/// void main() => runApp(GoFilamentAdmin());
/// ```
///
/// Everything shown (brand, colours, navigation, lists, forms, permissions)
/// comes from the panel API. Applications extend it with [pages].
class GoFilamentAdmin extends StatefulWidget {
  const GoFilamentAdmin(
      {super.key, this.api, this.pages = const [], this.controller});

  /// API client; defaults to the panel the page was served from.
  final AdminApi? api;

  /// Extra application pages (for example a point of sale screen).
  final List<AdminPage> pages;

  /// Supply your own controller to drive the app from outside (tests, embedding).
  final AdminController? controller;

  @override
  State<GoFilamentAdmin> createState() => _GoFilamentAdminState();
}

class _GoFilamentAdminState extends State<GoFilamentAdmin> {
  late final AdminController _admin = widget.controller ??
      AdminController(api: widget.api ?? AdminApi(), pages: widget.pages);

  @override
  void initState() {
    super.initState();
    _admin.start();
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: _admin,
      builder: (context, _) => MaterialApp(
        title: _admin.brandName,
        debugShowCheckedModeBanner: false,
        themeMode: _admin.themeMode,
        theme: buildAdminTheme(_admin.palette, Brightness.light,
            fontFamily: _admin.fontFamily),
        darkTheme: buildAdminTheme(_admin.palette, Brightness.dark,
            fontFamily: _admin.fontFamily),
        home: switch (_admin.stage) {
          AdminStage.loading => const _Splash(),
          AdminStage.failed => _Failure(
              message: _admin.error ?? 'Something went wrong',
              onRetry: _admin.start),
          AdminStage.signedOut => LoginScreen(admin: _admin),
          AdminStage.signedIn => AdminShell(admin: _admin),
        },
      ),
    );
  }
}

class _Splash extends StatelessWidget {
  const _Splash();

  @override
  Widget build(BuildContext context) =>
      const Scaffold(body: Center(child: CircularProgressIndicator()));
}

class _Failure extends StatelessWidget {
  const _Failure({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.cloud_off_outlined,
                    size: 48,
                    color: Theme.of(context).colorScheme.onSurfaceVariant),
                const SizedBox(height: 16),
                Text(message, textAlign: TextAlign.center),
                const SizedBox(height: 20),
                FilledButton.icon(
                    onPressed: onRetry,
                    icon: const Icon(Icons.refresh),
                    label: const Text('Try again')),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
