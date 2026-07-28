package emoji

import (
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"path/filepath"
)

//go:embed assets/fistpump.gif
var fistpumpGIF []byte

// fistCenters is where the fist sits in each frame of the asset, in its native
// 128px coordinates: frame 0 is the arm thrown out, frame 1 is it pulled back.
// The swing between the two is what shakes the held image.
// ponytail: hand-measured for this two-frame sprite; a detector like statham's
// detectHead only earns its keep on an asset with frames to spare. Re-measure
// (with -preview) if assets/fistpump.gif is ever replaced.
var fistCenters = [][2]int{{77, 20}, {51, 26}}

const (
	// fistNativePx is the asset's frame size, the coordinate space fistCenters and
	// the -dx/-dy nudges are expressed in.
	fistNativePx = 128
	// armDropPx pushes the arm down the tile. The sprite reaches within 7px of the
	// top edge but leaves its bottom ~45px empty, so dropping it costs nothing and
	// buys the headroom the held image needs above the fist.
	armDropPx = 40
)

func runFistpump(args []string) error {
	fs := flag.NewFlagSet("fistpump", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	scale := fs.Float64("scale", 0.45, "held-image width as a fraction of the tile (past ~0.6 the tile's top edge crops it)")
	side := fs.String("side", "left", `which side of the neighbouring emoji this sits on: "left" or "right" (mirrors the arm)`)
	dx := fs.Int("dx", 0, "nudge the held image right (px, at 128 tile)")
	dy := fs.Int("dy", 0, "nudge the held image down (px, at 128 tile)")
	preview := fs.Bool("preview", false, "outline the fist box instead of an image (no input needed)")
	name := fs.String("name", "", "emoji name / output basename (default: fistpump_<file>_<side>)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "fistpump [flags] <input>",
		"A shaking fist pumping your image in the air.")

	// Preview needs no input image, so the positional is optional. Same shape as
	// statham: branch on the parsed flag, not a raw arg scan.
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
	if *scale <= 0 || *scale > 1 {
		return fmt.Errorf("scale must be in (0, 1], got %v", *scale)
	}
	if *side != "left" && *side != "right" {
		return fmt.Errorf(`side must be "left" or "right", got %q`, *side)
	}

	base, delays, err := gifFrames(fistpumpGIF)
	if err != nil {
		return fmt.Errorf("decoding embedded fistpump asset: %w", err)
	}
	// fistCenters are measured in the asset's own pixels, so a swapped asset that
	// changed size or frame count would silently misplace the image.
	if len(base) != len(fistCenters) {
		return fmt.Errorf("fistpump asset has %d frames but %d fist positions are recorded", len(base), len(fistCenters))
	}
	if b := base[0].Bounds(); b.Dx() != fistNativePx || b.Dy() != fistNativePx {
		return fmt.Errorf("fistpump asset is %dx%d but fist positions are recorded in %dpx coordinates", b.Dx(), b.Dy(), fistNativePx)
	}

	var held *image.RGBA // nil in preview mode
	w := max(int(float64(*tile)**scale), 1)
	if !*preview {
		src, err := loadNative(input, *tile)
		if err != nil {
			return err
		}
		held = fitSquare(src, w)
	}

	s := float64(*tile) / fistNativePx // asset px -> tile px
	drop := int(armDropPx * s)
	fr := make([]*image.RGBA, len(base))
	for i, src := range base {
		arm := scaleTo(src, *tile, *tile)
		fx := fistCenters[i][0]
		if *side == "right" {
			arm = flipH(arm)
			fx = fistNativePx - 1 - fx // mirror the arm, not the image it holds
		}
		// Nudges apply after the mirror, so -dx always means "right on screen".
		// The held image rides a quarter of its height above the fist, so the fist
		// overlaps its lower edge at any -scale instead of covering the middle.
		cx := int(float64(fx+*dx) * s)
		cy := int(float64(fistCenters[i][1]+*dy)*s) + drop - w/4

		// Held image goes behind the arm so the fist visibly grips it, the same
		// front/back trick bongocat uses for its paws.
		f := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
		if !*preview {
			draw.Draw(f, image.Rect(cx-w/2, cy-w/2, cx-w/2+w, cy-w/2+w), held, held.Bounds().Min, draw.Over)
		}
		draw.Draw(f, image.Rect(0, drop, *tile, drop+*tile), arm, arm.Bounds().Min, draw.Over)
		if *preview {
			drawBox(f, cx, cy, w)
		}
		fr[i] = f
	}

	outName := *name
	if outName == "" {
		if *preview {
			outName = "fistpump_preview_" + *side
		} else {
			outName = "fistpump_" + sanitizeName(input) + "_" + *side
		}
	}
	gifPath := filepath.Join(*out, outName+".gif")
	if err := encodeGIFDelays(gifPath, fr, delays); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, outName)
	return nil
}
