package activitylog

import (
	"context"
	"fmt"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/infolists"
	"github.com/golaravel/gofilament/pkg/tables"
)

// NewActivityResource creates a pre-configured, production-ready Filament resource for auditing activities.
func NewActivityResource(store ...Store) *filament.BaseResource[*Activity] {
	s := GetDefaultStore()
	if len(store) > 0 && store[0] != nil {
		s = store[0]
	}

	res := filament.NewResource[*Activity]("activity-logs")
	res.ModelName("Audit Log").
		PluralModelName("Audit Logs").
		NavigationGroup("System").
		NavigationIcon(`<svg style="width: 1.25rem; height: 1.25rem;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>`).
		RecordID(func(a *Activity) string {
			if a == nil {
				return ""
			}
			return a.ID
		})

	res.Table(func(t *tables.Table[*Activity]) *tables.Table[*Activity] {
		return t.
			Columns(
				tables.BadgeColumn[*Activity]("event").
					Label("Event").
					Colors(map[string]string{
						"created":  "success",
						"updated":  "primary",
						"deleted":  "danger",
						"restored": "info",
						"approved": "success",
						"rejected": "danger",
						"login":    "warning",
					}).
					Value(func(a *Activity) any {
						return strings.ToLower(a.Event)
					}),
				tables.TextColumn[*Activity]("subject").
					Label("Subject").
					Value(func(a *Activity) any {
						if a.SubjectType == "" && a.SubjectID == "" {
							return "—"
						}
						return fmt.Sprintf("%s #%s", a.SubjectType, a.SubjectID)
					}),
				tables.TextColumn[*Activity]("causer").
					Label("Causer").
					Value(func(a *Activity) any {
						if a.CauserName != "" {
							return a.CauserName
						}
						if a.CauserID != "" {
							return "User #" + a.CauserID
						}
						return "System"
					}),
				tables.TextColumn[*Activity]("description").
					Label("Description").
					Value(func(a *Activity) any {
						return a.Description
					}),
				tables.TextColumn[*Activity]("ip_address").
					Label("IP Address").
					Value(func(a *Activity) any {
						if a.IPAddress == "" {
							return "—"
						}
						return a.IPAddress
					}),
				tables.TextColumn[*Activity]("created_at").
					Label("Time").
					Value(func(a *Activity) any {
						return a.TimeAgo()
					}),
			).
			Filters(
				tables.SelectFilter("event").
					Label("Filter by Event").
					Options(map[string]string{
						"created":  "Created",
						"updated":  "Updated",
						"deleted":  "Deleted",
						"restored": "Restored",
						"approved": "Approved",
						"rejected": "Rejected",
						"login":    "Login",
					}),
			)
	})

	res.Infolist(func(il *infolists.Infolist) *infolists.Infolist {
		entry := TimelineEntry("activities")
		return il.Schema(
			infolists.Section("Audit Record Timeline").Schema(
				entry,
			),
		)
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Activity, int, error) {
		filter := Filter{
			Search:  search,
			Page:    page,
			PerPage: perPage,
		}
		if ev, ok := filters["event"]; ok {
			filter.Event = ev
		}
		if st, ok := filters["subject_type"]; ok {
			filter.SubjectType = st
		}
		c := context.Background()
		if ctx != nil && ctx.Request != nil {
			c = ctx.Request.Context()
		}
		return s.Query(c, filter)
	})

	res.OnFind(func(ctx *filament.Context, id string) (*Activity, error) {
		c := context.Background()
		if ctx != nil && ctx.Request != nil {
			c = ctx.Request.Context()
		}
		return s.GetByID(c, id)
	})

	return res
}
