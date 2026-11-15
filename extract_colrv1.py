#!/usr/bin/env python3
"""
Extract an `fmt=colrv1` payload for a single codepoint from a
colour font (e.g. the vendored `fonts/Nabla.subset.ttx`).

The output is a base64 string ready to paste into a demo app as the
payload of an `ESC_25a1;r;cp=<hex>;fmt=colrv1;<base64>ESC\\` register.

Usage:
    python3 extract_colrv1.py <font.ttf|font.ttx> <codepoint-hex>

Example:
    python3 extract_colrv1.py fonts/Nabla.ttf 52  # letter R

The script auto-subsets the font to just the glyphs in the paint
graph for `cp` so the container fits inside the 1024-glyph cap and
64 KiB per-payload limit of Glyph Protocol. Subsetting is done via
`fontTools.subset` using `text=<char>`, which transitively pulls
all layer glyphs referenced from the base glyph's COLR paint.

The container packaged here matches Glyph Protocol spec §8.7:

    u16     n_glyphs
    per glyph:
      u16   glyf_len
      glyf_len bytes     # simple-glyph record, §8.2 subset
    u16     colr_len
    colr_len bytes       # OpenType COLR v1 table
    u16     cpal_len
    cpal_len bytes       # OpenType CPAL table (may be 0 for v1)

GIDs inside the COLR table index into our `glyphs[]` array, so we
ship the font's glyphs in GID order without remapping. That keeps
the extraction trivial at the cost of shipping any unreferenced
glyphs the font happens to carry. For the Nabla subset font this
is a handful of KB, comfortably under the 64 KiB per-payload cap.
"""

import base64
import struct
import sys
from pathlib import Path

from fontTools.pens.recordingPen import DecomposingRecordingPen
from fontTools.pens.ttGlyphPen import TTGlyphPen
from fontTools.subset import Options, Subsetter
from fontTools.ttLib import TTFont
from fontTools.ttLib.tables.ttProgram import Program

from extract_glyphs import check_decoded_budget


def load_font(path: Path) -> TTFont:
    """Load a .ttf directly, or compile a .ttx on the fly via
    fontTools' XML importer. We avoid writing a temp .ttf to disk."""
    if path.suffix == ".ttx":
        font = TTFont()
        font.importXML(str(path))
        return font
    return TTFont(str(path))


def compile_glyph_bytes(font: TTFont, glyph_name: str) -> bytes:
    """Compile a single simple-glyph record, stripping hinting to
    satisfy Glyph Protocol's §8.2 constraint. Composite glyphs are
    flattened to simple outlines.

    Note on the decomposition route: `TTGlyphPen(glyph_set)` is
    clever — it preserves components via `addComponent` when a
    `glyphSet` is available, so a composite glyph passed through it
    round-trips as a composite, which Glyph Protocol rejects. To
    force decomposition we first record the composite's outline into
    a `DecomposingRecordingPen` (which turns every `addComponent`
    into the equivalent raw `moveTo`/`lineTo`/`quadTo` stream) and
    then replay that stream into a fresh `TTGlyphPen(None)`, which
    has no glyphSet and therefore has no choice but to emit a simple
    glyph."""
    glyf = font["glyf"]
    g = glyf[glyph_name]
    if g.isComposite():
        glyph_set = font.getGlyphSet()
        recorder = DecomposingRecordingPen(glyph_set)
        glyph_set[glyph_name].draw(recorder)
        tt_pen = TTGlyphPen(None)
        recorder.replay(tt_pen)
        g = tt_pen.glyph()
    # Replace hinting bytecode with an empty program so §8.2's
    # `instructionLength == 0` constraint holds.
    g.program = Program()
    g.program.fromBytecode(b"")
    return g.compile(glyf)


def subset_to_codepoint(font: TTFont, cp: int) -> TTFont:
    """In-place-subset the font so only the glyphs reachable from the
    paint graph rooted at `cp` survive. fontTools' subsetter walks
    COLR references transitively, so the result is the minimal set
    of outlines needed to render this one codepoint."""
    opts = Options()
    # Drop tables the protocol doesn't consume. Keeping them bloats
    # the COLR/CPAL payload with no benefit — the terminal looks at
    # COLR and CPAL and asks for glyf outlines by GID; everything
    # else (layout tables, hinting, names) is ballast.
    opts.drop_tables = [
        "GPOS",
        "GSUB",
        "GDEF",
        "name",
        "post",
        "STAT",
        "fvar",
        "gvar",
        "HVAR",
        "MVAR",
    ]
    # Preserve COLR + CPAL + glyf references — fontTools recognises
    # COLR v1 paint graphs and pulls in every referenced layer glyph
    # automatically when `populate(unicodes=[...])` is called.
    sub = Subsetter(options=opts)
    sub.populate(unicodes=[cp])
    sub.subset(font)
    return font


def extract(font: TTFont, cp: int) -> bytes:
    """Build the colrv1 container for `cp`. The font is subsetted in
    place to only the glyphs reachable from `cp`'s paint graph, then
    its COLR/CPAL tables are compiled and packed alongside the
    remaining glyph outlines in GID order."""
    if "COLR" not in font:
        raise SystemExit("font has no COLR table")
    colr_version = font["COLR"].version
    if colr_version != 1:
        raise SystemExit(
            f"font COLR is v{colr_version}; this script produces fmt=colrv1. "
            "Use extract_colrv0.py for v0 fonts."
        )

    subset_to_codepoint(font, cp)

    # After subsetting the font's cmap holds only our codepoint.
    cmap = font.getBestCmap()
    glyph_name = cmap.get(cp)
    if glyph_name is None:
        raise SystemExit(
            f"codepoint U+{cp:X} not in cmap after subsetting — "
            "font may not cover it"
        )
    base_gid = font.getGlyphID(glyph_name)

    order = font.getGlyphOrder()
    if len(order) > 1024:
        raise SystemExit(
            f"after subset the container would carry {len(order)} glyphs, "
            "above the 1024 per-payload cap. The COLR paint graph for this "
            "codepoint is unusually deep."
        )

    # Compile each glyph's bytes in GID order. The COLR table still
    # references them by GID so no remapping is needed.
    glyph_bytes = [compile_glyph_bytes(font, name) for name in order]
    check_decoded_budget(glyph_bytes, f"U+{cp:04X}")

    # Ask fontTools for the compiled table bytes. `getTableData` gives
    # us the raw on-disk bytes without us having to assemble a full
    # sfnt directory.
    colr_bytes = font.getTableData("COLR")
    cpal_bytes = font.getTableData("CPAL") if "CPAL" in font else b""

    # Pack the container.
    out = bytearray()
    out.extend(struct.pack(">H", len(glyph_bytes)))
    for data in glyph_bytes:
        if len(data) > 0xFFFF:
            raise SystemExit(
                f"a glyph compiled to {len(data)} bytes, above the "
                "container's 16-bit per-glyph length field."
            )
        out.extend(struct.pack(">H", len(data)))
        out.extend(data)
    out.extend(struct.pack(">H", len(colr_bytes)))
    out.extend(colr_bytes)
    out.extend(struct.pack(">H", len(cpal_bytes)))
    out.extend(cpal_bytes)

    # The base GID is part of the cmap/COLR — the terminal looks up
    # the BaseGlyphPaintRecord for this GlyphId at render time, so
    # clients don't need to surface it separately.
    sys.stderr.write(
        f"note: base glyph is {glyph_name!r} (GID {base_gid}); "
        f"container carries {len(glyph_bytes)} glyph outlines, "
        f"COLR={len(colr_bytes)} B, CPAL={len(cpal_bytes)} B\n"
    )
    return bytes(out)


def main() -> None:
    if len(sys.argv) != 3:
        raise SystemExit("usage: extract_colrv1.py <font.ttf|font.ttx> <codepoint-hex>")
    font_path = Path(sys.argv[1])
    cp = int(sys.argv[2], 16)

    font = load_font(font_path)
    upm = font["head"].unitsPerEm
    payload = extract(font, cp)
    b64 = base64.b64encode(payload).decode("ascii")

    sys.stderr.write(f"upm = {upm}, payload = {len(payload)} B, base64 = {len(b64)} B\n")
    print(b64)


if __name__ == "__main__":
    main()
