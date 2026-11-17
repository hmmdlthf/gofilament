import 'package:flutter/material.dart';

import 'api.dart';
import 'i18n.dart';

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

  bool get _canDefine => _data?['can_define'] == true;

  List<Map<String, dynamic>> _scopes(String key) {
    final v = _data?[key];
    return v is List
        ? [
            for (final x in v)
              if (x is Map) Map<String, dynamic>.from(x)
          ]
        : const [];
  }

  String _scopeLabel(String key, String id) {
    for (final o in _scopes(key)) {
      if (o['id'] == id) return '${o['label']}';
    }
    if (id != '*') return id;
    return key == 'panels' ? tr('All panels') : tr('All tenants');
  }

  Future<void> _grant(Map<String, dynamic> user) async {
    final done = await showDialog<bool>(
      context: context,
      builder: (_) => _GrantDialog(
        api: widget.api,
        user: user,
        roles: _roles,
        panels: _scopes('panels'),
        tenants: _scopes('tenants'),
      ),
    );
    if (done == true) _loadUsers();
  }

  Future<void> _revoke(
      Map<String, dynamic> user, Map<String, dynamic> b) async {
    try {
      await widget.api.revokeRole(
          '${user['id']}', '${b['role']}', '${b['panel']}', '${b['tenant']}');
      _loadUsers();
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
            onPressed: data == null || !_canDefine ? null : () => _edit()),
      ]),
      body: data == null
          ? const Center(child: CircularProgressIndicator())
          : ListView(padding: const EdgeInsets.all(16), children: [
              for (final r in _roles)
                Card(
                  child: ListTile(
                    key: Key('role-${r['name']}'),
                    title: Text('${r['label']}'),
                    subtitle: Text((r['permissions'] as List).contains('*')
                        ? tr(
                            'Everything · {users} users', {'users': r['users']})
                        : tr('{n} permissions · {users} users', {
                            'n': (r['permissions'] as List).length,
                            'users': r['users']
                          })),
                    onTap: _canDefine ? () => _edit(r) : null,
                    trailing: !_canDefine
                        ? null
                        : IconButton(
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
                            Wrap(spacing: 8, runSpacing: 4, children: [
                              for (final b
                                  in (u['bindings'] as List? ?? const []))
                                if (b is Map)
                                  InputChip(
                                    key: Key(
                                        'binding-${u['id']}-${b['role']}-${b['panel']}-${b['tenant']}'),
                                    label: Text(
                                        '${b['role']} · ${_scopeLabel('panels', '${b['panel']}')} · ${_scopeLabel('tenants', '${b['tenant']}')}'),
                                    deleteIcon:
                                        const Icon(Icons.close, size: 16),
                                    onDeleted: b['can_remove'] == true
                                        ? () => _revoke(
                                            u, Map<String, dynamic>.from(b))
                                        : null,
                                  ),
                              ActionChip(
                                key: Key('grant-${u['id']}'),
                                avatar: const Icon(Icons.add, size: 16),
                                label: Text(tr('Give role')),
                                onPressed: () => _grant(u),
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

class _GrantDialog extends StatefulWidget {
  const _GrantDialog({
    required this.api,
    required this.user,
    required this.roles,
    required this.panels,
    required this.tenants,
  });
  final AdminApi api;
  final Map<String, dynamic> user;
  final List<Map<String, dynamic>> roles;
  final List<Map<String, dynamic>> panels;
  final List<Map<String, dynamic>> tenants;

  @override
  State<_GrantDialog> createState() => _GrantDialogState();
}

class _GrantDialogState extends State<_GrantDialog> {
  String? _role;
  final Set<String> _panels = {};
  final Set<String> _tenants = {};
  String? _error;

  @override
  void initState() {
    super.initState();
    if (widget.roles.isNotEmpty) _role = '${widget.roles.first['name']}';
    if (widget.panels.length == 1) {
      _panels.add('${widget.panels.first['id']}');
    }
    if (widget.tenants.length == 1) {
      _tenants.add('${widget.tenants.first['id']}');
    }
  }

  Future<void> _save() async {
    try {
      await widget.api.grantRole('${widget.user['id']}', _role ?? '',
          panels: _panels.toList(), tenants: _tenants.toList());
      if (mounted) Navigator.pop(context, true);
    } on ApiException catch (e) {
      final errs = e.body?['errors'];
      setState(() => _error =
          errs is Map && errs.isNotEmpty ? '${errs.values.first}' : e.message);
    }
  }

  Widget _checks(String title, List<Map<String, dynamic>> opts, Set<String> sel,
      String keyPrefix) {
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      Padding(
        padding: const EdgeInsets.only(top: 12),
        child: Text(title, style: const TextStyle(fontWeight: FontWeight.w700)),
      ),
      for (final o in opts)
        CheckboxListTile(
          key: Key('$keyPrefix-${o['id']}'),
          dense: true,
          value: sel.contains('${o['id']}'),
          title: Text(o['id'] == '*'
              ? (keyPrefix == 'grant-panel'
                  ? tr('All panels')
                  : tr('All tenants'))
              : '${o['label']}'),
          onChanged: (v) => setState(() =>
              v == true ? sel.add('${o['id']}') : sel.remove('${o['id']}')),
        ),
    ]);
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(tr('Give {name} a role', {'name': widget.user['name']})),
      content: SizedBox(
        width: 420,
        child: ListView(shrinkWrap: true, children: [
          DropdownButtonFormField<String>(
            key: const Key('grant-role'),
            initialValue: _role,
            decoration: InputDecoration(labelText: tr('Role')),
            items: [
              for (final r in widget.roles)
                DropdownMenuItem(
                    value: '${r['name']}', child: Text('${r['label']}')),
            ],
            onChanged: (v) => setState(() => _role = v),
          ),
          _checks(tr('Panels'), widget.panels, _panels, 'grant-panel'),
          _checks(tr('Tenants'), widget.tenants, _tenants, 'grant-tenant'),
          if (_error != null)
            Text(_error!,
                key: const Key('grant-error'),
                style: TextStyle(color: Theme.of(context).colorScheme.error)),
        ]),
      ),
      actions: [
        TextButton(
            onPressed: () => Navigator.pop(context), child: Text(tr('Cancel'))),
        FilledButton(
            key: const Key('grant-save'),
            onPressed: _panels.isEmpty || _tenants.isEmpty ? null : _save,
            child: Text(tr('Give role'))),
      ],
    );
  }
}
