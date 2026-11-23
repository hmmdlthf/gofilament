// Command gendocs snapshots the project's Markdown docs from every release tag
// into Hugo content, one docs version per minor release (its latest patch).
//
// Run it from site/ before `hugo`:
//
//	go run ./tools/gendocs            # reads gendocs.json, repo root is ..
//
// It writes (all git-ignored, regenerated on every build):
//
//	content/docs/_index.md                 redirect to /docs/latest/
//	content/docs/<v>/...                   one tree per version, plus "latest"
//	content/changelog.md                   CHANGELOG.md of the newest tag
//	data/versions.json                     version switcher entries
//	data/release.json                      newest release (landing page)
//
// Sources come from `git show <tag>:<file>`, never the working tree, so a
// version's docs are exactly what shipped with that tag.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type sectionConfig struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// File is split into one page per "## " heading.
	File string `json:"file"`
	// Glob (instead of File) makes one page per matching file, e.g. "docs/*.md".
	Glob string `json:"glob"`
	// Order (with Glob) is a file whose links set the page order, such as a
	// docs index; files it does not link to follow in name order.
	Order string `json:"order"`
	// Intro puts the text before the first "## " on the version's home page.
	Intro bool `json:"intro"`
	// Drop lists "## " sections to leave out (matched on the cleaned title).
	Drop []string `json:"drop"`
}

type config struct {
	RepoURL    string          `json:"repo_url"`
	MinVersion string          `json:"min_version"`
	Changelog  string          `json:"changelog"`
	Sections   []sectionConfig `json:"sections"`
}

type semver struct {
	Major, Minor, Patch int
	Tag                 string
}

func (v semver) minorID() string { return fmt.Sprintf("v%d.%d", v.Major, v.Minor) }

func (v semver) less(o semver) bool {
	if v.Major != o.Major {
		return v.Major < o.Major
	}
	if v.Minor != o.Minor {
		return v.Minor < o.Minor
	}
	return v.Patch < o.Patch
}

var tagRe = regexp.MustCompile(`^v(\d+)\.(\d+)\.(\d+)$`)

func parseTag(t string) (semver, bool) {
	m := tagRe.FindStringSubmatch(t)
	if m == nil {
		return semver{}, false // pre-releases (v1.2.0-rc.1) and other tags are skipped
	}
	a, _ := strconv.Atoi(m[1])
	b, _ := strconv.Atoi(m[2])
	c, _ := strconv.Atoi(m[3])
	return semver{a, b, c, t}, true
}

// versionEntry is one docs tree, as listed in data/versions.json.
type versionEntry struct {
	ID     string `json:"id"`  // URL segment: "latest" or "v1.39"
	Tag    string `json:"tag"` // v1.39.1
	Date   string `json:"date"`
	URL    string `json:"url"`
	Latest bool   `json:"latest"`
}

type gen struct {
	cfg  config
	repo string
	out  string
}

func main() {
	cfgPath := flag.String("config", "gendocs.json", "config file")
	repo := flag.String("repo", "..", "git repository holding the docs")
	out := flag.String("out", ".", "Hugo site directory to write into")
	flag.Parse()

	raw, err := os.ReadFile(*cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	var cfg config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		log.Fatalf("%s: %v", *cfgPath, err)
	}
	g := &gen{cfg: cfg, repo: *repo, out: *out}
	if err := g.run(); err != nil {
		log.Fatal(err)
	}
}

func (g *gen) git(args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", g.repo}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	b, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return b, nil
}

// show returns a file at a tag; ok is false when it does not exist there.
func (g *gen) show(tag, file string) (string, bool) {
	b, err := g.git("show", tag+":"+file)
	if err != nil {
		return "", false
	}
	return string(b), true
}

// releases returns the latest patch of every minor, newest first.
func (g *gen) releases() ([]semver, error) {
	b, err := g.git("tag", "--list", "v*")
	if err != nil {
		return nil, err
	}
	min, _ := parseTag(g.cfg.MinVersion)
	best := map[string]semver{}
	for _, t := range strings.Fields(string(b)) {
		v, ok := parseTag(t)
		if !ok || v.less(min) {
			continue
		}
		if cur, ok := best[v.minorID()]; !ok || cur.less(v) {
			best[v.minorID()] = v
		}
	}
	if len(best) == 0 {
		return nil, errors.New("no release tags found (CI checkouts need fetch-depth: 0)")
	}
	var list []semver
	for _, v := range best {
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool { return list[j].less(list[i]) })
	return list, nil
}

func (g *gen) run() error {
	rels, err := g.releases()
	if err != nil {
		return err
	}
	contentDocs := filepath.Join(g.out, "content", "docs")
	if err := os.RemoveAll(contentDocs); err != nil {
		return err
	}

	var versions []versionEntry
	for i, v := range rels {
		date := ""
		if b, err := g.git("log", "-1", "--format=%cs", v.Tag); err == nil {
			date = strings.TrimSpace(string(b))
		}
		ids := []string{v.minorID()}
		if i == 0 {
			ids = []string{"latest", v.minorID()}
		}
		for _, id := range ids {
			if err := g.writeVersion(id, v, date, i == 0); err != nil {
				return fmt.Errorf("%s: %w", v.Tag, err)
			}
		}
		e := versionEntry{ID: ids[0], Tag: v.Tag, Date: date, URL: "/docs/" + ids[0] + "/", Latest: i == 0}
		versions = append(versions, e)
	}

	if err := writeFile(filepath.Join(contentDocs, "_index.md"), frontMatter(map[string]any{
		"title":    "Documentation",
		"layout":   "redirect",
		"redirect": "/docs/latest/",
		"build":    map[string]any{"list": "never"},
		"sitemap":  map[string]any{"disable": true},
	}), ""); err != nil {
		return err
	}
	if err := g.writeChangelog(rels); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(g.out, "data", "versions.json"), versions); err != nil {
		return err
	}
	log.Printf("gendocs: %d versions, latest %s", len(rels), rels[0].Tag)
	return nil
}

// page is one generated Markdown page.
type page struct {
	Section string // section id
	Slug    string // "" for the section index
	Title   string
	Body    string
}

func (g *gen) writeVersion(id string, v semver, date string, latest bool) error {
	base := "/docs/" + id + "/"
	dir := filepath.Join(g.out, "content", "docs", id)

	// Every split file first, so links between files can be resolved.
	type split struct {
		sec   sectionConfig
		file  string
		intro string
		pages []page
		links map[string]string // heading anchor -> page URL with fragment
	}
	var splits []*split
	fileLinks := map[string]map[string]string{} // source file -> anchors
	fileHome := map[string]string{}             // source file -> its page URL
	for _, sec := range g.cfg.Sections {
		files := []string{sec.File}
		if sec.Glob != "" {
			b, err := g.git("ls-tree", "--name-only", v.Tag, path.Dir(sec.Glob)+"/")
			if err != nil {
				continue
			}
			files = nil
			for _, f := range strings.Fields(string(b)) {
				if ok, _ := path.Match(sec.Glob, f); ok {
					files = append(files, f)
				}
			}
			if sec.Order != "" {
				if idx, ok := g.show(v.Tag, sec.Order); ok {
					sortByLinks(files, idx, path.Dir(sec.Order))
				}
			}
		}
		for _, f := range files {
			src, ok := g.show(v.Tag, f)
			if !ok {
				continue // e.g. DOCUMENTATION.md did not exist yet
			}
			s := &split{sec: sec, file: f, links: map[string]string{}}
			secURL := base + sec.ID + "/"
			if sec.Glob != "" {
				title, body := wholeFile(src)
				slug := slugify(strings.TrimSuffix(path.Base(f), path.Ext(f)))
				if strings.EqualFold(path.Base(f), "README.md") {
					continue // the docs folder's own index
				}
				s.pages = []page{{Section: sec.ID, Slug: slug, Title: title, Body: body}}
				for _, a := range headingAnchors(body) {
					s.links[a] = secURL + slug + "/#" + a
				}
				fileHome[f] = secURL + slug + "/"
			} else {
				s.intro, s.pages = splitH2(src, sec)
				for _, p := range s.pages {
					s.links[githubSlug(p.Title)] = secURL + p.Slug + "/"
					for _, a := range headingAnchors(p.Body) {
						s.links[a] = secURL + p.Slug + "/#" + a
					}
				}
				fileHome[f] = secURL
				if sec.Intro {
					fileHome[f] = base
				}
			}
			fileLinks[f] = s.links
			splits = append(splits, s)
		}
	}
	if len(splits) == 0 {
		return errors.New("none of the configured doc files exist at this tag")
	}

	rw := &linkRewriter{repoURL: g.cfg.RepoURL, tag: v.Tag, fileLinks: fileLinks, fileHome: fileHome, changelog: g.cfg.Changelog}
	common := map[string]any{"version": id, "tag": v.Tag, "date": date, "latest": latest}
	if id != "latest" && latest {
		common["canonical_version"] = "latest" // same content as /docs/latest/
	}
	if id != "latest" {
		// Only /docs/latest/ is meant for search results: older versions are
		// noindex and the newest minor's own URL is canonical to latest.
		common["sitemap"] = map[string]any{"disable": true}
	}
	fm := func(extra map[string]any) string {
		m := map[string]any{}
		for k, val := range common {
			m[k] = val
		}
		for k, val := range extra {
			m[k] = val
		}
		return frontMatter(m)
	}

	introDone := false
	seenSection := map[string]bool{}
	for si, s := range splits {
		secDir := filepath.Join(dir, s.sec.ID)
		if !seenSection[s.sec.ID] {
			seenSection[s.sec.ID] = true
			body := ""
			if !s.sec.Intro {
				body = rw.rewrite(s.intro, s.file)
			}
			if err := writeFile(filepath.Join(secDir, "_index.md"), fm(map[string]any{
				"title": s.sec.Title, "weight": (si + 1) * 1000, "source": s.file,
				"description": describe(body),
			}), body); err != nil {
				return err
			}
		}
		if s.sec.Intro && !introDone {
			introDone = true
			if err := writeFile(filepath.Join(dir, "_index.md"), fm(map[string]any{
				"title": "Introduction", "source": s.file, "version_home": true,
				"description": describe(s.intro),
			}), rw.rewrite(s.intro, s.file)); err != nil {
				return err
			}
		}
		for pi, p := range s.pages {
			weight := (si+1)*1000 + pi + 1
			if err := writeFile(filepath.Join(secDir, p.Slug+".md"), fm(map[string]any{
				"title": p.Title, "weight": weight, "source": s.file,
				"description": describe(p.Body),
			}), rw.rewrite(p.Body, s.file)); err != nil {
				return err
			}
		}
	}
	if !introDone {
		return writeFile(filepath.Join(dir, "_index.md"), fm(map[string]any{
			"title": "Introduction", "version_home": true,
		}), "")
	}
	return nil
}

// sortByLinks orders files by where an index file first links to them (links
// relative to dir); files it never links to keep their order at the end.
func sortByLinks(files []string, index, dir string) {
	rank := map[string]int{}
	for _, m := range linkRe.FindAllStringSubmatch(index, -1) {
		target := m[1]
		if i := strings.IndexAny(target, "#?"); i >= 0 {
			target = target[:i]
		}
		if target == "" || strings.Contains(target, "://") {
			continue
		}
		f := path.Clean(path.Join(dir, target))
		if _, seen := rank[f]; !seen {
			rank[f] = len(rank)
		}
	}
	pos := func(f string) int {
		if r, ok := rank[f]; ok {
			return r
		}
		return len(rank)
	}
	sort.SliceStable(files, func(i, j int) bool { return pos(files[i]) < pos(files[j]) })
}

// ---------------------------------------------------------------- Markdown

// fence tracks ``` and ~~~ code blocks so headings and links inside code are
// left alone.
type fence struct{ marker string }

func (f *fence) step(line string) (inCode bool) {
	t := strings.TrimLeft(line, " ")
	if f.marker != "" {
		if strings.HasPrefix(t, f.marker) && strings.TrimSpace(strings.TrimLeft(t, f.marker[:1])) == "" {
			f.marker = ""
		}
		return true
	}
	for _, m := range []string{"```", "~~~"} {
		if strings.HasPrefix(t, m) {
			n := len(t) - len(strings.TrimLeft(t, m[:1]))
			f.marker = strings.Repeat(m[:1], n)
			return true
		}
	}
	return false
}

func headingLevel(line string) int {
	n := 0
	for n < len(line) && line[n] == '#' {
		n++
	}
	if n == 0 || n > 6 || n >= len(line) || line[n] != ' ' {
		return 0
	}
	return n
}

// splitH2 cuts a file into the text before the first "## " and one page per
// "## " section. Headings inside a page move up one level, so a page's "###"
// become its "##" (the "## " title is the page's own H1).
func splitH2(src string, sec sectionConfig) (string, []page) {
	var intro []string
	var pages []page
	var cur *page
	var body []string
	flush := func() {
		if cur != nil {
			cur.Body = trimRules(strings.Join(body, "\n"))
			pages = append(pages, *cur)
		}
		body = nil
	}
	drop := map[string]bool{}
	for _, d := range sec.Drop {
		drop[strings.ToLower(d)] = true
	}
	skipping := false
	used := map[string]int{}
	var f fence
	for _, line := range strings.Split(src, "\n") {
		if f.step(line) {
			if !skipping {
				if cur == nil {
					intro = append(intro, line)
				} else {
					body = append(body, line)
				}
			}
			continue
		}
		lvl := headingLevel(line)
		switch {
		case lvl == 1:
			continue // the file's title; the template prints its own
		case lvl == 2:
			flush()
			title := cleanTitle(line[3:])
			if drop[strings.ToLower(title)] {
				cur, skipping = nil, true
				continue
			}
			skipping = false
			slug := slugify(parenRe.ReplaceAllString(title, ""))
			if used[slug]++; used[slug] > 1 {
				slug = fmt.Sprintf("%s-%d", slug, used[slug])
			}
			cur = &page{Section: sec.ID, Slug: slug, Title: title}
			continue
		case lvl > 2:
			line = line[1:]
		}
		if skipping {
			continue
		}
		if cur == nil {
			intro = append(intro, line)
		} else {
			body = append(body, line)
		}
	}
	flush()
	return trimRules(strings.Join(intro, "\n")), pages
}

// wholeFile returns a file's H1 as its title and the rest as the body.
func wholeFile(src string) (string, string) {
	lines := strings.Split(src, "\n")
	var f fence
	for i, l := range lines {
		if f.step(l) {
			continue
		}
		if headingLevel(l) == 1 {
			return cleanTitle(l[2:]), trimRules(strings.Join(append(lines[:i:i], lines[i+1:]...), "\n"))
		}
	}
	return "Untitled", src
}

// trimRules drops blank lines and "---" rules at both ends of a chunk.
func trimRules(s string) string {
	lines := strings.Split(s, "\n")
	isRule := func(l string) bool {
		t := strings.TrimSpace(l)
		return t == "" || t == "---" || t == "***" || t == "___"
	}
	for len(lines) > 0 && isRule(lines[0]) {
		lines = lines[1:]
	}
	for len(lines) > 0 && isRule(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

var parenRe = regexp.MustCompile(`\s*\([^)]*\)`)

// isEmoji reports emoji and their joiners/selectors, but not other symbols
// such as "⌘" or "→" that headings use as text.
func isEmoji(r rune) bool {
	switch {
	case r >= 0x1F000 && r <= 0x1FAFF, // pictographs, emoticons, transport, flags
		r >= 0x2600 && r <= 0x27BF,                         // misc symbols and dingbats (⚡ ✨ ✅)
		r >= 0x2B00 && r <= 0x2BFF,                         // ⭐ ⬆
		r == 0x231A, r == 0x231B, r == 0x2328, r == 0x23CF, // ⌚ ⌛ ⌨ ⏏ (⌘ stays)
		r >= 0x23E9 && r <= 0x23F3, r >= 0x23F8 && r <= 0x23FA, // ⏩…⏳ ⏸ ⏹ ⏺
		r == 0xFE0F, r == 0x200D, r == 0x20E3:
		return true
	}
	return false
}

// prdRef is an internal spec reference in a heading: "(PRD §2.3)".
var prdRef = regexp.MustCompile(`\s*\(PRD[^)]*\)`)

var numberPrefix = regexp.MustCompile(`^\d+(\.\d+)*\.?\s+`)

// cleanTitle strips leading emoji and section numbers ("## 4. Panel" -> "Panel")
// and inline-code backticks.
func cleanTitle(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimLeftFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '`' && r != '['
	})
	s = numberPrefix.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "`", "")
	s = prdRef.ReplaceAllString(s, "")
	// emoji anywhere in a heading ("Routing ⚡️", "Soft Deletes 🛡️ (...)")
	s = strings.Map(func(r rune) rune {
		if isEmoji(r) {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

// slugify makes a short URL segment: "Form Builder Components" -> "form-builder-components".
func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 60 {
		out = out[:60]
		if i := strings.LastIndexByte(out, '-'); i > 30 {
			out = out[:i]
		}
	}
	if out == "" {
		out = "section"
	}
	return out
}

// githubSlug matches GitHub's (and Hugo's "github" autoHeadingIDType) heading
// anchors, so "#links" written for GitHub keep working.
func githubSlug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

var inlineMarkup = regexp.MustCompile("[`*]|\\[([^\\]]*)\\]\\([^)]*\\)")

func headingAnchors(body string) []string {
	var out []string
	var f fence
	for _, l := range strings.Split(body, "\n") {
		if f.step(l) {
			continue
		}
		if lvl := headingLevel(l); lvl > 0 {
			text := inlineMarkup.ReplaceAllString(l[lvl+1:], "$1")
			out = append(out, githubSlug(text))
			// The original file had one more '#'; the anchor is the same.
		}
	}
	return out
}

// ------------------------------------------------------------------ links

type linkRewriter struct {
	repoURL, tag, changelog string
	fileLinks               map[string]map[string]string
	fileHome                map[string]string
}

var linkRe = regexp.MustCompile(`\]\(([^)\s]+)((?:\s+"[^"]*")?)\)`)

// rewrite fixes links outside code: "#anchor" and "OTHER.md#anchor" go to the
// page that now holds that heading, CHANGELOG.md to /changelog/, and other
// repository paths to GitHub at this tag.
func (rw *linkRewriter) rewrite(body, file string) string {
	lines := strings.Split(body, "\n")
	var f fence
	for i, l := range lines {
		if f.step(l) {
			continue
		}
		parts := strings.Split(l, "`") // odd parts are inline code
		for j := 0; j < len(parts); j += 2 {
			parts[j] = linkRe.ReplaceAllStringFunc(parts[j], func(m string) string {
				sub := linkRe.FindStringSubmatch(m)
				return "](" + rw.target(sub[1], file) + sub[2] + ")"
			})
		}
		lines[i] = strings.Join(parts, "`")
	}
	return strings.Join(lines, "\n")
}

func (rw *linkRewriter) target(t, file string) string {
	if strings.Contains(t, "://") || strings.HasPrefix(t, "mailto:") || strings.HasPrefix(t, "/") {
		return t
	}
	p, frag, _ := strings.Cut(t, "#")
	if p == "" {
		p = file
	} else {
		p = path.Clean(path.Join(path.Dir(file), p))
	}
	if p == rw.changelog {
		return "/changelog/"
	}
	if links, ok := rw.fileLinks[p]; ok {
		if frag == "" {
			return rw.fileHome[p]
		}
		if u, ok := links[frag]; ok {
			return u
		}
		return rw.fileHome[p] + "#" + frag
	}
	if strings.HasPrefix(p, "..") {
		return t
	}
	kind := "blob"
	switch strings.ToLower(path.Ext(p)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp":
		kind = "raw"
	case "":
		kind = "tree"
	}
	u := fmt.Sprintf("%s/%s/%s/%s", strings.TrimSuffix(rw.repoURL, "/"), kind, rw.tag, p)
	if frag != "" {
		u += "#" + frag
	}
	return u
}

// -------------------------------------------------------------- changelog

var changelogHeading = regexp.MustCompile(`^## \[?(v?\d+\.\d+\.\d+)\]?(?:\s*-\s*(\S+))?`)
var boldRe = regexp.MustCompile(`^\s*[-*]\s+\*\*(.+?)\*\*`)

// writeChangelog renders the newest tag's CHANGELOG as /changelog/ (with
// links to each release's docs) and collects that release's highlights.
func (g *gen) writeChangelog(rels []semver) error {
	latest := rels[0]
	src, ok := g.show(latest.Tag, g.cfg.Changelog)
	if !ok {
		return nil
	}
	docs := map[string]bool{}
	for _, v := range rels {
		docs[v.minorID()] = true
	}
	var out []string
	var highlights []string
	inLatest := false
	var f fence
	lines := strings.Split(src, "\n")
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if f.step(l) {
			out = append(out, l)
			continue
		}
		if headingLevel(l) == 1 {
			continue
		}
		if strings.HasPrefix(strings.ToLower(l), "## [unreleased]") {
			// Keep it only when it lists something.
			j := i + 1
			for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
				j++
			}
			if j >= len(lines) || strings.HasPrefix(lines[j], "## ") {
				i = j - 1
				continue
			}
		}
		if m := changelogHeading.FindStringSubmatch(l); m != nil {
			inLatest = m[1] == latest.Tag
			ver, ok := parseTag(m[1])
			out = append(out, fmt.Sprintf("## %s", m[1]))
			meta := []string{}
			if m[2] != "" {
				meta = append(meta, m[2])
			}
			if ok && docs[ver.minorID()] {
				meta = append(meta, fmt.Sprintf("[Docs for %s](/docs/%s/)", ver.minorID(), ver.minorID()))
			}
			if len(meta) > 0 {
				out = append(out, "", strings.Join(meta, " · "), "{.release-meta}")
			}
			continue
		}
		if inLatest && len(highlights) < 4 {
			if m := boldRe.FindStringSubmatch(l); m != nil {
				highlights = append(highlights, strings.ReplaceAll(strings.TrimSuffix(m[1], ":"), "`", ""))
			}
		}
		out = append(out, l)
	}
	body := trimRules(strings.Join(out, "\n"))
	if err := writeFile(filepath.Join(g.out, "content", "changelog.md"), frontMatter(map[string]any{
		"title":       "Changelog",
		"description": "Release notes for every version, newest first, with links to the documentation for each release.",
		"tag":         latest.Tag,
	}), body); err != nil {
		return err
	}
	date := ""
	if b, err := g.git("log", "-1", "--format=%cs", latest.Tag); err == nil {
		date = strings.TrimSpace(string(b))
	}
	return writeJSON(filepath.Join(g.out, "data", "release.json"), map[string]any{
		"tag": latest.Tag, "date": date, "highlights": highlights,
		"docs": "/docs/latest/", "notes": fmt.Sprintf("%s/releases/tag/%s", strings.TrimSuffix(g.cfg.RepoURL, "/"), latest.Tag),
	})
}

// describe returns a search-result description for a page: the first
// paragraph of prose (not a heading, code, table, list or image), with the
// Markdown removed and cut to about 160 characters. "" when there is none;
// the theme then falls back to the site description.
func describe(md string) string {
	var f fence
	var para []string
	var first string
	flush := func() string {
		t := plainText(strings.Join(para, " "))
		para = para[:0]
		if strings.HasSuffix(t, ":") { // a lead-in to a list: still the page's summary
			t = strings.TrimSuffix(t, ":") + "."
		}
		return t
	}
	for _, line := range append(strings.Split(md, "\n"), "") {
		if f.step(line) {
			para = para[:0]
			continue
		}
		t := strings.TrimSpace(line)
		t = strings.TrimSpace(strings.TrimLeft(t, ">")) // blockquotes read as prose
		if t == "" {
			if len(para) > 0 {
				if d := flush(); len(d) >= 60 {
					return truncateWords(d, 160)
				} else if first == "" {
					first = d
				}
			}
			continue
		}
		if !isProse(line, t) {
			para = para[:0]
			continue
		}
		para = append(para, t)
	}
	return truncateWords(first, 160)
}

func isProse(raw, t string) bool {
	if strings.HasPrefix(raw, "    ") || strings.HasPrefix(raw, "\t") {
		return false // indented code
	}
	for _, p := range []string{"#", "|", "- ", "* ", "+ ", "<", "![", "[!", "---", "***", "{", "$ "} {
		if strings.HasPrefix(t, p) {
			return false
		}
	}
	return !orderedItem.MatchString(t)
}

var (
	orderedItem = regexp.MustCompile(`^\d+[.)]\s`)
	mdImage     = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	mdLink      = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	htmlTag     = regexp.MustCompile(`<[^>]+>`)
	spaces      = regexp.MustCompile(`\s+`)
)

func plainText(s string) string {
	s = mdImage.ReplaceAllString(s, "")
	s = mdLink.ReplaceAllString(s, "$1")
	s = htmlTag.ReplaceAllString(s, "")
	s = strings.NewReplacer("**", "", "__", "", "`", "", "*", "").Replace(s)
	s = strings.TrimLeftFunc(s, func(r rune) bool { // leading emoji / symbols
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return strings.TrimSpace(spaces.ReplaceAllString(s, " "))
}

func truncateWords(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	s = string(r[:max])
	if cut := strings.LastIndex(s, " "); cut > len(s)/2 {
		s = s[:cut]
	}
	return strings.TrimRight(s, " ,;:-–—") + "…"
}

// ---------------------------------------------------------------- output

func frontMatter(m map[string]any) string {
	b, _ := json.MarshalIndent(m, "", "  ")
	return string(b) + "\n"
}

func writeFile(name, fm, body string) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	return os.WriteFile(name, []byte(fm+"\n"+body+"\n"), 0o644)
}

func writeJSON(name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	return os.WriteFile(name, append(b, '\n'), 0o644)
}
