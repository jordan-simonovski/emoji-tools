package emoji

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"math"
	"path/filepath"
)

// runSpin fakes a 3D spin around the vertical axis (a spinning coin): each frame
// squashes the image horizontally by |cos(angle)| so it thins to an edge and
// back, mirroring on the far half so you appear to see its reverse side.
func runSpin(args []string) error {
	fs := flag.NewFlagSet("spin", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	frames := fs.Int("frames", 12, "number of frames in one full turn")
	dur := fs.Int("dur", 60, "milliseconds per frame (lower = faster)")
	name := fs.String("name", "", "emoji name / output basename (default: <file>_spin)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "spin [flags] <input>",
		"Make a spinning-coin GIF (3D flip around the vertical axis).")
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

	src, err := loadNative(input, *tile)
	if err != nil {
		return err
	}
	subject := fitSquare(src, *tile)
	mirror := flipH(subject)

	fr := make([]*image.RGBA, *frames)
	for i := 0; i < *frames; i++ {
		angle := 2 * math.Pi * float64(i) / float64(*frames)
		face := subject
		if math.Cos(angle) < 0 {
			face = mirror // past 90°, we're looking at the back
		}
		w := int(math.Round(float64(*tile) * math.Abs(math.Cos(angle))))
		f := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
		if w >= 1 {
			squashed := scaleTo(face, w, *tile)
			x := (*tile - w) / 2
			draw.Draw(f, image.Rect(x, 0, x+w, *tile), squashed, squashed.Bounds().Min, draw.Over)
		}
		fr[i] = f
	}

	base := orNameSuffix(*name, input, "_spin")
	gifPath := filepath.Join(*out, base+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, base)
	return nil
}
