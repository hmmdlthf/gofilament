package support_test

import (
	"context"
	"testing"

	"github.com/golaravel/gofilament/pkg/support"
)

func TestSupportService(t *testing.T) {
	ctx := context.Background()
	store := support.NewMemorySupportStore()
	svc := support.NewService(store)

	// Test 1: Send user message in AI mode
	userMsg, aiMsg, err := svc.SendUserMessage(ctx, "TKT-TEST", "amina", "Amina Siddiqui", "How does the Wali supervision work?")
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}
	if userMsg == nil || aiMsg == nil {
		t.Fatal("expected both userMsg and aiMsg to be returned in AI mode")
	}
	if aiMsg.SenderType != "ai" {
		t.Errorf("expected sender type 'ai', got %s", aiMsg.SenderType)
	}

	// Test 2: Switch to Live mode
	if err := svc.SwitchMode(ctx, "TKT-TEST", support.ModeLive); err != nil {
		t.Fatalf("failed to switch mode: %v", err)
	}

	// Test 3: Send user message in Live mode (no automated AI reply)
	userMsg2, aiMsg2, err := svc.SendUserMessage(ctx, "TKT-TEST", "amina", "Amina Siddiqui", "Can I speak with a human support concierge?")
	if err != nil {
		t.Fatalf("failed to send second message: %v", err)
	}
	if userMsg2 == nil {
		t.Fatal("expected user message")
	}
	if aiMsg2 != nil {
		t.Errorf("expected no automated AI message in Live mode, got %v", aiMsg2)
	}

	// Test 4: List tickets
	tickets, err := store.ListTickets(ctx)
	if err != nil {
		t.Fatalf("failed to list tickets: %v", err)
	}
	if len(tickets) < 3 {
		t.Errorf("expected at least 3 tickets (seeded + test), got %d", len(tickets))
	}
}
