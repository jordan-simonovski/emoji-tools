package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"os"

	"github.com/ericpauley/go-quantize/quantize"
)

// encodeGIF writes an infinitely-looping animated GIF from RGBA frames.
// A dedicated palette index is reserved for transparency (alpha < 128), so a
// dominant color can never be mistaken for the transparent color.
func encodeGIF(path string, frames []*image.RGBA, delayCs int) error {
	pal, transpIdx := palettize(frames)
	g := &gif.GIF{LoopCount: 0}
	for _, fr := range frames {
		g.Image = append(g.Image, toPaletted(fr, pal, transpIdx))
		g.Delay = append(g.Delay, delayCs)
		g.Disposal = append(g.Disposal, gif.DisposalBackground)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := gif.EncodeAll(f, g); err != nil {
		f.Close()
		os.Remove(path) // don't leave a truncated GIF behind
		return err
	}
	return f.Close() // surface a flush/close error instead of reporting success
}

// palettize quantizes the opaque colors (budget 255) across ALL frames and
// appends one fully transparent entry as the last index. Quantizing every frame
// matters for commands like panic where frames differ in scale/color (frame 0 is
// the most downscaled), so a single-frame palette would dull the sharper frames.
func palettize(frames []*image.RGBA) (color.Palette, int) {
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
	pal := q.Quantize(make(color.Palette, 0, 255), montage)
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
