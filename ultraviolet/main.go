package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/ultraviolet/screen"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/cancelreader"
)

const (
	fmtGlyfBit   uint8 = 0b0000_0001
	fmtColrv0Bit uint8 = 0b0000_0010
	fmtColrv1Bit uint8 = 0b0000_0100
)

var slotCps = [5]rune{
	0x100000, 0x100001, 0x100002, 0x100003, 0x100004,
}

var nerdGlyfsB64 = [5]string{
	"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",
}

func sendRegisterGlyf(w *uv.Terminal, cp rune, payload string) {
	fmt.Fprintf(w, "\x1b_25a1;r;cp=%x;reply=0;upm=1000;%s\x1b\\", cp, payload)
}

func sendRegisterColrv1(w *uv.Terminal, cp rune, payload string) {
	fmt.Fprintf(w, "\x1b_25a1;r;cp=%x;fmt=colrv1;reply=2;upm=1000;%s\x1b\\", cp, payload)
}

func sendRegisterColrv0(w *uv.Terminal, cp rune, upm uint32, payload string) {
	fmt.Fprintf(w, "\x1b_25a1;r;cp=%x;fmt=colrv0;reply=2;upm=%d;%s\x1b\\", cp, upm, payload)
}

func registerGlyfSlots(w *uv.Terminal) {
	for i, cp := range slotCps {
		sendRegisterGlyf(w, cp, nerdGlyfsB64[i])
		w.Screen().Flush()
	}
}

func registerNabla(w *uv.Terminal) {
	for _, e := range nablaLetters {
		sendRegisterColrv1(w, e.Cp, e.Payload)
		w.Screen().Flush()
	}
}

func registerZupra(w *uv.Terminal) {
	for _, e := range zupraLetters {
		sendRegisterColrv0(w, e.Cp, zupraUpm, e.Payload)
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

func probeSupport(timeout time.Duration) (uint8, bool) {
	fd := os.Stdin.Fd()
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, false
	}
	defer term.Restore(fd, oldState)

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
		drawLine("colrv1: ", nablaLine, "(not supported)", hasColrv1, nil)
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
