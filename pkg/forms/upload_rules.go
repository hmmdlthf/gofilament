package forms

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
)

// GetDisk returns the storage disk uploads of this field go to ("" when
// none is set).
func (f *FileUploadField) GetDisk() string { return f.disk }

// GetDirectory returns the directory uploads are stored in.
func (f *FileUploadField) GetDirectory() string { return f.directory }

// PreservesFilenames reports whether stored files keep their original name.
func (f *FileUploadField) PreservesFilenames() bool { return f.preserveFilenames }

// CheckUpload validates one uploaded file against the field: its size
// (MaxSize, in KB), and its type (AcceptedFileTypes: MIME types such as
// "application/pdf", wildcards such as "image/*", or extensions such as
// ".pdf"), judged from contentType (sniffed from the bytes by the caller) and
// the file name. Image fields without accepted types take image/* only.
// It returns "" when the file is acceptable.
func (f *FileUploadField) CheckUpload(filename, contentType string, size int64) string {
	label := f.displayName()
	if f.maxSize > 0 && size > f.maxSize*1024 {
		return fmt.Sprintf("%s may not be larger than %d KB.", label, f.maxSize)
	}
	if f.minSize > 0 && size < f.minSize*1024 {
		return fmt.Sprintf("%s must be at least %d KB.", label, f.minSize)
	}
	accepted := f.accepted
	if len(accepted) == 0 && (f.isImage || f.isAvatar) {
		accepted = []string{"image/*"}
	}
	if len(accepted) == 0 {
		return ""
	}
	ext := strings.ToLower(filepath.Ext(filename))
	extType := mime.TypeByExtension(ext)
	if i := strings.IndexByte(extType, ';'); i >= 0 {
		extType = extType[:i]
	}
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = contentType[:i]
	}
	for _, a := range accepted {
		a = strings.ToLower(strings.TrimSpace(a))
		switch {
		case strings.HasPrefix(a, "."):
			if ext == a {
				return ""
			}
		case strings.HasSuffix(a, "/*"):
			prefix := strings.TrimSuffix(a, "*")
			// both the bytes and the name must agree on the kind
			if strings.HasPrefix(contentType, prefix) && (extType == "" || strings.HasPrefix(extType, prefix)) {
				return ""
			}
		default:
			if contentType == a {
				return ""
			}
			// Types the bytes reliably reveal must match them; others (svg,
			// csv, office files...) are judged by their extension.
			if !sniffable[a] && extType == a {
				return ""
			}
		}
	}
	return label + " must be a file of type " + strings.Join(accepted, ", ") + "."
}

// sniffable are the MIME types http.DetectContentType recognises from the
// first bytes of a file.
var sniffable = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true,
	"image/bmp": true, "application/pdf": true, "application/zip": true,
	"application/x-gzip": true, "audio/mpeg": true, "video/mp4": true, "video/webm": true,
}

// Visibility chooses where files are stored by who may open them, as in
// Filament: "private" stores them on the private "local" disk (shown
// through signed links that expire) and "public" on the "public" disk.
// A disk set with Disk wins.
func (f *FileUploadField) Visibility(v string) *FileUploadField {
	if f.diskSet {
		return f
	}
	switch v {
	case "private":
		f.disk = "local"
	case "public":
		f.disk = "public"
	}
	return f
}

// MinSize sets the smallest accepted file, in KB.
func (f *FileUploadField) MinSize(kb int64) *FileUploadField {
	f.minSize = kb
	return f
}

// MinFiles sets how many files a Multiple field needs at least.
func (f *FileUploadField) MinFiles(n int) *FileUploadField {
	f.minFiles = n
	return f
}

// StoreFileNamesIn keeps the files' original names in another field of
// the record: a string for a single file, or a map of stored path to
// name for Multiple. Files are still stored under random names.
func (f *FileUploadField) StoreFileNamesIn(field string) *FileUploadField {
	f.fileNamesIn = field
	return f
}

// GetUploadedFileNameUsing names stored files: fn gets the client's file
// name and returns the name to store under (inside Directory). Path
// separators and ".." are removed from the result.
func (f *FileUploadField) GetUploadedFileNameUsing(fn func(original string) string) *FileUploadField {
	f.fileNameUsing = fn
	return f
}

// Downloadable shows a download button for stored files.
func (f *FileUploadField) Downloadable(enabled ...bool) *FileUploadField {
	f.downloadable = len(enabled) == 0 || enabled[0]
	return f
}

// Openable shows a button that opens stored files in a new tab.
func (f *FileUploadField) Openable(enabled ...bool) *FileUploadField {
	f.openable = len(enabled) == 0 || enabled[0]
	return f
}

// Reorderable lets users change the order of a Multiple field's files.
func (f *FileUploadField) Reorderable(enabled ...bool) *FileUploadField {
	f.reorderable = len(enabled) == 0 || enabled[0]
	return f
}

// AppendFiles adds new files after the existing ones of a Multiple field
// (by default they come first).
func (f *FileUploadField) AppendFiles(enabled ...bool) *FileUploadField {
	f.appendFiles = len(enabled) == 0 || enabled[0]
	return f
}

// KeepReplacedFiles keeps files on the disk when they are removed from
// the field or replaced. By default they are deleted once the record is
// saved.
func (f *FileUploadField) KeepReplacedFiles(enabled ...bool) *FileUploadField {
	f.keepReplaced = len(enabled) == 0 || enabled[0]
	return f
}

// ImageResizeTargetWidth shrinks uploaded images to at most px wide
// (PNG, JPEG and still GIF; other formats are stored as sent).
func (f *FileUploadField) ImageResizeTargetWidth(px int) *FileUploadField {
	f.resizeWidth = px
	return f
}

// ImageResizeTargetHeight shrinks uploaded images to at most px high.
func (f *FileUploadField) ImageResizeTargetHeight(px int) *FileUploadField {
	f.resizeHeight = px
	return f
}

// ImageResizeMode is how images are fitted into the target size:
// "contain" (default; keeps the whole image), "cover" (fills the size and
// crops the overflow) or "force" (stretches to the exact size).
func (f *FileUploadField) ImageResizeMode(mode string) *FileUploadField {
	f.resizeMode = mode
	return f
}

// ImageCropAspectRatio crops uploaded images to a ratio such as "16:9",
// "1:1" or "4/3", keeping the centre. Clients that can crop (the Flutter
// app) let the user choose the area first.
func (f *FileUploadField) ImageCropAspectRatio(ratio string) *FileUploadField {
	f.cropAspectRatio = ratio
	f.cropAspectFloat = parseRatio(ratio)
	return f
}

func parseRatio(s string) float64 {
	s = strings.NewReplacer("/", ":", "x", ":").Replace(strings.TrimSpace(s))
	a, b, ok := strings.Cut(s, ":")
	if !ok {
		return 0
	}
	var w, h float64
	if _, err := fmt.Sscan(a, &w); err != nil {
		return 0
	}
	if _, err := fmt.Sscan(b, &h); err != nil || w <= 0 || h <= 0 {
		return 0
	}
	return w / h
}

// FileNamesField returns the StoreFileNamesIn field ("" when unset).
func (f *FileUploadField) FileNamesField() string { return f.fileNamesIn }

// StoredFileName returns the name to store a file under, from
// GetUploadedFileNameUsing or PreserveFilenames; "" means a random name.
func (f *FileUploadField) StoredFileName(original string) string {
	if f.fileNameUsing != nil {
		return f.fileNameUsing(original)
	}
	return ""
}

// KeepsReplacedFiles reports whether removed files stay on the disk.
func (f *FileUploadField) KeepsReplacedFiles() bool { return f.keepReplaced }

// AppendsFiles reports whether new files go after the existing ones.
func (f *FileUploadField) AppendsFiles() bool { return f.appendFiles }

// GetMinFiles returns MinFiles.
func (f *FileUploadField) GetMinFiles() int { return f.minFiles }

// ImageProcessing returns the resize and crop settings: target width and
// height (0 when unset), the resize mode and the crop ratio (0 when none).
func (f *FileUploadField) ImageProcessing() (width, height int, mode string, ratio float64) {
	mode = f.resizeMode
	if mode == "" {
		mode = "contain"
	}
	return f.resizeWidth, f.resizeHeight, mode, f.cropAspectFloat
}

// FileFields returns the file fields of a form that hold their own value
// (not those inside repeaters), skipping components gated out for ctx
// (ctx may be nil).
func FileFields(ctx context.Context, components []Component) []*FileUploadField {
	var out []*FileUploadField
	var walk func([]Component)
	walk = func(cs []Component) {
		for _, c := range cs {
			if !gateAllows(ctx, c) {
				continue
			}
			switch f := c.(type) {
			case *FileUploadField:
				out = append(out, f)
			case *SectionComponent:
				walk(f.schema)
			case *GridComponent:
				walk(f.schema)
			case *FieldsetComponent:
				walk(f.schema)
			case *SplitComponent:
				walk(f.schema)
			case *TabsComponent:
				for _, t := range f.tabs {
					walk(t.schema)
				}
			case *WizardComponent:
				for _, s := range f.steps {
					walk(s.schema)
				}
			}
		}
	}
	walk(components)
	return out
}

// GetLabelOrName returns the field's label, or its name when it has none.
func (f *FileUploadField) GetLabelOrName() string { return f.displayName() }
