package globalsearch

import (
	"fmt"
	"html"
	"strings"
)

// ActionDTO represents a quick action on a search result.
type ActionDTO struct {
	Label string `json:"label"`
	URL   string `json:"url"`
	Icon  string `json:"icon,omitempty"`
}

// ItemDTO represents a single search match.
type ItemDTO struct {
	Title   string            `json:"title"`
	URL     string            `json:"url"`
	Details map[string]string `json:"details,omitempty"`
	Group   string            `json:"group"`
	Icon    string            `json:"icon,omitempty"`
	Badge   string            `json:"badge,omitempty"`
	Actions []ActionDTO       `json:"actions,omitempty"`
}

// GroupDTO represents a categorized collection of results.
type GroupDTO struct {
	Name  string    `json:"name"`
	Count int       `json:"count"`
	Items []ItemDTO `json:"items"`
}

// RenderTriggerHTML renders the command bar trigger button in the Topbar.
func RenderTriggerHTML(brandName, basePath, keyBinding string) string {
	if keyBinding == "" {
		keyBinding = "⌘K"
	} else if strings.ToLower(keyBinding) == "k" {
		keyBinding = "⌘K"
	}

	return fmt.Sprintf(`
<div class="spotlight-trigger" onclick="Filament.spotlight.open()" title="Quick search (%s)">
  <div style="display: flex; align-items: center; gap: 0.5rem; color: var(--gray-400); font-size: 0.8125rem;">
    <svg style="width: 0.875rem; height: 0.875rem; flex-shrink: 0;" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
    </svg>
    <span>Search %s...</span>
  </div>
  <kbd class="spotlight-trigger-kbd">%s</kbd>
</div>`, keyBinding, html.EscapeString(brandName), html.EscapeString(keyBinding))
}

// RenderModalHTML renders the Spotlight Command Palette modal markup.
func RenderModalHTML(brandName, basePath, keyBinding string) string {
	return fmt.Sprintf(`
<div id="filament-spotlight-modal" class="spotlight-backdrop" style="display: none;" onclick="if(event.target === this) Filament.spotlight.close();">
  <div class="spotlight-container">
    <div class="spotlight-input-wrapper">
      <svg class="spotlight-search-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input type="text" id="filament-spotlight-input" class="spotlight-input" placeholder="Search resources, records, pages, or jump to..." 
        autocomplete="off"
        oninput="Filament.spotlight.fetch(this.value)"
        onkeydown="
          if (event.key === 'ArrowDown') { event.preventDefault(); Filament.spotlight.navigate(1); }
          else if (event.key === 'ArrowUp') { event.preventDefault(); Filament.spotlight.navigate(-1); }
          else if (event.key === 'Enter') { event.preventDefault(); Filament.spotlight.select(); }
          else if (event.key === 'Escape') { event.preventDefault(); Filament.spotlight.close(); }
        "
      />
      <button type="button" class="spotlight-clear-btn" onclick="document.getElementById('filament-spotlight-input').value = ''; Filament.spotlight.fetch('');" title="Clear search">&times;</button>
      <kbd class="spotlight-esc-pill" onclick="Filament.spotlight.close()">ESC</kbd>
    </div>

    <div id="filament-spotlight-results" class="spotlight-results" data-panel-path="%s">
      <div style="padding: 2.5rem 1rem; text-align: center; color: var(--gray-400); font-size: 0.875rem;">
        Type anything to search records, models, or navigation across %s...
      </div>
    </div>

    <div class="spotlight-footer">
      <div class="spotlight-shortcuts">
        <span><kbd>↑</kbd><kbd>↓</kbd> Navigate</span>
        <span><kbd>↵</kbd> Select</span>
        <span><kbd>ESC</kbd> Close</span>
      </div>
      <div style="font-size: 0.6875rem; color: var(--gray-400); font-weight: 500;">
        GoFilament Spotlight
      </div>
    </div>
  </div>
</div>`, basePath, html.EscapeString(brandName))
}

// RenderResultsHTML formats categorized groups of search matches for the Spotlight modal.
func RenderResultsHTML(groups []GroupDTO, query string) string {
	totalCount := 0
	for _, g := range groups {
		totalCount += len(g.Items)
	}

	if totalCount == 0 {
		return fmt.Sprintf(`
<div style="padding: 3rem 1rem; text-align: center; color: var(--gray-400);">
  <div style="font-size: 2rem; margin-bottom: 0.5rem;">🔍</div>
  <div style="font-size: 0.875rem; font-weight: 500; color: var(--gray-700);">No matching results found</div>
  <div style="font-size: 0.75rem; margin-top: 0.25rem;">Try searching for a different keyword, name, or model.</div>
</div>`)
	}

	var sb strings.Builder
	itemIdx := 0

	for _, grp := range groups {
		if len(grp.Items) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf(`
<div class="spotlight-group-header">
  <span>%s</span>
  <span>%d</span>
</div>`, html.EscapeString(strings.ToUpper(grp.Name)), len(grp.Items)))

		for _, item := range grp.Items {
			highlightedTitle := Highlight(item.Title, query)

			detailsHTML := ""
			if len(item.Details) > 0 {
				var detSB strings.Builder
				detSB.WriteString(`<div class="spotlight-item-details">`)
				for k, v := range item.Details {
					detSB.WriteString(fmt.Sprintf(`<span class="spotlight-detail-tag"><strong>%s:</strong> %s</span>`,
						html.EscapeString(k), html.EscapeString(v)))
				}
				detSB.WriteString(`</div>`)
				detailsHTML = detSB.String()
			}

			actionsHTML := ""
			if len(item.Actions) > 0 {
				var actSB strings.Builder
				actSB.WriteString(`<div class="spotlight-item-actions">`)
				for _, act := range item.Actions {
					actSB.WriteString(fmt.Sprintf(`<a href="%s" class="btn btn-secondary btn-xs" onclick="event.stopPropagation()">%s</a>`,
						html.EscapeString(act.URL), html.EscapeString(act.Label)))
				}
				actSB.WriteString(`</div>`)
				actionsHTML = actSB.String()
			}

			iconSpan := `📁`
			if item.Group == "Navigation" {
				iconSpan = `⚡`
			} else if item.Icon != "" {
				iconSpan = html.EscapeString(item.Icon)
			}

			badgeHTML := ""
			if item.Badge != "" {
				badgeHTML = fmt.Sprintf(`<span class="badge badge-gray" style="font-size: 0.6875rem;">%s</span>`, html.EscapeString(item.Badge))
			}

			sb.WriteString(fmt.Sprintf(`
<a href="%s" class="spotlight-result-item" data-index="%d">
  <div style="display: flex; align-items: center; gap: 0.75rem; flex: 1; min-width: 0;">
    <span class="spotlight-item-icon">%s</span>
    <div style="flex: 1; min-width: 0;">
      <div style="display: flex; align-items: center; gap: 0.5rem;">
        <span class="spotlight-item-title">%s</span>
        %s
      </div>
      %s
    </div>
  </div>
  %s
</a>`, html.EscapeString(item.URL), itemIdx, iconSpan, highlightedTitle, badgeHTML, detailsHTML, actionsHTML))

			itemIdx++
		}
	}

	return sb.String()
}
