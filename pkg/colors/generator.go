package colors

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// FromHex generates an 11-shade Palette from a hex color string (e.g. "#6366f1", "6366f1", or "#3b8").
// If the input is invalid hex, it falls back to the default Emerald palette.
func FromHex(hexStr string) Palette {
	r, g, b, err := parseHex(hexStr)
	if err != nil {
		return Emerald.Clone()
	}
	return FromRGB(r, g, b)
}

// FromRGB generates an 11-shade Palette from 8-bit Red, Green, Blue values.
func FromRGB(r, g, b uint8) Palette {
	h, s, l := rgbToHSL(r, g, b)
	return FromHSL(h, s, l)
}

// FromHSL generates an 11-shade Palette from Hue (0..360), Saturation (0..1), and Lightness (0..1).
func FromHSL(h, s, l float64) Palette {
	// Reference target lightness scale for Tailwind-style 50..950 steps
	targetL := map[int]float64{
		50:  0.97,
		100: 0.93,
		200: 0.86,
		300: 0.74,
		400: 0.60,
		500: l, // preserve base lightness for 500
		600: 0.45,
		700: 0.36,
		800: 0.27,
		900: 0.18,
		950: 0.10,
	}

	palette := make(Palette, len(Shades))

	for _, shade := range Shades {
		var curL float64
		var curS float64

		if shade == 500 {
			curL = l
			curS = s
		} else if shade < 500 {
			// Interpolate lightness between base l and target high lightness
			t := float64(500-shade) / 450.0 // 0.0 at 500 -> 1.0 at 50
			curL = l + (targetL[shade]-l)*t
			// Subdue saturation slightly for very light tints to prevent neon glare
			curS = s * (1.0 - t*0.35)
		} else {
			// Interpolate lightness between base l and target deep dark
			t := float64(shade-500) / 450.0 // 0.0 at 500 -> 1.0 at 950
			curL = l - (l-targetL[shade])*t
			// Keep saturation rich for dark shades
			curS = math.Min(1.0, s*(1.0+t*0.1))
		}

		// Clamp values
		curL = math.Max(0.0, math.Min(1.0, curL))
		curS = math.Max(0.0, math.Min(1.0, curS))

		r, g, b := hslToRGB(h, curS, curL)
		palette[shade] = fmt.Sprintf("#%02x%02x%02x", r, g, b)
	}

	return palette
}

func parseHex(s string) (uint8, uint8, uint8, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")

	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	} else if len(s) == 4 { // 4-char #RGBA
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	} else if len(s) == 8 { // 8-char #RRGGBBAA
		s = s[:6]
	}

	if len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex length: %d", len(s))
	}

	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, err
	}

	r := uint8((v >> 16) & 0xFF)
	g := uint8((v >> 8) & 0xFF)
	b := uint8(v & 0xFF)

	return r, g, b, nil
}

func rgbToHSL(r, g, b uint8) (float64, float64, float64) {
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	max := math.Max(rf, math.Max(gf, bf))
	min := math.Min(rf, math.Min(gf, bf))
	delta := max - min

	l := (max + min) / 2.0
	var h, s float64

	if delta == 0 {
		h = 0
		s = 0
	} else {
		if l < 0.5 {
			s = delta / (max + min)
		} else {
			s = delta / (2.0 - max - min)
		}

		if max == rf {
			h = (gf - bf) / delta
			if gf < bf {
				h += 6.0
			}
		} else if max == gf {
			h = (bf-rf)/delta + 2.0
		} else {
			h = (rf-gf)/delta + 4.0
		}
		h *= 60.0
	}

	return h, s, l
}

func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	if s == 0 {
		v := uint8(math.Round(l * 255.0))
		return v, v, v
	}

	var q float64
	if l < 0.5 {
		q = l * (1.0 + s)
	} else {
		q = l + s - (l * s)
	}
	p := 2.0*l - q

	hk := h / 360.0
	tr := hk + 1.0/3.0
	tg := hk
	tb := hk - 1.0/3.0

	return colorCalc(p, q, tr), colorCalc(p, q, tg), colorCalc(p, q, tb)
}

func colorCalc(p, q, t float64) uint8 {
	if t < 0 {
		t += 1.0
	}
	if t > 1.0 {
		t -= 1.0
	}
	var res float64
	if t < 1.0/6.0 {
		res = p + (q-p)*6.0*t
	} else if t < 1.0/2.0 {
		res = q
	} else if t < 2.0/3.0 {
		res = p + (q-p)*(2.0/3.0-t)*6.0
	} else {
		res = p
	}
	return uint8(math.Round(math.Max(0.0, math.Min(1.0, res)) * 255.0))
}
