import 'package:flutter/material.dart';

import 'account.dart';
import 'controller.dart';
import 'i18n.dart';
import 'responsive.dart';
import 'roles_screen.dart';
import 'settings.dart';
import 'theme.dart';

/// Opens the account hub: a full page on a phone, a dialog elsewhere.
Future<void> showAccountHub(BuildContext context, AdminController admin) =>
    showAdaptivePage<void>(context,
        maxWidth: 440,
        maxHeight: 720,
        builder: (_) => AccountHub(admin: admin));

/// Opens a page of the account area ([SettingsScreen], [AccountScreen]): a
/// full page on a phone, a dialog elsewhere.
Future<void> showAccountPage(BuildContext context, WidgetBuilder builder) =>
    showAdaptivePage<void>(context,
        maxWidth: 640, maxHeight: 820, builder: builder);

/// The leading button of a page shown with [showAdaptivePage]: a close button
/// in a dialog, the usual back button on a full page.
Widget? adaptiveLeading(BuildContext context) =>
    AdminFormFactor.of(context) == AdminFormFactor.phone
        ? null
        : CloseButton(onPressed: () => Navigator.of(context).maybePop());

/// What the top bar avatar opens: who is signed in, the account pages,
/// workspace, appearance and language, and signing out.
class AccountHub extends StatefulWidget {
  const AccountHub({super.key, required this.admin});

  final AdminController admin;

  @override
  State<AccountHub> createState() => _AccountHubState();
}

class _AccountHubState extends State<AccountHub> {
  List<Map<String, dynamic>> _tenants = const [];

  AdminController get admin => widget.admin;

  @override
  void initState() {
    super.initState();
    if (admin.hasTenancy) _loadTenants();
  }

  Future<void> _loadTenants() async {
    try {
      final data = await admin.api.tenants();
      final t = data['tenants'];
      if (!mounted || t is! List) return;
      setState(() => _tenants = [
            for (final x in t)
              if (x is Map) Map<String, dynamic>.from(x)
          ]);
    } catch (_) {}
  }

  bool get _isPage => AdminFormFactor.of(context) == AdminFormFactor.phone;

  /// Opens a page from the hub. On a phone it stacks over the hub (back
  /// returns here); a dialog hub closes first.
  void _open(void Function(BuildContext context) show) {
    if (_isPage) {
      show(context);
      return;
    }
    final navigator = Navigator.of(context);
    final outer = navigator.context;
    navigator.pop();
    show(outer);
  }

  Future<void> _signOut() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(tr('Sign out')),
        content: Text(tr('Sign out of {name}?', {'name': admin.brandName})),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(tr('Cancel'))),
          FilledButton(
              key: const Key('confirm-sign-out'),
              style: FilledButton.styleFrom(
                  backgroundColor: Theme.of(ctx).colorScheme.error),
              onPressed: () => Navigator.pop(ctx, true),
              child: Text(tr('Sign out'))),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    Navigator.of(context).popUntil((r) => r.isFirst);
    await admin.logout();
  }

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: admin,
      builder: (context, _) {
        final theme = Theme.of(context);
        final sizes = AdminSizes.of(context);
        final palette = AdminPalette.of(context);
        final items = <Widget>[
          _ProfileCard(admin: admin),
          const SizedBox(height: 16),
          if (admin.hasAccountScreen)
            _NavCard(
              key: const Key('account-menu-profile'),
              icon: Icons.manage_accounts_outlined,
              color: palette.color('primary'),
              title: tr('Your account'),
              subtitle: tr('Profile, password and two-factor sign-in'),
              onTap: () => _open((c) =>
                  showAccountPage(c, (_) => AccountScreen(admin: admin))),
            ),
          _NavCard(
            key: const Key('account-menu-settings'),
            icon: Icons.settings_outlined,
            color: palette.color('info'),
            title: tr('Settings'),
            subtitle: tr('Appearance, language and sidebar'),
            onTap: () => _open(
                (c) => showAccountPage(c, (_) => SettingsScreen(admin: admin))),
          ),
          if (admin.schema['manage_roles'] == true)
            _NavCard(
              key: const Key('account-menu-roles'),
              icon: Icons.admin_panel_settings_outlined,
              color: palette.color('accent'),
              title: tr('Roles'),
              subtitle: tr('Who can see and change what'),
              onTap: () => _open((c) => Navigator.of(c).push(
                  MaterialPageRoute<void>(
                      builder: (_) => RolesScreen(api: admin.api)))),
            ),
          if (_tenants.length > 1) ...[
            _Heading(tr('Workspace')),
            Card(
              child: Column(children: [
                for (final t in _tenants)
                  ListTile(
                    key: Key('hub-tenant-${t['key']}'),
                    leading: const Icon(Icons.apartment_outlined),
                    title: Text('${t['name']}'),
                    trailing: t['key'] == admin.currentTenant
                        ? Icon(Icons.check, color: theme.colorScheme.primary)
                        : null,
                    onTap: t['key'] == admin.currentTenant
                        ? null
                        : () async {
                            Navigator.of(context).maybePop();
                            await admin.switchTenant('${t['key']}');
                          },
                  ),
              ]),
            ),
          ],
          if (!admin.themeLocked) ...[
            _Heading(tr('Appearance')),
            SegmentedButton<ThemeMode>(
              key: const Key('hub-theme'),
              showSelectedIcon: false,
              segments: [
                ButtonSegment(
                    value: ThemeMode.system,
                    icon: const Icon(Icons.brightness_auto_outlined, size: 18),
                    label: Text(tr('System'))),
                ButtonSegment(
                    value: ThemeMode.light,
                    icon: const Icon(Icons.light_mode_outlined, size: 18),
                    label: Text(tr('Light'))),
                ButtonSegment(
                    value: ThemeMode.dark,
                    icon: const Icon(Icons.dark_mode_outlined, size: 18),
                    label: Text(tr('Dark'))),
              ],
              selected: {admin.themeMode},
              onSelectionChanged: (s) => admin.setThemeMode(s.first),
            ),
          ],
          if (admin.languages.isNotEmpty) ...[
            _Heading(tr('Language')),
            Card(
              clipBehavior: Clip.antiAlias,
              child: ExpansionTile(
                key: const Key('hub-language'),
                leading: const Icon(Icons.translate),
                title: Text(admin.languages[admin.locale] ?? admin.locale),
                shape: const Border(),
                collapsedShape: const Border(),
                children: [
                  for (final l in admin.languages.entries)
                    ListTile(
                      key: Key('language-${l.key}'),
                      title: Text(l.value),
                      trailing: l.key == admin.locale
                          ? Icon(Icons.check, color: theme.colorScheme.primary)
                          : null,
                      onTap: l.key == admin.locale
                          ? null
                          : () => admin.setLanguage(l.key),
                    ),
                ],
              ),
            ),
          ],
          const SizedBox(height: 24),
          OutlinedButton.icon(
            key: const Key('account-menu-logout'),
            icon: Icon(Icons.logout_rounded,
                size: 20, color: theme.colorScheme.error),
            label: Text(tr('Sign out'),
                style: TextStyle(
                    color: theme.colorScheme.error,
                    fontWeight: FontWeight.w600)),
            style: OutlinedButton.styleFrom(
              minimumSize: Size.fromHeight(sizes.controlHeight + 4),
              side: BorderSide(color: theme.colorScheme.error),
            ),
            onPressed: _signOut,
          ),
          const SizedBox(height: 8),
        ];
        return Scaffold(
          appBar: AppBar(
            leading: adaptiveLeading(context),
            title: Text(tr('Account')),
            centerTitle: true,
          ),
          body: SafeArea(
            top: false,
            child: ListView(
              key: const Key('account-hub'),
              padding: EdgeInsets.fromLTRB(
                  sizes.pagePadding + 4, 12, sizes.pagePadding + 4, 16),
              children: items,
            ),
          ),
        );
      },
    );
  }
}

class _ProfileCard extends StatelessWidget {
  const _ProfileCard({required this.admin});

  final AdminController admin;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final name = '${admin.user['name'] ?? admin.user['login'] ?? ''}';
    final email = '${admin.user['email'] ?? ''}';
    final roles = admin.user['roles'];
    final role = roles is List && roles.isNotEmpty
        ? '${roles.first}'
        : '${admin.user['role'] ?? ''}';
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: primary.withValues(alpha: 0.07),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: primary.withValues(alpha: 0.15)),
      ),
      child: Row(children: [
        UserAvatar(admin: admin, radius: 28),
        const SizedBox(width: 14),
        Expanded(
          child:
              Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Text(name.isEmpty ? tr('Signed in') : name,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.titleMedium
                    ?.copyWith(fontWeight: FontWeight.w700)),
            if (email.isNotEmpty) ...[
              const SizedBox(height: 2),
              Text(email,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: theme.textTheme.bodySmall
                      ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
            ],
            if (role.isNotEmpty) ...[
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                decoration: BoxDecoration(
                  color: primary.withValues(alpha: 0.14),
                  borderRadius: BorderRadius.circular(20),
                ),
                child: Text(humanizeRole(role),
                    style: theme.textTheme.labelSmall?.copyWith(
                        color: primary, fontWeight: FontWeight.w700)),
              ),
            ],
          ]),
        ),
      ]),
    );
  }
}

/// "store_manager" → "Store manager".
String humanizeRole(String role) {
  final s = role.replaceAll(RegExp(r'[_-]+'), ' ').trim();
  return s.isEmpty ? s : s[0].toUpperCase() + s.substring(1);
}

/// The signed-in user's initial in a primary-coloured circle.
class UserAvatar extends StatelessWidget {
  const UserAvatar({super.key, required this.admin, this.radius = 17});

  final AdminController admin;
  final double radius;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final name = '${admin.user['name'] ?? admin.user['login'] ?? ''}';
    return CircleAvatar(
      radius: radius,
      backgroundColor: theme.colorScheme.primary,
      foregroundColor: theme.colorScheme.onPrimary,
      child: Text(name.isEmpty ? '?' : name[0].toUpperCase(),
          style:
              TextStyle(fontWeight: FontWeight.w700, fontSize: radius * 0.85)),
    );
  }
}

class _Heading extends StatelessWidget {
  const _Heading(this.text);

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(4, 20, 4, 8),
      child: Text(text.toUpperCase(),
          style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.7)),
    );
  }
}

class _NavCard extends StatelessWidget {
  const _NavCard(
      {super.key,
      required this.icon,
      required this.color,
      required this.title,
      required this.subtitle,
      required this.onTap});

  final IconData icon;
  final Color color;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final rtl = Directionality.of(context) == TextDirection.rtl;
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Card(
        clipBehavior: Clip.antiAlias,
        child: ListTile(
          contentPadding:
              const EdgeInsets.symmetric(horizontal: 14, vertical: 4),
          leading: Container(
            padding: const EdgeInsets.all(9),
            decoration: BoxDecoration(
              color: color.withValues(alpha: 0.12),
              borderRadius: BorderRadius.circular(12),
            ),
            child: Icon(icon, color: color, size: 22),
          ),
          title: Text(title,
              style: theme.textTheme.titleSmall
                  ?.copyWith(fontWeight: FontWeight.w700)),
          subtitle: Text(subtitle,
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          trailing: Icon(rtl ? Icons.chevron_left : Icons.chevron_right,
              color: theme.colorScheme.onSurfaceVariant),
          onTap: onTap,
        ),
      ),
    );
  }
}
