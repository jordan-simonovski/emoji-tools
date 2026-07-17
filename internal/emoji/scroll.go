package emoji

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"path/filepath"
	"strings"
)

func runScroll(args []string) error {
	fs := flag.NewFlagSet("scroll", flag.ContinueOnError)
	tile := fs.Int("tile", 112, "emoji size in px (square)")
	frames := fs.Int("frames", 8, "number of frames (must divide tile evenly)")
	dur := fs.Int("dur", 40, "milliseconds per frame (lower = faster)")
	logo := fs.Int("logo", 0, "logo size within the tile (default 0.72*tile; smaller = more gap)")
	dirn := fs.String("direction", "left", "scroll direction: left or right")
	name := fs.String("name", "", "emoji name / output basename (default: from filename)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "scroll [flags] <input>",
		"Make a seamless horizontal-scroll GIF; repeat the emoji in a row for a moving stream.")
	input, err := parseInput(fs, args)
	if err != nil {
		return err
	}
	if *tile < 1 || *tile > maxEmojiPx {
		return fmt.Errorf("tile must be between 1 and %d", maxEmojiPx)
	}
	if *frames <= 0 || *tile%*frames != 0 {
		return fmt.Errorf("frames (%d) must divide tile (%d) evenly for a seamless loop", *frames, *tile)
	}
	if *dirn != "left" && *dirn != "right" {
		return fmt.Errorf("direction must be left or right, got %q", *dirn)
	}

	logoSize := *logo
	if logoSize == 0 {
		logoSize = *tile * 72 / 100
	}
	src, err := loadImage(input, logoSize, logoSize)
	if err != nil {
		return err
	}

	// One tile-wide period with the logo centered, then doubled so a moving
	// window of tile width wraps seamlessly.
	off := (*tile - logoSize) / 2
	period := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
	draw.Draw(period, image.Rect(off, off, off+logoSize, off+logoSize), src, src.Bounds().Min, draw.Over)
	doubled := image.NewRGBA(image.Rect(0, 0, 2**tile, *tile))
	draw.Draw(doubled, image.Rect(0, 0, *tile, *tile), period, image.Point{}, draw.Src)
	draw.Draw(doubled, image.Rect(*tile, 0, 2**tile, *tile), period, image.Point{}, draw.Src)

	shift := *tile / *frames
	fr := make([]*image.RGBA, *frames)
	for i := 0; i < *frames; i++ {
		f := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
		draw.Draw(f, f.Bounds(), doubled, image.Pt(i*shift, 0), draw.Src)
		fr[i] = f
	}
	if *dirn == "right" {
		reverse(fr)
	}

	base := orName(*name, input)
	gifPath := filepath.Join(*out, base+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nPaste a row into Slack for a moving stream:\n\n%s\n",
		gifPath, strings.Repeat(fmt.Sprintf(":%s:", base), 6))
	return nil
}

func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
