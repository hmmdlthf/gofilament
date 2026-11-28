package filament

import (
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/golaravel/golaravel/pkg/storage"

	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

// fileCommit is the file work of one create or update: files moved out
// of the temporary area (put back if the save fails) and files removed
// from the record (deleted once it is saved).
type fileCommit struct {
	moved   []fileMove
	removed []fileRef
}

type fileMove struct{ disk, from, to string }
type fileRef struct{ disk, path string }

// finish completes the file work after the record handler ran: on
// success removed files are deleted, otherwise moved files go back to the
// temporary area so the form can be sent again.
func (fc *fileCommit) finish(saved bool) {
	if fc == nil {
		return
	}
	if !saved {
		for _, m := range fc.moved {
			if d := storage.Disk(m.disk); d != nil {
				_ = d.Move(m.to, m.from)
			}
		}
		return
	}
	for _, r := range fc.removed {
		if d := storage.Disk(r.disk); d != nil {
			if err := d.Delete(r.path); err != nil {
				slog.Warn("filament: could not delete a replaced file", "disk", r.disk, "path", r.path, "error", err)
			}
		}
	}
}

func valueList(v any) []string {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		if x == "" {
			return nil
		}
		return []string{x}
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			if s := fmt.Sprint(e); e != nil && s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return []string{fmt.Sprint(v)}
}

// finalPath is where a pending upload is stored for good.
func finalPath(field *forms.FileUploadField, u pendingUpload) string {
	dir := strings.Trim(field.GetDirectory(), "/")
	var name string
	switch {
	case field.StoredFileName(u.original) != "":
		name = strings.TrimPrefix(path.Clean("/"+strings.ReplaceAll(field.StoredFileName(u.original), `\`, "/")), "/")
	case field.PreservesFilenames():
		name = randomName()[:8] + "/" + safeFileName(u.original)
	default:
		name = randomName() + randomName()[:8]
		if ext := u.ext(); ext != "" {
			name += "." + ext
		}
	}
	return path.Join(dir, name)
}

// resolveFiles replaces the file fields' values in validated data with
// stored paths before the record handler runs:
//
//   - tmp: values (files picked but not saved) move from the temporary
//     area into the field's Directory;
//   - other values must already be on the record (they can't name some
//     other file on the disk);
//   - files on the record that are no longer in the field are deleted
//     after a successful save, unless the field KeepReplacedFiles;
//   - StoreFileNamesIn fields get the original names.
//
// old is the current record (nil when creating). Panels with a custom
// UploadsUsing handler keep their values unchanged.
func (r *BaseResource[T]) resolveFiles(ctx *Context, old map[string]any, data map[string]any) (*fileCommit, error) {
	if ctx.Panel() != nil && ctx.Panel().uploadHandler != nil {
		return nil, nil
	}
	comps := r.formComponents()
	fields := forms.FileFields(ctx.GoContext(), comps)
	if len(fields) == 0 {
		return nil, nil
	}
	fc := &fileCommit{}
	errs := map[string]string{}
	for _, f := range fields {
		name := f.GetName()
		raw, present := data[name]
		if !present {
			continue
		}
		disk := f.GetDisk()
		d := storage.Disk(disk)
		oldVals := valueList(old[name])
		known := map[string]bool{}
		for _, v := range oldVals {
			known[v] = true
		}
		oldNames := map[string]string{}
		if col := f.FileNamesField(); col != "" {
			switch x := old[col].(type) {
			case map[string]any:
				for k, v := range x {
					oldNames[k] = fmt.Sprint(v)
				}
			case string:
				if len(oldVals) == 1 {
					oldNames[oldVals[0]] = x
				}
			}
		}

		var kept, added []string
		names := map[string]string{}
		for _, v := range valueList(raw) {
			if u, ok := parsePending(v); ok {
				if d == nil {
					errs[name] = "Uploads are not configured for this field."
					break
				}
				from := tmpPath(r.Slug(), name, u.key)
				if !d.Exists(from) {
					errs[name] = f.GetLabelOrName() + ": an upload expired or was not found. Choose the file again."
					break
				}
				to := finalPath(f, u)
				if err := d.Move(from, to); err != nil {
					errs[name] = f.GetLabelOrName() + ": the file could not be stored."
					break
				}
				fc.moved = append(fc.moved, fileMove{disk: disk, from: from, to: to})
				added = append(added, to)
				names[to] = u.original
				continue
			}
			if !known[v] {
				errs[name] = f.GetLabelOrName() + " contains a file that was not uploaded to it."
				break
			}
			kept = append(kept, v)
			if n, ok := oldNames[v]; ok {
				names[v] = n
			}
		}
		if _, failed := errs[name]; failed {
			continue
		}

		values := append(kept, added...)
		if !f.AppendsFiles() && f.IsMultiple() {
			values = append(added, kept...)
		}
		if !f.IsMultiple() && len(values) > 1 {
			values = values[len(values)-1:] // a new file replaces the old one
		}
		inField := map[string]bool{}
		for _, v := range values {
			inField[v] = true
		}
		if !f.KeepsReplacedFiles() {
			for _, v := range oldVals {
				if !inField[v] && !isURL(v) {
					fc.removed = append(fc.removed, fileRef{disk: disk, path: v})
				}
			}
		}

		if f.IsMultiple() {
			if values == nil {
				values = []string{}
			}
			data[name] = values
		} else if len(values) == 1 {
			data[name] = values[0]
		} else {
			data[name] = ""
		}
		if col := f.FileNamesField(); col != "" {
			if f.IsMultiple() {
				m := map[string]string{}
				for _, v := range values {
					if n, ok := names[v]; ok {
						m[v] = n
					}
				}
				data[col] = m
			} else if len(values) == 1 {
				data[col] = names[values[0]]
			} else {
				data[col] = ""
			}
		}
	}
	if len(errs) > 0 {
		fc.finish(false)
		return nil, &forms.ValidationError{Errors: errs}
	}
	return fc, nil
}

func isURL(v string) bool {
	return strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "/")
}

// oldRecordFor loads the record a write updates as a map, for
// resolveFiles; nil when the resource has no finder or form file fields.
func (r *BaseResource[T]) oldRecordFor(ctx *Context, id string) map[string]any {
	if r.onFind == nil || len(forms.FileFields(ctx.GoContext(), r.formComponents())) == 0 {
		return nil
	}
	rec, err := r.onFind(ctx, id)
	if err != nil {
		return nil
	}
	return recordToMap(rec)
}

// FileInfo describes a stored file for clients: the field value, a URL
// to show or download it (signed and short-lived on private disks) and a
// display name.
type FileInfo struct {
	Value string `json:"value"`
	URL   string `json:"url,omitempty"`
	Name  string `json:"name"`
}

// FileURL returns a URL for a stored file of a disk: the value itself
// when it already is a URL, a plain URL on a public disk and a signed,
// short-lived one (tables.StorageURLTTL) on a private disk.
func FileURL(disk, value string) string { return tables.StorageURL(disk, value) }

// RecordFiles returns the stored files of each file field of the
// resource form for a record, keyed by field name.
func (r *BaseResource[T]) RecordFiles(ctx *Context, record any) map[string][]FileInfo {
	fields := forms.FileFields(ctx.GoContext(), r.formComponents())
	if len(fields) == 0 {
		return nil
	}
	m := recordToMap(record)
	out := map[string][]FileInfo{}
	for _, f := range fields {
		vals := valueList(m[f.GetName()])
		if len(vals) == 0 {
			continue
		}
		names := map[string]string{}
		if col := f.FileNamesField(); col != "" {
			switch x := m[col].(type) {
			case map[string]any:
				for k, v := range x {
					names[k] = fmt.Sprint(v)
				}
			case string:
				names[vals[0]] = x
			}
		}
		list := make([]FileInfo, 0, len(vals))
		for _, v := range vals {
			n := names[v]
			if n == "" {
				n = path.Base(v)
			}
			list = append(list, FileInfo{Value: v, URL: FileURL(f.GetDisk(), v), Name: n})
		}
		out[f.GetName()] = list
	}
	return out
}
