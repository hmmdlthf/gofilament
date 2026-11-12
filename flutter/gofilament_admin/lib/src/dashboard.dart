import 'package:flutter/material.dart';

import 'api.dart';
import 'components.dart';
import 'controller.dart';
import 'icons.dart';
import 'theme.dart';

/// The panel dashboard: the widgets registered on the server.
class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key, required this.admin});

  final AdminController admin;

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
        setState(() => _error = 'Could not load the dashboard.');
      }
    }
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
          Text(name.isEmpty ? 'Welcome' : 'Welcome back, $name',
              style: theme.textTheme.headlineSmall
                  ?.copyWith(fontWeight: FontWeight.w700)),
          const SizedBox(height: 4),
          Text('Here is how things stand right now.',
              style: theme.textTheme.bodyMedium
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          const SizedBox(height: 20),
          for (final w in _widgets!) ...[
            if (w['stats'] is List)
              _StatsRow(stats: (w['stats'] as List).whereType<Map>().toList()),
            if (w['type'] == 'custom')
              _CustomWidget(admin: widget.admin, dto: w),
            const SizedBox(height: 16),
          ],
          if (_widgets!.isEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 40),
              child: Center(
                child: Text('Nothing to show yet.',
                    style:
                        TextStyle(color: theme.colorScheme.onSurfaceVariant)),
              ),
            ),
        ],
      ),
    );
  }
}

class _StatsRow extends StatelessWidget {
  const _StatsRow({required this.stats});

  final List<Map> stats;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(builder: (context, c) {
      final columns = c.maxWidth >= 1100 ? 4 : (c.maxWidth >= 620 ? 2 : 1);
      const gap = 14.0;
      final width = (c.maxWidth - gap * (columns - 1)) / columns;
      return Wrap(
        spacing: gap,
        runSpacing: gap,
        children: [
          for (final s in stats)
            SizedBox(width: width, child: _StatCard(stat: s))
        ],
      );
    });
  }
}

class _StatCard extends StatelessWidget {
  const _StatCard({required this.stat});

  final Map stat;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final palette = AdminPalette.of(context);
    final dark = theme.brightness == Brightness.dark;
    final role = '${stat['color'] ?? 'primary'}';
    final color = palette.color(role, dark ? 400 : 600);
    final description = '${stat['description'] ?? ''}';
    final iconName = '${stat['description_icon'] ?? ''}';

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(18),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Row(children: [
            Container(
                width: 8,
                height: 8,
                decoration:
                    BoxDecoration(color: color, shape: BoxShape.circle)),
            const SizedBox(width: 8),
            Expanded(
              child: Text('${stat['label'] ?? ''}',
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                      fontWeight: FontWeight.w600)),
            ),
          ]),
          const SizedBox(height: 10),
          Text('${stat['value'] ?? ''}',
              style: theme.textTheme.headlineSmall
                  ?.copyWith(fontWeight: FontWeight.w700)),
          if (description.isEmpty) const SizedBox(height: 22),
          if (description.isNotEmpty) ...[
            const SizedBox(height: 6),
            Row(children: [
              if (iconName.isNotEmpty) ...[
                Icon(adminIcon(iconName, fallback: Icons.info_outline),
                    size: 14, color: color),
                const SizedBox(width: 4),
              ],
              Expanded(
                child: Text(description,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: theme.textTheme.bodySmall?.copyWith(color: color)),
              ),
            ]),
          ],
        ]),
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
      body = Text('This widget is not available in this app.',
          style: TextStyle(color: theme.colorScheme.onSurfaceVariant));
    } else {
      body = builder(context, AdminWidgetContext(admin: admin, widget: dto));
    }
    return Card(
      key: Key('widget-${dto['id']}'),
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
