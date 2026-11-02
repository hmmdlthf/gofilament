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

  Future<void> saveLocale(String locale, bool isRTL) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final prefix = _storageKeyPrefix;
      await prefs.setString('${prefix}_locale', locale);
      await prefs.setBool('${prefix}_is_rtl', isRTL);
    } catch (_) {}
  }

  Future<Map<String, dynamic>?> restoreLocale() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final prefix = _storageKeyPrefix;
      final loc = prefs.getString('${prefix}_locale');
      if (loc != null && loc.isNotEmpty) {
        final rtl = prefs.getBool('${prefix}_is_rtl') ?? (loc.toLowerCase() == 'ar');
        return {'locale': loc, 'is_rtl': rtl};
      }
    } catch (_) {}
    return null;
  }

  Future<void> saveLocale(String locale, bool isRTL) async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final prefix = _storageKeyPrefix;
      await prefs.setString('${prefix}_locale', locale);
      await prefs.setBool('${prefix}_is_rtl', isRTL);
    } catch (_) {}
  }

  Future<Map<String, dynamic>?> restoreLocale() async {
    try {
      final prefs = await SharedPreferences.getInstance();
      final prefix = _storageKeyPrefix;
      final loc = prefs.getString('${prefix}_locale');
      if (loc != null && loc.isNotEmpty) {
        final rtl = prefs.getBool('${prefix}_is_rtl') ?? (loc.toLowerCase() == 'ar');
        return {'locale': loc, 'is_rtl': rtl};
      }
    } catch (_) {}
    return null;
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



  Map<String, dynamic>? cachedSchema;

  Future<Map<String, dynamic>> fetchSchema() async {
    final res = await http.get(Uri.parse('$baseUrl/api/schema'), headers: _headers);
    if (res.statusCode == 200) {
      cachedSchema = json.decode(res.body);
      return cachedSchema!;
    }
    throw Exception('Failed to load panel schema: ${res.statusCode}');
  }

  Future<Map<String, dynamic>> fetchResourceRecords(
    String slug, {
    int page = 1,
    int? perPage,
    String search = '',
    Map<String, dynamic>? filters,
    String? sort,
    String? direction,
  }) async {
    final queryParams = <String, String>{
      'page': page.toString(),
      if (perPage != null && perPage > 0) 'perPage': perPage.toString(),
      if (search.isNotEmpty) 'search': search,
      if (sort != null && sort.isNotEmpty) 'sort': sort,
      if (direction != null && direction.isNotEmpty) 'direction': direction,
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

  Future<List<dynamic>> fetchWidgets() async {
    final res = await http.get(Uri.parse('$baseUrl/api/widgets'), headers: _headers);
    if (res.statusCode == 200) {
      final data = json.decode(res.body);
      return (data['widgets'] as List<dynamic>?) ?? [];
    }
    return [];
  }

  Future<dynamic> fetchWidgetData(String widgetId, {String filter = ''}) async {
    final uri = Uri.parse('$baseUrl/api/widgets/$widgetId/data').replace(
      queryParameters: filter.isNotEmpty ? {'filter': filter} : null,
    );
    final res = await http.get(uri, headers: _headers);
    if (res.statusCode == 200) {
      return json.decode(res.body);
    }
    return null;
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

	// 4b. lib/locale.dart - Multi-Language & RTL/LTR Localization Support
	localeDart := `// ============================================================================
// Multi-Language & RTL/LTR Localization Support
// ============================================================================
class AppLocale {
  static const Map<String, Map<String, String>> strings = {
    'en': {
      // Navigation & Sections
      'dashboard': 'Dashboard',
      'entities': 'Entities',
      'resources': 'Resources',
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
      'save_changes': 'Save Changes',
      'cancel': 'Cancel',
      'delete': 'Delete',
      'edit': 'Edit',
      'close': 'Close',
      'apply': 'Apply',
      'back': 'Back',
      'status': 'Status',
      'active_session': 'Active Session • Enclave Secured',
      'control_center': 'Control Center',
      'language_picker': 'Select Language & Direction',
      'theme_active_dark': 'Dark theme active',
      'theme_active_light': 'Light theme active',
      'dark_mode': 'Dark Mode',
      'light_mode': 'Light Mode',
      'refresh': 'Refresh',

      // Authentication / Login
      'sign_in': 'Sign In',
      'sign_in_title': 'Sign In to Your Workspace',
      'sign_in_subtitle': 'Enter your authorized credentials to access this administration panel.',
      'username_or_email': 'Username or Email',
      'password': 'Password',
      'remember_me': 'Remember me',
      'enter_credentials': 'Please enter both username and password',
      'quick_demo_roles': 'QUICK DEMO ROLES (CLICK TO AUTOFILL)',
      'enclave_secured': 'Enclave Secured • GoFilament & GoLaravel Engine',
      'switch_to_arabic': '🇸🇦 العربية (RTL)',
      'switch_to_english': '🇺🇸 English (LTR)',

      // Records & Tables
      'search_records': 'Search records...',
      'filters': 'Filters',
      'filter': 'Filter',
      'sort_by': 'Sort By',
      'clear_sort': 'Clear Sort',
      'ascending': 'Ascending (A-Z / 0-9)',
      'descending': 'Descending (Z-A / 9-0)',
      'pagination': 'Pagination',
      'infinite_scroll': 'Infinite Scroll',
      'previous_page': 'Previous',
      'next_page': 'Next',
      'first_page': 'First',
      'last_page': 'Last',
      'showing_records': 'Showing',
      'to': 'to',
      'load_more': 'Load More',
      'loading_more': 'Loading more records...',
      'all_records_loaded': 'All records loaded',
      'clear_all': 'Clear All',
      'select_all': 'Select All',
      'deselect_all': 'Deselect All',
      'bulk_actions': 'Bulk Actions',
      'bulk_delete': 'Bulk Delete',
      'export_selected': 'Export Selected',
      'clear_selection': 'Clear Selection',
      'no_records_found': 'No records found.',
      'rows_per_page': 'Rows per page',
      'page': 'Page',
      'of': 'of',
      'actions': 'Actions',
      'create_record': 'Create Record',
      'edit_record': 'Edit Record',
      'delete_record': 'Delete Record',
      'delete_selected': 'Delete Selected',
      'delete_confirm_title': 'Delete Record?',
      'delete_records_title': 'Delete Records?',
      'delete_confirm': 'Are you sure you want to delete this record? This action cannot be undone.',
      'delete_records_confirm': 'Are you sure you want to delete these selected records? This action cannot be undone.',
      'record_created': 'Record created successfully',
      'record_updated': 'Record updated successfully',
      'record_deleted': 'Record deleted successfully',
      'records_deleted': 'records deleted successfully',
      'operation_failed': 'Operation failed. Please try again.',
      'show_relations': 'Show Related Entities',
      'hide_relations': 'Hide Related Entities',
      'toggle_relations': 'Toggle Relations',
      'relations': 'Relations',
      'attributes': 'Attributes',
      'manage': 'Manage',
      'related_records': 'Related Records',

      // Dashboard
      'operational_dashboard': 'Operational Dashboard',
      'admin_dashboard': 'Admin Dashboard',
      'performance_overview': 'Performance Overview',
      'dashboard_subtitle': 'Live operational metrics, telematics & hardware analytics',
      'refresh_dashboard': 'Refresh Dashboard',
      'active_entities': 'Active Entities',
      'system_status': 'System Status',
      'operational': 'Operational',
      'multi_tenancy': 'Multi-Tenancy',
      'quick_access_entities': 'Quick Access Entities',

      // Direction & Languages
      'direction': 'Direction',
      'rtl': 'Right-to-Left (RTL)',
      'ltr': 'Left-to-Right (LTR)',
      'english': 'English',
      'arabic': 'العربية',
    },
    'ar': {
      // Navigation & Sections
      'dashboard': 'لوحة التحكم',
      'entities': 'الكيانات',
      'resources': 'الموارد',
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
      'save_changes': 'حفظ التغييرات',
      'cancel': 'إلغاء',
      'delete': 'حذف',
      'edit': 'تعديل',
      'close': 'إغلاق',
      'apply': 'تطبيق',
      'back': 'رجوع',
      'status': 'الحالة',
      'active_session': 'جلسة نشطة • الجيب الأمني مشفر',
      'control_center': 'مركز التحكم',
      'language_picker': 'اختر اللغة واتجاه الواجهة',
      'theme_active_dark': 'الوضع الداكن مفعّل',
      'theme_active_light': 'الوضع الفاتح مفعّل',
      'dark_mode': 'الوضع الداكن',
      'light_mode': 'الوضع الفاتح',
      'refresh': 'تحديث',

      // Authentication / Login
      'sign_in': 'تسجيل الدخول',
      'sign_in_title': 'تسجيل الدخول إلى مساحة العمل',
      'sign_in_subtitle': 'أدخل بيانات الاعتماد المصرح بها للوصول إلى لوحة الإدارة.',
      'username_or_email': 'اسم المستخدم أو البريد الإلكتروني',
      'password': 'كلمة المرور',
      'remember_me': 'تذكرني',
      'enter_credentials': 'يرجى إدخال اسم المستخدم وكلمة المرور معاً',
      'quick_demo_roles': 'أدوار العرض السريع (انقر للتعبئة التلقائية)',
      'enclave_secured': 'محمي بالجيب الأمني • محرك GoFilament و GoLaravel',
      'switch_to_arabic': '🇸🇦 العربية (RTL)',
      'switch_to_english': '🇺🇸 English (LTR)',

      // Records & Tables
      'search_records': 'البحث في السجلات...',
      'filters': 'عوامل التصفية',
      'filter': 'تصفية',
      'sort_by': 'الترتيب حسب',
      'clear_sort': 'إلغاء الترتيب',
      'ascending': 'تصاعدي (أ-ي / 0-9)',
      'descending': 'تنازلي (ي-أ / 9-0)',
      'pagination': 'ترقيم الصفحات',
      'infinite_scroll': 'التمرير اللانهائي',
      'previous_page': 'السابق',
      'next_page': 'التالي',
      'first_page': 'الأول',
      'last_page': 'الأخير',
      'showing_records': 'عرض',
      'to': 'إلى',
      'load_more': 'تحميل المزيد',
      'loading_more': 'جاري تحميل المزيد من السجلات...',
      'all_records_loaded': 'تم تحميل جميع السجلات',
      'clear_all': 'مسح الكل',
      'select_all': 'تحديد الكل',
      'deselect_all': 'إلغاء تحديد الكل',
      'bulk_actions': 'إجراءات جماعية',
      'bulk_delete': 'حذف جماعي',
      'export_selected': 'تصدير المحدد',
      'clear_selection': 'مسح التحديد',
      'no_records_found': 'لم يتم العثور على سجلات.',
      'rows_per_page': 'الصفوف لكل صفحة',
      'page': 'صفحة',
      'of': 'من',
      'actions': 'إجراءات',
      'create_record': 'إنشاء سجل جديد',
      'edit_record': 'تعديل السجل',
      'delete_record': 'حذف السجل',
      'delete_selected': 'حذف المحدد',
      'delete_confirm_title': 'حذف السجل؟',
      'delete_records_title': 'حذف السجلات؟',
      'delete_confirm': 'هل أنت متأكد من حذف هذا السجل؟ لا يمكن التراجع عن هذا الإجراء.',
      'delete_records_confirm': 'هل أنت متأكد من حذف السجلات المحددة؟ لا يمكن التراجع عن هذا الإجراء.',
      'record_created': 'تم إنشاء السجل بنجاح',
      'record_updated': 'تم تحديث السجل بنجاح',
      'record_deleted': 'تم حذف السجل بنجاح',
      'records_deleted': 'سجلات تم حذفها بنجاح',
      'operation_failed': 'فشلت العملية. يرجى المحاولة مرة أخرى.',
      'show_relations': 'عرض الكيانات المرتبطة',
      'hide_relations': 'إخفاء الكيانات المرتبطة',
      'toggle_relations': 'تبديل الكيانات المرتبطة',
      'relations': 'العلاقات',
      'attributes': 'الخصائص',
      'manage': 'إدارة',
      'related_records': 'الكيانات المرتبطة',

      // Dashboard
      'operational_dashboard': 'لوحة العمليات التشغيلية',
      'admin_dashboard': 'لوحة الإدارة',
      'performance_overview': 'نظرة عامة على الأداء',
      'dashboard_subtitle': 'مؤشرات تشغيلية حية، القياس عن بُعد وتحليلات الأجهزة',
      'refresh_dashboard': 'تحديث لوحة التحكم',
      'active_entities': 'الكيانات النشطة',
      'system_status': 'حالة النظام',
      'operational': 'يعمل بكفاءة',
      'multi_tenancy': 'تعدد المستأجرين',
      'quick_access_entities': 'الوصول السريع للكيانات',

      // Direction & Languages
      'direction': 'الاتجاه',
      'rtl': 'من اليمين إلى اليسار (RTL)',
      'ltr': 'من اليسار إلى اليمين (LTR)',
      'english': 'English',
      'arabic': 'العربية',
    },
  };

  static String t(String locale, dynamic keyOrText, {Map<String, String>? remoteOverrides}) {
    if (keyOrText == null) return '';
    final raw = keyOrText.toString().trim();
    if (raw.isEmpty) return '';
    if (remoteOverrides != null && remoteOverrides.containsKey(raw)) {
      return remoteOverrides[raw]!;
    }
    final locDict = strings[locale] ?? strings['en'] ?? {};
    if (locDict.containsKey(raw)) return locDict[raw]!;
    final lower = raw.toLowerCase();
    if (locDict.containsKey(lower)) return locDict[lower]!;
    final snake = lower.replaceAll(RegExp(r'[\s\-_/]+'), '_');
    if (locDict.containsKey(snake)) return locDict[snake]!;

    // Fallback to English dictionary if current locale is not English
    if (locale != 'en') {
      final enDict = strings['en'] ?? {};
      if (enDict.containsKey(raw)) return enDict[raw]!;
      if (enDict.containsKey(lower)) return enDict[lower]!;
      if (enDict.containsKey(snake)) return enDict[snake]!;
    }

    // Humanizer fallback: replace underscores with spaces and capitalize words
    if (raw.contains('_')) {
      return raw
          .split('_')
          .map((w) => w.isNotEmpty ? '${w[0].toUpperCase()}${w.substring(1)}' : '')
          .join(' ');
    }
    return raw;
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
`
	if err := os.WriteFile(filepath.Join(libDir, "locale.dart"), []byte(localeDart), 0644); err != nil {
		return err
	}

	// 5. lib/relation_record_screen.dart - Nested Relation Managers Screen
	relationRecordDart := `import 'package:flutter/material.dart';
import 'api.dart';
import 'locale.dart';

class RelationRecordScreen extends StatefulWidget {
  final FilamentApiClient api;
  final String parentSlug;
  final String parentId;
  final String parentTitle;
  final String relSlug;
  final String relTitle;
  final String currentLocale;

  const RelationRecordScreen({
    super.key,
    required this.api,
    required this.parentSlug,
    required this.parentId,
    required this.parentTitle,
    required this.relSlug,
    required this.relTitle,
    this.currentLocale = 'en',
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
        title: Text(AppLocale.t(widget.currentLocale, "delete_records_title")),
        content: Text(AppLocale.t(widget.currentLocale, "delete_records_confirm")),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(AppLocale.t(widget.currentLocale, "cancel")),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(AppLocale.t(widget.currentLocale, "delete_selected"), style: const TextStyle(color: Colors.redAccent, fontWeight: FontWeight.bold)),
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
              content: Text("$count ${AppLocale.t(widget.currentLocale, "records_deleted")}"),
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
        title: Text(AppLocale.t(widget.currentLocale, 'delete_confirm_title')),
        content: Text('Are you sure you want to remove "$title" (#$childId)? This action cannot be undone.'),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx, false), child: Text(AppLocale.t(widget.currentLocale, 'cancel'))),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.redAccent),
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(AppLocale.t(widget.currentLocale, 'delete')),
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
                    child: Text(AppLocale.t(widget.currentLocale, 'clear_all')),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton(
                    style: FilledButton.styleFrom(backgroundColor: Colors.amber.shade600, foregroundColor: Colors.black),
                    onPressed: () => Navigator.pop(ctx),
                    child: Text(AppLocale.t(widget.currentLocale, 'apply'), style: const TextStyle(fontWeight: FontWeight.bold)),
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
                                      child: Text(AppLocale.t(widget.currentLocale, 'deselect_all'), style: const TextStyle(fontSize: 12)),
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
                  label: Text(AppLocale.t(widget.currentLocale, 'edit'), style: const TextStyle(fontSize: 12)),
                  onPressed: () => _showAddEditSheet(item),
                ),
                const SizedBox(width: 8),
                TextButton.icon(
                  icon: const Icon(Icons.delete_outline, size: 15, color: Colors.redAccent),
                  label: Text(AppLocale.t(widget.currentLocale, 'delete'), style: const TextStyle(fontSize: 12, color: Colors.redAccent)),
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
                  label: Text(AppLocale.t(widget.currentLocale, 'delete'), style: const TextStyle(color: Colors.redAccent, fontSize: 12)),
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
                              Text(isAllSelected ? AppLocale.t(widget.currentLocale, 'deselect_all') : "${AppLocale.t(widget.currentLocale, 'select_all')} (${filteredRecords.length})"),
                            ],
                          ),
                        ),
                        if (selectedRecordIds.isNotEmpty) ...[
                          PopupMenuItem(
                            value: 'delete',
                            child: Row(
                              children: [
                                const Icon(Icons.delete_sweep_rounded, color: Colors.redAccent, size: 20),
                                const SizedBox(width: 10),
                                Text(AppLocale.t(widget.currentLocale, 'bulk_delete'), style: const TextStyle(color: Colors.redAccent)),
                              ],
                            ),
                          ),
                          PopupMenuItem(
                            value: 'export',
                            child: Row(
                              children: [
                                const Icon(Icons.download_rounded, size: 20),
                                const SizedBox(width: 10),
                                Text(AppLocale.t(widget.currentLocale, 'export_selected')),
                              ],
                            ),
                          ),
                          const PopupMenuDivider(),
                          PopupMenuItem(
                            value: 'deselect',
                            child: Row(
                              children: [
                                const Icon(Icons.close_rounded, size: 20),
                                const SizedBox(width: 10),
                                Text(AppLocale.t(widget.currentLocale, 'clear_selection')),
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
                        label: Text(AppLocale.t(widget.currentLocale, 'clear_all'), style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold)),
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
import 'locale.dart';
import 'relation_record_screen.dart';
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
    final savedLoc = await api.restoreLocale();
    if (savedLoc != null) {
      _locale = savedLoc['locale'] as String? ?? 'en';
      _isRTL = savedLoc['is_rtl'] as bool? ?? false;
    }
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
    api.saveLocale(locale, isRTL);
    api.switchLocale(locale);
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
                  currentLocale: _locale,
                  onLocaleChanged: setLocale,
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
  final String currentLocale;
  final void Function(String locale, bool isRTL) onLocaleChanged;

  const LoginScreen({
    super.key,
    required this.api,
    required this.panelTitle,
    required this.isDark,
    required this.onToggleTheme,
    required this.onLoginSuccess,
    this.currentLocale = 'en',
    required this.onLocaleChanged,
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

  void _fillDemo(String username, String password) {
    setState(() {
      _usernameController.text = username;
      _passwordController.text = password;
      _errorMessage = null;
    });
  }

  Future<void> _handleLogin() async {
    final username = _usernameController.text.trim();
    final password = _passwordController.text.trim();
    if (username.isEmpty || password.isEmpty) {
      setState(() {
        _errorMessage = AppLocale.t(widget.currentLocale, 'enter_credentials');
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
                        Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            ActionChip(
                              avatar: Text(widget.currentLocale == 'ar' ? '🇺🇸' : '🇸🇦', style: const TextStyle(fontSize: 14)),
                              label: Text(
                                widget.currentLocale == 'ar' ? 'English' : 'العربية',
                                style: const TextStyle(fontSize: 12, fontWeight: FontWeight.bold),
                              ),
                              onPressed: () {
                                final nextLoc = widget.currentLocale == 'ar' ? 'en' : 'ar';
                                widget.onLocaleChanged(nextLoc, nextLoc == 'ar');
                              },
                              visualDensity: VisualDensity.compact,
                            ),
                            const SizedBox(width: 8),
                            IconButton(
                              onPressed: widget.onToggleTheme,
                              icon: Icon(isDark ? Icons.light_mode_outlined : Icons.dark_mode_outlined),
                              tooltip: AppLocale.t(widget.currentLocale, 'theme'),
                            ),
                          ],
                        ),
                      ],
                    ),
                    const SizedBox(height: 28),
                    Text(
                      AppLocale.t(widget.currentLocale, 'sign_in_title'),
                      style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      AppLocale.t(widget.currentLocale, 'sign_in_subtitle'),
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
                        labelText: AppLocale.t(widget.currentLocale, 'username_or_email'),
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
                        labelText: AppLocale.t(widget.currentLocale, 'password'),
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
                          : Text(AppLocale.t(widget.currentLocale, 'sign_in'), style: const TextStyle(fontSize: 15, fontWeight: FontWeight.bold)),
                    ),
                    const SizedBox(height: 20),
                    const Divider(),
                    const SizedBox(height: 12),
                    Text(
                      AppLocale.t(widget.currentLocale, 'quick_demo_roles'),
                      textAlign: TextAlign.center,
                      style: TextStyle(
                        fontSize: 11,
                        fontWeight: FontWeight.bold,
                        letterSpacing: 0.8,
                        color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
                      ),
                    ),
                    const SizedBox(height: 10),
                    Wrap(
                      spacing: 8,
                      runSpacing: 8,
                      alignment: WrapAlignment.center,
                      children: [
                        ActionChip(
                          avatar: const Icon(Icons.security, size: 16, color: Color(0xFFF59E0B)),
                          label: const Text('Super Admin', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600)),
                          onPressed: () => _fillDemo('admin', 'admin123'),
                        ),
                        ActionChip(
                          avatar: const Icon(Icons.admin_panel_settings_outlined, size: 16, color: Color(0xFF38BDF8)),
                          label: const Text('Operations Admin', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600)),
                          onPressed: () => _fillDemo('admin', 'admin123'),
                        ),
                        ActionChip(
                          avatar: const Icon(Icons.account_balance_outlined, size: 16, color: Color(0xFF10B981)),
                          label: const Text('Finance Auditor', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600)),
                          onPressed: () => _fillDemo('admin', 'admin123'),
                        ),
                      ],
                    ),
                    const SizedBox(height: 16),
                    Center(
                      child: Text(
                        AppLocale.t(widget.currentLocale, 'enclave_secured'),
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

class SparklinePainter extends CustomPainter {
  final List<double> data;
  final Color color;

  SparklinePainter({required this.data, required this.color});

  @override
  void paint(Canvas canvas, Size size) {
    if (data.length < 2) return;
    final min = data.reduce((a, b) => a < b ? a : b);
    final max = data.reduce((a, b) => a > b ? a : b);
    final range = (max - min) == 0 ? 1.0 : (max - min);

    final path = Path();
    final fillPath = Path();
    final stepX = size.width / (data.length - 1);

    for (int i = 0; i < data.length; i++) {
      final x = i * stepX;
      final normalized = (data[i] - min) / range;
      final y = size.height - (normalized * (size.height - 8)) - 4;
      if (i == 0) {
        path.moveTo(x, y);
        fillPath.moveTo(x, size.height);
        fillPath.lineTo(x, y);
      } else {
        path.lineTo(x, y);
        fillPath.lineTo(x, y);
      }
    }
    fillPath.lineTo(size.width, size.height);
    fillPath.close();

    final fillPaint = Paint()
      ..shader = LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [color.withValues(alpha: 0.3), color.withValues(alpha: 0.0)],
      ).createShader(Rect.fromLTWH(0, 0, size.width, size.height));
    canvas.drawPath(fillPath, fillPaint);

    final linePaint = Paint()
      ..color = color
      ..strokeWidth = 2.0
      ..style = PaintingStyle.stroke
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round;
    canvas.drawPath(path, linePaint);
  }

  @override
  bool shouldRepaint(covariant SparklinePainter oldDelegate) => true;
}

class DashboardChartPainter extends CustomPainter {
  final List<String> labels;
  final List<dynamic> datasets;
  final bool isBar;
  final bool isDark;

  DashboardChartPainter({
    required this.labels,
    required this.datasets,
    this.isBar = false,
    this.isDark = true,
  });

  @override
  void paint(Canvas canvas, Size size) {
    if (labels.isEmpty || datasets.isEmpty) return;

    final paddingBottom = 24.0;
    final chartHeight = size.height - paddingBottom;
    final chartWidth = size.width;

    double maxVal = 1.0;
    for (final ds in datasets) {
      final rawData = (ds['data'] as List<dynamic>?) ?? [];
      for (final v in rawData) {
        final d = (v is num) ? v.toDouble() : double.tryParse('$v') ?? 0.0;
        if (d > maxVal) maxVal = d;
      }
    }

    final gridPaint = Paint()
      ..color = isDark ? Colors.white.withValues(alpha: 0.06) : Colors.black.withValues(alpha: 0.06)
      ..strokeWidth = 1.0;

    for (int i = 0; i <= 3; i++) {
      final y = (chartHeight / 3) * i;
      canvas.drawLine(Offset(0, y), Offset(chartWidth, y), gridPaint);
    }

    final n = labels.length;
    final stepX = n > 1 ? chartWidth / (n - 1) : chartWidth;

    final labelStyle = TextStyle(
      color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
      fontSize: 10,
      fontWeight: FontWeight.w500,
    );

    for (int i = 0; i < n; i++) {
      final tp = TextPainter(
        text: TextSpan(text: labels[i], style: labelStyle),
        textDirection: TextDirection.ltr,
      )..layout();
      final x = n > 1 ? (i * stepX) - (tp.width / 2) : (chartWidth / 2) - (tp.width / 2);
      tp.paint(canvas, Offset(x.clamp(0.0, chartWidth - tp.width), chartHeight + 6));
    }

    if (isBar) {
      final barWidth = (chartWidth / n) * 0.5 / datasets.length;
      for (int dIdx = 0; dIdx < datasets.length; dIdx++) {
        final ds = datasets[dIdx];
        final rawColor = ds['backgroundColor'] ?? ds['borderColor'] ?? '#10b981';
        final color = _parseColor(rawColor.toString());
        final rawData = (ds['data'] as List<dynamic>?) ?? [];
        final barPaint = Paint()..color = color;

        for (int i = 0; i < rawData.length && i < n; i++) {
          final val = (rawData[i] is num) ? rawData[i].toDouble() : double.tryParse('${rawData[i]}') ?? 0.0;
          final h = (val / maxVal) * (chartHeight - 10);
          final groupStartX = (i * (chartWidth / n)) + ((chartWidth / n) * 0.25);
          final barX = groupStartX + (dIdx * barWidth);
          final barY = chartHeight - h;
          final rRect = RRect.fromRectAndCorners(
            Rect.fromLTWH(barX, barY, barWidth * 0.85, h),
            topLeft: const Radius.circular(4),
            topRight: const Radius.circular(4),
          );
          canvas.drawRRect(rRect, barPaint);
        }
      }
    } else {
      for (final ds in datasets) {
        final rawColor = ds['borderColor'] ?? ds['backgroundColor'] ?? '#10b981';
        final color = _parseColor(rawColor.toString());
        final rawData = (ds['data'] as List<dynamic>?) ?? [];
        if (rawData.length < 2) continue;

        final path = Path();
        final fillPath = Path();

        for (int i = 0; i < rawData.length && i < n; i++) {
          final val = (rawData[i] is num) ? rawData[i].toDouble() : double.tryParse('${rawData[i]}') ?? 0.0;
          final x = i * stepX;
          final y = chartHeight - ((val / maxVal) * (chartHeight - 12)) - 6;

          if (i == 0) {
            path.moveTo(x, y);
            fillPath.moveTo(x, chartHeight);
            fillPath.lineTo(x, y);
          } else {
            path.lineTo(x, y);
            fillPath.lineTo(x, y);
          }
        }

        fillPath.lineTo((rawData.length - 1) * stepX, chartHeight);
        fillPath.close();

        final fillPaint = Paint()
          ..shader = LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [color.withValues(alpha: 0.25), color.withValues(alpha: 0.0)],
          ).createShader(Rect.fromLTWH(0, 0, chartWidth, chartHeight));
        canvas.drawPath(fillPath, fillPaint);

        final linePaint = Paint()
          ..color = color
          ..strokeWidth = 2.5
          ..style = PaintingStyle.stroke
          ..strokeCap = StrokeCap.round
          ..strokeJoin = StrokeJoin.round;
        canvas.drawPath(path, linePaint);

        final pointPaint = Paint()..color = color;
        final innerPaint = Paint()..color = isDark ? const Color(0xFF1E293B) : Colors.white;
        for (int i = 0; i < rawData.length && i < n; i++) {
          final val = (rawData[i] is num) ? rawData[i].toDouble() : double.tryParse('${rawData[i]}') ?? 0.0;
          final x = i * stepX;
          final y = chartHeight - ((val / maxVal) * (chartHeight - 12)) - 6;
          canvas.drawCircle(Offset(x, y), 4.5, pointPaint);
          canvas.drawCircle(Offset(x, y), 2.5, innerPaint);
        }
      }
    }
  }

  static Color _parseColor(String c) {
    c = c.replaceAll('#', '').trim();
    if (c.length == 6) {
      return Color(int.parse('FF$c', radix: 16));
    } else if (c.length == 8) {
      return Color(int.parse(c, radix: 16));
    }
    return const Color(0xFFF59E0B);
  }

  @override
  bool shouldRepaint(covariant DashboardChartPainter oldDelegate) => true;
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
          toolbarHeight: 48,
          titleSpacing: 8,
          title: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              const BrandLogoWidget(
                isCompact: true,
                brandName: 'Benzat',
                imageAsset: 'assets/images/app_logo.png',
              ),
              const SizedBox(width: 8),
              Flexible(
                child: Text(
                  !isHeaderPlacement && _currentIndex == 2 ? 'Account & Profile' : brandName,
                  style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ],
          ),
          actions: [
            IconButton(
              icon: const Icon(Icons.language, size: 19),
              tooltip: 'Language & Direction / اللغة',
              visualDensity: VisualDensity.compact,
              padding: const EdgeInsets.all(6),
              onPressed: () => showLanguageDialog(context),
            ),
            IconButton(
              icon: Icon(widget.isDark ? Icons.light_mode : Icons.dark_mode, size: 19),
              visualDensity: VisualDensity.compact,
              padding: const EdgeInsets.all(6),
              onPressed: widget.onToggleTheme,
            ),
            IconButton(
              icon: const Icon(Icons.corporate_fare, size: 19),
              visualDensity: VisualDensity.compact,
              padding: const EdgeInsets.all(6),
              onPressed: showTenantSwitcher,
            ),
            // Header circle account button is shown ONLY when placement is 'header' (never both!)
            if (isHeaderPlacement)
              Padding(
                padding: const EdgeInsets.only(right: 6, left: 2),
                child: CircularAccountButton(
                  radius: 14,
                  initial: (api.currentUser?['name'] ?? api.currentUser?['login'] ?? 'A')[0].toUpperCase(),
                  name: api.currentUser?['name'] ?? api.currentUser?['login'] ?? 'Administrator',
                  email: api.currentUser?['email'] ?? 'admin@benzat.com',
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
        bottomNavigationBar: NavigationBarTheme(
          data: NavigationBarThemeData(
            height: 52,
            iconTheme: WidgetStateProperty.all(const IconThemeData(size: 19)),
            labelTextStyle: WidgetStateProperty.all(
              const TextStyle(fontSize: 10, fontWeight: FontWeight.w600),
            ),
          ),
          child: NavigationBar(
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
                const BrandLogoWidget(
                  isCompact: true,
                  brandName: 'Benzat',
                  imageAsset: 'assets/images/app_logo.png',
                ),
                const SizedBox(width: 10),
              ],
              Text(
                '${AppLocale.t(widget.currentLocale, brandName)} ${AppLocale.t(widget.currentLocale, "control_center")}',
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
                  email: api.currentUser?['email'] ?? 'admin@benzat.com',
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
              child: const BrandLogoWidget(
                isCompact: true,
                brandName: 'Benzat',
                imageAsset: 'assets/images/app_logo.png',
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
                      imageAsset: 'assets/images/app_logo.png',
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
                      label: AppLocale.t(widget.currentLocale, r['model_name'] ?? r['slug'] ?? 'Resource'),
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
                      AppLocale.t(widget.currentLocale, 'resources').toUpperCase(),
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
                      label: AppLocale.t(widget.currentLocale, r['model_name'] ?? r['slug'] ?? 'Resource'),
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
                        AppLocale.t(widget.currentLocale, 'preferences').toUpperCase(),
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

  // --- Dynamic Dashboard Views (Mobile & Desktop) ---
  Widget _buildMobileDashboard() {
    return _buildDashboardView(true);
  }

  Widget _buildDesktopDashboard() {
    return _buildDashboardView(false);
  }

  Widget _buildDashboardView(bool isMobile) {
    final widgetsList = (schema?['widgets'] as List<dynamic>?) ?? [];
    final isDark = Theme.of(context).brightness == Brightness.dark;

    if (widgetsList.isEmpty) {
      return _buildFallbackDashboard(isMobile);
    }

    return RefreshIndicator(
      onRefresh: loadSchema,
      child: ListView(
        padding: EdgeInsets.all(isMobile ? 12 : 24),
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    AppLocale.t(widget.currentLocale, 'operational_dashboard'),
                    style: TextStyle(
                      fontSize: isMobile ? 18 : 26,
                      fontWeight: FontWeight.bold,
                      letterSpacing: -0.5,
                    ),
                  ),
                  const SizedBox(height: 2),
                  Text(
                    AppLocale.t(widget.currentLocale, 'dashboard_subtitle'),
                    style: TextStyle(
                      fontSize: isMobile ? 11 : 12,
                      color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
                    ),
                  ),
                ],
              ),
              IconButton.filledTonal(
                visualDensity: VisualDensity.compact,
                tooltip: AppLocale.t(widget.currentLocale, 'refresh_dashboard'),
                icon: const Icon(Icons.refresh_rounded, size: 18),
                onPressed: loadSchema,
              ),
            ],
          ),
          const SizedBox(height: 20),
          ...widgetsList.map((w) {
            return Padding(
              padding: const EdgeInsets.only(bottom: 20),
              child: _buildWidgetCard(Map<String, dynamic>.from(w as Map), isMobile, isDark),
            );
          }),
        ],
      ),
    );
  }

  Widget _buildFallbackDashboard(bool isMobile) {
    return RefreshIndicator(
      onRefresh: loadSchema,
      child: ListView(
        padding: EdgeInsets.all(isMobile ? 16 : 24),
        children: [
          Text(
            isMobile ? AppLocale.t(widget.currentLocale, 'performance_overview') : AppLocale.t(widget.currentLocale, 'admin_dashboard'),
            style: TextStyle(fontSize: isMobile ? 20 : 28, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(child: _kpiCard(AppLocale.t(widget.currentLocale, 'active_entities'), '${schema?['resources']?.length ?? 0}', Colors.indigo, Icons.layers)),
              const SizedBox(width: 12),
              Expanded(child: _kpiCard(AppLocale.t(widget.currentLocale, 'system_status'), AppLocale.t(widget.currentLocale, 'operational'), const Color(0xFF10B981), Icons.check_circle)),
              if (!isMobile) ...[
                const SizedBox(width: 12),
                Expanded(child: _kpiCard(AppLocale.t(widget.currentLocale, 'multi_tenancy'), schema?['active_tenant'] ?? 'Global', Colors.amber, Icons.business)),
              ],
            ],
          ),
          const SizedBox(height: 24),
          Text(AppLocale.t(widget.currentLocale, 'quick_access_entities'), style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
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

  Widget _buildWidgetCard(Map<String, dynamic> w, bool isMobile, bool isDark) {
    final type = (w['type'] ?? '').toString();
    switch (type) {
      case 'banner':
        return _buildBannerWidget(w, isMobile, isDark);
      case 'stats_overview':
        return _buildStatsOverviewWidget(w, isMobile, isDark);
      case 'chart':
        return _buildChartWidget(w, isMobile, isDark);
      case 'breakdown':
        return _buildBreakdownWidget(w, isMobile, isDark);
      case 'progress':
        return _buildProgressWidget(w, isMobile, isDark);
      case 'quick_actions':
        return _buildQuickActionsWidget(w, isMobile, isDark);
      case 'timeline':
        return _buildTimelineWidget(w, isMobile, isDark);
      case 'table':
        return _buildTableWidget(w, isMobile, isDark);
      default:
        return const SizedBox();
    }
  }

  Widget _buildBannerWidget(Map<String, dynamic> w, bool isMobile, bool isDark) {
    final title = AppLocale.t(widget.currentLocale, w['title'] ?? 'Welcome');
    final subtitle = AppLocale.t(widget.currentLocale, w['subtitle'] ?? '');
    final actionLabel = AppLocale.t(widget.currentLocale, w['action_label'] ?? '');
    final actionUrl = (w['action_url'] ?? '').toString();
    final icon = _resolveHeroicon((w['icon'] ?? 'heroicon-sparkles').toString());

    return Container(
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
          colors: isDark
              ? [const Color(0xFF1E293B), const Color(0xFF0F172A)]
              : [const Color(0xFFFEF3C7), const Color(0xFFFFFBEB)],
        ),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: Colors.amber.withValues(alpha: 0.35)),
        boxShadow: [
          BoxShadow(
            color: Colors.amber.withValues(alpha: 0.05),
            blurRadius: 10,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      padding: EdgeInsets.all(isMobile ? 12 : 20),
      child: Row(
        children: [
          Container(
            padding: EdgeInsets.all(isMobile ? 8 : 12),
            decoration: BoxDecoration(
              color: Colors.amber.withValues(alpha: 0.2),
              shape: BoxShape.circle,
            ),
            child: Icon(icon, color: Colors.amber.shade400, size: isMobile ? 22 : 28),
          ),
          SizedBox(width: isMobile ? 10 : 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(title, style: TextStyle(fontSize: isMobile ? 14 : 16, fontWeight: FontWeight.bold)),
                if (subtitle.isNotEmpty) ...[
                  const SizedBox(height: 2),
                  Text(
                    subtitle,
                    style: TextStyle(
                      fontSize: isMobile ? 11 : 13,
                      color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
                    ),
                  ),
                ],
              ],
            ),
          ),
          if (actionLabel.isNotEmpty && !isMobile) ...[
            const SizedBox(width: 16),
            FilledButton.tonal(
              style: FilledButton.styleFrom(
                backgroundColor: Colors.amber.withValues(alpha: 0.25),
                foregroundColor: isDark ? Colors.amber.shade200 : Colors.amber.shade900,
              ),
              onPressed: () => _handleWidgetUrl(actionUrl),
              child: Text(actionLabel, style: const TextStyle(fontWeight: FontWeight.bold)),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildStatsOverviewWidget(Map<String, dynamic> w, bool isMobile, bool isDark) {
    final stats = (w['stats'] as List<dynamic>?) ?? [];
    if (stats.isEmpty) return const SizedBox();

    final cardBg = isDark ? const Color(0xFF1E293B) : Colors.white;
    final borderColor = isDark ? const Color(0xFF334155) : const Color(0xFFE2E8F0);

    final statWidgets = stats.map((sRaw) {
      final s = Map<String, dynamic>.from(sRaw as Map);
      final label = AppLocale.t(widget.currentLocale, s['label'] ?? '');
      final value = (s['value'] ?? '').toString();
      final desc = AppLocale.t(widget.currentLocale, s['description'] ?? '');
      final colorRole = (s['color'] ?? 'primary').toString();
      final color = _resolveColorRole(colorRole);
      final chartRaw = (s['chart'] as List<dynamic>?) ?? [];
      final chartData = chartRaw.map((v) => (v as num).toDouble()).toList();

      return Container(
        padding: EdgeInsets.all(isMobile ? 10 : 16),
        decoration: BoxDecoration(
          color: cardBg,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: borderColor),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.03),
              blurRadius: 8,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Text(
                    label.toUpperCase(),
                    style: TextStyle(
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                      letterSpacing: 0.6,
                      color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
                    ),
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (chartData.length >= 2)
                  SizedBox(
                    width: isMobile ? 55 : 70,
                    height: isMobile ? 22 : 28,
                    child: CustomPaint(
                      painter: SparklinePainter(data: chartData, color: color),
                    ),
                  ),
              ],
            ),
            const SizedBox(height: 6),
            Text(
              value,
              style: TextStyle(fontSize: isMobile ? 18 : 24, fontWeight: FontWeight.bold, letterSpacing: -0.5),
            ),
            if (desc.isNotEmpty) ...[
              const SizedBox(height: 4),
              Row(
                children: [
                  Icon(
                    desc.contains('-') ? Icons.trending_down_rounded : Icons.trending_up_rounded,
                    size: 13,
                    color: color,
                  ),
                  const SizedBox(width: 4),
                  Expanded(
                    child: Text(
                      desc,
                      style: TextStyle(fontSize: isMobile ? 10 : 11, fontWeight: FontWeight.w500, color: color),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ],
              ),
            ],
          ],
        ),
      );
    }).toList();

    if (isMobile) {
      return GridView.count(
        crossAxisCount: 2,
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        mainAxisSpacing: 10,
        crossAxisSpacing: 10,
        childAspectRatio: 1.45,
        children: statWidgets,
      );
    }

    return Row(
      children: statWidgets
          .map((sw) => Expanded(child: Padding(padding: const EdgeInsets.symmetric(horizontal: 6), child: sw)))
          .toList(),
    );
  }

  Widget _buildChartWidget(Map<String, dynamic> w, bool isMobile, bool isDark) {
    final heading = (w['heading'] ?? 'Chart Analytics').toString();
    final description = (w['description'] ?? '').toString();
    final chartType = (w['chart_type'] ?? 'line').toString();
    final isBar = chartType == 'bar';
    final dataObj = w['data'] as Map<String, dynamic>? ?? {};
    final labels = ((dataObj['labels'] as List<dynamic>?) ?? []).map((e) => e.toString()).toList();
    final datasets = (dataObj['datasets'] as List<dynamic>?) ?? [];

    final cardBg = isDark ? const Color(0xFF1E293B) : Colors.white;
    final borderColor = isDark ? const Color(0xFF334155) : const Color(0xFFE2E8F0);

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cardBg,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: borderColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(heading, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
                    if (description.isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Text(
                        description,
                        style: TextStyle(
                          fontSize: 12,
                          color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
                        ),
                      ),
                    ],
                  ],
                ),
              ),
              if (datasets.isNotEmpty) ...[
                Wrap(
                  spacing: 12,
                  children: datasets.map((ds) {
                    final rawColor = ds['borderColor'] ?? ds['backgroundColor'] ?? '#10b981';
                    final color = DashboardChartPainter._parseColor(rawColor.toString());
                    final label = (ds['label'] ?? '').toString();
                    return Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Container(
                          width: 8,
                          height: 8,
                          decoration: BoxDecoration(color: color, shape: BoxShape.circle),
                        ),
                        const SizedBox(width: 4),
                        Text(label, style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w500)),
                      ],
                    );
                  }).toList(),
                ),
              ],
            ],
          ),
          const SizedBox(height: 24),
          SizedBox(
            height: isMobile ? 180 : 240,
            width: double.infinity,
            child: CustomPaint(
              painter: DashboardChartPainter(
                labels: labels,
                datasets: datasets,
                isBar: isBar,
                isDark: isDark,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildBreakdownWidget(Map<String, dynamic> w, bool isMobile, bool isDark) {
    final heading = AppLocale.t(widget.currentLocale, w['heading'] ?? 'Distribution Breakdown');
    final description = (w['description'] ?? '').toString();
    final items = (w['items'] as List<dynamic>?) ?? [];

    final cardBg = isDark ? const Color(0xFF1E293B) : Colors.white;
    final borderColor = isDark ? const Color(0xFF334155) : const Color(0xFFE2E8F0);

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cardBg,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: borderColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(heading, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
          if (description.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(
              description,
              style: TextStyle(
                fontSize: 12,
                color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
              ),
            ),
          ],
          const SizedBox(height: 16),
          // Multi-color segmented distribution bar
          ClipRRect(
            borderRadius: BorderRadius.circular(8),
            child: SizedBox(
              height: 12,
              child: Row(
                children: items.map((itRaw) {
                  final it = Map<String, dynamic>.from(itRaw as Map);
                  final pct = (it['percentage'] as num?)?.toDouble() ?? 10.0;
                  final colorStr = (it['color'] ?? '#10b981').toString();
                  final color = DashboardChartPainter._parseColor(colorStr);
                  return Expanded(
                    flex: (pct * 10).toInt().clamp(1, 1000),
                    child: Container(color: color),
                  );
                }).toList(),
              ),
            ),
          ),
          const SizedBox(height: 16),
          // Category items
          ...items.map((itRaw) {
            final it = Map<String, dynamic>.from(itRaw as Map);
            final label = AppLocale.t(widget.currentLocale, it['label'] ?? '');
            final formatted = (it['formatted'] ?? '').toString();
            final pct = (it['percentage'] as num?)?.toDouble() ?? 0.0;
            final colorStr = (it['color'] ?? '#10b981').toString();
            final color = DashboardChartPainter._parseColor(colorStr);

            return Padding(
              padding: const EdgeInsets.symmetric(vertical: 4),
              child: Row(
                children: [
                  Container(width: 10, height: 10, decoration: BoxDecoration(color: color, shape: BoxShape.circle)),
                  const SizedBox(width: 8),
                  Text(label, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500)),
                  const Spacer(),
                  if (formatted.isNotEmpty) ...[
                    Text(
                      formatted,
                      style: TextStyle(
                        fontSize: 12,
                        color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
                      ),
                    ),
                    const SizedBox(width: 10),
                  ],
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(
                      color: color.withValues(alpha: 0.15),
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: Text(
                      '${pct.toStringAsFixed(1)}%',
                      style: TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: color),
                    ),
                  ),
                ],
              ),
            );
          }),
        ],
      ),
    );
  }

  Widget _buildProgressWidget(Map<String, dynamic> w, bool isMobile, bool isDark) {
    final heading = (w['heading'] ?? 'Quota Progress').toString();
    final description = (w['description'] ?? '').toString();
    final items = (w['items'] as List<dynamic>?) ?? [];

    final cardBg = isDark ? const Color(0xFF1E293B) : Colors.white;
    final borderColor = isDark ? const Color(0xFF334155) : const Color(0xFFE2E8F0);

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cardBg,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: borderColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(heading, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
          if (description.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(
              description,
              style: TextStyle(
                fontSize: 12,
                color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
              ),
            ),
          ],
          const SizedBox(height: 16),
          ...items.map((itRaw) {
            final it = Map<String, dynamic>.from(itRaw as Map);
            final label = AppLocale.t(widget.currentLocale, it['label'] ?? '');
            final pct = (it['percentage'] as num?)?.toDouble() ?? 0.0;
            final colorRole = (it['color'] ?? 'emerald').toString();
            final color = _resolveColorRole(colorRole);
            final subtitle = (it['subtitle'] ?? '').toString();

            return Padding(
              padding: const EdgeInsets.only(bottom: 14),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Text(label, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w600)),
                      Text('${pct.toStringAsFixed(1)}%', style: TextStyle(fontSize: 12, fontWeight: FontWeight.bold, color: color)),
                    ],
                  ),
                  const SizedBox(height: 6),
                  ClipRRect(
                    borderRadius: BorderRadius.circular(6),
                    child: LinearProgressIndicator(
                      value: (pct / 100.0).clamp(0.0, 1.0),
                      minHeight: 7,
                      backgroundColor: isDark ? Colors.white12 : Colors.black12,
                      valueColor: AlwaysStoppedAnimation<Color>(color),
                    ),
                  ),
                  if (subtitle.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(
                      subtitle,
                      style: TextStyle(
                        fontSize: 11,
                        color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
                      ),
                    ),
                  ],
                ],
              ),
            );
          }),
        ],
      ),
    );
  }

  Widget _buildQuickActionsWidget(Map<String, dynamic> w, bool isMobile, bool isDark) {
    final heading = AppLocale.t(widget.currentLocale, w['heading'] ?? 'Quick Actions');
    final description = (w['description'] ?? '').toString();
    final actions = (w['actions'] as List<dynamic>?) ?? [];

    final cardBg = isDark ? const Color(0xFF1E293B) : Colors.white;
    final borderColor = isDark ? const Color(0xFF334155) : const Color(0xFFE2E8F0);

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cardBg,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: borderColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(heading, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
          if (description.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(
              description,
              style: TextStyle(
                fontSize: 12,
                color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
              ),
            ),
          ],
          const SizedBox(height: 14),
          isMobile
              ? Column(
                  children: actions.map((aRaw) => _buildQuickActionItem(Map<String, dynamic>.from(aRaw as Map), isDark)).toList(),
                )
              : GridView.count(
                  crossAxisCount: 2,
                  shrinkWrap: true,
                  physics: const NeverScrollableScrollPhysics(),
                  crossAxisSpacing: 12,
                  mainAxisSpacing: 12,
                  childAspectRatio: 2.8,
                  children: actions.map((aRaw) => _buildQuickActionItem(Map<String, dynamic>.from(aRaw as Map), isDark)).toList(),
                ),
        ],
      ),
    );
  }

  Widget _buildQuickActionItem(Map<String, dynamic> a, bool isDark) {
    final label = (a['label'] ?? '').toString();
    final desc = (a['description'] ?? '').toString();
    final url = (a['url'] ?? '').toString();
    final colorRole = (a['color'] ?? 'primary').toString();
    final color = _resolveColorRole(colorRole);
    final icon = _resolveHeroicon((a['icon'] ?? 'heroicon-arrow-right').toString());

    return Card(
      elevation: 0,
      margin: const EdgeInsets.only(bottom: 8),
      color: isDark ? const Color(0xFF0F172A) : const Color(0xFFF8FAFC),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: isDark ? const Color(0xFF334155) : const Color(0xFFE2E8F0)),
      ),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () => _handleWidgetUrl(url),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 12),
          child: Row(
            children: [
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: color.withValues(alpha: 0.15),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Icon(icon, size: 20, color: color),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Text(label, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.bold)),
                    if (desc.isNotEmpty) ...[
                      const SizedBox(height: 2),
                      Text(
                        desc,
                        style: TextStyle(
                          fontSize: 11,
                          color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ],
                  ],
                ),
              ),
              const Icon(Icons.arrow_forward_ios_rounded, size: 14, color: Colors.grey),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildTimelineWidget(Map<String, dynamic> w, bool isMobile, bool isDark) {
    final heading = (w['heading'] ?? 'Recent Activities').toString();
    final description = (w['description'] ?? '').toString();
    final items = (w['items'] as List<dynamic>?) ?? [];

    final cardBg = isDark ? const Color(0xFF1E293B) : Colors.white;
    final borderColor = isDark ? const Color(0xFF334155) : const Color(0xFFE2E8F0);

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cardBg,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: borderColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(heading, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
          if (description.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(
              description,
              style: TextStyle(
                fontSize: 12,
                color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
              ),
            ),
          ],
          const SizedBox(height: 16),
          ...items.map((itRaw) {
            final it = Map<String, dynamic>.from(itRaw as Map);
            final title = (it['title'] ?? '').toString();
            final desc = (it['description'] ?? '').toString();
            final timestamp = (it['timestamp'] ?? '').toString();
            final user = (it['user'] ?? '').toString();
            final colorRole = (it['color'] ?? 'sky').toString();
            final color = _resolveColorRole(colorRole);

            return Padding(
              padding: const EdgeInsets.only(bottom: 14),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    margin: const EdgeInsets.only(top: 4),
                    width: 10,
                    height: 10,
                    decoration: BoxDecoration(color: color, shape: BoxShape.circle),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Text(title, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.bold)),
                            if (timestamp.isNotEmpty)
                              Text(
                                timestamp,
                                style: TextStyle(
                                  fontSize: 11,
                                  color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
                                ),
                              ),
                          ],
                        ),
                        if (desc.isNotEmpty) ...[
                          const SizedBox(height: 2),
                          Text(
                            desc,
                            style: TextStyle(
                              fontSize: 12,
                              color: isDark ? const Color(0xFF94A3B8) : const Color(0xFF64748B),
                            ),
                          ),
                        ],
                        if (user.isNotEmpty) ...[
                          const SizedBox(height: 4),
                          Container(
                            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                            decoration: BoxDecoration(
                              color: isDark ? Colors.white12 : Colors.black12,
                              borderRadius: BorderRadius.circular(4),
                            ),
                            child: Text(
                              '@$user',
                              style: const TextStyle(fontSize: 10, fontWeight: FontWeight.bold),
                            ),
                          ),
                        ],
                      ],
                    ),
                  ),
                ],
              ),
            );
          }),
        ],
      ),
    );
  }

  Widget _buildTableWidget(Map<String, dynamic> w, bool isMobile, bool isDark) {
    final heading = AppLocale.t(widget.currentLocale, w['heading'] ?? 'Latest Records');
    final columns = (w['columns'] as List<dynamic>?) ?? [];
    final rows = (w['rows'] as List<dynamic>?) ?? [];

    final cardBg = isDark ? const Color(0xFF1E293B) : Colors.white;
    final borderColor = isDark ? const Color(0xFF334155) : const Color(0xFFE2E8F0);

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cardBg,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: borderColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(heading, style: const TextStyle(fontSize: 16, fontWeight: FontWeight.bold)),
          const SizedBox(height: 12),
          if (rows.isEmpty)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 16),
              child: Center(child: Text('No records available.')),
            )
          else
            SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: DataTable(
                columns: columns.map((c) {
                  final label = (c['label'] ?? c['name'] ?? '').toString();
                  return DataColumn(label: Text(label, style: const TextStyle(fontWeight: FontWeight.bold)));
                }).toList(),
                rows: rows.map((rRaw) {
                  final r = Map<String, dynamic>.from(rRaw as Map);
                  return DataRow(
                    cells: columns.map((c) {
                      final colName = (c['name'] ?? '').toString();
                      final val = (r[colName] ?? '').toString();
                      return DataCell(Text(val));
                    }).toList(),
                  );
                }).toList(),
              ),
            ),
        ],
      ),
    );
  }

  Color _resolveColorRole(String role) {
    switch (role.toLowerCase()) {
      case 'emerald':
      case 'success':
        return const Color(0xFF10B981);
      case 'amber':
      case 'warning':
        return const Color(0xFFF59E0B);
      case 'red':
      case 'danger':
        return const Color(0xFFEF4444);
      case 'sky':
      case 'info':
        return const Color(0xFF0284C7);
      case 'purple':
      case 'indigo':
        return const Color(0xFF8B5CF6);
      default:
        return const Color(0xFFF59E0B);
    }
  }

  IconData _resolveHeroicon(String name) {
    final lower = name.toLowerCase();
    if (lower.contains('sparkles') || lower.contains('star')) return Icons.auto_awesome;
    if (lower.contains('truck') || lower.contains('vehicle')) return Icons.local_shipping_outlined;
    if (lower.contains('building') || lower.contains('office') || lower.contains('station')) return Icons.business_outlined;
    if (lower.contains('cpu') || lower.contains('chip') || lower.contains('nfc')) return Icons.nfc_outlined;
    if (lower.contains('bank') || lower.contains('money') || lower.contains('wallet') || lower.contains('currency')) return Icons.account_balance_wallet_outlined;
    if (lower.contains('user') || lower.contains('person')) return Icons.people_outline;
    if (lower.contains('check') || lower.contains('shield')) return Icons.verified_outlined;
    if (lower.contains('clock') || lower.contains('time')) return Icons.schedule_outlined;
    return Icons.widgets_outlined;
  }

  void _handleWidgetUrl(String url) {
    if (url.isEmpty) return;
    final parts = url.split('/').where((s) => s.isNotEmpty).toList();
    if (parts.isNotEmpty) {
      final slug = parts.last;
      final resources = (schema?['resources'] as List<dynamic>?) ?? [];
      final match = resources.firstWhere((r) => r['slug'] == slug, orElse: () => null);
      if (match != null) {
        setState(() {
          selectedResourceSlug = slug;
          _currentIndex = 1;
        });
      }
    }
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
    final isDark = Theme.of(context).brightness == Brightness.dark;
    if (resources.isEmpty) {
      return const Center(child: Text('No resources available. Check server connection.'));
    }
    return Column(
      children: [
        // Horizontal scrollable resource pill selector
        Container(
          height: 42,
          padding: const EdgeInsets.symmetric(vertical: 4, horizontal: 8),
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.surface,
            border: Border(bottom: BorderSide(color: isDark ? Colors.white12 : Colors.black12)),
          ),
          child: ListView.separated(
            scrollDirection: Axis.horizontal,
            itemCount: resources.length,
            separatorBuilder: (_, __) => const SizedBox(width: 6),
            itemBuilder: (context, i) {
              final r = resources[i];
              final slug = r['slug'] as String;
              final name = (r['model_name'] ?? slug) as String;
              final isSel = slug == selectedResourceSlug;
              return ChoiceChip(
                visualDensity: VisualDensity.compact,
                padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 0),
                labelStyle: TextStyle(
                  fontSize: 12,
                  fontWeight: isSel ? FontWeight.bold : FontWeight.normal,
                ),
                label: Text(AppLocale.t(widget.currentLocale, name)),
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
              ? Center(child: Text(AppLocale.t(widget.currentLocale, 'select_resource_above')))
              : Builder(builder: (_) {
                  final cur = resources.firstWhere((r) => r['slug'] == selectedResourceSlug, orElse: () => null);
                  final rels = cur != null ? (cur['relations'] as List<dynamic>?) : null;
                  final filters = cur != null ? (cur['filters'] as List<dynamic>?) : null;
                  final tabs = cur != null ? (cur['tabs'] as List<dynamic>?) : null;
                  final listMode = cur != null ? cur['list_mode'] as String? : null;
                  final perPage = cur != null ? (cur['per_page'] as num?)?.toInt() : null;
                  List<int>? perPageOpts;
                  if (cur != null && cur['per_page_options'] is List) {
                    perPageOpts = (cur['per_page_options'] as List).map((e) => (e as num).toInt()).toList();
                  }
                  return ResourceRecordList(
                    api: api,
                    slug: selectedResourceSlug!,
                    relations: rels,
                    filters: filters,
                    tabs: tabs,
                    currentLocale: widget.currentLocale,
                    configuredListMode: listMode,
                    configuredPerPage: perPage,
                    configuredPerPageOptions: perPageOpts,
                  );
                }),
        ),
      ],
    );
  }

  // --- Desktop Views ---

  Widget _buildDesktopResources(List<dynamic> resources) {
    if (selectedResourceSlug == null && resources.isNotEmpty) {
      selectedResourceSlug = resources.first['slug'];
    }
    if (selectedResourceSlug == null) {
      return Center(child: Text(AppLocale.t(widget.currentLocale, 'select_resource_above')));
    }
    final cur = resources.firstWhere((r) => r['slug'] == selectedResourceSlug, orElse: () => null);
    final rels = cur != null ? (cur['relations'] as List<dynamic>?) : null;
    final filters = cur != null ? (cur['filters'] as List<dynamic>?) : null;
    final tabs = cur != null ? (cur['tabs'] as List<dynamic>?) : null;
    final listMode = cur != null ? cur['list_mode'] as String? : null;
    final perPage = cur != null ? (cur['per_page'] as num?)?.toInt() : null;
    List<int>? perPageOpts;
    if (cur != null && cur['per_page_options'] is List) {
      perPageOpts = (cur['per_page_options'] as List).map((e) => (e as num).toInt()).toList();
    }
    return ResourceRecordList(
      api: api,
      slug: selectedResourceSlug!,
      relations: rels,
      filters: filters,
      tabs: tabs,
      currentLocale: widget.currentLocale,
      configuredListMode: listMode,
      configuredPerPage: perPage,
      configuredPerPageOptions: perPageOpts,
    );
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
                  child: const Center(
                    child: Text('S', style: TextStyle(color: Colors.black, fontWeight: FontWeight.bold, fontSize: 22)),
                  ),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          const Text(
                            'Sarah Connor',
                            style: TextStyle(fontSize: 17, fontWeight: FontWeight.bold),
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
                      const Text(
                        'sarah@fleetpulse.io • Operations Lead',
                        style: TextStyle(fontSize: 12, color: Colors.grey),
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
                title: AppLocale.t(widget.currentLocale, 'billing'),
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
                title: AppLocale.t(widget.currentLocale, 'biometrics'),
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
        Text(AppLocale.t(widget.currentLocale, 'quick_preferences'), style: TextStyle(fontSize: 13, fontWeight: FontWeight.bold, letterSpacing: 0.6, color: Colors.grey)),
        const SizedBox(height: 8),

        Card(
          child: Column(
            children: [
              ListTile(
                leading: const Icon(Icons.switch_access_shortcut),
                title: Text(AppLocale.t(widget.currentLocale, 'account_navigation_entrypoint')),
                subtitle: Text(
                  _accountNavPlacement == 'header'
                      ? 'Top Header Avatar Circle (opens floating sheet)'
                      : 'Bottom Navigation Bar Tab (opens dedicated page)',
                  style: const TextStyle(fontSize: 12),
                ),
                trailing: DropdownButton<String>(
                  value: _accountNavPlacement,
                  underline: const SizedBox(),
                  items: [
                    DropdownMenuItem(value: 'header', child: Text(AppLocale.t(widget.currentLocale, 'header_circle'))),
                    DropdownMenuItem(value: 'bottom_bar', child: Text(AppLocale.t(widget.currentLocale, 'bottom_bar_tab'))),
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
  final List<dynamic>? tabs;
  final String currentLocale;
  final String? configuredListMode;
  final int? configuredPerPage;
  final List<int>? configuredPerPageOptions;

  const ResourceRecordList({
    super.key,
    required this.api,
    required this.slug,
    this.relations,
    this.filters,
    this.tabs,
    this.currentLocale = 'en',
    this.configuredListMode,
    this.configuredPerPage,
    this.configuredPerPageOptions,
  });

  @override
  State<ResourceRecordList> createState() => _ResourceRecordListState();
}

class _ResourceRecordListState extends State<ResourceRecordList> {
  List<dynamic> allRecords = [];
  bool isLoading = true;
  String search = '';
  final Map<String, Set<String>> activeFilters = {};
  String? _activeTabKey;
  bool _showRelations = true;
  String? _sortColumn;
  bool _sortAscending = true;
  final Map<String, double> _customColumnWidths = {};
  final ScrollController _horizontalScrollController = ScrollController();
  final ScrollController _verticalScrollController = ScrollController();
  Set<String> selectedRecordIds = <String>{};

  late String _currentListMode;
  late int _perPage;
  late List<int> _perPageOptions;
  int _currentPage = 1;
  int _totalRecords = 0;
  bool _isLoadingMore = false;
  bool _hasMoreRecords = true;

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
        title: Text(AppLocale.t(widget.currentLocale, 'delete_records_title')),
        content: Text(AppLocale.t(widget.currentLocale, 'delete_records_confirm')),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(AppLocale.t(widget.currentLocale, 'cancel')),
          ),
          TextButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(AppLocale.t(widget.currentLocale, 'delete_selected'), style: const TextStyle(color: Colors.redAccent, fontWeight: FontWeight.bold)),
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
              content: Text("$count ${AppLocale.t(widget.currentLocale, 'records_deleted')}"),
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
    _verticalScrollController.removeListener(_handleScroll);
    _horizontalScrollController.dispose();
    _verticalScrollController.dispose();
    super.dispose();
  }

  int get activeFilterCount {
    int count = 0;
    for (final s in activeFilters.values) {
      count += s.length;
    }
    if (_sortColumn != null) {
      count += 1;
    }
    return count;
  }

  void _clearAllFilters() {
    setState(() {
      activeFilters.clear();
      _sortColumn = null;
      _sortAscending = true;
      selectedRecordIds.clear();
    });
    loadRecords(resetPage: true);
  }

  void _toggleSort(String col) {
    setState(() {
      if (_sortColumn == col) {
        if (_sortAscending) {
          _sortAscending = false;
        } else {
          _sortColumn = null;
          _sortAscending = true;
        }
      } else {
        _sortColumn = col;
        _sortAscending = true;
      }
    });
    loadRecords(resetPage: true);
  }

  void _removeFilter(String key, String val) {
    setState(() {
      activeFilters[key]?.remove(val);
      if (activeFilters[key]?.isEmpty ?? false) {
        activeFilters.remove(key);
      }
    });
    loadRecords(resetPage: true);
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

  List<Map<String, dynamic>> _getTabs() {
    if (widget.tabs != null && widget.tabs!.isNotEmpty) {
      return widget.tabs!.map((e) => Map<String, dynamic>.from(e as Map)).toList();
    }
    final hasStatusOrActive = (widget.filters?.any((f) {
      final name = (f['name'] ?? f['column'] ?? '').toString().toLowerCase();
      return name == 'status' || name == 'active' || name == 'is_active';
    }) ?? false) || (allRecords.isNotEmpty && (
      allRecords.first.containsKey('status') ||
      allRecords.first.containsKey('Status') ||
      allRecords.first.containsKey('active') ||
      allRecords.first.containsKey('Active') ||
      allRecords.first.containsKey('is_active')
    ));

    if (hasStatusOrActive) {
      return [
        {'key': 'all', 'label': 'All'},
        {'key': 'active', 'label': 'Active', 'query': {'status': 'active'}},
        {'key': 'inactive', 'label': 'Inactive', 'query': {'status': 'inactive'}},
      ];
    }
    return [];
  }

  int _getTabCount(Map<String, dynamic> tab) {
    final key = tab['key'];
    if (key == 'all') {
      if (search.isEmpty) return allRecords.length;
      return allRecords.where((r) => _recordMatchesSearch(r, search)).length;
    }
    final query = tab['query'] as Map<String, dynamic>?;
    if (query == null || query.isEmpty) return 0;

    return allRecords.where((r) {
      if (search.isNotEmpty && !_recordMatchesSearch(r, search)) {
        return false;
      }
      for (final qe in query.entries) {
        final expectedVal = qe.value.toString().toLowerCase();
        final recVal = _getRecordValue(r, qe.key)?.toLowerCase() ?? '';
        if (expectedVal == 'active' || expectedVal == 'true') {
          if (recVal != 'active' && recVal != 'true' && recVal != '1') return false;
        } else if (expectedVal == 'inactive' || expectedVal == 'false') {
          if (recVal != 'inactive' && recVal != 'false' && recVal != '0') return false;
        } else if (recVal != expectedVal) {
          return false;
        }
      }
      return true;
    }).length;
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
      if (_activeTabKey != null && _activeTabKey != 'all') {
        final tabs = _getTabs();
        final currentTab = tabs.firstWhere(
          (t) => t['key'] == _activeTabKey,
          orElse: () => <String, dynamic>{},
        );
        final query = currentTab['query'] as Map<String, dynamic>?;
        if (query != null && query.isNotEmpty) {
          for (final qe in query.entries) {
            final expectedVal = qe.value.toString().toLowerCase();
            final recVal = _getRecordValue(r, qe.key)?.toLowerCase() ?? '';
            if (expectedVal == 'active' || expectedVal == 'true') {
              if (recVal != 'active' && recVal != 'true' && recVal != '1') return false;
            } else if (expectedVal == 'inactive' || expectedVal == 'false') {
              if (recVal != 'inactive' && recVal != 'false' && recVal != '0') return false;
            } else if (recVal != expectedVal) {
              return false;
            }
          }
        }
      }
      return true;
    }).toList();
  }

  List<dynamic> get records {
    final list = List<dynamic>.from(filteredRecords);
    if (_sortColumn != null) {
      list.sort((a, b) {
        dynamic valA;
        dynamic valB;
        if (_sortColumn == 'id') {
          valA = _getRecordId(a);
          valB = _getRecordId(b);
        } else {
          valA = _getRecordValue(a, _sortColumn!);
          valB = _getRecordValue(b, _sortColumn!);
        }
        if (valA == null && valB == null) return 0;
        if (valA == null) return _sortAscending ? 1 : -1;
        if (valB == null) return _sortAscending ? -1 : 1;

        final strA = valA.toString().trim();
        final strB = valB.toString().trim();

        final numA = num.tryParse(strA);
        final numB = num.tryParse(strB);
        int cmp;
        if (numA != null && numB != null) {
          cmp = numA.compareTo(numB);
        } else {
          final dateA = DateTime.tryParse(strA);
          final dateB = DateTime.tryParse(strB);
          if (dateA != null && dateB != null && strA.length >= 8 && strB.length >= 8) {
            cmp = dateA.compareTo(dateB);
          } else {
            cmp = strA.toLowerCase().compareTo(strB.toLowerCase());
          }
        }
        return _sortAscending ? cmp : -cmp;
      });
    }
    return list;
  }

  void _showFilterModal(BuildContext context) {
    final availableFilters = _getAvailableFilters();
    final columns = _extractColumns();
    if (availableFilters.isEmpty && columns.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('No filterable or sortable attributes found for this resource.')),
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
                      Text(
                        AppLocale.t(widget.currentLocale, 'filters'),
                        style: const TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                      ),
                    ],
                  ),
                  Row(
                    children: [
                      if (activeFilterCount > 0)
                        TextButton.icon(
                          icon: const Icon(Icons.refresh, size: 16),
                          label: Text(AppLocale.t(widget.currentLocale, 'clear_all'), style: const TextStyle(fontSize: 13)),
                          onPressed: () {
                            _clearAllFilters();
                            setModalState(() {});
                          },
                        ),
                      if (isWideScreen)
                        IconButton(
                          icon: const Icon(Icons.close, size: 20),
                          onPressed: () {
                    Navigator.pop(ctx);
                    loadRecords(resetPage: true);
                  },
                        ),
                    ],
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            Flexible(
              child: ListView(
                shrinkWrap: true,
                padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                children: [
                  // 1. Sorting Section (Card view sorting inside filter modal)
                  Card(
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
                              Row(
                                children: [
                                  Icon(Icons.sort_rounded, size: 16, color: theme.colorScheme.primary),
                                  const SizedBox(width: 6),
                                  Text(
                                    AppLocale.t(widget.currentLocale, 'sort_by').toUpperCase(),
                                    style: TextStyle(
                                      fontSize: 12,
                                      fontWeight: FontWeight.bold,
                                      letterSpacing: 0.8,
                                      color: theme.colorScheme.primary,
                                    ),
                                  ),
                                ],
                              ),
                              if (_sortColumn != null)
                                TextButton(
                                  style: TextButton.styleFrom(
                                    visualDensity: VisualDensity.compact,
                                    padding: EdgeInsets.zero,
                                  ),
                                  child: Text(AppLocale.t(widget.currentLocale, 'clear_sort'), style: const TextStyle(fontSize: 11)),
                                  onPressed: () {
                                    setState(() {
                                      _sortColumn = null;
                                      _sortAscending = true;
                                    });
                                    setModalState(() {});
                                  },
                                ),
                            ],
                          ),
                          const SizedBox(height: 10),
                          Wrap(
                            spacing: 6,
                            runSpacing: 6,
                            children: [
                              ChoiceChip(
                                visualDensity: VisualDensity.compact,
                                label: const Text('ID', style: TextStyle(fontSize: 12)),
                                selected: _sortColumn == 'id',
                                onSelected: (sel) {
                                  setState(() {
                                    _sortColumn = sel ? 'id' : null;
                                  });
                                  setModalState(() {});
                                },
                              ),
                              ...columns.map((col) {
                                final isSel = _sortColumn == col;
                                final label = AppLocale.t(widget.currentLocale, col);
                                return ChoiceChip(
                                  visualDensity: VisualDensity.compact,
                                  label: Text(label, style: const TextStyle(fontSize: 12)),
                                  selected: isSel,
                                  onSelected: (sel) {
                                    setState(() {
                                      _sortColumn = sel ? col : null;
                                    });
                                    setModalState(() {});
                                  },
                                );
                              }),
                              ChoiceChip(
                                visualDensity: VisualDensity.compact,
                                label: Text(AppLocale.t(widget.currentLocale, 'status'), style: const TextStyle(fontSize: 12)),
                                selected: _sortColumn == 'status',
                                onSelected: (sel) {
                                  setState(() {
                                    _sortColumn = sel ? 'status' : null;
                                  });
                                  setModalState(() {});
                                },
                              ),
                            ],
                          ),
                          if (_sortColumn != null) ...[
                            const SizedBox(height: 10),
                            Row(
                              children: [
                                Text(
                                  '${AppLocale.t(widget.currentLocale, "direction")}:',
                                  style: TextStyle(
                                    fontSize: 12,
                                    fontWeight: FontWeight.w600,
                                    color: isDark ? Colors.white70 : Colors.black87,
                                  ),
                                ),
                                const SizedBox(width: 8),
                                ChoiceChip(
                                  visualDensity: VisualDensity.compact,
                                  avatar: Icon(_sortAscending ? Icons.arrow_upward_rounded : Icons.arrow_downward_rounded, size: 14),
                                  label: Text(
                                    _sortAscending ? AppLocale.t(widget.currentLocale, 'ascending') : AppLocale.t(widget.currentLocale, 'descending'),
                                    style: const TextStyle(fontSize: 11),
                                  ),
                                  selected: true,
                                  onSelected: (_) {
                                    setState(() {
                                      _sortAscending = !_sortAscending;
                                    });
                                    setModalState(() {});
                                  },
                                ),
                              ],
                            ),
                          ],
                        ],
                      ),
                    ),
                  ),

                  // 2. Filter Cards
                  for (final f in availableFilters) ...[
                    Builder(builder: (bctx) {
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
                    }),
                  ],
                ],
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
                  onPressed: () {
                    Navigator.pop(ctx);
                    loadRecords(resetPage: true);
                  },
                  child: Text(
                    'Apply & Show ${records.length} Results',
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

  List<Map<String, dynamic>> _getRelations() {
    if (widget.relations != null && widget.relations!.isNotEmpty) {
      return widget.relations!.map((e) => Map<String, dynamic>.from(e as Map)).toList();
    }
    if (widget.slug == 'partners') {
      return [
        {'relationship': 'admins', 'title': 'Station Admins', 'icon': 'heroicon-users'},
        {'relationship': 'branches', 'title': 'Station Branches', 'icon': 'heroicon-building-office'},
      ];
    }
    if (widget.slug == 'customers') {
      return [
        {'relationship': 'admins', 'title': 'Fleet Admins', 'icon': 'heroicon-users'},
        {'relationship': 'branches', 'title': 'Fleet Branches', 'icon': 'heroicon-building-office'},
        {'relationship': 'vehicles', 'title': 'Fleet Vehicles', 'icon': 'heroicon-truck'},
      ];
    }
    if (widget.slug == 'nfc-chip-batches') {
      return [
        {'relationship': 'chips', 'title': 'Batch NFC Chips', 'icon': 'heroicon-cpu-chip'},
      ];
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

  @override
  void initState() {
    super.initState();
    _initPaginationConfig();
    _verticalScrollController.addListener(_handleScroll);
    loadRecords(resetPage: true);
  }

  void _initPaginationConfig() {
    Map<String, dynamic>? resMeta;
    final schema = widget.api.cachedSchema;
    if (schema != null && schema['resources'] is List) {
      for (final r in schema['resources']) {
        if (r is Map<String, dynamic> && r['slug'] == widget.slug) {
          resMeta = r;
          break;
        }
      }
    }
    _currentListMode = widget.configuredListMode ??
        resMeta?['list_mode']?.toString() ??
        schema?['default_list_mode']?.toString() ??
        'pagination';

    _perPage = widget.configuredPerPage ??
        (resMeta?['per_page'] as num?)?.toInt() ??
        (schema?['default_per_page'] as num?)?.toInt() ??
        15;

    if (widget.configuredPerPageOptions != null && widget.configuredPerPageOptions!.isNotEmpty) {
      _perPageOptions = List<int>.from(widget.configuredPerPageOptions!);
    } else if (resMeta?['per_page_options'] is List) {
      _perPageOptions = (resMeta!['per_page_options'] as List).map((e) => (e as num).toInt()).toList();
    } else if (schema?['default_per_page_options'] is List) {
      _perPageOptions = (schema!['default_per_page_options'] as List).map((e) => (e as num).toInt()).toList();
    } else {
      _perPageOptions = [10, 15, 25, 50, 100];
    }
    if (!_perPageOptions.contains(_perPage)) {
      _perPageOptions.insert(0, _perPage);
      _perPageOptions.sort();
    }
  }

  void _handleScroll() {
    if (_currentListMode != 'infinite_scroll') return;
    if (_isLoadingMore || isLoading || !_hasMoreRecords) return;
    if (!_verticalScrollController.hasClients) return;
    final max = _verticalScrollController.position.maxScrollExtent;
    final cur = _verticalScrollController.position.pixels;
    if (max - cur <= 250) {
      _loadMoreRecords();
    }
  }

  @override
  void didUpdateWidget(covariant ResourceRecordList oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.slug != widget.slug) {
      _activeTabKey = null;
      _initPaginationConfig();
      loadRecords(resetPage: true);
    }
  }

  Future<void> loadRecords({bool resetPage = true}) async {
    if (resetPage) {
      _currentPage = 1;
      _hasMoreRecords = true;
    }
    setState(() {
      isLoading = true;
      selectedRecordIds.clear();
    });
    try {
      final filterParams = <String, dynamic>{};
      for (final entry in activeFilters.entries) {
        if (entry.value.isNotEmpty) {
          filterParams[entry.key] = entry.value.toList();
        }
      }
      if (_activeTabKey != null && _activeTabKey != 'all') {
        final tabs = _getTabs();
        final curTab = tabs.firstWhere((t) => t['key'] == _activeTabKey, orElse: () => <String, dynamic>{});
        final q = curTab['query'] as Map<String, dynamic>?;
        if (q != null) {
          q.forEach((k, v) => filterParams[k] = v);
        }
      }
      final res = await widget.api.fetchResourceRecords(
        widget.slug,
        page: _currentPage,
        perPage: _perPage,
        search: search,
        filters: filterParams,
        sort: _sortColumn,
        direction: _sortAscending ? 'asc' : 'desc',
      );
      final data = (res['data'] as List<dynamic>?) ?? [];
      final total = (res['total'] as num?)?.toInt() ?? data.length;
      setState(() {
        allRecords = List<dynamic>.from(data);
        _totalRecords = total;
        _hasMoreRecords = allRecords.length < _totalRecords;
        isLoading = false;
      });
    } catch (e) {
      setState(() => isLoading = false);
    }
  }

  Future<void> _loadMoreRecords() async {
    if (_isLoadingMore || isLoading || !_hasMoreRecords) return;
    setState(() {
      _isLoadingMore = true;
    });
    try {
      final nextPage = _currentPage + 1;
      final filterParams = <String, dynamic>{};
      for (final entry in activeFilters.entries) {
        if (entry.value.isNotEmpty) {
          filterParams[entry.key] = entry.value.toList();
        }
      }
      if (_activeTabKey != null && _activeTabKey != 'all') {
        final tabs = _getTabs();
        final curTab = tabs.firstWhere((t) => t['key'] == _activeTabKey, orElse: () => <String, dynamic>{});
        final q = curTab['query'] as Map<String, dynamic>?;
        if (q != null) {
          q.forEach((k, v) => filterParams[k] = v);
        }
      }
      final res = await widget.api.fetchResourceRecords(
        widget.slug,
        page: nextPage,
        perPage: _perPage,
        search: search,
        filters: filterParams,
        sort: _sortColumn,
        direction: _sortAscending ? 'asc' : 'desc',
      );
      final data = (res['data'] as List<dynamic>?) ?? [];
      final total = (res['total'] as num?)?.toInt() ?? _totalRecords;
      setState(() {
        _currentPage = nextPage;
        allRecords.addAll(data);
        _totalRecords = total;
        _hasMoreRecords = data.isNotEmpty && allRecords.length < _totalRecords;
        _isLoadingMore = false;
      });
    } catch (e) {
      setState(() => _isLoadingMore = false);
    }
  }

  int get _totalPages {
    if (_totalRecords <= 0 || _perPage <= 0) return 1;
    return (_totalRecords + _perPage - 1) ~/ _perPage;
  }

  void _goToPage(int page) {
    if (page < 1 || page > _totalPages || page == _currentPage) return;
    setState(() {
      _currentPage = page;
    });
    loadRecords(resetPage: false);
  }

  void _changePerPage(int newPerPage) {
    if (newPerPage == _perPage) return;
    setState(() {
      _perPage = newPerPage;
    });
    loadRecords(resetPage: true);
  }

  void _toggleListMode(String mode) {
    if (_currentListMode == mode) return;
    setState(() {
      _currentListMode = mode;
    });
    loadRecords(resetPage: true);
  }

  Widget _buildFieldItem(String label, String value, {IconData? icon, Color? valueColor}) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          label.toUpperCase(),
          style: TextStyle(
            fontSize: 9,
            fontWeight: FontWeight.w600,
            color: Colors.grey.shade400,
            letterSpacing: 0.4,
          ),
        ),
        const SizedBox(height: 2),
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (icon != null) ...[
              Icon(icon, size: 13, color: valueColor ?? Colors.grey.shade300),
              const SizedBox(width: 4),
            ],
            Flexible(
              child: Text(
                value,
                style: TextStyle(
                  fontSize: 12,
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
    if (widget.slug == 'partners') headerIcon = Icons.local_gas_station_outlined;
    if (widget.slug == 'customers') headerIcon = Icons.business_outlined;
    if (widget.slug == 'partner-branches' || widget.slug == 'customer-branches') headerIcon = Icons.storefront_outlined;
    if (widget.slug == 'vehicles') headerIcon = Icons.local_shipping_outlined;
    if (widget.slug.contains('nfc')) headerIcon = Icons.nfc_outlined;

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
        borderRadius: BorderRadius.circular(14),
        side: BorderSide(
          color: isSelected ? Colors.amber.shade400 : (isDark ? Colors.white.withValues(alpha: 0.08) : Colors.black.withValues(alpha: 0.08)),
          width: isSelected ? 1.5 : 1,
        ),
      ),
      color: isSelected
          ? (isDark ? Colors.amber.withValues(alpha: 0.08) : Colors.amber.withValues(alpha: 0.05))
          : theme.colorScheme.surface,
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Header: Icon + Title + Status/Badge
            Row(
              children: [
                SizedBox(
                  width: 22,
                  height: 22,
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
                  radius: 15,
                  backgroundColor: Theme.of(context).colorScheme.primaryContainer,
                  child: Icon(headerIcon, size: 16, color: Theme.of(context).colorScheme.primary),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        name,
                        style: const TextStyle(fontSize: 13, fontWeight: FontWeight.bold),
                        overflow: TextOverflow.ellipsis,
                      ),
                      Text(
                        'ID: #$id',
                        style: TextStyle(fontSize: 10, color: Colors.grey.shade400),
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
                        Text(AppLocale.t(widget.currentLocale, status).toUpperCase(), style: TextStyle(fontSize: 10, fontWeight: FontWeight.bold, color: statusColor)),
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
            ] else if (widget.slug == 'partners') ...[
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Company Name', '${item['name'] ?? item['Name'] ?? 'N/A'}', icon: Icons.local_gas_station_outlined)),
                  Expanded(child: _buildFieldItem('Arabic Name', '${item['name_ar'] ?? item['NameAr'] ?? 'N/A'}', icon: Icons.translate)),
                ],
              ),
              const SizedBox(height: 10),
              Row(
                children: [
                  Expanded(child: _buildFieldItem('CR Number', '${item['cr_number'] ?? item['CrNumber'] ?? 'N/A'}', icon: Icons.badge_outlined)),
                  Expanded(child: _buildFieldItem('Phone / Contact', '${item['phone'] ?? item['Phone'] ?? item['email'] ?? 'N/A'}', icon: Icons.phone_outlined)),
                ],
              ),
            ] else if (widget.slug == 'customers') ...[
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Fleet Code', '${item['code'] ?? item['Code'] ?? 'N/A'}', icon: Icons.tag)),
                  Expanded(child: _buildFieldItem('Fleet Name', '${item['name'] ?? item['Name'] ?? 'N/A'}', icon: Icons.business)),
                ],
              ),
              const SizedBox(height: 10),
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Phone', '${item['phone'] ?? item['Phone'] ?? 'N/A'}', icon: Icons.phone)),
                  Expanded(child: _buildFieldItem('Credit Limit', 'SAR ${item['credit_limit'] ?? item['CreditLimit'] ?? '0.00'}', icon: Icons.account_balance_wallet_outlined, valueColor: const Color(0xFF10B981))),
                ],
              ),
            ] else if (widget.slug == 'customer-branches' || widget.slug == 'partner-branches') ...[
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Branch Name', '${item['name'] ?? item['Name'] ?? 'N/A'}', icon: Icons.storefront)),
                  Expanded(child: _buildFieldItem('City / Location', '${item['city'] ?? item['City'] ?? 'Saudi Arabia'}', icon: Icons.location_on_outlined)),
                ],
              ),
            ] else if (widget.slug == 'vehicles') ...[
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Plate Number', '${item['plate_number'] ?? item['PlateNumber'] ?? 'N/A'}', icon: Icons.pin)),
                  Expanded(child: _buildFieldItem('Odometer', '${item['odometer'] ?? item['Odometer'] ?? '0'} km', icon: Icons.speed)),
                ],
              ),
            ] else if (widget.slug == 'nfc-chips') ...[
              Row(
                children: [
                  Expanded(child: _buildFieldItem('Chip Serial', '${item['chip_serial'] ?? item['ChipSerial'] ?? 'N/A'}', icon: Icons.nfc)),
                  Expanded(child: _buildFieldItem('Fleet', '${item['customer_name'] ?? item['CustomerName'] ?? 'Unassigned'}', icon: Icons.business)),
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

            // Nested Relation Managers (Mobile-friendly link to open related resources)
            if (_getRelations().isNotEmpty && _showRelations) ...[
              const SizedBox(height: 12),
              const Divider(height: 1, thickness: 0.5),
              const SizedBox(height: 10),
              Row(
                children: [
                  Icon(Icons.hub_outlined, size: 14, color: Colors.amber.shade400),
                  const SizedBox(width: 6),
                  Text(
                    AppLocale.t(widget.currentLocale, 'sub_entities_relations').toUpperCase(),
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
                runSpacing: 8,
                children: _getRelations().map((rel) {
                  final relTitle = (rel['title'] ?? rel['relationship'] ?? 'Related').toString();
                  final relSlug = (rel['relationship'] ?? '').toString();
                  final iconName = (rel['icon'] ?? '').toString();
                  return ActionChip(
                    avatar: Icon(_iconForRelation(iconName), size: 15, color: Colors.amber.shade400),
                    label: Text(
                      AppLocale.t(widget.currentLocale, relTitle),
                      style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold),
                    ),
                    backgroundColor: Colors.amber.withValues(alpha: 0.1),
                    side: BorderSide(color: Colors.amber.withValues(alpha: 0.3)),
                    padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
                    visualDensity: VisualDensity.compact,
                    onPressed: () {
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
                            currentLocale: widget.currentLocale,
                          ),
                        ),
                      );
                    },
                  );
                }).toList(),
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
                  label: Text(AppLocale.t(widget.currentLocale, 'edit'), style: const TextStyle(fontSize: 12)),
                  onPressed: () => showEditSheet(context, id, item),
                ),
                const SizedBox(width: 8),
                TextButton.icon(
                  icon: const Icon(Icons.delete_outline, size: 15, color: Colors.redAccent),
                  label: Text(AppLocale.t(widget.currentLocale, 'delete'), style: const TextStyle(fontSize: 12, color: Colors.redAccent)),
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
    if (_customColumnWidths.containsKey(colKey)) {
      return _customColumnWidths[colKey]!;
    }
    final k = colKey.toLowerCase();
    if (k == 'id') return 70.0;
    if (k == 'status') return 130.0;
    if (k == 'name' || k == 'title') return 220.0;
    if (k == 'email' || k == 'username') return 200.0;
    if (k.contains('address') || k.contains('description') || k.contains('notes')) return 240.0;
    if (k == 'phone' || k.contains('phone')) return 150.0;
    if (k.contains('code') || k.contains('serial') || k.contains('plate') || k.contains('cr_')) return 150.0;
    if (k == 'city' || k == 'type' || k == 'role' || k == 'category') return 140.0;
    if (k.contains('amount') || k.contains('balance') || k.contains('price') || k.contains('rate')) return 140.0;
    return 150.0;
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
          currentLocale: widget.currentLocale,
        ),
      ),
    );
  }

  Widget _buildFloatingActionButtons(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final bool showPaginationBtn = _totalRecords > 0 || records.isNotEmpty;

    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.end,
      children: [
        if (showPaginationBtn) ...[
          FloatingActionButton.small(
            heroTag: 'pagination_fab_${widget.slug}',
            tooltip: AppLocale.t(widget.currentLocale, 'pagination'),
            backgroundColor: isDark ? const Color(0xFF1E293B) : Colors.white,
            foregroundColor: Colors.amber.shade700,
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(28),
              side: BorderSide(color: Colors.amber.shade400, width: 1.5),
            ),
            elevation: 4,
            onPressed: () => _showPaginationModal(context),
            child: Stack(
              clipBehavior: Clip.none,
              alignment: Alignment.center,
              children: [
                Icon(
                  _currentListMode == 'pagination' ? Icons.auto_stories_outlined : Icons.all_inclusive_rounded,
                  size: 20,
                  color: Colors.amber.shade700,
                ),
                if (_currentListMode == 'pagination' && _totalPages > 1)
                  Positioned(
                    right: -6,
                    top: -6,
                    child: Container(
                      padding: const EdgeInsets.all(3),
                      decoration: BoxDecoration(
                        color: Colors.amber.shade700,
                        shape: BoxShape.circle,
                      ),
                      constraints: const BoxConstraints(minWidth: 16, minHeight: 16),
                      child: Text(
                        '$_currentPage',
                        style: const TextStyle(
                          color: Colors.white,
                          fontSize: 9,
                          fontWeight: FontWeight.bold,
                        ),
                        textAlign: TextAlign.center,
                      ),
                    ),
                  ),
              ],
            ),
          ),
          const SizedBox(height: 12),
        ],
        FloatingActionButton(
          heroTag: 'create_fab_${widget.slug}',
          onPressed: () => showCreateSheet(context),
          child: const Icon(Icons.add),
        ),
      ],
    );
  }

  void _showPaginationModal(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final screenWidth = MediaQuery.of(context).size.width;
    final isWideScreen = screenWidth >= 720;

    Widget buildContent(BuildContext ctx, StateSetter setModalState) {
      final int startRecord = _totalRecords == 0 ? 0 : (_currentPage - 1) * _perPage + 1;
      int endRecord = _currentPage * _perPage;
      if (endRecord > _totalRecords) endRecord = _totalRecords;
      final infoText = '${AppLocale.t(widget.currentLocale, "showing_records")} $startRecord ${AppLocale.t(widget.currentLocale, "to")} $endRecord ${AppLocale.t(widget.currentLocale, "of")} $_totalRecords';

      return Container(
        constraints: BoxConstraints(
          maxWidth: isWideScreen ? 500 : double.infinity,
          maxHeight: MediaQuery.of(context).size.height * 0.82,
        ),
        decoration: BoxDecoration(
          color: isDark ? const Color(0xFF1E293B) : Colors.white,
          borderRadius: isWideScreen
              ? BorderRadius.circular(16)
              : const BorderRadius.vertical(top: Radius.circular(20)),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.2),
              blurRadius: 20,
              offset: const Offset(0, 4),
            ),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // Header
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 16, 16, 12),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Row(
                    children: [
                      Container(
                        padding: const EdgeInsets.all(8),
                        decoration: BoxDecoration(
                          color: Colors.amber.withValues(alpha: 0.15),
                          borderRadius: BorderRadius.circular(10),
                        ),
                        child: Icon(
                          _currentListMode == 'pagination' ? Icons.auto_stories_rounded : Icons.all_inclusive_rounded,
                          color: Colors.amber,
                          size: 20,
                        ),
                      ),
                      const SizedBox(width: 12),
                      Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            AppLocale.t(widget.currentLocale, 'pagination'),
                            style: const TextStyle(fontSize: 17, fontWeight: FontWeight.bold),
                          ),
                          const SizedBox(height: 2),
                          Text(
                            infoText,
                            style: TextStyle(
                              fontSize: 12,
                              color: isDark ? Colors.grey.shade400 : Colors.grey.shade600,
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                  IconButton(
                    icon: const Icon(Icons.close_rounded, size: 20),
                    onPressed: () => Navigator.pop(ctx),
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            Flexible(
              child: ListView(
                shrinkWrap: true,
                padding: const EdgeInsets.all(16),
                children: [
                  // Mode Switcher Card
                  Card(
                    margin: const EdgeInsets.only(bottom: 14),
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
                          Text(
                            'DISPLAY MODE',
                            style: TextStyle(
                              fontSize: 11,
                              fontWeight: FontWeight.bold,
                              letterSpacing: 0.8,
                              color: theme.colorScheme.primary,
                            ),
                          ),
                          const SizedBox(height: 10),
                          Row(
                            children: [
                              Expanded(
                                child: ChoiceChip(
                                  label: Row(
                                    mainAxisAlignment: MainAxisAlignment.center,
                                    children: [
                                      const Icon(Icons.auto_stories_outlined, size: 16),
                                      const SizedBox(width: 6),
                                      Text(AppLocale.t(widget.currentLocale, 'pagination'), style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600)),
                                    ],
                                  ),
                                  selected: _currentListMode == 'pagination',
                                  onSelected: (selected) {
                                    if (selected && _currentListMode != 'pagination') {
                                      _toggleListMode('pagination');
                                      setModalState(() {});
                                    }
                                  },
                                ),
                              ),
                              const SizedBox(width: 10),
                              Expanded(
                                child: ChoiceChip(
                                  label: Row(
                                    mainAxisAlignment: MainAxisAlignment.center,
                                    children: [
                                      const Icon(Icons.all_inclusive_rounded, size: 16),
                                      const SizedBox(width: 6),
                                      Text(AppLocale.t(widget.currentLocale, 'infinite_scroll'), style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600)),
                                    ],
                                  ),
                                  selected: _currentListMode == 'infinite_scroll',
                                  onSelected: (selected) {
                                    if (selected && _currentListMode != 'infinite_scroll') {
                                      _toggleListMode('infinite_scroll');
                                      setModalState(() {});
                                    }
                                  },
                                ),
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                  ),

                  // Pagination controls
                  if (_currentListMode == 'pagination') ...[
                    Card(
                      margin: const EdgeInsets.only(bottom: 14),
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
                            Text(
                              'PAGE NAVIGATION',
                              style: TextStyle(
                                fontSize: 11,
                                fontWeight: FontWeight.bold,
                                letterSpacing: 0.8,
                                color: theme.colorScheme.primary,
                              ),
                            ),
                            const SizedBox(height: 12),
                            Row(
                              mainAxisAlignment: MainAxisAlignment.spaceEvenly,
                              children: [
                                IconButton.filledTonal(
                                  icon: const Icon(Icons.first_page_rounded, size: 20),
                                  tooltip: AppLocale.t(widget.currentLocale, 'first_page'),
                                  onPressed: _currentPage > 1
                                      ? () {
                                          _goToPage(1);
                                          setModalState(() {});
                                        }
                                      : null,
                                ),
                                IconButton.filledTonal(
                                  icon: const Icon(Icons.chevron_left_rounded, size: 20),
                                  tooltip: AppLocale.t(widget.currentLocale, 'previous_page'),
                                  onPressed: _currentPage > 1
                                      ? () {
                                          _goToPage(_currentPage - 1);
                                          setModalState(() {});
                                        }
                                      : null,
                                ),
                                Container(
                                  padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 8),
                                  decoration: BoxDecoration(
                                    color: Colors.amber.withValues(alpha: 0.15),
                                    borderRadius: BorderRadius.circular(10),
                                    border: Border.all(color: Colors.amber.withValues(alpha: 0.3)),
                                  ),
                                  child: Text(
                                    '$_currentPage / $_totalPages',
                                    style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14, color: Colors.amber),
                                  ),
                                ),
                                IconButton.filledTonal(
                                  icon: const Icon(Icons.chevron_right_rounded, size: 20),
                                  tooltip: AppLocale.t(widget.currentLocale, 'next_page'),
                                  onPressed: _currentPage < _totalPages
                                      ? () {
                                          _goToPage(_currentPage + 1);
                                          setModalState(() {});
                                        }
                                      : null,
                                ),
                                IconButton.filledTonal(
                                  icon: const Icon(Icons.last_page_rounded, size: 20),
                                  tooltip: AppLocale.t(widget.currentLocale, 'last_page'),
                                  onPressed: _currentPage < _totalPages
                                      ? () {
                                          _goToPage(_totalPages);
                                          setModalState(() {});
                                        }
                                      : null,
                                ),
                              ],
                            ),
                            if (_totalPages > 1) ...[
                              const SizedBox(height: 14),
                              Text(
                                'JUMP TO PAGE',
                                style: TextStyle(
                                  fontSize: 10,
                                  fontWeight: FontWeight.w600,
                                  color: isDark ? Colors.grey.shade400 : Colors.grey.shade600,
                                ),
                              ),
                              const SizedBox(height: 8),
                              Wrap(
                                spacing: 6,
                                runSpacing: 6,
                                children: List.generate(_totalPages, (idx) {
                                  final pageNum = idx + 1;
                                  final isCurrent = pageNum == _currentPage;
                                  return ChoiceChip(
                                    visualDensity: VisualDensity.compact,
                                    label: Text('$pageNum', style: TextStyle(fontSize: 11, fontWeight: isCurrent ? FontWeight.bold : FontWeight.normal)),
                                    selected: isCurrent,
                                    onSelected: (selected) {
                                      if (selected && pageNum != _currentPage) {
                                        _goToPage(pageNum);
                                        setModalState(() {});
                                      }
                                    },
                                  );
                                }),
                              ),
                            ],
                          ],
                        ),
                      ),
                    ),
                  ],

                  // Infinite Scroll controls
                  if (_currentListMode == 'infinite_scroll') ...[
                    Card(
                      margin: const EdgeInsets.only(bottom: 14),
                      elevation: 0,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                        side: BorderSide(color: isDark ? Colors.white12 : Colors.black12),
                      ),
                      color: isDark ? const Color(0xFF0F172A) : const Color(0xFFF8FAFC),
                      child: Padding(
                        padding: const EdgeInsets.all(12),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            Text(
                              'INFINITE SCROLL STATUS',
                              style: TextStyle(
                                fontSize: 11,
                                fontWeight: FontWeight.bold,
                                letterSpacing: 0.8,
                                color: theme.colorScheme.primary,
                              ),
                            ),
                            const SizedBox(height: 10),
                            Text(
                              '${records.length} of $_totalRecords records loaded',
                              style: TextStyle(
                                fontSize: 13,
                                fontWeight: FontWeight.w600,
                                color: isDark ? Colors.grey.shade300 : Colors.grey.shade800,
                              ),
                            ),
                            const SizedBox(height: 12),
                            if (records.length < _totalRecords)
                              ElevatedButton.icon(
                                style: ElevatedButton.styleFrom(
                                  padding: const EdgeInsets.symmetric(vertical: 12),
                                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                                ),
                                icon: _isLoadingMore
                                    ? const SizedBox(
                                        width: 16,
                                        height: 16,
                                        child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                                      )
                                    : const Icon(Icons.arrow_downward_rounded, size: 18),
                                label: Text(
                                  _isLoadingMore
                                      ? AppLocale.t(widget.currentLocale, 'loading_more')
                                      : '${AppLocale.t(widget.currentLocale, 'load_more')} (+$_perPage)',
                                  style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 13),
                                ),
                                onPressed: _isLoadingMore
                                    ? null
                                    : () async {
                                        await _loadMoreRecords();
                                        setModalState(() {});
                                      },
                              )
                            else
                              Container(
                                padding: const EdgeInsets.all(10),
                                decoration: BoxDecoration(
                                  color: Colors.green.withValues(alpha: 0.12),
                                  borderRadius: BorderRadius.circular(8),
                                ),
                                child: Row(
                                  children: [
                                    const Icon(Icons.check_circle_outline_rounded, color: Colors.green, size: 18),
                                    const SizedBox(width: 8),
                                    Text(
                                      AppLocale.t(widget.currentLocale, 'all_records_loaded'),
                                      style: const TextStyle(color: Colors.green, fontWeight: FontWeight.w600, fontSize: 12),
                                    ),
                                  ],
                                ),
                              ),
                          ],
                        ),
                      ),
                    ),
                  ],

                  // Rows per page / Batch size
                  Card(
                    margin: EdgeInsets.zero,
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
                          Text(
                            _currentListMode == 'pagination' ? 'ROWS PER PAGE' : 'RECORDS PER BATCH',
                            style: TextStyle(
                              fontSize: 11,
                              fontWeight: FontWeight.bold,
                              letterSpacing: 0.8,
                              color: theme.colorScheme.primary,
                            ),
                          ),
                          const SizedBox(height: 10),
                          Wrap(
                            spacing: 8,
                            runSpacing: 8,
                            children: _perPageOptions.map((opt) {
                              final isSelected = opt == _perPage;
                              return ChoiceChip(
                                label: Text('$opt', style: TextStyle(fontWeight: isSelected ? FontWeight.bold : FontWeight.normal)),
                                selected: isSelected,
                                onSelected: (selected) {
                                  if (selected && opt != _perPage) {
                                    _changePerPage(opt);
                                    setModalState(() {});
                                  }
                                },
                              );
                            }).toList(),
                          ),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
            // Footer
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                border: Border(top: BorderSide(color: isDark ? Colors.white12 : Colors.black12)),
              ),
              child: SizedBox(
                width: double.infinity,
                child: ElevatedButton(
                  style: ElevatedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(vertical: 12),
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                  ),
                  onPressed: () => Navigator.pop(ctx),
                  child: const Text('Done', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 14)),
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
            child: buildContent(ctx, setModalState),
          ),
        ),
      );
    } else {
      showModalBottomSheet(
        context: context,
        isScrollControlled: true,
        backgroundColor: Colors.transparent,
        builder: (bottomSheetCtx) => StatefulBuilder(
          builder: (ctx, setModalState) => buildContent(ctx, setModalState),
        ),
      );
    }
  }

  Widget _buildPaginationBar(BuildContext context, {bool isMobile = false}) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final int startRecord = _totalRecords == 0 ? 0 : (_currentPage - 1) * _perPage + 1;
    int endRecord = _currentPage * _perPage;
    if (endRecord > _totalRecords) endRecord = _totalRecords;

    final infoText = '${AppLocale.t(widget.currentLocale, "showing_records")} $startRecord ${AppLocale.t(widget.currentLocale, "to")} $endRecord ${AppLocale.t(widget.currentLocale, "of")} $_totalRecords';

    if (isMobile) {
      return Container(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        decoration: BoxDecoration(
          color: isDark ? const Color(0xFF1E293B) : Colors.white,
          border: Border(
            top: BorderSide(
              color: isDark ? Colors.white12 : Colors.black12,
            ),
          ),
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Expanded(
                  child: Text(
                    infoText,
                    style: TextStyle(
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                      color: isDark ? Colors.grey.shade400 : Colors.grey.shade600,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                _buildModeToggleButton(isDark),
              ],
            ),
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                _buildPerPageDropdown(isDark),
                Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    IconButton(
                      icon: const Icon(Icons.chevron_left_rounded),
                      iconSize: 22,
                      padding: EdgeInsets.zero,
                      constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
                      tooltip: AppLocale.t(widget.currentLocale, 'previous_page'),
                      onPressed: _currentPage > 1 ? () => _goToPage(_currentPage - 1) : null,
                    ),
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                      decoration: BoxDecoration(
                        color: Colors.amber.withValues(alpha: 0.15),
                        borderRadius: BorderRadius.circular(6),
                      ),
                      child: Text(
                        '$_currentPage / $_totalPages',
                        style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12, color: Colors.amber),
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.chevron_right_rounded),
                      iconSize: 22,
                      padding: EdgeInsets.zero,
                      constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
                      tooltip: AppLocale.t(widget.currentLocale, 'next_page'),
                      onPressed: _currentPage < _totalPages ? () => _goToPage(_currentPage + 1) : null,
                    ),
                  ],
                ),
              ],
            ),
          ],
        ),
      );
    }

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
      decoration: BoxDecoration(
        color: isDark ? const Color(0xFF1E293B) : Colors.white,
        border: Border(
          top: BorderSide(
            color: isDark ? Colors.white12 : Colors.black12,
          ),
        ),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                infoText,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w500,
                  color: isDark ? Colors.grey.shade400 : Colors.grey.shade600,
                ),
              ),
              const SizedBox(width: 16),
              _buildPerPageDropdown(isDark),
              const SizedBox(width: 16),
              _buildModeToggleButton(isDark),
            ],
          ),
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              IconButton(
                icon: const Icon(Icons.first_page_rounded),
                iconSize: 20,
                tooltip: AppLocale.t(widget.currentLocale, 'first_page'),
                onPressed: _currentPage > 1 ? () => _goToPage(1) : null,
              ),
              IconButton(
                icon: const Icon(Icons.chevron_left_rounded),
                iconSize: 20,
                tooltip: AppLocale.t(widget.currentLocale, 'previous_page'),
                onPressed: _currentPage > 1 ? () => _goToPage(_currentPage - 1) : null,
              ),
              const SizedBox(width: 4),
              ..._buildPageNumberButtons(isDark),
              const SizedBox(width: 4),
              IconButton(
                icon: const Icon(Icons.chevron_right_rounded),
                iconSize: 20,
                tooltip: AppLocale.t(widget.currentLocale, 'next_page'),
                onPressed: _currentPage < _totalPages ? () => _goToPage(_currentPage + 1) : null,
              ),
              IconButton(
                icon: const Icon(Icons.last_page_rounded),
                iconSize: 20,
                tooltip: AppLocale.t(widget.currentLocale, 'last_page'),
                onPressed: _currentPage < _totalPages ? () => _goToPage(_totalPages) : null,
              ),
            ],
          ),
        ],
      ),
    );
  }

  List<Widget> _buildPageNumberButtons(bool isDark) {
    final buttons = <Widget>[];
    final total = _totalPages;
    if (total <= 1) return buttons;

    final pagesToShow = <int>{};
    pagesToShow.add(1);
    pagesToShow.add(total);
    for (int p = _currentPage - 2; p <= _currentPage + 2; p++) {
      if (p >= 1 && p <= total) pagesToShow.add(p);
    }
    final sortedPages = pagesToShow.toList()..sort();

    int lastP = 0;
    for (final p in sortedPages) {
      if (lastP != 0 && p - lastP > 1) {
        buttons.add(
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4),
            child: Text('...', style: TextStyle(color: Colors.grey.shade500, fontSize: 12)),
          ),
        );
      }
      final isCurrent = p == _currentPage;
      buttons.add(
        InkWell(
          onTap: isCurrent ? null : () => _goToPage(p),
          borderRadius: BorderRadius.circular(6),
          child: Container(
            margin: const EdgeInsets.symmetric(horizontal: 2),
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            decoration: BoxDecoration(
              color: isCurrent ? Colors.amber : (isDark ? Colors.white10 : Colors.black.withValues(alpha: 0.05)),
              borderRadius: BorderRadius.circular(6),
              border: Border.all(
                color: isCurrent ? Colors.amber : Colors.transparent,
              ),
            ),
            child: Text(
              '$p',
              style: TextStyle(
                fontSize: 12,
                fontWeight: isCurrent ? FontWeight.bold : FontWeight.w500,
                color: isCurrent
                    ? Colors.black87
                    : (isDark ? Colors.grey.shade300 : Colors.grey.shade800),
              ),
            ),
          ),
        ),
      );
      lastP = p;
    }
    return buttons;
  }

  Widget _buildPerPageDropdown(bool isDark) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: isDark ? Colors.white10 : Colors.black.withValues(alpha: 0.04),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: isDark ? Colors.white12 : Colors.black12),
      ),
      child: DropdownButtonHideUnderline(
        child: DropdownButton<int>(
          value: _perPageOptions.contains(_perPage) ? _perPage : _perPageOptions.first,
          isDense: true,
          style: TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w600,
            color: isDark ? Colors.white : Colors.black87,
          ),
          dropdownColor: isDark ? const Color(0xFF1E293B) : Colors.white,
          items: _perPageOptions.map((opt) {
            return DropdownMenuItem<int>(
              value: opt,
              child: Text('$opt / ${AppLocale.t(widget.currentLocale, "page")}'),
            );
          }).toList(),
          onChanged: (val) {
            if (val != null) _changePerPage(val);
          },
        ),
      ),
    );
  }

  Widget _buildModeToggleButton(bool isDark) {
    final isPagination = _currentListMode == 'pagination';
    return Tooltip(
      message: isPagination
          ? '${AppLocale.t(widget.currentLocale, "pagination")} (Click for ${AppLocale.t(widget.currentLocale, "infinite_scroll")})'
          : '${AppLocale.t(widget.currentLocale, "infinite_scroll")} (Click for ${AppLocale.t(widget.currentLocale, "pagination")})',
      child: InkWell(
        onTap: () => _toggleListMode(isPagination ? 'infinite_scroll' : 'pagination'),
        borderRadius: BorderRadius.circular(8),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
          decoration: BoxDecoration(
            color: isDark ? Colors.white10 : Colors.black.withValues(alpha: 0.04),
            borderRadius: BorderRadius.circular(8),
            border: Border.all(color: Colors.amber.withValues(alpha: 0.5)),
          ),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                isPagination ? Icons.auto_stories_outlined : Icons.all_inclusive_rounded,
                size: 15,
                color: Colors.amber,
              ),
              const SizedBox(width: 5),
              Text(
                isPagination
                    ? AppLocale.t(widget.currentLocale, 'pagination')
                    : AppLocale.t(widget.currentLocale, 'infinite_scroll'),
                style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: Colors.amber),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildInfiniteScrollFooter(bool isDark) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
      decoration: BoxDecoration(
        color: isDark ? const Color(0xFF1E293B) : Colors.white,
        border: Border(
          top: BorderSide(color: isDark ? Colors.white12 : Colors.black12),
        ),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                '${AppLocale.t(widget.currentLocale, "showing_records")} ${allRecords.length} ${AppLocale.t(widget.currentLocale, "of")} $_totalRecords',
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: isDark ? Colors.grey.shade400 : Colors.grey.shade600,
                ),
              ),
              const SizedBox(width: 14),
              _buildModeToggleButton(isDark),
            ],
          ),
          if (_isLoadingMore)
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                const SizedBox(width: 14, height: 14, child: CircularProgressIndicator(strokeWidth: 2)),
                const SizedBox(width: 8),
                Text(
                  AppLocale.t(widget.currentLocale, 'loading_more'),
                  style: TextStyle(fontSize: 12, color: Colors.grey.shade500),
                ),
              ],
            )
          else if (_hasMoreRecords)
            OutlinedButton.icon(
              icon: const Icon(Icons.expand_more_rounded, size: 16),
              label: Text(AppLocale.t(widget.currentLocale, 'load_more'), style: const TextStyle(fontSize: 12)),
              style: OutlinedButton.styleFrom(
                visualDensity: VisualDensity.compact,
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              ),
              onPressed: _loadMoreRecords,
            )
          else
            Text(
              '${AppLocale.t(widget.currentLocale, "all_records_loaded")} ($_totalRecords)',
              style: TextStyle(fontSize: 11, fontStyle: FontStyle.italic, color: Colors.grey.shade500),
            ),
        ],
      ),
    );
  }

  Widget _buildRecordTable(BuildContext context) {
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final relations = _getRelations();
    final hasRelations = relations.isNotEmpty;
    final columns = _extractColumns();

    double totalColsWidth = 48.0 + _columnWidth('id') + _columnWidth('status') + 140.0 + 80.0; // Checkbox + ID + STATUS + ACTIONS + clearance
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
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  LayoutBuilder(
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
                                    SizedBox(
                                      width: _columnWidth('id'),
                                      child: Row(
                                        children: [
                                          Expanded(
                                            child: InkWell(
                                              onTap: () => _toggleSort('id'),
                                              child: Row(
                                                mainAxisSize: MainAxisSize.min,
                                                children: [
                                                  Text(
                                                    'ID',
                                                    style: TextStyle(
                                                      fontWeight: FontWeight.bold,
                                                      fontSize: 12,
                                                      letterSpacing: 0.5,
                                                      color: _sortColumn == 'id' ? Colors.amber : Colors.grey,
                                                    ),
                                                  ),
                                                  const SizedBox(width: 4),
                                                  Icon(
                                                    _sortColumn == 'id'
                                                        ? (_sortAscending ? Icons.arrow_upward_rounded : Icons.arrow_downward_rounded)
                                                        : Icons.unfold_more_rounded,
                                                    size: 14,
                                                    color: _sortColumn == 'id' ? Colors.amber : (isDark ? Colors.white24 : Colors.black26),
                                                  ),
                                                ],
                                              ),
                                            ),
                                          ),
                                          MouseRegion(
                                            cursor: SystemMouseCursors.resizeColumn,
                                            child: GestureDetector(
                                              behavior: HitTestBehavior.opaque,
                                              onHorizontalDragUpdate: (details) {
                                                setState(() {
                                                  final cur = _columnWidth('id');
                                                  _customColumnWidths['id'] = (cur + details.delta.dx).clamp(50.0, 300.0);
                                                });
                                              },
                                              child: Container(
                                                width: 14,
                                                height: 28,
                                                alignment: Alignment.center,
                                                color: Colors.transparent,
                                                child: Container(
                                                  width: 2,
                                                  height: 14,
                                                  decoration: BoxDecoration(
                                                    color: isDark ? Colors.white24 : Colors.black26,
                                                    borderRadius: BorderRadius.circular(1),
                                                  ),
                                                ),
                                              ),
                                            ),
                                          ),
                                        ],
                                      ),
                                    ),
                                    for (final col in columns)
                                      SizedBox(
                                        width: _columnWidth(col),
                                        child: Row(
                                          children: [
                                            Expanded(
                                              child: InkWell(
                                                onTap: () => _toggleSort(col),
                                                child: Row(
                                                  mainAxisSize: MainAxisSize.min,
                                                  children: [
                                                    Flexible(
                                                      child: Text(
                                                        AppLocale.t(widget.currentLocale, col).toUpperCase(),
                                                        style: TextStyle(
                                                          fontWeight: FontWeight.bold,
                                                          fontSize: 12,
                                                          letterSpacing: 0.5,
                                                          color: _sortColumn == col ? Colors.amber : Colors.grey,
                                                        ),
                                                        overflow: TextOverflow.ellipsis,
                                                      ),
                                                    ),
                                                    const SizedBox(width: 4),
                                                    Icon(
                                                      _sortColumn == col
                                                          ? (_sortAscending ? Icons.arrow_upward_rounded : Icons.arrow_downward_rounded)
                                                          : Icons.unfold_more_rounded,
                                                      size: 14,
                                                      color: _sortColumn == col ? Colors.amber : (isDark ? Colors.white24 : Colors.black26),
                                                    ),
                                                  ],
                                                ),
                                              ),
                                            ),
                                            MouseRegion(
                                              cursor: SystemMouseCursors.resizeColumn,
                                              child: GestureDetector(
                                                behavior: HitTestBehavior.opaque,
                                                onHorizontalDragUpdate: (details) {
                                                  setState(() {
                                                    final cur = _columnWidth(col);
                                                    _customColumnWidths[col] = (cur + details.delta.dx).clamp(60.0, 600.0);
                                                  });
                                                },
                                                child: Container(
                                                  width: 14,
                                                  height: 28,
                                                  alignment: Alignment.center,
                                                  color: Colors.transparent,
                                                  child: Container(
                                                    width: 2,
                                                    height: 14,
                                                    decoration: BoxDecoration(
                                                      color: isDark ? Colors.white24 : Colors.black26,
                                                      borderRadius: BorderRadius.circular(1),
                                                    ),
                                                  ),
                                                ),
                                              ),
                                            ),
                                          ],
                                        ),
                                      ),
                                    SizedBox(
                                      width: _columnWidth('status'),
                                      child: Row(
                                        children: [
                                          Expanded(
                                            child: InkWell(
                                              onTap: () => _toggleSort('status'),
                                              child: Row(
                                                mainAxisSize: MainAxisSize.min,
                                                children: [
                                                  Text(
                                                    AppLocale.t(widget.currentLocale, 'status').toUpperCase(),
                                                    style: TextStyle(
                                                      fontWeight: FontWeight.bold,
                                                      fontSize: 12,
                                                      letterSpacing: 0.5,
                                                      color: _sortColumn == 'status' ? Colors.amber : Colors.grey,
                                                    ),
                                                  ),
                                                  const SizedBox(width: 4),
                                                  Icon(
                                                    _sortColumn == 'status'
                                                        ? (_sortAscending ? Icons.arrow_upward_rounded : Icons.arrow_downward_rounded)
                                                        : Icons.unfold_more_rounded,
                                                    size: 14,
                                                    color: _sortColumn == 'status' ? Colors.amber : (isDark ? Colors.white24 : Colors.black26),
                                                  ),
                                                ],
                                              ),
                                            ),
                                          ),
                                          MouseRegion(
                                            cursor: SystemMouseCursors.resizeColumn,
                                            child: GestureDetector(
                                              behavior: HitTestBehavior.opaque,
                                              onHorizontalDragUpdate: (details) {
                                                setState(() {
                                                  final cur = _columnWidth('status');
                                                  _customColumnWidths['status'] = (cur + details.delta.dx).clamp(60.0, 400.0);
                                                });
                                              },
                                              child: Container(
                                                width: 14,
                                                height: 28,
                                                alignment: Alignment.center,
                                                color: Colors.transparent,
                                                child: Container(
                                                  width: 2,
                                                  height: 14,
                                                  decoration: BoxDecoration(
                                                    color: isDark ? Colors.white24 : Colors.black26,
                                                    borderRadius: BorderRadius.circular(1),
                                                  ),
                                                ),
                                              ),
                                            ),
                                          ),
                                        ],
                                      ),
                                    ),
                                    SizedBox(
                                      width: 140,
                                      child: Align(
                                        alignment: Alignment.centerRight,
                                        child: Text(
                                          AppLocale.t(widget.currentLocale, 'actions').toUpperCase(),
                                          style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12, letterSpacing: 0.5, color: Colors.grey),
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
              if (_currentListMode == 'pagination')
                _buildPaginationBar(context, isMobile: false)
              else
                _buildInfiniteScrollFooter(isDark),
            ],
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
    if (widget.slug == 'partners') headerIcon = Icons.local_gas_station_outlined;
    if (widget.slug == 'customers') headerIcon = Icons.business_outlined;
    if (widget.slug.contains('branch')) headerIcon = Icons.storefront_outlined;
    if (widget.slug.contains('vehicle')) headerIcon = Icons.local_shipping_outlined;
    if (widget.slug.contains('nfc') || widget.slug.contains('chip')) headerIcon = Icons.nfc_outlined;
    if (widget.slug.contains('expense') || widget.slug.contains('invoice') || widget.slug.contains('account')) headerIcon = Icons.account_balance_wallet_outlined;

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
                      width: _columnWidth('id'),
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
                      width: _columnWidth('status'),
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
                                AppLocale.t(widget.currentLocale, status).toUpperCase(),
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
    final theme = Theme.of(context);
    final isDark = theme.brightness == Brightness.dark;
    final isMobile = MediaQuery.of(context).size.width < 768;
    final tabs = _getTabs();

    double bottomHeight = 0.0;
    if (!isMobile) {
      bottomHeight = 56.0;
      if (tabs.isNotEmpty) bottomHeight += 44.0;
      if (activeFilterCount > 0) bottomHeight += 44.0;
    } else {
      if (tabs.isNotEmpty) bottomHeight += 36.0;
      if (activeFilterCount > 0) bottomHeight += 34.0;
    }

    return Scaffold(
      appBar: AppBar(
        toolbarHeight: isMobile ? 48 : null,
        titleSpacing: isMobile ? 8 : null,
        title: isMobile
            ? Row(
                children: [
                  Expanded(
                    child: SizedBox(
                      height: 36,
                      child: TextField(
                        style: const TextStyle(fontSize: 13),
                        decoration: InputDecoration(
                          hintText: '${AppLocale.t(widget.currentLocale, "search")}...',
                          hintStyle: const TextStyle(fontSize: 12),
                          prefixIcon: const Icon(Icons.search, size: 18),
                          suffixIcon: search.isNotEmpty
                              ? IconButton(
                                  icon: const Icon(Icons.clear, size: 16),
                                  onPressed: () {
                                    setState(() => search = '');
                                    loadRecords(resetPage: true);
                                  },
                                )
                              : null,
                          isDense: true,
                          contentPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                          border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                        ),
                        onChanged: (v) {
                          setState(() => search = v);
                        },
                        onSubmitted: (v) {
                          setState(() => search = v);
                          loadRecords(resetPage: true);
                        },
                      ),
                    ),
                  ),
                  const SizedBox(width: 6),
                  IconButton.filledTonal(
                    visualDensity: VisualDensity.compact,
                    tooltip: _showRelations ? AppLocale.t(widget.currentLocale, 'hide_relations') : AppLocale.t(widget.currentLocale, 'show_relations'),
                    style: IconButton.styleFrom(
                      backgroundColor: _showRelations ? Colors.amber.withValues(alpha: 0.2) : null,
                      foregroundColor: _showRelations ? Colors.amber.shade400 : null,
                      padding: const EdgeInsets.all(6),
                    ),
                    icon: Icon(_showRelations ? Icons.account_tree : Icons.account_tree_outlined, size: 18),
                    onPressed: () {
                      setState(() {
                        _showRelations = !_showRelations;
                      });
                    },
                  ),
                  const SizedBox(width: 6),
                  Badge(
                    isLabelVisible: activeFilterCount > 0,
                    label: Text('$activeFilterCount', style: const TextStyle(fontSize: 10)),
                    backgroundColor: Colors.amber,
                    textColor: Colors.black,
                    child: IconButton.filledTonal(
                      visualDensity: VisualDensity.compact,
                      tooltip: AppLocale.t(widget.currentLocale, 'filters'),
                      style: IconButton.styleFrom(padding: const EdgeInsets.all(6)),
                      icon: const Icon(Icons.tune_rounded, size: 18),
                      onPressed: () => _showFilterModal(context),
                    ),
                  ),
                  const SizedBox(width: 6),
                  PopupMenuButton<String>(
                    tooltip: selectedRecordIds.isNotEmpty
                        ? 'Bulk Actions (${selectedRecordIds.length} selected)'
                        : 'Bulk Selection & Actions',
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
                    icon: Badge(
                      isLabelVisible: selectedRecordIds.isNotEmpty,
                      label: Text('${selectedRecordIds.length}', style: const TextStyle(fontSize: 10)),
                      backgroundColor: Colors.amber.shade700,
                      textColor: Colors.black,
                      child: IconButton.filledTonal(
                        visualDensity: VisualDensity.compact,
                        style: IconButton.styleFrom(
                          backgroundColor: selectedRecordIds.isNotEmpty ? Colors.amber.withValues(alpha: 0.2) : null,
                          foregroundColor: selectedRecordIds.isNotEmpty ? Colors.amber.shade400 : null,
                          padding: const EdgeInsets.all(6),
                        ),
                        icon: const Icon(Icons.checklist_rtl_rounded, size: 18),
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
                            Icon(isAllSelected ? Icons.deselect_rounded : Icons.select_all_rounded, size: 18),
                            const SizedBox(width: 10),
                            Text(isAllSelected ? AppLocale.t(widget.currentLocale, 'deselect_all') : '${AppLocale.t(widget.currentLocale, "select_all")} (${records.length})', style: const TextStyle(fontSize: 13)),
                          ],
                        ),
                      ),
                      if (selectedRecordIds.isNotEmpty) ...[
                        PopupMenuItem(
                          value: 'delete',
                          child: Row(
                            children: [
                              const Icon(Icons.delete_sweep_rounded, color: Colors.redAccent, size: 18),
                              const SizedBox(width: 10),
                              Text(AppLocale.t(widget.currentLocale, 'bulk_delete'), style: const TextStyle(color: Colors.redAccent, fontSize: 13)),
                            ],
                          ),
                        ),
                        PopupMenuItem(
                          value: 'export',
                          child: Row(
                            children: [
                              const Icon(Icons.download_rounded, size: 18),
                              const SizedBox(width: 10),
                              Text(AppLocale.t(widget.currentLocale, 'export_selected'), style: const TextStyle(fontSize: 13)),
                            ],
                          ),
                        ),
                        const PopupMenuDivider(),
                        PopupMenuItem(
                          value: 'deselect',
                          child: Row(
                            children: [
                              const Icon(Icons.close_rounded, size: 18),
                              const SizedBox(width: 10),
                              Text(AppLocale.t(widget.currentLocale, 'clear_selection'), style: const TextStyle(fontSize: 13)),
                            ],
                          ),
                        ),
                      ],
                    ],
                  ),
                ],
              )
            : Text(widget.slug.toUpperCase(), style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
        bottom: bottomHeight > 0
            ? PreferredSize(
                preferredSize: Size.fromHeight(bottomHeight),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (!isMobile)
                      Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                        child: Row(
                          children: [
                            Expanded(
                              child: TextField(
                                decoration: InputDecoration(
                                  hintText: '${AppLocale.t(widget.currentLocale, "search")} ${widget.slug}...',
                                  prefixIcon: const Icon(Icons.search, size: 20),
                                  suffixIcon: search.isNotEmpty
                                      ? IconButton(
                                          icon: const Icon(Icons.clear, size: 18),
                                          onPressed: () {
                                            setState(() => search = '');
                                            loadRecords(resetPage: true);
                                          },
                                        )
                                      : null,
                                  isDense: true,
                                  border: OutlineInputBorder(borderRadius: BorderRadius.circular(10)),
                                ),
                                onChanged: (v) {
                                  setState(() => search = v);
                                },
                                onSubmitted: (v) {
                                  setState(() => search = v);
                                  loadRecords(resetPage: true);
                                },
                              ),
                            ),
                            const SizedBox(width: 8),
                            IconButton.filledTonal(
                              tooltip: _showRelations ? AppLocale.t(widget.currentLocale, 'hide_relations') : AppLocale.t(widget.currentLocale, 'show_relations'),
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
                                tooltip: AppLocale.t(widget.currentLocale, 'filters'),
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
                                      Text(isAllSelected ? AppLocale.t(widget.currentLocale, 'deselect_all') : '${AppLocale.t(widget.currentLocale, "select_all")} (${records.length})'),
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
                    if (tabs.isNotEmpty)
                      Container(
                        height: isMobile ? 32 : 38,
                        margin: EdgeInsets.fromLTRB(isMobile ? 8 : 16, 2, isMobile ? 8 : 16, 4),
                        child: ListView.separated(
                          scrollDirection: Axis.horizontal,
                          itemCount: tabs.length,
                          separatorBuilder: (_, __) => SizedBox(width: isMobile ? 6 : 8),
                          itemBuilder: (ctx, idx) {
                            final tab = tabs[idx];
                            final isSelected = (_activeTabKey ?? tabs.first['key']) == tab['key'];
                            final count = _getTabCount(tab);
                            return ChoiceChip(
                              visualDensity: VisualDensity.compact,
                              padding: EdgeInsets.symmetric(horizontal: isMobile ? 4 : 8, vertical: 0),
                              label: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Text(tab['label'] ?? 'Tab', style: TextStyle(fontSize: isMobile ? 11 : 12)),
                                  const SizedBox(width: 4),
                                  Container(
                                    padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
                                    decoration: BoxDecoration(
                                      color: isSelected
                                          ? Colors.amber.shade700
                                          : (isDark ? Colors.white12 : Colors.black12),
                                      borderRadius: BorderRadius.circular(10),
                                    ),
                                    child: Text(
                                      '$count',
                                      style: TextStyle(
                                        fontSize: 9,
                                        fontWeight: FontWeight.bold,
                                        color: isSelected ? Colors.black : (isDark ? Colors.white70 : Colors.black87),
                                      ),
                                    ),
                                  ),
                                ],
                              ),
                              selected: isSelected,
                              onSelected: (selected) {
                                if (selected) {
                                  setState(() {
                                    _activeTabKey = tab['key'];
                                    selectedRecordIds.clear();
                                  });
                                  loadRecords(resetPage: true);
                                }
                              },
                              selectedColor: Colors.amber.withValues(alpha: 0.25),
                              backgroundColor: Colors.transparent,
                              shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(20),
                                side: BorderSide(
                                  color: isSelected ? Colors.amber : (isDark ? Colors.white24 : Colors.black26),
                                ),
                              ),
                            );
                          },
                        ),
                      ),
                    if (activeFilterCount > 0)
                      Container(
                        height: isMobile ? 30 : 42,
                        padding: EdgeInsets.symmetric(horizontal: isMobile ? 8 : 16, vertical: 2),
                        child: ListView(
                          scrollDirection: Axis.horizontal,
                    children: [
                      ActionChip(
                        avatar: const Icon(Icons.close, size: 14),
                        label: Text(AppLocale.t(widget.currentLocale, 'clear_all'), style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold)),
                        onPressed: _clearAllFilters,
                        visualDensity: VisualDensity.compact,
                      ),
                      const SizedBox(width: 6),
                      if (_sortColumn != null) ...[
                        InputChip(
                          avatar: Icon(
                            _sortAscending ? Icons.arrow_upward_rounded : Icons.arrow_downward_rounded,
                            size: 13,
                            color: Colors.amber,
                          ),
                          label: Text(
                            '${AppLocale.t(widget.currentLocale, "sort_by")}: ${_titleCase(_sortColumn!)} (${_sortAscending ? AppLocale.t(widget.currentLocale, "ascending") : AppLocale.t(widget.currentLocale, "descending")})',
                            style: const TextStyle(fontSize: 11),
                          ),
                          onDeleted: () {
                            setState(() {
                              _sortColumn = null;
                              _sortAscending = true;
                            });
                          },
                          visualDensity: VisualDensity.compact,
                          deleteIconColor: Colors.amber,
                        ),
                        const SizedBox(width: 6),
                      ],
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
        )
      : null,
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
                            : AppLocale.t(widget.currentLocale, 'no_records_found'),
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
                            child: NotificationListener<ScrollNotification>(
                              onNotification: (notif) {
                                if (_currentListMode == 'infinite_scroll' &&
                                    notif.metrics.pixels >= notif.metrics.maxScrollExtent - 200) {
                                  _loadMoreRecords();
                                }
                                return false;
                              },
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
                          ),
                        ],
                      );
                    },
                  ),
                ),
      floatingActionButton: _buildFloatingActionButtons(context),
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
    if (widget.slug == 'partners') {
      final nameCtrl = TextEditingController(text: item['name'] ?? item['Name'] ?? '');
      final nameArCtrl = TextEditingController(text: item['name_ar'] ?? item['NameAr'] ?? '');
      final crCtrl = TextEditingController(text: item['cr_number'] ?? item['CrNumber'] ?? '');
      final phoneCtrl = TextEditingController(text: item['phone'] ?? item['Phone'] ?? '');

      showModalBottomSheet(
        context: context,
        isScrollControlled: true,
        builder: (ctx) => Padding(
          padding: EdgeInsets.only(left: 20, right: 20, top: 20, bottom: MediaQuery.of(ctx).viewInsets.bottom + 20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text('Edit Station Partner #$id', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
              const SizedBox(height: 16),
              TextField(controller: nameCtrl, decoration: const InputDecoration(labelText: 'Company Name', border: OutlineInputBorder(), isDense: true)),
              const SizedBox(height: 12),
              TextField(controller: nameArCtrl, decoration: const InputDecoration(labelText: 'Arabic Name', border: OutlineInputBorder(), isDense: true)),
              const SizedBox(height: 12),
              TextField(controller: crCtrl, decoration: const InputDecoration(labelText: 'CR Number', border: OutlineInputBorder(), isDense: true)),
              const SizedBox(height: 12),
              TextField(controller: phoneCtrl, decoration: const InputDecoration(labelText: 'Phone', border: OutlineInputBorder(), isDense: true)),
              const SizedBox(height: 20),
              ElevatedButton.icon(
                icon: const Icon(Icons.save),
                label: const Text('Update Partner Details'),
                onPressed: () async {
                  await widget.api.updateRecord(widget.slug, id, {
                    'name': nameCtrl.text,
                    'name_ar': nameArCtrl.text,
                    'cr_number': crCtrl.text,
                    'phone': phoneCtrl.text,
                  });
                  if (!mounted) return;
                  Navigator.pop(ctx);
                  loadRecords();
                },
              ),
            ],
          ),
        ),
      );
      return;
    }

    if (widget.slug == 'customers') {
      final codeCtrl = TextEditingController(text: item['code'] ?? item['Code'] ?? '');
      final nameCtrl = TextEditingController(text: item['name'] ?? item['Name'] ?? '');
      final nameArCtrl = TextEditingController(text: item['name_ar'] ?? item['NameAr'] ?? '');
      final phoneCtrl = TextEditingController(text: item['phone'] ?? item['Phone'] ?? '');
      final limitCtrl = TextEditingController(text: '${item['credit_limit'] ?? item['CreditLimit'] ?? '0.00'}');

      showModalBottomSheet(
        context: context,
        isScrollControlled: true,
        builder: (ctx) => Padding(
          padding: EdgeInsets.only(left: 20, right: 20, top: 20, bottom: MediaQuery.of(ctx).viewInsets.bottom + 20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text('Edit Fleet Company #$id', style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18)),
              const SizedBox(height: 16),
              TextField(controller: codeCtrl, decoration: const InputDecoration(labelText: 'Fleet Code', border: OutlineInputBorder(), isDense: true)),
              const SizedBox(height: 12),
              TextField(controller: nameCtrl, decoration: const InputDecoration(labelText: 'Company Name', border: OutlineInputBorder(), isDense: true)),
              const SizedBox(height: 12),
              TextField(controller: nameArCtrl, decoration: const InputDecoration(labelText: 'Arabic Name', border: OutlineInputBorder(), isDense: true)),
              const SizedBox(height: 12),
              TextField(controller: phoneCtrl, decoration: const InputDecoration(labelText: 'Phone', border: OutlineInputBorder(), isDense: true)),
              const SizedBox(height: 12),
              TextField(controller: limitCtrl, decoration: const InputDecoration(labelText: 'Credit Limit (SAR)', border: OutlineInputBorder(), isDense: true)),
              const SizedBox(height: 20),
              ElevatedButton.icon(
                icon: const Icon(Icons.save),
                label: const Text('Update Fleet Company'),
                onPressed: () async {
                  await widget.api.updateRecord(widget.slug, id, {
                    'code': codeCtrl.text,
                    'name': nameCtrl.text,
                    'name_ar': nameArCtrl.text,
                    'phone': phoneCtrl.text,
                    'credit_limit': limitCtrl.text,
                  });
                  if (!mounted) return;
                  Navigator.pop(ctx);
                  loadRecords();
                },
              ),
            ],
          ),
        ),
      );
      return;
    }

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
                    Text(
                      AppLocale.t(currentLocale, 'account_profile'),
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
                                  api.currentUser?['email'] ?? 'admin@benzat.com',
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
                      title: AppLocale.t(currentLocale, 'orders_procurement'),
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
                      title: AppLocale.t(currentLocale, 'billing_license'),
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
                      title: AppLocale.t(currentLocale, 'security_biometrics'),
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
