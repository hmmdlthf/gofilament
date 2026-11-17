import 'package:flutter/material.dart';

import 'api.dart';
import 'controller.dart';

/// The topbar tenant menu: the current tenant, and the others to switch to.
class TenantSwitcher extends StatefulWidget {
  const TenantSwitcher({super.key, required this.admin});
  final AdminController admin;

  @override
  State<TenantSwitcher> createState() => _TenantSwitcherState();
}

class _TenantSwitcherState extends State<TenantSwitcher> {
  List<Map<String, dynamic>> _tenants = const [];

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final data = await widget.admin.api.tenants();
      final t = data['tenants'];
      if (mounted) {
        setState(() => _tenants = t is List
            ? [
                for (final x in t)
                  if (x is Map) Map<String, dynamic>.from(x)
              ]
            : const []);
      }
    } catch (_) {}
  }

  Future<void> _switch(String key) async {
    try {
      await widget.admin.switchTenant(key);
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.message)));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_tenants.length < 2) {
      return const SizedBox.shrink();
    }
    final current = widget.admin.currentTenant;
    final name = _tenants.firstWhere((t) => t['key'] == current,
        orElse: () => _tenants.first)['name'];
    return PopupMenuButton<String>(
      key: const Key('tenant-menu'),
      tooltip: 'Switch',
      position: PopupMenuPosition.under,
      onSelected: _switch,
      itemBuilder: (_) => [
        for (final t in _tenants)
          CheckedPopupMenuItem<String>(
            key: Key('tenant-${t['key']}'),
            value: '${t['key']}',
            checked: t['key'] == current,
            child: Text('${t['name']}'),
          ),
      ],
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 8),
        child: Row(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.apartment_outlined, size: 18),
          const SizedBox(width: 6),
          Text('$name', key: const Key('tenant-current')),
          const Icon(Icons.arrow_drop_down),
        ]),
      ),
    );
  }
}
