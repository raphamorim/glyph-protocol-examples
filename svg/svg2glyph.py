#!/usr/bin/env python3
"""Turn an icon SVG into a Glyph Protocol `fmt=vg` registration.

    python3 svg2glyph.py icon.svg                 print the registration
    python3 svg2glyph.py icon.svg --cp 100005     ...at another codepoint
    python3 svg2glyph.py icon.svg --payload       print vb and payload only

Covers what icon sets ship: <path>, <circle>, <ellipse>, <rect>, <line>,
<polyline> and <polygon> inside <g>s; fill (nonzero or evenodd) and
stroke, with its caps, joins and miter limit; `currentColor` (or no
colour at all) as the cell's foreground; `opacity`, `fill-opacity` and
`stroke-opacity` as alpha; inline `style="..."` declarations for those
same properties.

Anything outside that is refused with a message rather than drawn
wrong: transforms, <use>, clipping, masks, gradients and dashes.
`fmt=vg` itself has gradients and clips, but getting them out of an
arbitrary SVG means resolving CSS, transforms and gradient units, which
is a job for a full SVG library (usvg, for one) rather than this
script. Emoji sets need that; icon sets don't.

Standard library only.
"""

import argparse
import re
import sys
import xml.etree.ElementTree as ET

SVG_NS = "{http://www.w3.org/2000/svg}"
NUMBER = re.compile(r"[+-]?(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?")

# Limits from the draft spec (Rio PR #2095).
MAX_LAYERS = 128
MAX_PAYLOAD = 64 * 1024

PAINT_PROPS = (
    "fill", "stroke", "stroke-width", "fill-rule", "opacity", "fill-opacity",
    "stroke-opacity", "stroke-linecap", "stroke-linejoin",
    "stroke-miterlimit", "stroke-dasharray", "display", "visibility",
)
NAMED = {"black": "000000", "white": "ffffff", "red": "ff0000",
         "green": "008000", "blue": "0000ff"}


class Unsupported(Exception):
    pass


def num(v):
    s = f"{float(v):.3f}".rstrip("0").rstrip(".")
    return "0" if s in ("", "-0") else s


def length(el, key, default=0.0):
    raw = el.get(key)
    if raw is None:
        return default
    raw = raw.strip()
    if raw.endswith("px"):
        raw = raw[:-2]
    if raw.endswith("%"):
        raise Unsupported(f"<{tag(el)} {key}={raw}>: percentages")
    return float(raw)


def tag(el):
    return el.tag.replace(SVG_NS, "")


def absolute_start(d):
    """Path data whose first moveto is absolute, so it can follow other
    path data. A leading `m` is absolute anyway, but its implicit pairs
    are relative linetos: `m4 17 6-6` becomes `M4 17l6-6`."""
    d = d.strip()
    if not d.startswith("m"):
        return d
    numbers = list(NUMBER.finditer(d, 1))
    if len(numbers) < 2:
        return d
    x, y = numbers[0], numbers[1]
    rest = d[y.end():].lstrip(" ,")
    implicit = bool(rest) and (rest[0].isdigit() or rest[0] in "+-.")
    return f"M{x.group()} {y.group()}" + ("l" if implicit else "") + rest


def shape_to_d(el):
    t = tag(el)
    if t == "path":
        return el.get("d", "")
    if t in ("circle", "ellipse"):
        cx, cy = length(el, "cx"), length(el, "cy")
        rx = length(el, "r") if t == "circle" else length(el, "rx")
        ry = length(el, "r") if t == "circle" else length(el, "ry")
        if rx <= 0 or ry <= 0:
            return None
        return (f"M{num(cx - rx)} {num(cy)}a{num(rx)} {num(ry)} 0 1 0 {num(2 * rx)} 0"
                f"a{num(rx)} {num(ry)} 0 1 0 {num(-2 * rx)} 0Z")
    if t == "rect":
        x, y = length(el, "x"), length(el, "y")
        w, h = length(el, "width"), length(el, "height")
        if w <= 0 or h <= 0:
            return None
        rx_raw, ry_raw = el.get("rx"), el.get("ry")
        rx = length(el, "rx") if rx_raw is not None else (length(el, "ry") if ry_raw is not None else 0)
        ry = length(el, "ry") if ry_raw is not None else rx
        rx, ry = min(rx, w / 2), min(ry, h / 2)
        if not rx or not ry:
            return f"M{num(x)} {num(y)}h{num(w)}v{num(h)}h{num(-w)}Z"
        return (f"M{num(x + rx)} {num(y)}h{num(w - 2 * rx)}"
                f"a{num(rx)} {num(ry)} 0 0 1 {num(rx)} {num(ry)}v{num(h - 2 * ry)}"
                f"a{num(rx)} {num(ry)} 0 0 1 {num(-rx)} {num(ry)}h{num(-(w - 2 * rx))}"
                f"a{num(rx)} {num(ry)} 0 0 1 {num(-rx)} {num(-ry)}v{num(-(h - 2 * ry))}"
                f"a{num(rx)} {num(ry)} 0 0 1 {num(rx)} {num(-ry)}Z")
    if t == "line":
        return (f"M{num(length(el, 'x1'))} {num(length(el, 'y1'))}"
                f"L{num(length(el, 'x2'))} {num(length(el, 'y2'))}")
    if t in ("polyline", "polygon"):
        pts = NUMBER.findall(el.get("points", ""))
        if len(pts) < 4:
            return None
        return "M" + " ".join(num(p) for p in pts) + ("Z" if t == "polygon" else "")
    return None


def colour(value, alpha):
    """`fg`, `fg@a` or `#rrggbbaa` for an SVG paint value."""
    value = (value or "currentColor").strip()
    if value == "currentColor":
        return "fg" if alpha >= 1 else f"fg@{num(alpha)}"
    if value.startswith("url("):
        raise Unsupported(f"paint {value}: gradients and patterns need a full SVG library")
    hex_ = NAMED.get(value.lower())
    if hex_ is None:
        if not re.fullmatch(r"#[0-9a-fA-F]{3}|#[0-9a-fA-F]{6}", value):
            raise Unsupported(f"colour {value!r}: use currentColor or #rgb/#rrggbb")
        hex_ = value[1:]
        if len(hex_) == 3:
            hex_ = "".join(c * 2 for c in hex_)
    return f"#{hex_.lower()}{round(alpha * 255):02x}"


def stroke_paint(width, style):
    """`stroke=<w>[/<cap>[/<join>]]`. SVG's defaults are butt caps and
    miter joins with a limit of 4; the format's are round and round."""
    cap = style.get("stroke-linecap", "butt")
    join = style.get("stroke-linejoin", "miter")
    if cap not in ("butt", "round", "square"):
        raise Unsupported(f"stroke-linecap={cap}")
    if join in ("miter", "miter-clip", "arcs"):
        limit = float(style.get("stroke-miterlimit", 4))
        join = "miter" if limit == 4 else f"miter{num(max(limit, 1))}"
    elif join not in ("round", "bevel"):
        raise Unsupported(f"stroke-linejoin={join}")
    paint = f"stroke={num(width)}"
    if cap == "round" and join == "round":
        return paint
    return f"{paint}/{cap}" + ("" if join == "round" else f"/{join}")


def own_style(el):
    style = {k: el.get(k) for k in PAINT_PROPS if el.get(k) is not None}
    for decl in (el.get("style") or "").split(";"):
        if ":" in decl:
            k, v = (s.strip() for s in decl.split(":", 1))
            if k in PAINT_PROPS:
                style[k] = v
    return style


def convert(path):
    root = ET.parse(path).getroot()
    vb = root.get("viewBox")
    if vb is None:
        vb = f"0 0 {num(length(root, 'width', 24))} {num(length(root, 'height', 24))}"
    vb = ",".join(num(v) for v in NUMBER.findall(vb))

    # (paint, colour, path data) in drawing order.
    layers = []

    def walk(el, inherited, group_alpha):
        t = tag(el)
        if t in ("use", "clipPath", "mask", "linearGradient", "radialGradient",
                 "pattern", "filter", "image", "text"):
            raise Unsupported(f"<{t}>")
        if t in ("defs", "title", "desc", "metadata"):
            return
        for key in ("transform", "clip-path", "mask", "filter"):
            if el.get(key):
                raise Unsupported(f"<{t} {key}=...>")
        style = dict(inherited)
        own = own_style(el)
        style.update({k: v for k, v in own.items() if k != "opacity"})
        alpha = group_alpha * float(own.get("opacity", 1))
        if style.get("display") == "none" or style.get("visibility") == "hidden":
            return

        d = shape_to_d(el)
        if d:
            d = " ".join(d.split())  # tabs and newlines aren't allowed
            fill = style.get("fill", "currentColor")
            if fill != "none":
                rule = "evenodd" if style.get("fill-rule") == "evenodd" else "nonzero"
                a = alpha * float(style.get("fill-opacity", 1))
                layers.append((f"fill={rule}", colour(fill, a), d))
            stroke = style.get("stroke", "none")
            if stroke != "none":
                if style.get("stroke-dasharray", "none") not in ("none", ""):
                    raise Unsupported("stroke-dasharray: dashes")
                width = float(style.get("stroke-width", 1))
                a = alpha * float(style.get("stroke-opacity", 1))
                layers.append((stroke_paint(width, style), colour(stroke, a), d))
        for child in el:
            walk(child, style, alpha)

    walk(root, {}, 1.0)
    if not layers:
        raise Unsupported("nothing to draw")

    # Consecutive opaque strokes with the same paint draw the same as one
    # stroke of all their subpaths, and icon sets split one icon into many
    # <path>s. Fills aren't merged: one fill of overlapping subpaths can
    # cancel where two separate fills wouldn't.
    merged = []
    for paint, col, d in layers:
        opaque = col == "fg" or (col.startswith("#") and col.endswith("ff"))
        if (merged and paint.startswith("stroke=") and opaque
                and merged[-1][0] == paint and merged[-1][1] == col):
            merged[-1][2] += absolute_start(d)
        else:
            merged.append([paint, col, absolute_start(d)])

    if len(merged) > MAX_LAYERS:
        raise Unsupported(f"{len(merged)} layers, the limit is {MAX_LAYERS}")
    payload = "|".join(f"{p},{c}:{d}" for p, c, d in merged)
    if ";" in payload or len(payload) > MAX_PAYLOAD:
        raise Unsupported("payload too large or contains ';'")
    return vb, payload


def registration(cp, vb, payload, extra=""):
    extra = f";{extra}" if extra else ""
    return f"\x1b_25a1;r;cp={cp:x};fmt=vg;vb={vb};reply=0{extra};{payload}\x1b\\"


def main():
    ap = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    ap.add_argument("svg")
    ap.add_argument("--cp", default="100000", help="codepoint, hex (default 100000)")
    ap.add_argument("--payload", action="store_true", help="print vb and payload only")
    args = ap.parse_args()
    try:
        vb, payload = convert(args.svg)
    except Unsupported as e:
        sys.exit(f"{args.svg}: {e}")
    if args.payload:
        print(vb)
        print(payload)
    else:
        sys.stdout.write(registration(int(args.cp, 16), vb, payload))


if __name__ == "__main__":
    main()
