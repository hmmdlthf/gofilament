import 'package:flutter/material.dart';

import 'api.dart';
import 'files.dart';

/// Imports a CSV, TSV or Excel file into a resource: the user picks the
/// file, matches its columns to fields, sees every row's problems, then
/// imports the valid rows. Returns the number of imported rows (null when
/// cancelled).
Future<int?> showImportDialog(BuildContext context,
    {required AdminApi api,
    required String slug,
    required String title}) async {
  final picked =
      await adminPickFiles(extensions: const ['csv', 'tsv', 'txt', 'xlsx']);
  if (picked.isEmpty || !context.mounted) return null;
  return showDialog<int>(
    context: context,
    builder: (_) =>
        _ImportDialog(api: api, slug: slug, title: title, file: picked.first),
  );
}

class _ImportDialog extends StatefulWidget {
  const _ImportDialog(
      {required this.api,
      required this.slug,
      required this.title,
      required this.file});
  final AdminApi api;
  final String slug;
  final String title;
  final PickedFile file;

  @override
  State<_ImportDialog> createState() => _ImportDialogState();
}

class _ImportDialogState extends State<_ImportDialog> {
  Map<String, dynamic>? _preview;
  Map<String, String> _mapping = {};
  Map<String, dynamic>? _result;
  String? _error;
  bool _busy = true;

  @override
  void initState() {
    super.initState();
    _check();
  }

  Future<void> _check() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final pv = await widget.api.importFile(
          widget.slug, widget.file.name, widget.file.bytes,
          mapping: _preview == null ? null : _mapping, dryRun: true);
      final m = pv['mapping'];
      setState(() {
        _preview = pv;
        _mapping = m is Map
            ? m.map((k, v) => MapEntry('$k', '${v ?? ''}'))
            : <String, String>{};
      });
    } on ApiException catch (e) {
      final fileErr =
          e.body?['errors'] is Map ? e.body!['errors']['file'] : null;
      setState(() => _error = '${fileErr ?? e.message}');
    } catch (_) {
      setState(() => _error = 'Could not reach the server.');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _import() async {
    setState(() => _busy = true);
    try {
      final res = await widget.api.importFile(
          widget.slug, widget.file.name, widget.file.bytes,
          mapping: _mapping);
      setState(() => _result = res);
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } catch (_) {
      setState(() => _error = 'Could not reach the server.');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  List<Map<String, dynamic>> _list(String key, [Map<String, dynamic>? from]) {
    final raw = (from ?? _preview)?[key];
    return raw is List
        ? [
            for (final r in raw)
              if (r is Map) Map<String, dynamic>.from(r)
          ]
        : const [];
  }

  static String _describe(Map<String, dynamic> row) {
    if ('${row['message'] ?? ''}'.isNotEmpty) return '${row['message']}';
    final errs = row['errors'];
    if (errs is! Map) return '';
    return errs.entries
        .map((e) => '${e.key}'.isEmpty ? '${e.value}' : '${e.key}: ${e.value}')
        .join('; ');
  }

  Widget _body(BuildContext context) {
    final theme = Theme.of(context);
    if (_result != null) {
      final failed = _list('rows', _result);
      return Column(
          key: const Key('import-result'),
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('${_result!['imported'] ?? 0} rows imported.',
                style: const TextStyle(fontWeight: FontWeight.w600)),
            if ((_result!['failed'] as num? ?? 0) > 0) ...[
              const SizedBox(height: 8),
              Text('${_result!['failed']} rows were skipped.',
                  style: TextStyle(color: theme.colorScheme.error)),
              for (final r in failed)
                Text('Row ${r['row']}: ${_describe(r)}',
                    style: theme.textTheme.bodySmall),
            ],
          ]);
    }
    final pv = _preview;
    if (pv == null) {
      return _error != null
          ? Text(_error!, style: TextStyle(color: theme.colorScheme.error))
          : const Center(child: CircularProgressIndicator());
    }
    final headers = [for (final h in (pv['headers'] as List? ?? [])) '$h'];
    final fields = _list('fields');
    final rows = _list('rows');
    final problems = _list('problems');
    final cols = <String>[];
    for (final h in headers) {
      final f = _mapping[h] ?? '';
      if (f.isNotEmpty && !cols.contains(f)) cols.add(f);
    }
    String label(String name) =>
        '${fields.firstWhere((f) => f['name'] == name, orElse: () => {
              'label': name
            })['label']}';

    return SingleChildScrollView(
      child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
                '${pv['total']} rows: ${pv['valid']} ready to import, ${pv['invalid']} with problems.',
                key: const Key('import-summary'),
                style: TextStyle(
                    fontWeight: FontWeight.w600,
                    color: (pv['invalid'] as num? ?? 0) > 0
                        ? theme.colorScheme.error
                        : null)),
            if (_error != null)
              Text(_error!, style: TextStyle(color: theme.colorScheme.error)),
            const SizedBox(height: 12),
            Text('Columns', style: theme.textTheme.titleSmall),
            for (final h in headers)
              Row(children: [
                Expanded(child: Text(h)),
                const Icon(Icons.arrow_forward, size: 16),
                const SizedBox(width: 8),
                Expanded(
                  child: DropdownButton<String>(
                    key: Key('import-map-$h'),
                    isExpanded: true,
                    value: _mapping[h] ?? '',
                    items: [
                      const DropdownMenuItem(
                          value: '', child: Text('Do not import')),
                      if (fields.isEmpty && (_mapping[h] ?? '').isNotEmpty)
                        DropdownMenuItem(
                            value: _mapping[h], child: Text(_mapping[h]!)),
                      for (final f in fields)
                        DropdownMenuItem(
                            value: '${f['name']}',
                            child: Text(
                                '${f['label']}${f['required'] == true ? ' *' : ''}')),
                    ],
                    onChanged: _busy
                        ? null
                        : (v) => setState(() => _mapping[h] = v ?? ''),
                  ),
                ),
              ]),
            Align(
              alignment: Alignment.centerLeft,
              child: TextButton(
                  key: const Key('import-recheck'),
                  onPressed: _busy ? null : _check,
                  child: const Text('Check again')),
            ),
            const SizedBox(height: 8),
            Text('First ${rows.length} rows',
                style: theme.textTheme.titleSmall),
            SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: DataTable(
                columns: [
                  const DataColumn(label: Text('Row')),
                  for (final c in cols) DataColumn(label: Text(label(c))),
                  const DataColumn(label: Text('Problems')),
                ],
                rows: [
                  for (final r in rows)
                    DataRow(
                        color: WidgetStatePropertyAll((r['errors'] is Map &&
                                (r['errors'] as Map).isNotEmpty)
                            ? theme.colorScheme.errorContainer
                            : null),
                        cells: [
                          DataCell(Text('${r['row']}')),
                          for (final c in cols)
                            DataCell(
                                Text('${(r['values'] as Map?)?[c] ?? ''}')),
                          DataCell(Text(_describe(r),
                              style:
                                  TextStyle(color: theme.colorScheme.error))),
                        ]),
                ],
              ),
            ),
            if (problems.length >
                rows.where((r) => r['errors'] != null).length) ...[
              const SizedBox(height: 8),
              Text('Rows that will be skipped',
                  style: theme.textTheme.titleSmall),
              for (final p in problems)
                Text('Row ${p['row']}: ${_describe(p)}',
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: theme.colorScheme.error)),
            ],
          ]),
    );
  }

  @override
  Widget build(BuildContext context) {
    final valid = (_preview?['valid'] as num?)?.toInt() ?? 0;
    return AlertDialog(
      title: Text('Import ${widget.title}'),
      content: SizedBox(width: 720, child: _body(context)),
      actions: _result != null
          ? [
              FilledButton(
                  key: const Key('import-done'),
                  onPressed: () => Navigator.pop(
                      context, (_result!['imported'] as num?)?.toInt() ?? 0),
                  child: const Text('Done')),
            ]
          : [
              TextButton(
                  onPressed: () => Navigator.pop(context),
                  child: const Text('Cancel')),
              FilledButton(
                key: const Key('import-run'),
                onPressed: _busy || valid == 0 ? null : _import,
                child: Text('Import $valid rows'),
              ),
            ],
    );
  }
}
