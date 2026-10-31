package activitylog

import (
	"context"
	"fmt"
	"html"
	"reflect"
	"strings"
)

// TimelineComponent renders an interactive vertical timeline of audit trail activities.
type TimelineComponent struct {
	name             string
	title            string
	activities       []*Activity
	store            Store
	subjectType      string
	subjectID        string
	emptyState       string
	limit            int
	filterable       bool
	collapsibleDiffs bool
}

// Timeline creates a TimelineComponent populated with initial activities.
func Timeline(activities ...*Activity) *TimelineComponent {
	return &TimelineComponent{
		name:             "activities",
		title:            "Activity History & Audit Trail",
		activities:       activities,
		store:            GetDefaultStore(),
		emptyState:       "No activity recorded yet.",
		limit:            50,
		filterable:       true,
		collapsibleDiffs: true,
	}
}

// TimelineEntry creates a TimelineComponent intended for use within an Infolist schema.
func TimelineEntry(name string, opts ...func(*TimelineComponent)) *TimelineComponent {
	tc := Timeline()
	tc.name = name
	for _, opt := range opts {
		opt(tc)
	}
	return tc
}

// GetName satisfies the infolists.Entry interface.
func (tc *TimelineComponent) GetName() string {
	return tc.name
}

// Title sets the heading title for the timeline container.
func (tc *TimelineComponent) Title(title string) *TimelineComponent {
	tc.title = title
	return tc
}

// ForSubject binds the timeline to query a specific entity subject.
func (tc *TimelineComponent) ForSubject(subjectType string, subjectID ...string) *TimelineComponent {
	tc.subjectType = subjectType
	if len(subjectID) > 0 {
		tc.subjectID = subjectID[0]
	}
	return tc
}

// Store overrides the activity store used to resolve activities.
func (tc *TimelineComponent) Store(store Store) *TimelineComponent {
	if store != nil {
		tc.store = store
	}
	return tc
}

// EmptyState customizes the message displayed when no activities exist.
func (tc *TimelineComponent) EmptyState(text string) *TimelineComponent {
	tc.emptyState = text
	return tc
}

// Limit sets the maximum number of activity items to display.
func (tc *TimelineComponent) Limit(limit int) *TimelineComponent {
	tc.limit = limit
	return tc
}

// Filterable toggles client-side live search and event filtering controls.
func (tc *TimelineComponent) Filterable(enabled ...bool) *TimelineComponent {
	if len(enabled) > 0 {
		tc.filterable = enabled[0]
	} else {
		tc.filterable = true
	}
	return tc
}

// CollapsibleDiffs toggles whether changesets render inside expandable accordions.
func (tc *TimelineComponent) CollapsibleDiffs(enabled ...bool) *TimelineComponent {
	if len(enabled) > 0 {
		tc.collapsibleDiffs = enabled[0]
	} else {
		tc.collapsibleDiffs = true
	}
	return tc
}

// Render satisfies infolists.Entry by extracting activities from the record or querying the store.
func (tc *TimelineComponent) Render(record any) string {
	activities := tc.resolveActivities(record)
	return tc.RenderActivities(activities)
}

// RenderHTML renders the timeline using its pre-configured activities.
func (tc *TimelineComponent) RenderHTML() string {
	return tc.RenderActivities(tc.activities)
}

func (tc *TimelineComponent) resolveActivities(record any) []*Activity {
	// If explicit activities were supplied, use them
	if len(tc.activities) > 0 {
		return tc.activities
	}

	// Try extracting from record if record is map or struct
	if record != nil {
		if m, ok := record.(map[string]any); ok {
			if val, exists := m[tc.name]; exists {
				if acts, ok := val.([]*Activity); ok && len(acts) > 0 {
					return acts
				}
			}
			if val, exists := m["activities"]; exists {
				if acts, ok := val.([]*Activity); ok && len(acts) > 0 {
					return acts
				}
			}
		}

		// Try reflection for struct fields named Activities or matching tc.name
		v := reflect.ValueOf(record)
		if v.Kind() == reflect.Pointer {
			v = v.Elem()
		}
		if v.Kind() == reflect.Struct {
			for _, fieldName := range []string{tc.name, "Activities", "ActivityLog"} {
				f := v.FieldByName(fieldName)
				if f.IsValid() {
					if acts, ok := f.Interface().([]*Activity); ok && len(acts) > 0 {
						return acts
					}
				}
			}
		}
	}

	// If subject is bound or discoverable from record ID, query the store
	subjectType := tc.subjectType
	subjectID := tc.subjectID
	if subjectID == "" && record != nil {
		if m, ok := record.(map[string]any); ok {
			if idVal, exists := m["id"]; exists {
				subjectID = fmt.Sprintf("%v", idVal)
			}
		} else {
			v := reflect.ValueOf(record)
			if v.Kind() == reflect.Pointer {
				v = v.Elem()
			}
			if v.Kind() == reflect.Struct {
				idField := v.FieldByName("ID")
				if idField.IsValid() {
					subjectID = fmt.Sprintf("%v", idField.Interface())
				}
			}
		}
	}

	if subjectType != "" && subjectID != "" && tc.store != nil {
		acts, err := tc.store.GetBySubject(context.Background(), subjectType, subjectID, tc.limit)
		if err == nil && len(acts) > 0 {
			return acts
		}
	}

	return tc.activities
}

// RenderActivities produces the full HTML string for a list of activities.
func (tc *TimelineComponent) RenderActivities(activities []*Activity) string {
	if len(activities) > tc.limit && tc.limit > 0 {
		activities = activities[:tc.limit]
	}

	var filterBarHTML string
	if tc.filterable && len(activities) > 0 {
		filterBarHTML = `
<div class="timeline-filter-bar" style="display: flex; gap: 0.75rem; align-items: center; margin-bottom: 1.25rem; flex-wrap: wrap;">
  <div style="flex: 1; min-width: 12rem;">
    <input type="text" x-model="searchTerm" placeholder="Filter audit trail events or descriptions..." 
      class="form-control" style="width: 100%; font-size: 0.8125rem; padding: 0.375rem 0.625rem; border-radius: 0.375rem; border: 1px solid var(--gray-300);" />
  </div>
  <div>
    <select x-model="filterEvent" class="form-control" style="font-size: 0.8125rem; padding: 0.375rem 0.625rem; border-radius: 0.375rem; border: 1px solid var(--gray-300);">
      <option value="">All Events</option>
      <option value="created">Created</option>
      <option value="updated">Updated</option>
      <option value="deleted">Deleted</option>
      <option value="restored">Restored</option>
      <option value="approved">Approved</option>
      <option value="rejected">Rejected</option>
      <option value="login">Login</option>
    </select>
  </div>
</div>`
	}

	var itemsHTML strings.Builder
	if len(activities) == 0 {
		itemsHTML.WriteString(fmt.Sprintf(`
<div style="text-align: center; padding: 2rem 1rem; color: var(--gray-500); font-size: 0.875rem;">
  <svg style="width: 2.5rem; height: 2.5rem; margin: 0 auto 0.75rem auto; color: var(--gray-400);" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
  </svg>
  <p>%s</p>
</div>`, html.EscapeString(tc.emptyState)))
	} else {
		for i, act := range activities {
			itemsHTML.WriteString(tc.renderActivityItem(act, i == len(activities)-1))
		}
	}

	headingHTML := ""
	if tc.title != "" {
		headingHTML = fmt.Sprintf(`
<div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 1rem; border-bottom: 1px solid var(--gray-200); padding-bottom: 0.625rem;">
  <h3 style="font-size: 1.0625rem; font-weight: 600; color: var(--gray-900); margin: 0; display: flex; align-items: center; gap: 0.5rem;">
    <svg style="width: 1.25rem; height: 1.25rem; color: var(--primary-600);" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
    %s
  </h3>
  <span style="font-size: 0.75rem; font-weight: 600; background-color: var(--gray-100); color: var(--gray-600); padding: 0.125rem 0.5rem; border-radius: 9999px;">
    %d events
  </span>
</div>`, html.EscapeString(tc.title), len(activities))
	}

	return fmt.Sprintf(`
<div class="filament-card activity-timeline-container" x-data="{ searchTerm: '', filterEvent: '' }" style="background-color: #ffffff; border-radius: 0.5rem; padding: 1.25rem; border: 1px solid var(--gray-200); margin-bottom: 1.25rem;">
  %s
  %s
  <div class="activity-timeline-spine" style="position: relative; margin-left: 1rem; padding-left: 1.5rem; border-left: 2px solid var(--gray-200);">
    %s
  </div>
</div>`, headingHTML, filterBarHTML, itemsHTML.String())
}

func (tc *TimelineComponent) renderActivityItem(a *Activity, isLast bool) string {
	eventLower := strings.ToLower(a.Event)
	eventUpper := strings.ToUpper(a.Event)

	// Node styling & icon
	var iconBg, iconBorder, iconColor, iconSVG string
	switch eventLower {
	case "created", "added", "inserted":
		iconBg = "#ecfdf5"
		iconBorder = "#a7f3d0"
		iconColor = "#059669"
		iconSVG = `<svg style="width: 1rem; height: 1rem;" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M10 3a1 1 0 011 1v5h5a1 1 0 110 2h-5v5a1 1 0 11-2 0v-5H4a1 1 0 110-2h5V4a1 1 0 011-1z" clip-rule="evenodd"/></svg>`
	case "updated", "modified", "edited":
		iconBg = "#eef2ff"
		iconBorder = "#c7d2fe"
		iconColor = "#4f46e5"
		iconSVG = `<svg style="width: 1rem; height: 1rem;" viewBox="0 0 20 20" fill="currentColor"><path d="M13.586 3.586a2 2 0 112.828 2.828l-.793.793-2.828-2.828.793-.793zM11.379 5.793L3 14.172V17h2.828l8.38-8.379-2.83-2.828z"/></svg>`
	case "deleted", "removed", "destroyed":
		iconBg = "#fef2f2"
		iconBorder = "#fecaca"
		iconColor = "#e11d48"
		iconSVG = `<svg style="width: 1rem; height: 1rem;" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd"/></svg>`
	case "restored":
		iconBg = "#ecfeff"
		iconBorder = "#a5f3fc"
		iconColor = "#0891b2"
		iconSVG = `<svg style="width: 1rem; height: 1rem;" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1z" clip-rule="evenodd"/></svg>`
	case "approved", "completed":
		iconBg = "#f0fdf4"
		iconBorder = "#bbf7d0"
		iconColor = "#16a34a"
		iconSVG = `<svg style="width: 1rem; height: 1rem;" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/></svg>`
	case "rejected", "cancelled":
		iconBg = "#fef2f2"
		iconBorder = "#fca5a5"
		iconColor = "#dc2626"
		iconSVG = `<svg style="width: 1rem; height: 1rem;" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"/></svg>`
	case "login":
		iconBg = "#fffbeb"
		iconBorder = "#fde68a"
		iconColor = "#d97706"
		iconSVG = `<svg style="width: 1rem; height: 1rem;" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M18 8a6 6 0 01-7.743 5.743L10 14l-1 1-1 1H6v2H2v-4l4.257-4.257A6 6 0 1118 8zm-6-4a1 1 0 100 2 2 2 0 012 2 1 1 0 102 0 4 4 0 00-4-4z" clip-rule="evenodd"/></svg>`
	default:
		iconBg = "#f9fafb"
		iconBorder = "#e5e7eb"
		iconColor = "#6b7280"
		iconSVG = `<svg style="width: 1rem; height: 1rem;" viewBox="0 0 20 20" fill="currentColor"><circle cx="10" cy="10" r="4"/></svg>`
	}

	// Causer representation
	causerHTML := ""
	if a.CauserName != "" || a.CauserID != "" {
		displayName := a.CauserName
		if displayName == "" {
			displayName = "User #" + a.CauserID
		}
		initials := GetInitials(displayName)

		avatarElem := fmt.Sprintf(`<span style="width: 1.5rem; height: 1.5rem; border-radius: 9999px; background-color: var(--primary-100); color: var(--primary-700); font-weight: 700; font-size: 0.6875rem; display: inline-flex; align-items: center; justify-content: center;">%s</span>`, html.EscapeString(initials))
		if a.CauserAvatar != "" {
			avatarElem = fmt.Sprintf(`<img src="%s" alt="%s" style="width: 1.5rem; height: 1.5rem; border-radius: 9999px; object-fit: cover;" />`, html.EscapeString(a.CauserAvatar), html.EscapeString(displayName))
		}

		causerTypeBadge := ""
		if a.CauserType != "" {
			causerTypeBadge = fmt.Sprintf(`<span style="font-size: 0.6875rem; color: var(--gray-400); margin-left: 0.25rem;">(%s)</span>`, html.EscapeString(a.CauserType))
		}

		causerHTML = fmt.Sprintf(`
<div style="display: inline-flex; align-items: center; gap: 0.375rem; margin-right: 0.5rem;">
  %s
  <span style="font-weight: 600; color: var(--gray-800); font-size: 0.8125rem;">%s</span>
  %s
</div>`, avatarElem, html.EscapeString(displayName), causerTypeBadge)
	}

	// Timestamp
	timeAgo := a.TimeAgo()
	formattedTime := a.CreatedAt.Format("Jan 02, 2006 15:04:05 MST")
	timeHTML := fmt.Sprintf(`<span title="%s" style="font-size: 0.75rem; color: var(--gray-400); cursor: help;">%s</span>`, html.EscapeString(formattedTime), html.EscapeString(timeAgo))

	// IP / UserAgent meta badges
	metaBadges := ""
	if a.IPAddress != "" {
		metaBadges += fmt.Sprintf(`<span style="font-size: 0.6875rem; font-family: monospace; background-color: var(--gray-100); color: var(--gray-600); padding: 0.125rem 0.375rem; border-radius: 0.25rem;" title="IP Address">%s</span>`, html.EscapeString(a.IPAddress))
	}

	// Subject badge if present
	subjectBadge := ""
	if a.SubjectType != "" && a.SubjectID != "" {
		subjectBadge = fmt.Sprintf(`<span style="font-size: 0.6875rem; font-weight: 600; background-color: var(--gray-100); color: var(--gray-700); padding: 0.125rem 0.375rem; border-radius: 0.25rem;">%s #%s</span>`, html.EscapeString(a.SubjectType), html.EscapeString(a.SubjectID))
	}

	// Diff Inspector
	diffs := a.GetDiff()
	var diffHTML string
	if len(diffs) > 0 {
		var diffRows strings.Builder
		for _, d := range diffs {
			oldStr := FormatValue(d.Old)
			newStr := FormatValue(d.New)
			diffRows.WriteString(fmt.Sprintf(`
<tr style="border-bottom: 1px solid var(--gray-200);">
  <td style="padding: 0.375rem 0.625rem; font-weight: 600; color: var(--gray-700); font-family: monospace; font-size: 0.75rem;">%s</td>
  <td style="padding: 0.375rem 0.625rem; color: #991b1b; background-color: #fef2f2; font-family: monospace; font-size: 0.75rem;"><span style="text-decoration: line-through;">%s</span></td>
  <td style="padding: 0.375rem 0.625rem; color: #065f46; background-color: #ecfdf5; font-family: monospace; font-size: 0.75rem; font-weight: 600;">%s</td>
</tr>`, html.EscapeString(d.Field), html.EscapeString(oldStr), html.EscapeString(newStr)))
		}

		diffTable := fmt.Sprintf(`
<div style="background-color: var(--gray-50); border: 1px solid var(--gray-200); border-radius: 0.375rem; overflow-x: auto; margin-top: 0.5rem;">
  <table style="width: 100%%; border-collapse: collapse; text-align: left;">
    <thead>
      <tr style="background-color: var(--gray-100); border-bottom: 1px solid var(--gray-200); font-size: 0.6875rem; text-transform: uppercase; letter-spacing: 0.05em; color: var(--gray-500);">
        <th style="padding: 0.375rem 0.625rem; width: 30%%;">Field</th>
        <th style="padding: 0.375rem 0.625rem; width: 35%%;">Previous</th>
        <th style="padding: 0.375rem 0.625rem; width: 35%%;">Updated</th>
      </tr>
    </thead>
    <tbody>
      %s
    </tbody>
  </table>
</div>`, diffRows.String())

		if tc.collapsibleDiffs {
			diffHTML = fmt.Sprintf(`
<div x-data="{ openDiff: false }" style="margin-top: 0.375rem;">
  <button type="button" @click="openDiff = !openDiff" 
    class="btn btn-ghost" style="padding: 0.2rem 0.5rem; font-size: 0.75rem; font-weight: 600; display: inline-flex; align-items: center; gap: 0.25rem; border-radius: 0.25rem; background-color: var(--gray-100); color: var(--gray-700); border: 1px solid var(--gray-200); cursor: pointer;">
    <span x-text="openDiff ? '▾ Hide Changes (%d)' : '▸ Show Changes (%d)'"></span>
  </button>
  <div x-show="openDiff" style="display: none;" :style="openDiff ? 'display: block;' : 'display: none;'">
    %s
  </div>
</div>`, len(diffs), len(diffs), diffTable)
		} else {
			diffHTML = diffTable
		}
	}

	// Extra properties inspector if non-empty
	var propsHTML string
	if len(a.Properties) > 0 {
		var propPills strings.Builder
		for k, v := range a.Properties {
			propPills.WriteString(fmt.Sprintf(`
<span style="font-size: 0.6875rem; background-color: var(--gray-100); border: 1px solid var(--gray-200); padding: 0.125rem 0.375rem; border-radius: 0.25rem; color: var(--gray-600); font-family: monospace;">
  <strong>%s:</strong> %s
</span> `, html.EscapeString(k), html.EscapeString(FormatValue(v))))
		}
		propsHTML = fmt.Sprintf(`<div style="margin-top: 0.375rem; display: flex; gap: 0.25rem; flex-wrap: wrap;">%s</div>`, propPills.String())
	}

	// Node margin spacing
	bottomMargin := "1.75rem"
	if isLast {
		bottomMargin = "0.5rem"
	}

	searchContent := strings.ToLower(fmt.Sprintf("%s %s %s %s %s", a.Event, a.Description, a.CauserName, a.SubjectType, a.IPAddress))

	return fmt.Sprintf(`
<div class="activity-timeline-item" 
  data-event="%s"
  data-search="%s"
  x-show="(filterEvent === '' || filterEvent.toLowerCase() === '%s') && (searchTerm === '' || '%s'.includes(searchTerm.toLowerCase()))"
  style="position: relative; margin-bottom: %s;">
  <!-- Node Icon Circle -->
  <div class="activity-node-icon" style="position: absolute; left: -2.3125rem; top: 0; width: 1.625rem; height: 1.625rem; border-radius: 9999px; background-color: %s; border: 1.5px solid %s; color: %s; display: flex; align-items: center; justify-content: center; box-shadow: 0 1px 2px rgba(0,0,0,0.05);">
    %s
  </div>

  <!-- Item Body -->
  <div style="background-color: var(--gray-50); border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 0.75rem 1rem;">
    <!-- Header Row -->
    <div style="display: flex; justify-content: space-between; align-items: flex-start; gap: 0.5rem; flex-wrap: wrap;">
      <div style="display: flex; align-items: center; gap: 0.375rem; flex-wrap: wrap;">
        <span class="badge" style="font-size: 0.6875rem; font-weight: 700; padding: 0.125rem 0.5rem; border-radius: 9999px; background-color: %s; color: %s; border: 1px solid %s; letter-spacing: 0.05em;">%s</span>
        %s
        %s
        %s
      </div>
      <div>
        %s
      </div>
    </div>

    <!-- Description -->
    <div style="margin-top: 0.375rem; font-size: 0.875rem; color: var(--gray-800); line-height: 1.4;">
      %s
    </div>

    <!-- Interactive Diffs -->
    %s

    <!-- Metadata Properties -->
    %s
  </div>
</div>`,
		html.EscapeString(eventLower),
		html.EscapeString(searchContent),
		html.EscapeString(eventLower),
		html.EscapeString(searchContent),
		bottomMargin,
		iconBg, iconBorder, iconColor,
		iconSVG,
		iconBg, iconColor, iconBorder, eventUpper,
		subjectBadge,
		causerHTML,
		metaBadges,
		timeHTML,
		html.EscapeString(a.Description),
		diffHTML,
		propsHTML,
	)
}
