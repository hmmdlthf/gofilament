package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
//	reg.Register(commands.FlutterGenerateCommand(panels.Admin, commands.DefaultFlutterDir, flutterclient.GenerateOptions{}))
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

// FlutterBuildCommand returns `filament:flutter-build`: it regenerates the
// typed code (like filament:flutter-generate), runs `flutter build web` and,
// with --embed, copies the build into a Go package for panel.FlutterFS.
//
//	artisan filament:flutter-build [--dir=resources/flutter] [--embed=internal/adminweb]
//	    [--no-generate] [-- <extra flutter build web args>]
func FlutterBuildCommand(panel func() *filament.Panel, appDir string, o flutterclient.GenerateOptions) *artisan.Command {
	return &artisan.Command{
		Name:        "filament:flutter-build",
		Description: "Generate the typed Dart, build the Flutter web app and optionally embed it [--dir] [--embed=pkg dir] [--no-generate] [-- flutter args]",
		Run: func(args []string) error {
			return runFlutterBuild(panel, appDir, o, flutterclient.BuildOptions{}, args, os.Stdout)
		},
	}
}

func runFlutterBuild(panel func() *filament.Panel, appDir string, o flutterclient.GenerateOptions, b flutterclient.BuildOptions, args []string, out io.Writer) error {
	generate := true
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			b.Args = append(b.Args, args[i+1:]...)
			i = len(args)
		case strings.HasPrefix(a, "--dir="):
			appDir = strings.TrimPrefix(a, "--dir=")
		case strings.HasPrefix(a, "--embed="):
			b.EmbedDir = strings.TrimPrefix(a, "--embed=")
		case a == "--no-generate":
			generate = false
		default:
			return fmt.Errorf("usage: artisan filament:flutter-build [--dir=<app dir>] [--embed=<package dir>] [--no-generate] [-- <flutter args>]")
		}
	}
	if appDir == "" {
		appDir = DefaultFlutterDir
	}
	if generate {
		if err := flutterclient.GenerateFromPanel(panel(), appDir, o); err != nil {
			return err
		}
	}
	b.Output = out
	if err := flutterclient.BuildWeb(appDir, b); err != nil {
		return err
	}
	if b.EmbedDir != "" {
		fmt.Fprintf(out, "Web build embedded in %s: panel.FlutterFS(%s.FS)\n", b.EmbedDir, filepath.Base(b.EmbedDir))
	} else {
		fmt.Fprintf(out, "Web build in %s/build/web: panel.Flutter(%q)\n", appDir, filepath.Join(appDir, "build", "web"))
	}
	return nil
}
