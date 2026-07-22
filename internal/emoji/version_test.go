package emoji

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVersionNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.2.0", "v1.1.9", true},
		{"v1.2.0", "1.2.0", false},        // equal, with/without leading v
		{"v1.2.0", "v1.2.1", false},       // current is ahead
		{"v2.0.0", "v1.9.9", true},        // major bump
		{"v1.10.0", "v1.9.0", true},       // numeric, not lexical, compare
		{"v1.2.0", "v1.2.0-rc1", false},   // suffixes ignored; released binaries never carry them
		{"v1.2.0", "dev", true},           // dev parses to 0.0.0
		{"v1.2.3+build", "v1.2.3", false}, // build metadata ignored
	}
	for _, c := range cases {
		if got := versionNewer(c.latest, c.current); got != c.want {
			t.Errorf("versionNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

// A fresh cache must short-circuit and return the cached tag WITHOUT any network
// call — the core throttle contract. Isolating HOME points os.UserCacheDir at a
// temp dir; if this reached the network it would be slow/flaky instead of instant.
func TestCachedLatestReusesFreshCache(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", "") // linux: fall back to $HOME/.cache
	t.Setenv("HOME", t.TempDir())  // darwin: $HOME/Library/Caches
	path := cachePath()
	if path == "" {
		t.Skip("no user cache dir on this platform")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(updateCache{CheckedAt: time.Now(), Latest: "v9.9.9"})
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if got, ok := cachedOrFetchLatest(); !ok || got != "v9.9.9" {
		t.Errorf("fresh cache: got (%q, %v), want (v9.9.9, true)", got, ok)
	}
}
