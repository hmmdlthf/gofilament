import 'api.dart';

/// Lenient JSON readers used by the code `flutterclient.GenerateFromPanel`
/// writes. They never throw on a value of the wrong shape: they return null,
/// so a server that sends `"12"` for a number or `12` for a string still
/// decodes.
abstract final class GfJson {
  static String? string(dynamic v) => switch (v) {
        null => null,
        String s => s,
        num() || bool() => '$v',
        _ => null,
      };

  static int? integer(dynamic v) => switch (v) {
        int i => i,
        num n => n.toInt(),
        String s => int.tryParse(s) ?? double.tryParse(s)?.toInt(),
        _ => null,
      };

  static double? decimal(dynamic v) => switch (v) {
        num n => n.toDouble(),
        String s => double.tryParse(s),
        _ => null,
      };

  static num? number(dynamic v) => switch (v) {
        num n => n,
        String s => num.tryParse(s),
        _ => null,
      };

  static bool? boolean(dynamic v) => switch (v) {
        bool b => b,
        num n => n != 0,
        String s => s == 'true' || s == '1'
            ? true
            : (s == 'false' || s == '0' || s == '' ? false : null),
        _ => null,
      };

  /// Reads RFC 3339 / ISO 8601 strings. Go's zero time reads as null.
  static DateTime? dateTime(dynamic v) {
    if (v is! String || v.isEmpty) return null;
    final t = DateTime.tryParse(v);
    if (t == null || t.year <= 1) return null;
    return t;
  }

  static Map<String, dynamic>? map(dynamic v) =>
      v is Map ? Map<String, dynamic>.from(v) : null;

  static List<T>? list<T>(dynamic v, T Function(dynamic e) item) =>
      v is List ? v.map(item).toList() : null;

  static T? object<T>(
          dynamic v, T Function(Map<String, dynamic> json) decode) =>
      v is Map ? decode(Map<String, dynamic>.from(v)) : null;

  /// A calendar date as `yyyy-mm-dd` (what date fields send).
  static String? date(DateTime? d) => d == null
      ? null
      : '${d.year.toString().padLeft(4, '0')}-'
          '${d.month.toString().padLeft(2, '0')}-'
          '${d.day.toString().padLeft(2, '0')}';

  /// Query parameters from a JSON object: nulls are dropped, lists are
  /// joined with commas.
  static Map<String, String> query(Map<String, dynamic>? json) => {
        if (json != null)
          for (final e in json.entries)
            if (e.value != null)
              e.key:
                  e.value is List ? (e.value as List).join(',') : '${e.value}',
      };

  /// [path] with [query] appended.
  static String withQuery(String path, Map<String, String> query) => query
          .isEmpty
      ? path
      : '$path${path.contains('?') ? '&' : '?'}${Uri(queryParameters: query).query}';
}

/// A page of typed records and the per-row abilities the server granted.
class TypedPage<T> {
  TypedPage(this.page, this.items);

  factory TypedPage.from(
          ListPage page, T Function(Map<String, dynamic> json) decode) =>
      TypedPage(page, page.rows.map(decode).toList());

  /// The untyped page as the API returned it.
  final ListPage page;
  final List<T> items;

  int get total => page.total;
  int get pageNumber => page.page;
  int get perPage => page.perPage;

  /// What the user may do with the record [id]:
  /// `{update: bool, delete: bool, actions: [names], links: {name: url}}`.
  Map<String, dynamic> rowMeta(String id) => page.rowMeta[id] ?? const {};

  /// Whether the server offers the row action [action] for record [id].
  bool canRun(String id, String action) {
    final actions = rowMeta(id)['actions'];
    return actions is List && actions.contains(action);
  }
}
