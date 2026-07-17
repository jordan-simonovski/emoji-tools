// Command emoji-tools turns images (SVG or raster) into Slack emojis:
// grids of tiles, seamless scrolling loops, "intensifies" shakes, and
// "old man yells at" memes.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

const usage = `emoji-tools — make Slack emojis from images

Usage:
  emoji-tools <command> [flags] <input>

Commands:
  grid       Split an image into an NxM grid of square emoji tiles
  scroll     Seamless horizontal-scroll looping GIF (a "train" of the logo)
  intensify  Shaking "<x>-intensifies" animated GIF
  yells-at   Abe Simpson yelling at your image
  uwu        Overlay a uwu face onto your image
  petpet     A hand patting your image (petpet meme)
  panic      Frantic zoom-pulse + shake animation
  party      Cycle the image through party-parrot rainbow colours
  party-blob Rainbow colour-cycle plus a bouncing wobble
  spin       3D coin-flip spin around the vertical axis

Run "emoji-tools <command> -h" for command-specific flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "grid":
		err = runGrid(os.Args[2:])
	case "scroll":
		err = runScroll(os.Args[2:])
	case "intensify":
		err = runIntensify(os.Args[2:])
	case "yells-at", "yells":
		err = runYells(os.Args[2:])
	case "uwu":
		err = runUwu(os.Args[2:])
	case "petpet":
		err = runPetpet(os.Args[2:])
	case "panic":
		err = runPanic(os.Args[2:])
	case "party":
		err = runParty(os.Args[2:])
	case "party-blob":
		err = runPartyBlob(os.Args[2:])
	case "spin":
		err = runSpin(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return // usage already printed by the flag package
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
