import 'dart:typed_data';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';

import 'format.dart';

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

/// Edits a FileUpload / ImageUpload field: files are sent to the panel's
/// upload endpoint as soon as they are chosen, and the field keeps the
/// values the server returns.
class FileFieldEditor extends StatefulWidget {
  const FileFieldEditor({
    super.key,
    required this.node,
    required this.value,
    required this.enabled,
    required this.required,
    required this.upload,
    required this.onChanged,
  });

  final Map<String, dynamic> node;

  /// A stored value (single file) or a list of them.
  final dynamic value;
  final bool enabled;
  final bool required;

  /// Uploads one file; returns `{value, url, name}`.
  final Future<Map<String, dynamic>> Function(String name, Uint8List bytes)
      upload;
  final ValueChanged<dynamic> onChanged;

  @override
  State<FileFieldEditor> createState() => _FileFieldEditorState();
}

class _FileFieldEditorState extends State<FileFieldEditor> {
  late final List<({String value, String name, String? url})> _files = [
    for (final v in widget.value is List
        ? widget.value as List
        : [
            if (widget.value != null && '${widget.value}'.isNotEmpty)
              widget.value
          ])
      (value: '$v', name: '$v'.split('/').last, url: null),
  ];
  bool _busy = false;
  String? _error;

  bool get _multiple => widget.node['multiple'] == true;
  int get _maxFiles => (widget.node['max_files'] as num?)?.toInt() ?? 0;

  List<String> get _extensions {
    final accept = widget.node['accept'];
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

  void _emit() => widget.onChanged(_multiple
      ? [for (final f in _files) f.value]
      : (_files.isEmpty ? null : _files.first.value));

  Future<void> _pick(FormFieldState<dynamic> state) async {
    final picked =
        await adminPickFiles(multiple: _multiple, extensions: _extensions);
    if (picked.isEmpty || !mounted) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      for (final p in picked) {
        final out = await widget.upload(p.name, p.bytes);
        final file = (
          value: '${out['value']}',
          name: '${out['name'] ?? p.name}',
          url: out['url'] as String?
        );
        if (!_multiple) _files.clear();
        _files.add(file);
        if (_maxFiles > 0 && _files.length >= _maxFiles) break;
      }
    } catch (e) {
      _error = '$e';
    }
    if (!mounted) return;
    setState(() => _busy = false);
    _emit();
    state.didChange(_files.length);
  }

  @override
  Widget build(BuildContext context) {
    final name = '${widget.node['name']}';
    final label = '${widget.node['label'] ?? humanize(name)}';
    final image = widget.node['type'] == 'image';
    final canAdd = widget.enabled &&
        (_multiple ? (_maxFiles == 0 || _files.length < _maxFiles) : true);
    return FormField<dynamic>(
      key: Key('field-$name'),
      validator: (_) =>
          widget.required && _files.isEmpty ? '$label is required' : null,
      builder: (state) => InputDecorator(
        decoration: InputDecoration(
          labelText: widget.required ? '$label *' : label,
          errorText: _error ?? state.errorText,
          enabled: widget.enabled,
        ),
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          for (var i = 0; i < _files.length; i++)
            ListTile(
              key: Key('$name-file-$i'),
              dense: true,
              contentPadding: EdgeInsets.zero,
              leading: image && _files[i].url != null
                  ? ClipRRect(
                      borderRadius: BorderRadius.circular(6),
                      child: Image.network(_files[i].url!,
                          width: 40,
                          height: 40,
                          fit: BoxFit.cover,
                          errorBuilder: (_, __, ___) =>
                              const Icon(Icons.image_outlined)))
                  : Icon(image
                      ? Icons.image_outlined
                      : Icons.insert_drive_file_outlined),
              title: Text(_files[i].name,
                  maxLines: 1, overflow: TextOverflow.ellipsis),
              trailing: widget.enabled
                  ? IconButton(
                      key: Key('$name-file-remove-$i'),
                      tooltip: 'Remove',
                      icon: const Icon(Icons.close, size: 18),
                      onPressed: () {
                        setState(() => _files.removeAt(i));
                        _emit();
                        state.didChange(_files.length);
                      },
                    )
                  : null,
            ),
          if (canAdd)
            TextButton.icon(
              key: Key('$name-pick'),
              onPressed: _busy ? null : () => _pick(state),
              icon: _busy
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2))
                  : const Icon(Icons.upload_file_outlined),
              label: Text(_files.isEmpty || _multiple
                  ? (image ? 'Choose image' : 'Choose file')
                  : 'Replace'),
            ),
        ]),
      ),
    );
  }
}
