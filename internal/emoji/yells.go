package emoji

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
)

//go:embed assets/old_man_yells_at.png
var oldManPNG []byte

func runYells(args []string) error {
	fs := flag.NewFlagSet("yells-at", flag.ContinueOnError)
	width := fs.Int("width", 95, "width the target is scaled to, in px")
	size := fs.Int("size", 128, "output emoji size in px (square)")
	out := fs.String("out", "", "output file (default: old-man-yells-at-<input>.png)")
	fs.Usage = usageFor(fs, "yells-at [flags] <input>",
		"Composite Abe Simpson yelling at your image (art from oncilla/old-man-yells-at).")
	input, err := parseInput(fs, args)
	if err != nil {
		return err
	}
	if *size < 1 || *size > maxEmojiPx {
		return fmt.Errorf("size must be between 1 and %d", maxEmojiPx)
	}

	abe, err := png.Decode(bytes.NewReader(oldManPNG))
	if err != nil {
		return fmt.Errorf("decoding embedded old-man asset: %w", err)
	}
	target, err := loadNative(input, 256)
	if err != nil {
		return err
	}
	scaled := scaleToWidth(target, *width)

	canvas := image.NewRGBA(abe.Bounds())
	draw.Draw(canvas, canvas.Bounds(), abe, abe.Bounds().Min, draw.Src)
	draw.Draw(canvas, scaled.Bounds(), scaled, scaled.Bounds().Min, draw.Over)

	// Slack emoji must be square and within the size ceiling.
	out2 := fitSquare(canvas, *size)

	outPath := *out
	if outPath == "" {
		outPath = "old-man-yells-at-" + sanitizeName(input) + ".png"
	}
	if err := writePNG(out2, outPath); err != nil {
		return err
	}
	fmt.Printf("Wrote %s (%dx%d)\n", outPath, out2.Bounds().Dx(), out2.Bounds().Dy())
	return nil
}
