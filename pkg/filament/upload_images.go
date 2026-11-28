package filament

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"

	"github.com/golaravel/gofilament/pkg/forms"
)

// processImage applies a field's crop ratio and resize target to an
// uploaded image. It returns ok false (keep the original bytes) when the
// field asks for neither, the file is not a PNG, JPEG or still GIF, or it
// cannot be decoded; images are only ever made smaller.
func processImage(field *forms.FileUploadField, contentType string, r io.Reader) ([]byte, bool) {
	width, height, mode, ratio := field.ImageProcessing()
	if width <= 0 && height <= 0 && ratio <= 0 {
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(r, maxUploadBytes+1))
	if err != nil || len(data) > maxUploadBytes {
		return nil, false
	}
	var src image.Image
	switch contentType {
	case "image/png":
		src, err = png.Decode(bytes.NewReader(data))
	case "image/jpeg":
		src, err = jpeg.Decode(bytes.NewReader(data))
	case "image/gif":
		var g *gif.GIF
		g, err = gif.DecodeAll(bytes.NewReader(data))
		if err == nil {
			if len(g.Image) != 1 {
				return nil, false // keep animations as they are
			}
			src = g.Image[0]
		}
	default:
		return nil, false
	}
	if err != nil {
		return nil, false
	}
	// refuse decompression bombs
	if b := src.Bounds(); b.Dx()*b.Dy() > 50_000_000 {
		return nil, false
	}

	img := src
	if ratio > 0 {
		img = cropToRatio(img, ratio)
	}
	if width > 0 || height > 0 {
		img = fitImage(img, width, height, mode)
	}
	if img == src {
		return nil, false
	}

	var buf bytes.Buffer
	switch contentType {
	case "image/png":
		err = png.Encode(&buf, img)
	case "image/jpeg":
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 88})
	case "image/gif":
		err = gif.Encode(&buf, img, nil)
	}
	if err != nil {
		return nil, false
	}
	return buf.Bytes(), true
}

// cropToRatio keeps the largest centred area of img with width/height =
// ratio.
func cropToRatio(img image.Image, ratio float64) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return img
	}
	cw, ch := w, int(math.Round(float64(w)/ratio))
	if ch > h {
		cw, ch = int(math.Round(float64(h)*ratio)), h
	}
	if cw == w && ch == h {
		return img
	}
	x0 := b.Min.X + (w-cw)/2
	y0 := b.Min.Y + (h-ch)/2
	out := image.NewRGBA(image.Rect(0, 0, cw, ch))
	draw.Draw(out, out.Bounds(), img, image.Pt(x0, y0), draw.Src)
	return out
}

// fitImage shrinks img into maxW × maxH (0 = unbounded) by mode.
func fitImage(img image.Image, maxW, maxH int, mode string) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return img
	}
	tw, th := maxW, maxH
	if tw <= 0 {
		tw = w
	}
	if th <= 0 {
		th = h
	}
	switch mode {
	case "force":
		if tw >= w && th >= h {
			return img
		}
		return scale(img, min(tw, w), min(th, h))
	case "cover":
		if maxW > 0 && maxH > 0 {
			s := math.Max(float64(tw)/float64(w), float64(th)/float64(h))
			if s >= 1 {
				return cropToRatio(img, float64(tw)/float64(th))
			}
			sw, sh := int(math.Ceil(float64(w)*s)), int(math.Ceil(float64(h)*s))
			return cropToRatio(scale(img, sw, sh), float64(tw)/float64(th))
		}
	}
	// contain
	s := math.Min(float64(tw)/float64(w), float64(th)/float64(h))
	if s >= 1 {
		return img
	}
	return scale(img, max(1, int(math.Round(float64(w)*s))), max(1, int(math.Round(float64(h)*s))))
}

// scale resizes img to w × h by averaging the source pixels under each
// destination pixel (a box filter, good for shrinking).
func scale(img image.Image, w, h int) image.Image {
	b := img.Bounds()
	sw, sh := b.Dx(), b.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0 := b.Min.Y + y*sh/h
		y1 := b.Min.Y + max((y+1)*sh/h, y*sh/h+1)
		for x := 0; x < w; x++ {
			x0 := b.Min.X + x*sw/w
			x1 := b.Min.X + max((x+1)*sw/w, x*sw/w+1)
			var r, g, bl, a, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, ca := img.At(sx, sy).RGBA()
					r, g, bl, a = r+uint64(cr), g+uint64(cg), bl+uint64(cb), a+uint64(ca)
					n++
				}
			}
			out.Set(x, y, color.RGBA64{R: uint16(r / n), G: uint16(g / n), B: uint16(bl / n), A: uint16(a / n)})
		}
	}
	return out
}
