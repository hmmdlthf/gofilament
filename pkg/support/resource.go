package support

import (
	"context"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
)

// NewSupportResource creates a pre-configured Filament resource for managing support tickets and live chats.
func NewSupportResource(store ...Store) *filament.BaseResource[*Ticket] {
	s := GetDefaultStore()
	if len(store) > 0 && store[0] != nil {
		s = store[0]
	}

	res := filament.NewResource[*Ticket]("support-tickets")
	res.ModelName("Support Ticket").
		PluralModelName("Support Tickets").
		NavigationGroup("Support & Concierge").
		NavigationIcon(`<svg style="width: 1.25rem; height: 1.25rem;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z"/></svg>`).
		RecordID(func(t *Ticket) string {
			if t == nil {
				return ""
			}
			return t.ID
		})

	res.Table(func(t *tables.Table[*Ticket]) *tables.Table[*Ticket] {
		return t.
			Columns(
				tables.TextColumn[*Ticket]("id").
					Label("Ticket ID").
					Value(func(tk *Ticket) any {
						return tk.ID
					}),
				tables.BadgeColumn[*Ticket]("mode").
					Label("Mode").
					Colors(map[string]string{
						"ai":   "info",
						"live": "success",
					}).
					Value(func(tk *Ticket) any {
						if tk.Mode == ModeLive {
							return "Live Chat"
						}
						return "AI Chat"
					}),
				tables.TextColumn[*Ticket]("user_name").
					Label("User / Candidate").
					Value(func(tk *Ticket) any {
						return tk.UserName
					}),
				tables.TextColumn[*Ticket]("subject").
					Label("Subject").
					Value(func(tk *Ticket) any {
						return tk.Subject
					}),
				tables.TextColumn[*Ticket]("department").
					Label("Department").
					Value(func(tk *Ticket) any {
						if tk.Department == "" {
							return "General"
						}
						return tk.Department
					}),
				tables.BadgeColumn[*Ticket]("status").
					Label("Status").
					Colors(map[string]string{
						"open":     "warning",
						"pending":  "info",
						"resolved": "success",
						"closed":   "danger",
					}).
					Value(func(tk *Ticket) any {
						return strings.ToLower(tk.Status)
					}),
				tables.TextColumn[*Ticket]("preview").
					Label("Latest Message").
					Value(func(tk *Ticket) any {
						return tk.LastMessagePreview()
					}),
				tables.TextColumn[*Ticket]("created_at").
					Label("Created").
					Value(func(tk *Ticket) any {
						return tk.TimeAgo()
					}),
			).
			Filters(
				tables.SelectFilter("mode").
					Label("Filter by Mode").
					Options(map[string]string{
						"ai":   "AI Bot",
						"live": "Human Live Chat",
					}),
				tables.SelectFilter("status").
					Label("Filter by Status").
					Options(map[string]string{
						"open":     "Open",
						"pending":  "Pending",
						"resolved": "Resolved",
						"closed":   "Closed",
					}),
			)
	})

	// Bind data fetchers to the support store
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Ticket, int, error) {
		c := context.Background()
		if ctx != nil && ctx.Request != nil {
			c = ctx.Request.Context()
		}
		list, err := s.ListTickets(c)
		if err != nil {
			return nil, 0, err
		}

		// Apply filters if provided
		var filtered []*Ticket
		for _, tk := range list {
			if mode, ok := filters["mode"]; ok && mode != "" {
				if tk.Mode != mode {
					continue
				}
			}
			if status, ok := filters["status"]; ok && status != "" {
				if !strings.EqualFold(tk.Status, status) {
					continue
				}
			}
			if search != "" {
				sLower := strings.ToLower(search)
				if !strings.Contains(strings.ToLower(tk.Subject), sLower) &&
					!strings.Contains(strings.ToLower(tk.UserName), sLower) &&
					!strings.Contains(strings.ToLower(tk.ID), sLower) {
					continue
				}
			}
			filtered = append(filtered, tk)
		}

		return filtered, len(filtered), nil
	})

	res.OnFind(func(ctx *filament.Context, id string) (*Ticket, error) {
		c := context.Background()
		if ctx != nil && ctx.Request != nil {
			c = ctx.Request.Context()
		}
		return s.GetTicket(c, id)
	})

	return res
}
