package emoji

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"os"

	"github.com/ericpauley/go-quantize/quantize"
)

// encodeGIF writes an infinitely-looping animated GIF with one delay for every
// frame. See encodeGIFDelays for per-frame timing.
func encodeGIF(path string, frames []*image.RGBA, delayCs int) error {
	delays := make([]int, len(frames))
	for i := range delays {
		delays[i] = delayCs
	}
	return encodeGIFDelays(path, frames, delays)
}

// maxEmojiBytes is Slack's file-size ceiling for a custom emoji (128 KiB). Slack
// rejects anything larger even after its own downscale, so we must land under it.
const maxEmojiBytes = 128 * 1024

// colorBudgets are the opaque-palette sizes encodeGIFDelays tries, largest first,
// stopping at the first GIF that fits under maxEmojiBytes. Each value is one less
// than a power of two so that value+1 (the reserved transparent entry) lands on a
// 2^n palette, shrinking image/gif's per-pixel LZW code size — the biggest size
// lever short of dropping frames or resolution. Fewer colors trades fidelity for
// fit, so we only step down when the richer palette overflows.
var colorBudgets = []int{255, 127, 63, 31}

// encodeGIFDelays writes an infinitely-looping animated GIF from RGBA frames,
// each shown for delays[i] centiseconds. A dedicated palette index is reserved
// for transparency (alpha < 128), so a dominant color can never be mistaken for
// the transparent color. The palette shrinks through colorBudgets until the
// encoded GIF fits under Slack's size limit (or the smallest budget is reached).
func encodeGIFDelays(path string, frames []*image.RGBA, delays []int) error {
	var buf bytes.Buffer
	for i, colors := range colorBudgets {
		buf.Reset()
		if err := encodeGIFColors(&buf, frames, delays, colors); err != nil {
			return err
		}
		if buf.Len() <= maxEmojiBytes || i == len(colorBudgets)-1 {
			break // fits, or we're out of budgets to try
		}
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		os.Remove(path) // don't leave a truncated GIF behind
		return err
	}
	if buf.Len() > maxEmojiBytes {
		// Even the smallest palette overflowed. The file is written (still usable
		// elsewhere) but Slack will reject it, so warn instead of reporting success.
		fmt.Fprintf(os.Stderr, "warning: %s is %d bytes, over Slack's %d-byte emoji limit; try fewer frames or a smaller -tile\n",
			path, buf.Len(), maxEmojiBytes)
	}
	return nil
}

// encodeGIFColors renders every frame against a shared palette of up to `colors`
// opaque entries (plus one transparent) and writes the animated GIF to w.
func encodeGIFColors(w *bytes.Buffer, frames []*image.RGBA, delays []int, colors int) error {
	pal, transpIdx := palettize(frames, colors)
	g := &gif.GIF{LoopCount: 0}
	for i, fr := range frames {
		g.Image = append(g.Image, toPaletted(fr, pal, transpIdx))
		g.Delay = append(g.Delay, delays[i])
		g.Disposal = append(g.Disposal, gif.DisposalBackground)
	}
	return gif.EncodeAll(w, g)
}

// gifFrames decodes an animated GIF into fully-composited RGBA frames plus each
// frame's delay in centiseconds, honoring GIF disposal so partial frames build
// on (or clear) their predecessor instead of smearing.
func gifFrames(data []byte) ([]*image.RGBA, []int, error) {
	g, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}
	rect := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	if rect.Empty() { // some GIFs omit a logical screen size
		rect = g.Image[0].Bounds()
	}
	canvas := image.NewRGBA(rect)
	frames := make([]*image.RGBA, len(g.Image))
	for i, src := range g.Image {
		var backup *image.RGBA
		if g.Disposal[i] == gif.DisposalPrevious {
			backup = cloneRGBA(canvas)
		}
		b := src.Bounds()
		draw.Draw(canvas, b, src, b.Min, draw.Over)
		frames[i] = cloneRGBA(canvas)
		switch g.Disposal[i] {
		case gif.DisposalBackground:
			draw.Draw(canvas, b, image.Transparent, image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			canvas = backup
		}
	}
	return frames, g.Delay, nil
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

// subsampleFrames drops frames evenly until at most max remain, folding each
// dropped frame's delay into the kept frame before it so the loop keeps its
// original total duration. Slack caps animated emoji at 50 frames.
func subsampleFrames(frames []*image.RGBA, delays []int, max int) ([]*image.RGBA, []int) {
	n := len(frames)
	if n <= max || max < 1 {
		return frames, delays
	}
	outF := make([]*image.RGBA, 0, max)
	outD := make([]int, 0, max)
	prevBucket := -1
	for i := 0; i < n; i++ {
		if b := i * max / n; b != prevBucket {
			outF = append(outF, frames[i])
			outD = append(outD, delays[i])
			prevBucket = b
		} else {
			outD[len(outD)-1] += delays[i] // preserve total loop time
		}
	}
	return outF, outD
}

// palettize quantizes the opaque colors (budget `colors`) across ALL frames and
// appends one fully transparent entry as the last index. Quantizing every frame
// matters for commands like panic where frames differ in scale/color (frame 0 is
// the most downscaled), so a single-frame palette would dull the sharper frames.
func palettize(frames []*image.RGBA, colors int) (color.Palette, int) {
	// Lay every frame side by side so the quantizer sees the whole animation's colors.
	b := frames[0].Bounds()
	montage := image.NewRGBA(image.Rect(0, 0, b.Dx()*len(frames), b.Dy()))
	for i, fr := range frames {
		draw.Draw(montage, image.Rect(i*b.Dx(), 0, (i+1)*b.Dx(), b.Dy()), fr, fr.Bounds().Min, draw.Src)
	}
	// Mean aggregation forces every quantized entry to alpha 255, so the single
	// appended {0,0,0,0} below is the ONLY alpha-0 entry. image/gif's encoder
	// picks the first alpha-0 palette entry as the transparent index, so this
	// guarantees it selects transpIdx. The zero-value (Mode) quantizer would
	// instead keep the transparent background as its own alpha-0 entry, which the
	// encoder would pick first — rendering transparent pixels as opaque black.
	q := quantize.MedianCutQuantizer{Aggregation: quantize.Mean}
	pal := q.Quantize(make(color.Palette, 0, colors), montage)
	transpIdx := len(pal)
	pal = append(pal, color.RGBA{})
	return pal, transpIdx
}

func toPaletted(fr *image.RGBA, pal color.Palette, transpIdx int) *image.Paletted {
	out := image.NewPaletted(fr.Bounds(), pal)
	b := fr.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if fr.RGBAAt(x, y).A < 128 {
				out.SetColorIndex(x, y, uint8(transpIdx))
			} else {
				out.Set(x, y, fr.At(x, y)) // nearest opaque color; transparent entry is never closest
			}
		}
	}
	return out
}
