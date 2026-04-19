# glyph-protocol-examples

Three small plain-CLI demos that exercise [Glyph Protocol][blog]'s
three payload formats in Rust, Go, and TypeScript.

Each demo probes the terminal for `fmt` support, then prints four
lines:

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

All three share the same wire behaviour, printed layout, and payload
constants — only the language differs. The directory names are
historical: the demos no longer use ratatui / bubbletea / ink, since
the point is the protocol, not the TUI framework. If you want to port
it to another language, the wire helpers at the top of each entry file
are the whole story (APC sequence construction + base64 payloads).

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
hinting is stripped, output is base64 ready to paste.

**Colour (`fmt=colrv1`).** Pull a full COLR v1 paint graph for a
single codepoint from a colour font. The script auto-subsets the
font to the minimal set of glyphs the paint graph touches, so the
container fits in the 256-glyph / 64 KiB per-payload cap:

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
[nabla]: https://fonts.google.com/specimen/Nabla?preview.script=Latn
[nanoemoji-fonts]: https://github.com/googlefonts/nanoemoji/tree/main/tests/fonts
[zupra]: https://www.dafont.com/zupra-3d-eye-fs.font
