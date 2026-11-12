package flutterclient

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golaravel/gofilament/pkg/filament"
)

// PagesDir is where hand-written Flutter pages live, relative to the app root
// (like Laravel's resources/views). The generator creates a stub for each
// filament.FlutterPage that has no file yet and never changes existing ones.
const PagesDir = "lib/pages"

// snake makes a Dart file name part: "stock-take" -> "stock_take".
func snake(s string) string {
	ws := words(s)
	for i, w := range ws {
		ws[i] = strings.ToLower(w)
	}
	if len(ws) == 0 {
		return "page"
	}
	return strings.Join(ws, "_")
}

// PageFile is the path of a page's hand-written file, relative to the app root.
func PageFile(id string) string { return PagesDir + "/" + snake(id) + "_page.dart" }

// PageClass is the widget class a page's file must define.
func PageClass(id string) string { return pascal(id) + "Page" }

// PageStub returns the starting content of a page's file.
func PageStub(info filament.FlutterPageInfo, packageImport string) string {
	if packageImport == "" {
		packageImport = "package:gofilament_admin/gofilament_admin.dart"
	}
	class := PageClass(info.ID)
	body := fmt.Sprintf("Center(child: Text(%s))", dartString(info.Label))
	if info.Fullscreen {
		body = fmt.Sprintf(`Scaffold(
        appBar: AppBar(title: const Text(%s)),
        body: const Center(child: Text(%s)),
      )`, dartString(info.Label), dartString(info.Label))
	} else {
		body = "const " + body
	}
	return fmt.Sprintf(`import 'package:flutter/material.dart';
import '%s';

import '../generated/client.g.dart';

/// The %s page, declared in Go with filament.FlutterPage(%s, ...).
///
/// This file is yours: the generator created it once and never changes it.
/// Use [client] for typed calls to the panel API.
class %s extends StatelessWidget {
  const %s({super.key, required this.admin});

  final AdminController admin;

  PanelClient get client => PanelClient(admin.api);

  @override
  Widget build(BuildContext context) => %s;
}
`, packageImport, dartString(info.Label), dartString(info.ID), class, class, body)
}

// pageProblem reports why the page file in appDir cannot be used ("" when it
// can, or when it does not exist yet).
func pageProblem(appDir string, info filament.FlutterPageInfo) (missing bool, problem string) {
	b, err := os.ReadFile(filepath.Join(appDir, PageFile(info.ID)))
	if os.IsNotExist(err) {
		return true, ""
	}
	if err != nil {
		return false, err.Error()
	}
	if !regexp.MustCompile(`\bclass\s+` + PageClass(info.ID) + `\b`).Match(b) {
		return false, fmt.Sprintf("%s must define class %s (the generated registry builds it with %s(admin: admin))", PageFile(info.ID), PageClass(info.ID), PageClass(info.ID))
	}
	return false, ""
}

// registryFile wires the hand-written pages into the generated page list.
func (g *generator) registryFile() string {
	var b strings.Builder
	b.WriteString(g.header("Page registry"))
	b.WriteString("\nimport 'package:flutter/widgets.dart';\nimport '" + g.o.PackageImport + "';\n\n")
	for _, p := range g.snap.pages {
		fmt.Fprintf(&b, "import '../pages/%s_page.dart';\n", snake(p.ID))
	}
	b.WriteString("import 'pages.g.dart';\n\n")
	b.WriteString(`/// The panel's Flutter pages, built from your files in lib/pages:
/// ` + "`GoFilamentAdmin(pages: panelAppPages)`" + `.
final List<AdminPage> panelAppPages = panelPages(const PanelPageBuilders(`)
	if len(g.snap.pages) == 0 {
		b.WriteString("));\n")
		return b.String()
	}
	b.WriteString("\n")
	members := pageMembers(g.snap.pages)
	for i := range g.snap.pages {
		fmt.Fprintf(&b, "  %s: _%s,\n", members[i], members[i])
	}
	b.WriteString("));\n")
	for i, p := range g.snap.pages {
		fmt.Fprintf(&b, "\nWidget _%s(BuildContext context, AdminController admin) =>\n    %s(admin: admin);\n", members[i], PageClass(p.ID))
	}
	return b.String()
}

// pageMembers names the PanelPageBuilders parameter of each page.
func pageMembers(pages []filament.FlutterPageInfo) []string {
	names := nameSet{}
	out := make([]string, len(pages))
	for i, p := range pages {
		out[i] = names.unique(camel(p.ID))
	}
	return out
}

// pageStubs lists the stubs to create in appDir, or the first page file that
// cannot be used.
func pageStubs(snap *panelSnapshot, appDir, packageImport string) (map[string]string, error) {
	stubs := map[string]string{}
	seen := map[string]string{}
	for _, p := range snap.pages {
		file := PageFile(p.ID)
		if other, ok := seen[file]; ok {
			return nil, fmt.Errorf("flutterclient: Flutter pages %q and %q both map to %s", other, p.ID, file)
		}
		seen[file] = p.ID
		missing, problem := pageProblem(appDir, p)
		if problem != "" {
			return nil, fmt.Errorf("flutterclient: %s", problem)
		}
		if missing {
			stubs[file] = PageStub(p, packageImport)
		}
	}
	return stubs, nil
}
