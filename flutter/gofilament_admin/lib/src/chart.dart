import 'dart:math' as math;

import 'package:flutter/material.dart';

import 'controller.dart';
import 'theme.dart';

/// A chart widget of the dashboard (widgets.ChartWidget in Go): line and bar
/// charts for every dataset, pie / doughnut (and polar area, radar, drawn as
/// doughnut) for the first one. Period filters reload the data from
/// `/api/widgets/<id>/data`.
class AdminChartCard extends StatefulWidget {
  const AdminChartCard({super.key, required this.admin, required this.dto});

  final AdminController admin;
  final Map<String, dynamic> dto;

  @override
  State<AdminChartCard> createState() => _AdminChartCardState();
}

class _AdminChartCardState extends State<AdminChartCard> {
  late Map<String, dynamic> _data = _asMap(widget.dto['data']);
  late String _filter = '${widget.dto['default_filter'] ?? ''}';
  bool _loading = false;

  static Map<String, dynamic> _asMap(dynamic v) =>
      v is Map ? Map<String, dynamic>.from(v) : const {};

  Future<void> _select(String filter) async {
    setState(() {
      _filter = filter;
      _loading = true;
    });
    try {
      final d = await widget.admin.api
          .get('/widgets/${widget.dto['id']}/data', {'filter': filter});
      if (mounted) setState(() => _data = _asMap(d));
    } catch (_) {
      // keep the previous data
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final filters = widget.dto['filters'] is Map
        ? Map<String, dynamic>.from(widget.dto['filters'] as Map)
        : const <String, dynamic>{};
    final keys = filters.keys.toList()..sort();
    final type = '${widget.dto['chart_type'] ?? 'line'}';
    final labels =
        (_data['labels'] is List ? _data['labels'] as List : const [])
            .map((e) => '$e')
            .toList();
    final datasets =
        (_data['datasets'] is List ? _data['datasets'] as List : const [])
            .whereType<Map>()
            .map((d) => _Dataset(
                  '${d['label'] ?? ''}',
                  (d['data'] is List ? d['data'] as List : const [])
                      .map((v) => v is num ? v.toDouble() : 0.0)
                      .toList(),
                  _color(d['borderColor']) ?? _color(d['backgroundColor']),
                ))
            .toList();

    return Card(
      key: Key('chart-${widget.dto['id']}'),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Expanded(
                child: Text('${widget.dto['heading'] ?? ''}',
                    style: theme.textTheme.titleMedium
                        ?.copyWith(fontWeight: FontWeight.w600)),
              ),
              if (_loading)
                const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2)),
            ]),
            if (keys.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Wrap(spacing: 6, children: [
                  for (final k in keys)
                    ChoiceChip(
                      key: Key('chart-filter-$k'),
                      label: Text('${filters[k]}'),
                      selected: _filter == k,
                      onSelected: (_) => _select(k),
                    ),
                ]),
              ),
            const SizedBox(height: 16),
            if (datasets.isEmpty || labels.isEmpty)
              Text('No data yet.',
                  style: TextStyle(color: theme.colorScheme.onSurfaceVariant))
            else
              SizedBox(
                height: 220,
                child: CustomPaint(
                  size: Size.infinite,
                  painter: _ChartPainter(
                    type: type,
                    labels: labels,
                    datasets: datasets,
                    palette: [
                      theme.colorScheme.primary,
                      theme.colorScheme.tertiary,
                      theme.colorScheme.secondary,
                      Colors.orange,
                      Colors.teal,
                      Colors.pink,
                    ],
                    textStyle: theme.textTheme.bodySmall ?? const TextStyle(),
                    grid: theme.dividerColor,
                  ),
                ),
              ),
            if (datasets.length > 1 || _isRound(type))
              Padding(
                padding: const EdgeInsets.only(top: 10),
                child: Wrap(spacing: 12, runSpacing: 4, children: [
                  for (var i = 0;
                      i < (_isRound(type) ? labels.length : datasets.length);
                      i++)
                    Row(mainAxisSize: MainAxisSize.min, children: [
                      Container(
                          width: 10,
                          height: 10,
                          color: _colorAt(theme, datasets, type, i)),
                      const SizedBox(width: 4),
                      Text(_isRound(type) ? labels[i] : datasets[i].label,
                          style: theme.textTheme.bodySmall),
                    ]),
                ]),
              ),
          ],
        ),
      ),
    );
  }

  Color _colorAt(ThemeData theme, List<_Dataset> ds, String type, int i) {
    final palette = [
      theme.colorScheme.primary,
      theme.colorScheme.tertiary,
      theme.colorScheme.secondary,
      Colors.orange,
      Colors.teal,
      Colors.pink,
    ];
    if (!_isRound(type) && ds[i].color != null) return ds[i].color!;
    return palette[i % palette.length];
  }
}

Color? _color(dynamic v) => v is String ? parseColor(v) : null;

bool _isRound(String type) =>
    const {'pie', 'doughnut', 'polarArea', 'radar'}.contains(type);

class _Dataset {
  _Dataset(this.label, this.values, this.color);
  final String label;
  final List<double> values;
  final Color? color;
}

class _ChartPainter extends CustomPainter {
  _ChartPainter({
    required this.type,
    required this.labels,
    required this.datasets,
    required this.palette,
    required this.textStyle,
    required this.grid,
  });

  final String type;
  final List<String> labels;
  final List<_Dataset> datasets;
  final List<Color> palette;
  final TextStyle textStyle;
  final Color grid;

  Color _color(int i) => datasets[i].color ?? palette[i % palette.length];

  @override
  void paint(Canvas canvas, Size size) {
    if (_isRound(type)) {
      _paintRound(canvas, size);
    } else {
      _paintAxes(canvas, size);
    }
  }

  void _paintRound(Canvas canvas, Size size) {
    final values = datasets.first.values;
    final total = values.fold<double>(0, (a, b) => a + (b > 0 ? b : 0));
    if (total <= 0) return;
    final radius = math.min(size.width, size.height) / 2;
    final rect =
        Rect.fromCircle(center: size.center(Offset.zero), radius: radius);
    var start = -math.pi / 2;
    final stroke = type == 'pie' ? null : radius * 0.35;
    for (var i = 0; i < values.length; i++) {
      if (values[i] <= 0) continue;
      final sweep = values[i] / total * 2 * math.pi;
      final paint = Paint()..color = palette[i % palette.length];
      if (stroke == null) {
        canvas.drawArc(rect, start, sweep, true, paint);
      } else {
        paint
          ..style = PaintingStyle.stroke
          ..strokeWidth = stroke;
        canvas.drawArc(rect.deflate(stroke / 2), start, sweep, false, paint);
      }
      start += sweep;
    }
  }

  void _paintAxes(Canvas canvas, Size size) {
    const left = 40.0, bottom = 22.0;
    final plot = Rect.fromLTRB(left, 4, size.width, size.height - bottom);
    var maxV = 0.0, minV = 0.0;
    for (final d in datasets) {
      for (final v in d.values) {
        maxV = math.max(maxV, v);
        minV = math.min(minV, v);
      }
    }
    if (maxV == minV) maxV = minV + 1;
    double y(double v) =>
        plot.bottom - (v - minV) / (maxV - minV) * plot.height;

    final gridPaint = Paint()
      ..color = grid
      ..strokeWidth = 1;
    for (var i = 0; i <= 4; i++) {
      final v = minV + (maxV - minV) * i / 4;
      final yy = y(v);
      canvas.drawLine(Offset(plot.left, yy), Offset(plot.right, yy), gridPaint);
      _text(canvas, _short(v), Offset(0, yy - 7), left - 6, TextAlign.right);
    }

    final n = labels.length;
    final step = plot.width / math.max(n, 1);
    final every = (n / 8).ceil().clamp(1, n);
    for (var i = 0; i < n; i += every) {
      _text(canvas, labels[i], Offset(plot.left + step * i, plot.bottom + 4),
          step * every, TextAlign.center);
    }

    if (type == 'bar') {
      final groupW = step * 0.8;
      final barW = groupW / datasets.length;
      for (var d = 0; d < datasets.length; d++) {
        final paint = Paint()..color = _color(d);
        for (var i = 0; i < math.min(n, datasets[d].values.length); i++) {
          final x = plot.left + step * i + step * 0.1 + barW * d;
          final v = datasets[d].values[i];
          canvas.drawRect(
              Rect.fromLTRB(x, math.min(y(v), y(0)), x + barW * 0.9,
                  math.max(y(v), y(0))),
              paint);
        }
      }
      return;
    }
    for (var d = 0; d < datasets.length; d++) {
      final paint = Paint()
        ..color = _color(d)
        ..strokeWidth = 2
        ..style = PaintingStyle.stroke;
      final path = Path();
      for (var i = 0; i < math.min(n, datasets[d].values.length); i++) {
        final p =
            Offset(plot.left + step * i + step / 2, y(datasets[d].values[i]));
        if (i == 0) {
          path.moveTo(p.dx, p.dy);
        } else {
          path.lineTo(p.dx, p.dy);
        }
        canvas.drawCircle(p, 2.5, Paint()..color = _color(d));
      }
      canvas.drawPath(path, paint);
    }
  }

  String _short(double v) {
    final a = v.abs();
    if (a >= 1e6) return '${(v / 1e6).toStringAsFixed(1)}M';
    if (a >= 1e3) return '${(v / 1e3).toStringAsFixed(1)}k';
    return v == v.roundToDouble() ? v.toStringAsFixed(0) : v.toStringAsFixed(1);
  }

  void _text(
      Canvas canvas, String s, Offset at, double width, TextAlign align) {
    final tp = TextPainter(
      text: TextSpan(text: s, style: textStyle.copyWith(fontSize: 10)),
      textAlign: align,
      textDirection: TextDirection.ltr,
      maxLines: 1,
      ellipsis: '…',
    )..layout(minWidth: width, maxWidth: width);
    tp.paint(canvas, at);
  }

  @override
  bool shouldRepaint(_ChartPainter old) =>
      old.type != type || old.labels != labels || old.datasets != datasets;
}
