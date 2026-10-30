package tests

import (
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/notifications"
)

func TestNotificationBuilderAndSerialization(t *testing.T) {
	notif := notifications.Make("Order Created").
		Success().
		Body("Order #4219 was placed successfully.").
		DurationMs(5000)

	jsonStr, err := notif.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize notification to JSON: %v", err)
	}

	if !strings.Contains(jsonStr, `"title":"Order Created"`) {
		t.Errorf("Expected JSON to contain title, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"type":"success"`) {
		t.Errorf("Expected JSON to contain type success, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"body":"Order #4219 was placed successfully."`) {
		t.Errorf("Expected JSON to contain body text, got: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"duration":5000`) {
		t.Errorf("Expected JSON to contain duration 5000, got: %s", jsonStr)
	}
}
