package emoji

import (
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"math/rand/v2"
	"path/filepath"
)

//go:embed assets/narutorun.gif
var narutoGIF []byte

// narutoHeads is the head centre per frame, in the asset's native pixels.
// ponytail: hand-measured; re-measure with -preview if the asset is replaced.
var narutoHeads = [][2]int{{114, 16}, {111, 19}, {108, 21}, {110, 19}, {114, 17}, {114, 13}, {109, 20}}

const (
	narutoNativeW, narutoNativeH = 128, 87
	narutoHeadPx                 = 26 // head width at native size
)

func runNaruto(args []string) error {
	fs := flag.NewFlagSet("naruto", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	dur := fs.Int("dur", 50, "milliseconds per frame, minimum 20 (the source runs at 100)")
	lines := fs.Int("lines", 12, "speed lines per frame (0 for none)")
	scale := fs.Float64("scale", 1.0, "head-image size multiplier")
	dx := fs.Int("dx", 0, "nudge the image right (px, at 128 tile)")
	dy := fs.Int("dy", 0, "nudge the image down (px, at 128 tile)")
	preview := fs.Bool("preview", false, "outline the head box instead of an image (no input needed)")
	name := fs.String("name", "", "emoji name / output basename (default: naruto_<file>)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "naruto [flags] <input>",
		"Your image on Naruto's head as he sprints, with anime speed lines.")

	input, err := parseInputOpt(fs, args)
	if err != nil {
		return err
	}
	if !*preview && input == "" {
		fs.Usage()
		return fmt.Errorf("need exactly one input file (or use -preview)")
	}
	if *tile < 1 || *tile > maxEmojiPx {
		return fmt.Errorf("tile must be between 1 and %d", maxEmojiPx)
	}
	if *scale <= 0 {
		return fmt.Errorf("scale must be > 0, got %v", *scale)
	}
	// Browsers and Slack play delays under 2cs at ~100ms, slower than the default.
	if *dur < 20 {
		return fmt.Errorf("dur must be >= 20, got %d", *dur)
	}
	if *lines < 0 {
		return fmt.Errorf("lines must be >= 0, got %d", *lines)
	}

	base, _, err := gifFrames(narutoGIF)
	if err != nil {
		return fmt.Errorf("decoding embedded naruto asset: %w", err)
	}
	if len(base) != len(narutoHeads) {
		return fmt.Errorf("naruto asset has %d frames but %d head positions are recorded", len(base), len(narutoHeads))
	}
	if b := base[0].Bounds(); b.Dx() != narutoNativeW || b.Dy() != narutoNativeH {
		return fmt.Errorf("naruto asset is %dx%d but head positions are recorded in %dx%d coordinates", b.Dx(), b.Dy(), narutoNativeW, narutoNativeH)
	}

	s := float64(*tile) / narutoNativeW // asset px -> tile px
	w := max(int(narutoHeadPx*s**scale), 1)
	var head *image.RGBA // nil in preview mode
	if !*preview {
		src, err := loadNative(input, *tile)
		if err != nil {
			return err
		}
		head = fitSquare(src, w)
	}

	// The sprite is wider than tall, so centre it vertically in the square tile.
	top := (*tile - int(narutoNativeH*s)) / 2
	rng := rand.New(rand.NewPCG(1, 2)) // fixed seed: same input, same GIF
	fr := make([]*image.RGBA, len(base))
	for i, src := range base {
		f := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
		// He runs right, so the lines trail off to his left, behind the sprite.
		drawSpeedLines(f, rng, "left", *lines)
		body := scaleTo(src, *tile, int(narutoNativeH*s))
		draw.Draw(f, body.Bounds().Add(image.Pt(0, top)), body, image.Point{}, draw.Over)
		cx := int(float64(narutoHeads[i][0]+*dx) * s)
		cy := int(float64(narutoHeads[i][1]+*dy)*s) + top
		if *preview {
			drawBox(f, cx, cy, w)
		} else {
			draw.Draw(f, image.Rect(cx-w/2, cy-w/2, cx-w/2+w, cy-w/2+w), head, image.Point{}, draw.Over)
		}
		fr[i] = f
	}

	outName := *name
	if outName == "" {
		if *preview {
			outName = "naruto_preview"
		} else {
			outName = "naruto_" + sanitizeName(input)
		}
	}
	gifPath := filepath.Join(*out, outName+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, outName)
	return nil
}
