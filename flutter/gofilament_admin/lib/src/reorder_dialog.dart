import 'package:flutter/material.dart';
import 'i18n.dart';

/// Lets the user drag rows into a new order. Returns the ids in the new
/// order, or null when cancelled.
Future<List<String>?> showReorderDialog(BuildContext context,
    {required List<({String id, String title})> rows}) {
  return showDialog<List<String>>(
    context: context,
    builder: (_) => _ReorderDialog(rows: rows),
  );
}

class _ReorderDialog extends StatefulWidget {
  const _ReorderDialog({required this.rows});
  final List<({String id, String title})> rows;

  @override
  State<_ReorderDialog> createState() => _ReorderDialogState();
}

class _ReorderDialogState extends State<_ReorderDialog> {
  late final List<({String id, String title})> _rows = [...widget.rows];

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(tr('Reorder')),
      content: SizedBox(
        width: 420,
        height: 420,
        child: ReorderableListView(
          buildDefaultDragHandles: false,
          onReorderItem: (from, to) => setState(() {
            _rows.insert(to, _rows.removeAt(from));
          }),
          children: [
            for (var i = 0; i < _rows.length; i++)
              ListTile(
                key: ValueKey(_rows[i].id),
                title: Text(_rows[i].title),
                trailing: ReorderableDragStartListener(
                  key: Key('reorder-handle-${_rows[i].id}'),
                  index: i,
                  child: const Icon(Icons.drag_handle),
                ),
              ),
          ],
        ),
      ),
      actions: [
        TextButton(
            onPressed: () => Navigator.pop(context), child: Text(tr('Cancel'))),
        FilledButton(
            key: const Key('reorder-save'),
            onPressed: () =>
                Navigator.pop(context, [for (final r in _rows) r.id]),
            child: Text(tr('Save order'))),
      ],
    );
  }
}
