// Command emoji-tools turns images (SVG or raster) into Slack emojis. All the
// work lives in internal/emoji; this is just the entry point.
package main

import (
	"os"

	"github.com/jordan-simonovski/emoji-tools/internal/emoji"
)

func main() {
	os.Exit(emoji.Run(os.Args[1:]))
}
