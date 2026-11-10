import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

import 'session_store.dart';

/// An error answered by the panel API. [message] is safe to show to the user.
class ApiException implements Exception {
  ApiException(this.status, this.message);

  final int status;
  final String message;

  bool get isUnauthorized => status == 401;
  bool get isForbidden => status == 403;

  @override
  String toString() => message;
}

/// One page of a resource list.
class ListPage {
  ListPage(
      {required this.rows,
      required this.total,
      required this.page,
      required this.perPage});

  final List<Map<String, dynamic>> rows;
  final int total;
  final int page;
  final int perPage;
}

/// A resource form: its component tree, the record being edited and what the
/// user may do with it.
class FormPayload {
  FormPayload({
    required this.mode,
    required this.schema,
    required this.state,
    required this.abilities,
    required this.optionLabels,
  });

  final String mode;
  final List<Map<String, dynamic>> schema;
  final Map<String, dynamic> state;
  final Map<String, bool> abilities;
  final Map<String, String> optionLabels;

  bool get isEdit => mode == 'edit';
}

/// A choice of a select field.
class FieldOption {
  FieldOption(this.value, this.label, [this.description]);

  final String value;
  final String label;
  final String? description;

  static List<FieldOption> listFrom(dynamic raw) {
    if (raw is! List) return [];
    return raw
        .whereType<Map>()
        .map((m) => FieldOption('${m['value']}', '${m['label'] ?? m['value']}',
            m['description'] as String?))
        .toList();
  }
}

/// Client of a GoFilament panel API.
///
/// The panel path is taken from the page URL on web (the first path segment,
/// which is the `<base href>` the server injects), so one build serves every
/// panel and every host. Pass [baseUrl] explicitly on mobile and desktop.
class AdminApi {
  AdminApi(
      {String? baseUrl,
      http.Client? client,
      SessionStore? store,
      String fallbackPanelPath = '/admin'})
      : baseUrl = _trim(baseUrl ?? defaultBaseUrl(fallbackPanelPath)),
        _http = client ?? http.Client(),
        _store = store ?? PrefsSessionStore();

  final String baseUrl;
  final http.Client _http;
  final SessionStore _store;

  String? token;
  Map<String, dynamic>? user;

  /// Called when the server rejects the session (expired or revoked token).
  VoidCallback? onUnauthorized;

  bool get isAuthenticated => token != null && token!.isNotEmpty;

  static String _trim(String s) =>
      s.endsWith('/') ? s.substring(0, s.length - 1) : s;

  static String defaultBaseUrl(String fallbackPanelPath) {
    if (kIsWeb) {
      final segments =
          Uri.base.pathSegments.where((s) => s.isNotEmpty).toList();
      final panel =
          segments.isNotEmpty ? '/${segments.first}' : fallbackPanelPath;
      return '${Uri.base.origin}$panel';
    }
    return fallbackPanelPath;
  }

  String get _key =>
      'gofilament:${Uri.parse(baseUrl).host}${Uri.parse(baseUrl).path}';

  Map<String, String> get _headers => {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
        if (isAuthenticated) 'Authorization': 'Bearer $token',
      };

  Uri _uri(String path, [Map<String, String>? query]) {
    final uri = Uri.parse('$baseUrl/api$path');
    if (query == null || query.isEmpty) return uri;
    return uri.replace(queryParameters: {...uri.queryParameters, ...query});
  }

  dynamic _decode(http.Response res) {
    dynamic body;
    if (res.body.isNotEmpty) {
      try {
        body = json.decode(utf8.decode(res.bodyBytes));
      } catch (_) {
        body = null;
      }
    }
    if (res.statusCode >= 200 && res.statusCode < 300) return body;

    var message = 'Request failed (${res.statusCode})';
    if (body is Map && body['error'] is String) {
      message = body['error'] as String;
    } else if (body is Map && body['message'] is String) {
      message = body['message'] as String;
    }
    if (res.statusCode == 401 && isAuthenticated) {
      token = null;
      user = null;
      onUnauthorized?.call();
    }
    throw ApiException(res.statusCode, message);
  }

  Future<dynamic> _get(String path, [Map<String, String>? query]) async =>
      _decode(await _http.get(_uri(path, query), headers: _headers));

  Future<dynamic> _send(String method, String path, [Object? body]) async {
    final req = http.Request(method, _uri(path))..headers.addAll(_headers);
    if (body != null) req.body = json.encode(body);
    return _decode(await http.Response.fromStream(await _http.send(req)));
  }

  static Map<String, dynamic> _map(dynamic v) =>
      v is Map ? Map<String, dynamic>.from(v) : <String, dynamic>{};

  // ---------------------------------------------------------------- session

  /// Brand, theme and auth settings. Public: used to draw the login screen.
  Future<Map<String, dynamic>> boot() async => _map(await _get('/boot'));

  /// Full panel description for the signed-in user.
  Future<Map<String, dynamic>> schema() async => _map(await _get('/schema'));

  Future<void> login(String username, String password,
      {bool remember = false}) async {
    final data = _map(await _send('POST', '/login',
        {'username': username, 'password': password, 'remember': remember}));
    token = data['token'] as String?;
    user = _map(data['user']);
    if (token == null || token!.isEmpty) {
      throw ApiException(500, 'The server did not return a session');
    }
    await _store.write(
        '$_key:session',
        json.encode(
            {'token': token, 'user': user, 'expires_at': data['expires_at']}));
  }

  /// Restores a stored session. Returns false when there is none or it expired.
  Future<bool> restore() async {
    final raw = await _store.read('$_key:session');
    if (raw == null || raw.isEmpty) return false;
    try {
      final data = _map(json.decode(raw));
      final expires = DateTime.tryParse('${data['expires_at'] ?? ''}');
      if (expires != null && DateTime.now().isAfter(expires)) {
        await _store.remove('$_key:session');
        return false;
      }
      token = data['token'] as String?;
      user = _map(data['user']);
      return isAuthenticated;
    } catch (_) {
      return false;
    }
  }

  Future<void> logout() async {
    try {
      if (isAuthenticated) await _send('POST', '/logout');
    } catch (_) {}
    token = null;
    user = null;
    await _store.remove('$_key:session');
  }

  Future<String?> readPreference(String name) =>
      _store.read('$_key:pref:$name');

  Future<void> writePreference(String name, String value) =>
      _store.write('$_key:pref:$name', value);

  // ---------------------------------------------------------------- resources

  Future<ListPage> list(
    String slug, {
    int page = 1,
    int perPage = 25,
    String search = '',
    String sort = '',
    bool descending = false,
    Map<String, String> filters = const {},
  }) async {
    final data = _map(await _get('/resources/$slug', {
      'page': '$page',
      'per_page': '$perPage',
      if (search.isNotEmpty) 'search': search,
      if (sort.isNotEmpty) 'sort': sort,
      if (sort.isNotEmpty) 'direction': descending ? 'desc' : 'asc',
      for (final e in filters.entries)
        if (e.value.isNotEmpty) 'filter_${e.key}': e.value,
    }));
    final rows = (data['data'] is List ? data['data'] as List : const [])
        .whereType<Map>()
        .map((m) => Map<String, dynamic>.from(m))
        .toList();
    return ListPage(
      rows: rows,
      total: (data['total'] as num?)?.toInt() ?? rows.length,
      page: (data['page'] as num?)?.toInt() ?? page,
      perPage: (data['per_page'] as num?)?.toInt() ?? perPage,
    );
  }

  Future<FormPayload> form(String slug, {String? id}) async {
    final data =
        _map(await _get('/resources/$slug/form', {if (id != null) 'id': id}));
    return FormPayload(
      mode: '${data['mode'] ?? (id == null ? 'create' : 'edit')}',
      schema: (data['schema'] is List ? data['schema'] as List : const [])
          .whereType<Map>()
          .map((m) => Map<String, dynamic>.from(m))
          .toList(),
      state: _map(data['state']),
      abilities: _map(data['abilities']).map((k, v) => MapEntry(k, v == true)),
      optionLabels:
          _map(data['option_labels']).map((k, v) => MapEntry(k, '$v')),
    );
  }

  Future<List<FieldOption>> options(String slug, String field, String query,
      {int limit = 25}) async {
    final data = _map(await _get('/resources/$slug/form/options',
        {'field': field, 'q': query, 'limit': '$limit'}));
    return FieldOption.listFrom(data['options']);
  }

  Future<void> create(String slug, Map<String, dynamic> data) =>
      _send('POST', '/resources/$slug', data);

  Future<void> update(String slug, String id, Map<String, dynamic> data) =>
      _send('PUT', '/resources/$slug/$id', data);

  Future<void> delete(String slug, String id) =>
      _send('DELETE', '/resources/$slug/$id');

  Future<void> deleteMany(String slug, List<String> ids) => _send('POST',
      '/resources/$slug/bulk', {'bulk_action': 'delete_bulk', 'ids': ids});

  Future<void> updateCell(
          String slug, String id, String column, dynamic value) =>
      _send('POST', '/resources/$slug/$id/cell',
          {'column': column, 'value': value});

  // ---------------------------------------------------------------- dashboard

  Future<List<Map<String, dynamic>>> widgets() async {
    final data = _map(await _get('/widgets'));
    return (data['widgets'] is List ? data['widgets'] as List : const [])
        .whereType<Map>()
        .map((m) => Map<String, dynamic>.from(m))
        .toList();
  }

  /// Authenticated GET against a custom panel endpoint (for app-specific pages).
  Future<dynamic> get(String path, [Map<String, String>? query]) =>
      _get(path, query);

  /// Authenticated request with a JSON body against a custom panel endpoint.
  Future<dynamic> send(String method, String path, [Object? body]) =>
      _send(method, path, body);
}
