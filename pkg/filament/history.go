package filament

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golaravel/gofilament/pkg/forms"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
)

// Record history: who changed a record, when, and which fields (before and
// after). A resource gets it from a history provider, which
// activitylog.EnableAuditLog sets up; the web view/edit pages, the API and
// the Flutter app then show it.

// HistoryChange is one field's change.
type HistoryChange struct {
	Field string `json:"field"`
	Label string `json:"label,omitempty"`
	Old   string `json:"old"`
	New   string `json:"new"`
}

// HistoryEntry is one event in a record's history.
type HistoryEntry struct {
	ID          string          `json:"id"`
	Event       string          `json:"event"`
	Description string          `json:"description"`
	CauserName  string          `json:"causer_name,omitempty"`
	At          time.Time       `json:"at"`
	Changes     []HistoryChange `json:"changes,omitempty"`
}

// HistoryProvider returns the newest entries of a record (at most limit).
type HistoryProvider func(ctx *Context, id string, limit int) ([]HistoryEntry, error)

const ctxCreatedID = "filament_created_id"

// SetCreatedID lets an OnCreate handler report the new record's id, so the
// audit log can attach the "created" entry to the record.
func SetCreatedID(ctx *Context, id string) {
	if ctx != nil {
		ctx.Set(ctxCreatedID, id)
	}
}

// CreatedID returns the id reported with SetCreatedID ("" when none).
func CreatedID(ctx *Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Get(ctxCreatedID); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// History sets the provider of the record history shown on the record
// pages (see activitylog.EnableAuditLog).
func (r *BaseResource[T]) History(provider HistoryProvider) *BaseResource[T] {
	r.historyProvider = provider
	return r
}

// HistoryVisible decides who may see record history (default: everyone who
// can view the record).
func (r *BaseResource[T]) HistoryVisible(fn func(ctx *Context) bool) *BaseResource[T] {
	r.canSeeHistory = fn
	return r
}

func (r *BaseResource[T]) historyAvailable(ctx *Context) bool {
	return r.historyProvider != nil && (r.canSeeHistory == nil || r.canSeeHistory(ctx))
}

// RecordHistory returns a record's history, newest first, with field labels
// from the form. It checks that the user may view the record.
func (r *BaseResource[T]) RecordHistory(ctx *Context, id string, limit int) ([]HistoryEntry, error) {
	if !r.historyAvailable(ctx) {
		return nil, ErrForbidden
	}
	if err := r.authorizeRecord(ctx, id, nil); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	entries, err := r.historyProvider(ctx, id, limit)
	if err != nil {
		return nil, err
	}
	labels := r.formFieldLabels(ctx)
	for i := range entries {
		for j := range entries[i].Changes {
			c := &entries[i].Changes[j]
			if c.Label == "" {
				c.Label = labels[c.Field]
			}
			if c.Label == "" {
				c.Label = humanizeField(c.Field)
			}
		}
	}
	return entries, nil
}

func humanizeField(name string) string {
	name = strings.ReplaceAll(name, "_", " ")
	if name == "" {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}

// historyHTML renders the history card of a record page.
func (r *BaseResource[T]) historyHTML(ctx *Context, id string) string {
	if !r.historyAvailable(ctx) {
		return ""
	}
	entries, err := r.RecordHistory(ctx, id, 50)
	if err != nil {
		return ""
	}
	esc := html.EscapeString
	var b strings.Builder
	b.WriteString(`<details class="filament-card filament-record-history" style="margin-top: 2rem; padding: 1rem 1.25rem;" open><summary style="font-weight: 700; cursor: pointer;">History</summary>`)
	if len(entries) == 0 {
		b.WriteString(`<p style="font-size: 0.875rem; color: var(--gray-500); margin-top: 0.75rem;">No changes recorded yet.</p>`)
	}
	b.WriteString(`<ol style="list-style: none; padding: 0; margin: 0.75rem 0 0;">`)
	for _, e := range entries {
		who := e.CauserName
		if who == "" {
			who = "System"
		}
		fmt.Fprintf(&b, `<li style="padding: 0.625rem 0; border-top: 1px solid var(--gray-100);" data-history-event="%s"><div style="font-size: 0.875rem;"><strong>%s</strong> %s <span style="color: var(--gray-500);" title="%s">%s</span></div>`,
			esc(e.Event), esc(who), esc(strings.ToLower(eventVerb(e))), esc(e.At.UTC().Format(time.RFC1123)), esc(e.At.Local().Format("2006-01-02 15:04")))
		if len(e.Changes) > 0 {
			b.WriteString(`<table style="font-size: 0.8125rem; margin-top: 0.375rem; border-collapse: collapse;">`)
			for _, c := range e.Changes {
				fmt.Fprintf(&b, `<tr><td style="padding: 0.125rem 0.75rem 0.125rem 0; color: var(--gray-500);">%s</td><td style="padding: 0.125rem 0.5rem; text-decoration: line-through; color: #b91c1c;">%s</td><td style="padding: 0.125rem 0.5rem;">&rarr;</td><td style="padding: 0.125rem 0.5rem; color: #15803d;">%s</td></tr>`,
					esc(c.Label), esc(c.Old), esc(c.New))
			}
			b.WriteString(`</table>`)
		}
		b.WriteString(`</li>`)
	}
	b.WriteString(`</ol></details>`)
	return b.String()
}

func eventVerb(e HistoryEntry) string {
	switch e.Event {
	case "created":
		return "created this record"
	case "updated":
		return "changed"
	case "deleted":
		return "deleted this record"
	case "restored":
		return "restored this record"
	}
	if e.Description != "" {
		return e.Description
	}
	return e.Event
}

func (p *Panel) mountHistoryAPI(r *router.Router, res ResourceInterface, resPath string, preflight func(c *gcontext.Context) error) {
	h, ok := res.(interface {
		RecordHistory(ctx *Context, id string, limit int) ([]HistoryEntry, error)
		historyAvailable(ctx *Context) bool
	})
	if !ok {
		return
	}
	r.OPTIONS(resPath+"/:id/history", preflight)
	r.GET(resPath+"/:id/history", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		if !h.historyAvailable(ctx) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Not found"})
		}
		limit, _ := strconv.Atoi(c.Query("limit"))
		entries, err := h.RecordHistory(ctx, c.Param("id"), limit)
		if err != nil {
			return p.apiErrorStatus(c, err, http.StatusNotFound)
		}
		if entries == nil {
			entries = []HistoryEntry{}
		}
		return c.JSON(http.StatusOK, map[string]any{"entries": entries})
	})
}

// formFieldLabels maps the form's field names to their labels.
func (r *BaseResource[T]) formFieldLabels(ctx *Context) map[string]string {
	labels := map[string]string{}
	var walk func(ns []forms.Node)
	walk = func(ns []forms.Node) {
		for _, n := range ns {
			if n.Kind == "field" && n.Name != "" && n.Label != "" {
				labels[n.Name] = n.Label
			}
			walk(n.Children)
		}
	}
	walk(r.DescribeForm(ctx))
	return labels
}

// recordFormValues is the record as form values: its JSON keys, plus each
// form field matched loosely ("FullName" fills "full_name").
func (r *BaseResource[T]) recordFormValues(ctx *Context, record any) map[string]any {
	m := recordToMap(record)
	loose := func(k string) string { return strings.ToLower(strings.ReplaceAll(k, "_", "")) }
	byLoose := map[string]any{}
	for k, v := range m {
		byLoose[loose(k)] = v
	}
	var walk func(ns []forms.Node)
	walk = func(ns []forms.Node) {
		for _, n := range ns {
			if n.Kind == "field" && n.Type == "password" {
				delete(m, n.Name) // never echo a stored password (hash) into the page
			} else if n.Kind == "field" && n.Name != "" {
				if _, ok := m[n.Name]; !ok {
					if v, ok := byLoose[loose(n.Name)]; ok {
						m[n.Name] = v
					}
				}
			}
			walk(n.Children)
		}
	}
	walk(r.DescribeForm(ctx))
	return m
}
