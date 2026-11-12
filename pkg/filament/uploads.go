package filament

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	gcontext "github.com/golaravel/golaravel/pkg/context"
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

// UploadsUsing stores the files of FileUpload / ImageUpload fields sent by
// API clients (the Flutter app). Without it, fields with a Disk(...) are
// stored on that golaravel storage disk under Directory(...) with a random
// name; other fields refuse uploads.
func (p *Panel) UploadsUsing(fn UploadHandler) *Panel {
	p.uploadHandler = fn
	return p
}

const maxUploadBytes = 64 << 20 // hard cap when a field has no MaxSize

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func randomName() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// storeOnDisk is the default UploadHandler for fields with a Disk.
func storeOnDisk(ctx *Context, f UploadedFile) (string, string, error) {
	disk := f.Field.GetDisk()
	if disk == "" {
		return "", "", &endpointError{status: http.StatusNotImplemented, msg: "Uploads are not configured for this field"}
	}
	name := randomName() + strings.ToLower(filepath.Ext(f.Filename))
	if f.Field.PreservesFilenames() {
		clean := unsafeName.ReplaceAllString(filepath.Base(f.Filename), "_")
		name = randomName()[:8] + "/" + clean
	}
	dst := path.Join(strings.Trim(f.Field.GetDirectory(), "/"), name)
	r, err := f.Open()
	if err != nil {
		return "", "", err
	}
	defer r.Close()
	d := storage.Disk(disk)
	if err := d.PutReader(dst, r); err != nil {
		return "", "", err
	}
	return dst, d.URL(dst), nil
}

// registerUploadRoute mounts POST <resource>/uploads?field=<name> (multipart
// "file"): it checks the user may see the resource, that the field is a file
// field of its form, and the field's size and type rules, then stores the
// file.
func (p *Panel) registerUploadRoute(r *router.Router, res ResourceInterface, resPath string, preflight gcontext.HandlerFunc) {
	finder, ok := res.(interface {
		FormField(name string) forms.Component
	})
	if !ok {
		return
	}
	r.OPTIONS(resPath+"/uploads", preflight)
	r.POST(resPath+"/uploads", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		if res.NavigationItem(ctx) == nil {
			return p.apiError(c, ErrForbidden)
		}
		field, ok := finder.FormField(c.Query("field")).(*forms.FileUploadField)
		if !ok {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Unknown file field"})
		}
		limit := int64(maxUploadBytes)
		c.Request.Body = http.MaxBytesReader(&c.Response, c.Request.Body, limit+1<<20)
		if err := c.Request.ParseMultipartForm(8 << 20); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Send the file as multipart form data (field \"file\")"})
		}
		defer func() {
			if c.Request.MultipartForm != nil {
				_ = c.Request.MultipartForm.RemoveAll()
			}
		}()
		files := c.Request.MultipartForm.File["file"]
		if len(files) != 1 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Send exactly one file"})
		}
		fh := files[0]
		if fh.Size > limit {
			return p.apiError(c, &forms.ValidationError{Errors: map[string]string{field.GetName(): "The file is too large."}})
		}
		sniff := make([]byte, 512)
		rd, err := fh.Open()
		if err != nil {
			return p.apiError(c, err)
		}
		n, _ := io.ReadFull(rd, sniff)
		rd.Close()
		contentType := http.DetectContentType(sniff[:n])
		if msg := field.CheckUpload(fh.Filename, contentType, fh.Size); msg != "" {
			return p.apiError(c, &forms.ValidationError{Errors: map[string]string{field.GetName(): msg}})
		}

		withSelf(ctx)
		up := UploadedFile{Resource: res.Slug(), Field: field, Filename: fh.Filename, Size: fh.Size, ContentType: contentType, header: fh}
		store := p.uploadHandler
		if store == nil {
			store = storeOnDisk
		}
		value, url, err := store(ctx, up)
		if err != nil {
			var coded interface{ StatusCode() int }
			if errors.As(err, &coded) {
				return p.apiError(c, err)
			}
			return p.apiErrorStatus(c, errors.New("The file could not be stored"), http.StatusInternalServerError)
		}
		return c.JSON(http.StatusCreated, map[string]any{"value": value, "url": url, "name": fh.Filename})
	})
}
