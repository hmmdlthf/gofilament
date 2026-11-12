package forms

import (
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
