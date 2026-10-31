package activitylog

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/golaravel/gofilament/pkg/filament"
)

// EnableAuditLog instruments a Filament Resource with automatic CRUD audit trail logging.
func EnableAuditLog[T any](res *filament.BaseResource[T], store ...Store) *filament.BaseResource[T] {
	s := GetDefaultStore()
	if len(store) > 0 && store[0] != nil {
		s = store[0]
	}

	subjectType := res.GetModelName()
	if subjectType == "" {
		subjectType = res.Slug()
	}

	origCreate := res.GetOnCreate()
	if origCreate != nil {
		res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
			err := origCreate(ctx, data)
			if err != nil {
				return err
			}

			var subjID string
			if idVal, ok := data["id"]; ok {
				subjID = fmt.Sprintf("%v", idVal)
			}

			desc := fmt.Sprintf("Created %s", subjectType)
			if subjID != "" {
				desc = fmt.Sprintf("Created %s #%s", subjectType, subjID)
			}

			b := Log(desc).
				Store(s).
				Subject(subjectType, subjID).
				Event("created").
				New(data)

			populateCauserFromContext(b, ctx)
			_, _ = b.Save(toStdContext(ctx))
			return nil
		})
	}

	origUpdate := res.GetOnUpdate()
	if origUpdate != nil {
		res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
			var oldData map[string]any
			if origFind := res.GetOnFind(); origFind != nil {
				if oldRecord, err := origFind(ctx, id); err == nil {
					oldData = recordToMap(oldRecord)
				}
			}

			err := origUpdate(ctx, id, data)
			if err != nil {
				return err
			}

			b := Log(fmt.Sprintf("Updated %s #%s", subjectType, id)).
				Store(s).
				Subject(subjectType, id).
				Event("updated").
				WithDiff(oldData, data)

			populateCauserFromContext(b, ctx)
			_, _ = b.Save(toStdContext(ctx))
			return nil
		})
	}

	origDelete := res.GetOnDelete()
	if origDelete != nil {
		res.OnDelete(func(ctx *filament.Context, id string) error {
			err := origDelete(ctx, id)
			if err != nil {
				return err
			}

			b := Log(fmt.Sprintf("Deleted %s #%s", subjectType, id)).
				Store(s).
				Subject(subjectType, id).
				Event("deleted")

			populateCauserFromContext(b, ctx)
			_, _ = b.Save(toStdContext(ctx))
			return nil
		})
	}

	return res
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
