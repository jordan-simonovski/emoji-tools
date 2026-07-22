# emoji-tools

Turn any logo or image into Slack emojis. One binary, twelve makers:

| Command | What it makes |
|---|---|
| `grid` | Splits an image into an N×M grid of 128px tiles that reassemble into one big logo when placed together. |
| `scroll` | A seamless horizontal-scroll looping GIF — repeat the emoji in a row and it reads as one continuous moving stream. |
| `intensify` | The classic shaking `:x-intensifies:` animated GIF. |
| `yells-at` | Abe Simpson yelling at your image. |
| `uwu` | Overlays a uwu face onto your image. |
| `petpet` | A hand patting your image (the petpet meme). |
| `panic` | A frantic zoom-pulse + shake animation. |
| `party` | Cycles the image through party-parrot rainbow colours. |
| `party-blob` | The rainbow colour-cycle plus a bouncing squash-and-stretch wobble. |
| `spin` | A spinning-coin 3D flip around the vertical axis. |
| `confetti` | Rains confetti over your image (animated overlay). |
| `bongocat` | Bongo cat drumming on your image, overlaid so the cat sits in front. |

Every output is a **square** image within Slack's limits.

Inputs can be **SVG** (rendered with a pure-Go rasterizer — no external tools) or raster **PNG / JPEG / GIF / WebP**.

## Install

```sh
# Prebuilt binaries: see the Releases page, or Homebrew:
brew install jordan-simonovski/tap/emoji-tools

# Or with Go:
go install github.com/jordan-simonovski/emoji-tools@latest
```

`emoji-tools version` prints the build version. Released builds check GitHub for a
newer release at most once a day and, if one exists, print a one-line upgrade nudge.
The check is skipped for `go install`/source builds, in CI, and when output isn't a
terminal; set `EMOJI_TOOLS_NO_UPDATE_CHECK=1` to disable it entirely.

## Usage

```sh
emoji-tools <command> [flags] <input>
```

Flags may go before or after the input file.

### grid

```sh
emoji-tools grid clickhouse-logo.svg
emoji-tools grid banner.webp -cols 4 -rows 1        # single wide row
emoji-tools grid logo.svg -flip -name hc            # mirror horizontally
```

Writes `<name>_r<row>_c<col>.png` tiles plus a `_preview.png` into `<name>-grid/`, and
prints the paste-ready `:name_r_c:` block (one line per row). Upload each tile as a
custom emoji named to match, then paste the block — each row on its own line, no spaces.

### scroll

```sh
emoji-tools scroll hyperdx.svg -name hdx_scroll
emoji-tools scroll logo.png -dur 30 -logo 72        # faster, more gap between copies
emoji-tools scroll logo.png -direction right
```

Sized for Slack animated emoji (112×112 by default). Paste a row of the same emoji for
the moving-stream effect. `-frames` must divide `-tile` evenly so the loop is seamless.

### intensify

```sh
emoji-tools intensify new-pull-request.png          # -> new-pull-request_intensifies.gif
emoji-tools intensify logo.svg -shake 8 -dur 30
```

### yells-at

```sh
emoji-tools yells-at clickhouse-logo.svg            # -> old-man-yells-at-clickhouse-logo.png
```

### uwu

```sh
emoji-tools uwu hyperdx.svg                          # -> uwu-hyperdx.png (128x128)
emoji-tools uwu logo.png -scale 0.9 -y -0.05         # bigger face, nudged up
emoji-tools uwu logo.png -size 96                    # smaller square output
```

`yells-at` and `uwu` produce a square PNG (default 128px, `-size` up to 256), letterboxing
the composite onto a transparent canvas so it satisfies Slack's square requirement.

### petpet

```sh
emoji-tools petpet hyperdx.svg                       # -> pet_hyperdx.gif
emoji-tools petpet avatar.png -dur 40                # faster patting
```

### panic

```sh
emoji-tools panic hyperdx.svg                        # -> hyperdx_panic.gif
emoji-tools panic logo.png -min 0.3 -jitter 0.2      # smaller start, more shake
```

### party / party-blob

```sh
emoji-tools party clickhouse-logo.svg               # -> clickhouse-logo_party.gif
emoji-tools party-blob kat.png                       # rainbow cycle + wobble
emoji-tools party-blob kat.png -amp 0.2 -dur 50      # bigger, faster bounce
```

`party` cycles the hue through the rainbow in place; `party-blob` adds a bouncing
squash-and-stretch on top.

### spin

```sh
emoji-tools spin coin.png                            # -> coin_spin.gif
emoji-tools spin logo.svg -frames 16 -dur 45         # smoother, faster spin
```

A 3D coin-flip around the vertical axis: the image thins to an edge and shows its
mirror on the way round.

### confetti

```sh
emoji-tools confetti clickhouse-logo.svg            # -> clickhouse-logo_confetti.gif
emoji-tools confetti logo.png -frames 30            # fewer frames (smaller file)
```

Composites a falling-confetti animation over your image. The overlay is subsampled
to `-frames` (default 50) to stay within Slack's 50-frame cap.

### bongocat

```sh
emoji-tools bongocat clickhouse-logo.svg            # -> bongo_clickhouse-logo.gif
emoji-tools bongocat logo.png -scale 0.5 -dur 90    # smaller drum, faster drumming
```

Bongo cat drumming on your image: your image is the drum, and the cat is overlaid
on top so its body cuts off the top of the image — it reads as the cat in front.

## Slack emoji limits

Slack custom emoji must be **square**, **under 128 KB**, and animated GIFs are capped at
**50 frames**. Every command emits a square image; animated commands default to 112–128px
with ≤14 frames, and the `-tile`/`-size` flags are capped at 256px.

## Credits

- `yells-at` uses the Abe Simpson artwork from
  [oncilla/old-man-yells-at](https://github.com/oncilla/old-man-yells-at) (Apache-2.0).
- `intensify` is a Go port of the `intensify.sh` idea by
  [doctaphred](https://gist.github.com/doctaphred/f30716e341aaa0673294639093a0632a).
- `petpet` uses the hand-frame artwork from
  [aDu/pet-pet-gif](https://github.com/aDu/pet-pet-gif).
- `bongocat` uses the "bongo cat" meme (original cat art by StrayRogue,
  animation by DitzyFlama).
