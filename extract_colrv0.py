#!/usr/bin/env python3
"""
Extract an `fmt=colrv0` payload for a single codepoint from a COLR v0
colour font.

The output is a base64 string ready to paste into a demo app as the
payload of an `ESC_25a1;r;cp=<hex>;fmt=colrv0;<base64>ESC\\` register.

Usage:
    python3 extract_colrv0.py <font.ttf|font.otf|font.ttx> <codepoint-hex>

Example:
    python3 extract_colrv0.py fonts/zupra-3d-eye-fs.colr.ttf 55  # letter U

COLR v0's wire container is identical to colrv1's — see Glyph Protocol
spec §8.7 — the only difference is the COLR table carried inside:

    u16     n_glyphs
    per glyph:
      u16   glyf_len
      glyf_len bytes     # simple-glyph record, §8.2 subset
    u16     colr_len
    colr_len bytes       # OpenType COLR v0 table
    u16     cpal_len
    cpal_len bytes       # OpenType CPAL table

A COLR v0 font declares a base glyph plus a flat list of layer glyphs
each painted with a CPAL palette index. `paletteIndex == 0xFFFF` means
"use the current foreground colour" per the OpenType spec — so v0 can
express both fully-painted icons and mixed glyphs that pick up the
terminal's foreground colour for some layers.
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


def load_font(path: Path) -> TTFont:
    if path.suffix == ".ttx":
        font = TTFont()
        font.importXML(str(path))
        return font
    return TTFont(str(path))


def compile_glyph_bytes(font: TTFont, glyph_name: str) -> bytes:
    """Compile a single simple-glyph record, stripping hinting and
    decomposing composites. Identical to the helper used for colrv1 —
    see extract_colrv1.py for the decomposition-via-recorder trick."""
    glyf = font["glyf"]
    g = glyf[glyph_name]
    if g.isComposite():
        glyph_set = font.getGlyphSet()
        recorder = DecomposingRecordingPen(glyph_set)
        glyph_set[glyph_name].draw(recorder)
        tt_pen = TTGlyphPen(None)
        recorder.replay(tt_pen)
        g = tt_pen.glyph()
    g.program = Program()
    g.program.fromBytecode(b"")
    return g.compile(glyf)


def subset_to_codepoint(font: TTFont, cp: int) -> TTFont:
    opts = Options()
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
    sub = Subsetter(options=opts)
    sub.populate(unicodes=[cp])
    sub.subset(font)
    return font


def extract(font: TTFont, cp: int) -> bytes:
    if "COLR" not in font:
        raise SystemExit("font has no COLR table")
    colr_version = font["COLR"].version
    if colr_version != 0:
        raise SystemExit(
            f"font COLR is v{colr_version}; this script produces fmt=colrv0. "
            "Use extract_colrv1.py for v1 fonts."
        )
    if "glyf" not in font:
        raise SystemExit(
            "font has no glyf table (likely CFF/OTF). Glyph Protocol's "
            "colrv0 container carries simple-glyph outlines; convert the "
            "font to TrueType outlines first."
        )

    subset_to_codepoint(font, cp)

    cmap = font.getBestCmap()
    glyph_name = cmap.get(cp)
    if glyph_name is None:
        raise SystemExit(
            f"codepoint U+{cp:X} not in cmap after subsetting — "
            "font may not cover it"
        )
    base_gid = font.getGlyphID(glyph_name)

    order = font.getGlyphOrder()
    if len(order) > 256:
        raise SystemExit(
            f"after subset the container would carry {len(order)} glyphs, "
            "above the 256 per-payload cap."
        )

    glyph_bytes = [compile_glyph_bytes(font, name) for name in order]

    colr_bytes = font.getTableData("COLR")
    cpal_bytes = font.getTableData("CPAL") if "CPAL" in font else b""

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

    sys.stderr.write(
        f"note: base glyph is {glyph_name!r} (GID {base_gid}); "
        f"container carries {len(glyph_bytes)} glyph outlines, "
        f"COLR={len(colr_bytes)} B, CPAL={len(cpal_bytes)} B\n"
    )
    return bytes(out)


def main() -> None:
    if len(sys.argv) != 3:
        raise SystemExit("usage: extract_colrv0.py <font.ttf|font.otf|font.ttx> <codepoint-hex>")
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
