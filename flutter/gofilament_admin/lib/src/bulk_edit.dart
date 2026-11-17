import 'package:flutter/material.dart';
import 'i18n.dart';

import 'api.dart';

/// Asks for a field and a value and sets it on the selected records.
/// Returns the server's result, or null when cancelled.
Future<Map<String, dynamic>?> showBulkEditDialog(BuildContext context,
    {required AdminApi api,
    required String slug,
    required List<String> ids,
    required List<String> fields}) {
  return showDialog<Map<String, dynamic>>(
    context: context,
    builder: (_) =>
        _BulkEditDialog(api: api, slug: slug, ids: ids, fields: fields),
  );
}

class _BulkEditDialog extends StatefulWidget {
  const _BulkEditDialog(
      {required this.api,
      required this.slug,
      required this.ids,
      required this.fields});
  final AdminApi api;
  final String slug;
  final List<String> ids;
  final List<String> fields;

  @override
  State<_BulkEditDialog> createState() => _BulkEditDialogState();
}

class _BulkEditDialogState extends State<_BulkEditDialog> {
  final Map<String, Map<String, dynamic>> _nodes = {};
  late String _field = widget.fields.first;
  dynamic _value;
  final _text = TextEditingController();
  String? _error;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    widget.api.form(widget.slug).then((f) {
      void walk(List<dynamic> ns) {
        for (final n in ns) {
          if (n is! Map) continue;
          if (n['kind'] == 'field' && widget.fields.contains(n['name'])) {
            _nodes['${n['name']}'] = Map<String, dynamic>.from(n);
          }
          if (n['children'] is List) walk(n['children'] as List);
        }
      }

      walk(f.schema);
      if (mounted) setState(() {});
    }).catchError((_) {});
  }

  @override
  void dispose() {
    _text.dispose();
    super.dispose();
  }

  Map<String, dynamic> get _node => _nodes[_field] ?? const {};
  String _label(String f) => '${_nodes[f]?['label'] ?? f}';

  Widget _input() {
    final type = '${_node['type'] ?? 'text'}';
    final options = FieldOption.listFrom(_node['options']);
    if (type == 'toggle' || type == 'checkbox') {
      return SwitchListTile(
        key: const Key('bulk-value'),
        contentPadding: EdgeInsets.zero,
        title: Text(_label(_field)),
        value: _value == true,
        onChanged: (v) => setState(() => _value = v),
      );
    }
    if (options.isNotEmpty) {
      return DropdownButtonFormField<String>(
        key: const Key('bulk-value'),
        initialValue: _value is String ? _value as String : null,
        decoration: InputDecoration(labelText: _label(_field)),
        items: [
          for (final o in options)
            DropdownMenuItem(value: o.value, child: Text(o.label)),
        ],
        onChanged: (v) => setState(() => _value = v),
      );
    }
    return TextField(
      key: const Key('bulk-value'),
      controller: _text,
      decoration: InputDecoration(labelText: _label(_field)),
      onChanged: (v) => _value = v,
    );
  }

  Future<void> _apply() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final res =
          await widget.api.bulkUpdate(widget.slug, widget.ids, _field, _value);
      if (mounted) Navigator.pop(context, res);
    } on ApiException catch (e) {
      final errs = e.body?['errors'];
      setState(() => _error =
          errs is Map && errs[_field] != null ? '${errs[_field]}' : e.message);
    } catch (_) {
      setState(() => _error = tr('Could not reach the server.'));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(tr('Change {n} records', {'n': widget.ids.length})),
      content: SizedBox(
        width: 380,
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          if (widget.fields.length > 1)
            DropdownButtonFormField<String>(
              key: const Key('bulk-field'),
              initialValue: _field,
              decoration: InputDecoration(labelText: tr('Field')),
              items: [
                for (final f in widget.fields)
                  DropdownMenuItem(value: f, child: Text(_label(f))),
              ],
              onChanged: (v) => setState(() {
                _field = v ?? _field;
                _value = null;
                _text.clear();
              }),
            ),
          const SizedBox(height: 12),
          KeyedSubtree(key: ValueKey(_field), child: _input()),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(_error!,
                  key: const Key('bulk-error'),
                  style: TextStyle(color: Theme.of(context).colorScheme.error)),
            ),
        ]),
      ),
      actions: [
        TextButton(
            onPressed: () => Navigator.pop(context), child: Text(tr('Cancel'))),
        FilledButton(
            key: const Key('bulk-apply'),
            onPressed: _busy ? null : _apply,
            child: Text(tr('Apply'))),
      ],
    );
  }
}
