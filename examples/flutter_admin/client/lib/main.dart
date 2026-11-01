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
            Padding(
              padding: const EdgeInsets.only(right: 8, left: 4),
              child: CircularAccountButton(
                initial: 'S',
                name: 'Sarah Connor',
                email: 'sarah@fleetpulse.io',
                role: 'Operations Lead',
                onTap: () => AccountModalSheet.show(
                  context: context,
                  api: api,
                  activeTenant: api.activeTenant,
                  onThemeToggle: widget.onToggleTheme,
                  isDark: widget.isDark,
                ),
              ),
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
            NavigationDestination(icon: Icon(Icons.account_circle_outlined), selectedIcon: Icon(Icons.account_circle), label: 'Account'),
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
              trailing: Expanded(
                child: Align(
                  alignment: Alignment.bottomCenter,
                  child: Padding(
                    padding: const EdgeInsets.only(bottom: 20),
                    child: CircularAccountButton(
                      initial: 'S',
                      name: 'Sarah Connor',
                      email: 'sarah@fleetpulse.io',
                      role: 'Operations Lead',
                      onTap: () => AccountModalSheet.show(
                        context: context,
                        api: api,
                        activeTenant: api.activeTenant,
                        onThemeToggle: widget.onToggleTheme,
                        isDark: widget.isDark,
                      ),
                    ),
                  ),
                ),
              ),
              destinations: const [
                NavigationRailDestination(icon: Icon(Icons.dashboard_outlined), label: Text('Dashboard')),
                NavigationRailDestination(icon: Icon(Icons.layers_outlined), label: Text('Resources')),
                NavigationRailDestination(icon: Icon(Icons.account_circle_outlined), label: Text('Account')),
              ],
            ),
            const VerticalDivider(thickness: 1, width: 1),
            Expanded(
              child: Column(
                children: [
                  // Desktop Header Bar
                  Container(
                    height: 60,
                    padding: const EdgeInsets.symmetric(horizontal: 24),
                    decoration: BoxDecoration(
                      color: Theme.of(context).colorScheme.surface,
                      border: Border(bottom: BorderSide(color: Colors.white.withValues(alpha: 0.08))),
                    ),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Row(
                          children: [
                            Text(
                              brandName + ' Control Center',
                              style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16),
                            ),
                            const SizedBox(width: 12),
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                              decoration: BoxDecoration(
                                color: Colors.amber.withValues(alpha: 0.15),
                                borderRadius: BorderRadius.circular(6),
                              ),
                              child: Text(
                                api.activeTenant ?? 'GLOBAL TENANT',
                                style: const TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: Colors.amber),
                              ),
                            ),
                          ],
                        ),
                        Row(
                          children: [
                            IconButton(
                              icon: Icon(widget.isDark ? Icons.light_mode : Icons.dark_mode),
                              onPressed: widget.onToggleTheme,
                            ),
                            IconButton(
                              icon: const Icon(Icons.corporate_fare),
                              tooltip: 'Switch Tenant',
                              onPressed: showTenantSwitcher,
                            ),
                            const SizedBox(width: 8),
                            CircularAccountButton(
                              initial: 'S',
                              name: 'Sarah Connor',
                              email: 'sarah@fleetpulse.io',
                              role: 'Operations Lead',
                              onTap: () => AccountModalSheet.show(
                                context: context,
                                api: api,
                                activeTenant: api.activeTenant,
                                onThemeToggle: widget.onToggleTheme,
                                isDark: widget.isDark,
                              ),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
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
        // User Profile Summary Card
        Card(
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(16),
            side: BorderSide(color: Colors.amber.withValues(alpha: 0.3)),
          ),
          child: InkWell(
            borderRadius: BorderRadius.circular(16),
            onTap: () => AccountModalSheet.show(
              context: context,
              api: api,
              activeTenant: api.activeTenant,
              onThemeToggle: widget.onToggleTheme,
              isDark: widget.isDark,
            ),
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                children: [
                  Container(
                    width: 52,
                    height: 52,
                    decoration: BoxDecoration(
                      shape: BoxShape.circle,
                      gradient: const LinearGradient(
                        colors: [Color(0xFFF59E0B), Color(0xFFD97706)],
                      ),
                      border: Border.all(color: Colors.amber.withValues(alpha: 0.4), width: 2),
                    ),
                    child: const Center(
                      child: Text('S', style: TextStyle(color: Colors.black, fontWeight: FontWeight.bold, fontSize: 22)),
                    ),
                  ),
                  const SizedBox(width: 14),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text(
                          'Sarah Connor',
                          style: TextStyle(fontSize: 17, fontWeight: FontWeight.bold),
                        ),
                        const SizedBox(height: 2),
                        const Text(
                          'sarah@fleetpulse.io • Operations Lead',
                          style: TextStyle(fontSize: 12, color: Colors.grey),
                        ),
                        const SizedBox(height: 6),
                        Row(
                          children: [
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                              decoration: BoxDecoration(
                                color: const Color(0xFF10B981).withValues(alpha: 0.15),
                                borderRadius: BorderRadius.circular(12),
                              ),
                              child: const Text(
                                'PRO FLEET TIER',
                                style: TextStyle(fontSize: 9, fontWeight: FontWeight.bold, color: Color(0xFF10B981)),
                              ),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                  const Icon(Icons.chevron_right, color: Colors.grey),
                ],
              ),
            ),
          ),
        ),

        const SizedBox(height: 16),
        const Text('Account Hub', style: TextStyle(fontSize: 13, fontWeight: FontWeight.bold, letterSpacing: 0.6, color: Colors.grey)),
        const SizedBox(height: 8),

        // Quick Hub Grid
        Row(
          children: [
            Expanded(
              child: _buildAccountHubCard(
                icon: Icons.settings_outlined,
                iconColor: Colors.amber,
                title: 'Settings',
                subtitle: 'Theme & Bio',
                onTap: () => Navigator.push(
                  context,
                  MaterialPageRoute(builder: (_) => SettingsScreen(onThemeToggle: widget.onToggleTheme, isDark: widget.isDark, api: api)),
                ),
              ),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: _buildAccountHubCard(
                icon: Icons.shopping_bag_outlined,
                iconColor: Colors.blueAccent,
                title: 'Orders',
                subtitle: 'Procurement',
                onTap: () => Navigator.push(
                  context,
                  MaterialPageRoute(builder: (_) => const OrdersScreen()),
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 10),
        Row(
          children: [
            Expanded(
              child: _buildAccountHubCard(
                icon: Icons.credit_card_outlined,
                iconColor: Colors.tealAccent,
                title: 'Billing',
                subtitle: 'Fleet License',
                onTap: () => Navigator.push(
                  context,
                  MaterialPageRoute(builder: (_) => const BillingScreen()),
                ),
              ),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: _buildAccountHubCard(
                icon: Icons.fingerprint_rounded,
                iconColor: const Color(0xFF10B981),
                title: 'Biometrics',
                subtitle: 'Security Test',
                onTap: () => Navigator.push(
                  context,
                  MaterialPageRoute(builder: (_) => const SecuritySettingsScreen()),
                ),
              ),
            ),
          ],
        ),

        const SizedBox(height: 24),
        const Text('Quick Preferences', style: TextStyle(fontSize: 13, fontWeight: FontWeight.bold, letterSpacing: 0.6, color: Colors.grey)),
        const SizedBox(height: 8),

        Card(
          child: Column(
            children: [
              ListTile(
                leading: const Icon(Icons.brightness_6),
                title: const Text('Theme Mode'),
                subtitle: Text(widget.isDark ? 'Dark theme active' : 'Light theme active', style: const TextStyle(fontSize: 12)),
                trailing: Switch(
                  value: widget.isDark,
                  onChanged: (v) => widget.onToggleTheme(),
                ),
              ),
              const Divider(height: 1),
              ListTile(
                leading: const Icon(Icons.corporate_fare),
                title: const Text('Tenant Organization'),
                subtitle: Text(api.activeTenant ?? 'Default Workspace', style: const TextStyle(fontSize: 12)),
                trailing: const Icon(Icons.chevron_right),
                onTap: showTenantSwitcher,
              ),
            ],
          ),
        ),

        const SizedBox(height: 24),
        OutlinedButton.icon(
          icon: const Icon(Icons.logout_rounded, color: Colors.redAccent, size: 18),
          label: const Text('Sign Out', style: TextStyle(color: Colors.redAccent, fontWeight: FontWeight.bold)),
          style: OutlinedButton.styleFrom(
            side: const BorderSide(color: Colors.redAccent, width: 1.2),
            padding: const EdgeInsets.symmetric(vertical: 12),
            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
          ),
          onPressed: () {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('Operator session closed.'), behavior: SnackBarBehavior.floating),
            );
          },
        ),
      ],
    );
  }

  Widget _buildAccountHubCard({
    required IconData icon,
    required Color iconColor,
    required String title,
    required String subtitle,
    required VoidCallback onTap,
  }) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(14),
        side: BorderSide(color: Colors.white.withValues(alpha: 0.1)),
      ),
      child: InkWell(
        borderRadius: BorderRadius.circular(14),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Row(
            children: [
              Container(
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: iconColor.withValues(alpha: 0.15),
                  shape: BoxShape.circle,
                ),
                child: Icon(icon, color: iconColor, size: 20),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(title, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14)),
                    const SizedBox(height: 2),
                    Text(subtitle, style: const TextStyle(fontSize: 11, color: Colors.grey)),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
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

// ============================================================================
// Circular Account Button
// ============================================================================
class CircularAccountButton extends StatelessWidget {
  final String initial;
  final String name;
  final String email;
  final String role;
  final VoidCallback onTap;
  final double radius;

  const CircularAccountButton({
    super.key,
    required this.initial,
    required this.name,
    required this.email,
    required this.role,
    required this.onTap,
    this.radius = 18,
  });

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: '$name ($role)',
      child: Material(
        color: Colors.transparent,
        shape: const CircleBorder(),
        child: InkWell(
          customBorder: const CircleBorder(),
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.all(4),
            child: Stack(
              clipBehavior: Clip.none,
              alignment: Alignment.bottomRight,
              children: [
                Container(
                  width: radius * 2,
                  height: radius * 2,
                  decoration: BoxDecoration(
                    shape: BoxShape.circle,
                    gradient: const LinearGradient(
                      colors: [Color(0xFFF59E0B), Color(0xFFD97706)],
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                    ),
                    boxShadow: [
                      BoxShadow(
                        color: Colors.amber.withValues(alpha: 0.3),
                        blurRadius: 6,
                        offset: const Offset(0, 2),
                      ),
                    ],
                    border: Border.all(
                      color: Colors.amber.withValues(alpha: 0.6),
                      width: 1.5,
                    ),
                  ),
                  child: Center(
                    child: Text(
                      initial,
                      style: TextStyle(
                        color: Colors.black,
                        fontWeight: FontWeight.bold,
                        fontSize: radius * 0.9,
                      ),
                    ),
                  ),
                ),
                Positioned(
                  bottom: -1,
                  right: -1,
                  child: Container(
                    width: radius * 0.55,
                    height: radius * 0.55,
                    decoration: BoxDecoration(
                      color: const Color(0xFF10B981),
                      shape: BoxShape.circle,
                      border: Border.all(
                        color: const Color(0xFF0F172A),
                        width: 2,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

// ============================================================================
// Floating Account Modal Sheet
// ============================================================================
class AccountModalSheet extends StatelessWidget {
  final FilamentApiClient api;
  final String? activeTenant;
  final VoidCallback onThemeToggle;
  final bool isDark;

  const AccountModalSheet({
    super.key,
    required this.api,
    required this.activeTenant,
    required this.onThemeToggle,
    required this.isDark,
  });

  static Future<void> show({
    required BuildContext context,
    required FilamentApiClient api,
    required String? activeTenant,
    required VoidCallback onThemeToggle,
    required bool isDark,
  }) {
    return showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (ctx) => AccountModalSheet(
        api: api,
        activeTenant: activeTenant,
        onThemeToggle: onThemeToggle,
        isDark: isDark,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
        borderRadius: const BorderRadius.vertical(top: Radius.circular(28)),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.35),
            blurRadius: 30,
            offset: const Offset(0, -6),
          ),
        ],
      ),
      child: SafeArea(
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.of(context).size.height * 0.88,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              // Drag Handle
              Center(
                child: Container(
                  margin: const EdgeInsets.only(top: 10, bottom: 6),
                  width: 38,
                  height: 4,
                  decoration: BoxDecoration(
                    color: Colors.white.withValues(alpha: 0.2),
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),

              // Header Bar: Close button & Title
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    IconButton(
                      icon: const Icon(Icons.close_rounded),
                      tooltip: 'Close',
                      onPressed: () => Navigator.pop(context),
                    ),
                    const Text(
                      'Account & Profile',
                      style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                    ),
                    const SizedBox(width: 48),
                  ],
                ),
              ),

              const Divider(height: 1),

              // Scrollable Content
              Flexible(
                child: ListView(
                  padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
                  children: [
                    // Identity Card
                    Container(
                      padding: const EdgeInsets.all(16),
                      decoration: BoxDecoration(
                        color: Colors.amber.withValues(alpha: 0.08),
                        borderRadius: BorderRadius.circular(20),
                        border: Border.all(color: Colors.amber.withValues(alpha: 0.2)),
                      ),
                      child: Row(
                        children: [
                          CircleAvatar(
                            radius: 30,
                            backgroundColor: const Color(0xFFF59E0B),
                            child: const Text('S', style: TextStyle(fontSize: 26, fontWeight: FontWeight.bold, color: Colors.black)),
                          ),
                          const SizedBox(width: 16),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                const Text(
                                  'Sarah Connor',
                                  style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                                ),
                                const SizedBox(height: 2),
                                const Text(
                                  'sarah@fleetpulse.io',
                                  style: TextStyle(fontSize: 12, color: Colors.grey),
                                ),
                                const SizedBox(height: 8),
                                Row(
                                  children: [
                                    Container(
                                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                                      decoration: BoxDecoration(
                                        color: Colors.amber.withValues(alpha: 0.2),
                                        borderRadius: BorderRadius.circular(12),
                                      ),
                                      child: const Text(
                                        'OPERATIONS LEAD',
                                        style: TextStyle(fontSize: 9, fontWeight: FontWeight.bold, color: Colors.amber),
                                      ),
                                    ),
                                    const SizedBox(width: 8),
                                    Container(
                                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                                      decoration: BoxDecoration(
                                        color: const Color(0xFF10B981).withValues(alpha: 0.2),
                                        borderRadius: BorderRadius.circular(12),
                                      ),
                                      child: const Text(
                                        'PRO ACTIVE',
                                        style: TextStyle(fontSize: 9, fontWeight: FontWeight.bold, color: Color(0xFF10B981)),
                                      ),
                                    ),
                                  ],
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    ),

                    const SizedBox(height: 20),

                    // Account Pages Navigation
                    _buildModalNavItem(
                      icon: Icons.settings_outlined,
                      iconColor: Colors.amber,
                      title: 'Settings & Preferences',
                      subtitle: 'Theme, Biometrics, Auto-Lock, Notifications',
                      onTap: () {
                        Navigator.pop(context);
                        Navigator.push(
                          context,
                          MaterialPageRoute(builder: (_) => SettingsScreen(onThemeToggle: onThemeToggle, isDark: isDark, api: api)),
                        );
                      },
                    ),

                    _buildModalNavItem(
                      icon: Icons.shopping_bag_outlined,
                      iconColor: Colors.blueAccent,
                      title: 'Orders & Procurement',
                      subtitle: 'Hardware orders, delivery tracking, invoices',
                      onTap: () {
                        Navigator.pop(context);
                        Navigator.push(
                          context,
                          MaterialPageRoute(builder: (_) => const OrdersScreen()),
                        );
                      },
                    ),

                    _buildModalNavItem(
                      icon: Icons.credit_card_outlined,
                      iconColor: Colors.tealAccent,
                      title: 'Billing & Fleet License',
                      subtitle: 'Enterprise subscription, cards, tax receipts',
                      onTap: () {
                        Navigator.pop(context);
                        Navigator.push(
                          context,
                          MaterialPageRoute(builder: (_) => const BillingScreen()),
                        );
                      },
                    ),

                    _buildModalNavItem(
                      icon: Icons.fingerprint_rounded,
                      iconColor: const Color(0xFF10B981),
                      title: 'Security & Biometrics',
                      subtitle: 'Sensor test diagnostic, 2FA, active sessions',
                      onTap: () {
                        Navigator.pop(context);
                        Navigator.push(
                          context,
                          MaterialPageRoute(builder: (_) => const SecuritySettingsScreen()),
                        );
                      },
                    ),

                    const SizedBox(height: 20),

                    // Quick Switch Operator
                    const Text('Switch Operator Account', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 0.6, color: Colors.grey)),
                    const SizedBox(height: 8),

                    Card(
                      child: Column(
                        children: [
                          ListTile(
                            leading: const CircleAvatar(radius: 14, child: Text('S', style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold))),
                            title: const Text('Sarah Connor', style: TextStyle(fontSize: 13, fontWeight: FontWeight.bold)),
                            subtitle: const Text('Operations Lead • Active Now', style: TextStyle(fontSize: 11)),
                            trailing: const Icon(Icons.check, color: Color(0xFF10B981), size: 18),
                          ),
                          const Divider(height: 1),
                          ListTile(
                            leading: const CircleAvatar(radius: 14, child: Text('M', style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold))),
                            title: const Text('Marcus Vance', style: TextStyle(fontSize: 13, fontWeight: FontWeight.bold)),
                            subtitle: const Text('Firmware Engineer', style: TextStyle(fontSize: 11)),
                            onTap: () {
                              Navigator.pop(context);
                              ScaffoldMessenger.of(context).showSnackBar(
                                const SnackBar(content: Text('Switched to Marcus Vance account.'), behavior: SnackBarBehavior.floating),
                              );
                            },
                          ),
                        ],
                      ),
                    ),

                    const SizedBox(height: 24),

                    // Sign out button
                    OutlinedButton.icon(
                      icon: const Icon(Icons.logout_rounded, color: Colors.redAccent, size: 18),
                      label: const Text('Sign Out', style: TextStyle(color: Colors.redAccent, fontWeight: FontWeight.bold)),
                      style: OutlinedButton.styleFrom(
                        side: const BorderSide(color: Colors.redAccent, width: 1.2),
                        padding: const EdgeInsets.symmetric(vertical: 14),
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                      ),
                      onPressed: () {
                        Navigator.pop(context);
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('Operator signed out.'), behavior: SnackBarBehavior.floating),
                        );
                      },
                    ),
                    const SizedBox(height: 12),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildModalNavItem({
    required IconData icon,
    required Color iconColor,
    required String title,
    required String subtitle,
    required VoidCallback onTap,
  }) {
    return Card(
      margin: const EdgeInsets.only(bottom: 10),
      child: ListTile(
        leading: Container(
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
            color: iconColor.withValues(alpha: 0.12),
            borderRadius: BorderRadius.circular(12),
          ),
          child: Icon(icon, color: iconColor, size: 22),
        ),
        title: Text(title, style: const TextStyle(fontSize: 14, fontWeight: FontWeight.bold)),
        subtitle: Text(subtitle, style: const TextStyle(fontSize: 11, color: Colors.grey)),
        trailing: const Icon(Icons.chevron_right_rounded, color: Colors.grey),
        onTap: onTap,
      ),
    );
  }
}

// ============================================================================
// Settings Screen
// ============================================================================
class SettingsScreen extends StatefulWidget {
  final VoidCallback onThemeToggle;
  final bool isDark;
  final FilamentApiClient api;

  const SettingsScreen({super.key, required this.onThemeToggle, required this.isDark, required this.api});

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  bool isBiometricEnabled = true;
  String autoLockTimeout = '5 Minutes';
  bool requireBioForCriticalActions = true;
  bool twoFactorEnabled = true;
  bool pushAlerts = true;
  bool criticalSensorAlarms = true;
  bool firmwareBroadcasts = true;
  bool analyticsEnabled = true;
  String cacheSize = '14.8 MB';

  void _testBiometricSensor() {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
        title: const Row(
          children: [
            Icon(Icons.fingerprint, color: Color(0xFF10B981), size: 28),
            SizedBox(width: 10),
            Text('Biometric Diagnostic'),
          ],
        ),
        content: const Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Touch sensor or align Face ID scanner to test hardware enrollment.'),
            SizedBox(height: 16),
            Center(
              child: Icon(Icons.fingerprint, size: 64, color: Color(0xFF10B981)),
            ),
          ],
        ),
        actions: [
          ElevatedButton(
            onPressed: () {
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                const SnackBar(
                  content: Text('✓ Biometric Sensor Authenticated Successfully!'),
                  backgroundColor: Color(0xFF10B981),
                  behavior: SnackBarBehavior.floating,
                ),
              );
            },
            child: const Text('Simulate Verification'),
          ),
        ],
      ),
    );
  }

  void _clearCache() {
    setState(() => cacheSize = '0.0 KB');
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('✓ Cache and local storage cleared.'), behavior: SnackBarBehavior.floating),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Settings & Preferences', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
      ),
      body: ListView(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 20),
        children: [
          // Theme Section
          const Text('APPEARANCE & THEME', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 0.8, color: Colors.grey)),
          const SizedBox(height: 8),
          Card(
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.brightness_6, color: Colors.amber),
                  title: const Text('Theme Mode', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: Text(widget.isDark ? 'Dark theme active' : 'Light theme active', style: const TextStyle(fontSize: 12)),
                  trailing: Switch(
                    value: widget.isDark,
                    onChanged: (v) => widget.onThemeToggle(),
                  ),
                ),
              ],
            ),
          ),

          const SizedBox(height: 20),

          // Biometrics Section
          const Text('BIOMETRICS & SECURITY', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 0.8, color: Colors.grey)),
          const SizedBox(height: 8),
          Card(
            child: Column(
              children: [
                SwitchListTile(
                  secondary: const Icon(Icons.fingerprint_rounded, color: Color(0xFF10B981)),
                  title: const Text('Biometric Authentication', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Unlock with Fingerprint or Face ID', style: TextStyle(fontSize: 12)),
                  value: isBiometricEnabled,
                  onChanged: (v) => setState(() => isBiometricEnabled = v),
                ),
                const Divider(height: 1),
                ListTile(
                  leading: const Icon(Icons.timer_outlined, color: Colors.blueAccent),
                  title: const Text('Auto-Lock Application', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Require biometrics when returning to app', style: TextStyle(fontSize: 12)),
                  trailing: DropdownButton<String>(
                    value: autoLockTimeout,
                    underline: const SizedBox(),
                    items: const [
                      DropdownMenuItem(value: 'Immediately', child: Text('Immediately')),
                      DropdownMenuItem(value: '1 Minute', child: Text('1 Minute')),
                      DropdownMenuItem(value: '5 Minutes', child: Text('5 Minutes')),
                      DropdownMenuItem(value: '15 Minutes', child: Text('15 Minutes')),
                      DropdownMenuItem(value: 'Never', child: Text('Never')),
                    ],
                    onChanged: (v) {
                      if (v != null) setState(() => autoLockTimeout = v);
                    },
                  ),
                ),
                const Divider(height: 1),
                SwitchListTile(
                  secondary: const Icon(Icons.verified_user_outlined, color: Colors.tealAccent),
                  title: const Text('Require Biometrics for Critical Actions', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Firmware broadcasts & node decommissioning', style: TextStyle(fontSize: 12)),
                  value: requireBioForCriticalActions,
                  onChanged: (v) => setState(() => requireBioForCriticalActions = v),
                ),
                const Divider(height: 1),
                SwitchListTile(
                  secondary: const Icon(Icons.security, color: Colors.indigoAccent),
                  title: const Text('Two-Factor Authentication (2FA)', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Extra SMS or Authenticator verification code', style: TextStyle(fontSize: 12)),
                  value: twoFactorEnabled,
                  onChanged: (v) => setState(() => twoFactorEnabled = v),
                ),
                const Divider(height: 1),
                ListTile(
                  leading: const Icon(Icons.touch_app_outlined, color: Colors.amber),
                  title: const Text('Diagnostic Sensor Test', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Verify biometric prompt scanner response', style: TextStyle(fontSize: 12)),
                  trailing: const Icon(Icons.chevron_right),
                  onTap: _testBiometricSensor,
                ),
              ],
            ),
          ),

          const SizedBox(height: 20),

          // Notifications Section
          const Text('NOTIFICATIONS & ALERTS', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 0.8, color: Colors.grey)),
          const SizedBox(height: 8),
          Card(
            child: Column(
              children: [
                SwitchListTile(
                  secondary: const Icon(Icons.notifications_active_outlined, color: Colors.amber),
                  title: const Text('Push Notifications', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Mobile fleet alerts & incident reports', style: TextStyle(fontSize: 12)),
                  value: pushAlerts,
                  onChanged: (v) => setState(() => pushAlerts = v),
                ),
                const Divider(height: 1),
                SwitchListTile(
                  secondary: const Icon(Icons.warning_amber_rounded, color: Colors.redAccent),
                  title: const Text('Critical Sensor Telemetry Alarms', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Battery drain & temperature spike alerts', style: TextStyle(fontSize: 12)),
                  value: criticalSensorAlarms,
                  onChanged: (v) => setState(() => criticalSensorAlarms = v),
                ),
                const Divider(height: 1),
                SwitchListTile(
                  secondary: const Icon(Icons.system_update_alt, color: Colors.blueAccent),
                  title: const Text('Firmware Deployment Broadcasts', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('OTA release status & rollout logs', style: TextStyle(fontSize: 12)),
                  value: firmwareBroadcasts,
                  onChanged: (v) => setState(() => firmwareBroadcasts = v),
                ),
              ],
            ),
          ),

          const SizedBox(height: 20),

          // Privacy & Storage
          const Text('STORAGE & DATA', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 0.8, color: Colors.grey)),
          const SizedBox(height: 8),
          Card(
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.cleaning_services_outlined, color: Colors.tealAccent),
                  title: const Text('Clear Cached Telemetry Data', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: Text('$cacheSize of offline logs and telemetry cached', style: const TextStyle(fontSize: 12)),
                  trailing: OutlinedButton(
                    onPressed: _clearCache,
                    child: const Text('Clear'),
                  ),
                ),
                const Divider(height: 1),
                SwitchListTile(
                  secondary: const Icon(Icons.analytics_outlined, color: Colors.indigoAccent),
                  title: const Text('Anonymous Diagnostics Sharing', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Help improve fleet latency calculations', style: TextStyle(fontSize: 12)),
                  value: analyticsEnabled,
                  onChanged: (v) => setState(() => analyticsEnabled = v),
                ),
              ],
            ),
          ),

          const SizedBox(height: 20),

          // Server API Section
          const Text('BACKEND & API CONFIGURATION', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 0.8, color: Colors.grey)),
          const SizedBox(height: 8),
          Card(
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.dns_outlined, color: Colors.amber),
                  title: const Text('GoFilament API URL', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: Text(widget.api.baseUrl, style: const TextStyle(fontFamily: 'monospace', fontSize: 12)),
                  trailing: const Icon(Icons.lock_outline, size: 18),
                ),
                const Divider(height: 1),
                ListTile(
                  leading: const Icon(Icons.business_outlined, color: Colors.blueAccent),
                  title: const Text('Tenant Workspace', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: Text(widget.api.activeTenant ?? 'Global Fleet Cluster', style: const TextStyle(fontSize: 12)),
                ),
              ],
            ),
          ),
          const SizedBox(height: 30),
        ],
      ),
    );
  }
}

// ============================================================================
// Orders Screen
// ============================================================================
class OrdersScreen extends StatefulWidget {
  const OrdersScreen({super.key});

  @override
  State<OrdersScreen> createState() => _OrdersScreenState();
}

class _OrdersScreenState extends State<OrdersScreen> with SingleTickerProviderStateMixin {
  late TabController tabController;

  final List<Map<String, dynamic>> orders = [
    {
      'id': '#PO-9921',
      'title': '10x PulseEdge v4 Gateway Nodes (Frankfurt Cluster)',
      'date': 'Oct 29, 2026',
      'amount': 2490.00,
      'status': 'Delivered',
      'items': '10 units • Model PE-v4-Pro',
    },
    {
      'id': '#PO-9844',
      'title': '5x Solar Battery Backup Packs (5000mAh)',
      'date': 'Oct 22, 2026',
      'amount': 450.00,
      'status': 'Delivered',
      'items': '5 units • Solar Lithium Packs',
    },
    {
      'id': '#PO-9710',
      'title': 'High-Gain Mesh Antenna Array Add-ons',
      'date': 'Nov 01, 2026',
      'amount': 620.00,
      'status': 'In Transit',
      'items': '4 units • Omnidirectional 8dBi',
    },
    {
      'id': '#PO-9650',
      'title': 'Industrial Temperature & Humidity Sensor Probes',
      'date': 'Oct 31, 2026',
      'amount': 380.00,
      'status': 'Processing',
      'items': '12 units • IP67 Sealed Sensors',
    },
  ];

  @override
  void initState() {
    super.initState();
    tabController = TabController(length: 3, vsync: this);
  }

  @override
  void dispose() {
    tabController.dispose();
    super.dispose();
  }

  void _showReceipt(Map<String, dynamic> o) {
    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
        title: Row(
          children: [
            const Icon(Icons.receipt_long_rounded, color: Colors.amber),
            const SizedBox(width: 8),
            Text(o['id'] as String),
          ],
        ),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(o['title'] as String, style: const TextStyle(fontWeight: FontWeight.bold)),
            const SizedBox(height: 8),
            Text('Ordered on: ${o['date']}', style: const TextStyle(fontSize: 12, color: Colors.grey)),
            Text('Items: ${o['items']}', style: const TextStyle(fontSize: 12, color: Colors.grey)),
            const Divider(height: 24),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text('Total Amount:', style: TextStyle(fontWeight: FontWeight.bold)),
                Text(
                  '\$${(o['amount'] as double).toStringAsFixed(2)}',
                  style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold, color: Colors.amber),
                ),
              ],
            ),
          ],
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Close')),
          ElevatedButton.icon(
            icon: const Icon(Icons.download, size: 16),
            label: const Text('Download PDF'),
            onPressed: () {
              Navigator.pop(ctx);
              ScaffoldMessenger.of(context).showSnackBar(
                SnackBar(content: Text('Invoice ${o['id']}.pdf downloaded.'), behavior: SnackBarBehavior.floating),
              );
            },
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Hardware Orders & Procurement', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
        bottom: TabBar(
          controller: tabController,
          labelColor: Colors.amber,
          indicatorColor: Colors.amber,
          tabs: const [
            Tab(text: 'All Orders'),
            Tab(text: 'In Transit'),
            Tab(text: 'Delivered'),
          ],
        ),
      ),
      body: TabBarView(
        controller: tabController,
        children: [
          _buildList(orders),
          _buildList(orders.where((o) => o['status'] == 'In Transit').toList()),
          _buildList(orders.where((o) => o['status'] == 'Delivered').toList()),
        ],
      ),
    );
  }

  Widget _buildList(List<Map<String, dynamic>> items) {
    if (items.isEmpty) {
      return const Center(child: Text('No procurement orders in this category.'));
    }
    return ListView.builder(
      padding: const EdgeInsets.all(16),
      itemCount: items.length,
      itemBuilder: (context, i) {
        final o = items[i];
        final isDelivered = o['status'] == 'Delivered';
        final isTransit = o['status'] == 'In Transit';
        final statusColor = isDelivered ? const Color(0xFF10B981) : (isTransit ? Colors.blueAccent : Colors.amber);

        return Card(
          margin: const EdgeInsets.only(bottom: 12),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      o['id'] as String,
                      style: const TextStyle(fontFamily: 'monospace', fontWeight: FontWeight.bold, color: Colors.amber),
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                      decoration: BoxDecoration(
                        color: statusColor.withValues(alpha: 0.15),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Text(
                        (o['status'] as String).toUpperCase(),
                        style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: statusColor),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Text(o['title'] as String, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 15)),
                const SizedBox(height: 4),
                Text(o['items'] as String, style: const TextStyle(fontSize: 12, color: Colors.grey)),
                const SizedBox(height: 12),
                const Divider(height: 1),
                const SizedBox(height: 10),
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      '\$${(o['amount'] as double).toStringAsFixed(2)}',
                      style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                    ),
                    Row(
                      children: [
                        TextButton.icon(
                          icon: const Icon(Icons.receipt_outlined, size: 16),
                          label: const Text('Invoice'),
                          onPressed: () => _showReceipt(o),
                        ),
                        const SizedBox(width: 8),
                        ElevatedButton.icon(
                          icon: const Icon(Icons.refresh, size: 16),
                          label: const Text('Reorder'),
                          onPressed: () {
                            ScaffoldMessenger.of(context).showSnackBar(
                              SnackBar(content: Text('Reorder for ${o['id']} submitted to supply chain.'), behavior: SnackBarBehavior.floating),
                            );
                          },
                        ),
                      ],
                    ),
                  ],
                ),
              ],
            ),
          ),
        );
      },
    );
  }
}

// ============================================================================
// Billing Screen
// ============================================================================
class BillingScreen extends StatelessWidget {
  const BillingScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Billing & Fleet License', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          // Active Plan Card
          Card(
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(20),
              side: BorderSide(color: Colors.amber.withValues(alpha: 0.4)),
            ),
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text(
                        'Enterprise IoT Fleet License',
                        style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                      ),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                        decoration: BoxDecoration(
                          color: const Color(0xFF10B981).withValues(alpha: 0.2),
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: const Text(
                          'ACTIVE',
                          style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: Color(0xFF10B981)),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 6),
                  const Text('Up to 500 Active Edge Nodes • Auto-renews Dec 01, 2026', style: TextStyle(fontSize: 12, color: Colors.grey)),
                  const SizedBox(height: 16),
                  const Text(
                    '\$299.00 / month',
                    style: TextStyle(fontSize: 26, fontWeight: FontWeight.bold, color: Colors.amber),
                  ),
                  const SizedBox(height: 16),
                  ElevatedButton(
                    onPressed: () {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('Tier management portal opened.'), behavior: SnackBarBehavior.floating),
                      );
                    },
                    child: const Text('Manage Plan Tier'),
                  ),
                ],
              ),
            ),
          ),

          const SizedBox(height: 24),
          const Text('PAYMENT METHODS', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 0.8, color: Colors.grey)),
          const SizedBox(height: 8),

          Card(
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.credit_card, color: Colors.amber),
                  title: const Text('Corporate Visa ending in 4242', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Expires 12/28', style: TextStyle(fontSize: 12)),
                  trailing: Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                    decoration: BoxDecoration(
                      color: Colors.amber.withValues(alpha: 0.2),
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: const Text('DEFAULT', style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: Colors.amber)),
                  ),
                ),
              ],
            ),
          ),

          const SizedBox(height: 24),
          const Text('RECENT INVOICES', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 0.8, color: Colors.grey)),
          const SizedBox(height: 8),

          Card(
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.description_outlined),
                  title: const Text('Invoice #INV-2026-10', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Oct 01, 2026 • Paid', style: TextStyle(fontSize: 12)),
                  trailing: TextButton.icon(
                    icon: const Icon(Icons.download, size: 16),
                    label: const Text('\$299.00'),
                    onPressed: () {},
                  ),
                ),
                const Divider(height: 1),
                ListTile(
                  leading: const Icon(Icons.description_outlined),
                  title: const Text('Invoice #INV-2026-09', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Sep 01, 2026 • Paid', style: TextStyle(fontSize: 12)),
                  trailing: TextButton.icon(
                    icon: const Icon(Icons.download, size: 16),
                    label: const Text('\$299.00'),
                    onPressed: () {},
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

// ============================================================================
// Security Settings Screen
// ============================================================================
class SecuritySettingsScreen extends StatelessWidget {
  const SecuritySettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Security & Biometrics', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
      ),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Row(
                    children: [
                      Icon(Icons.shield_outlined, color: Color(0xFF10B981), size: 24),
                      SizedBox(width: 10),
                      Text('Hardware Security Status', style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
                    ],
                  ),
                  const SizedBox(height: 12),
                  const Text('Device hardware is protected with Biometric Keystore Enclave & WebAuthn standard encryption.'),
                  const SizedBox(height: 16),
                  ElevatedButton.icon(
                    icon: const Icon(Icons.fingerprint),
                    label: const Text('Run Live Biometric Sensor Check'),
                    onPressed: () {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('✓ Fingerprint & Face ID sensors verified operational.'), behavior: SnackBarBehavior.floating),
                      );
                    },
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          const Text('LOGGED-IN SESSIONS & DEVICES', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 0.8, color: Colors.grey)),
          const SizedBox(height: 8),
          Card(
            child: Column(
              children: [
                ListTile(
                  leading: const Icon(Icons.phone_android, color: Colors.amber),
                  title: const Text('Pixel 8 Pro (Active Session)', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('Frankfurt DC VPN • Active Now', style: TextStyle(fontSize: 12)),
                  trailing: const Icon(Icons.check_circle, color: Color(0xFF10B981), size: 18),
                ),
                const Divider(height: 1),
                ListTile(
                  leading: const Icon(Icons.laptop_chromebook, color: Colors.blueAccent),
                  title: const Text('Chrome Web Management Console', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: const Text('IP 192.168.1.102 • 14 minutes ago', style: TextStyle(fontSize: 12)),
                  trailing: TextButton(
                    onPressed: () {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('Web session terminated.'), behavior: SnackBarBehavior.floating),
                      );
                    },
                    child: const Text('Revoke', style: TextStyle(color: Colors.redAccent)),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

