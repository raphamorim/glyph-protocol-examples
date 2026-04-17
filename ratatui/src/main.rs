// Glyph Protocol demo — registers five real Nerd Font outlines at
// empty Supplementary PUA-B codepoints (U+100000..U+100004). The
// `glyf` bytes come straight out of JetBrainsMono Nerd Font via
// fontTools, so the icons look exactly like Nerd Fonts but the
// codepoints they're mounted at are empty — you don't need Nerd
// Fonts installed to see them.
//
// Keybindings:
//   ↑ re-register all (handy when you clear and want them back)
//   ↓ clear all registrations
//   q / esc  quit

use std::io::{self, Write};
use std::time::Duration;

use crossterm::{
    event::{self, Event, KeyCode, KeyEventKind},
    execute,
    terminal::{
        disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen,
    },
};
use ratatui::{
    prelude::*,
    widgets::{Block, Borders, Paragraph},
};

// -------------------------------------------------------------------------
// Slots + glyf payloads
// -------------------------------------------------------------------------

/// Five contiguous slots in Supplementary PUA-B. No known font covers
/// this range; every pixel you see is rendered from our registration.
const SLOT_CPS: [u32; 5] = [
    0x10_0000, 0x10_0001, 0x10_0002, 0x10_0003, 0x10_0004,
];

const SLOT_LABELS: [&str; 5] = ["branch", "folder", "home", "heart", "rust"];

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

// -------------------------------------------------------------------------
// Wire helpers
// -------------------------------------------------------------------------

fn send_clear_all(out: &mut impl Write) -> io::Result<()> {
    write!(out, "\x1b_1cc6D;c\x1b\\")
}

/// Emit a register APC. `payload` is already base64 — fastest path.
fn send_register(out: &mut impl Write, cp: u32, payload: &str) -> io::Result<()> {
    // `fmt` is implicit (`glyf`); `upm` defaults to 1000 but we send
    // it explicitly to match the font's authoring EM.
    write!(out, "\x1b_1cc6D;r;cp={:x};upm=1000;{}\x1b\\", cp, payload)
}

fn register_all(out: &mut impl Write) -> io::Result<()> {
    for (cp, payload) in SLOT_CPS.iter().zip(NERD_GLYFS_B64.iter()) {
        send_register(out, *cp, payload)?;
    }
    out.flush()
}

// -------------------------------------------------------------------------
// App
// -------------------------------------------------------------------------

struct App {
    cleared: bool,
}

impl App {
    fn new() -> Self {
        Self { cleared: false }
    }
}

// -------------------------------------------------------------------------
// UI
// -------------------------------------------------------------------------

fn ui(f: &mut Frame, app: &App) {
    let chunks = Layout::default()
        .direction(Direction::Vertical)
        .constraints([
            Constraint::Length(3),
            Constraint::Length(5),
            Constraint::Min(3),
            Constraint::Length(8),
        ])
        .split(f.area());

    let title = Paragraph::new(
        "Glyph Protocol — real Nerd Font outlines at empty PUA-B codepoints",
    )
    .block(
        Block::default()
            .borders(Borders::ALL)
            .title("ratatui × glyph protocol"),
    )
    .alignment(Alignment::Center);
    f.render_widget(title, chunks[0]);

    // Show the five slots inline, like they'd appear in a shell prompt.
    let prompt_line = format!(
        " {br} main  {fo} src  {ho} ~  {he} favs  {ru} rio ",
        br = char::from_u32(SLOT_CPS[0]).unwrap(),
        fo = char::from_u32(SLOT_CPS[1]).unwrap(),
        ho = char::from_u32(SLOT_CPS[2]).unwrap(),
        he = char::from_u32(SLOT_CPS[3]).unwrap(),
        ru = char::from_u32(SLOT_CPS[4]).unwrap(),
    );
    let prompt = Paragraph::new(Line::from(vec![Span::styled(
        prompt_line,
        Style::default().add_modifier(Modifier::BOLD),
    )]))
    .alignment(Alignment::Center)
    .block(Block::default().borders(Borders::ALL).title("mock prompt"));
    f.render_widget(prompt, chunks[1]);

    let mut rows: Vec<Line> = Vec::with_capacity(SLOT_CPS.len());
    for (i, &cp) in SLOT_CPS.iter().enumerate() {
        let ch = char::from_u32(cp).unwrap();
        rows.push(Line::from(format!(
            "U+{:06X}  {:<7}  {}",
            cp, SLOT_LABELS[i], ch
        )));
    }
    let table = Paragraph::new(rows)
        .block(Block::default().borders(Borders::ALL).title("codepoints"));
    f.render_widget(table, chunks[2]);

    let state = if app.cleared {
        "cleared (press ↑ to re-register)"
    } else {
        "5 Nerd Font outlines registered"
    };
    let help = Paragraph::new(vec![
        Line::from("↑  re-register all"),
        Line::from("↓  clear all registrations"),
        Line::from("q  quit"),
        Line::from(""),
        Line::from(state),
    ])
    .block(Block::default().borders(Borders::ALL).title("keys"));
    f.render_widget(help, chunks[3]);
}

// -------------------------------------------------------------------------
// Main loop
// -------------------------------------------------------------------------

fn run<B: Backend>(terminal: &mut Terminal<B>, app: &mut App) -> io::Result<()> {
    let mut stdout = io::stdout();
    register_all(&mut stdout)?;

    loop {
        terminal.draw(|f| ui(f, app))?;

        if !event::poll(Duration::from_millis(100))? {
            continue;
        }
        let Event::Key(key) = event::read()? else { continue };
        if key.kind != KeyEventKind::Press {
            continue;
        }
        match key.code {
            KeyCode::Char('q') | KeyCode::Esc => return Ok(()),
            KeyCode::Up => {
                register_all(&mut stdout)?;
                app.cleared = false;
            }
            KeyCode::Down => {
                send_clear_all(&mut stdout)?;
                stdout.flush()?;
                app.cleared = true;
            }
            _ => {}
        }
    }
}

fn main() -> io::Result<()> {
    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen)?;

    let backend = CrosstermBackend::new(io::stdout());
    let mut terminal = Terminal::new(backend)?;

    let mut app = App::new();
    let result = run(&mut terminal, &mut app);

    // Belt and suspenders: clear any leftover registrations so the
    // next program in this session sees a clean font stack.
    let mut stdout = io::stdout();
    let _ = send_clear_all(&mut stdout);
    let _ = stdout.flush();

    disable_raw_mode()?;
    execute!(io::stdout(), LeaveAlternateScreen)?;

    result
}
