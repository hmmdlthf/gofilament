package filament

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/golaravel/gofilament/pkg/forms"
	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
)

// Kanban board: a resource's records as cards in columns, one column per
// value of a field (for example status). Moving a card sets that field, as
// an edit of just that field: validation, CanEdit, the trashed check, edit
// conflicts and OnUpdate (so the audit log records it).

// KanbanColumn is one column of a board.
type KanbanColumn struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Color string `json:"color,omitempty"`
}

// KanbanConfig configures Resource.Kanban.
type KanbanConfig[T any] struct {
	// Field is the record field the columns stand for. OnList receives it
	// as a filter (filters[Field] = column value).
	Field   string
	Columns []KanbanColumn
	// Title and Description are the card's text.
	Title       func(record T) string
	Description func(record T) string
	// PerColumn is how many cards a column shows (default 50).
	PerColumn int
}

type kanbanBoard struct {
	field     string
	columns   []KanbanColumn
	perColumn int
	card      func(rec any) (title, description string, ok bool)
}

// KanbanCard is a card of a board.
type KanbanCard struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`
}

// KanbanColumnData is a column with its cards.
type KanbanColumnData struct {
	KanbanColumn
	Cards []KanbanCard `json:"cards"`
	Total int          `json:"total"`
}

// Kanban adds a board view of the resource.
func (r *BaseResource[T]) Kanban(cfg KanbanConfig[T]) *BaseResource[T] {
	if cfg.PerColumn <= 0 {
		cfg.PerColumn = 50
	}
	r.kanban = &kanbanBoard{
		field: cfg.Field, columns: cfg.Columns, perColumn: cfg.PerColumn,
		card: func(rec any) (string, string, bool) {
			x, ok := rec.(T)
			if !ok {
				return "", "", false
			}
			title, desc := "", ""
			if cfg.Title != nil {
				title = cfg.Title(x)
			}
			if cfg.Description != nil {
				desc = cfg.Description(x)
			}
			return title, desc, true
		},
	}
	return r
}

// HasKanban reports whether the resource has a board.
func (r *BaseResource[T]) HasKanban() bool { return r.kanban != nil }

// KanbanBoard loads the board: each column's first cards and its total.
func (r *BaseResource[T]) KanbanBoard(ctx *Context, search string) ([]KanbanColumnData, error) {
	if r.kanban == nil || r.onList == nil {
		return nil, ErrNoBoard
	}
	if r.canViewAny != nil && !r.canViewAny(ctx) {
		return nil, ErrForbidden
	}
	var out []KanbanColumnData
	for _, col := range r.kanban.columns {
		recs, total, err := r.onList(ctx, search, "", "asc", 1, r.kanban.perColumn, map[string]string{r.kanban.field: col.Value})
		if err != nil {
			return nil, err
		}
		cd := KanbanColumnData{KanbanColumn: col, Cards: []KanbanCard{}, Total: total}
		for _, rec := range recs {
			if r.canView != nil && !r.canView(ctx, rec) {
				continue
			}
			title, desc, _ := r.kanban.card(rec)
			id := ""
			if r.idExtractor != nil {
				id = r.idExtractor(rec)
			}
			if title == "" {
				title = r.modelName + " #" + id
			}
			cd.Cards = append(cd.Cards, KanbanCard{ID: id, Title: title, Description: desc, Version: r.RecordVersion(rec)})
		}
		out = append(out, cd)
	}
	return out, nil
}

// KanbanMove moves a card to the column value.
func (r *BaseResource[T]) KanbanMove(ctx *Context, id, value, version string) error {
	if r.kanban == nil {
		return ErrNoBoard
	}
	known := false
	for _, c := range r.kanban.columns {
		known = known || c.Value == value
	}
	if !known {
		return &forms.ValidationError{Errors: map[string]string{r.kanban.field: "Unknown column."}}
	}
	if r.onUpdate == nil {
		return ErrColumnNotEditable
	}
	if err := r.authorizeRecord(ctx, id, r.canEdit); err != nil {
		return err
	}
	if err := r.refuseTrashed(ctx, id); err != nil {
		return err
	}
	data := map[string]any{r.kanban.field: value}
	if version != "" {
		data[VersionField] = version
	}
	if err := r.checkVersion(ctx, id, data); err != nil {
		return err
	}
	if comps := r.formComponents(); comps != nil && !r.acceptUnvalidated && forms.FindField(comps, r.kanban.field) != nil {
		withSelf(ctx)
		clean, err := forms.Validate(ctx.GoContext(), comps, data, false)
		if err != nil {
			return err
		}
		data = map[string]any{r.kanban.field: clean[r.kanban.field]}
	}
	return r.onUpdate(ctx, id, data)
}

func (r *BaseResource[T]) handleKanbanPage(ctx *Context, basePath string) error {
	cols, err := r.KanbanBoard(ctx, ctx.Query("search"))
	if err != nil {
		return ctx.String(webStatus(err), "%s", err.Error())
	}
	esc := html.EscapeString
	var b strings.Builder
	fmt.Fprintf(&b, `<div class="filament-header"><div><h1 class="filament-title">%s</h1></div><div style="display: flex; gap: 0.5rem;"><a href="%s" class="btn btn-secondary">List</a></div></div>
<div class="filament-kanban" data-move-url="%s/board/move" style="display: flex; gap: 1rem; overflow-x: auto; align-items: flex-start;">`,
		esc(r.pluralModelName), basePath, basePath)
	for _, c := range cols {
		fmt.Fprintf(&b, `<section class="filament-card" data-column="%s" style="min-width: 16rem; flex: 0 0 16rem; padding: 0.75rem;" ondragover="event.preventDefault()" ondrop="Filament.kanbanDrop(event, this)">
<h3 style="font-weight: 700; margin-bottom: 0.5rem;">%s <span style="color: var(--gray-500); font-weight: 400;">%d</span></h3>`, esc(c.Value), esc(c.Label), c.Total)
		for _, card := range c.Cards {
			fmt.Fprintf(&b, `<article draggable="true" data-id="%s" data-version="%s" ondragstart="event.dataTransfer.setData('text/plain', this.dataset.id)" style="background: var(--white, #fff); border: 1px solid var(--gray-200); border-radius: 0.5rem; padding: 0.5rem 0.625rem; margin-bottom: 0.5rem; cursor: grab;"><a href="%s/%s/edit" style="font-weight: 600; font-size: 0.875rem;">%s</a>`,
				esc(card.ID), esc(card.Version), basePath, esc(card.ID), esc(card.Title))
			if card.Description != "" {
				fmt.Fprintf(&b, `<p style="font-size: 0.8125rem; color: var(--gray-500); margin-top: 0.25rem;">%s</p>`, esc(card.Description))
			}
			b.WriteString(`</article>`)
		}
		b.WriteString(`</section>`)
	}
	b.WriteString(`</div>
<script>
window.Filament = window.Filament || {};
Filament.kanbanDrop = function (event, column) {
  event.preventDefault();
  var id = event.dataTransfer.getData('text/plain');
  var card = document.querySelector('.filament-kanban article[data-id="' + CSS.escape(id) + '"]');
  if (!card || card.parentElement === column) return;
  var from = card.parentElement;
  column.appendChild(card);
  var board = column.closest('.filament-kanban');
  fetch(board.dataset.moveUrl, {method: 'POST', headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
    body: JSON.stringify({id: id, value: column.dataset.column, _version: card.dataset.version || ''})})
    .then(function (r) { return r.json().then(function (b) { return {ok: r.ok, body: b}; }); })
    .then(function (res) {
      if (!res.ok) { from.appendChild(card); alert(res.body.error || 'The card could not be moved.'); return; }
      if (res.body.version) card.dataset.version = res.body.version;
    })
    .catch(function () { from.appendChild(card); });
};
</script>`)
	return r.renderPage(ctx, r.pluralModelName, b.String())
}

func (r *BaseResource[T]) handleKanbanMove(ctx *Context) error {
	var in struct {
		ID      string `json:"id"`
		Value   string `json:"value"`
		Version string `json:"_version"`
	}
	if err := json.NewDecoder(ctx.Request.Body).Decode(&in); err != nil || in.ID == "" {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}
	if err := r.KanbanMove(ctx, in.ID, in.Value, in.Version); err != nil {
		body := map[string]any{"error": err.Error()}
		if c, ok := err.(*EditConflictError); ok {
			body["version"] = c.Current
		}
		return ctx.JSON(webStatus(err), body)
	}
	out := map[string]any{"success": true}
	if v := r.currentVersion(ctx, in.ID); v != "" {
		out["version"] = v
	}
	return ctx.JSON(http.StatusOK, out)
}

type kanbanResource interface {
	HasKanban() bool
	KanbanBoard(ctx *Context, search string) ([]KanbanColumnData, error)
	handleKanbanMove(ctx *Context) error
}

func (p *Panel) mountKanbanAPI(r *router.Router, res ResourceInterface, resPath string, preflight func(c *gcontext.Context) error) {
	k, ok := res.(kanbanResource)
	if !ok || !k.HasKanban() {
		return
	}
	r.OPTIONS(resPath+"/board", preflight)
	r.OPTIONS(resPath+"/board/move", preflight)
	r.GET(resPath+"/board", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		cols, err := k.KanbanBoard(ctx, c.Query("search"))
		if err != nil {
			return p.apiError(c, err)
		}
		return c.JSON(http.StatusOK, map[string]any{"columns": cols})
	})
	r.POST(resPath+"/board/move", func(c *gcontext.Context) error {
		p.setCorsHeaders(c)
		ctx, err := p.AuthorizeAccess(c)
		if err != nil {
			return err
		}
		return k.handleKanbanMove(ctx)
	})
}

func (r *BaseResource[T]) boardLink(basePath string) string {
	if r.kanban == nil {
		return ""
	}
	return fmt.Sprintf(`<a href="%s/board" class="btn btn-secondary" data-board-link>Board</a>`, basePath)
}

func (r *BaseResource[T]) boardMeta() map[string]any {
	if r.kanban == nil {
		return nil
	}
	return map[string]any{"field": r.kanban.field, "columns": r.kanban.columns}
}

// ErrNoBoard is returned (404) for a resource without a board.
var ErrNoBoard error = &endpointError{status: http.StatusNotFound, msg: "This resource has no board"}
