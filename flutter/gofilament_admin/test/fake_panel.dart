import 'dart:convert';

import 'package:gofilament_admin/gofilament_admin.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

/// An in-memory GoFilament panel API for widget tests.
class FakePanel {
  FakePanel();

  final List<http.Request> requests = [];
  final List<Map<String, dynamic>> items = [
    {
      'id': 1,
      'code': 'RING',
      'name': 'Rings',
      'price': 1500.5,
      'status': 'live',
      'is_active': true,
      'category_id': 2
    },
    {
      'id': 2,
      'code': 'CHAIN',
      'name': 'Chains',
      'price': 800,
      'status': 'draft',
      'category_id': 1
    },
    {
      'id': 3,
      'code': 'BANGLE',
      'name': 'Bangles',
      'price': 250,
      'status': 'live',
      'is_active': true
    },
  ];
  final List<Map<String, dynamic>> variants = [
    {'id': 10, 'item_id': 1, 'sku': 'RING-S', 'price': 1500},
    {'id': 11, 'item_id': 2, 'sku': 'CHAIN-L', 'price': 800},
  ];
  bool canCreate = true;
  bool canDelete = true;
  String? failNextSaveWith;

  static const brand = {
    'name': 'Acme Stores',
    'tagline': 'Retail made simple',
    'hide_powered_by': true
  };
  static const colors = {
    'primary': {
      '50': '#eef2ff',
      '400': '#818cf8',
      '500': '#6366f1',
      '600': '#4f46e5',
      '950': '#1e1b4b'
    },
    'accent': {'500': '#8b5cf6', '600': '#7c3aed'},
    'gray': {
      '50': '#f8fafc',
      '100': '#f1f5f9',
      '200': '#e2e8f0',
      '500': '#64748b',
      '700': '#334155',
      '900': '#0f172a',
      '950': '#020617'
    },
    'success': {'600': '#059669'},
    'danger': {'600': '#e11d48'},
  };

  AdminApi api() => AdminApi(
      baseUrl: 'http://acme.test/app',
      client: MockClient(_handle),
      store: MemorySessionStore());

  /// Extra items form fields, item columns and dashboard widgets (custom
  /// component tests).
  List<Map<String, dynamic>> extraFormFields = [];
  List<Map<String, dynamic>> extraColumns = [];
  List<Map<String, dynamic>> extraWidgets = [];
  List<Map<String, dynamic>> extraFilters = [];

  /// Sent as the list's "summaries" (totals over all results).
  Map<String, dynamic>? listSummaries;

  /// Extra keys of the items resource schema (advanced search, ...).
  Map<String, dynamic> itemsResourceExtras = {};

  /// Extra JSON routes keyed by "METHOD /path" (path without /app/api).
  Map<String, Object Function(http.Request req)> extraRoutes = {};

  bool globalSearch = false;
  bool board = false;

  /// Boot offers English and Arabic; Accept-Language "ar" answers RTL.
  bool bilingual = false;
  bool reorderable = false;
  bool notificationsOn = false;

  /// Fields sent as resource_meta.bulk_edit.
  List<String> bulkEdit = [];
  bool history = false;
  bool savedViews = false;
  bool backgroundJobs = false;

  /// Items accept imports; [importRequests] keeps each request's body.
  bool canImport = false;
  final List<String> importRequests = [];

  /// Account features sent in boot (auth.accounts); empty = none.
  Map<String, dynamic> accountFeatures = {};

  /// When true, signing in needs the code "123456" (two-factor step).
  bool twoFactorLogin = false;
  final List<String> resetRequests = [];

  /// Sign-ups received; /schema answers "email_unverified" while
  /// [emailUnverified] is true.
  final List<Map<String, dynamic>> registrations = [];
  bool emailUnverified = false;
  int verificationMails = 0;

  /// Versioned items: the form sends 'version' and PUT refuses a stale
  /// '_version' with 409 (edit-conflict tests).
  int? itemVersion;

  /// Bodies of inline cell saves.
  final List<Map<String, dynamic>> cellBodies = [];
  final List<Map<String, dynamic>> putBodies = [];

  /// Row meta merged into the list's row_meta by record id.
  Map<String, Map<String, dynamic>> extraRowMeta = {};

  /// Pages the server lists in `client_pages`.
  List<String> clientPages = const [];

  /// Answers one request (for tests that wrap the client).
  Future<http.Response> handle(http.Request req) => _handle(req);

  http.Response _json(Object body, [int status = 200]) =>
      http.Response(json.encode(body), status,
          headers: {'content-type': 'application/json; charset=utf-8'});

  Future<http.Response> _handle(http.Request req) async {
    requests.add(req);
    final path = req.url.path.replaceFirst('/app/api', '');
    final authed = req.headers['Authorization'] == 'Bearer token-1';
    final extra = extraRoutes['${req.method} $path'];
    if (extra != null && authed) return _json(extra(req));

    if (path == '/boot') {
      return _json({
        'brand': brand,
        'brand_name': brand['name'],
        'theme': {'colors': colors},
        if (bilingual) ...{
          'supported_locales': ['en', 'ar'],
          'locale_labels': {'en': 'English', 'ar': 'العربية'},
          'locale': req.headers['Accept-Language'] == 'ar' ? 'ar' : 'en',
          'direction': req.headers['Accept-Language'] == 'ar' ? 'rtl' : 'ltr',
        },
        'auth': {
          'login': true,
          if (accountFeatures.isNotEmpty) 'accounts': accountFeatures,
        }
      });
    }
    if (path == '/register') {
      final body = Map<String, dynamic>.from(json.decode(req.body) as Map);
      registrations.add(body);
      if (body['email'] == 'o@acme.test') {
        return _json({
          'error': 'An account with this email address already exists.',
          'errors': {
            'email': 'An account with this email address already exists.'
          }
        }, 422);
      }
      return _json({
        'token': 'token-1',
        'user': {'id': '9', 'login': body['email'], 'name': body['name']},
        'email_verified': !emailUnverified,
      }, 201);
    }
    if (path == '/forgot-password') {
      resetRequests.add('${(json.decode(req.body) as Map)['email']}');
      return _json(
          {'message': 'If an account uses that address, we sent a link.'});
    }
    final loginBody = path == '/login' || path == '/login/two-factor'
        ? json.decode(req.body) as Map
        : const {};
    if (path == '/login' &&
        twoFactorLogin &&
        loginBody['username'] == 'owner' &&
        loginBody['password'] == 'secret-pass') {
      return _json({'two_factor': true, 'challenge': 'ch-1'});
    }
    if (path == '/login/two-factor' &&
        (loginBody['challenge'] != 'ch-1' || loginBody['code'] != '123456')) {
      return _json({'error': 'The code is not valid.'}, 401);
    }
    if (path == '/login' || path == '/login/two-factor') {
      final body = path == '/login/two-factor'
          ? {'username': 'owner', 'password': 'secret-pass'}
          : loginBody;
      if (body['username'] == 'owner' && body['password'] == 'secret-pass') {
        return _json({
          'token': 'token-1',
          'user': {
            'id': '1',
            'login': 'owner',
            'name': 'Olivia Owner',
            'email': 'o@acme.test'
          }
        });
      }
      return _json({'error': 'Invalid credentials'}, 401);
    }
    if (!authed) return _json({'error': 'Unauthorized'}, 401);

    if (path == '/logout') return _json({'status': 'logged_out'});
    if (path == '/email/verification-notification') {
      verificationMails++;
      return _json({'email_verified': !emailUnverified});
    }
    if (path == '/schema' && emailUnverified) {
      return _json({
        'error': 'Verify your email address to continue.',
        'email_unverified': true
      }, 403);
    }
    if (path == '/schema') return _json(_schema());
    if (path == '/widgets') return _json({'widgets': _widgets()});

    if (path == '/resources/items/import') {
      final body = utf8.decode(req.bodyBytes, allowMalformed: true);
      importRequests.add(body);
      final mapped = body.contains('"Mail":"email"');
      final dry = body.contains('name="dry_run"');
      final invalid = mapped ? 1 : 2;
      if (!dry) {
        return _json({
          'imported': 2 - invalid + 1,
          'failed': invalid,
          'rows': [
            {
              'row': 3,
              'errors': {'email': 'Enter a valid email.'}
            }
          ],
        });
      }
      return _json({
        'headers': ['Name', 'Mail'],
        'mapping': {'Name': 'name', 'Mail': mapped ? 'email' : ''},
        'fields': [
          {'name': 'name', 'label': 'Name', 'required': true},
          {'name': 'email', 'label': 'Email', 'required': true},
        ],
        'rows': [
          {
            'row': 2,
            'values': {'name': 'Bea', if (mapped) 'email': 'bea@x.io'},
            if (!mapped) 'errors': {'email': 'Required.'}
          },
          {
            'row': 3,
            'values': {'name': 'Cal', if (mapped) 'email': 'nope'},
            'errors': {'email': mapped ? 'Enter a valid email.' : 'Required.'}
          },
        ],
        'total': 2,
        'valid': 2 - invalid,
        'invalid': invalid,
        'problems': [],
      });
    }
    if (path == '/resources/items' && req.method == 'GET') {
      return _json(_list(req.url.queryParameters));
    }
    if (path == '/resources/items/form') {
      return _form(req.url.queryParameters['id']);
    }
    if (path == '/resources/items/form/options') {
      final q = (req.url.queryParameters['q'] ?? '').toLowerCase();
      const all = [
        {'value': '40', 'label': 'Agasthi Traders'},
        {'value': '41', 'label': 'Bright Metals'},
      ];
      return _json({
        'options':
            all.where((o) => o['label']!.toLowerCase().contains(q)).toList()
      });
    }
    if (path == '/resources/items' && req.method == 'POST') {
      if (failNextSaveWith != null) {
        final message = failNextSaveWith!;
        failNextSaveWith = null;
        return _json({'error': message}, 409);
      }
      final data = Map<String, dynamic>.from(json.decode(req.body) as Map);
      data['id'] = items.length + 1;
      items.add(data);
      return _json({'success': true}, 201);
    }
    final act = RegExp(r'^/resources/items/(\d+)/actions/(\w+)(/form)?$')
        .firstMatch(path);
    if (act != null) {
      final item = items.firstWhere((i) => '${i['id']}' == act.group(1));
      if (act.group(3) != null) {
        return _json({
          'mode': 'action',
          'schema': [
            {
              'kind': 'field',
              'type': 'number',
              'name': 'percent',
              'label': 'Percent',
              'required': true
            },
          ],
        });
      }
      if (act.group(2) == 'archive') {
        item['archived'] = true;
        return _json({'success': true, 'message': 'Archived'});
      }
      final pct = num.parse('${(json.decode(req.body) as Map)['percent']}');
      item['price'] = (item['price'] as num) * (100 - pct) / 100;
      return _json({'success': true, 'message': 'Discount applied'});
    }
    if (path == '/resources/items/bulk') {
      final ids = (json.decode(req.body) as Map)['ids'] as List;
      items.removeWhere((i) => ids.contains('${i['id']}'));
      return _json({'success': true});
    }
    final rel = RegExp(
            r'^/resources/items/(\d+)/relations(/variants)?(/form)?(/(\d+))?$')
        .firstMatch(path);
    if (rel != null) {
      final parent = int.parse(rel.group(1)!);
      final mine = variants.where((v) => v['item_id'] == parent).toList();
      if (rel.group(2) == null) {
        return _json({
          'relations': [
            {
              'relationship': 'variants',
              'title': 'Variants',
              'has_form': true,
              'abilities': {'create': true, 'update': true, 'delete': true},
              'columns': [
                {'name': 'sku', 'label': 'SKU', 'type': 'text'},
                {'name': 'price', 'label': 'Price', 'type': 'money'},
              ],
            },
          ],
        });
      }
      const schema = [
        {
          'kind': 'field',
          'type': 'text',
          'name': 'sku',
          'label': 'SKU',
          'required': true
        },
        {'kind': 'field', 'type': 'number', 'name': 'price', 'label': 'Price'},
      ];
      if (rel.group(3) != null) {
        final childId = req.url.queryParameters['child_id'];
        if (childId == null) {
          return _json({
            'mode': 'create',
            'schema': schema,
            'state': {},
            'abilities': {'create': true}
          });
        }
        final child = mine.where((v) => '${v['id']}' == childId);
        if (child.isEmpty) return _json({'error': 'Record not found'}, 404);
        return _json({
          'mode': 'edit',
          'schema': schema,
          'state': child.first,
          'abilities': {'update': true, 'delete': true}
        });
      }
      final childId = rel.group(5);
      if (childId == null && req.method == 'GET') {
        return _json({'data': mine, 'total': mine.length});
      }
      if (childId == null && req.method == 'POST') {
        final data = Map<String, dynamic>.from(json.decode(req.body) as Map);
        variants.add({...data, 'id': 100 + variants.length, 'item_id': parent});
        return _json({'success': true}, 201);
      }
      final index = variants
          .indexWhere((v) => '${v['id']}' == childId && v['item_id'] == parent);
      if (index < 0) return _json({'error': 'Record not found'}, 404);
      if (req.method == 'DELETE') {
        variants.removeAt(index);
      } else {
        variants[index]
            .addAll(Map<String, dynamic>.from(json.decode(req.body) as Map));
      }
      return _json({'success': true});
    }
    final cell = RegExp(r'^/resources/items/(\d+)/cell$').firstMatch(path);
    if (cell != null) {
      final body = Map<String, dynamic>.from(json.decode(req.body) as Map);
      cellBodies.add({...body});
      final sent = body['_version'];
      if (itemVersion != null && sent != null && sent != '$itemVersion') {
        return _json({
          'error': 'This record was changed by someone else.',
          'conflict': true,
          'version': '$itemVersion',
        }, 409);
      }
      items.firstWhere(
              (i) => '${i['id']}' == cell.group(1))['${body['column']}'] =
          body['value'];
      if (itemVersion != null) itemVersion = itemVersion! + 1;
      return _json({
        'success': true,
        if (itemVersion != null) 'version': '$itemVersion',
      });
    }
    final one = RegExp(r'^/resources/items/(\d+)$').firstMatch(path);
    if (one != null) {
      final index = items.indexWhere((i) => '${i['id']}' == one.group(1));
      if (index < 0) return _json({'error': 'Record not found'}, 404);
      if (req.method == 'DELETE') {
        items.removeAt(index);
        return _json({'success': true});
      }
      if (req.method == 'PUT') {
        final body = Map<String, dynamic>.from(json.decode(req.body) as Map);
        putBodies.add({...body});
        final sent = body.remove('_version');
        if (itemVersion != null && sent != null && sent != '$itemVersion') {
          return _json({
            'error':
                'This record was changed by someone else after you opened it.',
            'version': '$itemVersion',
          }, 409);
        }
        items[index].addAll(body);
        if (itemVersion != null) itemVersion = itemVersion! + 1;
        return _json({'success': true});
      }
    }
    return _json({'error': 'Not found: ${req.method} $path'}, 404);
  }

  Map<String, dynamic> _list(Map<String, String> q) {
    var rows = [...items];
    final search = (q['search'] ?? '').toLowerCase();
    if (search.isNotEmpty) {
      rows = rows
          .where((r) => '${r['name']}'.toLowerCase().contains(search))
          .toList();
    }
    if (q['filter_status'] != null) {
      rows = rows.where((r) => r['status'] == q['filter_status']).toList();
    }
    if (q['sort'] != null) {
      rows.sort((a, b) => '${a[q['sort']]}'.compareTo('${b[q['sort']]}'));
      if (q['direction'] == 'desc') rows = rows.reversed.toList();
    }
    final perPage = int.parse(q['per_page'] ?? '25');
    final page = int.parse(q['page'] ?? '1');
    final start = (page - 1) * perPage;
    final page0 = start >= rows.length
        ? <Map<String, dynamic>>[]
        : rows.sublist(start, (start + perPage).clamp(0, rows.length));
    return {
      if (listSummaries != null) 'summaries': listSummaries,
      'row_meta': {
        for (final r in page0)
          '${r['id']}': {
            'update': r['id'] != 2,
            'delete': r['id'] != 2,
            if (itemVersion != null) 'version': '$itemVersion',
            ...?extraRowMeta['${r['id']}'],
            'actions': [
              if (r['archived'] != true) 'archive',
              'discount',
              if (r['id'] == 1) 'receipt',
            ],
            if (r['id'] == 1)
              'links': {'receipt': 'https://example.test/receipt/1'},
          },
      },
      'data': start >= rows.length
          ? []
          : rows.sublist(start, (start + perPage).clamp(0, rows.length)),
      'total': rows.length,
      'page': page,
      'per_page': perPage,
    };
  }

  http.Response _form(String? id) {
    final schema = [
      {
        'kind': 'section',
        'heading': 'Item',
        'description': 'Basic details',
        'children': [
          {
            'kind': 'grid',
            'columns': 2,
            'children': [
              {
                'kind': 'field',
                'type': 'text',
                'name': 'code',
                'label': 'Code',
                'required': true,
                'max_length': 10
              },
              {
                'kind': 'field',
                'type': 'text',
                'name': 'name',
                'label': 'Name',
                'required': true
              },
              {
                'kind': 'field',
                'type': 'number',
                'name': 'price',
                'label': 'Price',
                'prefix': 'Rs.',
                'default': 0
              },
              {
                'kind': 'field',
                'type': 'select',
                'name': 'category_id',
                'label': 'Category',
                'options': [
                  {'value': '1', 'label': 'Necklaces'},
                  {'value': '2', 'label': 'Jewellery'}
                ],
              },
              {
                'kind': 'field',
                'type': 'select',
                'name': 'supplier_id',
                'label': 'Supplier',
                'remote': true,
                'searchable': true
              },
            ],
          },
          {
            'kind': 'field',
            'type': 'textarea',
            'name': 'remarks',
            'label': 'Remarks',
            'rows': 2
          },
          {
            'kind': 'field',
            'type': 'toggle',
            'name': 'is_active',
            'label': 'Active',
            'default': true
          },
        ],
      },
      ...extraFormFields,
    ];
    if (id == null) {
      if (!canCreate) {
        return _json({'error': '403 Forbidden: Access denied'}, 403);
      }
      return _json({
        'mode': 'create',
        'schema': schema,
        'state': {},
        'abilities': {'create': true}
      });
    }
    final match = items.where((i) => '${i['id']}' == id);
    if (match.isEmpty) return _json({'error': 'Record not found'}, 404);
    return _json({
      'mode': 'edit',
      'schema': schema,
      'state': match.first,
      'option_labels': {'supplier_id': 'Agasthi Traders'},
      'abilities': {'view': true, 'update': true, 'delete': canDelete},
      if (itemVersion != null) 'version': '$itemVersion',
    });
  }

  List<Map<String, dynamic>> _widgets() => [
        {
          'type': 'stats_overview',
          'stats': [
            {
              'label': 'Sales Today',
              'value': '12,500.00',
              'description': '8 bills',
              'color': 'primary'
            },
            {
              'label': 'Low Stock',
              'value': '3',
              'description': 'Items needing attention',
              'color': 'warning'
            },
          ],
        },
        ...extraWidgets,
      ];

  Map<String, dynamic> _schema() => {
        'path': '/app',
        'global_search': {'enabled': globalSearch},
        'notifications': notificationsOn,
        'saved_views': savedViews,
        'background_jobs': backgroundJobs,
        'client_pages': [
          for (final id in clientPages) {'id': id, 'label': id}
        ],
        'brand': brand,
        'brand_name': brand['name'],
        'theme': {'colors': colors},
        'navigation': [
          {'name': 'Catalogue', 'items': []},
        ],
        'widgets': _widgets(),
        'export': {'max_rows': 1000},
        'currency': 'LKR',
        'resources': [
          {
            'slug': 'items',
            'model_name': 'Item',
            'plural_model_name': 'Items',
            'icon': 'cube',
            'group': 'Catalogue',
            'relations': [
              {'relationship': 'variants', 'title': 'Variants'},
            ],
            'per_page': 2,
            'per_page_options': [2, 25, 50],
            'columns': [
              {
                'name': 'code',
                'label': 'Code',
                'type': 'text',
                'sortable': true,
                'copyable': true
              },
              {
                'name': 'name',
                'label': 'Name',
                'type': 'text',
                'sortable': true,
                'weight': 'bold'
              },
              {'name': 'price', 'label': 'Price', 'type': 'money'},
              {
                'name': 'status',
                'label': 'Status',
                'type': 'badge',
                'colors': {'live': 'success', 'draft': 'warning'}
              },
              {
                'name': 'is_active',
                'label': 'Active',
                'type': 'toggle',
                'editable': true
              },
              {
                'name': 'secret',
                'label': 'Hidden',
                'type': 'text',
                'toggled_hidden_by_default': true
              },
              ...extraColumns,
            ],
            'filters': [
              {
                'name': 'status',
                'label': 'Status',
                'type': 'select',
                'column': 'status',
                'options': [
                  {'value': 'live', 'label': 'Live'},
                  {'value': 'draft', 'label': 'Draft'}
                ],
              },
              ...extraFilters,
            ],
            ...itemsResourceExtras,
          },
        ],
        'resource_meta': {
          'items': {
            'has_form': true,
            if (bulkEdit.isNotEmpty) 'bulk_edit': bulkEdit,
            if (board) 'board': {'field': 'status'},
            'abilities': {
              'create': canCreate,
              'delete_any': canDelete,
              if (history) 'history': true,
              if (canImport) 'import': true,
            },
            'table': {
              if (reorderable) 'reorderable': true,
              'actions': [
                {'name': 'view'},
                {'name': 'edit'},
                {'name': 'delete'},
                {
                  'name': 'archive',
                  'label': 'Archive',
                  'custom': true,
                  'requires_confirmation': true,
                  'modal_heading': 'Archive item?'
                },
                {
                  'name': 'discount',
                  'label': 'Apply discount',
                  'custom': true,
                  'has_form': true,
                  'modal_description': 'Reduce the price by a percentage.'
                },
                {
                  'name': 'receipt',
                  'label': 'Receipt',
                  'custom': true,
                  'link': true
                },
              ],
              'bulk_actions': [
                {'name': 'delete_bulk'}
              ],
              'header_actions': [],
            },
          },
        },
      };
}
