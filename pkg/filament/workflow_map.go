package filament

import (
	"fmt"
	"sort"
	"strings"
)

// A workflow map draws an area of the application as blocks of tasks joined
// by arrows, in the order the work happens: "Create invoices, then Receive
// payments, then Record deposits". Every task opens its screen.
//
// The panel declares the map and its blocks; resources and pages place
// themselves in it the way they pick a navigation group:
//
//	panel.WorkflowMap(filament.NewWorkflowMap("home").Label("Workflow").
//	    Block(filament.MapBlock("customers", "Customers").At(0, 1).Span(3, 1)).
//	    Block(filament.MapBlock("banking", "Banking").At(3, 0).Span(1, 2)))
//
//	invoices.OnMap("home", "customers").Label("Create invoices").Opens(filament.MapCreate).
//	    LeadsTo("receipts", "statements")
//
// Each user sees only the tasks whose screens they may open (the resource's
// or page's navigation item, feature gate and, for MapCreate, create
// permission); arrows to hidden tasks and empty blocks are left out.

// MapOpen is what a resource task opens.
type MapOpen string

const (
	// MapList opens the resource's list (the default).
	MapList MapOpen = "list"
	// MapCreate opens the resource's "new" form, or its list for users
	// who may view but not create.
	MapCreate MapOpen = "create"
)

// MapLinkKind is how two tasks are joined.
type MapLinkKind string

const (
	// MapLeads is a solid arrow: the second task comes next.
	MapLeads MapLinkKind = "leads"
	// MapFeeds is a dashed arrow: the second task uses what the first
	// produces (e.g. time entered is charged on an invoice).
	MapFeeds MapLinkKind = "feeds"
)

// WorkflowMap is a map of tasks in blocks, joined by arrows.
type WorkflowMap struct {
	id, label, icon, group string
	sort                   int
	onDashboard            bool
	blocks                 []*WorkflowMapBlock
	tasks                  []*MapTask // tasks declared on the map (pages, links)
	links                  []mapLink
}

type mapLink struct {
	from, to string
	kind     MapLinkKind
}

// NewWorkflowMap declares a map; id appears in its URL (#/maps/<id>).
func NewWorkflowMap(id string) *WorkflowMap {
	return &WorkflowMap{id: id, label: "Workflow", icon: "route"}
}

// ID returns the map's id.
func (m *WorkflowMap) ID() string { return m.id }

// Label sets the map's title (menu and screen).
func (m *WorkflowMap) Label(s string) *WorkflowMap { m.label = s; return m }

// Icon sets the menu icon.
func (m *WorkflowMap) Icon(s string) *WorkflowMap { m.icon = s; return m }

// NavigationGroup puts the map's menu entry in a group ("" = at the top,
// with the dashboard).
func (m *WorkflowMap) NavigationGroup(g string) *WorkflowMap { m.group = g; return m }

// NavigationSort orders the map among the menu entries of its group.
func (m *WorkflowMap) NavigationSort(n int) *WorkflowMap { m.sort = n; return m }

// OnDashboard also shows the map at the top of the dashboard.
func (m *WorkflowMap) OnDashboard(on ...bool) *WorkflowMap {
	m.onDashboard = len(on) == 0 || on[0]
	return m
}

// Block adds a block (a column or area of tasks).
func (m *WorkflowMap) Block(b *WorkflowMapBlock) *WorkflowMap {
	m.blocks = append(m.blocks, b)
	return m
}

// Task adds a task that no resource owns: a page or a link.
func (m *WorkflowMap) Task(t *MapTask) *WorkflowMap {
	t.mapID = m.id
	m.tasks = append(m.tasks, t)
	return m
}

// Link joins two tasks with a solid arrow (from, then to).
func (m *WorkflowMap) Link(from, to string) *WorkflowMap {
	m.links = append(m.links, mapLink{from: from, to: to, kind: MapLeads})
	return m
}

// Feed joins two tasks with a dashed arrow (to uses what from produces).
func (m *WorkflowMap) Feed(from, to string) *WorkflowMap {
	m.links = append(m.links, mapLink{from: from, to: to, kind: MapFeeds})
	return m
}

// WorkflowMapBlock is an area of a map with a title, placed on the map's
// grid.
type WorkflowMapBlock struct {
	id, label, icon, color string
	col, row, cols, rows   int
	taskColumns            int
}

// MapBlock declares a block. Place it with At and Span; blocks without a
// place are stacked one per row.
func MapBlock(id, label string) *WorkflowMapBlock {
	return &WorkflowMapBlock{id: id, label: label, col: -1, row: -1, cols: 1, rows: 1}
}

// At places the block's top-left corner on the map grid (0-based).
func (b *WorkflowMapBlock) At(col, row int) *WorkflowMapBlock { b.col, b.row = col, row; return b }

// Span sets how many grid columns and rows the block covers.
func (b *WorkflowMapBlock) Span(cols, rows int) *WorkflowMapBlock {
	b.cols, b.rows = max(cols, 1), max(rows, 1)
	return b
}

// Icon sets the block header's icon.
func (b *WorkflowMapBlock) Icon(s string) *WorkflowMapBlock { b.icon = s; return b }

// Color tints the block with a palette role ("primary", "success", ...).
func (b *WorkflowMapBlock) Color(s string) *WorkflowMapBlock { b.color = s; return b }

// TaskColumns fixes how many tasks a row of the block holds (0 = as many
// as fit).
func (b *WorkflowMapBlock) TaskColumns(n int) *WorkflowMapBlock { b.taskColumns = n; return b }

// MapTask is one task on a map: a tile that opens a screen.
type MapTask struct {
	id, mapID, block, label, icon string
	sort                          int
	col, row                      int // -1: placed automatically
	open                          MapOpen
	page, url                     string
	leads, feeds                  []string
	badge                         func(ctx *Context) string
	visible                       func(ctx *Context) bool
	owner                         mapResource // set for resource tasks
}

// NewMapTask declares a task no resource owns; give it a Page or a URL and
// add it with WorkflowMap.Task.
func NewMapTask(id, block, label string) *MapTask {
	return &MapTask{id: id, block: block, label: label, col: -1, row: -1, open: MapList}
}

// ID renames the task (resource tasks default to the resource's slug).
func (t *MapTask) ID(id string) *MapTask { t.id = id; return t }

// Label sets the tile's text (resource tasks default to the plural name).
func (t *MapTask) Label(s string) *MapTask { t.label = s; return t }

// Icon sets the tile's icon (resource tasks default to the menu icon).
func (t *MapTask) Icon(s string) *MapTask { t.icon = s; return t }

// Sort orders the task among those of its block placed automatically.
func (t *MapTask) Sort(n int) *MapTask { t.sort = n; return t }

// Cell places the task inside its block (0-based column and row), to line
// arrows up; tasks without a cell fill the free cells in Sort order.
func (t *MapTask) Cell(col, row int) *MapTask { t.col, t.row = col, row; return t }

// Opens chooses whether a resource task opens the list or the new form.
func (t *MapTask) Opens(o MapOpen) *MapTask { t.open = o; return t }

// Page makes the task open a panel page by id.
func (t *MapTask) Page(id string) *MapTask { t.page = id; return t }

// URL makes the task open a link.
func (t *MapTask) URL(u string) *MapTask { t.url = u; return t }

// LeadsTo draws solid arrows from this task to the tasks that come next.
func (t *MapTask) LeadsTo(ids ...string) *MapTask { t.leads = append(t.leads, ids...); return t }

// Feeds draws dashed arrows to tasks that use what this one produces.
func (t *MapTask) Feeds(ids ...string) *MapTask { t.feeds = append(t.feeds, ids...); return t }

// Badge shows a short count or note on the tile ("" = none).
func (t *MapTask) Badge(fn func(ctx *Context) string) *MapTask { t.badge = fn; return t }

// Visible hides the task from users fn rejects (on top of the screen's own
// permissions).
func (t *MapTask) Visible(fn func(ctx *Context) bool) *MapTask { t.visible = fn; return t }

// mapResource is what a resource task needs from its resource.
type mapResource interface {
	Slug() string
	GetPluralModelName() string
	NavigationItem(ctx *Context) *NavigationItem
	mayCreate(ctx *Context) bool
}

// OnMap places the resource on a map, in a block, as a task that opens its
// list. It returns the task to label and link; call it again with another
// ID for a second task of the same resource.
func (r *BaseResource[T]) OnMap(mapID, block string) *MapTask {
	t := &MapTask{id: r.slug, mapID: mapID, block: block, col: -1, row: -1, open: MapList, owner: r}
	r.mapTasks = append(r.mapTasks, t)
	return t
}

// WorkflowTasks returns the map tasks the resource declared.
func (r *BaseResource[T]) WorkflowTasks() []*MapTask { return r.mapTasks }

func (r *BaseResource[T]) mayCreate(ctx *Context) bool {
	return r.onCreate != nil && (r.canCreate == nil || r.canCreate(ctx))
}

// WorkflowMap adds a map to the panel.
func (p *Panel) WorkflowMap(m *WorkflowMap) *Panel {
	p.workflowMaps = append(p.workflowMaps, m)
	return p
}

// GetWorkflowMaps returns the panel's maps.
func (p *Panel) GetWorkflowMaps() []*WorkflowMap { return p.workflowMaps }

func (p *Panel) workflowMap(id string) *WorkflowMap {
	for _, m := range p.workflowMaps {
		if m.id == id {
			return m
		}
	}
	return nil
}

// tasksOf lists every task of a map: its own and those resources declared.
func (p *Panel) tasksOf(m *WorkflowMap) []*MapTask {
	out := append([]*MapTask{}, m.tasks...)
	for _, res := range p.resources {
		if tr, ok := res.(interface{ WorkflowTasks() []*MapTask }); ok {
			for _, t := range tr.WorkflowTasks() {
				if t.mapID == m.id {
					out = append(out, t)
				}
			}
		}
	}
	return out
}

// ValidateWorkflowMaps reports declaration mistakes: unknown maps or
// blocks, duplicate ids, arrows to unknown tasks, tasks that open nothing.
// Register calls it, so a mistake stops the app at start-up.
func (p *Panel) ValidateWorkflowMaps() error {
	var errs []string
	maps := map[string]bool{}
	for _, m := range p.workflowMaps {
		if m.id == "" || strings.ContainsAny(m.id, "/?#") {
			errs = append(errs, fmt.Sprintf("map id %q is not valid", m.id))
		}
		if maps[m.id] {
			errs = append(errs, fmt.Sprintf("map %q is declared twice", m.id))
		}
		maps[m.id] = true
		blocks := map[string]bool{}
		for _, b := range m.blocks {
			if blocks[b.id] {
				errs = append(errs, fmt.Sprintf("map %q: block %q is declared twice", m.id, b.id))
			}
			blocks[b.id] = true
		}
		tasks := map[string]bool{}
		all := p.tasksOf(m)
		for _, t := range all {
			if tasks[t.id] {
				errs = append(errs, fmt.Sprintf("map %q: task %q is declared twice (give one an ID)", m.id, t.id))
			}
			tasks[t.id] = true
			if !blocks[t.block] {
				errs = append(errs, fmt.Sprintf("map %q: task %q is in unknown block %q", m.id, t.id, t.block))
			}
			if t.owner == nil && t.page == "" && t.url == "" {
				errs = append(errs, fmt.Sprintf("map %q: task %q opens nothing (give it a Page or a URL)", m.id, t.id))
			}
			if t.page != "" && p.page(t.page) == nil {
				errs = append(errs, fmt.Sprintf("map %q: task %q opens unknown page %q", m.id, t.id, t.page))
			}
		}
		check := func(from, to string) {
			if !tasks[from] {
				errs = append(errs, fmt.Sprintf("map %q: an arrow starts at unknown task %q", m.id, from))
			}
			if !tasks[to] {
				errs = append(errs, fmt.Sprintf("map %q: an arrow from %q goes to unknown task %q", m.id, from, to))
			}
		}
		for _, t := range all {
			for _, to := range t.leads {
				check(t.id, to)
			}
			for _, to := range t.feeds {
				check(t.id, to)
			}
		}
		for _, l := range m.links {
			check(l.from, l.to)
		}
	}
	for _, res := range p.resources {
		if tr, ok := res.(interface{ WorkflowTasks() []*MapTask }); ok {
			for _, t := range tr.WorkflowTasks() {
				if !maps[t.mapID] {
					errs = append(errs, fmt.Sprintf("resource %q is on unknown map %q", res.Slug(), t.mapID))
				}
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("filament: workflow maps: %s", strings.Join(errs, "; "))
	}
	return nil
}

func (p *Panel) page(id string) PageInterface {
	for _, pg := range p.pages {
		if pg.ID() == id || pg.Slug() == id {
			return pg
		}
	}
	return nil
}

// WorkflowMapDTO is a map as the signed-in user sees it.
type WorkflowMapDTO struct {
	ID          string             `json:"id"`
	Label       string             `json:"label"`
	Icon        string             `json:"icon"`
	Group       string             `json:"group,omitempty"`
	Sort        int                `json:"sort"`
	OnDashboard bool               `json:"on_dashboard"`
	Blocks      []WorkflowBlockDTO `json:"blocks"`
	Tasks       []WorkflowTaskDTO  `json:"tasks"`
	Links       []WorkflowLinkDTO  `json:"links"`
}

// WorkflowBlockDTO is a block with its place on the map grid.
type WorkflowBlockDTO struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
	Col         int    `json:"col"`
	Row         int    `json:"row"`
	Cols        int    `json:"cols"`
	Rows        int    `json:"rows"`
	TaskColumns int    `json:"task_columns,omitempty"`
}

// WorkflowTaskDTO is a tile and what it opens.
type WorkflowTaskDTO struct {
	ID    string `json:"id"`
	Block string `json:"block"`
	Label string `json:"label"`
	Icon  string `json:"icon,omitempty"`
	Sort  int    `json:"sort"`
	Col   *int   `json:"col,omitempty"`
	Row   *int   `json:"row,omitempty"`
	Badge string `json:"badge,omitempty"`
	// Open: kind "resource" (slug, action "list" or "create"), "page"
	// (page) or "url" (url). URL is the panel link the client opens.
	Open WorkflowOpenDTO `json:"open"`
}

// WorkflowOpenDTO says what a task opens.
type WorkflowOpenDTO struct {
	Kind   string `json:"kind"`
	Slug   string `json:"slug,omitempty"`
	Action string `json:"action,omitempty"`
	Page   string `json:"page,omitempty"`
	URL    string `json:"url"`
}

// WorkflowLinkDTO is an arrow between two tasks.
type WorkflowLinkDTO struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

// BuildWorkflowMap returns map id for the request's user, or false when
// there is no such map or nothing on it the user may open.
func (p *Panel) BuildWorkflowMap(ctx *Context, id string) (*WorkflowMapDTO, bool) {
	m := p.workflowMap(id)
	if m == nil {
		return nil, false
	}
	out := &WorkflowMapDTO{ID: m.id, Label: m.label, Icon: m.icon, Group: m.group, Sort: m.sort,
		OnDashboard: m.onDashboard, Blocks: []WorkflowBlockDTO{}, Tasks: []WorkflowTaskDTO{}, Links: []WorkflowLinkDTO{}}

	shown := map[string]bool{}
	used := map[string]bool{}
	all := p.tasksOf(m)
	for _, t := range all {
		dto, ok := p.taskFor(ctx, t)
		if !ok {
			continue
		}
		shown[t.id] = true
		used[t.block] = true
		out.Tasks = append(out.Tasks, dto)
	}
	if len(out.Tasks) == 0 {
		return nil, false
	}
	blockAt := map[string]int{}
	for i, b := range m.blocks {
		blockAt[b.id] = i
	}
	sort.SliceStable(out.Tasks, func(i, j int) bool {
		a, b := out.Tasks[i], out.Tasks[j]
		if blockAt[a.Block] != blockAt[b.Block] {
			return blockAt[a.Block] < blockAt[b.Block]
		}
		return a.Sort < b.Sort
	})

	row := 0
	for _, b := range m.blocks {
		if b.row >= 0 && b.row+b.rows > row {
			row = b.row + b.rows
		}
	}
	for _, b := range m.blocks {
		if !used[b.id] {
			continue
		}
		dto := WorkflowBlockDTO{ID: b.id, Label: b.label, Icon: b.icon, Color: b.color,
			Col: b.col, Row: b.row, Cols: b.cols, Rows: b.rows, TaskColumns: b.taskColumns}
		if b.col < 0 || b.row < 0 {
			// not placed: one row of its own under the placed blocks
			dto.Col, dto.Row = 0, row
			row += b.rows
		}
		out.Blocks = append(out.Blocks, dto)
	}

	seen := map[string]bool{}
	add := func(from, to string, kind MapLinkKind) {
		key := from + ">" + to
		if !shown[from] || !shown[to] || from == to || seen[key] {
			return
		}
		seen[key] = true
		out.Links = append(out.Links, WorkflowLinkDTO{From: from, To: to, Kind: string(kind)})
	}
	for _, t := range all {
		for _, to := range t.leads {
			add(t.id, to, MapLeads)
		}
		for _, to := range t.feeds {
			add(t.id, to, MapFeeds)
		}
	}
	for _, l := range m.links {
		add(l.from, l.to, l.kind)
	}
	return out, true
}

// taskFor resolves a task for the user: false when its screen is hidden.
func (p *Panel) taskFor(ctx *Context, t *MapTask) (WorkflowTaskDTO, bool) {
	if t.visible != nil && !t.visible(ctx) {
		return WorkflowTaskDTO{}, false
	}
	dto := WorkflowTaskDTO{ID: t.id, Block: t.block, Label: t.label, Icon: t.icon, Sort: t.sort}
	if t.col >= 0 && t.row >= 0 {
		c, r := t.col, t.row
		dto.Col, dto.Row = &c, &r
	}
	base := strings.TrimRight(p.GetPath(), "/")
	switch {
	case t.owner != nil:
		item := t.owner.NavigationItem(ctx)
		if item == nil || !gateAllows(ctx, t.owner) || (item.Visible != nil && !item.Visible(ctx)) {
			return dto, false
		}
		if dto.Label == "" {
			dto.Label = t.owner.GetPluralModelName()
		}
		if dto.Icon == "" {
			dto.Icon = item.Icon
		}
		dto.Open = WorkflowOpenDTO{Kind: "resource", Slug: t.owner.Slug(), Action: string(MapList),
			URL: base + "/" + t.owner.Slug()}
		if t.open == MapCreate && t.owner.mayCreate(ctx) {
			dto.Open.Action = string(MapCreate)
			dto.Open.URL += "/new"
		}
	case t.page != "":
		pg := p.page(t.page)
		if pg == nil {
			return dto, false
		}
		item := pg.NavigationItem(ctx)
		if item == nil || (item.Visible != nil && !item.Visible(ctx)) || !item.Feature.Allows(ctx) || !gateAllows(ctx, pg) {
			return dto, false
		}
		if dto.Label == "" {
			dto.Label = pg.Title()
		}
		if dto.Icon == "" {
			dto.Icon = item.Icon
		}
		dto.Open = WorkflowOpenDTO{Kind: "page", Page: pg.ID(), URL: base + "/" + pg.Slug()}
	default:
		dto.Open = WorkflowOpenDTO{Kind: "url", URL: t.url}
	}
	if t.badge != nil {
		dto.Badge = t.badge(ctx)
	}
	return dto, true
}

// workflowMapsFor lists the maps the user can see, for the panel schema.
func (p *Panel) workflowMapsFor(ctx *Context) []map[string]any {
	out := []map[string]any{}
	for _, m := range p.workflowMaps {
		if _, ok := p.BuildWorkflowMap(ctx, m.id); !ok {
			continue
		}
		out = append(out, map[string]any{"id": m.id, "label": m.label, "icon": m.icon, "group": m.group,
			"sort": m.sort, "on_dashboard": m.onDashboard})
	}
	return out
}
