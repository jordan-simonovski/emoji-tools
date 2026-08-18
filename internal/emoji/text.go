package emoji

import (
	_ "embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
	"unicode/utf8"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Anton (SIL OFL 1.1) stands in for the heavy condensed grotesque used by the
// original :same-tbh: emoji, which is a proprietary face we can't ship.
//
//go:embed assets/Anton-Regular.ttf
var antonTTF []byte

// Defaults copied from the original :same-tbh: emoji: its red, and lines set
// edge to edge with barely any leading.
const (
	sameTbhRed = "#dd1c22"
	textLead   = 0.08 // gap between lines, as a fraction of the cap height
	textRender = 300  // px the lines are rasterised at before being scaled down
)

// maxLineRunes caps how long one word may be. Each line is rasterised at
// textRender px before being scaled down, so the mask grows with the word while
// the result shrinks: past this a word is a couple of px tall even at the 256px
// ceiling, and a few thousand characters costs gigabytes to render nothing.
const maxLineRunes = 64

func runText(args []string) error {
	fs := flag.NewFlagSet("text", flag.ContinueOnError)
	size := fs.Int("size", 128, "output emoji size in px (square)")
	hex := fs.String("color", sameTbhRed, "text colour as #rrggbb")
	out := fs.String("out", "", "output file (default: <words>.png)")
	fs.Usage = usageFor(fs, "text [flags] <words...>",
		`Block-capital word emoji in the ":same-tbh:" style: one word per line, stacked edge to edge.`)
	words, err := parseWords(fs, args)
	if err != nil {
		return err
	}
	if *size < 1 || *size > maxEmojiPx {
		return fmt.Errorf("size must be between 1 and %d", maxEmojiPx)
	}
	fg, err := parseHexColor(*hex)
	if err != nil {
		return err
	}
	lines := textLines(words)
	if len(lines) == 0 {
		fs.Usage()
		return fmt.Errorf("need at least one word")
	}
	for _, l := range lines {
		if n := utf8.RuneCountInString(l); n > maxLineRunes {
			return fmt.Errorf("a word of %d characters can't fit an emoji (max %d)", n, maxLineRunes)
		}
	}

	img, err := drawTextTile(lines, *size, fg)
	if err != nil {
		return err
	}
	outPath := *out
	if outPath == "" {
		outPath = sanitizeWords(strings.Join(lines, "_")) + ".png"
	}
	if err := writePNG(img, outPath); err != nil {
		return err
	}
	fmt.Printf("Wrote %s (%dx%d)\n", outPath, *size, *size)
	return nil
}

// parseWords splits args into leading positional words and trailing flags, so
// flags work on either side of the words like they do for the image commands.
func parseWords(fs *flag.FlagSet, args []string) ([]string, error) {
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	var words []string
	rest := fs.Args()
	for len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		words = append(words, rest[0])
		rest = rest[1:]
	}
	if err := fs.Parse(rest); err != nil {
		return nil, err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return nil, fmt.Errorf("put all the words before the flags, got %v after them", fs.Args())
	}
	return words, nil
}

// textLines turns the words into upper-case lines, one word per line — the
// stacked-word layout of the original.
func textLines(words []string) []string {
	var lines []string
	for _, w := range strings.Fields(strings.Join(words, " ")) {
		lines = append(lines, strings.ToUpper(w))
	}
	return lines
}

func parseHexColor(s string) (color.RGBA, error) {
	h := strings.TrimPrefix(s, "#")
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil || len(h) != 6 {
		return color.RGBA{}, fmt.Errorf("color %q is not #rrggbb", s)
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}, nil
}

// drawTextTile stacks the lines on a transparent square, scaled up until the
// widest line or the whole stack hits the edge — the tight, edge-to-edge look
// of the original.
func drawTextTile(lines []string, size int, fg color.RGBA) (*image.RGBA, error) {
	sf, face, err := antonFace(textRender)
	if err != nil {
		return nil, err
	}
	defer face.Close()

	masks := make([]*image.Alpha, len(lines))
	inks := make([]image.Rectangle, len(lines))
	maxW, maxH := 0, 0
	var buf sfnt.Buffer
	for i, l := range lines {
		// Anton covers Latin only. Without this every other script rasterises as
		// a row of .notdef boxes, which the ink check below can't tell from real
		// glyphs, so the command would report success on a tile full of tofu.
		for _, r := range l {
			if gi, err := sf.GlyphIndex(&buf, r); err != nil || gi == 0 {
				return nil, fmt.Errorf("Anton has no glyph for %q; it only sets Latin text", r)
			}
		}
		masks[i], inks[i] = lineMask(face, l)
		if inks[i].Empty() {
			return nil, fmt.Errorf("%q has no printable characters", l)
		}
		maxW = max(maxW, inks[i].Dx())
		maxH = max(maxH, inks[i].Dy())
	}

	// One scale for every line, so the caps all come out the same height.
	lead := int(float64(maxH) * textLead)
	stackH := len(lines)*maxH + (len(lines)-1)*lead
	scale := min(float64(size)/float64(maxW), float64(size)/float64(stackH))

	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	src := image.NewUniform(fg)
	y := (size - int(float64(stackH)*scale)) / 2
	step := int(float64(maxH+lead) * scale)
	for i := range lines {
		w := int(float64(inks[i].Dx()) * scale)
		h := int(float64(inks[i].Dy()) * scale)
		if w < 1 || h < 1 {
			// Every line rounding away would otherwise write a blank tile and
			// report success.
			return nil, fmt.Errorf("%d word(s) don't fit a %dpx tile; use fewer or shorter words, or a bigger -size", len(lines), size)
		}
		scaled := image.NewAlpha(image.Rect(0, 0, w, h))
		xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), masks[i], inks[i], xdraw.Src, nil)
		at := image.Pt((size-w)/2, y+i*step)
		draw.DrawMask(dst, scaled.Bounds().Add(at), src, image.Point{}, scaled, image.Point{}, draw.Over)
	}
	return dst, nil
}

// antonFace returns the parsed font alongside the face: the face rasterises,
// and the font is what can be asked whether a rune has a glyph at all.
func antonFace(px float64) (*sfnt.Font, font.Face, error) {
	f, err := opentype.Parse(antonTTF)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing embedded Anton font: %w", err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: px, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, nil, fmt.Errorf("sizing embedded Anton font: %w", err)
	}
	return f, face, nil
}

// lineMask rasterises one line and returns the mask plus the rectangle its ink
// actually covers — cropping to the ink is what makes the caps sit flush to the
// tile edges instead of floating inside the font's ascent/descent.
func lineMask(face font.Face, s string) (*image.Alpha, image.Rectangle) {
	m := face.Metrics()
	pad := 8
	w := font.MeasureString(face, s).Ceil() + 2*pad
	h := (m.Ascent + m.Descent).Ceil() + 2*pad
	mask := image.NewAlpha(image.Rect(0, 0, w, h))
	d := font.Drawer{Dst: mask, Src: image.Opaque, Face: face,
		Dot: fixed.P(pad, m.Ascent.Ceil()+pad)}
	d.DrawString(s)
	return mask, inkBounds(mask)
}

// inkBounds is the tightest rectangle containing every non-transparent pixel.
func inkBounds(m *image.Alpha) image.Rectangle {
	b := m.Bounds()
	ink := image.Rectangle{Min: b.Max, Max: b.Min}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		row := m.Pix[(y-b.Min.Y)*m.Stride:][:b.Dx()]
		for x, a := range row {
			if a == 0 {
				continue
			}
			ink.Min.X = min(ink.Min.X, b.Min.X+x)
			ink.Max.X = max(ink.Max.X, b.Min.X+x+1)
			ink.Min.Y = min(ink.Min.Y, y)
			ink.Max.Y = max(ink.Max.Y, y+1)
		}
	}
	if ink.Empty() {
		return image.Rectangle{}
	}
	return ink
}
