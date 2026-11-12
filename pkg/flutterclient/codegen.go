package flutterclient

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/golaravel/golaravel/pkg/auth"

	"github.com/golaravel/gofilament/pkg/filament"
)

// GeneratedDir is where GenerateFromPanel writes, relative to the app root.
const GeneratedDir = "lib/generated"

// GenerateOptions configures GenerateFromPanel.
type GenerateOptions struct {
	// User is who the panel is read as. Every authorization callback runs for
	// this user, so it must see every resource, action and page the app
	// should have code for. Defaults to a SimpleShieldUser with the
	// super-admin role and the "*" permission. Pass your own user type when
	// your callbacks type-assert on it.
	User auth.Authenticatable
	// PackageImport is the import of the gofilament_admin library
	// (default "package:gofilament_admin/gofilament_admin.dart").
	PackageImport string
}

func (o GenerateOptions) withDefaults() GenerateOptions {
	if o.User == nil {
		o.User = &filament.SimpleShieldUser{ID: "gofilament-codegen", Roles: []string{"super-admin"}, Permissions: []string{"*"}}
	}
	if o.PackageImport == "" {
		o.PackageImport = "package:gofilament_admin/gofilament_admin.dart"
	}
	return o
}

// GenerateFromPanel writes typed, static Dart for a panel into
// <appDir>/lib/generated:
//
//   - schema.g.dart: a snapshot of /api/schema and the resource, relation and
//     action forms. GoFilamentAdmin(initialSchema: panelSchema) draws the shell
//     from it without waiting for the network; the live schema still decides
//     permissions and replaces it as soon as it arrives.
//   - models.g.dart: a Dart class per resource record (from the Go record
//     type), per form input and per endpoint request / response type.
//   - client.g.dart: PanelClient, a typed API client per resource, relation,
//     row action and typed endpoint (filament.Endpoint).
//   - pages.g.dart: panelPages(...), the AdminPage list of the Go pages that
//     declare a Flutter screen (filament.FlutterPage), with one required
//     builder each.
//   - registry.g.dart: panelAppPages, those pages built from your files in
//     lib/pages (PageFile / PageClass). A page without a file gets a stub
//     there once; existing page files are never changed.
//
// The panel is booted in-process and read through its own API with httptest:
// nothing listens on the network and no handler that writes is called (only
// list / form / option loaders run). Output is deterministic, so CI can
// regenerate and fail on `git diff`. Hand-written code belongs in other files;
// the generator only writes the *.g.dart files and missing page stubs.
func GenerateFromPanel(p *filament.Panel, appDir string, o GenerateOptions) error {
	o = o.withDefaults()
	snap, err := capturePanel(p, o.User)
	if err != nil {
		return err
	}
	files, err := newGenerator(snap, o).render()
	if err != nil {
		return err
	}
	stubs, err := pageStubs(snap, appDir, o.PackageImport)
	if err != nil {
		return err
	}
	for _, name := range sortedKeys(stubs) {
		path := filepath.Join(appDir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(stubs[name]), 0o644); err != nil {
			return err
		}
	}
	for _, name := range sortedKeys(files) {
		path := filepath.Join(appDir, name)
		if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, files[name]) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, files[name], 0o644); err != nil {
			return err
		}
	}
	return nil
}

// CheckFromPanel reports the generated files under appDir that are missing
// or differ from what GenerateFromPanel would write, and the page files that
// do not exist yet (empty when up to date). A page file without its class is
// an error.
func CheckFromPanel(p *filament.Panel, appDir string, o GenerateOptions) ([]string, error) {
	o = o.withDefaults()
	snap, err := capturePanel(p, o.User)
	if err != nil {
		return nil, err
	}
	files, err := newGenerator(snap, o).render()
	if err != nil {
		return nil, err
	}
	stubs, err := pageStubs(snap, appDir, o.PackageImport)
	if err != nil {
		return nil, err
	}
	var stale []string
	for _, name := range sortedKeys(stubs) {
		stale = append(stale, name+" (missing page)")
	}
	for _, name := range sortedKeys(files) {
		old, err := os.ReadFile(filepath.Join(appDir, name))
		if err != nil || !bytes.Equal(old, files[name]) {
			stale = append(stale, name)
		}
	}
	return stale, nil
}

// RenderFromPanel returns the generated files keyed by their path relative
// to the app root, without writing them.
func RenderFromPanel(p *filament.Panel, o GenerateOptions) (map[string][]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("flutterclient: nil panel")
	}
	o = o.withDefaults()
	snap, err := capturePanel(p, o.User)
	if err != nil {
		return nil, err
	}
	g := newGenerator(snap, o)
	return g.render()
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
