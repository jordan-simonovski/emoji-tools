package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"path/filepath"
)

// runParty cycles the image's hue through the rainbow ("party parrot" colours)
// without moving it.
func runParty(args []string) error {
	fs := flag.NewFlagSet("party", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	frames := fs.Int("frames", 14, "number of frames (colour steps around the wheel)")
	dur := fs.Int("dur", 70, "milliseconds per frame (lower = faster)")
	name := fs.String("name", "", "emoji name / output basename (default: <file>_party)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "party [flags] <input>",
		"Cycle the image through party-parrot rainbow colours (no movement).")
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
	fr := partyColors(fitSquare(src, *tile), *frames)

	base := orNameSuffix(*name, input, "_party")
	gifPath := filepath.Join(*out, base+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, base)
	return nil
}

// runPartyBlob does the same rainbow cycle as party, plus a squash-and-stretch
// bounce so the subject wobbles like the classic party-blob emoji.
func runPartyBlob(args []string) error {
	fs := flag.NewFlagSet("party-blob", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	frames := fs.Int("frames", 14, "number of frames (colour steps + bounce)")
	dur := fs.Int("dur", 70, "milliseconds per frame (lower = faster)")
	amp := fs.Float64("amp", 0.14, "bounce/squish amplitude as a fraction of size")
	name := fs.String("name", "", "emoji name / output basename (default: <file>_party_blob)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "party-blob [flags] <input>",
		"Rainbow colour-cycle plus a bouncing squash-and-stretch wobble.")
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
	if *amp < 0 || *amp >= 0.5 {
		return fmt.Errorf("amp must be in [0, 0.5), got %v", *amp)
	}

	// Inset the content so a stretched frame and the sideways sway never clip.
	content := int(float64(*tile) * (1 - 2**amp))
	if content < 1 {
		return fmt.Errorf("tile %d too small for amp %v (need tile*(1-2*amp) >= 1)", *tile, *amp)
	}
	src, err := loadNative(input, content)
	if err != nil {
		return err
	}
	subject := fitSquare(src, content)
	floor := (*tile + content) / 2 // bottom edge of the centred content box

	fr := make([]*image.RGBA, *frames)
	for i, colored := range partyColors(subject, *frames) {
		phase := 2 * math.Pi * float64(i) / float64(*frames)
		squash := *amp * math.Sin(2*phase) // wide-short <-> tall-thin, twice per loop
		w := int(float64(content) * (1 + squash))
		h := int(float64(content) * (1 - squash))
		sq := scaleTo(colored, w, h)
		x := (*tile-w)/2 + int(float64(*tile)**amp*0.5*math.Sin(phase)) // gentle sway
		y := floor - h                                                  // grows/shrinks from a fixed floor
		f := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
		draw.Draw(f, image.Rect(x, y, x+w, y+h), sq, sq.Bounds().Min, draw.Over)
		fr[i] = f
	}

	base := orNameSuffix(*name, input, "_party_blob")
	gifPath := filepath.Join(*out, base+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, base)
	return nil
}

// partyColors returns one copy of subject per frame, each with its hue rotated a
// further step around the colour wheel — a full rainbow loop over all frames.
func partyColors(subject *image.RGBA, frames int) []*image.RGBA {
	fr := make([]*image.RGBA, frames)
	for i := range fr {
		fr[i] = hueRotate(subject, 360*float64(i)/float64(frames))
	}
	return fr
}

// hueRotate returns a copy of src with every opaque pixel's hue rotated by deg
// degrees, preserving luminance and alpha (the standard luma-preserving matrix).
func hueRotate(src *image.RGBA, deg float64) *image.RGBA {
	rad := deg * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	const lr, lg, lb = 0.213, 0.715, 0.072
	m := [3][3]float64{
		{lr + cos*(1-lr) - sin*lr, lg - cos*lg - sin*lg, lb - cos*lb + sin*(1-lb)},
		{lr - cos*lr + sin*0.143, lg + cos*(1-lg) + sin*0.140, lb - cos*lb - sin*0.283},
		{lr - cos*lr - sin*(1-lr), lg - cos*lg + sin*lg, lb + cos*(1-lb) + sin*lb},
	}
	dst := image.NewRGBA(src.Bounds())
	b := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := src.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			r, g, bl := float64(c.R), float64(c.G), float64(c.B)
			dst.SetRGBA(x, y, color.RGBA{
				R: clamp8(m[0][0]*r + m[0][1]*g + m[0][2]*bl),
				G: clamp8(m[1][0]*r + m[1][1]*g + m[1][2]*bl),
				B: clamp8(m[2][0]*r + m[2][1]*g + m[2][2]*bl),
				A: c.A,
			})
		}
	}
	return dst
}

func clamp8(v float64) uint8 {
	switch {
	case v < 0:
		return 0
	case v > 255:
		return 255
	default:
		return uint8(v + 0.5)
	}
}
