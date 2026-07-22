// Package emoji turns images (SVG or raster) into Slack emojis: grids of tiles,
// seamless scrolling loops, "intensifies" shakes, party colour-cycles, spins,
// and animated overlays like confetti and bongo cat.
package emoji

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
  confetti   Rain confetti over your image (animated overlay)
  bongocat   Bongo cat drumming on your image
  statham    Trace your image over Jason Statham's head as he dances

Run "emoji-tools <command> -h" for command-specific flags.
`

// Run dispatches a single CLI invocation (args is os.Args[1:]) and returns the
// process exit code: 2 for a usage error, 1 for a runtime failure, 0 otherwise.
func Run(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "grid":
		err = runGrid(args[1:])
	case "scroll":
		err = runScroll(args[1:])
	case "intensify":
		err = runIntensify(args[1:])
	case "yells-at", "yells":
		err = runYells(args[1:])
	case "uwu":
		err = runUwu(args[1:])
	case "petpet":
		err = runPetpet(args[1:])
	case "panic":
		err = runPanic(args[1:])
	case "party":
		err = runParty(args[1:])
	case "party-blob":
		err = runPartyBlob(args[1:])
	case "spin":
		err = runSpin(args[1:])
	case "confetti":
		err = runConfetti(args[1:])
	case "bongocat":
		err = runBongocat(args[1:])
	case "statham":
		err = runStatham(args[1:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", args[0], usage)
		return 2
	}
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0 // usage already printed by the flag package
		}
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}
