package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golaravel/golaravel/pkg/artisan"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/flutterclient"
)

// DefaultFlutterDir is where the Flutter app of a panel lives by default,
// next to the Go code, like Laravel's resources folder.
const DefaultFlutterDir = "resources/flutter"

func flagValue(args []string, name string) (string, bool) {
	for _, a := range args {
		if strings.HasPrefix(a, "--"+name+"=") {
			return strings.TrimPrefix(a, "--"+name+"="), true
		}
		if a == "--"+name {
			return "", true
		}
	}
	return "", false
}

// MakeFilamentFlutterAppCommand returns `make:filament-flutter-app`: a Flutter
// app that runs gofilament_admin with the typed code filament:flutter-generate
// writes. Existing files are kept.
//
//	artisan make:filament-flutter-app [--dir=resources/flutter] [--name=admin_app]
//	    [--title=Admin] [--panel-path=/admin] [--package-path=<gofilament_admin>]
func MakeFilamentFlutterAppCommand() *artisan.Command {
	return &artisan.Command{
		Name:        "make:filament-flutter-app",
		Description: "Create a Flutter app for a panel in resources/flutter (typed, generated code + your pages)",
		Run: func(args []string) error {
			dir := DefaultFlutterDir
			if v, ok := flagValue(args, "dir"); ok && v != "" {
				dir = v
			}
			o := flutterclient.AdminAppOptions{Typed: true}
			o.Name, _ = flagValue(args, "name")
			o.Title, _ = flagValue(args, "title")
			o.FallbackPanelPath, _ = flagValue(args, "panel-path")
			o.PackagePath, _ = flagValue(args, "package-path")
			if err := flutterclient.GenerateAdminApp(dir, o); err != nil {
				return err
			}
			for _, sub := range []string{flutterclient.PagesDir, flutterclient.GeneratedDir} {
				if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
					return err
				}
			}
			fmt.Printf("Flutter app created in %s. Next: run filament:flutter-generate (see commands.FlutterGenerateCommand), then flutter run.\n", dir)
			return nil
		},
	}
}

// MakeFilamentFlutterPageCommand returns `make:filament-flutter-page <id>`:
// the Dart file of a Flutter page (lib/pages/<id>_page.dart, never
// overwritten) and the Go line that declares it.
//
//	artisan make:filament-flutter-page pos [--label="Point of sale"] [--fullscreen] [--dir=resources/flutter]
func MakeFilamentFlutterPageCommand() *artisan.Command {
	return &artisan.Command{
		Name:        "make:filament-flutter-page",
		Description: "Create a Flutter page in resources/flutter/lib/pages and print its Go declaration",
		Run: func(args []string) error {
			if len(args) < 1 || strings.HasPrefix(args[0], "--") {
				return fmt.Errorf("usage: artisan make:filament-flutter-page <id> [--label=...] [--fullscreen] [--dir=%s]", DefaultFlutterDir)
			}
			id := args[0]
			dir := DefaultFlutterDir
			if v, ok := flagValue(args, "dir"); ok && v != "" {
				dir = v
			}
			label, _ := flagValue(args, "label")
			if label == "" {
				label = strings.ReplaceAll(strings.ReplaceAll(id, "-", " "), "_", " ")
				label = strings.ToUpper(label[:1]) + label[1:]
			}
			_, fullscreen := flagValue(args, "fullscreen")
			info := filament.FlutterPageInfo{ID: id, Label: label, Fullscreen: fullscreen}

			path := filepath.Join(dir, flutterclient.PageFile(id))
			if _, err := os.Stat(path); err == nil {
				fmt.Printf("%s already exists; left unchanged.\n", path)
			} else {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(path, []byte(flutterclient.PageStub(info, "")), 0o644); err != nil {
					return err
				}
				fmt.Printf("Flutter page created: %s (class %s)\n", path, flutterclient.PageClass(id))
			}
			decl := fmt.Sprintf("filament.FlutterPage(%q, %q)", id, label)
			if fullscreen {
				decl += ".Fullscreen()"
			}
			fmt.Printf("Declare it on the panel, then run filament:flutter-generate:\n\n\tpanel.Pages(%s)\n\n", decl)
			return nil
		},
	}
}
