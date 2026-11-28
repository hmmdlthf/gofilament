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
// Fields inside repeaters (nested ones too) work the same way: a row may
// keep any file that was in that repeater on the record.
//
// old is the current record (nil when creating). Panels with a custom
// UploadsUsing handler keep their values unchanged.
func (r *BaseResource[T]) resolveFiles(ctx *Context, old map[string]any, data map[string]any) (*fileCommit, error) {
	return resolveFormFiles(ctx, r.Slug(), r.formComponents(), old, data)
}

// resolveFormFiles is resolveFiles for any form: scope names the form's
// temporary upload area (the resource slug, or "<slug>.<relationship>").
func resolveFormFiles(ctx *Context, scope string, comps []forms.Component, old, data map[string]any) (*fileCommit, error) {
	if ctx.Panel() != nil && ctx.Panel().uploadHandler != nil {
		return nil, nil
	}
	if !forms.HasFileFields(ctx.GoContext(), comps) {
		return nil, nil
	}
	fr := &fileResolver{ctx: ctx, scope: scope, fc: &fileCommit{}, errs: map[string]string{}}
	var oldRows []map[string]any
	if old != nil {
		oldRows = []map[string]any{old}
	}
	fr.level(comps, oldRows, []fileRow{{data: data}})
	if len(fr.errs) > 0 {
		fr.fc.finish(false)
		return nil, &forms.ValidationError{Errors: fr.errs}
	}
	return fr.fc, nil
}

type fileResolver struct {
	ctx   *Context
	scope string
	fc    *fileCommit
	errs  map[string]string
}

// fileRow is one map of submitted values: the record, or a repeater row
// (prefix is its error key prefix, such as "lines.2.").
type fileRow struct {
	data   map[string]any
	prefix string
}

func rowMaps(v any) []map[string]any {
	var out []map[string]any
	switch x := v.(type) {
	case []any:
		for _, e := range x {
			if m, ok := e.(map[string]any); ok {
				out = append(out, m)
			}
		}
	case []map[string]any:
		out = x
	}
	return out
}

// level resolves the file fields of one form level for the submitted rows,
// against the values those fields held in oldRows (all rows of that level
// on the record), then descends into repeaters.
func (fr *fileResolver) level(comps []forms.Component, oldRows []map[string]any, rows []fileRow) {
	files, reps := forms.FileScope(fr.ctx.GoContext(), comps)
	for _, f := range files {
		fr.field(f, oldRows, rows)
	}
	for _, rep := range reps {
		name := rep.GetName()
		var childOld []map[string]any
		for _, o := range oldRows {
			childOld = append(childOld, rowMaps(o[name])...)
		}
		var childRows []fileRow
		present := false
		for _, row := range rows {
			raw, ok := row.data[name]
			if !ok {
				continue
			}
			present = true
			for i, m := range rowMaps(raw) {
				childRows = append(childRows, fileRow{data: m, prefix: fmt.Sprintf("%s%s.%d.", row.prefix, name, i)})
			}
		}
		if present {
			fr.level(rep.GetSchema(), childOld, childRows)
		}
	}
}

// field resolves one file field across rows; files that were in the field
// on any old row and are in none of the new rows are deleted after save.
func (fr *fileResolver) field(f *forms.FileUploadField, oldRows []map[string]any, rows []fileRow) {
	name := f.GetName()
	disk := f.GetDisk()
	d := storage.Disk(disk)

	known := map[string]bool{}
	var oldVals []string
	oldNames := map[string]string{}
	for _, o := range oldRows {
		vals := valueList(o[name])
		for _, v := range vals {
			if !known[v] {
				known[v] = true
				oldVals = append(oldVals, v)
			}
		}
		if col := f.FileNamesField(); col != "" {
			switch x := o[col].(type) {
			case map[string]any:
				for k, v := range x {
					oldNames[k] = fmt.Sprint(v)
				}
			case string:
				if len(vals) == 1 {
					oldNames[vals[0]] = x
				}
			}
		}
	}

	inField := map[string]bool{}
	anyPresent := false
	for _, row := range rows {
		raw, present := row.data[name]
		if !present {
			continue
		}
		anyPresent = true
		key := row.prefix + name
		var kept, added []string
		names := map[string]string{}
		failed := false
		for _, v := range valueList(raw) {
			if u, ok := parsePending(v); ok {
				if d == nil {
					fr.errs[key] = "Uploads are not configured for this field."
					failed = true
					break
				}
				from := tmpPath(fr.scope, name, u.key)
				if !d.Exists(from) {
					fr.errs[key] = f.GetLabelOrName() + ": an upload expired or was not found. Choose the file again."
					failed = true
					break
				}
				to := finalPath(f, u)
				if err := d.Move(from, to); err != nil {
					fr.errs[key] = f.GetLabelOrName() + ": the file could not be stored."
					failed = true
					break
				}
				fr.fc.moved = append(fr.fc.moved, fileMove{disk: disk, from: from, to: to})
				added = append(added, to)
				names[to] = u.original
				continue
			}
			if !known[v] {
				fr.errs[key] = f.GetLabelOrName() + " contains a file that was not uploaded to it."
				failed = true
				break
			}
			kept = append(kept, v)
			if n, ok := oldNames[v]; ok {
				names[v] = n
			}
		}
		if failed {
			// keep the old files of this field: the save will not happen
			for _, v := range oldVals {
				inField[v] = true
			}
			continue
		}

		values := append(kept, added...)
		if !f.AppendsFiles() && f.IsMultiple() {
			values = append(added, kept...)
		}
		if !f.IsMultiple() && len(values) > 1 {
			values = values[len(values)-1:] // a new file replaces the old one
		}
		for _, v := range values {
			inField[v] = true
		}
		if f.IsMultiple() {
			if values == nil {
				values = []string{}
			}
			row.data[name] = values
		} else if len(values) == 1 {
			row.data[name] = values[0]
		} else {
			row.data[name] = ""
		}
		if col := f.FileNamesField(); col != "" {
			if f.IsMultiple() {
				m := map[string]string{}
				for _, v := range values {
					if n, ok := names[v]; ok {
						m[v] = n
					}
				}
				row.data[col] = m
			} else if len(values) == 1 {
				row.data[col] = names[values[0]]
			} else {
				row.data[col] = ""
			}
		}
	}
	if !anyPresent || f.KeepsReplacedFiles() {
		return
	}
	for _, v := range oldVals {
		if !inField[v] && !isURL(v) {
			fr.fc.removed = append(fr.fc.removed, fileRef{disk: disk, path: v})
		}
	}
}

func isURL(v string) bool {
	return strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "/")
}

// oldRecordFor loads the record a write updates as a map, for
// resolveFiles; nil when the resource has no finder or form file fields.
func (r *BaseResource[T]) oldRecordFor(ctx *Context, id string) map[string]any {
	if r.onFind == nil || !forms.HasFileFields(ctx.GoContext(), r.formComponents()) {
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
// resource form for a record, keyed by field name; fields inside
// repeaters are keyed "<repeater>.<field>" and list the files of all rows.
func (r *BaseResource[T]) RecordFiles(ctx *Context, record any) map[string][]FileInfo {
	return formFiles(ctx, r.formComponents(), record)
}

func formFiles(ctx *Context, comps []forms.Component, record any) map[string][]FileInfo {
	if !forms.HasFileFields(ctx.GoContext(), comps) {
		return nil
	}
	out := map[string][]FileInfo{}
	collectFiles(ctx, comps, []map[string]any{recordToMap(record)}, "", out)
	return out
}

func collectFiles(ctx *Context, comps []forms.Component, rows []map[string]any, prefix string, out map[string][]FileInfo) {
	files, reps := forms.FileScope(ctx.GoContext(), comps)
	for _, f := range files {
		var list []FileInfo
		for _, m := range rows {
			vals := valueList(m[f.GetName()])
			names := map[string]string{}
			if col := f.FileNamesField(); col != "" {
				switch x := m[col].(type) {
				case map[string]any:
					for k, v := range x {
						names[k] = fmt.Sprint(v)
					}
				case string:
					if len(vals) > 0 {
						names[vals[0]] = x
					}
				}
			}
			for _, v := range vals {
				n := names[v]
				if n == "" {
					n = path.Base(v)
				}
				list = append(list, FileInfo{Value: v, URL: FileURL(f.GetDisk(), v), Name: n})
			}
		}
		if len(list) > 0 {
			out[prefix+f.GetName()] = list
		}
	}
	for _, rep := range reps {
		var child []map[string]any
		for _, m := range rows {
			child = append(child, rowMaps(m[rep.GetName()])...)
		}
		if len(child) > 0 {
			collectFiles(ctx, rep.GetSchema(), child, prefix+rep.GetName()+".", out)
		}
	}
}
