package support

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/forms"
	"github.com/golaravel/gofilament/pkg/tables"
)

// Scope enums for support resource
const (
	ScopeSuperAdmin  = "super_admin"
	ScopeTenantAdmin = "tenant_admin"
	ScopeStandard    = "standard"
)

// SupportResourceOption configures the support resource behavior.
type SupportResourceOption func(cfg *SupportResourceConfig)

// SupportResourceConfig holds options for the support admin resource.
type SupportResourceConfig struct {
	Store                      Store
	Scope                      string // ScopeSuperAdmin | ScopeTenantAdmin | ScopeStandard
	TenantScoping              bool
	SuperAdminPanelID          string
	AllowEscalateToSuperAdmin  bool
}

// WithStore sets the ticket repository store.
func WithStore(s Store) SupportResourceOption {
	return func(cfg *SupportResourceConfig) {
		cfg.Store = s
	}
}

// WithScope sets the hierarchy scope (ScopeSuperAdmin, ScopeTenantAdmin, ScopeStandard).
func WithScope(scope string) SupportResourceOption {
	return func(cfg *SupportResourceConfig) {
		cfg.Scope = scope
	}
}

// WithTenantScoping enables or disables tenant scoping.
func WithTenantScoping(scoping bool) SupportResourceOption {
	return func(cfg *SupportResourceConfig) {
		cfg.TenantScoping = scoping
	}
}

// WithSuperAdminPanelID sets the panel ID for upstream escalation.
func WithSuperAdminPanelID(id string) SupportResourceOption {
	return func(cfg *SupportResourceConfig) {
		cfg.SuperAdminPanelID = id
		cfg.AllowEscalateToSuperAdmin = true
	}
}

// NewSupportResource creates a pre-configured Filament resource for managing support tickets,
// live agent chats, and multilingual voice feature requests across Super Admin, Tenant, and standard panels.
func NewSupportResource(opts ...SupportResourceOption) *filament.BaseResource[*Ticket] {
	cfg := &SupportResourceConfig{
		Store:             GetDefaultStore(),
		Scope:             ScopeStandard,
		TenantScoping:     false,
		SuperAdminPanelID: "super-admin",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	service := NewService(cfg.Store)

	res := filament.NewResource[*Ticket]("support-tickets")
	res.ModelName("Support & Feature Request").
		PluralModelName("Support & Feature Requests").
		NavigationGroup("Support & Concierge").
		NavigationIcon(`<svg style="width: 1.25rem; height: 1.25rem;" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z"/></svg>`).
		RecordID(func(t *Ticket) string {
			if t == nil {
				return ""
			}
			return t.ID
		})

	res.Table(func(t *tables.Table[*Ticket]) *tables.Table[*Ticket] {
		cols := []tables.ColumnRenderer[*Ticket]{
			tables.TextColumn[*Ticket]("id").
				Label("Ticket ID").
				Sortable().
				Searchable().
				Value(func(tk *Ticket) any {
					return tk.ID
				}),
			tables.BadgeColumn[*Ticket]("category").
				Label("Category").
				Sortable().
				Colors(map[string]string{
					CategoryFeatureRequest: "info",
					CategorySupport:        "primary",
					CategoryBug:            "danger",
					CategoryBilling:        "warning",
					CategoryGeneral:        "gray",
				}).
				Value(func(tk *Ticket) any {
					switch tk.Category {
					case CategoryFeatureRequest:
						return "🎙️ Feature Request"
					case CategoryBug:
						return "🐛 Bug Report"
					case CategoryBilling:
						return "💳 Billing"
					default:
						return "💬 Support"
					}
				}),
			tables.BadgeColumn[*Ticket]("mode").
				Label("Mode").
				Sortable().
				Colors(map[string]string{
					ModeAI:   "gray",
					ModeLive: "success",
				}).
				Value(func(tk *Ticket) any {
					if tk.Mode == ModeLive {
						return "👨‍💼 Live Agent"
					}
					return "🤖 AI Bot"
				}),
			tables.BadgeColumn[*Ticket]("status").
				Label("Status").
				Sortable().
				Colors(map[string]string{
					StatusOpen:      "warning",
					StatusPending:   "info",
					StatusEscalated: "danger",
					StatusResolved:  "success",
					StatusClosed:    "gray",
				}).
				Value(func(tk *Ticket) any {
					if tk.Status == StatusEscalated {
						return "⚡ Escalated"
					}
					return strings.Title(tk.Status)
				}),
			tables.TextColumn[*Ticket]("subject").
				Label("Subject / Request").
				Sortable().
				Searchable().
				Weight("bold").
				Value(func(tk *Ticket) any {
					return tk.Subject
				}),
			tables.TextColumn[*Ticket]("language").
				Label("Language").
				Sortable().
				Value(func(tk *Ticket) any {
					flag := LanguageFlags[tk.Language]
					if flag == "" {
						flag = "🌐"
					}
					name := SupportedLanguages[tk.Language]
					if name == "" {
						name = tk.Language
					}
					return fmt.Sprintf("%s %s", flag, name)
				}),
			// Audio Player Column: if ticket has voice recording, provides inline audio player
			tables.TextColumn[*Ticket]("voice_audio").
				Label("Voice Recording").
				Value(func(tk *Ticket) any {
					if tk.Voice == nil {
						return "—"
					}
					dur := fmt.Sprintf("%.1fs", tk.Voice.DurationSeconds)
					if tk.Voice.AudioBase64 != "" {
						return fmt.Sprintf(`🎙️ Voice (%s) • Transcribed`, dur)
					}
					return fmt.Sprintf(`🎙️ Voice (%s)`, dur)
				}),
			// Feasibility column for feature requests
			tables.BadgeColumn[*Ticket]("feasibility").
				Label("AI Feasibility").
				Colors(map[string]string{
					FeasibilityExisting:    "info",
					FeasibilitySimple:      "success",
					FeasibilityComplex:     "warning",
					FeasibilityUnfeasible:  "danger",
					FeasibilityUnderReview: "gray",
				}).
				Value(func(tk *Ticket) any {
					if tk.Category != CategoryFeatureRequest {
						return "—"
					}
					switch tk.Feasibility {
					case FeasibilityExisting:
						return "Existing in System"
					case FeasibilitySimple:
						return "Simple (Planned)"
					case FeasibilityComplex:
						return "Complex (Review)"
					default:
						return "Under Review"
					}
				}),
		}

		// Super Admin panel sees which tenant the ticket belongs to
		if cfg.Scope == ScopeSuperAdmin {
			cols = append(cols, tables.TextColumn[*Ticket]("tenant_id").
				Label("Tenant / Workspace").
				Sortable().
				Value(func(tk *Ticket) any {
					if tk.TenantID == "" {
						return "Global"
					}
					return tk.TenantID
				}))
		}

		cols = append(cols,
			tables.TextColumn[*Ticket]("user_name").
				Label("User").
				Value(func(tk *Ticket) any {
					return tk.UserName
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
		)

		return t.
			Columns(cols...).
			Filters(
				tables.SelectFilter("category").
					Label("Category").
					Options(map[string]string{
						CategoryFeatureRequest: "Feature Requests",
						CategorySupport:        "Support Inquiries",
						CategoryBug:            "Bug Reports",
						CategoryBilling:        "Billing",
					}),
				tables.SelectFilter("mode").
					Label("Mode").
					Options(map[string]string{
						ModeAI:   "AI Bot",
						ModeLive: "Live Human Agent",
					}),
				tables.SelectFilter("status").
					Label("Status").
					Options(map[string]string{
						StatusOpen:      "Open",
						StatusPending:   "Pending Customer",
						StatusEscalated: "Escalated to Live",
						StatusResolved:  "Resolved",
						StatusClosed:    "Closed",
					}),
				tables.SelectFilter("feasibility").
					Label("AI Feasibility").
					Options(map[string]string{
						FeasibilityExisting: "Existing Feature",
						FeasibilitySimple:   "Simple / Planned",
						FeasibilityComplex:  "Complex / Review",
					}),
				tables.SelectFilter("language").
					Label("Language").
					Options(map[string]string{
						"en": "English",
						"si": "Sinhala (සිංහල)",
						"ta": "Tamil (தமிழ்)",
						"ar": "Arabic (العربية)",
						"es": "Spanish (Español)",
						"de": "German (Deutsch)",
					}),
			).
			Actions(
				// Action 1: Reply / View Conversation
				tables.Action[*Ticket]("reply").
					Label("View & Reply").
					Icon("chat").
					Color("primary").
					URL(func(record *Ticket) string {
						if record == nil {
							return ""
						}
						return "/admin/support-tickets/" + record.ID
					}),

				// Action 2: Escalate to Live Support (if currently in ModeAI)
				tables.Action[*Ticket]("escalate_live").
					Label("Escalate to Live").
					Icon("lightning-bolt").
					Color("warning").
					RequiresConfirmation().
					Visible(func(record *Ticket) bool {
						return record != nil && record.Mode == ModeAI
					}).
					Action(func(record *Ticket) error {
						return service.EscalateToLive(context.Background(), record.ID, "Escalated by panel operator")
					}),

				// Action 3: Escalate to Super Admin (for Tenant Admins)
				tables.Action[*Ticket]("escalate_super_admin").
					Label("Escalate to Super Admin").
					Icon("arrow-circle-up").
					Color("danger").
					RequiresConfirmation().
					Visible(func(record *Ticket) bool {
						return cfg.Scope == ScopeTenantAdmin || cfg.AllowEscalateToSuperAdmin
					}).
					Action(func(record *Ticket) error {
						return service.EscalateToSuperAdmin(context.Background(), record.ID, cfg.SuperAdminPanelID, "Tenant admin requested platform support")
					}),

				// Action 4: Resolve Ticket
				tables.Action[*Ticket]("resolve").
					Label("Resolve").
					Icon("check-circle").
					Color("success").
					RequiresConfirmation().
					Visible(func(record *Ticket) bool {
						return record != nil && record.Status != StatusResolved && record.Status != StatusClosed
					}).
					Action(func(record *Ticket) error {
						return service.ResolveTicket(context.Background(), record.ID, "Resolved via administrative action")
					}),
			)
	})

	// Data Fetching Bound to Store
	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*Ticket, int, error) {
		c := context.Background()
		if ctx != nil && ctx.Request != nil {
			c = ctx.Request.Context()
		}

		filter := TicketFilter{
			Search:     search,
			Category:   filters["category"],
			Mode:       filters["mode"],
			Status:     filters["status"],
			Department: filters["department"],
		}

		// Auto-scope by tenant if in Tenant Admin mode or TenantScoping is enabled
		if (cfg.Scope == ScopeTenantAdmin || cfg.TenantScoping) && ctx != nil {
			if tenant := ctx.Tenant(); tenant != "" {
				filter.TenantID = tenant
			}
		}

		list, err := cfg.Store.ListTickets(c, filter)
		if err != nil {
			return nil, 0, err
		}

		// Apply in-memory secondary filters
		var filtered []*Ticket
		for _, tk := range list {
			if feas, ok := filters["feasibility"]; ok && feas != "" {
				if tk.Feasibility != feas {
					continue
				}
			}
			if lang, ok := filters["language"]; ok && lang != "" {
				if tk.Language != lang {
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
		return cfg.Store.GetTicket(c, id)
	})

	// Detailed View Representation with Audio Player & Transcripts
	res.Form(func(f *forms.Form) *forms.Form {
		return f.Schema(
			forms.Section("Ticket Overview & Classification").Schema(
				forms.Grid(3).Schema(
					forms.TextInput("id").Label("Ticket ID").Disabled(),
					forms.TextInput("category").Label("Category").Disabled(),
					forms.TextInput("status").Label("Status").Disabled(),
					forms.TextInput("mode").Label("Mode").Disabled(),
					forms.TextInput("language").Label("Customer Language").Disabled(),
					forms.TextInput("feasibility").Label("AI Feasibility Rating").Disabled(),
				),
				forms.TextInput("subject").Label("Subject / Request").Disabled(),
			),
			forms.Section("Customer Information").Schema(
				forms.Grid(2).Schema(
					forms.TextInput("user_name").Label("Customer Name").Disabled(),
					forms.TextInput("user_email").Label("Customer Email").Disabled(),
				),
			),
			forms.Section("Voice Recording & Translation").
				Description("Preserved original voice recording and multilingual translations").
				Schema(
					forms.Textarea("complexity_notes").
						Label("Voice Analysis & English Translation").
						Disabled().
						Rows(3),
				),
		)
	})

	return res
}

// RenderVoicePlayerHTML generates an HTML5 audio player widget for table or detail views.
func RenderVoicePlayerHTML(vr *VoiceRecording) string {
	if vr == nil {
		return ""
	}
	dur := fmt.Sprintf("%.1fs", vr.DurationSeconds)
	if vr.AudioBase64 != "" {
		dataURI := fmt.Sprintf("data:%s;base64,%s", vr.MimeType, vr.AudioBase64)
		return fmt.Sprintf(`
<div style="display: flex; align-items: center; gap: 8px;">
  <audio controls preload="none" style="height: 32px; max-width: 220px;">
    <source src="%s" type="%s">
    Your browser does not support audio playback.
  </audio>
  <span style="font-size: 11px; color: var(--text-muted);">(%s • %s)</span>
</div>`, dataURI, html.EscapeString(vr.MimeType), dur, strings.ToUpper(vr.Language))
	}
	return fmt.Sprintf(`<span>🎙️ Audio (%s • %s)</span>`, dur, strings.ToUpper(vr.Language))
}
