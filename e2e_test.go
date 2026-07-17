package main

import (
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// End-to-end: run every command against real fixtures and assert the output
// files exist with the expected dimensions / frame counts.

const testSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100">` +
	`<rect width="100" height="100" fill="#f9ff69"/>` +
	`<rect x="20" y="20" width="20" height="60" fill="#161616"/></svg>`

func writeFixtures(t *testing.T, dir string) (pngPath, svgPath string) {
	t.Helper()
	pngPath = filepath.Join(dir, "in.png")
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			a := uint8(255)
			if x < 8 || y < 8 { // a transparent margin, so transparency handling is exercised
				a = 0
			}
			img.Set(x, y, color.RGBA{uint8(x * 4), uint8(y * 4), 120, a})
		}
	}
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()

	svgPath = filepath.Join(dir, "in.svg")
	if err := os.WriteFile(svgPath, []byte(testSVG), 0o644); err != nil {
		t.Fatal(err)
	}
	return pngPath, svgPath
}

func pngDims(t *testing.T, path string) (int, int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return cfg.Width, cfg.Height
}

func gifFrameCount(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("decode gif %s: %v", path, err)
	}
	if g.LoopCount != 0 {
		t.Errorf("%s: LoopCount = %d, want 0 (infinite)", path, g.LoopCount)
	}
	return len(g.Image)
}

// quiet silences stdout so the commands' paste-text banners don't spam test output.
func quiet(t *testing.T) func() {
	t.Helper()
	old := os.Stdout
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return func() {}
	}
	os.Stdout = null
	return func() { os.Stdout = old; null.Close() }
}

func TestE2E_Grid(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	_, svg := writeFixtures(t, dir)
	outDir := filepath.Join(dir, "grid")
	if err := runGrid([]string{svg, "-name", "g", "-tile", "16", "-out", outDir}); err != nil {
		t.Fatalf("grid: %v", err)
	}
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			p := filepath.Join(outDir, fmt.Sprintf("g_r%d_c%d.png", r, c))
			if w, h := pngDims(t, p); w != 16 || h != 16 {
				t.Errorf("tile %s = %dx%d, want 16x16", p, w, h)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(outDir, "_preview.png")); err != nil {
		t.Errorf("missing preview: %v", err)
	}
}

func TestE2E_Scroll(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	pngIn, _ := writeFixtures(t, dir)
	if err := runScroll([]string{pngIn, "-name", "s", "-tile", "16", "-frames", "4", "-out", dir}); err != nil {
		t.Fatalf("scroll: %v", err)
	}
	if n := gifFrameCount(t, filepath.Join(dir, "s.gif")); n != 4 {
		t.Errorf("scroll frames = %d, want 4", n)
	}
	// non-divisible frames must be rejected
	if err := runScroll([]string{pngIn, "-tile", "16", "-frames", "5", "-out", dir}); err == nil {
		t.Error("scroll accepted frames that don't divide tile; want error")
	}
}

func TestE2E_Intensify(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	pngIn, _ := writeFixtures(t, dir)
	if err := runIntensify([]string{pngIn, "-name", "i", "-tile", "32", "-frames", "5", "-shake", "4", "-out", dir}); err != nil {
		t.Fatalf("intensify: %v", err)
	}
	if n := gifFrameCount(t, filepath.Join(dir, "i.gif")); n != 5 {
		t.Errorf("intensify frames = %d, want 5", n)
	}
	if err := runIntensify([]string{pngIn, "-tile", "16", "-shake", "8", "-out", dir}); err == nil {
		t.Error("intensify accepted shake >= tile/2; want error")
	}
}

func TestE2E_Yells(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	pngIn, _ := writeFixtures(t, dir)
	out := filepath.Join(dir, "y.png")
	if err := runYells([]string{pngIn, "-out", out}); err != nil {
		t.Fatalf("yells: %v", err)
	}
	if w, h := pngDims(t, out); w != 128 || h != 128 { // square, default size
		t.Errorf("yells = %dx%d, want square 128x128", w, h)
	}
}

func TestE2E_Uwu(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	pngIn, _ := writeFixtures(t, dir)
	out := filepath.Join(dir, "u.png")
	if err := runUwu([]string{pngIn, "-out", out, "-size", "96"}); err != nil {
		t.Fatalf("uwu: %v", err)
	}
	if w, h := pngDims(t, out); w != 96 || h != 96 { // square, requested size
		t.Errorf("uwu = %dx%d, want square 96x96", w, h)
	}
	if err := runUwu([]string{pngIn, "-out", out, "-size", "300"}); err == nil {
		t.Error("uwu accepted size > 256; want error")
	}
}

func TestE2E_Petpet(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	pngIn, _ := writeFixtures(t, dir)
	if err := runPetpet([]string{pngIn, "-name", "p", "-tile", "64", "-out", dir}); err != nil {
		t.Fatalf("petpet: %v", err)
	}
	if n := gifFrameCount(t, filepath.Join(dir, "p.gif")); n != petFrames {
		t.Errorf("petpet frames = %d, want %d", n, petFrames)
	}
}

func TestE2E_Panic(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	pngIn, _ := writeFixtures(t, dir)
	if err := runPanic([]string{pngIn, "-name", "pnc", "-tile", "64", "-frames", "6", "-out", dir}); err != nil {
		t.Fatalf("panic: %v", err)
	}
	if n := gifFrameCount(t, filepath.Join(dir, "pnc.gif")); n != 6 {
		t.Errorf("panic frames = %d, want 6", n)
	}
	if err := runPanic([]string{pngIn, "-tile", "300", "-out", dir}); err == nil {
		t.Error("panic accepted tile > 256; want error")
	}
}

func TestE2E_Party(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	pngIn, _ := writeFixtures(t, dir)
	if err := runParty([]string{pngIn, "-name", "pty", "-tile", "64", "-frames", "8", "-out", dir}); err != nil {
		t.Fatalf("party: %v", err)
	}
	if n := gifFrameCount(t, filepath.Join(dir, "pty.gif")); n != 8 {
		t.Errorf("party frames = %d, want 8", n)
	}
}

func TestE2E_PartyBlob(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	pngIn, _ := writeFixtures(t, dir)
	if err := runPartyBlob([]string{pngIn, "-name", "pb", "-tile", "64", "-frames", "8", "-out", dir}); err != nil {
		t.Fatalf("party-blob: %v", err)
	}
	if n := gifFrameCount(t, filepath.Join(dir, "pb.gif")); n != 8 {
		t.Errorf("party-blob frames = %d, want 8", n)
	}
	if err := runPartyBlob([]string{pngIn, "-tile", "64", "-amp", "0.6", "-out", dir}); err == nil {
		t.Error("party-blob accepted amp >= 0.5; want error")
	}
	// amp valid on its own but too large for the tile -> content < 1px
	if err := runPartyBlob([]string{pngIn, "-tile", "1", "-out", dir}); err == nil {
		t.Error("party-blob accepted tile too small for amp; want error")
	}
}

func TestE2E_Spin(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	pngIn, _ := writeFixtures(t, dir)
	// frames divisible by 4 puts an exact edge-on (cos=0) frame in the loop.
	if err := runSpin([]string{pngIn, "-name", "sp", "-tile", "64", "-frames", "8", "-out", dir}); err != nil {
		t.Fatalf("spin: %v", err)
	}
	if n := gifFrameCount(t, filepath.Join(dir, "sp.gif")); n != 8 {
		t.Errorf("spin frames = %d, want 8", n)
	}
	if op := minOpaquePixels(t, filepath.Join(dir, "sp.gif")); op != 0 {
		t.Errorf("spin: no fully edge-on (blank) frame; min opaque pixels = %d, want 0", op)
	}
}

// minOpaquePixels returns the smallest number of non-transparent pixels across
// all frames of a GIF.
func minOpaquePixels(t *testing.T, path string) int {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("decode gif %s: %v", path, err)
	}
	min := -1
	for _, fr := range g.Image {
		b, count := fr.Bounds(), 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if _, _, _, a := fr.At(x, y).RGBA(); a > 0 {
					count++
				}
			}
		}
		if min < 0 || count < min {
			min = count
		}
	}
	return min
}
