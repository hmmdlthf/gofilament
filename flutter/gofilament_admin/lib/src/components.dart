import 'package:flutter/material.dart';

import 'controller.dart';

Map<String, dynamic> _propsOf(dynamic v) =>
    v is Map ? Map<String, dynamic>.from(v) : const {};

/// What a custom field component (`forms.Custom(name, component)` in Go)
/// receives. Call [onChanged] with the new JSON value; the server validates
/// it (Required, Rule) when the form is saved.
class AdminFieldContext {
  const AdminFieldContext({
    required this.admin,
    required this.node,
    required this.value,
    required this.onChanged,
    required this.enabled,
    required this.required,
    this.errorText,
  });

  final AdminController admin;

  /// The field as the form endpoint describes it (name, label, props, ...).
  final Map<String, dynamic> node;
  final dynamic value;
  final ValueChanged<dynamic> onChanged;
  final bool enabled;
  final bool required;

  /// The validation message to show, if any.
  final String? errorText;

  String get name => '${node['name']}';
  String get label => '${node['label'] ?? node['name']}';
  String? get helperText => node['helper_text'] as String?;

  /// The props set in Go (`.Props(...)`) as JSON.
  Map<String, dynamic> get props => _propsOf(node['props']);
}

/// What a custom cell component (`tables.CustomColumn(name, component)`)
/// receives.
class AdminCellContext {
  const AdminCellContext({
    required this.admin,
    required this.column,
    required this.row,
    required this.value,
  });

  final AdminController? admin;
  final Map<String, dynamic> column;

  /// The whole record of the row.
  final Map<String, dynamic> row;
  final dynamic value;

  Map<String, dynamic> get props => _propsOf(column['props']);
}

/// What a custom dashboard widget component (`filament.FlutterWidget(id,
/// component)`) receives: the data the server computed for this user.
class AdminWidgetContext {
  const AdminWidgetContext({required this.admin, required this.widget});

  final AdminController admin;

  /// The widget as `/api/widgets` describes it.
  final Map<String, dynamic> widget;

  String get id => '${widget['id']}';
  String get heading => '${widget['heading'] ?? ''}';
  dynamic get data => widget['data'];
  Map<String, dynamic> get props => _propsOf(widget['props']);
}

typedef AdminFieldBuilder = Widget Function(
    BuildContext context, AdminFieldContext field);
typedef AdminCellBuilder = Widget Function(
    BuildContext context, AdminCellContext cell);
typedef AdminWidgetBuilder = Widget Function(
    BuildContext context, AdminWidgetContext widget);

/// The custom components of an app, by the component name used in Go. The
/// generated `panelComponents` (registry.g.dart) builds this from the files
/// in lib/components.
class AdminComponents {
  const AdminComponents({
    this.fields = const {},
    this.cells = const {},
    this.widgets = const {},
  });

  final Map<String, AdminFieldBuilder> fields;
  final Map<String, AdminCellBuilder> cells;
  final Map<String, AdminWidgetBuilder> widgets;
}
