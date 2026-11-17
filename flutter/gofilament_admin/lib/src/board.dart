import 'package:flutter/material.dart';

import 'api.dart';

/// A resource's records as cards in columns (Resource.Kanban). Dragging a
/// card to another column sets the board's field.
class KanbanBoardView extends StatefulWidget {
  const KanbanBoardView({
    super.key,
    required this.api,
    required this.slug,
    this.search = '',
    this.onOpen,
  });

  final AdminApi api;
  final String slug;
  final String search;
  final ValueChanged<String>? onOpen;

  @override
  State<KanbanBoardView> createState() => _KanbanBoardViewState();
}

class _KanbanBoardViewState extends State<KanbanBoardView> {
  List<Map<String, dynamic>>? _columns;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void didUpdateWidget(KanbanBoardView old) {
    super.didUpdateWidget(old);
    if (old.search != widget.search) _load();
  }

  Future<void> _load() async {
    try {
      final cols = await widget.api.board(widget.slug, search: widget.search);
      if (mounted) setState(() => _columns = cols);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  List<Map<String, dynamic>> _cards(Map<String, dynamic> col) =>
      (col['cards'] is List ? col['cards'] as List : const [])
          .whereType<Map>()
          .map((m) => Map<String, dynamic>.from(m))
          .toList();

  Future<void> _move(Map<String, dynamic> card, Map<String, dynamic> from,
      Map<String, dynamic> to) async {
    if (identical(from, to)) return;
    final fromCards = _cards(from), toCards = _cards(to);
    fromCards.removeWhere((c) => c['id'] == card['id']);
    toCards.insert(0, card);
    setState(() {
      from['cards'] = fromCards;
      to['cards'] = toCards;
      from['total'] = ((from['total'] as num?) ?? 1) - 1;
      to['total'] = ((to['total'] as num?) ?? 0) + 1;
    });
    try {
      final v = await widget.api.moveCard(
          widget.slug, '${card['id']}', '${to['value']}',
          version: card['version']?.toString());
      if (v != null) card['version'] = v;
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text(e.isConflict
              ? 'This record was changed by someone else.'
              : e.message)));
      _load();
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (_error != null) return Center(child: Text(_error!));
    final cols = _columns;
    if (cols == null) return const Center(child: CircularProgressIndicator());
    return ListView(
      key: const Key('kanban-board'),
      scrollDirection: Axis.horizontal,
      padding: const EdgeInsets.all(12),
      children: [
        for (final col in cols)
          DragTarget<(Map<String, dynamic>, Map<String, dynamic>)>(
            key: Key('kanban-column-${col['value']}'),
            onAcceptWithDetails: (d) => _move(d.data.$1, d.data.$2, col),
            builder: (context, candidates, _) => Container(
              width: 260,
              margin: const EdgeInsetsDirectional.only(end: 12),
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: candidates.isNotEmpty
                    ? theme.colorScheme.primaryContainer
                    : theme.colorScheme.surfaceContainerLow,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text('${col['label']}  ${col['total'] ?? ''}',
                      style: const TextStyle(fontWeight: FontWeight.w700)),
                  const SizedBox(height: 8),
                  Expanded(
                    child: ListView(children: [
                      for (final card in _cards(col))
                        Draggable<(Map<String, dynamic>, Map<String, dynamic>)>(
                          key: Key('kanban-card-${card['id']}'),
                          data: (card, col),
                          feedback: SizedBox(
                              width: 240,
                              child: Card(
                                  child: Padding(
                                      padding: const EdgeInsets.all(10),
                                      child: Text('${card['title']}')))),
                          childWhenDragging:
                              Opacity(opacity: 0.4, child: _card(card)),
                          child: _card(card),
                        ),
                    ]),
                  ),
                ],
              ),
            ),
          ),
      ],
    );
  }

  Widget _card(Map<String, dynamic> card) => Card(
        child: InkWell(
          onTap: widget.onOpen == null
              ? null
              : () => widget.onOpen!('${card['id']}'),
          child: Padding(
            padding: const EdgeInsets.all(10),
            child:
                Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text('${card['title']}',
                  style: const TextStyle(fontWeight: FontWeight.w600)),
              if ('${card['description'] ?? ''}'.isNotEmpty)
                Text('${card['description']}',
                    style: Theme.of(context).textTheme.bodySmall),
            ]),
          ),
        ),
      );
}
