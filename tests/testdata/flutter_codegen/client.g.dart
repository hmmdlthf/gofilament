// GENERATED CODE - DO NOT EDIT.
//
// Typed API client of the 'admin' panel (/admin), written by gofilament's
// flutterclient.GenerateFromPanel. Change the panel in Go and regenerate;
// keep hand-written code in other files.

// ignore_for_file: type=lint

import 'package:gofilament_admin/gofilament_admin.dart';

import 'models.g.dart';

/// Typed client of the 'admin' panel API. Every call goes through [api], so it
/// uses the signed-in session; the server checks permissions on each request.
///
/// In a custom page: `final client = PanelClient(admin.api);`
class PanelClient {
  PanelClient(this.api);

  final AdminApi api;

  /// The `docs` resource (Docs).
  late final DocsClient docs = DocsClient(api);

  /// The `sales-orders` resource (Sales Orders).
  late final SalesOrdersClient salesOrders = SalesOrdersClient(api);

  /// Totals of one till since [cgSummaryQuery.from].
  ///
  /// `GET /pos/tills/:till/summary`
  Future<CgSummary> salesSummary({required String till, CgSummaryQuery? request}) async {
    return GfJson.object(await api.get('/pos/tills/${Uri.encodeComponent(till)}/summary', GfJson.query(request?.toJson())), CgSummary.fromJson) ?? CgSummary.fromJson(const <String, dynamic>{});
  }

  /// Calls the `closeTill` endpoint.
  ///
  /// `POST /pos/tills/:till/close`
  Future<void> closeTill({required String till, required CgCloseTill request}) async {
    await api.send('POST', '/pos/tills/${Uri.encodeComponent(till)}/close', request.toJson());
  }
}

/// Typed calls for the `docs` resource.
class DocsClient {
  const DocsClient(this.api);

  final AdminApi api;

  static const String slug = 'docs';

  /// Table columns, in display order.
  static const List<String> columns = <String>['number', 'total'];

  /// One page of records with the per-row abilities the server granted.
  Future<TypedPage<Doc>> list({
    int page = 1,
    int perPage = 25,
    String search = '',
    String sort = '',
    bool descending = false,
    Map<String, String> filters = const {},
  }) async =>
      TypedPage.from(
          await api.list(slug,
              page: page,
              perPage: perPage,
              search: search,
              sort: sort,
              descending: descending,
              filters: filters),
          Doc.fromJson);

  Future<Doc> find(String id) async => Doc.fromJson(await api.find(slug, id));

  Future<void> delete(String id) => api.delete(slug, id);

  Future<void> deleteMany(List<String> ids) => api.deleteMany(slug, ids);

  /// Runs the `post` row action (Post); returns the server's message.
  Future<String> runPost(String id) => api.runAction(slug, id, 'post');

  /// Runs the `discount` row action (Discount); returns the server's message.
  Future<String> runDiscount(String id, DocDiscountInput input) =>
      api.runAction(slug, id, 'discount', input.toJson());
}

/// Typed calls for the `sales-orders` resource.
class SalesOrdersClient {
  const SalesOrdersClient(this.api);

  final AdminApi api;

  static const String slug = 'sales-orders';

  /// Table columns, in display order.
  static const List<String> columns = <String>['number', 'total', 'paid'];

  /// One page of records with the per-row abilities the server granted.
  Future<TypedPage<SalesOrder>> list({
    int page = 1,
    int perPage = 25,
    String search = '',
    String sort = '',
    bool descending = false,
    Map<String, String> filters = const {},
  }) async =>
      TypedPage.from(
          await api.list(slug,
              page: page,
              perPage: perPage,
              search: search,
              sort: sort,
              descending: descending,
              filters: filters),
          SalesOrder.fromJson);

  Future<SalesOrder> find(String id) async => SalesOrder.fromJson(await api.find(slug, id));

  Future<void> create(SalesOrderInput input) => api.create(slug, input.toJson());

  Future<void> update(String id, SalesOrderInput input) =>
      api.update(slug, id, input.toJson());

  Future<void> delete(String id) => api.delete(slug, id);

  Future<void> deleteMany(List<String> ids) => api.deleteMany(slug, ids);

  /// Runs the `mark_paid` row action (Mark paid); returns the server's message.
  Future<String> runMarkPaid(String id) => api.runAction(slug, id, 'mark_paid');

  /// Runs the `refund` row action (Refund); returns the server's message.
  Future<String> runRefund(String id, SalesOrderRefundInput input) =>
      api.runAction(slug, id, 'refund', input.toJson());

  /// The `lines` relation (Order lines) of record [id].
  SalesOrdersLinesClient lines(String id) => SalesOrdersLinesClient(api, id);
}

/// Typed calls for the `lines` relation of one `sales-orders` record.
class SalesOrdersLinesClient {
  const SalesOrdersLinesClient(this.api, this.parentId);

  final AdminApi api;
  final String parentId;

  static const String relationship = 'lines';

  Future<TypedPage<CgLine>> list(
          {int page = 1, int perPage = 50, String search = ''}) async =>
      TypedPage.from(
          await api.relationList(SalesOrdersClient.slug, parentId, relationship,
              page: page, perPage: perPage, search: search),
          CgLine.fromJson);

  Future<void> create(SalesOrderLineInput input) =>
      api.relationCreate(SalesOrdersClient.slug, parentId, relationship, input.toJson());

  Future<void> update(String childId, SalesOrderLineInput input) => api.relationUpdate(
      SalesOrdersClient.slug, parentId, relationship, childId, input.toJson());

  Future<void> delete(String childId) =>
      api.relationDelete(SalesOrdersClient.slug, parentId, relationship, childId);
}
