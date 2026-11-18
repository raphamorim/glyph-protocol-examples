#!/usr/bin/env python3
"""Glyph Protocol `fmt=vg` demo: SVG icons drawn by the terminal.

    python3 svg/demo.py

Probes the terminal with `s`, and if it lists `vg` in `fmt`, registers
icons from seven icon sets, glyphs drawn for the demo (fixed colours,
gradients, clips, stroke styles) and emoji from four sets, and prints
them. Standard library only; needs a Unix tty.
"""

import os
import select
import sys
import termios
import tty

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from svg2glyph import convert, registration  # noqa: E402

HERE = os.path.dirname(os.path.abspath(__file__))
ICONS = os.path.join(HERE, "icons")
EMOJI = os.path.join(HERE, "emoji")
EMOJI_SETS = [("Noto", "noto"), ("Twemoji", "twemoji"), ("Fluent", "fluent"),
              ("OpenMoji", "openmoji")]
EMOJI_NAMES = ["grinning", "sunglasses", "heart", "rocket", "party", "fire",
               "pizza", "cat"]

ROWS = [
    ("stroked", [
        "lucide/folder.svg", "lucide/git-branch.svg", "lucide/terminal.svg",
        "tabler/bug.svg", "tabler/cloud.svg", "tabler/terminal-2.svg",
    ]),
    ("filled", [
        "heroicons/home.svg", "heroicons/heart.svg", "heroicons/star.svg",
        "octicons/repo-24.svg", "octicons/git-branch-24.svg",
        "octicons/git-pull-request-24.svg",
        "bootstrap/folder-fill.svg", "bootstrap/gear-fill.svg",
        "bootstrap/lightning-fill.svg",
        "material/folder_24px.svg", "material/home_24px.svg",
        "material/terminal_24px.svg",
    ]),
    ("duotone", [
        "phosphor/folder-duotone.svg", "phosphor/heart-duotone.svg",
        "phosphor/terminal-window-duotone.svg",
    ]),
]

# Drawn for this demo: fixed colours next to the cell's foreground.
FOLDER = ("M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9"
          "A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z")
TWO_COLOUR = [
    # A folder in the foreground with a red badge.
    f"stroke=2,fg:{FOLDER}|fill=nonzero,#e5534bff:M15 5a4 4 0 1 0 8 0a4 4 0 1 0-8 0Z",
    # A green status dot with a white check.
    "fill=nonzero,#2da44eff:M2 12a10 10 0 1 0 20 0a10 10 0 1 0-20 0Z"
    "|stroke=2.5,#ffffffff:M7 12.5l3.5 3.5 6.5-7",
    # A warning sign: amber triangle, dark mark.
    "fill=nonzero,#d4a72cff:M12 2.5L22.5 21h-21Z"
    "|stroke=2,#1f2328ff:M12 9v5M12 17.5v.01",
]

CIRCLE = "M2 12a10 10 0 1 0 20 0a10 10 0 1 0-20 0Z"
GRADIENTS = [
    # A sun: radial gradient with its focal point up and to the left.
    "fill=nonzero,rad(9 9 0 12 12 11 / 0 #fff6a8 / 0.55 #ffb000 / 1 #ff5a00):" + CIRCLE,
    # A folder that fades out: a gradient of the foreground alone, so it
    # stays an alpha mask and follows the text colour.
    f"fill=nonzero,lin(0 3 0 21 / 0 fg / 1 fg@0.15):{FOLDER}",
    # Linear gradient on a stroke.
    "stroke=3/round/round,lin(2 12 22 12 / 0 #3b82f6 / 1 #e5534b):M3 17l5-10 4 8 4-8 5 10",
]
CLIPS = [
    # A striped ball: the stripes are clipped to the circle.
    "fill=nonzero,#f4c430:" + CIRCLE
    + "|clip=1,nonzero:" + CIRCLE
    + "|fill=nonzero,#e5534b:M0 4h24v4H0ZM0 12h24v4H0ZM0 20h24v4H0Z",
    # Half a sun: the gradient clipped to the left half.
    "clip=1,nonzero:M0 0h12v24H0Z|fill=nonzero,rad(12 12 0 12 12 10 / 0 #fff6a8 / 1 #ff5a00):" + CIRCLE,
]
ZIGZAG = "M3 18l4.5-9 4.5 9 4.5-9 4.5 9"
STROKES = [
    f"stroke=3,fg:{ZIGZAG}",                 # round caps, round joins
    f"stroke=3/butt/miter,fg:{ZIGZAG}",      # SVG's defaults
    f"stroke=3/square/bevel,fg:{ZIGZAG}",
]

FG_COLOURS = [31, 32, 33, 34, 35, 36]
SIZES = ["height", "contain", "cover", "stretch"]
GREY_BG = "\x1b[48;5;240m"
RESET = "\x1b[0m"


def probe_formats(timeout=1.0):
    """Send `s` and return the advertised `fmt` list, or None."""
    fd = sys.stdin.fileno()
    if not os.isatty(fd):
        return None
    saved = termios.tcgetattr(fd)
    try:
        tty.setcbreak(fd)
        sys.stdout.write("\x1b_25a1;s\x1b\\")
        sys.stdout.flush()
        reply = b""
        while not reply.endswith(b"\x1b\\"):
            ready, _, _ = select.select([fd], [], [], timeout)
            if not ready:
                break
            reply += os.read(fd, 1024)
    finally:
        termios.tcsetattr(fd, termios.TCSADRAIN, saved)
    reply = reply.decode("ascii", "replace")
    if "fmt=" not in reply:
        return None
    value = reply.split("fmt=", 1)[1].split(";", 1)[0].split("\x1b", 1)[0]
    return [f for f in value.split(",") if f]


def main():
    formats = probe_formats()
    if formats is None:
        print("no Glyph Protocol reply: run this in Rio")
        return
    if "vg" not in formats:
        print(f"the terminal advertises fmt={','.join(formats)}, without vg")
        print("fmt=vg is a proposal: build Rio from raphamorim/rio#2095")
        return

    out = []
    cp = 0x10_0000

    def register(vb, payload, extra=""):
        nonlocal cp
        out.append(registration(cp, vb, payload, extra))
        cp += 1
        return chr(cp - 1)

    lines = []
    for label, files in ROWS:
        glyphs = [register(*convert(os.path.join(ICONS, f))) for f in files]
        lines.append(f"{label:<12}" + " ".join(glyphs))

    glyphs = [register("0,0,24,24", p) for p in TWO_COLOUR]
    lines.append(f"{'two colour':<12}" + " ".join(glyphs))
    # Two cells wide from here on, so gradients and stroke ends show.
    # Each glyph takes one column; the space after it is the second cell.
    for label, payloads in (("gradients", GRADIENTS), ("clips", CLIPS)):
        glyphs = [register("0,0,24,24", p, "width=2") for p in payloads]
        lines.append(f"{label:<12}" + "  ".join(glyphs))
    glyphs = [register("0,0,24,24", p, "width=2") for p in STROKES]
    lines.append(f"{'strokes':<12}" + "  ".join(glyphs) + "   round, butt/miter, square/bevel")
    for label, folder in EMOJI_SETS:
        glyphs = []
        for name in EMOJI_NAMES:
            with open(os.path.join(EMOJI, folder, name + ".vg")) as f:
                vb, payload = f.read().split("\n")[:2]
            glyphs.append(register(vb, payload, "width=2"))
        lines.append(f"{label:<12}" + "  ".join(glyphs))

    folder = register(*convert(os.path.join(ICONS, "lucide/folder.svg")))
    lines.append(f"{'follows fg':<12}"
                 + " ".join(f"\x1b[{c}m{folder}{RESET}" for c in FG_COLOURS))

    wide = [register(*convert(os.path.join(ICONS, f)), "width=2")
            for f in ("phosphor/heart-duotone.svg", "lucide/terminal.svg",
                      "octicons/git-branch-24.svg")]
    # Each takes one column; the space after it is the second cell.
    lines.append(f"{'width=2':<12}" + "  ".join(wide))

    heart_vb, heart = convert(os.path.join(ICONS, "heroicons/heart.svg"))
    sized = [register(heart_vb, heart, f"size={s}") for s in SIZES]
    lines.append(f"{'size=':<12}"
                 + " ".join(f"{GREY_BG}{g}{RESET}" for g in sized)
                 + "   " + " ".join(SIZES))

    sys.stdout.write("".join(out))
    print(f"fmt=vg demo (terminal advertises fmt={','.join(formats)})\n")
    for line in lines:
        print(line)
    print()


if __name__ == "__main__":
    main()
