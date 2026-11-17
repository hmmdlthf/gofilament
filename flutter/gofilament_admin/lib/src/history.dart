import 'package:flutter/material.dart';
import 'i18n.dart';

import 'api.dart';

/// Shows a record's history: who changed what, and when.
Future<void> showRecordHistory(BuildContext context,
    {required AdminApi api, required String slug, required String id}) {
  return showDialog<void>(
    context: context,
    builder: (_) => _HistoryDialog(api: api, slug: slug, id: id),
  );
}

class _HistoryDialog extends StatefulWidget {
  const _HistoryDialog(
      {required this.api, required this.slug, required this.id});
  final AdminApi api;
  final String slug;
  final String id;

  @override
  State<_HistoryDialog> createState() => _HistoryDialogState();
}

class _HistoryDialogState extends State<_HistoryDialog> {
  List<Map<String, dynamic>>? _entries;
  String? _error;

  @override
  void initState() {
    super.initState();
    widget.api.history(widget.slug, widget.id).then((e) {
      if (mounted) setState(() => _entries = e);
    }).catchError((Object e) {
      if (mounted) {
        setState(() => _error =
            e is ApiException ? e.message : tr('Could not load the history.'));
      }
    });
  }

  static String _verb(Map<String, dynamic> e) => switch ('${e['event']}') {
        'created' => 'created this record',
        'updated' => 'changed',
        'deleted' => 'deleted this record',
        'restored' => 'restored this record',
        _ => '${e['description'] ?? e['event'] ?? ''}',
      };

  static String _when(dynamic raw) {
    final t = DateTime.tryParse('${raw ?? ''}')?.toLocal();
    if (t == null) return '';
    String two(int n) => n.toString().padLeft(2, '0');
    return '${t.year}-${two(t.month)}-${two(t.day)} ${two(t.hour)}:${two(t.minute)}';
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    Widget body;
    if (_error != null) {
      body = Text(_error!);
    } else if (_entries == null) {
      body = const Center(child: CircularProgressIndicator());
    } else if (_entries!.isEmpty) {
      body = Text(tr('No changes recorded yet.'), key: Key('history-empty'));
    } else {
      body = ListView.separated(
        shrinkWrap: true,
        itemCount: _entries!.length,
        separatorBuilder: (_, __) => const Divider(height: 16),
        itemBuilder: (context, i) {
          final e = _entries![i];
          final changes =
              e['changes'] is List ? e['changes'] as List : const [];
          final who = '${e['causer_name'] ?? ''}'.isEmpty
              ? 'System'
              : '${e['causer_name']}';
          return Column(
            key: Key('history-entry-$i'),
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text.rich(TextSpan(children: [
                TextSpan(
                    text: who,
                    style: const TextStyle(fontWeight: FontWeight.w700)),
                TextSpan(text: ' ${_verb(e)}  '),
                TextSpan(
                    text: _when(e['at']),
                    style: TextStyle(
                        color: theme.colorScheme.onSurfaceVariant,
                        fontSize: 12)),
              ])),
              for (final c in changes)
                if (c is Map)
                  Padding(
                    padding: const EdgeInsets.only(top: 4, left: 8),
                    child: Text.rich(TextSpan(children: [
                      TextSpan(
                          text: '${c['label'] ?? c['field']}: ',
                          style: TextStyle(
                              color: theme.colorScheme.onSurfaceVariant)),
                      TextSpan(
                          text:
                              '${c['old'] ?? ''}'.isEmpty ? '—' : '${c['old']}',
                          style: TextStyle(
                              decoration: TextDecoration.lineThrough,
                              color: theme.colorScheme.error)),
                      const TextSpan(text: '  →  '),
                      TextSpan(
                          text:
                              '${c['new'] ?? ''}'.isEmpty ? '—' : '${c['new']}',
                          style: const TextStyle(fontWeight: FontWeight.w600)),
                    ])),
                  ),
            ],
          );
        },
      );
    }
    return AlertDialog(
      title: Text(tr('History')),
      content: SizedBox(width: 520, child: body),
      actions: [
        TextButton(
            onPressed: () => Navigator.pop(context), child: Text(tr('Close'))),
      ],
    );
  }
}
