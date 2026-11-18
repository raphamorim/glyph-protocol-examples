# glyph-protocol-examples

Small demos that exercise [Glyph Protocol][blog]'s payload formats
(`vg`, `glyf`, `colrv0` and `colrv1`) across Rust, Go, TypeScript,
and Jam — plus an [ultraviolet](./ultraviolet) full-screen TUI
variant in Go.

Each demo detects support the way the spec says (§3.4): it sends `s`
followed by DA1 (`CSI c`) and reads until the DA1 reply. An `s` reply
before it lists the formats the terminal accepts; none means the
terminal doesn't implement Glyph Protocol, and the demo sends nothing
else. Every format is optional, so each one is registered only if the
terminal lists it. Then the demo prints:

- `colrv1 example` (twice, once over a blue background block): the
  string **"Rio Terminal Emulator"** rendered from `fmt=colrv1`
  payloads extracted from Typearture's [Nabla][nabla] isometric
  COLRv1 typeface.
- `colrv0 example`: the string **"Using Glyph protocol"** rendered
  from `fmt=colrv0` payloads extracted from Antonio J. Morata's
  [Zupra 3D eYe/FS][zupra] layered COLRv0 typeface.
- `glyf examples`: five Nerd Font outlines (`branch`, `folder`,
  `home`, `heart`, Rust cog) registered at empty PUA-B codepoints
  `U+100000`–`U+100004` as `fmt=glyf` payloads, extracted from
  `JetBrainsMono Nerd Font` so the icons look like Nerd Fonts even
  without a Nerd Font installed.
- `glyf wide ex.`: the same five outlines re-registered with
  `width=2`, so each icon is drawn across two cells. The codepoint
  still takes one column (its normal wcwidth), so cursor movement,
  wrapping and selection don't change; the demo prints a space after
  each icon for the second cell.
- `placement …` (three rows): the heart outline registered nine times
  with different [§8.5 sizing/placement][spec] parameters — four of
  the `size=` modes (`height`/`contain`/`cover`/`stretch`), corner and
  baseline `align=`, fractional `pad=` insets, and a two-cell
  `width=2;size=contain` combo. Each glyph prints over a grey
  background block so the cell box (and where the §8.5 transform put
  the outline inside it) is visible.
- `vg …` (four rows; ratatui, bubbletea and ink):
  [`fmt=vg`](#svg-icons-gradients-and-emoji-fmtvg), SVG path
  data and paint sent as text. Six icons; gradients, clips and a
  butt/miter stroke; then Noto emoji (gradients) and Twemoji emoji, each
  two cells wide. A terminal without `vg` gets
  `(vg not advertised by terminal)`.

Every registration except the single emoji declares its authored
extent (`aw`, `lh`, `ymin`, spec §8.5.1), so the terminal places it
from the font's metrics instead of the outline's own bounds:

- The Nerd Font icons use the font's cell, `aw=600;lh=1320;ymin=-300`,
  which puts them exactly where the patched font would and keeps the
  five in proportion to each other.
- Each Nabla and Zupra letter uses its own advance plus the font's
  line height and descender, with `size=height`. Every letter gets the
  same scale and they share one baseline.
- The placement rows use a square em around the heart,
  `aw=1000;lh=1000;ymin=-120`. A cell is taller than it is wide, so
  that's where the size modes visibly differ.

Missing formats degrade to a labelled fallback line (e.g. `(colrv0
not advertised by terminal)`) instead of failing the whole demo.
[Rio][rio] is the reference implementation of the protocol.

Background, rationale, and the full protocol spec live in the blog
posts:

- <https://rapha.land/introducing-glyph-protocol-for-terminals/> —
  original introduction (`fmt=glyf` only).
- <https://rapha.land/adding-color-glyphs-to-glyph-protocol/> —
  colour formats (`fmt=colrv0`, `fmt=colrv1`).

## Demos

| Directory | Language   | Run |
|-----------|------------|-----|
| [ratatui](./ratatui)   | Rust       | `cd ratatui && cargo run` |
| [bubbletea](./bubbletea) | Go         | `cd bubbletea && go run .` |
| [ink](./ink)       | TypeScript | `cd ink && npm install && npm run build && npm start` |
| [ncurses-jam](./ncurses-jam) | Jam | `cd ncurses-jam && jam run -lncurses glyph.jam` |
| [ultraviolet](./ultraviolet) | Go | `cd ultraviolet && go run .` |

The first four share the same wire behaviour, printed layout, and
payload constants — only the language differs, except that
`ncurses-jam` doesn't print the `vg` rows yet. The `ratatui` /
`bubbletea` / `ink` directory names are historical: those demos no
longer use the framework they're named after, since the point is the
protocol, not the TUI. The exception is `ultraviolet`, which *does*
drive [Charm's ultraviolet][uv] as a real alt-screen TUI rather than a
plain-CLI print. If you want to port the protocol to another language,
the wire helpers at the top of each entry file are the whole story
(APC sequence construction + base64 payloads).

## SVG icons, gradients and emoji (`fmt=vg`)

[`svg/`](./svg) shows `fmt=vg`, the recommended payload format since
spec v1.17 (§8.9): SVG's path data and paint model as plain text
instead of a base64 font outline, in layers of fills, strokes and
clips, painted with the foreground, a fixed colour or a gradient. It
isn't an SVG document. [raphamorim/rio#2095][path-pr] adds it to the
spec and implements it in Rio.

```bash
python3 svg/demo.py
```

The ratatui, bubbletea and ink demos print a few `vg` rows next to the
other formats; this one shows all of it. It asks the terminal for `fmt`
support and, when `vg` is listed, prints:

- `stroked`: Lucide and Tabler line icons (round strokes, 2 units wide).
- `filled`: Heroicons, Octicons, Bootstrap Icons and Material Symbols
  (Material's viewBox is `0 -960 960 960`).
- `duotone`: Phosphor duotone, a 20% foreground layer under the icon.
- `two colour`: three glyphs drawn for the demo that mix fixed colours
  with the foreground.
- `gradients`: a radial gradient with a focal point, a folder that fades
  out (a gradient of the foreground alone, so it still follows the text
  colour), and a gradient on a stroke. Two cells wide, like the rows
  below, so the details show.
- `clips`: stripes clipped to a circle, and half of a radial gradient.
- `strokes`: the same zigzag with round caps and joins, butt and miter,
  and square and bevel.
- `Noto`, `Twemoji`, `Fluent`, `OpenMoji`: 😀 😎 ❤️ 🚀 🎉 🔥 🍕 🐱 from each
  set. Noto's use gradients; OpenMoji's outlines are butt and miter
  strokes.
- `follows fg`: one registration printed in six SGR colours.
- `width=2` and `fit=`: `vg` places the viewBox like SVG's
  `preserveAspectRatio`, with `fit` (`contain`/`cover`/`stretch`) and
  `align` (`start`/`center`/`end` on each axis, no baseline).

Without `vg` it prints which formats the terminal has and stops.

[`svg/svg2glyph.py`](svg/svg2glyph.py) is the converter the demo uses,
standard library only. It turns an icon SVG into a registration:

```bash
python3 svg/svg2glyph.py svg/icons/lucide/folder.svg              # print the APC
python3 svg/svg2glyph.py svg/icons/lucide/folder.svg --cp 100005  # another codepoint
python3 svg/svg2glyph.py svg/icons/lucide/folder.svg --payload    # vb and payload only
```

It handles what icon sets ship: paths and basic shapes inside groups,
fills and strokes with their caps and joins, `currentColor` as the
foreground, opacity as alpha. It refuses transforms, `<use>`, clipping,
masks, gradients and dashes with a message instead of drawing something
different. The format has gradients and clips, but getting them out of
an arbitrary SVG means resolving CSS, transforms and gradient units,
which needs a full SVG library; emoji sets need that, icon sets don't.
The gradients and clips in the demo are written by hand.

For emoji, [`svg/usvg2vg`](svg/usvg2vg) is a small Rust converter built
on [usvg](https://github.com/linebender/resvg), which resolves all of
that. It writes the viewBox and the payload, and refuses what the format
can't carry (filters, masks, patterns, images, text, dashes, blend
modes, nested clips):

```bash
cd svg/usvg2vg
cargo run --release -- ../emoji/noto/rocket.svg                  # print vb and payload
cargo run --release -- ../emoji/noto/rocket.svg -o rocket.vg     # write them to a file
```

The emoji in [`svg/emoji`](svg/emoji) are converted with it; see
[`svg/emoji/README.md`](svg/emoji/README.md) for their sources and
licences.

The icons in [`svg/icons`](svg/icons) are unmodified copies from each
set, with its licence next to them: Lucide (ISC), Heroicons, Octicons,
Bootstrap Icons, Phosphor and Tabler (MIT), Material Symbols
(Apache-2.0).

## Extracting your own glyphs

The demo payloads come from one-off extraction scripts. All three
scripts share a virtualenv:

```bash
python3 -m venv .venv && . .venv/bin/activate
pip install fonttools lxml
```

**Monochrome (`fmt=glyf`).** Pull `glyf` simple-glyph bytes from a
TrueType font (Nerd Fonts, Powerline, etc.):

```bash
python3 extract_glyphs.py /path/to/font.ttf  # edit TARGETS to pick codepoints
```

Composite glyphs are flattened via `fontTools.pens.ttGlyphPen`,
hinting is stripped, output is base64 ready to paste. All three
scripts refuse an outline (or a colour container's outlines together)
that decodes to more than 64 KiB, the budget terminals enforce with
`reason=outline_too_large`.

When you register what you extracted, send the extent from the
font's metrics too: `aw` from `hmtx`, and `lh`/`ymin` from the
`hhea` ascender and descender. Without it the terminal sizes each
glyph from its own outline, so a set of icons or letters stops
lining up.

**Colour (`fmt=colrv1`).** Pull a full COLR v1 paint graph for a
single codepoint from a colour font. The script auto-subsets the
font to the minimal set of glyphs the paint graph touches, so the
container fits in the 1024-glyph / 64 KiB per-payload cap:

```bash
python3 extract_colrv1.py fonts/Nabla.ttf 52  # letter "R"
```

Output is one base64 string — the colrv1 container defined in the
[spec's §8.7](https://github.com/raphamorim/rio/blob/main/specs/glyph-protocol.md#87-payload-format-colrv1).

**Colour (`fmt=colrv0`).** Same container shape, carrying a COLR v0
table (flat layered colour — each layer is a full-alpha glyph with a
CPAL palette index). Works against any TrueType-outline COLRv0 font:

```bash
python3 extract_colrv0.py fonts/zupra-3d-eye-fs.colr.ttf 55  # letter "U"
```

**Bundled demo generators.** `build_nabla_rio.py` and
`build_zupra_using.py` chain the extractors across every unique letter
of their respective strings and emit language-specific modules. Pass
`--lang` to pick the output dialect; default is Rust to stdout:

```bash
python3 build_nabla_rio.py                 > ratatui/src/colorv2.rs
python3 build_nabla_rio.py --lang go       > bubbletea/nabla_rio.go
python3 build_nabla_rio.py --lang ts       > ink/source/nabla_rio.ts

python3 build_zupra_using.py               > ratatui/src/zupra_using.rs
python3 build_zupra_using.py --lang go     > bubbletea/zupra_using.go
python3 build_zupra_using.py --lang ts     > ink/source/zupra_using.ts

python3 build_vg.py                        > ratatui/src/vg.rs
python3 build_vg.py --lang go              > bubbletea/vg.go
python3 build_vg.py --lang ts              > ink/source/vg.ts
```

`build_vg.py` needs no virtualenv: it reads the icons through
`svg/svg2glyph.py`, the hand-drawn glyphs from `svg/demo.py` and the
converted emoji from `svg/emoji/`.

The ultraviolet demo uses copies of the Go modules
(`ultraviolet/nabla_rio.go`, `ultraviolet/zupra_using.go`); run the
`--lang go` commands into those paths too.

Re-run whichever ones you care about after swapping fonts or demo
strings.

## Vendored fonts

- [`fonts/Nabla.ttf`](fonts/Nabla.ttf) — Typearture's [Nabla][nabla]
  isometric COLRv1 typeface (OFL-1.1, see [`fonts/OFL.txt`](fonts/OFL.txt)),
  pulled from <https://fonts.google.com/specimen/Nabla?preview.script=Latn>
  via `google/fonts`.
- [`fonts/Nabla.subset.ttx`](fonts/Nabla.subset.ttx) — the two-glyph
  subset `nanoemoji` ships in its [test corpus][nanoemoji-fonts]
  (`#` + space). Tiny, convenient for sanity-testing the extractor
  without downloading the full 1.6 MB font.
- [`fonts/zupra-3d-eye-fs.colr.ttf`](fonts/zupra-3d-eye-fs.colr.ttf) —
  Antonio J. Morata's [Zupra 3D eYe/FS][zupra] layered COLRv0 display
  typeface. See [`fonts/zupra-3d-eye-fs.LICENSE.txt`](fonts/zupra-3d-eye-fs.LICENSE.txt)
  for its licence terms.

## License

Code: MIT, same as Rio. Nabla: [SIL Open Font License 1.1](fonts/OFL.txt).
The icons under `svg/icons` and the emoji under `svg/emoji` keep their
own licences, in each set's directory (see
[`svg/emoji/README.md`](svg/emoji/README.md)); the `vg` modules
generated from them (`ratatui/src/vg.rs`, `bubbletea/vg.go`,
`ink/source/vg.ts`) carry the same terms.

[blog]: https://rapha.land/introducing-glyph-protocol-for-terminals/
[rio]: https://raphamorim.io/rio
[spec]: https://github.com/raphamorim/rio/blob/main/specs/glyph-protocol.md
[uv]: https://github.com/charmbracelet/ultraviolet
[nabla]: https://fonts.google.com/specimen/Nabla?preview.script=Latn
[nanoemoji-fonts]: https://github.com/googlefonts/nanoemoji/tree/main/tests/fonts
[zupra]: https://www.dafont.com/zupra-3d-eye-fs.font
[path-pr]: https://github.com/raphamorim/rio/pull/2095
