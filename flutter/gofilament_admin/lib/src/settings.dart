import 'package:flutter/material.dart';

import 'account.dart';
import 'controller.dart';
import 'i18n.dart';

/// The user's preferences on this device: appearance, language and the
/// desktop sidebar. Each section appears when the panel offers the choice.
class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key, required this.admin});

  final AdminController admin;

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: admin,
      builder: (context, _) {
        final theme = Theme.of(context);
        final flags = admin.boot['locale_flags'];
        final sections = <Widget>[
          if (!admin.themeLocked)
            _Section(
              title: tr('Appearance'),
              subtitle: tr('Light, dark, or follow this device.'),
              child: SegmentedButton<ThemeMode>(
                key: const Key('settings-theme'),
                showSelectedIcon: false,
                segments: [
                  ButtonSegment(
                      value: ThemeMode.system,
                      icon: const Icon(Icons.brightness_auto_outlined),
                      label: Text(tr('System'))),
                  ButtonSegment(
                      value: ThemeMode.light,
                      icon: const Icon(Icons.light_mode_outlined),
                      label: Text(tr('Light'))),
                  ButtonSegment(
                      value: ThemeMode.dark,
                      icon: const Icon(Icons.dark_mode_outlined),
                      label: Text(tr('Dark'))),
                ],
                selected: {admin.themeMode},
                onSelectionChanged: (s) => admin.setThemeMode(s.first),
              ),
            ),
          if (admin.languages.isNotEmpty)
            _Section(
              title: tr('Language'),
              child: RadioGroup<String>(
                groupValue: admin.locale,
                onChanged: (v) {
                  if (v != null && v != admin.locale) admin.setLanguage(v);
                },
                child: Column(children: [
                  for (final l in admin.languages.entries)
                    RadioListTile<String>(
                      key: Key('settings-language-${l.key}'),
                      contentPadding: EdgeInsets.zero,
                      value: l.key,
                      secondary: flags is Map && flags[l.key] != null
                          ? Text('${flags[l.key]}',
                              style: const TextStyle(fontSize: 20))
                          : null,
                      title: Text(l.value),
                    ),
                ]),
              ),
            ),
          if (admin.sidebarOptions.length > 1)
            _Section(
              title: tr('Sidebar'),
              subtitle: tr('How the menu shows on wide screens.'),
              child: SegmentedButton<String>(
                key: const Key('settings-sidebar'),
                showSelectedIcon: false,
                segments: [
                  for (final m in admin.sidebarOptions)
                    ButtonSegment(
                      value: m,
                      icon: Icon(switch (m) {
                        'icons' => Icons.view_sidebar_outlined,
                        'hidden' => Icons.fullscreen,
                        _ => Icons.menu_open,
                      }),
                      label: Text(switch (m) {
                        'icons' => tr('Icons only'),
                        'hidden' => tr('Hidden'),
                        _ => tr('Full'),
                      }),
                    ),
                ],
                selected: {admin.sidebarMode},
                onSelectionChanged: (s) => admin.setSidebarMode(s.first),
              ),
            ),
          if (admin.hasAccountScreen)
            _Section(
              title: tr('Account'),
              child: ListTile(
                key: const Key('settings-account'),
                contentPadding: EdgeInsets.zero,
                leading: const Icon(Icons.manage_accounts_outlined),
                title: Text(tr('Your account')),
                subtitle: Text(tr('Profile, password and two-factor sign-in')),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => Navigator.of(context).push(MaterialPageRoute<void>(
                    builder: (_) => AccountScreen(admin: admin))),
              ),
            ),
        ];
        return Scaffold(
          appBar: AppBar(title: Text(tr('Settings'))),
          body: ListView(
            padding: const EdgeInsets.all(16),
            children: [
              Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 640),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: sections.isEmpty
                        ? [
                            Text(tr('Nothing to set up here.'),
                                style: theme.textTheme.bodyMedium)
                          ]
                        : sections,
                  ),
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

class _Section extends StatelessWidget {
  const _Section({required this.title, this.subtitle, required this.child});

  final String title;
  final String? subtitle;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      margin: const EdgeInsets.only(bottom: 16),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(title,
              style: theme.textTheme.titleMedium
                  ?.copyWith(fontWeight: FontWeight.w600)),
          if (subtitle != null) ...[
            const SizedBox(height: 4),
            Text(subtitle!,
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: theme.colorScheme.onSurfaceVariant)),
          ],
          const SizedBox(height: 12),
          child,
        ]),
      ),
    );
  }
}
