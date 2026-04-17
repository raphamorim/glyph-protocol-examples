// Glyph Protocol demo — registers five real Nerd Font outlines at
// empty Supplementary PUA-B codepoints (U+100000..U+100004). The
// `glyf` bytes come straight out of JetBrainsMono Nerd Font via
// fontTools, so the icons look exactly like Nerd Fonts but the
// codepoints they're mounted at are empty — you don't need Nerd
// Fonts installed to see them.
//
// Keybindings:
//   ↑ re-register all
//   ↓ clear all registrations
//   q / esc / ctrl+c  quit

package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
)

// -------------------------------------------------------------------------
// Slots + glyf payloads
// -------------------------------------------------------------------------

var slotCps = [5]rune{
	0x100000, 0x100001, 0x100002, 0x100003, 0x100004,
}

var slotLabels = [5]string{"branch", "folder", "home", "heart", "rust"}

// Base64 `glyf` bytes extracted from JetBrainsMono Nerd Font via
// fontTools. units per em = 1000. Source Nerd Font codepoints:
//
//	branch  U+E0A0
//	folder  U+F07B
//	home    U+F015
//	heart   U+F004
//	rust    U+E7A8
var nerdGlyfsB64 = [5]string{
	"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",
}

// -------------------------------------------------------------------------
// Wire helpers
// -------------------------------------------------------------------------

func sendClearAll() {
	fmt.Fprint(os.Stdout, "\x1b_1cc6D;c\x1b\\")
}

func sendRegister(cp rune, payload string) {
	// `fmt` is implicit (`glyf`); `upm` defaults to 1000 but we send
	// it explicitly to match the font's authoring EM.
	fmt.Fprintf(
		os.Stdout,
		"\x1b_1cc6D;r;cp=%x;upm=1000;%s\x1b\\",
		cp, payload,
	)
}

func registerAll() {
	for i, cp := range slotCps {
		sendRegister(cp, nerdGlyfsB64[i])
	}
}

// -------------------------------------------------------------------------
// Styles
// -------------------------------------------------------------------------

var (
	// Charm-ish palette.
	colPink   = lipgloss.Color("#FF6AC7")
	colPurple = lipgloss.Color("#C4A0FA")
	colTeal   = lipgloss.Color("#00D7AF")
	colRed    = lipgloss.Color("#FF5F87")
	colDim    = lipgloss.Color("#7A7A8C")
	colFg     = lipgloss.Color("#F0F0F5")
	colMuted  = lipgloss.Color("#313138")
	colAccent = lipgloss.Color("#9D7CFF")

	titleStyle = lipgloss.NewStyle().
			Foreground(colPink).
			Bold(true).
			Padding(0, 2)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colDim).
			Italic(true).
			Padding(0, 2).
			MarginBottom(1)

	promptBox = lipgloss.NewStyle().
			Padding(1, 0).
			MarginBottom(1)

	segmentStyle = lipgloss.NewStyle().
			Foreground(colFg).
			Padding(0, 1)

	iconGlow = lipgloss.NewStyle().
			Foreground(colPink).
			Bold(true)

	arrow = lipgloss.NewStyle().
		Foreground(colAccent).
		Render(" \u276f ")

	tableBox = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colDim).
			Padding(0, 2).
			MarginBottom(1)

	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(colAccent).
				Bold(true).
				Underline(true)

	cpStyle = lipgloss.NewStyle().
		Foreground(colDim)

	labelStyle = lipgloss.NewStyle().
			Foreground(colFg)

	glyphStyle = lipgloss.NewStyle().
			Foreground(colPink).
			Bold(true)

	helpBox = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(colDim).
		Padding(0, 2)

	keyStyle = lipgloss.NewStyle().
			Foreground(colAccent).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(colFg)

	statusOn = lipgloss.NewStyle().
			Foreground(colTeal).
			Bold(true)

	statusOff = lipgloss.NewStyle().
			Foreground(colRed).
			Bold(true)
)

// -------------------------------------------------------------------------
// Model
// -------------------------------------------------------------------------

type model struct {
	cleared bool
}

func (m model) Init() tea.Cmd {
	registerAll()
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			sendClearAll()
			return m, tea.Quit
		case "up":
			registerAll()
			m.cleared = false
		case "down":
			sendClearAll()
			m.cleared = true
		}
	}
	return m, nil
}

// -------------------------------------------------------------------------
// View
// -------------------------------------------------------------------------

// renderPrompt builds a Powerline-flavoured line with colored segments.
// Each segment holds one icon + its label, separated by a chevron.
func renderPrompt() string {
	segments := []string{"main", "src", "~", "favs", "rio"}
	var parts []string
	for i, cp := range slotCps {
		glyph := iconGlow.Render(string(cp))
		label := segments[i]
		seg := segmentStyle.Render(glyph + "  " + label)
		parts = append(parts, seg)
	}
	return strings.Join(parts, arrow)
}

// renderTable lays out a neat three-column table.
func renderTable() string {
	var b strings.Builder
	b.WriteString(tableHeaderStyle.Render(
		fmt.Sprintf("%-10s  %-8s  %s", "CODEPOINT", "LABEL", "ICON"),
	))
	b.WriteByte('\n')
	for i, cp := range slotCps {
		b.WriteString(cpStyle.Render(fmt.Sprintf("U+%06X", cp)))
		b.WriteString("  ")
		b.WriteString(labelStyle.Render(fmt.Sprintf("%-8s", slotLabels[i])))
		b.WriteString("  ")
		b.WriteString(glyphStyle.Render(string(cp)))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderHelp(cleared bool) string {
	keyLine := func(key, desc string) string {
		return keyStyle.Render(key) + "  " + descStyle.Render(desc)
	}
	var status string
	if cleared {
		status = statusOff.Render("\u25cb ") +
			descStyle.Render("cleared") +
			" " + cpStyle.Render("(press \u2191 to re-register)")
	} else {
		status = statusOn.Render("\u25cf ") +
			descStyle.Render("5 Nerd Font outlines registered")
	}
	return strings.Join([]string{
		keyLine("\u2191", "re-register all"),
		keyLine("\u2193", "clear all registrations"),
		keyLine("q", "quit"),
		"",
		status,
	}, "\n")
}

func (m model) View() string {
	title := titleStyle.Render("\u2726 Glyph Protocol")
	subtitle := subtitleStyle.Render(
		"Real Nerd Font outlines at empty Supplementary PUA-B codepoints",
	)
	prompt := promptBox.Render(renderPrompt())
	table := tableBox.Render(renderTable())
	help := helpBox.Render(renderHelp(m.cleared))
	return lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		subtitle,
		prompt,
		table,
		help,
	)
}

// -------------------------------------------------------------------------
// Main
// -------------------------------------------------------------------------

func main() {
	// Note on cleanup: we deliberately do NOT send `c` on exit.
	// The terminal would reply with `status=0`, and because the reply
	// arrives in the PTY after bubbletea has torn down its input
	// reader, the bytes spill out to the user's shell as a visible
	// `.1cc6D;c;status=0`. Per spec, registrations live for the
	// terminal session anyway; the user can press `↓` to clear
	// explicitly if they want.
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
