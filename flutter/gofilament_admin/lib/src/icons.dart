import 'package:flutter/material.dart';

/// Maps the icon names used by GoFilament resources (Heroicons-style names)
/// to Material icons. Unknown names, and raw SVG markup, get [fallback].
IconData adminIcon(String? name,
    {IconData fallback = Icons.table_rows_outlined}) {
  if (name == null || name.isEmpty || name.contains('<')) return fallback;
  final key = name
      .toLowerCase()
      .replaceAll('heroicon-o-', '')
      .replaceAll('heroicon-s-', '')
      .replaceAll('_', '-');
  return _icons[key] ?? fallback;
}

const _icons = <String, IconData>{
  'home': Icons.space_dashboard_outlined,
  'dashboard': Icons.space_dashboard_outlined,
  'cube': Icons.inventory_2_outlined,
  'box': Icons.inventory_2_outlined,
  'archive': Icons.archive_outlined,
  'collection': Icons.category_outlined,
  'tag': Icons.sell_outlined,
  'scale': Icons.scale_outlined,
  'receipt-tax': Icons.receipt_long_outlined,
  'receipt': Icons.receipt_long_outlined,
  'users': Icons.groups_outlined,
  'user': Icons.person_outline,
  'user-group': Icons.groups_outlined,
  'truck': Icons.local_shipping_outlined,
  'office-building': Icons.store_mall_directory_outlined,
  'library': Icons.account_balance_outlined,
  'adjustments': Icons.tune,
  'cog': Icons.settings_outlined,
  'settings': Icons.settings_outlined,
  'shopping-cart': Icons.shopping_cart_outlined,
  'shopping-bag': Icons.shopping_bag_outlined,
  'cash': Icons.payments_outlined,
  'credit-card': Icons.credit_card,
  'chart-bar': Icons.bar_chart,
  'chart-pie': Icons.pie_chart_outline,
  'document-text': Icons.description_outlined,
  'document': Icons.insert_drive_file_outlined,
  'clipboard-list': Icons.fact_check_outlined,
  'switch-horizontal': Icons.swap_horiz,
  'refresh': Icons.refresh,
  'shield-check': Icons.verified_user_outlined,
  'key': Icons.key_outlined,
  'lock-closed': Icons.lock_outline,
  'globe': Icons.public,
  'globe-alt': Icons.public,
  'color-swatch': Icons.palette_outlined,
  'bell': Icons.notifications_none,
  'calendar': Icons.calendar_today_outlined,
  'clock': Icons.schedule,
  'star': Icons.star_border,
  'trending-up': Icons.trending_up,
  'trending-down': Icons.trending_down,
  'check-circle': Icons.check_circle_outline,
  'x-circle': Icons.cancel_outlined,
  'exclamation': Icons.warning_amber_outlined,
  'information-circle': Icons.info_outline,
  'pencil': Icons.edit_outlined,
  'trash': Icons.delete_outline,
  'eye': Icons.visibility_outlined,
  'download': Icons.download_outlined,
  'upload': Icons.upload_outlined,
  'printer': Icons.print_outlined,
  'qrcode': Icons.qr_code,
  'desktop-computer': Icons.point_of_sale,
  'calculator': Icons.calculate_outlined,
  'sparkles': Icons.auto_awesome_outlined,
  'chat': Icons.chat_bubble_outline,
  'support': Icons.support_agent,
  'identification': Icons.badge_outlined,
  'briefcase': Icons.work_outline,
  'currency-dollar': Icons.attach_money,
  'template': Icons.dashboard_customize_outlined,
  'view-grid': Icons.grid_view,
  'database': Icons.storage_outlined,
};
