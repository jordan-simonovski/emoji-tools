package main

import (
	"image"
	"image/color"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"new-pull-request.png": "new-pull-request",
		"ClickHouse Logo.svg":  "clickhouse_logo",
		"/a/b/HyperDX.SVG":     "hyperdx",
		"weird!!name@@.png":    "weird_name",
		"___.png":              "emoji",
	}
	for in, want := range cases {
		if got := sanitizeName(in); got != want {
			t.Errorf("sanitizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCentsFromMillis(t *testing.T) {
	cases := map[int]int{0: 1, 4: 1, 40: 4, 33: 3, 100: 10}
	for ms, want := range cases {
		if got := centsFromMillis(ms); got != want {
			t.Errorf("centsFromMillis(%d) = %d, want %d", ms, got, want)
		}
	}
}

// sliceTiles then reassemble must reproduce the source exactly.
func TestGridReassembles(t *testing.T) {
	const cols, rows, tile = 3, 3, 8
	src := image.NewRGBA(image.Rect(0, 0, cols*tile, rows*tile))
	for y := 0; y < rows*tile; y++ {
		for x := 0; x < cols*tile; x++ {
			src.Set(x, y, color.RGBA{uint8(x), uint8(y), uint8(x + y), 255})
		}
	}
	tiles := sliceTiles(src, cols, rows, tile)
	if len(tiles) != cols*rows {
		t.Fatalf("got %d tiles, want %d", len(tiles), cols*rows)
	}
	got := image.NewRGBA(src.Bounds())
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			tl := tiles[r*cols+c]
			for y := 0; y < tile; y++ {
				for x := 0; x < tile; x++ {
					got.Set(c*tile+x, r*tile+y, tl.At(x, y))
				}
			}
		}
	}
	for i := range src.Pix {
		if src.Pix[i] != got.Pix[i] {
			t.Fatalf("reassembled image differs at byte %d", i)
		}
	}
}

// At 0 degrees the hue-rotate matrix reduces to the identity, so every opaque
// pixel must come back unchanged and transparent pixels must be left alone. This
// pins the hand-written matrix against a transposed/sign-flipped coefficient.
func TestHueRotateIdentity(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			a := uint8(255)
			if x == 0 { // a transparent column, exercising the A==0 skip
				a = 0
			}
			src.Set(x, y, color.RGBA{uint8(x * 40), uint8(y * 40), 200, a})
		}
	}
	got := hueRotate(src, 0)
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			in, out := src.RGBAAt(x, y), got.RGBAAt(x, y)
			if in.A == 0 {
				if out.A != 0 { // transparent stays transparent (RGB is don't-care)
					t.Fatalf("hueRotate turned transparent pixel (%d,%d) opaque: %+v", x, y, out)
				}
				continue
			}
			if in != out { // opaque pixel must be byte-for-byte identical at 0 degrees
				t.Fatalf("hueRotate(_, 0) changed opaque pixel (%d,%d): got %+v, want %+v", x, y, out, in)
			}
		}
	}
}

// A scroll shifted by a full tile width must return to the start (seamless loop).
func TestScrollSeamless(t *testing.T) {
	const tile = 16
	period := image.NewRGBA(image.Rect(0, 0, tile, tile))
	for y := 0; y < tile; y++ {
		for x := 0; x < tile; x++ {
			period.Set(x, y, color.RGBA{uint8(x * 8), 0, 0, 255})
		}
	}
	doubled := image.NewRGBA(image.Rect(0, 0, 2*tile, tile))
	for x := 0; x < tile; x++ {
		for y := 0; y < tile; y++ {
			doubled.Set(x, y, period.At(x, y))
			doubled.Set(x+tile, y, period.At(x, y))
		}
	}
	// window at offset 0 and at offset tile must be identical
	for y := 0; y < tile; y++ {
		for x := 0; x < tile; x++ {
			if doubled.RGBAAt(x, y) != doubled.RGBAAt(x+tile, y) {
				t.Fatalf("scroll not seamless at (%d,%d)", x, y)
			}
		}
	}
}
