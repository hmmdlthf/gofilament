import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'api.dart';
import 'responsive.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const FilamentApp());
}

class FilamentApp extends StatefulWidget {
  const FilamentApp({super.key});

  @override
  State<FilamentApp> createState() => _FilamentAppState();
}

class _FilamentAppState extends State<FilamentApp> {
  ThemeMode _themeMode = ThemeMode.dark;

  void toggleTheme() {
    setState(() {
      _themeMode = _themeMode == ThemeMode.dark ? ThemeMode.light : ThemeMode.dark;
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Pulse Admin',
      debugShowCheckedModeBanner: false,
      themeMode: _themeMode,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFFF59E0B), // Filament Amber
          brightness: Brightness.light,
        ),
        useMaterial3: true,
        fontFamily: 'sans-serif',
      ),
      darkTheme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFFF59E0B),
          brightness: Brightness.dark,
          surface: const Color(0xFF1E293B),
        ),
        scaffoldBackgroundColor: const Color(0xFF0F172A),
        useMaterial3: true,
        fontFamily: 'sans-serif',
      ),
      home: AdminShellPage(onToggleTheme: toggleTheme, isDark: _themeMode == ThemeMode.dark),
    );
  }
}

class AdminShellPage extends StatefulWidget {
  final VoidCallback onToggleTheme;
  final bool isDark;

  const AdminShellPage({super.key, required this.onToggleTheme, required this.isDark});

  @override
  State<AdminShellPage> createState() => _AdminShellPageState();
}

class _AdminShellPageState extends State<AdminShellPage> {
  final FilamentApiClient api = FilamentApiClient();
  Map<String, dynamic>? schema;
  bool isLoading = true;
  int _currentIndex = 0;
  String? selectedResourceSlug;

  String? errorMessage;

  @override
  void initState() {
    super.initState();
    loadSchema();
  }

  Future<void> loadSchema() async {
    setState(() {
      isLoading = true;
      errorMessage = null;
    });
    try {
      final s = await api.fetchSchema();
      setState(() {
        schema = s;
        if (s['active_tenant'] != null) {
          api.activeTenant = s['active_tenant'];
        }
        final resources = s['resources'] as List<dynamic>?;
        if (resources != null && resources.isNotEmpty) {
          selectedResourceSlug = resources.first['slug'];
        }
        isLoading = false;
      });
    } catch (e) {
      debugPrint('Failed to load schema from ${api.baseUrl}: $e');
      setState(() {
        errorMessage = e.toString();
        isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    if (isLoading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    if (schema == null) {
      final urlCtrl = TextEditingController(text: api.baseUrl);
      return Scaffold(
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 480),
              child: Card(
                elevation: 0,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(16),
                  side: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
                ),
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(Icons.cloud_off, size: 48, color: Colors.amber),
                      const SizedBox(height: 16),
                      const Text(
                        'Backend Connection Required',
                        style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                      ),
                      const SizedBox(height: 8),
                      Text(
                        'Could not connect to GoFilament backend at:\n${api.baseUrl}',
                        textAlign: TextAlign.center,
                        style: const TextStyle(fontSize: 13, color: Colors.grey),
                      ),
                      if (errorMessage != null) ...[
                        const SizedBox(height: 8),
                        Text(
                          errorMessage!,
                          textAlign: TextAlign.center,
                          style: const TextStyle(fontSize: 11, color: Colors.redAccent),
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ],
                      const SizedBox(height: 16),
                      TextField(
                        controller: urlCtrl,
                        decoration: const InputDecoration(
                          labelText: 'GoFilament API URL',
                          border: OutlineInputBorder(),
                          isDense: true,
                        ),
                      ),
                      const SizedBox(height: 16),
                      ElevatedButton.icon(
                        icon: const Icon(Icons.refresh),
                        label: const Text('Connect to Backend'),
                        onPressed: () {
                          api.baseUrl = urlCtrl.text.trim();
                          loadSchema();
                        },
                      ),
                    ],
                  ),
                ),
              ),
            ),
          ),
        ),
      );
    }

    final brandName = schema?['brand_name'] ?? 'Pulse';
    final resources = (schema?['resources'] as List<dynamic>?) ?? [];

    return ResponsiveLayout(
      // 📱 Mobile-First Layout: One-thumb bottom navigation, pull-to-refresh, cards
      mobile: Scaffold(
        appBar: AppBar(
          title: Text(brandName, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
          actions: [
            IconButton(
              icon: Icon(widget.isDark ? Icons.light_mode : Icons.dark_mode),
              onPressed: widget.onToggleTheme,
            ),
            IconButton(
              icon: const Icon(Icons.corporate_fare),
              onPressed: showTenantSwitcher,
            ),
          ],
        ),
        body: IndexedStack(
          index: _currentIndex,
          children: [
            _buildMobileDashboard(),
            _buildMobileResourceFeed(resources),
            _buildMobileSettings(),
          ],
        ),
        bottomNavigationBar: NavigationBar(
          selectedIndex: _currentIndex,
          onDestinationSelected: (idx) {
            HapticFeedback.selectionClick();
            setState(() => _currentIndex = idx);
          },
          destinations: const [
            NavigationDestination(icon: Icon(Icons.dashboard_outlined), selectedIcon: Icon(Icons.dashboard), label: 'Dashboard'),
            NavigationDestination(icon: Icon(Icons.layers_outlined), selectedIcon: Icon(Icons.layers), label: 'Entities'),
            NavigationDestination(icon: Icon(Icons.settings_outlined), selectedIcon: Icon(Icons.settings), label: 'Settings'),
          ],
        ),
      ),

      // 💻 Desktop & Tablet Layout: Navigation rail, data grid tables, topbar search
      desktop: Scaffold(
        body: Row(
          children: [
            NavigationRail(
              selectedIndex: _currentIndex,
              onDestinationSelected: (idx) => setState(() => _currentIndex = idx),
              labelType: NavigationRailLabelType.all,
              leading: Padding(
                padding: const EdgeInsets.symmetric(vertical: 16),
                child: Text('⚡ ' + brandName, style: const TextStyle(fontWeight: FontWeight.bold)),
              ),
              destinations: const [
                NavigationRailDestination(icon: Icon(Icons.dashboard_outlined), label: Text('Dashboard')),
                NavigationRailDestination(icon: Icon(Icons.layers_outlined), label: Text('Resources')),
                NavigationRailDestination(icon: Icon(Icons.settings_outlined), label: Text('Settings')),
              ],
            ),
            const VerticalDivider(thickness: 1, width: 1),
            Expanded(
              child: IndexedStack(
                index: _currentIndex,
                children: [
                  _buildDesktopDashboard(),
                  _buildDesktopResources(resources),
                  _buildMobileSettings(),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  // --- Mobile Dashboard (KPI Cards Carousel & Quick Actions) ---
  Widget _buildMobileDashboard() {
    return RefreshIndicator(
      onRefresh: loadSchema,
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const Text('Performance Overview', style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
          const SizedBox(height: 12),
          SizedBox(
            height: 130,
            child: ListView(
              scrollDirection: Axis.horizontal,
              children: [
                _kpiCard('Active Entities', '${schema?['resources']?.length ?? 0}', Colors.indigo, Icons.layers),
                const SizedBox(width: 12),
                _kpiCard('System Status', 'Operational', const Color(0xFF10B981), Icons.check_circle),
                const SizedBox(width: 12),
                _kpiCard('Multi-Tenancy', schema?['active_tenant'] ?? 'Global', Colors.amber, Icons.business),
              ],
            ),
          ),
          const SizedBox(height: 24),
          const Text('Quick Access Entities', style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
          const SizedBox(height: 12),
          for (final res in (schema?['resources'] as List<dynamic>? ?? []))
            Card(
              margin: const EdgeInsets.only(bottom: 8),
              child: ListTile(
                leading: CircleAvatar(
                  backgroundColor: Theme.of(context).colorScheme.primaryContainer,
                  child: const Icon(Icons.folder, size: 20),
                ),
                title: Text(res['model_name'] ?? res['slug']),
                subtitle: Text('/admin/api/resources/${res['slug']}'),
                trailing: const Icon(Icons.chevron_right),
                onTap: () {
                  setState(() {
                    selectedResourceSlug = res['slug'];
                    _currentIndex = 1;
                  });
                },
              ),
            ),
        ],
      ),
    );
  }

  Widget _kpiCard(String title, String val, Color color, IconData icon) {
    return Container(
      width: 160,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: Colors.white10),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Icon(icon, color: color, size: 24),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(val, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
              Text(title, style: const TextStyle(fontSize: 11, color: Colors.grey)),
            ],
          )
        ],
      ),
    );
  }

  // --- Mobile Resource Feed with Horizontal Resource Chips Selector ---
  Widget _buildMobileResourceFeed(List<dynamic> resources) {
    if (resources.isEmpty) {
      return const Center(child: Text('No resources available. Check server connection.'));
    }
    return Column(
      children: [
        // Horizontal scrollable resource pill selector
        Container(
          height: 52,
          padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 12),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surface,
            border: Border(bottom: BorderSide(color: Colors.white.withValues(alpha: 0.08))),
          ),
          child: ListView.separated(
            scrollDirection: Axis.horizontal,
            itemCount: resources.length,
            separatorBuilder: (_, __) => const SizedBox(width: 8),
            itemBuilder: (context, i) {
              final r = resources[i];
              final slug = r['slug'] as String;
              final name = (r['model_name'] ?? slug) as String;
              final isSel = slug == selectedResourceSlug;
              return ChoiceChip(
                label: Text(name),
                selected: isSel,
                onSelected: (val) {
                  if (val) setState(() => selectedResourceSlug = slug);
                },
              );
            },
          ),
        ),
        Expanded(
          child: selectedResourceSlug == null
              ? const Center(child: Text('Select a resource above'))
              : ResourceRecordList(api: api, slug: selectedResourceSlug!),
        ),
      ],
    );
  }

  // --- Desktop Views ---
  Widget _buildDesktopDashboard() {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text('Admin Dashboard', style: TextStyle(fontSize: 28, fontWeight: FontWeight.bold)),
          const SizedBox(height: 20),
          Row(
            children: [
              Expanded(child: _kpiCard('Entities', '${schema?['resources']?.length ?? 0}', Colors.indigo, Icons.layers)),
              const SizedBox(width: 16),
              Expanded(child: _kpiCard('Kernel Engine', 'GoLaravel', const Color(0xFF10B981), Icons.check_circle)),
              const SizedBox(width: 16),
              Expanded(child: _kpiCard('UI Mode', 'Mobile-First Flutter', Colors.amber, Icons.devices)),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildDesktopResources(List<dynamic> resources) {
    return Row(
      children: [
        SizedBox(
          width: 240,
          child: ListView.builder(
            itemCount: resources.length,
            itemBuilder: (context, i) {
              final r = resources[i];
              final isSel = r['slug'] == selectedResourceSlug;
              return ListTile(
                selected: isSel,
                leading: const Icon(Icons.table_chart),
                title: Text(r['model_name']),
                onTap: () => setState(() => selectedResourceSlug = r['slug']),
              );
            },
          ),
        ),
        const VerticalDivider(width: 1),
        Expanded(
          child: selectedResourceSlug == null
              ? const Center(child: Text('Select a resource'))
              : ResourceRecordList(api: api, slug: selectedResourceSlug!),
        ),
      ],
    );
  }

  Widget _buildMobileSettings() {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        const Text('Preferences', style: TextStyle(fontSize: 20, fontWeight: FontWeight.bold)),
        const SizedBox(height: 12),
        ListTile(
          leading: const Icon(Icons.brightness_6),
          title: const Text('Theme Mode'),
          trailing: Switch(
            value: widget.isDark,
            onChanged: (v) => widget.onToggleTheme(),
          ),
        ),
        ListTile(
          leading: const Icon(Icons.business),
          title: const Text('Tenant Organization'),
          subtitle: Text(api.activeTenant ?? 'Default'),
          onTap: showTenantSwitcher,
        ),
      ],
    );
  }

  void showTenantSwitcher() async {
    final tenants = await api.fetchTenants();
    if (!mounted) return;
    showModalBottomSheet(
      context: context,
      builder: (context) {
        return SafeArea(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const ListTile(title: Text('Select Workspace', style: TextStyle(fontWeight: FontWeight.bold))),
              for (final t in tenants)
                ListTile(
                  leading: CircleAvatar(child: Text((t['Name'] ?? 'T')[0])),
                  title: Text(t['Name'] ?? t['Key']),
                  trailing: t['Key'] == api.activeTenant ? const Icon(Icons.check, color: Colors.green) : null,
                  onTap: () {
                    setState(() => api.activeTenant = t['Key']);
                    Navigator.pop(context);
                    loadSchema();
                  },
                )
            ],
          ),
        );
      },
    );
  }
}

class ResourceRecordList extends StatefulWidget {
  final FilamentApiClient api;
  final String slug;

  const ResourceRecordList({super.key, required this.api, required this.slug});

  @override
  State<ResourceRecordList> createState() => _ResourceRecordListState();
}

class _ResourceRecordListState extends State<ResourceRecordList> {
  List<dynamic> records = [];
  bool isLoading = true;
  String search = '';

  @override
  void initState() {
    super.initState();
    loadRecords();
  }

  @override
  void didUpdateWidget(covariant ResourceRecordList oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.slug != widget.slug) {
      loadRecords();
    }
  }

  Future<void> loadRecords() async {
    setState(() => isLoading = true);
    try {
      final res = await widget.api.fetchResourceRecords(widget.slug, search: search);
      setState(() {
        records = (res['data'] as List<dynamic>?) ?? [];
        isLoading = false;
      });
    } catch (e) {
      setState(() => isLoading = false);
    }
  }

  Widget _buildFieldItem(String label, String value, {IconData? icon, Color? valueColor}) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label.toUpperCase(),
          style: TextStyle(
            fontSize: 10,
            fontWeight: FontWeight.w600,
            color: Colors.grey.shade400,
            letterSpacing: 0.5,
          ),
        ),
        const SizedBox(height: 2),
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (icon != null) ...[
              Icon(icon, size: 14, color: valueColor ?? Colors.grey.shade300),
              const SizedBox(width: 4),
            ],
            Flexible(
              child: Text(
                value,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                  color: valueColor,
                ),
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildRecordCard(BuildContext context, dynamic item, String id) {
    final isDevice = widget.slug == 'devices';
    final isTelemetry = widget.slug == 'telemetry';
    final isTicket = widget.slug == 'tickets';
    final isFirmware = widget.slug == 'firmware';

    String name = 'Record #$id';
    if (isDevice) {
      name = (item['Name'] ?? item['name'] ?? 'Device #$id').toString();
    } else if (isTelemetry) {
      name = '${item['DeviceName'] ?? item['device_name'] ?? 'Node'}: ${item['Metric'] ?? item['metric'] ?? 'Metric'}';
    } else if (isTicket) {
      name = (item['Title'] ?? item['title'] ?? 'Ticket #$id').toString();
    } else if (isFirmware) {
      name = '${item['Version'] ?? item['version'] ?? 'Release'} (${item['HardwareTarget'] ?? item['hardware_target'] ?? 'All'})';
    } else {
      name = (item['Name'] ?? item['name'] ?? item['Title'] ?? item['title'] ?? 'Operator #$id').toString();
    }

    IconData headerIcon = Icons.person;
    if (isDevice) headerIcon = Icons.router;
    if (isTelemetry) headerIcon = Icons.sensors;
    if (isTicket) headerIcon = Icons.confirmation_number_outlined;
    if (isFirmware) headerIcon = Icons.system_update_alt;

    final status = (item['Status'] ?? item['status'] ?? 'online').toString().toLowerCase();
    Color statusColor = const Color(0xFF10B981);
    if (status == 'offline' || status == 'critical') statusColor = Colors.redAccent;
    if (status == 'maintenance' || status == 'warning' || status == 'in_progress') statusColor = Colors.amber;

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: Colors.white.withValues(alpha: 0.08)),
      ),
      color: Theme.of(context).colorScheme.surface,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header: Icon + Title + Status/Badge
            Row(
              children: [
                CircleAvatar(
                  radius: 18,
                  backgroundColor: Theme.of(context).colorScheme.primaryContainer,
                  child: Icon(headerIcon, size: 20, color: Theme.of(context).colorScheme.primary),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        name,
                        style: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold),
                        overflow: TextOverflow.ellipsis,
                      ),
                      Text(
                        'ID: #$id',
                        style: TextStyle(fontSize: 11, color: Colors.grey.shade400),
                      ),
                    ],
                  ),
                ),
                if (isDevice)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                    decoration: BoxDecoration(
                      color: statusColor.withValues(alpha: 0.15),
                      borderRadius: BorderRadius.circular(999),
                      border: Border.all(color: statusColor.withValues(alpha: 0.3)),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Container(width: 6, height: 6, decoration: BoxDecoration(color: statusColor, shape: BoxShape.circle)),
                        const SizedBox(width: 6),
                        Text(status.toUpperCase(), style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: statusColor)),
                      ],
                    ),
                  )
                else if (isTelemetry)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                    decoration: BoxDecoration(
                      color: (item['Severity'] == 'critical' ? Colors.redAccent : (item['Severity'] == 'warning' ? Colors.amber : const Color(0xFF10B981))).withValues(alpha: 0.15),
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      '${item['Severity'] ?? item['severity'] ?? 'normal'}'.toUpperCase(),
                      style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: item['Severity'] == 'critical' ? Colors.redAccent : (item['Severity'] == 'warning' ? Colors.amber : const Color(0xFF10B981))),
                    ),
                  )
                else if (isTicket)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                    decoration: BoxDecoration(
                      color: Colors.redAccent.withValues(alpha: 0.15),
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      '${item['Priority'] ?? item['priority'] ?? 'medium'}'.toUpperCase(),
                      style: const TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: Colors.redAccent),
                    ),
                  )
                else if (isFirmware)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                    decoration: BoxDecoration(
                      color: Colors.blueAccent.withValues(alpha: 0.15),
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      '${item['Channel'] ?? item['channel'] ?? 'stable'}'.toUpperCase(),
                      style: const TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: Colors.blueAccent),
                    ),
                  )
                else if (item['Role'] != null || item['role'] != null)
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                    decoration: BoxDecoration(
                      color: Colors.amber.withValues(alpha: 0.15),
                      borderRadius: BorderRadius.circular(999),
                    ),
                    child: Text(
                      '${item['Role'] ?? item['role']}',
                      style: const TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: Colors.amber),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 12),
            const Divider(height: 1, thickness: 0.5),
            const SizedBox(height: 12),

            // Columns Grid: Contextual attributes for each entity
            if (isDevice) ...[
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Serial Number', '${item['SerialNo'] ?? item['serial_no'] ?? 'N/A'}', icon: Icons.tag)),
                  Expanded(child: _buildFieldItem('Battery', '${item['BatteryLevel'] ?? item['battery_level'] ?? 100}%', icon: Icons.battery_charging_full, valueColor: statusColor)),
                ],
              ),
              const SizedBox(height: 10),
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Location', '${item['Location'] ?? item['location'] ?? 'Unknown'}', icon: Icons.location_on_outlined)),
                  Expanded(child: _buildFieldItem('Model', '${item['Model'] ?? item['model'] ?? 'EdgeCore-Pro'}', icon: Icons.memory)),
                ],
              ),
            ] else if (isTelemetry) ...[
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Sensor Metric', '${item['Metric'] ?? item['metric'] ?? 'Temperature'}', icon: Icons.sensors)),
                  Expanded(child: _buildFieldItem('Reading Value', '${item['Value'] ?? item['value'] ?? 'N/A'}', icon: Icons.speed, valueColor: const Color(0xFF10B981))),
                ],
              ),
              const SizedBox(height: 10),
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Device Source', '${item['DeviceName'] ?? item['device_name'] ?? 'Gateway'}', icon: Icons.router)),
                  Expanded(child: _buildFieldItem('Severity Alert', '${item['Severity'] ?? item['severity'] ?? 'normal'}', icon: Icons.notification_important_outlined)),
                ],
              ),
            ] else if (isTicket) ...[
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Assigned Tech', '${item['AssignedTo'] ?? item['assigned_to'] ?? 'Technician'}', icon: Icons.person_outline)),
                  Expanded(child: _buildFieldItem('Ticket Status', '${item['Status'] ?? item['status'] ?? 'open'}', icon: Icons.track_changes, valueColor: Colors.amber)),
                ],
              ),
              const SizedBox(height: 10),
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Department', '${item['Department'] ?? item['department'] ?? 'Hardware'}', icon: Icons.business)),
                  Expanded(child: _buildFieldItem('Priority Level', '${item['Priority'] ?? item['priority'] ?? 'high'}', icon: Icons.flag_outlined)),
                ],
              ),
            ] else if (isFirmware) ...[
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Target Model', '${item['HardwareTarget'] ?? item['hardware_target'] ?? 'All Nodes'}', icon: Icons.developer_board)),
                  Expanded(child: _buildFieldItem('Rollout Rate', '${item['RolloutPct'] ?? item['rollout_pct'] ?? 100}%', icon: Icons.pie_chart_outline, valueColor: Colors.blueAccent)),
                ],
              ),
              const SizedBox(height: 10),
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Release Channel', '${item['Channel'] ?? item['channel'] ?? 'stable'}', icon: Icons.alt_route)),
                  Expanded(child: _buildFieldItem('Active Binary', '${item['IsActive'] ?? item['is_active'] ?? true ? "Yes" : "Archived"}', icon: Icons.check_circle_outline)),
                ],
              ),
            ] else ...[
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Email Address', '${item['Email'] ?? item['email'] ?? 'N/A'}', icon: Icons.email_outlined)),
                  Expanded(child: _buildFieldItem('Department', '${item['Department'] ?? item['department'] ?? 'General'}', icon: Icons.business)),
                ],
              ),
            ],

            const SizedBox(height: 12),
            const Divider(height: 1, thickness: 0.5),
            const SizedBox(height: 8),

            // Footer Actions Row
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton.icon(
                  icon: const Icon(Icons.edit_outlined, size: 15),
                  label: const Text('Edit', style: TextStyle(fontSize: 12)),
                  onPressed: () => showEditSheet(context, id, item),
                ),
                const SizedBox(width: 8),
                TextButton.icon(
                  icon: const Icon(Icons.delete_outline, size: 15, color: Colors.redAccent),
                  label: const Text('Delete', style: TextStyle(fontSize: 12, color: Colors.redAccent)),
                  onPressed: () async {
                    final confirm = await showDialog<bool>(
                      context: context,
                      builder: (ctx) => AlertDialog(
                        title: const Text('Delete Record?'),
                        content: Text('Are you sure you want to delete #$id?'),
                        actions: [
                          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
                          TextButton(
                            onPressed: () => Navigator.pop(ctx, true),
                            child: const Text('Delete', style: TextStyle(color: Colors.redAccent)),
                          ),
                        ],
                      ),
                    );
                    if (confirm == true) {
                      await widget.api.deleteRecord(widget.slug, id);
                      loadRecords();
                    }
                  },
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.slug.toUpperCase(), style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(48),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
            child: TextField(
              decoration: InputDecoration(
                hintText: 'Search ${widget.slug}...',
                prefixIcon: const Icon(Icons.search),
                isDense: true,
                border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
              ),
              onChanged: (v) {
                search = v;
                loadRecords();
              },
            ),
          ),
        ),
      ),
      body: isLoading
          ? const Center(child: CircularProgressIndicator())
          : records.isEmpty
              ? const Center(child: Text('No records found.'))
              : RefreshIndicator(
                  onRefresh: loadRecords,
                  child: LayoutBuilder(
                    builder: (context, constraints) {
                      // Tablet & Desktop: 2-column card grid (Zero horizontal scrolling)
                      final isTabletOrWide = constraints.maxWidth >= 600;
                      if (isTabletOrWide) {
                        return GridView.builder(
                          padding: const EdgeInsets.all(16),
                          gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                            crossAxisCount: 2,
                            mainAxisExtent: 260,
                            crossAxisSpacing: 16,
                            mainAxisSpacing: 16,
                          ),
                          itemCount: records.length,
                          itemBuilder: (context, i) {
                            final item = records[i];
                            final id = (item['ID'] ?? item['id'] ?? '$i').toString();
                            return _buildRecordCard(context, item, id);
                          },
                        );
                      }

                      // Mobile: 1-column stacked cards (Zero horizontal scrolling)
                      return ListView.builder(
                        padding: const EdgeInsets.all(12),
                        itemCount: records.length,
                        itemBuilder: (context, i) {
                          final item = records[i];
                          final id = (item['ID'] ?? item['id'] ?? '$i').toString();
                          return Padding(
                            padding: const EdgeInsets.only(bottom: 12),
                            child: _buildRecordCard(context, item, id),
                          );
                        },
                      );
                    },
                  ),
                ),
      floatingActionButton: FloatingActionButton(
        onPressed: () => showCreateSheet(context),
        child: const Icon(Icons.add),
      ),
    );
  }

  void showCreateSheet(BuildContext context) {
    final isDevice = widget.slug == 'devices';
    final isTelemetry = widget.slug == 'telemetry';
    final isTicket = widget.slug == 'tickets';
    final isFirmware = widget.slug == 'firmware';

    final field1Ctrl = TextEditingController(
      text: isDevice ? 'Edge-Node-${DateTime.now().millisecondsSinceEpoch % 1000}' :
            isTelemetry ? 'Gateway Alpha-101' :
            isTicket ? 'Inspect sensor connection on tower' :
            isFirmware ? 'v3.3.0' : 'Alex Mercer'
    );
    final field2Ctrl = TextEditingController(
      text: isDevice ? 'SN-${DateTime.now().millisecondsSinceEpoch % 10000}-X' :
            isTelemetry ? 'Core Temperature' :
            isTicket ? 'Marcus Vance' :
            isFirmware ? 'PulseEdge v4 Pro' : 'alex.m@pulsefleet.io'
    );
    final field3Ctrl = TextEditingController(
      text: isDevice ? 'Frankfurt DC, Rack 4' :
            isTelemetry ? '38.5 °C' :
            isTicket ? 'Hardware Engineering' :
            isFirmware ? '100' : 'Core Operations'
    );
    String selectVal = isDevice ? 'online' : (isTelemetry ? 'normal' : (isTicket ? 'critical' : 'stable'));

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setSheetState) => Padding(
          padding: EdgeInsets.only(left: 20, right: 20, top: 20, bottom: MediaQuery.of(ctx).viewInsets.bottom + 20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(
                'Add New ${isDevice ? "IoT Device" : (isTelemetry ? "Telemetry Log" : (isTicket ? "Work Order Ticket" : (isFirmware ? "Firmware Release" : "Team Member")))}',
                style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
              ),
              const SizedBox(height: 16),
              TextField(
                controller: field1Ctrl,
                decoration: InputDecoration(
                  labelText: isDevice ? 'Device Name' : (isTelemetry ? 'Device Name' : (isTicket ? 'Ticket Title' : (isFirmware ? 'Firmware Version' : 'Full Name'))),
                  border: const OutlineInputBorder(),
                  isDense: true,
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: field2Ctrl,
                decoration: InputDecoration(
                  labelText: isDevice ? 'Serial Number' : (isTelemetry ? 'Sensor Metric' : (isTicket ? 'Assigned Technician' : (isFirmware ? 'Target Hardware' : 'Email Address'))),
                  border: const OutlineInputBorder(),
                  isDense: true,
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: field3Ctrl,
                decoration: InputDecoration(
                  labelText: isDevice ? 'Deployment Location' : (isTelemetry ? 'Reading Value' : (isTicket ? 'Department' : (isFirmware ? 'Rollout %' : 'Department'))),
                  border: const OutlineInputBorder(),
                  isDense: true,
                ),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: selectVal,
                decoration: InputDecoration(
                  labelText: isDevice ? 'Initial Status' : (isTelemetry ? 'Severity' : (isTicket ? 'Priority' : (isFirmware ? 'Channel' : 'Access Role'))),
                  border: const OutlineInputBorder(),
                  isDense: true,
                ),
                items: isDevice ? const [
                  DropdownMenuItem(value: 'online', child: Text('Online (Active)')),
                  DropdownMenuItem(value: 'offline', child: Text('Offline')),
                  DropdownMenuItem(value: 'maintenance', child: Text('Under Maintenance')),
                ] : (isTelemetry ? const [
                  DropdownMenuItem(value: 'normal', child: Text('Normal')),
                  DropdownMenuItem(value: 'warning', child: Text('Warning')),
                  DropdownMenuItem(value: 'critical', child: Text('Critical Alert')),
                ] : (isTicket ? const [
                  DropdownMenuItem(value: 'critical', child: Text('Critical')),
                  DropdownMenuItem(value: 'high', child: Text('High')),
                  DropdownMenuItem(value: 'medium', child: Text('Medium')),
                  DropdownMenuItem(value: 'low', child: Text('Low')),
                ] : (isFirmware ? const [
                  DropdownMenuItem(value: 'stable', child: Text('Stable (Production)')),
                  DropdownMenuItem(value: 'beta', child: Text('Beta Channel')),
                  DropdownMenuItem(value: 'edge', child: Text('Edge / Nightly')),
                ] : const [
                  DropdownMenuItem(value: 'Operations Lead', child: Text('Operations Lead')),
                  DropdownMenuItem(value: 'Firmware Engineer', child: Text('Firmware Engineer')),
                  DropdownMenuItem(value: 'Site Technician', child: Text('Site Technician')),
                ]))),
                onChanged: (val) {
                  if (val != null) setSheetState(() => selectVal = val);
                },
              ),
              const SizedBox(height: 20),
              ElevatedButton.icon(
                icon: const Icon(Icons.check),
                label: const Text('Save Record'),
                onPressed: () async {
                  if (field1Ctrl.text.isNotEmpty) {
                    final data = <String, dynamic>{
                      'name': field1Ctrl.text,
                      'title': field1Ctrl.text,
                      if (isDevice) ...{
                        'serial_no': field2Ctrl.text,
                        'location': field3Ctrl.text,
                        'status': selectVal,
                        'battery_level': 100,
                      } else if (isTelemetry) ...{
                        'device_name': field1Ctrl.text,
                        'metric': field2Ctrl.text,
                        'value': field3Ctrl.text,
                        'severity': selectVal,
                      } else if (isTicket) ...{
                        'title': field1Ctrl.text,
                        'assigned_to': field2Ctrl.text,
                        'department': field3Ctrl.text,
                        'priority': selectVal,
                        'status': 'open',
                      } else if (isFirmware) ...{
                        'version': field1Ctrl.text,
                        'hardware_target': field2Ctrl.text,
                        'rollout_pct': int.tryParse(field3Ctrl.text) ?? 100,
                        'channel': selectVal,
                      } else ...{
                        'email': field2Ctrl.text,
                        'department': field3Ctrl.text,
                        'role': selectVal,
                      },
                    };
                    await widget.api.createRecord(widget.slug, data);
                    if (!mounted) return;
                    Navigator.pop(ctx);
                    loadRecords();
                  }
                },
              ),
            ],
          ),
        ),
      ),
    );
  }

  void showEditSheet(BuildContext context, String id, dynamic item) {
    final titleCtrl = TextEditingController(text: item['Name'] ?? item['Title'] ?? item['name'] ?? '');
    final locCtrl = TextEditingController(text: item['Location'] ?? item['location'] ?? item['Department'] ?? item['department'] ?? '');
    String status = item['Status'] ?? item['status'] ?? 'online';

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setSheetState) => Padding(
          padding: EdgeInsets.only(left: 20, right: 20, top: 20, bottom: MediaQuery.of(ctx).viewInsets.bottom + 20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text('Edit ${widget.slug} #$id', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
              const SizedBox(height: 16),
              TextField(
                controller: titleCtrl,
                decoration: const InputDecoration(labelText: 'Name', border: OutlineInputBorder(), isDense: true),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: locCtrl,
                decoration: InputDecoration(
                  labelText: widget.slug == 'devices' ? 'Location' : 'Department',
                  border: const OutlineInputBorder(),
                  isDense: true,
                ),
              ),
              if (widget.slug == 'devices') ...[
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  initialValue: ['online', 'offline', 'maintenance'].contains(status) ? status : 'online',
                  decoration: const InputDecoration(labelText: 'Status', border: OutlineInputBorder(), isDense: true),
                  items: const [
                    DropdownMenuItem(value: 'online', child: Text('Online (Active)')),
                    DropdownMenuItem(value: 'offline', child: Text('Offline')),
                    DropdownMenuItem(value: 'maintenance', child: Text('Under Maintenance')),
                  ],
                  onChanged: (val) {
                    if (val != null) setSheetState(() => status = val);
                  },
                ),
              ],
              const SizedBox(height: 20),
              ElevatedButton.icon(
                icon: const Icon(Icons.save),
                label: const Text('Update Changes'),
                onPressed: () async {
                  final data = <String, dynamic>{
                    'name': titleCtrl.text,
                    if (widget.slug == 'devices') ...{
                      'location': locCtrl.text,
                      'status': status,
                    } else ...{
                      'department': locCtrl.text,
                    },
                  };
                  await widget.api.updateRecord(widget.slug, id, data);
                  if (!mounted) return;
                  Navigator.pop(ctx);
                  loadRecords();
                },
              ),
            ],
          ),
        ),
      ),
    );
  }
}
