package filament

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/golaravel/pkg/cache"
)

// The web import flow: upload -> preview (map columns, see each row's
// errors) -> import. The uploaded file waits in golaravel's cache under a
// token bound to the user for 30 minutes.

const importFileTTL = 30 * time.Minute

func (r *BaseResource[T]) importCacheKey(ctx *Context, token string) string {
	uid := "guest"
	if u := ctx.User(); u != nil {
		uid = u.GetAuthIdentifier()
	}
	return "gofilament:import:" + r.slug + ":" + importHashKey(uid+"|"+token)
}

func importHashKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:16])
}

func (r *BaseResource[T]) stashImportFile(ctx *Context, raw []byte) (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)
	return token, cache.Put(r.importCacheKey(ctx, token), base64.StdEncoding.EncodeToString(raw), importFileTTL)
}

func (r *BaseResource[T]) stashedImportFile(ctx *Context, token string) ([]byte, bool) {
	if token == "" {
		return nil, false
	}
	enc := cache.GetString(r.importCacheKey(ctx, token))
	raw, err := base64.StdEncoding.DecodeString(enc)
	return raw, err == nil && enc != ""
}

// importFormMapping reads mapping[<header>] fields of the form.
func importFormMapping(ctx *Context) map[string]string {
	mapping := map[string]string{}
	for k, v := range ctx.Request.PostForm {
		if strings.HasPrefix(k, "mapping[") && strings.HasSuffix(k, "]") && len(v) > 0 {
			mapping[strings.TrimSuffix(strings.TrimPrefix(k, "mapping["), "]")] = v[0]
		}
	}
	return mapping
}

// uploadedImportFile reads the "file" (or "csv_file") upload.
func uploadedImportFile(ctx *Context) ([]byte, error) {
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		file, _, err = ctx.Request.FormFile("csv_file")
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, MaxImportBytes+1))
	if err == nil && len(raw) > MaxImportBytes {
		err = fmt.Errorf("the file is larger than %d MB", MaxImportBytes>>20)
	}
	return raw, err
}

func (r *BaseResource[T]) handleImportPreview(ctx *Context, basePath string) error {
	if r.canCreate != nil && !r.canCreate(ctx) {
		return ctx.String(http.StatusForbidden, "403 Forbidden: Access denied")
	}
	_ = ctx.Request.ParseMultipartForm(MaxImportBytes + 1<<20)
	token := ctx.Request.FormValue("token")
	raw, ok := r.stashedImportFile(ctx, token)
	if !ok {
		var err error
		raw, err = uploadedImportFile(ctx)
		if err != nil {
			return r.renderImportUpload(ctx, basePath, http.StatusUnprocessableEntity, "Choose a CSV, TSV or Excel file to import.")
		}
		if token, err = r.stashImportFile(ctx, raw); err != nil {
			return r.renderImportUpload(ctx, basePath, http.StatusServiceUnavailable, "The file could not be kept for the preview.")
		}
	}
	preview, err := r.PreviewImport(ctx, bytes.NewReader(raw), importFormMapping(ctx))
	if err != nil {
		var verr *forms.ValidationError
		if errors.As(err, &verr) {
			return r.renderImportUpload(ctx, basePath, http.StatusUnprocessableEntity, verr.Errors["file"])
		}
		return r.renderImportUpload(ctx, basePath, webStatus(err), err.Error())
	}
	return r.renderImportPreview(ctx, basePath, token, preview)
}

func (r *BaseResource[T]) renderImportUpload(ctx *Context, basePath string, status int, errMsg string) error {
	return r.renderPageStatus(ctx, status, "Import "+r.pluralModelName, r.importUploadHTML(ctx, basePath, errMsg))
}

func (r *BaseResource[T]) importUploadHTML(ctx *Context, basePath, errMsg string) string {
	return fmt.Sprintf(`
<div style="max-width: 48rem; margin: 0 auto; padding: 2rem 1rem;">
  <div style="margin-bottom: 1.5rem;">
    <a href="%s" style="font-size: 0.875rem; color: var(--primary-600); text-decoration: none;">&larr; Back to %s</a>
    <h1 style="font-size: 1.875rem; font-weight: 700; color: var(--gray-900); margin-top: 0.5rem;">Import %s</h1>
    <p style="font-size: 0.875rem; color: var(--gray-500); margin-top: 0.25rem;">
      Upload a CSV, TSV or Excel file. You can match its columns to fields and check every row before anything is saved.
    </p>
  </div>
  %s
  <div class="filament-card" style="padding: 2rem;">
    <form method="POST" action="%s/import/preview" enctype="multipart/form-data">
      %s
      <div style="margin-bottom: 1.5rem;">
        <label style="display: block; font-size: 0.875rem; font-weight: 500; color: var(--gray-700); margin-bottom: 0.5rem;">
          CSV, TSV or Excel file
        </label>
        <input type="file" name="file" accept=".csv,.tsv,.txt,.xlsx" required class="form-control" style="padding: 0.5rem;" />
        <p style="font-size: 0.75rem; color: var(--gray-500); margin-top: 0.375rem;">
          The first row holds the column names. Need a sample format? <a href="%s/export?format=csv" style="color: var(--primary-600);">Download CSV Template</a>.
        </p>
      </div>
      <div style="display: flex; justify-content: flex-end; gap: 0.75rem; margin-top: 2rem;">
        <a href="%s" class="btn btn-secondary">Cancel</a>
        <button type="submit" class="btn btn-primary">Preview</button>
      </div>
    </form>
  </div>
</div>`, basePath, html.EscapeString(r.pluralModelName), html.EscapeString(r.pluralModelName), alertHTML("err", errMsg),
		basePath, csrfInputFor(ctx), basePath, basePath)
}

func csrfInputFor(ctx *Context) string {
	if ctx.Panel() == nil {
		return ""
	}
	return ctx.Panel().csrfHiddenInput(ctx.Context, "")
}

func (r *BaseResource[T]) renderImportPreview(ctx *Context, basePath, token string, pv *ImportPreview) error {
	var b strings.Builder
	esc := html.EscapeString
	csrf := csrfInputFor(ctx)
	fmt.Fprintf(&b, `<div style="max-width: 72rem; margin: 0 auto; padding: 2rem 1rem;">
<a href="%s/import" style="font-size: 0.875rem; color: var(--primary-600); text-decoration: none;">&larr; Choose another file</a>
<h1 style="font-size: 1.5rem; font-weight: 700; margin: 0.5rem 0 1rem;">Import %s: check the file</h1>`, basePath, esc(r.pluralModelName))
	kind := "ok"
	if pv.Invalid > 0 {
		kind = "err"
	}
	b.WriteString(alertHTML(kind, fmt.Sprintf("%d rows: %d ready to import, %d with problems.", pv.Total, pv.Valid, pv.Invalid)))

	fmt.Fprintf(&b, `<form method="POST" action="%s/import/preview" class="filament-card" style="padding: 1.25rem; margin-bottom: 1rem;" data-import-mapping>%s<input type="hidden" name="token" value="%s" />
<h3 style="font-weight: 700; margin-bottom: 0.75rem;">Columns</h3>
<table class="filament-table"><thead><tr><th>Column in the file</th><th>Imports into</th></tr></thead><tbody>`, basePath, csrf, esc(token))
	for _, h := range pv.Headers {
		fmt.Fprintf(&b, `<tr><td>%s</td><td><select name="mapping[%s]" class="form-control"><option value="">Do not import</option>`, esc(h), esc(h))
		if len(pv.Fields) == 0 {
			fmt.Fprintf(&b, `<option value="%s" selected>%s</option>`, esc(pv.Mapping[h]), esc(pv.Mapping[h]))
		}
		for _, f := range pv.Fields {
			sel := ""
			if pv.Mapping[h] == f.Name {
				sel = " selected"
			}
			req := ""
			if f.Required {
				req = " *"
			}
			fmt.Fprintf(&b, `<option value="%s"%s>%s%s</option>`, esc(f.Name), sel, esc(f.Label), req)
		}
		b.WriteString(`</select></td></tr>`)
	}
	b.WriteString(`</tbody></table><div style="margin-top: 0.75rem;"><button type="submit" class="btn btn-secondary">Check again</button></div></form>`)

	// Preview rows
	cols := make([]string, 0, len(pv.Headers))
	seen := map[string]bool{}
	for _, h := range pv.Headers {
		if f := pv.Mapping[h]; f != "" && !seen[f] {
			seen[f] = true
			cols = append(cols, f)
		}
	}
	label := map[string]string{}
	for _, f := range pv.Fields {
		label[f.Name] = f.Label
	}
	fmt.Fprintf(&b, `<div class="filament-card" style="padding: 1.25rem; margin-bottom: 1rem; overflow-x: auto;"><h3 style="font-weight: 700; margin-bottom: 0.75rem;">First %d rows</h3><table class="filament-table"><thead><tr><th>Row</th>`, len(pv.Rows))
	for _, c := range cols {
		l := label[c]
		if l == "" {
			l = c
		}
		fmt.Fprintf(&b, `<th>%s</th>`, esc(l))
	}
	b.WriteString(`<th>Problems</th></tr></thead><tbody>`)
	for _, row := range pv.Rows {
		style := ""
		if len(row.Errors) > 0 {
			style = ` style="background: #fef2f2;" data-import-invalid`
		}
		fmt.Fprintf(&b, `<tr%s><td>%d</td>`, style, row.Row)
		for _, c := range cols {
			fmt.Fprintf(&b, `<td>%s</td>`, esc(row.Values[c]))
		}
		fmt.Fprintf(&b, `<td style="color: #b91c1c;">%s</td></tr>`, esc(ImportRowError{Errors: row.Errors}.describe()))
	}
	b.WriteString(`</tbody></table></div>`)

	if len(pv.Problems) > 0 {
		b.WriteString(`<div class="filament-card" style="padding: 1.25rem; margin-bottom: 1rem;"><h3 style="font-weight: 700; margin-bottom: 0.5rem;">Rows that will be skipped</h3><ul style="font-size: 0.875rem; color: #b91c1c;">`)
		for _, p := range pv.Problems {
			fmt.Fprintf(&b, `<li>Row %d: %s</li>`, p.Row, esc(p.describe()))
		}
		b.WriteString(`</ul></div>`)
	}

	// Import with the same mapping
	fmt.Fprintf(&b, `<form method="POST" action="%s/import" style="display: flex; justify-content: flex-end; gap: 0.75rem;">%s<input type="hidden" name="token" value="%s" />`, basePath, csrf, esc(token))
	keys := make([]string, 0, len(pv.Mapping))
	for h := range pv.Mapping {
		keys = append(keys, h)
	}
	sort.Strings(keys)
	for _, h := range keys {
		fmt.Fprintf(&b, `<input type="hidden" name="mapping[%s]" value="%s" />`, esc(h), esc(pv.Mapping[h]))
	}
	disabled := ""
	if pv.Valid == 0 {
		disabled = " disabled"
	}
	fmt.Fprintf(&b, `<a href="%s" class="btn btn-secondary">Cancel</a><button type="submit" class="btn btn-primary"%s>Import %d rows</button></form></div>`, basePath, disabled, pv.Valid)
	return r.renderPage(ctx, "Import "+r.pluralModelName, b.String())
}

func (r *BaseResource[T]) renderImportResult(ctx *Context, basePath string, res *ImportResult) error {
	var b strings.Builder
	fmt.Fprintf(&b, `<div style="max-width: 48rem; margin: 0 auto; padding: 2rem 1rem;"><h1 style="font-size: 1.5rem; font-weight: 700; margin-bottom: 1rem;">Import %s</h1>`, html.EscapeString(r.pluralModelName))
	b.WriteString(alertHTML("ok", fmt.Sprintf("%d rows imported.", res.SuccessCount)))
	if res.FailedCount > 0 {
		b.WriteString(alertHTML("err", fmt.Sprintf("%d rows were skipped.", res.FailedCount)))
		b.WriteString(`<ul style="font-size: 0.875rem; color: #b91c1c; margin-bottom: 1rem;">`)
		for _, row := range res.Rows {
			fmt.Fprintf(&b, `<li>Row %d: %s</li>`, row.Row, html.EscapeString(row.describe()))
		}
		b.WriteString(`</ul>`)
	}
	fmt.Fprintf(&b, `<a href="%s" class="btn btn-primary">Back to %s</a></div>`, basePath, html.EscapeString(r.pluralModelName))
	return r.renderPage(ctx, "Import "+r.pluralModelName, b.String())
}
