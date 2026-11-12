package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/golaravel/golaravel/pkg/artisan"

	"github.com/golaravel/gofilament/pkg/filament"
	"github.com/golaravel/gofilament/pkg/flutterclient"
)

// FlutterGenerateCommand returns `filament:flutter-generate`, which writes the
// typed Dart for a panel (flutterclient.GenerateFromPanel) into a Flutter app.
// The command cannot find your panel by itself, so register it with a
// function that builds it, the same way your server does:
//
//	reg.Register(commands.FlutterGenerateCommand(panels.Admin, "flutter/admin_app", flutterclient.GenerateOptions{}))
//
// Usage: artisan filament:flutter-generate [--out=<app dir>] [--check]
//
// --check writes nothing and fails when the generated files are out of date
// (for CI).
func FlutterGenerateCommand(panel func() *filament.Panel, appDir string, o flutterclient.GenerateOptions) *artisan.Command {
	return &artisan.Command{
		Name:        "filament:flutter-generate",
		Description: "Generate typed Dart (schema snapshot, models, client, pages) for a Flutter app from the panel [--out=dir] [--check]",
		Run: func(args []string) error {
			return runFlutterGenerate(panel, appDir, o, args, os.Stdout)
		},
	}
}

func runFlutterGenerate(panel func() *filament.Panel, appDir string, o flutterclient.GenerateOptions, args []string, out io.Writer) error {
	check := false
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--out="):
			appDir = strings.TrimPrefix(a, "--out=")
		case a == "--check":
			check = true
		default:
			return fmt.Errorf("usage: artisan filament:flutter-generate [--out=<app dir>] [--check]")
		}
	}
	if appDir == "" {
		return fmt.Errorf("filament:flutter-generate: no app directory (pass --out=<dir>)")
	}
	p := panel()
	if check {
		stale, err := flutterclient.CheckFromPanel(p, appDir, o)
		if err != nil {
			return err
		}
		if len(stale) > 0 {
			return fmt.Errorf("generated Flutter code is out of date in %s: %s (run filament:flutter-generate)", appDir, strings.Join(stale, ", "))
		}
		fmt.Fprintf(out, "Generated Flutter code in %s is up to date\n", appDir)
		return nil
	}
	if err := flutterclient.GenerateFromPanel(p, appDir, o); err != nil {
		return err
	}
	fmt.Fprintf(out, "Generated typed Dart in %s/%s\n", appDir, flutterclient.GeneratedDir)
	return nil
}
