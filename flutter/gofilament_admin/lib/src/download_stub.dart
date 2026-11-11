import 'dart:typed_data';

/// Non-web platforms: saving is left to the host app (share sheet, file picker).
Future<bool> saveFile(
        String filename, Uint8List bytes, String mimeType) async =>
    false;

/// Non-web platforms: opening links is left to the host app (see [openLink]).
Future<bool> openUrl(String url) async => false;
