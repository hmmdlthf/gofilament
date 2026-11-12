import 'package:flutter/material.dart';

import 'api.dart';
import 'cells.dart';
import 'controller.dart';
import 'format.dart';
import 'icons.dart';
import 'resource_form.dart';

/// The relation managers of a record (for example a product's variants and
/// barcodes), shown under its form. Each is a small table with add, edit and
/// delete, driven by the relation's server-side description.
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

class _RelationPanelsState extends State<RelationPanels> {
  List<Map<String, dynamic>>? _relations;

  @override
  void initState() {
    super.initState();
    final declared = widget.admin.resource(widget.slug)?['relations'];
    if (declared is List && declared.isNotEmpty) _load();
  }

  Future<void> _load() async {
    try {
      final r = await widget.admin.api.relations(widget.slug, widget.parentId);
      if (mounted) setState(() => _relations = r);
    } catch (_) {
      // the record form stays usable without its relations
    }
  }

  @override
  Widget build(BuildContext context) {
    final relations = _relations;
    if (relations == null || relations.isEmpty) return const SizedBox.shrink();
    return Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
      for (final r in relations)
        Padding(
          padding: const EdgeInsets.only(bottom: 14),
          child: _RelationTable(
            admin: widget.admin,
            slug: widget.slug,
            parentId: widget.parentId,
            relation: r,
            readOnly: widget.readOnly,
          ),
        ),
    ]);
  }
}

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
          _error = 'Could not load.';
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
        title: const Text('Delete this row?'),
        content: const Text('This cannot be undone.'),
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
        child: Text(_loading ? 'Loading…' : 'Nothing added yet.',
            style: TextStyle(color: theme.colorScheme.onSurfaceVariant)),
      );
    } else {
      body = LayoutBuilder(
        builder: (context, c) => SingleChildScrollView(
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
                    label: Text('${col['label'] ?? humanize('${col['name']}')}'
                        .toUpperCase()),
                    numeric: col['type'] == 'money' || col['align'] == 'right',
                  ),
                if (canEdit || canDelete) const DataColumn(label: Text('')),
              ],
              rows: [
                for (final row in _rows)
                  DataRow(cells: [
                    for (final col in columns)
                      DataCell(adminCell(context, col, row,
                          onNotice: _notice,
                          admin: widget.admin,
                          currency:
                              '${widget.admin.schema['currency'] ?? ''}')),
                    if (canEdit || canDelete)
                      DataCell(Row(mainAxisSize: MainAxisSize.min, children: [
                        if (canEdit)
                          IconButton(
                            key: Key('rel-$_name-edit-${recordId(row)}'),
                            tooltip: 'Edit',
                            visualDensity: VisualDensity.compact,
                            icon: const Icon(Icons.edit_outlined, size: 18),
                            onPressed: () => _open(childId: recordId(row)),
                          ),
                        if (canDelete)
                          IconButton(
                            key: Key('rel-$_name-delete-${recordId(row)}'),
                            tooltip: 'Delete',
                            visualDensity: VisualDensity.compact,
                            icon: Icon(Icons.delete_outline,
                                size: 18, color: theme.colorScheme.error),
                            onPressed: () => _delete(row),
                          ),
                      ])),
                  ]),
              ],
            ),
          ),
        ),
      );
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
                label: const Text('Add'),
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
            child: Text('Showing ${_rows.length} of $_total',
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ),
      ]),
    );
  }
}
