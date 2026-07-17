package emoji

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"math/rand/v2"
	"path/filepath"
)

func runPanic(args []string) error {
	fs := flag.NewFlagSet("panic", flag.ContinueOnError)
	tile := fs.Int("tile", 112, "emoji size in px (square)")
	frames := fs.Int("frames", 10, "number of frames")
	dur := fs.Int("dur", 40, "milliseconds per frame (lower = faster)")
	minScale := fs.Float64("min", 0.45, "starting scale of the pulse (0-1)")
	jitter := fs.Float64("jitter", 0.12, "shake amplitude as a fraction of tile")
	name := fs.String("name", "", "emoji name / output basename (default: <file>_panic)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "panic [flags] <input>",
		"Make a frantic zoom-pulse + shake GIF (the image grows and vibrates, then loops).")
	input, err := parseInput(fs, args)
	if err != nil {
		return err
	}
	if *tile < 1 || *frames < 1 {
		return fmt.Errorf("tile and frames must be >= 1")
	}
	if *tile > maxEmojiPx {
		return fmt.Errorf("tile must be <= %d (Slack emoji limit)", maxEmojiPx)
	}
	if *minScale <= 0 || *minScale > 1 {
		return fmt.Errorf("min must be in (0, 1], got %v", *minScale)
	}

	base, err := loadImage(input, *tile, *tile)
	if err != nil {
		return err
	}

	span := 1.0 - *minScale
	fr := make([]*image.RGBA, *frames)
	for i := 0; i < *frames; i++ {
		s := *minScale
		if *frames > 1 {
			s += span * float64(i) / float64(*frames-1) // grow min -> full, then loop
		} else {
			s = 1.0
		}
		sz := int(float64(*tile) * s)
		if sz < 1 {
			sz = 1
		}
		subject := scaleTo(base, sz, sz)
		jx := int(float64(*tile) * *jitter * (rand.Float64()*2 - 1))
		jy := int(float64(*tile) * *jitter * (rand.Float64()*2 - 1))
		x := (*tile-sz)/2 + jx
		y := (*tile-sz)/2 + jy
		f := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
		// draw.Draw clips the rect to the canvas, so edge overflow is fine.
		draw.Draw(f, image.Rect(x, y, x+sz, y+sz), subject, subject.Bounds().Min, draw.Over)
		fr[i] = f
	}

	base2 := *name
	if base2 == "" {
		base2 = sanitizeName(input) + "_panic"
	}
	gifPath := filepath.Join(*out, base2+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, base2)
	return nil
}
