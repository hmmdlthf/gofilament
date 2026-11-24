import 'dart:convert';
import 'i18n.dart';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

import 'session_store.dart';

/// An error answered by the panel API. [message] is safe to show to the user.
class ApiException implements Exception {
  ApiException(this.status, this.message, [this.body]);

  final int status;
  final String message;

  /// The decoded error body, when the server sent JSON.
  final Map<String, dynamic>? body;

  bool get isConflict => status == 409;

  bool get isUnauthorized => status == 401;
  bool get isForbidden => status == 403;

  /// The account must verify its email address first.
  bool get isEmailUnverified =>
      status == 403 && body?['email_unverified'] == true;

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

  /// Per-record abilities: `{id: {update, delete, actions: [...]}}`.
  Map<String, Map<String, dynamic>> rowMeta = const {};

  /// Aggregates over all matching records (resource SummariesUsing):
  /// `{column: {sum: 1.5, count: 3}}`.
  Map<String, Map<String, double>> summaries = const {};

  /// Live counts of the refined filters' values, ignoring each column's own
  /// filter: `{column: {value: count}}`.
  Map<String, Map<String, int>> facets = const {};
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
    this.version,
  });

  /// The record's version (Versioned resources): sent back as `_version`
  /// on save so the server can refuse to overwrite a newer change.
  final String? version;

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

  /// The tenant to work in (sent as X-Tenant-ID; the panel checks the user
  /// belongs to it); null: the panel's default.
  String? tenant;

  /// The language the panel answers in (sent as Accept-Language); null:
  /// the panel's default.
  String? locale;

  Map<String, String> get _headers => {
        'Accept': 'application/json',
        'Content-Type': 'application/json',
        if (locale != null && locale!.isNotEmpty) 'Accept-Language': locale!,
        if (tenant != null && tenant!.isNotEmpty) 'X-Tenant-ID': tenant!,
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
    throw ApiException(res.statusCode, message,
        body is Map ? Map<String, dynamic>.from(body) : null);
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

  /// Signs in. Returns null when signed in, or the challenge id when the
  /// user has two-factor sign-in on: finish with [loginTwoFactor].
  Future<String?> login(String username, String password,
      {bool remember = false}) async {
    final data = _map(await _send('POST', '/login',
        {'username': username, 'password': password, 'remember': remember}));
    if (data['two_factor'] == true) {
      final challenge = '${data['challenge'] ?? ''}';
      if (challenge.isEmpty) {
        throw ApiException(500, tr('The server did not return a challenge'));
      }
      return challenge;
    }
    await _storeSession(data);
    return null;
  }

  /// The second sign-in step: an authenticator or recovery code.
  Future<void> loginTwoFactor(String challenge, String code) async {
    final data = _map(await _send(
        'POST', '/login/two-factor', {'challenge': challenge, 'code': code}));
    await _storeSession(data);
  }

  Future<void> _storeSession(Map<String, dynamic> data) async {
    token = data['token'] as String?;
    user = _map(data['user']);
    if (token == null || token!.isEmpty) {
      throw ApiException(500, tr('The server did not return a session'));
    }
    await _store.write(
        '$_key:session',
        json.encode(
            {'token': token, 'user': user, 'expires_at': data['expires_at']}));
  }

  // ---------------------------------------------------------------- account

  /// The signed-in user's account: name, email, features, two-factor state.
  Future<Map<String, dynamic>> account() async => _map(await _get('/account'));

  Future<Map<String, dynamic>> updateAccount(String name, String email) async =>
      _map(await _send('PUT', '/account', {'name': name, 'email': email}));

  Future<void> changePassword(
          String current, String password, String confirmation) =>
      _send('POST', '/account/password', {
        'current_password': current,
        'password': password,
        'password_confirmation': confirmation,
      });

  /// Starts two-factor setup: returns {secret, otpauth_uri}.
  Future<Map<String, dynamic>> twoFactorEnable(String currentPassword) async =>
      _map(await _send('POST', '/account/two-factor/enable',
          {'current_password': currentPassword}));

  /// Confirms setup with a code; the reply may hold new recovery_codes.
  Future<Map<String, dynamic>> twoFactorConfirm(String code) async =>
      _map(await _send('POST', '/account/two-factor/confirm', {'code': code}));

  Future<Map<String, dynamic>> twoFactorDisable(String currentPassword) async =>
      _map(await _send('POST', '/account/two-factor/disable',
          {'current_password': currentPassword}));

  /// Replaces the recovery codes; the reply holds the new recovery_codes.
  Future<Map<String, dynamic>> newRecoveryCodes(String currentPassword) async =>
      _map(await _send('POST', '/account/two-factor/recovery-codes',
          {'current_password': currentPassword}));

  /// Creates an account (Panel.Registration) and signs it in.
  Future<void> register(
      String name, String email, String password, String confirmation) async {
    final data = _map(await _send('POST', '/register', {
      'name': name,
      'email': email,
      'password': password,
      'password_confirmation': confirmation,
    }));
    if (data['token'] == null) {
      throw ApiException(
          200, tr('Your account was created. Sign in to continue.'), data);
    }
    await _storeSession(data);
  }

  /// Sends the email verification link again. Returns true when the
  /// address is already verified.
  Future<bool> resendVerification() async {
    final data = _map(await _send('POST', '/email/verification-notification'));
    return data['email_verified'] == true;
  }

  /// Asks for a password reset link; returns the server's message.
  Future<String> forgotPassword(String email) async {
    final data =
        _map(await _send('POST', '/forgot-password', {'email': email}));
    return '${data['message'] ?? 'Check your email.'}';
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
    String advancedSearch = '',
  }) async {
    final data = _map(await _get('/resources/$slug', {
      if (advancedSearch.isNotEmpty) 'advanced_search': advancedSearch,
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
    final meta = _map(data['row_meta']);
    return ListPage(
      rows: rows,
      total: (data['total'] as num?)?.toInt() ?? rows.length,
      page: (data['page'] as num?)?.toInt() ?? page,
      perPage: (data['per_page'] as num?)?.toInt() ?? perPage,
    )
      ..rowMeta = meta.map((k, v) => MapEntry(k, _map(v)))
      ..summaries = _map(data['summaries']).map((col, v) => MapEntry(col,
          _map(v).map((k, n) => MapEntry(k, (n is num) ? n.toDouble() : 0.0))))
      ..facets = _map(data['facets']).map((col, v) => MapEntry(col,
          _map(v).map((k, n) => MapEntry(k, (n is num) ? n.toInt() : 0))));
  }

  /// One record, as the resource's Go type serializes it.
  Future<Map<String, dynamic>> find(String slug, String id) async =>
      _map(_map(await _get('/resources/$slug/$id'))['data']);

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
      version: data['version'] == null ? null : '${data['version']}',
    );
  }

  /// The form that creates a missing option of select [field]
  /// (`create_option` in the field's schema).
  Future<FormPayload> createOptionForm(String slug, String field) async =>
      _formPayload(
          _map(await _get(
              '/resources/$slug/form/options/create', {'field': field})),
          false);

  /// Creates an option of select [field] from the create form's [data] and
  /// returns it, so it can be selected.
  Future<FieldOption> createOption(
      String slug, String field, Map<String, dynamic> data) async {
    final out = _map(await _send(
        'POST',
        '/resources/$slug/form/options/create'
            '?field=${Uri.encodeQueryComponent(field)}',
        data));
    final o = _map(out['option']);
    return FieldOption('${o['value']}', '${o['label'] ?? o['value']}');
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

  /// Uploads one file for the file field [field] of a resource form;
  /// returns `{value, url, name}` (value is what the field keeps).
  Future<Map<String, dynamic>> upload(
      String slug, String field, String filename, Uint8List bytes) async {
    final req = http.MultipartRequest(
        'POST', _uri('/resources/$slug/uploads', {'field': field}))
      ..headers.addAll({
        'Accept': 'application/json',
        if (isAuthenticated) 'Authorization': 'Bearer $token',
      })
      ..files
          .add(http.MultipartFile.fromBytes('file', bytes, filename: filename));
    return _map(_decode(await http.Response.fromStream(await _http.send(req))));
  }

  /// A record's history (Resource.History): newest first, each entry
  /// {event, description, causer_name, at, changes: [{field, label, old, new}]}.
  Future<List<Map<String, dynamic>>> history(String slug, String id) async {
    final data = _map(await _get('/resources/$slug/$id/history'));
    final e = data['entries'];
    return e is List
        ? [
            for (final x in e)
              if (x is Map) Map<String, dynamic>.from(x)
          ]
        : const [];
  }

  /// Saved views of a resource: {views: [{id, name, shared, mine,
  /// can_delete, state}], can_share}.
  Future<Map<String, dynamic>> views(String slug) async =>
      _map(await _get('/resources/$slug/views'));

  Future<Map<String, dynamic>> saveView(
          String slug, String name, Map<String, dynamic> state,
          {bool shared = false}) async =>
      _map(await _send('POST', '/resources/$slug/views',
          {'name': name, 'shared': shared, 'state': state}));

  Future<void> deleteView(String slug, String id) =>
      _send('DELETE', '/resources/$slug/views/$id');

  /// Sends a CSV, TSV or .xlsx file to a resource's import endpoint. With
  /// [dryRun] the server only previews it: {headers, mapping, fields, rows,
  /// total, valid, invalid, problems}. Otherwise it imports the valid rows:
  /// {imported, failed, rows}. [mapping] maps a file header to a field name
  /// ('' skips the column).
  Future<Map<String, dynamic>> importFile(
      String slug, String filename, Uint8List bytes,
      {Map<String, String>? mapping, bool dryRun = false}) async {
    final req = http.MultipartRequest('POST', _uri('/resources/$slug/import'))
      ..headers.addAll({
        'Accept': 'application/json',
        if (isAuthenticated) 'Authorization': 'Bearer $token',
      })
      ..fields.addAll({
        if (dryRun) 'dry_run': '1',
        if (mapping != null) 'mapping': json.encode(mapping),
      })
      ..files
          .add(http.MultipartFile.fromBytes('file', bytes, filename: filename));
    return _map(_decode(await http.Response.fromStream(await _http.send(req))));
  }

  /// The user's notifications: {unread_count, notifications: [{id, type,
  /// title, body, created_at, read, action_label, action_url}]}.
  Future<Map<String, dynamic>> notifications() async =>
      _map(await _get('/notifications'));

  Future<void> markNotificationRead(String id) =>
      _send('POST', '/notifications/mark-read', {'id': id});

  Future<void> markAllNotificationsRead() =>
      _send('POST', '/notifications/mark-all-read');

  Future<void> clearNotifications() => _send('POST', '/notifications/clear');

  /// Downloads a file from an API path; returns its bytes and file name.
  Future<({Uint8List bytes, String name})> downloadFile(String path) async {
    final res = await _http.get(_uri(path), headers: _headers);
    if (res.statusCode >= 400) _decode(res); // throws ApiException
    final m = RegExp(r'filename="([^"]+)"')
        .firstMatch(res.headers['content-disposition'] ?? '');
    return (bytes: res.bodyBytes, name: m?.group(1) ?? 'download');
  }

  /// Starts a server-side export job (Panel.BackgroundJobs) of the list as
  /// filtered now. Returns the job ({id, status, progress, ...}).
  Future<Map<String, dynamic>> startExportJob(String slug, String format,
      {String search = '',
      String sort = '',
      bool descending = false,
      Map<String, String> filters = const {},
      String advancedSearch = '',
      List<String> ids = const [],
      List<String> columns = const []}) async {
    return _map(await _get('/resources/$slug/export', {
      'background': '1',
      'format': format,
      if (search.isNotEmpty) 'search': search,
      if (sort.isNotEmpty) 'sort': sort,
      if (sort.isNotEmpty) 'direction': descending ? 'desc' : 'asc',
      for (final f in filters.entries) 'filter_${f.key}': f.value,
      if (advancedSearch.isNotEmpty) 'advanced_search': advancedSearch,
      if (ids.isNotEmpty) 'ids': ids.join(','),
      if (columns.isNotEmpty) 'columns': columns.join(','),
    }));
  }

  /// A background job's state.
  Future<Map<String, dynamic>> job(String id) async =>
      _map(await _get('/jobs/$id'));

  /// The file a finished job produced.
  Future<Uint8List> jobFile(String id) async {
    final res = await _http.get(_uri('/jobs/$id/download'), headers: _headers);
    if (res.statusCode >= 400) _decode(res); // throws ApiException
    return res.bodyBytes;
  }

  /// Restores a soft-deleted record.
  Future<void> restoreRecord(String slug, String id) =>
      _send('POST', '/resources/$slug/restore', {'id': id});

  /// Deletes a soft-deleted record for good.
  Future<void> forceDelete(String slug, String id) =>
      _send('POST', '/resources/$slug/force-delete', {'id': id});

  /// Sets [field] to [value] on the records [ids] (Resource.BulkEditable).
  /// Returns {updated, failed, errors: {id: message}}.
  Future<Map<String, dynamic>> bulkUpdate(
          String slug, List<String> ids, String field, dynamic value) async =>
      _map(await _send('POST', '/resources/$slug/bulk-update',
          {'ids': ids, 'field': field, 'value': value}));

  /// A resource's kanban board: [{value, label, color, cards: [{id,
  /// title, description, version}], total}].
  Future<List<Map<String, dynamic>>> board(String slug,
      {String search = ''}) async {
    final data = _map(await _get(
        '/resources/$slug/board', {if (search.isNotEmpty) 'search': search}));
    final c = data['columns'];
    return c is List
        ? [
            for (final x in c)
              if (x is Map) Map<String, dynamic>.from(x)
          ]
        : const [];
  }

  /// Moves a card to another column. Returns the record's new version.
  Future<String?> moveCard(String slug, String id, String value,
      {String? version}) async {
    final data = _map(await _send('POST', '/resources/$slug/board/move', {
      'id': id,
      'value': value,
      if (version != null && version.isNotEmpty) '_version': version,
    }));
    return data['version']?.toString();
  }

  /// Saves a new row order: [ids] in order, the first at position [offset].
  Future<void> reorder(String slug, List<String> ids, {int offset = 0}) =>
      _send(
          'POST', '/resources/$slug/reorder', {'order': ids, 'offset': offset});

  /// Roles (Panel.Roles): {roles: [{name, label, permissions, users}],
  /// permissions: {name: label}, can_find_users}.
  Future<Map<String, dynamic>> roles() async => _map(await _get('/roles'));

  /// Creates a role, or updates [original] (renaming it to [name]).
  Future<Map<String, dynamic>> saveRole(
          {String? original,
          required String name,
          required String label,
          required List<String> permissions}) async =>
      _map(await _send(original == null ? 'POST' : 'PUT',
          original == null ? '/roles' : '/roles/$original', {
        'name': name,
        'label': label,
        'permissions': permissions,
      }));

  Future<void> deleteRole(String name) => _send('DELETE', '/roles/$name');

  /// Users with their role bindings: [{id, name, bindings: [{role, panel,
  /// tenant, can_remove}]}].
  Future<List<Map<String, dynamic>>> roleUsers(String query) async {
    final data =
        _map(await _get('/roles/users', {if (query.isNotEmpty) 'q': query}));
    final u = data['users'];
    return u is List
        ? [
            for (final x in u)
              if (x is Map) Map<String, dynamic>.from(x)
          ]
        : const [];
  }

  /// Gives [userId] the role in every panel x tenant listed ('*' = all).
  Future<void> grantRole(String userId, String role,
          {required List<String> panels, required List<String> tenants}) =>
      _send('POST', '/roles/bindings', {
        'user': userId,
        'role': role,
        'panels': panels,
        'tenants': tenants,
      });

  /// Takes one role binding away.
  Future<void> revokeRole(
          String userId, String role, String panel, String tenant) =>
      _send('DELETE', '/roles/bindings',
          {'user': userId, 'role': role, 'panel': panel, 'tenant': tenant});

  /// The user's tenants: {active_tenant, tenants: [{key, name, avatar}]}.
  Future<Map<String, dynamic>> tenants() async => _map(await _get('/tenants'));

  /// Switches tenant (checked by the panel); later requests use it.
  Future<void> switchTenant(String key) async {
    await _send('POST', '/tenants/switch', {'tenant': key});
    tenant = key;
  }

  Future<void> deleteMany(String slug, List<String> ids) => _send('POST',
      '/resources/$slug/bulk', {'bulk_action': 'delete_bulk', 'ids': ids});

  /// Saves one cell. With [version] (the row's version from row meta) the
  /// server refuses the change with 409 when the record changed meanwhile.
  /// Returns the record's new version, when the resource is versioned.
  Future<String?> updateCell(
      String slug, String id, String column, dynamic value,
      {String? version}) async {
    final data = _map(await _send('POST', '/resources/$slug/$id/cell', {
      'column': column,
      'value': value,
      if (version != null) '_version': version,
    }));
    return data['version']?.toString();
  }

  // ---------------------------------------------------------------- actions

  /// The input form of a row action.
  Future<FormPayload> actionForm(String slug, String id, String action) async {
    final data = _map(await _get('/resources/$slug/$id/actions/$action/form'));
    return FormPayload(
      mode: 'action',
      schema: (data['schema'] is List ? data['schema'] as List : const [])
          .whereType<Map>()
          .map((m) => Map<String, dynamic>.from(m))
          .toList(),
      state: const {},
      abilities: const {'run': true},
      optionLabels: const {},
    );
  }

  /// Runs a row action; returns the server's success message.
  Future<String> runAction(String slug, String id, String action,
      [Map<String, dynamic> data = const {}]) async {
    final out =
        _map(await _send('POST', '/resources/$slug/$id/actions/$action', data));
    return '${out['message'] ?? 'Done'}';
  }

  // ---------------------------------------------------------------- relations

  FormPayload _formPayload(Map<String, dynamic> data, bool edit) => FormPayload(
        mode: '${data['mode'] ?? (edit ? 'edit' : 'create')}',
        schema: (data['schema'] is List ? data['schema'] as List : const [])
            .whereType<Map>()
            .map((m) => Map<String, dynamic>.from(m))
            .toList(),
        state: _map(data['state']),
        abilities:
            _map(data['abilities']).map((k, v) => MapEntry(k, v == true)),
        optionLabels:
            _map(data['option_labels']).map((k, v) => MapEntry(k, '$v')),
      );

  /// The relation managers of a record, with their columns and abilities.
  Future<List<Map<String, dynamic>>> relations(String slug, String id) async {
    final data = _map(await _get('/resources/$slug/$id/relations'));
    return (data['relations'] is List ? data['relations'] as List : const [])
        .whereType<Map>()
        .map((m) => Map<String, dynamic>.from(m))
        .toList();
  }

  Future<ListPage> relationList(String slug, String id, String relation,
      {int page = 1, int perPage = 50, String search = ''}) async {
    final data = _map(await _get('/resources/$slug/$id/relations/$relation', {
      'page': '$page',
      'per_page': '$perPage',
      if (search.isNotEmpty) 'search': search,
    }));
    final rows = (data['data'] is List ? data['data'] as List : const [])
        .whereType<Map>()
        .map((m) => Map<String, dynamic>.from(m))
        .toList();
    return ListPage(
      rows: rows,
      total: (data['total'] as num?)?.toInt() ?? rows.length,
      page: page,
      perPage: perPage,
    );
  }

  Future<FormPayload> relationForm(String slug, String id, String relation,
      {String? childId}) async {
    final data = _map(await _get(
        '/resources/$slug/$id/relations/$relation/form',
        {if (childId != null) 'child_id': childId}));
    return _formPayload(data, childId != null);
  }

  Future<List<FieldOption>> relationOptions(String slug, String id,
      String relation, String field, String query) async {
    final data = _map(await _get(
        '/resources/$slug/$id/relations/$relation/form/options',
        {'field': field, 'q': query}));
    return FieldOption.listFrom(data['options']);
  }

  Future<void> relationCreate(
          String slug, String id, String relation, Map<String, dynamic> data) =>
      _send('POST', '/resources/$slug/$id/relations/$relation', data);

  Future<void> relationUpdate(String slug, String id, String relation,
          String childId, Map<String, dynamic> data) =>
      _send('PUT', '/resources/$slug/$id/relations/$relation/$childId', data);

  Future<void> relationDelete(
          String slug, String id, String relation, String childId) =>
      _send('DELETE', '/resources/$slug/$id/relations/$relation/$childId');

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

  /// Authenticated GET returning the raw response text (HTML, CSV, ...).
  Future<String> getText(String path, [Map<String, String>? query]) async {
    final res = await _http.get(_uri(path, query), headers: _headers);
    if (res.statusCode >= 200 && res.statusCode < 300) {
      return utf8.decode(res.bodyBytes);
    }
    _decode(res); // throws ApiException with the server's message
    return '';
  }

  /// Authenticated request with a JSON body against a custom panel endpoint.
  Future<dynamic> send(String method, String path, [Object? body]) =>
      _send(method, path, body);
}
