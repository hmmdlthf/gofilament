/// Translations of the app's own words (buttons, dialogs, messages). The
/// panel sends them for the user's language in boot ("ui_translations"),
/// keyed by the English text; anything missing stays English.
class AdminStrings {
  AdminStrings._();

  static Map<String, String> _table = const {};

  /// Replaces the current table (done by the controller after boot).
  static void use(Map<String, String> table) => _table = table;
}

/// Translates [english]; `{name}` placeholders are filled from [args].
String tr(String english, [Map<String, Object?> args = const {}]) {
  var s = AdminStrings._table[english] ?? english;
  args.forEach((k, v) => s = s.replaceAll('{$k}', '$v'));
  return s;
}
