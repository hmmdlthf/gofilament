package assets

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	gcontext "github.com/golaravel/golaravel/pkg/context"
	"github.com/golaravel/golaravel/pkg/router"
)

//go:embed dist/*
var distFS embed.FS

// DistFS returns the embedded file system for GoFilament static assets.
func DistFS() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return distFS
	}
	return sub
}

// RegisterRoutes registers the embedded asset routes under the panel asset prefix.
func RegisterRoutes(r *router.Router, prefix string) {
	sub := DistFS()
	fileServer := http.FileServer(http.FS(sub))

	cleanPrefix := strings.TrimRight(prefix, "/")
	pattern := cleanPrefix + "/*filepath"

	r.GET(pattern, func(c *gcontext.Context) error {
		// Strip prefix for file server
		http.StripPrefix(cleanPrefix, fileServer).ServeHTTP(&c.Response, c.Request)
		return nil
	})
}
