// Glyph Protocol demo — sample output exercising the protocol's three
// payload formats:
//
//	colrv1 : "Rio Terminal Emulator" rendered via Typearture's Nabla.
//	colrv0 : "Using Glyph protocol" rendered via Zupra 3D eYe/FS.
//	glyf   : five Nerd Font outlines (branch / folder / home / heart /
//	         Rust cog) registered at consecutive PUA-B codepoints.
//
// The demo is deliberately non-TUI: it probes the protocol, sends the
// registrations, prints the lines in cooked mode, and exits.

package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
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
// direct visual comparison. Without `width=2` the terminal would call
// PUA-B narrow per UAX-11 / wcwidth and pack the icons into one cell.
var wideSlotCps = [5]rune{
	0x100005, 0x100006, 0x100007, 0x100008, 0x100009,
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
	// `reply=0` level tells the terminal to register silently — no
	// `status=0` ACK — which keeps the shell prompt clean on exit
	// (otherwise the queued ACK bytes spill to the next prompt).
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;reply=2;upm=1000;%s\x1b\\",
		cp, payload,
	)
}

func sendRegisterGlyfWide(out io.Writer, cp rune, payload string) {
	// Same as sendRegisterGlyf but with `width=2` so the terminal
	// reserves two cells for the codepoint at print time.
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;reply=2;upm=1000;width=2;%s\x1b\\",
		cp, payload,
	)
}

func sendRegisterColrv1(out io.Writer, cp rune, payload string) {
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;fmt=colrv1;reply=2;upm=1000;%s\x1b\\",
		cp, payload,
	)
}

func sendRegisterColrv0(out io.Writer, cp rune, upm uint32, payload string) {
	fmt.Fprintf(
		out,
		"\x1b_25a1;r;cp=%x;fmt=colrv0;reply=2;upm=%d;%s\x1b\\",
		cp, upm, payload,
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
}

func registerNabla(out io.Writer) {
	for _, e := range nablaLetters {
		sendRegisterColrv1(out, e.Cp, e.Payload)
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
		sendRegisterColrv0(out, e.Cp, zupraUpm, e.Payload)
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
	return strings.Join(parts, "  ")
}

func glyfWideIcons() string {
	parts := make([]string, 0, len(wideSlotCps))
	for _, cp := range wideSlotCps {
		parts = append(parts, string(cp))
	}
	return strings.Join(parts, "  ")
}

// Protocol bits, mirroring the spec's fmt bitfield.
const (
	fmtGlyfBit   uint8 = 0b0000_0001
	fmtColrv0Bit uint8 = 0b0000_0010
	fmtColrv1Bit uint8 = 0b0000_0100
)

// probeSupport sends `s` and waits up to `timeout` for the terminal
// to reply with its `fmt` bitfield. Returns (bits, true) on success,
// or (0, false) on timeout or malformed reply. Caller is responsible
// for ensuring stdin is currently owned by this process.
func probeSupport(timeout time.Duration) (uint8, bool) {
	fd := os.Stdin.Fd()
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, false
	}
	defer term.Restore(fd, oldState) //nolint:errcheck

	reader, err := cancelreader.NewReader(os.Stdin)
	if err != nil {
		return 0, false
	}
	defer reader.Close()

	fmt.Fprint(os.Stdout, "\x1b_25a1;s\x1b\\")

	type result struct {
		fmt uint8
		ok  bool
	}
	ch := make(chan result, 1)

	go func() {
		collected := make([]byte, 0, 64)
		buf := make([]byte, 256)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				ch <- result{0, false}
				return
			}
			collected = append(collected, buf[:n]...)
			if bytes.Contains(collected, []byte("\x1b\\")) {
				if v, ok := parseFmtField(collected); ok {
					ch <- result{v, true}
				} else {
					ch <- result{0, false}
				}
				return
			}
			if len(collected) > 2048 {
				ch <- result{0, false}
				return
			}
		}
	}()

	select {
	case r := <-ch:
		return r.fmt, r.ok
	case <-time.After(timeout):
		reader.Cancel()
		return 0, false
	}
}

// parseFmtField locates `fmt=` in an APC reply and parses the decimal
// u8 that follows.
func parseFmtField(data []byte) (uint8, bool) {
	needle := []byte("fmt=")
	i := bytes.Index(data, needle)
	if i < 0 {
		return 0, false
	}
	start := i + len(needle)
	end := start
	for end < len(data) && data[end] >= '0' && data[end] <= '9' {
		end++
	}
	if end == start {
		return 0, false
	}
	v, err := strconv.ParseUint(string(data[start:end]), 10, 8)
	if err != nil {
		return 0, false
	}
	return uint8(v), true
}

func main() {
	bits, ok := probeSupport(200 * time.Millisecond)
	if !ok {
		fmt.Fprintln(os.Stderr,
			"This terminal does not implement Glyph Protocol. Run in Rio or "+
				"another conformant terminal to see the demo.")
		return
	}

	hasGlyf := bits&fmtGlyfBit != 0
	hasColrv0 := bits&fmtColrv0Bit != 0
	hasColrv1 := bits&fmtColrv1Bit != 0

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
	flush(os.Stdout)

	fmt.Println()

	colrv1Line := "(colrv1 not advertised by terminal)"
	if hasColrv1 {
		colrv1Line = nablaRenderedText()
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
	fmt.Printf("colrv1 example: %s\n", colrv1Line)
	fmt.Printf("colrv1 example: %s\n", colrv1Emoji)
	fmt.Printf("colrv0 example: %s\n", colrv0Line)
	fmt.Printf("colrv0 example: %s\n", colrv0Emoji)
	fmt.Printf("glyf examples:  %s\n", glyfLine)
	fmt.Printf("glyf wide ex.:  %s\n", glyfWideLine)
}
