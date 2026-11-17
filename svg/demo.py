#!/usr/bin/env python3
"""Glyph Protocol `fmt=path` demo: SVG icons drawn by the terminal.

    python3 svg/demo.py

Probes the terminal with `s`, and if it lists `path` in `fmt`,
registers icons from seven icon sets plus a few two-colour glyphs and
prints them. Standard library only; needs a Unix tty.
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
    if "path" not in formats:
        print(f"the terminal advertises fmt={','.join(formats)}, without path")
        print("fmt=path is a proposal: build Rio from raphamorim/rio#2095")
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
    print(f"fmt=path demo (terminal advertises fmt={','.join(formats)})\n")
    for line in lines:
        print(line)
    print()


if __name__ == "__main__":
    main()
