import 'dart:async';

import 'package:flutter/material.dart';

import 'controller.dart';
import 'icons.dart';

/// Searches every globally searchable resource (`/api/global-search`) and
/// returns the URL of the chosen result.
class GlobalSearchDialog extends StatefulWidget {
  const GlobalSearchDialog({super.key, required this.admin});

  final AdminController admin;

  @override
  State<GlobalSearchDialog> createState() => _GlobalSearchDialogState();
}

class _GlobalSearchDialogState extends State<GlobalSearchDialog> {
  Timer? _debounce;
  List<Map<String, dynamic>> _groups = const [];
  bool _loading = false;
  String _query = '';
  int _seq = 0;

  @override
  void dispose() {
    _debounce?.cancel();
    super.dispose();
  }

  void _changed(String q) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 250), () => _search(q));
  }

  Future<void> _search(String q) async {
    final seq = ++_seq;
    setState(() {
      _query = q.trim();
      _loading = _query.isNotEmpty;
    });
    if (_query.isEmpty) {
      setState(() => _groups = const []);
      return;
    }
    try {
      final data =
          await widget.admin.api.get('/global-search', {'query': _query});
      if (!mounted || seq != _seq) return;
      final groups = data is Map && data['groups'] is List
          ? (data['groups'] as List)
              .whereType<Map>()
              .map((m) => Map<String, dynamic>.from(m))
              .toList()
          : <Map<String, dynamic>>[];
      setState(() {
        _groups = groups;
        _loading = false;
      });
    } catch (_) {
      if (mounted && seq == _seq) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Dialog(
      alignment: Alignment.topCenter,
      insetPadding: const EdgeInsets.fromLTRB(16, 60, 16, 16),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 640, maxHeight: 560),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Padding(
            padding: const EdgeInsets.all(12),
            child: TextField(
              key: const Key('global-search-input'),
              autofocus: true,
              decoration: InputDecoration(
                hintText: 'Search everything',
                prefixIcon: const Icon(Icons.search),
                suffixIcon: _loading
                    ? const Padding(
                        padding: EdgeInsets.all(12),
                        child: SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2)))
                    : null,
              ),
              onChanged: _changed,
              onSubmitted: _search,
            ),
          ),
          Flexible(
            child: _query.isNotEmpty && !_loading && _groups.isEmpty
                ? Padding(
                    padding: const EdgeInsets.all(24),
                    child: Text('No results for "$_query".',
                        style: TextStyle(
                            color: theme.colorScheme.onSurfaceVariant)),
                  )
                : ListView(shrinkWrap: true, children: [
                    for (final g in _groups) ...[
                      Padding(
                        padding: const EdgeInsets.fromLTRB(16, 8, 16, 4),
                        child: Text('${g['name'] ?? ''}',
                            style: theme.textTheme.labelMedium?.copyWith(
                                color: theme.colorScheme.onSurfaceVariant)),
                      ),
                      for (final item in (g['items'] is List
                              ? g['items'] as List
                              : const [])
                          .whereType<Map>())
                        ListTile(
                          key: Key('search-result-${item['url']}'),
                          leading: Icon(adminIcon(item['icon'] as String?,
                              fallback: Icons.arrow_outward)),
                          title: Text('${item['title'] ?? ''}'),
                          subtitle: item['details'] is Map &&
                                  (item['details'] as Map).isNotEmpty
                              ? Text((item['details'] as Map)
                                  .entries
                                  .map((e) => '${e.key}: ${e.value}')
                                  .join(' · '))
                              : null,
                          onTap: () =>
                              Navigator.of(context).pop('${item['url']}'),
                        ),
                    ],
                  ]),
          ),
        ]),
      ),
    );
  }
}
