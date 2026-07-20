// Package layout holds the CLI's reusable layout + component primitives, built
// on lipgloss v2. Components compose these instead of hand-rolling spacing,
// borders, truncation, and selection chrome — so the visual language stays
// consistent and the v2 wrap-discipline rule (see Panel) lives in one place.
package layout

import (
	"image/color"

	"charm.land/lipgloss/v2"

	"github.com/zeb-link/zeb/internal/ui/theme"
)

// Gutter is the fixed 2-cell left margin every list row carries so rows never
// shift when selection moves. Focused rows get a quiet accent tick; unfocused
// rows get blank space. Selection is a tick, never an inverted bar.
func Gutter(focused bool) string {
	if focused {
		return lipgloss.NewStyle().Foreground(theme.Bone).Render("▍") + " "
	}
	return "  "
}

// Dot renders the status bullet in a semantic tone (theme.Good/Warn/Bad).
func Dot(tone color.Color) string {
	return lipgloss.NewStyle().Foreground(tone).Render("●")
}

// ChipOpts tunes a context chip.
type ChipOpts struct {
	Tone    color.Color // accent tone when focused (e.g. theme.Collection for a collection chip)
	Active  bool        // the value is set/meaningful (vs. "none"/default)
	Focused bool        // the chip currently holds focus
}

// Chip renders a rounded label+value pill. Border is quiet (Dim) until focused,
// when it takes the chip's tone and the value ground gets a soft accent wash.
func Chip(label, value string, opts ChipOpts) string {
	border := theme.Dim
	if opts.Focused {
		border = opts.Tone
	}
	valueStyle := lipgloss.NewStyle().Foreground(theme.Bone)
	if !opts.Active {
		valueStyle = lipgloss.NewStyle().Foreground(theme.Faint)
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
	return box.Render(
		lipgloss.NewStyle().Foreground(theme.Muted).Render(label) + " " + valueStyle.Render(value),
	)
}

// Panel renders content inside a rounded, padded, width-framed box.
//
// Wrap discipline (lipgloss v2): Style.Width(n) is the FRAME width (border +
// padding included) and auto-wraps its text. Pass RAW text here — never
// pre-wrapped or pre-truncated text — or it double-wraps at mismatched widths
// and orphans the last word of every line. If you must pre-truncate, truncate
// at width − the frame size, which callers can read via FrameWidth.
func Panel(width int, focused bool, tone color.Color, content string) string {
	border := theme.Dim
	if focused {
		border = tone
	}
	return lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(1, 2).
		Render(content)
}

// FrameWidth returns the inner text width of a Panel of the given frame width
// (rounded border = 2 cells, Padding(1,2) = 4 cells horizontally).
func FrameWidth(frameWidth int) int {
	inner := frameWidth - 6
	if inner < 1 {
		return 1
	}
	return inner
}

// FillHeight pads content down to h rows, anchored top-left. Replaces manual
// strings.Repeat("\n", …) vertical fill.
func FillHeight(width, height int, content string) string {
	if height <= 0 {
		return content
	}
	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, content)
}

// Grid arranges tiles into rows, fitting as many columns as availW allows for
// the given tile width and inter-tile gap. Replaces hand-tuned 3→2→1 ladders.
func Grid(tiles []string, tileWidth, gap, availWidth int) string {
	if len(tiles) == 0 {
		return ""
	}
	columns := (availWidth + gap) / (tileWidth + gap)
	if columns < 1 {
		columns = 1
	}
	if columns > len(tiles) {
		columns = len(tiles)
	}
	gapCell := ""
	for i := 0; i < gap; i++ {
		gapCell += " "
	}
	var rows []string
	for i := 0; i < len(tiles); i += columns {
		end := i + columns
		if end > len(tiles) {
			end = len(tiles)
		}
		parts := make([]string, 0, (end-i)*2)
		for j := i; j < end; j++ {
			if j > i {
				parts = append(parts, gapCell)
			}
			parts = append(parts, tiles[j])
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, parts...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

// Truncate shortens value to at most limit display cells, counting runes (not
// bytes) so multi-byte glyphs never split mid-rune. An ellipsis takes the last
// cell when truncation happens.
func Truncate(value string, limit int) string {
	if limit <= 1 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-1]) + "…"
}
