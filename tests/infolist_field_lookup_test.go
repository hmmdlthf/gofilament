package tests

import (
	"strings"
	"testing"

	"github.com/golaravel/gofilament/pkg/infolists"
)

func TestInfolistEntryResolvesGoInitialisms(t *testing.T) {
	rec := &struct {
		RemoteIP string `json:"remote_ip"`
	}{RemoteIP: "10.9.8.7"}
	if out := infolists.TextEntry("remote_ip").Render(rec); !strings.Contains(out, "10.9.8.7") {
		t.Fatalf("remote_ip not resolved: %s", out)
	}
}
