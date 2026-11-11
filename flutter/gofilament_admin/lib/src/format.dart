/// Formatting helpers shared by lists, forms and exports.
library;

/// "created_at" -> "Created At".
String humanize(String name) {
  final words = name.replaceAll('-', '_').split('_').where((w) => w.isNotEmpty);
  return words.map((w) => w[0].toUpperCase() + w.substring(1)).join(' ');
}

/// 1234567.5 -> "1,234,567.50".
String formatNumber(num value, {int decimals = 2}) {
  final fixed = value.abs().toStringAsFixed(decimals);
  final parts = fixed.split('.');
  final digits = parts[0];
  final buf = StringBuffer();
  for (var i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 == 0) buf.write(',');
    buf.write(digits[i]);
  }
  final sign = value < 0 ? '-' : '';
  return decimals > 0 ? '$sign$buf.${parts[1]}' : '$sign$buf';
}

String formatMoney(dynamic value, [String currency = '']) {
  final n = value is num ? value : num.tryParse('$value');
  if (n == null) return '';
  return currency.isEmpty ? formatNumber(n) : '$currency ${formatNumber(n)}';
}

String _two(int n) => n < 10 ? '0$n' : '$n';

/// ISO timestamp -> "2026-11-10 14:05" in local time. Non-dates pass through.
String formatDateTime(dynamic value, {bool withTime = true}) {
  if (value == null) return '';
  final dt = DateTime.tryParse('$value');
  if (dt == null) return '$value';
  final l = dt.toLocal();
  final date = '${l.year}-${_two(l.month)}-${_two(l.day)}';
  return withTime ? '$date ${_two(l.hour)}:${_two(l.minute)}' : date;
}

String formatDate(DateTime d) => '${d.year}-${_two(d.month)}-${_two(d.day)}';

/// Display text of any cell value.
String displayValue(dynamic value) {
  if (value == null) return '';
  if (value is bool) return value ? 'Yes' : 'No';
  if (value is double && value == value.roundToDouble()) {
    return value.toInt().toString();
  }
  if (value is List) return value.map(displayValue).join(', ');
  return '$value';
}

/// Reads a record's id as a string.
String recordId(Map<String, dynamic> row) =>
    displayValue(row['id'] ?? row['ID'] ?? row['uuid']);

/// Reads a column value from a record. Dotted names walk into nested objects,
/// so a column named `edges.category.name` shows a related record's field.
dynamic valueAt(Map<String, dynamic> row, String name) {
  if (row.containsKey(name) || !name.contains('.')) return row[name];
  dynamic current = row;
  for (final part in name.split('.')) {
    if (current is Map && current.containsKey(part)) {
      current = current[part];
    } else {
      return null;
    }
  }
  return current;
}

/// Calendar date of a value without timezone conversion ("2026-11-09T00:00:00Z"
/// stays 2026-11-09 everywhere), for date-only columns.
String formatDateOnly(dynamic value) {
  if (value == null) return '';
  final text = '$value';
  final match = RegExp(r'^(\d{4}-\d{2}-\d{2})').firstMatch(text);
  return match?.group(1) ?? text;
}
