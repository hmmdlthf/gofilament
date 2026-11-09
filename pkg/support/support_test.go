package support_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/support"
)

func TestSupportService_AIAndLiveChat(t *testing.T) {
	ctx := context.Background()
	store := support.NewMemorySupportStore()
	svc := support.NewService(store)

	// Test 1: Send user message in AI mode
	userMsg, aiMsg, err := svc.SendUserMessage(ctx, "TKT-001", "user_1", "Fatima Al-Mansoor", "How do I invite team members?", "en")
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}
	if userMsg == nil || aiMsg == nil {
		t.Fatal("expected both userMsg and aiMsg in AI mode")
	}
	if aiMsg.SenderType != "ai" {
		t.Errorf("expected sender type 'ai', got %s", aiMsg.SenderType)
	}

	// Verify ticket status and mode
	tk, err := store.GetTicket(ctx, "TKT-001")
	if err != nil || tk == nil {
		t.Fatalf("failed to fetch ticket: %v", err)
	}
	if tk.Mode != support.ModeAI {
		t.Errorf("expected mode %s, got %s", support.ModeAI, tk.Mode)
	}

	// Test 2: Auto-escalation trigger in content
	userMsg2, sysMsg2, err := svc.SendUserMessage(ctx, "TKT-001", "user_1", "Fatima Al-Mansoor", "This is broken, please connect me to a human representative", "en")
	if err != nil {
		t.Fatalf("failed sending message: %v", err)
	}
	if userMsg2 == nil || sysMsg2 == nil {
		t.Fatal("expected user message and system escalation response")
	}

	// Verify mode switched to live
	tkEscalated, _ := store.GetTicket(ctx, "TKT-001")
	if tkEscalated.Mode != support.ModeLive {
		t.Errorf("expected mode to be %s after auto-escalation, got %s", support.ModeLive, tkEscalated.Mode)
	}
	if tkEscalated.Status != support.StatusEscalated {
		t.Errorf("expected status %s, got %s", support.StatusEscalated, tkEscalated.Status)
	}

	// Test 3: Subsequent messages in live mode do not produce bot replies
	userMsg3, aiMsg3, err := svc.SendUserMessage(ctx, "TKT-001", "user_1", "Fatima Al-Mansoor", "Here is my order number #12345", "en")
	if err != nil {
		t.Fatalf("failed sending live message: %v", err)
	}
	if userMsg3 == nil {
		t.Fatal("expected user message")
	}
	if aiMsg3 != nil {
		t.Errorf("expected nil AI message in Live mode, got %v", aiMsg3)
	}

	// Test 4: Agent reply resolves/progresses ticket
	agentMsg, err := svc.SendAgentMessage(ctx, "TKT-001", "agent_9", "Omar Agent", "support_agent", "Hello Fatima, I am here to help!")
	if err != nil || agentMsg == nil {
		t.Fatalf("failed sending agent message: %v", err)
	}
	if agentMsg.SenderRole != "support_agent" {
		t.Errorf("expected sender role 'support_agent', got %s", agentMsg.SenderRole)
	}

	// Test 5: Resolve ticket
	if err := svc.ResolveTicket(ctx, "TKT-001", "Issue successfully resolved by Omar"); err != nil {
		t.Fatalf("failed resolving ticket: %v", err)
	}
	tkResolved, _ := store.GetTicket(ctx, "TKT-001")
	if tkResolved.Status != support.StatusResolved {
		t.Errorf("expected status %s, got %s", support.StatusResolved, tkResolved.Status)
	}
}

func TestSupportService_VoiceFeatureRequestsMultilingual(t *testing.T) {
	ctx := context.Background()
	store := support.NewMemorySupportStore()
	svc := support.NewService(store)

	dummyAudio := []byte{0x1A, 0x45, 0xDF, 0xA3, 0x01, 0x02, 0x03, 0x04} // simulated webm bytes

	// Test 1: Feature request in Sinhala (si) - New complex feature
	tkSinhala, aiReplySi, err := svc.SubmitVoiceFeatureRequest(ctx, support.VoiceFeatureRequest{
		TicketID:        "TKT-VOICE-SI",
		AudioBytes:      dummyAudio,
		MimeType:        "audio/webm",
		DurationSeconds: 12.5,
		Language:        "si",
		UserKey:         "user_si",
		UserName:        "Kasun Perera",
		UserEmail:       "kasun@example.lk",
		TenantID:        "tenant_colombo",
		PanelID:         "admin",
	})
	if err != nil {
		t.Fatalf("Sinhala voice feature request failed: %v", err)
	}
	if tkSinhala == nil || aiReplySi == nil {
		t.Fatal("expected ticket and ai reply")
	}
	if tkSinhala.Voice == nil {
		t.Fatal("expected preserved voice recording on ticket")
	}
	if len(tkSinhala.Voice.AudioData) == 0 {
		t.Error("expected original audio bytes to be preserved")
	}
	if tkSinhala.Voice.AudioBase64 == "" {
		t.Error("expected base64 audio representation for browser player")
	}
	if tkSinhala.Category != support.CategoryFeatureRequest {
		t.Errorf("expected category %s, got %s", support.CategoryFeatureRequest, tkSinhala.Category)
	}

	// Test 2: Feature request in Arabic (ar)
	tkArabic, aiReplyAr, err := svc.SubmitVoiceFeatureRequest(ctx, support.VoiceFeatureRequest{
		TicketID:        "TKT-VOICE-AR",
		AudioBytes:      dummyAudio,
		MimeType:        "audio/webm",
		DurationSeconds: 8.0,
		Language:        "ar",
		UserKey:         "user_ar",
		UserName:        "Tariq Al-Sabah",
		UserEmail:       "tariq@example.ae",
		TenantID:        "tenant_dubai",
		PanelID:         "admin",
	})
	if err != nil {
		t.Fatalf("Arabic voice feature request failed: %v", err)
	}
	if tkArabic.Language != "ar" {
		t.Errorf("expected language 'ar', got %s", tkArabic.Language)
	}
	if aiReplyAr == nil || aiReplyAr.Content == "" {
		t.Error("expected non-empty localized Arabic AI reply")
	}

	// Test 3: Feature request in Tamil (ta)
	tkTamil, aiReplyTa, err := svc.SubmitVoiceFeatureRequest(ctx, support.VoiceFeatureRequest{
		TicketID:        "TKT-VOICE-TA",
		AudioBytes:      dummyAudio,
		MimeType:        "audio/webm",
		DurationSeconds: 6.2,
		Language:        "ta",
		UserKey:         "user_ta",
		UserName:        "Ravi Kumar",
		TenantID:        "tenant_jaffna",
		PanelID:         "admin",
	})
	if err != nil {
		t.Fatalf("Tamil voice feature request failed: %v", err)
	}
	if tkTamil.Language != "ta" {
		t.Errorf("expected language 'ta', got %s", tkTamil.Language)
	}
	if aiReplyTa == nil || aiReplyTa.Content == "" {
		t.Error("expected non-empty localized Tamil AI reply")
	}
}

func TestSupportMultiPanelAndTenantScoping(t *testing.T) {
	ctx := context.Background()
	store := support.NewMemorySupportStore()
	svc := support.NewService(store)

	// Seed 2 tickets in different tenants
	_, _, _ = svc.SendUserMessage(ctx, "TKT-T1", "u1", "Tenant 1 User", "Question from Tenant 1", "en")
	tk1, _ := store.GetTicket(ctx, "TKT-T1")
	tk1.TenantID = "tenant_alpha"
	_ = store.UpdateTicket(ctx, tk1)

	_, _, _ = svc.SendUserMessage(ctx, "TKT-T2", "u2", "Tenant 2 User", "Question from Tenant 2", "en")
	tk2, _ := store.GetTicket(ctx, "TKT-T2")
	tk2.TenantID = "tenant_beta"
	_ = store.UpdateTicket(ctx, tk2)

	// Tenant Admin filter for tenant_alpha
	alphaTickets, err := store.ListTickets(ctx, support.TicketFilter{TenantID: "tenant_alpha"})
	if err != nil {
		t.Fatalf("failed listing alpha tickets: %v", err)
	}
	for _, tk := range alphaTickets {
		if tk.TenantID != "tenant_alpha" {
			t.Errorf("expected tenant_alpha only, found %s", tk.TenantID)
		}
	}

	// Super Admin filter (sees all tenants)
	allTickets, err := store.ListTickets(ctx, support.TicketFilter{})
	if err != nil {
		t.Fatalf("failed listing all tickets: %v", err)
	}
	if len(allTickets) < 2 {
		t.Errorf("expected at least 2 tickets across tenants, got %d", len(allTickets))
	}

	// Escalate to Super Admin
	err = svc.EscalateToSuperAdmin(ctx, "TKT-T1", "super_admin_panel", "Escalating platform level bug")
	if err != nil {
		t.Fatalf("failed to escalate to super admin: %v", err)
	}
	tkEscalated, _ := store.GetTicket(ctx, "TKT-T1")
	if tkEscalated.Status != support.StatusEscalated {
		t.Errorf("expected status %s, got %s", support.StatusEscalated, tkEscalated.Status)
	}
	if tkEscalated.TargetPanel != "super_admin_panel" {
		t.Errorf("expected target panel 'super_admin_panel', got %s", tkEscalated.TargetPanel)
	}
}

func TestSupportLanguageCookie(t *testing.T) {
	// Set Cookie on Response
	rec := httptest.NewRecorder()
	support.SetSupportLanguageCookie(rec, "si")

	cookieHeader := rec.Header().Get("Set-Cookie")
	if cookieHeader == "" {
		t.Fatal("expected Set-Cookie header")
	}

	// Read Cookie on inbound request
	req := httptest.NewRequest("GET", "/admin/support", nil)
	req.Header.Set("Cookie", cookieHeader)

	detected := support.ResolveSupportLanguage(req)
	if detected != "si" {
		t.Errorf("expected detected language 'si', got %s", detected)
	}
}

func TestGoFilament_SupportResourceRegistration(t *testing.T) {
	// Super Admin Support Resource
	superRes := support.NewSupportResource(
		support.WithScope(support.ScopeSuperAdmin),
	)
	if superRes.Slug() != "support-tickets" {
		t.Errorf("expected slug 'support-tickets', got %s", superRes.Slug())
	}

	// Tenant Admin Support Resource
	tenantRes := support.NewSupportResource(
		support.WithScope(support.ScopeTenantAdmin),
		support.WithTenantScoping(true),
		support.WithSuperAdminPanelID("platform-super-admin"),
	)
	if tenantRes == nil {
		t.Fatal("expected valid tenant support resource")
	}

	// Mount on Panel
	p := filament.NewPanel("admin").
		SuperAdminSupport().
		Resource(superRes)

	if !p.IsSupportEnabled() {
		t.Error("expected support to be enabled on panel")
	}
	if p.GetSupportScope() != support.ScopeSuperAdmin {
		t.Errorf("expected scope %s, got %s", support.ScopeSuperAdmin, p.GetSupportScope())
	}
}
