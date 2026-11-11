import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'format.dart';
import 'theme.dart';

/// Renders one table cell according to its column type (text, money, date,
/// badge, toggle, icon, image, colour).
Widget adminCell(
  BuildContext context,
  Map<String, dynamic> column,
  Map<String, dynamic> row, {
  void Function(String column, bool value)? onToggle,
  void Function(String message)? onNotice,
  String currency = '',
}) {
  final theme = Theme.of(context);
  final palette = AdminPalette.of(context);
  final dark = theme.brightness == Brightness.dark;
  final name = '${column['name']}';
  final value = valueAt(row, name);

  switch ('${column['type']}') {
    case 'toggle':
      return Switch(
        key: Key('cell-$name-${recordId(row)}'),
        value: value == true,
        onChanged: column['editable'] == true && onToggle != null
            ? (v) => onToggle.call(name, v)
            : null,
      );
    case 'icon':
      if (value is bool || value == null) {
        final on = value == true;
        return Icon(on ? Icons.check_circle : Icons.remove_circle_outline,
            size: 18,
            color: on
                ? palette.color('success', dark ? 400 : 600)
                : theme.colorScheme.onSurfaceVariant);
      }
      return Text(displayValue(value));
    case 'badge':
      final colors = column['colors'];
      final options = column['options'];
      // booleans are keyed "true" / "false"; a missing false is common in
      // JSON (omitempty), so treat null as false when the badge expects it
      final bool expectsBool = (colors is Map && colors.containsKey('false')) ||
          (options is Map && options.containsKey('false'));
      final text = value is bool
          ? '$value'
          : (value == null && expectsBool ? 'false' : displayValue(value));
      if (text.isEmpty) return const SizedBox.shrink();
      final role = colors is Map ? '${colors[text] ?? 'gray'}' : 'gray';
      final label =
          options is Map && options[text] != null ? '${options[text]}' : text;
      final color = palette.color(role, dark ? 300 : 700);
      return Container(
        padding: const EdgeInsets.symmetric(horizontal: 9, vertical: 3),
        decoration: BoxDecoration(
          color: palette.color(role, 500).withValues(alpha: dark ? 0.2 : 0.12),
          borderRadius: BorderRadius.circular(999),
        ),
        child: Text(label,
            style: TextStyle(
                color: color, fontSize: 12, fontWeight: FontWeight.w600)),
      );
    case 'money':
      final code = '${column['currency'] ?? ''}';
      return Text(formatMoney(value, code.isEmpty ? currency : code),
          style: const TextStyle(fontFeatures: [FontFeature.tabularFigures()]));
    case 'datetime':
      final format = '${column['date_time_format'] ?? ''}';
      final dateOnly = format.isNotEmpty &&
          !format.contains('15') &&
          !format.contains('03') &&
          !format.contains('3:');
      return Text(dateOnly ? formatDateOnly(value) : formatDateTime(value));
    case 'image':
      final url = displayValue(value);
      if (url.isEmpty) return const SizedBox.shrink();
      return ClipRRect(
        borderRadius:
            BorderRadius.circular(column['circular'] == true ? 999 : 6),
        child: Image.network(url,
            width: 34,
            height: 34,
            fit: BoxFit.cover,
            errorBuilder: (_, __, ___) =>
                const Icon(Icons.image_not_supported_outlined, size: 18)),
      );
    case 'color':
      final c = parseColor(displayValue(value));
      return Row(mainAxisSize: MainAxisSize.min, children: [
        Container(
            width: 14,
            height: 14,
            decoration: BoxDecoration(
                color: c, borderRadius: BorderRadius.circular(4))),
        const SizedBox(width: 6),
        Text(displayValue(value)),
      ]);
  }

  var text = displayValue(value);
  final limit = (column['limit'] as num?)?.toInt() ?? 0;
  if (limit > 0 && text.length > limit) text = '${text.substring(0, limit)}…';
  final label = Text(
    text,
    maxLines: 1,
    overflow: TextOverflow.ellipsis,
    style: TextStyle(
        fontWeight: column['weight'] == 'bold' ? FontWeight.w600 : null),
  );
  if (column['copyable'] == true && text.isNotEmpty) {
    return InkWell(
      onTap: () {
        Clipboard.setData(ClipboardData(text: displayValue(value)));
        onNotice?.call('Copied');
      },
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        Flexible(child: label),
        const SizedBox(width: 4),
        Icon(Icons.copy, size: 13, color: theme.colorScheme.onSurfaceVariant),
      ]),
    );
  }
  return ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: 360), child: label);
}
