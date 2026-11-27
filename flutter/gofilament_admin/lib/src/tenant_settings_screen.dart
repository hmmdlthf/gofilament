import 'dart:convert';

import 'package:flutter/material.dart';

import 'api.dart';
import 'i18n.dart';

/// The workspace's own settings (Panel.Features on the server): the
/// features a tenant admin may change, grouped like on the web page. Bool
/// features are switches, enums dropdowns, ints number fields, strings text
/// fields and JSON values a monospaced editor.
///
/// Features only the platform may change are never sent here, and the
/// server refuses them if a client tries.
class TenantSettingsScreen extends StatefulWidget {
  const TenantSettingsScreen(
      {super.key, required this.api, required this.path, this.title});

  final AdminApi api;

  /// The endpoint relative to the panel API (schema tenant_settings.path).
  final String path;
  final String? title;

  @override
  State<TenantSettingsScreen> createState() => _TenantSettingsScreenState();
}

class _TenantSettingsScreenState extends State<TenantSettingsScreen> {
  List<Map<String, dynamic>> _groups = const [];
  bool _loading = true;
  bool _saving = false;
  String? _error;

  /// Edited values by key (text for text fields, bool for switches).
  final Map<String, Object> _edits = {};
  final Map<String, String> _fieldErrors = {};
  final Map<String, TextEditingController> _controllers = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    for (final c in _controllers.values) {
      c.dispose();
    }
    super.dispose();
  }

  void _apply(Map<String, dynamic> data) {
    final g = data['groups'];
    _groups = g is List
        ? [
            for (final x in g)
              if (x is Map) Map<String, dynamic>.from(x)
          ]
        : const [];
    _edits.clear();
    for (final c in _controllers.values) {
      c.dispose();
    }
    _controllers.clear();
  }

  Future<void> _load() async {
    try {
      final d = await widget.api.tenantSettings(widget.path);
      if (!mounted) return;
      setState(() {
        _apply(d);
        _loading = false;
        _error = null;
      });
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _loading = false;
          _error = e.message;
        });
      }
    }
  }

  void _toast(String m) {
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(m)));
    }
  }

  static String _text(String type, Object? v) {
    switch (type) {
      case 'json':
        return const JsonEncoder.withIndent('  ').convert(v);
      case 'bool':
        return v == true ? 'true' : 'false';
      default:
        return v == null ? '' : '$v';
    }
  }

  TextEditingController _controller(Map<String, dynamic> f) {
    final key = '${f['key']}';
    return _controllers.putIfAbsent(key,
        () => TextEditingController(text: _text('${f['type']}', f['value'])));
  }

  Future<void> _save() async {
    if (_edits.isEmpty) return;
    setState(() => _saving = true);
    try {
      final d = await widget.api
          .saveTenantSettings(widget.path, Map<String, dynamic>.from(_edits));
      if (!mounted) return;
      setState(() {
        _apply(d);
        _fieldErrors.clear();
      });
      _toast(tr('Settings saved.'));
    } on ApiException catch (e) {
      final errs = e.body?['errors'];
      if (errs is Map && mounted) {
        setState(() {
          _fieldErrors
            ..clear()
            ..addAll(errs.map((k, v) => MapEntry('$k', '$v')));
        });
        _toast(tr('Fix the marked fields.'));
      } else {
        _toast(e.message);
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  Future<void> _reset(String key) async {
    try {
      final d = await widget.api.resetTenantSetting(widget.path, key);
      if (!mounted) return;
      setState(() {
        _apply(d);
        _fieldErrors.remove(key);
      });
    } on ApiException catch (e) {
      _toast(e.message);
    }
  }

  Widget _control(Map<String, dynamic> f) {
    final key = '${f['key']}';
    final type = '${f['type']}';
    final label = '${f['label'] ?? key}';
    final error = _fieldErrors[key];
    switch (type) {
      case 'bool':
        final on =
            _edits.containsKey(key) ? _edits[key] == true : f['value'] == true;
        return SwitchListTile(
          key: Key('tenant-setting-$key'),
          contentPadding: EdgeInsets.zero,
          title: Text(label),
          subtitle: error != null
              ? Text(error,
                  style: TextStyle(color: Theme.of(context).colorScheme.error))
              : null,
          value: on,
          onChanged: (v) => setState(() => _edits[key] = v),
        );
      case 'enum':
        final options = [for (final o in (f['options'] as List? ?? [])) '$o'];
        final current =
            _edits.containsKey(key) ? '${_edits[key]}' : '${f['value']}';
        return DropdownButtonFormField<String>(
          key: Key('tenant-setting-$key'),
          initialValue: options.contains(current) ? current : null,
          decoration: InputDecoration(labelText: label, errorText: error),
          items: [
            for (final o in options) DropdownMenuItem(value: o, child: Text(o))
          ],
          onChanged: (v) {
            if (v != null) setState(() => _edits[key] = v);
          },
        );
      default:
        final isJson = type == 'json';
        final min = f['min'], max = f['max'];
        return TextField(
          key: Key('tenant-setting-$key'),
          controller: _controller(f),
          keyboardType: type == 'int'
              ? TextInputType.number
              : (isJson ? TextInputType.multiline : TextInputType.text),
          minLines: isJson ? 3 : 1,
          maxLines: isJson ? 8 : 1,
          style: isJson ? const TextStyle(fontFamily: 'monospace') : null,
          decoration: InputDecoration(
            labelText: label,
            errorText: error,
            helperText: type == 'int' && min != null && max != null
                ? '$min – $max'
                : null,
          ),
          onChanged: (v) => setState(() => _edits[key] = v),
        );
    }
  }

  Widget _field(Map<String, dynamic> f) {
    final theme = Theme.of(context);
    final key = '${f['key']}';
    final overridden = f['overridden'] == true;
    final description = '${f['description'] ?? ''}';
    return Padding(
      key: Key('tenant-setting-row-$key'),
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          _control(f),
          const SizedBox(height: 4),
          Wrap(
            spacing: 8,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              Chip(
                visualDensity: VisualDensity.compact,
                label: Text(overridden ? tr('Overridden') : tr('Default')),
                backgroundColor: overridden
                    ? theme.colorScheme.tertiaryContainer
                    : theme.colorScheme.surfaceContainerHighest,
              ),
              if (f['managed_by'] == 'super_admin')
                Chip(
                  visualDensity: VisualDensity.compact,
                  label: Text(tr('Super admin only')),
                ),
              if (overridden)
                TextButton.icon(
                  key: Key('tenant-setting-reset-$key'),
                  icon: const Icon(Icons.restart_alt, size: 18),
                  label: Text(tr('Reset to default')),
                  onPressed: () => _reset(key),
                ),
            ],
          ),
          if (description.isNotEmpty)
            Text(description, style: theme.textTheme.bodySmall),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    Widget body;
    if (_loading) {
      body = const Center(child: CircularProgressIndicator());
    } else if (_error != null) {
      body = Center(child: Text(_error!));
    } else {
      body = ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Text(tr('Change how this workspace works'),
              style: theme.textTheme.bodyMedium),
          const SizedBox(height: 12),
          for (final g in _groups)
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    if ('${g['name'] ?? ''}'.isNotEmpty)
                      Text('${g['name']}', style: theme.textTheme.titleMedium),
                    for (final f in (g['features'] as List? ?? []))
                      if (f is Map) _field(Map<String, dynamic>.from(f)),
                  ],
                ),
              ),
            ),
        ],
      );
    }
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.title ?? tr('Settings')),
        actions: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: FilledButton(
              key: const Key('tenant-settings-save'),
              onPressed: _edits.isEmpty || _saving ? null : _save,
              child: Text(tr('Save')),
            ),
          ),
        ],
      ),
      body: body,
    );
  }
}
