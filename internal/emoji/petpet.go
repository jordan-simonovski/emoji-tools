package emoji

import (
	"embed"
	"flag"
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"
	"path/filepath"
)

//go:embed assets/pet/pet0.gif assets/pet/pet1.gif assets/pet/pet2.gif assets/pet/pet3.gif assets/pet/pet4.gif assets/pet/pet5.gif assets/pet/pet6.gif assets/pet/pet7.gif assets/pet/pet8.gif assets/pet/pet9.gif
var petHandFS embed.FS

const petFrames = 10

func runPetpet(args []string) error {
	fs := flag.NewFlagSet("petpet", flag.ContinueOnError)
	tile := fs.Int("tile", 128, "emoji size in px (square)")
	dur := fs.Int("dur", 50, "milliseconds per frame (lower = faster patting)")
	name := fs.String("name", "", "emoji name / output basename (default: pet_<file>)")
	out := fs.String("out", ".", "output directory")
	fs.Usage = usageFor(fs, "petpet [flags] <input>",
		"Make a petpet GIF (a hand patting your image). Hand art from aDu/pet-pet-gif.")
	input, err := parseInput(fs, args)
	if err != nil {
		return err
	}
	if *tile < 1 || *tile > maxEmojiPx {
		return fmt.Errorf("tile must be between 1 and %d", maxEmojiPx)
	}

	base, err := loadImage(input, *tile, *tile)
	if err != nil {
		return err
	}

	fr := make([]*image.RGBA, petFrames)
	for i := 0; i < petFrames; i++ {
		// squash-and-stretch table from aDu/pet-pet-gif: as the hand presses down
		// the subject widens and flattens, anchored toward the bottom.
		j := i
		if i >= petFrames/2 {
			j = petFrames - i
		}
		w := 0.8 + float64(j)*0.02
		h := 0.8 - float64(j)*0.05
		ox := (1-w)*0.5 + 0.1
		oy := (1 - h) - 0.08

		aw := int(float64(*tile) * w)
		ah := int(float64(*tile) * h)
		x := int(float64(*tile) * ox)
		y := int(float64(*tile) * oy)

		f := image.NewRGBA(image.Rect(0, 0, *tile, *tile))
		subject := scaleTo(base, aw, ah)
		draw.Draw(f, image.Rect(x, y, x+aw, y+ah), subject, subject.Bounds().Min, draw.Over)

		hand, err := loadPetHand(i, *tile)
		if err != nil {
			return err
		}
		draw.Draw(f, f.Bounds(), hand, hand.Bounds().Min, draw.Over)
		fr[i] = f
	}

	base2 := *name
	if base2 == "" {
		base2 = "pet_" + sanitizeName(input)
	}
	gifPath := filepath.Join(*out, base2+".gif")
	if err := encodeGIF(gifPath, fr, centsFromMillis(*dur)); err != nil {
		return err
	}
	fmt.Printf("Wrote %s\n\nUpload it, then use  :%s:\n", gifPath, base2)
	return nil
}

func loadPetHand(i, size int) (*image.RGBA, error) {
	f, err := petHandFS.Open(fmt.Sprintf("assets/pet/pet%d.gif", i))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decoding embedded pet hand %d: %w", i, err)
	}
	return scaleTo(img, size, size), nil
}
