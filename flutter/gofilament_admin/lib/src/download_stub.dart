import 'dart:io';
import 'dart:typed_data';

import 'package:file_picker/file_picker.dart';

/// Native platforms: asks where to save (a save dialog on desktop, the
/// system picker on mobile) and writes the file. Returns false when the user
/// cancels or the platform cannot save.
Future<bool> saveFile(String filename, Uint8List bytes, String mimeType) async {
  try {
    final path =
        await FilePicker.platform.saveFile(fileName: filename, bytes: bytes);
    if (path == null) return false;
    // Desktop pickers only return the chosen path.
    final file = File(path);
    if (!await file.exists() || await file.length() != bytes.length) {
      await file.writeAsBytes(bytes, flush: true);
    }
    return true;
  } catch (_) {
    return false;
  }
}

/// Non-web platforms: opening links is left to the host app (see [openLink]).
Future<bool> openUrl(String url) async => false;
