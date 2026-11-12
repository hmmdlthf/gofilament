package filament

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"

	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/flutter"
	"github.com/golaravel/golaravel/pkg/router"
)

// FlutterFS serves the panel's Flutter web build from fsys instead of a
// directory on disk, so it can be compiled into the binary:
//
//	//go:embed all:dist
//	var dist embed.FS
//	sub, _ := fs.Sub(dist, "dist")
//	panel.FlutterFS(sub)
//
// fsys must hold index.html at its root (the contents of build/web).
// filament:flutter-build --embed=<package dir> writes such a package.
func (p *Panel) FlutterFS(fsys fs.FS) *Panel {
	p.engine = EngineFlutter
	p.flutterFS = fsys
	return p
}

// serveFlutterFS mounts the web build in p.flutterFS under the panel path,
// with the same behaviour as the directory server: base href set to the
// panel path, version / build id meta tags, the loading screen, no caching of
// the entry files, and index.html for client-side routes.
func (p *Panel) serveFlutterFS(r *router.Router) {
	fsys := p.flutterFS
	mount := strings.TrimSuffix(p.path, "/")
	version, buildID := p.GetVersion(), p.GetBuildID()
	loading := p.GetFlutterLoadingConfig()

	noCache := func(c *gcontext.Context) {
		c.SetHeader("Cache-Control", "no-cache, no-store, must-revalidate")
		c.SetHeader("Pragma", "no-cache")
		c.SetHeader("Expires", "0")
	}

	serveIndex := func(c *gcontext.Context) error {
		content, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			return c.String(http.StatusNotFound, "Flutter web index.html is missing from the embedded build")
		}
		base := mount + "/"
		content = bytes.Replace(content, []byte(`<base href="/">`), []byte(`<base href="`+base+`">`), 1)
		content = bytes.Replace(content, []byte(`<base href="$FLUTTER_BASE_HREF">`), []byte(`<base href="`+base+`">`), 1)
		if version != "" || buildID != "" {
			meta := ""
			if version != "" {
				meta += `<meta name="app-version" content="` + version + `">`
				c.SetHeader("X-App-Version", version)
			}
			if buildID != "" {
				meta += `<meta name="app-build-id" content="` + buildID + `">`
				c.SetHeader("X-App-Build-ID", buildID)
				content = bytes.Replace(content, []byte(`src="flutter_bootstrap.js"`), []byte(`src="flutter_bootstrap.js?v=`+buildID+`"`), 1)
			}
			content = bytes.Replace(content, []byte("<head>"), []byte("<head>\n  "+meta), 1)
		}
		if loading.Enabled {
			content = flutter.InjectLoadingScreen(content, loading)
		}
		noCache(c)
		return c.Bytes(http.StatusOK, "text/html; charset=utf-8", content)
	}

	files := http.StripPrefix(mount, http.FileServerFS(fsys))
	if mount != "" {
		r.GET(mount, func(c *gcontext.Context) error {
			return c.Redirect(http.StatusMovedPermanently, mount+"/")
		})
	}
	r.GET(mount+"/*filepath", func(c *gcontext.Context) error {
		name := strings.TrimPrefix(path.Clean("/"+c.Param("filepath")), "/")
		if name == "" || name == "." || name == "index.html" {
			return serveIndex(c)
		}
		info, err := fs.Stat(fsys, name)
		if err != nil || info.IsDir() {
			return serveIndex(c) // a client-side route
		}
		if name == "flutter_bootstrap.js" {
			content, err := fs.ReadFile(fsys, name)
			if err == nil {
				if buildID != "" {
					content = bytes.ReplaceAll(content, []byte(`"mainJsPath":"main.dart.js"`), []byte(`"mainJsPath":"main.dart.js?v=`+buildID+`"`))
				}
				content = bytes.ReplaceAll(content, []byte(`serviceWorkerSettings:`), []byte(`/* disabled */ _swSettings:`))
				noCache(c)
				return c.Bytes(http.StatusOK, "application/javascript; charset=utf-8", content)
			}
		}
		switch path.Ext(name) {
		case ".js", ".json", ".wasm", ".otf", ".ttf", ".woff", ".woff2":
			noCache(c)
		}
		files.ServeHTTP(&c.Response, c.Request)
		return nil
	})
}
