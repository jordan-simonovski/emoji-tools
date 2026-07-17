package emoji

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func isSVG(path string) bool { return strings.EqualFold(filepath.Ext(path), ".svg") }

// loadImage renders/resizes an image at path to exactly w x h (aspect ignored).
func loadImage(path string, w, h int) (*image.RGBA, error) {
	if isSVG(path) {
		return rasterizeSVG(path, w, h)
	}
	src, err := decodeRaster(path)
	if err != nil {
		return nil, err
	}
	return scaleTo(src, w, h), nil
}

// loadNative decodes an image at (roughly) its native resolution, preserving
// aspect. SVGs are rasterized to a canvas about `svgW` px wide.
func loadNative(path string, svgW int) (image.Image, error) {
	if isSVG(path) {
		icon, err := oksvg.ReadIcon(path, oksvg.WarnErrorMode)
		if err != nil {
			return nil, fmt.Errorf("reading svg %s: %w", path, err)
		}
		vw, vh := icon.ViewBox.W, icon.ViewBox.H
		if vw <= 0 {
			vw, vh = float64(svgW), float64(svgW)
		}
		w := svgW
		h := int(math.Round(float64(w) * vh / vw))
		return renderIcon(icon, w, h), nil
	}
	return decodeRaster(path)
}

func decodeRaster(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decoding %s: %w", path, err)
	}
	return img, nil
}

func scaleTo(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

// maxEmojiPx is Slack's practical upload ceiling for a custom emoji image.
const maxEmojiPx = 256

// fitSquare scales src to fit within a size x size box (preserving aspect) and
// centers it on a transparent square canvas. Slack emoji must be square, so
// every single-image emoji output goes through this.
func fitSquare(src image.Image, size int) *image.RGBA {
	s := src.Bounds().Size()
	longest := s.X
	if s.Y > longest {
		longest = s.Y
	}
	w, h := size, size
	if longest > 0 {
		w = int(math.Round(float64(s.X) * float64(size) / float64(longest)))
		h = int(math.Round(float64(s.Y) * float64(size) / float64(longest)))
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	scaled := scaleTo(src, w, h)
	canvas := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(canvas, image.Rect((size-w)/2, (size-h)/2, (size-w)/2+w, (size-h)/2+h),
		scaled, scaled.Bounds().Min, draw.Over)
	return canvas
}

// scaleToWidth scales src to the given width, preserving aspect ratio.
func scaleToWidth(src image.Image, w int) *image.RGBA {
	s := src.Bounds().Size()
	h := int(math.Round(float64(w) * float64(s.Y) / float64(s.X)))
	if h < 1 {
		h = 1
	}
	return scaleTo(src, w, h)
}

func rasterizeSVG(path string, w, h int) (*image.RGBA, error) {
	icon, err := oksvg.ReadIcon(path, oksvg.WarnErrorMode)
	if err != nil {
		return nil, fmt.Errorf("reading svg %s: %w", path, err)
	}
	return renderIcon(icon, w, h), nil
}

func renderIcon(icon *oksvg.SvgIcon, w, h int) *image.RGBA {
	icon.SetTarget(0, 0, float64(w), float64(h))
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())
	icon.Draw(rasterx.NewDasher(w, h, scanner), 1.0)
	return rgba
}
