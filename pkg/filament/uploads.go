package filament

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/middleware"
	"github.com/golaravel/golaravel/pkg/router"
	"github.com/golaravel/golaravel/pkg/storage"

	"github.com/golaravel/gofilament/pkg/forms"
)

// UploadedFile is a file sent to a resource form's upload endpoint, after
// the field's size and type checks passed.
type UploadedFile struct {
	Resource    string // resource slug
	Field       *forms.FileUploadField
	Filename    string // as sent by the client; never use it as a path
	Size        int64
	ContentType string // sniffed from the bytes
	header      *multipart.FileHeader
}

// Open returns the file's contents.
func (f UploadedFile) Open() (io.ReadCloser, error) { return f.header.Open() }

// UploadHandler stores a file and returns the value the form field keeps
// (a path or key your handlers understand) and, optionally, a URL to show it.
type UploadHandler func(ctx *Context, f UploadedFile) (value, url string, err error)

// UploadsUsing replaces how FileUpload / ImageUpload files are stored.
// Without it, files go to a temporary area of the field's disk when they
// are picked and move into Directory(...) when the record is saved (see
// the uploads section of the docs); files removed from a field are
// deleted on save. With it, fn stores each file and the field keeps the
// value fn returns as is: nothing is moved, checked against the record or
// deleted.
func (p *Panel) UploadsUsing(fn UploadHandler) *Panel {
	p.uploadHandler = fn
	return p
}

const maxUploadBytes = 64 << 20 // hard cap when a field has no MaxSize

// TemporaryUploadsDir is where picked files wait on their field's disk
// until the record is saved. Files older than TemporaryUploadsTTL are
// pruned from local disks; on S3 and other disks add a lifecycle rule for
// this prefix.
const TemporaryUploadsDir = "tmp-uploads"

// TemporaryUploadsTTL is how long an unsaved upload is kept.
var TemporaryUploadsTTL = 24 * time.Hour

// tmpToken prefixes the field value of a file that is not saved yet.
const tmpToken = "tmp:"

var (
	unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	tmpKey     = regexp.MustCompile(`^[0-9a-f]{32}(\.[a-z0-9]{1,10})?$`)
)

func randomName() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// safeFileName turns a client file name into one safe to store.
func safeFileName(name string) string {
	name = path.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Trim(unsafeName.ReplaceAllString(name, "_"), "._")
	if name == "" {
		return "file"
	}
	if len(name) > 120 {
		ext := filepath.Ext(name)
		if len(ext) > 11 {
			ext = ""
		}
		name = name[:120-len(ext)] + ext
	}
	return name
}

func tmpPath(resource, field, key string) string {
	return path.Join(TemporaryUploadsDir, safeFileName(resource), safeFileName(field), key)
}

// pendingUpload is a parsed tmp: field value.
type pendingUpload struct {
	key      string // random hex + "." + extension
	original string // the client's file name
}

func parsePending(v string) (pendingUpload, bool) {
	rest, ok := strings.CutPrefix(v, tmpToken)
	if !ok {
		return pendingUpload{}, false
	}
	key, name, _ := strings.Cut(rest, "/")
	if !tmpKey.MatchString(key) {
		return pendingUpload{}, false
	}
	orig, err := url.PathUnescape(name)
	if err != nil {
		orig = ""
	}
	return pendingUpload{key: key, original: path.Base(strings.ReplaceAll(orig, `\`, "/"))}, true
}

func (u pendingUpload) ext() string {
	_, ext, _ := strings.Cut(u.key, ".")
	return ext
}

// checkUploadedFile applies the field's size and type rules to a file.
func checkUploadedFile(field *forms.FileUploadField, f *storage.UploadedFile) error {
	limit := int64(maxUploadBytes)
	if f.Size() > limit {
		return &forms.ValidationError{Errors: map[string]string{field.GetName(): "The file is too large."}}
	}
	if msg := field.CheckUpload(f.ClientOriginalName(), f.MimeType(), f.Size()); msg != "" {
		return &forms.ValidationError{Errors: map[string]string{field.GetName(): msg}}
	}
	return nil
}

// stageUpload checks a file against its field and stores it in the
// temporary area of the field's disk. It returns the tmp: value the form
// keeps until the record is saved, and a URL to preview the file.
func stageUpload(scope string, field *forms.FileUploadField, f *storage.UploadedFile) (value, previewURL string, err error) {
	if err := checkUploadedFile(field, f); err != nil {
		return "", "", err
	}
	disk := field.GetDisk()
	if disk == "" {
		return "", "", &endpointError{status: http.StatusNotImplemented, msg: "Uploads are not configured for this field"}
	}
	d := storage.Disk(disk)
	if d == nil {
		return "", "", storage.ErrDiskNotFound
	}
	key := randomName()
	if ext := f.Extension(); ext != "" {
		key += "." + ext
	}
	dst := tmpPath(scope, field.GetName(), key)

	rd, err := f.Open()
	if err != nil {
		return "", "", err
	}
	defer rd.Close()
	var body io.Reader = rd
	if processed, ok := processImage(field, f.MimeType(), rd); ok {
		body = bytes.NewReader(processed)
	} else if _, err := rd.Seek(0, io.SeekStart); err != nil {
		return "", "", err
	}
	if err := d.PutReader(dst, body); err != nil {
		return "", "", err
	}
	go pruneTemporaryUploads(disk)
	preview, _ := storage.DisplayURL(dst, time.Hour, disk)
	return tmpToken + key + "/" + url.PathEscape(f.ClientOriginalName()), preview, nil
}

var lastPrune sync.Map // disk name -> time.Time

// pruneTemporaryUploads deletes unsaved uploads older than
// TemporaryUploadsTTL from a local or in-memory disk, at most once an
// hour per disk.
func pruneTemporaryUploads(disk string) {
	now := time.Now()
	if v, ok := lastPrune.Load(disk); ok && now.Sub(v.(time.Time)) < time.Hour {
		return
	}
	lastPrune.Store(disk, now)
	PruneTemporaryUploads(disk, TemporaryUploadsTTL)
}

// PruneTemporaryUploads deletes files in TemporaryUploadsDir of the disk
// that are older than olderThan, and returns how many it removed. It
// works on local and in-memory disks; others can't be listed, so give
// them a lifecycle rule instead. Uploads also prune their disk hourly.
func PruneTemporaryUploads(disk string, olderThan time.Duration) int {
	cutoff := time.Now().Add(-olderThan)
	removed := 0
	switch d := storage.Disk(disk).(type) {
	case *storage.LocalDriver:
		root := filepath.Join(d.Root(), TemporaryUploadsDir)
		_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || info.ModTime().After(cutoff) {
				return nil
			}
			if os.Remove(p) == nil {
				removed++
			}
			return nil
		})
	case *storage.MemoryDriver:
		for _, p := range d.Files(TemporaryUploadsDir) {
			if t, err := d.LastModified(p); err == nil && t.Before(cutoff) {
				_ = d.Delete(p)
				removed++
			}
		}
	}
	return removed
}

// registerUploadRoute mounts POST <resource>/uploads?field=<name> (multipart
// "file"): it checks the user may see the resource, that the field is a file
// field of its form, and the field's size and type rules, then stores the
// file (see UploadsUsing).
func (p *Panel) registerUploadRoute(r *router.Router, res ResourceInterface, resPath string, preflight gcontext.HandlerFunc) {
	finder, ok := res.(interface {
		FormFieldFor(ctx *Context, name string) forms.Component
	})
	if !ok {
		return
	}
	r.OPTIONS(resPath+"/uploads", preflight)
	// the engine's app-wide body cap (32 MiB by default) would cut uploads
	// short of the fields' own limits
	r.POST(resPath+"/uploads", middleware.MaxBodySize(maxUploadBytes+1<<20), func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		if res.NavigationItem(ctx) == nil {
			return p.apiError(c, ErrForbidden)
		}
		field, ok := finder.FormFieldFor(ctx, c.Query("field")).(*forms.FileUploadField)
		if !ok {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Unknown file field"})
		}
		return p.serveUpload(c, ctx, field, res.Slug(), res.Slug())
	})
}

// serveUpload reads the multipart "file" of an upload request and stores
// it for field: staged in the temporary area of scope, or through the
// panel's UploadsUsing handler.
func (p *Panel) serveUpload(c *gcontext.Context, ctx *Context, field *forms.FileUploadField, resource, scope string) error {
	if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return p.apiError(c, &forms.ValidationError{Errors: map[string]string{field.GetName(): "The file is too large."}})
		}
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Send the file as multipart form data (field \"file\")"})
	}
	defer func() {
		if c.Request.MultipartForm != nil {
			_ = c.Request.MultipartForm.RemoveAll()
		}
	}()
	files := c.Files("file")
	if len(files) != 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Send exactly one file"})
	}
	f := files[0]
	withSelf(ctx)

	if p.uploadHandler != nil {
		if err := checkUploadedFile(field, f); err != nil {
			return p.apiError(c, err)
		}
		up := UploadedFile{Resource: resource, Field: field, Filename: f.ClientOriginalName(), Size: f.Size(), ContentType: f.MimeType(), header: f.Header()}
		value, u, err := p.uploadHandler(ctx, up)
		if err != nil {
			return p.uploadError(c, err)
		}
		return c.JSON(http.StatusCreated, map[string]any{"value": value, "url": u, "name": f.ClientOriginalName()})
	}

	value, u, err := stageUpload(scope, field, f)
	if err != nil {
		return p.uploadError(c, err)
	}
	return c.JSON(http.StatusCreated, map[string]any{"value": value, "url": u, "name": f.ClientOriginalName()})
}

func (p *Panel) uploadError(c *gcontext.Context, err error) error {
	var coded interface{ StatusCode() int }
	var verr *forms.ValidationError
	if errors.As(err, &coded) || errors.As(err, &verr) {
		return p.apiError(c, err)
	}
	return p.apiErrorStatus(c, errors.New("The file could not be stored"), http.StatusInternalServerError)
}
