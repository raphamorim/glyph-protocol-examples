// Glyph Protocol demo — sample output exercising the protocol's payload
// formats:
//   colrv1 : "Rio Terminal Emulator" rendered via Typearture's Nabla.
//   colrv0 : "Using Glyph protocol" rendered via Zupra 3D eYe/FS.
//   glyf   : five Nerd Font outlines (branch / folder / home / heart /
//            Rust cog) registered at consecutive PUA-B codepoints.
//   vg     : SVG path data and paint as text (proposed, raphamorim/rio#2095):
//            icons, gradients, clips, and Noto and Twemoji emoji.
//
// The demo is deliberately non-TUI: it probes the protocol, sends the
// registrations, prints the lines in cooked mode, and exits.

mod colorv2;
mod noto_fruit;
mod twemoji;
mod vg;
mod zupra_using;

use std::io::{self, Write};

use crossterm::terminal::{disable_raw_mode, enable_raw_mode};

// Note:
// glyf slots — five Nerd Font outlines at contiguous PUA-B codepoints.

const SLOT_CPS: [u32; 5] = [
    0x10_0000, 0x10_0001, 0x10_0002, 0x10_0003, 0x10_0004,
];

/// `width=2` slots — the same five outlines re-registered at a fresh
/// PUA-B range so the wide row prints alongside the narrow row for a
/// direct visual comparison. `width=2` only widens the pixels a glyph
/// is drawn into: the codepoint still takes one column (its wcwidth),
/// so the printed text leaves a blank cell after each one.
const WIDE_SLOT_CPS: [u32; 5] = [
    0x10_0005, 0x10_0006, 0x10_0007, 0x10_0008, 0x10_0009,
];

/// The Nerd Font icons are drawn for the font's own cell: a 600 unit
/// advance, ascender 1020 and descender -300 at upm 1000. Declaring
/// that extent (§8.5.1) places every icon the way the patched font
/// would, and keeps the five in proportion to each other. Without it
/// the terminal would size each outline from its own bounds.
const NERD_EXTENT: &str = "aw=600;lh=1320;ymin=-300";

/// The placement rows declare a square em around the heart instead
/// (the heart spans x 0..923, y -35..761). A terminal cell is taller
/// than it is wide, so a square extent is where the size modes differ.
const PLACEMENT_EXTENT: &str = "aw=1000;lh=1000;ymin=-120";

/// Placement demo slots — the Nerd Font heart outline registered
/// repeatedly at a fresh PUA-B range, each registration carrying
/// different §8.5 sizing/placement parameters (`size=`, `align=`,
/// `pad=`, `width=`). Printed over a grey background block so the
/// cell box — and where the glyph lands inside it — is visible.
/// `(codepoint, extra wire params)`; the printed legend names each.
const PLACEMENT_SLOTS: [(u32, &str); 9] = [
    // Size modes (§8.5.3): height scales the extent to the cell's
    // height (may overflow sideways and get clipped), contain is the
    // default and fits it inside, cover fills the cell (may overflow),
    // stretch distorts to fill both axes exactly.
    (0x10_0070, "size=height"),
    (0x10_0071, "size=contain"),
    // In a cell taller than wide, a square extent makes cover pick the
    // height factor, so this heart renders the same as the height one,
    // while contain fits by width. That's correct: the modes only
    // differ by which axis wins.
    (0x10_0072, "size=cover"),
    (0x10_0073, "size=stretch"),
    // Alignment (§8.5.4) — where the scaled outline sits in the span.
    (0x10_0074, "size=contain;align=start,start"),
    (0x10_0075, "size=contain;align=end,end"),
    // `baseline` puts the outline's y=0 on the text baseline, so a
    // character-like glyph sits on the same line as adjacent text.
    (0x10_0076, "size=contain;align=center,baseline"),
    // Pad (§8.5.2) insets the render span before scaling.
    (0x10_0077, "size=stretch;pad=0.2,0.2,0.2,0.2"),
    // width=2 with contain: one glyph drawn across two cells.
    (0x10_0078, "width=2;size=contain"),
];

/// Each letter of the colour demos is registered with the font's line
/// height and descender, plus its own advance. size=height scales by
/// line height alone, so every letter gets the same scale and they all
/// sit on one baseline; the advance centres each letter in its cell.
fn letter_extent(advance: u32, line_height: u32, ymin: i32) -> String {
    format!("aw={advance};lh={line_height};ymin={ymin};size=height")
}

/// `glyf` simple-glyph bytes, base64-encoded, extracted from
/// JetBrainsMono Nerd Font via fontTools. `units per em` is 1000.
/// Source codepoints in the Nerd Font (for reference):
///   branch  U+E0A0
///   folder  U+F07B
///   home    U+F015
///   heart   U+F004
///   rust    U+E7A8
const NERD_GLYFS_B64: [&str; 5] = [
    "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",
];

fn send_register_glyf(
    out: &mut impl Write,
    cp: u32,
    payload: &str,
) -> io::Result<()> {
    // `fmt` is implicit (`glyf`). `upm` defaults to 1000 but we send
    // it explicitly to match the source font's authoring EM. The
    // `reply=0` level tells the terminal to register silently — no
    // `status=0` ACK — which keeps the shell prompt clean on exit
    // (otherwise the queued ACK bytes spill to the next prompt).
    write!(
        out,
        "\x1b_25a1;r;cp={:x};reply=0;upm=1000;{};{}\x1b\\",
        cp, NERD_EXTENT, payload
    )
}

fn send_register_glyf_wide(
    out: &mut impl Write,
    cp: u32,
    payload: &str,
) -> io::Result<()> {
    // Same as send_register_glyf but drawn across two cells. The
    // printed text keeps the cell after it blank for the overflow.
    write!(
        out,
        "\x1b_25a1;r;cp={:x};reply=0;upm=1000;{};width=2;{}\x1b\\",
        cp, NERD_EXTENT, payload
    )
}

fn send_register_glyf_placed(
    out: &mut impl Write,
    cp: u32,
    params: &str,
    payload: &str,
) -> io::Result<()> {
    // Same as send_register_glyf plus the registration's §8.5
    // sizing/placement parameters, verbatim from PLACEMENT_SLOTS.
    write!(
        out,
        "\x1b_25a1;r;cp={:x};reply=0;upm=1000;{};{};{}\x1b\\",
        cp, PLACEMENT_EXTENT, params, payload
    )
}

fn send_register_colrv1(
    out: &mut impl Write,
    cp: u32,
    payload: &str,
) -> io::Result<()> {
    write!(
        out,
        "\x1b_25a1;r;cp={:x};fmt=colrv1;reply=0;upm=1000;{}\x1b\\",
        cp, payload
    )
}

fn send_register_colrv1_letter(
    out: &mut impl Write,
    cp: u32,
    extent: &str,
    payload: &str,
) -> io::Result<()> {
    write!(
        out,
        "\x1b_25a1;r;cp={:x};fmt=colrv1;reply=0;upm=1000;{};{}\x1b\\",
        cp, extent, payload
    )
}

/// Offset for the `width=2` duplicates of the Nabla letters: the same
/// colrv1 payloads re-registered at `cp + NABLA_WIDE_OFFSET`
/// (0x100020–0x10002C, free between the Nabla and Zupra ranges) so the
/// second colrv1 line renders each letter across two cells.
const NABLA_WIDE_OFFSET: u32 = 0x10;

fn send_register_colrv1_wide(
    out: &mut impl Write,
    cp: u32,
    extent: &str,
    payload: &str,
) -> io::Result<()> {
    write!(
        out,
        "\x1b_25a1;r;cp={:x};fmt=colrv1;reply=0;upm=1000;{};width=2;{}\x1b\\",
        cp, extent, payload
    )
}

fn send_register_colrv0(
    out: &mut impl Write,
    cp: u32,
    upm: u32,
    payload: &str,
) -> io::Result<()> {
    write!(
        out,
        "\x1b_25a1;r;cp={:x};fmt=colrv0;reply=0;upm={};{}\x1b\\",
        cp, upm, payload
    )
}

fn send_register_colrv0_letter(
    out: &mut impl Write,
    cp: u32,
    upm: u32,
    extent: &str,
    payload: &str,
) -> io::Result<()> {
    write!(
        out,
        "\x1b_25a1;r;cp={:x};fmt=colrv0;reply=0;upm={};{};{}\x1b\\",
        cp, upm, extent, payload
    )
}

fn register_glyf_slots(out: &mut impl Write) -> io::Result<()> {
    // Flush after every register. The payloads are multi-KB and Rust's
    // LineWriter around stdout can otherwise split a single APC body
    // across two write() syscalls; if the PTY happens to hand the
    // first half to Rio's APC parser before the tail arrives, the
    // parser sees a short payload and replies `malformed_payload`.
    // A flush per message makes each APC a single atomic write.
    for (cp, payload) in SLOT_CPS.iter().zip(NERD_GLYFS_B64.iter()) {
        send_register_glyf(out, *cp, payload)?;
        out.flush()?;
    }
    // Wide row: the same five outlines, re-registered at a fresh PUA-B
    // range with `width=2` so each occupies two terminal cells.
    for (cp, payload) in WIDE_SLOT_CPS.iter().zip(NERD_GLYFS_B64.iter()) {
        send_register_glyf_wide(out, *cp, payload)?;
        out.flush()?;
    }
    // Placement rows: the heart outline under each §8.5 parameter set.
    for (cp, params) in PLACEMENT_SLOTS.iter() {
        send_register_glyf_placed(out, *cp, params, NERD_GLYFS_B64[3])?;
        out.flush()?;
    }
    Ok(())
}

fn register_nabla(out: &mut impl Write) -> io::Result<()> {
    for (_, cp, advance, payload) in colorv2::NABLA_LETTERS.iter() {
        let extent =
            letter_extent(*advance, colorv2::LINE_HEIGHT, colorv2::YMIN);
        send_register_colrv1_letter(out, *cp, &extent, payload)?;
        out.flush()?;
        // Wide duplicate for the second colrv1 line: same payload at a
        // fresh codepoint, drawn across two cells with `width=2`.
        send_register_colrv1_wide(
            out,
            *cp + NABLA_WIDE_OFFSET,
            &extent,
            payload,
        )?;
        out.flush()?;
    }
    Ok(())
}

fn register_noto_fruit(out: &mut impl Write) -> io::Result<()> {
    for (_, cp, payload) in noto_fruit::NOTO_FRUIT_LETTERS.iter() {
        send_register_colrv1(out, *cp, payload)?;
        out.flush()?;
    }
    Ok(())
}

fn register_zupra(out: &mut impl Write) -> io::Result<()> {
    for (_, cp, advance, payload) in zupra_using::ZUPRA_LETTERS.iter() {
        let extent = letter_extent(
            *advance,
            zupra_using::LINE_HEIGHT,
            zupra_using::YMIN,
        );
        send_register_colrv0_letter(
            out,
            *cp,
            zupra_using::UPM,
            &extent,
            payload,
        )?;
        out.flush()?;
    }
    Ok(())
}

fn register_twemoji(out: &mut impl Write) -> io::Result<()> {
    for (_, cp, payload) in twemoji::TWEMOJI_LETTERS.iter() {
        send_register_colrv0(out, *cp, twemoji::UPM, payload)?;
        out.flush()?;
    }
    Ok(())
}

fn nabla_text() -> String {
    colorv2::TEXT
        .chars()
        .map(|ch| {
            colorv2::NABLA_LETTERS
                .iter()
                .find(|(letter, _, _, _)| *letter == ch)
                .map(|(_, cp, _, _)| char::from_u32(*cp).unwrap())
                .unwrap_or(ch)
        })
        .collect()
}

/// Same text through the `width=2` duplicates — each letter occupies
/// two terminal cells.
fn nabla_wide_text() -> String {
    // A width=2 glyph overflows into the following cell in *pixels*; the
    // terminal does not reserve that cell, so the codepoint still counts
    // as one logical column (== system wcwidth → no cursor/copy desync).
    // We therefore author an explicit trailing space after each glyph
    // for the overflow to land on, which also makes the row read as a
    // sequence of two-cell glyphs.
    let mut out = String::new();
    for ch in colorv2::TEXT.chars() {
        match colorv2::NABLA_LETTERS
            .iter()
            .find(|(letter, _, _, _)| *letter == ch)
        {
            Some((_, cp, _, _)) => {
                out.push(char::from_u32(*cp + NABLA_WIDE_OFFSET).unwrap());
                out.push(' ');
            }
            None => out.push(ch),
        }
    }
    out
}

fn zupra_text() -> String {
    zupra_using::TEXT
        .chars()
        .map(|ch| {
            zupra_using::ZUPRA_LETTERS
                .iter()
                .find(|(letter, _, _, _)| *letter == ch)
                .map(|(_, cp, _, _)| char::from_u32(*cp).unwrap())
                .unwrap_or(ch)
        })
        .collect()
}

fn twemoji_text() -> String {
    twemoji::TEXT
        .chars()
        .map(|ch| {
            twemoji::TWEMOJI_LETTERS
                .iter()
                .find(|(letter, _, _)| *letter == ch)
                .map(|(_, cp, _)| char::from_u32(*cp).unwrap())
                .unwrap_or(ch)
        })
        .collect()
}

/// `fmt=vg` is text, not base64: the payload follows `vb` as is.
fn register_vg(out: &mut impl Write, glyphs: &[vg::VgGlyph]) -> io::Result<()> {
    for g in glyphs {
        let extra = if g.extra.is_empty() {
            String::new()
        } else {
            format!(";{}", g.extra)
        };
        write!(
            out,
            "\x1b_25a1;r;cp={:x};fmt=vg;reply=0;vb={}{};{}\x1b\\",
            g.cp, g.vb, extra, g.payload
        )?;
    }
    Ok(())
}

/// Glyphs separated by two spaces; a `width=2` glyph takes the first.
fn vg_line(glyphs: &[vg::VgGlyph]) -> String {
    glyphs
        .iter()
        .map(|g| char::from_u32(g.cp).unwrap().to_string())
        .collect::<Vec<_>>()
        .join("  ")
}

fn noto_fruit_text() -> String {
    noto_fruit::TEXT
        .chars()
        .map(|ch| {
            noto_fruit::NOTO_FRUIT_LETTERS
                .iter()
                .find(|(letter, _, _)| *letter == ch)
                .map(|(_, cp, _)| char::from_u32(*cp).unwrap())
                .unwrap_or(ch)
        })
        .collect()
}

fn glyf_icons() -> String {
    SLOT_CPS
        .iter()
        .map(|cp| char::from_u32(*cp).unwrap())
        .fold(String::new(), |mut acc, ch| {
            if !acc.is_empty() {
                acc.push_str("  ");
            }
            acc.push(ch);
            acc
        })
}

fn glyf_wide_icons() -> String {
    WIDE_SLOT_CPS
        .iter()
        .map(|cp| char::from_u32(*cp).unwrap())
        .fold(String::new(), |mut acc, ch| {
            if !acc.is_empty() {
                acc.push_str("  ");
            }
            acc.push(ch);
            acc
        })
}

/// One placement glyph over a grey background block, so the cell box
/// (and where the §8.5 transform put the outline inside it) is
/// visible.
fn boxed(cp: u32) -> String {
    format!("\x1b[48;2;64;64;64m{}\x1b[0m", char::from_u32(cp).unwrap())
}

/// The same for a `width=2` glyph. The codepoint takes one column and
/// the glyph overflows into the blank cell after it, so that cell is
/// printed as a space inside the same grey block.
fn boxed_wide(cp: u32) -> String {
    format!("\x1b[48;2;64;64;64m{} \x1b[0m", char::from_u32(cp).unwrap())
}

/// Send `s` and wait up to `timeout_ms` for the terminal's reply.
/// Returns the list of `fmt` names on success, `None` on timeout or
/// unrecognised reply. Caller owns raw-mode setup / teardown.
#[cfg(unix)]
fn probe_support(timeout_ms: i32) -> Option<Vec<String>> {
    use std::io::Read;
    use std::os::unix::io::AsRawFd;

    let mut out = io::stdout();
    out.write_all(b"\x1b_25a1;s\x1b\\").ok()?;
    out.flush().ok()?;

    let stdin = io::stdin();
    let fd = stdin.as_raw_fd();
    let start = std::time::Instant::now();
    let mut buf = [0u8; 256];
    let mut collected: Vec<u8> = Vec::new();

    loop {
        let elapsed = start.elapsed().as_millis() as i32;
        let remaining = timeout_ms.saturating_sub(elapsed);
        if remaining <= 0 {
            return None;
        }

        let mut pfd = libc::pollfd {
            fd,
            events: libc::POLLIN,
            revents: 0,
        };
        // SAFETY: single valid pollfd by value; libc::poll is a thin
        // syscall wrapper.
        let r = unsafe { libc::poll(&mut pfd, 1, remaining) };
        if r <= 0 {
            return None;
        }

        let n = stdin.lock().read(&mut buf).ok()?;
        if n == 0 {
            return None;
        }
        collected.extend_from_slice(&buf[..n]);
        if collected.windows(2).any(|w| w == b"\x1b\\") {
            break;
        }
        if collected.len() > 2048 {
            break;
        }
    }

    parse_fmt_field(&collected)
}

#[cfg(not(unix))]
fn probe_support(_timeout_ms: i32) -> Option<Vec<String>> {
    // No portable non-blocking TTY read on other platforms; assume
    // `glyf` support optimistically. Unsupported terminals silently
    // drop the register sequences below, so the demo degrades cleanly.
    Some(vec!["glyf".to_string()])
}

/// Scan an APC reply for the `fmt=` field and split its comma-separated
/// list of format names. Returns `Some(vec![])` when the field is empty
/// (terminal advertises no formats); `None` when no `fmt=` is found or
/// the bytes are not valid UTF-8.
fn parse_fmt_field(data: &[u8]) -> Option<Vec<String>> {
    let i = data.windows(4).position(|w| w == b"fmt=")? + 4;
    let mut end = i;
    while end < data.len() && data[end] != b';' && data[end] != b'\x1b' {
        end += 1;
    }
    let value = std::str::from_utf8(&data[i..end]).ok()?;
    if value.is_empty() {
        return Some(Vec::new());
    }
    Some(value.split(',').map(|s| s.to_string()).collect())
}

fn main() -> io::Result<()> {
    // Raw mode is needed only for the probe — we need stdin to hand
    // us the `s` reply byte-by-byte. After the probe we drop back to
    // cooked mode; the registers below use `silent=1` so the terminal
    // emits no ACK bytes that would need draining. Three println!s
    // follow, no alternate screen, no TUI.
    enable_raw_mode()?;
    let fmt = probe_support(200);
    disable_raw_mode()?;

    let Some(fmts) = fmt else {
        eprintln!(
            "This terminal does not implement Glyph Protocol. Run in Rio or \
             another conformant terminal to see the demo."
        );
        return Ok(());
    };
    let has_glyf = fmts.iter().any(|n| n == "glyf");
    let has_colrv0 = fmts.iter().any(|n| n == "colrv0");
    let has_colrv1 = fmts.iter().any(|n| n == "colrv1");
    let has_vg = fmts.iter().any(|n| n == "vg");

    let mut stdout = io::stdout();
    if has_glyf {
        register_glyf_slots(&mut stdout)?;
    }
    if has_colrv0 {
        register_zupra(&mut stdout)?;
        register_twemoji(&mut stdout)?;
    }
    if has_colrv1 {
        register_nabla(&mut stdout)?;
        register_noto_fruit(&mut stdout)?;
    }
    if has_vg {
        for group in [vg::ICONS, vg::PAINTS, vg::EMOJI_NOTO, vg::EMOJI_TWEMOJI] {
            register_vg(&mut stdout, group)?;
        }
    }
    stdout.flush()?;

    println!("");

    let colrv1_line = if has_colrv1 {
        nabla_text()
    } else {
        String::from("(colrv1 not advertised by terminal)")
    };
    let colrv1_wide_line = if has_colrv1 {
        nabla_wide_text()
    } else {
        String::from("(colrv1 not advertised by terminal)")
    };
    let colrv0_line = if has_colrv0 {
        zupra_text()
    } else {
        String::from("(colrv0 not advertised by terminal)")
    };
    let colrv0_emoji = if has_colrv0 {
        twemoji_text()
    } else {
        String::from("(colrv0 not advertised by terminal)")
    };
    let colrv1_emoji = if has_colrv1 {
        noto_fruit_text()
    } else {
        String::from("(colrv1 not advertised by terminal)")
    };
    let glyf_line = if has_glyf {
        glyf_icons()
    } else {
        String::from("(glyf not advertised by terminal)")
    };
    let glyf_wide_line = if has_glyf {
        glyf_wide_icons()
    } else {
        String::from("(glyf not advertised by terminal)")
    };

    println!("colrv1 example: \x1b[48;2;21;49;162m{}\x1b[0m", colrv1_line);
    // Second line: the same letters via the width=2 duplicates.
    println!("colrv1 example: {}", colrv1_wide_line);
    println!("colrv1 example: {}", colrv1_emoji);
    println!("colrv0 example: {}", colrv0_line);
    println!("colrv0 example: {}", colrv0_emoji);
    println!("glyf examples:  {}", glyf_line);
    println!("glyf wide ex.:  {}", glyf_wide_line);
    if has_vg {
        println!("vg icons:       {}", vg_line(vg::ICONS));
        println!("vg paints:      {}", vg_line(vg::PAINTS));
        println!("vg emoji:       {}", vg_line(vg::EMOJI_NOTO));
        println!("vg emoji:       {}", vg_line(vg::EMOJI_TWEMOJI));
    } else {
        println!("vg examples:    (vg not advertised by terminal)");
    }

    // §8.5 sizing/placement — the same heart outline under different
    // size= / align= / pad= / width= parameters. The grey block is the
    // cell box; where the heart lands inside it is the demo.
    if has_glyf {
        println!(
            "placement size: {}  {}  {}  {}   (height / contain / cover / stretch)",
            boxed(0x10_0070),
            boxed(0x10_0071),
            boxed(0x10_0072),
            boxed(0x10_0073),
        );
        println!(
            "placement align: {} {}  xy{}z   (start,start / end,end / baseline in text)",
            boxed(0x10_0074),
            boxed(0x10_0075),
            boxed(0x10_0076),
        );
        println!(
            "placement more: {} {}   (stretch + pad=0.2 / width=2 contain)",
            boxed(0x10_0077),
            boxed_wide(0x10_0078),
        );
    } else {
        println!("placement:      (glyf not advertised by terminal)");
    }

    Ok(())
}
