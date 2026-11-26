import 'package:flutter/material.dart';
import 'i18n.dart';

import 'api.dart';
import 'chart.dart';
import 'components.dart';
import 'controller.dart';
import 'dashboard_widgets.dart';

/// The panel dashboard: the widgets registered on the server.
class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key, required this.admin, this.onOpen});

  final AdminController admin;

  /// Opens a panel URL from a stat card, table row, quick action or banner.
  final DashboardOpen? onOpen;

  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  List<Map<String, dynamic>>? _widgets;
  String? _error;

  @override
  void initState() {
    super.initState();
    final initial = widget.admin.schema['widgets'];
    if (initial is List) {
      _widgets = initial
          .whereType<Map>()
          .map((m) => Map<String, dynamic>.from(m))
          .toList();
    }
    _load();
  }

  Future<void> _load() async {
    try {
      final w = await widget.admin.api.widgets();
      if (mounted) {
        setState(() {
          _widgets = w;
          _error = null;
        });
      }
    } on ApiException catch (e) {
      if (mounted && _widgets == null) setState(() => _error = e.message);
    } catch (_) {
      if (mounted && _widgets == null) {
        setState(() => _error = tr('Could not load the dashboard.'));
      }
    }
  }

  /// Widgets in their `sort` order (stable for equal values).
  static List<Map<String, dynamic>> _sorted(List<Map<String, dynamic>> ws) {
    final indexed = [for (var i = 0; i < ws.length; i++) (i, ws[i])];
    int sortOf(Map<String, dynamic> w) =>
        w['sort'] is num ? (w['sort'] as num).toInt() : 0;
    indexed.sort((a, b) {
      final c = sortOf(a.$2).compareTo(sortOf(b.$2));
      return c != 0 ? c : a.$1.compareTo(b.$1);
    });
    return [for (final e in indexed) e.$2];
  }

  Widget _widget(Map<String, dynamic> w) {
    final open = widget.onOpen;
    switch ('${w['type'] ?? ''}') {
      case 'chart':
        return AdminChartCard(
            key: ValueKey('chart-card-${w['id']}'),
            admin: widget.admin,
            dto: w);
      case 'table':
        return DashboardTable(dto: w, onOpen: open);
      case 'breakdown':
        return DashboardBreakdown(dto: w);
      case 'progress':
        return DashboardProgress(dto: w);
      case 'quick_actions':
        return DashboardQuickActions(dto: w, onOpen: open);
      case 'banner':
        return DashboardBanner(dto: w, onOpen: open);
      case 'custom':
        return _CustomWidget(admin: widget.admin, dto: w);
    }
    if (w['stats'] is List) {
      return DashboardStatsRow(
          stats: (w['stats'] as List).whereType<Map>().toList(), onOpen: open);
    }
    return const SizedBox.shrink();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (_widgets == null) {
      return Center(
          child: _error == null
              ? const CircularProgressIndicator()
              : Text(_error!));
    }
    final name = '${widget.admin.user['name'] ?? ''}'.split(' ').first;

    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          Text(
              name.isEmpty
                  ? 'Welcome'
                  : tr('Welcome back, {name}', {'name': name}),
              style: theme.textTheme.headlineSmall
                  ?.copyWith(fontWeight: FontWeight.w700)),
          const SizedBox(height: 4),
          Text(tr('Here is how things stand right now.'),
              style: theme.textTheme.bodyMedium
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          const SizedBox(height: 20),
          DashboardGrid(
            widgets: _sorted(_widgets!),
            builder: (w) => _widget(w),
          ),
          if (_widgets!.isEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 40),
              child: Center(
                child: Text(tr('Nothing to show yet.'),
                    style:
                        TextStyle(color: theme.colorScheme.onSurfaceVariant)),
              ),
            ),
        ],
      ),
    );
  }
}

/// A dashboard widget drawn by an app component (filament.FlutterWidget).
class _CustomWidget extends StatelessWidget {
  const _CustomWidget({required this.admin, required this.dto});

  final AdminController admin;
  final Map<String, dynamic> dto;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final heading = '${dto['heading'] ?? ''}';
    final builder = admin.components.widgets['${dto['component']}'];
    final Widget body;
    if (dto['error'] != null) {
      body = Text('${dto['error']}',
          style: TextStyle(color: theme.colorScheme.error));
    } else if (builder == null) {
      body = Text(tr('This widget is not available in this app.'),
          style: TextStyle(color: theme.colorScheme.onSurfaceVariant));
    } else {
      body = builder(context, AdminWidgetContext(admin: admin, widget: dto));
    }
    return Card(
      key: Key('widget-${dto['id']}'),
      margin: EdgeInsets.zero,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (heading.isNotEmpty) ...[
              Text(heading,
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w600)),
              const SizedBox(height: 12),
            ],
            body,
          ],
        ),
      ),
    );
  }
}
