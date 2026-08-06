package emoji

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"math"
	"math/rand/v2"
	"path/filepath"
)

// runContentAware makes the "content aware" meme GIF: every frame seam-carves the
// image a little smaller — flat areas collapse, busy areas survive — then stretches
// it back to a square, so the subject swells, fattens and morphs. Carving taller
// than wide leaves the subject stretched as well as magnified. Frying (saturation,
// contrast, unsharp crunch) ramps up with the carve. The back half replays the
// front half in reverse, so the loop is seamless.
func runContentAware(args []string) error {
	fs := flag.NewFlagSet("content-aware", flag.ContinueOnError)
	tile := fs.Int("tile", 112, "emoji size in px (square)")
	frames := fs.Int("frames", 10, "number of frames (in, then back out)")
	dur := fs.Int("dur", 80, "milliseconds per frame (lower = faster)")
	zoom := fs.Float64("zoom", 0.25, "extra magnification at the peak (0 = carve only)")
	warp := fs.Float64("warp", 0.55, "how far to seam-carve at the peak, as a fraction of the tile (0-0.9)")
	stretch := fs.Float64("stretch", 1.25, "how much taller than wide the subject ends up at the peak")
	fry := fs.Float64("fry", 1.0, "deep-fry strength at the peak (0 = none)")
	name := fs.String("name", "", "emoji name / output basename (default: <file>_content_aware)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "content-aware [flags] <input>",
		"Make a content-aware-scale GIF: the image squeezes in and deep-fries, then bounces back.")
	input, err := parseInput(fs, args)
	if err != nil {
		return err
	}
	// Below 8px the carve targets bottom out and every frame comes back identical.
	if *tile < 8 || *tile > maxEmojiPx {
		return fmt.Errorf("tile must be between 8 and %d", maxEmojiPx)
	}
	if *frames < 3 {
		return fmt.Errorf("frames must be >= 3, got %d (the bounce needs a midpoint)", *frames)
	}
	// Inverted range checks so NaN is rejected too — every comparison against NaN
	// is false, so the usual `v < lo || v > hi` form would wave it through.
	if !(*warp >= 0 && *warp <= 0.9) {
		return fmt.Errorf("warp must be in [0, 0.9], got %v", *warp)
	}
	if !(*zoom >= 0 && *zoom <= 4) {
		return fmt.Errorf("zoom must be in [0, 4], got %v", *zoom)
	}
	if !(*stretch >= 0.5 && *stretch <= 2) {
		return fmt.Errorf("stretch must be in [0.5, 2], got %v", *stretch)
	}
	if !(*fry >= 0 && *fry <= 5) {
		return fmt.Errorf("fry must be in [0, 5], got %v", *fry)
	}

	src, err := loadNative(input, *tile)
	if err != nil {
		return err
	}

	// Only the squeeze-in half is rendered; frame i mirrors it back out.
	peak := (*frames - 1) / 2
	steps := make([]*image.RGBA, peak+1)
	carved := fitSquare(src, *tile)
	steps[0] = carved
	for k := 1; k <= peak; k++ {
		t := float64(k) / float64(peak)
		// Carving to a shorter canvas than it is narrow, then stretching the whole
		// thing back to a square, is what turns the zoom into a morph.
		w := float64(*tile) * (1 - *warp*t)
		h := w / (1 + (*stretch-1)*t)
		carved = carveTo(carved, max(int(math.Round(w)), 4), max(int(math.Round(h)), 4))
		steps[k] = deepFry(zoomCrop(carved, *tile, 1+*zoom*t), *fry*t)
	}

	fr := make([]*image.RGBA, *frames)
	for i := range fr {
		k := i
		if back := *frames - 1 - i; back < k {
			k = back
		}
		fr[i] = steps[k]
	}

	base := orNameSuffix(*name, input, "_content_aware")
	gifPath := filepath.Join(*out, base+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, base)
	return nil
}

// zoomCrop stretches src back out to a size*mag square and keeps the middle
// size x size of it: the carve alone doesn't push the subject far enough into the
// frame, so a little crop-in finishes the job.
func zoomCrop(src *image.RGBA, size int, mag float64) *image.RGBA {
	big := max(int(math.Round(float64(size)*mag)), size)
	scaled := scaleTo(src, big, big)
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(dst, dst.Bounds(), scaled, image.Pt((big-size)/2, (big-size)/2), draw.Src)
	return dst
}

// carveTo seam-carves src down to w x h, one seam at a time. Carving only ever
// removes, so a target at or above the current size is a no-op on that axis;
// callers step the targets down so each frame carves the previous frame's result.
func carveTo(src *image.RGBA, w, h int) *image.RGBA {
	// removeSeam refuses to go below 1px, so a smaller target would spin forever.
	w, h = max(w, 1), max(h, 1)
	// Alternate axes, always carving whichever one is proportionally furthest from
	// its target. Taking all the columns out first leaves the row pass too little
	// flat background to route through, and its seams cut through the subject.
	img := src
	for {
		dx, dy := img.Bounds().Dx(), img.Bounds().Dy()
		switch {
		case dx > w && float64(dx)/float64(w) >= float64(dy)/float64(h):
			img = removeSeam(img)
		case dy > h:
			img = transposeRGBA(removeSeam(transposeRGBA(img)))
		default:
			return img
		}
	}
}

// removeSeam drops the cheapest top-to-bottom seam, narrowing img by 1px. The
// cost of a pixel is the *new* edge that removing it would create (forward
// energy): slicing through an eye joins two mismatched neighbours and is
// expensive, so seams stay in flat areas and features survive at full size.
func removeSeam(img *image.RGBA) *image.RGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	if w < 2 {
		return img
	}
	lum := luma(img)
	at := func(x, y int) float64 { return lum[min(max(y, 0), h-1)*w+min(max(x, 0), w-1)] }

	cost := make([]float64, w*h)
	back := make([]int, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Removing (x,y) butts x-1 against x+1; the seam may also step sideways,
			// which additionally butts the pixel above against its new neighbour.
			up := math.Abs(at(x+1, y) - at(x-1, y))
			if y == 0 {
				cost[x] = up
				continue
			}
			best, bx := cost[(y-1)*w+x]+up, x
			if x > 0 {
				if v := cost[(y-1)*w+x-1] + up + math.Abs(at(x, y-1)-at(x-1, y)); v < best {
					best, bx = v, x-1
				}
			}
			if x < w-1 {
				if v := cost[(y-1)*w+x+1] + up + math.Abs(at(x, y-1)-at(x+1, y)); v < best {
					best, bx = v, x+1
				}
			}
			cost[y*w+x], back[y*w+x] = best, bx
		}
	}

	seamX := 0
	for x := 1; x < w; x++ {
		if cost[(h-1)*w+x] < cost[(h-1)*w+seamX] {
			seamX = x
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w-1, h))
	for y := h - 1; y >= 0; y-- {
		row := img.Pix[y*img.Stride:]
		out := dst.Pix[y*dst.Stride:]
		copy(out, row[:seamX*4])
		copy(out[seamX*4:], row[(seamX+1)*4:w*4])
		seamX = back[y*w+seamX]
	}
	return dst
}

// luma flattens each pixel to one number for the seam cost. Alpha is added on top
// of brightness so a transparent margin reads as empty (cheap to remove) and the
// outline of a sprite reads as a hard edge (expensive to cut through).
func luma(img *image.RGBA) []float64 {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	out := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*img.Stride + x*4
			p := img.Pix[i : i+4]
			out[y*w+x] = 0.299*float64(p[0]) + 0.587*float64(p[1]) + 0.114*float64(p[2]) + float64(p[3])
		}
	}
	return out
}

func transposeRGBA(img *image.RGBA) *image.RGBA {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			s := y*img.Stride + x*4
			d := x*dst.Stride + y*4
			copy(dst.Pix[d:d+4], img.Pix[s:s+4])
		}
	}
	return dst
}

// deepFry crunches an image the way a re-saved-a-hundred-times meme looks:
// unsharp mask for the dark edge halos, then a saturation and contrast shove.
// amount 0 returns the image untouched; 1 is the usual meme dose.
func deepFry(img *image.RGBA, amount float64) *image.RGBA {
	if amount <= 0 {
		return img
	}
	sharpen := 1.8 * amount
	sat := 1 + 0.7*amount
	contrast := 1 + 0.5*amount
	grain := 5.0 * amount

	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	blur := boxBlur(img)
	dst := image.NewRGBA(img.Bounds())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*img.Stride + x*4
			a := float64(img.Pix[i+3])
			var rgb [3]float64
			for c := 0; c < 3; c++ {
				p := float64(img.Pix[i+c])
				rgb[c] = p + sharpen*(p-blur[i+c])
			}
			// Work in straight alpha so the shove doesn't push premultiplied
			// channels above their alpha (which shows up as bright fringing).
			if a > 0 {
				for c := 0; c < 3; c++ {
					rgb[c] = rgb[c] * 255 / a
				}
			}
			gray := 0.299*rgb[0] + 0.587*rgb[1] + 0.114*rgb[2]
			for c := 0; c < 3; c++ {
				v := gray + (rgb[c]-gray)*sat
				v = (v-128)*contrast + 128
				v += grain * (rand.Float64()*2 - 1) // the re-compressed-to-death speckle
				dst.Pix[i+c] = uint8(min(max(v*a/255, 0), a))
			}
			dst.Pix[i+3] = uint8(a)
		}
	}
	return dst
}

// boxBlur returns a 3x3 mean of every channel, laid out like img.Pix.
func boxBlur(img *image.RGBA) []float64 {
	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	out := make([]float64, len(img.Pix))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum [4]float64
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					i := min(max(y+dy, 0), h-1)*img.Stride + min(max(x+dx, 0), w-1)*4
					for c := 0; c < 4; c++ {
						sum[c] += float64(img.Pix[i+c])
					}
				}
			}
			o := y*img.Stride + x*4
			for c := 0; c < 4; c++ {
				out[o+c] = sum[c] / 9
			}
		}
	}
	return out
}
