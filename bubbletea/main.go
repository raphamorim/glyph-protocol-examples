// Glyph Protocol demo — sample output exercising the protocol's payload
// formats:
//
//	colrv1 : "Rio Terminal Emulator" rendered via Typearture's Nabla.
//	colrv0 : "Using Glyph protocol" rendered via Zupra 3D eYe/FS.
//	glyf   : five Nerd Font outlines (branch / folder / home / heart /
//	         Rust cog) registered at consecutive PUA-B codepoints.
//	vg     : SVG path data and paint as text (proposed, raphamorim/rio#2095):
//	         icons, gradients, clips, and Noto and Twemoji emoji.
//
// The demo is deliberately non-TUI: it probes the protocol, sends the
// registrations, prints the lines in cooked mode, and exits.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/muesli/cancelreader"
)

// glyf slots — five Nerd Font outlines at contiguous PUA-B codepoints.
var slotCps = [5]rune{
	0x100000, 0x100001, 0x100002, 0x100003, 0x100004,
}

// `width=2` slots — the same five outlines re-registered at a fresh
// PUA-B range so the wide row prints alongside the narrow row for a
// direct visual comparison. `width=2` only widens the pixels a glyph
// is drawn into: the codepoint still takes one column (its wcwidth),
// so the printed text leaves a blank cell after each one.
var wideSlotCps = [5]rune{
	0x100005, 0x100006, 0x100007, 0x100008, 0x100009,
}

// The Nerd Font icons are drawn for the font's own cell: a 600 unit
// advance, ascender 1020 and descender -300 at upm 1000. Declaring
// that extent (§8.5.1) places every icon the way the patched font
// would, and keeps the five in proportion to each other. Without it
// the terminal would size each outline from its own bounds.
const nerdExtent = "aw=600;lh=1320;ymin=-300"

// The placement rows declare a square em around the heart instead
// (the heart spans x 0..923, y -35..761). A terminal cell is taller
// than it is wide, so a square extent is where the size modes differ.
const placementExtent = "aw=1000;lh=1000;ymin=-120"

// Placement demo slots — the Nerd Font heart outline registered
// repeatedly at a fresh PUA-B range, each registration carrying
// different §8.5 sizing/placement parameters (`size=`, `align=`,
// `pad=`, `width=`). Printed over a grey background block so the
// cell box — and where the glyph lands inside it — is visible.
var placementSlots = []struct {
	Cp     rune
	Params string
}{
	// Size modes (§8.5.3): height scales the extent to the cell's
	// height (may overflow sideways and get clipped), contain is the
	// default and fits it inside, cover fills the cell (may overflow),
	// stretch distorts to fill both axes exactly.
	{0x100070, "size=height"},
	{0x100071, "size=contain"},
	// In a cell taller than wide, a square extent makes cover pick the
	// height factor, so this heart renders the same as the height one,
	// while contain fits by width. That's correct: the modes only
	// differ by which axis wins.
	{0x100072, "size=cover"},
	{0x100073, "size=stretch"},
	// Alignment (§8.5.4) — where the scaled outline sits in the span.
	{0x100074, "size=contain;align=start,start"},
	{0x100075, "size=contain;align=end,end"},
	// `baseline` puts the outline's y=0 on the text baseline, so a
	// character-like glyph sits on the same line as adjacent text.
	{0x100076, "size=contain;align=center,baseline"},
	// Pad (§8.5.2) insets the render span before scaling.
	{0x100077, "size=stretch;pad=0.2,0.2,0.2,0.2"},
	// width=2 with contain: one glyph drawn across two cells.
	{0x100078, "width=2;size=contain"},
}

// Each letter of the colour demos is registered with the font's line
// height and descender, plus its own advance. size=height scales by
// line height alone, so every letter gets the same scale and they all
// sit on one baseline; the advance centres each letter in its cell.
func letterExtent(advance, lineHeight, ymin int) string {
	return fmt.Sprintf("aw=%d;lh=%d;ymin=%d;size=height", advance, lineHeight, ymin)
}

// `glyf` simple-glyph bytes, base64-encoded, extracted from
// JetBrainsMono Nerd Font via fontTools. units per em = 1000.
// Source codepoints in the Nerd Font (for reference):
//
//	branch  U+E0A0
//	folder  U+F07B
//	home    U+F015
//	heart   U+F004
//	rust    U+E7A8
var nerdGlyfsB64 = [5]string{
	"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",
}

func sendRegisterGlyf(out io.Writer, cp rune, payload string) {
	// `fmt` is implicit (`glyf`). `upm` defaults to 1000 but we send
	// it explicitly to match the source font's authoring EM. The
	// `reply=2` level only answers failures, so a successful
	// registration leaves no `status=0` ACK to spill into the shell
	// prompt on exit.
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;fmt=glyf;reply=2;upm=1000;%s;%s\x1b\\",
		cp, nerdExtent, payload,
	)
}

func sendRegisterGlyfWide(out io.Writer, cp rune, payload string) {
	// Same as sendRegisterGlyf but drawn across two cells. The printed
	// text keeps the cell after it blank for the overflow.
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;fmt=glyf;reply=2;upm=1000;%s;width=2;%s\x1b\\",
		cp, nerdExtent, payload,
	)
}

func sendRegisterGlyfPlaced(out io.Writer, cp rune, params, payload string) {
	// Same as sendRegisterGlyf plus the registration's §8.5
	// sizing/placement parameters, verbatim from placementSlots.
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;fmt=glyf;reply=2;upm=1000;%s;%s;%s\x1b\\",
		cp, placementExtent, params, payload,
	)
}

func sendRegisterColrv1(out io.Writer, cp rune, payload string) {
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;fmt=colrv1;reply=2;upm=1000;%s\x1b\\",
		cp, payload,
	)
}

func sendRegisterColrv1Letter(out io.Writer, cp rune, extent, payload string) {
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;fmt=colrv1;reply=2;upm=1000;%s;%s\x1b\\",
		cp, extent, payload,
	)
}

// Offset for the width=2 duplicates of the Nabla letters: the same
// colrv1 payloads re-registered at cp+nablaWideOffset (0x100020 to
// 0x10002C, free between the Nabla and Zupra ranges) so the second
// colrv1 line renders each letter across two cells.
const nablaWideOffset rune = 0x10

func sendRegisterColrv1Wide(out io.Writer, cp rune, extent, payload string) {
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;fmt=colrv1;reply=2;upm=1000;%s;width=2;%s\x1b\\",
		cp, extent, payload,
	)
}

func sendRegisterColrv0(out io.Writer, cp rune, upm uint32, payload string) {
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;fmt=colrv0;reply=2;upm=%d;%s\x1b\\",
		cp, upm, payload,
	)
}

func sendRegisterColrv0Letter(out io.Writer, cp rune, upm uint32, extent, payload string) {
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;fmt=colrv0;reply=2;upm=%d;%s;%s\x1b\\",
		cp, upm, extent, payload,
	)
}

func registerGlyfSlots(out io.Writer) {
	// The payloads are multi-KB. We flush after every register so the
	// PTY never has a chance to split a single APC body across two
	// writes mid-payload; each APC goes out as one atomic chunk.
	for i, cp := range slotCps {
		sendRegisterGlyf(out, cp, nerdGlyfsB64[i])
		flush(out)
	}
	// Wide row: the same five outlines, re-registered at a fresh PUA-B
	// range with `width=2` so each occupies two terminal cells.
	for i, cp := range wideSlotCps {
		sendRegisterGlyfWide(out, cp, nerdGlyfsB64[i])
		flush(out)
	}
	// Placement rows: the heart outline under each §8.5 parameter set.
	for _, slot := range placementSlots {
		sendRegisterGlyfPlaced(out, slot.Cp, slot.Params, nerdGlyfsB64[3])
		flush(out)
	}
}

func registerNabla(out io.Writer) {
	for _, e := range nablaLetters {
		extent := letterExtent(e.Advance, nablaLineHeight, nablaYMin)
		sendRegisterColrv1Letter(out, e.Cp, extent, e.Payload)
		flush(out)
		// Wide duplicate for the second colrv1 line: same payload at
		// a fresh codepoint, drawn across two cells with width=2.
		sendRegisterColrv1Wide(out, e.Cp+nablaWideOffset, extent, e.Payload)
		flush(out)
	}
}

func registerNotoFruit(out io.Writer) {
	for _, e := range notoFruitLetters {
		sendRegisterColrv1(out, e.Cp, e.Payload)
		flush(out)
	}
}

func registerZupra(out io.Writer) {
	for _, e := range zupraLetters {
		extent := letterExtent(e.Advance, zupraLineHeight, zupraYMin)
		sendRegisterColrv0Letter(out, e.Cp, zupraUpm, extent, e.Payload)
		flush(out)
	}
}

func registerTwemoji(out io.Writer) {
	for _, e := range twemojiLetters {
		sendRegisterColrv0(out, e.Cp, twemojiUpm, e.Payload)
		flush(out)
	}
}

func flush(out io.Writer) {
	if f, ok := out.(interface{ Sync() error }); ok {
		_ = f.Sync()
	}
}

func nablaRenderedText() string {
	var b strings.Builder
	for _, ch := range nablaText {
		mapped := ch
		for _, e := range nablaLetters {
			if e.Letter == ch {
				mapped = e.Cp
				break
			}
		}
		b.WriteRune(mapped)
	}
	return b.String()
}

// Same text through the width=2 duplicates. A width=2 glyph overflows
// into the following cell in *pixels*; the terminal does not reserve
// that cell, so the codepoint still counts as one logical column
// (== system wcwidth → no cursor/copy desync). We author an explicit
// trailing space after each glyph for the overflow to land on, which
// also makes the row read as a sequence of two-cell glyphs.
func nablaWideRenderedText() string {
	var b strings.Builder
	for _, ch := range nablaText {
		mapped := rune(-1)
		for _, e := range nablaLetters {
			if e.Letter == ch {
				mapped = e.Cp + nablaWideOffset
				break
			}
		}
		if mapped >= 0 {
			b.WriteRune(mapped)
			b.WriteRune(' ')
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func zupraRenderedText() string {
	var b strings.Builder
	for _, ch := range zupraText {
		mapped := ch
		for _, e := range zupraLetters {
			if e.Letter == ch {
				mapped = e.Cp
				break
			}
		}
		b.WriteRune(mapped)
	}
	return b.String()
}

func twemojiRenderedText() string {
	var b strings.Builder
	for _, ch := range twemojiText {
		mapped := ch
		for _, e := range twemojiLetters {
			if e.Letter == ch {
				mapped = e.Cp
				break
			}
		}
		b.WriteRune(mapped)
	}
	return b.String()
}

func notoFruitRenderedText() string {
	var b strings.Builder
	for _, ch := range notoFruitText {
		mapped := ch
		for _, e := range notoFruitLetters {
			if e.Letter == ch {
				mapped = e.Cp
				break
			}
		}
		b.WriteRune(mapped)
	}
	return b.String()
}

func glyfIcons() string {
	parts := make([]string, 0, len(slotCps))
	for _, cp := range slotCps {
		parts = append(parts, string(cp))
	}
	return strings.Join(parts, "")
}

// registerVg sends fmt=vg glyphs. The payload is text, not base64.
func registerVg(w io.Writer, glyphs []vgGlyph) {
	for _, g := range glyphs {
		extra := ""
		if g.extra != "" {
			extra = ";" + g.extra
		}
		fmt.Fprintf(w, "\x1b_25a1;r;cp=%x;fmt=vg;reply=0;vb=%s%s;%s\x1b\\",
			g.cp, g.vb, extra, g.payload)
	}
}

// vgLine joins glyphs with two spaces; a width=2 glyph takes the first.
func vgLine(glyphs []vgGlyph) string {
	parts := make([]string, 0, len(glyphs))
	for _, g := range glyphs {
		parts = append(parts, string(rune(g.cp)))
	}
	return strings.Join(parts, "  ")
}

func glyfWideIcons() string {
	// Two spaces between icons: a width=2 glyph overflows into the first
	// (blank) space in pixels — the cell is never reserved, so the
	// codepoint stays one logical column — and the second space is the
	// visible gap before the next icon.
	parts := make([]string, 0, len(wideSlotCps))
	for _, cp := range wideSlotCps {
		parts = append(parts, string(cp))
	}
	return strings.Join(parts, "  ")
}

// boxed wraps one placement glyph in a grey background block, so the
// cell box (and where the §8.5 transform put the outline inside it)
// is visible.
func boxed(cp rune) string {
	return fmt.Sprintf("\x1b[48;2;64;64;64m%s\x1b[0m", string(cp))
}

// boxedWide does the same for a width=2 glyph. The codepoint takes one
// column and the glyph overflows into the blank cell after it, so that
// cell is printed as a space inside the same grey block.
func boxedWide(cp rune) string {
	return fmt.Sprintf("\x1b[48;2;64;64;64m%s \x1b[0m", string(cp))
}

// probeSupport sends `s` followed by DA1 and reads until the DA1
// reply. Returns (names, true) when an `s` reply came first, or
// (nil, false) when it didn't. `timeout` only guards against a
// terminal that answers neither. Caller is responsible for ensuring
// stdin is currently owned by this process.
func probeSupport(timeout time.Duration) ([]string, bool) {
	fd := os.Stdin.Fd()
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, false
	}
	defer term.Restore(fd, oldState) //nolint:errcheck

	reader, err := cancelreader.NewReader(os.Stdin)
	if err != nil {
		return nil, false
	}
	defer reader.Close()

	// `s` then DA1 (spec §3.4): the DA1 reply settles the question.
	fmt.Fprint(os.Stdout, "\x1b_25a1;s\x1b\\\x1b[c")

	type result struct {
		fmts []string
		ok   bool
	}
	ch := make(chan result, 1)

	go func() {
		collected := make([]byte, 0, 64)
		buf := make([]byte, 256)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				ch <- result{nil, false}
				return
			}
			collected = append(collected, buf[:n]...)
			if v, ok, done := probeAnswer(collected); done {
				ch <- result{v, ok}
				return
			}
			if len(collected) > 4096 {
				ch <- result{nil, false}
				return
			}
		}
	}()

	select {
	case r := <-ch:
		return r.fmts, r.ok
	case <-time.After(timeout):
		reader.Cancel()
		return nil, false
	}
}

// probeAnswer reports whether the DA1 reply (`CSI ? … c`) has
// arrived, and if so the `fmt` list from an `s` reply before it. No
// `s` reply before DA1 means the terminal doesn't implement Glyph
// Protocol. Waiting for DA1 also drains its reply so it doesn't reach
// the app as input.
func probeAnswer(data []byte) (fmts []string, ok bool, done bool) {
	da1 := bytes.Index(data, []byte("\x1b[?"))
	if da1 < 0 || bytes.IndexByte(data[da1:], 'c') < 0 {
		return nil, false, false
	}
	fmts, ok = parseFmtField(data[:da1])
	return fmts, ok, true
}

// parseFmtField locates `fmt=` in an APC reply and splits its
// comma-separated list of format names. Returns ([]string{}, true)
// when the field is empty (terminal advertises no formats); (nil,
// false) when no `fmt=` field is present.
func parseFmtField(data []byte) ([]string, bool) {
	needle := []byte("fmt=")
	i := bytes.Index(data, needle)
	if i < 0 {
		return nil, false
	}
	start := i + len(needle)
	end := start
	for end < len(data) && data[end] != ';' && data[end] != 0x1b {
		end++
	}
	value := string(data[start:end])
	if value == "" {
		return []string{}, true
	}
	return strings.Split(value, ","), true
}

func main() {
	fmts, ok := probeSupport(2 * time.Second)
	if !ok {
		fmt.Fprintln(os.Stderr,
			"This terminal does not implement Glyph Protocol. Run in Rio or "+
				"another conformant terminal to see the demo.")
		return
	}

	hasGlyf := slices.Contains(fmts, "glyf")
	hasColrv0 := slices.Contains(fmts, "colrv0")
	hasColrv1 := slices.Contains(fmts, "colrv1")
	hasVg := slices.Contains(fmts, "vg")

	if hasGlyf {
		registerGlyfSlots(os.Stdout)
	}
	if hasColrv0 {
		registerZupra(os.Stdout)
		registerTwemoji(os.Stdout)
	}
	if hasColrv1 {
		registerNabla(os.Stdout)
		registerNotoFruit(os.Stdout)
	}
	if hasVg {
		for _, group := range [][]vgGlyph{vgIcons, vgPaints, vgEmojiNoto, vgEmojiTwemoji} {
			registerVg(os.Stdout, group)
		}
	}
	flush(os.Stdout)

	fmt.Println()

	colrv1Line := "(colrv1 not advertised by terminal)"
	colrv1WideLine := "(colrv1 not advertised by terminal)"
	if hasColrv1 {
		colrv1Line = nablaRenderedText()
		colrv1WideLine = nablaWideRenderedText()
	}
	colrv0Line := "(colrv0 not advertised by terminal)"
	colrv0Emoji := "(colrv0 not advertised by terminal)"
	if hasColrv0 {
		colrv0Line = zupraRenderedText()
		colrv0Emoji = twemojiRenderedText()
	}
	glyfLine := "(glyf not advertised by terminal)"
	glyfWideLine := "(glyf not advertised by terminal)"
	if hasGlyf {
		glyfLine = glyfIcons()
		glyfWideLine = glyfWideIcons()
	}

	colrv1Emoji := "(colrv1 not advertised by terminal)"
	if hasColrv1 {
		colrv1Emoji = notoFruitRenderedText()
	}

	fmt.Printf("colrv1 example: \x1b[48;2;21;49;162m%s\x1b[0m\n", colrv1Line)
	// Second line: the same letters via the width=2 duplicates.
	fmt.Printf("colrv1 example: %s\n", colrv1WideLine)
	fmt.Printf("colrv1 example: %s\n", colrv1Emoji)
	fmt.Printf("colrv0 example: %s\n", colrv0Line)
	fmt.Printf("colrv0 example: %s\n", colrv0Emoji)
	fmt.Printf("glyf examples:  %s\n", glyfLine)
	fmt.Printf("glyf wide ex.:  %s\n", glyfWideLine)
	if hasVg {
		fmt.Printf("vg icons:       %s\n", vgLine(vgIcons))
		fmt.Printf("vg paints:      %s\n", vgLine(vgPaints))
		fmt.Printf("vg emoji:       %s\n", vgLine(vgEmojiNoto))
		fmt.Printf("vg emoji:       %s\n", vgLine(vgEmojiTwemoji))
	} else {
		fmt.Println("vg examples:    (vg not advertised by terminal)")
	}

	// §8.5 sizing/placement — the same heart outline under different
	// size= / align= / pad= / width= parameters. The grey block is the
	// cell box; where the heart lands inside it is the demo.
	if hasGlyf {
		fmt.Printf(
			"placement size: %s  %s  %s  %s   (height / contain / cover / stretch)\n",
			boxed(0x100070), boxed(0x100071), boxed(0x100072), boxed(0x100073),
		)
		fmt.Printf(
			"placement align: %s %s  xy%sz   (start,start / end,end / baseline in text)\n",
			boxed(0x100074), boxed(0x100075), boxed(0x100076),
		)
		fmt.Printf(
			"placement more: %s %s   (stretch + pad=0.2 / width=2 contain)\n",
			boxed(0x100077), boxedWide(0x100078),
		)
	} else {
		fmt.Println("placement:      (glyf not advertised by terminal)")
	}
}
