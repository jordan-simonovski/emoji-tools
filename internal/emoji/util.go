package emoji

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// usageFor returns a flag.FlagSet usage function with a one-line description.
func usageFor(fs *flag.FlagSet, use, desc string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "%s\n\nusage: emoji-tools %s\n", desc, use)
		fs.PrintDefaults()
	}
}

// parseInput parses flags and returns the single input file, tolerating flags
// placed before or after the positional argument (Go's flag package otherwise
// stops at the first non-flag token).
func parseInput(fs *flag.FlagSet, args []string) (string, error) {
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fs.Usage()
		return "", fmt.Errorf("need exactly one input file")
	}
	input := rest[0]
	if err := fs.Parse(rest[1:]); err != nil {
		return "", err
	}
	if fs.NArg() != 0 {
		fs.Usage()
		return "", fmt.Errorf("unexpected extra arguments: %v", fs.Args())
	}
	return input, nil
}

// orName returns the flag value if set, else a name derived from the path.
func orName(flagVal, path string) string {
	if flagVal != "" {
		return flagVal
	}
	return sanitizeName(path)
}

// orNameSuffix is orName with a per-command suffix on the derived name, so each
// maker writes a distinct default file (e.g. "_party" vs "_party_blob").
func orNameSuffix(flagVal, path, suffix string) string {
	if flagVal != "" {
		return flagVal
	}
	return sanitizeName(path) + suffix
}

var nonName = regexp.MustCompile(`[^a-z0-9_-]+`)

// sanitizeName turns a file path into a Slack-safe emoji name.
func sanitizeName(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = nonName.ReplaceAllString(strings.ToLower(base), "_")
	base = strings.Trim(base, "_")
	if base == "" {
		base = "emoji"
	}
	return base
}

func writePNG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		os.Remove(path) // don't leave a truncated PNG behind
		return err
	}
	return f.Close() // surface a flush/close error instead of reporting success
}

// centsFromMillis converts a millisecond frame duration to GIF centiseconds,
// clamped to at least 1 (0 makes many renderers fall back to ~100ms).
func centsFromMillis(ms int) int {
	cs := (ms + 5) / 10
	if cs < 1 {
		cs = 1
	}
	return cs
}
