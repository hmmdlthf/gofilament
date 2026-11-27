import 'package:flutter/material.dart';

import 'api.dart';
import 'i18n.dart';
import 'tenant_settings_screen.dart';

/// The central panel's tenant features screen (Panel.TenantFeaturesAdmin on
/// the server): pick a tenant, then change any of its features, including
/// those only the platform may change.
class TenantFeaturesAdminScreen extends StatefulWidget {
  const TenantFeaturesAdminScreen(
      {super.key, required this.api, required this.path, this.title});

  final AdminApi api;

  /// The endpoint relative to the panel API (schema
  /// tenant_features_admin.path).
  final String path;
  final String? title;

  @override
  State<TenantFeaturesAdminScreen> createState() =>
      _TenantFeaturesAdminScreenState();
}

class _TenantFeaturesAdminScreenState extends State<TenantFeaturesAdminScreen> {
  List<Map<String, dynamic>> _tenants = const [];
  bool _freeEntry = false;
  bool _loading = true;
  String? _error;
  final _key = TextEditingController();

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _key.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final d = await widget.api.tenantFeatureTenants(widget.path);
      if (!mounted) return;
      final t = d['tenants'];
      setState(() {
        _tenants = t is List
            ? [
                for (final x in t)
                  if (x is Map) Map<String, dynamic>.from(x)
              ]
            : const [];
        _freeEntry = d['free_entry'] == true;
        _loading = false;
      });
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _loading = false;
          _error = e.message;
        });
      }
    }
  }

  void _open(String key, String name) {
    if (key.isEmpty) return;
    Navigator.of(context).push(MaterialPageRoute<void>(
        builder: (_) => TenantSettingsScreen(
            api: widget.api,
            path: '${widget.path}/${Uri.encodeComponent(key)}',
            title: name)));
  }

  @override
  Widget build(BuildContext context) {
    Widget body;
    if (_loading) {
      body = const Center(child: CircularProgressIndicator());
    } else if (_error != null) {
      body = Center(child: Text(_error!));
    } else {
      body = ListView(
        padding: const EdgeInsets.all(16),
        children: [
          if (_freeEntry)
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Row(children: [
                  Expanded(
                    child: TextField(
                      key: const Key('tenant-features-key'),
                      controller: _key,
                      decoration: InputDecoration(labelText: tr('Tenant key')),
                      onSubmitted: (v) => _open(v.trim(), v.trim()),
                    ),
                  ),
                  const SizedBox(width: 12),
                  FilledButton(
                    key: const Key('tenant-features-open'),
                    onPressed: () => _open(_key.text.trim(), _key.text.trim()),
                    child: Text(tr('Open')),
                  ),
                ]),
              ),
            )
          else if (_tenants.isEmpty)
            Center(child: Text(tr('No tenants.')))
          else
            Card(
              child: Column(children: [
                for (final t in _tenants)
                  ListTile(
                    key: Key('tenant-features-${t['key']}'),
                    leading: const Icon(Icons.apartment_outlined),
                    title: Text('${t['name'] ?? t['key']}'),
                    subtitle: Text('${t['key']}'),
                    trailing: const Icon(Icons.chevron_right),
                    onTap: () =>
                        _open('${t['key']}', '${t['name'] ?? t['key']}'),
                  ),
              ]),
            ),
        ],
      );
    }
    return Scaffold(
      appBar: AppBar(title: Text(widget.title ?? tr('Tenant key'))),
      body: body,
    );
  }
}
