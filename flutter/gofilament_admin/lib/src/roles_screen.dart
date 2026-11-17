import 'package:flutter/material.dart';
import 'i18n.dart';

import 'api.dart';

/// Roles and the users who have them (Panel.Roles).
class RolesScreen extends StatefulWidget {
  const RolesScreen({super.key, required this.api});
  final AdminApi api;

  @override
  State<RolesScreen> createState() => _RolesScreenState();
}

class _RolesScreenState extends State<RolesScreen> {
  Map<String, dynamic>? _data;
  List<Map<String, dynamic>>? _users;
  final _query = TextEditingController();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _query.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final d = await widget.api.roles();
      if (mounted) setState(() => _data = d);
      if (d['can_find_users'] == true) _loadUsers();
    } on ApiException catch (e) {
      _toast(e.message);
    }
  }

  Future<void> _loadUsers() async {
    try {
      final u = await widget.api.roleUsers(_query.text.trim());
      if (mounted) setState(() => _users = u);
    } on ApiException catch (e) {
      _toast(e.message);
    }
  }

  void _toast(String m) {
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(m)));
    }
  }

  List<Map<String, dynamic>> get _roles {
    final r = _data?['roles'];
    return r is List
        ? [
            for (final x in r)
              if (x is Map) Map<String, dynamic>.from(x)
          ]
        : const [];
  }

  Map<String, String> get _permissions {
    final p = _data?['permissions'];
    return p is Map ? p.map((k, v) => MapEntry('$k', '$v')) : const {};
  }

  Future<void> _edit([Map<String, dynamic>? role]) async {
    final saved = await showDialog<bool>(
      context: context,
      builder: (_) =>
          _RoleDialog(api: widget.api, role: role, permissions: _permissions),
    );
    if (saved == true) _load();
  }

  Future<void> _delete(Map<String, dynamic> role) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(tr('Delete {name}?', {'name': role['label']})),
        content: Text(tr('Users with this role lose its permissions.')),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(context, false),
              child: Text(tr('Cancel'))),
          FilledButton(
              key: const Key('role-delete-ok'),
              onPressed: () => Navigator.pop(context, true),
              child: Text(tr('Delete'))),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await widget.api.deleteRole('${role['name']}');
      _load();
    } on ApiException catch (e) {
      _toast(e.message);
    }
  }

  Future<void> _toggleUserRole(
      Map<String, dynamic> user, String role, bool on) async {
    final roles = [for (final r in (user['roles'] as List? ?? const [])) '$r'];
    on ? roles.add(role) : roles.remove(role);
    try {
      await widget.api.setUserRoles('${user['id']}', roles);
      setState(() => user['roles'] = roles);
    } on ApiException catch (e) {
      _toast(e.message);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final data = _data;
    return Scaffold(
      appBar: AppBar(title: Text(tr('Roles')), actions: [
        IconButton(
            key: const Key('role-new'),
            tooltip: tr('New role'),
            icon: const Icon(Icons.add),
            onPressed: data == null ? null : () => _edit()),
      ]),
      body: data == null
          ? const Center(child: CircularProgressIndicator())
          : ListView(padding: const EdgeInsets.all(16), children: [
              for (final r in _roles)
                Card(
                  child: ListTile(
                    key: Key('role-${r['name']}'),
                    title: Text('${r['label']}'),
                    subtitle: Text(
                        '${(r['permissions'] as List).contains('*') ? 'Everything' : '${(r['permissions'] as List).length} permissions'} · ${r['users']} users'),
                    onTap: () => _edit(r),
                    trailing: IconButton(
                      key: Key('role-delete-${r['name']}'),
                      tooltip: tr('Delete'),
                      icon: Icon(Icons.delete_outline,
                          color: theme.colorScheme.error),
                      onPressed: () => _delete(r),
                    ),
                  ),
                ),
              if (data['can_find_users'] == true) ...[
                const SizedBox(height: 24),
                Text(tr('Users'), style: theme.textTheme.titleMedium),
                TextField(
                  key: const Key('role-user-search'),
                  controller: _query,
                  decoration: InputDecoration(
                      prefixIcon: Icon(Icons.search),
                      hintText: tr('Search users')),
                  onSubmitted: (_) => _loadUsers(),
                ),
                for (final u in _users ?? const <Map<String, dynamic>>[])
                  Card(
                    key: Key('role-user-${u['id']}'),
                    child: Padding(
                      padding: const EdgeInsets.all(12),
                      child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text('${u['name']}',
                                style: const TextStyle(
                                    fontWeight: FontWeight.w600)),
                            Wrap(spacing: 8, children: [
                              for (final r in _roles)
                                FilterChip(
                                  key: Key('user-${u['id']}-role-${r['name']}'),
                                  label: Text('${r['label']}'),
                                  selected: (u['roles'] as List? ?? const [])
                                      .contains(r['name']),
                                  onSelected: (on) =>
                                      _toggleUserRole(u, '${r['name']}', on),
                                ),
                            ]),
                          ]),
                    ),
                  ),
              ],
            ]),
    );
  }
}

class _RoleDialog extends StatefulWidget {
  const _RoleDialog(
      {required this.api, required this.role, required this.permissions});
  final AdminApi api;
  final Map<String, dynamic>? role;
  final Map<String, String> permissions;

  @override
  State<_RoleDialog> createState() => _RoleDialogState();
}

class _RoleDialogState extends State<_RoleDialog> {
  late final _name =
      TextEditingController(text: '${widget.role?['name'] ?? ''}');
  late final _label =
      TextEditingController(text: '${widget.role?['label'] ?? ''}');
  late final Set<String> _perms = {
    for (final p in (widget.role?['permissions'] as List? ?? const [])) '$p'
  };
  String? _error;
  Map<String, String> _fieldErrors = const {};

  @override
  void dispose() {
    _name.dispose();
    _label.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    try {
      await widget.api.saveRole(
          original: widget.role == null ? null : '${widget.role!['name']}',
          name: _name.text.trim(),
          label: _label.text.trim(),
          permissions: _perms.toList()..sort());
      if (mounted) Navigator.pop(context, true);
    } on ApiException catch (e) {
      final errs = e.body?['errors'];
      setState(() {
        _error = e.message;
        _fieldErrors =
            errs is Map ? errs.map((k, v) => MapEntry('$k', '$v')) : const {};
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final groups = <String, List<String>>{};
    for (final p in widget.permissions.keys) {
      final g = p.contains('.') ? p.substring(0, p.indexOf('.')) : p;
      groups.putIfAbsent(g, () => []).add(p);
    }
    final names = groups.keys.toList()..sort();
    return AlertDialog(
      title: Text(widget.role == null ? tr('New role') : tr('Edit role')),
      content: SizedBox(
        width: 520,
        child: ListView(shrinkWrap: true, children: [
          if (_error != null && _fieldErrors.isEmpty)
            Text(_error!,
                key: const Key('role-error'),
                style: TextStyle(color: Theme.of(context).colorScheme.error)),
          TextField(
            key: const Key('role-name'),
            controller: _name,
            decoration: InputDecoration(
                labelText: tr('Name (used in code)'),
                errorText: _fieldErrors['name']),
          ),
          TextField(
            key: const Key('role-label'),
            controller: _label,
            decoration: InputDecoration(labelText: tr('Label')),
          ),
          CheckboxListTile(
            key: const Key('perm-*'),
            value: _perms.contains('*'),
            title: Text(tr('Everything')),
            onChanged: (v) => setState(
                () => v == true ? _perms.add('*') : _perms.remove('*')),
          ),
          for (final g in names) ...[
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child:
                  Text(g, style: const TextStyle(fontWeight: FontWeight.w700)),
            ),
            for (final p in (groups[g]!..sort()))
              CheckboxListTile(
                key: Key('perm-$p'),
                dense: true,
                value: _perms.contains(p),
                title: Text(widget.permissions[p] ?? p),
                onChanged: (v) => setState(
                    () => v == true ? _perms.add(p) : _perms.remove(p)),
              ),
          ],
        ]),
      ),
      actions: [
        TextButton(
            onPressed: () => Navigator.pop(context), child: Text(tr('Cancel'))),
        FilledButton(
            key: const Key('role-save'),
            onPressed: _save,
            child: Text(tr('Save'))),
      ],
    );
  }
}
