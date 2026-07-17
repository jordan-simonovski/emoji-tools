package emoji

import (
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"math"
	"path/filepath"
)

//go:embed assets/bongocat.gif
var bongocatGIF []byte

func runBongocat(args []string) error {
	fs := flag.NewFlagSet("bongocat", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	dur := fs.Int("dur", 130, "milliseconds per frame (lower = faster drumming)")
	scale := fs.Float64("scale", 0.62, "drum (your image) size as a fraction of the tile")
	name := fs.String("name", "", "emoji name / output basename (default: bongo_<file>)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "bongocat [flags] <input>",
		"Bongo cat drumming on your image (the cat overlays and cuts off the top of your image).")
	input, err := parseInput(fs, args)
	if err != nil {
		return err
	}
	if *tile < 1 || *tile > maxEmojiPx {
		return fmt.Errorf("tile must be between 1 and %d", maxEmojiPx)
	}
	if *scale <= 0 || *scale > 1 {
		return fmt.Errorf("scale must be in (0, 1], got %v", *scale)
	}

	src, err := loadNative(input, *tile)
	if err != nil {
		return err
	}
	// The drum sits low and centered; the cat is drawn on top of it so its body
	// occludes the drum's top edge, reading as "cat in front".
	drumPx := int(float64(*tile) * *scale)
	drum := fitSquare(src, drumPx)
	drumX := (*tile - drumPx) / 2
	drumY := *tile - drumPx // bottom-aligned

	cats, _, err := gifFrames(bongocatGIF)
	if err != nil {
		return fmt.Errorf("decoding embedded bongocat asset: %w", err)
	}
	// Cat spans the full width at the top; its paws reach down onto the drum.
	catW := *tile
	catH := int(math.Round(float64(*tile) * float64(cats[0].Bounds().Dy()) / float64(cats[0].Bounds().Dx())))

	fr := make([]*image.RGBA, len(cats))
	for i, cat := range cats {
		f := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
		draw.Draw(f, image.Rect(drumX, drumY, drumX+drumPx, drumY+drumPx), drum, drum.Bounds().Min, draw.Over)
		scaledCat := scaleTo(cat, catW, catH)
		draw.Draw(f, image.Rect(0, 0, catW, catH), scaledCat, scaledCat.Bounds().Min, draw.Over)
		fr[i] = f
	}

	outName := *name
	if outName == "" {
		outName = "bongo_" + sanitizeName(input)
	}
	gifPath := filepath.Join(*out, outName+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, outName)
	return nil
}
