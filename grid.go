package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"strings"
)

func runGrid(args []string) error {
	fs := flag.NewFlagSet("grid", flag.ContinueOnError)
	cols := fs.Int("cols", 3, "number of columns")
	rows := fs.Int("rows", 3, "number of rows")
	tile := fs.Int("tile", 128, "tile size in px (square)")
	name := fs.String("name", "", "emoji name prefix (default: from filename)")
	out := fs.String("out", "", "output dir (default: <name>-grid)")
	flip := fs.Bool("flip", false, "mirror horizontally before slicing")
	fs.Usage = usageFor(fs, "grid [flags] <input>",
		"Split an image into cols*rows square tiles that reassemble into the original.")
	input, err := parseInput(fs, args)
	if err != nil {
		return err
	}
	if *cols < 1 || *rows < 1 || *tile < 1 {
		return fmt.Errorf("cols, rows, and tile must be >= 1")
	}
	if *tile > maxEmojiPx {
		return fmt.Errorf("tile must be <= %d (Slack emoji limit)", maxEmojiPx)
	}

	prefix := orName(*name, input)
	dir := *out
	if dir == "" {
		dir = prefix + "-grid"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	W, H := *cols**tile, *rows**tile
	img, err := loadImage(input, W, H)
	if err != nil {
		return err
	}
	var src image.Image = img
	if *flip {
		src = flipH(img)
	}

	tiles := sliceTiles(src, *cols, *rows, *tile)
	for r := 0; r < *rows; r++ {
		for c := 0; c < *cols; c++ {
			p := filepath.Join(dir, fmt.Sprintf("%s_r%d_c%d.png", prefix, r, c))
			if err := writePNG(tiles[r**cols+c], p); err != nil {
				return err
			}
		}
	}
	if err := writePNG(src, filepath.Join(dir, "_preview.png")); err != nil {
		return err
	}

	fmt.Printf("Wrote %d tiles to %s/\n\nPaste into Slack (one line per row, no spaces):\n\n", *cols**rows, dir)
	var b strings.Builder
	for r := 0; r < *rows; r++ {
		for c := 0; c < *cols; c++ {
			fmt.Fprintf(&b, ":%s_r%d_c%d:", prefix, r, c)
		}
		b.WriteByte('\n')
	}
	fmt.Print(b.String())
	return nil
}

// sliceTiles cuts src into cols*rows tiles of tile px, in row-major order.
func sliceTiles(src image.Image, cols, rows, tile int) []*image.RGBA {
	tiles := make([]*image.RGBA, 0, cols*rows)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			t := image.NewRGBA(image.Rect(0, 0, tile, tile))
			draw.Draw(t, t.Bounds(), src, image.Pt(src.Bounds().Min.X+c*tile, src.Bounds().Min.Y+r*tile), draw.Src)
			tiles = append(tiles, t)
		}
	}
	return tiles
}

func flipH(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.X-1-x, y, src.At(x, y))
		}
	}
	return dst
}
