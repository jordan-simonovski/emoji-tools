package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"math/rand/v2"
	"path/filepath"
)

func runIntensify(args []string) error {
	fs := flag.NewFlagSet("intensify", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	frames := fs.Int("frames", 10, "number of frames")
	dur := fs.Int("dur", 33, "milliseconds per frame (~30fps default)")
	shake := fs.Int("shake", 6, "max shake amplitude in px")
	name := fs.String("name", "", "emoji name / output basename (default: <file>_intensifies)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "intensify [flags] <input>",
		`Make a shaking ":x-intensifies:" GIF.`)
	input, err := parseInput(fs, args)
	if err != nil {
		return err
	}
	if *tile < 1 || *tile > maxEmojiPx {
		return fmt.Errorf("tile must be between 1 and %d", maxEmojiPx)
	}
	if *frames <= 0 {
		return fmt.Errorf("frames must be > 0, got %d", *frames)
	}
	if *shake < 0 || 2**shake >= *tile {
		return fmt.Errorf("shake (%d) must be >= 0 and less than tile/2 (%d)", *shake, *tile/2)
	}

	// Content is inset by `shake` on every side so jitter never clips it.
	content := *tile - 2**shake
	logo, err := loadImage(input, content, content)
	if err != nil {
		return err
	}

	fr := make([]*image.RGBA, *frames)
	for i := 0; i < *frames; i++ {
		dx, dy := 0, 0
		if i > 0 { // first frame is centered/rest
			dx = rand.IntN(2**shake+1) - *shake
			dy = rand.IntN(2**shake+1) - *shake
		}
		f := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
		at := image.Rect(*shake+dx, *shake+dy, *shake+dx+content, *shake+dy+content)
		draw.Draw(f, at, logo, logo.Bounds().Min, draw.Over)
		fr[i] = f
	}

	base := *name
	if base == "" {
		base = sanitizeName(input) + "_intensifies"
	}
	gifPath := filepath.Join(*out, base+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, base)
	return nil
}
