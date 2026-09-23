package emoji

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"path/filepath"
)

// runSparkle scatters four-point sparkles over a still image that twinkle in
// and out on staggered cycles.
func runSparkle(args []string) error {
	fs := flag.NewFlagSet("sparkle", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	frames := fs.Int("frames", 12, "number of frames (one full twinkle cycle)")
	dur := fs.Int("dur", 70, "milliseconds per frame (lower = faster)")
	count := fs.Int("count", 7, "number of sparkles")
	col := fs.String("color", "#ffd84d", "sparkle colour as hex")
	name := fs.String("name", "", "emoji name / output basename (default: <file>_sparkle)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "sparkle [flags] <input>",
		"Twinkling sparkles over your image.")
	input, err := parseInput(fs, args)
	if err != nil {
		return err
	}
	if *tile < 1 || *tile > maxEmojiPx {
		return fmt.Errorf("tile must be between 1 and %d", maxEmojiPx)
	}
	if *frames < 1 {
		return fmt.Errorf("frames must be >= 1, got %d", *frames)
	}
	if *count < 0 {
		return fmt.Errorf("count must be >= 0, got %d", *count)
	}
	c, err := parseHexColor(*col)
	if err != nil {
		return err
	}

	src, err := loadNative(input, *tile)
	if err != nil {
		return err
	}
	base := fitSquare(src, *tile)

	type spark struct{ x, y, size, phase float64 }
	rng := rand.New(rand.NewPCG(3, 4)) // fixed seed: same input, same GIF
	t := float64(*tile)
	sparks := make([]spark, *count)
	for i := range sparks {
		sparks[i] = spark{
			x: t * (0.1 + 0.8*rng.Float64()), y: t * (0.1 + 0.8*rng.Float64()),
			size: t * (0.07 + 0.08*rng.Float64()), phase: rng.Float64(),
		}
	}
	white := color.RGBA{255, 255, 255, 255}
	// A darker rim keeps light sparkle colours visible on Slack's light theme.
	rim := color.RGBA{c.R / 2, c.G / 2, c.B / 2, 255}
	rimW := max(1, t/64)
	fr := make([]*image.RGBA, *frames)
	for i := range fr {
		f := cloneRGBA(base)
		for _, s := range sparks {
			// Visible for half the cycle, so sparkles pop in and out rather than all pulsing.
			k := math.Sin(2 * math.Pi * (float64(i)/float64(*frames) + s.phase))
			if k <= 0 {
				continue
			}
			drawStar(f, rim, s.x, s.y, s.size*k+rimW)
			drawStar(f, c, s.x, s.y, s.size*k)
			drawStar(f, white, s.x, s.y, s.size*k*0.45)
		}
		fr[i] = f
	}

	outName := orNameSuffix(*name, input, "_sparkle")
	gifPath := filepath.Join(*out, outName+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, outName)
	return nil
}

// drawStar paints a four-point sparkle of radius r centred on (x, y).
func drawStar(dst *image.RGBA, c color.Color, x, y, r float64) {
	w := r * 0.22 // waist: how pinched the star is between points
	fillPoly(dst, c,
		float32(x), float32(y-r), float32(x+w), float32(y-w),
		float32(x+r), float32(y), float32(x+w), float32(y+w),
		float32(x), float32(y+r), float32(x-w), float32(y+w),
		float32(x-r), float32(y), float32(x-w), float32(y-w))
}
