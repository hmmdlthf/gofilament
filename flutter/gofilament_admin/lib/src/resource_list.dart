import 'dart:async';
import 'dart:convert';
import 'i18n.dart';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'api.dart';
import 'board.dart';
import 'bulk_edit.dart';
import 'cells.dart';
import 'controller.dart';
import 'export.dart';
import 'format.dart';
import 'icons.dart';
import 'import_dialog.dart';
import 'links.dart';
import 'list_tools.dart';
import 'reorder_dialog.dart';
import 'resource_form.dart';
import 'responsive.dart';
import 'theme.dart';

/// Lists the records of a resource: search, sort, filters, paging, inline
/// toggles, row and bulk actions and client-side export.
/// How often a running background job is checked (tests shorten it).
Duration jobPollInterval = const Duration(seconds: 1);

class ResourceListPage extends StatefulWidget {
  const ResourceListPage(
      {super.key,
      required this.admin,
      required this.slug,
      this.openRecord,
      this.initialFilters = const {},
      this.initialSearch = ''});

  final AdminController admin;
  final String slug;

  /// Filters (name -> value) and search to start with, e.g. from a dashboard
  /// link.
  final Map<String, String> initialFilters;
  final String initialSearch;

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
  AdvancedSearch _advanced = AdvancedSearch();

  /// The column rows are grouped by ("" for none).
  late String _group;
  Map<String, Map<String, double>> _totals = const {};
  Map<String, Map<String, int>> _facets = const {};

  List<String> get _groups {
    final g = _resource['groups'];
    return g is List ? g.map((e) => '$e').toList() : const [];
  }

  Map<String, List<String>> get _summarized => {
        for (final c in _allColumns)
          if (c['summaries'] is List && (c['summaries'] as List).isNotEmpty)
            '${c['name']}': (c['summaries'] as List).map((e) => '$e').toList(),
      };

  String _columnLabel(String name) {
    final c = _allColumns.where((c) => c['name'] == name);
    return c.isEmpty ? humanize(name) : '${c.first['label'] ?? humanize(name)}';
  }

  /// Column widths the user set by dragging (name -> width).
  final Map<String, double> _widths = {};
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

  /// Saved views (Panel.SavedViews) and the one applied, if any.
  List<Map<String, dynamic>> _views = const [];
  bool _canShareViews = false;
  String? _viewId;
  bool get _viewsEnabled => admin.schema['saved_views'] == true;

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

  String get _widthsPref => 'widths:${widget.slug}';

  Future<void> _loadWidths() async {
    try {
      final raw = await api.readPreference(_widthsPref);
      if (raw == null || !mounted) return;
      final m = json.decode(raw);
      if (m is Map) {
        setState(() {
          for (final e in m.entries) {
            final w = (e.value as num?)?.toDouble();
            if (w != null) _widths['${e.key}'] = w;
          }
        });
      }
    } catch (_) {}
  }

  void _saveWidths() => api.writePreference(_widthsPref, json.encode(_widths));

  bool get _advancedEnabled =>
      _resource['advanced_search_enabled'] == true &&
      _advancedFields.isNotEmpty;

  Map<String, String> get _advancedFields {
    final names = _resource['advanced_search_columns'];
    final labels = {
      for (final c in _allColumns)
        '${c['name']}': '${c['label'] ?? humanize('${c['name']}')}'
    };
    return {
      if (names is List)
        for (final n in names) '$n': labels['$n'] ?? humanize('$n'),
    };
  }

  Future<void> _editAdvanced() async {
    final next = await showAdvancedSearchDialog(context,
        fields: _advancedFields, current: _advanced);
    if (next == null || !mounted) return;
    setState(() {
      _advanced = next;
      _page = 1;
    });
    _load();
  }

  // ------------------------------------------------------------- saved views

  Future<void> _loadViews() async {
    if (!_viewsEnabled) return;
    try {
      final data = await api.views(widget.slug);
      if (!mounted) return;
      final list = data['views'];
      setState(() {
        _views = list is List
            ? [
                for (final v in list)
                  if (v is Map) Map<String, dynamic>.from(v)
              ]
            : const [];
        _canShareViews = data['can_share'] == true;
      });
    } catch (_) {}
  }

  Map<String, dynamic> get _viewState => {
        if (_search.text.trim().isNotEmpty) 'search': _search.text.trim(),
        if (_sort.isNotEmpty) 'sort': _sort,
        if (_sort.isNotEmpty) 'direction': _descending ? 'desc' : 'asc',
        if (_filters.isNotEmpty) 'filters': {..._filters},
        if (!_advanced.isEmpty) 'advanced_search': _advanced.encode(),
        if (_shown != null) 'columns': _shown!.toList()..sort(),
        if (_group.isNotEmpty) 'group': _group,
        'per_page': _perPage,
      };

  void _applyView(Map<String, dynamic>? view) {
    final st = view?['state'] is Map
        ? Map<String, dynamic>.from(view!['state'] as Map)
        : <String, dynamic>{};
    setState(() {
      _viewId = view == null ? null : '${view['id']}';
      _search.text = '${st['search'] ?? ''}';
      _sort = '${st['sort'] ?? ''}';
      _descending = st['direction'] == 'desc';
      _filters
        ..clear()
        ..addAll(st['filters'] is Map
            ? (st['filters'] as Map).map((k, v) => MapEntry('$k', '$v'))
            : const {});
      _advanced = AdvancedSearch.decode('${st['advanced_search'] ?? ''}');
      final cols = st['columns'];
      if (cols is List && cols.isNotEmpty) {
        _shown = {for (final c in cols) '$c'};
      }
      final g = '${st['group'] ?? ''}';
      _group = _groups.contains(g) ? g : '';
      final pp = (st['per_page'] as num?)?.toInt();
      if (pp != null && pp > 0) _perPage = pp;
      _page = 1;
    });
    _load();
  }

  Future<void> _saveView() async {
    final result = await showDialog<(String, bool)>(
      context: context,
      builder: (_) => _SaveViewDialog(canShare: _canShareViews),
    );
    if (result == null) return;
    try {
      final v = await api.saveView(widget.slug, result.$1, _viewState,
          shared: result.$2);
      await _loadViews();
      if (mounted) setState(() => _viewId = '${v['id']}');
      _toast(tr('View saved.'));
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  Future<void> _deleteView(Map<String, dynamic> view) async {
    try {
      await api.deleteView(widget.slug, '${view['id']}');
      if (_viewId == '${view['id']}') setState(() => _viewId = null);
      await _loadViews();
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  Map<String, dynamic>? get _currentView {
    for (final v in _views) {
      if ('${v['id']}' == _viewId) return v;
    }
    return null;
  }

  void _onViewSelected(String v) {
    final current = _currentView;
    if (v == ':all') {
      _applyView(null);
    } else if (v == ':save') {
      _saveView();
    } else if (v == ':delete' && current != null) {
      _deleteView(current);
    } else {
      _applyView(_views.firstWhere((x) => '${x['id']}' == v));
    }
  }

  List<PopupMenuEntry<String>> _viewItems() {
    final current = _currentView;
    return [
      CheckedPopupMenuItem(
          key: const Key('view-all'),
          value: ':all',
          checked: _viewId == null,
          child: Text(tr('All records'))),
      for (final v in _views)
        CheckedPopupMenuItem(
          key: Key('view-${v['id']}'),
          value: '${v['id']}',
          checked: '${v['id']}' == _viewId,
          child: Text(
              '${v['name']}${v['shared'] == true ? ' ${tr('(shared)')}' : ''}'),
        ),
      const PopupMenuDivider(),
      PopupMenuItem(
          key: const Key('view-save'),
          value: ':save',
          child: Text(tr('Save current view…'))),
      if (current != null && current['can_delete'] == true)
        PopupMenuItem(
            key: const Key('view-delete'),
            value: ':delete',
            child: Text(tr('Delete "{name}"', {'name': current['name']}))),
    ];
  }

  Widget _viewsMenu() {
    final current = _currentView;
    return PopupMenuButton<String>(
      key: const Key('list-views'),
      tooltip: tr('Saved views'),
      onSelected: _onViewSelected,
      itemBuilder: (_) => _viewItems(),
      child: Padding(
        padding: const EdgeInsets.all(8),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.bookmarks_outlined, size: 20),
          if (current != null) ...[
            const SizedBox(width: 4),
            Text('${current['name']}', key: const Key('view-current')),
          ],
        ]),
      ),
    );
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
        _toast(tr('Link copied'));
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
                child: Text(tr('Cancel'))),
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
    _group = '${_resource['default_group'] ?? ''}';
    if (!_groups.contains(_group)) _group = '';
    _filters.addAll(widget.initialFilters);
    if (widget.initialSearch.isNotEmpty) _search.text = widget.initialSearch;
    _loadColumnChoice();
    _loadWidths();
    _loadViews();
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
          sort: _group.isNotEmpty ? _group : _sort,
          descending: _group.isNotEmpty ? false : _descending,
          filters: _filters,
          advancedSearch: _advanced.encode());
      if (!mounted || request != _requestId) return;
      setState(() {
        _rows = page.rows;
        _rowMeta = page.rowMeta;
        _total = page.total;
        _totals = page.summaries;
        _facets = page.facets;
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
        _error = tr('Could not load the records. Please try again.');
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
              child: Text(tr('Cancel'))),
          FilledButton(
            key: const Key('confirm-delete'),
            style: FilledButton.styleFrom(
                backgroundColor: Theme.of(ctx).colorScheme.error),
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(tr('Delete')),
          ),
        ],
      ),
    );
    return ok == true;
  }

  Future<void> _delete(Map<String, dynamic> row) async {
    if (!await _confirm(
        tr('Delete this record?'), tr('This cannot be undone.'))) {
      return;
    }
    try {
      await api.delete(widget.slug, recordId(row));
      _toast(tr('Deleted'));
      _load();
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  Future<void> _restore(Map<String, dynamic> row) async {
    try {
      await api.restoreRecord(widget.slug, recordId(row));
      _toast(tr('Restored'));
      _load();
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  Future<void> _forceDelete(Map<String, dynamic> row) async {
    if (!await _confirm(tr('Delete this record for good?'),
        tr('It cannot be restored afterwards.'))) {
      return;
    }
    try {
      await api.forceDelete(widget.slug, recordId(row));
      _toast(tr('Deleted for good'));
      _load();
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  Future<void> _deleteSelected() async {
    final ids = _selected.toList();
    if (ids.isEmpty) return;
    if (!await _confirm(tr('Delete {n} records?', {'n': ids.length}),
        tr('This cannot be undone.'))) {
      return;
    }
    try {
      await api.deleteMany(widget.slug, ids);
      _selected.clear();
      _toast(tr('Deleted'));
      _load();
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  Future<void> _toggleCell(
      Map<String, dynamic> row, String column, bool value) async {
    final previous = row[column];
    setState(() => row[column] = value);
    final id = recordId(row);
    final meta = _rowMeta[id];
    try {
      final version = await api.updateCell(widget.slug, id, column, value,
          version: meta?['version']?.toString());
      if (version != null && meta != null) meta['version'] = version;
    } on ApiException catch (e) {
      if (mounted) setState(() => row[column] = previous);
      _toast(
          e.isConflict
              ? tr(
                  'This record was changed by someone else. Reload the list to see their changes.')
              : e.message,
          error: true);
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

  /// Exports on the server as a background job, then saves the file.
  Future<void> _exportJob(String format) async {
    final columns = [
      for (final c in _columns)
        if (c['type'] != 'image') '${c['name']}'
    ];
    var job = await api.startExportJob(widget.slug, format,
        search: _search.text.trim(),
        sort: _sort,
        descending: _descending,
        filters: _filters,
        advancedSearch: _advanced.encode(),
        ids: _selected.toList(),
        columns: columns);
    final id = '${job['id']}';
    _toast(tr('Preparing the export…'));
    while (job['status'] == 'queued' || job['status'] == 'running') {
      await Future<void>.delayed(jobPollInterval);
      if (!mounted) return;
      job = await api.job(id);
    }
    if (job['status'] != 'done') {
      _toast(
          tr('The export failed: {message}', {'message': job['message'] ?? ''}),
          error: true);
      return;
    }
    final bytes = await api.jobFile(id);
    final name = '${job['file_name'] ?? '${widget.slug}.$format'}';
    final ok = await saveFile(
        name,
        bytes,
        format == 'xlsx'
            ? 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'
            : 'text/csv');
    _toast(ok
        ? tr('Exported {n} rows.', {'n': job['total'] ?? ''})
        : tr('Export is not available on this device yet.'));
  }

  Future<void> _export(String format) async {
    if (_exporting) return;
    setState(() => _exporting = true);
    if (admin.schema['background_jobs'] == true) {
      try {
        await _exportJob(format);
      } on ApiException catch (e) {
        _toast(e.message, error: true);
      } finally {
        if (mounted) setState(() => _exporting = false);
      }
      return;
    }
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
        _toast(rows.length >= maxRows && _selected.isEmpty
            ? tr('Exported {n} rows (limit reached).', {'n': rows.length})
            : tr('Exported {n} rows.', {'n': rows.length}));
      } else {
        _toast(tr('Export is not available on this device yet.'), error: true);
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
    final sizes = AdminSizes.of(context);
    final wide = !sizes.isPhone;
    final page = Padding(
      padding: EdgeInsets.all(sizes.pagePadding),
      child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        _toolbar(context),
        SizedBox(height: sizes.gap + 2),
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
    if (!sizes.isPhone || !_canCreate || _showBoard) return page;
    // phones: "New" is a floating button above the pagination bar
    return Stack(children: [
      Positioned.fill(child: page),
      PositionedDirectional(
        end: sizes.pagePadding + 12,
        bottom: sizes.pagePadding + 60,
        child: FloatingActionButton(
          key: const Key('list-create'),
          tooltip: _createLabel,
          onPressed: () => _openForm(),
          child: const Icon(Icons.add),
        ),
      ),
    ]);
  }

  /// The toolbar's secondary actions, most important first: the ones that
  /// do not fit go under the "more" menu (on a phone, all of them).
  List<_ToolAction> _toolActions(BuildContext context) {
    final primary = Theme.of(context).colorScheme.primary;
    return [
      if (_advancedEnabled)
        _ToolAction(
          id: 'advanced-search',
          label: tr('Advanced search'),
          icon: Badge(
            isLabelVisible: !_advanced.isEmpty,
            label: Text('${_advanced.length}'),
            child: Icon(Icons.manage_search,
                color: _advanced.isEmpty ? null : primary),
          ),
          onPressed: _editAdvanced,
        ),
      if (_viewsEnabled)
        _ToolAction(
          id: 'views',
          label: _currentView == null
              ? tr('Saved views')
              : '${tr('Saved views')}: ${_currentView!['name']}',
          icon: const Icon(Icons.bookmarks_outlined),
          menu: _viewItems,
          onSelected: _onViewSelected,
          inline: _viewsMenu(),
        ),
      if (_groups.isNotEmpty)
        _ToolAction(
          id: 'group',
          label: tr('Group by'),
          icon: Icon(Icons.segment, color: _group.isEmpty ? null : primary),
          menu: () => [
            CheckedPopupMenuItem(
                key: const Key('group-none'),
                value: '',
                checked: _group.isEmpty,
                child: Text(tr('No grouping'))),
            for (final g in _groups)
              CheckedPopupMenuItem(
                  key: Key('group-$g'),
                  value: g,
                  checked: _group == g,
                  child: Text(_columnLabel(g))),
          ],
          onSelected: (g) {
            setState(() {
              _group = g;
              _page = 1;
            });
            _load();
          },
        ),
      if (_allColumns.any((c) => c['toggleable'] == true))
        _ToolAction(
          id: 'columns',
          label: tr('Columns'),
          icon: const Icon(Icons.view_column_outlined),
          menu: () => [
            for (final c in _allColumns.where((c) => c['toggleable'] == true))
              CheckedPopupMenuItem<String>(
                key: Key('column-toggle-${c['name']}'),
                value: '${c['name']}',
                checked: _isShown(c),
                child: Text('${c['label'] ?? humanize('${c['name']}')}'),
              ),
          ],
          onSelected: _toggleColumn,
        ),
      if (_hasBoard)
        _ToolAction(
          id: 'board',
          label: _showBoard ? 'Table' : 'Board',
          icon: Icon(_showBoard
              ? Icons.table_rows_outlined
              : Icons.view_kanban_outlined),
          onPressed: () => setState(() => _showBoard = !_showBoard),
        ),
      if (_exportEnabled)
        _ToolAction(
          id: 'export',
          label: tr('Export'),
          icon: const Icon(Icons.download_outlined),
          enabled: !_exporting,
          menu: () => [
            PopupMenuItem(value: 'csv', child: Text(tr('Export CSV'))),
            PopupMenuItem(value: 'xlsx', child: Text(tr('Export Excel'))),
          ],
          onSelected: _export,
          inline: PopupMenuButton<String>(
            key: const Key('list-export'),
            tooltip: tr('Export'),
            enabled: !_exporting,
            onSelected: _export,
            itemBuilder: (_) => [
              PopupMenuItem(value: 'csv', child: Text(tr('Export CSV'))),
              PopupMenuItem(value: 'xlsx', child: Text(tr('Export Excel'))),
            ],
            child: Padding(
              padding: const EdgeInsets.all(8),
              child: _exporting
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(strokeWidth: 2))
                  : const Icon(Icons.download_outlined, size: 20),
            ),
          ),
        ),
      if (admin.can(widget.slug, 'import'))
        _ToolAction(
          id: 'import',
          label: tr('Import'),
          icon: const Icon(Icons.upload_file_outlined),
          onPressed: () async {
            final n = await showImportDialog(context,
                api: api,
                slug: widget.slug,
                title: '${_resource['plural_model_name'] ?? widget.slug}');
            if (n != null && n > 0) _load();
          },
        ),
      if (_reorderable)
        _ToolAction(
          id: 'reorder',
          label: _canReorderNow
              ? 'Reorder'
              : tr('Clear search, filters, sorting and grouping to reorder'),
          icon: const Icon(Icons.swap_vert),
          enabled: _canReorderNow && _rows.length > 1,
          onPressed: _reorder,
        ),
      _ToolAction(
        id: 'refresh',
        label: tr('Refresh'),
        icon: const Icon(Icons.refresh),
        onPressed: _load,
      ),
    ];
  }

  Widget _inlineAction(_ToolAction a) {
    if (a.inline != null) return a.inline!;
    if (a.menu != null) {
      return PopupMenuButton<String>(
        key: Key('list-${a.id}'),
        tooltip: a.label,
        enabled: a.enabled,
        icon: a.icon,
        onSelected: a.onSelected,
        itemBuilder: (_) => a.menu!(),
      );
    }
    return IconButton(
      key: Key('list-${a.id}'),
      tooltip: a.label,
      icon: a.icon,
      onPressed: a.enabled ? a.onPressed : null,
    );
  }

  final _moreKey = GlobalKey();

  /// The ⋮ menu with the actions that did not fit. An action with its own
  /// menu (group by, columns, export, views) opens it from the same spot.
  Widget _moreMenu(List<_ToolAction> actions) {
    return KeyedSubtree(
      key: _moreKey,
      child: PopupMenuButton<String>(
        key: const Key('list-more'),
        tooltip: tr('More'),
        icon: const Icon(Icons.more_vert),
        position: PopupMenuPosition.under,
        itemBuilder: (context) => [
          for (final a in actions)
            PopupMenuItem<String>(
              key: Key('list-more-${a.id}'),
              value: a.id,
              enabled: a.enabled,
              child: Row(children: [
                IconTheme.merge(
                    data: const IconThemeData(size: 20), child: a.icon),
                const SizedBox(width: 12),
                Expanded(
                    child: Text(a.label,
                        maxLines: 1, overflow: TextOverflow.ellipsis)),
                if (a.menu != null)
                  const Padding(
                    padding: EdgeInsetsDirectional.only(start: 8),
                    child: Icon(Icons.arrow_right, size: 20),
                  ),
              ]),
            ),
        ],
        onSelected: (id) async {
          final a = actions.firstWhere((a) => a.id == id);
          if (a.menu == null) {
            a.onPressed?.call();
            return;
          }
          final anchor = _moreKey.currentContext;
          if (anchor == null) return;
          final button = anchor.findRenderObject()! as RenderBox;
          final overlay = Navigator.of(anchor)
              .overlay!
              .context
              .findRenderObject()! as RenderBox;
          final rect = Rect.fromPoints(
            button.localToGlobal(Offset(0, button.size.height),
                ancestor: overlay),
            button.localToGlobal(button.size.bottomRight(Offset.zero),
                ancestor: overlay),
          );
          final v = await showMenu<String>(
            context: anchor,
            position: RelativeRect.fromRect(rect, Offset.zero & overlay.size),
            items: a.menu!(),
          );
          if (v != null) a.onSelected?.call(v);
        },
      ),
    );
  }

  String get _createLabel =>
      tr('New {name}', {'name': _resource['model_name'] ?? ''}).trim();

  Widget _toolbar(BuildContext context) {
    final theme = Theme.of(context);
    final sizes = AdminSizes.of(context);
    final searchField = SizedBox(
      height: sizes.searchHeight,
      child: TextField(
        key: const Key('list-search'),
        controller: _search,
        onChanged: _onSearchChanged,
        textInputAction: TextInputAction.search,
        onSubmitted: (_) {
          _debounce?.cancel();
          _page = 1;
          _load();
        },
        style: theme.textTheme.bodyMedium,
        textAlignVertical: TextAlignVertical.center,
        // fill the toolbar's search height exactly
        expands: true,
        maxLines: null,
        decoration: InputDecoration(
          hintText: tr('Search'),
          contentPadding: const EdgeInsets.symmetric(horizontal: 12),
          prefixIcon: const Icon(Icons.search, size: 20),
          suffixIcon: _search.text.isEmpty
              ? null
              : IconButton(
                  tooltip: tr('Clear'),
                  icon: const Icon(Icons.close, size: 18),
                  onPressed: () {
                    _search.clear();
                    _page = 1;
                    _load();
                  },
                ),
        ),
      ),
    );

    final filters = [for (final f in _filterDefs) _filterChip(context, f)];
    Widget filterRow() => SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: Row(children: [
            for (final f in filters)
              Padding(
                  padding: EdgeInsetsDirectional.only(end: sizes.gap - 2),
                  child: f)
          ]),
        );

    final all = _toolActions(context);

    return LayoutBuilder(builder: (context, c) {
      // which actions show as icons: all of them when they fit, otherwise
      // the most important ones and a ⋮ menu; phones keep only ⋮
      final slot = sizes.iconButtonSize + 4;
      final createText =
          sizes.isDesktop || c.maxWidth >= 720 ? _createLabel : tr('New');
      var createWidth = 0.0;
      if (_canCreate && !sizes.isPhone) {
        final painter = TextPainter(
            text: TextSpan(
                text: createText,
                style: theme.textTheme.labelLarge
                    ?.copyWith(fontWeight: FontWeight.w600)),
            textDirection: Directionality.of(context))
          ..layout();
        createWidth = painter.width + 18 + 8 + 32 + sizes.gap;
      }
      final filtersInline = sizes.isDesktop && filters.isNotEmpty;
      final searchWidth = sizes.isPhone
          ? c.maxWidth * 0.6
          : (sizes.isDesktop ? 300.0 : (c.maxWidth * 0.42).clamp(200.0, 300.0));
      final room = c.maxWidth -
          searchWidth -
          sizes.gap -
          createWidth -
          (filtersInline ? 180 : 0);
      final fit = sizes.isPhone ? 0 : (room / slot).floor();
      final List<_ToolAction> inline;
      final List<_ToolAction> overflow;
      if (all.length <= fit || (sizes.isPhone && all.length == 1)) {
        inline = all;
        overflow = const [];
      } else {
        final n = (fit - 1).clamp(0, all.length);
        // the n most important stay inline, the rest go under ⋮
        final keep = all.take(n).toSet();
        inline = [
          for (final a in all)
            if (keep.contains(a)) a
        ];
        overflow = [
          for (final a in all)
            if (!keep.contains(a)) a
        ];
      }
      // the inline icons in a stable visual order
      inline.sort((x, y) => _actionOrder(x.id).compareTo(_actionOrder(y.id)));

      final actions = <Widget>[
        for (final a in inline) _inlineAction(a),
        if (overflow.isNotEmpty) _moreMenu(overflow),
        if (_canCreate && !sizes.isPhone) ...[
          SizedBox(width: sizes.gap),
          SizedBox(
            height: sizes.searchHeight,
            child: FilledButton.icon(
              key: const Key('list-create'),
              onPressed: () => _openForm(),
              icon: const Icon(Icons.add, size: 18),
              label: Text(createText),
            ),
          ),
        ],
      ];

      if (filtersInline) {
        return Row(children: [
          SizedBox(width: searchWidth, child: searchField),
          SizedBox(width: sizes.gap),
          Expanded(child: filterRow()),
          ...actions,
        ]);
      }
      return Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        Row(children: [
          if (sizes.isPhone)
            Expanded(child: searchField)
          else ...[
            SizedBox(width: searchWidth, child: searchField),
            const Spacer(),
          ],
          if (sizes.isPhone) const SizedBox(width: 4),
          ...actions,
        ]),
        if (filters.isNotEmpty) ...[
          SizedBox(height: sizes.gap),
          filterRow(),
        ],
      ]);
    });
  }

  static const _visualOrder = [
    'refresh',
    'group',
    'advanced-search',
    'columns',
    'board',
    'reorder',
    'views',
    'import',
    'export',
  ];

  static int _actionOrder(String id) => _visualOrder.indexOf(id);

  Widget _filterChip(BuildContext context, Map<String, dynamic> filter) {
    if (filter['type'] == 'refined') return _refinedChip(context, filter);
    if (filter['type'] == 'date_range' || filter['type'] == 'number_range') {
      return _rangeChip(context, filter);
    }
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
        PopupMenuItem(value: '', child: Text(tr('All'))),
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

  /// The kanban board instead of the table (Resource.Kanban).
  bool _showBoard = false;
  bool get _hasBoard => _meta['board'] is Map;

  bool get _reorderable {
    final t = _meta['table'];
    return t is Map && t['reorderable'] == true;
  }

  /// Rows can be reordered on the plain list only, where they are shown in
  /// their real order.
  bool get _canReorderNow =>
      _search.text.trim().isEmpty &&
      _filters.isEmpty &&
      _advanced.isEmpty &&
      _sort.isEmpty &&
      _group.isEmpty;

  Future<void> _reorder() async {
    final columns = _columns;
    final ids = await showReorderDialog(context, rows: [
      for (final r in _rows)
        (
          id: recordId(r),
          title: columns.isEmpty
              ? recordId(r)
              : displayValue(valueAt(r, '${columns.first['name']}')),
        )
    ]);
    if (ids == null) return;
    try {
      await api.reorder(widget.slug, ids, offset: (_page - 1) * _perPage);
      _toast(tr('Order saved.'));
      _load();
    } on ApiException catch (e) {
      _toast(e.message, error: true);
    }
  }

  List<String> get _bulkEditFields {
    final f = _meta['bulk_edit'];
    return f is List ? [for (final x in f) '$x'] : const [];
  }

  Future<void> _bulkEdit() async {
    final res = await showBulkEditDialog(context,
        api: api,
        slug: widget.slug,
        ids: _selected.toList(),
        fields: _bulkEditFields);
    if (res == null) return;
    final failed = (res['failed'] as num?)?.toInt() ?? 0;
    _toast(
        failed == 0
            ? tr('{n} records changed.', {'n': res['updated']})
            : tr('{n} changed, {failed} not changed.',
                {'n': res['updated'], 'failed': failed}),
        error: failed > 0);
    setState(_selected.clear);
    _load();
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
        Text(tr('{n} selected', {'n': _selected.length}),
            style: const TextStyle(fontWeight: FontWeight.w600)),
        const Spacer(),
        TextButton(
            onPressed: () => setState(_selected.clear),
            child: Text(tr('Clear'))),
        if (_bulkEditFields.isNotEmpty)
          TextButton.icon(
            key: const Key('bulk-edit'),
            onPressed: _bulkEdit,
            icon: const Icon(Icons.edit_note, size: 18),
            label: Text(tr('Change')),
          ),
        if (_canDeleteAny && _bulkActions.contains('delete_bulk'))
          TextButton.icon(
            key: const Key('bulk-delete'),
            onPressed: _deleteSelected,
            icon: Icon(Icons.delete_outline,
                size: 18, color: theme.colorScheme.error),
            label: Text(tr('Delete'),
                style: TextStyle(color: theme.colorScheme.error)),
          ),
      ]),
    );
  }

  Widget _content(BuildContext context, bool wide) {
    final theme = Theme.of(context);
    if (_showBoard) {
      return KanbanBoardView(
        api: api,
        slug: widget.slug,
        search: _search.text.trim(),
        onOpen: _hasForm ? (id) => _openForm(id: id) : null,
      );
    }
    if (_error != null) {
      return Center(
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Text(_error!, style: TextStyle(color: theme.colorScheme.error)),
          const SizedBox(height: 12),
          OutlinedButton(onPressed: _load, child: Text(tr('Try again'))),
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
          Text(
              filtered
                  ? tr('No records match your search.')
                  : tr('No records yet.'),
              style: TextStyle(color: theme.colorScheme.onSurfaceVariant)),
        ]),
      );
    }
    return wide ? _table(context) : _cards(context);
  }

  Widget _refinedChip(BuildContext context, Map<String, dynamic> filter) {
    final theme = Theme.of(context);
    final column = '${filter['column'] ?? filter['name']}';
    final label = '${filter['label'] ?? humanize(column)}';
    final options = refinedOptions(filter['options'], facets: _facets[column]);
    final current =
        (_filters[column] ?? '').split(',').where((v) => v.isNotEmpty).toSet();
    final labels = [
      for (final o in options)
        if (current.contains('${o['value']}')) '${o['label'] ?? o['value']}'
    ];
    final active = current.isNotEmpty;
    return InkWell(
      key: Key('filter-$column'),
      borderRadius: BorderRadius.circular(10),
      onTap: () async {
        final chosen = await showRefinedFilterDialog(context,
            label: label,
            options: options,
            selected: current,
            multiple: filter['multiple'] != false);
        if (chosen == null || !mounted) return;
        setState(() {
          if (chosen.isEmpty) {
            _filters.remove(column);
          } else {
            _filters[column] = (chosen.toList()..sort()).join(',');
          }
          _page = 1;
        });
        _load();
      },
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(10),
          border: Border.all(
              color: active ? theme.colorScheme.primary : theme.dividerColor),
          color:
              active ? theme.colorScheme.primary.withValues(alpha: 0.08) : null,
        ),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          Icon(Icons.checklist,
              size: 16,
              color: active
                  ? theme.colorScheme.primary
                  : theme.colorScheme.onSurfaceVariant),
          const SizedBox(width: 6),
          Text(
              !active
                  ? label
                  : labels.length <= 2
                      ? '$label: ${labels.join(', ')}'
                      : '$label: ${labels.length} selected',
              style: const TextStyle(fontSize: 13)),
        ]),
      ),
    );
  }

  Widget _rangeChip(BuildContext context, Map<String, dynamic> filter) {
    final theme = Theme.of(context);
    final column = '${filter['column'] ?? filter['name']}';
    final label = '${filter['label'] ?? humanize(column)}';
    final dates = filter['type'] == 'date_range';
    final value = _filters[column] ?? '';
    final active = value.isNotEmpty;
    return InkWell(
      key: Key('filter-$column'),
      borderRadius: BorderRadius.circular(10),
      onTap: () async {
        final next = await showRangeFilterDialog(context,
            label: label, dates: dates, value: value);
        if (next == null || !mounted) return;
        setState(() {
          if (next.isEmpty) {
            _filters.remove(column);
          } else {
            _filters[column] = next;
          }
          _page = 1;
        });
        _load();
      },
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 9),
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(10),
          border: Border.all(
              color: active ? theme.colorScheme.primary : theme.dividerColor),
          color:
              active ? theme.colorScheme.primary.withValues(alpha: 0.08) : null,
        ),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          Icon(dates ? Icons.date_range : Icons.straighten,
              size: 16,
              color: active
                  ? theme.colorScheme.primary
                  : theme.colorScheme.onSurfaceVariant),
          const SizedBox(width: 6),
          Text(rangeSummary(label, value),
              style: const TextStyle(fontSize: 13)),
        ]),
      ),
    );
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
                    label: ResizableHeader(
                      name: '${c['name']}',
                      width: _widths['${c['name']}'],
                      label: Text(
                          '${c['label'] ?? humanize('${c['name']}')}'
                              .toUpperCase(),
                          overflow: TextOverflow.ellipsis),
                      onResize: (w) =>
                          setState(() => _widths['${c['name']}'] = w),
                      onResizeEnd: _saveWidths,
                      onReset: () {
                        setState(() => _widths.remove('${c['name']}'));
                        _saveWidths();
                      },
                    ),
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
                for (final entry in _tableRows(columns))
                  if (entry.row == null)
                    _extraRow(context, entry, columns, hasRowActions)
                  else ...[
                    for (final row in [entry.row!])
                      DataRow(
                        key: ValueKey(recordId(row)),
                        selected: _selected.contains(recordId(row)),
                        onSelectChanged: _selectable
                            ? (v) => setState(() => v == true
                                ? _selected.add(recordId(row))
                                : _selected.remove(recordId(row)))
                            : null,
                        cells: [
                          for (final c in columns)
                            DataCell(sizedCell(_widths['${c['name']}'],
                                _cell(context, c, row))),
                          if (hasRowActions)
                            DataCell(_rowActions(context, row)),
                        ],
                      ),
                  ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  /// The table's rows in order: group headers, records, group subtotals
  /// and the summary rows at the end.
  List<_TableRow> _tableRows(List<Map<String, dynamic>> columns) {
    final out = <_TableRow>[];
    final summarized = _summarized;
    Map<String, Map<String, double>> sums(List<Map<String, dynamic>> rows) => {
          for (final e in summarized.entries)
            e.key:
                computeSummaries(rows.map((r) => valueAt(r, e.key)), e.value),
        };
    if (_group.isEmpty) {
      out.addAll(_rows.map((r) => _TableRow(row: r)));
    } else {
      String key(Map<String, dynamic> r) => displayValue(valueAt(r, _group));
      var i = 0;
      while (i < _rows.length) {
        final k = key(_rows[i]);
        final members = <Map<String, dynamic>>[];
        while (i < _rows.length && key(_rows[i]) == k) {
          members.add(_rows[i++]);
        }
        out.add(_TableRow(
            label:
                '${_columnLabel(_group)}: ${k.isEmpty ? '(none)' : k} (${members.length})',
            kind: 'group'));
        out.addAll(members.map((r) => _TableRow(row: r)));
        if (summarized.isNotEmpty) {
          out.add(_TableRow(
              label: 'Subtotal', kind: 'subtotal', sums: sums(members)));
        }
      }
    }
    if (summarized.isNotEmpty && _rows.isNotEmpty) {
      out.add(_TableRow(
          label: tr('This page'), kind: 'summary', sums: sums(_rows)));
      if (_totals.isNotEmpty) {
        out.add(_TableRow(
            label: tr('All results'), kind: 'summary', sums: _totals));
      }
    }
    for (var i = 0; i < out.length; i++) {
      out[i].pos = i;
    }
    return out;
  }

  DataRow _extraRow(BuildContext context, _TableRow entry,
      List<Map<String, dynamic>> columns, bool hasRowActions) {
    final theme = Theme.of(context);
    final group = entry.kind == 'group';
    final summarized = _summarized;
    return DataRow(
      key: ValueKey('${entry.kind}-${entry.pos}'),
      color: WidgetStatePropertyAll(group
          ? theme.colorScheme.surfaceContainerHighest
          : theme.colorScheme.surfaceContainerLow),
      cells: [
        for (var i = 0; i < columns.length; i++)
          DataCell(sizedCell(
            _widths['${columns[i]['name']}'],
            group
                ? (i == 0
                    ? Text(entry.label,
                        key: Key('group-row-${entry.label}'),
                        style: const TextStyle(fontWeight: FontWeight.w700))
                    : const SizedBox.shrink())
                : summarized.containsKey('${columns[i]['name']}')
                    ? summaryCell(context, entry.sums?['${columns[i]['name']}'],
                        summarized['${columns[i]['name']}']!)
                    : (i == 0
                        ? Text(entry.label,
                            style: TextStyle(
                                fontWeight: FontWeight.w600,
                                color: theme.colorScheme.onSurfaceVariant))
                        : const SizedBox.shrink()),
          )),
        if (hasRowActions) const DataCell(SizedBox.shrink()),
      ],
    );
  }

  Widget _cards(BuildContext context) {
    final theme = Theme.of(context);
    // the first image column becomes the card's picture; the others are
    // listed like any column
    final imageColumn = _columns
        .cast<Map<String, dynamic>?>()
        .firstWhere((c) => c!['type'] == 'image', orElse: () => null);
    final columns = [
      for (final c in _columns)
        if (imageColumn == null || c['name'] != imageColumn['name']) c
    ];
    final entries = _tableRows(_columns);
    return ListView.separated(
      // room for the floating "New" button over the last card
      padding: EdgeInsets.only(bottom: _canCreate ? 76 : 0),
      itemCount: entries.length,
      separatorBuilder: (_, i) => entries[i].row == null &&
              (i + 1 >= entries.length || entries[i + 1].row == null)
          ? const SizedBox.shrink()
          : const Divider(height: 1),
      itemBuilder: (context, i) {
        final entry = entries[i];
        if (entry.row == null) return _extraCard(context, entry);
        final row = entry.row!;
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
            child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
              if (imageColumn != null)
                Padding(
                  padding: const EdgeInsets.only(right: 12, top: 2),
                  child: cardPicture(context, imageColumn, row),
                ),
              Expanded(
                child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(children: [
                        Expanded(
                            child: Text(title,
                                style: const TextStyle(
                                    fontWeight: FontWeight.w600,
                                    fontSize: 15))),
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
            ]),
          ),
        );
      },
    );
  }

  /// A group header or summary block of the card layout.
  Widget _extraCard(BuildContext context, _TableRow entry) {
    final theme = Theme.of(context);
    if (entry.kind == 'group') {
      return Container(
        key: Key('group-card-${entry.label}'),
        color: theme.colorScheme.surfaceContainerHighest,
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
        child: Text(entry.label,
            key: Key('group-row-${entry.label}'),
            style: const TextStyle(fontWeight: FontWeight.w700)),
      );
    }
    final summarized = _summarized;
    return Container(
      key: Key('${entry.kind}-card-${entry.pos}'),
      color: theme.colorScheme.surfaceContainerLow,
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Text(entry.label,
            style: TextStyle(
                fontWeight: FontWeight.w600,
                color: theme.colorScheme.onSurfaceVariant)),
        const SizedBox(height: 4),
        Wrap(spacing: 20, runSpacing: 6, children: [
          for (final e in summarized.entries)
            if (entry.sums?[e.key] != null)
              Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(_columnLabel(e.key),
                        style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant)),
                    summaryCell(context, entry.sums![e.key], e.value),
                  ]),
        ]),
      ]),
    );
  }

  Widget _rowActions(BuildContext context, Map<String, dynamic> row) {
    final theme = Theme.of(context);
    final id = recordId(row);
    return Row(mainAxisSize: MainAxisSize.min, children: [
      if (_actions.contains('view') && _hasForm)
        IconButton(
          key: Key('row-view-$id'),
          tooltip: tr('View'),
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
          tooltip: tr('Edit'),
          visualDensity: VisualDensity.compact,
          icon: const Icon(Icons.edit_outlined, size: 19),
          onPressed: () => _openForm(id: id),
        ),
      if (_rowMeta[id]?['restore'] == true)
        IconButton(
          key: Key('row-restore-$id'),
          tooltip: tr('Restore'),
          visualDensity: VisualDensity.compact,
          icon: const Icon(Icons.restore_from_trash_outlined, size: 19),
          onPressed: () => _restore(row),
        ),
      if (_rowMeta[id]?['force_delete'] == true)
        IconButton(
          key: Key('row-force-delete-$id'),
          tooltip: tr('Delete for good'),
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
          tooltip: tr('Delete'),
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
          child: Text(
              tr('{from}–{to} of {total}',
                  {'from': from, 'to': to, 'total': _total}),
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
              DropdownMenuItem(
                  value: o, child: Text(tr('{n} / page', {'n': o})))
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
          tooltip: tr('Previous page'),
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
          tooltip: tr('Next page'),
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

class _TableRow {
  _TableRow({this.row, this.label = '', this.kind = '', this.sums});
  final Map<String, dynamic>? row;
  final String label;
  final String kind;
  final Map<String, Map<String, double>>? sums;
  int pos = 0;
}

class _SaveViewDialog extends StatefulWidget {
  const _SaveViewDialog({required this.canShare});
  final bool canShare;

  @override
  State<_SaveViewDialog> createState() => _SaveViewDialogState();
}

class _SaveViewDialogState extends State<_SaveViewDialog> {
  final _name = TextEditingController();
  bool _shared = false;

  @override
  void dispose() {
    _name.dispose();
    super.dispose();
  }

  void _ok() {
    final n = _name.text.trim();
    if (n.isNotEmpty) Navigator.pop(context, (n, _shared));
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
        title: Text(tr('Save view')),
        content: Column(mainAxisSize: MainAxisSize.min, children: [
          TextField(
            key: const Key('view-name'),
            controller: _name,
            autofocus: true,
            maxLength: 80,
            decoration: InputDecoration(labelText: tr('Name')),
            onSubmitted: (_) => _ok(),
          ),
          if (widget.canShare)
            SwitchListTile(
              key: const Key('view-shared'),
              contentPadding: EdgeInsets.zero,
              title: Text(tr('Share with everyone')),
              value: _shared,
              onChanged: (v) => setState(() => _shared = v),
            ),
        ]),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context),
              child: Text(tr('Cancel'))),
          FilledButton(
              key: const Key('view-save-ok'),
              onPressed: _ok,
              child: Text(tr('Save'))),
        ],
      );
}

/// A list toolbar action: an icon button, or an icon with its own menu.
class _ToolAction {
  const _ToolAction({
    required this.id,
    required this.label,
    required this.icon,
    this.onPressed,
    this.menu,
    this.onSelected,
    this.enabled = true,
    this.inline,
  });

  /// Key suffix: the inline button is `list-<id>`, its ⋮ entry
  /// `list-more-<id>`.
  final String id;
  final String label;
  final Widget icon;
  final VoidCallback? onPressed;

  /// Items of the action's own menu (group by, columns, ...).
  final List<PopupMenuEntry<String>> Function()? menu;
  final ValueChanged<String>? onSelected;
  final bool enabled;

  /// A custom inline widget (e.g. the export button's progress).
  final Widget? inline;
}
