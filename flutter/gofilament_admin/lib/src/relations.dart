import 'package:flutter/material.dart';
import 'i18n.dart';

import 'api.dart';
import 'cells.dart';
import 'controller.dart';
import 'format.dart';
import 'icons.dart';
import 'resource_form.dart';

/// The relation managers of a record (for example a product's variants and
/// barcodes), shown under its form. Each is a small table with add, edit and
/// delete, driven by the relation's server-side description. A record with
/// more than one relation shows them as tabs, one table at a time.
class RelationPanels extends StatefulWidget {
  const RelationPanels({
    super.key,
    required this.admin,
    required this.slug,
    required this.parentId,
    this.readOnly = false,
  });

  final AdminController admin;
  final String slug;
  final String parentId;
  final bool readOnly;

  @override
  State<RelationPanels> createState() => _RelationPanelsState();
}

class _RelationPanelsState extends State<RelationPanels>
    with TickerProviderStateMixin {
  List<Map<String, dynamic>>? _relations;
  TabController? _tabs;

  @override
  void initState() {
    super.initState();
    final declared = widget.admin.resource(widget.slug)?['relations'];
    if (declared is List && declared.isNotEmpty) _load();
  }

  @override
  void dispose() {
    _tabs?.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final r = await widget.admin.api.relations(widget.slug, widget.parentId);
      if (!mounted) return;
      setState(() {
        _relations = r;
        _tabs?.dispose();
        _tabs = null;
        if (r.length > 1) {
          _tabs = TabController(length: r.length, vsync: this)
            ..addListener(() {
              if (mounted) setState(() {});
            });
        }
      });
    } catch (_) {
      // the record form stays usable without its relations
    }
  }

  Widget _table(Map<String, dynamic> r) => _RelationTable(
        admin: widget.admin,
        slug: widget.slug,
        parentId: widget.parentId,
        relation: r,
        readOnly: widget.readOnly,
      );

  @override
  Widget build(BuildContext context) {
    final relations = _relations;
    if (relations == null || relations.isEmpty) return const SizedBox.shrink();
    final tabs = _tabs;
    if (relations.length == 1 || tabs == null) {
      return Padding(
        padding: const EdgeInsets.only(bottom: 14),
        child: _table(relations.first),
      );
    }
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(top: 8, bottom: 14),
      child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        Align(
          alignment: AlignmentDirectional.centerStart,
          child: TabBar(
            key: const Key('relation-tabs'),
            controller: tabs,
            isScrollable: true,
            tabAlignment: TabAlignment.start,
            dividerColor: theme.dividerColor,
            tabs: [
              for (final r in relations)
                Tab(
                  key: Key('rel-tab-${r['relationship']}'),
                  height: 40,
                  child: Row(mainAxisSize: MainAxisSize.min, children: [
                    Icon(
                        adminIcon(r['icon'] as String?,
                            fallback: Icons.list_alt_outlined),
                        size: 16),
                    const SizedBox(width: 6),
                    Text('${r['title'] ?? humanize('${r['relationship']}')}'),
                  ]),
                ),
            ],
          ),
        ),
        const SizedBox(height: 12),
        // every table stays mounted so switching tabs keeps its rows
        IndexedStack(
          index: tabs.index,
          sizing: StackFit.passthrough,
          children: [
            for (var i = 0; i < relations.length; i++)
              Visibility(
                visible: i == tabs.index,
                maintainState: true,
                child: _table(relations[i]),
              ),
          ],
        ),
      ]),
    );
  }
}

/// Below this width a relation manager shows its records as cards instead
/// of a table (the resource list switches at 760 px of screen width; a
/// relation table goes by its own width, since it often sits in a dialog).
const relationCardBreakpoint = 600.0;

class _RelationTable extends StatefulWidget {
  const _RelationTable({
    required this.admin,
    required this.slug,
    required this.parentId,
    required this.relation,
    required this.readOnly,
  });

  final AdminController admin;
  final String slug;
  final String parentId;
  final Map<String, dynamic> relation;
  final bool readOnly;

  @override
  State<_RelationTable> createState() => _RelationTableState();
}

class _RelationTableState extends State<_RelationTable> {
  static const _pageSize = 50;

  List<Map<String, dynamic>> _rows = const [];
  int _total = 0;
  bool _loading = true;
  String? _error;

  AdminApi get api => widget.admin.api;
  String get _name => '${widget.relation['relationship']}';
  String get _title => '${widget.relation['title'] ?? humanize(_name)}';

  bool _can(String ability) {
    if (widget.readOnly) return false;
    final a = widget.relation['abilities'];
    if (a is! Map || a[ability] != true) return false;
    // adding and editing need a form; deleting does not
    return ability == 'delete' || widget.relation['has_form'] == true;
  }

  List<Map<String, dynamic>> get _columns {
    final c = widget.relation['columns'];
    return c is List
        ? c
            .whereType<Map>()
            .map((m) => Map<String, dynamic>.from(m))
            .where((m) => m['toggled_hidden_by_default'] != true)
            .toList()
        : const [];
  }

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final page = await api.relationList(widget.slug, widget.parentId, _name,
          perPage: _pageSize);
      if (!mounted) return;
      setState(() {
        _rows = page.rows;
        _total = page.total;
        _loading = false;
      });
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _error = e.message;
          _loading = false;
        });
      }
    } catch (_) {
      if (mounted) {
        setState(() {
          _error = tr('Could not load.');
          _loading = false;
        });
      }
    }
  }

  void _notice(String message, {bool error = false}) {
    if (!mounted) return;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(
        content: Text(message),
        backgroundColor: error ? Theme.of(context).colorScheme.error : null,
      ));
  }

  Future<void> _open({String? childId}) async {
    final saved = await showResourceForm(
      context,
      admin: widget.admin,
      slug: widget.slug,
      id: childId,
      relation: _name,
      parentId: widget.parentId,
      title: _singular(_title),
    );
    if (saved == true) _load();
  }

  static String _singular(String plural) => plural.endsWith('ies')
      ? '${plural.substring(0, plural.length - 3)}y'
      : (plural.endsWith('s')
          ? plural.substring(0, plural.length - 1)
          : plural);

  Future<void> _delete(Map<String, dynamic> row) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(tr('Delete this row?')),
        content: Text(tr('This cannot be undone.')),
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
    if (ok != true) return;
    try {
      await api.relationDelete(
          widget.slug, widget.parentId, _name, recordId(row));
      _load();
    } on ApiException catch (e) {
      _notice(e.message, error: true);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final columns = _columns;
    final canEdit = _can('update');
    final canDelete = _can('delete');

    Widget body;
    if (_error != null) {
      body = Padding(
          padding: const EdgeInsets.all(16),
          child:
              Text(_error!, style: TextStyle(color: theme.colorScheme.error)));
    } else if (_rows.isEmpty) {
      body = Padding(
        padding: const EdgeInsets.all(18),
        child: Text(_loading ? 'Loading…' : tr('Nothing added yet.'),
            style: TextStyle(color: theme.colorScheme.onSurfaceVariant)),
      );
    } else {
      Widget cell(Map<String, dynamic> col, Map<String, dynamic> row) =>
          adminCell(context, col, row,
              onNotice: _notice,
              admin: widget.admin,
              currency: '${widget.admin.schema['currency'] ?? ''}');
      Widget actions(Map<String, dynamic> row) =>
          Row(mainAxisSize: MainAxisSize.min, children: [
            if (canEdit)
              IconButton(
                key: Key('rel-$_name-edit-${recordId(row)}'),
                tooltip: tr('Edit'),
                visualDensity: VisualDensity.compact,
                icon: const Icon(Icons.edit_outlined, size: 18),
                onPressed: () => _open(childId: recordId(row)),
              ),
            if (canDelete)
              IconButton(
                key: Key('rel-$_name-delete-${recordId(row)}'),
                tooltip: tr('Delete'),
                visualDensity: VisualDensity.compact,
                icon: Icon(Icons.delete_outline,
                    size: 18, color: theme.colorScheme.error),
                onPressed: () => _delete(row),
              ),
          ]);
      String label(Map<String, dynamic> col) =>
          '${col['label'] ?? humanize('${col['name']}')}';

      body = LayoutBuilder(builder: (context, c) {
        // Like the resource list: cards when there is no room for a table
        // (a phone, or a narrow form dialog).
        if (c.maxWidth < relationCardBreakpoint) {
          final imageColumn = columns
              .cast<Map<String, dynamic>?>()
              .firstWhere((c) => c!['type'] == 'image', orElse: () => null);
          final cardColumns = [
            for (final c in columns)
              if (imageColumn == null || c['name'] != imageColumn['name']) c
          ];
          return Column(
            key: Key('rel-$_name-cards'),
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              for (final (i, row) in _rows.indexed) ...[
                if (i > 0) const Divider(height: 1),
                InkWell(
                  key: Key('rel-$_name-card-${recordId(row)}'),
                  onTap: canEdit ? () => _open(childId: recordId(row)) : null,
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(14, 10, 6, 10),
                    child: Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          if (imageColumn != null)
                            Padding(
                              padding: const EdgeInsets.only(right: 12, top: 2),
                              child: cardPicture(context, imageColumn, row,
                                  size: 44),
                            ),
                          Expanded(
                              child: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                Row(children: [
                                  Expanded(
                                    child: cardColumns.isEmpty
                                        ? Text(recordId(row))
                                        : DefaultTextStyle.merge(
                                            style: const TextStyle(
                                                fontWeight: FontWeight.w600,
                                                fontSize: 15),
                                            child:
                                                cell(cardColumns.first, row)),
                                  ),
                                  actions(row),
                                ]),
                                if (cardColumns.length > 1) ...[
                                  const SizedBox(height: 4),
                                  Wrap(spacing: 16, runSpacing: 6, children: [
                                    for (final col in cardColumns.skip(1))
                                      Row(
                                          mainAxisSize: MainAxisSize.min,
                                          children: [
                                            Text('${label(col)}: ',
                                                style: theme.textTheme.bodySmall
                                                    ?.copyWith(
                                                        color: theme.colorScheme
                                                            .onSurfaceVariant)),
                                            cell(col, row),
                                          ]),
                                  ]),
                                ],
                              ]))
                        ]),
                  ),
                ),
              ],
            ],
          );
        }
        return SingleChildScrollView(
          key: Key('rel-$_name-table'),
          scrollDirection: Axis.horizontal,
          child: ConstrainedBox(
            constraints: BoxConstraints(minWidth: c.maxWidth),
            child: DataTable(
              columnSpacing: 24,
              horizontalMargin: 16,
              headingRowHeight: 38,
              dataRowMinHeight: 42,
              dataRowMaxHeight: 50,
              columns: [
                for (final col in columns)
                  DataColumn(
                    label: Text(label(col).toUpperCase()),
                    numeric: col['type'] == 'money' || col['align'] == 'right',
                  ),
                if (canEdit || canDelete) const DataColumn(label: Text('')),
              ],
              rows: [
                for (final row in _rows)
                  DataRow(cells: [
                    for (final col in columns) DataCell(cell(col, row)),
                    if (canEdit || canDelete) DataCell(actions(row)),
                  ]),
              ],
            ),
          ),
        );
      });
    }

    return Container(
      decoration: BoxDecoration(
        border: Border.all(color: theme.dividerColor),
        borderRadius: BorderRadius.circular(12),
      ),
      clipBehavior: Clip.antiAlias,
      child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 8, 8, 8),
          child: Row(children: [
            Icon(
                adminIcon(widget.relation['icon'] as String?,
                    fallback: Icons.list_alt_outlined),
                size: 18,
                color: theme.colorScheme.onSurfaceVariant),
            const SizedBox(width: 8),
            Expanded(
              child: Text(_total > 0 ? '$_title ($_total)' : _title,
                  style: theme.textTheme.titleSmall
                      ?.copyWith(fontWeight: FontWeight.w700)),
            ),
            if (_can('create'))
              TextButton.icon(
                key: Key('rel-$_name-add'),
                onPressed: () => _open(),
                icon: const Icon(Icons.add, size: 18),
                label: Text(tr('Add')),
              ),
          ]),
        ),
        const Divider(),
        if (_loading && _rows.isNotEmpty)
          const LinearProgressIndicator(minHeight: 2),
        body,
        if (_total > _rows.length)
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 16, 10),
            child: Text(
                tr('Showing {n} of {total}',
                    {'n': _rows.length, 'total': _total}),
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ),
      ]),
    );
  }
}
