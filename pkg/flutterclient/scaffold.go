package flutterclient

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateFlutterAdmin creates a responsive, mobile-first Flutter admin client project.
func GenerateFlutterAdmin(targetDir string, panelName, apiBaseURL string) error {
	libDir := filepath.Join(targetDir, "lib")
	webDir := filepath.Join(targetDir, "web")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(webDir, 0755); err != nil {
		return err
	}

	titleName := strings.Title(panelName)

	// 1. pubspec.yaml
	pubspec := fmt.Sprintf(`name: %s_admin
description: "Mobile-First Responsive Admin Panel powered by GoFilament and GoLaravel"
publish_to: 'none'
version: 1.0.0+1

environment:
  sdk: '>=3.0.0 <4.0.0'

dependencies:
  flutter:
    sdk: flutter
  http: ^1.2.0

dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^3.0.0

flutter:
  uses-material-design: true
`, strings.ToLower(panelName))

	if err := os.WriteFile(filepath.Join(targetDir, "pubspec.yaml"), []byte(pubspec), 0644); err != nil {
		return err
	}

	// 2. web/index.html
	indexHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <base href="/">
  <meta charset="UTF-8">
  <meta content="IE=Edge" http-equiv="X-UA-Compatible">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
  <title>%s Admin</title>
  <link rel="manifest" href="manifest.json">
</head>
<body>
  <script src="flutter_bootstrap.js" async></script>
</body>
</html>`, titleName)
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte(indexHTML), 0644); err != nil {
		return err
	}

	// 3. lib/api.dart - GoFilament API Client
	apiDart := fmt.Sprintf(`import 'dart:convert';
import 'package:http/http.dart' as http;

class FilamentApiClient {
  final String baseUrl;
  String? activeTenant;

  FilamentApiClient({this.baseUrl = '%s'});

  Map<String, String> get _headers {
    final h = <String, String>{
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
    if (activeTenant != null && activeTenant!.isNotEmpty) {
      h['X-Tenant-ID'] = activeTenant!;
    }
    return h;
  }

  Future<Map<String, dynamic>> fetchSchema() async {
    final res = await http.get(Uri.parse('$baseUrl/api/schema'), headers: _headers);
    if (res.statusCode == 200) {
      return json.decode(res.body);
    }
    throw Exception('Failed to load panel schema: ${res.statusCode}');
  }

  Future<Map<String, dynamic>> fetchResourceRecords(String slug, {int page = 1, String search = ''}) async {
    final uri = Uri.parse('$baseUrl/api/resources/$slug').replace(queryParameters: {
      'page': page.toString(),
      if (search.isNotEmpty) 'search': search,
    });
    final res = await http.get(uri, headers: _headers);
    if (res.statusCode == 200) {
      return json.decode(res.body);
    }
    throw Exception('Failed to load records for $slug: ${res.statusCode}');
  }

  Future<Map<String, dynamic>> fetchRecord(String slug, String id) async {
    final res = await http.get(Uri.parse('$baseUrl/api/resources/$slug/$id'), headers: _headers);
    if (res.statusCode == 200) {
      return json.decode(res.body);
    }
    throw Exception('Failed to load record: ${res.statusCode}');
  }

  Future<bool> createRecord(String slug, Map<String, dynamic> data) async {
    final res = await http.post(
      Uri.parse('$baseUrl/api/resources/$slug'),
      headers: _headers,
      body: json.encode(data),
    );
    return res.statusCode == 200 || res.statusCode == 201;
  }

  Future<bool> updateRecord(String slug, String id, Map<String, dynamic> data) async {
    final res = await http.put(
      Uri.parse('$baseUrl/api/resources/$slug/$id'),
      headers: _headers,
      body: json.encode(data),
    );
    return res.statusCode == 200;
  }

  Future<bool> deleteRecord(String slug, String id) async {
    final res = await http.delete(Uri.parse('$baseUrl/api/resources/$slug/$id'), headers: _headers);
    return res.statusCode == 200;
  }

  Future<List<dynamic>> fetchTenants() async {
    final res = await http.get(Uri.parse('$baseUrl/api/tenants'), headers: _headers);
    if (res.statusCode == 200) {
      final data = json.decode(res.body);
      return data['tenants'] ?? [];
    }
    return [];
  }
}
`, apiBaseURL)
	if err := os.WriteFile(filepath.Join(libDir, "api.dart"), []byte(apiDart), 0644); err != nil {
		return err
	}

	// 4. lib/responsive.dart - Responsive Layout Utilities
	responsiveDart := `import 'package:flutter/material.dart';

class ResponsiveLayout extends StatelessWidget {
  final Widget mobile;
  final Widget? tablet;
  final Widget desktop;

  const ResponsiveLayout({
    super.key,
    required this.mobile,
    this.tablet,
    required this.desktop,
  });

  static bool isMobile(BuildContext context) =>
      MediaQuery.of(context).size.width < 640;

  static bool isTablet(BuildContext context) =>
      MediaQuery.of(context).size.width >= 640 &&
      MediaQuery.of(context).size.width < 1024;

  static bool isDesktop(BuildContext context) =>
      MediaQuery.of(context).size.width >= 1024;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth >= 1024) {
          return desktop;
        } else if (constraints.maxWidth >= 640) {
          return tablet ?? desktop;
        } else {
          return mobile;
        }
      },
    );
  }
}
`
	if err := os.WriteFile(filepath.Join(libDir, "responsive.dart"), []byte(responsiveDart), 0644); err != nil {
		return err
	}

	// 5. lib/main.dart - Mobile-First Responsive Flutter Admin Application
	mainDart := fmt.Sprintf(`import 'package:flutter/material.dart';
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
      title: '%s Admin',
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

  @override
  void initState() {
    super.initState();
    loadSchema();
  }

  Future<void> loadSchema() async {
    setState(() => isLoading = true);
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
      setState(() => isLoading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (isLoading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    final brandName = schema?['brand_name'] ?? '%s';
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

  // --- Mobile Resource Feed with Swipe-to-Delete and Pull-to-Refresh ---
  Widget _buildMobileResourceFeed(List<dynamic> resources) {
    if (selectedResourceSlug == null) {
      return const Center(child: Text('Select an entity'));
    }
    return ResourceRecordList(api: api, slug: selectedResourceSlug!);
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
                  child: ListView.builder(
                    padding: const EdgeInsets.all(12),
                    itemCount: records.length,
                    itemBuilder: (context, i) {
                      final item = records[i];
                      final id = item['ID'] ?? item['id'] ?? '$i';
                      final title = item['Name'] ?? item['Title'] ?? item['name'] ?? item['title'] ?? 'Record #$id';

                      return Dismissible(
                        key: ValueKey(id),
                        direction: DismissDirection.endToStart,
                        background: Container(
                          alignment: Alignment.centerRight,
                          padding: const EdgeInsets.only(right: 20),
                          color: Colors.red,
                          child: const Icon(Icons.delete, color: Colors.white),
                        ),
                        confirmDismiss: (dir) async {
                          return await showDialog(
                            context: context,
                            builder: (ctx) => AlertDialog(
                              title: const Text('Delete Record?'),
                              content: Text('Are you sure you want to delete #$id?'),
                              actions: [
                                TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
                                TextButton(onPressed: () => Navigator.pop(ctx, true), child: const Text('Delete', style: TextStyle(color: Colors.red))),
                              ],
                            ),
                          );
                        },
                        onDismissed: (dir) async {
                          await widget.api.deleteRecord(widget.slug, id.toString());
                        },
                        child: Card(
                          margin: const EdgeInsets.only(bottom: 8),
                          child: ListTile(
                            title: Text(title.toString(), style: const TextStyle(fontWeight: FontWeight.w600)),
                            subtitle: Text('ID: $id'),
                            trailing: const Icon(Icons.edit_outlined, size: 18),
                            onTap: () => showEditSheet(context, id.toString(), item),
                          ),
                        ),
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
    final titleCtrl = TextEditingController();
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(left: 16, right: 16, top: 16, bottom: MediaQuery.of(ctx).viewInsets.bottom + 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text('Create ${widget.slug}', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
            const SizedBox(height: 12),
            TextField(controller: titleCtrl, decoration: const InputDecoration(labelText: 'Name / Title', border: OutlineInputBorder())),
            const SizedBox(height: 16),
            ElevatedButton(
              onPressed: () async {
                if (titleCtrl.text.isNotEmpty) {
                  await widget.api.createRecord(widget.slug, {'name': titleCtrl.text, 'title': titleCtrl.text});
                  if (!mounted) return;
                  Navigator.pop(ctx);
                  loadRecords();
                }
              },
              child: const Text('Save Record'),
            ),
          ],
        ),
      ),
    );
  }

  void showEditSheet(BuildContext context, String id, dynamic item) {
    final titleCtrl = TextEditingController(text: item['Name'] ?? item['Title'] ?? item['name'] ?? '');
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(left: 16, right: 16, top: 16, bottom: MediaQuery.of(ctx).viewInsets.bottom + 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text('Edit ${widget.slug} #$id', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
            const SizedBox(height: 12),
            TextField(controller: titleCtrl, decoration: const InputDecoration(labelText: 'Name / Title', border: OutlineInputBorder())),
            const SizedBox(height: 16),
            ElevatedButton(
              onPressed: () async {
                await widget.api.updateRecord(widget.slug, id, {'name': titleCtrl.text, 'title': titleCtrl.text});
                if (!mounted) return;
                Navigator.pop(ctx);
                loadRecords();
              },
              child: const Text('Update Changes'),
            ),
          ],
        ),
      ),
    );
  }
}
`, titleName, titleName)

	return os.WriteFile(filepath.Join(libDir, "main.dart"), []byte(mainDart), 0644)
}
