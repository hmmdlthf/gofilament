import 'dart:convert';
import 'dart:typed_data';

import 'package:archive/archive.dart';

import 'download_stub.dart' if (dart.library.js_interop) 'download_web.dart'
    as platform;

/// Builds export files in the client. Nothing is generated on the server: it
/// only pages the rows, which keeps export cost off the API.
class TableExport {
  TableExport({required this.headers, required this.rows});

  final List<String> headers;
  final List<List<String>> rows;

  /// RFC 4180 CSV with a UTF-8 byte order mark so Excel opens it correctly.
  Uint8List toCsv() {
    final buf = StringBuffer();
    void line(List<String> cells) => buf.writeln(cells.map(_csvCell).join(','));
    line(headers);
    rows.forEach(line);
    return Uint8List.fromList(
        [0xEF, 0xBB, 0xBF, ...utf8.encode(buf.toString())]);
  }

  static String _csvCell(String v) {
    // neutralise spreadsheet formula injection
    final safe = v.isNotEmpty && '=+-@\t\r'.contains(v[0]) ? "'$v" : v;
    if (safe.contains(',') ||
        safe.contains('"') ||
        safe.contains('\n') ||
        safe.contains('\r')) {
      return '"${safe.replaceAll('"', '""')}"';
    }
    return safe;
  }

  /// Minimal single-sheet .xlsx (inline strings, numbers stored as numbers).
  Uint8List toXlsx({String sheetName = 'Export'}) {
    final sheet = StringBuffer()
      ..write('<?xml version="1.0" encoding="UTF-8" standalone="yes"?>')
      ..write(
          '<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>');
    var r = 0;
    void row(List<String> cells, {bool header = false}) {
      r++;
      sheet.write('<row r="$r">');
      for (var c = 0; c < cells.length; c++) {
        final ref = '${_column(c)}$r';
        final v = cells[c];
        final isNumber = !header && _numeric.hasMatch(v) && v.length < 16;
        if (isNumber) {
          sheet.write('<c r="$ref"><v>$v</v></c>');
        } else {
          sheet.write(
              '<c r="$ref" t="inlineStr"${header ? ' s="1"' : ''}><is><t xml:space="preserve">${_xml(v)}</t></is></c>');
        }
      }
      sheet.write('</row>');
    }

    row(headers, header: true);
    for (final cells in rows) {
      row(cells);
    }
    sheet.write('</sheetData></worksheet>');

    final files = <String, String>{
      '[Content_Types].xml': '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
          '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">'
          '<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>'
          '<Default Extension="xml" ContentType="application/xml"/>'
          '<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>'
          '<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>'
          '<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>'
          '</Types>',
      '_rels/.rels': '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
          '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
          '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>'
          '</Relationships>',
      'xl/workbook.xml': '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
          '<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" '
          'xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">'
          '<sheets><sheet name="${_xml(sheetName)}" sheetId="1" r:id="rId1"/></sheets></workbook>',
      'xl/_rels/workbook.xml.rels':
          '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
              '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">'
              '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>'
              '<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>'
              '</Relationships>',
      'xl/styles.xml': '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>'
          '<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">'
          '<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts>'
          '<fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills>'
          '<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>'
          '<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>'
          '<cellXfs count="2"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>'
          '<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/></cellXfs>'
          '</styleSheet>',
      'xl/worksheets/sheet1.xml': sheet.toString(),
    };

    final archive = Archive();
    files.forEach((name, content) {
      final bytes = utf8.encode(content);
      archive.addFile(ArchiveFile.bytes(name, bytes));
    });
    return Uint8List.fromList(ZipEncoder().encode(archive));
  }

  static final _numeric = RegExp(r'^-?\d+(\.\d+)?$');

  static String _column(int index) {
    var n = index, s = '';
    do {
      s = String.fromCharCode(65 + n % 26) + s;
      n = n ~/ 26 - 1;
    } while (n >= 0);
    return s;
  }

  static String _xml(String v) => v
      .replaceAll('&', '&amp;')
      .replaceAll('<', '&lt;')
      .replaceAll('>', '&gt;')
      .replaceAll('"', '&quot;')
      .replaceAll(RegExp(r'[\x00-\x08\x0B\x0C\x0E-\x1F]'), '');
}

/// Hands a generated file to the user (browser download on web). Returns false
/// when the platform has no way to save it.
Future<bool> saveFile(String filename, Uint8List bytes, String mimeType) =>
    platform.saveFile(filename, bytes, mimeType);
