import 'dart:math' as math;

import 'package:flutter/material.dart';

import 'format.dart';
import 'i18n.dart';
import 'icons.dart';
import 'theme.dart';

/// Opens a panel URL (a resource list, optionally filtered, a record or a page).
typedef DashboardOpen = void Function(String url);

/// The colour of a widget colour token: a palette role (primary, success,
/// warning, danger, info, accent, gray), a Tailwind-style name used by the Go
/// widgets (emerald, amber, red, sky, purple, ...) or a hex colour.
Color dashboardColor(BuildContext context, String? token, {int shade = 600}) {
  final palette = AdminPalette.of(context);
  final dark = Theme.of(context).brightness == Brightness.dark;
  final s = dark ? (shade >= 500 ? 400 : shade) : shade;
  final t = (token ?? '').trim();
  if (t.startsWith('#') || t.startsWith('rgb')) {
    final c = parseColor(t);
    if (c != null) return c;
  }
  const aliases = {
    'emerald': 'success',
    'green': 'success',
    'teal': 'success',
    'amber': 'warning',
    'yellow': 'warning',
    'orange': 'warning',
    'red': 'danger',
    'rose': 'danger',
    'sky': 'info',
    'blue': 'info',
    'cyan': 'info',
    'purple': 'accent',
    'violet': 'accent',
    'indigo': 'primary',
    'slate': 'gray',
    'grey': 'gray',
  };
  return palette.color(aliases[t] ?? (t.isEmpty ? 'primary' : t), s);
}

/// Formats a widget value: "money", "percent", "number" or plain text.
String formatWidgetValue(dynamic value, String format, {String currency = ''}) {
  if (value == null) return '';
  final n = value is num ? value : num.tryParse('$value');
  if (n == null) return '$value';
  switch (format) {
    case 'money':
      return formatMoney(n, currency);
    case 'percent':
      return '${formatNumber(n, decimals: 1)}%';
    case 'number':
      return formatNumber(n, decimals: n == n.roundToDouble() ? 0 : 2);
  }
  return '$value';
}

/// The card every dashboard widget sits in: heading, optional description
/// and the body.
class DashboardCard extends StatelessWidget {
  const DashboardCard({
    super.key,
    required this.heading,
    this.description = '',
    this.trailing,
    required this.child,
  });

  final String heading;
  final String description;
  final Widget? trailing;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      margin: EdgeInsets.zero,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            if (heading.isNotEmpty || trailing != null)
              Row(children: [
                Expanded(
                  child: Text(heading,
                      style: theme.textTheme.titleMedium
                          ?.copyWith(fontWeight: FontWeight.w600)),
                ),
                if (trailing != null) trailing!,
              ]),
            if (description.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 2),
                child: Text(description,
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
              ),
            if (heading.isNotEmpty || description.isNotEmpty)
              const SizedBox(height: 12),
            child,
          ],
        ),
      ),
    );
  }
}

// ------------------------------------------------------------------ stats

/// A row of stat cards (widgets.StatsOverviewWidget): value, description,
/// change against the comparison period and trend line; a card with a URL
/// opens it.
class DashboardStatsRow extends StatelessWidget {
  const DashboardStatsRow({super.key, required this.stats, this.onOpen});

  final List<Map> stats;
  final DashboardOpen? onOpen;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(builder: (context, c) {
      final max = c.maxWidth >= 1100 ? 4 : (c.maxWidth >= 620 ? 2 : 1);
      final columns = math.max(1, math.min(max, stats.length));
      const gap = 14.0;
      final width = (c.maxWidth - gap * (columns - 1)) / columns;
      return Wrap(
        spacing: gap,
        runSpacing: gap,
        children: [
          for (var i = 0; i < stats.length; i++)
            SizedBox(
                width: width,
                child: DashboardStatCard(
                    key: Key('stat-$i'), stat: stats[i], onOpen: onOpen)),
        ],
      );
    });
  }
}

class DashboardStatCard extends StatelessWidget {
  const DashboardStatCard({super.key, required this.stat, this.onOpen});

  final Map stat;
  final DashboardOpen? onOpen;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final role = '${stat['color'] ?? 'primary'}';
    final color = dashboardColor(context, role);
    final description = '${stat['description'] ?? ''}';
    final iconName = '${stat['description_icon'] ?? ''}';
    final url = '${stat['url'] ?? ''}';
    final trend = (stat['chart'] is List ? stat['chart'] as List : const [])
        .map((v) => v is num ? v.toDouble() : double.tryParse('$v') ?? 0)
        .toList();
    final change =
        stat['change'] is num ? (stat['change'] as num).toDouble() : null;
    final lowerIsBetter = stat['lower_is_better'] == true;

    Widget? changeRow;
    if (change != null) {
      final up = change > 0, flat = change == 0;
      final good = flat || (up != lowerIsBetter);
      final c = flat
          ? theme.colorScheme.onSurfaceVariant
          : dashboardColor(context, good ? 'success' : 'danger');
      final caption = '${stat['change_caption'] ?? ''}';
      changeRow = Row(key: const Key('stat-change'), children: [
        Icon(
            flat
                ? Icons.trending_flat
                : (up ? Icons.trending_up : Icons.trending_down),
            size: 16,
            color: c),
        const SizedBox(width: 4),
        Text('${change > 0 ? '+' : ''}${formatNumber(change, decimals: 1)}%',
            style: theme.textTheme.bodySmall
                ?.copyWith(color: c, fontWeight: FontWeight.w600)),
        if (caption.isNotEmpty) ...[
          const SizedBox(width: 4),
          Expanded(
            child: Text(caption,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ),
        ],
      ]);
    }

    final body = Padding(
      padding: const EdgeInsets.all(18),
      child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
        Row(children: [
          Container(
              width: 8,
              height: 8,
              decoration: BoxDecoration(color: color, shape: BoxShape.circle)),
          const SizedBox(width: 8),
          Expanded(
            child: Text('${stat['label'] ?? ''}',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                    fontWeight: FontWeight.w600)),
          ),
          if (url.isNotEmpty)
            Icon(Icons.chevron_right,
                size: 16, color: theme.colorScheme.onSurfaceVariant),
        ]),
        const SizedBox(height: 10),
        Row(crossAxisAlignment: CrossAxisAlignment.end, children: [
          Expanded(
            child: Text('${stat['value'] ?? ''}',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.headlineSmall
                    ?.copyWith(fontWeight: FontWeight.w700)),
          ),
          if (trend.length >= 2)
            SizedBox(
              key: const Key('stat-sparkline'),
              width: 72,
              height: 28,
              child: CustomPaint(painter: _SparklinePainter(trend, color)),
            ),
        ]),
        if (changeRow != null) ...[const SizedBox(height: 6), changeRow],
        if (description.isEmpty && changeRow == null)
          const SizedBox(height: 22),
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
    );
    return Card(
      margin: EdgeInsets.zero,
      clipBehavior: Clip.antiAlias,
      child: url.isEmpty || onOpen == null
          ? body
          : InkWell(onTap: () => onOpen!(url), child: body),
    );
  }
}

class _SparklinePainter extends CustomPainter {
  _SparklinePainter(this.values, this.color);
  final List<double> values;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    var lo = values.reduce(math.min), hi = values.reduce(math.max);
    if (hi == lo) hi = lo + 1;
    final step = size.width / (values.length - 1);
    final path = Path();
    for (var i = 0; i < values.length; i++) {
      final p = Offset(i * step,
          size.height - 2 - (values[i] - lo) / (hi - lo) * (size.height - 4));
      i == 0 ? path.moveTo(p.dx, p.dy) : path.lineTo(p.dx, p.dy);
    }
    final fill = Path.from(path)
      ..lineTo(size.width, size.height)
      ..lineTo(0, size.height)
      ..close();
    canvas.drawPath(fill, Paint()..color = color.withValues(alpha: 0.12));
    canvas.drawPath(
        path,
        Paint()
          ..color = color
          ..style = PaintingStyle.stroke
          ..strokeWidth = 2
          ..strokeJoin = StrokeJoin.round);
  }

  @override
  bool shouldRepaint(_SparklinePainter old) =>
      old.values != values || old.color != color;
}

// ------------------------------------------------------------------ table

/// A small table (widgets.TableWidget). A row with "_url" opens it.
class DashboardTable extends StatelessWidget {
  const DashboardTable({super.key, required this.dto, this.onOpen});

  final Map<String, dynamic> dto;
  final DashboardOpen? onOpen;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final cols = (dto['columns'] is List ? dto['columns'] as List : const [])
        .whereType<Map>()
        .toList();
    final rows = (dto['rows'] is List ? dto['rows'] as List : const [])
        .whereType<Map>()
        .toList();
    TextAlign align(Map c) {
      final a = '${c['align'] ?? ''}';
      if (a == 'center') return TextAlign.center;
      if (a == 'right') return TextAlign.right;
      if (a == 'left') return TextAlign.left;
      return const {'number', 'money', 'percent'}
              .contains('${c['format'] ?? ''}')
          ? TextAlign.right
          : TextAlign.left;
    }

    int flex(Map c) => align(c) == TextAlign.left ? 3 : 2;
    Widget cells(List<Widget> children) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 8),
        child: Row(children: children));

    final empty = '${dto['empty_text'] ?? ''}';
    return DashboardCard(
      key: Key('widget-${dto['id']}'),
      heading: '${dto['heading'] ?? ''}',
      description: '${dto['description'] ?? ''}',
      child: rows.isEmpty
          ? Text(empty.isEmpty ? tr('Nothing to show yet.') : empty,
              style: TextStyle(color: theme.colorScheme.onSurfaceVariant))
          : Column(mainAxisSize: MainAxisSize.min, children: [
              cells([
                for (final c in cols)
                  Expanded(
                    flex: flex(c),
                    child: Text('${c['label'] ?? ''}',
                        textAlign: align(c),
                        style: theme.textTheme.bodySmall?.copyWith(
                            fontWeight: FontWeight.w600,
                            color: theme.colorScheme.onSurfaceVariant)),
                  ),
              ]),
              const Divider(height: 1),
              for (var i = 0; i < rows.length; i++) ...[
                InkWell(
                  key: Key('widget-${dto['id']}-row-$i'),
                  onTap: '${rows[i]['_url'] ?? ''}'.isEmpty || onOpen == null
                      ? null
                      : () => onOpen!('${rows[i]['_url']}'),
                  child: cells([
                    for (final c in cols)
                      Expanded(
                        flex: flex(c),
                        child: Text(
                            formatWidgetValue(rows[i]['${c['name']}'],
                                '${c['format'] ?? ''}'),
                            textAlign: align(c),
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: theme.textTheme.bodyMedium),
                      ),
                  ]),
                ),
                if (i < rows.length - 1) const Divider(height: 1),
              ],
            ]),
    );
  }
}

// ------------------------------------------------------------------ breakdown

/// Shares of a whole (widgets.BreakdownWidget): a segmented bar and a list.
class DashboardBreakdown extends StatelessWidget {
  const DashboardBreakdown({super.key, required this.dto});

  final Map<String, dynamic> dto;

  static const _cycle = [
    'primary',
    'success',
    'warning',
    'info',
    'accent',
    'danger',
    'gray'
  ];

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final items = (dto['items'] is List ? dto['items'] as List : const [])
        .whereType<Map>()
        .toList();
    Color colorOf(int i) {
      final t = '${items[i]['color'] ?? ''}';
      return dashboardColor(context, t.isEmpty ? _cycle[i % _cycle.length] : t);
    }

    double pct(Map m) =>
        m['percentage'] is num ? (m['percentage'] as num).toDouble() : 0;
    return DashboardCard(
      key: Key('widget-${dto['id']}'),
      heading: '${dto['heading'] ?? ''}',
      description: '${dto['description'] ?? ''}',
      child: items.isEmpty
          ? Text(tr('Nothing to show yet.'),
              style: TextStyle(color: theme.colorScheme.onSurfaceVariant))
          : Column(mainAxisSize: MainAxisSize.min, children: [
              ClipRRect(
                borderRadius: BorderRadius.circular(5),
                child: SizedBox(
                  height: 10,
                  child: Row(children: [
                    for (var i = 0; i < items.length; i++)
                      if (pct(items[i]) > 0)
                        Expanded(
                            flex: math.max(1, (pct(items[i]) * 10).round()),
                            child: Container(color: colorOf(i))),
                  ]),
                ),
              ),
              const SizedBox(height: 12),
              for (var i = 0; i < items.length; i++)
                Padding(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  child: Row(children: [
                    Container(
                        width: 10,
                        height: 10,
                        decoration: BoxDecoration(
                            color: colorOf(i), shape: BoxShape.circle)),
                    const SizedBox(width: 8),
                    Expanded(
                        child: Text('${items[i]['label'] ?? ''}',
                            maxLines: 1, overflow: TextOverflow.ellipsis)),
                    Text(
                        '${items[i]['formatted'] ?? ''}'.isNotEmpty
                            ? '${items[i]['formatted']}'
                            : formatWidgetValue(items[i]['value'], 'number'),
                        style: const TextStyle(fontWeight: FontWeight.w600)),
                    SizedBox(
                      width: 52,
                      child: Text(
                          '${formatNumber(pct(items[i]), decimals: 1)}%',
                          textAlign: TextAlign.right,
                          style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant)),
                    ),
                  ]),
                ),
            ]),
    );
  }
}

// ------------------------------------------------------------------ progress

/// Targets and capacity meters (widgets.ProgressWidget).
class DashboardProgress extends StatelessWidget {
  const DashboardProgress({super.key, required this.dto});

  final Map<String, dynamic> dto;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final items = (dto['items'] is List ? dto['items'] as List : const [])
        .whereType<Map>()
        .toList();
    return DashboardCard(
      key: Key('widget-${dto['id']}'),
      heading: '${dto['heading'] ?? ''}',
      description: '${dto['description'] ?? ''}',
      child: Column(mainAxisSize: MainAxisSize.min, children: [
        for (final it in items)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 6),
            child:
                Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Row(children: [
                Expanded(
                    child: Text('${it['label'] ?? ''}',
                        style: const TextStyle(fontWeight: FontWeight.w600))),
                Text(
                    '${formatWidgetValue(it['current'], 'number')} / ${formatWidgetValue(it['max'], 'number')}'
                    '${'${it['unit'] ?? ''}'.isEmpty ? '' : ' ${it['unit']}'}',
                    style: theme.textTheme.bodySmall),
              ]),
              const SizedBox(height: 6),
              ClipRRect(
                borderRadius: BorderRadius.circular(4),
                child: LinearProgressIndicator(
                  minHeight: 8,
                  value: ((it['percentage'] is num
                              ? (it['percentage'] as num).toDouble()
                              : 0) /
                          100)
                      .clamp(0.0, 1.0),
                  color: dashboardColor(context, '${it['color'] ?? 'primary'}'),
                  backgroundColor: theme.colorScheme.surfaceContainerHighest,
                ),
              ),
              if ('${it['subtitle'] ?? ''}'.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(top: 4),
                  child: Text('${it['subtitle']}',
                      style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant)),
                ),
            ]),
          ),
      ]),
    );
  }
}

// ------------------------------------------------------------------ quick actions

/// Shortcut buttons (widgets.QuickActionsWidget), each with an optional badge.
class DashboardQuickActions extends StatelessWidget {
  const DashboardQuickActions({super.key, required this.dto, this.onOpen});

  final Map<String, dynamic> dto;
  final DashboardOpen? onOpen;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final actions = (dto['actions'] is List ? dto['actions'] as List : const [])
        .whereType<Map>()
        .toList();
    return DashboardCard(
      key: Key('widget-${dto['id']}'),
      heading: '${dto['heading'] ?? ''}',
      description: '${dto['description'] ?? ''}',
      child: actions.isEmpty
          ? Text(tr('Nothing to show yet.'),
              style: TextStyle(color: theme.colorScheme.onSurfaceVariant))
          : Wrap(spacing: 10, runSpacing: 10, children: [
              for (var i = 0; i < actions.length; i++)
                _QuickAction(
                    key: Key('quick-action-$i'),
                    item: actions[i],
                    onOpen: onOpen),
            ]),
    );
  }
}

class _QuickAction extends StatelessWidget {
  const _QuickAction({super.key, required this.item, this.onOpen});
  final Map item;
  final DashboardOpen? onOpen;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = dashboardColor(context, '${item['color'] ?? 'primary'}');
    final badge = '${item['badge'] ?? ''}';
    final url = '${item['url'] ?? ''}';
    return OutlinedButton(
      onPressed: url.isEmpty || onOpen == null ? null : () => onOpen!(url),
      style: OutlinedButton.styleFrom(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12)),
      child: Row(mainAxisSize: MainAxisSize.min, children: [
        Icon(adminIcon('${item['icon'] ?? ''}', fallback: Icons.bolt_outlined),
            size: 18, color: color),
        const SizedBox(width: 8),
        Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('${item['label'] ?? ''}',
                  style: const TextStyle(fontWeight: FontWeight.w600)),
              if ('${item['description'] ?? ''}'.isNotEmpty)
                Text('${item['description']}',
                    style: theme.textTheme.bodySmall
                        ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
            ]),
        if (badge.isNotEmpty) ...[
          const SizedBox(width: 8),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 2),
            decoration: BoxDecoration(
                color: color, borderRadius: BorderRadius.circular(10)),
            child: Text(badge,
                style: theme.textTheme.labelSmall?.copyWith(
                    color: Colors.white, fontWeight: FontWeight.w700)),
          ),
        ],
      ]),
    );
  }
}

// ------------------------------------------------------------------ banner

/// A notice across the dashboard (widgets.BannerWidget); a dismissible one
/// can be closed for this visit.
class DashboardBanner extends StatefulWidget {
  const DashboardBanner({super.key, required this.dto, this.onOpen});

  final Map<String, dynamic> dto;
  final DashboardOpen? onOpen;

  @override
  State<DashboardBanner> createState() => _DashboardBannerState();
}

class _DashboardBannerState extends State<DashboardBanner> {
  bool _closed = false;

  @override
  Widget build(BuildContext context) {
    if (_closed) return const SizedBox.shrink();
    final theme = Theme.of(context);
    final dto = widget.dto;
    final color = dashboardColor(context, '${dto['color'] ?? 'info'}');
    final label = '${dto['action_label'] ?? ''}';
    final url = '${dto['action_url'] ?? ''}';
    return Card(
      key: Key('widget-${dto['id']}'),
      margin: EdgeInsets.zero,
      color: color.withValues(alpha: 0.10),
      shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(12),
          side: BorderSide(color: color.withValues(alpha: 0.4))),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(children: [
          Icon(adminIcon('${dto['icon'] ?? ''}', fallback: Icons.info_outline),
              color: color),
          const SizedBox(width: 12),
          Expanded(
            child:
                Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text('${dto['title'] ?? ''}',
                  style: const TextStyle(fontWeight: FontWeight.w700)),
              if ('${dto['subtitle'] ?? ''}'.isNotEmpty)
                Text('${dto['subtitle']}', style: theme.textTheme.bodySmall),
            ]),
          ),
          if (label.isNotEmpty && url.isNotEmpty && widget.onOpen != null)
            TextButton(
                onPressed: () => widget.onOpen!(url), child: Text(label)),
          if (dto['dismissible'] == true)
            IconButton(
              key: const Key('banner-dismiss'),
              tooltip: tr('Dismiss'),
              icon: const Icon(Icons.close, size: 18),
              onPressed: () => setState(() => _closed = true),
            ),
        ]),
      ),
    );
  }
}

// ------------------------------------------------------------------ grid

/// Lays dashboard widgets out on a grid three columns wide (two on tablets,
/// one on phones). A widget takes `column_span` columns (at most the grid's
/// width); widgets fill each row in order.
class DashboardGrid extends StatelessWidget {
  const DashboardGrid(
      {super.key, required this.widgets, required this.builder});

  final List<Map<String, dynamic>> widgets;
  final Widget Function(Map<String, dynamic> w) builder;

  static int columnsFor(double width) =>
      width >= 980 ? 3 : (width >= 640 ? 2 : 1);

  static int spanOf(Map<String, dynamic> w, int columns) {
    final s = w['column_span'];
    if (s is num) return s.toInt().clamp(1, columns);
    // without a span, a row of stats and a banner take the whole width
    return const {'stats_overview', 'banner'}.contains(w['type']) ||
            w['stats'] is List
        ? columns
        : 1;
  }

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(builder: (context, c) {
      final columns = columnsFor(c.maxWidth);
      final rows = <List<Map<String, dynamic>>>[];
      var used = columns;
      for (final w in widgets) {
        final span = spanOf(w, columns);
        if (used + span > columns) {
          rows.add([]);
          used = 0;
        }
        rows.last.add(w);
        used += span;
      }
      const gap = 16.0;
      return Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
        for (var r = 0; r < rows.length; r++) ...[
          if (r > 0) const SizedBox(height: gap),
          _row(rows[r], columns, gap),
        ],
      ]);
    });
  }

  Widget _row(List<Map<String, dynamic>> row, int columns, double gap) {
    if (row.length == 1 && spanOf(row.first, columns) == columns) {
      return builder(row.first);
    }
    final used = row.fold<int>(0, (a, w) => a + spanOf(w, columns));
    final children = <Widget>[];
    for (var i = 0; i < row.length; i++) {
      if (i > 0) children.add(SizedBox(width: gap));
      children
          .add(Expanded(flex: spanOf(row[i], columns), child: builder(row[i])));
    }
    if (used < columns) {
      children.add(SizedBox(width: gap));
      children.add(Spacer(flex: columns - used));
    }
    // (no IntrinsicHeight: cards hold Wraps and LayoutBuilders, which
    // cannot report intrinsic sizes)
    return Row(
        crossAxisAlignment: CrossAxisAlignment.start, children: children);
  }
}
