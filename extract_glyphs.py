#!/usr/bin/env python3
"""
Extract `glyf` simple-glyph byte records from a TrueType font (Nerd
Font or otherwise) for a list of codepoints. Prints the records as
base64 strings ready to paste into the demo apps.

Usage:
    python3 extract_glyphs.py <font.ttf>

Edit TARGETS below to choose which codepoints you want. The script:
  1. Opens the font and looks up each codepoint's glyph name via cmap.
  2. If the glyph is composite, flattens it to simple via fontTools'
     TTGlyphPen.
  3. Strips hinting instructions (Glyph Protocol v1 rejects them).
  4. Prints base64 of the raw simple-glyph bytes.
"""

import base64
import sys

from fontTools.pens.ttGlyphPen import TTGlyphPen
from fontTools.ttLib import TTFont
from fontTools.ttLib.tables.ttProgram import Program

# Edit to taste. (codepoint, label) pairs.
TARGETS = [
    (0xE0A0, "branch"),  # Powerline branch
    (0xF07B, "folder"),  # fa-folder
    (0xF015, "home"),    # fa-home
    (0xF004, "heart"),   # fa-heart
    (0xE7A8, "rust"),    # dev-rust
]


def extract(font: TTFont, cp: int) -> bytes:
    cmap = font.getBestCmap()
    glyph_name = cmap.get(cp)
    if glyph_name is None:
        raise SystemExit(f"codepoint U+{cp:04X} not in cmap")
    glyf = font["glyf"]
    glyph = glyf[glyph_name]

    # Flatten composites by re-drawing into a simple glyph via the
    # pen. TTGlyphPen produces a pure on-curve/off-curve outline.
    if glyph.isComposite():
        glyph_set = font.getGlyphSet()
        pen = TTGlyphPen(glyph_set)
        glyph_set[glyph_name].draw(pen)
        glyph = pen.glyph()

    # `compile` insists `program` exists; strip hinting by replacing
    # with an empty program (no bytecode).
    glyph.program = Program()
    glyph.program.fromBytecode(b"")

    return glyph.compile(glyf)


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: extract_glyphs.py <font.ttf>")
    font_path = sys.argv[1]

    font = TTFont(font_path)
    upm = font["head"].unitsPerEm
    print(f"// Extracted from {font_path}")
    print(f"// units per em: {upm}")
    print()
    for cp, label in TARGETS:
        data = extract(font, cp)
        b64 = base64.b64encode(data).decode("ascii")
        print(f"// {label} (U+{cp:04X}) — {len(data)} bytes")
        print(f'"{b64}",')
        print()


if __name__ == "__main__":
    main()
