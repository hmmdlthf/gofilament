// GENERATED CODE - DO NOT EDIT.
//
// Models of the 'admin' panel (/admin), written by gofilament's
// flutterclient.GenerateFromPanel. Change the panel in Go and regenerate;
// keep hand-written code in other files.

// ignore_for_file: type=lint

import 'package:gofilament_admin/gofilament_admin.dart';

/// A `docs` record (Go type `*tests.actDoc`).
class Doc {
  const Doc({
    this.id = 0,
    this.number = '',
    this.posted = false,
    this.total = 0.0,
    this.raw = const <String, dynamic>{},
  });

  factory Doc.fromJson(Map<String, dynamic> json) => Doc(
        id: GfJson.integer(json['id']) ?? 0,
        number: GfJson.string(json['number']) ?? '',
        posted: GfJson.boolean(json['posted']) ?? false,
        total: GfJson.decimal(json['total']) ?? 0.0,
        raw: json,
      );

  final int id;
  final String number;
  final bool posted;
  final double total;

  /// The JSON the record was read from, including fields not modelled here.
  final Map<String, dynamic> raw;

  Map<String, dynamic> toJson() => <String, dynamic>{
        ...raw,
        'id': id,
        'number': number,
        'posted': posted,
        'total': total,
      };
}

/// The input of the `discount` row action (Discount) of `docs`.
class DocDiscountInput {
  const DocDiscountInput({
    this.percent,
    this.extra = const <String, dynamic>{},
  });

  factory DocDiscountInput.fromJson(Map<String, dynamic> json) => DocDiscountInput(
        percent: GfJson.number(json['percent']),
      );

  /// Percent (required; number).
  final num? percent;

  /// Sent as well, after the fields above (overrides them).
  final Map<String, dynamic> extra;

  Map<String, dynamic> toJson() => <String, dynamic>{
        if (percent != null) 'percent': percent,
        ...extra,
      };
}

/// A `sales-orders` record (Go type `*tests.cgOrder`).
class SalesOrder {
  const SalesOrder({
    this.id = 0,
    this.number = '',
    this.paid = false,
    this.total = 0.0,
    this.note,
    this.tags,
    this.lines,
    this.ship,
    this.meta,
    this.extraValue,
    this.createdAt,
    this.deletedAt,
    this.raw = const <String, dynamic>{},
  });

  factory SalesOrder.fromJson(Map<String, dynamic> json) => SalesOrder(
        id: GfJson.integer(json['id']) ?? 0,
        number: GfJson.string(json['number']) ?? '',
        paid: GfJson.boolean(json['paid']) ?? false,
        total: GfJson.decimal(json['total']) ?? 0.0,
        note: GfJson.string(json['note']),
        tags: GfJson.list(json['tags'], (e) => GfJson.string(e) ?? ''),
        lines: GfJson.list(json['lines'], (e) => GfJson.object(e, CgLine.fromJson) ?? CgLine.fromJson(const <String, dynamic>{})),
        ship: GfJson.object(json['ship'], CgAddress.fromJson),
        meta: GfJson.map(json['meta']),
        extraValue: json['extra'],
        createdAt: GfJson.dateTime(json['created_at']),
        deletedAt: GfJson.dateTime(json['deleted_at']),
        raw: json,
      );

  final int id;
  final String number;
  final bool paid;
  final double total;
  final String? note;
  final List<String>? tags;
  final List<CgLine>? lines;
  final CgAddress? ship;
  final Map<String, dynamic>? meta;
  final dynamic extraValue;
  final DateTime? createdAt;
  final DateTime? deletedAt;

  /// The JSON the record was read from, including fields not modelled here.
  final Map<String, dynamic> raw;

  Map<String, dynamic> toJson() => <String, dynamic>{
        ...raw,
        'id': id,
        'number': number,
        'paid': paid,
        'total': total,
        'note': note,
        'tags': tags,
        'lines': lines?.map((e) => e.toJson()).toList(),
        'ship': ship?.toJson(),
        'meta': meta,
        'extra': extraValue,
        'created_at': createdAt?.toUtc().toIso8601String(),
        'deleted_at': deletedAt?.toUtc().toIso8601String(),
      };
}

/// Go type `tests.cgLine`.
class CgLine {
  const CgLine({
    this.id = 0,
    this.sku = '',
    this.qty = 0,
    this.price = 0.0,
    this.raw = const <String, dynamic>{},
  });

  factory CgLine.fromJson(Map<String, dynamic> json) => CgLine(
        id: GfJson.integer(json['id']) ?? 0,
        sku: GfJson.string(json['sku']) ?? '',
        qty: GfJson.integer(json['qty']) ?? 0,
        price: GfJson.decimal(json['price']) ?? 0.0,
        raw: json,
      );

  final int id;
  final String sku;
  final int qty;
  final double price;

  /// The JSON the record was read from, including fields not modelled here.
  final Map<String, dynamic> raw;

  Map<String, dynamic> toJson() => <String, dynamic>{
        ...raw,
        'id': id,
        'sku': sku,
        'qty': qty,
        'price': price,
      };
}

/// Go type `tests.cgAddress`.
class CgAddress {
  const CgAddress({
    this.city = '',
    this.country,
    this.raw = const <String, dynamic>{},
  });

  factory CgAddress.fromJson(Map<String, dynamic> json) => CgAddress(
        city: GfJson.string(json['city']) ?? '',
        country: GfJson.string(json['country']),
        raw: json,
      );

  final String city;
  final String? country;

  /// The JSON the record was read from, including fields not modelled here.
  final Map<String, dynamic> raw;

  Map<String, dynamic> toJson() => <String, dynamic>{
        ...raw,
        'city': city,
        'country': country,
      };
}

/// The values of the `sales-orders` form (create and edit).
///
/// Only non-null fields are sent; put explicit nulls and extra keys in [extra].
class SalesOrderInput {
  const SalesOrderInput({
    this.number,
    this.total,
    this.paid,
    this.status,
    this.customerId,
    this.dueOn,
    this.shippedAt,
    this.tags,
    this.meta,
    this.signature,
    this.lines,
    this.extra = const <String, dynamic>{},
  });

  factory SalesOrderInput.fromJson(Map<String, dynamic> json) => SalesOrderInput(
        number: GfJson.string(json['number']),
        total: GfJson.number(json['total']),
        paid: GfJson.boolean(json['paid']),
        status: GfJson.string(json['status']),
        customerId: GfJson.string(json['customer_id']),
        dueOn: GfJson.dateTime(json['due_on']),
        shippedAt: GfJson.dateTime(json['shipped_at']),
        tags: GfJson.list(json['tags'], (e) => GfJson.string(e) ?? ''),
        meta: GfJson.map(json['meta']),
        signature: GfJson.string(json['signature']),
        lines: GfJson.list(json['lines'], (e) => GfJson.object(e, SalesOrderInputLinesItem.fromJson) ?? SalesOrderInputLinesItem.fromJson(const <String, dynamic>{})),
      );

  /// Number (required).
  final String? number;
  /// Total (number).
  final num? total;
  /// Paid (toggle).
  final bool? paid;
  /// Status (select; one of `draft`, `sent`).
  final String? status;
  /// Customer Id (select).
  final String? customerId;
  /// Due On (date).
  final DateTime? dueOn;
  /// Shipped At (datetime).
  final DateTime? shippedAt;
  /// Tags (tags).
  final List<String>? tags;
  /// Meta (key_value).
  final Map<String, dynamic>? meta;
  /// Customer signature (custom).
  final String? signature;
  /// Lines.
  final List<SalesOrderInputLinesItem>? lines;

  /// Sent as well, after the fields above (overrides them).
  final Map<String, dynamic> extra;

  Map<String, dynamic> toJson() => <String, dynamic>{
        if (number != null) 'number': number,
        if (total != null) 'total': total,
        if (paid != null) 'paid': paid,
        if (status != null) 'status': status,
        if (customerId != null) 'customer_id': customerId,
        if (dueOn != null) 'due_on': GfJson.date(dueOn),
        if (shippedAt != null) 'shipped_at': shippedAt?.toUtc().toIso8601String(),
        if (tags != null) 'tags': tags,
        if (meta != null) 'meta': meta,
        if (signature != null) 'signature': signature,
        if (lines != null) 'lines': lines?.map((e) => e.toJson()).toList(),
        ...extra,
      };
}

/// One entry of [SalesOrderInput].lines.
class SalesOrderInputLinesItem {
  const SalesOrderInputLinesItem({
    this.sku,
    this.qty,
    this.extra = const <String, dynamic>{},
  });

  factory SalesOrderInputLinesItem.fromJson(Map<String, dynamic> json) => SalesOrderInputLinesItem(
        sku: GfJson.string(json['sku']),
        qty: GfJson.number(json['qty']),
      );

  /// Sku (required).
  final String? sku;
  /// Qty (number).
  final num? qty;

  /// Sent as well, after the fields above (overrides them).
  final Map<String, dynamic> extra;

  Map<String, dynamic> toJson() => <String, dynamic>{
        if (sku != null) 'sku': sku,
        if (qty != null) 'qty': qty,
        ...extra,
      };
}

/// The input of the `refund` row action (Refund) of `sales-orders`.
class SalesOrderRefundInput {
  const SalesOrderRefundInput({
    this.amount,
    this.reason,
    this.approval,
    this.extra = const <String, dynamic>{},
  });

  factory SalesOrderRefundInput.fromJson(Map<String, dynamic> json) => SalesOrderRefundInput(
        amount: GfJson.number(json['amount']),
        reason: GfJson.string(json['reason']),
        approval: GfJson.string(json['approval']),
      );

  /// Amount (required; number).
  final num? amount;
  /// Reason (textarea).
  final String? reason;
  /// Approval (required; custom).
  final String? approval;

  /// Sent as well, after the fields above (overrides them).
  final Map<String, dynamic> extra;

  Map<String, dynamic> toJson() => <String, dynamic>{
        if (amount != null) 'amount': amount,
        if (reason != null) 'reason': reason,
        if (approval != null) 'approval': approval,
        ...extra,
      };
}

/// The values of the `lines` relation form of `sales-orders`.
///
/// Only non-null fields are sent; put explicit nulls and extra keys in [extra].
class SalesOrderLineInput {
  const SalesOrderLineInput({
    this.sku,
    this.qty,
    this.extra = const <String, dynamic>{},
  });

  factory SalesOrderLineInput.fromJson(Map<String, dynamic> json) => SalesOrderLineInput(
        sku: GfJson.string(json['sku']),
        qty: GfJson.number(json['qty']),
      );

  /// Sku (required).
  final String? sku;
  /// Qty (number).
  final num? qty;

  /// Sent as well, after the fields above (overrides them).
  final Map<String, dynamic> extra;

  Map<String, dynamic> toJson() => <String, dynamic>{
        if (sku != null) 'sku': sku,
        if (qty != null) 'qty': qty,
        ...extra,
      };
}

/// Go type `tests.cgSummaryQuery`.
class CgSummaryQuery {
  const CgSummaryQuery({
    this.from = '',
    this.limit = 0,
    this.raw = const <String, dynamic>{},
  });

  factory CgSummaryQuery.fromJson(Map<String, dynamic> json) => CgSummaryQuery(
        from: GfJson.string(json['from']) ?? '',
        limit: GfJson.integer(json['limit']) ?? 0,
        raw: json,
      );

  final String from;
  final int limit;

  /// The JSON the record was read from, including fields not modelled here.
  final Map<String, dynamic> raw;

  Map<String, dynamic> toJson() => <String, dynamic>{
        ...raw,
        'from': from,
        'limit': limit,
      };
}

/// Go type `tests.cgSummary`.
class CgSummary {
  const CgSummary({
    this.till = '',
    this.count = 0,
    this.total = 0.0,
    this.recent,
    this.raw = const <String, dynamic>{},
  });

  factory CgSummary.fromJson(Map<String, dynamic> json) => CgSummary(
        till: GfJson.string(json['till']) ?? '',
        count: GfJson.integer(json['count']) ?? 0,
        total: GfJson.decimal(json['total']) ?? 0.0,
        recent: GfJson.list(json['recent'], (e) => GfJson.object(e, CgLine.fromJson) ?? CgLine.fromJson(const <String, dynamic>{})),
        raw: json,
      );

  final String till;
  final int count;
  final double total;
  final List<CgLine>? recent;

  /// The JSON the record was read from, including fields not modelled here.
  final Map<String, dynamic> raw;

  Map<String, dynamic> toJson() => <String, dynamic>{
        ...raw,
        'till': till,
        'count': count,
        'total': total,
        'recent': recent?.map((e) => e.toJson()).toList(),
      };
}

/// Go type `tests.cgCloseTill`.
class CgCloseTill {
  const CgCloseTill({
    this.counted = 0.0,
    this.note,
    this.raw = const <String, dynamic>{},
  });

  factory CgCloseTill.fromJson(Map<String, dynamic> json) => CgCloseTill(
        counted: GfJson.decimal(json['counted']) ?? 0.0,
        note: GfJson.string(json['note']),
        raw: json,
      );

  final double counted;
  final String? note;

  /// The JSON the record was read from, including fields not modelled here.
  final Map<String, dynamic> raw;

  Map<String, dynamic> toJson() => <String, dynamic>{
        ...raw,
        'counted': counted,
        'note': note,
      };
}

/// The props of the `signature_pad` field component, inferred from the values set in Go.
class SignaturePadFieldProps {
  const SignaturePadFieldProps({
    this.height = 0,
    this.penColor = '',
    this.clearable,
    this.raw = const <String, dynamic>{},
  });

  factory SignaturePadFieldProps.fromJson(Map<String, dynamic> json) => SignaturePadFieldProps(
        height: GfJson.integer(json['height']) ?? 0,
        penColor: GfJson.string(json['pen_color']) ?? '',
        clearable: GfJson.boolean(json['clearable']),
        raw: json,
      );

  final int height;
  final String penColor;
  final bool? clearable;

  /// The JSON the record was read from, including fields not modelled here.
  final Map<String, dynamic> raw;

  Map<String, dynamic> toJson() => <String, dynamic>{
        ...raw,
        'height': height,
        'pen_color': penColor,
        'clearable': clearable,
      };
}

/// The props of the `paid_badge` cell component, inferred from the values set in Go.
class PaidBadgeCellProps {
  const PaidBadgeCellProps({
    this.tone = '',
    this.raw = const <String, dynamic>{},
  });

  factory PaidBadgeCellProps.fromJson(Map<String, dynamic> json) => PaidBadgeCellProps(
        tone: GfJson.string(json['tone']) ?? '',
        raw: json,
      );

  final String tone;

  /// The JSON the record was read from, including fields not modelled here.
  final Map<String, dynamic> raw;

  Map<String, dynamic> toJson() => <String, dynamic>{
        ...raw,
        'tone': tone,
      };
}

/// The props of the `sales_total` widget component, inferred from the values set in Go.
class SalesTotalWidgetProps {
  const SalesTotalWidgetProps({
    this.currency = '',
    this.raw = const <String, dynamic>{},
  });

  factory SalesTotalWidgetProps.fromJson(Map<String, dynamic> json) => SalesTotalWidgetProps(
        currency: GfJson.string(json['currency']) ?? '',
        raw: json,
      );

  final String currency;

  /// The JSON the record was read from, including fields not modelled here.
  final Map<String, dynamic> raw;

  Map<String, dynamic> toJson() => <String, dynamic>{
        ...raw,
        'currency': currency,
      };
}
