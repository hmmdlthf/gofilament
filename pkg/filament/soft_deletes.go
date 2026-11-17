package filament

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golaravel/gofilament/pkg/tables"
)

// Soft deletes: deleted records stay in the database and can be restored.
// The list's "trashed" filter (filters["trashed"] in OnList: "" or
// "without_trashed", "with_trashed", "only_trashed") shows them; trashed
// records cannot be changed until they are restored.

// ErrRecordTrashed is returned (409) for a change to a trashed record.
var ErrRecordTrashed error = &endpointError{status: http.StatusConflict, msg: "This record is in the trash. Restore it before changing it."}

// ErrNotSoftDeletable is returned (404) for restore / force delete on a
// resource without soft deletes or without the handler.
var ErrNotSoftDeletable error = &endpointError{status: http.StatusNotFound, msg: "Not found"}

// refuseTrashed stops changes to a record that is in the trash.
func (r *BaseResource[T]) refuseTrashed(ctx *Context, id string) error {
	if !r.softDeletes || r.onFind == nil {
		return nil
	}
	rec, err := r.onFind(ctx, id)
	if err != nil {
		return nil // the change reports a missing record itself
	}
	if r.checkIsTrashed(rec) {
		return ErrRecordTrashed
	}
	return nil
}

// mayRestore: CanRestore, or else the resource's delete permission.
func (r *BaseResource[T]) mayRestore(ctx *Context, id string) bool {
	if r.canRestore != nil {
		return r.canRestore(ctx, id)
	}
	return r.canDeleteAny == nil || r.canDeleteAny(ctx)
}

// mayForceDelete: CanForceDelete, or else the resource's delete permission.
func (r *BaseResource[T]) mayForceDelete(ctx *Context, id string) bool {
	if r.canForceDelete != nil {
		return r.canForceDelete(ctx, id)
	}
	return r.canDeleteAny == nil || r.canDeleteAny(ctx)
}

// InvokeRestore restores trashed records.
func (r *BaseResource[T]) InvokeRestore(ctx *Context, ids ...string) error {
	if !r.softDeletes || (r.onRestore == nil && (len(ids) < 2 || r.onRestoreBulk == nil)) {
		return ErrNotSoftDeletable
	}
	for _, id := range ids {
		if !r.mayRestore(ctx, id) {
			return ErrForbidden
		}
	}
	if len(ids) > 1 && r.onRestoreBulk != nil {
		return r.onRestoreBulk(ctx, ids)
	}
	var errs []error
	for _, id := range ids {
		if err := r.onRestore(ctx, id); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

// InvokeForceDelete deletes trashed records for good.
func (r *BaseResource[T]) InvokeForceDelete(ctx *Context, ids ...string) error {
	if !r.softDeletes || (r.onForceDelete == nil && (len(ids) < 2 || r.onForceDeleteBulk == nil)) {
		return ErrNotSoftDeletable
	}
	for _, id := range ids {
		if !r.mayForceDelete(ctx, id) {
			return ErrForbidden
		}
	}
	if len(ids) > 1 && r.onForceDeleteBulk != nil {
		return r.onForceDeleteBulk(ctx, ids)
	}
	var errs []error
	for _, id := range ids {
		if err := r.onForceDelete(ctx, id); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

// withTrashedFilter adds the "trashed" filter to a soft-deleting resource's
// filters (web, API and Flutter) unless the table has one.
func withTrashedFilter[T any](r *BaseResource[T], tbl *tables.Table[T]) {
	if r.softDeletes && !tbl.HasFilter("trashed") {
		tbl.Filters(append(tbl.GetFilters(), tables.TrashedFilter())...)
	}
}

func wantsJSON(ctx *Context, basePath string) bool {
	return strings.Contains(ctx.Request.Header.Get("Accept"), "application/json") ||
		strings.Contains(ctx.Request.Header.Get("Content-Type"), "application/json") ||
		strings.Contains(basePath, "/api")
}
