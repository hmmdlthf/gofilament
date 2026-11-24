package main

import (
	"embed"
	"io/fs"
)

// flutterweb holds the Flutter admin's web build (demo/client, built with
// `flutter build web`); CI copies it in before compiling. Only a README is
// committed, so a plain `go build` gives a demo without /app.
//
//go:embed all:flutterweb
var flutterweb embed.FS

// flutterBuild returns the embedded web build, if there is one.
func flutterBuild() (fs.FS, bool) {
	sub, err := fs.Sub(flutterweb, "flutterweb")
	if err != nil {
		return nil, false
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, false
	}
	return sub, true
}
