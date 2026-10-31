package tests

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golaravel/gofilament/pkg/activitylog"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type SoftDeleteDocument struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

func TestSoftDeletes_TableFilterAndTrashedActions(t *testing.T) {
	now := time.Now()
	docActive := SoftDeleteDocument{ID: "1", Title: "Active Document", DeletedAt: nil}
	docTrashed := SoftDeleteDocument{ID: "2", Title: "Trashed Document", DeletedAt: &now}

	table := tables.New[SoftDeleteDocument]().
		Columns(
			tables.TextColumn[SoftDeleteDocument]("title").Label("Document Title"),
		).
		Actions(
			tables.EditAction[SoftDeleteDocument](),
			tables.DeleteAction[SoftDeleteDocument](),
		).
		BulkActions(
			tables.DeleteBulkAction(),
		).
		SoftDeletes(true).
		IsTrashed(func(doc SoftDeleteDocument) bool {
			return doc.DeletedAt != nil
		}).
		SetData([]SoftDeleteDocument{docActive, docTrashed}, 2, 1, 10, "", "", "asc", nil)

	html := table.Render("/admin/documents")

	// 1. Verifies Active record has Edit and Delete actions
	if !strings.Contains(html, "Active Document") {
		t.Fatalf("expected Active Document in HTML")
	}

	// 2. Verifies Trashed record has Restore and Force Delete actions
	if !strings.Contains(html, "Trashed Document") {
		t.Fatalf("expected Trashed Document in HTML")
	}
	if !strings.Contains(html, "table-row-trashed") {
		t.Fatalf("expected table-row-trashed class on trashed row")
	}
	if !strings.Contains(html, "Restore") || !strings.Contains(html, "Force Delete") {
		t.Fatalf("expected Restore and Force Delete actions for trashed record, got: %s", html)
	}

	// 3. Verifies Restore & Force-Delete action URLs
	if !strings.Contains(html, "/admin/documents/restore?id=2") {
		t.Fatalf("expected restore URL for record 2, got: %s", html)
	}
	if !strings.Contains(html, "/admin/documents/force-delete?id=2") {
		t.Fatalf("expected force-delete URL for record 2, got: %s", html)
	}
}

func TestSoftDeletes_LifecycleEndpointsAndBulk(t *testing.T) {
	app := golaravel.New()

	now := time.Now()
	database := map[string]*SoftDeleteDocument{
		"1": {ID: "1", Title: "Doc 1", DeletedAt: nil},
		"2": {ID: "2", Title: "Doc 2", DeletedAt: &now},
		"3": {ID: "3", Title: "Doc 3", DeletedAt: &now},
	}

	res := filament.NewResource[*SoftDeleteDocument]("documents").
		ModelName("Document").
		SoftDeletes(true).
		IsTrashed(func(doc *SoftDeleteDocument) bool {
			return doc != nil && doc.DeletedAt != nil
		}).
		Table(func(t *tables.Table[*SoftDeleteDocument]) *tables.Table[*SoftDeleteDocument] {
			return t.Columns(
				tables.TextColumn[*SoftDeleteDocument]("title").Label("Title"),
			)
		})

	res.OnFind(func(ctx *filament.Context, id string) (*SoftDeleteDocument, error) {
		return database[id], nil
	})

	res.OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]*SoftDeleteDocument, int, error) {
		trashedScope := filters["trashed"]
		var out []*SoftDeleteDocument
		for _, doc := range database {
			if trashedScope == "only_trashed" {
				if doc.DeletedAt != nil {
					out = append(out, doc)
				}
			} else if trashedScope == "with_trashed" {
				out = append(out, doc)
			} else {
				// without_trashed (default)
				if doc.DeletedAt == nil {
					out = append(out, doc)
				}
			}
		}
		return out, len(out), nil
	})

	res.OnDelete(func(ctx *filament.Context, id string) error {
		t := time.Now()
		database[id].DeletedAt = &t
		return nil
	})

	res.OnRestore(func(ctx *filament.Context, id string) error {
		database[id].DeletedAt = nil
		return nil
	})

	res.OnForceDelete(func(ctx *filament.Context, id string) error {
		delete(database, id)
		return nil
	})

	panel := filament.NewPanel("admin").
		Path("/admin").
		Resource(res)

	_ = panel.Register(app)
	client := app.Test(t)

	// 1. Initial list view (default without_trashed): Should only show Doc 1
	respList := client.Get("/admin/documents").AssertOK()
	if !strings.Contains(respList.BodyString(), "Doc 1") {
		t.Fatalf("expected Doc 1 in active list")
	}
	if strings.Contains(respList.BodyString(), "Doc 2") {
		t.Fatalf("did not expect Doc 2 in default active list")
	}
	// Trashed filter dropdown should be automatically rendered
	if !strings.Contains(respList.BodyString(), "Trashed Records") || !strings.Contains(respList.BodyString(), "only_trashed") {
		t.Fatalf("expected Trashed filter options in table toolbar")
	}

	// 2. Trashed bin view (filter_trashed=only_trashed): Should show Doc 2 and Doc 3
	respTrashed := client.Get("/admin/documents?filter_trashed=only_trashed").AssertOK()
	if !strings.Contains(respTrashed.BodyString(), "Doc 2") || !strings.Contains(respTrashed.BodyString(), "Doc 3") {
		t.Fatalf("expected Doc 2 and Doc 3 in trashed list view")
	}
	if strings.Contains(respTrashed.BodyString(), "Doc 1") {
		t.Fatalf("did not expect active Doc 1 in trash bin view")
	}

	// 3. Single record restore via GET /admin/documents/restore?id=2
	client.Get("/admin/documents/restore?id=2").AssertStatus(http.StatusSeeOther)
	if database["2"].DeletedAt != nil {
		t.Fatalf("expected Doc 2 to be restored (DeletedAt == nil)")
	}

	// 4. Single record force delete via POST /admin/documents/force-delete
	client.PostForm("/admin/documents/force-delete", map[string]string{"id": "3"}).AssertStatus(http.StatusSeeOther)
	if _, exists := database["3"]; exists {
		t.Fatalf("expected Doc 3 to be purged from database")
	}

	// 5. REST API Restore & Force Delete
	database["4"] = &SoftDeleteDocument{ID: "4", Title: "Doc 4", DeletedAt: &now}
	client.Post("/admin/api/resources/documents/restore", strings.NewReader(`{"id":"4"}`), func(r *http.Request) {
		r.Header.Set("Content-Type", "application/json")
	}).AssertOK()
	if database["4"].DeletedAt != nil {
		t.Fatalf("expected Doc 4 to be restored via REST API")
	}

	client.Post("/admin/api/resources/documents/force-delete", strings.NewReader(`{"id":"4"}`), func(r *http.Request) {
		r.Header.Set("Content-Type", "application/json")
	}).AssertOK()
	if _, exists := database["4"]; exists {
		t.Fatalf("expected Doc 4 to be permanently deleted via REST API")
	}
}

func TestSoftDeletes_ActivityLogIntegration(t *testing.T) {
	store := activitylog.NewMemoryStore()

	type Item struct {
		ID        string     `json:"id"`
		Name      string     `json:"name"`
		DeletedAt *time.Time `json:"deleted_at,omitempty"`
	}

	now := time.Now()
	items := map[string]*Item{
		"10": {ID: "10", Name: "Item 10", DeletedAt: &now},
	}

	res := filament.NewResource[*Item]("items").
		ModelName("Inventory Item").
		SoftDeletes(true)

	res.OnFind(func(ctx *filament.Context, id string) (*Item, error) {
		return items[id], nil
	})
	res.OnRestore(func(ctx *filament.Context, id string) error {
		items[id].DeletedAt = nil
		return nil
	})
	res.OnForceDelete(func(ctx *filament.Context, id string) error {
		delete(items, id)
		return nil
	})

	// Attach activity audit log
	activitylog.EnableAuditLog(res, store)

	// Execute restore
	_ = res.GetOnRestore()(nil, "10")

	acts, _ := store.GetBySubject(context.Background(), "Inventory Item", "10")
	if len(acts) != 1 || acts[0].Event != "restored" {
		t.Fatalf("expected restored event in activity log, got: %+v", acts)
	}

	// Execute force delete
	_ = res.GetOnForceDelete()(nil, "10")

	acts, _ = store.GetBySubject(context.Background(), "Inventory Item", "10")
	if len(acts) != 2 || acts[0].Event != "deleted" || acts[0].Properties["force_deleted"] != true {
		t.Fatalf("expected deleted event with force_deleted=true in activity log, got: %+v", acts)
	}
}
