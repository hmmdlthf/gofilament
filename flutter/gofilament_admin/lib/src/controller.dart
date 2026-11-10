import 'package:flutter/material.dart';

import 'api.dart';
import 'theme.dart';

/// Where the app is in its start-up sequence.
enum AdminStage { loading, signedOut, signedIn, failed }

/// A custom page an application adds to the panel (for example a POS screen).
class AdminPage {
  const AdminPage({
    required this.id,
    required this.label,
    required this.builder,
    this.icon = Icons.widgets_outlined,
    this.group = '',
    this.fullscreen = false,
    this.visible,
  });

  /// Unique id, also used in navigation state.
  final String id;
  final String label;
  final IconData icon;

  /// Navigation group the page appears in ('' = top, with the dashboard).
  final String group;

  /// Fullscreen pages take the whole window: no sidebar and no top bar.
  final bool fullscreen;

  /// Hide the page for some users (receives the signed-in user payload).
  final bool Function(Map<String, dynamic> user)? visible;

  final Widget Function(BuildContext context, AdminController admin) builder;
}

/// Holds the panel state: brand, theme, session and schema.
class AdminController extends ChangeNotifier {
  AdminController({required this.api, this.pages = const []}) {
    api.onUnauthorized = _sessionRejected;
  }

  final AdminApi api;
  final List<AdminPage> pages;

  /// The URL fragment the app was opened with (for example
  /// `/resources/products/12`). Captured at start-up, before sign-in, so a deep
  /// link survives the login screen.
  String initialLocation = Uri.base.fragment;

  AdminStage stage = AdminStage.loading;
  String? error;

  Map<String, dynamic> boot = const {};
  Map<String, dynamic> schema = const {};
  AdminPalette palette = const AdminPalette({});
  ThemeMode themeMode = ThemeMode.system;

  // ------------------------------------------------------------ derived

  Map<String, dynamic> get brand {
    final b = (schema['brand'] ?? boot['brand']);
    return b is Map ? Map<String, dynamic>.from(b) : const {};
  }

  String get brandName =>
      '${brand['name'] ?? schema['brand_name'] ?? boot['brand_name'] ?? 'Admin'}';

  String? get fontFamily {
    final t = schema['theme'] ?? boot['theme'];
    final f = t is Map ? t['font_family'] : null;
    return f is String && f.isNotEmpty ? f : null;
  }

  Map<String, dynamic> get user => api.user ?? const {};

  List<Map<String, dynamic>> get resources => _maps(schema['resources']);

  Map<String, dynamic>? resource(String slug) {
    for (final r in resources) {
      if (r['slug'] == slug) return r;
    }
    return null;
  }

  Map<String, dynamic> resourceMeta(String slug) {
    final all = schema['resource_meta'];
    final m = all is Map ? all[slug] : null;
    return m is Map ? Map<String, dynamic>.from(m) : const {};
  }

  bool can(String slug, String ability) {
    final a = resourceMeta(slug)['abilities'];
    return a is Map && a[ability] == true;
  }

  static List<Map<String, dynamic>> _maps(dynamic v) => v is List
      ? v.whereType<Map>().map((m) => Map<String, dynamic>.from(m)).toList()
      : const [];

  // ------------------------------------------------------------ lifecycle

  /// Loads the brand, restores a stored session and, when signed in, the schema.
  Future<void> start() async {
    stage = AdminStage.loading;
    error = null;
    notifyListeners();
    try {
      final saved = await api.readPreference('theme');
      if (saved == 'dark') themeMode = ThemeMode.dark;
      if (saved == 'light') themeMode = ThemeMode.light;

      boot = await api.boot();
      _applyTheme(boot);

      if (await api.restore()) {
        await _loadSchema();
        return;
      }
      stage = AdminStage.signedOut;
    } on ApiException catch (e) {
      error = e.message;
      stage = AdminStage.failed;
    } catch (_) {
      error =
          'Could not reach the server. Check your connection and try again.';
      stage = AdminStage.failed;
    }
    notifyListeners();
  }

  Future<void> _loadSchema() async {
    try {
      schema = await api.schema();
      _applyTheme(schema);
      stage = AdminStage.signedIn;
    } on ApiException catch (e) {
      if (e.isUnauthorized) {
        stage = AdminStage.signedOut;
      } else {
        error = e.message;
        stage = AdminStage.failed;
      }
    }
    notifyListeners();
  }

  void _applyTheme(Map<String, dynamic> source) {
    final theme = source['theme'];
    if (theme is Map && theme['colors'] is Map) {
      palette = AdminPalette.fromJson(theme['colors']);
    }
    final forced = brand['theme_mode'];
    if (forced == 'dark') themeMode = ThemeMode.dark;
    if (forced == 'light') themeMode = ThemeMode.light;
  }

  /// Signs in. Throws [ApiException] with a message for the login form.
  Future<void> login(String username, String password,
      {bool remember = false}) async {
    await api.login(username, password, remember: remember);
    await _loadSchema();
  }

  Future<void> logout() async {
    await api.logout();
    schema = const {};
    stage = AdminStage.signedOut;
    notifyListeners();
  }

  /// Reloads navigation, permissions and dashboard after something changed.
  Future<void> refresh() => _loadSchema();

  void _sessionRejected() {
    schema = const {};
    stage = AdminStage.signedOut;
    notifyListeners();
  }

  void toggleTheme(Brightness current) {
    themeMode = current == Brightness.dark ? ThemeMode.light : ThemeMode.dark;
    api.writePreference(
        'theme', themeMode == ThemeMode.dark ? 'dark' : 'light');
    notifyListeners();
  }
}
