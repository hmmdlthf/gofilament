import 'package:flutter/material.dart';

/// The three layouts the admin is designed for. Every screen picks its
/// layout from these instead of its own width checks, so a phone, a tablet
/// and a desktop each get one consistent design.
enum AdminFormFactor {
  /// Narrower than [AdminBreakpoints.tablet]: one column, full-screen pages.
  phone,

  /// From [AdminBreakpoints.tablet] up to [AdminBreakpoints.desktop].
  tablet,

  /// [AdminBreakpoints.desktop] and wider: persistent sidebar.
  desktop;

  /// The layout for a window [width] logical pixels wide.
  static AdminFormFactor forWidth(double width) {
    if (width < AdminBreakpoints.tablet) return AdminFormFactor.phone;
    if (width < AdminBreakpoints.desktop) return AdminFormFactor.tablet;
    return AdminFormFactor.desktop;
  }

  /// The layout of the current window.
  static AdminFormFactor of(BuildContext context) =>
      forWidth(MediaQuery.sizeOf(context).width);
}

/// Window widths where the layout changes.
abstract final class AdminBreakpoints {
  static const double tablet = 600;
  static const double desktop = 1024;
}

/// Sizes that change with the layout: bar heights, control heights, icon
/// sizes and spacing. Read with [AdminSizes.of]; the theme carries the set
/// for the current window.
@immutable
class AdminSizes extends ThemeExtension<AdminSizes> {
  const AdminSizes({
    required this.formFactor,
    required this.topBarHeight,
    required this.controlHeight,
    required this.searchHeight,
    required this.iconButtonSize,
    required this.iconSize,
    required this.pagePadding,
    required this.gap,
    required this.radius,
  });

  factory AdminSizes.forFormFactor(AdminFormFactor f) => switch (f) {
        AdminFormFactor.phone => const AdminSizes(
            formFactor: AdminFormFactor.phone,
            topBarHeight: 56,
            controlHeight: 42,
            searchHeight: 40,
            iconButtonSize: 40,
            iconSize: 21,
            pagePadding: 12,
            gap: 8,
            radius: 12,
          ),
        AdminFormFactor.tablet => const AdminSizes(
            formFactor: AdminFormFactor.tablet,
            topBarHeight: 60,
            controlHeight: 42,
            searchHeight: 40,
            iconButtonSize: 40,
            iconSize: 21,
            pagePadding: 16,
            gap: 10,
            radius: 10,
          ),
        AdminFormFactor.desktop => const AdminSizes(
            formFactor: AdminFormFactor.desktop,
            topBarHeight: 60,
            controlHeight: 38,
            searchHeight: 38,
            iconButtonSize: 36,
            iconSize: 20,
            pagePadding: 20,
            gap: 12,
            radius: 10,
          ),
      };

  final AdminFormFactor formFactor;

  /// Height of the shell's top bar.
  final double topBarHeight;

  /// Height of buttons and text fields.
  final double controlHeight;

  /// Height of toolbar search fields.
  final double searchHeight;

  /// Width and height of icon buttons.
  final double iconButtonSize;

  /// Size of icons in icon buttons.
  final double iconSize;

  /// Padding around page content.
  final double pagePadding;

  /// Spacing between neighbouring controls.
  final double gap;

  /// Corner radius of controls.
  final double radius;

  bool get isPhone => formFactor == AdminFormFactor.phone;
  bool get isTablet => formFactor == AdminFormFactor.tablet;
  bool get isDesktop => formFactor == AdminFormFactor.desktop;

  /// The sizes in the theme, or the set for the current window when the
  /// theme has none (a widget used outside the admin theme).
  static AdminSizes of(BuildContext context) =>
      Theme.of(context).extension<AdminSizes>() ??
      AdminSizes.forFormFactor(AdminFormFactor.of(context));

  @override
  AdminSizes copyWith({AdminFormFactor? formFactor}) =>
      formFactor == null ? this : AdminSizes.forFormFactor(formFactor);

  @override
  AdminSizes lerp(ThemeExtension<AdminSizes>? other, double t) =>
      t < 0.5 ? this : (other as AdminSizes? ?? this);
}

/// Shows [child] as a full page on a phone and as a dialog of at most
/// [maxWidth] × [maxHeight] elsewhere. [child] is usually a [Scaffold] with
/// an app bar; in the dialog its leading button closes the dialog.
Future<T?> showAdaptivePage<T>(
  BuildContext context, {
  required WidgetBuilder builder,
  double maxWidth = 560,
  double maxHeight = 760,
}) {
  if (AdminFormFactor.of(context) == AdminFormFactor.phone) {
    return Navigator.of(context)
        .push<T>(MaterialPageRoute<T>(builder: builder));
  }
  return showDialog<T>(
    context: context,
    builder: (ctx) {
      final size = MediaQuery.sizeOf(ctx);
      return Dialog(
        insetPadding: const EdgeInsets.all(24),
        clipBehavior: Clip.antiAlias,
        child: ConstrainedBox(
          constraints: BoxConstraints(
              maxWidth: maxWidth,
              maxHeight: (size.height * 0.85).clamp(0, maxHeight)),
          child: builder(ctx),
        ),
      );
    },
  );
}

/// Shows [builder] as a bottom sheet on a phone and as a dialog elsewhere.
/// The builder returns an [AdaptiveDialog], which lays itself out for
/// either.
Future<T?> showAdaptiveSheet<T>(
  BuildContext context, {
  required WidgetBuilder builder,
}) {
  if (AdminFormFactor.of(context) == AdminFormFactor.phone) {
    return showModalBottomSheet<T>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      showDragHandle: true,
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(bottom: MediaQuery.viewInsetsOf(ctx).bottom),
        child: _SheetScope(child: Builder(builder: builder)),
      ),
    );
  }
  return showDialog<T>(context: context, builder: builder);
}

class _SheetScope extends InheritedWidget {
  const _SheetScope({required super.child});

  static bool of(BuildContext context) =>
      context.dependOnInheritedWidgetOfExactType<_SheetScope>() != null;

  @override
  bool updateShouldNotify(_SheetScope oldWidget) => false;
}

/// An [AlertDialog] that becomes a bottom sheet body when shown with
/// [showAdaptiveSheet] on a phone: title on top, scrolling content, and the
/// actions along the bottom.
class AdaptiveDialog extends StatelessWidget {
  const AdaptiveDialog(
      {super.key, this.title, required this.content, this.actions});

  final Widget? title;
  final Widget content;
  final List<Widget>? actions;

  @override
  Widget build(BuildContext context) {
    if (!_SheetScope.of(context)) {
      return AlertDialog(title: title, content: content, actions: actions);
    }
    final theme = Theme.of(context);
    final buttons = actions ?? const <Widget>[];
    return SafeArea(
      top: false,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (title != null)
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 12),
              child: DefaultTextStyle.merge(
                  style: theme.textTheme.titleLarge
                      ?.copyWith(fontWeight: FontWeight.w600, fontSize: 18),
                  child: title!),
            ),
          Flexible(
            child: SingleChildScrollView(
              padding: const EdgeInsets.symmetric(horizontal: 20),
              child: content,
            ),
          ),
          if (buttons.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
              child: Wrap(
                alignment: WrapAlignment.end,
                spacing: 8,
                runSpacing: 8,
                children: buttons,
              ),
            ),
        ],
      ),
    );
  }
}
