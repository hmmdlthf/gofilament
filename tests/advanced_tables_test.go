package tests

import (
	"net/http"
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/tables"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

type TestItem struct {
	ID        int
	Name      string
	Status    string
	IsActive  bool
	SortOrder int
}

func TestTableColumnTogglingRendering(t *testing.T) {
	table := tables.New[TestItem]().
		Toggleable(true).
		Columns(
			tables.TextColumn[TestItem]("id").Sortable(),
			tables.TextColumn[TestItem]("name").Sortable(),
			tables.TextColumn[TestItem]("status").ToggledHiddenByDefault(true),
			tables.ToggleColumn[TestItem]("is_active"),
		)

	table.SetData([]TestItem{
		{ID: 1, Name: "Product Alpha", Status: "draft", IsActive: true, SortOrder: 1},
	}, 1, 1, 15, "", "", "", nil)

	html := table.Render("/admin/items")

	// 1. Verify Toggle Columns dropdown button rendered
	if !strings.Contains(html, "<span>Columns</span>") {
		t.Errorf("Expected Columns toggle dropdown button, got:\n%s", html)
	}

	// 2. Verify Alpine.js x-data for visibleCols
	if !strings.Contains(html, `visibleCols: {"id": true, "name": true, "status": false, "is_active": true}`) {
		t.Errorf("Expected visibleCols alpine data with status: false, got:\n%s", html)
	}

	// 3. Verify data-col and :style attributes on th and td
	if !strings.Contains(html, `data-col="status" :style="!visibleCols['status'] ? 'display: none;' : ''"`) {
		t.Errorf("Expected data-col and :style on status column, got:\n%s", html)
	}
	if !strings.Contains(html, `data-col="name" :style="!visibleCols['name'] ? 'display: none;' : ''"`) {
		t.Errorf("Expected data-col and :style on name column, got:\n%s", html)
	}
}

func TestTableReorderableRendering(t *testing.T) {
	table := tables.New[TestItem]().
		Reorderable("sort_order").
		Columns(
			tables.TextColumn[TestItem]("name"),
			tables.TextColumn[TestItem]("status"),
		)

	table.SetData([]TestItem{
		{ID: 10, Name: "Step 1", Status: "ready", SortOrder: 1},
		{ID: 11, Name: "Step 2", Status: "pending", SortOrder: 2},
	}, 2, 1, 15, "", "", "", nil)

	html := table.Render("/admin/steps")

	// 1. Verify grip handle header
	if !strings.Contains(html, `title="Drag to reorder"`) {
		t.Errorf("Expected drag to reorder header, got:\n%s", html)
	}

	// 2. Verify rows have table-row-draggable and draggable="true"
	if !strings.Contains(html, `class="table-row-draggable" draggable="true"`) {
		t.Errorf("Expected draggable row attributes, got:\n%s", html)
	}

	// 3. Verify drag handle cell exists
	if !strings.Contains(html, `class="drag-handle"`) {
		t.Errorf("Expected drag-handle cell in rows")
	}

	// 4. Verify ondrop calls Filament.reorderRows
	if !strings.Contains(html, `Filament.reorderRows('/admin/steps/reorder'`) {
		t.Errorf("Expected Filament.reorderRows call on drop")
	}
}

func TestTableInlineCellEditingRendering(t *testing.T) {
	table := tables.New[TestItem]().
		Columns(
			tables.TextColumn[TestItem]("name").Editable(true),
			tables.SelectColumn[TestItem]("status").Options(map[string]string{
				"draft":     "Draft",
				"published": "Published",
			}),
			tables.ToggleColumn[TestItem]("is_active"),
		)

	table.SetData([]TestItem{
		{ID: 5, Name: "Editable Post", Status: "published", IsActive: true, SortOrder: 1},
	}, 1, 1, 15, "", "", "", nil)

	html := table.Render("/admin/posts")

	// 1. TextColumn editable click-to-edit
	if !strings.Contains(html, "inline-edit-wrapper") {
		t.Errorf("Expected inline-edit-wrapper for TextColumn, got:\n%s", html)
	}
	if !strings.Contains(html, `Filament.updateCell('/admin/posts/cell', '5', 'name'`) {
		t.Errorf("Expected Filament.updateCell for name, got:\n%s", html)
	}

	// 2. SelectColumn inline select
	if !strings.Contains(html, `Filament.updateCell('/admin/posts/cell', '5', 'status'`) {
		t.Errorf("Expected Filament.updateCell for status select, got:\n%s", html)
	}
	if !strings.Contains(html, `<option value="published" selected>Published</option>`) {
		t.Errorf("Expected selected option Published")
	}

	// 3. ToggleColumn interactive switch button
	if !strings.Contains(html, "filament-toggle-switch") {
		t.Errorf("Expected filament-toggle-switch button, got:\n%s", html)
	}
	if !strings.Contains(html, `aria-checked="true"`) {
		t.Errorf("Expected aria-checked=true on toggle")
	}
	if !strings.Contains(html, `Filament.updateCell('/admin/posts/cell', '5', 'is_active'`) {
		t.Errorf("Expected Filament.updateCell for is_active toggle, got:\n%s", html)
	}
}

func TestResourceCellUpdateAndReorderEndpoints(t *testing.T) {
	app := golaravel.Default()
	panel := filament.NewPanel("admin").Path("/admin")

	var updatedCellCol string
	var updatedCellVal any
	var updatedRecordID string
	var reorderedIDs []string

	mockItems := []TestItem{
		{ID: 1, Name: "Task One", Status: "todo", IsActive: false, SortOrder: 1},
		{ID: 2, Name: "Task Two", Status: "in_progress", IsActive: true, SortOrder: 2},
	}

	res := filament.NewResource[TestItem]("tasks").
		ModelName("Task").
		PluralModelName("Tasks").
		OnList(func(ctx *filament.Context, search, sortCol, sortDir string, page, perPage int, filters map[string]string) ([]TestItem, int, error) {
			return mockItems, len(mockItems), nil
		}).
		OnFind(func(ctx *filament.Context, id string) (TestItem, error) {
			return mockItems[0], nil
		}).
		OnCellUpdate(func(ctx *filament.Context, id, column string, value any) error {
			updatedRecordID = id
			updatedCellCol = column
			updatedCellVal = value
			return nil
		}).
		OnReorder(func(ctx *filament.Context, orderedIDs []string) error {
			reorderedIDs = orderedIDs
			return nil
		})

	panel.Resource(res)
	panel.Register(app)

	cli := app.Test(t)

	// 1. Test Web Inline Cell Update: POST /admin/tasks/cell
	resp := cli.PostJSON("/admin/tasks/cell", map[string]any{
		"record_id": "2",
		"column":    "status",
		"value":     "completed",
	})

	resp.AssertStatus(http.StatusOK)
	if updatedRecordID != "2" || updatedCellCol != "status" || updatedCellVal != "completed" {
		t.Errorf("Cell update handler not triggered correctly: id=%s, col=%s, val=%v",
			updatedRecordID, updatedCellCol, updatedCellVal)
	}

	// 2. Test Web Row Reorder: POST /admin/tasks/reorder
	reorderResp := cli.PostJSON("/admin/tasks/reorder", map[string]any{
		"order": []string{"2", "1"},
	})

	reorderResp.AssertStatus(http.StatusOK)
	if len(reorderedIDs) != 2 || reorderedIDs[0] != "2" || reorderedIDs[1] != "1" {
		t.Errorf("Reorder handler not triggered correctly: got %v", reorderedIDs)
	}

	// 3. Test REST API Row Reorder: POST /admin/api/resources/tasks/reorder
	apiReorderResp := cli.PostJSON("/admin/api/resources/tasks/reorder", map[string]any{
		"order": []string{"2", "1"},
	})

	apiReorderResp.AssertStatus(http.StatusOK)
	apiReorderResp.AssertSee("Rows reordered successfully")

	// 4. Test REST API Cell Edit: POST /admin/api/resources/tasks/1/cell
	apiCellResp := cli.PostJSON("/admin/api/resources/tasks/1/cell", map[string]any{
		"column": "is_active",
		"value":  true,
	})

	apiCellResp.AssertStatus(http.StatusOK)
	apiCellResp.AssertSee("Cell updated successfully")
	if updatedRecordID != "1" || updatedCellCol != "is_active" || updatedCellVal != true {
		t.Errorf("API cell update handler mismatch: id=%s, col=%s, val=%v",
			updatedRecordID, updatedCellCol, updatedCellVal)
	}
}

