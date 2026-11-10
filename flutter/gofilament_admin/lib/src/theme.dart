import 'package:flutter/material.dart';

/// The panel's colour palettes (primary, gray, success, danger, warning, info,
/// accent, ...), each with the shades 50..950, as sent by the server.
@immutable
class AdminPalette extends ThemeExtension<AdminPalette> {
  const AdminPalette(this.roles);

  final Map<String, Map<int, Color>> roles;

  static const _fallback = <String, int>{
    'primary': 0xFF4F46E5,
    'accent': 0xFF7C3AED,
    'success': 0xFF059669,
    'danger': 0xFFE11D48,
    'warning': 0xFFD97706,
    'info': 0xFF2563EB,
    'gray': 0xFF64748B,
  };

  /// Parses the `theme.colors` object of the boot / schema payload.
  factory AdminPalette.fromJson(dynamic json) {
    final roles = <String, Map<int, Color>>{};
    if (json is Map) {
      json.forEach((role, shades) {
        if (shades is! Map) return;
        final parsed = <int, Color>{};
        shades.forEach((shade, hex) {
          final s = int.tryParse('$shade');
          final c = parseColor('$hex');
          if (s != null && c != null) parsed[s] = c;
        });
        if (parsed.isNotEmpty) roles['$role'] = parsed;
      });
    }
    return AdminPalette(roles);
  }

  /// The colour of [role] at [shade], falling back to the nearest shade and
  /// then to a built-in default so widgets never crash on a missing palette.
  Color color(String role, [int shade = 600]) {
    final shades = roles[role] ?? roles[_alias(role)];
    if (shades != null && shades.isNotEmpty) {
      if (shades.containsKey(shade)) return shades[shade]!;
      final keys = shades.keys.toList()
        ..sort((a, b) => (a - shade).abs().compareTo((b - shade).abs()));
      return shades[keys.first]!;
    }
    return Color(
        _fallback[role] ?? _fallback[_alias(role)] ?? _fallback['gray']!);
  }

  static String _alias(String role) {
    switch (role) {
      case 'secondary':
        return 'gray';
      case 'error':
        return 'danger';
      default:
        return role;
    }
  }

  @override
  AdminPalette copyWith({Map<String, Map<int, Color>>? roles}) =>
      AdminPalette(roles ?? this.roles);

  @override
  AdminPalette lerp(ThemeExtension<AdminPalette>? other, double t) =>
      t < 0.5 ? this : (other as AdminPalette? ?? this);

  static AdminPalette of(BuildContext context) =>
      Theme.of(context).extension<AdminPalette>() ?? const AdminPalette({});
}

/// Parses "#RRGGBB" / "#RRGGBBAA" / "RRGGBB".
Color? parseColor(String hex) {
  var h = hex.trim().replaceFirst('#', '');
  if (h.length == 3) h = h.split('').map((c) => '$c$c').join();
  if (h.length == 6) h = 'FF$h';
  if (h.length == 8 && hex.trim().replaceFirst('#', '').length == 8) {
    // CSS order is RRGGBBAA, Flutter wants AARRGGBB
    h = h.substring(6) + h.substring(0, 6);
  }
  final v = int.tryParse(h, radix: 16);
  return v == null || h.length != 8 ? null : Color(v);
}

/// Builds the Material theme from the panel palettes.
ThemeData buildAdminTheme(AdminPalette palette, Brightness brightness,
    {String? fontFamily}) {
  final dark = brightness == Brightness.dark;
  final primary = palette.color('primary', dark ? 500 : 600);
  final accent = palette.color('accent', dark ? 500 : 600);

  final background =
      dark ? palette.color('gray', 950) : palette.color('gray', 50);
  final surface = dark ? palette.color('gray', 900) : Colors.white;
  final outline =
      dark ? palette.color('gray', 700) : palette.color('gray', 200);
  final text = dark ? palette.color('gray', 50) : palette.color('gray', 900);
  final muted = dark ? palette.color('gray', 400) : palette.color('gray', 500);

  final scheme = ColorScheme.fromSeed(
          seedColor: palette.color('primary', 600), brightness: brightness)
      .copyWith(
    primary: primary,
    onPrimary: Colors.white,
    secondary: accent,
    onSecondary: Colors.white,
    error: palette.color('danger', dark ? 400 : 600),
    surface: surface,
    onSurface: text,
    onSurfaceVariant: muted,
    outline: outline,
    outlineVariant: outline,
    surfaceContainerLowest: surface,
    surfaceContainerLow:
        dark ? palette.color('gray', 900) : palette.color('gray', 50),
    surfaceContainer:
        dark ? palette.color('gray', 800) : palette.color('gray', 100),
    surfaceContainerHigh:
        dark ? palette.color('gray', 800) : palette.color('gray', 100),
    surfaceContainerHighest:
        dark ? palette.color('gray', 700) : palette.color('gray', 200),
  );

  final radius = BorderRadius.circular(10);
  return ThemeData(
    useMaterial3: true,
    brightness: brightness,
    colorScheme: scheme,
    fontFamily: (fontFamily == null || fontFamily.isEmpty) ? null : fontFamily,
    scaffoldBackgroundColor: background,
    dividerColor: outline,
    dividerTheme: DividerThemeData(color: outline, thickness: 1, space: 1),
    extensions: [palette],
    appBarTheme: AppBarTheme(
      backgroundColor: surface,
      foregroundColor: text,
      elevation: 0,
      scrolledUnderElevation: 0,
      surfaceTintColor: Colors.transparent,
    ),
    cardTheme: CardThemeData(
      color: surface,
      elevation: 0,
      margin: EdgeInsets.zero,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(14),
          side: BorderSide(color: outline)),
    ),
    inputDecorationTheme: InputDecorationTheme(
      isDense: true,
      filled: true,
      fillColor: dark ? palette.color('gray', 950) : Colors.white,
      contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 14),
      border: OutlineInputBorder(
          borderRadius: radius, borderSide: BorderSide(color: outline)),
      enabledBorder: OutlineInputBorder(
          borderRadius: radius, borderSide: BorderSide(color: outline)),
      focusedBorder: OutlineInputBorder(
          borderRadius: radius,
          borderSide: BorderSide(color: primary, width: 1.6)),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        minimumSize: const Size(0, 44),
        shape: RoundedRectangleBorder(borderRadius: radius),
        textStyle: const TextStyle(fontWeight: FontWeight.w600),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        minimumSize: const Size(0, 44),
        shape: RoundedRectangleBorder(borderRadius: radius),
        side: BorderSide(color: outline),
      ),
    ),
    dialogTheme: DialogThemeData(
      backgroundColor: surface,
      surfaceTintColor: Colors.transparent,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
    ),
    dataTableTheme: DataTableThemeData(
      headingTextStyle: TextStyle(
          fontWeight: FontWeight.w600,
          fontSize: 12,
          color: muted,
          letterSpacing: 0.4),
      dataTextStyle: TextStyle(fontSize: 13.5, color: text),
      headingRowHeight: 44,
      dataRowMinHeight: 48,
      dataRowMaxHeight: 56,
      dividerThickness: 1,
    ),
    snackBarTheme: const SnackBarThemeData(behavior: SnackBarBehavior.floating),
  );
}
