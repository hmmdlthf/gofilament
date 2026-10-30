package tests

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/notifications"
	"github.com/golaravel/golaravel/pkg/broadcast"
	"github.com/golaravel/golaravel/pkg/golaravel"
)

func TestNotificationStoreAndDispatch(t *testing.T) {
	store := notifications.NewMemoryStore()
	notifications.SetDefaultStore(store)

	// 1. Create and send notifications
	n1 := notifications.Make("Order #101 Placed").
		Success().
		Body("Customer placed a new order for $250.").
		Action("View Order", "/admin/orders/101")

	n2 := notifications.Make("Server Warning").
		Warning().
		Body("High memory usage detected on node-1.")

	_ = notifications.Send("admin_user", n1)
	_ = notifications.SendToAll(n2)

	// 2. Count unread
	unread, err := store.CountUnread("admin_user")
	if err != nil || unread != 2 {
		t.Fatalf("expected 2 unread notifications, got %d (err: %v)", unread, err)
	}

	// 3. List
	items, err := store.List("admin_user", 10)
	if err != nil || len(items) != 2 {
		t.Fatalf("expected 2 items, got %d (err: %v)", len(items), err)
	}

	// 4. Mark one as read
	_ = store.MarkAsRead("admin_user", n1.ID)
	unread, _ = store.CountUnread("admin_user")
	if unread != 1 {
		t.Fatalf("expected 1 unread notification after marking one read, got %d", unread)
	}

	// 5. Mark all as read
	_ = store.MarkAllAsRead("admin_user")
	unread, _ = store.CountUnread("admin_user")
	if unread != 0 {
		t.Fatalf("expected 0 unread notifications, got %d", unread)
	}

	// 6. Clear
	_ = store.Clear("admin_user")
	_ = store.Clear("all")
	items, _ = store.List("admin_user", 10)
	if len(items) != 0 {
		t.Fatalf("expected 0 notifications after clear, got %d", len(items))
	}
}

func TestNotificationsDrawerWebRendering(t *testing.T) {
	app := golaravel.New()
	store := notifications.NewMemoryStore()
	notifications.SetDefaultStore(store)

	// Send an unread alert
	notif := notifications.Make("Welcome to GoFilament").
		Info().
		Body("Your real-time notification engine is active.")
	_ = notifications.SendToAll(notif)

	panel := filament.NewPanel("admin").
		Path("/admin").
		BrandName("CloudPanel").
		DatabaseNotifications(true)

	_ = panel.Register(app)
	client := app.Test(t)

	// GET /admin dashboard
	resp := client.Get("/admin")
	resp.AssertOK()

	// 1. Verifies topbar bell with badge count
	resp.AssertSee("notif-unread-badge")
	resp.AssertSee("Filament.openNotificationDrawer()")
	resp.AssertSee(">1</span>") // 1 unread

	// 2. Verifies slide-out drawer markup
	resp.AssertSee("filament-notifications-drawer")
	resp.AssertSee("notifications-drawer")
	resp.AssertSee("Welcome to GoFilament")
	resp.AssertSee("Your real-time notification engine is active.")

	// 3. Verifies drawer actions (Mark all read, Clear all)
	resp.AssertSee("Filament.markAllNotificationsRead('/admin/notifications/mark-all-read')")
	resp.AssertSee("Filament.clearAllNotifications('/admin/notifications/clear')")

	// 4. Verifies SSE listener script
	resp.AssertSee("Filament.initSSE('/admin/notifications/stream')")
}

func TestNotificationsWebEndpoints(t *testing.T) {
	app := golaravel.New()
	store := notifications.NewMemoryStore()
	notifications.SetDefaultStore(store)

	n1 := notifications.Make("Backup Completed").Success().Body("Database backup stored on S3.")
	n2 := notifications.Make("Failed Login Attempt").Danger().Body("IP 192.168.1.5 blocked.")
	_ = notifications.SendToAll(n1)
	_ = notifications.SendToAll(n2)

	panel := filament.NewPanel("admin").
		Path("/admin").
		DatabaseNotifications(true)

	_ = panel.Register(app)
	client := app.Test(t)

	// 1. GET /admin/notifications (JSON)
	jsonResp := client.Request(http.MethodGet, "/admin/notifications").
		WithHeader("Accept", "application/json").
		Send()
	jsonResp.AssertOK()

	var notifsData map[string]any
	_ = json.Unmarshal(jsonResp.Body(), &notifsData)
	if int(notifsData["unread_count"].(float64)) != 2 {
		t.Fatalf("expected 2 unread, got %v", notifsData["unread_count"])
	}

	// 2. POST /admin/notifications/mark-read
	markResp := client.PostJSON("/admin/notifications/mark-read", map[string]string{
		"id": n1.ID,
	})
	markResp.AssertOK()

	unread, _ := store.CountUnread("all")
	if unread != 1 {
		t.Fatalf("expected 1 unread, got %d", unread)
	}

	// 3. POST /admin/notifications/mark-all-read
	markAllResp := client.Post("/admin/notifications/mark-all-read", "")
	markAllResp.AssertOK()

	unread, _ = store.CountUnread("all")
	if unread != 0 {
		t.Fatalf("expected 0 unread, got %d", unread)
	}

	// 4. POST /admin/notifications/clear
	clearResp := client.Post("/admin/notifications/clear", "")
	clearResp.AssertOK()

	items, _ := store.List("all", 10)
	if len(items) != 0 {
		t.Fatalf("expected 0 items after clear, got %d", len(items))
	}
}

func TestNotificationsRESTAPI(t *testing.T) {
	app := golaravel.New()
	store := notifications.NewMemoryStore()
	notifications.SetDefaultStore(store)

	n := notifications.Make("Deployment Successful").
		Success().
		Body("Service v2.0 deployed to production.")
	_ = notifications.SendToAll(n)

	panel := filament.NewPanel("admin").
		Path("/admin").
		DatabaseNotifications(true)

	_ = panel.Register(app)
	client := app.Test(t)

	// 1. GET /admin/api/notifications
	getResp := client.Get("/admin/api/notifications")
	getResp.AssertOK().AssertSee("Deployment Successful")

	var apiData map[string]any
	_ = json.Unmarshal(getResp.Body(), &apiData)
	if int(apiData["unread_count"].(float64)) != 1 {
		t.Fatalf("expected 1 unread, got %v", apiData["unread_count"])
	}

	// 2. POST /admin/api/notifications/mark-all-read
	readResp := client.PostJSON("/admin/api/notifications/mark-all-read", nil)
	readResp.AssertOK()

	unread, _ := store.CountUnread("all")
	if unread != 0 {
		t.Fatalf("expected 0 unread via API, got %d", unread)
	}

	// 3. POST /admin/api/notifications/clear
	clearResp := client.PostJSON("/admin/api/notifications/clear", nil)
	clearResp.AssertOK()

	items, _ := store.List("all", 10)
	if len(items) != 0 {
		t.Fatalf("expected 0 items via API clear, got %d", len(items))
	}
}

func TestNotificationsRealtimeBroadcastAndSSE(t *testing.T) {
	// Subscribe to channel
	msgChan, unsubscribe := broadcast.Subscribe("filament:notifications:all")
	defer unsubscribe()

	// Dispatch notification
	notif := notifications.Make("Security Alert").
		Danger().
		Body("Suspicious token refresh from unusual location.")

	err := notifications.SendToAll(notif)
	if err != nil {
		t.Fatalf("failed to send notification: %v", err)
	}

	// Verify realtime message arrives over channel
	select {
	case msg := <-msgChan:
		if msg.Event != "notification" {
			t.Fatalf("expected event 'notification', got '%s'", msg.Event)
		}
		if msg.Data["title"] != "Security Alert" {
			t.Fatalf("expected Security Alert title, got %v", msg.Data["title"])
		}
		if msg.Data["type"] != "danger" {
			t.Fatalf("expected danger type, got %v", msg.Data["type"])
		}
		if !strings.Contains(msg.Data["body"].(string), "Suspicious token") {
			t.Fatalf("expected body, got %v", msg.Data["body"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for realtime broadcast message")
	}
}
