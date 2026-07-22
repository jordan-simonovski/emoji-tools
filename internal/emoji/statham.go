package emoji

import (
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"path/filepath"
)

//go:embed assets/statham.gif
var stathamGIF []byte

// headBand is the vertical slice below the figure's crown (in native 128px) that
// the head occupies; detectHead searches it for the head's horizontal center.
const headBand = 40

func runStatham(args []string) error {
	fs := flag.NewFlagSet("statham", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	scale := fs.Float64("scale", 1.0, "head-image size multiplier")
	dx := fs.Int("dx", 0, "nudge the image right (px, at 128 tile)")
	dy := fs.Int("dy", 0, "nudge the image down (px, at 128 tile)")
	preview := fs.Bool("preview", false, "outline the detected head box instead of an image (no input needed)")
	name := fs.String("name", "", "emoji name / output basename (default: statham_<file>)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "statham [flags] <input>",
		"Trace your image over Jason Statham's head as he dances.")

	// Preview mode needs no input image, so the positional is optional; branch on
	// the parsed flag (not a raw arg scan, which misses -preview=true and flag
	// ordering) to decide whether an input is required.
	input, err := parseInputOpt(fs, args)
	if err != nil {
		return err
	}
	if !*preview && input == "" {
		fs.Usage()
		return fmt.Errorf("need exactly one input file (or use -preview)")
	}
	if *tile < 1 || *tile > maxEmojiPx {
		return fmt.Errorf("tile must be between 1 and %d", maxEmojiPx)
	}
	if *scale <= 0 {
		return fmt.Errorf("scale must be > 0, got %v", *scale)
	}

	base, delays, err := gifFrames(stathamGIF)
	if err != nil {
		return fmt.Errorf("decoding embedded statham asset: %w", err)
	}

	var head *image.RGBA // nil in preview mode
	if !*preview {
		src, err := loadNative(input, *tile)
		if err != nil {
			return err
		}
		head = fitSquare(src, *tile) // resized per-frame below
	}

	s := float64(*tile) / float64(base[0].Bounds().Dx()) // asset px -> tile px
	fr := make([]*image.RGBA, len(base))
	for i, bg := range base {
		f := scaleTo(bg, *tile, *tile)
		hx, hy, hw := detectHead(bg) // in the asset's native pixels
		cx := int(float64(hx+*dx) * s)
		cy := int(float64(hy+*dy) * s)
		w := int(float64(2*hw) * s * *scale)
		if w < 1 {
			w = 1
		}
		if *preview {
			drawBox(f, cx, cy, w)
		} else {
			h := scaleTo(head, w, w)
			draw.Draw(f, image.Rect(cx-w/2, cy-w/2, cx-w/2+w, cy-w/2+w), h, h.Bounds().Min, draw.Over)
		}
		fr[i] = f
	}

	outName := *name
	if outName == "" {
		if *preview {
			outName = "statham_preview"
		} else {
			outName = "statham_" + sanitizeName(input)
		}
	}
	gifPath := filepath.Join(*out, outName+".gif")
	if err := encodeGIFDelays(gifPath, fr, delays); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, outName)
	return nil
}

// detectHead finds the head's center (cx, cy) and half-width in a composited
// frame. The head is the densest headBand-tall band of figure pixels below the
// crown: yTop is the first row carrying figure, and a sliding column window over
// [yTop, yTop+band) locks onto the dense skull, ignoring the thin flailing arms
// that a plain centroid would chase. Falls back to the frame center if empty.
func detectHead(fr *image.RGBA) (cx, cy, halfW int) {
	b := fr.Bounds()
	w, h := b.Dx(), b.Dy()
	halfW = w * 20 / 128 // ~40px head at 128
	fg := func(x, y int) bool {
		c := fr.RGBAAt(b.Min.X+x, b.Min.Y+y)
		return c.A >= 128 && (c.R >= 24 || c.G >= 24 || c.B >= 24) // opaque, not background-black
	}
	yTop := -1
	for y := 0; y < h; y++ {
		n := 0
		for x := 0; x < w; x++ {
			if fg(x, y) {
				n++
			}
		}
		if n >= 3 { // ignore lone stray pixels
			yTop = y
			break
		}
	}
	if yTop < 0 {
		return w / 2, h / 2, halfW // blank frame
	}
	band := headBand * h / 128
	col := make([]int, w)
	for y := yTop; y < yTop+band && y < h; y++ {
		for x := 0; x < w; x++ {
			if fg(x, y) {
				col[x]++
			}
		}
	}
	best, bestX := -1, halfW
	for c := halfW; c < w-halfW; c++ {
		sum := 0
		for x := c - halfW; x < c+halfW; x++ {
			sum += col[x]
		}
		if sum > best {
			best, bestX = sum, c
		}
	}
	return bestX, yTop + band/2, halfW
}

// drawBox outlines the head box and marks its center, for verifying detection.
func drawBox(dst *image.RGBA, cx, cy, w int) {
	c := color.RGBA{255, 0, 255, 255}
	r := image.Rect(cx-w/2, cy-w/2, cx+w/2, cy+w/2)
	for x := r.Min.X; x < r.Max.X; x++ {
		dst.Set(x, r.Min.Y, c)
		dst.Set(x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		dst.Set(r.Min.X, y, c)
		dst.Set(r.Max.X-1, y, c)
	}
	for d := -5; d <= 5; d++ {
		dst.Set(cx+d, cy, c)
		dst.Set(cx, cy+d, c)
	}
}
