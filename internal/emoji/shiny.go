package emoji

import (
	"flag"
	"fmt"
	"image"
	"math"
	"path/filepath"
)

// runShiny sweeps a metallic glint diagonally across a still image, like light
// catching a tilting coin.
func runShiny(args []string) error {
	fs := flag.NewFlagSet("shiny", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	frames := fs.Int("frames", 16, "number of frames (one sweep plus a pause)")
	dur := fs.Int("dur", 60, "milliseconds per frame (lower = faster)")
	name := fs.String("name", "", "emoji name / output basename (default: <file>_shiny)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "shiny [flags] <input>",
		"A metallic glint sweeping across your image.")
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
	base := fitSquare(src, *tile)

	t := float64(*tile)
	fr := make([]*image.RGBA, *frames)
	for i := range fr {
		f := cloneRGBA(base)
		// The glint crosses in the first 70% of the loop, then the image rests so each
		// flash reads as a separate catch of the light.
		p := -0.35 + 1.7*float64(i)/(0.7*float64(*frames))
		for y := 0; y < *tile; y++ {
			for x := 0; x < *tile; x++ {
				d := (float64(x) + float64(y)) / (2 * t) // 0 top-left, 1 bottom-right
				k := 0.9*glint(d-p, 0.09) + 0.6*glint(d-p+0.2, 0.03)
				if k < 0.01 {
					continue
				}
				// Pixels are premultiplied, so pushing each channel towards alpha lights only
				// the opaque parts of the image and leaves transparent areas alone.
				o := f.PixOffset(x, y)
				a := float64(f.Pix[o+3])
				for c := range 3 {
					v := float64(f.Pix[o+c])
					f.Pix[o+c] = uint8(v + (a-v)*math.Min(k, 1))
				}
			}
		}
		fr[i] = f
	}

	outName := orNameSuffix(*name, input, "_shiny")
	gifPath := filepath.Join(*out, outName+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, outName)
	return nil
}

// glint is a soft band of light, 1 at its centre and fading over width w.
func glint(off, w float64) float64 { return math.Exp(-(off * off) / (w * w)) }
