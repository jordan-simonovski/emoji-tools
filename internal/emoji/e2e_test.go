package emoji

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
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
	if lo, _ := opaqueRange(t, filepath.Join(dir, "sp.gif")); lo != 0 {
		t.Errorf("spin: no fully edge-on (blank) frame; min opaque pixels = %d, want 0", lo)
	}
}

func TestE2E_Confetti(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	pngIn, _ := writeFixtures(t, dir)
	// -frames below the overlay's native count exercises subsampling.
	if err := runConfetti([]string{pngIn, "-name", "cf", "-tile", "64", "-frames", "10", "-out", dir}); err != nil {
		t.Fatalf("confetti: %v", err)
	}
	if n := gifFrameCount(t, filepath.Join(dir, "cf.gif")); n != 10 {
		t.Errorf("confetti frames = %d, want 10 (subsampled)", n)
	}
	// The overlay must actually composite and animate: a no-op overlay would
	// repeat the static base tile, giving an identical opaque count every frame.
	if lo, hi := opaqueRange(t, filepath.Join(dir, "cf.gif")); hi <= lo {
		t.Errorf("confetti frames don't vary (opaque range %d..%d); overlay not compositing", lo, hi)
	}
}

func TestE2E_Bongocat(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	pngIn, _ := writeFixtures(t, dir)
	if err := runBongocat([]string{pngIn, "-name", "bc", "-tile", "64", "-out", dir}); err != nil {
		t.Fatalf("bongocat: %v", err)
	}
	if n := gifFrameCount(t, filepath.Join(dir, "bc.gif")); n != 2 {
		t.Errorf("bongocat frames = %d, want 2", n)
	}
	if err := runBongocat([]string{pngIn, "-scale", "1.5", "-out", dir}); err == nil {
		t.Error("bongocat accepted scale > 1; want error")
	}
}

func TestE2E_Statham(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	// The base statham.gif is itself animated (a dancing figure), so a varying
	// opaque-pixel count proves nothing about the overlay. Instead feed a solid
	// cyan fixture — a color absent from the bronze source — and assert cyan
	// pixels land in the output, which only a working head overlay can produce.
	cyanIn := filepath.Join(dir, "cyan.png")
	writeSolidPNG(t, cyanIn, color.RGBA{0, 255, 255, 255})
	if err := runStatham([]string{cyanIn, "-name", "st", "-tile", "64", "-out", dir}); err != nil {
		t.Fatalf("statham: %v", err)
	}
	const wantFrames = 30 // one output frame per source frame in statham.gif
	if n := gifFrameCount(t, filepath.Join(dir, "st.gif")); n != wantFrames {
		t.Errorf("statham frames = %d, want %d", n, wantFrames)
	}
	if n := countColorNear(t, filepath.Join(dir, "st.gif"), color.RGBA{0, 255, 255, 255}, 40); n < 100 {
		t.Errorf("statham: only %d cyan pixels in output; overlay not compositing onto the figure", n)
	}

	// Preview needs no input and stamps a magenta head box; assert the box drew.
	if err := runStatham([]string{"-preview", "-name", "stp", "-tile", "64", "-out", dir}); err != nil {
		t.Fatalf("statham -preview: %v", err)
	}
	if n := gifFrameCount(t, filepath.Join(dir, "stp.gif")); n != wantFrames {
		t.Errorf("statham preview frames = %d, want %d", n, wantFrames)
	}
	if n := countColorNear(t, filepath.Join(dir, "stp.gif"), color.RGBA{255, 0, 255, 255}, 40); n < 30 {
		t.Errorf("statham -preview: only %d magenta pixels; head box not drawn", n)
	}
	// -preview=true (the standard Go bool-flag form) must route to preview, not error.
	if err := runStatham([]string{"-preview=true", "-name", "stp2", "-tile", "64", "-out", dir}); err != nil {
		t.Fatalf("statham -preview=true: %v", err)
	}

	// Error paths, mirroring the sibling makers' negative assertions.
	if err := runStatham([]string{cyanIn, "-tile", "9999", "-out", dir}); err == nil {
		t.Error("statham accepted tile > max; want error")
	}
	if err := runStatham([]string{cyanIn, "-scale", "0", "-out", dir}); err == nil {
		t.Error("statham accepted scale <= 0; want error")
	}
	if err := runStatham([]string{"-out", dir}); err == nil {
		t.Error("statham accepted no input without -preview; want error")
	}
}

func TestE2E_Fistpump(t *testing.T) {
	defer quiet(t)()
	dir := t.TempDir()
	// Same trick as statham: a solid cyan fixture proves the held image composited,
	// since the arm sprite is yellow/black only.
	cyanIn := filepath.Join(dir, "cyan.png")
	writeSolidPNG(t, cyanIn, color.RGBA{0, 255, 255, 255})
	if err := runFistpump([]string{cyanIn, "-name", "fp", "-tile", "64", "-out", dir}); err != nil {
		t.Fatalf("fistpump: %v", err)
	}
	if n := gifFrameCount(t, filepath.Join(dir, "fp.gif")); n != 2 {
		t.Errorf("fistpump frames = %d, want 2", n)
	}
	if n := countColorNear(t, filepath.Join(dir, "fp.gif"), color.RGBA{0, 255, 255, 255}, 40); n < 100 {
		t.Errorf("fistpump: only %d cyan pixels in output; held image not compositing", n)
	}
	// The pump IS the animation: the fist swings between the two frames and drags
	// the held image with it. Frozen fist coordinates would still satisfy every
	// other assertion here, so check the image actually moves.
	if c := centroidsX(t, filepath.Join(dir, "fp.gif"), isCyan); len(c) != 2 {
		t.Errorf("fistpump: got %d frames of held image, want 2", len(c))
	} else if d := c[0] - c[1]; d > -5 && d < 5 {
		t.Errorf("fistpump: held image barely moves between frames (%.1f -> %.1f); the pump is missing", c[0], c[1])
	}

	// -side right with a real image. Preview skips the held-image draw entirely,
	// so without this the mirror branch is never exercised alongside compositing --
	// which is exactly the "mirrors the arm, not the logo" contract.
	if err := runFistpump([]string{cyanIn, "-side", "right", "-name", "fpr2", "-tile", "64", "-out", dir}); err != nil {
		t.Fatalf("fistpump -side right: %v", err)
	}
	if n := countColorNear(t, filepath.Join(dir, "fpr2.gif"), color.RGBA{0, 255, 255, 255}, 40); n < 100 {
		t.Errorf("fistpump -side right: only %d cyan pixels; held image not compositing on the mirrored side", n)
	}

	// -side right must mirror the arm. Compare the previews' opaque centroids: a
	// true mirror puts them equidistant from the tile's midline, which a plain
	// pixel-equality check can't assert (the box/image rects round asymmetrically).
	const tile = 64
	if err := runFistpump([]string{"-preview", "-name", "fpl", "-tile", "64", "-out", dir}); err != nil {
		t.Fatalf("fistpump -preview: %v", err)
	}
	if err := runFistpump([]string{"-preview", "-side", "right", "-name", "fpr", "-tile", "64", "-out", dir}); err != nil {
		t.Fatalf("fistpump -preview -side right: %v", err)
	}
	l := centroidsX(t, filepath.Join(dir, "fpl.gif"), isOpaque)[0]
	r := centroidsX(t, filepath.Join(dir, "fpr.gif"), isOpaque)[0]
	if diff := l + r - (tile - 1); diff < -2 || diff > 2 {
		t.Errorf("fistpump centroids: left %.1f + right %.1f = %.1f, want ~%d (mirrored)", l, r, l+r, tile-1)
	}
	if l >= r { // the native arm leans left; mirrored it must lean right
		t.Errorf("fistpump -side right centroid %.1f is not right of left's %.1f", r, l)
	}
	if n := countColorNear(t, filepath.Join(dir, "fpl.gif"), color.RGBA{255, 0, 255, 255}, 40); n < 30 {
		t.Errorf("fistpump -preview: only %d magenta pixels; fist box not drawn", n)
	}

	// Error paths, mirroring the sibling makers' negative assertions.
	if err := runFistpump([]string{cyanIn, "-tile", "9999", "-out", dir}); err == nil {
		t.Error("fistpump accepted tile > max; want error")
	}
	if err := runFistpump([]string{cyanIn, "-scale", "1.5", "-out", dir}); err == nil {
		t.Error("fistpump accepted scale > 1; want error")
	}
	if err := runFistpump([]string{cyanIn, "-side", "sideways", "-out", dir}); err == nil {
		t.Error("fistpump accepted a bogus -side; want error")
	}
	if err := runFistpump([]string{"-out", dir}); err == nil {
		t.Error("fistpump accepted no input without -preview; want error")
	}
}

func isOpaque(_, _, _ uint8, a uint8) bool { return a > 0 }

// isCyan matches the writeSolidPNG cyan fixture with enough slack for the GIF
// palette quantizer to shift it.
func isCyan(r, g, b, a uint8) bool { return a > 0 && r < 80 && g > 180 && b > 180 }

// centroidsX returns the mean x of every pixel matching keep, one entry per GIF
// frame — a cheap handle on which way a sprite leans and whether it moves.
func centroidsX(t *testing.T, path string, keep func(r, g, b, a uint8) bool) []float64 {
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
	out := make([]float64, 0, len(g.Image))
	for i, fr := range g.Image {
		b := fr.Bounds()
		sum, n := 0, 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				cr, cg, cb, ca := fr.At(x, y).RGBA()
				if keep(uint8(cr>>8), uint8(cg>>8), uint8(cb>>8), uint8(ca>>8)) {
					sum += x
					n++
				}
			}
		}
		if n == 0 {
			t.Fatalf("%s: frame %d has no matching pixels", path, i)
		}
		out = append(out, float64(sum)/float64(n))
	}
	return out
}

// writeSolidPNG writes a 64x64 image filled with c.
func writeSolidPNG(t *testing.T, path string, c color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	draw.Draw(img, img.Bounds(), &image.Uniform{c}, image.Point{}, draw.Src)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
}

// countColorNear totals, across all GIF frames, the pixels within tol (per
// channel) of target — used to confirm a distinctly-colored overlay composited.
func countColorNear(t *testing.T, path string, target color.RGBA, tol int) int {
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
	near := func(a, b uint8) bool { return int(a)-int(b) <= tol && int(b)-int(a) <= tol }
	count := 0
	for _, fr := range g.Image {
		b := fr.Bounds()
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, gg, bb, _ := fr.At(x, y).RGBA()
				if near(uint8(r>>8), target.R) && near(uint8(gg>>8), target.G) && near(uint8(bb>>8), target.B) {
					count++
				}
			}
		}
	}
	return count
}

// opaqueRange returns the smallest and largest number of non-transparent pixels
// across all frames of a GIF.
func opaqueRange(t *testing.T, path string) (lo, hi int) {
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
	lo = -1
	for _, fr := range g.Image {
		b, count := fr.Bounds(), 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				if _, _, _, a := fr.At(x, y).RGBA(); a > 0 {
					count++
				}
			}
		}
		if lo < 0 || count < lo {
			lo = count
		}
		if count > hi {
			hi = count
		}
	}
	return lo, hi
}
