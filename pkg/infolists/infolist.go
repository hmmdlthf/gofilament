package infolists

import (
	"fmt"
	"html"
	"strings"
)

// SectionComponent groups infolist entries.
type SectionComponent struct {
	heading string
	columns int
	schema  []Entry
}

func Section(heading string) *SectionComponent {
	return &SectionComponent{
		heading: heading,
		columns: 1,
		schema:  make([]Entry, 0),
	}
}

func (s *SectionComponent) GetName() string {
	return ""
}

func (s *SectionComponent) Columns(c int) *SectionComponent {
	s.columns = c
	return s
}

func (s *SectionComponent) Schema(entries ...Entry) *SectionComponent {
	s.schema = entries
	return s
}

func (s *SectionComponent) Render(record any) string {
	var body strings.Builder
	colClass := fmt.Sprintf("grid-cols-%d", s.columns)
	body.WriteString(fmt.Sprintf(`<div class="grid %s">`, colClass))
	for _, e := range s.schema {
		body.WriteString(e.Render(record))
	}
	body.WriteString(`</div>`)

	return fmt.Sprintf(`
<div class="filament-card">
  <div style="margin-bottom: 1.25rem; border-bottom: 1px solid var(--gray-200); padding-bottom: 0.5rem;">
    <h3 style="font-size: 1.125rem; font-weight: 600;">%s</h3>
  </div>
  %s
</div>`,
		html.EscapeString(s.heading),
		body.String(),
	)
}

// Infolist presents a read-only view schema for a record.
type Infolist struct {
	schema []Entry
	record any
}

func New() *Infolist {
	return &Infolist{
		schema: make([]Entry, 0),
	}
}

func (i *Infolist) Schema(entries ...Entry) *Infolist {
	i.schema = entries
	return i
}

func (i *Infolist) SetRecord(rec any) *Infolist {
	i.record = rec
	return i
}

func (i *Infolist) Render() string {
	var sb strings.Builder
	for _, e := range i.schema {
		sb.WriteString(e.Render(i.record))
	}
	return sb.String()
}
