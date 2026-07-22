package emoji

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Version is the build version, injected via -ldflags "-X main.version=..." and
// copied here from main. It stays "dev" for source builds (go run / go install),
// which disables the update check — there's no released version to compare to.
var Version = "dev"

// latestReleaseURL is the GitHub endpoint for the repo's newest non-prerelease tag.
const latestReleaseURL = "https://api.github.com/repos/jordan-simonovski/emoji-tools/releases/latest"

// updateCheckInterval throttles the network check: at most once per day.
const updateCheckInterval = 24 * time.Hour

// maybeNotifyUpdate prints a one-line upgrade nudge to stderr when a newer
// release exists. It is best-effort: it stays silent on any error, when run
// non-interactively (piped/CI), for dev builds, or when checked recently. Never
// blocks meaningfully — the HTTP call has a short timeout and runs at most daily.
// ponytail: synchronous with a 1.5s ceiling; move to a goroutine only if that
// pause is ever noticeable (it runs once a day, so it isn't).
func maybeNotifyUpdate() {
	if Version == "dev" || os.Getenv("CI") != "" || os.Getenv("EMOJI_TOOLS_NO_UPDATE_CHECK") != "" {
		return
	}
	if fi, err := os.Stderr.Stat(); err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return // not an interactive terminal; don't nag scripts/pipes
	}

	latest, ok := cachedOrFetchLatest()
	if !ok || !versionNewer(latest, Version) {
		return
	}
	fmt.Fprintf(os.Stderr,
		"\nemoji-tools %s is available (you have %s). Upgrade with:\n  brew upgrade --cask emoji-tools\n",
		latest, Version)
}

type updateCache struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

// cachedOrFetchLatest returns the latest release tag, reusing a cached value
// when it was fetched within updateCheckInterval and otherwise querying GitHub
// and refreshing the cache. The bool is false when no version could be resolved.
//
// The cache is stamped on both success AND failure: an offline or rate-limited
// user must be throttled too, otherwise every command would re-attempt the
// network and pay the request timeout. With no usable cache dir there's nothing
// to throttle against, so the check is skipped entirely rather than run per call.
func cachedOrFetchLatest() (string, bool) {
	path := cachePath()
	if path == "" {
		return "", false // can't throttle without a cache; don't hit the network every run
	}

	var cached updateCache
	if data, err := os.ReadFile(path); err == nil && json.Unmarshal(data, &cached) == nil {
		// d >= 0 guards a future CheckedAt (clock skew), which would otherwise
		// read as "always fresh" and suppress the check indefinitely.
		if d := time.Since(cached.CheckedAt); d >= 0 && d < updateCheckInterval {
			return cached.Latest, cached.Latest != ""
		}
	}

	latest, err := fetchLatestRelease()
	if err != nil {
		latest = cached.Latest // keep last-known tag so a transient failure still nudges
	}
	if data, mErr := json.Marshal(updateCache{CheckedAt: time.Now(), Latest: latest}); mErr == nil {
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		_ = os.WriteFile(path, data, 0o644) // stamps CheckedAt even on failure -> throttles retries
	}
	return latest, latest != ""
}

func cachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "emoji-tools", "update-check.json")
}

func fetchLatestRelease() (string, error) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	req, err := http.NewRequest(http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "emoji-tools/"+Version) // GitHub 403s an empty User-Agent
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github returned %s", resp.Status)
	}
	var body struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.TagName == "" {
		return "", fmt.Errorf("empty tag_name")
	}
	return body.TagName, nil
}

// versionNewer reports whether latest is a higher semver than current. Both may
// carry a leading "v" and a prerelease/build suffix, which are ignored — a plain
// major.minor.patch comparison is enough to decide whether to nudge.
func versionNewer(latest, current string) bool {
	l, c := parseSemver(latest), parseSemver(current)
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v, _, _ = strings.Cut(v, "-") // drop prerelease
	v, _, _ = strings.Cut(v, "+") // drop build metadata
	var out [3]int
	for i, part := range strings.SplitN(v, ".", 3) {
		if i >= 3 {
			break
		}
		out[i], _ = strconv.Atoi(part)
	}
	return out
}
