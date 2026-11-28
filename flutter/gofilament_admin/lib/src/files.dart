import 'dart:async';

import 'package:crop_your_image/crop_your_image.dart';
import 'package:desktop_drop/desktop_drop.dart';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:image_picker/image_picker.dart';

import 'api.dart';
import 'format.dart';
import 'i18n.dart';
import 'links.dart';
import 'responsive.dart';

/// A file chosen by the user.
typedef PickedFile = ({String name, Uint8List bytes});

/// Lets the user choose files. Replace it in tests or to use another picker.
Future<List<PickedFile>> Function({bool multiple, List<String> extensions})
    adminPickFiles = _pickWithFilePicker;

Future<List<PickedFile>> _pickWithFilePicker(
    {bool multiple = false, List<String> extensions = const []}) async {
  final result = await FilePicker.platform.pickFiles(
    allowMultiple: multiple,
    withData: true,
    type: extensions.isEmpty ? FileType.any : FileType.custom,
    allowedExtensions: extensions.isEmpty ? null : extensions,
  );
  if (result == null) return const [];
  return [
    for (final f in result.files)
      if (f.bytes != null) (name: f.name, bytes: f.bytes!),
  ];
}

/// Takes a photo with the camera (phones). Replace it in tests.
Future<PickedFile?> Function() adminTakePhoto = _takePhoto;

Future<PickedFile?> _takePhoto() async {
  final x = await ImagePicker().pickImage(source: ImageSource.camera);
  if (x == null) return null;
  return (name: x.name, bytes: await x.readAsBytes());
}

/// Lets the user crop an image to [aspectRatio] (width / height) before
/// it is uploaded; returns the cropped bytes, or null when they cancel.
/// Replace it in tests.
Future<Uint8List?> Function(
        BuildContext context, Uint8List image, double aspectRatio)
    adminCropImage = _cropDialog;

Future<Uint8List?> _cropDialog(
    BuildContext context, Uint8List image, double aspectRatio) {
  final controller = CropController();
  var cropping = false;
  return showAdaptivePage<Uint8List>(
    context,
    maxWidth: 720,
    maxHeight: 720,
    builder: (ctx) => StatefulBuilder(
      builder: (ctx, setState) => Scaffold(
        appBar: AppBar(
          leading: const CloseButton(),
          title: Text(tr('Crop image')),
          actions: [
            TextButton(
              key: const Key('crop-apply'),
              onPressed: cropping
                  ? null
                  : () {
                      setState(() => cropping = true);
                      controller.crop();
                    },
              child: Text(tr('Use')),
            ),
          ],
        ),
        body: Crop(
          image: image,
          controller: controller,
          aspectRatio: aspectRatio,
          interactive: true,
          baseColor: Theme.of(ctx).colorScheme.surface,
          onCropped: (result) {
            if (result is CropSuccess) {
              Navigator.of(ctx).pop(result.croppedImage);
            } else {
              Navigator.of(ctx).pop();
            }
          },
          progressIndicator: const Center(child: CircularProgressIndicator()),
        ),
      ),
    ),
  );
}

/// Uploads one file; returns `{value, url, name}`.
typedef FileUploader = Future<Map<String, dynamic>> Function(
    String name, Uint8List bytes,
    {UploadProgress? onProgress, Future<void>? cancel});

/// One file of a file field: stored, uploading or failed.
class _Entry {
  _Entry({this.value, required this.name, this.url, this.bytes});

  String? value; // the field value once uploaded
  String name;
  String? url;
  Uint8List? bytes; // local bytes while uploading (image preview, retry)
  double? progress; // 0..1 while uploading
  String? error;
  Completer<void>? cancel;

  bool get uploading => progress != null;
  bool get done => value != null && !uploading && error == null;
}

bool get _dropSupported =>
    kIsWeb ||
    defaultTargetPlatform == TargetPlatform.linux ||
    defaultTargetPlatform == TargetPlatform.macOS ||
    defaultTargetPlatform == TargetPlatform.windows;

/// Width / height of a ratio such as "16:9", "4/3" or "1"; 0 when none.
double parseAspectRatio(String? s) {
  if (s == null || s.trim().isEmpty) return 0;
  final parts = s.replaceAll('/', ':').replaceAll('x', ':').split(':');
  if (parts.length == 1) return double.tryParse(parts.first) ?? 0;
  final w = double.tryParse(parts[0].trim()) ?? 0;
  final h = double.tryParse(parts[1].trim()) ?? 0;
  return w > 0 && h > 0 ? w / h : 0;
}

/// "2.4 MB" for a byte count.
String formatFileSize(int bytes) {
  if (bytes < 1024) return '$bytes B';
  if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(0)} KB';
  return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
}

bool _isImageName(String name) {
  final n = name.toLowerCase();
  return n.endsWith('.png') ||
      n.endsWith('.jpg') ||
      n.endsWith('.jpeg') ||
      n.endsWith('.gif') ||
      n.endsWith('.webp') ||
      n.endsWith('.bmp');
}

/// Renames [name] to match image bytes the cropper produced.
String _nameForBytes(String name, Uint8List bytes) {
  String? ext;
  if (bytes.length > 4 && bytes[0] == 0x89 && bytes[1] == 0x50) ext = 'png';
  if (bytes.length > 3 && bytes[0] == 0xFF && bytes[1] == 0xD8) ext = 'jpg';
  if (ext == null) return name;
  final dot = name.lastIndexOf('.');
  final base = dot > 0 ? name.substring(0, dot) : name;
  final had = dot > 0 ? name.substring(dot + 1).toLowerCase() : '';
  if (had == ext || (ext == 'jpg' && had == 'jpeg')) return name;
  return '$base.$ext';
}

/// The link that downloads [url] instead of showing it (golaravel's
/// storage server honours ?download=name; presigned S3 links are left as
/// they are, since extra parameters would break their signature).
String downloadLink(String url, String name) {
  if (url.contains('X-Amz-')) return url;
  return '$url${url.contains('?') ? '&' : '?'}download=${Uri.encodeQueryComponent(name)}';
}

/// Edits a FileUpload / ImageUpload field. Files are sent to the panel's
/// upload endpoint as soon as they are chosen (with progress, cancel and
/// retry); the field keeps the values the server returns, and the server
/// stores them for good when the record is saved.
class FileFieldEditor extends StatefulWidget {
  const FileFieldEditor({
    super.key,
    required this.node,
    required this.value,
    required this.enabled,
    required this.required,
    required this.upload,
    required this.onChanged,
    this.files = const [],
  });

  final Map<String, dynamic> node;

  /// A stored value (single file) or a list of them.
  final dynamic value;

  /// What the server knows about the stored values: links and names.
  final List<StoredFile> files;
  final bool enabled;
  final bool required;
  final FileUploader upload;
  final ValueChanged<dynamic> onChanged;

  @override
  State<FileFieldEditor> createState() => _FileFieldEditorState();
}

class _FileFieldEditorState extends State<FileFieldEditor> {
  late final List<_Entry> _entries = [
    for (final v in widget.value is List
        ? widget.value as List
        : [
            if (widget.value != null && '${widget.value}'.isNotEmpty)
              widget.value
          ])
      _stored('$v'),
  ];
  bool _dragging = false;
  String? _notice;
  FormFieldState<dynamic>? _field;

  _Entry _stored(String v) {
    for (final f in widget.files) {
      if (f.value == v) return _Entry(value: v, name: f.name, url: f.url);
    }
    final isLink = v.startsWith('http://') ||
        v.startsWith('https://') ||
        v.startsWith('/');
    return _Entry(value: v, name: v.split('/').last, url: isLink ? v : null);
  }

  Map<String, dynamic> get n => widget.node;

  /// The field name used in widget keys; rows of a repeater prefix it
  /// ("lines.0.photo").
  String get _name => '${n['key_prefix'] ?? ''}${n['name']}';
  bool get _multiple => n['multiple'] == true;
  bool get _image => n['type'] == 'image';
  bool get _avatar => n['avatar'] == true;
  int get _maxFiles => (n['max_files'] as num?)?.toInt() ?? 0;
  int get _minFiles => (n['min_files'] as num?)?.toInt() ?? 0;
  int get _maxKB => (n['max_size'] as num?)?.toInt() ?? 0;
  int get _minKB => (n['min_size'] as num?)?.toInt() ?? 0;
  bool get _reorderable => _multiple && n['reorderable'] == true;
  bool get _appendFiles => n['append_files'] == true;
  double get _ratio => parseAspectRatio(n['crop_aspect_ratio'] as String?) != 0
      ? parseAspectRatio(n['crop_aspect_ratio'] as String?)
      : (_avatar ? 1 : 0);
  bool get _canAdd =>
      widget.enabled &&
      (!_multiple || _maxFiles == 0 || _entries.length < _maxFiles);

  List<String> get _extensions {
    final accept = n['accept'];
    if (accept is! List) return const [];
    final out = <String>[];
    for (final a in accept.map((e) => '$e'.toLowerCase())) {
      if (a.startsWith('.')) out.add(a.substring(1));
      if (a == 'application/pdf') out.add('pdf');
      if (a == 'image/png') out.add('png');
      if (a == 'image/jpeg') out.addAll(['jpg', 'jpeg']);
      if (a == 'image/webp') out.add('webp');
      if (a == 'image/gif') out.add('gif');
      if (a == 'image/svg+xml') out.add('svg');
      if (a.endsWith('/*')) return const []; // let the server judge wildcards
    }
    return out.toSet().toList();
  }

  @override
  void dispose() {
    for (final e in _entries) {
      if (e.uploading && !(e.cancel?.isCompleted ?? true)) e.cancel!.complete();
    }
    super.dispose();
  }

  void _emit() {
    final values = [
      for (final e in _entries)
        if (e.done) e.value!
    ];
    widget
        .onChanged(_multiple ? values : (values.isEmpty ? null : values.first));
    _field?.didChange(values.length);
  }

  Future<void> _choose() async {
    final picked =
        await adminPickFiles(multiple: _multiple, extensions: _extensions);
    await _add(picked);
  }

  Future<void> _camera() async {
    final p = await adminTakePhoto();
    if (p != null) await _add([p]);
  }

  Future<void> _add(List<PickedFile> picked) async {
    if (picked.isEmpty || !mounted) return;
    setState(() => _notice = null);
    var files = picked;
    if (!_multiple) files = [files.last];
    if (_multiple && _maxFiles > 0) {
      final room = _maxFiles - _entries.length;
      if (files.length > room) {
        files = files.take(room < 0 ? 0 : room).toList();
        setState(() =>
            _notice = tr('Only {n} files are allowed.', {'n': '$_maxFiles'}));
      }
    }
    final added = <_Entry>[];
    for (var p in files) {
      final kb = p.bytes.length / 1024;
      String? error;
      if (_maxKB > 0 && kb > _maxKB) {
        error = tr('Larger than {n} KB', {'n': '$_maxKB'});
      } else if (_minKB > 0 && kb < _minKB) {
        error = tr('Smaller than {n} KB', {'n': '$_minKB'});
      }
      if (error == null &&
          _ratio > 0 &&
          _isImageName(p.name) &&
          !p.name.toLowerCase().endsWith('.gif') &&
          mounted) {
        final cropped = await adminCropImage(context, p.bytes, _ratio);
        if (cropped == null) continue; // cancelled
        p = (name: _nameForBytes(p.name, cropped), bytes: cropped);
      }
      final e = _Entry(name: p.name, bytes: p.bytes)..error = error;
      added.add(e);
    }
    if (!mounted || added.isEmpty) return;
    setState(() {
      if (!_multiple) {
        for (final old in _entries) {
          _cancel(old);
        }
        _entries
          ..clear()
          ..addAll(added);
      } else if (_appendFiles) {
        _entries.addAll(added);
      } else {
        _entries.insertAll(0, added);
      }
    });
    _emit();
    await Future.wait([
      for (final e in added)
        if (e.error == null) _upload(e)
    ]);
  }

  Future<void> _upload(_Entry e) async {
    final cancel = Completer<void>();
    setState(() {
      e
        ..progress = 0
        ..error = null
        ..cancel = cancel;
    });
    try {
      // the entry leaves at once even if the HTTP client is slow to abort
      final out = await Future.any([
        widget.upload(e.name, e.bytes!, onProgress: (sent, total) {
          if (!mounted || total <= 0 || cancel.isCompleted) return;
          setState(() => e.progress = (sent / total).clamp(0, 1));
        }, cancel: cancel.future),
        cancel.future.then<Map<String, dynamic>>(
            (_) => throw http.RequestAbortedException()),
      ]);
      if (!mounted) return;
      setState(() {
        e
          ..value = '${out['value']}'
          ..name = '${out['name'] ?? e.name}'
          ..url = out['url'] as String?
          ..progress = null;
      });
    } on http.RequestAbortedException {
      if (!mounted) return;
      setState(() => _entries.remove(e));
    } catch (err) {
      if (!mounted) return;
      setState(() {
        e
          ..progress = null
          ..error = '$err'.replaceFirst('Exception: ', '');
      });
    }
    _emit();
  }

  void _cancel(_Entry e) {
    if (e.uploading && !(e.cancel?.isCompleted ?? true)) e.cancel!.complete();
  }

  void _remove(_Entry e) {
    _cancel(e);
    setState(() => _entries.remove(e));
    _emit();
  }

  void _open(_Entry e) {
    if (e.url != null) openLink(e.url!);
  }

  void _download(_Entry e) {
    if (e.url != null) openLink(downloadLink(e.url!, e.name));
  }

  Future<void> _preview(_Entry e) async {
    final img = _thumb(e, fit: BoxFit.contain);
    if (img == null) return _open(e);
    await showDialog<void>(
      context: context,
      builder: (ctx) => Dialog(
        key: Key('$_name-preview'),
        clipBehavior: Clip.antiAlias,
        insetPadding: const EdgeInsets.all(16),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Row(children: [
            const SizedBox(width: 16),
            Expanded(
                child:
                    Text(e.name, maxLines: 1, overflow: TextOverflow.ellipsis)),
            if (e.url != null && n['downloadable'] == true)
              IconButton(
                  tooltip: tr('Download'),
                  icon: const Icon(Icons.download_outlined),
                  onPressed: () => _download(e)),
            if (e.url != null && n['openable'] == true)
              IconButton(
                  tooltip: tr('Open'),
                  icon: const Icon(Icons.open_in_new),
                  onPressed: () => _open(e)),
            const CloseButton(),
          ]),
          Flexible(
              child: InteractiveViewer(
                  maxScale: 6,
                  child: ConstrainedBox(
                      constraints: const BoxConstraints(minHeight: 120),
                      child: img))),
        ]),
      ),
    );
  }

  /// The picture of an image entry: local bytes or the stored link.
  Widget? _thumb(_Entry e, {BoxFit fit = BoxFit.cover}) {
    if (!_image && !_isImageName(e.name)) return null;
    Widget broken(BuildContext _, Object __, StackTrace? ___) =>
        const Center(child: Icon(Icons.broken_image_outlined));
    if (e.bytes != null) {
      return Image.memory(e.bytes!, fit: fit, errorBuilder: broken);
    }
    if (e.url != null) {
      return Image.network(e.url!, fit: fit, errorBuilder: broken);
    }
    return null;
  }

  String? _validate() {
    final label = '${n['label'] ?? humanize('${n['name']}')}';
    if (_entries.any((e) => e.uploading)) {
      return tr('Wait for the uploads to finish');
    }
    if (_entries.any((e) => e.error != null)) {
      return tr('Remove or retry the files that failed');
    }
    final count = _entries.where((e) => e.done).length;
    if (widget.required && count == 0) return '$label is required';
    if (_minFiles > 0 && count > 0 && count < _minFiles) {
      return tr('{label} needs at least {n} files',
          {'label': label, 'n': '$_minFiles'});
    }
    return null;
  }

  @override
  Widget build(BuildContext context) {
    final label = '${n['label'] ?? humanize('${n['name']}')}';
    return FormField<dynamic>(
      key: Key('field-$_name'),
      validator: (_) => _validate(),
      builder: (state) {
        _field = state;
        return InputDecorator(
          decoration: InputDecoration(
            labelText: widget.required ? '$label *' : label,
            helperText: n['helper_text'] as String?,
            errorText: state.errorText,
            enabled: widget.enabled,
            filled: false,
            border: InputBorder.none,
            enabledBorder: InputBorder.none,
            focusedBorder: InputBorder.none,
            disabledBorder: InputBorder.none,
            errorBorder: InputBorder.none,
            focusedErrorBorder: InputBorder.none,
            contentPadding: const EdgeInsets.only(top: 8, bottom: 4),
          ),
          child:
              Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
            if (_avatar)
              _avatarView(context)
            else if (_image && !_reorderable)
              _imageGrid(context)
            else
              _fileList(context),
            if (_notice != null)
              Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Text(_notice!,
                    style: Theme.of(context).textTheme.bodySmall),
              ),
          ]),
        );
      },
    );
  }

  // ---- add zone -------------------------------------------------------

  Widget _addZone(BuildContext context, {bool compact = false}) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final sizes = AdminSizes.of(context);
    final phone = sizes.isPhone;
    final drop = _dropSupported && !phone;
    final chooseLabel = _image
        ? (_multiple ? tr('Choose images') : tr('Choose image'))
        : (_multiple ? tr('Choose files') : tr('Choose file'));
    final hint = <String>[
      if (_maxKB > 0) tr('Up to {n}', {'n': formatFileSize(_maxKB * 1024)}),
      if (_multiple && _maxFiles > 0) tr('{n} files max', {'n': '$_maxFiles'}),
      if (_ratio > 0 && _image)
        tr('Cropped to {r}', {'r': '${n['crop_aspect_ratio'] ?? '1:1'}'}),
    ].join(' · ');

    Widget zone = Container(
      key: Key('$_name-drop-zone'),
      padding: EdgeInsets.symmetric(
          horizontal: 12, vertical: compact ? 10 : (phone ? 14 : 18)),
      decoration: BoxDecoration(
        color: _dragging
            ? scheme.primaryContainer.withValues(alpha: 0.5)
            : scheme.surfaceContainerHighest.withValues(alpha: 0.35),
        borderRadius: BorderRadius.circular(sizes.radius),
        border: Border.all(
            color: _dragging ? scheme.primary : scheme.outlineVariant,
            width: _dragging ? 2 : 1),
      ),
      child: Column(mainAxisSize: MainAxisSize.min, children: [
        if (!compact) ...[
          Icon(
              _image
                  ? Icons.add_photo_alternate_outlined
                  : Icons.upload_file_outlined,
              color: scheme.primary,
              size: phone ? 26 : 30),
          const SizedBox(height: 6),
          if (drop)
            Text(tr('Drag files here, or'),
                style: theme.textTheme.bodySmall
                    ?.copyWith(color: scheme.onSurfaceVariant)),
          const SizedBox(height: 4),
        ],
        Wrap(
          alignment: WrapAlignment.center,
          spacing: 8,
          runSpacing: 6,
          children: [
            FilledButton.tonalIcon(
              key: Key('$_name-pick'),
              onPressed: _choose,
              icon: Icon(
                  _image ? Icons.photo_library_outlined : Icons.attach_file,
                  size: 18),
              label: Text(_entries.isNotEmpty && !_multiple
                  ? tr('Replace')
                  : chooseLabel),
            ),
            if (_image && phone)
              OutlinedButton.icon(
                key: Key('$_name-camera'),
                onPressed: _camera,
                icon: const Icon(Icons.photo_camera_outlined, size: 18),
                label: Text(tr('Camera')),
              ),
          ],
        ),
        if (hint.isNotEmpty && !compact) ...[
          const SizedBox(height: 6),
          Text(hint,
              textAlign: TextAlign.center,
              style: theme.textTheme.bodySmall
                  ?.copyWith(color: scheme.onSurfaceVariant)),
        ],
      ]),
    );
    zone = _dropWrap(zone);
    return zone;
  }

  // ---- avatar ---------------------------------------------------------

  Widget _avatarView(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final e = _entries.isEmpty ? null : _entries.first;
    final size = AdminSizes.of(context).isPhone ? 88.0 : 96.0;
    final pic = e == null ? null : _thumb(e);
    return Row(children: [
      InkWell(
        key: Key('$_name-avatar'),
        customBorder: const CircleBorder(),
        onTap: e != null && pic != null
            ? () => _preview(e)
            : (widget.enabled ? _choose : null),
        child: SizedBox(
          width: size,
          height: size,
          child: Stack(fit: StackFit.expand, children: [
            ClipOval(
              child: ColoredBox(
                color: scheme.surfaceContainerHighest,
                child: pic ??
                    Icon(Icons.person_outline,
                        size: size * 0.45, color: scheme.onSurfaceVariant),
              ),
            ),
            if (e != null) _status(e, round: true),
          ]),
        ),
      ),
      const SizedBox(width: 16),
      if (widget.enabled)
        Expanded(
          child: Wrap(spacing: 8, runSpacing: 6, children: [
            FilledButton.tonalIcon(
              key: Key('$_name-pick'),
              onPressed: _choose,
              icon: const Icon(Icons.photo_library_outlined, size: 18),
              label: Text(e == null ? tr('Choose image') : tr('Change')),
            ),
            if (AdminSizes.of(context).isPhone)
              OutlinedButton.icon(
                key: Key('$_name-camera'),
                onPressed: _camera,
                icon: const Icon(Icons.photo_camera_outlined, size: 18),
                label: Text(tr('Camera')),
              ),
            if (e != null)
              TextButton(
                key: Key('$_name-file-remove-0'),
                onPressed: () => _remove(e),
                child: Text(tr('Remove')),
              ),
          ]),
        ),
    ]);
  }

  // ---- image grid -----------------------------------------------------

  Widget _imageGrid(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final sizes = AdminSizes.of(context);
    final tile = sizes.isPhone ? 92.0 : 108.0;
    return Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
      if (_entries.isEmpty && _canAdd) _addZone(context),
      if (_entries.isNotEmpty)
        Padding(
          padding: const EdgeInsets.only(bottom: 4),
          child: Wrap(spacing: 10, runSpacing: 10, children: [
            for (var i = 0; i < _entries.length; i++)
              SizedBox(
                key: Key('$_name-file-$i'),
                width: tile,
                height: tile,
                child: Stack(fit: StackFit.expand, children: [
                  ClipRRect(
                    borderRadius: BorderRadius.circular(sizes.radius),
                    child: Material(
                      color: scheme.surfaceContainerHighest,
                      child: InkWell(
                        onTap: () => _preview(_entries[i]),
                        child: _thumb(_entries[i]) ??
                            const Icon(Icons.image_outlined),
                      ),
                    ),
                  ),
                  _status(_entries[i], radius: sizes.radius),
                  if (widget.enabled)
                    Positioned(
                      top: 4,
                      right: 4,
                      child: _RoundIcon(
                        key: Key('$_name-file-remove-$i'),
                        icon: Icons.close,
                        tooltip: tr('Remove'),
                        onTap: () => _remove(_entries[i]),
                      ),
                    ),
                ]),
              ),
            if (_canAdd) ...[
              _dropWrap(_addTile(
                context,
                key: Key('$_name-pick'),
                size: tile,
                icon: Icons.add_photo_alternate_outlined,
                label: _multiple ? tr('Add') : tr('Replace'),
                onTap: _choose,
              )),
              if (sizes.isPhone)
                _addTile(
                  context,
                  key: Key('$_name-camera'),
                  size: tile,
                  icon: Icons.photo_camera_outlined,
                  label: tr('Camera'),
                  onTap: _camera,
                ),
            ],
          ]),
        ),
    ]);
  }

  /// A square button in the image grid.
  Widget _addTile(BuildContext context,
      {required Key key,
      required double size,
      required IconData icon,
      required String label,
      required VoidCallback onTap}) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final radius = AdminSizes.of(context).radius;
    return SizedBox(
      key: key,
      width: size,
      height: size,
      child: Material(
        color: _dragging
            ? scheme.primaryContainer.withValues(alpha: 0.5)
            : scheme.surfaceContainerHighest.withValues(alpha: 0.35),
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radius),
          side: BorderSide(
              color: _dragging ? scheme.primary : scheme.outlineVariant),
        ),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onTap,
          child: Column(mainAxisAlignment: MainAxisAlignment.center, children: [
            Icon(icon, color: scheme.primary),
            const SizedBox(height: 4),
            Text(label,
                style: theme.textTheme.labelMedium
                    ?.copyWith(color: scheme.primary)),
          ]),
        ),
      ),
    );
  }

  /// Accepts dropped files on desktop and web.
  Widget _dropWrap(Widget child) {
    if (!_dropSupported || AdminSizes.of(context).isPhone) return child;
    return DropTarget(
      onDragEntered: (_) => setState(() => _dragging = true),
      onDragExited: (_) => setState(() => _dragging = false),
      onDragDone: (d) async {
        setState(() => _dragging = false);
        final picked = <PickedFile>[];
        for (final f in d.files) {
          picked.add((name: f.name, bytes: await f.readAsBytes()));
        }
        await _add(picked);
      },
      child: child,
    );
  }

  /// Progress, failure and retry over a thumbnail.
  Widget _status(_Entry e, {bool round = false, double radius = 0}) {
    if (!e.uploading && e.error == null) return const SizedBox.shrink();
    final shape = round
        ? const ShapeDecoration(shape: CircleBorder(), color: Colors.black45)
        : ShapeDecoration(
            shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(radius)),
            color: e.error != null
                ? Colors.red.withValues(alpha: 0.55)
                : Colors.black45);
    return DecoratedBox(
      decoration: shape,
      child: Center(
        child: e.uploading
            ? Stack(alignment: Alignment.center, children: [
                SizedBox(
                  width: 34,
                  height: 34,
                  child: CircularProgressIndicator(
                      value: e.progress == 0 ? null : e.progress,
                      strokeWidth: 3,
                      color: Colors.white),
                ),
                IconButton(
                  key: Key('$_name-cancel'),
                  tooltip: tr('Cancel upload'),
                  iconSize: 16,
                  color: Colors.white,
                  icon: const Icon(Icons.close),
                  onPressed: () => _cancel(e),
                ),
              ])
            : IconButton(
                key: Key('$_name-retry'),
                tooltip: e.error,
                color: Colors.white,
                icon: const Icon(Icons.refresh),
                onPressed: e.bytes == null || !_uploadableAgain(e)
                    ? null
                    : () => _upload(e),
              ),
      ),
    );
  }

  /// Files refused on this device (too large) can't be retried.
  bool _uploadableAgain(_Entry e) {
    final kb = (e.bytes?.length ?? 0) / 1024;
    return !(_maxKB > 0 && kb > _maxKB) && !(_minKB > 0 && kb < _minKB);
  }

  // ---- file list ------------------------------------------------------

  Widget _fileTile(BuildContext context, int i) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final e = _entries[i];
    final thumb = _thumb(e);
    Widget? subtitle;
    if (e.uploading) {
      subtitle = Padding(
        padding: const EdgeInsets.only(top: 4),
        child: LinearProgressIndicator(
            value: e.progress == 0 ? null : e.progress, minHeight: 3),
      );
    } else if (e.error != null) {
      subtitle = Text(e.error!,
          maxLines: 2,
          overflow: TextOverflow.ellipsis,
          style: theme.textTheme.bodySmall?.copyWith(color: scheme.error));
    } else if (e.bytes != null) {
      subtitle = Text(formatFileSize(e.bytes!.length),
          style: theme.textTheme.bodySmall);
    }
    final actions = <Widget>[
      if (e.uploading)
        IconButton(
            key: Key('$_name-cancel-$i'),
            tooltip: tr('Cancel upload'),
            icon: const Icon(Icons.stop_circle_outlined, size: 20),
            onPressed: () => _cancel(e)),
      if (e.error != null && e.bytes != null && _uploadableAgain(e))
        IconButton(
            key: Key('$_name-retry-$i'),
            tooltip: tr('Retry'),
            icon: const Icon(Icons.refresh, size: 20),
            onPressed: () => _upload(e)),
      if (e.done && e.url != null && n['openable'] == true)
        IconButton(
            key: Key('$_name-open-$i'),
            tooltip: tr('Open'),
            icon: const Icon(Icons.open_in_new, size: 20),
            onPressed: () => _open(e)),
      if (e.done && e.url != null && n['downloadable'] == true)
        IconButton(
            key: Key('$_name-download-$i'),
            tooltip: tr('Download'),
            icon: const Icon(Icons.download_outlined, size: 20),
            onPressed: () => _download(e)),
      if (widget.enabled)
        IconButton(
            key: Key('$_name-file-remove-$i'),
            tooltip: tr('Remove'),
            icon: const Icon(Icons.close, size: 20),
            onPressed: () => _remove(e)),
      if (_reorderable && widget.enabled)
        ReorderableDragStartListener(
          index: i,
          child: const Padding(
            padding: EdgeInsets.symmetric(horizontal: 4),
            child: Icon(Icons.drag_handle),
          ),
        ),
    ];
    return Material(
      key: Key('$_name-file-$i'),
      color: Colors.transparent,
      child: ListTile(
        dense: true,
        contentPadding: const EdgeInsets.symmetric(horizontal: 4),
        leading: SizedBox(
          width: 40,
          height: 40,
          child: thumb != null
              ? ClipRRect(borderRadius: BorderRadius.circular(6), child: thumb)
              : Icon(_iconFor(e.name), color: scheme.onSurfaceVariant),
        ),
        title: Text(e.name, maxLines: 1, overflow: TextOverflow.ellipsis),
        subtitle: subtitle,
        onTap: thumb != null
            ? () => _preview(e)
            : (e.done && e.url != null && n['openable'] == true
                ? () => _open(e)
                : null),
        trailing: Row(mainAxisSize: MainAxisSize.min, children: actions),
      ),
    );
  }

  IconData _iconFor(String name) {
    final n = name.toLowerCase();
    if (n.endsWith('.pdf')) return Icons.picture_as_pdf_outlined;
    if (_isImageName(n)) return Icons.image_outlined;
    if (n.endsWith('.csv') || n.endsWith('.xlsx') || n.endsWith('.xls')) {
      return Icons.table_chart_outlined;
    }
    if (n.endsWith('.zip') || n.endsWith('.gz')) {
      return Icons.folder_zip_outlined;
    }
    if (n.endsWith('.mp4') || n.endsWith('.mov') || n.endsWith('.webm')) {
      return Icons.movie_outlined;
    }
    if (n.endsWith('.mp3') || n.endsWith('.wav')) {
      return Icons.audio_file_outlined;
    }
    return Icons.insert_drive_file_outlined;
  }

  Widget _fileList(BuildContext context) {
    final list = _reorderable
        ? ReorderableListView.builder(
            key: Key('$_name-files'),
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            buildDefaultDragHandles: false,
            itemCount: _entries.length,
            onReorderItem: (from, to) {
              setState(() => _entries.insert(to, _entries.removeAt(from)));
              _emit();
            },
            itemBuilder: (ctx, i) => _fileTile(ctx, i),
          )
        : Column(
            key: Key('$_name-files'),
            children: [
              for (var i = 0; i < _entries.length; i++) _fileTile(context, i)
            ],
          );
    return Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
      if (_entries.isNotEmpty) list,
      if (_canAdd)
        Padding(
          padding: EdgeInsets.only(top: _entries.isEmpty ? 0 : 8),
          child: _addZone(context, compact: _entries.isNotEmpty),
        ),
    ]);
  }
}

class _RoundIcon extends StatelessWidget {
  const _RoundIcon(
      {super.key,
      required this.icon,
      required this.tooltip,
      required this.onTap});

  final IconData icon;
  final String tooltip;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) => Tooltip(
        message: tooltip,
        child: Material(
          color: Colors.black54,
          shape: const CircleBorder(),
          child: InkWell(
            customBorder: const CircleBorder(),
            onTap: onTap,
            child: Padding(
              padding: const EdgeInsets.all(4),
              child: Icon(icon, size: 14, color: Colors.white),
            ),
          ),
        ),
      );
}
