import 'download_stub.dart' if (dart.library.js_interop) 'download_web.dart'
    as platform;

/// Opens a URL from a link action. Returns false when the platform cannot
/// open it (the list then copies the link instead).
typedef LinkOpener = Future<bool> Function(String url);

/// The opener used by link actions. Web opens a new tab; host apps on other
/// platforms (or tests) can replace it, e.g. with url_launcher.
LinkOpener openLink = platform.openUrl;
