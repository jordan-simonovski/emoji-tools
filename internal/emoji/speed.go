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

// runSpeed draws flickering anime speed lines streaking in from the right over
// a still image.
func runSpeed(args []string) error {
	fs := flag.NewFlagSet("speed", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	frames := fs.Int("frames", 10, "number of frames")
	dur := fs.Int("dur", 40, "milliseconds per frame (lower = faster)")
	lines := fs.Int("lines", 12, "speed lines per frame")
	name := fs.String("name", "", "emoji name / output basename (default: <file>_speed)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "speed [flags] <input>",
		"Anime speed lines streaking past your image.")
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
	if *lines < 0 {
		return fmt.Errorf("lines must be >= 0, got %d", *lines)
	}

	src, err := loadNative(input, *tile)
	if err != nil {
		return err
	}
	base := fitSquare(src, *tile)

	// Fixed seed so the same input always gives the same GIF.
	rng := rand.New(rand.NewPCG(1, 2))
	shades := []color.RGBA{{0, 0, 0, 255}, {0, 0, 0, 255}, {70, 70, 70, 255}}
	// A light edge under each dark line keeps it visible on Slack's dark theme too.
	edge := color.RGBA{210, 210, 210, 255}
	t := float64(*tile)
	// Lines radiate from a point left of centre, so they fan out towards the right edge.
	fx, fy := -0.3*t, t/2
	fr := make([]*image.RGBA, *frames)
	for i := range fr {
		f := cloneRGBA(base)
		for range *lines {
			ang := (rng.Float64()*2 - 1) * 0.6 // radians either side of horizontal
			inner := t * (0.7 + 0.6*rng.Float64())
			half := t / 128 * (2 + 6*rng.Float64())
			shade := shades[rng.IntN(len(shades))]
			cos, sin := math.Cos(ang), math.Sin(ang)
			wedge := func(c color.Color, inner, half float64) {
				outer := 2 * t
				fillPoly(f, c,
					float32(fx+inner*cos), float32(fy+inner*sin),
					float32(fx+outer*cos-half*sin), float32(fy+outer*sin+half*cos),
					float32(fx+outer*cos+half*sin), float32(fy+outer*sin-half*cos))
			}
			wedge(edge, inner-t/32, half+t/64+1)
			wedge(shade, inner, half)
		}
		fr[i] = f
	}

	outName := orNameSuffix(*name, input, "_speed")
	gifPath := filepath.Join(*out, outName+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, outName)
	return nil
}
