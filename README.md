# glyph-protocol-examples

Small demos that exercise [Glyph Protocol][blog]'s three payload
formats across Rust, Go, TypeScript, and Jam — plus an
[ultraviolet](./ultraviolet) full-screen TUI variant in Go.

Each demo probes the terminal for `fmt` support, then prints:

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

Format support is negotiated per-session via the protocol's `s` verb.
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
payload constants — only the language differs. The `ratatui` /
`bubbletea` / `ink` directory names are historical: those demos no
longer use the framework they're named after, since the point is the
protocol, not the TUI. The exception is `ultraviolet`, which *does*
drive [Charm's ultraviolet][uv] as a real alt-screen TUI rather than a
plain-CLI print. If you want to port the protocol to another language,
the wire helpers at the top of each entry file are the whole story
(APC sequence construction + base64 payloads).

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
```

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

[blog]: https://rapha.land/introducing-glyph-protocol-for-terminals/
[rio]: https://raphamorim.io/rio
[spec]: https://github.com/raphamorim/rio/blob/main/specs/glyph-protocol.md
[uv]: https://github.com/charmbracelet/ultraviolet
[nabla]: https://fonts.google.com/specimen/Nabla?preview.script=Latn
[nanoemoji-fonts]: https://github.com/googlefonts/nanoemoji/tree/main/tests/fonts
[zupra]: https://www.dafont.com/zupra-3d-eye-fs.font
