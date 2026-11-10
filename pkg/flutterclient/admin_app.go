package flutterclient

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AdminAppOptions configures GenerateAdminApp.
type AdminAppOptions struct {
	// Name is the Dart package name of the generated app (snake_case).
	Name string
	// Title is the initial page title, shown until the panel brand loads.
	Title string
	// PackagePath is a local path to the gofilament_admin package (relative to
	// the generated app). Used during framework development.
	PackagePath string
	// PackageGitURL / PackageGitRef pull gofilament_admin from the GoFilament
	// repository instead (used when PackagePath is empty).
	PackageGitURL string
	PackageGitRef string
	// FallbackPanelPath is the panel path native builds call when no base URL
	// is configured (web builds read it from the page URL).
	FallbackPanelPath string
}

// GenerateAdminApp writes a thin Flutter application that runs the reusable
// gofilament_admin package. Everything it shows comes from the panel API, so
// the generated code is a few lines: it is the place to add application pages
// (for example a point of sale screen) and nothing else.
func GenerateAdminApp(targetDir string, o AdminAppOptions) error {
	if o.Name == "" {
		o.Name = "admin_app"
	}
	if o.Title == "" {
		o.Title = "Admin"
	}
	if o.FallbackPanelPath == "" {
		o.FallbackPanelPath = "/admin"
	}
	if o.PackageGitURL == "" {
		o.PackageGitURL = "https://github.com/hmmdlthf/gofilament.git"
	}

	dep := fmt.Sprintf("  gofilament_admin:\n    path: %s\n", o.PackagePath)
	if o.PackagePath == "" {
		dep = fmt.Sprintf("  gofilament_admin:\n    git:\n      url: %s\n      path: flutter/gofilament_admin\n", o.PackageGitURL)
		if o.PackageGitRef != "" {
			dep += fmt.Sprintf("      ref: %s\n", o.PackageGitRef)
		}
	}

	files := map[string]string{
		"pubspec.yaml": fmt.Sprintf(`name: %s
description: %s
publish_to: none
version: 1.0.0+1

environment:
  sdk: '>=3.4.0 <4.0.0'

dependencies:
  flutter:
    sdk: flutter
%s
dev_dependencies:
  flutter_test:
    sdk: flutter
  flutter_lints: ^5.0.0

flutter:
  uses-material-design: true
`, o.Name, o.Title, dep),

		"analysis_options.yaml": "include: package:flutter_lints/flutter.yaml\n",

		".gitignore": ".dart_tool/\nbuild/\n",

		"lib/main.dart": fmt.Sprintf(`import 'package:flutter/material.dart';
import 'package:gofilament_admin/gofilament_admin.dart';

void main() {
  runApp(GoFilamentAdmin(
    api: AdminApi(fallbackPanelPath: '%s'),
    // Add application pages here, e.g.
    // pages: [AdminPage(id: 'pos', label: 'Point of Sale', fullscreen: true, builder: (context, admin) => ...)],
  ));
}
`, o.FallbackPanelPath),

		"web/index.html": fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <base href="/">
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <meta name="robots" content="noindex">
  <title>%s</title>
</head>
<body>
  <script src="flutter_bootstrap.js" async></script>
</body>
</html>
`, htmlEscape(o.Title)),
	}

	for name, content := range files {
		path := filepath.Join(targetDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func htmlEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}
