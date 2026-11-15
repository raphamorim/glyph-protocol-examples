package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/screen"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/cancelreader"
)

var slotCps = [5]rune{
	0x100000, 0x100001, 0x100002, 0x100003, 0x100004,
}

var nerdGlyfsB64 = [5]string{
	"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",
}

// The Nerd Font icons are drawn for the font's own cell: a 600 unit
// advance, ascender 1020 and descender -300 at upm 1000. Declaring
// that extent (§8.5.1) places every icon the way the patched font
// would, and keeps the five in proportion to each other.
const nerdExtent = "aw=600;lh=1320;ymin=-300"

// Each letter of the colour demos is registered with the font's line
// height and descender, plus its own advance. size=height scales by
// line height alone, so every letter gets the same scale and they all
// sit on one baseline; the advance centres each letter in its cell.
func letterExtent(advance, lineHeight, ymin int) string {
	return fmt.Sprintf("aw=%d;lh=%d;ymin=%d;size=height", advance, lineHeight, ymin)
}

func sendRegisterGlyf(w *uv.Terminal, cp rune, payload string) {
	fmt.Fprintf(w, "\x1b_25a1;r;cp=%x;reply=0;upm=1000;%s;%s\x1b\\", cp, nerdExtent, payload)
}

func sendRegisterColrv1(w *uv.Terminal, cp rune, extent, payload string) {
	fmt.Fprintf(w, "\x1b_25a1;r;cp=%x;fmt=colrv1;reply=2;upm=1000;%s;%s\x1b\\", cp, extent, payload)
}

// Offset for the width=2 duplicates of the Nabla letters: the same
// colrv1 payloads re-registered at cp+nablaWideOffset (0x100020 to
// 0x10002C, free between the Nabla and Zupra ranges) so the second
// colrv1 line renders each letter across two cells.
const nablaWideOffset rune = 0x10

func sendRegisterColrv1Wide(w *uv.Terminal, cp rune, extent, payload string) {
	fmt.Fprintf(w, "\x1b_25a1;r;cp=%x;fmt=colrv1;reply=2;upm=1000;%s;width=2;%s\x1b\\", cp, extent, payload)
}

func sendRegisterColrv0(w *uv.Terminal, cp rune, upm uint32, extent, payload string) {
	fmt.Fprintf(w, "\x1b_25a1;r;cp=%x;fmt=colrv0;reply=2;upm=%d;%s;%s\x1b\\", cp, upm, extent, payload)
}

func registerGlyfSlots(w *uv.Terminal) {
	for i, cp := range slotCps {
		sendRegisterGlyf(w, cp, nerdGlyfsB64[i])
		w.Screen().Flush()
	}
}

func registerNabla(w *uv.Terminal) {
	for _, e := range nablaLetters {
		extent := letterExtent(e.Advance, nablaLineHeight, nablaYMin)
		sendRegisterColrv1(w, e.Cp, extent, e.Payload)
		w.Screen().Flush()
		// Wide duplicate for the second colrv1 line: same payload at
		// a fresh codepoint, drawn across two cells with width=2.
		sendRegisterColrv1Wide(w, e.Cp+nablaWideOffset, extent, e.Payload)
		w.Screen().Flush()
	}
}

func registerZupra(w *uv.Terminal) {
	for _, e := range zupraLetters {
		extent := letterExtent(e.Advance, zupraLineHeight, zupraYMin)
		sendRegisterColrv0(w, e.Cp, zupraUpm, extent, e.Payload)
		w.Screen().Flush()
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

func glyfIcons() string {
	parts := make([]string, 0, len(slotCps))
	for _, cp := range slotCps {
		parts = append(parts, string(cp))
	}
	return strings.Join(parts, "  ")
}

func probeSupport(timeout time.Duration) ([]string, bool) {
	fd := os.Stdin.Fd()
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return nil, false
	}
	defer term.Restore(fd, oldState)

	reader, err := cancelreader.NewReader(os.Stdin)
	if err != nil {
		return nil, false
	}
	defer reader.Close()

	fmt.Fprint(os.Stdout, "\x1b_25a1;s\x1b\\")

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
			if bytes.Contains(collected, []byte("\x1b\\")) {
				if v, ok := parseFmtField(collected); ok {
					ch <- result{v, true}
				} else {
					ch <- result{nil, false}
				}
				return
			}
			if len(collected) > 2048 {
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
	fmts, ok := probeSupport(200 * time.Millisecond)
	if !ok {
		fmt.Fprintln(os.Stderr,
			"This terminal does not implement Glyph Protocol. Run in Rio or "+
				"another conformant terminal to see the demo.")
		return
	}

	hasGlyf := slices.Contains(fmts, "glyf")
	hasColrv0 := slices.Contains(fmts, "colrv0")
	hasColrv1 := slices.Contains(fmts, "colrv1")

	t := uv.DefaultTerminal()
	if err := run(t, hasGlyf, hasColrv0, hasColrv1); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func run(t *uv.Terminal, hasGlyf, hasColrv0, hasColrv1 bool) error {
	scr := t.Screen()
	scr.EnterAltScreen()

	if err := t.Start(); err != nil {
		return err
	}
	defer t.Stop()

	if hasGlyf {
		registerGlyfSlots(t)
	}
	if hasColrv0 {
		registerZupra(t)
	}
	if hasColrv1 {
		registerNabla(t)
	}

	nablaLine := nablaRenderedText()
	nablaWideLine := nablaWideRenderedText()
	zupraLine := zupraRenderedText()
	glyfLine := glyfIcons()

	nablaBg := color.RGBA{21, 49, 162, 255}
	labelStyle := uv.Style{
		Fg:    color.RGBA{150, 180, 220, 255},
		Attrs: uv.AttrBold,
	}
	fallbackStyle := uv.Style{
		Fg: color.RGBA{180, 80, 80, 255},
	}

	display := func() {
		screen.Clear(scr)
		ctx := screen.NewContext(scr)
		bounds := scr.Bounds()
		y := max(2, bounds.Min.Y+1)

		drawLine := func(label, content string, fallback string, supported bool, bg color.Color) {
			if y >= bounds.Max.Y-1 {
				return
			}

			ctx.SetStyle(labelStyle)
			ctx.DrawString(label, bounds.Min.X+2, y)
			labelW := scr.StringWidth(label)

			if supported {
				if bg != nil {
					for x := bounds.Min.X + 2 + labelW; x < bounds.Max.X-2; x++ {
						cell := scr.CellAt(x, y)
						if cell != nil && cell.Content != " " {
							cell.Style.Bg = bg
						} else {
							scr.SetCell(x, y, &uv.Cell{
								Content: " ",
								Width:   1,
								Style:   uv.Style{Bg: bg},
							})
						}
					}
					ctx.SetStyle(uv.Style{Bg: bg})
				} else {
					ctx.SetStyle(uv.Style{})
				}
				ctx.DrawString(content, bounds.Min.X+2+labelW, y)
			} else {
				ctx.SetStyle(fallbackStyle)
				ctx.DrawString(fallback, bounds.Min.X+2+labelW, y)
			}
			y++
		}

		drawLine("colrv1: ", nablaLine, "(not supported)", hasColrv1, nablaBg)
		// Second line: the same letters via the width=2 duplicates.
		drawLine("colrv1: ", nablaWideLine, "(not supported)", hasColrv1, nil)
		drawLine("colrv0: ", zupraLine, "(not supported)", hasColrv0, nil)
		drawLine("glyf:   ", glyfLine, "(not supported)", hasGlyf, nil)

		y++
		if y < bounds.Max.Y-1 {
			ctx.SetStyle(uv.Style{Fg: color.RGBA{120, 120, 140, 255}})
			ctx.DrawString("Press any key to exit.", bounds.Min.X+2, y)
		}

		scr.Render()
		scr.Flush()
	}

	display()
	defer display()

	for ev := range t.Events() {
		switch ev := ev.(type) {
		case uv.WindowSizeEvent:
			scr.Resize(ev.Width, ev.Height)
			display()
		case uv.KeyPressEvent:
			return nil
		}
	}

	return nil
}
