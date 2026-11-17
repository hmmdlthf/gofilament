import 'dart:async';

import 'package:flutter/material.dart';

import 'api.dart';
import 'export.dart';
import 'links.dart';

/// How often the bell checks for new notifications.
Duration notificationPollInterval = const Duration(seconds: 60);

/// The topbar bell: unread count, and the list of notifications.
class NotificationBell extends StatefulWidget {
  const NotificationBell({super.key, required this.api});
  final AdminApi api;

  @override
  State<NotificationBell> createState() => _NotificationBellState();
}

class _NotificationBellState extends State<NotificationBell> {
  int _unread = 0;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _refresh();
    _timer = Timer.periodic(notificationPollInterval, (_) => _refresh());
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  Future<void> _refresh() async {
    try {
      final data = await widget.api.notifications();
      if (mounted) {
        setState(() => _unread = (data['unread_count'] as num?)?.toInt() ?? 0);
      }
    } catch (_) {}
  }

  Future<void> _open() async {
    await showDialog<void>(
      context: context,
      builder: (_) => _NotificationsDialog(api: widget.api),
    );
    _refresh();
  }

  @override
  Widget build(BuildContext context) {
    return IconButton(
      key: const Key('notifications'),
      tooltip: 'Notifications',
      onPressed: _open,
      icon: Badge(
        key: const Key('notifications-badge'),
        isLabelVisible: _unread > 0,
        label: Text('$_unread'),
        child: const Icon(Icons.notifications_none),
      ),
    );
  }
}

class _NotificationsDialog extends StatefulWidget {
  const _NotificationsDialog({required this.api});
  final AdminApi api;

  @override
  State<_NotificationsDialog> createState() => _NotificationsDialogState();
}

class _NotificationsDialogState extends State<_NotificationsDialog> {
  List<Map<String, dynamic>>? _items;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final data = await widget.api.notifications();
      final raw = data['notifications'];
      if (mounted) {
        setState(() => _items = raw is List
            ? [
                for (final n in raw)
                  if (n is Map) Map<String, dynamic>.from(n)
              ]
            : []);
      }
    } catch (_) {
      if (mounted) setState(() => _items = []);
    }
  }

  static final _jobDownload = RegExp(r'/jobs/([A-Za-z0-9_]+)/download$');

  Future<void> _act(Map<String, dynamic> n) async {
    if (n['read'] != true) {
      try {
        await widget.api.markNotificationRead('${n['id']}');
      } catch (_) {}
    }
    final url = '${n['action_url'] ?? ''}';
    if (url.isEmpty) {
      _load();
      return;
    }
    final job = _jobDownload.firstMatch(url);
    if (job != null) {
      // A finished background job: download through the API.
      try {
        final f =
            await widget.api.downloadFile('/jobs/${job.group(1)}/download');
        await saveFile(f.name, f.bytes, 'application/octet-stream');
      } catch (_) {}
    } else {
      openLink(Uri.parse(widget.api.baseUrl).resolve(url).toString());
    }
    if (mounted) Navigator.pop(context);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final items = _items;
    Widget body;
    if (items == null) {
      body = const Center(child: CircularProgressIndicator());
    } else if (items.isEmpty) {
      body = const Text('No notifications.', key: Key('notifications-empty'));
    } else {
      body = ListView.separated(
        shrinkWrap: true,
        itemCount: items.length,
        separatorBuilder: (_, __) => const Divider(height: 1),
        itemBuilder: (context, i) {
          final n = items[i];
          final color = switch ('${n['type']}') {
            'success' => Colors.green,
            'warning' => Colors.orange,
            'danger' => theme.colorScheme.error,
            _ => theme.colorScheme.primary,
          };
          return ListTile(
            key: Key('notification-${n['id']}'),
            leading: Icon(Icons.circle,
                size: 10,
                color: n['read'] == true ? Colors.transparent : color),
            title: Text('${n['title'] ?? ''}',
                style: TextStyle(
                    fontWeight: n['read'] == true
                        ? FontWeight.normal
                        : FontWeight.w600)),
            subtitle:
                '${n['body'] ?? ''}'.isEmpty ? null : Text('${n['body']}'),
            trailing: '${n['action_label'] ?? ''}'.isEmpty
                ? null
                : Text('${n['action_label']}',
                    style: TextStyle(color: theme.colorScheme.primary)),
            onTap: () => _act(n),
          );
        },
      );
    }
    return AlertDialog(
      title: const Text('Notifications'),
      content: SizedBox(width: 420, child: body),
      actions: [
        TextButton(
          key: const Key('notifications-clear'),
          onPressed: () async {
            await widget.api.clearNotifications();
            _load();
          },
          child: const Text('Clear'),
        ),
        TextButton(
          key: const Key('notifications-read-all'),
          onPressed: () async {
            await widget.api.markAllNotificationsRead();
            _load();
          },
          child: const Text('Mark all read'),
        ),
        TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Close')),
      ],
    );
  }
}
