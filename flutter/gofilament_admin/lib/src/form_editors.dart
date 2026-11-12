import 'package:flutter/material.dart';

import 'format.dart';

/// Edits a repeater: a list of items, each with the repeater's child fields.
/// Supports text-like, number, toggle and select / radio (listed options)
/// children; other child types are edited as text. The server validates
/// every item again on save.
class RepeaterEditor extends StatefulWidget {
  const RepeaterEditor({
    super.key,
    required this.node,
    required this.items,
    required this.enabled,
    required this.onChanged,
  });

  final Map<String, dynamic> node;
  final List<Map<String, dynamic>> items;
  final bool enabled;
  final ValueChanged<List<Map<String, dynamic>>> onChanged;

  @override
  State<RepeaterEditor> createState() => _RepeaterEditorState();
}

class _RepeaterEditorState extends State<RepeaterEditor> {
  late List<Map<String, dynamic>> _items =
      widget.items.map((m) => Map<String, dynamic>.from(m)).toList();

  // Stable keys so text fields keep their state when items move.
  late final List<int> _ids = List.generate(_items.length, (i) => i);
  late int _nextId = _items.length;

  String get _name => '${widget.node['name']}';

  List<Map<String, dynamic>> get _fields {
    final out = <Map<String, dynamic>>[];
    void walk(dynamic nodes) {
      if (nodes is! List) return;
      for (final n in nodes.whereType<Map>()) {
        if (n['kind'] == 'field' && n['name'] != null) {
          out.add(Map<String, dynamic>.from(n));
        } else {
          walk(n['children']);
        }
      }
    }

    walk(widget.node['children']);
    return out;
  }

  int get _max => (widget.node['max_items'] as num?)?.toInt() ?? 0;
  int get _min => (widget.node['min_items'] as num?)?.toInt() ?? 0;

  void _emit() => widget.onChanged(_items);

  void _add() {
    setState(() {
      final item = <String, dynamic>{};
      for (final f in _fields) {
        if (f['default'] != null) item['${f['name']}'] = f['default'];
        if (f['type'] == 'toggle') {
          item.putIfAbsent('${f['name']}', () => false);
        }
      }
      _items = [..._items, item];
      _ids.add(_nextId++);
    });
    _emit();
  }

  void _remove(int i) {
    setState(() {
      _items = [..._items]..removeAt(i);
      _ids.removeAt(i);
    });
    _emit();
  }

  void _move(int i, int delta) {
    final j = i + delta;
    if (j < 0 || j >= _items.length) return;
    setState(() {
      final items = [..._items];
      final it = items.removeAt(i);
      items.insert(j, it);
      _items = items;
      final id = _ids.removeAt(i);
      _ids.insert(j, id);
    });
    _emit();
  }

  void _set(int i, String field, dynamic value) {
    _items[i][field] = value;
    _emit();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final label = '${widget.node['label'] ?? humanize(_name)}';
    final required = widget.node['required'] == true;
    return FormField<List<Map<String, dynamic>>>(
      key: Key('field-$_name'),
      validator: (_) {
        if (required && _items.isEmpty) return '$label is required';
        if (_min > 0 && _items.length < _min) {
          return '$label needs at least $_min items';
        }
        return null;
      },
      builder: (state) => InputDecorator(
        decoration: InputDecoration(
            labelText: required ? '$label *' : label,
            errorText: state.errorText,
            enabled: widget.enabled),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            for (var i = 0; i < _items.length; i++)
              Container(
                key: ValueKey('$_name-item-${_ids[i]}'),
                margin: const EdgeInsets.only(bottom: 10),
                padding: const EdgeInsets.fromLTRB(12, 4, 4, 12),
                decoration: BoxDecoration(
                  border: Border.all(color: theme.dividerColor),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: Column(children: [
                  Row(children: [
                    Text('${i + 1}', style: theme.textTheme.labelMedium),
                    const Spacer(),
                    if (widget.enabled) ...[
                      IconButton(
                          tooltip: 'Move up',
                          onPressed: i == 0 ? null : () => _move(i, -1),
                          icon: const Icon(Icons.arrow_upward, size: 18)),
                      IconButton(
                          tooltip: 'Move down',
                          onPressed:
                              i == _items.length - 1 ? null : () => _move(i, 1),
                          icon: const Icon(Icons.arrow_downward, size: 18)),
                      IconButton(
                          key: Key('$_name-remove-$i'),
                          tooltip: 'Remove',
                          onPressed: () => _remove(i),
                          icon: Icon(Icons.delete_outline,
                              size: 18, color: theme.colorScheme.error)),
                    ],
                  ]),
                  for (final f in _fields)
                    Padding(
                      padding: const EdgeInsets.only(top: 8),
                      child: _ItemField(
                        key: Key('$_name.$i.${f['name']}'),
                        field: f,
                        value: _items[i]['${f['name']}'],
                        enabled: widget.enabled && f['disabled'] != true,
                        onChanged: (v) => _set(i, '${f['name']}', v),
                      ),
                    ),
                ]),
              ),
            if (widget.enabled && (_max == 0 || _items.length < _max))
              Align(
                alignment: Alignment.centerLeft,
                child: TextButton.icon(
                  key: Key('$_name-add'),
                  onPressed: () {
                    _add();
                    state.didChange(_items);
                  },
                  icon: const Icon(Icons.add),
                  label: const Text('Add item'),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

/// One child field of a repeater item.
class _ItemField extends StatelessWidget {
  const _ItemField({
    super.key,
    required this.field,
    required this.value,
    required this.enabled,
    required this.onChanged,
  });

  final Map<String, dynamic> field;
  final dynamic value;
  final bool enabled;
  final ValueChanged<dynamic> onChanged;

  @override
  Widget build(BuildContext context) {
    final name = '${field['name']}';
    final type = '${field['type']}';
    final label = '${field['label'] ?? humanize(name)}';
    final required = field['required'] == true;
    final decoration = InputDecoration(
        labelText: required ? '$label *' : label, enabled: enabled);

    switch (type) {
      case 'toggle':
        return SwitchListTile(
          value: value == true,
          onChanged: enabled ? onChanged : null,
          title: Text(label),
          contentPadding: EdgeInsets.zero,
          dense: true,
        );
      case 'select':
      case 'radio':
        final options =
            (field['options'] is List ? field['options'] as List : const [])
                .whereType<Map>()
                .toList();
        if (options.isNotEmpty) {
          final current = value == null ? null : '$value';
          return DropdownButtonFormField<String>(
            initialValue:
                options.any((o) => '${o['value']}' == current) ? current : null,
            decoration: decoration,
            items: [
              for (final o in options)
                DropdownMenuItem(
                    value: '${o['value']}',
                    child: Text('${o['label'] ?? o['value']}')),
            ],
            onChanged: enabled ? onChanged : null,
            validator: (v) => required && (v == null || v.isEmpty)
                ? '$label is required'
                : null,
          );
        }
    }
    final number = type == 'number';
    return TextFormField(
      initialValue: value == null ? '' : displayValue(value),
      enabled: enabled,
      decoration: decoration,
      keyboardType: number
          ? const TextInputType.numberWithOptions(decimal: true, signed: true)
          : null,
      onChanged: (text) {
        final t = text.trim();
        onChanged(number ? (t.isEmpty ? null : (num.tryParse(t) ?? t)) : text);
      },
      validator: (v) {
        final t = v?.trim() ?? '';
        if (required && t.isEmpty) return '$label is required';
        if (number && t.isNotEmpty && num.tryParse(t) == null) {
          return '$label must be a number';
        }
        return null;
      },
    );
  }
}

/// Edits a key-value field as rows of key and value text inputs.
class KeyValueEditor extends StatefulWidget {
  const KeyValueEditor({
    super.key,
    required this.node,
    required this.value,
    required this.enabled,
    required this.onChanged,
  });

  final Map<String, dynamic> node;
  final Map<String, dynamic> value;
  final bool enabled;
  final ValueChanged<Map<String, dynamic>> onChanged;

  @override
  State<KeyValueEditor> createState() => _KeyValueEditorState();
}

class _KeyValueEditorState extends State<KeyValueEditor> {
  late final List<(TextEditingController, TextEditingController)> _rows = [
    for (final e in widget.value.entries)
      (
        TextEditingController(text: e.key),
        TextEditingController(text: displayValue(e.value)),
      ),
  ];

  @override
  void dispose() {
    for (final (k, v) in _rows) {
      k.dispose();
      v.dispose();
    }
    super.dispose();
  }

  void _emit() {
    final out = <String, dynamic>{};
    for (final (k, v) in _rows) {
      if (k.text.trim().isNotEmpty) out[k.text.trim()] = v.text;
    }
    widget.onChanged(out);
  }

  @override
  Widget build(BuildContext context) {
    final name = '${widget.node['name']}';
    final label = '${widget.node['label'] ?? humanize(name)}';
    final keyLabel = '${widget.node['key_label'] ?? 'Key'}';
    final valueLabel = '${widget.node['value_label'] ?? 'Value'}';
    return InputDecorator(
      key: Key('field-$name'),
      decoration: InputDecoration(labelText: label, enabled: widget.enabled),
      child: Column(children: [
        for (var i = 0; i < _rows.length; i++)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Row(children: [
              Expanded(
                child: TextField(
                  key: Key('$name-key-$i'),
                  controller: _rows[i].$1,
                  enabled: widget.enabled,
                  decoration: InputDecoration(labelText: keyLabel),
                  onChanged: (_) => _emit(),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: TextField(
                  key: Key('$name-value-$i'),
                  controller: _rows[i].$2,
                  enabled: widget.enabled,
                  decoration: InputDecoration(labelText: valueLabel),
                  onChanged: (_) => _emit(),
                ),
              ),
              if (widget.enabled)
                IconButton(
                  tooltip: 'Remove',
                  onPressed: () {
                    setState(() {
                      final (k, v) = _rows.removeAt(i);
                      k.dispose();
                      v.dispose();
                    });
                    _emit();
                  },
                  icon: const Icon(Icons.close, size: 18),
                ),
            ]),
          ),
        if (widget.enabled)
          Align(
            alignment: Alignment.centerLeft,
            child: TextButton.icon(
              key: Key('$name-add'),
              onPressed: () => setState(() => _rows
                  .add((TextEditingController(), TextEditingController()))),
              icon: const Icon(Icons.add),
              label: const Text('Add row'),
            ),
          ),
      ]),
    );
  }
}

/// Tabs or wizard steps. Every step stays mounted, so its fields keep their
/// values and take part in validation.
class FormSteps extends StatefulWidget {
  const FormSteps({
    super.key,
    required this.labels,
    required this.children,
    this.wizard = false,
  });

  final List<String> labels;
  final List<Widget> children;
  final bool wizard;

  @override
  State<FormSteps> createState() => _FormStepsState();
}

class _FormStepsState extends State<FormSteps> {
  int _index = 0;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final last = widget.children.length - 1;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      mainAxisSize: MainAxisSize.min,
      children: [
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(children: [
            for (var i = 0; i < widget.labels.length; i++)
              Padding(
                padding: const EdgeInsets.only(right: 8),
                child: ChoiceChip(
                  key: Key('step-$i'),
                  label: Text(widget.wizard
                      ? '${i + 1}. ${widget.labels[i]}'
                      : widget.labels[i]),
                  selected: i == _index,
                  onSelected: (_) => setState(() => _index = i),
                ),
              ),
          ]),
        ),
        const SizedBox(height: 14),
        IndexedStack(
          index: _index.clamp(0, last < 0 ? 0 : last),
          children: [
            for (var i = 0; i < widget.children.length; i++)
              Visibility(
                visible: i == _index,
                maintainState: true,
                child: widget.children[i],
              ),
          ],
        ),
        if (widget.wizard && widget.children.length > 1)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Row(children: [
              if (_index > 0)
                TextButton(
                  key: const Key('step-back'),
                  onPressed: () => setState(() => _index--),
                  child: const Text('Back'),
                ),
              const Spacer(),
              Text('Step ${_index + 1} of ${widget.children.length}',
                  style: theme.textTheme.bodySmall),
              const SizedBox(width: 12),
              if (_index < last)
                FilledButton.tonal(
                  key: const Key('step-next'),
                  onPressed: () => setState(() => _index++),
                  child: const Text('Next'),
                ),
            ]),
          ),
      ],
    );
  }
}
