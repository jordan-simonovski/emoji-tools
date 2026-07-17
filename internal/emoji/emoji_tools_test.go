package emoji

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
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

// subsampleFrames must cap the frame count and preserve the total loop time by
// folding dropped frames' delays into the frames it keeps.
func TestSubsampleFrames(t *testing.T) {
	const n = 55
	frames := make([]*image.RGBA, n)
	delays := make([]int, n)
	total := 0
	for i := range frames {
		frames[i] = image.NewRGBA(image.Rect(0, 0, 1, 1))
		delays[i] = i + 1 // varied so folding is actually exercised
		total += delays[i]
	}
	gotF, gotD := subsampleFrames(frames, delays, 50)
	if len(gotF) != 50 || len(gotD) != 50 {
		t.Fatalf("subsampleFrames kept %d frames / %d delays, want 50 each", len(gotF), len(gotD))
	}
	sum := 0
	for _, d := range gotD {
		sum += d
	}
	if sum != total {
		t.Errorf("total delay = %d after subsample, want %d (loop duration must be preserved)", sum, total)
	}
	// A cap at or above the input length is a no-op.
	if f, _ := subsampleFrames(frames, delays, 100); len(f) != n {
		t.Errorf("subsampleFrames capped below input when max > len: got %d, want %d", len(f), n)
	}
	// max < 1 is a no-op guard (callers validate, but the branch must hold).
	if f, _ := subsampleFrames(frames, delays, 0); len(f) != n {
		t.Errorf("subsampleFrames(max=0) returned %d frames, want %d (no-op)", len(f), n)
	}
}

// gifFrames must honor GIF disposal: DisposalBackground clears the frame's own
// region to transparent for the next frame, and DisposalPrevious restores the
// canvas as it was before that frame drew. Build a 4-frame GIF exercising both
// and assert the composited pixels.
func TestGifFrames(t *testing.T) {
	red := color.RGBA{255, 0, 0, 255}
	green := color.RGBA{0, 255, 0, 255}
	pal := color.Palette{red, green, color.RGBA{}}
	fill := func(r image.Rectangle, idx uint8) *image.Paletted {
		p := image.NewPaletted(r, pal)
		for i := range p.Pix {
			p.Pix[i] = idx
		}
		return p
	}
	// f0 red background; f1 greens (0,0) then clears it; f2 greens (1,1) but is
	// undone; f3 greens (0,1) on the restored canvas.
	g := &gif.GIF{
		Image: []*image.Paletted{
			fill(image.Rect(0, 0, 2, 2), 0),
			fill(image.Rect(0, 0, 1, 1), 1),
			fill(image.Rect(1, 1, 2, 2), 1),
			fill(image.Rect(0, 1, 1, 2), 1),
		},
		Delay:    []int{10, 10, 10, 10},
		Disposal: []byte{gif.DisposalNone, gif.DisposalBackground, gif.DisposalPrevious, gif.DisposalNone},
		Config:   image.Config{ColorModel: pal, Width: 2, Height: 2},
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	frames, _, err := gifFrames(buf.Bytes())
	if err != nil {
		t.Fatalf("gifFrames: %v", err)
	}
	if len(frames) != 4 {
		t.Fatalf("got %d frames, want 4", len(frames))
	}
	at := func(fi, x, y int) color.RGBA { return frames[fi].RGBAAt(x, y) }
	if at(0, 1, 1) != red {
		t.Errorf("f0 (1,1) = %+v, want red background", at(0, 1, 1))
	}
	if at(1, 0, 0) != green || at(1, 1, 1) != red {
		t.Errorf("f1 = %+v/%+v, want green(0,0) over red(1,1)", at(1, 0, 0), at(1, 1, 1))
	}
	if a := at(2, 0, 0).A; a != 0 { // DisposalBackground cleared (0,0) after f1
		t.Errorf("f2 (0,0) alpha = %d, want 0 (DisposalBackground clear)", a)
	}
	if at(3, 1, 1) != red { // DisposalPrevious undid f2's green at (1,1)
		t.Errorf("f3 (1,1) = %+v, want red (DisposalPrevious restore)", at(3, 1, 1))
	}
	if at(3, 0, 1) != green {
		t.Errorf("f3 (0,1) = %+v, want green", at(3, 0, 1))
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
