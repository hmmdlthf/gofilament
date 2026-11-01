import 'package:flutter/material.dart';

class ResponsiveLayout extends StatelessWidget {
  final Widget mobile;
  final Widget? tablet;
  final Widget desktop;

  const ResponsiveLayout({
    super.key,
    required this.mobile,
    this.tablet,
    required this.desktop,
  });

  static bool isMobile(BuildContext context) =>
      MediaQuery.of(context).size.width <= 900;

  static bool isTablet(BuildContext context) =>
      MediaQuery.of(context).size.width > 600 &&
      MediaQuery.of(context).size.width <= 900;

  static bool isDesktop(BuildContext context) =>
      MediaQuery.of(context).size.width > 900;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth > 900) {
          return desktop;
        } else {
          return mobile;
        }
      },
    );
  }
}
