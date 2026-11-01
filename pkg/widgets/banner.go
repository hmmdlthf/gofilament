package widgets

import (
	"fmt"
	"html"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
)

// BannerWidget displays a welcoming hero card, system notification, or high-priority announcement.
type BannerWidget struct {
	BaseWidget
	title       string
	subtitle    string
	icon        string
	color       string // amber, emerald, indigo, red, gradient
	actionLabel string
	actionURL   string
	dismissible bool
	provider    func(ctx *filament.Context) (title, subtitle string)
}

func NewBannerWidget(title, subtitle string) *BannerWidget {
	w := &BannerWidget{
		title:    title,
		subtitle: subtitle,
		color:    "amber",
		icon:     "heroicon-sparkles",
	}
	w.columnSpan = 3
	return w
}

func (w *BannerWidget) Icon(icon string) *BannerWidget {
	w.icon = icon
	return w
}

func (w *BannerWidget) Color(color string) *BannerWidget {
	w.color = color
	return w
}

func (w *BannerWidget) Action(label, url string) *BannerWidget {
	w.actionLabel = label
	w.actionURL = url
	return w
}

func (w *BannerWidget) Dismissible(d ...bool) *BannerWidget {
	if len(d) > 0 {
		w.dismissible = d[0]
	} else {
		w.dismissible = true
	}
	return w
}

func (w *BannerWidget) Content(fn func(ctx *filament.Context) (title, subtitle string)) *BannerWidget {
	w.provider = fn
	return w
}

func (w *BannerWidget) GetTitle(ctx *filament.Context) string {
	if w.provider != nil {
		t, _ := w.provider(ctx)
		if t != "" {
			return t
		}
	}
	return w.title
}

func (w *BannerWidget) GetSubtitle(ctx *filament.Context) string {
	if w.provider != nil {
		_, s := w.provider(ctx)
		if s != "" {
			return s
		}
	}
	return w.subtitle
}

func (w *BannerWidget) GetIcon() string {
	return w.icon
}

func (w *BannerWidget) GetColor() string {
	return w.color
}

func (w *BannerWidget) GetActionLabel() string {
	return w.actionLabel
}

func (w *BannerWidget) GetActionURL() string {
	return w.actionURL
}

func (w *BannerWidget) IsDismissible() bool {
	return w.dismissible
}

func (w *BannerWidget) WidgetType() string {
	return "banner"
}

func (w *BannerWidget) ToWidgetDTO(ctx *filament.Context) map[string]any {
	return map[string]any{
		"type":         "banner",
		"title":        w.GetTitle(ctx),
		"subtitle":     w.GetSubtitle(ctx),
		"icon":         w.icon,
		"color":        w.color,
		"action_label": w.actionLabel,
		"action_url":   w.actionURL,
		"dismissible":  w.dismissible,
	}
}

func (w *BannerWidget) Render(ctx *filament.Context) string {
	title := w.GetTitle(ctx)
	subtitle := w.GetSubtitle(ctx)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`
<div class="filament-banner filament-card" style="background: linear-gradient(135deg, rgba(245, 158, 11, 0.12) 0%%, rgba(245, 158, 11, 0.03) 100%%); border-left: 4px solid #f59e0b; padding: 1.25rem 1.5rem; margin-bottom: 1.5rem;">
  <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 1rem;">
    <div>
      <h2 style="font-size: 1.25rem; font-weight: 700; margin: 0; color: var(--gray-900);">%s</h2>
      <p style="font-size: 0.875rem; color: var(--gray-600); margin: 0.375rem 0 0 0;">%s</p>
    </div>`,
		html.EscapeString(title),
		html.EscapeString(subtitle),
	))

	if w.actionLabel != "" && w.actionURL != "" {
		sb.WriteString(fmt.Sprintf(`
    <a href="%s" class="filament-btn filament-btn-primary" style="display: inline-block; padding: 0.5rem 1rem; background-color: #f59e0b; color: #000; font-weight: 600; border-radius: 0.5rem; text-decoration: none;">%s</a>`,
			html.EscapeString(w.actionURL),
			html.EscapeString(w.actionLabel),
		))
	}

	sb.WriteString(`</div></div>`)
	return sb.String()
}
