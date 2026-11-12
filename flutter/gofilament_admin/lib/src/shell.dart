import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'controller.dart';
import 'dashboard.dart';
import 'icons.dart';
import 'login.dart';
import 'resource_list.dart';
import 'theme.dart';

class _NavEntry {
  _NavEntry(this.id, this.label, this.icon, this.group);

  final String id;
  final String label;
  final IconData icon;
  final String group;
}

/// The signed-in layout: grouped sidebar, top bar and the current page.
class AdminShell extends StatefulWidget {
  const AdminShell({super.key, required this.admin});

  final AdminController admin;

  @override
  State<AdminShell> createState() => _AdminShellState();
}

class _AdminShellState extends State<AdminShell> {
  static const _dashboard = 'dashboard';
  String _current = _dashboard;

  /// Record to open once after a deep link such as `#/resources/products/12`
  /// ("new" opens the create form).
  String? _pendingRecord;

  @override
  void initState() {
    super.initState();
    _restoreFromLocation();
  }

  /// Deep links: `#/resources/<slug>[/<id>|/new]` and `#/pages/<id>`, so a
  /// refresh or a shared link lands on the same screen.
  void _restoreFromLocation() {
    if (admin.initialLocation.isEmpty) return;
    final parts =
        admin.initialLocation.split('/').where((p) => p.isNotEmpty).toList();
    // Drawn from a schema snapshot: pages wait for the live schema, which
    // decides who may open them (see build).
    if (!admin.schemaIsLive && parts.isNotEmpty && parts[0] == 'pages') return;
    if (parts.length >= 2 &&
        parts[0] == 'resources' &&
        admin.resource(parts[1]) != null) {
      _current = 'resource:${parts[1]}';
      if (parts.length >= 3) _pendingRecord = parts[2];
    } else if (parts.length >= 2 && parts[0] == 'pages') {
      final page =
          admin.pages.where((p) => p.id == parts[1] && admin.canOpenPage(p));
      if (page.isNotEmpty) {
        if (page.first.fullscreen) {
          // full-screen pages (e.g. a till) open over the shell once it is built
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (mounted) _go('page:${parts[1]}');
          });
        } else {
          _current = 'page:${parts[1]}';
        }
      }
    }
    admin.initialLocation = '';
  }

  void _publishLocation() {
    admin.initialLocation = '';
    final path = _current == _dashboard
        ? '/'
        : (_current.startsWith('resource:')
            ? '/resources/${_current.substring(9)}'
            : '/pages/${_current.substring(5)}');
    SystemNavigator.routeInformationUpdated(
        uri: Uri.parse(path), replace: true);
  }

  AdminController get admin => widget.admin;

  /// Navigation entries grouped in the order the server lists its groups.
  Map<String, List<_NavEntry>> _groups() {
    final entries = <_NavEntry>[];
    for (final page in admin.pages) {
      if (!admin.canOpenPage(page)) continue;
      entries
          .add(_NavEntry('page:${page.id}', page.label, page.icon, page.group));
    }
    for (final r in admin.resources) {
      entries.add(_NavEntry(
        'resource:${r['slug']}',
        '${r['plural_model_name'] ?? r['model_name'] ?? r['slug']}',
        adminIcon(r['icon'] as String?),
        '${r['group'] ?? ''}',
      ));
    }

    final order = <String>[''];
    final nav = admin.schema['navigation'];
    if (nav is List) {
      for (final g in nav) {
        if (g is Map && g['name'] is String && !order.contains(g['name'])) {
          order.add(g['name'] as String);
        }
      }
    }
    for (final e in entries) {
      if (!order.contains(e.group)) order.add(e.group);
    }

    final grouped = <String, List<_NavEntry>>{};
    for (final name in order) {
      final items = entries.where((e) => e.group == name).toList();
      if (items.isNotEmpty) grouped[name] = items;
    }
    return grouped;
  }

  void _go(String id, {bool closeDrawer = false}) {
    if (closeDrawer) Navigator.of(context).pop();
    if (id.startsWith('page:')) {
      final page = admin.pages.firstWhere((p) => 'page:${p.id}' == id);
      if (page.fullscreen) {
        Navigator.of(context).push(MaterialPageRoute<void>(
          fullscreenDialog: true,
          builder: (ctx) => page.builder(ctx, admin),
        ));
        return;
      }
    }
    setState(() {
      _current = id;
      _pendingRecord = null;
    });
    _publishLocation();
  }

  String _title() {
    if (_current == _dashboard) return 'Dashboard';
    if (_current.startsWith('resource:')) {
      final r = admin.resource(_current.substring(9));
      return '${r?['plural_model_name'] ?? r?['model_name'] ?? ''}';
    }
    final page = admin.pages.where((p) => 'page:${p.id}' == _current);
    return page.isEmpty ? '' : page.first.label;
  }

  Widget _body() {
    if (_current.startsWith('resource:')) {
      final slug = _current.substring(9);
      if (admin.resource(slug) != null) {
        final open = _pendingRecord;
        _pendingRecord = null;
        return ResourceListPage(
            key: ValueKey(slug), admin: admin, slug: slug, openRecord: open);
      }
    }
    if (_current.startsWith('page:')) {
      final page = admin.pages
          .where((p) => 'page:${p.id}' == _current && admin.canOpenPage(p));
      if (page.isNotEmpty) return page.first.builder(context, admin);
    }
    return DashboardPage(admin: admin);
  }

  @override
  Widget build(BuildContext context) {
    if (admin.initialLocation.isNotEmpty && admin.schemaIsLive) {
      // a deep link held back while the shell showed the schema snapshot
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted) setState(_restoreFromLocation);
      });
    }
    final wide = MediaQuery.sizeOf(context).width >= 1000;
    final sidebar = _Sidebar(
      admin: admin,
      groups: _groups(),
      current: _current,
      onSelect: (id) => _go(id, closeDrawer: !wide),
    );

    return Scaffold(
      drawer: wide ? null : Drawer(width: 280, child: SafeArea(child: sidebar)),
      body: SafeArea(
        child: Row(
          children: [
            if (wide)
              Container(
                width: 264,
                decoration: BoxDecoration(
                  color: Theme.of(context).colorScheme.surface,
                  border: Border(
                      right: BorderSide(color: Theme.of(context).dividerColor)),
                ),
                child: sidebar,
              ),
            Expanded(
              child: Column(children: [
                _TopBar(admin: admin, title: _title(), showMenu: !wide),
                Expanded(child: _body()),
              ]),
            ),
          ],
        ),
      ),
    );
  }
}

class _Sidebar extends StatelessWidget {
  const _Sidebar(
      {required this.admin,
      required this.groups,
      required this.current,
      required this.onSelect});

  final AdminController admin;
  final Map<String, List<_NavEntry>> groups;
  final String current;
  final ValueChanged<String> onSelect;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final children = <Widget>[
      _item(
          context,
          _NavEntry(
              'dashboard', 'Dashboard', Icons.space_dashboard_outlined, '')),
    ];
    groups.forEach((name, items) {
      if (name.isNotEmpty) {
        children.add(Padding(
          padding: const EdgeInsets.fromLTRB(16, 18, 16, 6),
          child: Text(name.toUpperCase(),
              style: theme.textTheme.labelSmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 0.8)),
        ));
      }
      children.addAll(items.map((e) => _item(context, e)));
    });

    return Column(children: [
      Padding(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 12),
        child: Row(children: [
          BrandMark(admin: admin, size: 34),
          const SizedBox(width: 10),
          Expanded(
            child: Text(admin.brandName,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.w700)),
          ),
        ]),
      ),
      const Divider(),
      Expanded(
          child: ListView(
              padding: const EdgeInsets.symmetric(vertical: 8),
              children: children)),
    ]);
  }

  Widget _item(BuildContext context, _NavEntry e) {
    final theme = Theme.of(context);
    final palette = AdminPalette.of(context);
    final selected = e.id == current;
    final color =
        selected ? theme.colorScheme.primary : theme.colorScheme.onSurface;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 1),
      child: Material(
        color: selected
            ? palette.color('primary', 500).withValues(alpha: 0.12)
            : Colors.transparent,
        borderRadius: BorderRadius.circular(10),
        child: InkWell(
          key: Key('nav-${e.id}'),
          borderRadius: BorderRadius.circular(10),
          onTap: () => onSelect(e.id),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 10),
            child: Row(children: [
              Icon(e.icon,
                  size: 20,
                  color: selected ? color : theme.colorScheme.onSurfaceVariant),
              const SizedBox(width: 12),
              Expanded(
                child: Text(e.label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                        color: color,
                        fontWeight:
                            selected ? FontWeight.w600 : FontWeight.w500)),
              ),
            ]),
          ),
        ),
      ),
    );
  }
}

class _TopBar extends StatelessWidget {
  const _TopBar(
      {required this.admin, required this.title, required this.showMenu});

  final AdminController admin;
  final String title;
  final bool showMenu;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final name = '${admin.user['name'] ?? admin.user['login'] ?? ''}';
    final initial = name.isEmpty ? '?' : name[0].toUpperCase();

    return Container(
      height: 60,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        border: Border(bottom: BorderSide(color: theme.dividerColor)),
      ),
      child: Row(children: [
        if (showMenu)
          IconButton(
            tooltip: 'Menu',
            icon: const Icon(Icons.menu),
            onPressed: () => Scaffold.of(context).openDrawer(),
          ),
        const SizedBox(width: 4),
        Expanded(
          child: Text(title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w600)),
        ),
        if (admin.brand['theme_mode'] != 'dark' &&
            admin.brand['theme_mode'] != 'light')
          IconButton(
            tooltip: theme.brightness == Brightness.dark
                ? 'Light mode'
                : 'Dark mode',
            icon: Icon(theme.brightness == Brightness.dark
                ? Icons.light_mode_outlined
                : Icons.dark_mode_outlined),
            onPressed: () => admin.toggleTheme(theme.brightness),
          ),
        const SizedBox(width: 4),
        PopupMenuButton<String>(
          key: const Key('account-menu'),
          tooltip: 'Account',
          position: PopupMenuPosition.under,
          onSelected: (v) {
            if (v == 'logout') admin.logout();
          },
          itemBuilder: (_) => [
            PopupMenuItem<String>(
              enabled: false,
              child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(name,
                        style: TextStyle(
                            fontWeight: FontWeight.w600,
                            color: theme.colorScheme.onSurface)),
                    if ('${admin.user['email'] ?? ''}'.isNotEmpty)
                      Text('${admin.user['email']}',
                          style: theme.textTheme.bodySmall),
                  ]),
            ),
            const PopupMenuDivider(),
            const PopupMenuItem<String>(
              value: 'logout',
              child: Row(children: [
                Icon(Icons.logout, size: 18),
                SizedBox(width: 10),
                Text('Sign out')
              ]),
            ),
          ],
          child: Padding(
            padding: const EdgeInsets.all(6),
            child: CircleAvatar(
              radius: 17,
              backgroundColor: theme.colorScheme.primary,
              foregroundColor: theme.colorScheme.onPrimary,
              child: Text(initial,
                  style: const TextStyle(fontWeight: FontWeight.w700)),
            ),
          ),
        ),
      ]),
    );
  }
}
