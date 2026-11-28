package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"github.com/golaravel/golaravel/pkg/storage"
)

// Uploads in the demo go to a temporary folder that is wiped at every
// reset, on two disks: "demo-public" (product photos and avatars, linked
// directly) and "demo-private" (spec sheets, shown only through signed
// links that expire). The sandbox caps each request at 1 MB, and the
// fields accept only images and PDFs.
const (
	publicDisk  = "demo-public"
	privateDisk = "demo-private"
)

var uploadsRoot string

// setupUploads registers the demo's disks (once, before the panel).
func setupUploads() error {
	root, err := os.MkdirTemp("", "gofilament-demo-uploads-")
	if err != nil {
		return err
	}
	uploadsRoot = root
	// golaravel's default disks live here too, not in the working directory
	if os.Getenv("STORAGE_PATH") == "" {
		_ = os.Setenv("STORAGE_PATH", filepath.Join(root, "app"))
	}
	secret := os.Getenv("APP_KEY")
	pub, err := storage.NewLocalDriver(filepath.Join(root, "public"), "/storage/demo", secret)
	if err != nil {
		return err
	}
	priv, err := storage.NewLocalDriver(filepath.Join(root, "private"), "/storage/demo-private", secret)
	if err != nil {
		return err
	}
	storage.RegisterDisk(publicDisk, pub.SetVisibility(storage.VisibilityPublic))
	storage.RegisterDisk(privateDisk, priv)
	return nil
}

// resetUploads empties both disks; seedPhotos then puts sample photos back.
func resetUploads() {
	if uploadsRoot == "" {
		return
	}
	for _, dir := range []string{"public", "private"} {
		p := filepath.Join(uploadsRoot, dir)
		if err := os.RemoveAll(p); err != nil {
			log.Printf("demo: clearing uploads: %v", err)
		}
		_ = os.MkdirAll(p, 0o755)
	}
}

// productPhoto draws a simple sample photo for a seeded product and stores
// it, returning its path ("" without upload disks, as in tests).
func productPhoto(n int, hue color.RGBA) string {
	if uploadsRoot == "" {
		return ""
	}
	const w, h = 480, 360
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(x+y) / float64(w+h)
			img.Set(x, y, color.RGBA{
				R: uint8(float64(hue.R)*(1-t) + 245*t),
				G: uint8(float64(hue.G)*(1-t) + 245*t),
				B: uint8(float64(hue.B)*(1-t) + 245*t),
				A: 255,
			})
		}
	}
	// a soft circle as the "product"
	cx, cy, r := w/2, h/2, 90
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r {
				img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 235})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	p := fmt.Sprintf("products/sample-%02d.png", n)
	if err := storage.Disk(publicDisk).Put(p, buf.Bytes()); err != nil {
		return ""
	}
	return p
}
