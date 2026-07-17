package main

import (
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"path/filepath"
)

//go:embed assets/confetti.gif
var confettiGIF []byte

func runConfetti(args []string) error {
	fs := flag.NewFlagSet("confetti", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	maxFrames := fs.Int("frames", 50, "max frames (the overlay is subsampled to fit; Slack caps at 50)")
	name := fs.String("name", "", "emoji name / output basename (default: <file>_confetti)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "confetti [flags] <input>",
		"Rain confetti over your image (animated overlay).")
	input, err := parseInput(fs, args)
	if err != nil {
		return err
	}
	if *tile < 1 || *tile > maxEmojiPx {
		return fmt.Errorf("tile must be between 1 and %d", maxEmojiPx)
	}
	if *maxFrames < 1 {
		return fmt.Errorf("frames must be >= 1, got %d", *maxFrames)
	}

	src, err := loadNative(input, *tile)
	if err != nil {
		return err
	}
	base := fitSquare(src, *tile)

	overlay, delays, err := gifFrames(confettiGIF)
	if err != nil {
		return fmt.Errorf("decoding embedded confetti asset: %w", err)
	}

	fr := make([]*image.RGBA, len(overlay))
	for i, cf := range overlay {
		f := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
		draw.Draw(f, f.Bounds(), base, image.Point{}, draw.Src)
		confetti := scaleTo(cf, *tile, *tile)
		draw.Draw(f, f.Bounds(), confetti, confetti.Bounds().Min, draw.Over)
		fr[i] = f
	}
	fr, delays = subsampleFrames(fr, delays, *maxFrames)

	outName := orNameSuffix(*name, input, "_confetti")
	gifPath := filepath.Join(*out, outName+".gif")
	if err := encodeGIFDelays(gifPath, fr, delays); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, outName)
	return nil
}
