// Command emoji-tools turns images (SVG or raster) into Slack emojis. All the
// work lives in internal/emoji; this is just the entry point.
package main

import (
	"os"

	"github.com/jordan-simonovski/emoji-tools/internal/emoji"
)

// version is set at release time via -ldflags "-X main.version=...". It stays
// "dev" for source builds.
var version = "dev"

func main() {
	emoji.Version = version
	os.Exit(emoji.Run(os.Args[1:]))
}
