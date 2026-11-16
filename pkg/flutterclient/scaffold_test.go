package flutterclient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateFlutterAdmin(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test_scaffold_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	err = GenerateFlutterAdmin(tmpDir, "FleetPulse", "http://localhost:8081/admin")
	if err != nil {
		t.Fatalf("GenerateFlutterAdmin failed: %v", err)
	}

	// Verify pubspec.yaml
	pubspecBytes, err := os.ReadFile(filepath.Join(tmpDir, "pubspec.yaml"))
	if err != nil {
		t.Fatalf("pubspec.yaml not created: %v", err)
	}
	if !strings.Contains(string(pubspecBytes), "name: fleetpulse_admin") {
		t.Errorf("expected pubspec to contain name: fleetpulse_admin")
	}

	// Verify lib/main.dart
	mainBytes, err := os.ReadFile(filepath.Join(tmpDir, "lib", "main.dart"))
	if err != nil {
		t.Fatalf("lib/main.dart not created: %v", err)
	}
	mainStr := string(mainBytes)
	if !strings.Contains(mainStr, "FleetPulse Admin") {
		t.Errorf("expected main.dart to contain FleetPulse Admin")
	}
	if !strings.Contains(mainStr, "CircularAccountButton") {
		t.Errorf("expected main.dart to contain CircularAccountButton")
	}
	if !strings.Contains(mainStr, "AccountModalSheet") {
		t.Errorf("expected main.dart to contain AccountModalSheet")
	}
	if !strings.Contains(mainStr, "SettingsScreen") {
		t.Errorf("expected main.dart to contain SettingsScreen")
	}
	if !strings.Contains(mainStr, "OrdersScreen") {
		t.Errorf("expected main.dart to contain OrdersScreen")
	}
	if !strings.Contains(mainStr, "BillingScreen") {
		t.Errorf("expected main.dart to contain BillingScreen")
	}
	if !strings.Contains(mainStr, "SecuritySettingsScreen") {
		t.Errorf("expected main.dart to contain SecuritySettingsScreen")
	}
	if !strings.Contains(mainStr, "_accountNavPlacement") {
		t.Errorf("expected main.dart to contain _accountNavPlacement")
	}
	if !strings.Contains(mainStr, "SegmentedButton") {
		t.Errorf("expected main.dart to contain SegmentedButton")
	}
}

// The legacy scaffold's demo shortcuts fill in an account; they must only
// exist in debug builds of the generated app.
func TestLegacyScaffoldDemoShortcutsAreDebugOnly(t *testing.T) {
	dir := t.TempDir()
	if err := GenerateFlutterAdmin(dir, "Demo", "/admin"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "lib", "main.dart"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	guard := strings.Index(src, "if (kDebugMode) ...[")
	first := strings.Index(src, "_fillDemo('")
	if guard < 0 || first < 0 || first < guard {
		t.Fatal("demo login shortcuts are not inside a kDebugMode check")
	}
	if !strings.Contains(src, "import 'package:flutter/foundation.dart';") {
		t.Fatal("generated main.dart must import flutter/foundation.dart for kDebugMode")
	}
}
