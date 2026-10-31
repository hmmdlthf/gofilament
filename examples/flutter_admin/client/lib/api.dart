import 'dart:convert';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class FilamentApiClient {
  final String baseUrl;
  String? activeTenant;

  FilamentApiClient({String? baseUrl}) : baseUrl = baseUrl ?? _defaultBaseUrl();

  static String _defaultBaseUrl() {
    if (kIsWeb) {
      final origin = Uri.base.origin;
      return '$origin/admin';
    }
    return 'http://localhost:8081/admin';
  }

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
