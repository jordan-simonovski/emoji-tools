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

const (
	// catRotateDeg levels the asset, which is drawn leaning to the right.
	catRotateDeg = 12.0
	// armpitFrac is where the image's top edge sits, as a fraction of the cat's
	// height: the cat's arms rest here, so the image tucks under them instead of
	// rising past the armpits. Drawn behind the cat, the paws overlap its top edge.
	armpitFrac = 0.5
)

func runBongocat(args []string) error {
	fs := flag.NewFlagSet("bongocat", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	dur := fs.Int("dur", 130, "milliseconds per frame (lower = faster drumming)")
	scale := fs.Float64("scale", 0.9, "image width as a fraction of the tile")
	name := fs.String("name", "", "emoji name / output basename (default: bongo_<file>)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "bongocat [flags] <input>",
		"Bongo cat drumming on your image (arms rest on the image's top edge).")
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

	cats, _, err := gifFrames(bongocatGIF)
	if err != nil {
		return fmt.Errorf("decoding embedded bongocat asset: %w", err)
	}
	// Cat spans the tile width at the top; rotate it upright first.
	catW := *tile
	catH := int(math.Round(float64(catW) * float64(cats[0].Bounds().Dy()) / float64(cats[0].Bounds().Dx())))

	// Sized by width (a square), so a wide image runs off the bottom and the canvas
	// clips it. See armpitFrac for why the top sits where it does.
	armpitY := int(armpitFrac * float64(catH))
	imgPx := int(float64(*tile) * *scale)
	img := fitSquare(src, imgPx)
	imgX := (*tile - imgPx) / 2
	imgY := armpitY // top at the armpit line; bottom may overflow and get cropped

	fr := make([]*image.RGBA, len(cats))
	for i, cat := range cats {
		f := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
		draw.Draw(f, image.Rect(imgX, imgY, imgX+imgPx, imgY+imgPx), img, img.Bounds().Min, draw.Over)
		scaledCat := scaleTo(rotateRGBA(cat, catRotateDeg), catW, catH)
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
