// Glyph Protocol demo — sample output exercising the protocol's three
// payload formats:
//   colrv1 : "Rio Terminal Emulator" rendered via Typearture's Nabla.
//   colrv0 : "Using Glyph protocol" rendered via Zupra 3D eYe/FS.
//   glyf   : five Nerd Font outlines (branch / folder / home / heart /
//            Rust cog) registered at consecutive PUA-B codepoints.
//
// The demo is deliberately non-TUI: it probes the protocol, sends the
// registrations, prints the lines in cooked mode, and exits.

mod colorv2;
mod zupra_using;

use std::io::{self, Write};

use crossterm::terminal::{disable_raw_mode, enable_raw_mode};

// Note:
// glyf slots — five Nerd Font outlines at contiguous PUA-B codepoints.

const SLOT_CPS: [u32; 5] = [
    0x10_0000, 0x10_0001, 0x10_0002, 0x10_0003, 0x10_0004,
];

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
        "\x1b_25a1;r;cp={:x};reply=0;upm=1000;{}\x1b\\",
        cp, payload
    )
}

fn send_register_colrv1(
    out: &mut impl Write,
    cp: u32,
    payload: &str,
) -> io::Result<()> {
    write!(
        out,
        "\x1b_25a1;r;cp={:x};fmt=colrv1;reply=2;upm=1000;{}\x1b\\",
        cp, payload
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
        "\x1b_25a1;r;cp={:x};fmt=colrv0;reply=2;upm={};{}\x1b\\",
        cp, upm, payload
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
    Ok(())
}

fn register_nabla(out: &mut impl Write) -> io::Result<()> {
    for (_, cp, payload) in colorv2::NABLA_LETTERS.iter() {
        send_register_colrv1(out, *cp, payload)?;
        out.flush()?;
    }
    Ok(())
}

fn register_zupra(out: &mut impl Write) -> io::Result<()> {
    for (_, cp, payload) in zupra_using::ZUPRA_LETTERS.iter() {
        send_register_colrv0(out, *cp, zupra_using::UPM, payload)?;
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
                .find(|(letter, _, _)| *letter == ch)
                .map(|(_, cp, _)| char::from_u32(*cp).unwrap())
                .unwrap_or(ch)
        })
        .collect()
}

fn zupra_text() -> String {
    zupra_using::TEXT
        .chars()
        .map(|ch| {
            zupra_using::ZUPRA_LETTERS
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

/// Send `s` and wait up to `timeout_ms` for the terminal's reply.
/// Returns the `fmt` bitfield on success, `None` on timeout or
/// unrecognised reply. Caller owns raw-mode setup / teardown.
#[cfg(unix)]
fn probe_support(timeout_ms: i32) -> Option<u8> {
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
fn probe_support(_timeout_ms: i32) -> Option<u8> {
    // No portable non-blocking TTY read on other platforms; assume
    // `glyf` support optimistically. Unsupported terminals silently
    // drop the register sequences below, so the demo degrades cleanly.
    Some(0b0000_0001)
}

/// Scan an APC reply for the `fmt=` field and parse its decimal u8.
fn parse_fmt_field(data: &[u8]) -> Option<u8> {
    let i = data.windows(4).position(|w| w == b"fmt=")? + 4;
    let mut v: u32 = 0;
    let mut any = false;
    for &b in &data[i..] {
        if !b.is_ascii_digit() {
            break;
        }
        any = true;
        v = v.checked_mul(10)?.checked_add((b - b'0') as u32)?;
        if v > u8::MAX as u32 {
            return None;
        }
    }
    if any {
        Some(v as u8)
    } else {
        None
    }
}

const FMT_GLYF_BIT: u8 = 0b0000_0001;
const FMT_COLRV0_BIT: u8 = 0b0000_0010;
const FMT_COLRV1_BIT: u8 = 0b0000_0100;

fn main() -> io::Result<()> {
    // Raw mode is needed only for the probe — we need stdin to hand
    // us the `s` reply byte-by-byte. After the probe we drop back to
    // cooked mode; the registers below use `silent=1` so the terminal
    // emits no ACK bytes that would need draining. Three println!s
    // follow, no alternate screen, no TUI.
    enable_raw_mode()?;
    let fmt = probe_support(200);
    disable_raw_mode()?;

    let Some(bits) = fmt else {
        eprintln!(
            "This terminal does not implement Glyph Protocol. Run in Rio or \
             another conformant terminal to see the demo."
        );
        return Ok(());
    };
    let has_glyf = bits & FMT_GLYF_BIT != 0;
    let has_colrv0 = bits & FMT_COLRV0_BIT != 0;
    let has_colrv1 = bits & FMT_COLRV1_BIT != 0;

    let mut stdout = io::stdout();
    if has_glyf {
        register_glyf_slots(&mut stdout)?;
    }
    if has_colrv0 {
        register_zupra(&mut stdout)?;
    }
    if has_colrv1 {
        register_nabla(&mut stdout)?;
    }
    stdout.flush()?;

    println!("");

    let colrv1_line = if has_colrv1 {
        nabla_text()
    } else {
        String::from("(colrv1 not advertised by terminal)")
    };
    let colrv0_line = if has_colrv0 {
        zupra_text()
    } else {
        String::from("(colrv0 not advertised by terminal)")
    };
    let glyf_line = if has_glyf {
        glyf_icons()
    } else {
        String::from("(glyf not advertised by terminal)")
    };

    println!("colrv1 example: \x1b[48;2;21;49;162m{}\x1b[0m", colrv1_line);
    println!("colrv1 example: {}", colrv1_line);
    println!("colrv0 example: {}", colrv0_line);
    println!("glyf examples:  {}", glyf_line);

    Ok(())
}
