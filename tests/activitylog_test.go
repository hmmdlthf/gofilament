package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golaravel/gofilament/pkg/activitylog"
	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/infolists"
	gcontext "github.com/golaravel/golaravel/pkg/context"
)

type mockUser struct {
	id   string
	name string
}

func (m *mockUser) GetAuthIdentifier() string {
	return m.id
}

func (m *mockUser) GetName() string {
	return m.name
}

func TestActivity_DiffAndFormat(t *testing.T) {
	oldVals := map[string]any{
		"status": "pending",
		"price":  100,
		"notes":  "same",
	}
	newVals := map[string]any{
		"status": "shipped",
		"price":  150,
		"notes":  "same",
		"tag":    "vip",
	}

	diffs := activitylog.Diff(oldVals, newVals)
	if len(diffs) != 3 {
		t.Fatalf("expected 3 diffs (price, status, tag), got %d", len(diffs))
	}

	// Should be sorted by field
	if diffs[0].Field != "price" || diffs[1].Field != "status" || diffs[2].Field != "tag" {
		t.Fatalf("diffs not sorted properly: %+v", diffs)
	}

	if diffs[0].Old != 100 || diffs[0].New != 150 {
		t.Fatalf("unexpected price diff: %+v", diffs[0])
	}
	if diffs[1].Old != "pending" || diffs[1].New != "shipped" {
		t.Fatalf("unexpected status diff: %+v", diffs[1])
	}
	if diffs[2].Old != nil || diffs[2].New != "vip" {
		t.Fatalf("unexpected tag diff: %+v", diffs[2])
	}

	// Initials
	if initials := activitylog.GetInitials("Alice Admin"); initials != "AA" {
		t.Fatalf("expected AA, got %s", initials)
	}
	if initials := activitylog.GetInitials("Administrator"); initials != "AD" {
		t.Fatalf("expected AD, got %s", initials)
	}

	// Time ago
	now := time.Now()
	if ago := activitylog.FormatTimeAgo(now.Add(-2 * time.Minute)); ago != "2 minutes ago" {
		t.Fatalf("expected 2 minutes ago, got %s", ago)
	}
	if ago := activitylog.FormatTimeAgo(now.Add(-2 * time.Hour)); ago != "2 hours ago" {
		t.Fatalf("expected 2 hours ago, got %s", ago)
	}
	if ago := activitylog.FormatTimeAgo(now.Add(-25 * time.Hour)); ago != "yesterday" {
		t.Fatalf("expected yesterday, got %s", ago)
	}

	// Values formatting
	if s := activitylog.FormatValue(nil); s != "null" {
		t.Fatalf("expected null, got %s", s)
	}
	if s := activitylog.FormatValue(""); s != `""` {
		t.Fatalf("expected empty quotes, got %s", s)
	}
	if s := activitylog.FormatValue(true); s != "true" {
		t.Fatalf("expected true, got %s", s)
	}
}

func TestMemoryStore_CRUDAndQuery(t *testing.T) {
	store := activitylog.NewMemoryStore()
	ctx := context.Background()

	act1 := &activitylog.Activity{
		SubjectType: "Order",
		SubjectID:   "101",
		CauserType:  "User",
		CauserID:    "1",
		CauserName:  "Bob Builder",
		Event:       "created",
		Description: "Created order #101",
		CreatedAt:   time.Now().Add(-10 * time.Minute),
	}
	act2 := &activitylog.Activity{
		SubjectType: "Order",
		SubjectID:   "101",
		CauserType:  "User",
		CauserID:    "2",
		CauserName:  "Alice Admin",
		Event:       "updated",
		Description: "Updated status to processing",
		OldValues:   map[string]any{"status": "pending"},
		NewValues:   map[string]any{"status": "processing"},
		CreatedAt:   time.Now().Add(-5 * time.Minute),
	}
	act3 := &activitylog.Activity{
		SubjectType: "Invoice",
		SubjectID:   "999",
		CauserType:  "User",
		CauserID:    "1",
		CauserName:  "Bob Builder",
		Event:       "created",
		Description: "Created invoice #999",
		CreatedAt:   time.Now(),
	}

	if err := store.Log(ctx, act1); err != nil {
		t.Fatalf("failed to log act1: %v", err)
	}
	if err := store.Log(ctx, act2); err != nil {
		t.Fatalf("failed to log act2: %v", err)
	}
	if err := store.Log(ctx, act3); err != nil {
		t.Fatalf("failed to log act3: %v", err)
	}

	// GetByID
	found, err := store.GetByID(ctx, act1.ID)
	if err != nil || found.SubjectID != "101" {
		t.Fatalf("failed to find act1 by ID: %v", err)
	}

	// GetBySubject
	orderActs, err := store.GetBySubject(ctx, "Order", "101")
	if err != nil || len(orderActs) != 2 {
		t.Fatalf("expected 2 Order activities, got %d (err: %v)", len(orderActs), err)
	}

	// GetByCauser
	bobActs, err := store.GetByCauser(ctx, "User", "1")
	if err != nil || len(bobActs) != 2 {
		t.Fatalf("expected 2 activities for Bob, got %d", len(bobActs))
	}

	// Query with Filter
	res, total, err := store.Query(ctx, activitylog.Filter{
		Event: "updated",
	})
	if err != nil || total != 1 || len(res) != 1 {
		t.Fatalf("expected 1 updated activity, got total %d", total)
	}

	// Search filter
	res, total, err = store.Query(ctx, activitylog.Filter{
		Search: "processing",
	})
	if err != nil || total != 1 {
		t.Fatalf("expected 1 match for search 'processing', got %d", total)
	}

	// Clear
	if err := store.Clear(ctx); err != nil {
		t.Fatalf("failed to clear store: %v", err)
	}
	orderActs, _ = store.GetBySubject(ctx, "Order", "101")
	if len(orderActs) != 0 {
		t.Fatalf("expected 0 activities after clear, got %d", len(orderActs))
	}
}

func TestLogger_FluentBuilder(t *testing.T) {
	store := activitylog.NewMemoryStore()
	ctx := context.Background()

	act, err := activitylog.Log("Customer profile updated").
		Store(store).
		Subject("Customer", "55").
		Causer("Admin", "1", "Super Admin").
		CauserAvatar("https://example.com/avatar.jpg").
		Event("updated").
		WithDiff(map[string]any{"tier": "silver"}, map[string]any{"tier": "gold"}).
		Property("channel", "web").
		IP("127.0.0.1").
		UserAgent("Mozilla/5.0").
		Save(ctx)

	if err != nil {
		t.Fatalf("failed to save activity: %v", err)
	}
	if act.ID == "" {
		t.Fatalf("expected autogenerated ID")
	}
	if act.SubjectType != "Customer" || act.SubjectID != "55" {
		t.Fatalf("unexpected subject: %s #%s", act.SubjectType, act.SubjectID)
	}
	if act.CauserName != "Super Admin" || act.CauserAvatar != "https://example.com/avatar.jpg" {
		t.Fatalf("unexpected causer: %s (%s)", act.CauserName, act.CauserAvatar)
	}
	if !act.HasDiff() || len(act.GetDiff()) != 1 {
		t.Fatalf("expected 1 diff in saved activity")
	}
	if act.Properties["channel"] != "web" {
		t.Fatalf("expected channel=web in properties")
	}
}

func TestTimelineComponent_RenderingAndInfolist(t *testing.T) {
	act1 := &activitylog.Activity{
		ID:          "act_1",
		SubjectType: "Order",
		SubjectID:   "777",
		CauserType:  "User",
		CauserID:    "10",
		CauserName:  "Alice Admin",
		Event:       "created",
		Description: "Order placed by customer",
		CreatedAt:   time.Now().Add(-1 * time.Hour),
	}
	act2 := &activitylog.Activity{
		ID:          "act_2",
		SubjectType: "Order",
		SubjectID:   "777",
		CauserType:  "User",
		CauserID:    "10",
		CauserName:  "Alice Admin",
		Event:       "updated",
		Description: "Changed shipping address and status",
		OldValues:   map[string]any{"status": "pending", "city": "Boston"},
		NewValues:   map[string]any{"status": "shipped", "city": "New York"},
		Properties:  map[string]any{"carrier": "FedEx", "tracking": "TRK12345"},
		IPAddress:   "192.168.1.100",
		CreatedAt:   time.Now().Add(-5 * time.Minute),
	}

	timeline := activitylog.Timeline(act2, act1).
		Title("Order Audit Trail").
		Filterable(true).
		CollapsibleDiffs(true)

	html := timeline.RenderHTML()

	// Verifies container, spine, and title
	if !strings.Contains(html, "activity-timeline-container") || !strings.Contains(html, "activity-timeline-spine") {
		t.Fatalf("expected timeline container and spine in HTML")
	}
	if !strings.Contains(html, "Order Audit Trail") || !strings.Contains(html, "2 events") {
		t.Fatalf("expected title and event count in HTML")
	}

	// Verifies filter bar
	if !strings.Contains(html, "timeline-filter-bar") || !strings.Contains(html, "x-model=\"searchTerm\"") || !strings.Contains(html, "x-model=\"filterEvent\"") {
		t.Fatalf("expected live filter controls in HTML")
	}

	// Verifies badges & icons
	if !strings.Contains(html, "CREATED") || !strings.Contains(html, "UPDATED") {
		t.Fatalf("expected event badges CREATED and UPDATED")
	}
	if !strings.Contains(html, "Alice Admin") || !strings.Contains(html, "AA") {
		t.Fatalf("expected causer name and initials AA")
	}
	if !strings.Contains(html, "192.168.1.100") {
		t.Fatalf("expected IP address badge")
	}

	// Verifies Diff Table & interactive accordion
	if !strings.Contains(html, "openDiff") || !strings.Contains(html, "Hide Changes (2)") {
		t.Fatalf("expected Alpine diff accordion toggle")
	}
	if !strings.Contains(html, "city") || !strings.Contains(html, "Boston") || !strings.Contains(html, "New York") {
		t.Fatalf("expected city diff row in table")
	}
	if !strings.Contains(html, "status") || !strings.Contains(html, "pending") || !strings.Contains(html, "shipped") {
		t.Fatalf("expected status diff row in table")
	}

	// Verifies Properties inspector
	if !strings.Contains(html, "carrier:") || !strings.Contains(html, "FedEx") || !strings.Contains(html, "tracking:") {
		t.Fatalf("expected properties chips in HTML")
	}

	// Verifies Infolist Entry integration
	record := map[string]any{
		"id":         "777",
		"activities": []*activitylog.Activity{act2, act1},
	}

	infolist := infolists.New().Schema(
		infolists.Section("History").Schema(
			activitylog.TimelineEntry("activities"),
		),
	)
	infolist.SetRecord(record)
	renderedInfolist := infolist.Render()

	if !strings.Contains(renderedInfolist, "activity-timeline-container") || !strings.Contains(renderedInfolist, "Order placed by customer") {
		t.Fatalf("expected timeline entry to render inside infolist schema")
	}
}

func TestEnableAuditLog_ResourceObserver(t *testing.T) {
	store := activitylog.NewMemoryStore()

	// Create test resource with mock handlers
	type Item struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Price int    `json:"price"`
	}

	database := map[string]Item{
		"1": {ID: "1", Name: "Widget A", Price: 100},
	}

	res := filament.NewResource[Item]("items").
		ModelName("Item")

	res.OnFind(func(ctx *filament.Context, id string) (Item, error) {
		return database[id], nil
	})
	res.OnCreate(func(ctx *filament.Context, data map[string]any) error {
		database["2"] = Item{
			ID:    "2",
			Name:  data["name"].(string),
			Price: int(data["price"].(float64)),
		}
		return nil
	})
	res.OnUpdate(func(ctx *filament.Context, id string, data map[string]any) error {
		it := database[id]
		if n, ok := data["name"]; ok {
			it.Name = n.(string)
		}
		if p, ok := data["price"]; ok {
			it.Price = int(p.(float64))
		}
		database[id] = it
		return nil
	})
	res.OnDelete(func(ctx *filament.Context, id string) error {
		delete(database, id)
		return nil
	})

	// Enable Audit Log on resource
	activitylog.EnableAuditLog(res, store)

	// Mock HTTP context with authenticated user
	req := httptest.NewRequest(http.MethodPost, "/admin/items", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("User-Agent", "GoFilamentClient/1.0")

	rec := httptest.NewRecorder()
	gc := gcontext.NewContext()
	gc.Reset(rec, req)
	panel := filament.NewPanel("admin")
	fctx := filament.NewContext(gc, panel)

	// Set auth user in context
	fctx.Set("auth_user", &mockUser{id: "42", name: "Alice Admin"})

	// 1. Test Create auditing
	err := res.GetOnCreate()(fctx, map[string]any{
		"id":    "2",
		"name":  "Widget B",
		"price": float64(200),
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	acts, _ := store.GetBySubject(context.Background(), "Item", "2")
	if len(acts) != 1 {
		t.Fatalf("expected 1 create activity logged, got %d", len(acts))
	}
	if acts[0].Event != "created" || acts[0].CauserName != "Alice Admin" || acts[0].CauserID != "42" {
		t.Fatalf("unexpected create activity details: %+v", acts[0])
	}

	// 2. Test Update auditing with diff calculation
	err = res.GetOnUpdate()(fctx, "1", map[string]any{
		"name":  "Widget A+",
		"price": float64(150),
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	acts, _ = store.GetBySubject(context.Background(), "Item", "1")
	if len(acts) != 1 {
		t.Fatalf("expected 1 update activity logged, got %d", len(acts))
	}
	if acts[0].Event != "updated" || !acts[0].HasDiff() {
		t.Fatalf("expected update activity with diff: %+v", acts[0])
	}
	diffs := acts[0].GetDiff()
	if len(diffs) < 2 {
		t.Fatalf("expected at least 2 field diffs, got %d", len(diffs))
	}

	// 3. Test Delete auditing
	err = res.GetOnDelete()(fctx, "1")
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	acts, _ = store.GetBySubject(context.Background(), "Item", "1")
	if len(acts) != 2 {
		t.Fatalf("expected 2 activities for Item #1 (update, delete), got %d", len(acts))
	}
	if acts[0].Event != "deleted" {
		t.Fatalf("expected newest activity to be deleted, got %s", acts[0].Event)
	}
}

func TestActivityResource(t *testing.T) {
	store := activitylog.NewMemoryStore()
	_ = store.Log(context.Background(), &activitylog.Activity{
		SubjectType: "Order",
		SubjectID:   "500",
		Event:       "created",
		Description: "Order placed",
		CauserName:  "John Customer",
		CreatedAt:   time.Now(),
	})

	res := activitylog.NewActivityResource(store)
	if res.Slug() != "activity-logs" {
		t.Fatalf("expected slug activity-logs, got %s", res.Slug())
	}
	if res.GetModelName() != "Audit Log" {
		t.Fatalf("expected ModelName Audit Log, got %s", res.GetModelName())
	}

	// Test OnList
	records, total, err := res.GetOnList()(nil, "", "", "", 1, 10, map[string]string{})
	if err != nil || total != 1 || len(records) != 1 {
		t.Fatalf("expected 1 record from OnList, got %d (err: %v)", total, err)
	}

	// Test OnFind
	record, err := res.GetOnFind()(nil, records[0].ID)
	if err != nil || record.SubjectID != "500" {
		t.Fatalf("failed to find record: %v", err)
	}
}
