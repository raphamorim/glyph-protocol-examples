# glyph-protocol-examples

Three small demo apps that register real Nerd Font outlines through
[Glyph Protocol][blog] and render them at empty Supplementary PUA-B
codepoints (`U+100000`–`U+100004`). The `glyf` payloads are extracted
from `JetBrainsMono Nerd Font` via `fontTools`, so the icons look like
Nerd Fonts even on systems with no Nerd Font installed.

Each demo:

- Registers `branch`, `folder`, `home`, `heart`, and a Rust cog at
  startup.
- Shows a mock prompt line with those icons inline.
- Lets you re-register (`↑`) or clear (`↓`) the glossary from the UI.
- Needs a Glyph Protocol v1 terminal to render correctly. [Rio][rio]
  is the reference implementation.

Background, rationale, and the full protocol spec live in the blog
post: <https://rapha.land/introducing-glyph-protocol-for-terminals/>.

## Demos

| Framework | Language   | Run |
|-----------|------------|-----|
| [ratatui](./ratatui)   | Rust       | `cd ratatui && cargo run` |
| [bubbletea](./bubbletea) | Go (bubbletea v2) | `cd bubbletea && go run .` |
| [ink](./ink)       | TypeScript (ink 5)   | `cd ink && npm install && npm run build && npm start` |

All three share the same wire behaviour — only the UI framework
differs. If you want to port the protocol to another framework, the
wire helpers at the top of each `main.*` file are the whole story
(APC sequence construction + base64 glyf payloads).

## Extracting your own glyphs

The demo payloads come from a one-off extraction script. To pull
`glyf` bytes for different Nerd Font codepoints:

```bash
python3 -m venv .venv && . .venv/bin/activate
pip install fonttools
python3 extract_glyphs.py /path/to/font.ttf  # edit TARGETS to pick codepoints
```

The script opens a TTF, flattens composite glyphs to simple ones via
`fontTools.pens.ttGlyphPen.TTGlyphPen`, strips hinting, and prints
base64 strings ready to paste into the demos.

## License

MIT, same as Rio.

[blog]: https://rapha.land/introducing-glyph-protocol-for-terminals/
[rio]: https://raphamorim.io/rio
