import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:flutter/material.dart';

import 'api.dart';
import 'controller.dart';
import 'i18n.dart';
import 'icons.dart';
import 'links.dart';
import 'theme.dart';

/// Opens a task: a panel URL (`<panel>/<slug>[/new]`, `<panel>/<page>`) or
/// an external link.
typedef WorkflowOpen = void Function(String url);

/// A workflow map from `<panel>/api/maps/<id>` (filament.WorkflowMap):
/// blocks of tasks joined by arrows. Every task opens its screen.
class WorkflowMapPage extends StatefulWidget {
  const WorkflowMapPage(
      {super.key,
      required this.admin,
      required this.id,
      required this.onOpen,
      this.card = false});

  final AdminController admin;
  final String id;
  final WorkflowOpen onOpen;

  /// Drawn inside the dashboard (no page padding, a card around it).
  final bool card;

  @override
  State<WorkflowMapPage> createState() => _WorkflowMapPageState();
}

class _WorkflowMapPageState extends State<WorkflowMapPage> {
  WorkflowMapData? _map;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final m = await widget.admin.api.workflowMap(widget.id);
      if (mounted) {
        setState(() {
          _map = WorkflowMapData.fromJson(m);
          _error = null;
        });
      }
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    } catch (_) {
      if (mounted) setState(() => _error = tr('Could not load the map.'));
    }
  }

  void _open(String url) {
    if (url.startsWith('http://') || url.startsWith('https://')) {
      openLink(url);
    } else {
      widget.onOpen(url);
    }
  }

  @override
  Widget build(BuildContext context) {
    final map = _map;
    final Widget body;
    if (map == null) {
      body = Padding(
        padding: const EdgeInsets.all(32),
        child: Center(
            child: _error == null
                ? const CircularProgressIndicator()
                : Text(_error!, key: const Key('workflow-map-error'))),
      );
    } else {
      body = WorkflowMapView(map: map, onOpen: _open);
    }
    if (widget.card) {
      return Card(
        key: Key('workflow-map-card-${widget.id}'),
        margin: EdgeInsets.zero,
        clipBehavior: Clip.antiAlias,
        child: Padding(padding: const EdgeInsets.all(12), child: body),
      );
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(padding: const EdgeInsets.all(20), children: [body]),
    );
  }
}

// ------------------------------------------------------------------ data

/// A map as the server sends it.
class WorkflowMapData {
  WorkflowMapData(
      {required this.id,
      required this.label,
      required this.blocks,
      required this.tasks,
      required this.links});

  factory WorkflowMapData.fromJson(Map<String, dynamic> j) {
    List<Map<String, dynamic>> list(String k) => (j[k] as List? ?? const [])
        .whereType<Map>()
        .map((m) => Map<String, dynamic>.from(m))
        .toList();
    return WorkflowMapData(
      id: '${j['id'] ?? ''}',
      label: '${j['label'] ?? ''}',
      blocks: list('blocks').map(WorkflowBlock.fromJson).toList(),
      tasks: list('tasks').map(WorkflowTask.fromJson).toList(),
      links: list('links').map(WorkflowLink.fromJson).toList(),
    );
  }

  final String id, label;
  final List<WorkflowBlock> blocks;
  final List<WorkflowTask> tasks;
  final List<WorkflowLink> links;

  WorkflowTask? task(String id) {
    for (final t in tasks) {
      if (t.id == id) return t;
    }
    return null;
  }

  /// The tasks [id] leads to (solid arrows), in declaration order.
  List<WorkflowTask> next(String id) => [
        for (final l in links)
          if (l.from == id && l.kind == 'leads' && task(l.to) != null)
            task(l.to)!
      ];
}

int _int(dynamic v, [int fallback = 0]) =>
    v is num ? v.toInt() : (int.tryParse('$v') ?? fallback);

class WorkflowBlock {
  WorkflowBlock.fromJson(Map<String, dynamic> j)
      : id = '${j['id'] ?? ''}',
        label = '${j['label'] ?? ''}',
        icon = '${j['icon'] ?? ''}',
        color = '${j['color'] ?? ''}',
        col = _int(j['col']),
        row = _int(j['row']),
        cols = math.max(1, _int(j['cols'], 1)),
        rows = math.max(1, _int(j['rows'], 1)),
        taskColumns = _int(j['task_columns']);

  final String id, label, icon, color;
  final int col, row, cols, rows, taskColumns;
}

class WorkflowTask {
  WorkflowTask.fromJson(Map<String, dynamic> j)
      : id = '${j['id'] ?? ''}',
        block = '${j['block'] ?? ''}',
        label = '${j['label'] ?? ''}',
        icon = '${j['icon'] ?? ''}',
        badge = '${j['badge'] ?? ''}',
        col = j['col'] == null ? null : _int(j['col']),
        row = j['row'] == null ? null : _int(j['row']),
        url = '${(j['open'] is Map ? j['open']['url'] : null) ?? ''}';

  final String id, block, label, icon, badge, url;
  final int? col, row;
}

class WorkflowLink {
  WorkflowLink.fromJson(Map<String, dynamic> j)
      : from = '${j['from'] ?? ''}',
        to = '${j['to'] ?? ''}',
        kind = '${j['kind'] ?? 'leads'}';

  final String from, to, kind;
}

// ------------------------------------------------------------------ view

/// Draws a map: blocks and arrows when there is room, a list of blocks with
/// "Then" chips on phones.
class WorkflowMapView extends StatelessWidget {
  const WorkflowMapView({super.key, required this.map, required this.onOpen});

  final WorkflowMapData map;
  final WorkflowOpen onOpen;

  /// Narrower than this, arrows across blocks can't be read: list instead.
  static const listBelow = 720.0;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(builder: (context, c) {
      if (c.maxWidth < listBelow) {
        return _MapList(map: map, onOpen: onOpen);
      }
      return _MapCanvas(map: map, onOpen: onOpen, width: c.maxWidth);
    });
  }
}

/// The phone layout: each block a card, each task a row with its next
/// steps as chips.
class _MapList extends StatelessWidget {
  const _MapList({required this.map, required this.onOpen});

  final WorkflowMapData map;
  final WorkflowOpen onOpen;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final blocks = [...map.blocks]
      ..sort((a, b) => a.row != b.row ? a.row - b.row : a.col - b.col);
    return Column(
      key: const Key('workflow-map-list'),
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        for (final b in blocks)
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: Card(
              key: Key('workflow-block-${b.id}'),
              margin: EdgeInsets.zero,
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    ListTile(
                      dense: true,
                      leading: Icon(adminIcon(b.icon,
                          fallback: Icons.dashboard_customize_outlined)),
                      title: Text(b.label,
                          style: theme.textTheme.titleMedium
                              ?.copyWith(fontWeight: FontWeight.w700)),
                    ),
                    for (final t in map.tasks.where((t) => t.block == b.id))
                      _listTask(context, t),
                  ],
                ),
              ),
            ),
          ),
      ],
    );
  }

  Widget _listTask(BuildContext context, WorkflowTask t) {
    final next = map.next(t.id);
    return ListTile(
      key: Key('workflow-task-${t.id}'),
      leading: Icon(adminIcon(t.icon, fallback: Icons.task_alt_outlined)),
      title: Text(t.label),
      trailing: t.badge.isEmpty ? null : _Badge(t.badge),
      subtitle: next.isEmpty
          ? null
          : Wrap(
              spacing: 6,
              runSpacing: 4,
              crossAxisAlignment: WrapCrossAlignment.center,
              children: [
                Text(tr('Then')),
                for (final n in next)
                  ActionChip(
                    key: Key('workflow-next-${t.id}-${n.id}'),
                    visualDensity: VisualDensity.compact,
                    label: Text(n.label),
                    onPressed: () => onOpen(n.url),
                  ),
              ],
            ),
      onTap: () => onOpen(t.url),
    );
  }
}

// Sizes of the drawn map.
const _gap = 16.0;
const _pad = 12.0;
const _header = 40.0;
const _tileW = 112.0;
const _tileH = 96.0;
const _tileGap = 12.0;
const _minColW = 220.0;
const _maxStep = 260.0; // widest spacing between tile columns

/// Where every block and task is drawn.
class _Layout {
  _Layout(this.size, this.blocks, this.tasks);

  final Size size;
  final Map<String, Rect> blocks;
  final Map<String, Rect> tasks;

  static _Layout of(WorkflowMapData map, double width, bool rtl) {
    var cols = 1, rows = 1;
    for (final b in map.blocks) {
      cols = math.max(cols, b.col + b.cols);
      rows = math.max(rows, b.row + b.rows);
    }
    final total = math.max(width, cols * _minColW + (cols - 1) * _gap);
    final colW = (total - (cols - 1) * _gap) / cols;
    double blockW(WorkflowBlock b) => b.cols * colW + (b.cols - 1) * _gap;

    // tasks of each block in cells
    final cells = <String, Map<String, (int, int)>>{};
    final need = <String, double>{};
    for (final b in map.blocks) {
      final inner = blockW(b) - 2 * _pad;
      var n = b.taskColumns > 0
          ? b.taskColumns
          : math.max(1, ((inner + _tileGap) / (_tileW + _tileGap)).floor());
      final mine = map.tasks.where((t) => t.block == b.id).toList();
      for (final t in mine) {
        if (t.col != null) n = math.max(n, t.col! + 1);
      }
      final taken = <(int, int)>{};
      final placed = <String, (int, int)>{};
      for (final t in mine) {
        if (t.col != null && t.row != null) {
          placed[t.id] = (t.col!, t.row!);
          taken.add((t.col!, t.row!));
        }
      }
      var cursor = 0;
      for (final t in mine) {
        if (placed.containsKey(t.id)) continue;
        while (taken.contains((cursor % n, cursor ~/ n))) {
          cursor++;
        }
        placed[t.id] = (cursor % n, cursor ~/ n);
        taken.add(placed[t.id]!);
      }
      var lines = 0;
      for (final c in placed.values) {
        lines = math.max(lines, c.$2 + 1);
      }
      cells[b.id] = placed;
      need[b.id] = _header +
          2 * _pad +
          math.max(1, lines) * _tileH +
          (math.max(1, lines) - 1) * _tileGap;
    }

    // grid row heights: one-row blocks first, then make taller blocks fit
    final rowH = List<double>.filled(rows, 0);
    for (final b in map.blocks.where((b) => b.rows == 1)) {
      rowH[b.row] = math.max(rowH[b.row], need[b.id]!);
    }
    for (final b in map.blocks.where((b) => b.rows > 1)) {
      var have = (b.rows - 1) * _gap;
      for (var r = b.row; r < b.row + b.rows; r++) {
        have += rowH[r];
      }
      if (have < need[b.id]!) {
        rowH[b.row + b.rows - 1] += need[b.id]! - have;
      }
    }
    for (var r = 0; r < rows; r++) {
      if (rowH[r] == 0) rowH[r] = _header + 2 * _pad;
    }
    double rowY(int r) {
      var y = 0.0;
      for (var i = 0; i < r; i++) {
        y += rowH[i] + _gap;
      }
      return y;
    }

    Rect mirror(Rect r) =>
        rtl ? Rect.fromLTWH(total - r.right, r.top, r.width, r.height) : r;

    final blockRects = <String, Rect>{};
    final taskRects = <String, Rect>{};
    for (final b in map.blocks) {
      final x = b.col * (colW + _gap);
      final y = rowY(b.row);
      var h = (b.rows - 1) * _gap;
      for (var r = b.row; r < b.row + b.rows; r++) {
        h += rowH[r];
      }
      final rect = Rect.fromLTWH(x, y, blockW(b), h);
      blockRects[b.id] = mirror(rect);
      // spread the tiles over the block's width, by the columns in use
      var used = 1;
      for (final c in cells[b.id]!.values) {
        used = math.max(used, c.$1 + 1);
      }
      final inner = rect.width - 2 * _pad;
      final step = used > 1
          ? math.min(_maxStep,
              math.max(_tileW + _tileGap, (inner - _tileW) / (used - 1)))
          : 0.0;
      final left = used > 1
          ? rect.left +
              _pad +
              math.max(0.0, (inner - _tileW - step * (used - 1)) / 2)
          : rect.center.dx - _tileW / 2;
      cells[b.id]!.forEach((id, c) {
        taskRects[id] = mirror(Rect.fromLTWH(
            left + c.$1 * step,
            rect.top + _header + _pad + c.$2 * (_tileH + _tileGap),
            _tileW,
            _tileH));
      });
    }
    final height = rowY(rows - 1) + rowH[rows - 1];
    return _Layout(Size(total, height), blockRects, taskRects);
  }
}

/// The drawn map: blocks, arrows behind the tiles, tiles on top.
class _MapCanvas extends StatefulWidget {
  const _MapCanvas(
      {required this.map, required this.onOpen, required this.width});

  final WorkflowMapData map;
  final WorkflowOpen onOpen;
  final double width;

  @override
  State<_MapCanvas> createState() => _MapCanvasState();
}

class _MapCanvasState extends State<_MapCanvas> {
  String? _hover;

  Set<String> _near(String id) => {
        id,
        for (final l in widget.map.links)
          if (l.from == id) l.to else if (l.to == id) l.from,
      };

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final palette = AdminPalette.of(context);
    final rtl = Directionality.of(context) == TextDirection.rtl;
    final layout = _Layout.of(widget.map, widget.width, rtl);
    final near = _hover == null ? null : _near(_hover!);

    final canvas = SizedBox(
      key: const Key('workflow-map-canvas'),
      width: layout.size.width,
      height: layout.size.height,
      child: Stack(children: [
        for (final b in widget.map.blocks)
          if (layout.blocks[b.id] != null)
            Positioned.fromRect(
              rect: layout.blocks[b.id]!,
              child: _BlockPanel(
                block: b,
                tint: b.color.isEmpty ? null : palette.color(b.color, 500),
              ),
            ),
        Positioned.fill(
          child: IgnorePointer(
            child: CustomPaint(
              painter: _LinksPainter(
                links: widget.map.links,
                rects: layout.tasks,
                hover: _hover,
                line: theme.colorScheme.onSurfaceVariant,
                highlight: theme.colorScheme.primary,
              ),
            ),
          ),
        ),
        for (final t in widget.map.tasks)
          if (layout.tasks[t.id] != null)
            Positioned.fromRect(
              rect: layout.tasks[t.id]!,
              child: AnimatedOpacity(
                duration: const Duration(milliseconds: 120),
                opacity: near == null || near.contains(t.id) ? 1 : 0.4,
                child: _TaskTile(
                  task: t,
                  next: widget.map.next(t.id),
                  active: _hover == t.id,
                  onHover: (on) => setState(() =>
                      _hover = on ? t.id : (_hover == t.id ? null : _hover)),
                  onOpen: () => widget.onOpen(t.url),
                ),
              ),
            ),
      ]),
    );
    if (layout.size.width <= widget.width + 0.5) return canvas;
    return Scrollbar(
      child: SingleChildScrollView(
          scrollDirection: Axis.horizontal, child: canvas),
    );
  }
}

class _BlockPanel extends StatelessWidget {
  const _BlockPanel({required this.block, this.tint});

  final WorkflowBlock block;
  final Color? tint;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final accent = tint ?? scheme.primary;
    return DecoratedBox(
      key: Key('workflow-block-${block.id}'),
      decoration: BoxDecoration(
        color: Color.alphaBlend(
            accent.withValues(alpha: 0.05), scheme.surfaceContainerLow),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: scheme.outlineVariant),
      ),
      child: Align(
        alignment: AlignmentDirectional.topStart,
        child: SizedBox(
          height: _header,
          child: Padding(
            padding: const EdgeInsetsDirectional.only(start: 14, end: 14),
            child: Row(children: [
              Icon(
                  adminIcon(block.icon,
                      fallback: Icons.dashboard_customize_outlined),
                  size: 18,
                  color: accent),
              const SizedBox(width: 8),
              Flexible(
                child: Text(block.label,
                    overflow: TextOverflow.ellipsis,
                    style: theme.textTheme.titleSmall
                        ?.copyWith(fontWeight: FontWeight.w700)),
              ),
            ]),
          ),
        ),
      ),
    );
  }
}

class _TaskTile extends StatelessWidget {
  const _TaskTile(
      {required this.task,
      required this.next,
      required this.active,
      required this.onHover,
      required this.onOpen});

  final WorkflowTask task;
  final List<WorkflowTask> next;
  final bool active;
  final ValueChanged<bool> onHover;
  final VoidCallback onOpen;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final then = next.isEmpty
        ? ''
        : '. ${tr('Then')}: ${next.map((n) => n.label).join(', ')}';
    return Semantics(
      button: true,
      label: '${task.label}$then',
      excludeSemantics: true,
      child: MouseRegion(
        onEnter: (_) => onHover(true),
        onExit: (_) => onHover(false),
        child: Material(
          color: scheme.surface,
          elevation: active ? 3 : 0.5,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(12),
            side: BorderSide(
                color: active ? scheme.primary : scheme.outlineVariant),
          ),
          child: InkWell(
            key: Key('workflow-task-${task.id}'),
            borderRadius: BorderRadius.circular(12),
            onTap: onOpen,
            onFocusChange: onHover,
            child: Stack(children: [
              Padding(
                padding: const EdgeInsets.all(8),
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    CircleAvatar(
                      radius: 18,
                      backgroundColor: scheme.primaryContainer,
                      child: Icon(
                          adminIcon(task.icon,
                              fallback: Icons.task_alt_outlined),
                          size: 20,
                          color: scheme.onPrimaryContainer),
                    ),
                    const SizedBox(height: 6),
                    Text(task.label,
                        textAlign: TextAlign.center,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.labelMedium
                            ?.copyWith(fontWeight: FontWeight.w600)),
                  ],
                ),
              ),
              if (task.badge.isNotEmpty)
                PositionedDirectional(
                    top: 4, end: 4, child: _Badge(task.badge)),
            ]),
          ),
        ),
      ),
    );
  }
}

class _Badge extends StatelessWidget {
  const _Badge(this.text);

  final String text;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
          color: scheme.error, borderRadius: BorderRadius.circular(10)),
      child: Text(text,
          style: TextStyle(
              color: scheme.onError,
              fontSize: 11,
              fontWeight: FontWeight.w700)),
    );
  }
}

/// Draws the arrows between task tiles: elbowed lines from side to side
/// (or top to bottom), an arrowhead at the end, dashed for "feeds".
class _LinksPainter extends CustomPainter {
  _LinksPainter(
      {required this.links,
      required this.rects,
      required this.hover,
      required this.line,
      required this.highlight});

  final List<WorkflowLink> links;
  final Map<String, Rect> rects;
  final String? hover;
  final Color line, highlight;

  @override
  void paint(Canvas canvas, Size size) {
    for (final l in links) {
      final a = rects[l.from], b = rects[l.to];
      if (a == null || b == null) continue;
      final on = hover != null && (l.from == hover || l.to == hover);
      final paint = Paint()
        ..color =
            on ? highlight : line.withValues(alpha: hover == null ? 0.55 : 0.15)
        ..strokeWidth = on ? 2.4 : 1.6
        ..style = PaintingStyle.stroke
        ..strokeCap = StrokeCap.round;
      final points = route(a, b);
      final path = Path()..moveTo(points.first.dx, points.first.dy);
      for (final p in points.skip(1)) {
        path.lineTo(p.dx, p.dy);
      }
      if (l.kind == 'feeds') {
        _dashed(canvas, path, paint);
      } else {
        canvas.drawPath(path, paint);
      }
      _arrowHead(canvas, points[points.length - 2], points.last, paint.color);
    }
  }

  /// The corner points of the line from tile [a] to tile [b].
  static List<Offset> route(Rect a, Rect b) {
    final dx = b.center.dx - a.center.dx, dy = b.center.dy - a.center.dy;
    if (dx.abs() >= dy.abs()) {
      final start = Offset(dx > 0 ? a.right : a.left, a.center.dy);
      final end = Offset(dx > 0 ? b.left : b.right, b.center.dy);
      if ((start.dy - end.dy).abs() < 1) return [start, end];
      final midX = (start.dx + end.dx) / 2;
      return [start, Offset(midX, start.dy), Offset(midX, end.dy), end];
    }
    final start = Offset(a.center.dx, dy > 0 ? a.bottom : a.top);
    final end = Offset(b.center.dx, dy > 0 ? b.top : b.bottom);
    if ((start.dx - end.dx).abs() < 1) return [start, end];
    final midY = (start.dy + end.dy) / 2;
    return [start, Offset(start.dx, midY), Offset(end.dx, midY), end];
  }

  static void _dashed(Canvas canvas, Path path, Paint paint) {
    for (final ui.PathMetric m in path.computeMetrics()) {
      var d = 0.0;
      while (d < m.length) {
        canvas.drawPath(m.extractPath(d, math.min(d + 6, m.length)), paint);
        d += 10;
      }
    }
  }

  static void _arrowHead(Canvas canvas, Offset from, Offset to, Color color) {
    final angle = math.atan2(to.dy - from.dy, to.dx - from.dx);
    const len = 9.0, spread = 0.45;
    final head = Path()
      ..moveTo(to.dx, to.dy)
      ..lineTo(to.dx - len * math.cos(angle - spread),
          to.dy - len * math.sin(angle - spread))
      ..lineTo(to.dx - len * math.cos(angle + spread),
          to.dy - len * math.sin(angle + spread))
      ..close();
    canvas.drawPath(head, Paint()..color = color);
  }

  @override
  bool shouldRepaint(_LinksPainter old) =>
      old.hover != hover ||
      old.links != links ||
      old.rects != rects ||
      old.line != line ||
      old.highlight != highlight;
}
