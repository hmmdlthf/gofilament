import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'i18n.dart';

import 'account.dart';
import 'controller.dart';
import 'global_search.dart';
import 'dashboard.dart';
import 'icons.dart';
import 'login.dart';
import 'notifications.dart';
import 'roles_screen.dart';
import 'settings.dart';
import 'tenant_switcher.dart';
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

  /// Filters and search a link carried (`?filter_status=open&search=x`),
  /// applied once to the list it opens.
  Map<String, String> _pendingQuery = const {};
  int _listGeneration = 0;

  @override
  void initState() {
    super.initState();
    _restoreFromLocation();
  }

  void _cycleSidebar() {
    final modes = admin.sidebarOptions;
    admin.setSidebarMode(
        modes[(modes.indexOf(admin.sidebarMode) + 1) % modes.length]);
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

  bool get _searchEnabled {
    final g = admin.schema['global_search'];
    return g is Map && g['enabled'] == true;
  }

  Future<void> _openSearch() async {
    final url = await showDialog<String>(
        context: context, builder: (_) => GlobalSearchDialog(admin: admin));
    if (url != null && mounted) openUrl(url);
  }

  /// Opens a panel URL from the server (`<panel>/<slug>[/<id>[/edit]]` or a
  /// page): global search results, notifications.
  void openUrl(String url) {
    final uri = Uri.tryParse(url);
    var path = uri?.path ?? url;
    final query = uri?.queryParameters ?? const <String, String>{};
    final base = '${admin.schema['path'] ?? ''}';
    if (base.isNotEmpty && path.startsWith(base)) {
      path = path.substring(base.length);
    }
    final parts = path.split('/').where((p) => p.isNotEmpty).toList();
    if (parts.isNotEmpty && parts[0] == 'resources') parts.removeAt(0);
    if (parts.isEmpty) {
      _go(_dashboard);
      return;
    }
    if (admin.resource(parts[0]) != null) {
      setState(() {
        _current = 'resource:${parts[0]}';
        _pendingRecord = parts.length >= 2 && parts[1] != 'create'
            ? (parts[1] == 'new' ? 'new' : parts[1])
            : null;
        _pendingQuery = query;
        _listGeneration++;
      });
      _publishLocation();
      return;
    }
    final page =
        admin.pages.where((p) => p.id == parts[0] && admin.canOpenPage(p));
    if (page.isNotEmpty) _go('page:${page.first.id}');
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
        final query = _pendingQuery;
        _pendingQuery = const {};
        return ResourceListPage(
            key: ValueKey(query.isEmpty ? slug : '$slug#$_listGeneration'),
            admin: admin,
            slug: slug,
            openRecord: open,
            initialFilters: linkFilters(query),
            initialSearch: query['search'] ?? '');
      }
    }
    if (_current.startsWith('page:')) {
      final page = admin.pages
          .where((p) => 'page:${p.id}' == _current && admin.canOpenPage(p));
      if (page.isNotEmpty) return page.first.builder(context, admin);
    }
    return DashboardPage(admin: admin, onOpen: openUrl);
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
    final mode = wide ? admin.sidebarMode : 'expanded';
    final modes = admin.sidebarOptions;
    final next = modes[(modes.indexOf(mode) + 1) % modes.length];
    _Sidebar sidebarFor({required bool compact}) => _Sidebar(
          admin: admin,
          groups: _groups(),
          current: _current,
          compact: compact,
          onSelect: (id) => _go(id, closeDrawer: !wide),
        );
    final sidebar = sidebarFor(compact: false);

    return Scaffold(
      drawer: wide ? null : Drawer(width: 280, child: SafeArea(child: sidebar)),
      body: SafeArea(
        child: Row(
          children: [
            if (wide && mode != 'hidden')
              Container(
                key: Key('sidebar-$mode'),
                width: mode == 'icons' ? 72 : 264,
                decoration: BoxDecoration(
                  color: Theme.of(context).colorScheme.surface,
                  border: Border(
                      right: BorderSide(color: Theme.of(context).dividerColor)),
                ),
                child: mode == 'icons' ? sidebarFor(compact: true) : sidebar,
              ),
            Expanded(
              child: Column(children: [
                _TopBar(
                    admin: admin,
                    title: _title(),
                    showMenu: !wide,
                    sidebarMode: mode,
                    nextSidebarMode: next,
                    onToggleSidebar:
                        wide && modes.length > 1 ? _cycleSidebar : null,
                    onSearch: _searchEnabled ? _openSearch : null),
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
      required this.onSelect,
      this.compact = false});

  final AdminController admin;
  final Map<String, List<_NavEntry>> groups;
  final String current;
  final ValueChanged<String> onSelect;

  /// Icon rail: icons only, labels as tooltips, groups as dividers.
  final bool compact;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (compact) return _rail(context);
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

  Widget _rail(BuildContext context) {
    final children = <Widget>[
      _railItem(
          context,
          _NavEntry(
              'dashboard', 'Dashboard', Icons.space_dashboard_outlined, '')),
    ];
    groups.forEach((name, items) {
      if (name.isNotEmpty) {
        children.add(const Padding(
            padding: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
            child: Divider(height: 1)));
      }
      children.addAll(items.map((e) => _railItem(context, e)));
    });
    return Column(children: [
      Padding(
        padding: const EdgeInsets.symmetric(vertical: 16),
        child: Tooltip(
            message: admin.brandName, child: BrandMark(admin: admin, size: 34)),
      ),
      const Divider(),
      Expanded(
          child: ListView(
              padding: const EdgeInsets.symmetric(vertical: 8),
              children: children)),
    ]);
  }

  Widget _railItem(BuildContext context, _NavEntry e) {
    final theme = Theme.of(context);
    final palette = AdminPalette.of(context);
    final selected = e.id == current;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 2),
      child: Tooltip(
        message: e.label,
        preferBelow: false,
        waitDuration: const Duration(milliseconds: 250),
        child: Material(
          color: selected
              ? palette.color('primary', 500).withValues(alpha: 0.12)
              : Colors.transparent,
          borderRadius: BorderRadius.circular(10),
          child: InkWell(
            key: Key('nav-${e.id}'),
            borderRadius: BorderRadius.circular(10),
            onTap: () => onSelect(e.id),
            child: SizedBox(
              height: 44,
              child: Center(
                child: Icon(e.icon,
                    size: 22,
                    semanticLabel: e.label,
                    color: selected
                        ? theme.colorScheme.primary
                        : theme.colorScheme.onSurfaceVariant),
              ),
            ),
          ),
        ),
      ),
    );
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
      {required this.admin,
      required this.title,
      required this.showMenu,
      this.sidebarMode = 'expanded',
      this.nextSidebarMode = 'expanded',
      this.onToggleSidebar,
      this.onSearch});

  final AdminController admin;
  final String title;
  final bool showMenu;
  final String sidebarMode;
  final String nextSidebarMode;

  /// Switches the desktop sidebar to [nextSidebarMode] (null: no button).
  final VoidCallback? onToggleSidebar;
  final VoidCallback? onSearch;

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
            tooltip: tr('Menu'),
            icon: const Icon(Icons.menu),
            onPressed: () => Scaffold.of(context).openDrawer(),
          ),
        if (onToggleSidebar != null)
          IconButton(
            key: const Key('sidebar-toggle'),
            tooltip: switch (nextSidebarMode) {
              'icons' => tr('Collapse to icons'),
              'hidden' => tr('Hide sidebar'),
              _ => tr('Expand sidebar'),
            },
            icon: Icon(switch (sidebarMode) {
              'expanded' => Icons.menu_open,
              'icons' => Icons.view_sidebar_outlined,
              _ => Icons.menu,
            }),
            onPressed: onToggleSidebar,
          ),
        const SizedBox(width: 4),
        Expanded(
          child: Text(title,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w600)),
        ),
        if (admin.hasTenancy) TenantSwitcher(admin: admin),
        if (admin.schema['notifications'] == true)
          NotificationBell(api: admin.api),
        if (onSearch != null)
          IconButton(
            key: const Key('global-search'),
            tooltip: tr('Search'),
            icon: const Icon(Icons.search),
            onPressed: onSearch,
          ),
        if (admin.brand['theme_mode'] != 'dark' &&
            admin.brand['theme_mode'] != 'light')
          IconButton(
            tooltip: theme.brightness == Brightness.dark
                ? tr('Light mode')
                : tr('Dark mode'),
            icon: Icon(theme.brightness == Brightness.dark
                ? Icons.light_mode_outlined
                : Icons.dark_mode_outlined),
            onPressed: () => admin.toggleTheme(theme.brightness),
          ),
        const SizedBox(width: 4),
        PopupMenuButton<String>(
          key: const Key('account-menu'),
          tooltip: tr('Account'),
          position: PopupMenuPosition.under,
          onSelected: (v) {
            if (v == 'logout') admin.logout();
            if (v.startsWith('locale:')) admin.setLanguage(v.substring(7));
            if (v == 'roles') {
              Navigator.of(context).push(MaterialPageRoute<void>(
                  builder: (_) => RolesScreen(api: admin.api)));
            }
            if (v == 'account') {
              Navigator.of(context).push(MaterialPageRoute<void>(
                  builder: (_) => AccountScreen(admin: admin)));
            }
            if (v == 'settings') {
              Navigator.of(context).push(MaterialPageRoute<void>(
                  builder: (_) => SettingsScreen(admin: admin)));
            }
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
            if (admin.schema['manage_roles'] == true)
              PopupMenuItem<String>(
                key: Key('account-menu-roles'),
                value: 'roles',
                child: Row(children: [
                  Icon(Icons.admin_panel_settings_outlined, size: 18),
                  SizedBox(width: 10),
                  Text(tr('Roles'))
                ]),
              ),
            if (admin.hasAccountScreen)
              PopupMenuItem<String>(
                key: Key('account-menu-profile'),
                value: 'account',
                child: Row(children: [
                  Icon(Icons.manage_accounts_outlined, size: 18),
                  SizedBox(width: 10),
                  Text(tr('Your account'))
                ]),
              ),
            PopupMenuItem<String>(
              key: Key('account-menu-settings'),
              value: 'settings',
              child: Row(children: [
                Icon(Icons.settings_outlined, size: 18),
                SizedBox(width: 10),
                Text(tr('Settings'))
              ]),
            ),
            for (final l in admin.languages.entries)
              CheckedPopupMenuItem<String>(
                key: Key('language-${l.key}'),
                value: 'locale:${l.key}',
                checked: l.key == admin.locale,
                child: Text(l.value),
              ),
            if (admin.languages.isNotEmpty) const PopupMenuDivider(),
            PopupMenuItem<String>(
              value: 'logout',
              child: Row(children: [
                Icon(Icons.logout, size: 18),
                SizedBox(width: 10),
                Text(tr('Sign out'))
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

/// The list filters a panel link carries: `filter_<name>=value` or
/// `filters[<name>]=value`.
Map<String, String> linkFilters(Map<String, String> query) {
  final out = <String, String>{};
  final bracket = RegExp(r'^filters\[(.+)\]$');
  for (final e in query.entries) {
    if (e.key.startsWith('filter_')) {
      out[e.key.substring(7)] = e.value;
    } else {
      final m = bracket.firstMatch(e.key);
      if (m != null) out[m.group(1)!] = e.value;
    }
  }
  return out;
}
