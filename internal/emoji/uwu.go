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

//go:embed assets/uwu-face.png
var uwuPNG []byte

func runUwu(args []string) error {
	fs := flag.NewFlagSet("uwu", flag.ContinueOnError)
	scale := fs.Float64("scale", 0.85, "face width as a fraction of the image width")
	yShift := fs.Float64("y", 0.0, "vertical nudge as a fraction of image height (+down, -up)")
	size := fs.Int("size", 128, "output emoji size in px (square)")
	svgW := fs.Int("svg-width", 256, "render width for SVG inputs")
	out := fs.String("out", "", "output file (default: uwu-<input>.png)")
	fs.Usage = usageFor(fs, "uwu [flags] <input>",
		"Overlay a uwu face onto your image.")
	input, err := parseInput(fs, args)
	if err != nil {
		return err
	}
	if *scale <= 0 || *scale > 2 {
		return fmt.Errorf("scale must be in (0, 2], got %v", *scale)
	}
	if *size < 1 || *size > maxEmojiPx {
		return fmt.Errorf("size must be between 1 and %d", maxEmojiPx)
	}

	target, err := loadNative(input, *svgW)
	if err != nil {
		return err
	}
	face, err := png.Decode(bytes.NewReader(uwuPNG))
	if err != nil {
		return fmt.Errorf("decoding embedded uwu asset: %w", err)
	}

	b := target.Bounds()
	W, H := b.Dx(), b.Dy()
	faceW := int(float64(W) * *scale)
	scaledFace := scaleToWidth(face, faceW)
	fb := scaledFace.Bounds()

	// center horizontally and vertically, then apply the y nudge
	x := b.Min.X + (W-fb.Dx())/2
	y := b.Min.Y + (H-fb.Dy())/2 + int(float64(H)**yShift)

	canvas := image.NewRGBA(b)
	draw.Draw(canvas, b, target, b.Min, draw.Src)
	draw.Draw(canvas, image.Rect(x, y, x+fb.Dx(), y+fb.Dy()), scaledFace, fb.Min, draw.Over)

	// Slack emoji must be square and within the size ceiling.
	out2 := fitSquare(canvas, *size)

	outPath := *out
	if outPath == "" {
		outPath = "uwu-" + sanitizeName(input) + ".png"
	}
	if err := writePNG(out2, outPath); err != nil {
		return err
	}
	fmt.Printf("Wrote %s (%dx%d)\n", outPath, out2.Bounds().Dx(), out2.Bounds().Dy())
	return nil
}
