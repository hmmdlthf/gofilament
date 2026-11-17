import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';

// Refined filters, advanced search and resizable columns for the resource
// list.

/// Shows the checkbox list of a refined filter (several values, with the
/// server's live counts). Returns the chosen values, or null when cancelled.
Future<Set<String>?> showRefinedFilterDialog(
  BuildContext context, {
  required String label,
  required List<Map<String, dynamic>> options,
  required Set<String> selected,
  bool multiple = true,
}) {
  final chosen = {...selected};
  return showDialog<Set<String>>(
    context: context,
    builder: (context) => StatefulBuilder(
      builder: (context, setState) => AlertDialog(
        title: Text(label),
        content: SizedBox(
          width: 360,
          child: options.isEmpty
              ? const Text('No options.')
              : ListView(shrinkWrap: true, children: [
                  for (final o in options)
                    CheckboxListTile(
                      key: Key('refined-option-${o['value']}'),
                      dense: true,
                      controlAffinity: ListTileControlAffinity.leading,
                      value: chosen.contains('${o['value']}'),
                      title: Text('${o['label'] ?? o['value']}'),
                      secondary: (o['count'] as num?) != null &&
                              (o['count'] as num) > 0
                          ? Text('${o['count']}',
                              style: Theme.of(context).textTheme.bodySmall)
                          : null,
                      onChanged: (v) => setState(() {
                        final value = '${o['value']}';
                        if (!multiple) chosen.clear();
                        v == true ? chosen.add(value) : chosen.remove(value);
                      }),
                    ),
                ]),
        ),
        actions: [
          TextButton(
              key: const Key('refined-clear'),
              onPressed: () => Navigator.pop(context, <String>{}),
              child: const Text('Clear')),
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Cancel')),
          FilledButton(
              key: const Key('refined-apply'),
              onPressed: () => Navigator.pop(context, chosen),
              child: const Text('Apply')),
        ],
      ),
    ),
  );
}

/// Parses a refined filter's options ({value, label, count}).
List<Map<String, dynamic>> refinedOptions(dynamic raw) {
  if (raw is! List) return const [];
  return [
    for (final o in raw)
      if (o is Map)
        Map<String, dynamic>.from(o)
      else if (o != null)
        {'value': '$o', 'label': '$o'}
  ];
}

// ------------------------------------------------------------ advanced search

/// Operators understood by the server (tables.AdvancedSearchRule).
const advancedOperators = <String, String>{
  'contains': 'contains',
  'not_contains': 'does not contain',
  'equals': 'equals',
  'not_equals': 'does not equal',
  'starts_with': 'starts with',
  'ends_with': 'ends with',
  'greater_than': '>',
  'greater_or_equal': '≥',
  'less_than': '<',
  'less_or_equal': '≤',
  'between': 'between',
  'in': 'is one of',
  'not_in': 'is not one of',
  'is_empty': 'is empty',
  'is_not_empty': 'is not empty',
};

bool _needsValue(String op) => op != 'is_empty' && op != 'is_not_empty';
bool _listValue(String op) => op == 'between' || op == 'in' || op == 'not_in';

class AdvancedRule {
  AdvancedRule(this.field, [this.operator = 'contains', this.value = '']);
  String field;
  String operator;
  String value;

  bool get complete =>
      field.isNotEmpty && (!_needsValue(operator) || value.trim().isNotEmpty);

  Map<String, dynamic> toJson() => {
        'field': field,
        'operator': operator,
        'value': value.trim(),
        if (_listValue(operator))
          'values': [
            for (final v in value.split(','))
              if (v.trim().isNotEmpty) v.trim()
          ],
      };
}

/// A set of rules joined by AND or OR, sent as the advanced_search query.
class AdvancedSearch {
  AdvancedSearch({this.conjunction = 'AND', List<AdvancedRule>? rules})
      : rules = rules ?? [];
  String conjunction;
  final List<AdvancedRule> rules;

  List<AdvancedRule> get _complete => rules.where((r) => r.complete).toList();
  bool get isEmpty => _complete.isEmpty;
  int get length => _complete.length;

  /// The JSON the server parses, or '' when there are no complete rules.
  String encode() => isEmpty
      ? ''
      : json.encode({
          'conjunction': conjunction,
          'rules': [for (final r in _complete) r.toJson()],
        });

  /// Reads [encode]'s JSON back ('' or invalid JSON: an empty search).
  static AdvancedSearch decode(String raw) {
    try {
      final m = json.decode(raw);
      if (m is! Map) return AdvancedSearch();
      final rules = m['rules'];
      return AdvancedSearch(
        conjunction: '${m['conjunction'] ?? 'AND'}' == 'OR' ? 'OR' : 'AND',
        rules: [
          if (rules is List)
            for (final r in rules)
              if (r is Map)
                AdvancedRule(
                    '${r['field'] ?? ''}',
                    '${r['operator'] ?? 'contains'}',
                    r['values'] is List && '${r['value'] ?? ''}'.isEmpty
                        ? (r['values'] as List).join(', ')
                        : '${r['value'] ?? ''}'),
        ],
      );
    } catch (_) {
      return AdvancedSearch();
    }
  }

  AdvancedSearch copy() => AdvancedSearch(conjunction: conjunction, rules: [
        for (final r in rules) AdvancedRule(r.field, r.operator, r.value)
      ]);
}

/// Edits an advanced search over [fields] (name -> label). Returns the new
/// search, or null when cancelled.
Future<AdvancedSearch?> showAdvancedSearchDialog(
  BuildContext context, {
  required Map<String, String> fields,
  required AdvancedSearch current,
}) {
  final search = current.copy();
  if (search.rules.isEmpty && fields.isNotEmpty) {
    search.rules.add(AdvancedRule(fields.keys.first));
  }
  return showDialog<AdvancedSearch>(
    context: context,
    builder: (context) => StatefulBuilder(builder: (context, setState) {
      Widget rule(int i) {
        final r = search.rules[i];
        return Padding(
          padding: const EdgeInsets.only(bottom: 8),
          child: Wrap(
              spacing: 8,
              runSpacing: 8,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: [
                DropdownButton<String>(
                  key: Key('adv-field-$i'),
                  value: fields.containsKey(r.field) ? r.field : null,
                  items: [
                    for (final f in fields.entries)
                      DropdownMenuItem(value: f.key, child: Text(f.value)),
                  ],
                  onChanged: (v) => setState(() => r.field = v ?? r.field),
                ),
                DropdownButton<String>(
                  key: Key('adv-op-$i'),
                  value: r.operator,
                  items: [
                    for (final o in advancedOperators.entries)
                      DropdownMenuItem(value: o.key, child: Text(o.value)),
                  ],
                  onChanged: (v) =>
                      setState(() => r.operator = v ?? r.operator),
                ),
                if (_needsValue(r.operator))
                  SizedBox(
                    width: 200,
                    child: TextFormField(
                      key: Key('adv-value-$i'),
                      initialValue: r.value,
                      decoration: InputDecoration(
                          isDense: true,
                          hintText: r.operator == 'between'
                              ? 'from, to'
                              : _listValue(r.operator)
                                  ? 'a, b, c'
                                  : 'value'),
                      onChanged: (v) => r.value = v,
                    ),
                  ),
                IconButton(
                  key: Key('adv-remove-$i'),
                  tooltip: 'Remove rule',
                  icon: const Icon(Icons.close, size: 18),
                  onPressed: () => setState(() => search.rules.removeAt(i)),
                ),
              ]),
        );
      }

      return AlertDialog(
        title: const Text('Advanced search'),
        content: SizedBox(
          width: 620,
          child: SingleChildScrollView(
            child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  if (search.rules.length > 1)
                    Padding(
                      padding: const EdgeInsets.only(bottom: 12),
                      child: SegmentedButton<String>(
                        key: const Key('adv-conjunction'),
                        segments: const [
                          ButtonSegment(value: 'AND', label: Text('Match all')),
                          ButtonSegment(value: 'OR', label: Text('Match any')),
                        ],
                        selected: {search.conjunction},
                        onSelectionChanged: (s) =>
                            setState(() => search.conjunction = s.first),
                      ),
                    ),
                  for (var i = 0; i < search.rules.length; i++) rule(i),
                  TextButton.icon(
                    key: const Key('adv-add'),
                    onPressed: fields.isEmpty
                        ? null
                        : () => setState(() =>
                            search.rules.add(AdvancedRule(fields.keys.first))),
                    icon: const Icon(Icons.add, size: 18),
                    label: const Text('Add rule'),
                  ),
                ]),
          ),
        ),
        actions: [
          TextButton(
              key: const Key('adv-clear'),
              onPressed: () => Navigator.pop(context, AdvancedSearch()),
              child: const Text('Clear')),
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Cancel')),
          FilledButton(
              key: const Key('adv-apply'),
              onPressed: () => Navigator.pop(context, search),
              child: const Text('Apply')),
        ],
      );
    }),
  );
}

// ---------------------------------------------------------- column resizing

const double minColumnWidth = 48;
const double maxColumnWidth = 800;

/// A column header label with a drag handle on its right edge. Dragging
/// sets the column width (starting from its current width); a double tap on
/// the handle goes back to automatic width.
class ResizableHeader extends StatefulWidget {
  const ResizableHeader({
    super.key,
    required this.name,
    required this.label,
    required this.width,
    required this.onResize,
    required this.onResizeEnd,
    required this.onReset,
  });

  final String name;
  final Widget label;
  final double? width;
  final ValueChanged<double> onResize;
  final VoidCallback onResizeEnd;
  final VoidCallback onReset;

  @override
  State<ResizableHeader> createState() => _ResizableHeaderState();
}

class _ResizableHeaderState extends State<ResizableHeader> {
  double? _dragWidth;

  /// The current width of the label area: the table cell holding this
  /// header, less its padding and the handle.
  double _currentWidth() {
    double? cell;
    context.visitAncestorElements((e) {
      final ro = e.renderObject;
      if (ro is RenderBox && ro.parent is RenderTable) {
        cell = ro.size.width;
        return false;
      }
      return true;
    });
    final own = (context.findRenderObject() as RenderBox?)?.size.width ?? 120;
    final w = cell == null ? own : cell! - 28;
    return (w - 12).clamp(minColumnWidth, maxColumnWidth).toDouble();
  }

  @override
  Widget build(BuildContext context) {
    final handle = MouseRegion(
      cursor: SystemMouseCursors.resizeColumn,
      child: GestureDetector(
        key: Key('column-resize-${widget.name}'),
        behavior: HitTestBehavior.opaque,
        onHorizontalDragStart: (_) =>
            _dragWidth = widget.width ?? _currentWidth(),
        onHorizontalDragUpdate: (d) {
          final w = ((_dragWidth ?? _currentWidth()) + d.delta.dx)
              .clamp(minColumnWidth, maxColumnWidth)
              .toDouble();
          _dragWidth = w;
          widget.onResize(w);
        },
        onHorizontalDragEnd: (_) {
          _dragWidth = null;
          widget.onResizeEnd();
        },
        onDoubleTap: widget.onReset,
        child: SizedBox(
          width: 12,
          height: 28,
          child: Center(
            child: Container(
                width: 2, height: 18, color: Theme.of(context).dividerColor),
          ),
        ),
      ),
    );
    final w = widget.width;
    return Row(mainAxisSize: MainAxisSize.min, children: [
      if (w != null) SizedBox(width: w, child: widget.label) else widget.label,
      handle,
    ]);
  }
}

/// Keeps cell content inside a fixed column width.
Widget sizedCell(double? width, Widget child) => width == null
    ? child
    : SizedBox(
        width: width + 12, // the header's handle
        child: ClipRect(
            child: Align(
                alignment: AlignmentDirectional.centerStart, child: child)));

// ------------------------------------------------------------- range filters

/// Edits a date or number range filter ("from,to", either side optional).
/// Returns the new value ("" clears it), or null when cancelled.
Future<String?> showRangeFilterDialog(
  BuildContext context, {
  required String label,
  required bool dates,
  required String value,
}) {
  final parts = value.split(',');
  final from = TextEditingController(text: parts.isNotEmpty ? parts[0] : '');
  final to = TextEditingController(text: parts.length > 1 ? parts[1] : '');
  String fmt(DateTime d) =>
      '${d.year.toString().padLeft(4, '0')}-${d.month.toString().padLeft(2, '0')}-${d.day.toString().padLeft(2, '0')}';

  Widget field(String key, String hint, TextEditingController c) => SizedBox(
        width: 160,
        child: TextField(
          key: Key(key),
          controller: c,
          readOnly: dates,
          keyboardType: dates
              ? null
              : const TextInputType.numberWithOptions(
                  decimal: true, signed: true),
          decoration: InputDecoration(
            isDense: true,
            labelText: hint,
            suffixIcon: dates
                ? const Icon(Icons.calendar_today_outlined, size: 18)
                : null,
          ),
          onTap: !dates
              ? null
              : () async {
                  final initial = DateTime.tryParse(c.text) ?? DateTime.now();
                  final picked = await showDatePicker(
                      context: context,
                      initialDate: initial,
                      firstDate: DateTime(1900),
                      lastDate: DateTime(2200));
                  if (picked != null) c.text = fmt(picked);
                },
        ),
      );

  return showDialog<String>(
    context: context,
    builder: (context) => AlertDialog(
      title: Text(label),
      content: Wrap(spacing: 12, runSpacing: 12, children: [
        field('range-from', dates ? 'From' : 'Min', from),
        field('range-to', dates ? 'To' : 'Max', to),
      ]),
      actions: [
        TextButton(
            key: const Key('range-clear'),
            onPressed: () => Navigator.pop(context, ''),
            child: const Text('Clear')),
        TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel')),
        FilledButton(
            key: const Key('range-apply'),
            onPressed: () {
              final a = from.text.trim(), b = to.text.trim();
              Navigator.pop(context, a.isEmpty && b.isEmpty ? '' : '$a,$b');
            },
            child: const Text('Apply')),
      ],
    ),
  );
}

/// The chip text of a range value: "Amount: 10 – 20", "Created: ≥ 2026-01-01".
String rangeSummary(String label, String value) {
  final p = value.split(',');
  final a = p.isNotEmpty ? p[0] : '', b = p.length > 1 ? p[1] : '';
  if (a.isEmpty && b.isEmpty) return label;
  if (b.isEmpty) return '$label: ≥ $a';
  if (a.isEmpty) return '$label: ≤ $b';
  return '$label: $a – $b';
}

// ------------------------------------------------------ summaries & groups

const summaryLabels = {
  'sum': 'Sum',
  'avg': 'Average',
  'count': 'Count',
  'min': 'Min',
  'max': 'Max',
};

double? _num(dynamic v) {
  if (v is num) return v.toDouble();
  if (v is String) return double.tryParse(v.trim());
  return null;
}

/// Computes the aggregates (sum, avg, count, min, max) of [values], like
/// the server's tables.Compute.
Map<String, double> computeSummaries(
    Iterable<dynamic> values, List<String> kinds) {
  final nums = <double>[];
  var count = 0;
  for (final v in values) {
    if (v == null || v == '') continue;
    count++;
    final n = _num(v);
    if (n != null && !n.isNaN) nums.add(n);
  }
  final out = <String, double>{};
  for (final k in kinds) {
    switch (k) {
      case 'count':
        out[k] = count.toDouble();
      case 'sum':
        out[k] = nums.fold(0.0, (a, b) => a + b);
      case 'avg':
        if (nums.isNotEmpty) {
          out[k] = nums.fold(0.0, (a, b) => a + b) / nums.length;
        }
      case 'min':
        if (nums.isNotEmpty) out[k] = nums.reduce((a, b) => a < b ? a : b);
      case 'max':
        if (nums.isNotEmpty) out[k] = nums.reduce((a, b) => a > b ? a : b);
    }
  }
  return out;
}

/// Prints an aggregate with at most two decimals.
String formatSummary(double v) {
  if (v == v.roundToDouble() && v.abs() < 1e15) return v.toInt().toString();
  final s = v.toStringAsFixed(2);
  return s.endsWith('0') ? s.substring(0, s.length - 1) : s;
}

/// The summary lines of one column ("Sum 115", "Count 3").
Widget summaryCell(
    BuildContext context, Map<String, double>? values, List<String> kinds) {
  if (values == null) return const SizedBox.shrink();
  final theme = Theme.of(context);
  return Column(
    mainAxisSize: MainAxisSize.min,
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      for (final k in kinds)
        if (values[k] != null)
          Text.rich(
              TextSpan(children: [
                TextSpan(
                    text: '${summaryLabels[k] ?? k} ',
                    style:
                        TextStyle(color: theme.colorScheme.onSurfaceVariant)),
                TextSpan(
                    text: formatSummary(values[k]!),
                    style: const TextStyle(fontWeight: FontWeight.w700)),
              ]),
              style: const TextStyle(fontSize: 12)),
    ],
  );
}
