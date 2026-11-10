import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'api.dart';
import 'controller.dart';
import 'format.dart';

/// Opens the create / edit / view form of a resource. Resolves to true when a
/// record was saved.
Future<bool?> showResourceForm(
  BuildContext context, {
  required AdminController admin,
  required String slug,
  String? id,
  bool readOnly = false,
}) {
  final wide = MediaQuery.sizeOf(context).width >= 760;
  final form =
      ResourceForm(admin: admin, slug: slug, id: id, readOnly: readOnly);
  return showDialog<bool>(
    context: context,
    barrierDismissible: false,
    builder: (ctx) => wide
        ? Dialog(
            insetPadding: const EdgeInsets.all(24),
            clipBehavior: Clip.antiAlias,
            child: ConstrainedBox(
                constraints:
                    const BoxConstraints(maxWidth: 880, maxHeight: 820),
                child: form),
          )
        : Dialog.fullscreen(child: form),
  );
}

/// A form rendered from the server's description of the resource form.
class ResourceForm extends StatefulWidget {
  const ResourceForm(
      {super.key,
      required this.admin,
      required this.slug,
      this.id,
      this.readOnly = false});

  final AdminController admin;
  final String slug;
  final String? id;
  final bool readOnly;

  @override
  State<ResourceForm> createState() => _ResourceFormState();
}

class _ResourceFormState extends State<ResourceForm> {
  final _formKey = GlobalKey<FormState>();
  final Map<String, TextEditingController> _text = {};
  final Map<String, dynamic> _values = {};
  final Map<String, String> _labels = {};
  final List<Map<String, dynamic>> _fields = [];

  FormPayload? _payload;
  String? _loadError;
  String? _saveError;
  bool _saving = false;

  bool get _isEdit => widget.id != null;
  bool get _locked =>
      widget.readOnly || (_isEdit && _payload?.abilities['update'] != true);

  static const _textTypes = {
    'text',
    'email',
    'password',
    'number',
    'tel',
    'url',
    'textarea',
    'markdown',
    'code',
    'rich_text',
    'tags',
    'color',
    'time',
  };

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    for (final c in _text.values) {
      c.dispose();
    }
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final payload = await widget.admin.api.form(widget.slug, id: widget.id);
      _collectFields(payload.schema);
      for (final f in _fields) {
        final name = '${f['name']}';
        final type = '${f['type']}';
        var value = payload.state.containsKey(name)
            ? payload.state[name]
            : (payload.isEdit ? null : f['default']);
        if (type == 'toggle') {
          value = value == true || value == 'true' || value == 1;
        }
        if (type == 'checkbox_list') {
          value = value is List ? value.map((e) => '$e').toList() : <String>[];
        }
        if (type == 'tags' && value is List) value = value.join(', ');
        if (type == 'password') value = null; // never prefilled
        if (_textTypes.contains(type)) {
          _text[name] = TextEditingController(text: displayValue(value));
        } else if (type == 'select' || type == 'radio') {
          _values[name] = value == null ? null : displayValue(value);
        } else {
          _values[name] = value;
        }
      }
      _labels.addAll(payload.optionLabels);
      if (mounted) setState(() => _payload = payload);
    } on ApiException catch (e) {
      if (mounted) setState(() => _loadError = e.message);
    } catch (_) {
      if (mounted) {
        setState(
            () => _loadError = 'Could not load the form. Please try again.');
      }
    }
  }

  void _collectFields(List<dynamic> nodes) {
    for (final n in nodes.whereType<Map>()) {
      if (n['kind'] == 'field' && n['name'] != null) {
        _fields.add(Map<String, dynamic>.from(n));
      }
      if (n['children'] is List && n['kind'] != 'repeater') {
        _collectFields(n['children'] as List);
      }
    }
  }

  Map<String, dynamic> _collect() {
    final data = <String, dynamic>{};
    for (final f in _fields) {
      final name = '${f['name']}';
      final type = '${f['type']}';
      if (f['disabled'] == true ||
          const {'file', 'image', 'key_value'}.contains(type)) {
        continue;
      }
      if (_textTypes.contains(type)) {
        final text = _text[name]?.text ?? '';
        if (type == 'password' && text.isEmpty) continue; // unchanged
        if (type == 'number') {
          data[name] = text.trim().isEmpty
              ? ''
              : (num.tryParse(text.trim()) ?? text.trim());
        } else {
          data[name] = type == 'textarea' ||
                  type == 'markdown' ||
                  type == 'code' ||
                  type == 'rich_text'
              ? text
              : text.trim();
        }
      } else {
        data[name] = _values[name] ?? (type == 'toggle' ? false : '');
      }
    }
    return data;
  }

  Future<void> _save() async {
    if (_saving || !(_formKey.currentState?.validate() ?? false)) return;
    setState(() {
      _saving = true;
      _saveError = null;
    });
    try {
      final data = _collect();
      if (_isEdit) {
        await widget.admin.api.update(widget.slug, widget.id!, data);
      } else {
        await widget.admin.api.create(widget.slug, data);
      }
      if (mounted) Navigator.of(context).pop(true);
    } on ApiException catch (e) {
      if (mounted) setState(() => _saveError = e.message);
    } catch (_) {
      if (mounted) {
        setState(() => _saveError = 'Could not save. Please try again.');
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  // ------------------------------------------------------------------ build

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final model = '${widget.admin.resource(widget.slug)?['model_name'] ?? ''}';
    final title =
        widget.readOnly ? model : (_isEdit ? 'Edit $model' : 'New $model');

    Widget body;
    if (_loadError != null) {
      body = Center(
          child: Padding(
              padding: const EdgeInsets.all(24), child: Text(_loadError!)));
    } else if (_payload == null) {
      body = const Center(
          child: Padding(
              padding: EdgeInsets.all(40), child: CircularProgressIndicator()));
    } else {
      body = Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(20),
          shrinkWrap: true,
          children: [
            if (_saveError != null)
              Container(
                key: const Key('form-error'),
                margin: const EdgeInsets.only(bottom: 16),
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: theme.colorScheme.error.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: Text(_saveError!,
                    style: TextStyle(color: theme.colorScheme.error)),
              ),
            ..._nodes(context, _payload!.schema, 1),
          ],
        ),
      );
    }

    return Material(
      color: theme.colorScheme.surface,
      child: Column(mainAxisSize: MainAxisSize.min, children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 12, 8, 12),
          child: Row(children: [
            Expanded(
              child: Text(title.trim(),
                  style: theme.textTheme.titleLarge
                      ?.copyWith(fontWeight: FontWeight.w700)),
            ),
            IconButton(
              key: const Key('form-close'),
              tooltip: 'Close',
              icon: const Icon(Icons.close),
              onPressed: () => Navigator.of(context).pop(false),
            ),
          ]),
        ),
        const Divider(),
        Flexible(child: body),
        const Divider(),
        Padding(
          padding: const EdgeInsets.all(14),
          child: Row(mainAxisAlignment: MainAxisAlignment.end, children: [
            OutlinedButton(
              onPressed: () => Navigator.of(context).pop(false),
              child: Text(_locked ? 'Close' : 'Cancel'),
            ),
            if (!_locked && _payload != null) ...[
              const SizedBox(width: 10),
              FilledButton(
                key: const Key('form-save'),
                onPressed: _saving ? null : _save,
                child: _saving
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(strokeWidth: 2))
                    : Text(_isEdit ? 'Save changes' : 'Create'),
              ),
            ],
          ]),
        ),
      ]),
    );
  }

  /// Lays [nodes] out in a grid of [columns] columns (one column on narrow screens).
  List<Widget> _nodes(BuildContext context, List<dynamic> nodes, int columns) {
    final items = <(int, Widget)>[];
    for (final raw in nodes.whereType<Map>()) {
      final node = Map<String, dynamic>.from(raw);
      final widget = _node(context, node);
      if (widget == null) continue;
      final span = (node['column_span'] as num?)?.toInt() ?? 1;
      final full = node['kind'] != 'field' ||
          const {'textarea', 'markdown', 'rich_text', 'code'}
              .contains(node['type']);
      items.add((full ? columns : span.clamp(1, columns), widget));
    }
    if (columns <= 1) {
      return [
        for (final (_, w) in items)
          Padding(padding: const EdgeInsets.only(bottom: 14), child: w)
      ];
    }
    return [
      LayoutBuilder(builder: (context, c) {
        final cols = c.maxWidth < 520 ? 1 : columns;
        const gap = 14.0;
        final unit = (c.maxWidth - gap * (cols - 1)) / cols;
        return Wrap(spacing: gap, runSpacing: 14, children: [
          for (final (span, w) in items)
            SizedBox(
                width: cols == 1
                    ? c.maxWidth
                    : unit * span.clamp(1, cols) +
                        gap * (span.clamp(1, cols) - 1),
                child: w),
        ]);
      }),
      const SizedBox(height: 14),
    ];
  }

  Widget? _node(BuildContext context, Map<String, dynamic> node) {
    final theme = Theme.of(context);
    final children =
        node['children'] is List ? node['children'] as List : const [];

    Widget titled(String? heading, String? description, List<Widget> content) {
      return Container(
        padding: const EdgeInsets.fromLTRB(16, 14, 16, 2),
        decoration: BoxDecoration(
          border: Border.all(color: theme.dividerColor),
          borderRadius: BorderRadius.circular(12),
        ),
        child:
            Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
          if (heading != null && heading.isNotEmpty)
            Text(heading,
                style: theme.textTheme.titleSmall
                    ?.copyWith(fontWeight: FontWeight.w700)),
          if (description != null && description.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(description,
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
            ),
          if ((heading ?? '').isNotEmpty || (description ?? '').isNotEmpty)
            const SizedBox(height: 14),
          ...content,
        ]),
      );
    }

    switch ('${node['kind']}') {
      case 'field':
        return _field(context, node);
      case 'section':
        return titled(
            node['heading'] as String?,
            node['description'] as String?,
            _nodes(context, children, (node['columns'] as num?)?.toInt() ?? 1));
      case 'fieldset':
        return titled(
            node['heading'] as String?, null, _nodes(context, children, 1));
      case 'grid':
        return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: _nodes(context, children,
                ((node['columns'] as num?)?.toInt() ?? 2).clamp(1, 6)));
      case 'split':
        return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: _nodes(context, children, children.length.clamp(1, 3)));
      case 'tabs':
      case 'wizard':
        // v1: tabs and wizard steps are shown as consecutive sections
        return Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: [
              for (final child in children.whereType<Map>())
                Padding(
                  padding: const EdgeInsets.only(bottom: 14),
                  child: titled(
                      '${child['label'] ?? ''}',
                      child['description'] as String?,
                      _nodes(
                          context,
                          child['children'] is List
                              ? child['children'] as List
                              : const [],
                          1)),
                ),
            ]);
      case 'repeater':
        return _unsupported(context, '${node['label'] ?? node['name']}',
            'Repeating rows are not editable here yet.');
    }
    return null;
  }

  Widget _unsupported(BuildContext context, String label, String message) {
    final theme = Theme.of(context);
    return InputDecorator(
      decoration: InputDecoration(labelText: label, enabled: false),
      child: Text(message,
          style: theme.textTheme.bodySmall
              ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
    );
  }

  bool _required(Map<String, dynamic> f) =>
      f['required'] == true || (!_isEdit && f['required_on_create'] == true);

  Widget? _field(BuildContext context, Map<String, dynamic> f) {
    final name = '${f['name']}';
    final type = '${f['type']}';
    final label = '${f['label'] ?? humanize(name)}';
    final enabled = !_locked && f['disabled'] != true;
    final required = _required(f);
    final helper = f['helper_text'] as String?;

    InputDecoration decoration({Widget? suffixIcon}) => InputDecoration(
          labelText: required ? '$label *' : label,
          hintText: f['placeholder'] as String?,
          helperText: (helper == null || helper.isEmpty) ? null : helper,
          helperMaxLines: 2,
          prefixText: (f['prefix'] as String?)?.isNotEmpty == true
              ? '${f['prefix']} '
              : null,
          suffixText: (f['suffix'] as String?)?.isNotEmpty == true
              ? ' ${f['suffix']}'
              : null,
          suffixIcon: suffixIcon,
          enabled: enabled,
        );

    String? requiredCheck(String? v) =>
        required && (v == null || v.trim().isEmpty)
            ? '$label is required'
            : null;

    if (type == 'hidden') return null;

    if (_textTypes.contains(type)) {
      final multiline =
          const {'textarea', 'markdown', 'code', 'rich_text'}.contains(type);
      final maxLength = (f['max_length'] as num?)?.toInt() ?? 0;
      final minLength = (f['min_length'] as num?)?.toInt() ?? 0;
      return TextFormField(
        key: Key('field-$name'),
        controller: _text[name],
        enabled: enabled,
        obscureText: type == 'password',
        minLines: multiline ? ((f['rows'] as num?)?.toInt() ?? 3) : 1,
        maxLines: multiline ? 10 : 1,
        keyboardType: switch (type) {
          'email' => TextInputType.emailAddress,
          'number' =>
            const TextInputType.numberWithOptions(decimal: true, signed: true),
          'tel' => TextInputType.phone,
          'url' => TextInputType.url,
          _ => multiline ? TextInputType.multiline : TextInputType.text,
        },
        inputFormatters: [
          if (type == 'number')
            FilteringTextInputFormatter.allow(RegExp(r'[0-9.\-]')),
          if (maxLength > 0) LengthLimitingTextInputFormatter(maxLength),
        ],
        style: type == 'code' ? const TextStyle(fontFamily: 'monospace') : null,
        decoration: decoration(),
        validator: (v) {
          final missing =
              type == 'password' && _isEdit ? null : requiredCheck(v);
          if (missing != null) return missing;
          final text = v?.trim() ?? '';
          if (text.isEmpty) return null;
          if (minLength > 0 && text.length < minLength) {
            return '$label must be at least $minLength characters';
          }
          if (type == 'number' && num.tryParse(text) == null) {
            return '$label must be a number';
          }
          if (type == 'email' &&
              !RegExp(r'^[^@\s]+@[^@\s]+\.[^@\s]+$').hasMatch(text)) {
            return 'Enter a valid email address';
          }
          return null;
        },
      );
    }

    switch (type) {
      case 'toggle':
        return SwitchListTile(
          key: Key('field-$name'),
          value: _values[name] == true,
          onChanged: enabled ? (v) => setState(() => _values[name] = v) : null,
          title: Text(label),
          subtitle: (helper == null || helper.isEmpty) ? null : Text(helper),
          contentPadding: EdgeInsets.zero,
          dense: true,
        );

      case 'select':
      case 'radio':
        return _SelectInput(
          key: Key('field-$name'),
          api: widget.admin.api,
          slug: widget.slug,
          field: f,
          value: _values[name] as String?,
          valueLabel: _labels[name],
          enabled: enabled,
          decoration: decoration(),
          validator: requiredCheck,
          onChanged: (value, label) => setState(() {
            _values[name] = value;
            if (label != null) _labels[name] = label;
          }),
        );

      case 'checkbox_list':
        final selected = (_values[name] as List?)?.cast<String>() ?? <String>[];
        return InputDecorator(
          decoration: decoration(),
          child: Wrap(spacing: 8, runSpacing: 4, children: [
            for (final o in FieldOption.listFrom(f['options']))
              FilterChip(
                label: Text(o.label),
                selected: selected.contains(o.value),
                onSelected: enabled
                    ? (on) => setState(() {
                          final next = [...selected];
                          on ? next.add(o.value) : next.remove(o.value);
                          _values[name] = next;
                        })
                    : null,
              ),
          ]),
        );

      case 'date':
      case 'datetime':
        final withTime = type == 'datetime';
        final current = _values[name];
        return FormField<String>(
          key: Key('field-$name'),
          initialValue: current == null ? '' : '$current',
          validator: (_) =>
              requiredCheck(_values[name] == null ? '' : '${_values[name]}'),
          builder: (state) => InkWell(
            onTap: enabled ? () => _pickDate(name, withTime, state) : null,
            child: InputDecorator(
              decoration: decoration(
                      suffixIcon:
                          const Icon(Icons.calendar_today_outlined, size: 18))
                  .copyWith(errorText: state.errorText),
              child: Text(_values[name] == null
                  ? ''
                  : formatDateTime(_values[name], withTime: withTime)),
            ),
          ),
        );
    }

    return _unsupported(
        context, label, 'This field type cannot be edited here yet.');
  }

  Future<void> _pickDate(
      String name, bool withTime, FormFieldState<String> state) async {
    final initial = DateTime.tryParse('${_values[name] ?? ''}')?.toLocal() ??
        DateTime.now();
    final date = await showDatePicker(
        context: context,
        initialDate: initial,
        firstDate: DateTime(1900),
        lastDate: DateTime(2200));
    if (date == null || !mounted) return;
    var result = date;
    if (withTime) {
      final time = await showTimePicker(
          context: context, initialTime: TimeOfDay.fromDateTime(initial));
      if (time == null || !mounted) return;
      result =
          DateTime(date.year, date.month, date.day, time.hour, time.minute);
    }
    setState(() => _values[name] =
        withTime ? result.toUtc().toIso8601String() : formatDate(result));
    state.didChange('${_values[name]}');
  }
}

/// Select input: a dropdown for short lists, a searchable picker for long or
/// server-searched lists.
class _SelectInput extends StatelessWidget {
  const _SelectInput({
    super.key,
    required this.api,
    required this.slug,
    required this.field,
    required this.value,
    required this.valueLabel,
    required this.enabled,
    required this.decoration,
    required this.validator,
    required this.onChanged,
  });

  final AdminApi api;
  final String slug;
  final Map<String, dynamic> field;
  final String? value;
  final String? valueLabel;
  final bool enabled;
  final InputDecoration decoration;
  final String? Function(String?) validator;
  final void Function(String? value, String? label) onChanged;

  @override
  Widget build(BuildContext context) {
    final options = FieldOption.listFrom(field['options']);
    final remote = field['remote'] == true;
    final required = field['required'] == true;

    if (!remote && options.length <= 12 && field['searchable'] != true) {
      final known = options.any((o) => o.value == value);
      return DropdownButtonFormField<String>(
        initialValue: known ? value : null,
        isExpanded: true,
        decoration: decoration,
        validator: validator,
        items: [
          if (!required)
            const DropdownMenuItem<String>(value: '', child: Text('—')),
          for (final o in options)
            DropdownMenuItem<String>(
                value: o.value,
                child: Text(o.label, overflow: TextOverflow.ellipsis)),
        ],
        onChanged: enabled
            ? (v) => onChanged((v == null || v.isEmpty) ? null : v, null)
            : null,
      );
    }

    String display = '';
    if (value != null && value!.isNotEmpty) {
      final match = options.where((o) => o.value == value);
      display = match.isNotEmpty ? match.first.label : (valueLabel ?? value!);
    }

    return FormField<String>(
      initialValue: value,
      validator: (_) => validator(value),
      builder: (state) => InkWell(
        onTap: enabled
            ? () async {
                final picked = await showDialog<FieldOption>(
                  context: context,
                  builder: (_) => _OptionPicker(
                    title: '${field['label'] ?? ''}',
                    options: options,
                    search: remote
                        ? (q) => api.options(slug, '${field['name']}', q)
                        : null,
                    allowClear: !required,
                  ),
                );
                if (picked == null) return;
                onChanged(
                    picked.value.isEmpty ? null : picked.value, picked.label);
                state.didChange(picked.value);
              }
            : null,
        child: InputDecorator(
          decoration: decoration.copyWith(
              errorText: state.errorText,
              suffixIcon: const Icon(Icons.unfold_more, size: 18)),
          isEmpty: display.isEmpty,
          child: Text(display, maxLines: 1, overflow: TextOverflow.ellipsis),
        ),
      ),
    );
  }
}

class _OptionPicker extends StatefulWidget {
  const _OptionPicker(
      {required this.title,
      required this.options,
      required this.search,
      required this.allowClear});

  final String title;
  final List<FieldOption> options;
  final Future<List<FieldOption>> Function(String query)? search;
  final bool allowClear;

  @override
  State<_OptionPicker> createState() => _OptionPickerState();
}

class _OptionPickerState extends State<_OptionPicker> {
  final _query = TextEditingController();
  Timer? _debounce;
  List<FieldOption> _results = const [];
  bool _loading = false;
  int _request = 0;

  @override
  void initState() {
    super.initState();
    _run('');
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _query.dispose();
    super.dispose();
  }

  Future<void> _run(String q) async {
    if (widget.search == null) {
      final needle = q.trim().toLowerCase();
      setState(() => _results = needle.isEmpty
          ? widget.options
          : widget.options
              .where((o) => o.label.toLowerCase().contains(needle))
              .toList());
      return;
    }
    final request = ++_request;
    setState(() => _loading = true);
    try {
      final found = await widget.search!(q.trim());
      if (mounted && request == _request) setState(() => _results = found);
    } catch (_) {
      if (mounted && request == _request) setState(() => _results = const []);
    } finally {
      if (mounted && request == _request) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Dialog(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 460, maxHeight: 560),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
            child: TextField(
              key: const Key('picker-search'),
              controller: _query,
              autofocus: true,
              decoration: InputDecoration(
                hintText: 'Search ${widget.title}'.trim(),
                prefixIcon: const Icon(Icons.search, size: 20),
              ),
              onChanged: (v) {
                _debounce?.cancel();
                _debounce = Timer(
                    Duration(milliseconds: widget.search == null ? 0 : 300),
                    () => _run(v));
              },
            ),
          ),
          if (_loading) const LinearProgressIndicator(minHeight: 2),
          Flexible(
            child: ListView(shrinkWrap: true, children: [
              if (widget.allowClear)
                ListTile(
                  dense: true,
                  leading: const Icon(Icons.clear, size: 18),
                  title: const Text('None'),
                  onTap: () => Navigator.pop(context, FieldOption('', '')),
                ),
              for (final o in _results)
                ListTile(
                  key: Key('option-${o.value}'),
                  dense: true,
                  title: Text(o.label),
                  subtitle: o.description == null ? null : Text(o.description!),
                  onTap: () => Navigator.pop(context, o),
                ),
              if (_results.isEmpty && !_loading)
                Padding(
                  padding: const EdgeInsets.all(20),
                  child: Text('No matches',
                      textAlign: TextAlign.center,
                      style:
                          TextStyle(color: theme.colorScheme.onSurfaceVariant)),
                ),
            ]),
          ),
          Align(
            alignment: Alignment.centerRight,
            child: Padding(
              padding: const EdgeInsets.all(8),
              child: TextButton(
                  onPressed: () => Navigator.pop(context),
                  child: const Text('Cancel')),
            ),
          ),
        ]),
      ),
    );
  }
}
