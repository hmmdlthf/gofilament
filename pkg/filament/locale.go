package filament

import (
	"fmt"
	"html"
	"strings"
)

var defaultLocaleLabels = map[string]string{
	"en": "English",
	"ar": "العربية",
	"es": "Español",
	"fr": "Français",
	"de": "Deutsch",
	"it": "Italiano",
	"pt": "Português",
	"ru": "Русский",
	"zh": "中文",
	"ja": "日本語",
	"ko": "한국어",
	"tr": "Türkçe",
	"fa": "فارسی",
	"ur": "اردو",
	"he": "עברית",
	"id": "Bahasa Indonesia",
	"nl": "Nederlands",
	"pl": "Polski",
	"sv": "Svenska",
}

var defaultLocaleFlags = map[string]string{
	"en": "🇺🇸",
	"ar": "🇸🇦",
	"es": "🇪🇸",
	"fr": "🇫🇷",
	"de": "🇩🇪",
	"it": "🇮🇹",
	"pt": "🇵🇹",
	"ru": "🇷🇺",
	"zh": "🇨🇳",
	"ja": "🇯🇵",
	"ko": "🇰🇷",
	"tr": "🇹🇷",
	"fa": "🇮🇷",
	"ur": "🇵🇰",
	"he": "🇮🇱",
	"id": "🇮🇩",
	"nl": "🇳🇱",
	"pl": "🇵🇱",
	"sv": "🇸🇪",
}

// Locales registers supported languages for the admin panel.
func (p *Panel) Locales(locales ...string) *Panel {
	p.locales = locales
	if p.defaultLocale == "" && len(locales) > 0 {
		p.defaultLocale = locales[0]
	}
	return p
}

// DefaultLocale sets the fallback language code for the panel.
func (p *Panel) DefaultLocale(loc string) *Panel {
	p.defaultLocale = loc
	return p
}

// LocaleLabels configures custom human-readable display names for locales.
func (p *Panel) LocaleLabels(labels map[string]string) *Panel {
	if p.localeLabels == nil {
		p.localeLabels = make(map[string]string)
	}
	for k, v := range labels {
		p.localeLabels[k] = v
	}
	return p
}

// LocaleFlags configures custom flag emojis or icons for locales.
func (p *Panel) LocaleFlags(flags map[string]string) *Panel {
	if p.localeFlags == nil {
		p.localeFlags = make(map[string]string)
	}
	for k, v := range flags {
		p.localeFlags[k] = v
	}
	return p
}

// GetLocales returns all configured language codes.
func (p *Panel) GetLocales() []string {
	return p.locales
}

// GetDefaultLocale returns the default language code.
func (p *Panel) GetDefaultLocale() string {
	if p.defaultLocale != "" {
		return p.defaultLocale
	}
	if len(p.locales) > 0 {
		return p.locales[0]
	}
	return "en"
}

// GetLocaleLabels returns the map of custom display names.
func (p *Panel) GetLocaleLabels() map[string]string {
	return p.localeLabels
}

// GetLocaleFlags returns the map of custom flag symbols.
func (p *Panel) GetLocaleFlags() map[string]string {
	return p.localeFlags
}

// GetLocaleLabel returns the display name for a given language code.
func (p *Panel) GetLocaleLabel(locale string) string {
	if p.localeLabels != nil {
		if lbl, ok := p.localeLabels[locale]; ok && lbl != "" {
			return lbl
		}
	}
	if lbl, ok := defaultLocaleLabels[locale]; ok && lbl != "" {
		return lbl
	}
	return strings.ToUpper(locale)
}

// GetLocaleFlag returns the flag emoji/icon for a given language code.
func (p *Panel) GetLocaleFlag(locale string) string {
	if p.localeFlags != nil {
		if f, ok := p.localeFlags[locale]; ok && f != "" {
			return f
		}
	}
	if f, ok := defaultLocaleFlags[locale]; ok && f != "" {
		return f
	}
	return "🌐"
}

// resolveLocale detects the user's preferred locale from query, cookie, context, or defaults.
func (p *Panel) resolveLocale(ctx *Context) {
	if ctx == nil || ctx.Context == nil {
		return
	}

	target := ""
	// 1. URL Query Parameter ?lang= or ?locale=
	if q := ctx.Query("lang"); q != "" {
		target = q
	} else if q := ctx.Query("locale"); q != "" {
		target = q
	}

	// 2. Cookie
	if target == "" {
		if c, err := ctx.Cookie("locale"); err == nil && c != "" {
			target = c
		}
	}

	// 3. Current context locale
	if target == "" {
		target = ctx.GetLocale()
	}

	// 4. Default locale
	if target == "" && p.defaultLocale != "" {
		target = p.defaultLocale
	} else if target == "" && len(p.locales) > 0 {
		target = p.locales[0]
	}

	// Validate against supported locales if configured
	if len(p.locales) > 0 && target != "" {
		supported := false
		for _, loc := range p.locales {
			if strings.EqualFold(loc, target) || strings.HasPrefix(strings.ToLower(target), strings.ToLower(loc)+"-") {
				target = loc
				supported = true
				break
			}
		}
		if !supported {
			if p.defaultLocale != "" {
				target = p.defaultLocale
			} else {
				target = p.locales[0]
			}
		}
	}

	if target != "" {
		ctx.SetLocale(target)
		if ctx.Response.ResponseWriter != nil {
			ctx.Response.Header().Set("Content-Language", target)
		}
	}
}

// renderLocaleSwitcherHTML renders an interactive language dropdown pill for the Topbar.
func (p *Panel) renderLocaleSwitcherHTML(ctx *Context) string {
	if len(p.locales) <= 1 {
		return ""
	}

	currentLocale := ctx.Locale()
	currentLabel := p.GetLocaleLabel(currentLocale)
	currentFlag := p.GetLocaleFlag(currentLocale)
	isRTL := ctx.IsRTL()

	alignStyle := "left: 0;"
	if isRTL {
		alignStyle = "right: 0;"
	}

	var sb strings.Builder
	sb.WriteString(`<div class="filament-locale-switcher" style="position: relative;" x-data="{ open: false }">`)
	sb.WriteString(fmt.Sprintf(`
  <button type="button" @click="open = !open" class="btn btn-secondary btn-sm" style="display: flex; align-items: center; gap: 0.375rem; border-radius: 0.5rem; padding: 0.25rem 0.625rem; font-weight: 500; cursor: pointer; border: 1px solid var(--gray-200); background: var(--gray-50);" title="Change Language">
    <span style="font-size: 1rem; line-height: 1;">%s</span>
    <span style="font-size: 0.8125rem;">%s</span>
    <svg style="width: 0.75rem; height: 0.75rem; color: var(--gray-400); flex-shrink: 0;" fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
    </svg>
  </button>`, currentFlag, html.EscapeString(currentLabel)))

	sb.WriteString(fmt.Sprintf(`
  <div x-show="open" @click.outside="open = false" style="display: none; position: absolute; %s top: 100%%; margin-top: 0.375rem; min-width: 10rem; background: #fff; border: 1px solid var(--gray-200); border-radius: 0.5rem; box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1); z-index: 50; overflow: hidden;">
    <div style="padding: 0.375rem 0.75rem; font-size: 0.6875rem; font-weight: 700; text-transform: uppercase; color: var(--gray-400); border-bottom: 1px solid var(--gray-100); display: flex; justify-content: space-between; align-items: center;">
      <span>Language</span>
      <span>%d</span>
    </div>
    <div style="max-height: 14rem; overflow-y: auto;">`, alignStyle, len(p.locales)))

	for _, loc := range p.locales {
		isActive := loc == currentLocale
		label := p.GetLocaleLabel(loc)
		flag := p.GetLocaleFlag(loc)

		bg := "transparent"
		fontWeight := "normal"
		checkMark := ""
		if isActive {
			bg = "var(--primary-50)"
			fontWeight = "600"
			checkMark = `<span style="color: var(--primary-600); font-weight: 700;">✓</span>`
		}

		sb.WriteString(fmt.Sprintf(`
      <a href="%s/locale/switch?lang=%s" style="display: flex; align-items: center; justify-content: space-between; padding: 0.5rem 0.75rem; font-size: 0.8125rem; color: inherit; text-decoration: none; border-bottom: 1px solid var(--gray-50); background: %s;">
        <div style="display: flex; align-items: center; gap: 0.5rem;">
          <span style="font-size: 1rem; line-height: 1;">%s</span>
          <span style="font-weight: %s;">%s</span>
        </div>
        %s
      </a>`, p.path, loc, bg, flag, fontWeight, html.EscapeString(label), checkMark))
	}

	sb.WriteString(`
    </div>
  </div>
</div>`)

	return sb.String()
}
