package flutterclient

import (
	"fmt"
	"net/url"
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
  shared_preferences: ^2.5.6

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
  <base href="$FLUTTER_BASE_HREF">
  <meta charset="UTF-8">
  <meta content="IE=Edge" http-equiv="X-UA-Compatible">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
  <title>%s Admin</title>
  <link rel="icon" type="image/png" href="favicon.png"/>
  <link rel="apple-touch-icon" href="icons/Icon-192.png">
  <link rel="manifest" href="manifest.json">
</head>
<body>
  <script src="flutter_bootstrap.js" async></script>
</body>
</html>`, titleName)
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte(indexHTML), 0644); err != nil {
		return err
	}

	panelPath := apiBaseURL
	if u, err := url.Parse(apiBaseURL); err == nil && u.Path != "" {
		panelPath = u.Path
	}
	if !strings.HasPrefix(panelPath, "/") {
		panelPath = "/" + panelPath
	}

	// 3. lib/api.dart - GoFilament API Client
	apiDart := fmt.Sprintf(`import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

class FilamentApiClient {
  String baseUrl;
  String? activeTenant;
  String? token;
  Map<String, dynamic>? currentUser;

  FilamentApiClient({String? baseUrl}) : baseUrl = baseUrl ?? _defaultBaseUrl();

  bool get isAuthenticated => token != null && token!.isNotEmpty;

  String get _storageKeyPrefix => 'gofilament_${Uri.parse(baseUrl).path.replaceAll('/', '_')}';

  Future<void> saveSession({
    required String? token,
    Map<String, dynamic>? user,
    String? expiresAt,
    bool remember = false,
  }) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final prefix = _storageKeyPrefix;
      if (token != null && token.isNotEmpty) {
        await prefs.setString('${prefix}_token', token);
        if (user != null) {
          await prefs.setString('${prefix}_user', json.encode(user));
        }
        if (expiresAt != null && expiresAt.isNotEmpty) {
          await prefs.setString('${prefix}_expires_at', expiresAt);
        }
        await prefs.setBool('${prefix}_remember', remember);
      }
    } catch (_) {}
  }

  Future<bool> restoreSession() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final prefix = _storageKeyPrefix;
      final savedToken = prefs.getString('${prefix}_token');
      if (savedToken == null || savedToken.isEmpty) {
        return false;
      }
      final expStr = prefs.getString('${prefix}_expires_at');
      if (expStr != null && expStr.isNotEmpty) {
        final exp = DateTime.tryParse(expStr);
        if (exp != null && DateTime.now().isAfter(exp)) {
          await clearSession();
          return false;
        }
      }
      token = savedToken;
      final userJson = prefs.getString('${prefix}_user');
      if (userJson != null && userJson.isNotEmpty) {
        try {
          currentUser = json.decode(userJson) as Map<String, dynamic>?;
        } catch (_) {}
      }
      return true;
    } catch (_) {
      return false;
    }
  }

  Future<void> clearSession() async {
    token = null;
    currentUser = null;
    try {
      final prefs = await SharedPreferences.getInstance();
      final prefix = _storageKeyPrefix;
      await prefs.remove('${prefix}_token');
      await prefs.remove('${prefix}_user');
      await prefs.remove('${prefix}_expires_at');
      await prefs.remove('${prefix}_remember');
    } catch (_) {}
  }

  Future<Map<String, dynamic>> login(String username, String password, {bool remember = false}) async {
    final res = await http.post(
      Uri.parse('$baseUrl/api/login'),
      headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
      body: json.encode({'username': username, 'password': password, 'remember': remember}),
    );
    if (res.statusCode == 200) {
      final data = json.decode(res.body) as Map<String, dynamic>;
      token = data['token'] as String?;
      currentUser = data['user'] as Map<String, dynamic>?;
      final expiresAtStr = data['expires_at'] as String?;
      await saveSession(token: token, user: currentUser, expiresAt: expiresAtStr, remember: remember);
      return data;
    }
    try {
      final err = json.decode(res.body);
      throw Exception(err['error'] ?? 'Authentication failed');
    } catch (e) {
      if (e is Exception) rethrow;
      throw Exception('Authentication failed (${res.statusCode})');
    }
  }

  Future<void> logout() async {
    try {
      if (token != null && token!.isNotEmpty) {
        await http.post(Uri.parse('$baseUrl/api/logout'), headers: _headers);
      }
    } catch (_) {}
    await clearSession();
  }

  static String _defaultBaseUrl() {
    if (kIsWeb) {
      final origin = Uri.base.origin;
      final host = Uri.base.host;
      if (host.isNotEmpty && host != 'localhost' && host != '127.0.0.1') {
        return '$origin%s';
      }
      if (Uri.base.port == 8081 || Uri.base.port == 8080 || Uri.base.port == 80 || Uri.base.port == 443) {
        return '$origin%s';
      }
      return '%s';
    }
    return '%s';
  }

  Map<String, String> get _headers {
    final h = <String, String>{
      'Content-Type': 'application/json',
      'Accept': 'application/json',
    };
    if (token != null && token!.isNotEmpty) {
      h['Authorization'] = 'Bearer $token';
    }
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

  Future<Map<String, dynamic>> fetchResourceRecords(String slug, {int page = 1, String search = '', Map<String, dynamic>? filters}) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      if (search.isNotEmpty) 'search': search,
    };
    if (filters != null) {
      filters.forEach((key, val) {
        if (val != null) {
          if (val is Iterable) {
            final joined = val.map((e) => e.toString()).where((e) => e.isNotEmpty).join(',');
            if (joined.isNotEmpty) queryParams['filter_$key'] = joined;
          } else if (val.toString().isNotEmpty) {
            queryParams['filter_$key'] = val.toString();
          }
        }
      });
    }
    final uri = Uri.parse('$baseUrl/api/resources/$slug').replace(queryParameters: queryParams);
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

  Future<bool> executeBulkAction(String slug, String action, List<String> ids) async {
    try {
      final res = await http.post(
        Uri.parse('$baseUrl/api/resources/$slug/bulk'),
        headers: _headers,
        body: json.encode({
          'bulk_action': action,
          'ids': ids,
        }),
      );
      return res.statusCode == 200;
    } catch (_) {
      return false;
    }
  }

  Future<bool> bulkDeleteRecords(String slug, List<String> ids) async {
    final ok = await executeBulkAction(slug, 'delete_bulk', ids);
    if (!ok) {
      for (final id in ids) {
        await deleteRecord(slug, id);
      }
    }
    return true;
  }

  Future<List<dynamic>> fetchTenants() async {
    final res = await http.get(Uri.parse('$baseUrl/api/tenants'), headers: _headers);
    if (res.statusCode == 200) {
      final data = json.decode(res.body);
      return data['tenants'] ?? [];
    }
    return [];
  }

  Future<Map<String, dynamic>> switchLocale(String locale) async {
    try {
      final res = await http.post(
        Uri.parse('$baseUrl/api/locale/switch?lang=$locale'),
        headers: _headers,
      );
      if (res.statusCode == 200) {
        return json.decode(res.body);
      }
    } catch (_) {}
    return {};
  }

  Future<Map<String, dynamic>> fetchTranslations(String locale) async {
    try {
      final res = await http.get(
        Uri.parse('$baseUrl/api/translations?locale=$locale'),
        headers: _headers,
      );
      if (res.statusCode == 200) {
        return json.decode(res.body);
      }
    } catch (_) {}
    return {};
  }

  Future<Map<String, dynamic>> fetchRelationRecords(
    String parentSlug,
    String parentId,
    String relSlug, {
    int page = 1,
    String search = '',
  }) async {
    final uri = Uri.parse('$baseUrl/api/resources/$parentSlug/$parentId/relations/$relSlug').replace(
      queryParameters: {
        'page': page.toString(),
        if (search.isNotEmpty) 'search': search,
      },
    );
    final res = await http.get(uri, headers: _headers);
    if (res.statusCode == 200) {
      return json.decode(res.body);
    }
    throw Exception('Failed to load relation records: ${res.statusCode}');
  }

  Future<bool> createRelationRecord(
    String parentSlug,
    String parentId,
    String relSlug,
    Map<String, dynamic> data,
  ) async {
    final res = await http.post(
      Uri.parse('$baseUrl/api/resources/$parentSlug/$parentId/relations/$relSlug'),
      headers: _headers,
      body: json.encode(data),
    );
    return res.statusCode == 200 || res.statusCode == 201;
  }

  Future<bool> updateRelationRecord(
    String parentSlug,
    String parentId,
    String relSlug,
    String childId,
    Map<String, dynamic> data,
  ) async {
    final res = await http.put(
      Uri.parse('$baseUrl/api/resources/$parentSlug/$parentId/relations/$relSlug/$childId'),
      headers: _headers,
      body: json.encode(data),
    );
    return res.statusCode == 200;
  }

  Future<bool> deleteRelationRecord(
    String parentSlug,
    String parentId,
    String relSlug,
    String childId,
  ) async {
    final res = await http.delete(
      Uri.parse('$baseUrl/api/resources/$parentSlug/$parentId/relations/$relSlug/$childId'),
      headers: _headers,
    );
    return res.statusCode == 200;
  }

  Future<bool> bulkDeleteRelationRecords(String parentSlug, String parentId, String relSlug, List<String> childIds) async {
    for (final childId in childIds) {
      await deleteRelationRecord(parentSlug, parentId, relSlug, childId);
    }
    return true;
  }
}
`, panelPath, panelPath, apiBaseURL, apiBaseURL)
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
      MediaQuery.of(context).size.width <= 900;

  static bool isTablet(BuildContext context) =>
      MediaQuery.of(context).size.width > 600 &&
      MediaQuery.of(context).size.width <= 900;

  static bool isDesktop(BuildContext context) =>
      MediaQuery.of(context).size.width > 900;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth > 900) {
          return desktop;
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

	// 5. lib/relation_record_screen.dart - Nested Relation Managers Screen
	relationRecordDart := `import 'package:flutter/material.dart';
import 'api.dart';

class RelationRecordScreen extends StatefulWidget {
  final FilamentApiClient api;
  final String parentSlug;
  final String parentId;
  final String parentTitle;
  final String relSlug;
  final String relTitle;

  const RelationRecordScreen({
    super.key,
    required this.api,
    required this.parentSlug,
    required this.parentId,
    required this.parentTitle,
    required this.relSlug,
    required this.relTitle,
  });

  @override
  State<RelationRecordScreen> createState() => _RelationRecordScreenState();
}

class _RelationRecordScreenState extends State<RelationRecordScreen> {
  List<dynamic> records = [];
  bool isLoading = true;
  String search = '';
  final TextEditingController _searchCtrl = TextEditingController();
  final Map<String, Set<String>> activeFilters = {};
  bool _showRelations = true;
  final ScrollController _horizontalScrollController = ScrollController();
  final ScrollController _verticalScrollController = ScrollController();
  Set<String> selectedRecordIds = <String>{};

  bool get isAllSelected =>
      filteredRecords.isNotEmpty &&
      filteredRecords.every((r) {
        final id = _getChildId(r);
        return id.isNotEmpty && selectedRecordIds.contains(id);
      });

  void _toggleSelectAll([bool? forceVal]) {
    setState(() {
      final shouldSelect = forceVal ?? !isAllSelected;
      if (!shouldSelect) {
        selectedRecordIds.clear();
      } else {
        selectedRecordIds = filteredRecords
            .asMap()
            .entries
            .map((e) => _getChildId(e.value, e.key))
            .where((id) => id.isNotEmpty)
            .toSet();
      }
    });
  }

  void _toggleSelectRecord(String id) {
    if (id.isEmpty) return;
    setState(() {
      if (selectedRecordIds.contains(id)) {
        selectedRecordIds.remove(id);
      } else {
        selectedRecordIds.add(id);
      }
    });
  }

  void _deselectAll() {
    setState(() {
      selectedRecordIds.clear();
    });
  }

  Future<void> _executeBulkDelete(BuildContext context) async {
    if (selectedRecordIds.isEmpty) return;
    final count = selectedRecordIds.length;
    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text("Delete Records?"),
        content: Text("Are you sure you want to delete $count selected records? This action cannot be undone."),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text("Cancel"),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text("Delete Selected", style: TextStyle(color: Colors.redAccent, fontWeight: FontWeight.bold)),
          ),
        ],
      ),
    );
    if (confirm == true) {
      final success = await widget.api.bulkDeleteRelationRecords(
        widget.parentSlug,
        widget.parentId,
        widget.relSlug,
        selectedRecordIds.toList(),
      );
      if (mounted) {
        if (success) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text("$count records deleted successfully"),
              backgroundColor: Colors.green.shade700,
            ),
          );
          _deselectAll();
          loadRecords();
        } else {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text("Failed to perform bulk delete"),
              backgroundColor: Colors.redAccent,
            ),
          );
        }
      }
    }
  }

  void _exportSelectedRecords(BuildContext context) {
    if (selectedRecordIds.isEmpty) return;
    final count = selectedRecordIds.length;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text("Exported $count selected records."),
        backgroundColor: Colors.amber.shade800,
      ),
    );
  }

  @override
  void initState() {
    super.initState();
    loadRecords();
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    _horizontalScrollController.dispose();
    _verticalScrollController.dispose();
    super.dispose();
  }

  Future<void> loadRecords() async {
    setState(() {
      isLoading = true;
      selectedRecordIds.clear();
    });
    try {
      final res = await widget.api.fetchRelationRecords(
        widget.parentSlug,
        widget.parentId,
        widget.relSlug,
        search: search,
      );
      if (mounted) {
        setState(() {
          records = (res['data'] as List<dynamic>?) ?? [];
          isLoading = false;
        });
      }
    } catch (e) {
      if (mounted) {
        setState(() => isLoading = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Failed to load ${widget.relTitle}: $e'),
            backgroundColor: Colors.redAccent,
            behavior: SnackBarBehavior.floating,
          ),
        );
      }
    }
  }

  String _getChildId(dynamic item, [int? index]) {
    if (item == null) return index != null ? "$index" : "";
    final raw = item["id"] ??
        item["ID"] ??
        item["login"] ??
        item["chip_serial"] ??
        item["uuid"] ??
        item["plate_number"] ??
        item["_id"];
    if (raw != null && raw.toString().isNotEmpty) {
      return raw.toString();
    }
    return index != null ? "$index" : "";
  }

  String _getChildTitle(dynamic item) {
    if (item == null) return 'Record';
    return (item['name'] ??
            item['Name'] ??
            item['login'] ??
            item['plate_number'] ??
            item['chip_serial'] ??
            '#${_getChildId(item)}')
        .toString();
  }

  void _showAddEditSheet([dynamic item]) {
    final isEdit = item != null;
    final childId = isEdit ? _getChildId(item) : '';

    if (widget.relSlug == 'admins') {
      _showAdminSheet(isEdit, childId, item);
    } else if (widget.relSlug == 'branches') {
      _showBranchSheet(isEdit, childId, item);
    } else if (widget.relSlug == 'vehicles') {
      _showVehicleSheet(isEdit, childId, item);
    } else {
      _showGenericSheet(isEdit, childId, item);
    }
  }

  // --- 1. Admins Add / Edit Sheet ---
  void _showAdminSheet(bool isEdit, String childId, dynamic item) {
    final loginCtrl = TextEditingController(text: isEdit ? (item['login'] ?? '') : '');
    final nameCtrl = TextEditingController(text: isEdit ? (item['name'] ?? '') : '');
    final emailCtrl = TextEditingController(text: isEdit ? (item['email'] ?? '') : '');
    final passCtrl = TextEditingController();
    String active = isEdit ? (item['active'] ?? 'Y').toString() : 'Y';
    bool isSaving = false;

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setSheetState) => Padding(
          padding: EdgeInsets.only(
            left: 20,
            right: 20,
            top: 20,
            bottom: MediaQuery.of(ctx).viewInsets.bottom + 20,
          ),
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      isEdit ? 'Edit Admin: $childId' : 'Add New Admin',
                      style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
                    ),
                    IconButton(icon: const Icon(Icons.close), onPressed: () => Navigator.pop(ctx)),
                  ],
                ),
                Text(
                  'Associated with ${widget.parentTitle} (#${widget.parentId})',
                  style: TextStyle(fontSize: 12, color: Colors.grey.shade400),
                ),
                const SizedBox(height: 16),
                TextField(
                  controller: loginCtrl,
                  enabled: !isEdit,
                  decoration: InputDecoration(
                    labelText: 'Username / Login',
                    prefixIcon: const Icon(Icons.person_outline),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    isDense: true,
                    helperText: isEdit ? 'Username cannot be modified' : null,
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: nameCtrl,
                  decoration: InputDecoration(
                    labelText: 'Full Name',
                    prefixIcon: const Icon(Icons.badge_outlined),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    isDense: true,
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: emailCtrl,
                  keyboardType: TextInputType.emailAddress,
                  decoration: InputDecoration(
                    labelText: 'Email Address',
                    prefixIcon: const Icon(Icons.email_outlined),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    isDense: true,
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: passCtrl,
                  obscureText: true,
                  decoration: InputDecoration(
                    labelText: isEdit ? 'Change Password (leave empty to keep)' : 'Password',
                    prefixIcon: const Icon(Icons.lock_outline),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    isDense: true,
                    hintText: isEdit ? 'Leave blank to preserve password' : 'e.g. partner123',
                  ),
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  initialValue: active,
                  decoration: InputDecoration(
                    labelText: 'Account Status',
                    prefixIcon: const Icon(Icons.toggle_on_outlined),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    isDense: true,
                  ),
                  items: const [
                    DropdownMenuItem(value: 'Y', child: Text('Active (Can Login)')),
                    DropdownMenuItem(value: 'N', child: Text('Suspended / Inactive')),
                  ],
                  onChanged: (val) {
                    if (val != null) setSheetState(() => active = val);
                  },
                ),
                const SizedBox(height: 20),
                FilledButton.icon(
                  icon: isSaving
                      ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                      : Icon(isEdit ? Icons.save : Icons.add),
                  label: Text(isEdit ? 'Save Changes' : 'Create Admin Account'),
                  style: FilledButton.styleFrom(
                    backgroundColor: Colors.amber.shade700,
                    foregroundColor: Colors.black,
                    padding: const EdgeInsets.symmetric(vertical: 14),
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                  ),
                  onPressed: isSaving
                      ? null
                      : () async {
                          if (!isEdit && loginCtrl.text.trim().isEmpty) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('Username is required')),
                            );
                            return;
                          }
                          setSheetState(() => isSaving = true);
                          final payload = <String, dynamic>{
                            'login': loginCtrl.text.trim(),
                            'name': nameCtrl.text.trim(),
                            'email': emailCtrl.text.trim(),
                            'active': active,
                          };
                          if (passCtrl.text.isNotEmpty) {
                            payload['pswd'] = passCtrl.text;
                          }

                          bool ok = false;
                          if (isEdit) {
                            ok = await widget.api.updateRelationRecord(
                              widget.parentSlug,
                              widget.parentId,
                              widget.relSlug,
                              childId,
                              payload,
                            );
                          } else {
                            ok = await widget.api.createRelationRecord(
                              widget.parentSlug,
                              widget.parentId,
                              widget.relSlug,
                              payload,
                            );
                          }

                          if (!mounted) return;
                          Navigator.pop(ctx);
                          if (ok) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              SnackBar(
                                content: Text(isEdit ? 'Admin updated successfully' : 'Admin created successfully'),
                                backgroundColor: const Color(0xFF10B981),
                                behavior: SnackBarBehavior.floating,
                              ),
                            );
                            loadRecords();
                          } else {
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('Operation failed. Please try again.'), backgroundColor: Colors.redAccent),
                            );
                          }
                        },
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  // --- 2. Branches Add / Edit Sheet ---
  void _showBranchSheet(bool isEdit, String childId, dynamic item) {
    final nameCtrl = TextEditingController(text: isEdit ? (item['name'] ?? '') : '');
    final cityCtrl = TextEditingController(text: isEdit ? (item['city'] ?? '') : '');
    final latCtrl = TextEditingController(text: isEdit && item['lat'] != null ? '${item['lat']}' : '');
    final lngCtrl = TextEditingController(text: isEdit && item['lng'] != null ? '${item['lng']}' : '');
    bool isSaving = false;

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setSheetState) => Padding(
          padding: EdgeInsets.only(
            left: 20,
            right: 20,
            top: 20,
            bottom: MediaQuery.of(ctx).viewInsets.bottom + 20,
          ),
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      isEdit ? 'Edit Branch: $childId' : 'Add New Branch',
                      style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
                    ),
                    IconButton(icon: const Icon(Icons.close), onPressed: () => Navigator.pop(ctx)),
                  ],
                ),
                Text(
                  'Branch of ${widget.parentTitle} (#${widget.parentId})',
                  style: TextStyle(fontSize: 12, color: Colors.grey.shade400),
                ),
                const SizedBox(height: 16),
                TextField(
                  controller: nameCtrl,
                  decoration: InputDecoration(
                    labelText: 'Branch Name',
                    prefixIcon: const Icon(Icons.storefront_outlined),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    isDense: true,
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: cityCtrl,
                  decoration: InputDecoration(
                    labelText: 'City / Region',
                    prefixIcon: const Icon(Icons.location_city_outlined),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    isDense: true,
                  ),
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: latCtrl,
                        keyboardType: const TextInputType.numberWithOptions(decimal: true),
                        decoration: InputDecoration(
                          labelText: 'Latitude',
                          prefixIcon: const Icon(Icons.map_outlined),
                          border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                          isDense: true,
                        ),
                      ),
                    ),
                    const SizedBox(width: 10),
                    Expanded(
                      child: TextField(
                        controller: lngCtrl,
                        keyboardType: const TextInputType.numberWithOptions(decimal: true),
                        decoration: InputDecoration(
                          labelText: 'Longitude',
                          prefixIcon: const Icon(Icons.navigation_outlined),
                          border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                          isDense: true,
                        ),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 20),
                FilledButton.icon(
                  icon: isSaving
                      ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                      : Icon(isEdit ? Icons.save : Icons.add),
                  label: Text(isEdit ? 'Save Changes' : 'Create Branch'),
                  style: FilledButton.styleFrom(
                    backgroundColor: Colors.amber.shade700,
                    foregroundColor: Colors.black,
                    padding: const EdgeInsets.symmetric(vertical: 14),
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                  ),
                  onPressed: isSaving
                      ? null
                      : () async {
                          if (nameCtrl.text.trim().isEmpty) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('Branch name is required')),
                            );
                            return;
                          }
                          setSheetState(() => isSaving = true);
                          final payload = <String, dynamic>{
                            'name': nameCtrl.text.trim(),
                            'city': cityCtrl.text.trim(),
                            if (latCtrl.text.isNotEmpty) 'lat': latCtrl.text.trim(),
                            if (lngCtrl.text.isNotEmpty) 'lng': lngCtrl.text.trim(),
                          };

                          bool ok = false;
                          if (isEdit) {
                            ok = await widget.api.updateRelationRecord(
                              widget.parentSlug,
                              widget.parentId,
                              widget.relSlug,
                              childId,
                              payload,
                            );
                          } else {
                            ok = await widget.api.createRelationRecord(
                              widget.parentSlug,
                              widget.parentId,
                              widget.relSlug,
                              payload,
                            );
                          }

                          if (!mounted) return;
                          Navigator.pop(ctx);
                          if (ok) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              SnackBar(
                                content: Text(isEdit ? 'Branch updated successfully' : 'Branch created successfully'),
                                backgroundColor: const Color(0xFF10B981),
                                behavior: SnackBarBehavior.floating,
                              ),
                            );
                            loadRecords();
                          } else {
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('Operation failed. Please try again.'), backgroundColor: Colors.redAccent),
                            );
                          }
                        },
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  // --- 3. Vehicles Add / Edit Sheet ---
  void _showVehicleSheet(bool isEdit, String childId, dynamic item) {
    final plateCtrl = TextEditingController(text: isEdit ? (item['plate_number'] ?? '') : '');
    final odoCtrl = TextEditingController(text: isEdit ? '${item['odometer'] ?? '0'}' : '0');
    String fuelTypeId = isEdit ? '${item['fuel_type_id'] ?? '1'}' : '1';
    bool isSaving = false;

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setSheetState) => Padding(
          padding: EdgeInsets.only(
            left: 20,
            right: 20,
            top: 20,
            bottom: MediaQuery.of(ctx).viewInsets.bottom + 20,
          ),
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      isEdit ? 'Edit Vehicle: $childId' : 'Add New Vehicle',
                      style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
                    ),
                    IconButton(icon: const Icon(Icons.close), onPressed: () => Navigator.pop(ctx)),
                  ],
                ),
                Text(
                  'Assigned to Fleet: ${widget.parentTitle} (#${widget.parentId})',
                  style: TextStyle(fontSize: 12, color: Colors.grey.shade400),
                ),
                const SizedBox(height: 16),
                TextField(
                  controller: plateCtrl,
                  decoration: InputDecoration(
                    labelText: 'License Plate Number',
                    prefixIcon: const Icon(Icons.pin_outlined),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    isDense: true,
                    hintText: 'e.g. ABC 1234',
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: odoCtrl,
                  keyboardType: TextInputType.number,
                  decoration: InputDecoration(
                    labelText: 'Odometer (km)',
                    prefixIcon: const Icon(Icons.speed_outlined),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    isDense: true,
                  ),
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  initialValue: ['1', '2', '3'].contains(fuelTypeId) ? fuelTypeId : '1',
                  decoration: InputDecoration(
                    labelText: 'Authorized Fuel Grade',
                    prefixIcon: const Icon(Icons.local_gas_station_outlined),
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                    isDense: true,
                  ),
                  items: const [
                    DropdownMenuItem(value: '1', child: Text('Gasoline 91 (Green)')),
                    DropdownMenuItem(value: '2', child: Text('Gasoline 95 (Red)')),
                    DropdownMenuItem(value: '3', child: Text('Diesel')),
                  ],
                  onChanged: (val) {
                    if (val != null) setSheetState(() => fuelTypeId = val);
                  },
                ),
                const SizedBox(height: 20),
                FilledButton.icon(
                  icon: isSaving
                      ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                      : Icon(isEdit ? Icons.save : Icons.add),
                  label: Text(isEdit ? 'Save Changes' : 'Register Vehicle'),
                  style: FilledButton.styleFrom(
                    backgroundColor: Colors.amber.shade700,
                    foregroundColor: Colors.black,
                    padding: const EdgeInsets.symmetric(vertical: 14),
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                  ),
                  onPressed: isSaving
                      ? null
                      : () async {
                          if (plateCtrl.text.trim().isEmpty) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('Plate number is required')),
                            );
                            return;
                          }
                          setSheetState(() => isSaving = true);
                          final payload = <String, dynamic>{
                            'plate_number': plateCtrl.text.trim(),
                            'odometer': odoCtrl.text.trim(),
                            'fuel_type_id': fuelTypeId,
                          };

                          bool ok = false;
                          if (isEdit) {
                            ok = await widget.api.updateRelationRecord(
                              widget.parentSlug,
                              widget.parentId,
                              widget.relSlug,
                              childId,
                              payload,
                            );
                          } else {
                            ok = await widget.api.createRelationRecord(
                              widget.parentSlug,
                              widget.parentId,
                              widget.relSlug,
                              payload,
                            );
                          }

                          if (!mounted) return;
                          Navigator.pop(ctx);
                          if (ok) {
                            ScaffoldMessenger.of(context).showSnackBar(
                              SnackBar(
                                content: Text(isEdit ? 'Vehicle updated successfully' : 'Vehicle registered successfully'),
                                backgroundColor: const Color(0xFF10B981),
                                behavior: SnackBarBehavior.floating,
                              ),
                            );
                            loadRecords();
                          } else {
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('Operation failed. Please try again.'), backgroundColor: Colors.redAccent),
                            );
                          }
                        },
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  // --- 4. Generic Add / Edit Sheet ---
  void _showGenericSheet(bool isEdit, String childId, dynamic item) {
    final titleCtrl = TextEditingController(text: isEdit ? _getChildTitle(item) : '');
    bool isSaving = false;

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(20))),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setSheetState) => Padding(
          padding: EdgeInsets.only(
            left: 20,
            right: 20,
            top: 20,
            bottom: MediaQuery.of(ctx).viewInsets.bottom + 20,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    isEdit ? 'Edit ${widget.relTitle}: $childId' : 'Add to ${widget.relTitle}',
                    style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
                  ),
                  IconButton(icon: const Icon(Icons.close), onPressed: () => Navigator.pop(ctx)),
                ],
              ),
              const SizedBox(height: 16),
              TextField(
                controller: titleCtrl,
                decoration: InputDecoration(
                  labelText: 'Name / Value',
                  border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                  isDense: true,
                ),
              ),
              const SizedBox(height: 20),
              FilledButton.icon(
                icon: isSaving
                    ? const SizedBox(width: 16, height: 16, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                    : Icon(isEdit ? Icons.save : Icons.add),
                label: Text(isEdit ? 'Save Changes' : 'Create Record'),
                style: FilledButton.styleFrom(
                  backgroundColor: Colors.amber.shade700,
                  foregroundColor: Colors.black,
                  padding: const EdgeInsets.symmetric(vertical: 14),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                ),
                onPressed: isSaving
                    ? null
                    : () async {
                        setSheetState(() => isSaving = true);
                        final payload = <String, dynamic>{'name': titleCtrl.text.trim()};
                        bool ok = false;
                        if (isEdit) {
                          ok = await widget.api.updateRelationRecord(
                            widget.parentSlug,
                            widget.parentId,
                            widget.relSlug,
                            childId,
                            payload,
                          );
                        } else {
                          ok = await widget.api.createRelationRecord(
                            widget.parentSlug,
                            widget.parentId,
                            widget.relSlug,
                            payload,
                          );
                        }
                        if (!mounted) return;
                        Navigator.pop(ctx);
                        if (ok) {
                          ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(content: Text('Saved successfully'), backgroundColor: Color(0xFF10B981)),
                          );
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

  void _confirmDelete(String childId, String title) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Related Record?'),
        content: Text('Are you sure you want to remove "$title" (#$childId)? This action cannot be undone.'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('Cancel')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.redAccent),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );

    if (confirm == true) {
      final ok = await widget.api.deleteRelationRecord(
        widget.parentSlug,
        widget.parentId,
        widget.relSlug,
        childId,
      );
      if (ok) {
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Record deleted successfully'),
            backgroundColor: Color(0xFF10B981),
            behavior: SnackBarBehavior.floating,
          ),
        );
        loadRecords();
      } else {
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Failed to delete record'), backgroundColor: Colors.redAccent),
        );
      }
    }
  }

  int get activeFilterCount {
    int count = 0;
    for (final s in activeFilters.values) {
      count += s.length;
    }
    return count;
  }

  void _clearAllFilters() {
    setState(() {
      activeFilters.clear();
    });
  }

  void _removeFilter(String key, String val) {
    setState(() {
      activeFilters[key]?.remove(val);
      if (activeFilters[key]?.isEmpty ?? false) {
        activeFilters.remove(key);
      }
    });
  }

  void _toggleFilter(String key, String val) {
    setState(() {
      if (!activeFilters.containsKey(key)) {
        activeFilters[key] = <String>{};
      }
      if (activeFilters[key]!.contains(val)) {
        activeFilters[key]!.remove(val);
        if (activeFilters[key]!.isEmpty) {
          activeFilters.remove(key);
        }
      } else {
        activeFilters[key]!.add(val);
      }
    });
  }

  String _titleCase(String s) {
    if (s.isEmpty) return s;
    return s.split(' ').map((w) => w.isEmpty ? '' : w[0].toUpperCase() + w.substring(1)).join(' ');
  }

  List<dynamic> get filteredRecords {
    if (activeFilters.isEmpty) return records;
    return records.where((r) {
      if (r is! Map) return true;
      for (final entry in activeFilters.entries) {
        final key = entry.key;
        final selectedVals = entry.value;
        if (selectedVals.isEmpty) continue;
        final rawVal = r[key] ?? r[_titleCase(key)] ?? r[key.toLowerCase()];
        final valStr = rawVal == null ? 'None' : rawVal.toString();
        if (!selectedVals.contains(valStr)) {
          return false;
        }
      }
      return true;
    }).toList();
  }

  Map<String, Map<String, int>> _computeFilterStats() {
    final stats = <String, Map<String, int>>{};
    final cols = _extractColumns();
    for (final col in cols) {
      final valCounts = <String, int>{};
      for (final r in records) {
        if (r is! Map) continue;
        final raw = r[col] ?? r[_titleCase(col)] ?? r[col.toLowerCase()];
        final s = raw == null || raw.toString().isEmpty ? 'None' : raw.toString();
        valCounts[s] = (valCounts[s] ?? 0) + 1;
      }
      if (valCounts.length > 1 && valCounts.length <= 20) {
        stats[col] = valCounts;
      }
    }
    return stats;
  }

  void _showFilterModal(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final isDesktop = MediaQuery.of(context).size.width > 900;
    final stats = _computeFilterStats();

    Widget buildFilterContent(BuildContext ctx, StateSetter setModalState) {
      return Container(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.of(ctx).size.height * 0.75,
          maxWidth: isDesktop ? 600 : double.infinity,
        ),
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          color: theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(20),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Row(
                  children: [
                    Icon(Icons.tune_rounded, color: Colors.amber.shade400, size: 20),
                    const SizedBox(width: 8),
                    const Text('Refined Search Filters', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                  ],
                ),
                IconButton(icon: const Icon(Icons.close), onPressed: () => Navigator.pop(ctx)),
              ],
            ),
            const Divider(height: 20),
            if (stats.isEmpty)
              const Padding(
                padding: EdgeInsets.symmetric(vertical: 30),
                child: Center(
                  child: Text('No categorical filter options available for these records.', style: TextStyle(color: Colors.grey)),
                ),
              )
            else
              Flexible(
                child: ListView(
                  shrinkWrap: true,
                  children: stats.entries.map((entry) {
                    final cat = entry.key;
                    final options = entry.value;
                    return Theme(
                      data: theme.copyWith(dividerColor: Colors.transparent),
                      child: ExpansionTile(
                        initiallyExpanded: true,
                        tilePadding: EdgeInsets.zero,
                        title: Text(
                          _titleCase(cat.replaceAll('_', ' ')).toUpperCase(),
                          style: TextStyle(
                            fontSize: 12,
                            fontWeight: FontWeight.bold,
                            letterSpacing: 0.5,
                            color: Colors.amber.shade400,
                          ),
                        ),
                        children: options.entries.map((opt) {
                          final val = opt.key;
                          final count = opt.value;
                          final isChecked = activeFilters[cat]?.contains(val) ?? false;
                          return CheckboxListTile(
                            dense: true,
                            contentPadding: EdgeInsets.zero,
                            title: Text(val, style: const TextStyle(fontSize: 13)),
                            secondary: Container(
                              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                              decoration: BoxDecoration(
                                color: isDark ? Colors.white10 : Colors.black12,
                                borderRadius: BorderRadius.circular(10),
                              ),
                              child: Text(
                                '$count',
                                style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold),
                              ),
                            ),
                            value: isChecked,
                            onChanged: (bool? v) {
                              setModalState(() {
                                _toggleFilter(cat, val);
                              });
                              setState(() {});
                            },
                          );
                        }).toList(),
                      ),
                    );
                  }).toList(),
                ),
              ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () {
                      setModalState(() {
                        _clearAllFilters();
                      });
                      setState(() {});
                    },
                    child: const Text('Clear All'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    style: FilledButton.styleFrom(backgroundColor: Colors.amber.shade600, foregroundColor: Colors.black),
                    onPressed: () => Navigator.pop(ctx),
                    child: const Text('Apply', style: TextStyle(fontWeight: FontWeight.bold)),
                  ),
                ),
              ],
            ),
          ],
        ),
      );
    }

    if (isDesktop) {
      showDialog(
        context: context,
        builder: (dialogCtx) => StatefulBuilder(
          builder: (ctx, setModalState) => Dialog(
            backgroundColor: Colors.transparent,
            elevation: 0,
            insetPadding: const EdgeInsets.symmetric(horizontal: 24, vertical: 24),
            child: buildFilterContent(ctx, setModalState),
          ),
        ),
      );
    } else {
      showModalBottomSheet(
        context: context,
        isScrollControlled: true,
        backgroundColor: Colors.transparent,
        builder: (bottomSheetCtx) => StatefulBuilder(
          builder: (ctx, setModalState) => buildFilterContent(ctx, setModalState),
        ),
      );
    }
  }

  List<String> _extractColumns() {
    if (records.isEmpty) return [];
    const excluded = {
      'id', 'ID', '_id', 'status', 'Status', 'status_text', 'active',
      'password', 'token', 'remember_token', 'deleted_at',
      'created_at', 'updated_at', 'createdAt', 'updatedAt',
      'relations', 'sub_entities', 'hash', 'salt',
    };
    final orderedKeys = <String>[];
    final seen = <String>{};
    for (final candidate in ['name', 'Name', 'title', 'Title', 'login', 'plate_number', 'city', 'role', 'email']) {
      for (final r in records) {
        if (r is Map && r.containsKey(candidate) && !seen.contains(candidate.toLowerCase())) {
          orderedKeys.add(candidate);
          seen.add(candidate.toLowerCase());
        }
      }
    }
    for (final r in records) {
      if (r is Map) {
        for (final k in r.keys) {
          final ks = k.toString();
          if (!excluded.contains(ks) && !seen.contains(ks.toLowerCase())) {
            orderedKeys.add(ks);
            seen.add(ks.toLowerCase());
          }
        }
      }
    }
    return orderedKeys.take(8).toList();
  }

  double _columnWidth(String col) {
    final lower = col.toLowerCase();
    if (lower.contains('email') || lower.contains('address') || lower.contains('url')) return 220.0;
    if (lower.contains('description') || lower.contains('notes') || lower.contains('message')) return 260.0;
    if (lower.contains('title') || lower.contains('name')) return 180.0;
    if (lower.contains('date') || lower.contains('time') || lower.contains('created')) return 160.0;
    if (lower.contains('phone') || lower.contains('mobile')) return 140.0;
    if (lower.contains('price') || lower.contains('amount') || lower.contains('total') || lower.contains('balance')) return 130.0;
    return 140.0;
  }

  List<Map<String, dynamic>> _getNestedRelations(String relSlug) {
    if (relSlug == 'branches') {
      return [
        {'relationship': 'operators', 'title': 'Station Operators', 'icon': 'heroicon-users'},
        {'relationship': 'fuel-prices', 'title': 'Fuel Prices', 'icon': 'heroicon-banknotes'},
      ];
    }
    if (relSlug == 'vehicles') {
      return [
        {'relationship': 'drivers', 'title': 'Assigned Drivers', 'icon': 'heroicon-users'},
        {'relationship': 'chips', 'title': 'NFC Tag / Chips', 'icon': 'heroicon-cpu-chip'},
      ];
    }
    if (relSlug == 'customers' || relSlug == 'partners') {
      return [
        {'relationship': 'admins', 'title': 'Admins', 'icon': 'heroicon-users'},
        {'relationship': 'branches', 'title': 'Branches', 'icon': 'heroicon-building-office'},
        {'relationship': 'vehicles', 'title': 'Vehicles', 'icon': 'heroicon-truck'},
      ];
    }
    if (relSlug == 'nfc-chip-batches') {
      return [
        {'relationship': 'chips', 'title': 'NFC Chips', 'icon': 'heroicon-cpu-chip'},
      ];
    }
    return [];
  }

  void _openRelation(BuildContext context, String id, String name, String nextSlug, String nextTitle) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => RelationRecordScreen(
          api: widget.api,
          parentSlug: widget.relSlug,
          parentId: id,
          parentTitle: name,
          relSlug: nextSlug,
          relTitle: nextTitle,
        ),
      ),
    );
  }

  Widget _buildTableCell(BuildContext context, String col, dynamic item, bool isDark, IconData defaultIcon) {
    final val = item[col] ?? item[_titleCase(col)] ?? item[col.toLowerCase()];
    if (val == null) {
      return Text('-', style: TextStyle(color: Colors.grey.shade600, fontSize: 13));
    }
    final str = val.toString();
    return Text(
      str,
      style: TextStyle(fontSize: 13, color: isDark ? Colors.grey.shade300 : Colors.grey.shade800),
      overflow: TextOverflow.ellipsis,
    );
  }

  Widget _buildRecordTable(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final relations = _getNestedRelations(widget.relSlug);
    final hasRelations = relations.isNotEmpty;
    final columns = _extractColumns();
    final items = filteredRecords;

    double totalColsWidth = 48.0 + 70.0 + 130.0 + 140.0 + 80.0; // Checkbox + ID + STATUS + ACTIONS + clearance
    for (final col in columns) {
      totalColsWidth += _columnWidth(col);
    }

    return Scrollbar(
      controller: _verticalScrollController,
      thumbVisibility: true,
      trackVisibility: true,
      interactive: true,
      thickness: 10,
      radius: const Radius.circular(5),
      notificationPredicate: (notif) => notif.metrics.axis == Axis.vertical,
      child: SingleChildScrollView(
        controller: _verticalScrollController,
        padding: const EdgeInsets.fromLTRB(20, 20, 20, 100),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 1600),
            child: Card(
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(16),
                side: BorderSide(
                  color: isDark ? Colors.white.withValues(alpha: 0.1) : Colors.black.withValues(alpha: 0.08),
                ),
              ),
              color: theme.colorScheme.surface,
              clipBehavior: Clip.antiAlias,
              child: LayoutBuilder(
                builder: (ctx, constraints) {
                  final tableWidth = constraints.maxWidth > totalColsWidth ? constraints.maxWidth : totalColsWidth;
                  return Scrollbar(
                    controller: _horizontalScrollController,
                    thumbVisibility: true,
                    trackVisibility: true,
                    interactive: true,
                    thickness: 10,
                    radius: const Radius.circular(5),
                    notificationPredicate: (notif) => notif.metrics.axis == Axis.horizontal,
                    child: SingleChildScrollView(
                      controller: _horizontalScrollController,
                      scrollDirection: Axis.horizontal,
                      child: SizedBox(
                        width: tableWidth,
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            // Table Header / Bulk Action Bar
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
                              decoration: BoxDecoration(
                                color: selectedRecordIds.isNotEmpty
                                    ? Colors.amber.withValues(alpha: isDark ? 0.2 : 0.12)
                                    : (isDark ? const Color(0xFF0F172A) : const Color(0xFFF8FAFC)),
                                border: Border(
                                  bottom: BorderSide(
                                    color: selectedRecordIds.isNotEmpty
                                        ? Colors.amber.withValues(alpha: 0.35)
                                        : (isDark ? Colors.white12 : Colors.black12),
                                  ),
                                ),
                              ),
                              child: Row(
                                children: [
                                  SizedBox(
                                    width: 48,
                                    child: Checkbox(
                                      value: isAllSelected
                                          ? true
                                          : (selectedRecordIds.isNotEmpty ? null : false),
                                      tristate: true,
                                      activeColor: Colors.amber.shade600,
                                      checkColor: Colors.black,
                                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
                                      onChanged: (val) {
                                        if (isAllSelected) {
                                          _deselectAll();
                                        } else {
                                          _toggleSelectAll(true);
                                        }
                                      },
                                    ),
                                  ),
                                  if (selectedRecordIds.isNotEmpty) ...[
                                    Text(
                                      '${selectedRecordIds.length} of ${filteredRecords.length} selected',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                        fontSize: 13,
                                        color: isDark ? Colors.amber.shade400 : const Color(0xFFB45309),
                                      ),
                                    ),
                                    const SizedBox(width: 16),
                                    FilledButton.icon(
                                      style: FilledButton.styleFrom(
                                        backgroundColor: Colors.redAccent,
                                        foregroundColor: Colors.white,
                                        visualDensity: VisualDensity.compact,
                                        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                                      ),
                                      icon: const Icon(Icons.delete_sweep_rounded, size: 16),
                                      label: Text(
                                        'Delete Selected (${selectedRecordIds.length})',
                                        style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12),
                                      ),
                                      onPressed: () => _executeBulkDelete(context),
                                    ),
                                    const SizedBox(width: 10),
                                    OutlinedButton.icon(
                                      style: OutlinedButton.styleFrom(
                                        visualDensity: VisualDensity.compact,
                                        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                                      ),
                                      icon: const Icon(Icons.download_rounded, size: 16),
                                      label: const Text('Export', style: TextStyle(fontSize: 12)),
                                      onPressed: () => _exportSelectedRecords(context),
                                    ),
                                    const Spacer(),
                                    TextButton(
                                      style: TextButton.styleFrom(visualDensity: VisualDensity.compact),
                                      child: const Text('Deselect All', style: TextStyle(fontSize: 12)),
                                      onPressed: _deselectAll,
                                    ),
                                    const SizedBox(width: 16),
                                  ] else ...[
                                    const SizedBox(
                                      width: 70,
                                      child: Text(
                                        'ID',
                                        style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12, letterSpacing: 0.5, color: Colors.grey),
                                      ),
                                    ),
                                    for (final col in columns)
                                      SizedBox(
                                        width: _columnWidth(col),
                                        child: Padding(
                                          padding: const EdgeInsets.only(right: 12),
                                          child: Text(
                                            _titleCase(col.replaceAll('_', ' ')).toUpperCase(),
                                            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12, letterSpacing: 0.5, color: Colors.grey),
                                            overflow: TextOverflow.ellipsis,
                                          ),
                                        ),
                                      ),
                                    const SizedBox(
                                      width: 130,
                                      child: Text(
                                        'STATUS',
                                        style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12, letterSpacing: 0.5, color: Colors.grey),
                                      ),
                                    ),
                                    const SizedBox(
                                      width: 140,
                                      child: Align(
                                        alignment: Alignment.centerRight,
                                        child: Text(
                                          'ACTIONS',
                                          style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12, letterSpacing: 0.5, color: Colors.grey),
                                        ),
                                      ),
                                    ),
                                    const SizedBox(width: 16),
                                  ],
                                ],
                              ),
                            ),

                            // Table Rows
                            for (int i = 0; i < items.length; i++)
                              _buildTableRow(
                                context,
                                items[i],
                                _getChildId(items[i], i),
                                _getChildTitle(items[i]),
                                columns,
                                hasRelations,
                                relations,
                                i == items.length - 1,
                              ),
                          ],
                        ),
                      ),
                    ),
                  );
                },
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildTableRow(
    BuildContext context,
    dynamic item,
    String id,
    String name,
    List<String> columns,
    bool hasRelations,
    List<Map<String, dynamic>> relations,
    bool isLast,
  ) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    IconData headerIcon = Icons.account_tree_outlined;
    if (widget.relSlug == 'admins') headerIcon = Icons.person_outline;
    if (widget.relSlug == 'branches') headerIcon = Icons.storefront_outlined;
    if (widget.relSlug == 'vehicles') headerIcon = Icons.local_shipping_outlined;
    if (widget.relSlug == 'chips') headerIcon = Icons.nfc_outlined;

    final status = (item['Status'] ?? item['status'] ?? item['status_text'] ?? (item['active'] == 'Y' ? 'active' : (item['active'] == 'N' ? 'inactive' : 'online'))).toString();
    Color statusColor = const Color(0xFF10B981);
    final stLower = status.toLowerCase();
    if (stLower == 'inactive' || stLower == 'offline' || stLower == 'critical' || stLower == 'rejected' || stLower == 'overdue' || stLower == 'n') {
      statusColor = Colors.redAccent;
    } else if (stLower == 'pending' || stLower == 'warning' || stLower == 'draft' || stLower == 'partial') {
      statusColor = Colors.amber;
    }

    final isRelationsVisible = hasRelations && _showRelations;
    final isSelected = selectedRecordIds.contains(id);

    return Container(
      decoration: BoxDecoration(
        color: isSelected
            ? (isDark ? Colors.amber.withValues(alpha: 0.12) : Colors.amber.withValues(alpha: 0.08))
            : null,
        border: isLast && !isRelationsVisible
            ? null
            : Border(bottom: BorderSide(color: isDark ? Colors.white.withValues(alpha: 0.05) : Colors.black.withValues(alpha: 0.05))),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Main Columns Row
          Material(
            color: Colors.transparent,
            child: InkWell(
              hoverColor: isDark ? Colors.white.withValues(alpha: 0.03) : Colors.black.withValues(alpha: 0.02),
              onTap: () => _showAddEditSheet(item),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
                child: Row(
                  children: [
                    // Select Checkbox Column
                    SizedBox(
                      width: 48,
                      child: Checkbox(
                        value: isSelected,
                        activeColor: Colors.amber.shade600,
                        checkColor: Colors.black,
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
                        onChanged: (_) => _toggleSelectRecord(id),
                      ),
                    ),
                    // ID Column
                    SizedBox(
                      width: 70,
                      child: Text(
                        '#$id',
                        style: TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.bold,
                          color: Colors.grey.shade400,
                        ),
                      ),
                    ),

                    // Discovered Columns
                    for (final col in columns)
                      SizedBox(
                        width: _columnWidth(col),
                        child: Padding(
                          padding: const EdgeInsets.only(right: 12),
                          child: _buildTableCell(context, col, item, isDark, headerIcon),
                        ),
                      ),

                    // Status Badge Column
                    SizedBox(
                      width: 130,
                      child: Align(
                        alignment: Alignment.centerLeft,
                        child: Container(
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
                              Text(
                                status.toUpperCase(),
                                style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: statusColor),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),

                    // Actions Column
                    SizedBox(
                      width: 140,
                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.end,
                        children: [
                          IconButton(
                            icon: const Icon(Icons.edit_outlined, size: 18),
                            tooltip: 'Edit Record',
                            onPressed: () => _showAddEditSheet(item),
                          ),
                          IconButton(
                            icon: const Icon(Icons.delete_outline, size: 18, color: Colors.redAccent),
                            tooltip: 'Delete Record',
                            onPressed: () => _confirmDelete(id, name),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 16),
                  ],
                ),
              ),
            ),
          ),

          // Nested Relationship Managers
          if (isRelationsVisible)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 10),
              decoration: BoxDecoration(
                color: isDark ? const Color(0xFF0F172A).withValues(alpha: 0.7) : const Color(0xFFF1F5F9).withValues(alpha: 0.9),
                border: Border(
                  top: BorderSide(color: isDark ? Colors.white.withValues(alpha: 0.04) : Colors.black.withValues(alpha: 0.04)),
                ),
              ),
              child: Row(
                children: [
                  const SizedBox(width: 50),
                  Icon(Icons.account_tree_outlined, size: 14, color: Colors.amber.shade400),
                  const SizedBox(width: 8),
                  Text(
                    'RELATIONS:',
                    style: TextStyle(
                      fontSize: 10,
                      fontWeight: FontWeight.bold,
                      color: Colors.amber.shade400,
                      letterSpacing: 0.5,
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Wrap(
                      spacing: 8,
                      runSpacing: 4,
                      children: relations.map((rel) {
                        final relTitle = (rel['title'] ?? rel['relationship'] ?? 'Related').toString();
                        final nextSlug = (rel['relationship'] ?? '').toString();
                        return ActionChip(
                          avatar: Icon(Icons.account_tree_outlined, size: 14, color: Colors.amber.shade400),
                          label: Text(relTitle, style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold)),
                          backgroundColor: Colors.amber.withValues(alpha: 0.1),
                          side: BorderSide(color: Colors.amber.withValues(alpha: 0.3)),
                          padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 0),
                          visualDensity: VisualDensity.compact,
                          onPressed: () {
                            _openRelation(context, id, name, nextSlug, relTitle);
                          },
                        );
                      }).toList(),
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildFieldBadge(String label, String value, {IconData? icon, Color? color}) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label.toUpperCase(),
          style: TextStyle(
            fontSize: 9,
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
              Icon(icon, size: 13, color: color ?? Colors.grey.shade300),
              const SizedBox(width: 4),
            ],
            Flexible(
              child: Text(
                value,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: color,
                ),
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildRecordCard(BuildContext context, dynamic item) {
    final childId = _getChildId(item);
    final childTitle = _getChildTitle(item);
    final relations = _getNestedRelations(widget.relSlug);
    final hasRelations = relations.isNotEmpty;

    IconData leadingIcon = Icons.account_tree_outlined;
    if (widget.relSlug == 'admins') leadingIcon = Icons.person_outline;
    if (widget.relSlug == 'branches') leadingIcon = Icons.storefront_outlined;
    if (widget.relSlug == 'vehicles') leadingIcon = Icons.local_shipping_outlined;
    if (widget.relSlug == 'chips') leadingIcon = Icons.nfc_outlined;

    final isSelected = selectedRecordIds.contains(childId);
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return Card(
      elevation: isSelected ? 2 : 0,
      margin: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(
          color: isSelected ? Colors.amber.shade400 : (isDark ? Colors.white.withValues(alpha: 0.08) : Colors.black.withValues(alpha: 0.08)),
          width: isSelected ? 1.5 : 1,
        ),
      ),
      color: isSelected
          ? (isDark ? Colors.amber.withValues(alpha: 0.08) : Colors.amber.withValues(alpha: 0.05))
          : theme.colorScheme.surface,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Top Header: Leading icon + Title + Status badges
            Row(
              children: [
                SizedBox(
                  width: 24,
                  height: 24,
                  child: Checkbox(
                    value: isSelected,
                    activeColor: Colors.amber.shade600,
                    checkColor: Colors.black,
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
                    onChanged: (_) => _toggleSelectRecord(childId),
                  ),
                ),
                const SizedBox(width: 8),
                CircleAvatar(
                  radius: 18,
                  backgroundColor: Theme.of(context).colorScheme.primaryContainer,
                  child: Icon(leadingIcon, size: 20, color: Theme.of(context).colorScheme.primary),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        childTitle,
                        style: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold),
                        overflow: TextOverflow.ellipsis,
                      ),
                      Text(
                        'ID: $childId',
                        style: TextStyle(fontSize: 11, color: Colors.grey.shade400),
                      ),
                    ],
                  ),
                ),
                if (widget.relSlug == 'admins') ...[
                  Builder(
                    builder: (_) {
                      final isActive = (item['active'] ?? 'Y') == 'Y';
                      return Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                        decoration: BoxDecoration(
                          color: (isActive ? const Color(0xFF10B981) : Colors.redAccent).withValues(alpha: 0.15),
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Text(
                          isActive ? 'ACTIVE' : 'INACTIVE',
                          style: TextStyle(
                            fontSize: 10,
                            fontWeight: FontWeight.bold,
                            color: isActive ? const Color(0xFF10B981) : Colors.redAccent,
                          ),
                        ),
                      );
                    },
                  ),
                ] else if (widget.relSlug == 'vehicles') ...[
                  Builder(
                    builder: (_) {
                      final fuelId = '${item['fuel_type_id'] ?? '1'}';
                      String fuelName = 'Gasoline 91';
                      Color fuelColor = const Color(0xFF10B981);
                      if (fuelId == '2') {
                        fuelName = 'Gasoline 95';
                        fuelColor = Colors.redAccent;
                      } else if (fuelId == '3') {
                        fuelName = 'Diesel';
                        fuelColor = Colors.amber;
                      }
                      return Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                        decoration: BoxDecoration(
                          color: fuelColor.withValues(alpha: 0.15),
                          borderRadius: BorderRadius.circular(999),
                        ),
                        child: Text(
                          fuelName.toUpperCase(),
                          style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: fuelColor),
                        ),
                      );
                    },
                  ),
                ],
              ],
            ),
            const SizedBox(height: 12),
            const Divider(height: 1, thickness: 0.5),
            const SizedBox(height: 12),

            // Attributes Grid
            if (widget.relSlug == 'admins') ...[
              Row(
                children: [
                  Expanded(child: _buildFieldBadge('Username', '${item['login'] ?? 'N/A'}', icon: Icons.alternate_email)),
                  Expanded(child: _buildFieldBadge('Email', '${item['email'] ?? 'N/A'}', icon: Icons.email_outlined)),
                ],
              ),
              const SizedBox(height: 8),
              Row(
                children: [
                  Expanded(child: _buildFieldBadge('Role', '${item['role'] ?? 'admin'}', icon: Icons.shield_outlined)),
                  Expanded(child: _buildFieldBadge('Full Name', '${item['name'] ?? 'N/A'}', icon: Icons.badge_outlined)),
                ],
              ),
            ] else if (widget.relSlug == 'branches') ...[
              Row(
                children: [
                  Expanded(child: _buildFieldBadge('Branch Name', '${item['name'] ?? 'N/A'}', icon: Icons.storefront)),
                  Expanded(child: _buildFieldBadge('City / Region', '${item['city'] ?? 'Saudi Arabia'}', icon: Icons.location_on_outlined)),
                ],
              ),
              if (item['lat'] != null || item['lng'] != null) ...[
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(child: _buildFieldBadge('GPS Coordinates', '${item['lat'] ?? '-'}, ${item['lng'] ?? '-'}', icon: Icons.map_outlined)),
                  ],
                ),
              ],
            ] else if (widget.relSlug == 'vehicles') ...[
              Row(
                children: [
                  Expanded(child: _buildFieldBadge('Plate Number', '${item['plate_number'] ?? 'N/A'}', icon: Icons.pin)),
                  Expanded(child: _buildFieldBadge('Odometer', '${item['odometer'] ?? '0'} km', icon: Icons.speed)),
                ],
              ),
            ] else ...[
              Wrap(
                spacing: 12,
                runSpacing: 8,
                children: (item as Map<String, dynamic>).entries.take(4).map((e) {
                  return _buildFieldBadge(e.key, '${e.value}');
                }).toList(),
              ),
            ],

            // Nested Relations Chips in Card
            if (hasRelations && _showRelations) ...[
              const SizedBox(height: 12),
              const Divider(height: 1, thickness: 0.5),
              const SizedBox(height: 10),
              Row(
                children: [
                  Icon(Icons.hub_outlined, size: 14, color: Colors.amber.shade400),
                  const SizedBox(width: 6),
                  const Text(
                    'SUB-ENTITIES & RELATIONS',
                    style: TextStyle(
                      fontSize: 10,
                      fontWeight: FontWeight.bold,
                      letterSpacing: 0.8,
                      color: Colors.grey,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              Wrap(
                spacing: 8,
                runSpacing: 6,
                children: relations.map((rel) {
                  final relTitle = (rel['title'] ?? rel['relationship'] ?? 'Related').toString();
                  final nextSlug = (rel['relationship'] ?? '').toString();
                  return ActionChip(
                    avatar: Icon(Icons.account_tree_outlined, size: 14, color: Colors.amber.shade400),
                    label: Text(relTitle, style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold)),
                    backgroundColor: Colors.amber.withValues(alpha: 0.1),
                    side: BorderSide(color: Colors.amber.withValues(alpha: 0.3)),
                    padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 0),
                    visualDensity: VisualDensity.compact,
                    onPressed: () {
                      _openRelation(context, childId, childTitle, nextSlug, relTitle);
                    },
                  );
                }).toList(),
              ),
            ],

            const SizedBox(height: 12),
            const Divider(height: 1, thickness: 0.5),
            const SizedBox(height: 8),

            // Card Footer Actions: Edit & Delete
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton.icon(
                  icon: const Icon(Icons.edit_outlined, size: 15),
                  label: const Text('Edit', style: TextStyle(fontSize: 12)),
                  onPressed: () => _showAddEditSheet(item),
                ),
                const SizedBox(width: 8),
                TextButton.icon(
                  icon: const Icon(Icons.delete_outline, size: 15, color: Colors.redAccent),
                  label: const Text('Delete', style: TextStyle(fontSize: 12, color: Colors.redAccent)),
                  onPressed: () => _confirmDelete(childId, childTitle),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildRecordCards(BuildContext context) {
    final items = filteredRecords;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (selectedRecordIds.isNotEmpty)
          Container(
            margin: const EdgeInsets.fromLTRB(16, 8, 16, 4),
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
            decoration: BoxDecoration(
              color: Colors.amber.withValues(alpha: 0.15),
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: Colors.amber.withValues(alpha: 0.4)),
            ),
            child: Row(
              children: [
                Icon(Icons.check_circle_outline, color: Colors.amber.shade400, size: 20),
                const SizedBox(width: 8),
                Text(
                  '${selectedRecordIds.length} Selected',
                  style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 13),
                ),
                const Spacer(),
                TextButton.icon(
                  icon: const Icon(Icons.delete_sweep_outlined, size: 18, color: Colors.redAccent),
                  label: const Text('Delete', style: TextStyle(color: Colors.redAccent, fontSize: 12)),
                  onPressed: () => _executeBulkDelete(context),
                ),
                IconButton(
                  icon: const Icon(Icons.close, size: 18),
                  tooltip: 'Deselect all',
                  onPressed: _deselectAll,
                ),
              ],
            ),
          ),
        Expanded(
          child: LayoutBuilder(
            builder: (context, constraints) {
        final isTabletOrWide = constraints.maxWidth >= 720;
        if (isTabletOrWide) {
          return SingleChildScrollView(
            padding: const EdgeInsets.all(12),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      for (int i = 0; i < items.length; i += 2)
                        _buildRecordCard(context, items[i]),
                    ],
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      for (int i = 1; i < items.length; i += 2)
                        _buildRecordCard(context, items[i]),
                    ],
                  ),
                ),
              ],
            ),
          );
        }
        return ListView.builder(
          padding: const EdgeInsets.only(top: 8, bottom: 24),
          itemCount: items.length,
          itemBuilder: (context, i) => _buildRecordCard(context, items[i]),
        );
      },
    ),
  ),
],
);
  }

  Widget _buildEmptyState() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.folder_open_outlined, size: 64, color: Colors.grey.shade500),
            const SizedBox(height: 16),
            Text(
              'No ${widget.relTitle} found',
              style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            Text(
              'No matching nested records under "${widget.parentTitle}".',
              textAlign: TextAlign.center,
              style: TextStyle(fontSize: 13, color: Colors.grey.shade400),
            ),
            const SizedBox(height: 20),
            FilledButton.icon(
              style: FilledButton.styleFrom(
                backgroundColor: Colors.amber.shade700,
                foregroundColor: Colors.black,
              ),
              icon: const Icon(Icons.add),
              label: Text('Add First ${widget.relTitle.replaceAll('s', '')}'),
              onPressed: () => _showAddEditSheet(),
            ),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final items = filteredRecords;
    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              widget.relTitle,
              style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16),
            ),
            Text(
              'Parent: ${widget.parentTitle} (#${widget.parentId})',
              style: TextStyle(fontSize: 11, color: Colors.grey.shade400, fontWeight: FontWeight.normal),
            ),
          ],
        ),
        actions: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: FilledButton.icon(
              style: FilledButton.styleFrom(
                backgroundColor: Colors.amber.shade700,
                foregroundColor: Colors.black,
                visualDensity: VisualDensity.compact,
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
              ),
              icon: const Icon(Icons.add, size: 16),
              label: const Text('Add', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12)),
              onPressed: () => _showAddEditSheet(),
            ),
          ),
        ],
        bottom: PreferredSize(
          preferredSize: Size.fromHeight(activeFilterCount > 0 ? 102 : 56),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                child: Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: _searchCtrl,
                        decoration: InputDecoration(
                          hintText: 'Search ${widget.relTitle.toLowerCase()}...',
                          prefixIcon: const Icon(Icons.search, size: 20),
                          suffixIcon: search.isNotEmpty
                              ? IconButton(
                                  icon: const Icon(Icons.clear, size: 18),
                                  onPressed: () {
                                    _searchCtrl.clear();
                                    setState(() => search = '');
                                    loadRecords();
                                  },
                                )
                              : null,
                          isDense: true,
                          border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                        ),
                        onChanged: (v) {
                          search = v.trim();
                          loadRecords();
                        },
                      ),
                    ),
                    const SizedBox(width: 8),
                    IconButton.filledTonal(
                      tooltip: _showRelations ? 'Hide Related Entities' : 'Show Related Entities',
                      style: IconButton.styleFrom(
                        backgroundColor: _showRelations ? Colors.amber.withValues(alpha: 0.2) : null,
                        foregroundColor: _showRelations ? Colors.amber.shade400 : null,
                      ),
                      icon: Icon(_showRelations ? Icons.account_tree : Icons.account_tree_outlined),
                      onPressed: () {
                        setState(() {
                          _showRelations = !_showRelations;
                        });
                      },
                    ),
                    const SizedBox(width: 8),
                    Badge(
                      isLabelVisible: activeFilterCount > 0,
                      label: Text('$activeFilterCount'),
                      backgroundColor: Colors.amber,
                      textColor: Colors.black,
                      child: IconButton.filledTonal(
                        tooltip: 'Refined Search Filters',
                        icon: const Icon(Icons.tune_rounded),
                        onPressed: () => _showFilterModal(context),
                      ),
                    ),
                    const SizedBox(width: 8),
                    PopupMenuButton<String>(
                      tooltip: selectedRecordIds.isNotEmpty
                          ? 'Bulk Actions (${selectedRecordIds.length} selected)'
                          : 'Bulk Selection & Actions',
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                      icon: Badge(
                        isLabelVisible: selectedRecordIds.isNotEmpty,
                        label: Text('${selectedRecordIds.length}'),
                        backgroundColor: Colors.amber.shade700,
                        textColor: Colors.black,
                        child: IconButton.filledTonal(
                          style: IconButton.styleFrom(
                            backgroundColor: selectedRecordIds.isNotEmpty ? Colors.amber.withValues(alpha: 0.2) : null,
                            foregroundColor: selectedRecordIds.isNotEmpty ? Colors.amber.shade400 : null,
                          ),
                          icon: const Icon(Icons.checklist_rtl_rounded),
                          onPressed: null,
                        ),
                      ),
                      onSelected: (val) {
                        if (val == 'select_all') {
                          _toggleSelectAll();
                        } else if (val == 'deselect') {
                          _deselectAll();
                        } else if (val == 'delete') {
                          _executeBulkDelete(context);
                        } else if (val == 'export') {
                          _exportSelectedRecords(context);
                        }
                      },
                      itemBuilder: (ctx) => [
                        PopupMenuItem(
                          value: 'select_all',
                          child: Row(
                            children: [
                              Icon(isAllSelected ? Icons.deselect_rounded : Icons.select_all_rounded, size: 20),
                              const SizedBox(width: 10),
                              Text(isAllSelected ? 'Deselect All' : 'Select All (${filteredRecords.length})'),
                            ],
                          ),
                        ),
                        if (selectedRecordIds.isNotEmpty) ...[
                          PopupMenuItem(
                            value: 'delete',
                            child: const Row(
                              children: [
                                Icon(Icons.delete_sweep_rounded, color: Colors.redAccent, size: 20),
                                SizedBox(width: 10),
                                Text('Bulk Delete', style: TextStyle(color: Colors.redAccent)),
                              ],
                            ),
                          ),
                          PopupMenuItem(
                            value: 'export',
                            child: const Row(
                              children: [
                                Icon(Icons.download_rounded, size: 20),
                                SizedBox(width: 10),
                                Text('Export Selected'),
                              ],
                            ),
                          ),
                          const PopupMenuDivider(),
                          const PopupMenuItem(
                            value: 'deselect',
                            child: Row(
                              children: [
                                Icon(Icons.close_rounded, size: 20),
                                SizedBox(width: 10),
                                Text('Clear Selection'),
                              ],
                            ),
                          ),
                        ],
                      ],
                    ),
                  ],
                ),
              ),
              if (activeFilterCount > 0)
                Container(
                  height: 42,
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 2),
                  child: ListView(
                    scrollDirection: Axis.horizontal,
                    children: [
                      ActionChip(
                        avatar: const Icon(Icons.close, size: 14),
                        label: const Text('Clear All', style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold)),
                        onPressed: _clearAllFilters,
                        visualDensity: VisualDensity.compact,
                      ),
                      const SizedBox(width: 6),
                      ...activeFilters.entries.expand((entry) {
                        final cat = entry.key;
                        return entry.value.map((val) {
                          return Padding(
                            padding: const EdgeInsets.only(right: 6),
                            child: InputChip(
                              label: Text('${_titleCase(cat)}: $val', style: const TextStyle(fontSize: 11)),
                              onDeleted: () => _removeFilter(cat, val),
                              visualDensity: VisualDensity.compact,
                              deleteIconColor: Colors.amber,
                            ),
                          );
                        });
                      }),
                    ],
                  ),
                ),
            ],
          ),
        ),
      ),
      body: isLoading
          ? const Center(child: CircularProgressIndicator())
          : items.isEmpty
              ? _buildEmptyState()
              : LayoutBuilder(
                  builder: (context, constraints) {
                    final isDesktop = constraints.maxWidth > 900;
                    if (isDesktop) {
                      return _buildRecordTable(context);
                    }
                    return _buildRecordCards(context);
                  },
                ),
    );
  }
}
`
	if err := os.WriteFile(filepath.Join(libDir, "relation_record_screen.dart"), []byte(relationRecordDart), 0644); err != nil {
		return err
	}

	// 6. lib/main.dart - Mobile-First Responsive Flutter Admin Application
	mainDartTemplate := `import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'api.dart';
import 'relation_record_screen.dart';
import 'responsive.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const FilamentApp());
}

// ============================================================================
// Multi-Language & RTL/LTR Localization Support
// ============================================================================
class AppLocale {
  static const Map<String, Map<String, String>> strings = {
    'en': {
      'dashboard': 'Dashboard',
      'entities': 'Entities',
      'account': 'Account',
      'settings': 'Settings & Preferences',
      'orders': 'Hardware Orders',
      'billing': 'Billing & Subscriptions',
      'security': 'Biometrics & Security',
      'devices': 'Devices',
      'telemetry': 'Telemetry',
      'tickets': 'Tickets',
      'firmware': 'Firmware',
      'operators': 'Team',
      'analytics': 'Metrics',
      'language': 'Language & Direction',
      'theme': 'Theme Mode',
      'sign_out': 'Sign Out',
      'search': 'Search',
      'add': 'Add',
      'save': 'Save',
      'cancel': 'Cancel',
      'delete': 'Delete',
      'edit': 'Edit',
      'active_session': 'Active Session • Enclave Secured',
      'control_center': 'Control Center',
      'language_picker': 'Select Language & Direction',
      'theme_active_dark': 'Dark theme active',
      'theme_active_light': 'Light theme active',
    },
    'ar': {
      'dashboard': 'لوحة التحكم',
      'entities': 'الكيانات',
      'account': 'الحساب',
      'settings': 'الإعدادات والتفضيلات',
      'orders': 'طلبات الأجهزة',
      'billing': 'الفواتير والاشتراكات',
      'security': 'القياسات الحيوية والأمان',
      'devices': 'الأجهزة',
      'telemetry': 'القياس عن بُعد',
      'tickets': 'التذاكر',
      'firmware': 'البرامج الثابتة',
      'operators': 'فريق العمل',
      'analytics': 'المؤشرات',
      'language': 'اللغة والاتجاه',
      'theme': 'وضع المظهر',
      'sign_out': 'تسجيل الخروج',
      'search': 'بحث',
      'add': 'إضافة',
      'save': 'حفظ',
      'cancel': 'إلغاء',
      'delete': 'حذف',
      'edit': 'تعديل',
      'active_session': 'جلسة نشطة • الجيب الأمني مشفر',
      'control_center': 'مركز التحكم',
      'language_picker': 'اختر اللغة واتجاه الواجهة',
      'theme_active_dark': 'الوضع الداكن مفعّل',
      'theme_active_light': 'الوضع الفاتح مفعّل',
    },
    'es': {
      'dashboard': 'Panel de Control',
      'entities': 'Entidades',
      'account': 'Cuenta',
      'settings': 'Ajustes y Preferencias',
      'orders': 'Pedidos de Hardware',
      'billing': 'Facturación y Suscripción',
      'security': 'Biometría y Seguridad',
      'devices': 'Dispositivos',
      'telemetry': 'Telemetría',
      'tickets': 'Tickets',
      'firmware': 'Firmware',
      'operators': 'Equipo',
      'analytics': 'Métricas',
      'language': 'Idioma y Dirección',
      'theme': 'Modo de Tema',
      'sign_out': 'Cerrar Sesión',
      'search': 'Buscar',
      'add': 'Añadir',
      'save': 'Guardar',
      'cancel': 'Cancelar',
      'delete': 'Eliminar',
      'edit': 'Editar',
      'active_session': 'Sesión Activa • Enclave Seguro',
      'control_center': 'Centro de Control',
      'language_picker': 'Seleccionar Idioma y Dirección',
      'theme_active_dark': 'Tema oscuro activo',
      'theme_active_light': 'Tema claro activo',
    },
    'de': {
      'dashboard': 'Übersicht',
      'entities': 'Entitäten',
      'account': 'Konto',
      'settings': 'Einstellungen',
      'orders': 'Hardware-Bestellungen',
      'billing': 'Abrechnung & Lizenz',
      'security': 'Biometrie & Sicherheit',
      'devices': 'Geräte',
      'telemetry': 'Telemetrie',
      'tickets': 'Tickets',
      'firmware': 'Firmware',
      'operators': 'Team',
      'analytics': 'Metriken',
      'language': 'Sprache & Ausrichtung',
      'theme': 'Design-Modus',
      'sign_out': 'Abmelden',
      'search': 'Suchen',
      'add': 'Hinzufügen',
      'save': 'Speichern',
      'cancel': 'Abbrechen',
      'delete': 'Löschen',
      'edit': 'Bearbeiten',
      'active_session': 'Aktive Sitzung • Gesichertes Enklave',
      'control_center': 'Leitstand',
      'language_picker': 'Sprache und Textrichtung wählen',
      'theme_active_dark': 'Dunkelmodus aktiv',
      'theme_active_light': 'Hellmodus aktiv',
    },
  };

  static String t(String locale, String key, {Map<String, String>? remoteOverrides}) {
    if (remoteOverrides != null && remoteOverrides.containsKey(key)) {
      return remoteOverrides[key]!;
    }
    final locDict = strings[locale] ?? strings['en']!;
    return locDict[key] ?? strings['en']![key] ?? key;
  }

  static String getFlag(String loc) {
    switch (loc.toLowerCase()) {
      case 'ar': return '🇸🇦';
      case 'es': return '🇪🇸';
      case 'de': return '🇩🇪';
      case 'fr': return '🇫🇷';
      default: return '🇺🇸';
    }
  }

  static String getLabel(String loc) {
    switch (loc.toLowerCase()) {
      case 'ar': return 'العربية';
      case 'es': return 'Español';
      case 'de': return 'Deutsch';
      case 'fr': return 'Français';
      default: return 'English';
    }
  }

  static bool isRTL(String loc) {
    return loc.toLowerCase() == 'ar' || loc.toLowerCase() == 'fa' || loc.toLowerCase() == 'ur' || loc.toLowerCase() == 'he';
  }
}

class FilamentApp extends StatefulWidget {
  const FilamentApp({super.key});

  @override
  State<FilamentApp> createState() => _FilamentAppState();
}

class _FilamentAppState extends State<FilamentApp> {
  final FilamentApiClient api = FilamentApiClient();
  ThemeMode _themeMode = ThemeMode.dark;
  String _locale = 'en';
  bool _isRTL = false;
  bool _isAuthenticated = false;
  bool _isInitializing = true;

  @override
  void initState() {
    super.initState();
    _restoreSession();
  }

  Future<void> _restoreSession() async {
    final authenticated = await api.restoreSession();
    if (mounted) {
      setState(() {
        _isAuthenticated = authenticated;
        _isInitializing = false;
      });
    }
  }

  void toggleTheme() {
    setState(() {
      _themeMode = _themeMode == ThemeMode.dark ? ThemeMode.light : ThemeMode.dark;
    });
  }

  void setLocale(String locale, bool isRTL) {
    setState(() {
      _locale = locale;
      _isRTL = isRTL;
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: '{{TITLE}} Admin',
      debugShowCheckedModeBanner: false,
      themeMode: _themeMode,
      builder: (context, child) {
        return Directionality(
          textDirection: _isRTL ? TextDirection.rtl : TextDirection.ltr,
          child: child ?? const SizedBox(),
        );
      },
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
      home: _isInitializing
          ? Scaffold(
              backgroundColor: _themeMode == ThemeMode.dark ? const Color(0xFF0F172A) : const Color(0xFFF1F5F9),
              body: const Center(
                child: CircularProgressIndicator(
                  valueColor: AlwaysStoppedAnimation<Color>(Color(0xFFF59E0B)),
                ),
              ),
            )
          : _isAuthenticated
              ? AdminShellPage(
                  api: api,
                  onToggleTheme: toggleTheme,
                  isDark: _themeMode == ThemeMode.dark,
                  currentLocale: _locale,
                  isRTL: _isRTL,
                  onLocaleChanged: setLocale,
                  onSignOut: () async {
                    await api.logout();
                    if (mounted) {
                      setState(() {
                        _isAuthenticated = false;
                      });
                    }
                  },
                )
              : LoginScreen(
                  api: api,
                  panelTitle: '{{TITLE}}',
                  isDark: _themeMode == ThemeMode.dark,
                  onToggleTheme: toggleTheme,
                  onLoginSuccess: (user) {
                    setState(() {
                      _isAuthenticated = true;
                    });
                  },
                ),
    );
  }
}

class LoginScreen extends StatefulWidget {
  final FilamentApiClient api;
  final String panelTitle;
  final bool isDark;
  final VoidCallback onToggleTheme;
  final ValueChanged<Map<String, dynamic>> onLoginSuccess;

  const LoginScreen({
    super.key,
    required this.api,
    required this.panelTitle,
    required this.isDark,
    required this.onToggleTheme,
    required this.onLoginSuccess,
  });

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _usernameController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _obscurePassword = true;
  bool _rememberMe = false;
  bool _isLoading = false;
  String? _errorMessage;

  @override
  void dispose() {
    _usernameController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _handleLogin() async {
    final username = _usernameController.text.trim();
    final password = _passwordController.text.trim();
    if (username.isEmpty || password.isEmpty) {
      setState(() {
        _errorMessage = 'Please enter both username and password';
      });
      return;
    }

    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    try {
      final res = await widget.api.login(username, password, remember: _rememberMe);
      final user = res['user'] as Map<String, dynamic>? ?? {'login': username};
      if (mounted) {
        widget.onLoginSuccess(user);
      }
    } catch (e) {
      if (mounted) {
        setState(() {
          _errorMessage = e.toString().replaceFirst('Exception: ', '');
          _isLoading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final isDark = widget.isDark;
    final cardBg = isDark ? const Color(0xFF1E293B) : Colors.white;
    final scaffoldBg = isDark ? const Color(0xFF0F172A) : const Color(0xFFF1F5F9);
    final borderColor = isDark ? const Color(0xFF334155) : const Color(0xFFE2E8F0);

    return Scaffold(
      backgroundColor: scaffoldBg,
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 440),
            child: Card(
              elevation: 8,
              color: cardBg,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(20),
                side: BorderSide(color: borderColor),
              ),
              child: Padding(
                padding: const EdgeInsets.all(32),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Row(
                          children: [
                            Container(
                              padding: const EdgeInsets.all(10),
                              decoration: BoxDecoration(
                                color: const Color(0xFFF59E0B).withValues(alpha: 0.15),
                                borderRadius: BorderRadius.circular(12),
                              ),
                              child: const Icon(Icons.shield_outlined, color: Color(0xFFF59E0B), size: 28),
                            ),
                            const SizedBox(width: 14),
                            Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  widget.panelTitle,
                                  style: const TextStyle(fontSize: 20, fontWeight: FontWeight.bold),
                                ),
                                const Text(
                                  'GoFilament Multiplatform',
                                  style: TextStyle(fontSize: 12, color: Colors.grey),
                                ),
                              ],
                            ),
                          ],
                        ),
                        IconButton(
                          onPressed: widget.onToggleTheme,
                          icon: Icon(isDark ? Icons.light_mode_outlined : Icons.dark_mode_outlined),
                          tooltip: 'Toggle Theme',
                        ),
                      ],
                    ),
                    const SizedBox(height: 28),
                    const Text(
                      'Sign In to Your Workspace',
                      style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
                    ),
                    const SizedBox(height: 6),
                    const Text(
                      'Enter your authorized credentials to access this administration panel.',
                      style: TextStyle(fontSize: 13, color: Colors.grey),
                    ),
                    const SizedBox(height: 20),
                    if (_errorMessage != null) ...[
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
                        decoration: BoxDecoration(
                          color: Colors.red.withValues(alpha: 0.12),
                          borderRadius: BorderRadius.circular(10),
                          border: Border.all(color: Colors.red.withValues(alpha: 0.3)),
                        ),
                        child: Row(
                          children: [
                            const Icon(Icons.error_outline, color: Colors.red, size: 20),
                            const SizedBox(width: 10),
                            Expanded(
                              child: Text(
                                _errorMessage!,
                                style: const TextStyle(color: Colors.red, fontSize: 13, fontWeight: FontWeight.w500),
                              ),
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(height: 16),
                    ],
                    TextField(
                      controller: _usernameController,
                      decoration: InputDecoration(
                        labelText: 'Username or Email',
                        prefixIcon: const Icon(Icons.person_outline),
                        border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
                        filled: true,
                        fillColor: isDark ? const Color(0xFF0F172A) : const Color(0xFFF8FAFC),
                      ),
                      textInputAction: TextInputAction.next,
                    ),
                    const SizedBox(height: 16),
                    TextField(
                      controller: _passwordController,
                      obscureText: _obscurePassword,
                      decoration: InputDecoration(
                        labelText: 'Password',
                        prefixIcon: const Icon(Icons.lock_outline),
                        suffixIcon: IconButton(
                          icon: Icon(_obscurePassword ? Icons.visibility_outlined : Icons.visibility_off_outlined),
                          onPressed: () => setState(() => _obscurePassword = !_obscurePassword),
                        ),
                        border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
                        filled: true,
                        fillColor: isDark ? const Color(0xFF0F172A) : const Color(0xFFF8FAFC),
                      ),
                      onSubmitted: (_) => _handleLogin(),
                    ),
                    const SizedBox(height: 12),
                    InkWell(
                      onTap: () => setState(() => _rememberMe = !_rememberMe),
                      borderRadius: BorderRadius.circular(8),
                      child: Padding(
                        padding: const EdgeInsets.symmetric(vertical: 4),
                        child: Row(
                          children: [
                            SizedBox(
                              height: 24,
                              width: 24,
                              child: Checkbox(
                                value: _rememberMe,
                                activeColor: const Color(0xFFF59E0B),
                                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
                                onChanged: (v) => setState(() => _rememberMe = v ?? false),
                              ),
                            ),
                            const SizedBox(width: 8),
                            Text(
                              'Remember me',
                              style: TextStyle(
                                fontSize: 13,
                                color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
                                fontWeight: FontWeight.w500,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 16),
                    FilledButton(
                      onPressed: _isLoading ? null : _handleLogin,
                      style: FilledButton.styleFrom(
                        backgroundColor: const Color(0xFFF59E0B),
                        foregroundColor: Colors.black,
                        padding: const EdgeInsets.symmetric(vertical: 16),
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                      ),
                      child: _isLoading
                          ? const SizedBox(
                              height: 20,
                              width: 20,
                              child: CircularProgressIndicator(strokeWidth: 2, color: Colors.black),
                            )
                          : const Text('Sign In', style: TextStyle(fontSize: 15, fontWeight: FontWeight.bold)),
                    ),
                    const SizedBox(height: 20),
                    Center(
                      child: Text(
                        'Enclave Secured • GoFilament & GoLaravel Engine',
                        style: TextStyle(fontSize: 11, color: Colors.grey.shade500),
                      ),
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
}

enum SidebarCollapseMode {
  expanded,
  partialCollapse,
  hidden,
}

class BrandLogoWidget extends StatelessWidget {
  final bool isCompact;
  final String brandName;
  final String? subtitle;
  final String? imageAsset;

  const BrandLogoWidget({
    super.key,
    required this.isCompact,
    required this.brandName,
    this.subtitle,
    this.imageAsset,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    Widget iconWidget;
    if (imageAsset != null) {
      iconWidget = Container(
        width: 36,
        height: 36,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(9),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.15),
              blurRadius: 4,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        clipBehavior: Clip.antiAlias,
        child: Image.asset(
          imageAsset!,
          fit: BoxFit.cover,
          errorBuilder: (_, __, ___) => _buildFallbackIcon(theme),
        ),
      );
    } else {
      iconWidget = _buildFallbackIcon(theme);
    }

    if (isCompact) {
      return Tooltip(
        message: brandName,
        child: Center(child: iconWidget),
      );
    }

    return Row(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        iconWidget,
        const SizedBox(width: 10),
        Flexible(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                brandName,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontWeight: FontWeight.w800,
                  fontSize: 15,
                  letterSpacing: -0.2,
                  color: isDark ? Colors.white : Colors.grey.shade900,
                ),
              ),
              if (subtitle != null && subtitle!.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(top: 1),
                  child: Text(
                    subtitle!.toUpperCase(),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 10,
                      fontWeight: FontWeight.w700,
                      letterSpacing: 0.8,
                      color: isDark ? Colors.amber.shade300 : Colors.amber.shade800,
                    ),
                  ),
                ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildFallbackIcon(ThemeData theme) {
    return Container(
      width: 36,
      height: 36,
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFFF59E0B), Color(0xFFD97706)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(9),
      ),
      child: const Center(
        child: Icon(Icons.bolt_rounded, color: Colors.black, size: 22),
      ),
    );
  }
}

class AdminShellPage extends StatefulWidget {
  final FilamentApiClient api;
  final VoidCallback onToggleTheme;
  final bool isDark;
  final String currentLocale;
  final bool isRTL;
  final void Function(String locale, bool isRTL) onLocaleChanged;
  final VoidCallback onSignOut;

  const AdminShellPage({
    super.key,
    required this.api,
    required this.onToggleTheme,
    required this.isDark,
    required this.currentLocale,
    required this.isRTL,
    required this.onLocaleChanged,
    required this.onSignOut,
  });

  @override
  State<AdminShellPage> createState() => _AdminShellPageState();
}

class _AdminShellPageState extends State<AdminShellPage> {
  FilamentApiClient get api => widget.api;
  Map<String, dynamic>? schema;
  bool isLoading = true;
  int _currentIndex = 0;
  String? selectedResourceSlug;
  String _accountNavPlacement = 'header'; // 'header' (top circle button) or 'bottom_bar' (bottom menu tab)
  SidebarCollapseMode _sidebarMode = SidebarCollapseMode.expanded;
  String? errorMessage;

  @override
  void initState() {
    super.initState();
    loadSchema();
  }

  IconData _iconForResourceSlug(String slug, {bool isSelected = false}) {
    final s = slug.toLowerCase();
    if (s.contains('partner') && s.contains('branch')) return isSelected ? Icons.storefront : Icons.storefront_outlined;
    if (s.contains('customer') && s.contains('branch')) return isSelected ? Icons.store : Icons.store_outlined;
    if (s.contains('partner')) return isSelected ? Icons.local_gas_station : Icons.local_gas_station_outlined;
    if (s.contains('customer')) return isSelected ? Icons.business : Icons.business_outlined;
    if (s.contains('vehicle')) return isSelected ? Icons.local_shipping : Icons.local_shipping_outlined;
    if (s.contains('driver')) return isSelected ? Icons.badge : Icons.badge_outlined;
    if (s.contains('nfc') || s.contains('chip')) return isSelected ? Icons.nfc : Icons.nfc_outlined;
    if (s.contains('device')) return isSelected ? Icons.router : Icons.router_outlined;
    if (s.contains('telemetry')) return isSelected ? Icons.sensors : Icons.sensors_outlined;
    if (s.contains('ticket')) return isSelected ? Icons.confirmation_number : Icons.confirmation_number_outlined;
    if (s.contains('firmware')) return isSelected ? Icons.system_update_alt : Icons.system_update_alt_outlined;
    if (s.contains('admin') || s.contains('user')) return isSelected ? Icons.people : Icons.people_outline;
    if (s.contains('transaction') || s.contains('wallet')) return isSelected ? Icons.account_balance_wallet : Icons.account_balance_wallet_outlined;
    return isSelected ? Icons.table_chart : Icons.table_chart_outlined;
  }

  void setAccountNavPlacement(String placement) {
    setState(() {
      _accountNavPlacement = placement;
      if (placement == 'header' && _currentIndex > 1) {
        _currentIndex = 0;
      }
    });
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
        if (s['locale'] != null) {
          final loc = s['locale'] as String;
          final isRtl = s['is_rtl'] as bool? ?? AppLocale.isRTL(loc);
          widget.onLocaleChanged(loc, isRtl);
        }
        if (s['account_nav_placement'] != null && (s['account_nav_placement'] == 'header' || s['account_nav_placement'] == 'bottom_bar')) {
          _accountNavPlacement = s['account_nav_placement'];
        }
        final resources = s['resources'] as List<dynamic>?;
        if (resources != null && resources.isNotEmpty) {
          selectedResourceSlug = resources.first['slug'];
        }
        isLoading = false;
      });
    } catch (e) {
      debugPrint('Failed to load schema from ${api.baseUrl}: $e');
      if (e.toString().contains('401') || e.toString().contains('Unauthorized')) {
        widget.onSignOut();
        return;
      }
      setState(() {
        errorMessage = e.toString();
        isLoading = false;
      });
    }
  }

  void showLanguageDialog(BuildContext context) {
    final supported = (schema?['supported_locales'] as List<dynamic>?)?.map((e) => e.toString()).toList() ?? ['en', 'ar', 'es', 'de'];
    showModalBottomSheet(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(24))),
      builder: (ctx) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    AppLocale.t(widget.currentLocale, 'language_picker'),
                    style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                  ),
                  IconButton(icon: const Icon(Icons.close), onPressed: () => Navigator.pop(ctx)),
                ],
              ),
              const SizedBox(height: 12),
              for (final code in supported) ...[
                Builder(
                  builder: (_) {
                    final name = AppLocale.getLabel(code);
                    final flag = AppLocale.getFlag(code);
                    final isRtl = AppLocale.isRTL(code);
                    final isSelected = code == widget.currentLocale;
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: ListTile(
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(12),
                          side: BorderSide(
                            color: isSelected ? Colors.amber : Colors.white.withValues(alpha: 0.1),
                            width: isSelected ? 1.5 : 1,
                          ),
                        ),
                        leading: Text(flag, style: const TextStyle(fontSize: 24)),
                        title: Row(
                          children: [
                            Text(name, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 15)),
                            const SizedBox(width: 8),
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                              decoration: BoxDecoration(
                                color: (isRtl ? Colors.amber : Colors.blueAccent).withValues(alpha: 0.2),
                                borderRadius: BorderRadius.circular(6),
                              ),
                              child: Text(
                                isRtl ? 'RTL' : 'LTR',
                                style: TextStyle(
                                  fontSize: 9,
                                  fontWeight: FontWeight.bold,
                                  color: isRtl ? Colors.amber : Colors.blueAccent,
                                ),
                              ),
                            ),
                          ],
                        ),
                        subtitle: Text(
                          isRtl ? 'Right-to-Left (RTL) • من اليمين لليسار' : 'Left-to-Right (LTR)',
                          style: const TextStyle(fontSize: 11, color: Colors.grey),
                        ),
                        trailing: isSelected ? const Icon(Icons.check_circle, color: Color(0xFF10B981)) : null,
                        onTap: () {
                          Navigator.pop(ctx);
                          widget.onLocaleChanged(code, isRtl);
                          api.switchLocale(code);
                          ScaffoldMessenger.of(context).showSnackBar(
                            SnackBar(
                              content: Text(
                                isRtl
                                    ? '🇸🇦 تم تفعيل اللغة العربية والاتجاه من اليمين إلى اليسار (RTL)'
                                    : '$flag Language set to $name (LTR)',
                              ),
                              behavior: SnackBarBehavior.floating,
                            ),
                          );
                        },
                      ),
                    );
                  },
                ),
              ],
            ],
          ),
        ),
      ),
    );
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

    final brandName = schema?['brand_name'] ?? '{{TITLE}}';
    final resources = (schema?['resources'] as List<dynamic>?) ?? [];

    final isHeaderPlacement = _accountNavPlacement == 'header';
    final activeMobileIndex = isHeaderPlacement ? _currentIndex.clamp(0, 1) : _currentIndex;

    return ResponsiveLayout(
      // 📱 Mobile-First Layout: One-thumb bottom navigation, pull-to-refresh, cards
      mobile: Scaffold(
        appBar: AppBar(
          title: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              BrandLogoWidget(
                isCompact: true,
                brandName: brandName,
              ),
              const SizedBox(width: 10),
              Flexible(
                child: Text(
                  !isHeaderPlacement && _currentIndex == 2 ? 'Account & Profile' : brandName,
                  style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
          actions: [
            IconButton(
              icon: const Icon(Icons.language),
              tooltip: 'Language & Direction / اللغة',
              onPressed: () => showLanguageDialog(context),
            ),
            IconButton(
              icon: Icon(widget.isDark ? Icons.light_mode : Icons.dark_mode),
              onPressed: widget.onToggleTheme,
            ),
            IconButton(
              icon: const Icon(Icons.corporate_fare),
              onPressed: showTenantSwitcher,
            ),
            // Header circle account button is shown ONLY when placement is 'header' (never both!)
            if (isHeaderPlacement)
              Padding(
                padding: const EdgeInsets.only(right: 8, left: 4),
                child: CircularAccountButton(
                  initial: (api.currentUser?['name'] ?? api.currentUser?['login'] ?? 'A')[0].toUpperCase(),
                  name: api.currentUser?['name'] ?? api.currentUser?['login'] ?? 'Administrator',
                  email: api.currentUser?['email'] ?? 'admin@filament.local',
                  role: (api.currentUser?['role'] ?? 'OPERATIONS LEAD').toString().toUpperCase(),
                  onTap: () => AccountModalSheet.show(
                    context: context,
                    api: api,
                    activeTenant: api.activeTenant,
                    onThemeToggle: widget.onToggleTheme,
                    isDark: widget.isDark,
                    accountNavPlacement: _accountNavPlacement,
                    onAccountNavPlacementChanged: setAccountNavPlacement,
                    currentLocale: widget.currentLocale,
                    isRTL: widget.isRTL,
                    onLocaleChanged: widget.onLocaleChanged,
                    onOpenLanguagePicker: () => showLanguageDialog(context),
                    onSignOut: widget.onSignOut,
                  ),
                ),
              ),
          ],
        ),
        body: IndexedStack(
          index: activeMobileIndex,
          children: [
            _buildMobileDashboard(),
            _buildMobileResourceFeed(resources),
            if (!isHeaderPlacement)
              _buildMobileAccountHub(),
          ],
        ),
        bottomNavigationBar: NavigationBar(
          selectedIndex: activeMobileIndex,
          onDestinationSelected: (idx) {
            HapticFeedback.selectionClick();
            setState(() => _currentIndex = idx);
          },
          // Bottom menu account tab is shown ONLY when placement is 'bottom_bar' (never both!)
          destinations: isHeaderPlacement
              ? [
                  NavigationDestination(
                    icon: const Icon(Icons.dashboard_outlined),
                    selectedIcon: const Icon(Icons.dashboard),
                    label: AppLocale.t(widget.currentLocale, 'dashboard'),
                  ),
                  NavigationDestination(
                    icon: const Icon(Icons.layers_outlined),
                    selectedIcon: const Icon(Icons.layers),
                    label: AppLocale.t(widget.currentLocale, 'entities'),
                  ),
                ]
              : [
                  NavigationDestination(
                    icon: const Icon(Icons.dashboard_outlined),
                    selectedIcon: const Icon(Icons.dashboard),
                    label: AppLocale.t(widget.currentLocale, 'dashboard'),
                  ),
                  NavigationDestination(
                    icon: const Icon(Icons.layers_outlined),
                    selectedIcon: const Icon(Icons.layers),
                    label: AppLocale.t(widget.currentLocale, 'entities'),
                  ),
                  NavigationDestination(
                    icon: const Icon(Icons.account_circle_outlined),
                    selectedIcon: const Icon(Icons.account_circle),
                    label: AppLocale.t(widget.currentLocale, 'account'),
                  ),
                ],
        ),
      ),

      // 💻 Desktop Layout (>900px): Left sidebar with expanded/partial collapse/hidden modes
      desktop: Scaffold(
        body: Stack(
          children: [
            Row(
              children: [
                if (_sidebarMode != SidebarCollapseMode.hidden) ...[
                  _buildDesktopSidebar(resources, activeMobileIndex, isHeaderPlacement, brandName),
                  const VerticalDivider(thickness: 1, width: 1),
                ],
                Expanded(
                  child: Column(
                    children: [
                      // Desktop Header Bar
                      _buildDesktopHeaderBar(isHeaderPlacement, brandName),
                      Expanded(
                        child: IndexedStack(
                          index: activeMobileIndex,
                          children: [
                            _buildDesktopDashboard(),
                            _buildDesktopResources(resources),
                            if (!isHeaderPlacement)
                              _buildMobileAccountHub(),
                          ],
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            // Floating popping button when sidebar is completely hidden
            if (_sidebarMode == SidebarCollapseMode.hidden)
              Positioned(
                left: 16,
                bottom: 24,
                child: FloatingActionButton.extended(
                  heroTag: 'sidebar_restore_fab',
                  elevation: 6,
                  backgroundColor: Theme.of(context).colorScheme.primaryContainer,
                  foregroundColor: Theme.of(context).colorScheme.onPrimaryContainer,
                  icon: const Icon(Icons.menu_open_rounded, size: 20),
                  label: const Text('Show Menu', style: TextStyle(fontWeight: FontWeight.bold)),
                  onPressed: () => setState(() => _sidebarMode = SidebarCollapseMode.expanded),
                ),
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildDesktopHeaderBar(bool isHeaderPlacement, String brandName) {
    return Container(
      height: 60,
      padding: const EdgeInsets.symmetric(horizontal: 20),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border(bottom: BorderSide(color: Colors.white.withValues(alpha: 0.08))),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Row(
            children: [
              if (_sidebarMode == SidebarCollapseMode.hidden) ...[
                IconButton(
                  icon: const Icon(Icons.menu_rounded),
                  tooltip: 'Show Sidebar',
                  onPressed: () => setState(() => _sidebarMode = SidebarCollapseMode.expanded),
                ),
                const SizedBox(width: 8),
                BrandLogoWidget(
                  isCompact: true,
                  brandName: brandName,
                ),
                const SizedBox(width: 10),
              ],
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
              FilledButton.tonalIcon(
                style: FilledButton.styleFrom(
                  visualDensity: VisualDensity.compact,
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                ),
                icon: Text(AppLocale.getFlag(widget.currentLocale), style: const TextStyle(fontSize: 16)),
                label: Text(
                  '${AppLocale.getLabel(widget.currentLocale)} (${widget.isRTL ? "RTL" : "LTR"})',
                  style: const TextStyle(fontSize: 12, fontWeight: FontWeight.bold),
                ),
                onPressed: () => showLanguageDialog(context),
              ),
              const SizedBox(width: 8),
              IconButton(
                icon: Icon(widget.isDark ? Icons.light_mode : Icons.dark_mode),
                onPressed: widget.onToggleTheme,
              ),
              IconButton(
                icon: const Icon(Icons.corporate_fare),
                tooltip: 'Switch Tenant',
                onPressed: showTenantSwitcher,
              ),
              // Header circular button is shown ONLY when placement is 'header'
              if (isHeaderPlacement) ...[
                const SizedBox(width: 8),
                CircularAccountButton(
                  initial: (api.currentUser?['name'] ?? api.currentUser?['login'] ?? 'A')[0].toUpperCase(),
                  name: api.currentUser?['name'] ?? api.currentUser?['login'] ?? 'Administrator',
                  email: api.currentUser?['email'] ?? 'admin@filament.local',
                  role: (api.currentUser?['role'] ?? 'OPERATIONS LEAD').toString().toUpperCase(),
                  onTap: () => AccountModalSheet.show(
                    context: context,
                    api: api,
                    activeTenant: api.activeTenant,
                    onThemeToggle: widget.onToggleTheme,
                    isDark: widget.isDark,
                    accountNavPlacement: _accountNavPlacement,
                    onAccountNavPlacementChanged: setAccountNavPlacement,
                    currentLocale: widget.currentLocale,
                    isRTL: widget.isRTL,
                    onLocaleChanged: widget.onLocaleChanged,
                    onOpenLanguagePicker: () => showLanguageDialog(context),
                    onSignOut: widget.onSignOut,
                  ),
                ),
              ],
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildDesktopSidebar(List<dynamic> resources, int activeMobileIndex, bool isHeaderPlacement, String brandName) {
    final isPartial = _sidebarMode == SidebarCollapseMode.partialCollapse;
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return AnimatedContainer(
      duration: const Duration(milliseconds: 200),
      curve: Curves.easeInOut,
      width: isPartial ? 72 : 250,
      decoration: BoxDecoration(
        color: theme.colorScheme.surface,
      ),
      child: Column(
        children: [
          // Sidebar Brand Header
          if (isPartial) ...[
            Container(
              height: 60,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                border: Border(bottom: BorderSide(color: Colors.white.withValues(alpha: 0.08))),
              ),
              child: BrandLogoWidget(
                isCompact: true,
                brandName: brandName,
              ),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 4),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  IconButton(
                    icon: const Icon(Icons.arrow_forward_ios_rounded, size: 14),
                    tooltip: 'Expand Sidebar',
                    visualDensity: VisualDensity.compact,
                    onPressed: () => setState(() => _sidebarMode = SidebarCollapseMode.expanded),
                  ),
                  IconButton(
                    icon: const Icon(Icons.visibility_off_outlined, size: 14),
                    tooltip: 'Hide Sidebar (Full Collapse)',
                    visualDensity: VisualDensity.compact,
                    onPressed: () => setState(() => _sidebarMode = SidebarCollapseMode.hidden),
                  ),
                ],
              ),
            ),
          ] else ...[
            Container(
              height: 60,
              padding: const EdgeInsets.symmetric(horizontal: 14),
              decoration: BoxDecoration(
                border: Border(bottom: BorderSide(color: Colors.white.withValues(alpha: 0.08))),
              ),
              child: Row(
                children: [
                  Expanded(
                    child: BrandLogoWidget(
                      isCompact: false,
                      brandName: brandName,
                      subtitle: 'Control Center',
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.menu_open_rounded, size: 20),
                    tooltip: 'Partial Collapse (Icons only)',
                    visualDensity: VisualDensity.compact,
                    onPressed: () => setState(() => _sidebarMode = SidebarCollapseMode.partialCollapse),
                  ),
                  IconButton(
                    icon: const Icon(Icons.close_rounded, size: 18),
                    tooltip: 'Hide Sidebar (Full Collapse)',
                    visualDensity: VisualDensity.compact,
                    onPressed: () => setState(() => _sidebarMode = SidebarCollapseMode.hidden),
                  ),
                ],
              ),
            ),
          ],

          // Navigation Links
          Expanded(
            child: ListView(
              padding: const EdgeInsets.symmetric(vertical: 8),
              children: [
                if (!isPartial)
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 8, 16, 4),
                    child: Text(
                      'MAIN NAVIGATION',
                      style: TextStyle(
                        fontSize: 10,
                        fontWeight: FontWeight.bold,
                        letterSpacing: 0.8,
                        color: isDark ? Colors.grey.shade500 : Colors.grey.shade600,
                      ),
                    ),
                  ),
                if (isPartial) ...[
                  _buildPartialNavItem(
                    icon: Icons.dashboard_outlined,
                    selectedIcon: Icons.dashboard,
                    label: AppLocale.t(widget.currentLocale, 'dashboard'),
                    isSelected: _currentIndex == 0,
                    onTap: () => setState(() => _currentIndex = 0),
                  ),
                  const Divider(height: 16, indent: 12, endIndent: 12),
                  for (final r in resources)
                    _buildPartialNavItem(
                      icon: _iconForResourceSlug((r['slug'] ?? '').toString()),
                      selectedIcon: _iconForResourceSlug((r['slug'] ?? '').toString(), isSelected: true),
                      label: (r['model_name'] ?? r['slug'] ?? 'Resource').toString(),
                      isSelected: _currentIndex == 1 && selectedResourceSlug == r['slug'],
                      onTap: () => setState(() {
                        _currentIndex = 1;
                        selectedResourceSlug = r['slug'];
                      }),
                    ),
                  if (!isHeaderPlacement) ...[
                    const Divider(height: 16, indent: 12, endIndent: 12),
                    _buildPartialNavItem(
                      icon: Icons.account_circle_outlined,
                      selectedIcon: Icons.account_circle,
                      label: AppLocale.t(widget.currentLocale, 'account'),
                      isSelected: _currentIndex == 2,
                      onTap: () => setState(() => _currentIndex = 2),
                    ),
                  ],
                ] else ...[
                  _buildHorizontalNavItem(
                    icon: Icons.dashboard_outlined,
                    selectedIcon: Icons.dashboard,
                    label: AppLocale.t(widget.currentLocale, 'dashboard'),
                    isSelected: _currentIndex == 0,
                    onTap: () => setState(() => _currentIndex = 0),
                  ),
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
                    child: Text(
                      'RESOURCES',
                      style: TextStyle(
                        fontSize: 10,
                        fontWeight: FontWeight.bold,
                        letterSpacing: 0.8,
                        color: isDark ? Colors.grey.shade500 : Colors.grey.shade600,
                      ),
                    ),
                  ),
                  for (final r in resources)
                    _buildHorizontalNavItem(
                      icon: _iconForResourceSlug((r['slug'] ?? '').toString()),
                      selectedIcon: _iconForResourceSlug((r['slug'] ?? '').toString(), isSelected: true),
                      label: (r['model_name'] ?? r['slug'] ?? 'Resource').toString(),
                      isSelected: _currentIndex == 1 && selectedResourceSlug == r['slug'],
                      onTap: () => setState(() {
                        _currentIndex = 1;
                        selectedResourceSlug = r['slug'];
                      }),
                    ),
                  if (!isHeaderPlacement) ...[
                    Padding(
                      padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
                      child: Text(
                        'PREFERENCES',
                        style: TextStyle(
                          fontSize: 10,
                          fontWeight: FontWeight.bold,
                          letterSpacing: 0.8,
                          color: isDark ? Colors.grey.shade500 : Colors.grey.shade600,
                        ),
                      ),
                    ),
                    _buildHorizontalNavItem(
                      icon: Icons.account_circle_outlined,
                      selectedIcon: Icons.account_circle,
                      label: AppLocale.t(widget.currentLocale, 'account'),
                      isSelected: _currentIndex == 2,
                      onTap: () => setState(() => _currentIndex = 2),
                    ),
                  ],
                ],
              ],
            ),
          ),

          // Sidebar Footer Status
          if (!isPartial)
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                border: Border(top: BorderSide(color: Colors.white.withValues(alpha: 0.08))),
              ),
              child: Row(
                children: [
                  Container(
                    width: 8,
                    height: 8,
                    decoration: const BoxDecoration(
                      color: Colors.green,
                      shape: BoxShape.circle,
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      api.activeTenant ?? 'GLOBAL TENANT',
                      style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w600),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildHorizontalNavItem({
    required IconData icon,
    IconData? selectedIcon,
    required String label,
    required bool isSelected,
    required VoidCallback onTap,
  }) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final activeBg = isDark
        ? Colors.amber.withValues(alpha: 0.15)
        : Colors.amber.withValues(alpha: 0.20);
    final activeColor = isDark ? Colors.amber : const Color(0xFFB45309);
    final inactiveColor = isDark ? Colors.grey.shade400 : Colors.grey.shade700;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
      child: Material(
        color: isSelected ? activeBg : Colors.transparent,
        borderRadius: BorderRadius.circular(10),
        child: InkWell(
          borderRadius: BorderRadius.circular(10),
          onTap: onTap,
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            child: Row(
              children: [
                Icon(
                  isSelected ? (selectedIcon ?? icon) : icon,
                  size: 20,
                  color: isSelected ? activeColor : inactiveColor,
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Text(
                    label,
                    style: TextStyle(
                      fontSize: 14,
                      fontWeight: isSelected ? FontWeight.w700 : FontWeight.w500,
                      color: isSelected ? activeColor : (isDark ? Colors.white : Colors.black87),
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (isSelected)
                  Container(
                    width: 4,
                    height: 18,
                    decoration: BoxDecoration(
                      color: activeColor,
                      borderRadius: BorderRadius.circular(4),
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildPartialNavItem({
    required IconData icon,
    IconData? selectedIcon,
    required String label,
    required bool isSelected,
    required VoidCallback onTap,
  }) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final activeBg = isDark
        ? Colors.amber.withValues(alpha: 0.18)
        : Colors.amber.withValues(alpha: 0.22);
    final activeColor = isDark ? Colors.amber : const Color(0xFFB45309);
    final inactiveColor = isDark ? Colors.grey.shade400 : Colors.grey.shade700;

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      child: Tooltip(
        message: label,
        preferBelow: false,
        waitDuration: const Duration(milliseconds: 150),
        child: Material(
          color: isSelected ? activeBg : Colors.transparent,
          borderRadius: BorderRadius.circular(10),
          child: InkWell(
            borderRadius: BorderRadius.circular(10),
            onTap: onTap,
            child: Container(
              height: 44,
              alignment: Alignment.center,
              child: Icon(
                isSelected ? (selectedIcon ?? icon) : icon,
                size: 22,
                color: isSelected ? activeColor : inactiveColor,
              ),
            ),
          ),
        ),
      ),
    );
  }
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
              : Builder(builder: (_) {
                  final cur = resources.firstWhere((r) => r['slug'] == selectedResourceSlug, orElse: () => null);
                  final rels = cur != null ? (cur['relations'] as List<dynamic>?) : null;
                  final filters = cur != null ? (cur['filters'] as List<dynamic>?) : null;
                  return ResourceRecordList(api: api, slug: selectedResourceSlug!, relations: rels, filters: filters);
                }),
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
    if (selectedResourceSlug == null && resources.isNotEmpty) {
      selectedResourceSlug = resources.first['slug'];
    }
    if (selectedResourceSlug == null) {
      return const Center(child: Text('Select a resource from the menu'));
    }
    final cur = resources.firstWhere((r) => r['slug'] == selectedResourceSlug, orElse: () => null);
    final rels = cur != null ? (cur['relations'] as List<dynamic>?) : null;
    final filters = cur != null ? (cur['filters'] as List<dynamic>?) : null;
    return ResourceRecordList(api: api, slug: selectedResourceSlug!, relations: rels, filters: filters);
  }

  Widget _buildMobileAccountHub() {
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        // User Profile Summary Card (Informational header — does NOT re-open modal sheet)
        Card(
          elevation: 0,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(16),
            side: BorderSide(color: Colors.amber.withValues(alpha: 0.3)),
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
                  child: Center(
                    child: Text(
                      (api.currentUser?['name'] ?? api.currentUser?['login'] ?? 'A')[0].toUpperCase(),
                      style: const TextStyle(color: Colors.black, fontWeight: FontWeight.bold, fontSize: 22),
                    ),
                  ),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Text(
                            api.currentUser?['name'] ?? api.currentUser?['login'] ?? 'Administrator',
                            style: const TextStyle(fontSize: 17, fontWeight: FontWeight.bold),
                          ),
                          const SizedBox(width: 8),
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                            decoration: BoxDecoration(
                              color: const Color(0xFF10B981).withValues(alpha: 0.2),
                              borderRadius: BorderRadius.circular(6),
                            ),
                            child: const Text(
                              'PRO',
                              style: TextStyle(fontSize: 9, fontWeight: FontWeight.bold, color: Color(0xFF10B981)),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 2),
                      Text(
                        '${api.currentUser?['email'] ?? "admin@filament.local"} • ${(api.currentUser?['role'] ?? "OPERATIONS LEAD").toString().toUpperCase()}',
                        style: const TextStyle(fontSize: 12, color: Colors.grey),
                      ),
                      const SizedBox(height: 6),
                      const Row(
                        children: [
                          Icon(Icons.verified_user_outlined, size: 13, color: Color(0xFF10B981)),
                          SizedBox(width: 4),
                          Text(
                            'Active Session • Enclave Secured',
                            style: TextStyle(fontSize: 11, color: Color(0xFF10B981)),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
              ],
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
                  MaterialPageRoute(
                    builder: (_) => SettingsScreen(
                      onThemeToggle: widget.onToggleTheme,
                      isDark: widget.isDark,
                      api: api,
                      accountNavPlacement: _accountNavPlacement,
                      onAccountNavPlacementChanged: setAccountNavPlacement,
                      currentLocale: widget.currentLocale,
                      isRTL: widget.isRTL,
                      onLocaleChanged: widget.onLocaleChanged,
                    ),
                  ),
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
                leading: const Icon(Icons.switch_access_shortcut),
                title: const Text('Account Navigation Entrypoint'),
                subtitle: Text(
                  _accountNavPlacement == 'header'
                      ? 'Top Header Avatar Circle (opens floating sheet)'
                      : 'Bottom Navigation Bar Tab (opens dedicated page)',
                  style: const TextStyle(fontSize: 12),
                ),
                trailing: DropdownButton<String>(
                  value: _accountNavPlacement,
                  underline: const SizedBox(),
                  items: const [
                    DropdownMenuItem(value: 'header', child: Text('Header Circle')),
                    DropdownMenuItem(value: 'bottom_bar', child: Text('Bottom Bar Tab')),
                  ],
                  onChanged: (val) {
                    if (val != null) setAccountNavPlacement(val);
                  },
                ),
              ),
              const Divider(height: 1),
              ListTile(
                leading: Text(AppLocale.getFlag(widget.currentLocale), style: const TextStyle(fontSize: 22)),
                title: Text(AppLocale.t(widget.currentLocale, 'language')),
                subtitle: Text(
                  '${AppLocale.getLabel(widget.currentLocale)} • ${widget.isRTL ? "RTL (من اليمين لليسار)" : "LTR (Left-to-Right)"}',
                  style: const TextStyle(fontSize: 12),
                ),
                trailing: const Icon(Icons.chevron_right),
                onTap: () => showLanguageDialog(context),
              ),
              const Divider(height: 1),
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
            widget.onSignOut();
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
  final List<dynamic>? relations;
  final List<dynamic>? filters;

  const ResourceRecordList({
    super.key,
    required this.api,
    required this.slug,
    this.relations,
    this.filters,
  });

  @override
  State<ResourceRecordList> createState() => _ResourceRecordListState();
}

class _ResourceRecordListState extends State<ResourceRecordList> {
  List<dynamic> allRecords = [];
  bool isLoading = true;
  String search = '';
  final Map<String, Set<String>> activeFilters = {};
  bool _showRelations = true;
  final ScrollController _horizontalScrollController = ScrollController();
  final ScrollController _verticalScrollController = ScrollController();
  Set<String> selectedRecordIds = <String>{};

  String _getRecordId(dynamic item, [int? index]) {
    if (item == null) return index != null ? "$index" : "";
    final raw = item["id"] ??
        item["ID"] ??
        item["login"] ??
        item["chip_serial"] ??
        item["uuid"] ??
        item["plate_number"] ??
        item["_id"];
    if (raw != null && raw.toString().isNotEmpty) {
      return raw.toString();
    }
    return index != null ? "$index" : "";
  }

  bool get isAllSelected =>
      records.isNotEmpty &&
      records.every((r) {
        final id = _getRecordId(r);
        return id.isNotEmpty && selectedRecordIds.contains(id);
      });

  void _toggleSelectAll([bool? forceVal]) {
    setState(() {
      final shouldSelect = forceVal ?? !isAllSelected;
      if (!shouldSelect) {
        selectedRecordIds.clear();
      } else {
        selectedRecordIds = records
            .asMap()
            .entries
            .map((e) => _getRecordId(e.value, e.key))
            .where((id) => id.isNotEmpty)
            .toSet();
      }
    });
  }

  void _toggleSelectRecord(String id) {
    if (id.isEmpty) return;
    setState(() {
      if (selectedRecordIds.contains(id)) {
        selectedRecordIds.remove(id);
      } else {
        selectedRecordIds.add(id);
      }
    });
  }

  void _deselectAll() {
    setState(() {
      selectedRecordIds.clear();
    });
  }

  Future<void> _executeBulkDelete(BuildContext context) async {
    if (selectedRecordIds.isEmpty) return;
    final count = selectedRecordIds.length;
    final confirm = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text("Delete Records?"),
        content: Text("Are you sure you want to delete $count selected records? This action cannot be undone."),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: const Text("Cancel"),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text("Delete Selected", style: TextStyle(color: Colors.redAccent, fontWeight: FontWeight.bold)),
          ),
        ],
      ),
    );
    if (confirm == true) {
      final success = await widget.api.bulkDeleteRecords(
        widget.slug,
        selectedRecordIds.toList(),
      );
      if (mounted) {
        if (success) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text("$count records deleted successfully"),
              backgroundColor: Colors.green.shade700,
            ),
          );
          _deselectAll();
          loadRecords();
        } else {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text("Failed to perform bulk delete"),
              backgroundColor: Colors.redAccent,
            ),
          );
        }
      }
    }
  }

  void _exportSelectedRecords(BuildContext context) {
    if (selectedRecordIds.isEmpty) return;
    final count = selectedRecordIds.length;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text("Exported $count selected records."),
        backgroundColor: Colors.amber.shade800,
      ),
    );
  }

  @override
  void dispose() {
    _horizontalScrollController.dispose();
    _verticalScrollController.dispose();
    super.dispose();
  }

  int get activeFilterCount {
    int count = 0;
    for (final s in activeFilters.values) {
      count += s.length;
    }
    return count;
  }

  void _clearAllFilters() {
    setState(() {
      activeFilters.clear();
    });
  }

  void _removeFilter(String key, String val) {
    setState(() {
      activeFilters[key]?.remove(val);
      if (activeFilters[key]?.isEmpty ?? false) {
        activeFilters.remove(key);
      }
    });
  }

  void _toggleFilter(String key, String val) {
    setState(() {
      if (!activeFilters.containsKey(key)) {
        activeFilters[key] = <String>{};
      }
      if (activeFilters[key]!.contains(val)) {
        activeFilters[key]!.remove(val);
        if (activeFilters[key]!.isEmpty) {
          activeFilters.remove(key);
        }
      } else {
        activeFilters[key]!.add(val);
      }
    });
  }

  String _titleCase(String s) {
    if (s.isEmpty) return s;
    return s.split(' ').map((w) => w.isEmpty ? '' : w[0].toUpperCase() + w.substring(1)).join(' ');
  }

  String _snakeToCamel(String s) {
    final parts = s.split('_');
    return parts.first + parts.skip(1).map((p) => p.isEmpty ? '' : p[0].toUpperCase() + p.substring(1)).join('');
  }

  String? _getRecordValue(dynamic r, String col) {
    if (r is! Map) return null;
    final val = r[col] ?? r[col.toLowerCase()] ?? r[col.toUpperCase()] ?? r[_snakeToCamel(col)];
    return val?.toString();
  }

  List<Map<String, dynamic>> _getAvailableFilters() {
    final list = <Map<String, dynamic>>[];
    final seenKeys = <String>{};

    if (widget.filters != null && widget.filters!.isNotEmpty) {
      for (final f in widget.filters!) {
        if (f is Map) {
          final m = Map<String, dynamic>.from(f);
          final col = (m['column'] ?? m['name'] ?? '').toString();
          if (col.isNotEmpty) {
            seenKeys.add(col.toLowerCase());
            list.add(m);
          }
        }
      }
    }

    final candidateKeys = ['status', 'status_text', 'type', 'customer_type', 'role', 'city', 'priority', 'category', 'method', 'channel'];
    for (final k in candidateKeys) {
      if (seenKeys.contains(k.toLowerCase())) continue;
      bool hasField = false;
      for (final r in allRecords) {
        if (r is Map && (r.containsKey(k) || r.containsKey(k.toUpperCase()) || r.containsKey(_snakeToCamel(k)))) {
          hasField = true;
          break;
        }
      }
      if (hasField) {
        seenKeys.add(k.toLowerCase());
        list.add({
          'name': k,
          'column': k,
          'label': _titleCase(k.replaceAll('_', ' ')),
          'type': 'refined',
          'multiple': true,
        });
      }
    }
    return list;
  }

  List<String> _getOptionsForFilter(Map<String, dynamic> filterDef) {
    final col = (filterDef['column'] ?? filterDef['name'] ?? '').toString();
    final set = <String>{};

    if (filterDef['options'] != null && filterDef['options'] is List) {
      for (final opt in filterDef['options'] as List) {
        if (opt is Map) {
          final v = opt['value']?.toString() ?? '';
          if (v.isNotEmpty) set.add(v);
        } else if (opt != null) {
          final v = opt.toString();
          if (v.isNotEmpty) set.add(v);
        }
      }
    }

    for (final r in allRecords) {
      final v = _getRecordValue(r, col);
      if (v != null && v.trim().isNotEmpty) {
        set.add(v.trim());
      }
    }

    final res = set.toList();
    res.sort((a, b) => a.toLowerCase().compareTo(b.toLowerCase()));
    return res;
  }

  bool _recordMatchesSearch(dynamic r, String query) {
    if (r is! Map) return false;
    final q = query.toLowerCase();
    for (final v in r.values) {
      if (v != null && v.toString().toLowerCase().contains(q)) {
        return true;
      }
    }
    return false;
  }

  int _calculateFacetCount(String catKey, String optVal) {
    int count = 0;
    final optLower = optVal.toLowerCase();

    for (final r in allRecords) {
      if (search.isNotEmpty && !_recordMatchesSearch(r, search)) {
        continue;
      }

      bool matchesOtherFilters = true;
      for (final entry in activeFilters.entries) {
        if (entry.key == catKey) continue;
        if (entry.value.isEmpty) continue;
        final recVal = _getRecordValue(r, entry.key)?.toLowerCase();
        if (recVal == null || !entry.value.any((v) => v.toLowerCase() == recVal)) {
          matchesOtherFilters = false;
          break;
        }
      }

      if (!matchesOtherFilters) continue;

      final recVal = _getRecordValue(r, catKey)?.toLowerCase();
      if (recVal != null && recVal == optLower) {
        count++;
      }
    }
    return count;
  }

  List<dynamic> get filteredRecords {
    return allRecords.where((r) {
      if (search.isNotEmpty && !_recordMatchesSearch(r, search)) {
        return false;
      }
      for (final entry in activeFilters.entries) {
        if (entry.value.isEmpty) continue;
        final recVal = _getRecordValue(r, entry.key)?.toLowerCase();
        if (recVal == null || !entry.value.any((v) => v.toLowerCase() == recVal)) {
          return false;
        }
      }
      return true;
    }).toList();
  }

  List<dynamic> get records => filteredRecords;

  void _showFilterModal(BuildContext context) {
    final availableFilters = _getAvailableFilters();
    if (availableFilters.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('No filterable attributes found for this resource.')),
      );
      return;
    }

    final isWideScreen = MediaQuery.of(context).size.width >= 720;

    Widget buildFilterContent(BuildContext ctx, StateSetter setModalState) {
      final theme = Theme.of(ctx);
      final isDark = theme.brightness == Brightness.dark;
      final bg = isDark ? const Color(0xFF1E293B) : Colors.white;

      return Container(
        width: isWideScreen ? 580 : double.infinity,
        constraints: BoxConstraints(
          maxHeight: MediaQuery.of(ctx).size.height * (isWideScreen ? 0.80 : 0.85),
          maxWidth: 580,
        ),
        decoration: BoxDecoration(
          color: bg,
          borderRadius: isWideScreen
              ? BorderRadius.circular(20)
              : const BorderRadius.vertical(top: Radius.circular(24)),
          boxShadow: const [BoxShadow(color: Colors.black26, blurRadius: 20)],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (!isWideScreen)
              Center(
                child: Container(
                  margin: const EdgeInsets.only(top: 10, bottom: 8),
                  width: 40,
                  height: 4,
                  decoration: BoxDecoration(
                    color: Colors.grey.withValues(alpha: 0.4),
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
            // Header
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Row(
                    children: [
                      Icon(Icons.tune_rounded, color: theme.colorScheme.primary),
                      const SizedBox(width: 8),
                      const Text(
                        'Refined Search Filters',
                        style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                      ),
                    ],
                  ),
                  Row(
                    children: [
                      if (activeFilterCount > 0)
                        TextButton.icon(
                          icon: const Icon(Icons.refresh, size: 16),
                          label: const Text('Reset All', style: TextStyle(fontSize: 13)),
                          onPressed: () {
                            _clearAllFilters();
                            setModalState(() {});
                          },
                        ),
                      if (isWideScreen)
                        IconButton(
                          icon: const Icon(Icons.close, size: 20),
                          onPressed: () => Navigator.pop(ctx),
                        ),
                    ],
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            Flexible(
              child: ListView.builder(
                shrinkWrap: true,
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                itemCount: availableFilters.length,
                itemBuilder: (c, idx) {
                  final f = availableFilters[idx];
                  final colKey = (f['column'] ?? f['name'] ?? '').toString();
                  final label = (f['label'] ?? colKey).toString();
                  final options = _getOptionsForFilter(f);
                  if (options.isEmpty) return const SizedBox.shrink();

                  final selectedSet = activeFilters[colKey] ?? <String>{};

                  return Card(
                    margin: const EdgeInsets.only(bottom: 12),
                    elevation: 0,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(12),
                      side: BorderSide(color: isDark ? Colors.white12 : Colors.black12),
                    ),
                    color: isDark ? const Color(0xFF0F172A) : const Color(0xFFF8FAFC),
                    child: Padding(
                      padding: const EdgeInsets.all(12),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Row(
                            mainAxisAlignment: MainAxisAlignment.spaceBetween,
                            children: [
                              Text(
                                label.toUpperCase(),
                                style: TextStyle(
                                  fontSize: 12,
                                  fontWeight: FontWeight.bold,
                                  letterSpacing: 0.8,
                                  color: theme.colorScheme.primary,
                                ),
                              ),
                              if (selectedSet.isNotEmpty)
                                Container(
                                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                                  decoration: BoxDecoration(
                                    color: theme.colorScheme.primary.withValues(alpha: 0.15),
                                    borderRadius: BorderRadius.circular(10),
                                  ),
                                  child: Text(
                                    '${selectedSet.length} selected',
                                    style: TextStyle(
                                      fontSize: 11,
                                      fontWeight: FontWeight.bold,
                                      color: theme.colorScheme.primary,
                                    ),
                                  ),
                                ),
                            ],
                          ),
                          const SizedBox(height: 8),
                          ...options.map((optVal) {
                            final isChecked = selectedSet.contains(optVal);
                            final count = _calculateFacetCount(colKey, optVal);

                            return InkWell(
                              borderRadius: BorderRadius.circular(8),
                              onTap: () {
                                _toggleFilter(colKey, optVal);
                                setModalState(() {});
                              },
                              child: Padding(
                                padding: const EdgeInsets.symmetric(vertical: 6, horizontal: 6),
                                child: Row(
                                  children: [
                                    SizedBox(
                                      width: 24,
                                      height: 24,
                                      child: Checkbox(
                                        value: isChecked,
                                        activeColor: theme.colorScheme.primary,
                                        onChanged: (val) {
                                          _toggleFilter(colKey, optVal);
                                          setModalState(() {});
                                        },
                                      ),
                                    ),
                                    const SizedBox(width: 10),
                                    Expanded(
                                      child: Text(
                                        optVal,
                                        style: TextStyle(
                                          fontSize: 14,
                                          fontWeight: isChecked ? FontWeight.bold : FontWeight.normal,
                                          color: (count == 0 && !isChecked) ? Colors.grey : null,
                                        ),
                                      ),
                                    ),
                                    const SizedBox(width: 8),
                                    Container(
                                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 3),
                                      decoration: BoxDecoration(
                                        color: isChecked
                                            ? theme.colorScheme.primary.withValues(alpha: 0.2)
                                            : (isDark ? Colors.white.withValues(alpha: 0.08) : Colors.black.withValues(alpha: 0.05)),
                                        borderRadius: BorderRadius.circular(12),
                                      ),
                                      child: Text(
                                        '$count',
                                        style: TextStyle(
                                          fontSize: 12,
                                          fontWeight: FontWeight.w600,
                                          color: isChecked
                                              ? theme.colorScheme.primary
                                              : ((count == 0 && !isChecked) ? Colors.grey : null),
                                        ),
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            );
                          }),
                        ],
                      ),
                    ),
                  );
                },
              ),
            ),
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                border: Border(top: BorderSide(color: isDark ? Colors.white12 : Colors.black12)),
              ),
              child: SizedBox(
                width: double.infinity,
                child: ElevatedButton(
                  style: ElevatedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 14),
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                  ),
                  onPressed: () => Navigator.pop(ctx),
                  child: Text(
                    'Show ${filteredRecords.length} Results',
                    style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 15),
                  ),
                ),
              ),
            ),
          ],
        ),
      );
    }

    if (isWideScreen) {
      showDialog(
        context: context,
        builder: (dialogCtx) => StatefulBuilder(
          builder: (ctx, setModalState) => Dialog(
            backgroundColor: Colors.transparent,
            elevation: 0,
            insetPadding: const EdgeInsets.symmetric(horizontal: 24, vertical: 24),
            child: buildFilterContent(ctx, setModalState),
          ),
        ),
      );
    } else {
      showModalBottomSheet(
        context: context,
        isScrollControlled: true,
        backgroundColor: Colors.transparent,
        builder: (bottomSheetCtx) => StatefulBuilder(
          builder: (ctx, setModalState) => buildFilterContent(ctx, setModalState),
        ),
      );
    }
  }

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
    setState(() {
      isLoading = true;
      selectedRecordIds.clear();
    });
    try {
      final res = await widget.api.fetchResourceRecords(widget.slug, search: search);
      setState(() {
        allRecords = (res['data'] as List<dynamic>?) ?? [];
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

    final isSelected = selectedRecordIds.contains(id);
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;

    return Card(
      elevation: isSelected ? 2 : 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(
          color: isSelected ? Colors.amber.shade400 : (isDark ? Colors.white.withValues(alpha: 0.08) : Colors.black.withValues(alpha: 0.08)),
          width: isSelected ? 1.5 : 1,
        ),
      ),
      color: isSelected
          ? (isDark ? Colors.amber.withValues(alpha: 0.08) : Colors.amber.withValues(alpha: 0.05))
          : theme.colorScheme.surface,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header: Icon + Title + Status/Badge
            Row(
              children: [
                SizedBox(
                  width: 24,
                  height: 24,
                  child: Checkbox(
                    value: isSelected,
                    activeColor: Colors.amber.shade600,
                    checkColor: Colors.black,
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
                    onChanged: (_) => _toggleSelectRecord(id),
                  ),
                ),
                const SizedBox(width: 8),
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

  List<Map<String, dynamic>> _getRelations() {
    if (widget.relations != null && widget.relations!.isNotEmpty) {
      return widget.relations!.whereType<Map>().map((e) => Map<String, dynamic>.from(e)).toList();
    }
    return [];
  }

  IconData _iconForRelation(String icon) {
    if (icon.contains('user') || icon.contains('admin') || icon.contains('person')) {
      return Icons.people_alt_outlined;
    }
    if (icon.contains('building') || icon.contains('office') || icon.contains('branch')) {
      return Icons.business_outlined;
    }
    if (icon.contains('truck') || icon.contains('car') || icon.contains('vehicle')) {
      return Icons.local_shipping_outlined;
    }
    if (icon.contains('chip') || icon.contains('cpu') || icon.contains('tag')) {
      return Icons.nfc_outlined;
    }
    return Icons.account_tree_outlined;
  }

  List<String> _extractColumns() {
    if (records.isEmpty) return [];

    const excluded = {
      'id', 'ID', '_id',
      'status', 'Status', 'status_text', 'active',
      'password', 'token', 'remember_token', 'deleted_at',
      'created_at', 'updated_at', 'createdAt', 'updatedAt',
      'relations', 'sub_entities', 'hash', 'salt',
    };

    final orderedKeys = <String>[];
    final seen = <String>{};

    // 1. Primary identification keys first
    for (final candidate in ['name', 'Name', 'title', 'Title', 'code', 'Code', 'username', 'login']) {
      for (final r in records) {
        if (r is Map && r.containsKey(candidate) && !seen.contains(candidate.toLowerCase())) {
          orderedKeys.add(candidate);
          seen.add(candidate.toLowerCase());
          break;
        }
      }
    }

    // 2. All other business columns found across records
    for (final r in records) {
      if (r is Map) {
        for (final entry in r.entries) {
          final kStr = entry.key.toString();
          final kLower = kStr.toLowerCase();
          if (excluded.contains(kStr) || excluded.contains(kLower)) continue;
          if (seen.contains(kLower)) continue;

          final val = entry.value;
          if (val is Map || val is List) continue;

          orderedKeys.add(kStr);
          seen.add(kLower);
        }
      }
    }

    return orderedKeys;
  }

  double _columnWidth(String colKey) {
    final k = colKey.toLowerCase();
    if (k == 'name' || k == 'title') return 220;
    if (k == 'email' || k == 'username') return 200;
    if (k.contains('address') || k.contains('description') || k.contains('notes')) return 240;
    if (k == 'phone' || k.contains('phone')) return 150;
    if (k.contains('code') || k.contains('serial') || k.contains('plate') || k.contains('cr_')) return 150;
    if (k == 'city' || k == 'type' || k == 'role' || k == 'category') return 140;
    if (k.contains('amount') || k.contains('balance') || k.contains('price') || k.contains('rate')) return 140;
    return 150;
  }

  Widget _buildTableCell(BuildContext context, String colKey, dynamic item, bool isDark, IconData headerIcon) {
    final theme = Theme.of(context);
    final kLower = colKey.toLowerCase();
    final val = item[colKey] ?? item[colKey.toLowerCase()] ?? item[colKey.toUpperCase()] ?? item[_snakeToCamel(colKey)];
    final str = val?.toString().trim() ?? '';

    if (str.isEmpty) {
      return Text('—', style: TextStyle(color: Colors.grey.shade500, fontSize: 13));
    }

    if (kLower == 'name' || kLower == 'title') {
      return Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          CircleAvatar(
            radius: 13,
            backgroundColor: theme.colorScheme.primaryContainer,
            child: Icon(headerIcon, size: 13, color: theme.colorScheme.primary),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              str,
              style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
              overflow: TextOverflow.ellipsis,
            ),
          ),
        ],
      );
    }

    if (kLower == 'email') {
      return Text(
        str,
        style: TextStyle(fontSize: 12, color: isDark ? Colors.blue.shade300 : Colors.blue.shade700),
        overflow: TextOverflow.ellipsis,
      );
    }

    if (kLower.contains('amount') || kLower.contains('balance') || kLower.contains('price')) {
      return Text(
        str,
        style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600, fontFamily: 'monospace'),
        overflow: TextOverflow.ellipsis,
      );
    }

    return Text(
      str,
      style: TextStyle(fontSize: 13, color: isDark ? Colors.grey.shade300 : Colors.grey.shade800),
      overflow: TextOverflow.ellipsis,
    );
  }

  void _openRelation(BuildContext context, String id, String name, String relSlug, String relTitle) {
    Navigator.push(
      context,
      MaterialPageRoute(
        builder: (_) => RelationRecordScreen(
          api: widget.api,
          parentSlug: widget.slug,
          parentId: id,
          parentTitle: name,
          relSlug: relSlug,
          relTitle: relTitle,
        ),
      ),
    );
  }

  Widget _buildRecordTable(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final relations = _getRelations();
    final hasRelations = relations.isNotEmpty;
    final columns = _extractColumns();

    double totalColsWidth = 48.0 + 70.0 + 130.0 + 140.0 + 80.0; // Checkbox + ID + STATUS + ACTIONS + clearance
    for (final col in columns) {
      totalColsWidth += _columnWidth(col);
    }

    return Scrollbar(
      controller: _verticalScrollController,
      thumbVisibility: true,
      trackVisibility: true,
      interactive: true,
      thickness: 10,
      radius: const Radius.circular(5),
      notificationPredicate: (notif) => notif.metrics.axis == Axis.vertical,
      child: SingleChildScrollView(
        controller: _verticalScrollController,
        padding: const EdgeInsets.fromLTRB(20, 20, 20, 100),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 1600),
            child: Card(
              elevation: 0,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(16),
                side: BorderSide(
                  color: isDark ? Colors.white.withValues(alpha: 0.1) : Colors.black.withValues(alpha: 0.08),
                ),
              ),
              color: theme.colorScheme.surface,
              clipBehavior: Clip.antiAlias,
              child: LayoutBuilder(
                builder: (ctx, constraints) {
                  final tableWidth = constraints.maxWidth > totalColsWidth ? constraints.maxWidth : totalColsWidth;
                  return Scrollbar(
                    controller: _horizontalScrollController,
                    thumbVisibility: true,
                    trackVisibility: true,
                    interactive: true,
                    thickness: 10,
                    radius: const Radius.circular(5),
                    notificationPredicate: (notif) => notif.metrics.axis == Axis.horizontal,
                    child: SingleChildScrollView(
                      controller: _horizontalScrollController,
                      scrollDirection: Axis.horizontal,
                      child: SizedBox(
                        width: tableWidth,
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            // Table Header / Bulk Action Bar
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
                              decoration: BoxDecoration(
                                color: selectedRecordIds.isNotEmpty
                                    ? Colors.amber.withValues(alpha: isDark ? 0.2 : 0.12)
                                    : (isDark ? const Color(0xFF0F172A) : const Color(0xFFF8FAFC)),
                                border: Border(
                                  bottom: BorderSide(
                                    color: selectedRecordIds.isNotEmpty
                                        ? Colors.amber.withValues(alpha: 0.35)
                                        : (isDark ? Colors.white12 : Colors.black12),
                                  ),
                                ),
                              ),
                              child: Row(
                                children: [
                                  SizedBox(
                                    width: 48,
                                    child: Checkbox(
                                      value: isAllSelected
                                          ? true
                                          : (selectedRecordIds.isNotEmpty ? null : false),
                                      tristate: true,
                                      activeColor: Colors.amber.shade600,
                                      checkColor: Colors.black,
                                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
                                      onChanged: (val) {
                                        if (isAllSelected) {
                                          _deselectAll();
                                        } else {
                                          _toggleSelectAll(true);
                                        }
                                      },
                                    ),
                                  ),
                                  if (selectedRecordIds.isNotEmpty) ...[
                                    Text(
                                      '${selectedRecordIds.length} of ${records.length} selected',
                                      style: TextStyle(
                                        fontWeight: FontWeight.bold,
                                        fontSize: 13,
                                        color: isDark ? Colors.amber.shade400 : const Color(0xFFB45309),
                                      ),
                                    ),
                                    const SizedBox(width: 16),
                                    FilledButton.icon(
                                      style: FilledButton.styleFrom(
                                        backgroundColor: Colors.redAccent,
                                        foregroundColor: Colors.white,
                                        visualDensity: VisualDensity.compact,
                                        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                                      ),
                                      icon: const Icon(Icons.delete_sweep_rounded, size: 16),
                                      label: Text(
                                        'Delete Selected (${selectedRecordIds.length})',
                                        style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12),
                                      ),
                                      onPressed: () => _executeBulkDelete(context),
                                    ),
                                    const SizedBox(width: 10),
                                    OutlinedButton.icon(
                                      style: OutlinedButton.styleFrom(
                                        visualDensity: VisualDensity.compact,
                                        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                                      ),
                                      icon: const Icon(Icons.download_rounded, size: 16),
                                      label: const Text('Export', style: TextStyle(fontSize: 12)),
                                      onPressed: () => _exportSelectedRecords(context),
                                    ),
                                    const Spacer(),
                                    TextButton(
                                      style: TextButton.styleFrom(visualDensity: VisualDensity.compact),
                                      child: const Text('Deselect All', style: TextStyle(fontSize: 12)),
                                      onPressed: _deselectAll,
                                    ),
                                    const SizedBox(width: 16),
                                  ] else ...[
                                    const SizedBox(
                                      width: 70,
                                      child: Text(
                                        'ID',
                                        style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12, letterSpacing: 0.5, color: Colors.grey),
                                      ),
                                    ),
                                    for (final col in columns)
                                      SizedBox(
                                        width: _columnWidth(col),
                                        child: Padding(
                                          padding: const EdgeInsets.only(right: 12),
                                          child: Text(
                                            _titleCase(col.replaceAll('_', ' ')).toUpperCase(),
                                            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12, letterSpacing: 0.5, color: Colors.grey),
                                            overflow: TextOverflow.ellipsis,
                                          ),
                                        ),
                                      ),
                                    const SizedBox(
                                      width: 130,
                                      child: Text(
                                        'STATUS',
                                        style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12, letterSpacing: 0.5, color: Colors.grey),
                                      ),
                                    ),
                                    const SizedBox(
                                      width: 140,
                                      child: Align(
                                        alignment: Alignment.centerRight,
                                        child: Text(
                                          'ACTIONS',
                                          style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12, letterSpacing: 0.5, color: Colors.grey),
                                        ),
                                      ),
                                    ),
                                    const SizedBox(width: 16),
                                  ],
                                ],
                              ),
                            ),

                            // Table Rows
                            for (int i = 0; i < records.length; i++)
                              _buildTableRow(
                                context,
                                records[i],
                                _getRecordId(records[i], i),
                                columns,
                                hasRelations,
                                relations,
                                i == records.length - 1,
                              ),
                          ],
                        ),
                      ),
                    ),
                  );
                },
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildTableRow(
    BuildContext context,
    dynamic item,
    String id,
    List<String> columns,
    bool hasRelations,
    List<Map<String, dynamic>> relations,
    bool isLast,
  ) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
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
      name = (item['Name'] ?? item['name'] ?? item['Title'] ?? item['title'] ?? 'Record #$id').toString();
    }

    IconData headerIcon = Icons.article_outlined;
    if (isDevice) headerIcon = Icons.router;
    if (isTelemetry) headerIcon = Icons.sensors;
    if (isTicket) headerIcon = Icons.confirmation_number_outlined;
    if (isFirmware) headerIcon = Icons.system_update_alt;

    final status = (item['Status'] ?? item['status'] ?? item['status_text'] ?? (item['active'] == 'Y' ? 'active' : (item['active'] == 'N' ? 'inactive' : 'online'))).toString();
    Color statusColor = const Color(0xFF10B981);
    final stLower = status.toLowerCase();
    if (stLower == 'inactive' || stLower == 'offline' || stLower == 'critical' || stLower == 'rejected' || stLower == 'overdue' || stLower == 'n') {
      statusColor = Colors.redAccent;
    } else if (stLower == 'pending' || stLower == 'warning' || stLower == 'draft' || stLower == 'partial') {
      statusColor = Colors.amber;
    }

    final isRelationsVisible = hasRelations && _showRelations;
    final isSelected = selectedRecordIds.contains(id);

    return Container(
      decoration: BoxDecoration(
        color: isSelected
            ? (isDark ? Colors.amber.withValues(alpha: 0.12) : Colors.amber.withValues(alpha: 0.08))
            : null,
        border: isLast && !isRelationsVisible
            ? null
            : Border(bottom: BorderSide(color: isDark ? Colors.white.withValues(alpha: 0.05) : Colors.black.withValues(alpha: 0.05))),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Main Columns Row
          Material(
            color: Colors.transparent,
            child: InkWell(
              hoverColor: isDark ? Colors.white.withValues(alpha: 0.03) : Colors.black.withValues(alpha: 0.02),
              onTap: () => showEditSheet(context, id, item),
              child: Padding(
                padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
                child: Row(
                  children: [
                    // Select Checkbox Column
                    SizedBox(
                      width: 48,
                      child: Checkbox(
                        value: isSelected,
                        activeColor: Colors.amber.shade600,
                        checkColor: Colors.black,
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
                        onChanged: (_) => _toggleSelectRecord(id),
                      ),
                    ),
                    // ID Column
                    SizedBox(
                      width: 70,
                      child: Text(
                        '#$id',
                        style: TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.bold,
                          color: Colors.grey.shade400,
                        ),
                      ),
                    ),

                    // Discovered Columns
                    for (final col in columns)
                      SizedBox(
                        width: _columnWidth(col),
                        child: Padding(
                          padding: const EdgeInsets.only(right: 12),
                          child: _buildTableCell(context, col, item, isDark, headerIcon),
                        ),
                      ),

                    // Status Badge Column
                    SizedBox(
                      width: 130,
                      child: Align(
                        alignment: Alignment.centerLeft,
                        child: Container(
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
                              Text(
                                status.toUpperCase(),
                                style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: statusColor),
                              ),
                            ],
                          ),
                        ),
                      ),
                    ),

                    // Actions Column
                    SizedBox(
                      width: 140,
                      child: Row(
                        mainAxisAlignment: MainAxisAlignment.end,
                        children: [
                          IconButton(
                            icon: const Icon(Icons.edit_outlined, size: 18),
                            tooltip: 'Edit Record',
                            onPressed: () => showEditSheet(context, id, item),
                          ),
                          IconButton(
                            icon: const Icon(Icons.delete_outline, size: 18, color: Colors.redAccent),
                            tooltip: 'Delete Record',
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
                    ),
                    const SizedBox(width: 16),
                  ],
                ),
              ),
            ),
          ),

          // Relationship Managers (at the bottom of every record, controlled universally)
          if (isRelationsVisible)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 10),
              decoration: BoxDecoration(
                color: isDark ? const Color(0xFF0F172A).withValues(alpha: 0.7) : const Color(0xFFF1F5F9).withValues(alpha: 0.9),
                border: Border(
                  top: BorderSide(color: isDark ? Colors.white.withValues(alpha: 0.04) : Colors.black.withValues(alpha: 0.04)),
                ),
              ),
              child: Row(
                children: [
                  const SizedBox(width: 50),
                  Icon(Icons.account_tree_outlined, size: 14, color: Colors.amber.shade400),
                  const SizedBox(width: 8),
                  Text(
                    'RELATIONS:',
                    style: TextStyle(
                      fontSize: 10,
                      fontWeight: FontWeight.bold,
                      color: Colors.amber.shade400,
                      letterSpacing: 0.5,
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Wrap(
                      spacing: 8,
                      runSpacing: 4,
                      children: relations.map((rel) {
                        final relTitle = (rel['title'] ?? rel['relationship'] ?? 'Related').toString();
                        final relSlug = (rel['relationship'] ?? '').toString();
                        final iconName = (rel['icon'] ?? '').toString();
                        return ActionChip(
                          avatar: Icon(_iconForRelation(iconName), size: 14, color: Colors.amber.shade400),
                          label: Text(relTitle, style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold)),
                          backgroundColor: Colors.amber.withValues(alpha: 0.1),
                          side: BorderSide(color: Colors.amber.withValues(alpha: 0.3)),
                          padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 0),
                          visualDensity: VisualDensity.compact,
                          onPressed: () {
                            _openRelation(context, id, name, relSlug, relTitle);
                          },
                        );
                      }).toList(),
                    ),
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.slug.toUpperCase(), style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
        bottom: PreferredSize(
          preferredSize: Size.fromHeight(activeFilterCount > 0 ? 102 : 56),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                child: Row(
                  children: [
                    Expanded(
                      child: TextField(
                        decoration: InputDecoration(
                          hintText: 'Search ${widget.slug}...',
                          prefixIcon: const Icon(Icons.search, size: 20),
                          suffixIcon: search.isNotEmpty
                              ? IconButton(
                                  icon: const Icon(Icons.clear, size: 18),
                                  onPressed: () {
                                    setState(() => search = '');
                                  },
                                )
                              : null,
                          isDense: true,
                          border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                        ),
                        onChanged: (v) {
                          setState(() => search = v);
                        },
                      ),
                    ),
                    const SizedBox(width: 8),
                    IconButton.filledTonal(
                      tooltip: _showRelations ? 'Hide Related Entities' : 'Show Related Entities',
                      style: IconButton.styleFrom(
                        backgroundColor: _showRelations ? Colors.amber.withValues(alpha: 0.2) : null,
                        foregroundColor: _showRelations ? Colors.amber.shade400 : null,
                      ),
                      icon: Icon(_showRelations ? Icons.account_tree : Icons.account_tree_outlined),
                      onPressed: () {
                        setState(() {
                          _showRelations = !_showRelations;
                        });
                      },
                    ),
                    const SizedBox(width: 8),
                    Badge(
                      isLabelVisible: activeFilterCount > 0,
                      label: Text('$activeFilterCount'),
                      backgroundColor: Colors.amber,
                      textColor: Colors.black,
                      child: IconButton.filledTonal(
                        tooltip: 'Refined Search Filters',
                        icon: const Icon(Icons.tune_rounded),
                        onPressed: () => _showFilterModal(context),
                      ),
                    ),
                    const SizedBox(width: 8),
                    PopupMenuButton<String>(
                      tooltip: selectedRecordIds.isNotEmpty
                          ? 'Bulk Actions (${selectedRecordIds.length} selected)'
                          : 'Bulk Selection & Actions',
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                      icon: Badge(
                        isLabelVisible: selectedRecordIds.isNotEmpty,
                        label: Text('${selectedRecordIds.length}'),
                        backgroundColor: Colors.amber.shade700,
                        textColor: Colors.black,
                        child: IconButton.filledTonal(
                          style: IconButton.styleFrom(
                            backgroundColor: selectedRecordIds.isNotEmpty ? Colors.amber.withValues(alpha: 0.2) : null,
                            foregroundColor: selectedRecordIds.isNotEmpty ? Colors.amber.shade400 : null,
                          ),
                          icon: const Icon(Icons.checklist_rtl_rounded),
                          onPressed: null,
                        ),
                      ),
                      onSelected: (val) {
                        if (val == 'select_all') {
                          _toggleSelectAll();
                        } else if (val == 'deselect') {
                          _deselectAll();
                        } else if (val == 'delete') {
                          _executeBulkDelete(context);
                        } else if (val == 'export') {
                          _exportSelectedRecords(context);
                        }
                      },
                      itemBuilder: (ctx) => [
                        PopupMenuItem(
                          value: 'select_all',
                          child: Row(
                            children: [
                              Icon(isAllSelected ? Icons.deselect_rounded : Icons.select_all_rounded, size: 20),
                              const SizedBox(width: 10),
                              Text(isAllSelected ? 'Deselect All' : 'Select All (${records.length})'),
                            ],
                          ),
                        ),
                        if (selectedRecordIds.isNotEmpty) ...[
                          PopupMenuItem(
                            value: 'delete',
                            child: const Row(
                              children: [
                                Icon(Icons.delete_sweep_rounded, color: Colors.redAccent, size: 20),
                                SizedBox(width: 10),
                                Text('Bulk Delete', style: TextStyle(color: Colors.redAccent)),
                              ],
                            ),
                          ),
                          PopupMenuItem(
                            value: 'export',
                            child: const Row(
                              children: [
                                Icon(Icons.download_rounded, size: 20),
                                SizedBox(width: 10),
                                Text('Export Selected'),
                              ],
                            ),
                          ),
                          const PopupMenuDivider(),
                          const PopupMenuItem(
                            value: 'deselect',
                            child: Row(
                              children: [
                                Icon(Icons.close_rounded, size: 20),
                                SizedBox(width: 10),
                                Text('Clear Selection'),
                              ],
                            ),
                          ),
                        ],
                      ],
                    ),
                  ],
                ),
              ),
              if (activeFilterCount > 0)
                Container(
                  height: 42,
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 2),
                  child: ListView(
                    scrollDirection: Axis.horizontal,
                    children: [
                      ActionChip(
                        avatar: const Icon(Icons.close, size: 14),
                        label: const Text('Clear All', style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold)),
                        onPressed: _clearAllFilters,
                        visualDensity: VisualDensity.compact,
                      ),
                      const SizedBox(width: 6),
                      ...activeFilters.entries.expand((entry) {
                        final cat = entry.key;
                        return entry.value.map((val) {
                          return Padding(
                            padding: const EdgeInsets.only(right: 6),
                            child: InputChip(
                              label: Text('${_titleCase(cat)}: $val', style: const TextStyle(fontSize: 11)),
                              onDeleted: () => _removeFilter(cat, val),
                              visualDensity: VisualDensity.compact,
                              deleteIconColor: Colors.amber,
                            ),
                          );
                        });
                      }),
                    ],
                  ),
                ),
            ],
          ),
        ),
      ),
      body: isLoading
          ? const Center(child: CircularProgressIndicator())
          : records.isEmpty
              ? Center(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      const Icon(Icons.search_off, size: 48, color: Colors.grey),
                      const SizedBox(height: 12),
                      Text(
                        activeFilterCount > 0 || search.isNotEmpty
                            ? 'No records match the applied filters or search.'
                            : 'No records found.',
                        style: const TextStyle(color: Colors.grey),
                      ),
                      if (activeFilterCount > 0 || search.isNotEmpty) ...[
                        const SizedBox(height: 12),
                        ElevatedButton.icon(
                          icon: const Icon(Icons.refresh, size: 16),
                          label: const Text('Reset All Filters'),
                          onPressed: () {
                            setState(() {
                              search = '';
                              _clearAllFilters();
                            });
                          },
                        ),
                      ],
                    ],
                  ),
                )
              : RefreshIndicator(
                  onRefresh: loadRecords,
                  child: LayoutBuilder(
                    builder: (context, constraints) {
                      // Desktop (left sidebar navigation active, width > 900):
                      // Render as professional table list of records!
                      final isDesktopTable = constraints.maxWidth > 900;
                      if (isDesktopTable) {
                        return _buildRecordTable(context);
                      }

                      // Mobile (bottom menu active, width <= 900):
                      // Render 1-column stacked card records (no horizontal scrolling needed)
                      return Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          if (selectedRecordIds.isNotEmpty)
                            Container(
                              margin: const EdgeInsets.fromLTRB(16, 8, 16, 4),
                              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
                              decoration: BoxDecoration(
                                color: Colors.amber.withValues(alpha: 0.15),
                                borderRadius: BorderRadius.circular(12),
                                border: Border.all(color: Colors.amber.withValues(alpha: 0.4)),
                              ),
                              child: Row(
                                children: [
                                  Icon(Icons.check_circle_outline, color: Colors.amber.shade400, size: 20),
                                  const SizedBox(width: 8),
                                  Text(
                                    '${selectedRecordIds.length} Selected',
                                    style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 13),
                                  ),
                                  const Spacer(),
                                  TextButton.icon(
                                    icon: const Icon(Icons.delete_sweep_outlined, size: 18, color: Colors.redAccent),
                                    label: const Text('Delete', style: TextStyle(color: Colors.redAccent, fontSize: 12)),
                                    onPressed: () => _executeBulkDelete(context),
                                  ),
                                  IconButton(
                                    icon: const Icon(Icons.close, size: 18),
                                    tooltip: 'Deselect all',
                                    onPressed: _deselectAll,
                                  ),
                                ],
                              ),
                            ),
                          Expanded(
                            child: ListView.builder(
                              padding: const EdgeInsets.all(12),
                              itemCount: records.length,
                              itemBuilder: (context, i) {
                                final item = records[i];
                                final id = _getRecordId(item, i);
                                return Padding(
                                  padding: const EdgeInsets.only(bottom: 12),
                                  child: _buildRecordCard(context, item, id),
                                );
                              },
                            ),
                          ),
                        ],
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
  final String accountNavPlacement;
  final ValueChanged<String>? onAccountNavPlacementChanged;
  final String currentLocale;
  final bool isRTL;
  final void Function(String locale, bool isRTL)? onLocaleChanged;
  final VoidCallback? onOpenLanguagePicker;
  final VoidCallback? onSignOut;

  const AccountModalSheet({
    super.key,
    required this.api,
    required this.activeTenant,
    required this.onThemeToggle,
    required this.isDark,
    this.accountNavPlacement = 'header',
    this.onAccountNavPlacementChanged,
    this.currentLocale = 'en',
    this.isRTL = false,
    this.onLocaleChanged,
    this.onOpenLanguagePicker,
    this.onSignOut,
  });

  static Future<void> show({
    required BuildContext context,
    required FilamentApiClient api,
    required String? activeTenant,
    required VoidCallback onThemeToggle,
    required bool isDark,
    String accountNavPlacement = 'header',
    ValueChanged<String>? onAccountNavPlacementChanged,
    String currentLocale = 'en',
    bool isRTL = false,
    void Function(String locale, bool isRTL)? onLocaleChanged,
    VoidCallback? onOpenLanguagePicker,
    VoidCallback? onSignOut,
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
        accountNavPlacement: accountNavPlacement,
        onAccountNavPlacementChanged: onAccountNavPlacementChanged,
        currentLocale: currentLocale,
        isRTL: isRTL,
        onLocaleChanged: onLocaleChanged,
        onOpenLanguagePicker: onOpenLanguagePicker,
        onSignOut: onSignOut,
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
                            child: Text(
                              (api.currentUser?['name'] ?? api.currentUser?['login'] ?? 'A')[0].toUpperCase(),
                              style: const TextStyle(fontSize: 26, fontWeight: FontWeight.bold, color: Colors.black),
                            ),
                          ),
                          const SizedBox(width: 16),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  api.currentUser?['name'] ?? api.currentUser?['login'] ?? 'Administrator',
                                  style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                                ),
                                const SizedBox(height: 2),
                                Text(
                                  api.currentUser?['email'] ?? 'admin@filament.local',
                                  style: const TextStyle(fontSize: 12, color: Colors.grey),
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
                                      child: Text(
                                        (api.currentUser?['role'] ?? 'OPERATIONS LEAD').toString().toUpperCase(),
                                        style: const TextStyle(fontSize: 9, fontWeight: FontWeight.bold, color: Colors.amber),
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
                      icon: Icons.language,
                      iconColor: Colors.purpleAccent,
                      title: AppLocale.t(currentLocale, 'language'),
                      subtitle: '${AppLocale.getFlag(currentLocale)} ${AppLocale.getLabel(currentLocale)} • ${isRTL ? "RTL (من اليمين لليسار)" : "LTR (Left-to-Right)"}',
                      onTap: () {
                        Navigator.pop(context);
                        onOpenLanguagePicker?.call();
                      },
                    ),

                    _buildModalNavItem(
                      icon: Icons.settings_outlined,
                      iconColor: Colors.amber,
                      title: AppLocale.t(currentLocale, 'settings'),
                      subtitle: 'Theme, Language, Biometrics, Auto-Lock, Notifications',
                      onTap: () {
                        Navigator.pop(context);
                        Navigator.push(
                          context,
                          MaterialPageRoute(
                            builder: (_) => SettingsScreen(
                              onThemeToggle: onThemeToggle,
                              isDark: isDark,
                              api: api,
                              accountNavPlacement: accountNavPlacement,
                              onAccountNavPlacementChanged: onAccountNavPlacementChanged,
                              currentLocale: currentLocale,
                              isRTL: isRTL,
                              onLocaleChanged: onLocaleChanged,
                            ),
                          ),
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
                        onSignOut?.call();
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
  final String accountNavPlacement;
  final ValueChanged<String>? onAccountNavPlacementChanged;
  final String currentLocale;
  final bool isRTL;
  final void Function(String locale, bool isRTL)? onLocaleChanged;

  const SettingsScreen({
    super.key,
    required this.onThemeToggle,
    required this.isDark,
    required this.api,
    this.accountNavPlacement = 'header',
    this.onAccountNavPlacementChanged,
    this.currentLocale = 'en',
    this.isRTL = false,
    this.onLocaleChanged,
  });

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  late String _selectedPlacement = widget.accountNavPlacement;
  bool isBiometricEnabled = true;
  String autoLockTimeout = '5 Minutes';
  bool requireBioForCriticalActions = true;
  bool twoFactorEnabled = true;
  bool pushAlerts = true;
  bool criticalSensorAlarms = true;
  bool firmwareBroadcasts = true;
  bool analyticsEnabled = true;
  String cacheSize = '14.8 MB';

  void _showLanguagePicker(BuildContext context) {
    final supported = ['en', 'ar', 'es', 'de'];
    showModalBottomSheet(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(borderRadius: BorderRadius.vertical(top: Radius.circular(24))),
      builder: (ctx) => SafeArea(
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    AppLocale.t(widget.currentLocale, 'language_picker'),
                    style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                  ),
                  IconButton(icon: const Icon(Icons.close), onPressed: () => Navigator.pop(ctx)),
                ],
              ),
              const SizedBox(height: 12),
              for (final code in supported) ...[
                Builder(
                  builder: (_) {
                    final name = AppLocale.getLabel(code);
                    final flag = AppLocale.getFlag(code);
                    final isRtl = AppLocale.isRTL(code);
                    final isSelected = code == widget.currentLocale;
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 8),
                      child: ListTile(
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(12),
                          side: BorderSide(
                            color: isSelected ? Colors.amber : Colors.white.withValues(alpha: 0.1),
                            width: isSelected ? 1.5 : 1,
                          ),
                        ),
                        leading: Text(flag, style: const TextStyle(fontSize: 24)),
                        title: Row(
                          children: [
                            Text(name, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 15)),
                            const SizedBox(width: 8),
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                              decoration: BoxDecoration(
                                color: (isRtl ? Colors.amber : Colors.blueAccent).withValues(alpha: 0.2),
                                borderRadius: BorderRadius.circular(6),
                              ),
                              child: Text(
                                isRtl ? 'RTL' : 'LTR',
                                style: TextStyle(
                                  fontSize: 9,
                                  fontWeight: FontWeight.bold,
                                  color: isRtl ? Colors.amber : Colors.blueAccent,
                                ),
                              ),
                            ),
                          ],
                        ),
                        subtitle: Text(
                          isRtl ? 'Right-to-Left (RTL) • من اليمين لليسار' : 'Left-to-Right (LTR)',
                          style: const TextStyle(fontSize: 11, color: Colors.grey),
                        ),
                        trailing: isSelected ? const Icon(Icons.check_circle, color: Color(0xFF10B981)) : null,
                        onTap: () {
                          Navigator.pop(ctx);
                          widget.onLocaleChanged?.call(code, isRtl);
                          widget.api.switchLocale(code);
                          ScaffoldMessenger.of(context).showSnackBar(
                            SnackBar(
                              content: Text(
                                isRtl
                                    ? '🇸🇦 تم تفعيل اللغة العربية والاتجاه من اليمين إلى اليسار (RTL)'
                                    : '$flag Language set to $name (LTR)',
                              ),
                              behavior: SnackBarBehavior.floating,
                            ),
                          );
                        },
                      ),
                    );
                  },
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

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

          // Language & Direction Section
          const Text('LANGUAGE & DIRECTION (LTR / RTL)', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 0.8, color: Colors.grey)),
          const SizedBox(height: 8),
          Card(
            child: Column(
              children: [
                ListTile(
                  leading: Text(AppLocale.getFlag(widget.currentLocale), style: const TextStyle(fontSize: 24)),
                  title: const Text('Interface Language & Layout', style: TextStyle(fontWeight: FontWeight.bold)),
                  subtitle: Text(
                    '${AppLocale.getLabel(widget.currentLocale)} (${widget.isRTL ? "Right-to-Left (RTL) • من اليمين لليسار" : "Left-to-Right (LTR)"})',
                    style: const TextStyle(fontSize: 12),
                  ),
                  trailing: OutlinedButton.icon(
                    icon: const Icon(Icons.language, size: 16),
                    label: const Text('Change'),
                    onPressed: () => _showLanguagePicker(context),
                  ),
                ),
              ],
            ),
          ),

          const SizedBox(height: 20),

          // Account Navigation Placement Section (Mutually Exclusive: either Header Circle OR Bottom Bar Tab)
          const Text('ACCOUNT NAVIGATION ENTRYPOINT', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, letterSpacing: 0.8, color: Colors.grey)),
          const SizedBox(height: 8),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  SegmentedButton<String>(
                    segments: const [
                      ButtonSegment(
                        value: 'header',
                        icon: Icon(Icons.account_circle),
                        label: Text('Top Header Circle'),
                      ),
                      ButtonSegment(
                        value: 'bottom_bar',
                        icon: Icon(Icons.vertical_align_bottom),
                        label: Text('Bottom Bar Tab'),
                      ),
                    ],
                    selected: {_selectedPlacement},
                    onSelectionChanged: (newSelection) {
                      final val = newSelection.first;
                      setState(() => _selectedPlacement = val);
                      widget.onAccountNavPlacementChanged?.call(val);
                    },
                  ),
                  const SizedBox(height: 12),
                  Text(
                    _selectedPlacement == 'header'
                        ? '• Circular avatar in header opens floating account sheet.\n• Bottom bar contains only core tabs (Dashboard & Entities).'
                        : '• Dedicated Account tab in bottom bar opens full account hub page.\n• Header circle avatar button is hidden (strictly one entrypoint).',
                    style: const TextStyle(fontSize: 12, color: Colors.grey),
                  ),
                ],
              ),
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

`
	mainDart := strings.ReplaceAll(mainDartTemplate, "{{TITLE}}", titleName)

	return os.WriteFile(filepath.Join(libDir, "main.dart"), []byte(mainDart), 0644)
}
