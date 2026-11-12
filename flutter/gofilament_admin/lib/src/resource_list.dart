import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'api.dart';
import 'cells.dart';
import 'controller.dart';
import 'export.dart';
import 'format.dart';
import 'icons.dart';
import 'links.dart';
import 'resource_form.dart';
import 'theme.dart';

/// Lists the records of a resource: search, sort, filters, paging, inline
/// toggles, row and bulk actions and client-side export.
class ResourceListPage extends StatefulWidget {
  const ResourceListPage(
      {super.key, required this.admin, required this.slug, this.openRecord});

  final AdminController admin;
  final String slug;

  /// Opens this record's form once the list is shown ("new" = create form).
  final String? openRecord;

  @override
  State<ResourceListPage> createState() => _ResourceListPageState();
}

class _ResourceListPageState extends State<ResourceListPage> {
  final _search = TextEditingController();
  Timer? _debounce;

  List<Map<String, dynamic>> _rows = const [];
  int _total = 0;
  int _page = 1;
  late int _perPage;
  String _sort = '';
  bool _descending = false;
  final Map<String, String> _filters = {};
  final Set<String> _selected = {};
  Map<String, Map<String, dynamic>> _rowMeta = const {};
  bool _loading = true;
  bool _exporting = false;
  String? _error;
  int _requestId = 0;

  AdminController get admin => widget.admin;
  AdminApi get api => admin.api;
  Map<String, dynamic> get _resource => admin.resource(widget.slug) ?? const {};
  Map<String, dynamic> get _meta => admin.resourceMeta(widget.slug);

  /// Columns the user chose to show (null: the server's defaults).
  Set<String>? _shown;

  List<Map<String, dynamic>> get _allColumns {
    final cols = _resource['columns'];
    return cols is List
        ? cols
            .whereType<Map>()
            .map((m) => Map<String, dynamic>.from(m))
            .toList()
        : <Map<String, dynamic>>[];
  }

  bool _isShown(Map<String, dynamic> c) {
    final shown = _shown;
    if (shown == null || c['toggleable'] != true) {
      return c['toggled_hidden_by_default'] != true;
    }
    return shown.contains('${c['name']}');
  }

  List<Map<String, dynamic>> get _columns =>
      _allColumns.where(_isShown).toList();

  String get _columnsPref => 'columns:${widget.slug}';

  Future<void> _loadColumnChoice() async {
    try {
      final raw = await api.readPreference(_columnsPref);
      if (raw == null || !mounted) return;
      final list = json.decode(raw);
      if (list is List) setState(() => _shown = list.map((e) => '$e').toSet());
    } catch (_) {}
  }

  void _toggleColumn(String name) {
    final shown = _shown ??
        _allColumns
            .where((c) => c['toggled_hidden_by_default'] != true)
            .map((c) => '${c['name']}')
            .toSet();
    final next = {...shown};
    if (!next.remove(name)) next.add(name);
    if (_allColumns.where((c) => next.contains('${c['name']}')).isEmpty) {
      return; // keep at least one column
    }
    setState(() => _shown = next);
    api.writePreference(_columnsPref, json.encode(next.toList()..sort()));
  }

  List<Map<String, dynamic>> get _filterDefs {
    final f = _resource['filters'];
    return f is List
        ? f.whereType<Map>().map((m) => Map<String, dynamic>.from(m)).toList()
        : const [];
  }

  Set<String> get _actions {
    final table = _meta['table'];
    final list = table is Map ? table['actions'] : null;
    return list is List
        ? list.whereType<Map>().map((a) => '${a['name']}').toSet()
        : const {};
  }

  List<Map<String, dynamic>> get _customActions {
    final table = _meta['table'];
    final list = table is Map ? table['actions'] : null;
    return list is List
        ? list
            .whereType<Map>()
            .map((a) => Map<String, dynamic>.from(a))
            .where((a) => a['custom'] == true)
            .toList()
        : const [];
  }

  bool _rowAllows(Map<String, dynamic> row, String ability) {
    final m = _rowMeta[recordId(row)];
    return m == null || m[ability] != false;
  }

  List<Map<String, dynamic>> _rowCustomActions(Map<String, dynamic> row) {
    final m = _rowMeta[recordId(row)];
    final allowed = m?['actions'];
    if (allowed is! List) return const [];
    return _customActions.where((a) => allowed.contains(a['name'])).toList();
  }

  Future<void> _runAction(
      Map<String, dynamic> row, Map<String, dynamic> action) async {
    final id = recordId(row);
    final name = '${action['name']}';
    if (action['link'] == true) {
      final links = _rowMeta[id]?['links'];
      final url = links is Map ? links[name]?.toString() : null;
      if (url == null || url.isEmpty) return;
      if (!await openLink(url)) {
        await Clipboard.setData(ClipboardData(text: url));
        _toast('Link copied');
      }
      return;
    }
    if (action['has_form'] == true) {
      final done = await showResourceForm(context,
          admin: admin, slug: widget.slug, id: id, action: action);
      if (done == true) {
        _toast('${action['label'] ?? 'Done'}');
        _load();
      }
      return;
    }
    if (action['requires_confirmation'] == true) {
      final ok = await showDialog<bool>(
        context: context,
        builder: (ctx) => AlertDialog(
          title: Text(
              '${action['modal_heading'] ?? '${action['label'] ?? name}?'}'),
          content: Text(
              '${action['modal_description'] ?? 'Do you want to continue?'}'),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(ctx, false),
                child: const Text('Cancel')),
            FilledButton(
              key: const Key('confirm-action'),
              onPressed: () => Navigator.pop(ctx, true),
              child: Text('${action['label'] ?? 'Continue'}'),
            ),
          ],
        ),
      );
      if (ok != true) return;
    }
    try {
      final message = await api.runAction(widget.slug, id, name);
      _toast(message);
      _load();
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  Set<String> get _bulkActions {
    final table = _meta['table'];
    final list = table is Map ? table['bulk_actions'] : null;
    return list is List
        ? list.whereType<Map>().map((a) => '${a['name']}').toSet()
        : const {};
  }

  bool get _exportEnabled {
    final e = admin.schema['export'];
    return !(e is Map && e['enabled'] == false);
  }

  bool get _hasForm => _meta['has_form'] == true;
  bool get _canCreate => _hasForm && admin.can(widget.slug, 'create');
  bool get _canDeleteAny => admin.can(widget.slug, 'delete_any');

  @override
  void initState() {
    super.initState();
    _perPage = (_resource['per_page'] as num?)?.toInt() ?? 25;
    _loadColumnChoice();
    _load();
    final open = widget.openRecord;
    if (open != null && _hasForm) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) return;
        if (open == 'new') {
          if (_canCreate) _openForm();
        } else {
          _openForm(id: open, readOnly: !_actions.contains('edit'));
        }
      });
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _search.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    final request = ++_requestId;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final page = await api.list(widget.slug,
          page: _page,
          perPage: _perPage,
          search: _search.text.trim(),
          sort: _sort,
          descending: _descending,
          filters: _filters);
      if (!mounted || request != _requestId) return;
      setState(() {
        _rows = page.rows;
        _rowMeta = page.rowMeta;
        _total = page.total;
        _loading = false;
        _selected.removeWhere((id) => !_rows.any((r) => recordId(r) == id));
      });
    } on ApiException catch (e) {
      if (!mounted || request != _requestId) return;
      setState(() {
        _error = e.message;
        _loading = false;
      });
    } catch (_) {
      if (!mounted || request != _requestId) return;
      setState(() {
        _error = 'Could not load the records. Please try again.';
        _loading = false;
      });
    }
  }

  void _onSearchChanged(String _) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 350), () {
      _page = 1;
      _load();
    });
  }

  void _toast(String message, {bool error = false}) {
    if (!mounted) return;
    final scheme = Theme.of(context).colorScheme;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(
          content: Text(message),
          backgroundColor: error ? scheme.error : null));
  }

  Future<void> _openForm({String? id, bool readOnly = false}) async {
    final saved = await showResourceForm(context,
        admin: admin, slug: widget.slug, id: id, readOnly: readOnly);
    if (saved == true) {
      _toast(id == null ? 'Created' : 'Saved');
      _load();
    }
  }

  Future<bool> _confirm(String title, String message) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(title),
        content: Text(message),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: const Text('Cancel')),
          FilledButton(
            key: const Key('confirm-delete'),
            style: FilledButton.styleFrom(
                backgroundColor: Theme.of(ctx).colorScheme.error),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    return ok == true;
  }

  Future<void> _delete(Map<String, dynamic> row) async {
    if (!await _confirm('Delete this record?', 'This cannot be undone.')) {
      return;
    }
    try {
      await api.delete(widget.slug, recordId(row));
      _toast('Deleted');
      _load();
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  Future<void> _restore(Map<String, dynamic> row) async {
    try {
      await api.restoreRecord(widget.slug, recordId(row));
      _toast('Restored');
      _load();
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  Future<void> _forceDelete(Map<String, dynamic> row) async {
    if (!await _confirm(
        'Delete this record for good?', 'It cannot be restored afterwards.')) {
      return;
    }
    try {
      await api.forceDelete(widget.slug, recordId(row));
      _toast('Deleted for good');
      _load();
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  Future<void> _deleteSelected() async {
    final ids = _selected.toList();
    if (ids.isEmpty) return;
    if (!await _confirm(
        'Delete ${ids.length} record${ids.length == 1 ? '' : 's'}?',
        'This cannot be undone.')) {
      return;
    }
    try {
      await api.deleteMany(widget.slug, ids);
      _selected.clear();
      _toast('Deleted');
      _load();
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  Future<void> _toggleCell(
      Map<String, dynamic> row, String column, bool value) async {
    final previous = row[column];
    setState(() => row[column] = value);
    try {
      await api.updateCell(widget.slug, recordId(row), column, value);
    } on ApiException catch (e) {
      if (mounted) setState(() => row[column] = previous);
      _toast(e.message, error: true);
    }
  }

  // ------------------------------------------------------------------ export

  String _exportText(Map<String, dynamic> column, dynamic value) {
    switch ('${column['type']}') {
      case 'money':
        final n = value is num ? value : num.tryParse('$value');
        return n == null ? '' : n.toStringAsFixed(2);
      case 'datetime':
        return formatDateTime(value);
      default:
        return displayValue(value);
    }
  }

  Future<void> _export(String format) async {
    if (_exporting) return;
    setState(() => _exporting = true);
    try {
      final columns = _columns.where((c) => c['type'] != 'image').toList();
      final limits = admin.schema['export'];
      final maxRows =
          (limits is Map ? (limits['max_rows'] as num?)?.toInt() : null) ??
              5000;
      final chunkSize =
          ((limits is Map ? (limits['max_per_page'] as num?)?.toInt() : null) ??
                  200)
              .clamp(1, 1000);

      var rows = <Map<String, dynamic>>[];
      if (_selected.isNotEmpty) {
        rows = _rows.where((r) => _selected.contains(recordId(r))).toList();
      } else {
        // page through the current search / filter result, capped
        var page = 1;
        while (rows.length < maxRows) {
          final chunk = await api.list(widget.slug,
              page: page,
              perPage: chunkSize,
              search: _search.text.trim(),
              sort: _sort,
              descending: _descending,
              filters: _filters);
          rows.addAll(chunk.rows);
          if (chunk.rows.isEmpty || rows.length >= chunk.total) break;
          page++;
        }
        if (rows.length > maxRows) rows = rows.sublist(0, maxRows);
      }

      final export = TableExport(
        headers: [
          for (final c in columns) '${c['label'] ?? humanize('${c['name']}')}'
        ],
        rows: [
          for (final r in rows)
            [
              for (final c in columns)
                _exportText(c, valueAt(r, '${c['name']}'))
            ]
        ],
      );
      final name = '${widget.slug}-${formatDate(DateTime.now())}';
      final ok = format == 'xlsx'
          ? await saveFile(
              '$name.xlsx',
              export.toXlsx(
                  sheetName: '${_resource['plural_model_name'] ?? 'Export'}'),
              'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')
          : await saveFile('$name.csv', export.toCsv(), 'text/csv');
      if (ok) {
        _toast('Exported ${rows.length} row${rows.length == 1 ? '' : 's'}'
            '${rows.length >= maxRows && _selected.isEmpty ? ' (limit reached)' : ''}');
      } else {
        _toast('Export is not available on this device yet.', error: true);
      }
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    } finally {
      if (mounted) setState(() => _exporting = false);
    }
  }

  // ------------------------------------------------------------------ build

  @override
  Widget build(BuildContext context) {
    final wide = MediaQuery.sizeOf(context).width >= 760;
    return Padding(
      padding: EdgeInsets.all(wide ? 20 : 12),
      child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        _toolbar(context, wide),
        const SizedBox(height: 12),
        if (_selected.isNotEmpty) _selectionBar(context),
        Expanded(
          child: Card(
            clipBehavior: Clip.antiAlias,
            child: Column(children: [
              if (_loading)
                const LinearProgressIndicator(minHeight: 2)
              else
                const SizedBox(height: 2),
              Expanded(child: _content(context, wide)),
              const Divider(),
              _pagination(context),
            ]),
          ),
        ),
      ]),
    );
  }

  Widget _toolbar(BuildContext context, bool wide) {
    final searchField = TextField(
      key: const Key('list-search'),
      controller: _search,
      onChanged: _onSearchChanged,
      onSubmitted: (_) {
        _debounce?.cancel();
        _page = 1;
        _load();
      },
      decoration: InputDecoration(
        hintText: 'Search',
        prefixIcon: const Icon(Icons.search, size: 20),
        suffixIcon: _search.text.isEmpty
            ? null
            : IconButton(
                tooltip: 'Clear',
                icon: const Icon(Icons.close, size: 18),
                onPressed: () {
                  _search.clear();
                  _page = 1;
                  _load();
                },
              ),
      ),
    );

    final filters = [for (final f in _filterDefs) _filterChip(context, f)];

    final actions = <Widget>[
      IconButton(
          tooltip: 'Refresh',
          icon: const Icon(Icons.refresh),
          onPressed: _load),
      if (_allColumns.any((c) => c['toggleable'] == true))
        PopupMenuButton<String>(
          key: const Key('list-columns'),
          tooltip: 'Columns',
          icon: const Icon(Icons.view_column_outlined),
          onSelected: _toggleColumn,
          itemBuilder: (_) => [
            for (final c in _allColumns.where((c) => c['toggleable'] == true))
              CheckedPopupMenuItem<String>(
                key: Key('column-toggle-${c['name']}'),
                value: '${c['name']}',
                checked: _isShown(c),
                child: Text('${c['label'] ?? humanize('${c['name']}')}'),
              ),
          ],
        ),
      if (_exportEnabled)
        PopupMenuButton<String>(
          key: const Key('list-export'),
          tooltip: 'Export',
          enabled: !_exporting,
          onSelected: _export,
          itemBuilder: (_) => const [
            PopupMenuItem(value: 'csv', child: Text('Export CSV')),
            PopupMenuItem(value: 'xlsx', child: Text('Export Excel')),
          ],
          child: Padding(
            padding: const EdgeInsets.all(8),
            child: _exporting
                ? const SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(strokeWidth: 2))
                : const Icon(Icons.download_outlined),
          ),
        ),
      if (_canCreate) ...[
        const SizedBox(width: 8),
        FilledButton.icon(
          key: const Key('list-create'),
          onPressed: () => _openForm(),
          icon: const Icon(Icons.add, size: 18),
          label: Text(
              wide ? 'New ${_resource['model_name'] ?? ''}'.trim() : 'New'),
        ),
      ],
    ];

    if (wide) {
      return Row(children: [
        SizedBox(width: 300, child: searchField),
        const SizedBox(width: 12),
        Expanded(
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: Row(children: [
              for (final f in filters)
                Padding(padding: const EdgeInsets.only(right: 8), child: f)
            ]),
          ),
        ),
        ...actions,
      ]);
    }
    return Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
      Row(children: [Expanded(child: searchField), ...actions]),
      if (filters.isNotEmpty) ...[
        const SizedBox(height: 8),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(children: [
            for (final f in filters)
              Padding(padding: const EdgeInsets.only(right: 8), child: f)
          ]),
        ),
      ],
    ]);
  }

  Widget _filterChip(BuildContext context, Map<String, dynamic> filter) {
    final theme = Theme.of(context);
    final column = '${filter['column'] ?? filter['name']}';
    final options = FieldOption.listFrom(filter['options']);
    final current = _filters[column];
    final selected = options.where((o) => o.value == current);
    final label = '${filter['label'] ?? humanize(column)}';

    return PopupMenuButton<String>(
      key: Key('filter-$column'),
      tooltip: label,
      onSelected: (v) {
        setState(() {
          if (v.isEmpty) {
            _filters.remove(column);
          } else {
            _filters[column] = v;
          }
          _page = 1;
        });
        _load();
      },
      itemBuilder: (_) => [
        const PopupMenuItem(value: '', child: Text('All')),
        for (final o in options)
          PopupMenuItem(value: o.value, child: Text(o.label)),
      ],
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(10),
          border: Border.all(
              color: selected.isEmpty
                  ? theme.dividerColor
                  : theme.colorScheme.primary),
          color: selected.isEmpty
              ? null
              : theme.colorScheme.primary.withValues(alpha: 0.08),
        ),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          Icon(Icons.filter_list,
              size: 16,
              color: selected.isEmpty
                  ? theme.colorScheme.onSurfaceVariant
                  : theme.colorScheme.primary),
          const SizedBox(width: 6),
          Text(selected.isEmpty ? label : '$label: ${selected.first.label}',
              style: const TextStyle(fontSize: 13)),
        ]),
      ),
    );
  }

  Widget _selectionBar(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 6),
      decoration: BoxDecoration(
        color: theme.colorScheme.primary.withValues(alpha: 0.08),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Row(children: [
        Text('${_selected.length} selected',
            style: const TextStyle(fontWeight: FontWeight.w600)),
        const Spacer(),
        TextButton(
            onPressed: () => setState(_selected.clear),
            child: const Text('Clear')),
        if (_canDeleteAny && _bulkActions.contains('delete_bulk'))
          TextButton.icon(
            key: const Key('bulk-delete'),
            onPressed: _deleteSelected,
            icon: Icon(Icons.delete_outline,
                size: 18, color: theme.colorScheme.error),
            label: Text('Delete',
                style: TextStyle(color: theme.colorScheme.error)),
          ),
      ]),
    );
  }

  Widget _content(BuildContext context, bool wide) {
    final theme = Theme.of(context);
    if (_error != null) {
      return Center(
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Text(_error!, style: TextStyle(color: theme.colorScheme.error)),
          const SizedBox(height: 12),
          OutlinedButton(onPressed: _load, child: const Text('Try again')),
        ]),
      );
    }
    if (_rows.isEmpty && !_loading) {
      final filtered = _search.text.isNotEmpty || _filters.isNotEmpty;
      return Center(
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Icon(Icons.inbox_outlined,
              size: 44, color: theme.colorScheme.onSurfaceVariant),
          const SizedBox(height: 10),
          Text(filtered ? 'No records match your search.' : 'No records yet.',
              style: TextStyle(color: theme.colorScheme.onSurfaceVariant)),
        ]),
      );
    }
    return wide ? _table(context) : _cards(context);
  }

  bool get _selectable => _bulkActions.isNotEmpty;

  Widget _table(BuildContext context) {
    final columns = _columns;
    final sortIndex = columns.indexWhere((c) => c['name'] == _sort);
    final hasRowActions =
        _actions.intersection({'view', 'edit', 'delete'}).isNotEmpty;

    return LayoutBuilder(
      builder: (context, constraints) => SingleChildScrollView(
        child: SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: ConstrainedBox(
            constraints: BoxConstraints(minWidth: constraints.maxWidth),
            child: DataTable(
              showCheckboxColumn: _selectable,
              sortColumnIndex: sortIndex < 0 ? null : sortIndex,
              sortAscending: !_descending,
              columnSpacing: 28,
              horizontalMargin: 18,
              columns: [
                for (final c in columns)
                  DataColumn(
                    label: Text('${c['label'] ?? humanize('${c['name']}')}'
                        .toUpperCase()),
                    numeric: c['type'] == 'money' || c['align'] == 'right',
                    onSort: c['sortable'] == true
                        ? (_, ascending) {
                            setState(() {
                              _sort = '${c['name']}';
                              _descending = !ascending;
                              _page = 1;
                            });
                            _load();
                          }
                        : null,
                  ),
                if (hasRowActions) const DataColumn(label: Text('')),
              ],
              rows: [
                for (final row in _rows)
                  DataRow(
                    key: ValueKey(recordId(row)),
                    selected: _selected.contains(recordId(row)),
                    onSelectChanged: _selectable
                        ? (v) => setState(() => v == true
                            ? _selected.add(recordId(row))
                            : _selected.remove(recordId(row)))
                        : null,
                    cells: [
                      for (final c in columns) DataCell(_cell(context, c, row)),
                      if (hasRowActions) DataCell(_rowActions(context, row)),
                    ],
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _cards(BuildContext context) {
    final theme = Theme.of(context);
    final columns = _columns;
    return ListView.separated(
      itemCount: _rows.length,
      separatorBuilder: (_, __) => const Divider(),
      itemBuilder: (context, i) {
        final row = _rows[i];
        final title = columns.isEmpty
            ? recordId(row)
            : displayValue(valueAt(row, '${columns.first['name']}'));
        return InkWell(
          onTap:
              _actions.contains('edit') && _hasForm && _rowAllows(row, 'update')
                  ? () => _openForm(id: recordId(row))
                  : (_actions.contains('view') && _hasForm
                      ? () => _openForm(id: recordId(row), readOnly: true)
                      : null),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
            child:
                Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Expanded(
                    child: Text(title,
                        style: const TextStyle(
                            fontWeight: FontWeight.w600, fontSize: 15))),
                _rowActions(context, row),
              ]),
              const SizedBox(height: 4),
              Wrap(spacing: 16, runSpacing: 6, children: [
                for (final c in columns.skip(1))
                  Row(mainAxisSize: MainAxisSize.min, children: [
                    Text('${c['label'] ?? humanize('${c['name']}')}: ',
                        style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant)),
                    _cell(context, c, row),
                  ]),
              ]),
            ]),
          ),
        );
      },
    );
  }

  Widget _rowActions(BuildContext context, Map<String, dynamic> row) {
    final theme = Theme.of(context);
    final id = recordId(row);
    return Row(mainAxisSize: MainAxisSize.min, children: [
      if (_actions.contains('view') && _hasForm)
        IconButton(
          key: Key('row-view-$id'),
          tooltip: 'View',
          visualDensity: VisualDensity.compact,
          icon: const Icon(Icons.visibility_outlined, size: 19),
          onPressed: () => _openForm(id: id, readOnly: true),
        ),
      for (final action in _rowCustomActions(row))
        IconButton(
          key: Key('row-action-${action['name']}-$id'),
          tooltip: '${action['label'] ?? action['name']}',
          visualDensity: VisualDensity.compact,
          icon: Icon(
              adminIcon(action['icon'] as String?,
                  fallback: Icons.play_circle_outline),
              size: 19,
              color: AdminPalette.of(context)
                  .color('${action['color'] ?? 'primary'}', 500)),
          onPressed: () => _runAction(row, action),
        ),
      if (_actions.contains('edit') && _hasForm && _rowAllows(row, 'update'))
        IconButton(
          key: Key('row-edit-$id'),
          tooltip: 'Edit',
          visualDensity: VisualDensity.compact,
          icon: const Icon(Icons.edit_outlined, size: 19),
          onPressed: () => _openForm(id: id),
        ),
      if (_rowMeta[id]?['restore'] == true)
        IconButton(
          key: Key('row-restore-$id'),
          tooltip: 'Restore',
          visualDensity: VisualDensity.compact,
          icon: const Icon(Icons.restore_from_trash_outlined, size: 19),
          onPressed: () => _restore(row),
        ),
      if (_rowMeta[id]?['force_delete'] == true)
        IconButton(
          key: Key('row-force-delete-$id'),
          tooltip: 'Delete for good',
          visualDensity: VisualDensity.compact,
          icon: Icon(Icons.delete_forever_outlined,
              size: 19, color: theme.colorScheme.error),
          onPressed: () => _forceDelete(row),
        ),
      if (_actions.contains('delete') &&
          _canDeleteAny &&
          _rowAllows(row, 'delete'))
        IconButton(
          key: Key('row-delete-$id'),
          tooltip: 'Delete',
          visualDensity: VisualDensity.compact,
          icon: Icon(Icons.delete_outline,
              size: 19, color: theme.colorScheme.error),
          onPressed: () => _delete(row),
        ),
    ]);
  }

  Widget _cell(BuildContext context, Map<String, dynamic> column,
      Map<String, dynamic> row) {
    return adminCell(context, column, row,
        onToggle: (name, v) => _toggleCell(row, name, v),
        onNotice: _toast,
        currency: '${admin.schema['currency'] ?? ''}',
        admin: admin);
  }

  Widget _pagination(BuildContext context) {
    final theme = Theme.of(context);
    final pages = _total == 0 ? 1 : ((_total - 1) ~/ _perPage) + 1;
    final from = _total == 0 ? 0 : (_page - 1) * _perPage + 1;
    final to = (_page * _perPage) > _total ? _total : _page * _perPage;
    final raw = _resource['per_page_options'];
    final options = <int>{
      ...(raw is List
          ? raw.whereType<num>().map((n) => n.toInt())
          : const <int>[10, 25, 50, 100]),
      _perPage,
    }.toList()
      ..sort();

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 6),
      child: Row(children: [
        Expanded(
          child: Text('$from–$to of $_total',
              key: const Key('list-count'),
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
        ),
        DropdownButton<int>(
          value: _perPage,
          underline: const SizedBox.shrink(),
          isDense: true,
          style: theme.textTheme.bodySmall
              ?.copyWith(color: theme.colorScheme.onSurface),
          items: [
            for (final o in options)
              DropdownMenuItem(value: o, child: Text('$o / page'))
          ],
          onChanged: (v) {
            if (v == null) return;
            setState(() {
              _perPage = v;
              _page = 1;
            });
            _load();
          },
        ),
        IconButton(
          key: const Key('list-prev'),
          tooltip: 'Previous page',
          icon: const Icon(Icons.chevron_left),
          onPressed: _page > 1
              ? () {
                  setState(() => _page--);
                  _load();
                }
              : null,
        ),
        Text('$_page / $pages', style: theme.textTheme.bodySmall),
        IconButton(
          key: const Key('list-next'),
          tooltip: 'Next page',
          icon: const Icon(Icons.chevron_right),
          onPressed: _page < pages
              ? () {
                  setState(() => _page++);
                  _load();
                }
              : null,
        ),
      ]),
    );
  }
}
