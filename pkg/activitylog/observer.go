package activitylog

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
)

// AuditOptions configures EnableAuditLogWith.
type AuditOptions struct {
	// Store keeps the entries (default: the package's default store).
	Store Store
	// Except lists fields whose values are never stored; a change to them
	// is recorded as "(hidden)". Password fields of the form and fields
	// whose names contain "password", "secret" or "token" are always hidden.
	Except []string
	// NoHistory leaves the record pages without a History section.
	NoHistory bool
}

// EnableAuditLog records who created, changed, deleted or restored the
// resource's records, with the changed fields' old and new values, and
// shows that history on the record pages (web, API and Flutter). Call it
// after the resource's On* handlers are set: it wraps them.
//
// To attach "created" entries to the new record, call
// filament.SetCreatedID(ctx, id) in OnCreate (or put "id" in the data).
func EnableAuditLog[T any](res *filament.BaseResource[T], store ...Store) *filament.BaseResource[T] {
	opts := AuditOptions{}
	if len(store) > 0 {
		opts.Store = store[0]
	}
	return EnableAuditLogWith(res, opts)
}

// EnableAuditLogWith is EnableAuditLog with options.
func EnableAuditLogWith[T any](res *filament.BaseResource[T], opts AuditOptions) *filament.BaseResource[T] {
	s := opts.Store
	if s == nil {
		s = GetDefaultStore()
	}
	subjectType := res.GetModelName()
	if subjectType == "" {
		subjectType = res.Slug()
	}
	hidden := map[string]bool{}
	for _, f := range opts.Except {
		hidden[f] = true
	}
	// hiddenFor returns the check for fields whose values are never stored.
	hiddenFor := func(ctx *filament.Context) func(string) bool {
		pw := map[string]bool{}
		var walk func(ns []forms.Node)
		walk = func(ns []forms.Node) {
			for _, n := range ns {
				if n.Kind == "field" && n.Type == "password" {
					pw[n.Name] = true
				}
				walk(n.Children)
			}
		}
		walk(res.DescribeForm(ctx))
		return func(field string) bool {
			if hidden[field] || pw[field] {
				return true
			}
			lower := strings.ToLower(field)
			for _, w := range []string{"password", "secret", "token"} {
				if strings.Contains(lower, w) {
					return true
				}
			}
			return false
		}
	}
	save := func(ctx *filament.Context, b *Builder) {
		populateCauserFromContext(b, ctx)
		_, _ = b.Store(s).Save(toStdContext(ctx))
	}

	if origCreate := res.GetOnCreate(); origCreate != nil {
		res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
			if err := origCreate(ctx, data); err != nil {
				return err
			}
			subjID := filament.CreatedID(ctx)
			if subjID == "" {
				if idVal, ok := data["id"]; ok && idVal != nil {
					subjID = fmt.Sprintf("%v", idVal)
				}
			}
			desc := fmt.Sprintf("Created %s", subjectType)
			if subjID != "" {
				desc = fmt.Sprintf("Created %s #%s", subjectType, subjID)
			}
			isHidden := hiddenFor(ctx)
			values := map[string]any{}
			for k, v := range data {
				if strings.HasPrefix(k, "_") {
					continue
				}
				if isHidden(k) {
					v = "(hidden)"
				}
				values[k] = v
			}
			save(ctx, Log(desc).Subject(subjectType, subjID).Event("created").New(values))
			return nil
		})
	}

	// logChanges records the fields of data that differ from the record
	// before the change (nothing when no field changed).
	logChanges := func(ctx *filament.Context, id string, before map[string]any, data map[string]any, cell bool) {
		oldVals, newVals := changedFields(before, data, hiddenFor(ctx))
		if len(newVals) == 0 {
			return
		}
		b := Log(fmt.Sprintf("Updated %s #%s", subjectType, id)).Subject(subjectType, id).Event("updated").WithDiff(oldVals, newVals)
		if cell {
			b.Property("inline", true)
		}
		save(ctx, b)
	}
	findBefore := func(ctx *filament.Context, id string) map[string]any {
		if find := res.GetOnFind(); find != nil {
			if rec, err := find(ctx, id); err == nil {
				return recordToMap(rec)
			}
		}
		return nil
	}

	if origUpdate := res.GetOnUpdate(); origUpdate != nil {
		res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
			before := findBefore(ctx, id)
			if err := origUpdate(ctx, id, data); err != nil {
				return err
			}
			logChanges(ctx, id, before, data, false)
			return nil
		})
	}

	if origCell := res.GetOnCellUpdate(); origCell != nil {
		res.OnCellUpdate(func(ctx *filament.Context, id, column string, value any) error {
			before := findBefore(ctx, id)
			if err := origCell(ctx, id, column, value); err != nil {
				return err
			}
			logChanges(ctx, id, before, map[string]any{column: value}, true)
			return nil
		})
	}

	if origDelete := res.GetOnDelete(); origDelete != nil {
		res.OnDelete(func(ctx *filament.Context, id string) error {
			if err := origDelete(ctx, id); err != nil {
				return err
			}
			save(ctx, Log(fmt.Sprintf("Deleted %s #%s", subjectType, id)).Subject(subjectType, id).Event("deleted"))
			return nil
		})
	}

	if origBulk := res.GetOnDeleteBulk(); origBulk != nil {
		res.OnDeleteBulk(func(ctx *filament.Context, ids []string) error {
			if err := origBulk(ctx, ids); err != nil {
				return err
			}
			for _, id := range ids {
				save(ctx, Log(fmt.Sprintf("Deleted %s #%s", subjectType, id)).Subject(subjectType, id).Event("deleted").Property("bulk", true))
			}
			return nil
		})
	}

	if origRestore := res.GetOnRestore(); origRestore != nil {
		res.OnRestore(func(ctx *filament.Context, id string) error {
			if err := origRestore(ctx, id); err != nil {
				return err
			}
			save(ctx, Log(fmt.Sprintf("Restored %s #%s", subjectType, id)).Subject(subjectType, id).Event("restored"))
			return nil
		})
	}

	if origForceDelete := res.GetOnForceDelete(); origForceDelete != nil {
		res.OnForceDelete(func(ctx *filament.Context, id string) error {
			if err := origForceDelete(ctx, id); err != nil {
				return err
			}
			save(ctx, Log(fmt.Sprintf("Permanently deleted %s #%s", subjectType, id)).Subject(subjectType, id).Event("deleted").Property("force_deleted", true))
			return nil
		})
	}

	if !opts.NoHistory {
		res.History(func(ctx *filament.Context, id string, limit int) ([]filament.HistoryEntry, error) {
			acts, err := s.GetBySubject(toStdContext(ctx), subjectType, id, limit)
			if err != nil {
				return nil, err
			}
			out := make([]filament.HistoryEntry, 0, len(acts))
			for _, a := range acts {
				e := filament.HistoryEntry{ID: a.ID, Event: a.Event, Description: a.Description, CauserName: a.CauserName, At: a.CreatedAt}
				if a.Event == "updated" {
					for _, d := range a.GetDiff() {
						e.Changes = append(e.Changes, filament.HistoryChange{Field: d.Field, Old: displayValue(d.Old), New: displayValue(d.New)})
					}
				}
				out = append(out, e)
			}
			return out, nil
		})
	}
	return res
}

// changedFields compares the submitted data with the record before the
// change. Record keys are matched loosely ("CustomerName", "customer_name"),
// and only fields that really changed are returned.
func changedFields(before, data map[string]any, hidden func(string) bool) (map[string]any, map[string]any) {
	loose := func(k string) string { return strings.ToLower(strings.ReplaceAll(k, "_", "")) }
	byLoose := map[string]any{}
	for k, v := range before {
		byLoose[loose(k)] = v
	}
	oldVals, newVals := map[string]any{}, map[string]any{}
	for k, nv := range data {
		if strings.HasPrefix(k, "_") {
			continue
		}
		ov, ok := before[k]
		if !ok {
			ov, ok = byLoose[loose(k)]
		}
		if ok && sameValue(ov, nv) {
			continue
		}
		if before != nil && !ok && isBlank(nv) {
			continue
		}
		if hidden(k) {
			if isBlank(nv) {
				continue // a blank password field keeps the current one
			}
			oldVals[k], newVals[k] = "(hidden)", "(changed)"
			continue
		}
		oldVals[k], newVals[k] = ov, nv
	}
	return oldVals, newVals
}

func sameValue(a, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	if isBlank(a) && isBlank(b) {
		return true
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func isBlank(v any) bool {
	return v == nil || v == "" || fmt.Sprint(v) == "<nil>"
}

func displayValue(v any) string {
	if isBlank(v) {
		return ""
	}
	return FormatValue(v)
}

func toStdContext(ctx *filament.Context) context.Context {
	if ctx != nil && ctx.Request != nil {
		return ctx.Request.Context()
	}
	return context.Background()
}

func populateCauserFromContext(b *Builder, ctx *filament.Context) {
	if ctx == nil {
		return
	}

	if u := ctx.User(); u != nil {
		id := u.GetAuthIdentifier()
		name := extractUserName(u, id)
		b.Causer("User", id, name)
	}

	if ip := ctx.ClientIP(); ip != "" {
		b.IP(ip)
	}

	if ctx.Request != nil && ctx.Request.UserAgent() != "" {
		b.UserAgent(ctx.Request.UserAgent())
	}
}

func extractUserName(u any, fallbackID string) string {
	type nameGetter interface {
		GetName() string
	}
	if ng, ok := u.(nameGetter); ok && ng.GetName() != "" {
		return ng.GetName()
	}

	type nameFunc interface {
		Name() string
	}
	if nf, ok := u.(nameFunc); ok && nf.Name() != "" {
		return nf.Name()
	}

	v := reflect.ValueOf(u)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		for _, nameField := range []string{"Name", "Username", "FullName", "Email"} {
			f := v.FieldByName(nameField)
			if f.IsValid() && f.Kind() == reflect.String && f.String() != "" {
				return f.String()
			}
		}
	}

	if fallbackID != "" {
		return "User #" + fallbackID
	}
	return "User"
}

func recordToMap(record any) map[string]any {
	if record == nil {
		return nil
	}
	if m, ok := record.(map[string]any); ok {
		return m
	}
	data, err := json.Marshal(record)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}
